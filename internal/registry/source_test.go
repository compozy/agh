package registry

import (
	"errors"
	"fmt"
	"testing"
)

func TestSourceCapsZeroValue(t *testing.T) {
	t.Parallel()

	var caps SourceCaps
	if caps.Search {
		t.Fatal("SourceCaps zero value Search = true, want false")
	}
}

func TestErrNotSupportedMatchesWrappedError(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf("wrapped: %w", ErrNotSupported)
	if !errors.Is(err, ErrNotSupported) {
		t.Fatalf("errors.Is(%v, ErrNotSupported) = false, want true", err)
	}
}
