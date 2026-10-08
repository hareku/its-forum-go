package rc016_test

import (
	"bytes"
	"encoding"
	"encoding/hex"
	"errors"
	rc016 "github.com/hareku/its-forum-go/rc016v2"
	"reflect"
	"testing"
)

func coreCommon() rc016.PersonalCommon {
	return rc016.PersonalCommon{Level: 5, SystemDelay: 29, WatchData: 0x12345678}
}
func coreBasic() rc016.BicycleBasic {
	return rc016.BicycleBasic{AssistType: 2, BicycleType: 7, AssistState: 3, Pedaling: 2, DrivePower: 173, Collision: 11}
}
func coreExtended() rc016.BicycleExtended {
	return rc016.BicycleExtended{MainGear: 17, MainMaxGear: 23, SubGear: 9, SubMaxGear: 13, TireCircumference: 201, Cadence: 87, GearRatio: 683, RiderTorque: 42, MotorTorque: 91, MaxAssistPower: 123, AssistPower: 145, HumanPower: 167, MaxBattery: 189, Battery: 211, RearLight: 2, DriveUnitStatus: 1, Maintenance: 2, Reserved: 10}
}
func corePedestrian() rc016.Pedestrian {
	return rc016.Pedestrian{ItemInfo: 2, Steps: 0x9abc, Motion: 2, Reserved: 0x1357}
}

func TestFrameAndPayloadIndependentGoldens(t *testing.T) {
	for _, tc := range []struct {
		value encoding.BinaryMarshaler
		hex   string
	}{
		{coreCommon(), "bd12345678"},
		{coreBasic(), "27eadb"},
		{coreExtended(), "8dd2dc957aaca96dee469ef74e6a"},
		{corePedestrian(), "0a6af21357"},
		{rc016.BicycleData{PersonalCommon: coreCommon(), Basic: coreBasic(), Extended: coreExtended()}, "bd1234567827eadb8dd2dc957aaca96dee469ef74e6a"},
		{rc016.PedestrianData{PersonalCommon: coreCommon(), Pedestrian: corePedestrian()}, "bd123456780a6af21357"},
	} {
		t.Run(reflect.TypeOf(tc.value).Name(), func(t *testing.T) {
			want, err := hex.DecodeString(tc.hex)
			if err != nil {
				t.Fatal(err)
			}
			got, err := tc.value.MarshalBinary()
			if err != nil || !bytes.Equal(got, want) {
				t.Fatalf("encode %x want %x: %v", got, want, err)
			}
			dst := reflect.New(reflect.TypeOf(tc.value))
			if err = dst.Interface().(encoding.BinaryUnmarshaler).UnmarshalBinary(want); err != nil || !reflect.DeepEqual(dst.Elem().Interface(), tc.value) {
				t.Fatalf("decode %v: %v", dst, err)
			}
			before := dst.Elem().Interface()
			for n := 0; n < len(want); n++ {
				err = dst.Interface().(encoding.BinaryUnmarshaler).UnmarshalBinary(want[:n])
				assertFrameError(t, err, rc016.ErrTruncated)
				if !reflect.DeepEqual(before, dst.Elem().Interface()) {
					t.Fatal("truncation changed receiver")
				}
			}
			err = dst.Interface().(encoding.BinaryUnmarshaler).UnmarshalBinary(append(want, 0))
			assertFrameError(t, err, rc016.ErrMalformed)
			if !reflect.DeepEqual(before, dst.Elem().Interface()) {
				t.Fatal("excess bytes changed receiver")
			}
		})
	}
}

func TestPayloadErrorOffsets(t *testing.T) {
	for _, tc := range []struct {
		value  encoding.BinaryMarshaler
		field  string
		offset int
	}{
		{rc016.BicycleData{Basic: rc016.BicycleBasic{Pedaling: 4}}, "BicycleBasic.Pedaling", 6},
		{rc016.BicycleData{Extended: rc016.BicycleExtended{GearRatio: 1024}}, "BicycleExtended.GearRatio", 12},
		{rc016.PedestrianData{Pedestrian: rc016.Pedestrian{Motion: 4}}, "Pedestrian.Motion", 7},
	} {
		out, err := tc.value.MarshalBinary()
		assertFrameError(t, err, rc016.ErrRange)
		var loc *rc016.Error
		errors.As(err, &loc)
		if out != nil || loc.Field != tc.field || loc.Offset != tc.offset {
			t.Fatalf("out %x error %+v", out, loc)
		}
	}
}

func TestPedestrianPayloadUnavailableCodes(t *testing.T) {
	value := rc016.PedestrianData{PersonalCommon: rc016.PersonalCommon{Level: rc016.LevelUnavailable, SystemDelay: rc016.SystemDelayUnavailable}, Pedestrian: rc016.Pedestrian{ItemInfo: rc016.ItemInfoUnavailable, Steps: rc016.StepsUnavailable, Motion: rc016.MotionUnavailable}}
	// First byte combines level 111 and delay 11111; pedestrian's first 24 bits
	// combine item 111111, steps 16 ones, and motion 11.
	want := []byte{0xff, 0, 0, 0, 0, 0xff, 0xff, 0xff, 0, 0}
	got, err := value.MarshalBinary()
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("sentinels %x: %v", got, err)
	}
	if issues := value.Validate(); len(issues) != 0 {
		t.Fatal(issues)
	}
	var decoded rc016.PedestrianData
	if err := decoded.UnmarshalBinary(want); err != nil || decoded != value {
		t.Fatalf("sentinel decode %+v: %v", decoded, err)
	}
}
