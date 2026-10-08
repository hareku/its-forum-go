package rc016

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/hareku/its-forum-go/rc013v1"
	"testing"
)

const roadsideGolden = "b12a123401020304890a12340027a55a00000109112233445524008000000000000001ffffffff80001200640050ff9c2ff80045014014"
const roadsideProfileGolden = "b12a123401020304890a1234003ba55a00800d020500000180000500000280010109112233445524018000000000000001ffffffff80001200640050ff9c2ff8004501401421910002b52b"

func roadsideHex(t testing.TB, s string) []byte {
	t.Helper()
	b, e := hex.DecodeString(s)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func roadsideFixture() RoadsideFields {
	return RoadsideFields{Header: RoadsideHeader{ServiceID: 5, Operation: 1, Version: 1, Counter: 42, MessageID: 0x1234, RoadsideID: 0x01020304, Time: rc013.Time{LeapSecondCorrection: true, Hour: 9, Minute: 10, Millisecond: 0x1234}, Reserved: 0xa55a}, Groups: []RoadsideObjectGroup{{SensorIndex: -1, Objects: []RoadsideObject{{MessageID: 1, Version: 1, ID: 0x11223344, Counter: 0x55, Common: rc013.CommonData{Time: rc013.Time{LeapSecondCorrection: true}, Position: rc013.Position{Latitude: 1, Longitude: -1, Elevation: 0x8000, PositionConfidence: 1, ElevationConfidence: 2}, VehicleState: rc013.VehicleState{Speed: 100, Heading: 80, Acceleration: -100, SpeedConfidence: 1, HeadingConfidence: 3, AccelerationConfidence: 7, Transmission: 7, SteeringWheelAngle: -2048}, VehicleAttributes: rc013.VehicleAttributes{SizeClass: 4, RoleClass: 5, Width: 5, Length: 20}}}}}}}
}
func roadsideProfile() RoadsideProfile {
	return RoadsideProfile{Sensors: []SensorProfile{{Layout: SensorExample}, {Layout: SensorExample}}, GroupOrder: []int{1, 0}, ObjectExtensions: ObjectExtensionProfile{Layouts: map[uint8]ObjectExtensionLayout{0x91: {Fields: []ObjectExtensionField{ObjectLevel, ObjectFusion, ObjectFusionSources, ObjectAssistType}, Complete: true}}}}
}
func roadsideProfileFixture() RoadsideFields {
	f := roadsideFixture()
	f.Options = []RoadsideOption{{Index: 0, Sensors: []RoadsideSensor{{Example: &RoadsideSensorAttributes{ID: 1, Operation: 1}}, {Example: &RoadsideSensorAttributes{ID: 2, Operation: 1, Status: 1}}}}}
	f.Groups[0].SensorIndex = 0
	f.Groups[0].Objects[0].Extensions = []RoadsideObjectExtension{{ServiceID: 0x91, Data: []byte{0, 0x0b}, Fields: &ObjectExtensionFields{Level: 5, Fusion: 2, FusionSources: 5, AssistType: 2}}}
	return f
}
func TestRoadsideGolden(t *testing.T) {
	b := roadsideHex(t, roadsideGolden)
	m, e := DecodeRoadside(b)
	if e != nil {
		t.Fatal(e)
	}
	if !m.ObjectsDecoded() || m.Header.MessageSize != 39 || len(m.Groups) != 1 || len(m.Groups[0].Objects) != 1 {
		t.Fatalf("metadata: %+v", m)
	}
	o := m.Groups[0].Objects[0]
	if o.ID != 0x11223344 || o.Common.Position.Longitude != -1 || o.Common.VehicleState.SteeringWheelAngle != -2048 || o.Common.VehicleAttributes.Width != 5 {
		t.Fatalf("object: %+v", o)
	}
	built, e := NewRoadside(roadsideFixture(), RoadsideProfile{})
	if e != nil {
		t.Fatal(e)
	}
	got, e := built.MarshalBinary()
	if e != nil || !bytes.Equal(got, b) {
		t.Fatalf("independent encode: %x %v", got, e)
	}
	m.Groups[0].Objects[0].Common.VehicleAttributes.Width = 6
	got, e = m.MarshalBinary()
	if e != nil {
		t.Fatal(e)
	}
	expected := bytes.Clone(b)
	expected[53] = 0x80
	if !bytes.Equal(got, expected) {
		t.Fatalf("width delta: %x", got)
	}
	for n := 0; n < len(b); n++ {
		if _, e := DecodeRoadside(b[:n]); e == nil {
			t.Fatalf("prefix %d accepted", n)
		}
	}
}
func TestRoadsideCountsAndOptional(t *testing.T) {
	for _, n := range []int{0, 1, 255} {
		f := roadsideFixture()
		o := f.Groups[0].Objects[0]
		f.Groups[0].Objects = make([]RoadsideObject, n)
		for i := range f.Groups[0].Objects {
			f.Groups[0].Objects[i] = o
		}
		m, e := NewRoadside(f, RoadsideProfile{})
		if e != nil {
			t.Fatalf("count%d: %v", n, e)
		}
		b, e := m.MarshalBinary()
		if e != nil || len(b) != 19+36*n {
			t.Fatalf("count%d: %v len%d", n, e, len(b))
		}
		if len(m.Groups[0].Objects) != n {
			t.Fatal("count")
		}
	}
	f := roadsideFixture()
	f.Groups[0].Objects = make([]RoadsideObject, 256)
	if _, e := NewRoadside(f, RoadsideProfile{}); !errors.Is(e, ErrRange) {
		t.Fatal(e)
	}
	f = roadsideFixture()
	f.Groups[0].Objects[0].Common.PositionOptional = &rc013.PositionOptional{Delay: 1}
	m, e := NewRoadside(f, RoadsideProfile{})
	if e != nil {
		t.Fatal(e)
	}
	b, e := m.MarshalBinary()
	if e != nil || b[25] != 38 || b[26] != 0x80 || m.Groups[0].Objects[0].Common.PositionOptional.Delay != 1 {
		t.Fatalf("common length inclusion: %x %v", b, e)
	}
}
func TestRoadsideUnknownAndOwnership(t *testing.T) {
	for _, body := range [][]byte{{1}, {7, 0xde, 0xad}, {0, 0x81, 0xff, 0x5a}} {
		b := append(roadsideHex(t, roadsideGolden)[:16], body...)
		b[12] = 0
		b[13] = byte(len(body))
		m, e := DecodeRoadside(b)
		if e != nil {
			t.Fatal(e)
		}
		got, e := m.MarshalBinary()
		if e != nil || !bytes.Equal(got, b) {
			t.Fatal(e)
		}
		if body[0] != 1 && m.ObjectsDecoded() {
			t.Fatal("unknown marked decoded")
		}
		b[len(b)-1] ^= 1
		next, e := m.MarshalBinary()
		if e != nil || !bytes.Equal(next, got) {
			t.Fatal("input alias")
		}
		got[0] ^= 1
		next2, _ := m.MarshalBinary()
		if !bytes.Equal(next2, next) {
			t.Fatal("output alias")
		}
		if body[0] != 1 {
			m.Groups = []RoadsideObjectGroup{{}}
			if _, e := m.MarshalBinary(); e == nil {
				t.Fatal("ignored groups")
			}
		}
	}
	f := roadsideFixture()
	f.Header.Version = 15
	m, e := NewRoadside(f, RoadsideProfile{})
	if e != nil || len(m.Validate()) == 0 {
		t.Fatal(e)
	}
	b := roadsideHex(t, roadsideGolden)
	b[13]++
	if _, e := DecodeRoadside(b); e == nil {
		t.Fatal("wrong size")
	}
}
func TestRoadsideOpaqueExtensions(t *testing.T) {
	f := roadsideFixture()
	f.Groups[0].Objects[0].Extensions = []RoadsideObjectExtension{{ServiceID: 0xee, Gap: []byte{0xab}, Data: []byte{0xcd}}, {ServiceID: 0xef, Gap: []byte{0x11, 0x22}, Data: []byte{0x33}}}
	m, e := NewRoadside(f, RoadsideProfile{})
	if e != nil {
		t.Fatal(e)
	}
	b, _ := m.MarshalBinary()
	d, e := DecodeRoadside(b)
	if e != nil {
		t.Fatal(e)
	}
	again, _ := d.MarshalBinary()
	if !bytes.Equal(b, again) {
		t.Fatal("gaps")
	}
	d.Groups[0].Objects[0].Extensions[1].Data = append(d.Groups[0].Objects[0].Extensions[1].Data, 0x44)
	if _, e := d.MarshalBinary(); e != nil {
		t.Fatal("safe growth", e)
	}
	d.Groups[0].Objects[0].Extensions[0].Data = append(d.Groups[0].Objects[0].Extensions[0].Data, 0x55)
	if _, e := d.MarshalBinary(); !errors.Is(e, ErrLayout) {
		t.Fatal("unsafe movement", e)
	}
	d, _ = DecodeRoadside(b)
	d.Groups[0].Objects[0].Common.PositionOptional = &rc013.PositionOptional{}
	if _, e := d.MarshalBinary(); !errors.Is(e, ErrLayout) {
		t.Fatal("outer movement", e)
	}
	for _, v := range []byte{0, 255} {
		bad := bytes.Clone(b)
		bad[25] = v
		if _, e := DecodeRoadside(bad); e == nil {
			t.Fatal("object length", v)
		}
	}
	bad := bytes.Clone(b)
	bad[58] = 255
	if _, e := DecodeRoadside(bad); e == nil {
		t.Fatal("free length")
	}
}

func TestRoadsideUnknownOptionsAndNewOwnership(t *testing.T) {
	f := roadsideFixture()
	f.Options = []RoadsideOption{{Index: 2, Data: []byte{0xa5, 0x5a}}}
	m, e := NewRoadside(f, RoadsideProfile{})
	if e != nil {
		t.Fatal(e)
	}
	f.Options[0].Data[0] = 0
	f.Groups[0].Objects[0].ID = 0
	b, e := m.MarshalBinary()
	if e != nil || b[17] != 0x20 || b[18] != 2 || b[19] != 0xa5 || m.Groups[0].Objects[0].ID != 0x11223344 {
		t.Fatal("new alias", e)
	}
	bad := bytes.Clone(b)
	bad[18] = 255
	if _, e := DecodeRoadside(bad); e == nil {
		t.Fatal("option bounds")
	}
	m.Options = append(m.Options, RoadsideOption{Index: 2, Data: []byte{1}})
	if _, e := m.MarshalBinary(); e == nil {
		t.Fatal("duplicate option")
	}
}

func TestRoadsideNestedErrorOffsets(t *testing.T) {
	// A declared optional position frame starts just beyond the 36-byte object.
	// The nested common decoder sees byte 28; the public roadside input sees 55.
	common := roadsideHex(t, roadsideGolden)
	common[26] = 0x80
	// The profiled packet's object starts at 33, so its free header starts at 69.
	freeHeader := roadsideHex(t, roadsideProfileGolden)
	freeHeader[69] = 0
	freeEntry := roadsideHex(t, roadsideProfileGolden)
	freeEntry[72] = 0
	for _, tc := range []struct {
		name    string
		data    []byte
		profile RoadsideProfile
		offset  int
		field   string
		kind    error
	}{
		{"common optional", common, RoadsideProfile{}, 55, "common", ErrTruncated},
		{"free header", freeHeader, roadsideProfile(), 69, "free.header", ErrMalformed},
		{"free entry", freeEntry, roadsideProfile(), 70, "free.entry", ErrMalformed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeRoadsideWithProfile(tc.data, tc.profile)
			var located *Error
			if !errors.Is(err, tc.kind) || !errors.As(err, &located) {
				t.Fatalf("error category: %v", err)
			}
			if located.Offset != tc.offset || located.Field != tc.field {
				t.Fatalf("diagnostic = %+v; want offset%d field%s", located, tc.offset, tc.field)
			}
		})
	}
	// Profile errors have no input location and must retain the unknown offset.
	p := roadsideProfile()
	p.ObjectExtensions.Layouts[0x91] = ObjectExtensionLayout{Fields: []ObjectExtensionField{ObjectLevel}, Complete: true}
	_, err := DecodeRoadsideWithProfile(roadsideHex(t, roadsideProfileGolden), p)
	var located *Error
	if !errors.As(err, &located) || located.Offset != -1 {
		t.Fatalf("unknown offset: %v", err)
	}
}

