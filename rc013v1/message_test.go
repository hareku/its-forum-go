package rc013_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/hareku/its-forum-go/rc013v1"
)

// RC-013 1.1 tables 5-1..5-14, sections 6.1..6.13. Independently calculated
// field arithmetic and provenance: .omx/plans/rc013-field-matrix.md.
// Common data is 54 bytes; free header is seven bytes, payload is six bytes.
const fullHex = `29 01 02 03 04 05 36 fd
89 0a 0b b8
ff ff ff ff 00 00 00 02 ff ff ab
00 01 1c 20 ff ff 29 af ff
21 19 00 c8
08 9c
01 02 1c 20
c5 9a
ff fe aa 64 81 1b e4
23 22 ff ff ff fe 00 00 00 03
a1
3a e1 01 02 e2 05 01
99 aa bb 77 88 cc`

const minimumHex = `29 01 02 03 04 05 1c 00
89 0a 0b b8
ff ff ff ff 00 00 00 02 ff ff ab
00 01 1c 20 ff ff 29 af ff
21 19 00 c8`

func hexBytes(t testing.TB, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.Join(strings.Fields(s), ""))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func expectedCommon() rc013.CommonData {
	return rc013.CommonData{
		Time:                        rc013.Time{LeapSecondCorrection: true, Hour: 9, Minute: 10, Millisecond: 3000},
		Position:                    rc013.Position{Latitude: -1, Longitude: 2, Elevation: 0xffff, PositionConfidence: 10, ElevationConfidence: 11},
		VehicleState:                rc013.VehicleState{Speed: 1, Heading: 7200, Acceleration: -1, SpeedConfidence: 1, HeadingConfidence: 2, AccelerationConfidence: 3, Transmission: 2, SteeringWheelAngle: -1},
		VehicleAttributes:           rc013.VehicleAttributes{SizeClass: 2, RoleClass: 1, Width: 100, Length: 200},
		PositionOptional:            &rc013.PositionOptional{Delay: 1, Revision: 2, RoadFacilities: 3, RoadClass: 4},
		GPSStatusOptional:           &rc013.GPSStatusOptional{SemiMajorAxis: 1, SemiMinorAxis: 2, Orientation: 7200},
		PositionAcquisitionOptional: &rc013.PositionAcquisitionOptional{Mode: 3, PDOP: 5, Satellites: 9, Multipath: 2, DeadReckoning: true, MapMatching: false},
		VehicleStateOptional:        &rc013.VehicleStateOptional{YawRate: -2, Brakes: 42, AuxiliaryBrakes: 2, Throttle: 100, Lights: 0x81, ACC: 0, CACC: 1, PCS: 2, ABS: 3, TRC: 3, ESC: 2, LKA: 1, LDW: 0},
		Intersection:                &rc013.Intersection{DistanceSource: 1, Distance: 100, PositionSource: 2, Latitude: -2, Longitude: 3},
		Extension:                   &rc013.Extension{Upper: 10, Status: 1},
	}
}
func expectedMessage() rc013.Message {
	return rc013.Message{
		Header: rc013.Header{ServiceID: 1, MessageID: 1, Version: 1, VehicleID: 0x01020304, Counter: 5}, Common: expectedCommon(),
		Free: &rc013.FreeArea{Entries: []rc013.FreeEntry{{ServiceID: 0xe1, Gap: []byte{0x99}, Data: []byte{0xaa, 0xbb}}, {ServiceID: 0xe2, Gap: []byte{0x77, 0x88}, Data: []byte{0xcc}}}},
	}
}
func minimalMessage() rc013.Message {
	m := expectedMessage()
	m.Free = nil
	m.Common.PositionOptional = nil
	m.Common.GPSStatusOptional = nil
	m.Common.PositionAcquisitionOptional = nil
	m.Common.VehicleStateOptional = nil
	m.Common.Intersection = nil
	m.Common.Extension = nil
	return m
}

