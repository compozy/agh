package session

import (
	"fmt"
	"slices"
	"strings"
)

// SupportedReasoningEfforts is the canonical ordered enum accepted by session creation.
var SupportedReasoningEfforts = []string{"minimal", "low", "medium", "high", "xhigh"}

// IsSupportedReasoningEffort reports whether value is an accepted reasoning effort.
func IsSupportedReasoningEffort(value string) bool {
	return slices.Contains(SupportedReasoningEfforts, strings.TrimSpace(value))
}

// ValidateReasoningEffort validates one reasoning effort override.
func ValidateReasoningEffort(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || IsSupportedReasoningEffort(trimmed) {
		return nil
	}
	return fmt.Errorf(
		"%w: reasoning_effort must be one of minimal, low, medium, high, xhigh",
		ErrInvalidRuntimeOverride,
	)
}
