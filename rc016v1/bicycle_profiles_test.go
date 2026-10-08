package rc016_test

import (
	"bytes"
	"encoding"
	"math/big"
	"reflect"
	"testing"

	rc016 "github.com/hareku/its-forum-go/rc016v1"
)

func TestPersonalFixedFramesEveryField(t *testing.T) {
	tests := []struct {
		value  any
		size   int
		names  []string
		widths []int
	}{
		{rc016.PersonalCommon{}, 5, []string{"Level", "SystemDelay", "WatchData"}, []int{3, 5, 32}},
		{rc016.BicycleBasic{}, 3, []string{"AssistType", "BicycleType", "AssistState", "Pedaling", "DrivePower", "Collision"}, []int{4, 4, 2, 2, 8, 4}},
		{rc016.BicycleExtended{}, 14, []string{"MainGear", "MainMaxGear", "SubGear", "SubMaxGear", "TireCircumference", "Cadence", "GearRatio", "RiderTorque", "MotorTorque", "MaxAssistPower", "AssistPower", "HumanPower", "MaxBattery", "Battery", "RearLight", "DriveUnitStatus", "Maintenance", "Reserved"}, []int{5, 5, 5, 5, 8, 8, 10, 8, 8, 8, 8, 8, 8, 8, 2, 2, 2, 4}},
		{rc016.Pedestrian{}, 5, []string{"ShoeAttribute", "Steps", "Motion", "Reserved"}, []int{6, 14, 2, 18}},
	}
	for _, tc := range tests {
		t.Run(reflect.TypeOf(tc.value).Name(), func(t *testing.T) {
			typ := reflect.TypeOf(tc.value)
			offset := 0
			for i, name := range tc.names {
				width := tc.widths[i]
				max := uint64(1)<<width - 1
				for _, value := range []uint64{0, 1, max} {
					src := reflect.New(typ)
					src.Elem().FieldByName(name).SetUint(value)
					// Independent integer placement, without the production bit helper.
					integer := new(big.Int).Lsh(new(big.Int).SetUint64(value), uint(tc.size*8-offset-width))
					literal := make([]byte, tc.size)
					integer.FillBytes(literal)
					got, e := src.Interface().(encoding.BinaryMarshaler).MarshalBinary()
					if e != nil || !bytes.Equal(got, literal) {
						t.Fatalf("%s=%d encode %x want %x: %v", name, value, got, literal, e)
					}
					dst := reflect.New(typ)
					if e = dst.Interface().(encoding.BinaryUnmarshaler).UnmarshalBinary(literal); e != nil || !reflect.DeepEqual(dst.Elem().Interface(), src.Elem().Interface()) {
						t.Fatalf("%s decode %v %v", name, dst.Elem(), e)
					}
				}
				if width < int(typ.Field(i).Type.Bits()) {
					bad := reflect.New(typ)
					bad.Elem().FieldByName(name).SetUint(max + 1)
					b, e := bad.Interface().(encoding.BinaryMarshaler).MarshalBinary()
					assertPersonalError(t, e, rc016.ErrRange)
					if b != nil {
						t.Fatal("partial output")
					}
				}
				offset += width
			}
			if offset != tc.size*8 {
				t.Fatal("test omitted bits")
			}
			all := bytes.Repeat([]byte{0xff}, tc.size)
			dst := reflect.New(typ)
			if e := dst.Interface().(encoding.BinaryUnmarshaler).UnmarshalBinary(all); e != nil {
				t.Fatal(e)
			}
			for i, name := range tc.names {
				if dst.Elem().FieldByName(name).Uint() != (uint64(1)<<tc.widths[i])-1 {
					t.Fatal(name)
				}
			}
			out, e := dst.Interface().(encoding.BinaryMarshaler).MarshalBinary()
			if e != nil || !bytes.Equal(out, all) {
				t.Fatal("unknown/sentinel/reserved preservation", e)
			}
			before := dst.Elem().Interface()
			for n := 0; n < tc.size; n++ {
				e := dst.Interface().(encoding.BinaryUnmarshaler).UnmarshalBinary(all[:n])
				assertPersonalError(t, e, rc016.ErrTruncated)
				if !reflect.DeepEqual(before, dst.Elem().Interface()) {
					t.Fatal("non-atomic decode")
				}
			}
			e = dst.Interface().(encoding.BinaryUnmarshaler).UnmarshalBinary(append(all, 0))
			assertPersonalError(t, e, rc016.ErrMalformed)
			if !reflect.DeepEqual(before, dst.Elem().Interface()) {
				t.Fatal("non-atomic long decode")
			}
		})
	}
}

func TestPersonalSemanticValidationPreservesBits(t *testing.T) {
	m, e := rc016.NewBicycle(personalFields(), personalProfile())
	if e != nil {
		t.Fatal(e)
	}
	m.Applications[0].PersonalCommon.Level = 7
	m.Applications[1].Basic.AssistType = 15
	before, e := m.MarshalBinary()
	if e != nil {
		t.Fatal(e)
	}
	if len(m.Validate()) == 0 {
		t.Fatal("expected reserved/unassigned diagnostics")
	}
	after, e := m.MarshalBinary()
	if e != nil || !bytes.Equal(before, after) {
		t.Fatal("validation changed bytes")
	}
}
