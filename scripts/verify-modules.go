// Command verify-modules checks the three module candidates using an isolated
// file proxy. Its synthetic checksums are never copied into source.
// Run from the repository root: GOWORK=off go run ./scripts/verify-modules.go.
package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

const (
	moduleBase = "github.com/hareku/its-forum-go/"
	version    = "v0.1.0"
	goVersion  = "1.27.0"
)

type module struct {
	name         string
	consumer     string
	dependencies []string
	files        []string
}

// This inventory deliberately describes only the three current modules. Adding a
// module or distribution file requires updating this list and its consumer.
var modules = []module{
	{"rc013v1", "consumer-rc013", nil, strings.Fields(`LICENSE README.md go.mod
 common.go errors.go errors_test.go example_test.go frame_codec.go frames.go
 frames_test.go free.go message.go message_test.go validation.go values.go
 internal/bitio/bitio.go internal/bitio/bitio_test.go`)},
	{"rc016v1", "consumer-rc016", []string{moduleBase + "rc013v1"}, strings.Fields(`LICENSE README.md go.mod
 bicycle.go bicycle_fuzz_test.go bicycle_profiles.go bicycle_profiles_test.go bicycle_test.go
 csma.go csma_fuzz_test.go csma_test.go doc.go errors.go example_test.go
 roadside.go roadside_fuzz_test.go roadside_profiles.go roadside_profiles_test.go roadside_test.go
 internal/bitio/bitio.go internal/bitio/bitio_test.go
 testdata/bicycle.hex testdata/csma.hex testdata/personal-vectors.md`)},
	{"rc016v2", "consumer-rc016v2", []string{moduleBase + "rc013v1"}, strings.Fields(`LICENSE README.md go.mod doc.go errors.go
 frames.go payloads.go values.go validation.go profiles.go message.go message_validation.go
 frames_test.go payloads_test.go validation_test.go message_test.go
 message_validation_test.go message_fuzz_test.go example_test.go
 internal/bitio/bitio.go internal/bitio/bitio_test.go
 testdata/spec-vectors.md testdata/bicycle.hex testdata/pedestrian.hex`)},
}

type savedFile struct {
	exists bool
	data   []byte
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL %v\n", err)
		os.Exit(1)
	}
}

func report(label, subject, detail string) {
	fmt.Printf("[%s] module=%s version=%s go=%s %s\n", label, subject, version, runtime.Version(), detail)
}

