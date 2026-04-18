package session

import "context"

// StartupPromptContext carries the durable session metadata available during
// startup prompt assembly and overlay selection.
type StartupPromptContext struct {
	SessionID   string
	SessionName string
	AgentName   string
	WorkspaceID string
	Workspace   string
	Channel     string
	SessionType Type
}

// StartupPromptOverlay applies daemon-owned startup prompt overlays after the
// base assembler has produced the startup prompt.
type StartupPromptOverlay interface {
	Apply(ctx context.Context, startup StartupPromptContext, prompt string) (string, error)
}