func TestIndependentFullPacket(t *testing.T) {
	literal := hexBytes(t, fullHex)
	m, err := rc013.Decode(literal)
	if err != nil {
		t.Fatal(err)
	}
	wantHeader := rc013.Header{ServiceID: 1, MessageID: 1, Version: 1, VehicleID: 0x01020304, Counter: 5, DataLength: 54, OptionFlags: 0xfd}
	if m.Header != wantHeader {
		t.Fatalf("header %+v", m.Header)
	}
	// All mandatory and optional fields are compared to independent typed values.
	if !reflect.DeepEqual(m.Common, expectedCommon()) {
		t.Fatalf("common fields differ:\ngot %#v\nwant %#v", m.Common, expectedCommon())
	}
	for i, e := range m.Free.Entries {
		want := expectedMessage().Free.Entries[i]
		if e.ServiceID != want.ServiceID || !bytes.Equal(e.Gap, want.Gap) || !bytes.Equal(e.Data, want.Data) {
			t.Fatalf("free entry %d: %#v", i, e)
		}
	}
	encoded, err := expectedMessage().MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encoded, literal) {
		t.Fatalf("independent encode %x want%x", encoded, literal)
	}
	again, err := m.MarshalBinary()
	if err != nil || !bytes.Equal(again, literal) {
		t.Fatalf("preservation %x %v", again, err)
	}
	if issues := m.Validate(); len(issues) != 2 {
		t.Fatalf("expected reserved light and upper-nibble issues, got%+v", issues)
	}
}

func TestMinimumAndEveryShorterPrefix(t *testing.T) {
	literal := hexBytes(t, minimumHex)
	encoded, err := minimalMessage().MarshalBinary()
	if err != nil || !bytes.Equal(encoded, literal) {
		t.Fatalf("minimal encode %x %v", encoded, err)
	}
	m, err := rc013.Decode(literal)
	if err != nil {
		t.Fatal(err)
	}
	if m.Free != nil || m.Common.OptionFlags() != 0 {
		t.Fatal("absent areas appeared")
	}
	for _, s := range []string{minimumHex, fullHex} {
		data := hexBytes(t, s)
		for n := 0; n < len(data); n++ {
			if _, err := rc013.Decode(data[:n]); err == nil {
				t.Fatalf("accepted prefix %d/%d", n, len(data))
			}
		}
	}
}

func TestIndependentLimitsAndFreePrefix(t *testing.T) {
	// 8+28+4+60=100 bytes. The literal prefix is independent of the encoder.
	for _, tc := range []struct {
		name, descriptor string
		gap              int
		payload          []byte
	}{{"length60", "21 e1 00 3c", 0, bytes.Repeat([]byte{0xa5}, 60)}, {"offset59", "21 e1 3b 01", 59, []byte{0xa5}}} {
		t.Run(tc.name, func(t *testing.T) {
			literal := hexBytes(t, minimumHex)
			literal[7] = 1
			literal = append(literal, hexBytes(t, tc.descriptor)...)
			literal = append(literal, bytes.Repeat([]byte{0xa5}, 60)...)
			m := minimalMessage()
			m.Free = &rc013.FreeArea{Entries: []rc013.FreeEntry{{ServiceID: 0xe1, Gap: bytes.Repeat([]byte{0xa5}, tc.gap), Data: tc.payload}}}
			out, err := m.MarshalBinary()
			if err != nil || !bytes.Equal(out, literal) {
				t.Fatalf("boundary encode%x %v", out, err)
			}
			if _, err := rc013.Decode(literal); err != nil {
				t.Fatal(err)
			}
			if _, err := rc013.Decode(append(literal, 0)); !errors.Is(err, rc013.ErrRange) {
				t.Fatal("101-byte input accepted", err)
			}
		})
	}
	literal := hexBytes(t, `b7 01 00 01 02 01 01 03 02 01 04 03 01 05 04 01 06 05 01 07 06 01 a1 a2 a3 a4 a5 a6 a7`)
	entries := make([]rc013.FreeEntry, 7)
	for i := range entries {
		entries[i] = rc013.FreeEntry{ServiceID: uint8(i + 1), Data: []byte{byte(0xa1 + i)}}
	}
	out, err := (rc013.FreeArea{Entries: entries}).MarshalBinary()
	if err != nil || !bytes.Equal(out, literal) {
		t.Fatalf("seven-entry encode %x %v", out, err)
	}
	extra := append(bytes.Clone(literal), 0xde, 0xad)
	area, n, err := rc013.DecodeFreeArea(extra)
	if err != nil || n != 29 || len(area.Entries) != 7 {
		t.Fatalf("prefix %d %v", n, err)
	}
	if out, err := (rc013.FreeArea{Entries: append(entries, entries[0])}).MarshalBinary(); out != nil || !errors.Is(err, rc013.ErrRange) {
		t.Fatal("eight entries", err)
	}
	if _, _, err := rc013.DecodeFreeArea([]byte{8}); !errors.Is(err, rc013.ErrMalformed) {
		t.Fatal("present zero count", err)
	}
	// The helper can consume beyond standalone's 100-byte packet limit.
	wide := rc013.FreeArea{Entries: []rc013.FreeEntry{{Gap: make([]byte, 59), Data: make([]byte, 60)}}}
	out, err = wide.MarshalBinary()
	if err != nil || len(out) != 123 {
		t.Fatal(len(out), err)
	}
	_, n, err = rc013.DecodeFreeArea(out)
	if err != nil || n != 123 {
		t.Fatal(n, err)
	}
}