func run() (result error) {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	before, err := snapshot(root)
	if err != nil {
		return fmt.Errorf("SOURCE_IMMUTABILITY: %w", err)
	}
	// Check even when inventory, subprocesses, or negative checks fail.
	defer func() {
		after, err := snapshot(root)
		if err == nil {
			for name, original := range before {
				current := after[name]
				if current.exists != original.exists || !bytes.Equal(current.data, original.data) {
					err = fmt.Errorf("source metadata changed: %s", name)
					break
				}
			}
		}
		if err != nil {
			result = errors.Join(result, fmt.Errorf("SOURCE_IMMUTABILITY: %w", err))
		} else {
			report("SOURCE_IMMUTABILITY", "all", "PASS metadata existence and bytes unchanged")
		}
	}()
	if runtime.Version() != "go"+goVersion {
		return fmt.Errorf("ISOLATED_ENV: need Go %s, got %s", goVersion, runtime.Version())
	}
	if err := checkSource(root); err != nil {
		return err
	}
	report("SOURCE_INVENTORY", "all", "PASS explicit module README, codec, test, internal and applicable testdata files")
	report("BITIO_PARITY", "all", "PASS implementation and tests are identical")
	report("LICENSE_SOURCE", "all", "PASS explicit module copies match root MIT license")
	temp, err := os.MkdirTemp("", "its-forum-verify-")
	if err != nil {
		return err
	}
	defer func() {
		// Go extracts module directories read-only; restore directory permissions
		// before removing this runner's private cache on both success and failure.
		walkErr := filepath.WalkDir(temp, func(name string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return os.Chmod(name, 0700)
			}
			return nil
		})
		cleanErr := errors.Join(walkErr, os.RemoveAll(temp))
		if cleanErr != nil {
			result = errors.Join(result, fmt.Errorf("TEMP_CLEANUP: %w", cleanErr))
		} else {
			report("TEMP_CLEANUP", "all", "PASS temporary proxy, stages, consumers and caches removed")
		}
	}()
	proxy := filepath.Join(temp, "proxy")
	inventories := make(map[string]map[string][]byte)
	for _, m := range modules {
		files, err := inventory(filepath.Join(root, m.name), m)
		if err != nil {
			return err
		}
		inventories[m.name] = files
		if err := writeProxy(proxy, m, files); err != nil {
			return fmt.Errorf("PROXY_FIXTURE: %w", err)
		}
		if err := checkZip(proxy, m, files); err != nil {
			return err
		}
	}
	report("PROXY_FIXTURE", "all", "PASS synthetic current candidates; not public VCS checksums")
	env, cache, err := isolatedEnv(filepath.Join(temp, "positive"), proxy)
	if err != nil {
		return err
	}
	report("ISOLATED_ENV", "all", "PASS GOWORK=off GOENV=off GOTOOLCHAIN=local; single file proxy; fresh private caches")
	for _, m := range modules {
		stage := filepath.Join(temp, "stage", m.name)
		for _, sibling := range modules {
			if sibling.name != m.name {
				if err := checkStageAbsent(filepath.Join(temp, "stage", sibling.name)); err != nil {
					return err
				}
			}
		}
		report("ISOLATED_STAGE", moduleBase+m.name, "PASS sibling source stages absent before creation")
		if err := writeFiles(stage, inventories[m.name]); err != nil {
			return err
		}
		label := "ISOLATED_" + strings.ToUpper(m.name)
		if _, err := goCommand(stage, env, label, "test", "-mod=mod", "-count=1", "./..."); err != nil {
			return err
		}
		if err := sameFile(filepath.Join(stage, "go.mod"), inventories[m.name]["go.mod"]); err != nil {
			return err
		}
		dependencies := m.dependencies
		if err := readonlyChecks(stage, env, cache, moduleBase+m.name, dependencies, label); err != nil {
			return err
		}
		if err := os.RemoveAll(stage); err != nil {
			return fmt.Errorf("ISOLATED_STAGE: remove completed %s stage: %w", m.name, err)
		}
		if err := checkStageAbsent(stage); err != nil {
			return err
		}
		report("ISOLATED_STAGE", moduleBase+m.name, "PASS completed source stage removed before next scenario")
	}
	for _, m := range modules {
		name := m.consumer
		stage := filepath.Join(temp, "consumers", name)
		fixture, err := os.ReadFile(filepath.Join(root, "scripts", "testdata", name, "consumer_test.go"))
		if err != nil {
			return err
		}
		initial := fmt.Sprintf("module example.com/%s\n\ngo %s\n\nrequire %s%s %s\n", name, goVersion, moduleBase, m.name, version)
		if err := writeFiles(stage, map[string][]byte{"go.mod": []byte(initial), "consumer_test.go": fixture}); err != nil {
			return err
		}
		label := "CONSUMER_" + strings.ToUpper(strings.TrimPrefix(name, "consumer-"))
		if _, err := goCommand(stage, env, label, "mod", "download"); err != nil {
			return err
		}
		if _, err := goCommand(stage, env, label, "test", "-mod=mod", "-count=1", "./..."); err != nil {
			return err
		}
		dependencies := append([]string{moduleBase + m.name}, m.dependencies...)
		if slices.Contains(m.dependencies, moduleBase+"rc013v1") {
			resolved, err := goCommand(stage, env, label, "mod", "edit", "-json")
			if err != nil {
				return err
			}
			var metadata struct {
				Require []struct {
					Path, Version string
					Indirect      bool
				}
			}
			if err := json.Unmarshal(resolved, &metadata); err != nil {
				return err
			}
			direct := false
			for _, required := range metadata.Require {
				if required.Path == moduleBase+"rc013v1" && required.Version == version && !required.Indirect {
					direct = true
				}
			}
			if !direct {
				return fmt.Errorf("PUBLIC_TYPE_IDENTITY: RC013 direct require missing from resolved consumer")
			}

			report("PUBLIC_TYPE_IDENTITY", moduleBase+m.name, "PASS RC013 field, Issue, Error and sentinel identity")
		}
		if err := readonlyChecks(stage, env, cache, "example.com/"+name, dependencies, label); err != nil {
			return err
		}
	}
	if err := negativeChecks(root, temp, proxy, inventories); err != nil {
		return err
	}
	report("VERIFY_MODULES", "all", "PASS local candidate distribution and independent consumption; public fetch NOT RUN")
	return nil
}