func TestRoadsideEncodeErrorOffsets(t *testing.T) {
	for _, tc := range []struct {
		name   string
		edit   func(*RoadsideMessage)
		offset int
	}{
		{"header hour", func(m *RoadsideMessage) { m.Header.Time.Hour = 128 }, 8},
		{"object service", func(m *RoadsideMessage) { m.Groups[0].Objects[0].ServiceID = 8 }, 19},
		{"object position confidence", func(m *RoadsideMessage) { m.Groups[0].Objects[0].Common.Position.PositionConfidence = 16 }, 41},
		{"extension length unknown", func(m *RoadsideMessage) {
			m.Groups[0].Objects[0].Extensions = []RoadsideObjectExtension{{ServiceID: 9}}
		}, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, err := NewRoadside(roadsideFixture(), RoadsideProfile{})
			if err != nil {
				t.Fatal(err)
			}
			tc.edit(&m)
			out, err := m.MarshalBinary()
			var located *Error
			if out != nil || !errors.Is(err, ErrRange) || !errors.As(err, &located) {
				t.Fatalf("error/output: %x %v", out, err)
			}
			if located.Offset != tc.offset {
				t.Fatalf("offset%d want%d (%v)", located.Offset, tc.offset, err)
			}
		})
	}
}

func TestRoadsideSensorErrorOffsets(t *testing.T) {
	for _, tc := range []struct {
		index int
		value byte
		want  int
	}{{20, 0, 20}, {19, 255, 19}} {
		b := roadsideHex(t, roadsideProfileGolden)
		b[tc.index] = tc.value
		_, err := DecodeRoadsideWithProfile(b, roadsideProfile())
		var located *Error
		if !errors.Is(err, ErrMalformed) || !errors.As(err, &located) || located.Offset != tc.want {
			t.Fatalf("index%d: %v; want offset%d", tc.index, err, tc.want)
		}
	}
}
func TestRoadsideOpaqueOptionAnchors(t *testing.T) {
	f := roadsideFixture()
	f.Groups[0].Objects = nil
	f.Options = []RoadsideOption{{Index: 1, Data: []byte{0xaa}}, {Index: 2, Data: []byte{0xbb}}}
	m, err := NewRoadside(f, RoadsideProfile{})
	if err != nil {
		t.Fatal(err)
	}
	m.Options[0].Data = append(m.Options[0].Data, 0xcc)
	if out, err := m.MarshalBinary(); !errors.Is(err, ErrLayout) || out != nil {
		t.Fatalf("opaque option moved: %x %v", out, err)
	}
	rebuilt, err := NewRoadside(m.RoadsideFields, RoadsideProfile{})
	if err != nil {
		t.Fatal("explicit reconstruction", err)
	}
	if out, err := rebuilt.MarshalBinary(); err != nil || !bytes.Equal(out[19:24], []byte{0xaa, 0xcc, 1, 0xbb, 0}) {
		t.Fatalf("reconstruction %x %v", out, err)
	}
}