func TestKnownEditsAndDerivedFields(t *testing.T) {
	literal := hexBytes(t, fullHex)
	m, err := rc013.Decode(literal)
	if err != nil {
		t.Fatal(err)
	}
	m.Common.VehicleState.SteeringWheelAngle = 0
	m.Header.DataLength = 0
	m.Header.OptionFlags = 0
	want := bytes.Clone(literal)
	want[30] &= 0xf0
	want[31] = 0
	out, err := m.MarshalBinary()
	if err != nil || !bytes.Equal(out, want) {
		t.Fatalf("edit %x %v want%x", out, err, want)
	}
	m = minimalMessage()
	m.Common.PositionOptional = &rc013.PositionOptional{}
	out, err = m.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 38 || out[6] != 30 || out[7] != 0x80 || out[36] != 0 || out[37] != 0 {
		t.Fatalf("presentzero%x", out)
	}
	decoded, err := rc013.Decode(out)
	if err != nil || decoded.Common.PositionOptional == nil {
		t.Fatal("zero optional vanished", err)
	}
	decoded.Common.PositionOptional = nil
	out, err = decoded.MarshalBinary()
	if err != nil || !bytes.Equal(out, hexBytes(t, minimumHex)) {
		t.Fatal("optional removal", err)
	}
}

func TestOpaqueAnchorsAndSafeResize(t *testing.T) {
	literal := hexBytes(t, fullHex)
	m, err := rc013.Decode(literal)
	if err != nil {
		t.Fatal(err)
	}
	m.Free.Entries[1].Data = append(m.Free.Entries[1].Data, 0xdd)
	out, err := m.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	want := append(bytes.Clone(literal), 0xdd)
	want[68] = 2
	if !bytes.Equal(out, want) {
		t.Fatalf("safe resize %x want%x", out, want)
	}
	m.Free.Entries[0].Data = append(m.Free.Entries[0].Data, 0xee)
	if out, err := m.MarshalBinary(); out != nil || !errors.Is(err, rc013.ErrLayout) {
		t.Fatal("unsafe prior payload move", err)
	}
	m, err = rc013.Decode(literal)
	if err != nil {
		t.Fatal(err)
	}
	m.Common.PositionOptional = nil
	if out, err := m.MarshalBinary(); out != nil || !errors.Is(err, rc013.ErrLayout) {
		t.Fatal("unsafe outer free move", err)
	}
	// Reconstructing the free region explicitly opts into a new placement.
	m.Free = &rc013.FreeArea{Entries: m.Free.Entries}
	if _, err := m.MarshalBinary(); err != nil {
		t.Fatal("explicit reconstruction", err)
	}
	m = minimalMessage()
	m.Common.ExtendedOptions = true
	m.Common.OpaqueTail = []byte{0xde, 0xad}
	out, err = m.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := rc013.Decode(out)
	if err != nil {
		t.Fatal(err)
	}
	decoded.Common.Time.Hour = 8
	if _, err := decoded.MarshalBinary(); err != nil {
		t.Fatal("same-width opaque edit", err)
	}
	decoded.Common.PositionOptional = &rc013.PositionOptional{}
	if _, err := decoded.MarshalBinary(); !errors.Is(err, rc013.ErrLayout) {
		t.Fatal("unknown common moved", err)
	}
}

func TestInputAndOutputOwnership(t *testing.T) {
	source := hexBytes(t, fullHex)
	m, err := rc013.Decode(source)
	if err != nil {
		t.Fatal(err)
	}
	for i := range source {
		source[i] = 0
	}
	out, err := m.MarshalBinary()
	if err != nil || !bytes.Equal(out, hexBytes(t, fullHex)) {
		t.Fatal("input alias", err)
	}
	for i := range out {
		out[i] = 0
	}
	again, err := m.MarshalBinary()
	if err != nil || !bytes.Equal(again, hexBytes(t, fullHex)) {
		t.Fatal("output alias", err)
	}
	source = append(hexBytes(t, minimumHex), 0xde)
	source[6] = 29
	source[7] = 2
	m, err = rc013.Decode(source)
	if err != nil {
		t.Fatal(err)
	}
	source[36] = 0
	if m.Common.OpaqueTail[0] != 0xde {
		t.Fatal("tail alias")
	}
}

