package hooks

import "context"

// OnAgentEvent remains a no-op until the richer direct runtime integrations
// land in the daemon/session wiring tasks.
func (h *Hooks) OnAgentEvent(_ context.Context, _ string, _ any) {}
