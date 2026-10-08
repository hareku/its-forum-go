package rc016_test

import (
	"bytes"
	"encoding"
	rc016 "github.com/hareku/its-forum-go/rc016v2"
	"reflect"
	"testing"
)

type coreValidator interface {
	encoding.BinaryMarshaler
	Validate() []rc016.Issue
}

func TestFrameSemanticBoundaries(t *testing.T) {
	tests := []struct {
		value  coreValidator
		issues int
	}{
		{rc016.PersonalCommon{Level: 0}, 1}, {rc016.PersonalCommon{Level: 1}, 0}, {rc016.PersonalCommon{Level: 5}, 0}, {rc016.PersonalCommon{Level: 6}, 1}, {rc016.PersonalCommon{Level: rc016.LevelUnavailable}, 0},
		{rc016.BicycleBasic{AssistType: 2, BicycleType: 7, AssistState: 3, Pedaling: 2, Collision: 15}, 0},
		{rc016.BicycleBasic{AssistType: 3, BicycleType: 8, Pedaling: 3}, 3},
		{rc016.BicycleExtended{RearLight: 2, DriveUnitStatus: 2, Maintenance: 2}, 0},
		{rc016.BicycleExtended{RearLight: 3, DriveUnitStatus: 3, Maintenance: 3, Reserved: 15}, 4},
		{rc016.Pedestrian{ItemInfo: 1, Motion: 3}, 0}, {rc016.Pedestrian{ItemInfo: 2}, 0}, {rc016.Pedestrian{ItemInfo: 63}, 0},
		{rc016.Pedestrian{ItemInfo: 0}, 1}, {rc016.Pedestrian{ItemInfo: 3}, 1}, {rc016.Pedestrian{ItemInfo: 62}, 1}, {rc016.Pedestrian{ItemInfo: 63, Reserved: 65535}, 1},
	}
	for _, tc := range tests {
		before, err := tc.value.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		if got := tc.value.Validate(); len(got) != tc.issues {
			t.Fatalf("%+v issues %v want %d", tc.value, got, tc.issues)
		}
		after, err := tc.value.MarshalBinary()
		if err != nil || !bytes.Equal(before, after) {
			t.Fatal("validation mutated bytes")
		}
	}
	for _, delay := range []uint8{0, 29, rc016.SystemDelaySaturated, rc016.SystemDelayUnavailable} {
		if got := (rc016.PersonalCommon{Level: 7, SystemDelay: delay}).Validate(); len(got) != 0 {
			t.Fatal(got)
		}
	}
	for _, steps := range []uint16{0, 65533, rc016.StepsSaturated, rc016.StepsUnavailable} {
		if got := (rc016.Pedestrian{ItemInfo: 63, Steps: steps, Motion: 3}).Validate(); len(got) != 0 {
			t.Fatal(got)
		}
	}
	for _, field := range []string{"TireCircumference", "Cadence", "RiderTorque", "MotorTorque", "MaxAssistPower", "AssistPower", "HumanPower", "MaxBattery", "Battery"} {
		for _, raw := range []uint64{0, 1, 253, 254, 255} {
			v := rc016.BicycleExtended{}
			reflect.ValueOf(&v).Elem().FieldByName(field).SetUint(raw)
			if got := v.Validate(); len(got) != 0 {
				t.Fatalf("%s %d: %v", field, raw, got)
			}
		}
	}
	for _, field := range []string{"MainGear", "MainMaxGear", "SubGear", "SubMaxGear", "GearRatio"} {
		max := uint64(31)
		if field == "GearRatio" {
			max = 1023
		}
		for _, raw := range []uint64{0, 1, max - 1, max} {
			v := rc016.BicycleExtended{}
			reflect.ValueOf(&v).Elem().FieldByName(field).SetUint(raw)
			if got := v.Validate(); len(got) != 0 {
				t.Fatalf("%s %d: %v", field, raw, got)
			}
		}
	}
	for raw := 0; raw < 256; raw++ {
		if got := (rc016.BicycleBasic{DrivePower: uint8(raw)}).Validate(); len(got) != 0 {
			t.Fatal(got)
		}
	}
}

func TestPayloadValidationComposition(t *testing.T) {
	if got := (rc016.BicycleData{PersonalCommon: rc016.PersonalCommon{Level: 6}, Basic: rc016.BicycleBasic{Pedaling: 3}, Extended: rc016.BicycleExtended{Reserved: 1}}).Validate(); len(got) != 3 {
		t.Fatal(got)
	}
	if got := (rc016.PedestrianData{PersonalCommon: rc016.PersonalCommon{Level: 6}, Pedestrian: rc016.Pedestrian{ItemInfo: 0, Reserved: 1}}).Validate(); len(got) != 3 {
		t.Fatal(got)
	}
}

func TestV2SentinelConstants(t *testing.T) {
	if rc016.LevelUnavailable != 7 || rc016.SystemDelaySaturated != 30 || rc016.SystemDelayUnavailable != 31 || rc016.ItemInfoUnavailable != 63 || rc016.StepsSaturated != 65534 || rc016.StepsUnavailable != 65535 || rc016.CadenceSaturated != 254 || rc016.CadenceUnavailable != 255 {
		t.Fatal("v2 sentinel values")
	}
}

func TestCategoricalCodes(t *testing.T) {
	for raw := 0; raw < 8; raw++ {
		want := 0
		if raw == 0 || raw == 6 {
			want = 1
		}
		if got := (rc016.PersonalCommon{Level: uint8(raw), WatchData: 0xffffffff}).Validate(); len(got) != want {
			t.Fatalf("level %d: %v", raw, got)
		}
	}
	for raw := 0; raw < 16; raw++ {
		want := 0
		if raw > 7 {
			want = 1
		}
		if got := (rc016.BicycleBasic{BicycleType: uint8(raw), Collision: uint8(raw)}).Validate(); len(got) != want {
			t.Fatalf("bicycle type %d: %v", raw, got)
		}
		want = 0
		if raw > 2 {
			want = 1
		}
		if got := (rc016.BicycleBasic{AssistType: uint8(raw)}).Validate(); len(got) != want {
			t.Fatalf("assist type %d: %v", raw, got)
		}
	}
	for raw := 0; raw < 4; raw++ {
		want := 0
		if raw == 3 {
			want = 1
		}
		if got := (rc016.BicycleBasic{AssistState: uint8(raw), Pedaling: uint8(raw)}).Validate(); len(got) != want {
			t.Fatalf("basic states %d: %v", raw, got)
		}
		if got := (rc016.BicycleExtended{RearLight: uint8(raw), DriveUnitStatus: uint8(raw), Maintenance: uint8(raw)}).Validate(); len(got) != 3*want {
			t.Fatalf("extended states %d: %v", raw, got)
		}
		if got := (rc016.Pedestrian{ItemInfo: 63, Motion: uint8(raw)}).Validate(); len(got) != 0 {
			t.Fatalf("motion %d: %v", raw, got)
		}
	}
	for raw := 0; raw < 64; raw++ {
		want := 1
		if raw == 1 || raw == 2 || raw == 63 {
			want = 0
		}
		if got := (rc016.Pedestrian{ItemInfo: uint8(raw)}).Validate(); len(got) != want {
			t.Fatalf("item %d: %v", raw, got)
		}
	}
}
