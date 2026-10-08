package rc013

import (
	"errors"
	"fmt"
)

var (
	ErrTruncated   = errors.New("truncated data")
	ErrMalformed   = errors.New("malformed data")
	ErrRange       = errors.New("value outside wire range")
	ErrUnsupported = errors.New("unsupported format")
	ErrLayout      = errors.New("unsafe opaque data relocation")
)

// Error locates a structural codec error. Offset is a byte offset, or -1 for construction.
type Error struct {
	Field  string
	Offset int
	Err    error
}

func (e *Error) Error() string {
	if e.Offset < 0 {
		return fmt.Sprintf("%s: %v", e.Field, e.Err)
	}
	return fmt.Sprintf("%s at byte %d: %v", e.Field, e.Offset, e.Err)
}
func (e *Error) Unwrap() error { return e.Err }

// Issue describes a semantic concern without changing the encoded values.
type Issue struct {
	Field   string
	Message string
}

func fieldError(field string, offset int, err error) error {
	return &Error{Field: field, Offset: offset, Err: err}
}

// rebaseError copies a nested diagnostic into the enclosing byte coordinate system.
// Unknown construction offsets remain negative; the original error is not mutated.
func rebaseError(err error, base int) error {
	var located *Error
	if !errors.As(err, &located) || located.Offset < 0 {
		return err
	}
	next := *located
	next.Offset += base
	return &next
}
