package modelcatalog

import (
	"fmt"
	"regexp"
	"strings"
)

var sourceSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// ValidateSourceID checks a stable catalog source identity.
func ValidateSourceID(sourceID string) error {
	trimmed := strings.TrimSpace(sourceID)
	if trimmed == "" {
		return fmt.Errorf("model catalog source id is required")
	}
	if staticSourceKind(trimmed) != "" {
		return nil
	}
	kind, slug, ok := strings.Cut(trimmed, ":")
	if !ok || kind == "" || slug == "" {
		return fmt.Errorf("model catalog source id %q must be static or <kind>:<slug>", sourceID)
	}
	switch SourceKind(kind) {
	case SourceKindProviderLive, SourceKindExtension, SourceKindACPSession:
	default:
		return fmt.Errorf("model catalog source id %q uses unsupported dynamic kind %q", sourceID, kind)
	}
	if !sourceSlugPattern.MatchString(slug) {
		return fmt.Errorf("model catalog source id %q slug must match ^[a-z0-9][a-z0-9_-]*$", sourceID)
	}
	return nil
}

// ValidateSourceIdentity checks that a source id and kind describe the same source family.
func ValidateSourceIdentity(sourceID string, kind SourceKind) error {
	trimmedID := strings.TrimSpace(sourceID)
	if err := ValidateSourceID(trimmedID); err != nil {
		return err
	}
	trimmedKind := SourceKind(strings.TrimSpace(string(kind)))
	if trimmedKind == "" {
		return fmt.Errorf("model catalog source kind is required")
	}
	if staticKind := staticSourceKind(trimmedID); staticKind != "" {
		if staticKind != trimmedKind {
			return fmt.Errorf("model catalog source id %q requires kind %q, got %q", trimmedID, staticKind, trimmedKind)
		}
		return nil
	}
	prefix, _, _ := strings.Cut(trimmedID, ":")
	if SourceKind(prefix) != trimmedKind {
		return fmt.Errorf("model catalog source id %q requires kind %q, got %q", trimmedID, prefix, trimmedKind)
	}
	return nil
}

func staticSourceKind(sourceID string) SourceKind {
	switch sourceID {
	case SourceIDBuiltin:
		return SourceKindBuiltin
	case SourceIDConfig:
		return SourceKindConfig
	case SourceIDModelsDev:
		return SourceKindModelsDev
	default:
		return ""
	}
}
