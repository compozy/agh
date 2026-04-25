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

	t.Run("Should keep non positive byte budgets bounded", func(t *testing.T) {
		t.Parallel()

		for _, maxBytes := range []int{0, -1} {
			if got := RedactAndBound("token=super-secret", maxBytes); got != "" {
				t.Fatalf("RedactAndBound(maxBytes=%d) = %q, want empty bounded result", maxBytes, got)
			}
		}
	})
}
