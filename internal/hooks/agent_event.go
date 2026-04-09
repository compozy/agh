package hooks

import (
	"context"

	"github.com/pedronauck/agh/internal/acp"
)

// OnAgentEvent remains a no-op until the richer direct runtime integrations
// land in the daemon/session wiring tasks.
func (h *Hooks) OnAgentEvent(_ context.Context, _ string, _ acp.AgentEvent) {}
