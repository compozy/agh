package modelcatalog

import (
	"strings"
	"testing"
)

func TestRedactString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		secret string
	}{
		{
			name:   "Should redact OpenAI style API keys",
			input:  "models.dev failed with api_key=sk-super-secret-token-123",
			secret: "sk-super-secret-token-123",
		},
		{
			name:   "Should redact OAuth bearer tokens",
			input:  "provider returned Authorization: Bearer ya29.secret-oauth-token",
			secret: "ya29.secret-oauth-token",
		},
		{
			name:   "Should redact secret shaped environment values",
			input:  "discovery failed with OPENAI_API_KEY=env-secret-value CLIENT_SECRET=client-secret-value",
			secret: "env-secret-value",
		},
		{
			name:   "Should redact OAuth token environment values",
			input:  "extension failed with OAUTH_TOKEN=oauth-secret-value",
			secret: "oauth-secret-value",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			redacted := RedactString(tc.input)
			if strings.Contains(redacted, tc.secret) {
				t.Fatalf("RedactString() = %q, want secret removed", redacted)
			}
			if !strings.Contains(redacted, "[REDACTED]") {
				t.Fatalf("RedactString() = %q, want redaction marker", redacted)
			}
		})
	}
}
