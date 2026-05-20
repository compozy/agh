package diagnostics

import (
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

const redactedValue = "[REDACTED]"
const protectedRedactionMarker = "__AGH_REDACTED"
const truncationSuffix = "...[truncated]"

const minDynamicSecretLength = 8

var (
	sensitiveKeyPattern = strings.Join([]string{
		"api[_-]?key",
		"access[_-]?token",
		"refresh[_-]?token",
		"mcp[_-]?auth[_-]?token",
		"claim[_-]?token",
		"lease[_-]?token",
		"bot[_-]?token",
		"oauth[_-]?code",
		"authorization[_-]?code",
		"oauth[_-]?client[_-]?secret",
		"client[_-]?secret",
		"webhook[_-]?secret",
		"code[_-]?verifier",
		"pkce[_-]?verifier",
		"secret[_-]?binding",
		"token",
		"secret",
		"password",
		"authorization",
	}, "|")
	authorizationHeaderPattern = regexp.MustCompile(
		`(?i)\b((?:proxy[-_])?authorization)\b(\s*[=:]\s*)([^\r\n,;]+)`,
	)
	bearerTokenPattern    = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+`)
	bareClaimTokenPattern = regexp.MustCompile(`\bagh_claim_[A-Za-z0-9_-]+\b`)
	quotedSecretPattern   = regexp.MustCompile(
		`(?i)(["'])(` + sensitiveKeyPattern + `)(["'])(\s*:\s*)(["'])(?:\\.|[^\\])*?(["'])`,
	)
	secretPattern = regexp.MustCompile(
		`(?i)\b(` + sensitiveKeyPattern + `)\b(\s*[=:]\s*)("[^"]*"|'[^']*'|[^\s,;]+)`,
	)
	sensitiveEvidenceKeyPattern = regexp.MustCompile(`(?i)^(?:` + sensitiveKeyPattern + `)$`)
	dynamicSecrets              = newDynamicSecretRegistry()
)

type dynamicSecretRegistry struct {
	mu       sync.Mutex
	values   map[string]int
	snapshot atomic.Value
}

func newDynamicSecretRegistry() *dynamicSecretRegistry {
	registry := &dynamicSecretRegistry{values: make(map[string]int)}
	registry.snapshot.Store([]string(nil))
	return registry
}

// RegisterDynamicSecret registers one runtime-resolved secret for diagnostic redaction.
func RegisterDynamicSecret(value string) func() {
	secret := strings.TrimSpace(value)
	if len(secret) < minDynamicSecretLength {
		return func() {}
	}
	dynamicSecrets.register(secret)

	var once sync.Once
	return func() {
		once.Do(func() {
			dynamicSecrets.unregister(secret)
		})
	}
}

func (r *dynamicSecretRegistry) register(secret string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.values[secret] == 0 {
		r.values[secret] = 1
		r.storeSnapshotLocked()
		return
	}
	r.values[secret]++
}

func (r *dynamicSecretRegistry) unregister(secret string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := r.values[secret]
	if count <= 1 {
		delete(r.values, secret)
		r.storeSnapshotLocked()
		return
	}
	r.values[secret] = count - 1
}

func (r *dynamicSecretRegistry) storeSnapshotLocked() {
	secrets := make([]string, 0, len(r.values))
	for secret := range r.values {
		secrets = append(secrets, secret)
	}
	sortDynamicSecrets(secrets)
	r.snapshot.Store(secrets)
}

// Redact removes common credential shapes from diagnostic text before the text
// is persisted or exposed to operators.
func Redact(text string) string {
	if strings.TrimSpace(text) == "" {
		return strings.TrimSpace(text)
	}
	redacted := redactAuthorizationHeaders(text)
	redacted = bearerTokenPattern.ReplaceAllString(redacted, "Bearer "+redactedValue)
	redacted = bareClaimTokenPattern.ReplaceAllString(redacted, "agh_claim_"+redactedValue)
	redacted = quotedSecretPattern.ReplaceAllString(redacted, "${1}${2}${3}${4}${5}"+redactedValue+"${6}")
	redacted = redactSecretAssignments(redacted)
	return redactDynamicSecrets(redacted)
}

func redactAuthorizationHeaders(text string) string {
	return authorizationHeaderPattern.ReplaceAllStringFunc(text, func(match string) string {
		parts := authorizationHeaderPattern.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}
		if strings.Contains(parts[3], redactedValue) || strings.Contains(parts[3], protectedRedactionMarker) {
			return parts[1] + parts[2] + parts[3]
		}
		return parts[1] + parts[2] + redactedValue
	})
}

func redactSecretAssignments(text string) string {
	return secretPattern.ReplaceAllStringFunc(text, func(match string) string {
		parts := secretPattern.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}
		if strings.Contains(parts[3], redactedValue) || strings.Contains(parts[3], protectedRedactionMarker) {
			return parts[1] + parts[2] + parts[3]
		}
		return parts[1] + parts[2] + redactedValue
	})
}

func redactDynamicSecrets(text string) string {
	secrets := dynamicSecrets.snapshotSecrets()
	for _, secret := range secrets {
		text = strings.ReplaceAll(text, secret, redactedValue)
	}
	return text
}

func (r *dynamicSecretRegistry) snapshotSecrets() []string {
	secrets, ok := r.snapshot.Load().([]string)
	if !ok {
		return nil
	}
	return secrets
}

func sortDynamicSecrets(secrets []string) {
	sort.Slice(secrets, func(i, j int) bool {
		if len(secrets[i]) == len(secrets[j]) {
			return secrets[i] < secrets[j]
		}
		return len(secrets[i]) > len(secrets[j])
	})
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
	if maxBytes <= len(truncationSuffix) {
		return truncateUTF8WithinBytes(redacted, maxBytes)
	}
	return truncateUTF8WithinBytes(redacted, maxBytes-len(truncationSuffix)) + truncationSuffix
}

func truncateUTF8WithinBytes(text string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(text) <= maxBytes {
		return text
	}
	boundary := 0
	for idx := range text {
		if idx > maxBytes {
			break
		}
		boundary = idx
	}
	return text[:boundary]
}