func checkStageAbsent(stage string) error {
	_, err := os.Lstat(stage)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("ISOLATED_STAGE: inspect source stage %s: %w", stage, err)
	}
	return fmt.Errorf("ISOLATED_STAGE: source stage exists: %s", stage)
}

func snapshot(root string) (map[string]savedFile, error) {
	out := make(map[string]savedFile)
	dirs := []string{""}
	for _, m := range modules {
		dirs = append(dirs, m.name)
	}
	for _, dir := range dirs {
		for _, name := range []string{"go.mod", "go.sum", "go.work", "go.work.sum"} {
			relative := filepath.Join(dir, name)
			data, err := os.ReadFile(filepath.Join(root, relative))
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
			out[relative] = savedFile{err == nil, data}
		}
	}
	return out, nil
}

func checkSource(root string) error {
	if _, err := os.Lstat(filepath.Join(root, "go.mod")); !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("MODULE_METADATA: run from repository root without a root go.mod")
	}
	work, err := os.ReadFile(filepath.Join(root, "go.work"))
	if err != nil {
		return fmt.Errorf("MODULE_METADATA: run from repository root: %w", err)
	}
	expectedWork := "go " + goVersion + " use ("
	for _, m := range modules {
		expectedWork += " ./" + m.name
	}
	expectedWork += " )"
	if strings.Join(strings.Fields(string(work)), " ") != expectedWork {
		return fmt.Errorf("MODULE_METADATA: unexpected go.work directives")
	}
	license, err := os.ReadFile(filepath.Join(root, "LICENSE"))
	if err != nil {
		return fmt.Errorf("LICENSE_SOURCE: %w", err)
	}
	// Canonical MIT text with Copyright (c) 2026 hareku and a final newline.
	if fmt.Sprintf("%x", sha256.Sum256(license)) != "c602602e391405dfea0f763db1505f361c082869459738b2345a942092db3285" {
		return fmt.Errorf("LICENSE_SOURCE: expected canonical MIT license and copyright")
	}
	for _, m := range modules {
		files, err := inventory(filepath.Join(root, m.name), m)
		if err != nil {
			return err
		}
		expected := "module " + moduleBase + m.name + " go " + goVersion
		for _, dependency := range m.dependencies {
			expected += " require " + dependency + " " + version
		}
		if strings.Join(strings.Fields(string(files["go.mod"])), " ") != expected {
			return fmt.Errorf("MODULE_METADATA: %s module path, Go version or dependency directives differ (replace is forbidden)", m.name)
		}
		if !bytes.Equal(files["LICENSE"], license) {
			return fmt.Errorf("LICENSE_SOURCE: %s differs from root LICENSE", m.name)
		}
	}
	for _, name := range []string{"bitio.go", "bitio_test.go"} {
		first, err := os.ReadFile(filepath.Join(root, "rc013v1", "internal", "bitio", name))
		if err != nil {
			return err
		}
		for _, m := range modules[1:] {
			other, err := os.ReadFile(filepath.Join(root, m.name, "internal", "bitio", name))
			if err != nil {
				return err
			}
			if !bytes.Equal(first, other) {
				return fmt.Errorf("BITIO_PARITY: %s/%s differs", m.name, name)
			}
		}
	}
	return nil
}

func inventory(dir string, m module) (map[string][]byte, error) {
	allowed := make(map[string]bool)
	for _, name := range m.files {
		if !fs.ValidPath(name) || path.Clean(name) != name || strings.Contains(name, "\\") {
			return nil, fmt.Errorf("SOURCE_INVENTORY: unsafe path %q", name)
		}
		allowed[name] = true
	}
	files := make(map[string][]byte)
	err := filepath.WalkDir(dir, func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, name)
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink rejected: %s", name)
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("nonregular file rejected: %s", name)
		}
		rel = filepath.ToSlash(rel)
		if rel != "go.mod" && path.Base(rel) == "go.mod" {
			return fmt.Errorf("nested module rejected: %s", name)
		}
		// A future public checksum file remains in source but must not authenticate
		// these synthetic candidate ZIPs. Only temporary stages regenerate sums.
		if rel == "go.sum" {
			return nil
		}
		if !allowed[rel] {
			return fmt.Errorf("file outside explicit inventory: %s", name)
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		files[rel] = data
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("SOURCE_INVENTORY: %w", err)
	}
	for _, name := range m.files {
		if _, exists := files[name]; !exists {
			return nil, fmt.Errorf("SOURCE_INVENTORY: missing %s/%s", m.name, name)
		}
	}
	return files, nil
}

