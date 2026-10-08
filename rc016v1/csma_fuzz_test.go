package rc016_test

import (
	"bytes"
	"testing"

	rc016 "github.com/hareku/its-forum-go/rc016v1"
)

func FuzzCSMA(f *testing.F) {
	literal := personalFixture(f, "csma")
	f.Add(literal)
	f.Add([]byte{})
	f.Add([]byte{0xff})
	f.Add(literal[:20])
	zero := bytes.Clone(literal[:20])
	zero[17] = 0
	f.Add(zero)
	five := bytes.Clone(literal[:20])
	five[17] = 80
	for i := 0; i < 5; i++ {
		five = append(five, literal[20:]...)
	}
	f.Add(five)
	extreme := bytes.Clone(literal)
	extreme[0] = 0xff
	extreme[35] = 0xff
	f.Add(extreme)
	malformed := bytes.Clone(literal)
	malformed[16] = 0xff
	malformed[17] = 0xff
	f.Add(malformed)
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 256 {
			return
		}
		m, e := rc016.DecodeCSMA(data)
		if e != nil {
			return
		}
		encoded, e := m.MarshalBinary()
		if e != nil || !bytes.Equal(encoded, data) {
			t.Fatalf("lossless decode: %x => %x: %v", data, encoded, e)
		}
		repeated, e := rc016.DecodeCSMA(encoded)
		if e != nil {
			t.Fatal(e)
		}
		encoded, e = repeated.MarshalBinary()
		if e != nil || !bytes.Equal(encoded, data) {
			t.Fatalf("unstable decode: %v", e)
		}
	})
}
