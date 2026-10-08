package rc016_test

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/hareku/its-forum-go/rc013v1"
	rc016 "github.com/hareku/its-forum-go/rc016v1"
)

func csmaFields() rc016.CSMAMessage {
	return rc016.CSMAMessage{Header: rc016.CSMAHeader{ServiceID: 5, Operation: 1, Version: 1, Counter: 42, MessageID: 0x1234, RoadsideID: 0x01020304, IntersectionID: 0xa0b0c0d0, Time: rc013.Time{LeapSecondCorrection: true, Hour: 12, Minute: 34, Millisecond: 56789}, Reserved: 0xbeef}, Objects: []rc016.CSMAObject{{ID: 7, Latitude: -350000000, Longitude: 1390000000, Speed: 1234, Heading: 7200, Acceleration: -123, Type: 4, Size: 2}}}
}
func TestCSMAIndependentGolden(t *testing.T) {
	literal := personalFixture(t, "csma")
	decoded, e := rc016.DecodeCSMA(literal)
	if e != nil {
		t.Fatal(e)
	}
	expected := csmaFields()
	expected.Header.PayloadSize = 16
	if !reflect.DeepEqual(decoded, expected) {
		t.Fatalf("decoded %#v %#v", decoded.Header, decoded.Objects)
	}
	constructed := csmaFields()
	constructed.Header.PayloadSize = 65535
	for _, m := range []rc016.CSMAMessage{decoded, constructed} {
		out, e := m.MarshalBinary()
		if e != nil || !bytes.Equal(out, literal) {
			t.Fatalf("golden %x %v", out, e)
		}
	}
	decoded.Objects[0].Type = 6
	want := bytes.Clone(literal)
	want[35] = 0x62
	out, e := decoded.MarshalBinary()
	if e != nil || !bytes.Equal(out, want) {
		t.Fatal("nibble edit", e)
	}
}

func TestCSMACountEditsAndLimits(t *testing.T) {
	literal := personalFixture(t, "csma")
	m := csmaFields()
	object := m.Objects[0]
	for _, count := range []int{0, 1, 5, 0} {
		m.Objects = make([]rc016.CSMAObject, count)
		for i := range m.Objects {
			m.Objects[i] = object
		}
		want := bytes.Clone(literal[:20])
		want[17] = byte(count * 16)
		for i := 0; i < count; i++ {
			want = append(want, literal[20:]...)
		}
		out, e := m.MarshalBinary()
		if e != nil || !bytes.Equal(out, want) {
			t.Fatalf("count %d: %x %v", count, out, e)
		}
		decoded, e := rc016.DecodeCSMA(want)
		if e != nil || len(decoded.Objects) != count || decoded.Header.PayloadSize != uint16(count*16) {
			t.Fatalf("decode count %d %v", count, e)
		}
	}
	m.Objects = make([]rc016.CSMAObject, 6)
	out, e := m.MarshalBinary()
	assertPersonalError(t, e, rc016.ErrRange)
	if out != nil {
		t.Fatal("partial output")
	}
}

func TestCSMANegativeAndAtomicBoundaries(t *testing.T) {
	literal := personalFixture(t, "csma")
	for n := 0; n < len(literal); n++ {
		if _, e := rc016.DecodeCSMA(literal[:n]); e == nil {
			t.Fatalf("accepted prefix %d", n)
		}
	}
	for _, size := range []byte{0, 1, 15, 17, 80, 81, 96, 255} {
		b := bytes.Clone(literal)
		b[17] = size
		if _, e := rc016.DecodeCSMA(b); e == nil {
			t.Fatalf("accepted size %d", size)
		}
	}
	if _, e := rc016.DecodeCSMA(append(bytes.Clone(literal), 0)); e == nil {
		t.Fatal("trailing byte")
	}
	cases := []func(*rc016.CSMAMessage){func(m *rc016.CSMAMessage) { m.Header.ServiceID = 8 }, func(m *rc016.CSMAMessage) { m.Header.Operation = 2 }, func(m *rc016.CSMAMessage) { m.Header.Version = 16 }, func(m *rc016.CSMAMessage) { m.Header.Time.Hour = 128 }, func(m *rc016.CSMAMessage) { m.Objects[0].Type = 16 }, func(m *rc016.CSMAMessage) { m.Objects[0].Size = 16 }}
	for _, mutate := range cases {
		m := csmaFields()
		mutate(&m)
		b, e := m.MarshalBinary()
		assertPersonalError(t, e, rc016.ErrRange)
		if b != nil {
			t.Fatal("partial bytes")
		}
	}
}

