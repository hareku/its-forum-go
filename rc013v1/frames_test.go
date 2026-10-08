package rc013_test

import (
	"bytes"
	"encoding"
	"errors"
	"reflect"
	"testing"

	"github.com/hareku/its-forum-go/rc013v1"
)

func TestEveryFixedFrameAndAtomicDecode(t *testing.T) {
	c := expectedCommon()
	cases := []struct {
		name, hex string
		value     encoding.BinaryMarshaler
	}{
		{"header", "29010203040536fd", rc013.Header{ServiceID: 1, MessageID: 1, Version: 1, VehicleID: 0x01020304, Counter: 5, DataLength: 54, OptionFlags: 0xfd}},
		{"time", "890a0bb8", c.Time},
		{"position", "ffffffff00000002ffffab", c.Position},
		{"state", "00011c20ffff29afff", c.VehicleState},
		{"attributes", "211900c8", c.VehicleAttributes},
		{"positionOptional", "089c", *c.PositionOptional},
		{"GPSOptional", "01021c20", *c.GPSStatusOptional},
		{"acquisitionOptional", "c59a", *c.PositionAcquisitionOptional},
		{"stateOptional", "fffeaa64811be4", *c.VehicleStateOptional},
		{"intersection", "2322fffffffe00000003", *c.Intersection},
		{"extension", "a1", *c.Extension},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			literal := hexBytes(t, tc.hex)
			out, err := tc.value.MarshalBinary()
			if err != nil || !bytes.Equal(out, literal) {
				t.Fatalf("encode %x %v", out, err)
			}
			ptr := reflect.New(reflect.TypeOf(tc.value))
			decoder := ptr.Interface().(encoding.BinaryUnmarshaler)
			if err := decoder.UnmarshalBinary(literal); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(ptr.Elem().Interface(), tc.value) {
				t.Fatalf("decoded fields %#v", ptr.Elem().Interface())
			}
			for n := 0; n < len(literal); n++ {
				if err := decoder.UnmarshalBinary(literal[:n]); !errors.Is(err, rc013.ErrTruncated) {
					t.Fatalf("prefix%d:%v", n, err)
				}
				if !reflect.DeepEqual(ptr.Elem().Interface(), tc.value) {
					t.Fatal("receiver modified on truncated input")
				}
			}
			if err := decoder.UnmarshalBinary(append(bytes.Clone(literal), 0)); !errors.Is(err, rc013.ErrMalformed) {
				t.Fatal("extra byte", err)
			}
			if !reflect.DeepEqual(ptr.Elem().Interface(), tc.value) {
				t.Fatal("receiver modified on extra input")
			}
		})
	}
}

func TestFieldOverflow(t *testing.T) {
	for _, v := range []encoding.BinaryMarshaler{
		rc013.Header{ServiceID: 8}, rc013.Header{MessageID: 4}, rc013.Header{Version: 8}, rc013.Time{Hour: 128},
		rc013.Position{PositionConfidence: 16}, rc013.Position{ElevationConfidence: 16},
		rc013.VehicleState{SpeedConfidence: 8}, rc013.VehicleState{HeadingConfidence: 8}, rc013.VehicleState{AccelerationConfidence: 8}, rc013.VehicleState{Transmission: 8}, rc013.VehicleState{SteeringWheelAngle: -2049}, rc013.VehicleState{SteeringWheelAngle: 2048},
		rc013.VehicleAttributes{SizeClass: 16}, rc013.VehicleAttributes{RoleClass: 16}, rc013.VehicleAttributes{Width: 1024}, rc013.VehicleAttributes{Length: 16384},
		rc013.PositionOptional{Delay: 32}, rc013.PositionOptional{Revision: 32}, rc013.PositionOptional{RoadFacilities: 8}, rc013.PositionOptional{RoadClass: 8},
		rc013.PositionAcquisitionOptional{Mode: 4}, rc013.PositionAcquisitionOptional{PDOP: 64}, rc013.PositionAcquisitionOptional{Satellites: 16}, rc013.PositionAcquisitionOptional{Multipath: 4},
		rc013.VehicleStateOptional{Brakes: 64}, rc013.VehicleStateOptional{AuxiliaryBrakes: 4}, rc013.VehicleStateOptional{ACC: 4}, rc013.VehicleStateOptional{CACC: 4}, rc013.VehicleStateOptional{PCS: 4}, rc013.VehicleStateOptional{ABS: 4}, rc013.VehicleStateOptional{TRC: 4}, rc013.VehicleStateOptional{ESC: 4}, rc013.VehicleStateOptional{LKA: 4}, rc013.VehicleStateOptional{LDW: 4},
		rc013.Intersection{DistanceSource: 8}, rc013.Intersection{Distance: 1024}, rc013.Intersection{PositionSource: 8}, rc013.Extension{Upper: 16}, rc013.Extension{Status: 16},
	} {
		out, err := v.MarshalBinary()
		if out != nil || !errors.Is(err, rc013.ErrRange) {
			t.Fatalf("overflow accepted %#v: %x %v", v, out, err)
		}
	}
}

