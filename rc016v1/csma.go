package rc016

import (
	"encoding/binary"
	"fmt"

	"github.com/hareku/its-forum-go/rc013v1"
)

// CSMAHeader is the separate twenty-byte section-4.5 header. PayloadSize is an
// observation on decode and is derived from Objects on encode. IDs are raw
// experiment-selected values; Version occupies four bits.
type CSMAHeader struct {
	ServiceID, Operation, Version, Counter uint8
	MessageID                              uint16
	RoadsideID, IntersectionID             uint32
	Time                                   rc013.Time
	PayloadSize, Reserved                  uint16
}

// CSMAObject is one sixteen-byte compact object. Type and Size are four-bit raw
// values; reserved values remain representable. Coordinates and state use RC-013 units.
type CSMAObject struct {
	ID           uint8
	Latitude     rc013.Latitude
	Longitude    rc013.Longitude
	Speed        rc013.Speed
	Heading      rc013.Heading
	Acceleration rc013.Acceleration
	Type, Size   uint8
}

// CSMAMessage contains zero through five compact objects, with no hidden count,
// trailing extension, or padding. A struct literal is sufficient for construction.
type CSMAMessage struct {
	Header  CSMAHeader
	Objects []CSMAObject
}

// DecodeCSMA decodes exactly one 20+16*N-byte message and owns its object slice.
func DecodeCSMA(data []byte) (CSMAMessage, error) {
	if len(data) < 20 {
		return CSMAMessage{}, &Error{Field: "csma.header", Offset: len(data), Err: ErrTruncated}
	}
	size := int(binary.BigEndian.Uint16(data[16:18]))
	if size > 80 || size%16 != 0 {
		return CSMAMessage{}, &Error{Field: "csma.payloadSize", Offset: 16, Err: ErrMalformed}
	}
	if len(data)-20 < size {
		return CSMAMessage{}, &Error{Field: "csma.payload", Offset: 20, Err: ErrTruncated}
	}
	if len(data)-20 != size {
		return CSMAMessage{}, &Error{Field: "csma.trailing", Offset: 20 + size, Err: ErrMalformed}
	}
	m := CSMAMessage{Header: CSMAHeader{
		ServiceID: data[0] >> 5, Operation: data[0] >> 4 & 1, Version: data[0] & 15, Counter: data[1],
		MessageID: binary.BigEndian.Uint16(data[2:4]), RoadsideID: binary.BigEndian.Uint32(data[4:8]),
		IntersectionID: binary.BigEndian.Uint32(data[8:12]), PayloadSize: uint16(size), Reserved: binary.BigEndian.Uint16(data[18:20]),
	}, Objects: make([]CSMAObject, size/16)}
	if err := m.Header.Time.UnmarshalBinary(data[12:16]); err != nil {
		return CSMAMessage{}, fmt.Errorf("csma.time: %w", rebaseError(err, 12))
	}
	for i := range m.Objects {
		p := data[20+16*i : 20+16*(i+1)]
		m.Objects[i] = CSMAObject{ID: p[0], Latitude: rc013.Latitude(int32(binary.BigEndian.Uint32(p[1:5]))), Longitude: rc013.Longitude(int32(binary.BigEndian.Uint32(p[5:9]))), Speed: rc013.Speed(binary.BigEndian.Uint16(p[9:11])), Heading: rc013.Heading(binary.BigEndian.Uint16(p[11:13])), Acceleration: rc013.Acceleration(int16(binary.BigEndian.Uint16(p[13:15]))), Type: p[15] >> 4, Size: p[15] & 15}
	}
	return m, nil
}

// MarshalBinary derives payload size and returns fresh bytes. Structural widths
// are checked before writing; semantic unknowns and all reserved bits survive.
func (m CSMAMessage) MarshalBinary() ([]byte, error) {
	h := m.Header
	if len(m.Objects) > 5 {
		return nil, &Error{Field: "csma.objects", Offset: -1, Err: ErrRange}
	}
	for _, f := range []struct {
		name       string
		value, max uint8
	}{{"serviceID", h.ServiceID, 7}, {"operation", h.Operation, 1}, {"version", h.Version, 15}} {
		if f.value > f.max {
			return nil, &Error{Field: "csma.header." + f.name, Offset: 0, Err: ErrRange}
		}
	}
	time, err := h.Time.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("csma.time: %w", rebaseError(err, 12))
	}
	for i, o := range m.Objects {
		if o.Type > 15 || o.Size > 15 {
			return nil, &Error{Field: fmt.Sprintf("csma.objects[%d].typeSize", i), Offset: 20 + 16*i + 15, Err: ErrRange}
		}
	}
	out := make([]byte, 20+16*len(m.Objects))
	out[0] = h.ServiceID<<5 | h.Operation<<4 | h.Version
	out[1] = h.Counter
	binary.BigEndian.PutUint16(out[2:4], h.MessageID)
	binary.BigEndian.PutUint32(out[4:8], h.RoadsideID)
	binary.BigEndian.PutUint32(out[8:12], h.IntersectionID)
	copy(out[12:16], time)
	binary.BigEndian.PutUint16(out[16:18], uint16(16*len(m.Objects)))
	binary.BigEndian.PutUint16(out[18:20], h.Reserved)
	for i, o := range m.Objects {
		p := out[20+16*i : 20+16*(i+1)]
		p[0] = o.ID
		binary.BigEndian.PutUint32(p[1:5], uint32(o.Latitude))
		binary.BigEndian.PutUint32(p[5:9], uint32(o.Longitude))
		binary.BigEndian.PutUint16(p[9:11], uint16(o.Speed))
		binary.BigEndian.PutUint16(p[11:13], uint16(o.Heading))
		binary.BigEndian.PutUint16(p[13:15], uint16(o.Acceleration))
		p[15] = o.Type<<4 | o.Size
	}
	return out, nil
}

// Validate reports semantic concerns while preserving every wire value.
func (m CSMAMessage) Validate() []Issue {
	var issues []Issue
	for _, issue := range (rc013.CommonData{Time: m.Header.Time}).Validate() {
		if len(issue.Field) >= 5 && issue.Field[:5] == "time." {
			issue.Field = "csma.header." + issue.Field
			issues = append(issues, issue)
		}
	}
	if m.Header.Version != 1 {
		issues = append(issues, Issue{Field: "csma.header.version", Message: "unassigned message version"})
	}
	if m.Header.Reserved != 0 {
		issues = append(issues, Issue{Field: "csma.header.reserved", Message: "reserved bits are nonzero"})
	}
	for i, o := range m.Objects {
		c := rc013.CommonData{Position: rc013.Position{Latitude: o.Latitude, Longitude: o.Longitude}, VehicleState: rc013.VehicleState{Speed: o.Speed, Heading: o.Heading, Acceleration: o.Acceleration}, VehicleAttributes: rc013.VehicleAttributes{Width: 1, Length: 1}}
		for _, issue := range c.Validate() {
			issue.Field = fmt.Sprintf("csma.objects[%d].%s", i, issue.Field)
			issues = append(issues, issue)
		}
		if o.Type >= 8 && o.Type <= 14 {
			issues = append(issues, Issue{Field: fmt.Sprintf("csma.objects[%d].type", i), Message: "reserved object type"})
		}
	}
	return issues
}
