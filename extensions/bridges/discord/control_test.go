package main

import (
	"errors"
	"strings"
	"testing"

	bridgepkg "github.com/compozy/agh/internal/bridges"
)

func TestDiscordIdentityChecksValidateTheConfiguredApplication(t *testing.T) {
	t.Run("Should pass for the authenticated configured bot", func(t *testing.T) {
		t.Parallel()

		checks := discordIdentityChecks(&discordBotIdentity{ID: "app-1", Username: "agh"}, "app-1", nil)
		if len(checks) != 1 || checks[0].Status != bridgepkg.BridgeCheckStatusPass {
			t.Fatalf("discordIdentityChecks() = %#v, want one pass", checks)
		}
	})

	t.Run("Should fail an application identity mismatch with config remediation", func(t *testing.T) {
		t.Parallel()

		checks := discordIdentityChecks(&discordBotIdentity{ID: "app-2"}, "app-1", nil)
		if len(checks) != 1 || checks[0].Status != bridgepkg.BridgeCheckStatusFail {
			t.Fatalf("discordIdentityChecks() = %#v, want one fail", checks)
		}
		if !strings.Contains(checks[0].Remediation, "application_id") {
			t.Fatalf("remediation = %q, want application_id", checks[0].Remediation)
		}
	})

	t.Run("Should map authentication failure to the bot token binding", func(t *testing.T) {
		t.Parallel()

		checks := discordIdentityChecks(nil, "app-1", errors.New("unauthorized"))
		if len(checks) != 1 || !strings.Contains(checks[0].Remediation, "bot_token") {
			t.Fatalf("discordIdentityChecks() = %#v, want bot_token remediation", checks)
		}
	})
}
