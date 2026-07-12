// Suite: bridge control contracts
// Invariant: control requests and public check records remain closed, actionable, and secret-free.
// Boundary IN: daemon and provider control-runtime payloads.
// Boundary OUT: provider HTTP behavior, owned by provider suites.
package bridges

import (
	"strings"
	"testing"
)

func TestBridgeControlContractsValidateClosedPayloads(t *testing.T) {
	t.Parallel()

	t.Run("Should accept every public check status", func(t *testing.T) {
		t.Parallel()

		for _, status := range []BridgeCheckStatus{
			BridgeCheckStatusPass,
			BridgeCheckStatusWarn,
			BridgeCheckStatusFail,
			BridgeCheckStatusSkipped,
		} {
			status := status
			t.Run(string(status), func(t *testing.T) {
				t.Parallel()

				record := BridgeCheckRecord{
					Check:       "provider.identity",
					Status:      status,
					Remediation: "Review the provider configuration.",
				}
				if err := record.Validate(); err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
			})
		}
	})

	t.Run("Should reject an unknown check status", func(t *testing.T) {
		t.Parallel()

		err := (BridgeCheckRecord{
			Check:       "provider.identity",
			Status:      BridgeCheckStatus("unknown"),
			Remediation: "Review the provider configuration.",
		}).Validate()
		if err == nil || !strings.Contains(err.Error(), "status") {
			t.Fatalf("Validate() error = %v, want closed status error", err)
		}
	})

	t.Run("Should require remediation for a non-passing check", func(t *testing.T) {
		t.Parallel()

		err := (BridgeCheckRecord{
			Check:  "provider.identity",
			Status: BridgeCheckStatusFail,
		}).Validate()
		if err == nil || !strings.Contains(err.Error(), "remediation") {
			t.Fatalf("Validate() error = %v, want remediation error", err)
		}
	})

	t.Run("Should reject secret-bearing remediation", func(t *testing.T) {
		t.Parallel()

		err := (BridgeCheckRecord{
			Check:       "provider.identity",
			Status:      BridgeCheckStatusFail,
			Remediation: "Replace xoxb-secret-token before retrying.",
		}).Validate()
		if err == nil || !strings.Contains(err.Error(), "sensitive") {
			t.Fatalf("Validate() error = %v, want sensitive remediation error", err)
		}
	})

	t.Run("Should require at least one valid response check", func(t *testing.T) {
		t.Parallel()

		if err := (BridgeCheckResponse{}).Validate(); err == nil {
			t.Fatal("Validate() error = nil, want checks required error")
		}
		response := BridgeCheckResponse{Checks: []BridgeCheckRecord{PassCheck("provider.identity")}}
		if err := response.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("Should validate check and webhook instance identity", func(t *testing.T) {
		t.Parallel()

		if err := (BridgeCheckRequest{}).Validate(); err == nil {
			t.Fatal("BridgeCheckRequest.Validate() error = nil, want instance id error")
		}
		if err := (BridgeWebhookRegistrationRequest{}).Validate(); err == nil {
			t.Fatal("BridgeWebhookRegistrationRequest.Validate() error = nil, want instance id error")
		}
	})

	t.Run("Should validate webhook registration response status", func(t *testing.T) {
		t.Parallel()

		response := BridgeWebhookRegistrationResponse{
			Status:      BridgeCheckStatusPass,
			Remediation: "",
		}
		if err := response.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		response.Status = BridgeCheckStatusSkipped
		if err := response.Validate(); err == nil {
			t.Fatal("Validate() error = nil, want skipped remediation error")
		}
	})
}

func TestBridgeCheckRecordHelpersDoNotCarryProviderErrors(t *testing.T) {
	t.Parallel()

	t.Run("Should build a passing record without diagnostic prose", func(t *testing.T) {
		t.Parallel()

		got := PassCheck("provider.identity")
		if got.Status != BridgeCheckStatusPass || got.Remediation != "" {
			t.Fatalf("PassCheck() = %#v, want pass with empty remediation", got)
		}
	})

	t.Run("Should build an actionable skipped record", func(t *testing.T) {
		t.Parallel()

		got := SkippedCheck("webhook.reachability", "Enable the bridge and retry.")
		if got.Status != BridgeCheckStatusSkipped || got.Remediation == "" {
			t.Fatalf("SkippedCheck() = %#v, want actionable skipped record", got)
		}
	})

	t.Run("Should identify a missing binding without accepting a raw error", func(t *testing.T) {
		t.Parallel()

		got := FailedSecretCheck("provider.identity", "bot_token")
		if got.Status != BridgeCheckStatusFail || !strings.Contains(got.Remediation, "bot_token") {
			t.Fatalf("FailedSecretCheck() = %#v, want binding remediation", got)
		}
	})
}
