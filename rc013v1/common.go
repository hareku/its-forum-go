package rc013

import (
	"bytes"
	"encoding"
)

// Option flag masks follow RC-013 section 6.1.7: index zero is the MSB.
const (
	OptionPosition            uint8 = 0x80
	OptionGPSStatus           uint8 = 0x40
	OptionPositionAcquisition uint8 = 0x20
	OptionVehicleState        uint8 = 0x10
	OptionIntersection        uint8 = 0x08
	OptionExtension           uint8 = 0x04
	OptionExtended            uint8 = 0x02
	OptionFree                uint8 = 0x01
)

// CommonData holds data frames without the eight-byte management header.
// Optional pointers distinguish absence from a present, all-zero data frame.
// OpaqueTail is owned data after known frames, bounded by the declared length.
// A decoded tail's start is pinned; resizing earlier frames returns ErrLayout.
type CommonData struct {
	Time                        Time
	Position                    Position
	VehicleState                VehicleState
	VehicleAttributes           VehicleAttributes
	PositionOptional            *PositionOptional
	GPSStatusOptional           *GPSStatusOptional
	PositionAcquisitionOptional *PositionAcquisitionOptional
	VehicleStateOptional        *VehicleStateOptional
	Intersection                *Intersection
	Extension                   *Extension
	ExtendedOptions             bool
	OpaqueTail                  []byte
	tailStart                   int
	tailAnchored                bool
}

// OptionFlags derives common-frame presence; the enclosing message owns OptionFree.
func (c CommonData) OptionFlags() uint8 {
	var flags uint8
	if c.PositionOptional != nil {
		flags |= OptionPosition
	}
	if c.GPSStatusOptional != nil {
		flags |= OptionGPSStatus
	}
	if c.PositionAcquisitionOptional != nil {
		flags |= OptionPositionAcquisition
	}
	if c.VehicleStateOptional != nil {
		flags |= OptionVehicleState
	}
	if c.Intersection != nil {
		flags |= OptionIntersection
	}
	if c.Extension != nil {
		flags |= OptionExtension
	}
	if c.ExtendedOptions {
		flags |= OptionExtended
	}
	return flags
}

// DecodeCommonData decodes exactly the declared common data span, excluding its
// management header and free area. Consumed equals len(data) on success.
// OptionFree does not change this helper's boundary. No whole-packet size cap applies.
func DecodeCommonData(data []byte, flags uint8) (CommonData, int, error) {
	var c CommonData
	if len(data) < 28 {
		return CommonData{}, 0, fieldError("common", len(data), ErrTruncated)
	}
	pos := 0
	read := func(dst encoding.BinaryUnmarshaler, size int) error {
		if len(data)-pos < size {
			return fieldError("common", pos, ErrTruncated)
		}
		err := rebaseError(dst.UnmarshalBinary(data[pos:pos+size]), pos)
		pos += size
		return err
	}
	mandatory := []struct {
		dst  encoding.BinaryUnmarshaler
		size int
	}{{&c.Time, 4}, {&c.Position, 11}, {&c.VehicleState, 9}, {&c.VehicleAttributes, 4}}
	for _, f := range mandatory {
		if err := read(f.dst, f.size); err != nil {
			return CommonData{}, 0, err
		}
	}
	if flags&OptionPosition != 0 {
		c.PositionOptional = &PositionOptional{}
		if err := read(c.PositionOptional, 2); err != nil {
			return CommonData{}, 0, err
		}
	}
	if flags&OptionGPSStatus != 0 {
		c.GPSStatusOptional = &GPSStatusOptional{}
		if err := read(c.GPSStatusOptional, 4); err != nil {
			return CommonData{}, 0, err
		}
	}
	if flags&OptionPositionAcquisition != 0 {
		c.PositionAcquisitionOptional = &PositionAcquisitionOptional{}
		if err := read(c.PositionAcquisitionOptional, 2); err != nil {
			return CommonData{}, 0, err
		}
	}
	if flags&OptionVehicleState != 0 {
		c.VehicleStateOptional = &VehicleStateOptional{}
		if err := read(c.VehicleStateOptional, 7); err != nil {
			return CommonData{}, 0, err
		}
	}
	if flags&OptionIntersection != 0 {
		c.Intersection = &Intersection{}
		if err := read(c.Intersection, 10); err != nil {
			return CommonData{}, 0, err
		}
	}
	if flags&OptionExtension != 0 {
		c.Extension = &Extension{}
		if err := read(c.Extension, 1); err != nil {
			return CommonData{}, 0, err
		}
	}
	c.ExtendedOptions = flags&OptionExtended != 0
	if c.ExtendedOptions {
		if pos == len(data) {
			return CommonData{}, 0, fieldError("common.extended", pos, ErrTruncated)
		}
		c.OpaqueTail = bytes.Clone(data[pos:])
		c.tailStart = pos
		c.tailAnchored = true
	} else if pos != len(data) {
		return CommonData{}, 0, fieldError("common.trailing", pos, ErrMalformed)
	}
	return c, len(data), nil
}

// MarshalBinary derives DF order and preserves the bounded unknown suffix.
func (c CommonData) MarshalBinary() ([]byte, error) {
	frames := []encoding.BinaryMarshaler{c.Time, c.Position, c.VehicleState, c.VehicleAttributes}
	if c.PositionOptional != nil {
		frames = append(frames, c.PositionOptional)
	}
	if c.GPSStatusOptional != nil {
		frames = append(frames, c.GPSStatusOptional)
	}
	if c.PositionAcquisitionOptional != nil {
		frames = append(frames, c.PositionAcquisitionOptional)
	}
	if c.VehicleStateOptional != nil {
		frames = append(frames, c.VehicleStateOptional)
	}
	if c.Intersection != nil {
		frames = append(frames, c.Intersection)
	}
	if c.Extension != nil {
		frames = append(frames, c.Extension)
	}
	out := make([]byte, 0, 54)
	for _, frame := range frames {
		data, err := frame.MarshalBinary()
		if err != nil {
			return nil, rebaseError(err, len(out))
		}
		out = append(out, data...)
	}
	if c.ExtendedOptions != (len(c.OpaqueTail) > 0) {
		return nil, fieldError("common.extended", len(out), ErrMalformed)
	}
	if len(c.OpaqueTail) > 0 && c.tailAnchored && len(out) != c.tailStart {
		return nil, fieldError("common.extended", len(out), ErrLayout)
	}
	return append(out, c.OpaqueTail...), nil
}
