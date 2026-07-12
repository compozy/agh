package store

import (
	"context"
	"fmt"
	"strings"
)

// SessionAgentCountQuery describes one workspace-scoped grouped catalog read.
// Active runtime session IDs are excluded before the session manager overlays
// their current snapshots.
type SessionAgentCountQuery struct {
	WorkspaceID         string
	ExcludeIDs          []string
	ExcludeSessionTypes []string
	ExcludeSpawnRoles   []string
}

// Validate ensures grouped counts cannot cross workspace boundaries.
func (q SessionAgentCountQuery) Validate() error {
	if strings.TrimSpace(q.WorkspaceID) == "" {
		return fmt.Errorf("store: session agent counts workspace id is required")
	}
	return nil
}

// SessionAgentCount contains exact visible-session totals for one agent.
type SessionAgentCount struct {
	AgentName string
	Total     int
	Active    int
}

// SessionAgentCounter exposes one grouped durable-catalog read for agent fleet
// projections.
type SessionAgentCounter interface {
	CountSessionsByAgent(
		ctx context.Context,
		query SessionAgentCountQuery,
	) ([]SessionAgentCount, error)
}
