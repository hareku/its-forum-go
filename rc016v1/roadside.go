package rc016

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/hareku/its-forum-go/rc013v1"
	"github.com/hareku/its-forum-go/rc016v1/internal/bitio"
)

// RoadsideHeader uses table 4-3's four-bit version and 16-byte header.
// Section 4.4.2.1.1 instead states one bit; no alternate packing is inferred.
type RoadsideHeader struct {
	ServiceID, Operation, Version, Counter uint8
	MessageID                              uint16
	RoadsideID                             uint32
	Time                                   rc013.Time
	MessageSize, Reserved                  uint16
}

// RoadsideFields contains the editable message fields and explicitly opaque suffix.
type RoadsideFields struct {
	Header            RoadsideHeader
	SystemState       uint8
	CommonOptionFlags uint8
	Options           []RoadsideOption
	Groups            []RoadsideObjectGroup
	OpaqueRemainder   []byte
}

// RoadsideMessage retains an immutable interpretation and opaque placement anchors.
// Public slices own their storage. Ordinary struct copies have Go's shallow-copy semantics.
type RoadsideMessage struct {
	RoadsideFields
	profile        RoadsideProfile
	decodedObjects bool
	suffixAnchor   int
	anchored       bool
}

// RoadsideOption is one flag-indexed common option. Index zero owns Sensors.
type RoadsideOption struct {
	Index          uint8
	Data           []byte
	Sensors        []RoadsideSensor
	opaqueStart    int
	opaqueAnchored bool
}

// RoadsideSensor owns raw attributes or the explicit five-byte example layout.
type RoadsideSensor struct {
	Attributes     []byte
	Example        *RoadsideSensorAttributes
	opaqueStart    int
	opaqueAnchored bool
}

// RoadsideSensorAttributes is the optional example in section 4.4.2.2.4.
type RoadsideSensorAttributes struct {
	ID            uint32
	Operation     uint8
	Status        uint16
	Extra         []byte
	extraStart    int
	extraAnchored bool
}

// RoadsideObjectGroup identifies a sensor by index; -1 is the no-sensor group.
type RoadsideObjectGroup struct {
	SensorIndex int
	Objects     []RoadsideObject
}

// RoadsideObject reuses RC-013 common frames with roadside management semantics.
type RoadsideObject struct {
	ServiceID, MessageID, Version, Counter uint8
	ID                                     uint32
	Common                                 rc013.CommonData
	Extensions                             []RoadsideObjectExtension
	commonAnchor                           int
	commonAnchored                         bool
}

// RoadsideObjectExtension retains gaps and unassigned payload bits.
// Mapped Fields override known Data bits; Data supplies only padding and unknown suffixes.
type RoadsideObjectExtension struct {
	ServiceID uint8
	Gap, Data []byte
	Fields    *ObjectExtensionFields
	anchor    int
	anchored  bool
}

func roadsideError(field string, offset int, err error) error {
	return &Error{Field: "roadside." + field, Offset: offset, Err: err}
}

// ObjectsDecoded distinguishes an unresolved suffix from a decoded empty object list.
func (m RoadsideMessage) ObjectsDecoded() bool { return m.decodedObjects }

// DecodeRoadside decodes provable fields without guessing experimental sensor layouts.
func DecodeRoadside(data []byte) (RoadsideMessage, error) {
	return DecodeRoadsideWithProfile(data, RoadsideProfile{})
}