func TestUnavailableDFValuesStayPresent(t *testing.T) {
	for _, tc := range []struct {
		value encoding.BinaryMarshaler
		hex   string
	}{
		{rc013.Time{Hour: rc013.HourUnavailable, Minute: rc013.MinuteUnavailable, Millisecond: rc013.MillisecondUnavailable}, "7fffffff"},
		{rc013.Position{Latitude: rc013.LatitudeUnavailable, Longitude: rc013.LongitudeUnavailable, Elevation: rc013.ElevationUnavailable}, "8000000080000000f00000"},
		{rc013.VehicleState{Speed: rc013.SpeedUnavailable, Heading: rc013.HeadingUnavailable, Acceleration: rc013.AccelerationUnavailable, Transmission: 7, SteeringWheelAngle: rc013.SteeringWheelAngleUnavailable}, "ffffffff8000007800"},
		{rc013.VehicleAttributes{SizeClass: 15, RoleClass: 15, Width: 1023, Length: 16383}, "ffffffff"},
		{rc013.PositionOptional{Delay: 31, Revision: 31}, "ffc0"},
		{rc013.GPSStatusOptional{SemiMajorAxis: 255, SemiMinorAxis: 255, Orientation: rc013.HeadingUnavailable}, "ffffffff"},
		{rc013.PositionAcquisitionOptional{PDOP: 63, Satellites: 15}, "3ff0"},
		{rc013.VehicleStateOptional{YawRate: rc013.YawRateUnavailable, Throttle: 255}, "800000ff000000"},
		{rc013.Intersection{Distance: 1023, Latitude: rc013.LatitudeUnavailable, Longitude: rc013.LongitudeUnavailable}, "1ff88000000080000000"},
	} {
		literal := hexBytes(t, tc.hex)
		out, err := tc.value.MarshalBinary()
		if err != nil || !bytes.Equal(out, literal) {
			t.Fatalf("unavailable%T: %x want%x %v", tc.value, out, literal, err)
		}
		ptr := reflect.New(reflect.TypeOf(tc.value))
		if err := ptr.Interface().(encoding.BinaryUnmarshaler).UnmarshalBinary(literal); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(ptr.Elem().Interface(), tc.value) {
			t.Fatal("unavailable changed")
		}
	}
}

func TestElevationSpecialEncoding(t *testing.T) {
	for _, tc := range []struct {
		raw   rc013.Elevation
		dm    int32
		valid bool
	}{{0, 0, true}, {0x7fff, 32767, true}, {0x8000, 32768, true}, {0xefff, 61439, true}, {0xf000, 0, false}, {0xf001, -4095, true}, {0xffff, -1, true}} {
		dm, ok := tc.raw.Decimeters()
		if dm != tc.dm || ok != tc.valid {
			t.Fatalf("%04x: %d %t", tc.raw, dm, ok)
		}
		if ok {
			raw, err := rc013.ElevationFromDecimeters(dm)
			if err != nil || raw != tc.raw {
				t.Fatal(raw, err)
			}
		}
	}
	for _, dm := range []int32{-4096, 61440} {
		if _, err := rc013.ElevationFromDecimeters(dm); !errors.Is(err, rc013.ErrRange) {
			t.Fatal("elevation overflow", err)
		}
	}
}

func TestSemanticValidationDoesNotNormalize(t *testing.T) {
	m := expectedMessage()
	m.Common.Time.Hour = 126
	m.Common.Position.Latitude = 2147483647
	m.Common.VehicleState.Heading = 65534
	m.Common.VehicleState.Transmission = 6
	m.Common.VehicleAttributes.RoleClass = 14
	before, err := m.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Validate()) < 5 {
		t.Fatal("missing semantic diagnostics")
	}
	after, err := m.MarshalBinary()
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("validation changed data")
	}
	decoded, err := rc013.Decode(before)
	if err != nil {
		t.Fatal("semantic value rejected structurally", err)
	}
	out, err := decoded.MarshalBinary()
	if err != nil || !bytes.Equal(before, out) {
		t.Fatal("raw semantics lost", err)
	}
}

func TestExtensionRoleDiagnostics(t *testing.T) {
	for _, tc := range []struct{ role, upper, status uint8 }{{0, 7, 4}, {1, 0, 2}, {2, 2, 5}, {3, 4, 5}, {4, 0, 1}, {5, 0, 1}, {15, 0, 0}} {
		c := minimalMessage().Common
		c.VehicleAttributes.RoleClass = tc.role
		c.Extension = &rc013.Extension{Upper: tc.upper, Status: tc.status}
		if issues := c.Validate(); len(issues) != 0 {
			t.Fatalf("known role%d %+v", tc.role, issues)
		}
		c.Extension.Upper = 15
		c.Extension.Status = 14
		if issues := c.Validate(); len(issues) != 2 {
			t.Fatalf("reserved role%d %+v", tc.role, issues)
		}
		c.Extension.Status = 15
		if issues := c.Validate(); len(issues) != 1 {
			t.Fatalf("emergency stop role%d %+v", tc.role, issues)
		}
		data, err := c.Extension.MarshalBinary()
		if err != nil || !bytes.Equal(data, []byte{0xff}) {
			t.Fatal("reserved bits normalized", err)
		}
	}
}
