package redact

import (
	"strings"
	"testing"
)

func TestStringRedactsCanonicalSecretTaxonomy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
		leaks []string
	}{
		{
			name: "Should redact provider token shapes",
			input: strings.Join([]string{
				"Bearer bearer-token-value",
				"sk-openai-secret-value",
				"xoxb-slack-secret-value",
				"xapp-slack-app-secret",
				"ghp_githubsecretvalue",
			}, " "),
			leaks: []string{
				"bearer-token-value",
				"sk-openai-secret-value",
				"xoxb-slack-secret-value",
				"xapp-slack-app-secret",
				"ghp_githubsecretvalue",
			},
		},
		{
			name:  "Should redact one-character bearer credentials without consuming delimiters",
			input: "Bearer z, visible",
			want:  "Bearer [REDACTED], visible",
			leaks: []string{"Bearer z"},
		},
		{
			name: "Should redact AGH MCP OAuth PKCE and binding secrets",
			input: strings.Join([]string{
				"agh_claim_raw-claim-value",
				"mcp_auth_token=mcp-token-value",
				"authorization_code=oauth-code-value",
				"code_verifier=pkce-verifier-value",
				"secret_ref=vault:bridges/slack/token",
				"client_secret_ref=env:CLIENT_SECRET",
				"webhook_secret_ref=vault:webhook-secret",
				"workspace_secret=workspace-secret-value",
			}, " "),
			leaks: []string{
				"agh_claim_raw-claim-value",
				"mcp-token-value",
				"oauth-code-value",
				"pkce-verifier-value",
				"vault:bridges/slack/token",
				"env:CLIENT_SECRET",
				"vault:webhook-secret",
				"workspace-secret-value",
			},
		},
		{
			name:  "Should redact quoted generic secret assignments",
			input: `{"api_key":"api-value","auth_token":"auth-value","private_key":"private-value","safe":"visible"}`,
			leaks: []string{"api-value", "auth-value", "private-value"},
		},
		{
			name: "Should redact camel case secret assignments",
			input: `{"apiKey":"api-camel-value","accessToken":"access-camel-value",` +
				`"privateKey":"private-camel-value","codeVerifier":"verifier-camel-value",` +
				`"secretRef":"reference-camel-value"}`,
			leaks: []string{
				"api-camel-value",
				"access-camel-value",
				"private-camel-value",
				"verifier-camel-value",
				"reference-camel-value",
			},
		},
		{
			name:  "Should redact complete authorization headers",
			input: "Proxy-Authorization: Basic dXNlcjpwYXNz",
			leaks: []string{"dXNlcjpwYXNz"},
		},
		{
			name:  "Should redact generic token assignments",
			input: "connect: token=cause-secret",
			leaks: []string{"cause-secret"},
		},
		{
			name: "Should redact shell flag values and bare secret binding references",
			input: "deploy --password hunter2 --secret-ref vault:bridges/slack/token " +
				"--api-key='api-flag-value' env:OPENAI_API_KEY",
			leaks: []string{
				"hunter2",
				"vault:bridges/slack/token",
				"api-flag-value",
				"env:OPENAI_API_KEY",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := String(tc.input)
			if tc.want != "" && got != tc.want {
				t.Fatalf("String() = %q, want %q", got, tc.want)
			}
			for _, leak := range tc.leaks {
				if strings.Contains(got, leak) {
					t.Fatalf("String() = %q leaked %q", got, leak)
				}
			}
			if !strings.Contains(got, Marker) {
				t.Fatalf("String() = %q, want marker %q", got, Marker)
			}
		})
	}
}

func TestStringUsesDynamicSecretsAndRemainsIdempotent(t *testing.T) {
	t.Parallel()

	t.Run("Should redact registered dynamic secrets until cleanup", func(t *testing.T) {
		secret := "runtime-secret-material-123456"
		cleanup := RegisterDynamicSecret(secret)
		t.Cleanup(cleanup)

		got := String("provider leaked " + secret)
		if strings.Contains(got, secret) {
			t.Fatalf("String(dynamic) = %q leaked registered secret", got)
		}
		if twice := String(got); twice != got {
			t.Fatalf("String(String(value)) = %q, want idempotent %q", twice, got)
		}
	})

	t.Run("Should expose the shared sensitive key classifier", func(t *testing.T) {
		t.Parallel()

		for _, key := range []string{
			"api-key",
			"apiKey",
			"mcp_auth_token",
			"accessToken",
			"authorization_code",
			"codeVerifier",
			"PKCE",
			"pkce_verifier",
			"Bearer",
			"client_secret_ref",
			"secretRef",
			"privateKey",
			"workspace_secret",
		} {
			if !IsSensitiveKey(key) {
				t.Fatalf("IsSensitiveKey(%q) = false, want true", key)
			}
		}
		for _, key := range []string{"token_present", "tokenPresent", "maxInputTokens", "maxOutputTokens"} {
			if IsSensitiveKey(key) {
				t.Fatalf("IsSensitiveKey(%q) = true, want public diagnostic key", key)
			}
		}
	})
}
