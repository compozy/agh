package session

import "context"

// PromptProvider returns one workspace-scoped prompt section for composed
// system-prompt assembly.
type PromptProvider interface {
	PromptSection(ctx context.Context, workspace string) (string, error)
}
