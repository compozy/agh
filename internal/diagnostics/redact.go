package diagnostics

import (
	"regexp"
	"sort"
	"strings"
	"sync"
)

const redactedValue = "[REDACTED]"

const sensitiveKeyPattern = `api[_-]?key|access[_-]?token|refresh[_-]?token|mcp[_-]?auth[_-]?token|oauth[_-]?code|authorization[_-]?code|code[_-]?verifier|pkce[_-]?verifier|secret[_-]?binding|token|secret|password|authorization`

const assignmentSensitiveKeyPattern = `api[_-]?key|access[_-]?token|refresh[_-]?token|mcp[_-]?auth[_-]?token|oauth[_-]?code|authorization[_-]?code|code[_-]?verifier|pkce[_-]?verifier|secret[_-]?binding|secret|password|authorization`
const minDynamicSecretLength = 8

var (
	bearerTokenPattern  = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+`)
	quotedSecretPattern = regexp.MustCompile(
		`(?i)(["'])(` + sensitiveKeyPattern + `)(["'])(\s*:\s*)(["'])(?:\\.|[^\\])*?(["'])`,
	)
	secretPattern = regexp.MustCompile(
		`(?i)\b(` + assignmentSensitiveKeyPattern + `)\b\s*([:=])\s*("[^"]*"|'[^']*'|[^\s,;]+)`,
	)
	tokenAssignmentPattern = regexp.MustCompile(`(?i)\b(token)\b\s*(=)\s*("[^"]*"|'[^']*'|[^\s,;]+)`)
	dynamicSecrets         = dynamicSecretRegistry{values: make(map[string]int)}
)

type dynamicSecretRegistry struct {
	mu     sync.RWMutex
	values map[string]int
}

// RegisterDynamicSecret registers one runtime-resolved secret for diagnostic redaction.
func RegisterDynamicSecret(value string) func() {
	secret := strings.TrimSpace(value)
	if len(secret) < minDynamicSecretLength {
		return func() {}
	}
	dynamicSecrets.mu.Lock()
	dynamicSecrets.values[secret]++
	dynamicSecrets.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			dynamicSecrets.mu.Lock()
			defer dynamicSecrets.mu.Unlock()
			count := dynamicSecrets.values[secret]
			if count <= 1 {
				delete(dynamicSecrets.values, secret)
				return
			}
			dynamicSecrets.values[secret] = count - 1
		})
	}
}

// Redact removes common credential shapes from diagnostic text before the text
// is persisted or exposed to operators.
func Redact(text string) string {
	if strings.TrimSpace(text) == "" {
		return strings.TrimSpace(text)
	}
	redacted := bearerTokenPattern.ReplaceAllString(text, "Bearer "+redactedValue)
	redacted = quotedSecretPattern.ReplaceAllString(redacted, "${1}${2}${3}${4}${5}"+redactedValue+"${6}")
	redacted = secretPattern.ReplaceAllString(redacted, "${1}${2}"+redactedValue)
	redacted = tokenAssignmentPattern.ReplaceAllString(redacted, "${1}${2}"+redactedValue)
	return redactDynamicSecrets(redacted)
}

func redactDynamicSecrets(text string) string {
	secrets := dynamicSecretSnapshot()
	for _, secret := range secrets {
		text = strings.ReplaceAll(text, secret, redactedValue)
	}
	return text
}

func dynamicSecretSnapshot() []string {
	dynamicSecrets.mu.RLock()
	defer dynamicSecrets.mu.RUnlock()
	secrets := make([]string, 0, len(dynamicSecrets.values))
	for secret := range dynamicSecrets.values {
		secrets = append(secrets, secret)
	}
	sort.Slice(secrets, func(i, j int) bool {
		if len(secrets[i]) == len(secrets[j]) {
			return secrets[i] < secrets[j]
		}
		return len(secrets[i]) > len(secrets[j])
	})
	return secrets
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
