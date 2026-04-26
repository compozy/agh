package core_test

import (
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/core"
	aghconfig "github.com/pedronauck/agh/internal/config"
)

func TestCoordinatorConfigPayloadFromConfig(t *testing.T) {
	t.Parallel()

	payload := core.CoordinatorConfigPayloadFromConfig(
		aghconfig.CoordinatorConfig{
			Enabled:               true,
			AgentName:             " coordinator ",
			Provider:              " codex ",
			Model:                 " gpt-5.4 ",
			DefaultTTL:            90 * time.Minute,
			MaxChildren:           5,
			MaxActivePerWorkspace: 1,
		},
		contract.CoordinatorConfigSourceWorkspace,
		" ws-1 ",
	)

	if !payload.Enabled ||
		payload.AgentName != "coordinator" ||
		payload.Provider != "codex" ||
		payload.Model != "gpt-5.4" ||
		payload.DefaultTTLSeconds != 5400 ||
		payload.MaxChildren != 5 ||
		payload.MaxActivePerWorkspace != 1 ||
		payload.Source != contract.CoordinatorConfigSourceWorkspace ||
		payload.WorkspaceID != "ws-1" {
		t.Fatalf("CoordinatorConfigPayloadFromConfig() = %#v", payload)
	}
}
