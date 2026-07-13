package workspace

import (
	"context"
	"time"
)

// CoordinationSettings owns the persisted workspace opt-in consulted by
// coordinated execution resolution.
type CoordinationSettings interface {
	Get(ctx context.Context, workspaceID string) (CoordinationSetting, error)
	Set(
		ctx context.Context,
		workspaceID string,
		enabled bool,
		actor string,
	) (CoordinationSetting, error)
}

// CoordinationSetting is the workspace-scoped, revisioned coordination state.
type CoordinationSetting struct {
	WorkspaceID string    `json:"workspace_id"`
	Enabled     bool      `json:"enabled"`
	Revision    int64     `json:"revision"`
	UpdatedAt   time.Time `json:"updated_at"`
	UpdatedBy   string    `json:"updated_by"`
}
