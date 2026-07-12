package main

import (
	"testing"

	bridgepkg "github.com/compozy/agh/internal/bridges"
)

func TestLinearControlCheckIsExplicitlyUnsupported(t *testing.T) {
	t.Run("Should return a skipped identity record instead of silent absence", func(t *testing.T) {
		t.Parallel()

		check := linearIdentityCheck()
		if check.Check != "provider.identity" || check.Status != bridgepkg.BridgeCheckStatusSkipped {
			t.Fatalf("linearIdentityCheck() = %#v, want explicit skipped identity", check)
		}
		if check.Remediation == "" {
			t.Fatal("linearIdentityCheck() remediation is empty")
		}
	})
}
