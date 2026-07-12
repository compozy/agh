package session

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
)

// AgentSessionCount is the exact visible-session aggregate for one agent.
type AgentSessionCount struct {
	Total  int
	Active int
}

// CountSessionsByAgent returns workspace-scoped durable counts overlaid with
// the manager's current live-session snapshots.
func (m *Manager) CountSessionsByAgent(
	ctx context.Context,
	workspaceID string,
) (map[string]AgentSessionCount, error) {
	if ctx == nil {
		return nil, errors.New("session: count sessions by agent context is required")
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, errors.New("session: count sessions by agent workspace id is required")
	}
	counter, ok := m.sessionCatalog.(store.SessionAgentCounter)
	if !ok || counter == nil {
		return nil, errors.New("session: grouped agent session counter is required")
	}

	_, activeIDs, activeMatches := m.activeSessionCatalogRows(ListQuery{
		WorkspaceID: workspaceID,
	})
	durable, err := counter.CountSessionsByAgent(ctx, store.SessionAgentCountQuery{
		WorkspaceID:         workspaceID,
		ExcludeIDs:          activeIDs,
		ExcludeSessionTypes: []string{string(SessionTypeDream)},
		ExcludeSpawnRoles:   []string{SpawnRoleMemoryExtractor},
	})
	if err != nil {
		return nil, fmt.Errorf("session: count durable sessions by agent: %w", err)
	}

	counts := make(map[string]AgentSessionCount, len(durable)+len(activeMatches))
	for _, durableCount := range durable {
		name := strings.TrimSpace(durableCount.AgentName)
		if name == "" {
			continue
		}
		counts[name] = AgentSessionCount{
			Total:  durableCount.Total,
			Active: durableCount.Active,
		}
	}
	for _, active := range activeMatches {
		name := strings.TrimSpace(active.AgentName)
		if name == "" {
			continue
		}
		count := counts[name]
		count.Total++
		if strings.TrimSpace(active.State) == string(StateActive) {
			count.Active++
		}
		counts[name] = count
	}
	return counts, nil
}
