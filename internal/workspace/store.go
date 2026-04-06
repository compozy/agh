package workspace

import "context"

// WorkspaceStore persists and looks up registered workspaces.
type WorkspaceStore interface {
	InsertWorkspace(ctx context.Context, ws Workspace) error
	UpdateWorkspace(ctx context.Context, ws Workspace) error
	DeleteWorkspace(ctx context.Context, id string) error
	GetWorkspace(ctx context.Context, id string) (Workspace, error)
	GetWorkspaceByPath(ctx context.Context, rootDir string) (Workspace, error)
	GetWorkspaceByName(ctx context.Context, name string) (Workspace, error)
	ListWorkspaces(ctx context.Context) ([]Workspace, error)
}
