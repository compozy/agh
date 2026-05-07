package modelcatalog

import (
	"regexp"
	"strings"
)

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`agh_claim_[A-Za-z0-9_-]+`),
	regexp.MustCompile(`sk-[A-Za-z0-9_-]{8,}`),
	regexp.MustCompile(`gh[pousr]_[A-Za-z0-9_]{8,}`),
	regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{8,}`),
	regexp.MustCompile(`(?i)(api[_-]?key|access[_-]?token|refresh[_-]?token|secret|password|credential)=([^&\s]+)`),
}

// RedactString removes secret-shaped values from catalog errors.
func RedactString(value string) string {
	redacted := value
	for _, pattern := range secretPatterns {
		redacted = pattern.ReplaceAllStringFunc(redacted, redactMatch)
	}
	return redacted
}

func redactMatch(value string) string {
	if key, _, ok := strings.Cut(value, "="); ok {
		return key + "=[REDACTED]"
	}
	return "[REDACTED]"
}
