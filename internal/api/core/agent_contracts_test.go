package core_test

import (
	"testing"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/api/core"
	aghconfig "github.com/compozy/agh/internal/config"
)

func TestCoordinatorConfigPayloadFromConfig(t *testing.T) {
	t.Parallel()

	baseConfig := aghconfig.CoordinatorConfig{
		Enabled:               true,
		AgentName:             " coordinator ",
		Provider:              " codex ",
		Model:                 " gpt-5.4 ",
		DefaultTTL:            90 * time.Minute,
		MaxChildren:           5,
		MaxActivePerWorkspace: 1,
	}

	tests := []struct {
		name        string
		cfg         aghconfig.CoordinatorConfig
		source      contract.CoordinatorConfigSource
		workspaceID string
		assert      func(*testing.T, contract.CoordinatorConfigPayload)
	}{
		{
			name:        "Should trim coordinator string fields",
			cfg:         baseConfig,
			source:      contract.CoordinatorConfigSourceWorkspace,
			workspaceID: " ws-1 ",
			assert: func(t *testing.T, payload contract.CoordinatorConfigPayload) {
				t.Helper()
				if payload.AgentName != "coordinator" || payload.Provider != "codex" || payload.Model != "gpt-5.4" {
					t.Fatalf("trimmed fields = %q/%q/%q", payload.AgentName, payload.Provider, payload.Model)
				}
			},
		},
		{
			name:        "Should convert default TTL to seconds",
			cfg:         baseConfig,
			source:      contract.CoordinatorConfigSourceWorkspace,
			workspaceID: "ws-1",
			assert: func(t *testing.T, payload contract.CoordinatorConfigPayload) {
				t.Helper()
				if payload.DefaultTTLSeconds != 5400 {
					t.Fatalf("DefaultTTLSeconds = %d, want 5400", payload.DefaultTTLSeconds)
				}
			},
		},
		{
			name:        "Should map coordinator limits",
			cfg:         baseConfig,
			source:      contract.CoordinatorConfigSourceWorkspace,
			workspaceID: "ws-1",
			assert: func(t *testing.T, payload contract.CoordinatorConfigPayload) {
				t.Helper()
				if payload.MaxChildren != 5 || payload.MaxActivePerWorkspace != 1 {
					t.Fatalf("limits = %d/%d, want 5/1", payload.MaxChildren, payload.MaxActivePerWorkspace)
				}
			},
		},
		{
			name:        "Should preserve disabled configs",
			cfg:         aghconfig.CoordinatorConfig{Enabled: false},
			source:      contract.CoordinatorConfigSourceDefault,
			workspaceID: "",
			assert: func(t *testing.T, payload contract.CoordinatorConfigPayload) {
				t.Helper()
				if payload.Enabled {
					t.Fatal("Enabled = true, want false")
				}
			},
		},
		{
			name:        "Should trim source workspace id",
			cfg:         baseConfig,
			source:      contract.CoordinatorConfigSourceWorkspace,
			workspaceID: " ws-1 ",
			assert: func(t *testing.T, payload contract.CoordinatorConfigPayload) {
				t.Helper()
				if payload.Source != contract.CoordinatorConfigSourceWorkspace || payload.WorkspaceID != "ws-1" {
					t.Fatalf("source/workspace = %q/%q, want workspace/ws-1", payload.Source, payload.WorkspaceID)
				}
			},
		},
		{
			name:        "Should keep empty workspace id empty",
			cfg:         baseConfig,
			source:      contract.CoordinatorConfigSourceDefault,
			workspaceID: " ",
			assert: func(t *testing.T, payload contract.CoordinatorConfigPayload) {
				t.Helper()
				if payload.WorkspaceID != "" {
					t.Fatalf("WorkspaceID = %q, want empty", payload.WorkspaceID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			payload := core.CoordinatorConfigPayloadFromConfig(tt.cfg, tt.source, tt.workspaceID)
			tt.assert(t, payload)
		})
	}
}