func writeFiles(dir string, files map[string][]byte) error {
	for name, data := range files {
		target := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0600); err != nil {
			return err
		}
	}
	return nil
}

func writeProxy(proxy string, m module, files map[string][]byte) error {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, name := range m.files {
		entry, err := writer.Create(moduleBase + m.name + "@" + version + "/" + name)
		if err != nil {
			return err
		}
		if _, err := entry.Write(files[name]); err != nil {
			return err
		}
	}
	if err := writer.Close(); err != nil {
		return err
	}
	dir := filepath.Join(proxy, filepath.FromSlash(moduleBase+m.name), "@v")
	return writeFiles(dir, map[string][]byte{
		version + ".mod":  files["go.mod"],
		version + ".info": []byte(`{"Version":"v0.1.0","Time":"2026-01-01T00:00:00Z"}`),
		version + ".zip":  buffer.Bytes(),
		"list":            []byte(version + "\n"),
	})
}

func checkZip(proxy string, m module, files map[string][]byte) error {
	filename := filepath.Join(proxy, filepath.FromSlash(moduleBase+m.name), "@v", version+".zip")
	archive, err := zip.OpenReader(filename)
	if err != nil {
		return fmt.Errorf("ZIP_CONTENTS: %w", err)
	}
	defer archive.Close()
	seen := make(map[string]bool)
	prefix := moduleBase + m.name + "@" + version + "/"
	for _, entry := range archive.File {
		name, ok := strings.CutPrefix(entry.Name, prefix)
		expected, exists := files[name]
		if !ok || !exists || seen[name] || !entry.Mode().IsRegular() {
			return fmt.Errorf("ZIP_CONTENTS: unexpected entry %s", entry.Name)
		}
		stream, err := entry.Open()
		if err != nil {
			return err
		}
		data, readErr := io.ReadAll(stream)
		closeErr := stream.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			return err
		}
		if !bytes.Equal(data, expected) {
			return fmt.Errorf("ZIP_CONTENTS: bytes differ for %s", entry.Name)
		}
		seen[name] = true
	}
	if len(seen) != len(files) {
		return fmt.Errorf("ZIP_CONTENTS: incomplete %s inventory", m.name)
	}
	report("LICENSE_ZIP", moduleBase+m.name, fmt.Sprintf("PASS explicit root-to-module copy sha256=%x", sha256.Sum256(files["LICENSE"])))
	report("ZIP_CONTENTS", moduleBase+m.name, fmt.Sprintf("PASS %d exact inventory entries, including module README, internal sources/tests and applicable testdata", len(seen)))
	return nil
}

func isolatedEnv(base, proxy string) ([]string, string, error) {
	cache := filepath.Join(base, "modcache")
	for _, dir := range []string{base, cache, filepath.Join(base, "gopath"), filepath.Join(base, "buildcache")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, "", err
		}
	}
	proxyURL := (&url.URL{Scheme: "file", Path: filepath.ToSlash(proxy)}).String()
	overrides := map[string]string{
		"GOWORK": "off", "GOENV": "off", "GOFLAGS": "", "GOTOOLCHAIN": "local",
		"GOPROXY": proxyURL, "GOSUMDB": "off", "GOPRIVATE": "", "GONOPROXY": "", "GONOSUMDB": "",
		"GOPATH": filepath.Join(base, "gopath"), "GOMODCACHE": cache, "GOCACHE": filepath.Join(base, "buildcache"),
	}
	env := []string{}
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		if _, overridden := overrides[key]; !overridden {
			env = append(env, item)
		}
	}
	for key, value := range overrides {
		env = append(env, key+"="+value)
	}
	return env, cache, nil
}

