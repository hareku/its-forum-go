package rc016

import (
	"errors"
	"github.com/hareku/its-forum-go/rc013v1"
)

// Error locates a structural codec error.
type Error = rc013.Error

// Issue describes a semantic concern without changing encoded values.
type Issue = rc013.Issue

var (
	ErrTruncated   = rc013.ErrTruncated
	ErrMalformed   = rc013.ErrMalformed
	ErrRange       = rc013.ErrRange
	ErrUnsupported = rc013.ErrUnsupported
	ErrLayout      = rc013.ErrLayout
)

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
