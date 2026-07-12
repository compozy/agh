package redact

import (
	"regexp"
	"strings"
)

const sensitiveAssignmentKeyPattern = `[A-Za-z0-9_.-]*(?:` +
	`api[_-]?key|token|secret|password|credential|private[_-]?key|authorization|` +
	`oauth[_-]?code|code[_-]?verifier|pkce[_-]?verifier|secret[_-]?(?:binding|ref)` +
	`)[A-Za-z0-9_.-]*`

var (
	sensitiveKeyCompactor      = strings.NewReplacer("_", "", "-", "", ".", "")
	authorizationHeaderPattern = regexp.MustCompile(
		`(?i)\b((?:proxy[-_])?authorization)\b(\s*[=:]\s*)([^\r\n,;]+)`,
	)
	bearerTokenPattern    = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+`)
	claimTokenPattern     = regexp.MustCompile(`\bagh_claim_[A-Za-z0-9_-]+\b`)
	providerTokenPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{8,}\b`),
		regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9_]{8,}\b`),
		regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{8,}\b`),
		regexp.MustCompile(`\bxapp-[A-Za-z0-9-]{8,}\b`),
	}
	quotedAssignmentPattern = regexp.MustCompile(
		`(?i)(["'])(` + sensitiveAssignmentKeyPattern + `)(["'])(\s*:\s*)(["'])(?:\\.|[^\\])*?(["'])`,
	)
	assignmentPattern = regexp.MustCompile(
		`(?i)\b(` + sensitiveAssignmentKeyPattern + `)(\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;]+)`,
	)
	shellFlagPattern = regexp.MustCompile(
		`(?i)(--)(` + sensitiveAssignmentKeyPattern + `)(\s+)("[^"]*"|'[^']*'|[^\s,;]+)`,
	)
	secretReferencePattern = regexp.MustCompile(
		`(?i)\b(?:env:[A-Za-z_][A-Za-z0-9_]*|vault:(?://)?[A-Za-z0-9_./-]+)`,
	)
)

// IsSensitiveKey reports whether a field name belongs to the shared secret taxonomy.
func IsSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.Trim(strings.TrimSpace(key), `"'`))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	if normalized == "" {
		return false
	}
	compact := compactSensitiveKey(normalized)
	switch compact {
	case "tokenpresent", "maxinputtokens", "maxoutputtokens":
		return false
	}

	for _, fragment := range []string{
		"api_key",
		"auth_token",
		"oauth_token",
		"access_token",
		"refresh_token",
		"id_token",
		"mcp_auth_token",
		"claim_token",
		"lease_token",
		"bot_token",
		"oauth_code",
		"authorization_code",
		"oauth_client_secret",
		"client_secret",
		"webhook_secret",
		"code_verifier",
		"pkce_verifier",
		"pkce",
		"bearer",
		"secret_binding",
		"secret_ref",
		"password",
		"credential",
		"private_key",
		"authorization",
	} {
		if strings.Contains(normalized, fragment) ||
			strings.Contains(compact, compactSensitiveKey(fragment)) {
			return true
		}
	}

	return normalized == "token" || normalized == "secret" ||
		strings.HasSuffix(normalized, "_token") ||
		strings.HasSuffix(normalized, "_secret") ||
		strings.HasSuffix(normalized, "_secret_ref") ||
		compact == "token" || compact == "secret" ||
		strings.HasSuffix(compact, "token") ||
		strings.HasSuffix(compact, "secret") ||
		strings.HasSuffix(compact, "secretref")
}

func compactSensitiveKey(key string) string {
	compact := sensitiveKeyCompactor.Replace(key)
	return strings.ToLower(compact)
}
