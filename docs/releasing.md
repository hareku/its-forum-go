# モジュールのリリース

全3モジュールの初回リリース `v0.1.0` は2026年10月8日に公開済みです。各モジュールで、以下の公開consumer検証による公開proxy・sumdb経由の取得、test／vet、依存graph、配布LICENSEの確認に成功しました。GitHub Releaseへのリンクは[ルートREADME](../README.md)を参照してください。

以下は初回公開で使用した手順と再検証用のスクリプトです。次回公開では対象のバージョンに合わせてタグ・require・検証スクリプトを更新し、同じ候補検証・公開取得検証を行います。

## 規格版とGo API版

| 対象 | module path | Go API版の例 | Git tag |
|---|---|---|---|
| RC-013 1.1 | `github.com/hareku/its-forum-go/rc013v1` | 初回 `v0.1.0` | `rc013v1/v0.1.0` |
| RC-016 1.0 | `github.com/hareku/its-forum-go/rc016v1` | 初回 `v0.1.0` | `rc016v1/v0.1.0` |
| 将来、規格major 1のままGo API major 2 | `github.com/hareku/its-forum-go/rc016v1/v2` | `v2.0.0` | `rc016v1/v2.0.0` |
| RC-016 2.0 | `github.com/hareku/its-forum-go/rc016v2` | 初回 `v0.1.0` | `rc016v2/v0.1.0` |

Go API major 2の行は将来の命名方針で、実装済みモジュールではありません。規格major、対応する完全な規格版、Go API SemVer、wire上の `Header.Version` はそれぞれ独立です。`rc016v1` は同じ規格majorの全minor版対応を意味しません。末尾のGo API major suffixはmajor 2から付けます。Go API major 2のtagに余分な `/v2/` を挿入しません。

各 `go.mod` は自分自身のリリース番号を宣言せず、Git tagがその版を識別します。RC-013とRC-016は独立に版を上げられます。公開済みのtagの付け替えや、同じ版への内容差し替えは行わず、新しい版を発行してください。

## 公開前の候補検証

