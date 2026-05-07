package modelcatalog

import (
	"errors"
	"fmt"
)

var (
	// ErrAllSourcesFailed reports that refresh could not produce usable rows.
	ErrAllSourcesFailed = errors.New("model catalog: all usable sources failed")
	// ErrSourceDisabled reports that a source is intentionally disabled.
	ErrSourceDisabled = errors.New("model catalog: source disabled")
	// ErrSourceNotRegistered reports that a requested source id is not registered.
	ErrSourceNotRegistered = errors.New("model catalog: source not registered")
)

// StaleFallbackError reports a refresh failure that returned stale fallback rows.
type StaleFallbackError struct {
	SourceID string
	Err      error
}

func (e *StaleFallbackError) Error() string {
	if e == nil {
		return "model catalog: stale fallback"
	}
	return fmt.Sprintf("model catalog: source %q returned stale fallback", e.SourceID)
}

func (e *StaleFallbackError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func sourceErrorText(err error) string {
	if err == nil {
		return ""
	}
	var fallback *StaleFallbackError
	if errors.As(err, &fallback) && fallback.Err != nil {
		return RedactString(fallback.Err.Error())
	}
	return RedactString(err.Error())
}