func TestMalformedPackets(t *testing.T) {
	for _, tc := range []struct {
		name     string
		mutate   func([]byte) []byte
		category error
	}{
		{"unsupported", func(b []byte) []byte { b[0] = 0; return b }, rc013.ErrUnsupported},
		{"short common", func(b []byte) []byte { b[6] = 27; return b }, rc013.ErrMalformed},
		{"long common", func(b []byte) []byte { b[6] = 255; return b }, rc013.ErrTruncated},
		{"absent free trailing", func(b []byte) []byte { b[7] &^= 1; return b }, rc013.ErrMalformed},
		{"missing optional", func(b []byte) []byte { b[7] &^= 0x80; return b }, rc013.ErrMalformed},
		{"bad free header", func(b []byte) []byte { b[62] = 0x32; return b }, rc013.ErrMalformed},
		{"overlap", func(b []byte) []byte { b[67] = 2; return b }, rc013.ErrMalformed},
		{"address60", func(b []byte) []byte { b[64] = 60; return b }, rc013.ErrMalformed},
		{"zero length", func(b []byte) []byte { b[65] = 0; return b }, rc013.ErrMalformed},
		{"span outside", func(b []byte) []byte { b[68] = 60; return b }, rc013.ErrTruncated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := rc013.Decode(tc.mutate(hexBytes(t, fullHex)))
			if !errors.Is(err, tc.category) {
				t.Fatalf("got %v want%v", err, tc.category)
			}
			var ce *rc013.Error
			if !errors.As(err, &ce) || ce.Field == "" {
				t.Fatal("missing structural context")
			}
		})
	}
	m := minimalMessage()
	m.Common.OpaqueTail = []byte{1}
	if _, err := m.MarshalBinary(); !errors.Is(err, rc013.ErrMalformed) {
		t.Fatal(err)
	}
	m.Common.OpaqueTail = nil
	m.Common.ExtendedOptions = true
	if _, err := m.MarshalBinary(); !errors.Is(err, rc013.ErrMalformed) {
		t.Fatal(err)
	}
	b := hexBytes(t, minimumHex)
	b[7] = 2
	if _, err := rc013.Decode(b); !errors.Is(err, rc013.ErrTruncated) {
		t.Fatal(err)
	}
	m = minimalMessage()
	m.Free = &rc013.FreeArea{Entries: []rc013.FreeEntry{{Data: make([]byte, 61)}}}
	if _, err := m.MarshalBinary(); !errors.Is(err, rc013.ErrRange) {
		t.Fatal(err)
	}
}

func FuzzMessage(f *testing.F) {
	f.Add(hexBytes(f, fullHex))
	f.Add(hexBytes(f, minimumHex))
	f.Add([]byte{})
	f.Add([]byte{0xff})
	extended := append(hexBytes(f, minimumHex), 0xde, 0xad)
	extended[6], extended[7] = 30, 2
	f.Add(extended)
	presentZero := append(hexBytes(f, minimumHex), 0, 0)
	presentZero[6], presentZero[7] = 30, 0x80
	f.Add(presentZero)
	f.Add(hexBytes(f, fullHex)[:62])
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1024 {
			return
		}
		m, err := rc013.Decode(data)
		if err != nil {
			return
		}
		out, err := m.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(out, data) {
			t.Fatalf("changed %x to%x", data, out)
		}
		again, err := rc013.Decode(out)
		if err != nil {
			t.Fatal(err)
		}
		out2, err := again.MarshalBinary()
		if err != nil || !bytes.Equal(out2, out) {
			t.Fatal("unstable", err)
		}
	})
}