// DecodeRoadsideWithProfile snapshots the caller-selected experiment interpretation.
func DecodeRoadsideWithProfile(data []byte, p RoadsideProfile) (RoadsideMessage, error) {
	q, err := freezeRoadsideProfile(p)
	if err != nil {
		return RoadsideMessage{}, err
	}
	m, err := decodeRoadside(data, q)
	if err != nil {
		return RoadsideMessage{}, err
	}
	return m, nil
}
func decodeRoadside(data []byte, p RoadsideProfile) (RoadsideMessage, error) {
	m := RoadsideMessage{profile: p}
	if len(data) < 17 {
		return m, roadsideError("header", len(data), ErrTruncated)
	}
	h := &m.Header
	h.ServiceID = data[0] >> 5
	h.Operation = (data[0] >> 4) & 1
	h.Version = data[0] & 15
	h.Counter = data[1]
	h.MessageID = binary.BigEndian.Uint16(data[2:4])
	h.RoadsideID = binary.BigEndian.Uint32(data[4:8])
	if err := h.Time.UnmarshalBinary(data[8:12]); err != nil {
		return m, rebaseError(err, 8)
	}
	h.MessageSize = binary.BigEndian.Uint16(data[12:14])
	h.Reserved = binary.BigEndian.Uint16(data[14:16])
	if len(data) != 16+int(h.MessageSize) {
		return m, roadsideError("size", 12, ErrMalformed)
	}
	m.SystemState = data[16]
	pos := 17
	opaque := func() { m.OpaqueRemainder = bytes.Clone(data[pos:]); m.suffixAnchor = pos; m.anchored = true }
	if m.SystemState == 1 {
		if pos != len(data) {
			return m, roadsideError("invalid_state", pos, ErrMalformed)
		}
		m.decodedObjects = true
		return m, nil
	}
	if m.SystemState != 0 {
		opaque()
		return m, nil
	}
	if pos == len(data) {
		return m, roadsideError("flags", pos, ErrTruncated)
	}
	m.CommonOptionFlags = data[pos]
	pos++
	if m.CommonOptionFlags&1 != 0 {
		opaque()
		return m, nil
	}
	var sensors []RoadsideSensor
	for index := uint8(0); index < 7; index++ {
		if m.CommonOptionFlags&(0x80>>index) == 0 {
			continue
		}
		if pos >= len(data) {
			return m, roadsideError("option", pos, ErrTruncated)
		}
		n := int(data[pos])
		pos++
		if n == 0 || n > len(data)-pos {
			return m, roadsideError("option.length", pos-1, ErrMalformed)
		}
		o := RoadsideOption{Index: index}
		if index == 0 {
			var err error
			o.Sensors, err = decodeRoadsideSensors(data[pos:pos+n], p, pos)
			if err != nil {
				return m, rebaseError(err, pos)
			}
			sensors = o.Sensors
		} else {
			o.Data = bytes.Clone(data[pos : pos+n])
			o.opaqueStart, o.opaqueAnchored = pos, true
		}
		m.Options = append(m.Options, o)
		pos += n
	}
	order, known, err := roadsideOrder(sensors, m.CommonOptionFlags&0x80 != 0, p)
	if err != nil {
		return m, err
	}
	if !known {
		opaque()
		return m, nil
	}
	for _, i := range order {
		if i >= 0 {
			active, err := roadsideSensorActive(sensors[i], p.Sensors[i])
			if err != nil {
				return m, err
			}
			if !active {
				continue
			}
		}
		if pos >= len(data) {
			return m, roadsideError("object_count", pos, ErrTruncated)
		}
		count := int(data[pos])
		pos++
		if count > (len(data)-pos)/36 {
			return m, roadsideError("objects", pos, ErrTruncated)
		}
		g := RoadsideObjectGroup{SensorIndex: i}
		for j := 0; j < count; j++ {
			o, n, err := decodeRoadsideObject(data[pos:], pos, p)
			if err != nil {
				return m, err
			}
			g.Objects = append(g.Objects, o)
			pos += n
		}
		m.Groups = append(m.Groups, g)
	}
	if pos != len(data) {
		return m, roadsideError("trailing", pos, ErrMalformed)
	}
	m.decodedObjects = true
	return m, nil
}
func decodeRoadsideSensors(data []byte, p RoadsideProfile, absolute int) ([]RoadsideSensor, error) {
	if len(data) < 1 {
		return nil, roadsideError("sensors", 0, ErrTruncated)
	}
	n := int(data[0])
	if n > (len(data)-1)/2 {
		return nil, roadsideError("sensors.count", 0, ErrMalformed)
	}
	sensors := make([]RoadsideSensor, 0, n)
	pos := 1
	for i := 0; i < n; i++ {
		if pos >= len(data) {
			return nil, roadsideError("sensor", pos, ErrTruncated)
		}
		size := int(data[pos])
		pos++
		if size == 0 || size > len(data)-pos {
			return nil, roadsideError("sensor.length", pos-1, ErrMalformed)
		}
		s := RoadsideSensor{Attributes: bytes.Clone(data[pos : pos+size]), opaqueStart: absolute + pos, opaqueAnchored: true}
		if i < len(p.Sensors) && p.Sensors[i].Layout == SensorExample {
			if size < 5 {
				return nil, roadsideError("sensor.example", pos, ErrTruncated)
			}
			s.opaqueAnchored = false
			s.Example = &RoadsideSensorAttributes{extraStart: absolute + pos + 5, extraAnchored: size > 5, ID: uint32(data[pos])<<16 | uint32(data[pos+1])<<8 | uint32(data[pos+2]), Operation: data[pos+3] >> 7, Status: binary.BigEndian.Uint16(data[pos+3:pos+5]) & 0x7fff, Extra: bytes.Clone(data[pos+5 : pos+size])}
			if s.Example.Status != 0 && len(s.Example.Extra) > 0 {
				return nil, roadsideError("sensor.inactive_extra", pos+5, ErrMalformed)
			}
		}
		sensors = append(sensors, s)
		pos += size
	}
	if pos != len(data) {
		return nil, roadsideError("sensor.trailing", pos, ErrMalformed)
	}
	return sensors, nil
}
func roadsideOrder(s []RoadsideSensor, present bool, p RoadsideProfile) ([]int, bool, error) {
	if !present {
		if len(p.Sensors) > 0 || p.GroupOrder != nil {
			return nil, false, roadsideError("profile.sensors", -1, ErrMalformed)
		}
		return []int{-1}, true, nil
	}
	if p.GroupOrder == nil {
		return nil, false, nil
	}
	if len(s) != len(p.Sensors) {
		return nil, false, roadsideError("profile.sensor_count", -1, ErrMalformed)
	}
	return p.GroupOrder, true, nil
}
func roadsideSensorBytes(s RoadsideSensor, p SensorProfile) ([]byte, error) {
	if p.Layout != SensorExample {
		if s.Example != nil {
			return nil, roadsideError("sensor.layout", -1, ErrMalformed)
		}
		return bytes.Clone(s.Attributes), nil
	}
	if s.Example == nil {
		return nil, roadsideError("sensor.example", -1, ErrMalformed)
	}
	v := s.Example
	if v.ID > 0xffffff || v.Operation > 1 || v.Status > 0x7fff {
		return nil, roadsideError("sensor.value", -1, ErrRange)
	}
	if v.Status != 0 && len(v.Extra) > 0 {
		return nil, roadsideError("sensor.inactive_extra", -1, ErrMalformed)
	}
	out := []byte{byte(v.ID >> 16), byte(v.ID >> 8), byte(v.ID), byte(v.Operation<<7) | byte(v.Status>>8), byte(v.Status)}
	return append(out, v.Extra...), nil
}
func roadsideSensorActive(s RoadsideSensor, p SensorProfile) (bool, error) {
	if p.Layout == SensorExample {
		if s.Example == nil {
			return false, roadsideError("sensor.example", -1, ErrMalformed)
		}
		return s.Example.Status == 0, nil
	}
	if p.Status == nil {
		return true, nil
	}
	v, e := bitio.Read(s.Attributes, p.Status.BitOffset, p.Status.Width)
	if e != nil {
		return false, roadsideError("sensor.status", -1, ErrMalformed)
	}
	return v == uint64(p.Status.ActiveValue), nil
}
func decodeRoadsideObject(data []byte, absolute int, p RoadsideProfile) (RoadsideObject, int, error) {
	var o RoadsideObject
	if len(data) < 8 {
		return o, 0, roadsideError("object.header", absolute, ErrTruncated)
	}
	var h rc013.Header
	if err := h.UnmarshalBinary(data[:8]); err != nil {
		return o, 0, rebaseError(err, absolute)
	}
	n := int(h.DataLength)
	if n < 36 || n > len(data) {
		return o, 0, roadsideError("object.length", absolute+6, ErrMalformed)
	}
	o.ServiceID = h.ServiceID
	o.MessageID = h.MessageID
	o.Version = h.Version
	o.ID = h.VehicleID
	o.Counter = h.Counter
	c, _, err := rc013.DecodeCommonData(data[8:n], h.OptionFlags)
	if err != nil {
		return o, 0, rebaseError(err, absolute+8)
	}
	o.Common = c
	if len(c.OpaqueTail) > 0 {
		o.commonAnchor = absolute + 8
		o.commonAnchored = true
	}
	if h.OptionFlags&rc013.OptionFree != 0 {
		f, used, err := rc013.DecodeFreeArea(data[n:])
		if err != nil {
			return o, 0, rebaseError(err, absolute+n)
		}
		offset := absolute + n + 1 + 3*len(f.Entries)
		for _, e := range f.Entries {
			offset += len(e.Gap)
			x := RoadsideObjectExtension{ServiceID: e.ServiceID, Gap: e.Gap, Data: e.Data, anchor: offset}
			if l, ok := p.ObjectExtensions.Layouts[e.ServiceID]; ok {
				x.Fields, err = decodeRoadsideObjectFields(e.Data, l)
				if err != nil {
					return o, 0, rebaseError(err, offset)
				}
				x.anchored = !l.Complete
			} else {
				x.anchored = true
			}
			o.Extensions = append(o.Extensions, x)
			offset += len(e.Data)
		}
		n += used
	}
	return o, n, nil
}

