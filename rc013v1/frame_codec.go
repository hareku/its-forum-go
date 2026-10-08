package rc013

import "github.com/hareku/its-forum-go/rc013v1/internal/bitio"

type frameCodec struct {
	data []byte
	pos  int
	err  error
}

func (c *frameCodec) read(name string, width int) uint64 {
	if c.err != nil {
		return 0
	}
	v, err := bitio.Read(c.data, c.pos, width)
	if err != nil {
		c.err = fieldError(name, c.pos/8, ErrTruncated)
	}
	c.pos += width
	return v
}
func (c *frameCodec) signed(name string, width int) int64 {
	if c.err != nil {
		return 0
	}
	v, err := bitio.ReadSigned(c.data, c.pos, width)
	if err != nil {
		c.err = fieldError(name, c.pos/8, ErrTruncated)
	}
	c.pos += width
	return v
}
func (c *frameCodec) write(name string, width int, value uint64) {
	if c.err != nil {
		return
	}
	if err := bitio.Write(c.data, c.pos, width, value); err != nil {
		c.err = fieldError(name, c.pos/8, ErrRange)
	}
	c.pos += width
}
func (c *frameCodec) writeSigned(name string, width int, value int64) {
	if c.err != nil {
		return
	}
	if err := bitio.WriteSigned(c.data, c.pos, width, value); err != nil {
		c.err = fieldError(name, c.pos/8, ErrRange)
	}
	c.pos += width
}
func exactFrame(data []byte, size int, name string) error {
	if len(data) < size {
		return fieldError(name, len(data), ErrTruncated)
	}
	if len(data) > size {
		return fieldError(name, size, ErrMalformed)
	}
	return nil
}
func boolValue(v bool) uint64 {
	if v {
		return 1
	}
	return 0
}