リポジトリのルートで[READMEのローカル検証](../README.md#開発と公開前の検証)をすべて実行します。workspaceのtest／vetは `GOPROXY=off go test -count=1 ./rc013v1/... ./rc016v1/... ./rc016v2/...` と `GOPROXY=off go vet ./rc013v1/... ./rc016v1/... ./rc016v2/...` で実行し、依存メタデータを公開サービスから取得することを避けます。`GOWORK=off go run ./scripts/verify-modules.go` の独立検証も必要で、runnerは自身の一時file proxyを設定します。このオフライン指定は候補のworkspace検証だけに適用し、以下の実公開取得には適用しません。

runnerは候補ソースのZIPを一時file proxyに作り、独立stage・外部consumer・空cacheで検査します。このZIP、checksum、cacheを本番へ流用しないでください。候補のファイル名・内容が実際の公開ZIPと異なればchecksumも異なります。`GOSUMDB=off` は既存checksumとの不一致を無視する設定ではありません。

製品の `go.mod` にlocal `replace` がないこと、公開物に私有資料や検証用一時ファイルが混ざらないこと、全3モジュールのLICENSEが[ルートのMIT本文](../LICENSE)と完全一致することを確認します。ルートと同じLICENSEを各モジュール内へ明示コピーする方針であり、GoのルートLICENSE自動継承の実証に依存しません。RC-016の実公開版由来の `go.sum` は管理対象です。

## 初回公開で実施した順序

1. RC-013を先に確定し、`rc013v1/v0.1.0` tagを公開しました。下記の公開consumer検証をRC-013に実行し、公開proxyとsumdbを通じて取得できることを確認しました。
2. RC-013取得に成功した後、下記のRC-016依存確定を両モジュールに実行しました。requireが実公開されたRC-013 `v0.1.0` であることを確認し、実公開ZIP由来の `go.sum` を生成しました。公開前のRC-016候補にもconsumer fixtureを適用しました。取得できない場合はRC-016の公開へ進まない方針です。
3. test/vetに成功したRC-016のrequireと本物のgo.sumを別のリリースcommitに含め、そのcommitに `rc016v1/v0.1.0` と `rc016v2/v0.1.0` tagを付けて公開しました。RC-013と同じcommitを指す必要はありません。
4. 各RC-016にも下記の公開consumer検証を実行しました。RC-013の推移的依存、型の同一性、v1の3系統とv2の自転車・歩行者の操作、配布LICENSEを確認し、全3モジュールのGitHub Releaseを公開しました。

v1とv2はどちらもRC-013を直接依存とし、相互の公開順序に制約はありません。各規格モジュールのGo API SemVerも独立です。

将来のRC-016更新も、workspaceの兄弟ソースだけではなく実公開されたrequire版で検証します。新しいRC-013 APIが必要ならRC-013を先に公開し、その後RC-016のrequireを明示更新してください。

## 公開consumerの検証（公開後のみ）

Bash、Go 1.27.0、Python 3、公開tagを持つローカルGit checkoutを使用します。以下をリポジトリのルートから同じBashセッションに読み込みます。環境変数は関数内のsubshellだけに適用し、ユーザーのGo設定を永続変更しません。

```bash
release_repo_dir=$(pwd -P)
public_consumer() (
    set -euo pipefail
    release_module=$1
    release_fixture=$2
    release_temp=$(mktemp -d)
    trap '
        release_status=$?
        if ! chmod -R u+w "$release_temp" || ! rm -rf "$release_temp"; then
            printf "%s\n" "Temporary release directory cleanup error: $release_temp" >&2
            if [ "$release_status" -eq 0 ]; then release_status=1; fi
        fi
        exit "$release_status"
    ' EXIT
    export GOWORK=off GOENV=off GOFLAGS= GOTOOLCHAIN=local
    export GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org
    export GOPRIVATE= GONOPROXY= GONOSUMDB=
    export GOPATH="$release_temp/gopath" GOMODCACHE="$release_temp/modcache"
    export GOCACHE="$release_temp/buildcache"
    mkdir "$release_temp/consumer"
    cp "$release_repo_dir/scripts/testdata/$release_fixture/consumer_test.go" \
        "$release_temp/consumer/consumer_test.go"
    cd "$release_temp/consumer"
    go mod init example.com/its-forum-consumer
    go get "github.com/hareku/its-forum-go/$release_module@v0.1.0"
    go mod tidy
    go test -mod=readonly -count=1 ./...
    go vet -mod=readonly ./...
    go mod verify
    go list -mod=readonly -m -json all > "$release_temp/graph.json"
    python3 - "$release_repo_dir" "$release_module" "$release_temp/graph.json" <<'PY'
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import zipfile

repo, selected, graph_file = sys.argv[1:]
base = 'github.com/hareku/its-forum-go/'
expected = {base + 'rc013v1': 'v0.1.0'}
if selected in ('rc016v1', 'rc016v2'):
    expected[base + selected] = 'v0.1.0'
raw = Path(graph_file).read_text()
decoder = json.JSONDecoder()
modules = []
while raw.strip():
    item, end = decoder.raw_decode(raw.lstrip())
    modules.append(item)
    raw = raw.lstrip()[end:]
assert all(not m.get('Replace') for m in modules), modules
main = [m for m in modules if m.get('Main')]
assert len(main) == 1 and main[0]['Path'] == 'example.com/its-forum-consumer'
assert Path(main[0]['Dir']).resolve() == Path.cwd().resolve()
deps = [m for m in modules if not m.get('Main')]
assert {m['Path']: m['Version'] for m in deps} == expected, modules
cache = Path(os.environ['GOMODCACHE']).resolve()
for m in deps:
    location = Path(m['Dir']).resolve()
    assert cache in location.parents, location
    info = json.loads(subprocess.check_output([
        'go', 'mod', 'download', '-json', m['Path'] + '@' + m['Version']
    ]))
    module_dir = m['Path'].removeprefix(base)
    tag = module_dir + '/' + m['Version']
    root_license = subprocess.check_output(['git', '-C', repo, 'show', tag + ':LICENSE'])
    module_license = subprocess.check_output([
        'git', '-C', repo, 'show', tag + ':' + module_dir + '/LICENSE'
    ])
    with zipfile.ZipFile(info['Zip']) as archive:
        distributed = archive.read(m['Path'] + '@' + m['Version'] + '/LICENSE')
    assert distributed == module_license == root_license, tag
    print('PUBLIC_FETCH', m['Path'], m['Version'], 'LICENSE SHA256',
          hashlib.sha256(distributed).hexdigest())
PY
)
```

RC-013のtag公開後、RC-016公開前に実行します。

```bash
public_consumer rc013v1 consumer-rc013
```

RC-016のtag公開後、別の空cache・一時directoryで実行します。

```bash
public_consumer rc016v1 consumer-rc016
public_consumer rc016v2 consumer-rc016v2
```

RC-016用fixtureはRC-013も直接importするため、初回の依存解決ではRC-013のdirect require追加を許容します。その後はreadonly検証と正確なgraph検査を行います。各ZIPのLICENSEは作業中のソースではなく、取得した版のGit tagにあるルート・モジュールLICENSEと照合します。tagを作成しただけでは取得検証の成功とは扱いません。

## RC-016の依存確定（RC-013公開後、RC-016公開前のみ）

次はリポジトリのルートで実行します。この工程だけはリリース対象の `rc016v1` または `rc016v2` の `go.mod` / `go.sum` を更新します。Go 1.27.0を用い、RC-013公開consumer検証の成功後に進んでください。

```bash
(
    set -euo pipefail
    release_repo_dir=$(pwd -P)
    release_module=rc016v2 # Set rc016v1 for the RC-016 1.0 release.
    case "$release_module" in
        rc016v1) release_fixture=consumer-rc016 ;;
        rc016v2) release_fixture=consumer-rc016v2 ;;
        *) exit 1 ;;
    esac
    release_temp=$(mktemp -d)
    trap '
        release_status=$?
        if ! chmod -R u+w "$release_temp" || ! rm -rf "$release_temp"; then
            printf "%s\n" "Temporary release directory cleanup error: $release_temp" >&2
            if [ "$release_status" -eq 0 ]; then release_status=1; fi
        fi
        exit "$release_status"
    ' EXIT
    export GOWORK=off GOENV=off GOFLAGS= GOTOOLCHAIN=local
    export GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org
    export GOPRIVATE= GONOPROXY= GONOSUMDB=
    export GOPATH="$release_temp/gopath" GOMODCACHE="$release_temp/modcache"
    export GOCACHE="$release_temp/buildcache"
    cd "$release_module"
    go mod edit -json > "$release_temp/module.json"
    python3 - "$release_temp/module.json" "$release_module" <<'PY'
import json
import sys
m = json.load(open(sys.argv[1]))
assert m['Module']['Path'] == 'github.com/hareku/its-forum-go/' + sys.argv[2]
assert not m.get('Replace')
assert [(r['Path'], r['Version']) for r in m['Require']] == [
    ('github.com/hareku/its-forum-go/rc013v1', 'v0.1.0')
]
PY
    go mod tidy
    go test -mod=readonly -count=1 ./...
    go vet -mod=readonly ./...
    go test -mod=readonly -count=1 \
        "$release_repo_dir/scripts/testdata/$release_fixture/consumer_test.go"
    go mod verify
    go list -mod=readonly -m -json all
)
```

この段階のconsumer fixtureはRC-016候補を現在の主モジュールとして読み、RC-013は公開proxyから取得します。RC-016自体の公開取得はtag公開後に別の外部consumerで検証します。

生成した差分を確認し、本物のgo.sumをRC-016の公開内容へ含めます。公開前runnerが作るfixture用checksumで置き換えないでください。公開後は前節のRC-016外部consumerでも確認します。

## 新しい規格・モジュールを追加するとき

- 実装する規格と完全な版番号を確定し、独立directoryと `go.mod`、package、テストを追加します。規格majorとGo API majorを区別します。
- ルートの `go.work` に追加し、`scripts/verify-modules.go` のモジュール一覧、ZIP内容、依存graph、負の検証を更新します。各モジュールの内部bitioは現行方針の同一性検査を維持します。
- `scripts/testdata` に独立consumerを追加し、workspace外・replaceなし・空cacheで配布物を検証します。
- CIのformat・workspace test/vet対象と独立runner対象を追加し、ローカルでも同じコマンドを通します。
- ルートREADMEの対応表・検証手順、各モジュールのREADME・Examples・LICENSEコピー、本書の公開consumer検証を更新します。
- モジュール別のtag prefix・独立SemVerと依存先から先に公開する順序を決め、循環依存を避けます。実装しない将来規格の空モジュールは追加しません。

版管理の根拠はGo公式の[複数モジュール](https://go.dev/doc/modules/managing-source#sourcing-multiple-modules-in-a-single-repository)、[major suffix](https://go.dev/ref/mod#major-version-suffixes)、[tagとdirectory](https://go.dev/ref/mod#vcs-version)、[LICENSEの配布](https://go.dev/ref/mod#vcs-license)を参照してください。