func TestCSMAUnknownExtremesAndOwnership(t *testing.T) {
	// Independent all-one header, minimum coordinates/acceleration, and maxima.
	literal := personalHex(t, "ff ff ff ff ff ff ff ff ff ff ff ff ff ff ff ff 00 10 ff ff ff 80 00 00 00 7f ff ff ff ff ff ff ff 80 00 ff")
	m, e := rc016.DecodeCSMA(literal)
	if e != nil {
		t.Fatal(e)
	}
	if m.Header.ServiceID != 7 || m.Header.Version != 15 || m.Header.Time.Hour != 127 || m.Objects[0].Latitude != -2147483648 || m.Objects[0].Longitude != 2147483647 || m.Objects[0].Acceleration != -32768 || m.Objects[0].Size != 15 {
		t.Fatalf("extremes %#v", m)
	}
	before := bytes.Clone(literal)
	literal[0] = 0
	out, e := m.MarshalBinary()
	if e != nil || !bytes.Equal(out, before) {
		t.Fatal("input alias/unknown", e)
	}
	if len(m.Validate()) == 0 {
		t.Fatal("semantic concerns absent")
	}
	out[20] = 0
	out, e = m.MarshalBinary()
	if e != nil || !bytes.Equal(out, before) {
		t.Fatal("output alias/validation mutation", e)
	}
	m.Objects[0].Latitude = 2147483647
	m.Objects[0].Longitude = -2147483648
	m.Objects[0].Acceleration = 32767
	m.Header.Version = 14
	out, e = m.MarshalBinary()
	if e != nil {
		t.Fatal(e)
	}
	want := personalHex(t, "fe ff ff ff ff ff ff ff ff ff ff ff ff ff ff ff 00 10 ff ff ff 7f ff ff ff 80 00 00 00 ff ff ff ff 7f ff ff")
	if !bytes.Equal(out, want) {
		t.Fatalf("signed max edit %x", out)
	}
}

func TestCSMAMarshalErrorPacketOffsets(t *testing.T) {
	for _, tc := range []struct {
		change func(*rc016.CSMAMessage)
		want   int
	}{
		{func(m *rc016.CSMAMessage) { m.Header.Time.Hour = 128 }, 12},
		{func(m *rc016.CSMAMessage) { m.Header.Version = 16 }, 0},
		{func(m *rc016.CSMAMessage) { m.Objects = append(m.Objects, m.Objects[0]); m.Objects[1].Size = 16 }, 51},
	} {
		m := csmaFields()
		tc.change(&m)
		out, err := m.MarshalBinary()
		assertPersonalError(t, err, rc016.ErrRange)
		var located *rc016.Error
		if out != nil || !errors.As(err, &located) || located.Offset != tc.want {
			t.Fatalf("offset: %v; want %d", err, tc.want)
		}
	}
	// A minute of 64 fits its byte: report semantics without rejecting or changing it.
	m := csmaFields()
	m.Header.Time.Minute = 64
	out, err := m.MarshalBinary()
	if err != nil || out[13] != 64 {
		t.Fatalf("minute representation: %x %v", out, err)
	}
	found := false
	for _, issue := range m.Validate() {
		if issue.Field == "csma.header.time.minute" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing minute diagnostic")
	}
}
