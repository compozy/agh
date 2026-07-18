package extensionpkg

import (
	"context"

	"github.com/compozy/agh/internal/acp"
	"github.com/compozy/agh/internal/session"
)

func (h *HostAPIHandler) promptBridgeSession(
	ctx context.Context,
	sessionID string,
	message string,
	meta acp.PromptNetworkMeta,
) (<-chan acp.AgentEvent, error) {
	if promptSessions, ok := h.sessions.(hostAPIBridgePromptSessionManager); ok {
		return h.retryBusyBridgePrompt(ctx, sessionID, func() (<-chan acp.AgentEvent, error) {
			return promptSessions.PromptWithOpts(ctx, sessionID, session.PromptOpts{
				Message:    message,
				TurnSource: session.TurnSourceNetwork,
				PromptMeta: acp.PromptMeta{Network: &meta},
			})
		})
	}

	return h.retryBusyBridgePrompt(ctx, sessionID, func() (<-chan acp.AgentEvent, error) {
		return h.sessions.Prompt(ctx, sessionID, message)
	})
}
