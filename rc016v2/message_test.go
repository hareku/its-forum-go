package rc016_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/hareku/its-forum-go/rc013v1"
	rc016 "github.com/hareku/its-forum-go/rc016v2"
	"os"
	"reflect"
	"strings"
	"testing"
)

func messageHex(t testing.TB, s string) []byte {
	t.Helper()
	b, e := hex.DecodeString(strings.Join(strings.Fields(s), ""))
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func messageFixture(t testing.TB, name string) []byte {
	t.Helper()
	b, e := os.ReadFile("testdata/" + name + ".hex")
	if e != nil {
		t.Fatal(e)
	}
	return messageHex(t, string(b))
}
func messageProfile() rc016.Profile {
	return rc016.Profile{Applications: map[uint8]rc016.ApplicationKind{0x90: rc016.ApplicationBicycle, 0x91: rc016.ApplicationPedestrian}}
}
func messageFields(pedestrian bool) rc016.Fields {
	f := rc016.Fields{Header: rc013.Header{ServiceID: 1, MessageID: 1, Version: 1, VehicleID: 0x01020304, Counter: 5}, Common: rc013.CommonData{
		Time:              rc013.Time{LeapSecondCorrection: true, Hour: 9, Minute: 10, Millisecond: 3000},
		Position:          rc013.Position{Latitude: -1, Longitude: 2, Elevation: 0xffff, PositionConfidence: 10, ElevationConfidence: 11},
		VehicleState:      rc013.VehicleState{Speed: 1, Heading: 7200, Acceleration: -1, SpeedConfidence: 1, HeadingConfidence: 2, AccelerationConfidence: 3, Transmission: 2, SteeringWheelAngle: -1},
		VehicleAttributes: rc013.VehicleAttributes{SizeClass: 2, RoleClass: 1, Width: 100, Length: 200},
	}}
	personal := rc016.PersonalCommon{Level: 5, SystemDelay: 3, WatchData: 0x01020304}
	if pedestrian {
		f.Applications = []rc016.Application{{ServiceID: 0x91, Pedestrian: &rc016.PedestrianData{PersonalCommon: personal, Pedestrian: rc016.Pedestrian{ItemInfo: 1, Steps: 0x1234, Motion: 2, Reserved: 0xabcd}}}}
	} else {
		f.Applications = []rc016.Application{{ServiceID: 0x90, Bicycle: &rc016.BicycleData{PersonalCommon: personal, Basic: rc016.BicycleBasic{AssistType: 2, BicycleType: 3, AssistState: 2, Pedaling: 1, DrivePower: 16, Collision: 5}, Extended: rc016.BicycleExtended{MainGear: 3, MainMaxGear: 11, SubGear: 2, SubMaxGear: 3, TireCircumference: 210, Cadence: 91, GearRatio: 345, RiderTorque: 19, MotorTorque: 27, MaxAssistPower: 42, AssistPower: 23, HumanPower: 37, MaxBattery: 60, Battery: 41, RearLight: 2, DriveUnitStatus: 1, Maintenance: 2, Reserved: 9}}}}
	}
	return f
}
func messageError(t testing.TB, err, kind error) *rc016.Error {
	t.Helper()
	if !errors.Is(err, kind) {
		t.Fatalf("error %v; want %v", err, kind)
	}
	var e *rc016.Error
	if !errors.As(err, &e) || e.Field == "" {
		t.Fatalf("missing context: %v", err)
	}
	return e
}
func messageBytes(t testing.TB, m rc016.Message) []byte {
	t.Helper()
	b, e := m.MarshalBinary()
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestMessageIndependentGoldens(t *testing.T) {
	for _, name := range []string{"bicycle", "pedestrian"} {
		t.Run(name, func(t *testing.T) {
			input := messageFixture(t, name)
			m, e := rc016.DecodeWithProfile(input, messageProfile())
			if e != nil {
				t.Fatal(e)
			}
			fields := messageFields(name == "pedestrian")
			fields.Header.DataLength = 28
			fields.Header.OptionFlags = 1
			if !reflect.DeepEqual(m.Header, fields.Header) || !reflect.DeepEqual(m.Common, fields.Common) || !reflect.DeepEqual(m.Applications[0].Bicycle, fields.Applications[0].Bicycle) || !reflect.DeepEqual(m.Applications[0].Pedestrian, fields.Applications[0].Pedestrian) {
				t.Fatalf("decoded mismatch: %#v", m)
			}
			fresh, e := rc016.NewMessage(fields, messageProfile())
			if e != nil {
				t.Fatal(e)
			}
			raw, e := rc016.Decode(input)
			if e != nil {
				t.Fatal(e)
			}
			if raw.Applications[0].Bicycle != nil || raw.Applications[0].Pedestrian != nil {
				t.Fatal("guessed profile")
			}
			for _, v := range []rc016.Message{m, fresh, raw} {
				if !bytes.Equal(messageBytes(t, v), input) {
					t.Fatal("golden mismatch")
				}
			}
			envelope, e := rc013.Decode(input)
			if e != nil || len(envelope.Free.Entries) != 1 {
				t.Fatal("not a complete single application", e)
			}
			expectedSize := 22
			if name == "pedestrian" {
				expectedSize = 10
			}
			if len(input) != 40+expectedSize || len(envelope.Free.Entries[0].Data) != expectedSize {
				t.Fatal("wrong payload size")
			}
			m.Header.DataLength = 0
			m.Header.OptionFlags = 255
			if !bytes.Equal(messageBytes(t, m), input) {
				t.Fatal("management fields not derived")
			}
		})
	}
}
func TestMessageTypedEdits(t *testing.T) {
	cases := []struct {
		name   string
		edit   func(*rc016.Message)
		offset int
		mask   byte
	}{
		{"personal", func(m *rc016.Message) { m.Applications[0].Bicycle.PersonalCommon.SystemDelay = 2 }, 40, 1},
		{"basic", func(m *rc016.Message) { m.Applications[0].Bicycle.Basic.DrivePower = 17 }, 47, 0x10},
		{"extended", func(m *rc016.Message) { m.Applications[0].Bicycle.Extended.Cadence = 90 }, 52, 0x10},
		{"pedestrian", func(m *rc016.Message) { m.Applications[0].Pedestrian.Pedestrian.Steps = 0x1235 }, 47, 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			name := "bicycle"
			if tc.name == "pedestrian" {
				name = "pedestrian"
			}
			input := messageFixture(t, name)
			m, e := rc016.DecodeWithProfile(input, messageProfile())
			if e != nil {
				t.Fatal(e)
			}
			tc.edit(&m)
			want := bytes.Clone(input)
			want[tc.offset] ^= tc.mask
			if !bytes.Equal(messageBytes(t, m), want) {
				t.Fatalf("edit mask: %x want %x", messageBytes(t, m), want)
			}
		})
	}
}
func TestMessageOwnership(t *testing.T) {
	input := messageFixture(t, "bicycle")
	want := bytes.Clone(input)
	profile := messageProfile()
	m, e := rc016.DecodeWithProfile(input, profile)
	if e != nil {
		t.Fatal(e)
	}
	input[40] ^= 255
	profile.Applications[0x90] = rc016.ApplicationPedestrian
	out := messageBytes(t, m)
	out[40] ^= 255
	if !bytes.Equal(messageBytes(t, m), want) {
		t.Fatal("decode/profile/output alias")
	}
	f := messageFields(false)
	f.Common.PositionOptional = &rc013.PositionOptional{Delay: 1}
	f.Common.GPSStatusOptional = &rc013.GPSStatusOptional{SemiMajorAxis: 1}
	f.Common.PositionAcquisitionOptional = &rc013.PositionAcquisitionOptional{Mode: 1}
	f.Common.VehicleStateOptional = &rc013.VehicleStateOptional{Brakes: 1}
	f.Common.Intersection = &rc013.Intersection{Distance: 1}
	f.Common.Extension = &rc013.Extension{Upper: 1}
	f.Common.ExtendedOptions = true
	f.Common.OpaqueTail = []byte{0xa5}
	f.Applications[0].Gap = []byte{0x55}
	f.Applications[0].Data = []byte{0x66}
	p := messageProfile()
	m, e = rc016.NewMessage(f, p)
	if e != nil {
		t.Fatal(e)
	}
	want = messageBytes(t, m)
	f.Common.PositionOptional.Delay = 2
	f.Common.GPSStatusOptional.SemiMajorAxis = 2
	f.Common.PositionAcquisitionOptional.Mode = 2
	f.Common.VehicleStateOptional.Brakes = 2
	f.Common.Intersection.Distance = 2
	f.Common.Extension.Upper = 2
	f.Common.OpaqueTail[0] = 0
	f.Applications[0].Bicycle.Basic.DrivePower++
	f.Applications[0].Gap[0] = 0
	f.Applications[0].Data[0] = 0
	p.Applications[0x90] = rc016.ApplicationPedestrian
	if !bytes.Equal(messageBytes(t, m), want) || m.Applications[0].Data[0] != 0x66 {
		t.Fatal("constructor alias")
	}
	pf := messageFields(true)
	pm, e := rc016.NewMessage(pf, messageProfile())
	if e != nil {
		t.Fatal(e)
	}
	pf.Applications[0].Pedestrian.Pedestrian.Steps++
	if pm.Applications[0].Pedestrian.Pedestrian.Steps != 0x1234 {
		t.Fatal("pedestrian alias")
	}
}
func TestMessageOpaqueAnchors(t *testing.T) {
	f := messageFields(false)
	f.Common.PositionOptional = &rc013.PositionOptional{Delay: 1, Revision: 1}
	f.Applications = append(f.Applications, rc016.Application{ServiceID: 0x99, Gap: []byte{0xde, 0xad}, Data: []byte{0xbe, 0xef}})
	original, e := rc016.NewMessage(f, messageProfile())
	if e != nil {
		t.Fatal(e)
	}
	input := messageBytes(t, original)
	// Independent layout: common length30, flags81, free header7/count2,
	// app starts0/24, payload lengths22/2, and a two-byte retained gap.
	literal := messageHex(t, "2901020304051e81 890a0bb8ffffffff00000002ffffab00011c20ffff29afff211900c8 0840 3a900016991802 a3010203042391051ac43d25b5644c6ca85c94f0a669 deadbeef")
	if !bytes.Equal(input, literal) {
		t.Fatalf("optional/gap packet: %x", input)
	}
	for _, tc := range []struct {
		name string
		edit func(*rc016.Message)
	}{
		{"common", func(m *rc016.Message) { m.Common.GPSStatusOptional = &rc013.GPSStatusOptional{} }},
		{"gap", func(m *rc016.Message) { m.Applications[1].Gap = append(m.Applications[1].Gap, 0) }},
		{"count", func(m *rc016.Message) {
			m.Applications = append(m.Applications, rc016.Application{ServiceID: 0x98, Data: []byte{1}})
		}},
		{"order", func(m *rc016.Message) { m.Applications[0], m.Applications[1] = m.Applications[1], m.Applications[0] }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, e := rc016.DecodeWithProfile(input, messageProfile())
			if e != nil {
				t.Fatal(e)
			}
			tc.edit(&m)
			_, e = m.MarshalBinary()
			messageError(t, e, rc016.ErrLayout)
			rebuilt, e := rc016.NewMessage(m.Fields, messageProfile())
			if e != nil {
				t.Fatal(e)
			}
			messageBytes(t, rebuilt)
		})
	}
	m, e := rc016.DecodeWithProfile(input, messageProfile())
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(messageBytes(t, m), input) {
		t.Fatal("lossy gaps")
	}
	m.Applications[0].Bicycle.Basic.DrivePower++
	m.Applications[1].Data = append(m.Applications[1].Data, 0xfa)
	messageBytes(t, m)
	// Shrinking the common area and growing a gap by the same amount keeps the
	// unknown payload's absolute start, and therefore remains safe.
	m, e = rc016.DecodeWithProfile(input, messageProfile())
	if e != nil {
		t.Fatal(e)
	}
	m.Common.PositionOptional = nil
	m.Applications[1].Gap = append(m.Applications[1].Gap, 0, 0)
	messageBytes(t, m)
}
func TestMessageCommonOpaqueAndFreeAbsent(t *testing.T) {
	input := messageFixture(t, "bicycle")[:36]
	input = bytes.Clone(input)
	input[7] = 0
	m, e := rc016.Decode(input)
	if e != nil {
		t.Fatal(e)
	}
	if len(m.Applications) != 0 || !bytes.Equal(messageBytes(t, m), input) {
		t.Fatal("free absence")
	}
	input[6] = 30
	input[7] = rc013.OptionExtended
	input = append(input, 0xde, 0xad)
	m, e = rc016.Decode(input)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(messageBytes(t, m), input) {
		t.Fatal("opaque tail")
	}
	input[36] = 0
	if m.Common.OpaqueTail[0] != 0xde {
		t.Fatal("tail aliases input")
	}
	m.Common.PositionOptional = &rc013.PositionOptional{}
	_, e = m.MarshalBinary()
	located := messageError(t, e, rc016.ErrLayout)
	if located.Offset != 38 {
		t.Fatal(located)
	}
	rebuilt, e := rc016.NewMessage(m.Fields, rc016.Profile{})
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(rebuilt.Common.OpaqueTail, []byte{0xde, 0xad}) {
		t.Fatal("rebuild tail")
	}
	messageBytes(t, rebuilt)
}
func TestMessageMalformedAndErrors(t *testing.T) {
	golden := messageFixture(t, "bicycle")
	for n := 0; n < len(golden); n++ {
		if _, e := rc016.DecodeWithProfile(golden[:n], messageProfile()); e == nil {
			t.Fatalf("accepted truncation %d", n)
		}
	}
	for _, tc := range []struct {
		name string
		edit func([]byte) []byte
		kind error
	}{
		{"oversize", func(b []byte) []byte { return append(b, make([]byte, 101-len(b))...) }, rc016.ErrRange},
		{"header", func(b []byte) []byte { b[0] = 0x2a; return b }, rc016.ErrUnsupported},
		{"commonShort", func(b []byte) []byte { b[6] = 27; return b }, rc016.ErrMalformed},
		{"commonTruncated", func(b []byte) []byte { b[6] = 99; return b }, rc016.ErrTruncated},
		{"countZero", func(b []byte) []byte { b[36] = 8; return b }, rc016.ErrMalformed},
		{"address", func(b []byte) []byte { b[38] = 60; return b }, rc016.ErrMalformed},
		{"payloadTruncated", func(b []byte) []byte { b[39] = 23; return b }, rc016.ErrTruncated},
		{"payloadZero", func(b []byte) []byte { b[39] = 0; return b }, rc016.ErrMalformed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, e := rc016.DecodeWithProfile(tc.edit(bytes.Clone(golden)), messageProfile())
			messageError(t, e, tc.kind)
		})
	}
	for _, name := range []string{"bicycle", "pedestrian"} {
		for _, n := range []int{3, 5, 9, 11, 14, 21, 23} {
			f := messageFields(name == "pedestrian")
			id := f.Applications[0].ServiceID
			f.Applications = []rc016.Application{{ServiceID: id, Data: make([]byte, n)}}
			raw, e := rc016.NewMessage(f, rc016.Profile{})
			if e != nil {
				t.Fatal(e)
			}
			_, e = rc016.DecodeWithProfile(messageBytes(t, raw), messageProfile())
			kind := rc016.ErrMalformed
			size := 22
			if name == "pedestrian" {
				size = 10
			}
			if n < size {
				kind = rc016.ErrTruncated
			}
			located := messageError(t, e, kind)
			wantOffset := 40 + n
			if located.Offset != wantOffset || !strings.HasPrefix(located.Field, "message.applications[0].") {
				t.Fatalf("offset: %v want %d", located, wantOffset)
			}
		}
	}
	for _, tc := range []struct {
		name string
		edit func(*rc016.Fields, *rc016.Profile)
		kind error
	}{
		{"rawMapped", func(f *rc016.Fields, p *rc016.Profile) {
			f.Applications[0].Bicycle = nil
			f.Applications[0].Data = make([]byte, 22)
		}, rc016.ErrMalformed},
		{"typedUnmapped", func(f *rc016.Fields, p *rc016.Profile) { p.Applications = nil }, rc016.ErrMalformed},
		{"wrongKind", func(f *rc016.Fields, p *rc016.Profile) { p.Applications[0x90] = rc016.ApplicationPedestrian }, rc016.ErrMalformed},
		{"multiple", func(f *rc016.Fields, p *rc016.Profile) { f.Applications[0].Pedestrian = &rc016.PedestrianData{} }, rc016.ErrMalformed},
		{"unsupported", func(f *rc016.Fields, p *rc016.Profile) { p.Applications[0x90] = 255 }, rc016.ErrUnsupported},
		{"zeroKind", func(f *rc016.Fields, p *rc016.Profile) { p.Applications[0x90] = 0 }, rc016.ErrUnsupported},
		{"eightApps", func(f *rc016.Fields, p *rc016.Profile) { f.Applications = make([]rc016.Application, 8) }, rc016.ErrRange},
		{"largeGap", func(f *rc016.Fields, p *rc016.Profile) { f.Applications[0].Gap = make([]byte, 60) }, rc016.ErrRange},
		{"emptyRaw", func(f *rc016.Fields, p *rc016.Profile) { f.Applications = []rc016.Application{{ServiceID: 0x99}} }, rc016.ErrRange},
		{"largeRaw", func(f *rc016.Fields, p *rc016.Profile) {
			f.Applications = []rc016.Application{{ServiceID: 0x99, Data: make([]byte, 61)}}
		}, rc016.ErrRange},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, p := messageFields(false), messageProfile()
			tc.edit(&f, &p)
			_, e := rc016.NewMessage(f, p)
			located := messageError(t, e, tc.kind)
			if located.Offset != -1 {
				t.Fatalf("construction offset: %v", located)
			}
		})
	}
	m, e := rc016.DecodeWithProfile(golden, messageProfile())
	if e != nil {
		t.Fatal(e)
	}
	m.Applications[0].Bicycle.Basic.AssistType = 16
	_, e = m.MarshalBinary()
	located := messageError(t, e, rc016.ErrRange)
	if located.Offset != 45 || !strings.Contains(located.Field, "BicycleBasic.AssistType") {
		t.Fatal(located)
	}
	m.Applications[0].Bicycle.Basic.AssistType = 2
	m.Applications[0].Bicycle.Extended.MainGear = 32
	_, e = m.MarshalBinary()
	located = messageError(t, e, rc016.ErrRange)
	if located.Offset != 48 || !strings.Contains(located.Field, "BicycleExtended.MainGear") {
		t.Fatal(located)
	}
	m.Applications[0].Bicycle.Extended.MainGear = 3
	m.Common.VehicleState.SpeedConfidence = 8
	_, e = m.MarshalBinary()
	located = messageError(t, e, rc016.ErrRange)
	if located.Offset != 29 {
		t.Fatal(located)
	}
	_, e = rc016.DecodeWithProfile(golden, rc016.Profile{Applications: map[uint8]rc016.ApplicationKind{1: 255}})
	if messageError(t, e, rc016.ErrUnsupported).Offset != -1 {
		t.Fatal("profile offset")
	}
}

