package rc016_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/hareku/its-forum-go/rc013v1"
	rc016 "github.com/hareku/its-forum-go/rc016v1"
)

func personalHex(t testing.TB, s string) []byte {
	t.Helper()
	b, e := hex.DecodeString(strings.Join(strings.Fields(s), ""))
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func personalFixture(t testing.TB, name string) []byte {
	t.Helper()
	b, e := os.ReadFile("testdata/" + name + ".hex")
	if e != nil {
		t.Fatal(e)
	}
	return personalHex(t, string(b))
}
func personalProfile() rc016.BicycleProfile {
	return rc016.BicycleProfile{Applications: map[uint8]rc016.BicycleApplicationKind{0x90: rc016.ApplicationPersonalCommon, 0x91: rc016.ApplicationBicycleBasic, 0x92: rc016.ApplicationBicycleExtended, 0x93: rc016.ApplicationPedestrian}}
}
func personalFields() rc016.BicycleFields {
	return rc016.BicycleFields{Header: rc013.Header{ServiceID: 1, MessageID: 1, Version: 1, VehicleID: 0x01020304, Counter: 5}, Common: rc013.CommonData{
		Time:              rc013.Time{LeapSecondCorrection: true, Hour: 9, Minute: 10, Millisecond: 3000},
		Position:          rc013.Position{Latitude: -1, Longitude: 2, Elevation: 0xffff, PositionConfidence: 10, ElevationConfidence: 11},
		VehicleState:      rc013.VehicleState{Speed: 1, Heading: 7200, Acceleration: -1, SpeedConfidence: 1, HeadingConfidence: 2, AccelerationConfidence: 3, Transmission: 2, SteeringWheelAngle: -1},
		VehicleAttributes: rc013.VehicleAttributes{SizeClass: 2, RoleClass: 1, Width: 100, Length: 200},
	}, Applications: []rc016.BicycleApplication{
		{ServiceID: 0x90, PersonalCommon: &rc016.PersonalCommon{Level: 5, SystemDelay: 3, WatchData: 0x01020304}},
		{ServiceID: 0x91, Basic: &rc016.BicycleBasic{AssistType: 2, BicycleType: 3, AssistState: 2, Pedaling: 1, DrivePower: 16, Collision: 5}},
		{ServiceID: 0x92, Extended: &rc016.BicycleExtended{MainGear: 31, MainMaxGear: 31, SubGear: 31, SubMaxGear: 31, TireCircumference: 255, Cadence: 255, GearRatio: 1023, RiderTorque: 255, MotorTorque: 255, MaxAssistPower: 255, AssistPower: 255, HumanPower: 255, MaxBattery: 255, Battery: 255, RearLight: 3, DriveUnitStatus: 3, Maintenance: 3, Reserved: 15}},
		{ServiceID: 0x93, Pedestrian: &rc016.Pedestrian{ShoeAttribute: 3, Steps: 1, Motion: 2, Reserved: 0x15555}},
	}}
}
func assertPersonalError(t testing.TB, err, errorKind error) {
	t.Helper()
	if !errors.Is(err, errorKind) {
		t.Fatalf("error %v; want %v", err, errorKind)
	}
	var located *rc016.Error
	if !errors.As(err, &located) || located.Field == "" {
		t.Fatalf("missing error context: %v", err)
	}
}
func TestBicycleIndependentGolden(t *testing.T) {
	literal := personalFixture(t, "bicycle")
	m, err := rc016.DecodeBicycleWithProfile(literal, personalProfile())
	if err != nil {
		t.Fatal(err)
	}
	expected := personalFields()
	expected.Header.DataLength = 28
	expected.Header.OptionFlags = 1
	if !reflect.DeepEqual(m.Header, expected.Header) || !reflect.DeepEqual(m.Common, expected.Common) {
		t.Fatalf("header/common: %#v %#v", m.Header, m.Common)
	}
	for i, a := range m.Applications {
		e := expected.Applications[i]
		if a.ServiceID != e.ServiceID || !reflect.DeepEqual(a.PersonalCommon, e.PersonalCommon) || !reflect.DeepEqual(a.Basic, e.Basic) || !reflect.DeepEqual(a.Extended, e.Extended) || !reflect.DeepEqual(a.Pedestrian, e.Pedestrian) {
			t.Fatalf("application %d: %#v", i, a)
		}
	}
	fresh, err := rc016.NewBicycle(personalFields(), personalProfile())
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []rc016.BicycleMessage{m, fresh} {
		b, e := v.MarshalBinary()
		if e != nil || !bytes.Equal(b, literal) {
			t.Fatalf("golden encode: %x %v", b, e)
		}
	}
	m.Applications[1].Basic.DrivePower = 17
	want := bytes.Clone(literal)
	want[56] = 0x15
	b, e := m.MarshalBinary()
	if e != nil || !bytes.Equal(b, want) {
		t.Fatalf("edited: %x %v", b, e)
	}
	opaque, e := rc016.DecodeBicycle(literal)
	if e != nil {
		t.Fatal(e)
	}
	b, e = opaque.MarshalBinary()
	if e != nil || !bytes.Equal(b, literal) {
		t.Fatal("opaque roundtrip", e)
	}
	if opaque.Applications[0].PersonalCommon != nil {
		t.Fatal("profile was guessed")
	}
}

func TestBicycleResizeAndOpaqueAnchors(t *testing.T) {
	literal := personalFixture(t, "bicycle")
	known, e := rc016.DecodeBicycleWithProfile(literal, personalProfile())
	if e != nil {
		t.Fatal(e)
	}
	known.Applications = known.Applications[1:]
	b, e := known.MarshalBinary()
	if e != nil {
		t.Fatal(e)
	}
	// Three entries: 10-byte management and 22 payload bytes; header/common unchanged.
	want := append(bytes.Clone(literal[:36]), personalHex(t, "53 91 00 03 92 03 0e 93 11 05")...)
	want = append(want, literal[54:]...)
	if !bytes.Equal(b, want) {
		t.Fatalf("known removal %x want %x", b, want)
	}
	known.Common.PositionOptional = &rc013.PositionOptional{}
	b, e = known.MarshalBinary()
	if e != nil || b[6] != 30 || b[7] != 0x81 || len(b) != 70 {
		t.Fatalf("optional resize %x %v", b, e)
	}
	known.Applications = nil
	known.Common.PositionOptional = nil
	b, e = known.MarshalBinary()
	if e != nil || len(b) != 36 || b[7] != 0 {
		t.Fatalf("remove free %x %v", b, e)
	}
	known.Applications = []rc016.BicycleApplication{{ServiceID: 0x93, Pedestrian: &rc016.Pedestrian{}}}
	b, e = known.MarshalBinary()
	if e != nil || len(b) != 45 || b[36] != 0x21 {
		t.Fatalf("add present-zero %x %v", b, e)
	}

	raw, e := rc016.DecodeBicycle(literal)
	if e != nil {
		t.Fatal(e)
	}
	raw.Common.PositionOptional = &rc013.PositionOptional{}
	b, e = raw.MarshalBinary()
	assertPersonalError(t, e, rc016.ErrLayout)
	if b != nil {
		t.Fatal("partial output")
	}
	raw.Common.PositionOptional = nil
	raw.Applications = raw.Applications[1:]
	_, e = raw.MarshalBinary()
	assertPersonalError(t, e, rc016.ErrLayout)
	raw, e = rc016.DecodeBicycle(literal)
	if e != nil {
		t.Fatal(e)
	}
	raw.Applications[3].Data = append(raw.Applications[3].Data, 0xab)
	b, e = raw.MarshalBinary()
	if e != nil || len(b) != 77 || b[48] != 6 || b[76] != 0xab {
		t.Fatalf("last growth %x %v", b, e)
	}
	raw.Applications[0].Data = append(raw.Applications[0].Data, 0x00)
	_, e = raw.MarshalBinary()
	assertPersonalError(t, e, rc016.ErrLayout)
	rebuilt, e := rc016.NewBicycle(raw.BicycleFields, rc016.BicycleProfile{})
	if e != nil {
		t.Fatal("explicit placement", e)
	}
	if _, e = rebuilt.MarshalBinary(); e != nil {
		t.Fatal(e)
	}
}

func TestBicycleGapsOwnershipAndDuplicateIDs(t *testing.T) {
	fields := personalFields()
	fields.Applications = []rc016.BicycleApplication{{ServiceID: 0xfa, Gap: []byte{0xa5, 0x5a}, Data: []byte{0xde, 0xad}}, {ServiceID: 0xfa, Gap: []byte{0xcc}, Data: []byte{0xbe, 0xef}}}
	fields.Common.PositionOptional = &rc013.PositionOptional{Delay: 1}
	m, e := rc016.NewBicycle(fields, rc016.BicycleProfile{})
	if e != nil {
		t.Fatal(e)
	}
	b, e := m.MarshalBinary()
	if e != nil {
		t.Fatal(e)
	}
	// Two entries, payload-relative starts 2 and 5, with nonzero gaps.
	if !bytes.Equal(b[38:], personalHex(t, "3a fa 02 02 fa 05 02 a5 5a de ad cc be ef")) {
		t.Fatalf("gaps %x", b)
	}
	fields.Applications[0].Data[0] = 0
	fields.Applications[0].Gap[0] = 0
	fields.Common.PositionOptional.Delay = 2
	again, e := m.MarshalBinary()
	if e != nil || !bytes.Equal(again, b) {
		t.Fatal("New aliases inputs")
	}
	decoded, e := rc016.DecodeBicycle(b)
	if e != nil {
		t.Fatal(e)
	}
	saved := bytes.Clone(b)
	for i := range b {
		b[i] = 0
	}
	again, e = decoded.MarshalBinary()
	if e != nil || !bytes.Equal(again, saved) {
		t.Fatal("Decode aliases input")
	}
	again[0] = 0
	again, e = decoded.MarshalBinary()
	if e != nil || !bytes.Equal(again, saved) {
		t.Fatal("Marshal aliases output")
	}
	decoded.Common.Time.Hour = 10
	again, e = decoded.MarshalBinary()
	saved[8] = 0x8a
	if e != nil || !bytes.Equal(again, saved) {
		t.Fatal("known edit damaged gaps")
	}
	decoded.Applications[0].Gap = append(decoded.Applications[0].Gap, 1)
	_, e = decoded.MarshalBinary()
	assertPersonalError(t, e, rc016.ErrLayout)
}

func TestBicycleProfileFreezeAndConflicts(t *testing.T) {
	for _, decode := range []bool{false, true} {
		p := personalProfile()
		fields := personalFields()
		var m rc016.BicycleMessage
		var e error
		if decode {
			m, e = rc016.DecodeBicycleWithProfile(personalFixture(t, "bicycle"), p)
		} else {
			m, e = rc016.NewBicycle(fields, p)
		}
		if e != nil {
			t.Fatal(e)
		}
		p.Applications[0x91] = rc016.ApplicationPedestrian
		delete(p.Applications, 0x90)
		fields.Applications[1].Basic.DrivePower = 99
		m.Applications[1].Basic.DrivePower = 17
		b, e := m.MarshalBinary()
		if e != nil || b[56] != 0x15 {
			t.Fatal("mutable profile", e)
		}
	}
	p := personalProfile()
	p.Applications[0x91] = rc016.ApplicationPedestrian
	_, e := rc016.DecodeBicycleWithProfile(personalFixture(t, "bicycle"), p)
	assertPersonalError(t, e, rc016.ErrTruncated)
	var located *rc016.Error
	if !errors.As(e, &located) || located.Offset != 57 {
		t.Fatalf("application error offset: %v; want 57", e)
	}
	p.Applications[0x91] = 99
	_, e = rc016.NewBicycle(personalFields(), p)
	assertPersonalError(t, e, rc016.ErrUnsupported)
	f := personalFields()
	f.Applications[0].Basic = &rc016.BicycleBasic{}
	_, e = rc016.NewBicycle(f, personalProfile())
	assertPersonalError(t, e, rc016.ErrMalformed)
	f = personalFields()
	f.Applications[0].PersonalCommon = nil
	_, e = rc016.NewBicycle(f, personalProfile())
	assertPersonalError(t, e, rc016.ErrMalformed)
	_, e = rc016.NewBicycle(personalFields(), rc016.BicycleProfile{})
	assertPersonalError(t, e, rc016.ErrMalformed)
	// Duplicate mapped IDs are separately typed, with no global registry.
	f = personalFields()
	f.Applications = []rc016.BicycleApplication{f.Applications[0], f.Applications[0]}
	m, e := rc016.NewBicycle(f, personalProfile())
	if e != nil {
		t.Fatal(e)
	}
	b, e := m.MarshalBinary()
	if e != nil {
		t.Fatal(e)
	}
	d, e := rc016.DecodeBicycleWithProfile(b, personalProfile())
	if e != nil || len(d.Applications) != 2 || d.Applications[1].PersonalCommon.Level != 5 {
		t.Fatal("duplicate mapped IDs", e)
	}
}

func TestBicycleNegativeBoundaries(t *testing.T) {
	literal := personalFixture(t, "bicycle")
	for n := 0; n < len(literal); n++ {
		if _, e := rc016.DecodeBicycleWithProfile(literal[:n], personalProfile()); e == nil {
			t.Fatalf("accepted prefix %d", n)
		}
	}
	for _, tc := range []struct {
		at    int
		value byte
	}{{0, 0}, {6, 27}, {6, 255}, {7, 0}, {36, 0}, {36, 0x64}, {38, 60}, {39, 0}, {41, 0}, {42, 61}} {
		b := bytes.Clone(literal)
		b[tc.at] = tc.value
		if _, e := rc016.DecodeBicycle(b); e == nil {
			t.Fatalf("accepted mutation %d=%d", tc.at, tc.value)
		}
	}
	if _, e := rc016.DecodeBicycle(append(bytes.Clone(literal), 0)); e == nil {
		t.Fatal("trailing byte")
	}
	f := personalFields()
	f.Applications = []rc016.BicycleApplication{{ServiceID: 0xee, Data: make([]byte, 60)}}
	m, e := rc016.NewBicycle(f, rc016.BicycleProfile{})
	if e != nil {
		t.Fatal(e)
	}
	b, e := m.MarshalBinary()
	if e != nil || len(b) != 100 {
		t.Fatal("100 byte boundary", e)
	}
	m.Applications[0].Gap = []byte{0}
	_, e = m.MarshalBinary()
	assertPersonalError(t, e, rc016.ErrRange)
	m.Applications[0].Gap = nil
	m.Applications[0].Data = make([]byte, 61)
	_, e = m.MarshalBinary()
	assertPersonalError(t, e, rc016.ErrRange)
	m.Applications = make([]rc016.BicycleApplication, 8)
	_, e = m.MarshalBinary()
	assertPersonalError(t, e, rc016.ErrRange)
	// Unknown common suffix stays anchored through shared CommonData policy.
	f = personalFields()
	f.Applications = nil
	f.Common.ExtendedOptions = true
	f.Common.OpaqueTail = []byte{0xab, 0xcd}
	m, e = rc016.NewBicycle(f, rc016.BicycleProfile{})
	if e != nil {
		t.Fatal(e)
	}
	b, e = m.MarshalBinary()
	if e != nil {
		t.Fatal(e)
	}
	m, e = rc016.DecodeBicycle(b)
	if e != nil {
		t.Fatal(e)
	}
	m.Common.PositionOptional = &rc013.PositionOptional{}
	_, e = m.MarshalBinary()
	assertPersonalError(t, e, rc016.ErrLayout)
}

func TestBicycleMarshalErrorPacketOffsets(t *testing.T) {
	cases := []struct {
		name   string
		change func(*rc016.BicycleFields)
		want   int
	}{
		{"common", func(f *rc016.BicycleFields) { f.Common.Position.PositionConfidence = 16 }, 22},
		{"basic", func(f *rc016.BicycleFields) { f.Applications[1].Basic.AssistState = 4 }, 55},
		{"extended", func(f *rc016.BicycleFields) { f.Applications[2].Extended.GearRatio = 1024 }, 61},
		{"pedestrian", func(f *rc016.BicycleFields) { f.Applications[3].Pedestrian.Motion = 4 }, 73},
		{"gap", func(f *rc016.BicycleFields) {
			f.Applications[1].Gap = []byte{0xaa, 0xbb}
			f.Applications[1].Basic.AssistState = 4
		}, 57},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fields := personalFields()
			tc.change(&fields)
			_, err := rc016.NewBicycle(fields, personalProfile())
			assertPersonalError(t, err, rc016.ErrRange)
			var located *rc016.Error
			if !errors.As(err, &located) || located.Offset != tc.want {
				t.Fatalf("New offset: %v; want %d", err, tc.want)
			}
			m, err := rc016.NewBicycle(personalFields(), personalProfile())
			if err != nil {
				t.Fatal(err)
			}
			tc.change(&m.BicycleFields)
			out, err := m.MarshalBinary()
			assertPersonalError(t, err, rc016.ErrRange)
			if out != nil || !errors.As(err, &located) || located.Offset != tc.want {
				t.Fatalf("Marshal offset: %v; want %d", err, tc.want)
			}
		})
	}
	// An incompatible supplied representation has no specific wire field location.
	m, err := rc016.NewBicycle(personalFields(), personalProfile())
	if err != nil {
		t.Fatal(err)
	}
	m.Applications[1].Basic = nil
	_, err = m.MarshalBinary()
	assertPersonalError(t, err, rc016.ErrMalformed)
	var located *rc016.Error
	if !errors.As(err, &located) || located.Offset != -1 {
		t.Fatalf("unknown offset lost: %v", err)
	}
}
