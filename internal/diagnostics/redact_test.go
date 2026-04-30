package diagnostics

import (
	"strings"
	"testing"
)

func TestRedactHandlesQuotedJSONSecretsAndBounds(t *testing.T) {
	t.Parallel()

	t.Run("Should redact quoted JSON secret values", func(t *testing.T) {
		t.Parallel()

		redacted := Redact(`{"access_token":"abc","refresh_token":"def","safe":"ok"}`)
		if strings.Contains(redacted, "abc") || strings.Contains(redacted, "def") {
			t.Fatalf("Redact(JSON secrets) = %q, want token material removed", redacted)
		}
		if !strings.Contains(redacted, `"access_token":"[REDACTED]"`) ||
			!strings.Contains(redacted, `"refresh_token":"[REDACTED]"`) ||
			!strings.Contains(redacted, `"safe":"ok"`) {
			t.Fatalf("Redact(JSON secrets) = %q, want quoted redacted values and safe fields preserved", redacted)
		}
	})

	t.Run("Should preserve non JSON secret redaction shape", func(t *testing.T) {
		t.Parallel()

		if got, want := Redact("token=super-secret"), "token=[REDACTED]"; got != want {
			t.Fatalf("Redact(token=) = %q, want %q", got, want)
		}
	})

	t.Run("Should redact MCP OAuth PKCE and secret binding values", func(t *testing.T) {
		t.Parallel()

		redacted := Redact(
			`{"mcp_auth_token":"mcp-raw","authorization_code":"code-raw","code_verifier":"verifier-raw","secret_binding":"binding-raw","safe":"ok"} oauth_code=oauth-raw pkce_verifier=pkce-raw`,
		)
		for _, leaked := range []string{
			"mcp-raw",
			"code-raw",
			"verifier-raw",
			"binding-raw",
			"oauth-raw",
			"pkce-raw",
		} {
			if strings.Contains(redacted, leaked) {
				t.Fatalf("Redact(MCP/OAuth secrets) = %q leaked %q", redacted, leaked)
			}
		}
		if !strings.Contains(redacted, `"safe":"ok"`) {
			t.Fatalf("Redact(MCP/OAuth secrets) = %q, want safe field preserved", redacted)
		}
	})

	t.Run("Should keep non positive byte budgets bounded", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name     string
			maxBytes int
		}{
			{name: "Should return empty bounded result for zero bytes", maxBytes: 0},
			{name: "Should return empty bounded result for negative bytes", maxBytes: -1},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				if got := RedactAndBound("token=super-secret", tc.maxBytes); got != "" {
					t.Fatalf("RedactAndBound(maxBytes=%d) = %q, want empty bounded result", tc.maxBytes, got)
				}
			})
		}
	})
}
