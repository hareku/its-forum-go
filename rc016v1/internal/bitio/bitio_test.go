package bitio

import (
	"bytes"
	"math"
	"testing"
)

func TestIndependentFields(t *testing.T) {
	data := []byte{0xb5, 0x2e, 0x81}
	cases := []struct {
		off, width int
		want       uint64
	}{{0, 1, 1}, {0, 8, 0xb5}, {3, 10, 0x2a5}, {8, 8, 0x2e}, {16, 8, 0x81}, {23, 1, 1}}
	for _, tc := range cases {
		v, err := Read(data, tc.off, tc.width)
		if err != nil || v != tc.want {
			t.Errorf("Read(%d,%d)=%x,%v want%x", tc.off, tc.width, v, err, tc.want)
		}
	}
	out := []byte{0xff, 0xff, 0xff}
	if err := Write(out, 3, 10, 0x2a5); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, []byte{0xf5, 0x2f, 0xff}) {
		t.Fatalf("cross-boundary write %x", out)
	}
	if err := Write(out, 23, 1, 0); err != nil {
		t.Fatal(err)
	}
	if out[2] != 0xfe {
		t.Fatal(out)
	}
}

func TestSignedAnd64Bits(t *testing.T) {
	for _, tc := range []struct {
		width int
		value int64
		want  []byte
	}{{12, -2048, []byte{0x80, 0}}, {12, 2047, []byte{0x7f, 0xf0}}, {16, -1, []byte{0xff, 0xff}}, {64, math.MinInt64, []byte{0x80, 0, 0, 0, 0, 0, 0, 0}}, {64, math.MaxInt64, []byte{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}}} {
		out := make([]byte, len(tc.want))
		if err := WriteSigned(out, 0, tc.width, tc.value); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(out, tc.want) {
			t.Fatalf("signed%d %x want%x", tc.width, out, tc.want)
		}
		v, err := ReadSigned(tc.want, 0, tc.width)
		if err != nil || v != tc.value {
			t.Fatalf("signed decode %d %v", v, err)
		}
	}
	data := make([]byte, 8)
	if err := Write(data, 0, 64, math.MaxUint64); err != nil {
		t.Fatal(err)
	}
	v, err := Read(data, 0, 64)
	if err != nil || v != math.MaxUint64 {
		t.Fatal(v, err)
	}
}

func TestErrorsAreAtomic(t *testing.T) {
	for _, tc := range []struct {
		off, width int
		value      uint64
	}{{-1, 1, 0}, {0, 0, 0}, {0, 65, 0}, {8, 1, 0}, {7, 2, 0}, {0, 3, 8}, {math.MaxInt, 1, 0}} {
		data := []byte{0xa5}
		if err := Write(data, tc.off, tc.width, tc.value); err == nil {
			t.Fatalf("Write accepted %+v", tc)
		}
		if data[0] != 0xa5 {
			t.Fatal("write changed data on error")
		}
		if tc.value == 0 {
			if _, err := Read(data, tc.off, tc.width); err == nil {
				t.Fatalf("Read accepted %+v", tc)
			}
		}
	}
	for _, v := range []int64{-2049, 2048} {
		data := []byte{0xa5, 0x5a}
		if err := WriteSigned(data, 0, 12, v); err == nil {
			t.Fatal("signed overflow accepted")
		}
		if !bytes.Equal(data, []byte{0xa5, 0x5a}) {
			t.Fatal("signed error changed data")
		}
	}
	if _, err := ReadSigned(nil, 0, 1); err == nil {
		t.Fatal("empty signed read accepted")
	}
}

// On 32-bit systems a valid field can cross MaxInt bits even though every byte
// index fits int. Only the last two bytes are touched; the backing allocation
// is large because this test exercises the real public operations.
func Test32BitOffsetBoundary(t *testing.T) {
	if uint64(^uint(0)>>1) > 0x7fffffff {
		t.Skip("32-bit int boundary")
	}
	data := make([]byte, 268435457)
	data[len(data)-2] = 0xfa
	data[len(data)-1] = 0x5f
	const offset = 2147483644
	v, err := Read(data, offset, 8)
	if err != nil || v != 0xa5 {
		t.Fatal(v, err)
	}
	if err := Write(data, offset, 8, 0x3c); err != nil {
		t.Fatal(err)
	}
	if data[len(data)-2] != 0xf3 || data[len(data)-1] != 0xcf {
		t.Fatal("cross-MaxInt write")
	}
	if err := WriteSigned(data, offset, 8, -1); err != nil {
		t.Fatal(err)
	}
	signed, err := ReadSigned(data, offset, 8)
	if err != nil || signed != -1 {
		t.Fatal(signed, err)
	}
	before := bytes.Clone(data[len(data)-2:])
	if err := Write(data, offset, 64, 0); err == nil {
		t.Fatal("out-of-buffer field accepted")
	}
	if !bytes.Equal(before, data[len(data)-2:]) {
		t.Fatal("invalid write mutated bytes")
	}
}