// NewRoadside validates construction and copies all caller-owned profile and field data.
func NewRoadside(f RoadsideFields, p RoadsideProfile) (RoadsideMessage, error) {
	q, err := freezeRoadsideProfile(p)
	if err != nil {
		return RoadsideMessage{}, err
	}
	m := RoadsideMessage{RoadsideFields: roadsideConstructionFields(f), profile: q}
	data, err := m.MarshalBinary()
	if err != nil {
		return RoadsideMessage{}, err
	}
	return DecodeRoadsideWithProfile(data, q)
}

// roadsideConstructionFields drops decoded placement provenance for explicit construction.
// Copy containers before changing private fields so the original model is unaffected.
func roadsideConstructionFields(f RoadsideFields) RoadsideFields {
	f.Options = append([]RoadsideOption(nil), f.Options...)
	for i := range f.Options {
		o := &f.Options[i]
		o.opaqueAnchored = false
		o.Sensors = append([]RoadsideSensor(nil), o.Sensors...)
		for j := range o.Sensors {
			sensor := &o.Sensors[j]
			sensor.opaqueAnchored = false
			if sensor.Example != nil {
				example := *sensor.Example
				example.extraAnchored = false
				sensor.Example = &example
			}
		}
	}
	f.Groups = append([]RoadsideObjectGroup(nil), f.Groups...)
	for i := range f.Groups {
		f.Groups[i].Objects = append([]RoadsideObject(nil), f.Groups[i].Objects...)
		for j := range f.Groups[i].Objects {
			o := &f.Groups[i].Objects[j]
			o.commonAnchored = false
			c := o.Common
			o.Common = rc013.CommonData{Time: c.Time, Position: c.Position, VehicleState: c.VehicleState, VehicleAttributes: c.VehicleAttributes,
				PositionOptional: c.PositionOptional, GPSStatusOptional: c.GPSStatusOptional, PositionAcquisitionOptional: c.PositionAcquisitionOptional,
				VehicleStateOptional: c.VehicleStateOptional, Intersection: c.Intersection, Extension: c.Extension, ExtendedOptions: c.ExtendedOptions, OpaqueTail: c.OpaqueTail}
			o.Extensions = append([]RoadsideObjectExtension(nil), o.Extensions...)
			for k := range o.Extensions {
				o.Extensions[k].anchored = false
			}
		}
	}
	return f
}

