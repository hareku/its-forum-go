package rc016

import (
	"bytes"
	"testing"
)

func FuzzRoadside(f *testing.F) {
	f.Add(roadsideHex(f, roadsideGolden))
	f.Add(roadsideHex(f, roadsideProfileGolden))
	f.Add([]byte{})
	f.Add([]byte{0})
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > 65551 {
			return
		}
		for _, p := range []RoadsideProfile{{}, roadsideProfile()} {
			m, e := DecodeRoadsideWithProfile(b, p)
			if e != nil {
				continue
			}
			out, e := m.MarshalBinary()
			if e != nil {
				t.Fatalf("accepted input cannot encode: %v", e)
			}
			if !bytes.Equal(out, b) {
				t.Fatalf("roundtrip %x -> %x", b, out)
			}
		}
	})
}
