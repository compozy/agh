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
	regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]{8,}`),
	regexp.MustCompile(
		`(?i)\b([A-Z0-9_-]*(?:api[_-]?key|auth[_-]?token|oauth[_-]?token|access[_-]?token|refresh[_-]?token|id[_-]?token|secret|password|credential|private[_-]?key)[A-Z0-9_-]*)\s*[:=]\s*([^&\s]+)`,
	),
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
	if idx := strings.IndexAny(value, "=:"); idx > 0 {
		key := strings.TrimSpace(value[:idx])
		sep := string(value[idx])
		return key + sep + "[REDACTED]"
	}
	return "[REDACTED]"
}