// MarshalBinary derives lengths/counts and refuses movement of unresolved content.
func (m RoadsideMessage) MarshalBinary() ([]byte, error) {
	h := m.Header
	if h.ServiceID > 7 || h.Operation > 1 || h.Version > 15 {
		return nil, roadsideError("header.value", -1, ErrRange)
	}
	t, err := h.Time.MarshalBinary()
	if err != nil {
		return nil, rebaseError(err, 8)
	}
	out := make([]byte, 16)
	out[0] = h.ServiceID<<5 | h.Operation<<4 | h.Version
	out[1] = h.Counter
	binary.BigEndian.PutUint16(out[2:4], h.MessageID)
	binary.BigEndian.PutUint32(out[4:8], h.RoadsideID)
	copy(out[8:12], t)
	binary.BigEndian.PutUint16(out[14:16], h.Reserved)
	out = append(out, m.SystemState)
	if m.SystemState == 1 {
		if len(m.Options) > 0 || len(m.Groups) > 0 || len(m.OpaqueRemainder) > 0 || m.CommonOptionFlags != 0 {
			return nil, roadsideError("invalid_state", -1, ErrMalformed)
		}
	} else if m.SystemState != 0 {
		if len(m.Options) > 0 || len(m.Groups) > 0 || m.CommonOptionFlags != 0 {
			return nil, roadsideError("unknown_state", -1, ErrMalformed)
		}
		out, err = m.appendOpaque(out)
	} else if m.CommonOptionFlags&1 != 0 {
		if len(m.Options) > 0 || len(m.Groups) > 0 {
			return nil, roadsideError("extended_options", -1, ErrMalformed)
		}
		out = append(out, m.CommonOptionFlags)
		out, err = m.appendOpaque(out)
	} else {
		flags := uint8(0)
		last := -1
		var sensors []RoadsideSensor
		out = append(out, 0)
		for _, o := range m.Options {
			if o.Index > 6 || int(o.Index) <= last {
				return nil, roadsideError("options.order", -1, ErrMalformed)
			}
			last = int(o.Index)
			flags |= 0x80 >> o.Index
			var d []byte
			if o.Index == 0 {
				if len(o.Data) > 0 || len(o.Sensors) > 255 {
					return nil, roadsideError("sensors", -1, ErrMalformed)
				}
				sensors = o.Sensors
				d = append(d, byte(len(sensors)))
				for i, s := range sensors {
					p := SensorProfile{}
					if i < len(m.profile.Sensors) {
						p = m.profile.Sensors[i]
					}
					start := len(out) + 1 + len(d) + 1
					if s.opaqueAnchored && start != s.opaqueStart {
						return nil, roadsideError("sensor.relocation", -1, ErrLayout)
					}
					if s.Example != nil && s.Example.extraAnchored && start+5 != s.Example.extraStart {
						return nil, roadsideError("sensor.extra_relocation", -1, ErrLayout)
					}
					b, e := roadsideSensorBytes(s, p)
					if e != nil {
						return nil, e
					}
					if len(b) == 0 || len(b) > 255 {
						return nil, roadsideError("sensor.length", -1, ErrRange)
					}
					d = append(d, byte(len(b)))
					d = append(d, b...)
				}
			} else {
				if len(o.Sensors) > 0 {
					return nil, roadsideError("option.sensors", -1, ErrMalformed)
				}
				if o.opaqueAnchored && len(out)+1 != o.opaqueStart {
					return nil, roadsideError("option.relocation", -1, ErrLayout)
				}
				d = o.Data
			}
			if len(d) == 0 || len(d) > 255 {
				return nil, roadsideError("option.length", -1, ErrRange)
			}
			out = append(out, byte(len(d)))
			out = append(out, d...)
		}
		out[17] = flags
		order, known, e := roadsideOrder(sensors, flags&0x80 != 0, m.profile)
		if e != nil {
			return nil, e
		}
		if !known {
			if len(m.Groups) > 0 {
				return nil, roadsideError("unresolved_groups", -1, ErrMalformed)
			}
			out, err = m.appendOpaque(out)
		} else {
			if len(m.OpaqueRemainder) > 0 {
				return nil, roadsideError("opaque_groups", -1, ErrMalformed)
			}
			gidx := 0
			for _, i := range order {
				if i >= 0 {
					active, e := roadsideSensorActive(sensors[i], m.profile.Sensors[i])
					if e != nil {
						return nil, e
					}
					if !active {
						continue
					}
				}
				if gidx >= len(m.Groups) || m.Groups[gidx].SensorIndex != i {
					return nil, roadsideError("groups.order", -1, ErrMalformed)
				}
				g := m.Groups[gidx]
				gidx++
				if len(g.Objects) > 255 {
					return nil, roadsideError("object_count", -1, ErrRange)
				}
				out = append(out, byte(len(g.Objects)))
				for _, o := range g.Objects {
					b, e := marshalRoadsideObject(o, len(out), m.profile)
					if e != nil {
						return nil, e
					}
					out = append(out, b...)
					if len(out) > 65551 {
						return nil, roadsideError("size", -1, ErrRange)
					}
				}
			}
			if gidx != len(m.Groups) {
				return nil, roadsideError("groups.extra", -1, ErrMalformed)
			}
		}
	}
	if err != nil {
		return nil, err
	}
	if len(out)-16 > 65535 {
		return nil, roadsideError("size", -1, ErrRange)
	}
	binary.BigEndian.PutUint16(out[12:14], uint16(len(out)-16))
	return out, nil
}
func (m RoadsideMessage) appendOpaque(out []byte) ([]byte, error) {
	if m.anchored && len(out) != m.suffixAnchor {
		return nil, roadsideError("opaque_relocation", -1, ErrLayout)
	}
	return append(out, m.OpaqueRemainder...), nil
}
func marshalRoadsideObject(o RoadsideObject, absolute int, p RoadsideProfile) ([]byte, error) {
	common, err := o.Common.MarshalBinary()
	if err != nil {
		return nil, rebaseError(err, absolute+8)
	}
	if len(common) > 247 {
		return nil, roadsideError("object.common_length", -1, ErrRange)
	}
	if o.commonAnchored && absolute+8 != o.commonAnchor {
		return nil, roadsideError("object.common_relocation", -1, ErrLayout)
	}
	h := rc013.Header{ServiceID: o.ServiceID, MessageID: o.MessageID, Version: o.Version, VehicleID: o.ID, Counter: o.Counter, DataLength: uint8(8 + len(common)), OptionFlags: o.Common.OptionFlags()}
	if len(o.Extensions) > 0 {
		h.OptionFlags |= rc013.OptionFree
	}
	out, err := h.MarshalBinary()
	if err != nil {
		return nil, rebaseError(err, absolute)
	}
	out = append(out, common...)
	if len(o.Extensions) > 0 {
		if len(o.Extensions) > 7 {
			return nil, roadsideError("extension.count", -1, ErrRange)
		}
		f := rc013.FreeArea{}
		offset := absolute + len(out) + 1 + 3*len(o.Extensions)
		for _, e := range o.Extensions {
			offset += len(e.Gap)
			if e.anchored && offset != e.anchor {
				return nil, roadsideError("extension.relocation", -1, ErrLayout)
			}
			d := e.Data
			if l, ok := p.ObjectExtensions.Layouts[e.ServiceID]; ok {
				d, err = encodeRoadsideObjectFields(e, l)
				if err != nil {
					return nil, rebaseError(err, offset)
				}
			} else if e.Fields != nil {
				return nil, roadsideError("extension.profile", -1, ErrMalformed)
			}
			f.Entries = append(f.Entries, rc013.FreeEntry{ServiceID: e.ServiceID, Gap: e.Gap, Data: d})
			offset += len(d)
		}
		b, e := f.MarshalBinary()
		if e != nil {
			return nil, rebaseError(e, absolute+8+len(common))
		}
		out = append(out, b...)
	}
	return out, nil
}

