package rc013

import (
	"errors"
	"fmt"
	"testing"
)

func TestRebaseErrorPreservesDiagnostic(t *testing.T) {
	original := &Error{Field: "field", Offset: 10, Err: ErrRange}
	rebased := rebaseError(original, 4)
	var got *Error
	if !errors.As(rebased, &got) || got.Offset != 14 || got.Field != original.Field || got.Err != original.Err || !errors.Is(rebased, ErrRange) {
		t.Fatalf("rebased error: %#v", got)
	}
	if original.Offset != 10 || got == original {
		t.Fatal("original error was mutated")
	}
	negative := &Error{Field: "construction", Offset: -1, Err: ErrLayout}
	if rebaseError(negative, 100) != negative {
		t.Fatal("negative construction offset changed")
	}
	if rebaseError(nil, 100) != nil {
		t.Fatal("nil error changed")
	}
	plain := errors.New("plain")
	if rebaseError(plain, 100) != plain {
		t.Fatal("unlocated error changed")
	}
	wrapped := fmt.Errorf("outer context: %w", original)
	if gotErr := rebaseError(wrapped, 8); !errors.As(gotErr, &got) || got.Offset != 18 || !errors.Is(gotErr, ErrRange) {
		t.Fatal("wrapped diagnostic", gotErr)
	}
}
