package diagnostics

import (
	"regexp"
	"strings"
)

const redactedValue = "[REDACTED]"

var (
	bearerTokenPattern  = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+`)
	quotedSecretPattern = regexp.MustCompile(
		`(?i)(["'])(api[_-]?key|access[_-]?token|refresh[_-]?token|token|secret|password|authorization)(["'])(\s*:\s*)(["'])(?:\\.|[^\\])*?(["'])`,
	)
	secretPattern = regexp.MustCompile(
		`(?i)\b(api[_-]?key|access[_-]?token|refresh[_-]?token|token|secret|password|authorization)\b\s*([:=])\s*("[^"]*"|'[^']*'|[^\s,;]+)`,
	)
)

// Redact removes common credential shapes from diagnostic text before the text
// is persisted or exposed to operators.
func Redact(text string) string {
	if strings.TrimSpace(text) == "" {
		return strings.TrimSpace(text)
	}
	redacted := bearerTokenPattern.ReplaceAllString(text, "Bearer "+redactedValue)
	redacted = quotedSecretPattern.ReplaceAllString(redacted, "${1}${2}${3}${4}${5}"+redactedValue+"${6}")
	return secretPattern.ReplaceAllString(redacted, "${1}${2}"+redactedValue)
}

// RedactAndBound redacts diagnostic text and caps it to a deterministic byte
// budget. Callers should use this before storing crash evidence.
func RedactAndBound(text string, maxBytes int) string {
	redacted := strings.TrimSpace(Redact(text))
	if maxBytes <= 0 {
		return ""
	}
	if len(redacted) <= maxBytes {
		return redacted
	}
	if maxBytes <= len("...[truncated]") {
		return redacted[:maxBytes]
	}
	return redacted[:maxBytes-len("...[truncated]")] + "...[truncated]"
}
