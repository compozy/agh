package session

import "testing"

func TestSessionStartEnvFiltersDaemonSecrets(t *testing.T) {
	t.Parallel()

	t.Run("Should remove credential shaped daemon variables and keep AGH session context", func(t *testing.T) {
		t.Parallel()

		env := sessionStartEnv(
			[]string{
				"PATH=/usr/bin",
				"OPENAI_API_KEY=sk-secret",
				"GITHUB_TOKEN=ghp-secret",
				"PROVIDER_HOME=/tmp/provider",
			},
			&Session{
				ID:        "sess-1",
				AgentName: "coder",
				Channel:   "ops",
			},
		)

		if got := envValue(env, "OPENAI_API_KEY"); got != "" {
			t.Fatalf("OPENAI_API_KEY = %q, want filtered", got)
		}
		if got := envValue(env, "GITHUB_TOKEN"); got != "" {
			t.Fatalf("GITHUB_TOKEN = %q, want filtered", got)
		}
		if got := envValue(env, "PROVIDER_HOME"); got != "/tmp/provider" {
			t.Fatalf("PROVIDER_HOME = %q, want %q", got, "/tmp/provider")
		}
		if got := envValue(env, "AGH_SESSION_ID"); got != "sess-1" {
			t.Fatalf("AGH_SESSION_ID = %q, want %q", got, "sess-1")
		}
		if got := envValue(env, "AGH_PEER_ID"); got == "" {
			t.Fatal("AGH_PEER_ID = empty, want network peer id")
		}
	})
}
