package rc016_test

import (
	"bytes"
	"testing"

	rc016 "github.com/hareku/its-forum-go/rc016v1"
)

func FuzzBicycle(f *testing.F) {
	literal := personalFixture(f, "bicycle")
	f.Add(literal)
	f.Add([]byte{})
	f.Add([]byte{0x29})
	f.Add(literal[:36])
	f.Add(literal[:49])
	minimal := bytes.Clone(literal[:36])
	minimal[7] = 0
	f.Add(minimal)
	presentZero := bytes.Clone(minimal)
	presentZero[6] = 30
	presentZero[7] = 0x80
	presentZero = append(presentZero, 0, 0)
	f.Add(presentZero)
	maximal := bytes.Clone(literal)
	maximal[36] = 0xff
	f.Add(maximal)
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 256 {
			return
		}
		for _, p := range []rc016.BicycleProfile{{}, personalProfile()} {
			m, e := rc016.DecodeBicycleWithProfile(data, p)
			if e != nil {
				continue
			}
			encoded, e := m.MarshalBinary()
			if e != nil || !bytes.Equal(encoded, data) {
				t.Fatalf("lossless decode: %x => %x: %v", data, encoded, e)
			}
			repeated, e := rc016.DecodeBicycleWithProfile(encoded, p)
			if e != nil {
				t.Fatal(e)
			}
			encoded, e = repeated.MarshalBinary()
			if e != nil || !bytes.Equal(encoded, data) {
				t.Fatalf("unstable decode: %v", e)
			}
		}
	})
}
