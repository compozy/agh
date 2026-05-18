package vault

import (
	"strings"
	"testing"
)

func TestSecretLikeEnvName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  string
		want bool
	}{
		{name: "Should reject token values", env: "GITHUB_TOKEN", want: true},
		{name: "Should reject secret values", env: "CLIENT_SECRET", want: true},
		{name: "Should reject API key values", env: "OPENAI_API_KEY", want: true},
		{name: "Should reject JWT private key values", env: "JWT_PRIVATE_KEY", want: true},
		{name: "Should reject SSH private key values", env: "SSH_PRIVATE_KEY", want: true},
		{name: "Should reject GitHub app private key values", env: "GITHUB_APP_PRIVATE_KEY", want: true},
		{name: "Should reject compact private key values", env: "SERVICE_PRIVATEKEY", want: true},
		{name: "Should allow token endpoint URLs", env: "AGH_BRIDGE_LINEAR_TOKEN_URL", want: false},
		{name: "Should allow secret named path variables", env: "AGH_SECRET_GUARD_HOST_CALL_PATH", want: false},
		{name: "Should allow credential file paths", env: "AWS_SHARED_CREDENTIALS_FILE", want: false},
		{name: "Should allow private key file paths", env: "GITHUB_APP_PRIVATE_KEY_FILE", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := SecretLikeEnvName(tc.env); got != tc.want {
				t.Fatalf("SecretLikeEnvName(%q) = %v, want %v", tc.env, got, tc.want)
			}
		})
	}
}

func TestValidateNonSecretEnvMapRejectsPrivateKeyNames(t *testing.T) {
	t.Parallel()

	t.Run("Should reject private key literals in non-secret env maps", func(t *testing.T) {
		t.Parallel()

		err := ValidateNonSecretEnvMap("skill.mcp_servers[0]", map[string]string{
			"GITHUB_APP_PRIVATE_KEY": "-----BEGIN PRIVATE KEY-----",
		})
		if err == nil {
			t.Fatal("ValidateNonSecretEnvMap() error = nil, want private key rejection")
		}
		if !strings.Contains(err.Error(), "GITHUB_APP_PRIVATE_KEY must move secret-like values to secret_env") {
			t.Fatalf("ValidateNonSecretEnvMap() error = %q, want private key guidance", err)
		}
	})
}