func goCommand(dir string, env []string, label string, args ...string) ([]byte, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s: go %s: %w\n%s", label, strings.Join(args, " "), err, output)
	}
	report(label, filepath.Base(dir), "PASS go "+strings.Join(args, " "))
	return output, nil
}

func sameFile(name string, expected []byte) error {
	actual, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	if !bytes.Equal(actual, expected) {
		return fmt.Errorf("MODULE_METADATA: temporary stage metadata changed: %s", name)
	}
	return nil
}

func readonlyChecks(dir string, env []string, cache, mainPath string, dependencies []string, label string) error {
	mod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return err
	}
	for _, args := range [][]string{{"test", "-mod=readonly", "-count=1", "./..."}, {"vet", "-mod=readonly", "./..."}, {"mod", "verify"}} {
		if _, err := goCommand(dir, env, label, args...); err != nil {
			return err
		}
	}
	output, err := goCommand(dir, env, "ISOLATED_GRAPH", "list", "-mod=readonly", "-m", "-json", "all")
	if err != nil {
		return err
	}
	if err := checkGraph(output, dir, cache, mainPath, dependencies); err != nil {
		return err
	}
	if err := sameFile(filepath.Join(dir, "go.mod"), mod); err != nil {
		return err
	}
	report("ISOLATED_GRAPH", mainPath, "PASS exact main directory, allowed versions, cache containment, no replace")
	return nil
}

func checkGraph(data []byte, dir, cache, mainPath string, dependencies []string) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	seen := make(map[string]bool)
	mainCount := 0
	for {
		var entry struct {
			Path, Version, Dir string
			Main               bool
			Replace            json.RawMessage
		}
		err := decoder.Decode(&entry)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("ISOLATED_GRAPH: %w", err)
		}
		if seen[entry.Path] || (len(entry.Replace) > 0 && string(entry.Replace) != "null") {
			return fmt.Errorf("ISOLATED_GRAPH: duplicate or replaced module %s", entry.Path)
		}
		seen[entry.Path] = true
		if entry.Main {
			mainCount++
			if entry.Path != mainPath || entry.Dir == "" || filepath.Clean(entry.Dir) != filepath.Clean(dir) {
				return fmt.Errorf("ISOLATED_GRAPH: unexpected main %s at %s", entry.Path, entry.Dir)
			}
		} else {
			if !slices.Contains(dependencies, entry.Path) || entry.Version != version || !strictlyWithin(cache, entry.Dir) {
				return fmt.Errorf("ISOLATED_GRAPH: unexpected dependency %s@%s at %s", entry.Path, entry.Version, entry.Dir)
			}
		}
	}
	if mainCount != 1 || !seen[mainPath] || len(seen) != len(dependencies)+1 {
		return fmt.Errorf("ISOLATED_GRAPH: incomplete graph for %s", mainPath)
	}
	return nil
}

