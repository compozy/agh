// Package core provides the shared transport-facing API layer used by HTTP and UDS bindings.
package core

import (
	"context"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/transcript"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

// AgentLoader loads one parsed AGENT.md definition.
type AgentLoader func(name string, homePaths aghconfig.HomePaths) (aghconfig.AgentDef, error)

// SessionManager is the runtime session surface exposed by API transports.
// List returns the current in-memory session snapshot without performing I/O.
// ListAll may perform I/O to return the authoritative session set, so it accepts a context.
type SessionManager interface {
	Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error)
	List() []*session.SessionInfo
	ListAll(ctx context.Context) ([]*session.SessionInfo, error)
	Status(ctx context.Context, id string) (*session.SessionInfo, error)
	Events(ctx context.Context, id string, query store.EventQuery) ([]store.SessionEvent, error)
	History(ctx context.Context, id string, query store.EventQuery) ([]store.TurnHistory, error)
	Transcript(ctx context.Context, id string) ([]transcript.Message, error)
	Stop(ctx context.Context, id string) error
	Resume(ctx context.Context, id string) (*session.Session, error)
	Prompt(ctx context.Context, id string, msg string) (<-chan acp.AgentEvent, error)
	ApprovePermission(ctx context.Context, id string, req acp.ApproveRequest) error
}

// Observer is the observability surface exposed by API transports.
type Observer interface {
	QueryEvents(ctx context.Context, query store.EventSummaryQuery) ([]store.EventSummary, error)
	Health(ctx context.Context) (observe.Health, error)
}

// DreamTrigger exposes consolidation controls and state to the API layer.
type DreamTrigger interface {
	Trigger(ctx context.Context, workspace string) (bool, string, error)
	LastConsolidatedAt() (time.Time, error)
	Enabled() bool
}

// SkillsRegistry exposes the skill catalog to the API layer.
type SkillsRegistry interface {
	Get(name string) (*skills.Skill, bool)
	List() []*skills.Skill
	ForWorkspace(ctx context.Context, resolved workspacepkg.ResolvedWorkspace) ([]*skills.Skill, error)
	SetEnabled(name string, resolved *workspacepkg.ResolvedWorkspace, enabled bool) error
}

// WorkspaceService exposes workspace registration and resolution to the API layer.
type WorkspaceService interface {
	Register(ctx context.Context, opts workspacepkg.RegisterOptions) (workspacepkg.Workspace, error)
	Unregister(ctx context.Context, id string) error
	Update(ctx context.Context, id string, opts workspacepkg.UpdateOptions) error
	List(ctx context.Context) ([]workspacepkg.Workspace, error)
	Get(ctx context.Context, idOrNameOrPath string) (workspacepkg.Workspace, error)
	Resolve(ctx context.Context, idOrNameOrPath string) (workspacepkg.ResolvedWorkspace, error)
	ResolveOrRegister(ctx context.Context, path string) (workspacepkg.ResolvedWorkspace, error)
}
