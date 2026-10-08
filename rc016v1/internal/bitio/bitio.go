// Package bitio reads and writes MSB-first bit fields.
package bitio

import "fmt"

func check(data []byte, offset, width int) error {
	if width < 1 || width > 64 {
		return fmt.Errorf("bit width %d outside 1..64", width)
	}
	if offset < 0 || offset/8 > len(data) {
		return fmt.Errorf("bit offset %d outside buffer", offset)
	}
	// Subtraction avoids overflow in offset+width or len(data)*8.
	remainingBytes := len(data) - offset/8
	if remainingBytes < (offset%8+width+7)/8 {
		return fmt.Errorf("bit field at %d of width %d exceeds buffer", offset, width)
	}
	return nil
}

// Read returns an unsigned field. Offset zero denotes the first byte's MSB.
func Read(data []byte, offset, width int) (uint64, error) {
	if err := check(data, offset, width); err != nil {
		return 0, err
	}
	var v uint64
	for i := 0; i < width; i++ {
		bit := offset%8 + i
		index := offset/8 + bit/8
		v = v<<1 | uint64((data[index]>>uint(7-bit%8))&1)
	}
	return v, nil
}

// Write changes only the selected field and leaves data unchanged on error.
func Write(data []byte, offset, width int, value uint64) error {
	if err := check(data, offset, width); err != nil {
		return err
	}
	if width < 64 && value>>uint(width) != 0 {
		return fmt.Errorf("value %d exceeds %d bits", value, width)
	}
	for i := 0; i < width; i++ {
		bit := offset%8 + i
		index := offset/8 + bit/8
		mask := byte(1 << uint(7-bit%8))
		data[index] = (data[index] &^ mask) | byte((value>>uint(width-1-i))&1)*mask
	}
	return nil
}

// ReadSigned sign-extends a two's-complement field.
func ReadSigned(data []byte, offset, width int) (int64, error) {
	v, err := Read(data, offset, width)
	if err != nil {
		return 0, err
	}
	if width < 64 && v&(uint64(1)<<uint(width-1)) != 0 {
		v |= ^uint64(0) << uint(width)
	}
	return int64(v), nil
}

// WriteSigned writes a two's-complement field without modifying data on error.
func WriteSigned(data []byte, offset, width int, value int64) error {
	if err := check(data, offset, width); err != nil {
		return err
	}
	if width < 64 {
		limit := int64(1) << uint(width-1)
		if value < -limit || value >= limit {
			return fmt.Errorf("value %d exceeds signed %d bits", value, width)
		}
	}
	v := uint64(value)
	if width < 64 {
		v &= (uint64(1) << uint(width)) - 1
	}
	return Write(data, offset, width, v)
}
