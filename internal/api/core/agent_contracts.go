package core

import (
	"strings"

	"github.com/compozy/agh/internal/api/contract"
	aghconfig "github.com/compozy/agh/internal/config"
)

// CoordinatorConfigPayloadFromConfig converts resolved coordinator config into a safe read model.
func CoordinatorConfigPayloadFromConfig(
	cfg aghconfig.CoordinatorConfig,
	source contract.CoordinatorConfigSource,
	workspaceID string,
) contract.CoordinatorConfigPayload {
	return contract.CoordinatorConfigPayload{
		Enabled:               cfg.Enabled,
		AgentName:             strings.TrimSpace(cfg.AgentName),
		Provider:              strings.TrimSpace(cfg.Provider),
		Model:                 strings.TrimSpace(cfg.Model),
		DefaultTTLSeconds:     int64(cfg.DefaultTTL.Seconds()),
		MaxChildren:           cfg.MaxChildren,
		MaxActivePerWorkspace: cfg.MaxActivePerWorkspace,
		Source:                source,
		WorkspaceID:           strings.TrimSpace(workspaceID),
	}
}
