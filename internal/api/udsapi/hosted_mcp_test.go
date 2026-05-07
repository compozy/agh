package udsapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	mcppkg "github.com/pedronauck/agh/internal/mcp"
)

func TestHostedMCPStreamErrorData(t *testing.T) {
	t.Parallel()

	t.Run("Should emit stable stream error without raw backend details", func(t *testing.T) {
		t.Parallel()

		payload := hostedMCPStreamErrorData(
			fmt.Errorf("bind failed for agh_claim_secret: %w", mcppkg.ErrHostedBindNotFound),
		)
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("json.Marshal(hosted MCP stream error) error = %v", err)
		}
		if strings.Contains(string(encoded), "agh_claim_secret") || strings.Contains(string(encoded), "bind failed") {
			t.Fatalf("hosted MCP stream error payload leaked backend detail: %s", encoded)
		}
		if payload.Error != "hosted_mcp_projection_failed" ||
			payload.Status != http.StatusForbidden ||
			payload.Message != http.StatusText(http.StatusForbidden) {
			t.Fatalf("hosted MCP stream error payload = %#v, want stable forbidden error", payload)
		}
	})
}