func TestFreeDuplicateIDsAndIndependentFlags(t *testing.T) {
	// IDs identify services, not unique keys; both descriptors may select one service.
	literal := hexBytes(t, "3a e1 00 01 e1 01 01 aa bb")
	area, n, err := rc013.DecodeFreeArea(literal)
	if err != nil || n != len(literal) {
		t.Fatal(n, err)
	}
	out, err := area.MarshalBinary()
	if err != nil || !bytes.Equal(out, literal) {
		t.Fatal("duplicate IDs changed", err)
	}
	common := hexBytes(t, minimumHex)[8:]
	for _, tc := range []struct {
		mask uint8
		size int
	}{{0x80, 2}, {0x40, 4}, {0x20, 2}, {0x10, 7}, {0x08, 10}, {0x04, 1}} {
		input := append(bytes.Clone(common), make([]byte, tc.size)...)
		decoded, consumed, err := rc013.DecodeCommonData(input, tc.mask|rc013.OptionFree)
		if err != nil || consumed != len(input) || decoded.OptionFlags() != tc.mask {
			t.Fatalf("flag%x consumed%d error%v", tc.mask, consumed, err)
		}
		out, err := decoded.MarshalBinary()
		if err != nil || !bytes.Equal(out, input) {
			t.Fatalf("flag%x changed", tc.mask)
		}
	}
	// Adding an opaque entry changes the descriptor table and moves prior payloads.
	area.Entries = append(area.Entries, rc013.FreeEntry{Data: []byte{1}})
	if _, err := area.MarshalBinary(); !errors.Is(err, rc013.ErrLayout) {
		t.Fatal("changed opaque header length", err)
	}
}

func TestNestedDiagnosticOffsets(t *testing.T) {
	assertOffset := func(err error, offset int, field string, category error) {
		t.Helper()
		var located *rc013.Error
		if !errors.As(err, &located) || located.Offset != offset || located.Field != field || !errors.Is(err, category) {
			t.Fatalf("error%v want%s at%d (%v)", err, field, offset, category)
		}
	}
	malformed := hexBytes(t, fullHex)
	malformed[62] = 0x32
	_, err := rc013.Decode(malformed)
	assertOffset(err, 62, "free.header", rc013.ErrMalformed)
	missing := hexBytes(t, minimumHex)
	missing[7] = 0x80
	_, err = rc013.Decode(missing)
	assertOffset(err, 36, "common", rc013.ErrTruncated)
	m := minimalMessage()
	m.Common.Position.PositionConfidence = 16
	_, err = m.Common.Position.MarshalBinary()
	assertOffset(err, 10, "Position.PositionConfidence", rc013.ErrRange)
	_, err = m.Common.MarshalBinary()
	assertOffset(err, 14, "Position.PositionConfidence", rc013.ErrRange)
	_, err = m.MarshalBinary()
	assertOffset(err, 22, "Position.PositionConfidence", rc013.ErrRange)
	m, err = rc013.Decode(hexBytes(t, fullHex))
	if err != nil {
		t.Fatal(err)
	}
	m.Free.Entries[0].Data = append(m.Free.Entries[0].Data, 0)
	_, err = m.Free.MarshalBinary()
	assertOffset(err, 13, "free.payload", rc013.ErrLayout)
	_, err = m.MarshalBinary()
	assertOffset(err, 75, "free.payload", rc013.ErrLayout)
	m = minimalMessage()
	m.Free = &rc013.FreeArea{}
	_, err = m.MarshalBinary()
	assertOffset(err, -1, "free.entries", rc013.ErrRange)
}

func TestFreeEntryReorderAnchors(t *testing.T) {
	for _, literal := range []string{"3a110001220101aabb", "3a110001110101aabb"} {
		data := hexBytes(t, literal)
		area, _, err := rc013.DecodeFreeArea(data)
		if err != nil {
			t.Fatal(err)
		}
		area.Entries[0], area.Entries[1] = area.Entries[1], area.Entries[0]
		if out, err := area.MarshalBinary(); out != nil || !errors.Is(err, rc013.ErrLayout) {
			t.Fatalf("swapped entries accepted: %x %v", out, err)
		}
		rebuilt := rc013.FreeArea{Entries: area.Entries}
		out, err := rebuilt.MarshalBinary()
		if err != nil || !bytes.Equal(out[len(out)-2:], []byte{0xbb, 0xaa}) {
			t.Fatalf("explicit reconstruction: %x %v", out, err)
		}
		packet := hexBytes(t, minimumHex)
		packet[7] |= 1
		packet = append(packet, data...)
		m, err := rc013.Decode(packet)
		if err != nil {
			t.Fatal(err)
		}
		m.Free.Entries[0], m.Free.Entries[1] = m.Free.Entries[1], m.Free.Entries[0]
		if out, err := m.MarshalBinary(); out != nil || !errors.Is(err, rc013.ErrLayout) {
			t.Fatalf("whole swapped entries accepted: %x %v", out, err)
		}
		m.Free = &rc013.FreeArea{Entries: m.Free.Entries}
		if _, err := m.MarshalBinary(); err != nil {
			t.Fatalf("whole explicit reconstruction: %v", err)
		}
	}
}