func strictlyWithin(parent, child string) bool {
	if child == "" || !filepath.IsAbs(child) {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(child))
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func negativeChecks(root, temp, proxy string, inventories map[string]map[string][]byte) error {
	for _, m := range modules {
		if len(m.dependencies) == 0 {
			continue
		}
		missingProxy := filepath.Join(temp, "missing-proxy", m.name)
		if err := writeProxy(missingProxy, m, inventories[m.name]); err != nil {
			return err
		}
		env, _, err := isolatedEnv(filepath.Join(temp, "negative-missing-cache", m.name), missingProxy)
		if err != nil {
			return err
		}
		stage := filepath.Join(temp, "negative-missing-consumer", m.name)
		fixture, err := os.ReadFile(filepath.Join(root, "scripts", "testdata", m.consumer, "consumer_test.go"))
		if err != nil {
			return err
		}
		mod := "module example.com/negative-missing\n\ngo " + goVersion + "\n\nrequire " + moduleBase + m.name + " " + version + "\n"
		if err := writeFiles(stage, map[string][]byte{"go.mod": []byte(mod), "consumer_test.go": fixture}); err != nil {
			return err
		}
		output, commandErr := goCommand(stage, env, "NEGATIVE_MISSING_DEPENDENCY", "test", "-mod=mod", "./...")
		if commandErr == nil || !bytes.Contains(output, []byte(moduleBase+"rc013v1@"+version)) || !bytes.Contains(output, []byte("no such file or directory")) || !bytes.Contains(output, []byte("missing-proxy")) {
			return fmt.Errorf("NEGATIVE_MISSING_DEPENDENCY: %s expected missing RC013 file-proxy error, got %v\n%s", m.name, commandErr, output)
		}
		report("NEGATIVE_MISSING_DEPENDENCY", m.name, "PASS expected missing dependency failure with fresh cache")
	}
	for _, owner := range modules {
		env, _, err := isolatedEnv(filepath.Join(temp, "negative-internal-cache", owner.name), proxy)
		if err != nil {
			return err
		}
		stage := filepath.Join(temp, "negative-internal-consumer", owner.name)
		mod := "module example.com/negative-internal\n\ngo " + goVersion + "\n\nrequire (\n"
		for _, m := range modules {
			mod += moduleBase + m.name + " " + version + "\n"
		}
		mod += ")\n"
		illegal := "package consumer_test\nimport _ \"" + moduleBase + "rc016v2\"\nimport _ \"" + moduleBase + owner.name + "/internal/bitio\"\n"
		if err := writeFiles(stage, map[string][]byte{"go.mod": []byte(mod), "consumer_test.go": []byte(illegal)}); err != nil {
			return err
		}
		if _, err := goCommand(stage, env, "NEGATIVE_INTERNAL_IMPORT", "mod", "download", "all"); err != nil {
			return err
		}
		output, commandErr := goCommand(stage, env, "NEGATIVE_INTERNAL_IMPORT", "test", "-mod=mod", "./...")
		if commandErr == nil || !bytes.Contains(output, []byte("use of internal package "+moduleBase+owner.name+"/internal/bitio not allowed")) {
			return fmt.Errorf("NEGATIVE_INTERNAL_IMPORT: expected internal visibility error, got %v\n%s", commandErr, output)
		}
		report("NEGATIVE_INTERNAL_IMPORT", owner.name, "PASS expected internal visibility failure after successful module download")
	}
	return mutationChecks(root, temp, inventories)
}

func mutationChecks(root, temp string, inventories map[string]map[string][]byte) error {
	clone := filepath.Join(temp, "mutation-clone")
	for _, m := range modules {
		if err := writeFiles(filepath.Join(clone, m.name), inventories[m.name]); err != nil {
			return err
		}
	}
	if detected := checkStageAbsent(filepath.Join(clone, "rc013v1")); detected == nil || !strings.Contains(detected.Error(), "ISOLATED_STAGE: source stage exists:") {
		return fmt.Errorf("NEGATIVE_STAGE_ISOLATION: existing sibling source stage was not rejected: %v", detected)
	}
	report("NEGATIVE_STAGE_ISOLATION", moduleBase+"rc013v1", "PASS expected existing sibling source rejection in temporary clone")
	for _, name := range []string{"LICENSE", "go.work"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return err
		}
		if err := writeFiles(clone, map[string][]byte{name: data}); err != nil {
			return err
		}
	}
	type mutationCase struct {
		name, file, label string
		data              []byte
	}
	var cases []mutationCase
	for _, m := range modules {
		original := inventories[m.name]["go.mod"]
		for _, mutation := range []struct {
			name string
			data []byte
		}{
			{"replace", append(bytes.Clone(original), []byte("\nreplace "+moduleBase+"rc013v1 => ../rc013v1\n")...)},
			{"module-path", bytes.ReplaceAll(original, []byte("module "+moduleBase+m.name), []byte("module example.com/wrong"))},
			{"go-version", bytes.ReplaceAll(original, []byte("go "+goVersion), []byte("go 1.26.0"))},
		} {
			cases = append(cases, mutationCase{m.name + "-" + mutation.name, m.name + "/go.mod", "MODULE_METADATA", mutation.data})
		}
		if len(m.dependencies) > 0 {
			for _, mutation := range []struct{ name, old, replacement string }{
				{"missing-require", "require " + moduleBase + "rc013v1 " + version, ""},
				{"wrong-require", moduleBase + "rc013v1", moduleBase + "rc016v1"},
				{"require-version", version, "v0.2.0"},
			} {
				cases = append(cases, mutationCase{m.name + "-" + mutation.name, m.name + "/go.mod", "MODULE_METADATA", bytes.ReplaceAll(original, []byte(mutation.old), []byte(mutation.replacement))})
			}
		}
		if m.name != "rc013v1" {
			for _, name := range []string{"bitio.go", "bitio_test.go"} {
				cases = append(cases, mutationCase{m.name + "-" + name, m.name + "/internal/bitio/" + name, "BITIO_PARITY", []byte("package bitio\n")})
			}
		}
		cases = append(cases, mutationCase{m.name + "-license", m.name + "/LICENSE", "LICENSE_SOURCE", []byte("different license\n")})
	}
	workspace, err := os.ReadFile(filepath.Join(clone, "go.work"))
	if err != nil {
		return err
	}
	cases = append(cases, mutationCase{"workspace-v2-missing", "go.work", "MODULE_METADATA", bytes.ReplaceAll(workspace, []byte("./rc016v2"), nil)})
	for _, c := range cases {
		filename := filepath.Join(clone, filepath.FromSlash(c.file))
		original, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filename, c.data, 0600); err != nil {
			return err
		}
		detected := checkSource(clone)
		if err := os.WriteFile(filename, original, 0600); err != nil {
			return err
		}
		if detected == nil || !strings.Contains(detected.Error(), c.label) {
			return fmt.Errorf("NEGATIVE_METADATA: %s mutation was not rejected by %s: %v", c.name, c.label, detected)
		}
		report("NEGATIVE_METADATA", c.name, "PASS expected "+c.label+" rejection in temporary clone")
	}
	// Exercise inventory and ZIP validators for every module in private clones.
	for _, m := range modules {
		dir := filepath.Join(clone, m.name)
		for _, name := range m.files {
			filename := filepath.Join(dir, filepath.FromSlash(name))
			if err := os.Remove(filename); err != nil {
				return err
			}
			detected := checkSource(clone)
			if err := os.WriteFile(filename, inventories[m.name][name], 0600); err != nil {
				return err
			}
			if detected == nil || !strings.Contains(detected.Error(), "SOURCE_INVENTORY") {
				return fmt.Errorf("NEGATIVE_METADATA: missing %s/%s not rejected: %v", m.name, name, detected)
			}
		}
		report("NEGATIVE_METADATA", m.name, "PASS every missing inventory file rejected")
		for _, kind := range []string{"extra", "nested", "symlink"} {
			name := "unexpected.txt"
			if kind == "nested" {
				name = "nested/go.mod"
			}
			filename := filepath.Join(dir, filepath.FromSlash(name))
			if kind == "symlink" {
				if err := os.Symlink("README.md", filename); err != nil {
					return err
				}
			} else if err := writeFiles(dir, map[string][]byte{name: []byte("unexpected\n")}); err != nil {
				return err
			}
			detected := checkSource(clone)
			if err := os.Remove(filename); err != nil {
				return err
			}
			if kind == "nested" {
				if err := os.Remove(filepath.Dir(filename)); err != nil {
					return err
				}
			}
			if detected == nil || !strings.Contains(detected.Error(), "SOURCE_INVENTORY") {
				return fmt.Errorf("NEGATIVE_METADATA: %s %s not rejected: %v", m.name, kind, detected)
			}
			report("NEGATIVE_METADATA", m.name+"-"+kind, "PASS expected SOURCE_INVENTORY rejection")
		}
		for _, name := range []string{"LICENSE", "internal/bitio/bitio.go", "README.md"} {
			modified := make(map[string][]byte)
			for file, data := range inventories[m.name] {
				modified[file] = data
			}
			modified[name] = []byte("damaged temporary ZIP entry\n")
			proxy := filepath.Join(temp, "mutation-proxy", m.name)
			if err := writeProxy(proxy, m, modified); err != nil {
				return err
			}
			detected := checkZip(proxy, m, inventories[m.name])
			if detected == nil || !strings.Contains(detected.Error(), "ZIP_CONTENTS") {
				return fmt.Errorf("NEGATIVE_METADATA: ZIP %s/%s mutation not rejected: %v", m.name, name, detected)
			}
			report("NEGATIVE_METADATA", m.name+"/"+name, "PASS expected ZIP_CONTENTS rejection")
		}
	}
	return nil
}
