package modelcatalog

import (
	"strings"
	"testing"
)

func TestRedactString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		secrets []string
	}{
		{
			name:    "Should redact OpenAI style API keys",
			input:   "models.dev failed with api_key=sk-super-secret-token-123",
			secrets: []string{"sk-super-secret-token-123"},
		},
		{
			name:    "Should redact OAuth bearer tokens",
			input:   "provider returned Authorization: Bearer ya29.secret-oauth-token",
			secrets: []string{"ya29.secret-oauth-token"},
		},
		{
			name:    "Should redact secret shaped environment values",
			input:   "discovery failed with OPENAI_API_KEY=env-secret-value CLIENT_SECRET=client-secret-value",
			secrets: []string{"env-secret-value", "client-secret-value"},
		},
		{
			name:    "Should redact OAuth token environment values",
			input:   "extension failed with OAUTH_TOKEN=oauth-secret-value",
			secrets: []string{"oauth-secret-value"},
		},
		{
			name:    "Should redact colon-delimited secret values",
			input:   "models.dev failed with api_key: sk-colon-secret-token client_secret: colon-client-secret",
			secrets: []string{"sk-colon-secret-token", "colon-client-secret"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			redacted := RedactString(tc.input)
			for _, secret := range tc.secrets {
				if strings.Contains(redacted, secret) {
					t.Fatalf("RedactString() = %q, want secret removed: %q", redacted, secret)
				}
			}
			if !strings.Contains(redacted, "[REDACTED]") {
				t.Fatalf("RedactString() = %q, want redaction marker", redacted)
			}
		})
	}
}
