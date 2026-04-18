package session

import (
	"context"

	aghconfig "github.com/pedronauck/agh/internal/config"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

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

// StartupPromptAssembler optionally extends PromptAssembler with durable
// startup context so daemon-owned assemblers can select sections before the
// final system prompt is concatenated.
type StartupPromptAssembler interface {
	AssembleStartup(
		ctx context.Context,
		startup StartupPromptContext,
		agent aghconfig.AgentDef,
		workspace *workspacepkg.ResolvedWorkspace,
	) (string, error)
}

// StartupPromptOverlay applies daemon-owned startup prompt overlays after the
// base assembler has produced the startup prompt.
type StartupPromptOverlay interface {
	Apply(ctx context.Context, startup StartupPromptContext, prompt string) (string, error)
}