func TestMessageDescriptorOverlapAndRawOwnership(t *testing.T) {
	f := messageFields(false)
	f.Applications = []rc016.Application{{ServiceID: 0x98, Gap: []byte{0x11}, Data: []byte{0x22, 0x33}}, {ServiceID: 0x99, Data: []byte{0x44, 0x55}}}
	m, e := rc016.NewMessage(f, rc016.Profile{})
	if e != nil {
		t.Fatal(e)
	}
	wire := messageBytes(t, m)
	f.Applications[0].Gap[0] = 0
	f.Applications[0].Data[0] = 0
	if !bytes.Equal(wire, messageBytes(t, m)) {
		t.Fatal("raw constructor aliases input")
	}
	for _, address := range []byte{0, 1, 2} {
		bad := bytes.Clone(wire)
		bad[41] = address
		_, e := rc016.Decode(bad)
		messageError(t, e, rc016.ErrMalformed)
	}
	decoded, e := rc016.Decode(wire)
	if e != nil {
		t.Fatal(e)
	}
	wire[43] = 0
	wire[44] = 0
	if decoded.Applications[0].Gap[0] != 0x11 || decoded.Applications[0].Data[0] != 0x22 {
		t.Fatal("raw decode aliases input")
	}
}

func TestMessageKnownPayloadMayMove(t *testing.T) {
	input := messageFixture(t, "bicycle")
	m, e := rc016.DecodeWithProfile(input, messageProfile())
	if e != nil {
		t.Fatal(e)
	}
	m.Common.PositionOptional = &rc013.PositionOptional{Delay: 1, Revision: 1}
	m.Applications[0].Gap = []byte{0xaa, 0xbb}
	wire := messageBytes(t, m)
	if len(wire) != len(input)+4 || !bytes.Equal(wire[44:], input[40:]) {
		t.Fatal("known payload relocation")
	}
}