// Validate reports semantic issues without changing any encoded value.
func (m RoadsideMessage) Validate() []Issue {
	var issues []Issue
	add := func(f, s string) { issues = append(issues, Issue{Field: "roadside." + f, Message: s}) }
	if m.Header.Version != 1 {
		add("version", "unassigned version")
	}
	if m.SystemState > 1 {
		add("system_state", "unassigned system state")
	}
	t := m.Header.Time
	if t.Hour > 23 && t.Hour != 127 {
		add("time.hour", "hour outside defined range")
	}
	if t.Minute > 59 && t.Minute != 255 {
		add("time.minute", "minute outside defined range")
	}
	if t.Millisecond > 60999 && t.Millisecond != 65535 {
		add("time.millisecond", "millisecond outside defined range")
	}
	for _, o := range m.Options {
		if o.Index == 0 && (len(o.Sensors) < 1 || len(o.Sensors) > 7) {
			add("sensors.count", "sensor count outside defined range")
		}
	}
	if !m.decodedObjects {
		add("objects", "object interpretation is incomplete")
	}
	for gi, g := range m.Groups {
		for oi, o := range g.Objects {
			for _, issue := range o.Common.Validate() {
				add(fmt.Sprintf("groups[%d].objects[%d].%s", gi, oi, issue.Field), issue.Message)
			}
			if o.MessageID != 1 || o.Version != 1 {
				add(fmt.Sprintf("groups[%d].objects[%d]", gi, oi), "unassigned object message or version")
			}
		}
	}
	return issues
}
