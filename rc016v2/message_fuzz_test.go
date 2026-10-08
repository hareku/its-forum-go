package rc016_test

import (
	"bytes"
	"encoding"
	"errors"
	rc016 "github.com/hareku/its-forum-go/rc016v2"
	"reflect"
	"testing"
)

func fuzzMessage(f *testing.F, profile rc016.Profile) {
	for _, name := range []string{"bicycle", "pedestrian"} {
		b := messageFixture(f, name)
		f.Add(b)
		f.Add(b[:len(b)-1])
	}
	b := messageFixture(f, "bicycle")
	b[37] = 0x99
	f.Add(b)
	absent := bytes.Clone(b[:36])
	absent[7] = 0
	f.Add(absent)
	opaque := append(bytes.Clone(absent), 0xde, 0xad)
	opaque[6] = 30
	opaque[7] = 2
	f.Add(opaque)
	gap := append(bytes.Clone(b[:40]), append([]byte{0xa5}, b[40:]...)...)
	gap[38] = 1
	f.Add(gap)
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, input []byte) {
		before := bytes.Clone(input)
		var m rc016.Message
		var err error
		if profile.Applications == nil {
			m, err = rc016.Decode(input)
		} else {
			m, err = rc016.DecodeWithProfile(input, profile)
		}
		if !bytes.Equal(input, before) {
			t.Fatal("decode mutated input")
		}
		if err != nil {
			var located *rc016.Error
			if !errors.As(err, &located) {
				t.Fatalf("unclassified error: %v", err)
			}
			return
		}
		out, err := m.MarshalBinary()
		if err != nil || !bytes.Equal(input, out) {
			t.Fatalf("accepted input not lossless: %x => %x (%v)", input, out, err)
		}
		m.Validate()
		if !bytes.Equal(messageBytes(t, m), input) {
			t.Fatal("validation changed message")
		}
	})
}
func FuzzDecode(f *testing.F)            { fuzzMessage(f, rc016.Profile{}) }
func FuzzDecodeWithProfile(f *testing.F) { fuzzMessage(f, messageProfile()) }
func FuzzFrameRoundTrip(f *testing.F) {
	for _, s := range []string{"a301020304", "239105", "1ac43d25b5644c6ca85c94f0a669", "0448d2abcd", "a3010203042391051ac43d25b5644c6ca85c94f0a669", "a3010203040448d2abcd", "ffffff0000", ""} {
		f.Add(messageHex(f, s))
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		frames := []interface {
			encoding.BinaryMarshaler
			encoding.BinaryUnmarshaler
		}{&rc016.PersonalCommon{Level: 5}, &rc016.BicycleBasic{AssistType: 2}, &rc016.BicycleExtended{Cadence: 91}, &rc016.Pedestrian{Steps: 0x1234}, &rc016.BicycleData{PersonalCommon: rc016.PersonalCommon{Level: 5}}, &rc016.PedestrianData{Pedestrian: rc016.Pedestrian{Steps: 42}}}
		original := bytes.Clone(input)
		for _, frame := range frames {
			before := reflect.ValueOf(frame).Elem().Interface()
			err := frame.UnmarshalBinary(input)
			if !bytes.Equal(input, original) {
				t.Fatal("frame mutated input")
			}
			if err != nil {
				if !reflect.DeepEqual(before, reflect.ValueOf(frame).Elem().Interface()) {
					t.Fatalf("%T receiver changed on error", frame)
				}
				var located *rc016.Error
				if !errors.As(err, &located) {
					t.Fatal(err)
				}
				continue
			}
			out, err := frame.MarshalBinary()
			if err != nil || !bytes.Equal(out, input) {
				t.Fatalf("%T roundtrip %x %v", frame, out, err)
			}
		}
	})
}
