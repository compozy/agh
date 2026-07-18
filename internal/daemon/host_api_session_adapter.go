package daemon

import (
	"context"
	"errors"
	"fmt"

	"github.com/compozy/agh/internal/acp"
	"github.com/compozy/agh/internal/session"
)

type hostAPIPromptOptsSessionManager interface {
	PromptWithOpts(
		ctx context.Context,
		id string,
		opts session.PromptOpts,
	) (<-chan acp.AgentEvent, error)
}

var (
	_ hostAPIPromptOptsSessionManager = (*hostAPISessionManagerAdapter)(nil)
	_ hostAPIPromptOptsSessionManager = (*hostAPINetworkSessionManagerAdapter)(nil)
)

// PromptWithOpts forwards provenance-aware prompts to the concrete session manager.
// Bridge ingress relies on this path so Local bridge sessions keep turn_source=network
// metadata without requiring Live NetworkParticipation.
func (a hostAPISessionManagerAdapter) PromptWithOpts(
	ctx context.Context,
	id string,
	opts session.PromptOpts,
) (<-chan acp.AgentEvent, error) {
	prompter, ok := a.SessionManager.(hostAPIPromptOptsSessionManager)
	if !ok {
		return nil, errors.New("daemon: session manager does not support PromptWithOpts")
	}
	events, err := prompter.PromptWithOpts(ctx, id, opts)
	if err != nil {
		return nil, fmt.Errorf("daemon: host api PromptWithOpts: %w", err)
	}
	return events, nil
}
