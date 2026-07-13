package globaldb

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
)

// CountSessionsByAgent returns exact durable visible-session counts grouped by
// agent for one workspace.
func (g *SessionRepo) CountSessionsByAgent(
	ctx context.Context,
	query store.SessionAgentCountQuery,
) (counts []store.SessionAgentCount, err error) {
	if err := g.checkReady(ctx, "count sessions by agent"); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}
	if _, err := g.SweepExpiredSessionAttachLocks(ctx, g.now()); err != nil {
		return nil, fmt.Errorf("store: sweep expired session attach locks before counting agents: %w", err)
	}

	where, args, err := sessionCatalogPageFilters(store.SessionCatalogPageQuery{
		WorkspaceID:         strings.TrimSpace(query.WorkspaceID),
		ExcludeIDs:          query.ExcludeIDs,
		ExcludeSessionTypes: query.ExcludeSessionTypes,
		ExcludeSpawnRoles:   query.ExcludeSpawnRoles,
	}, store.FormatTimestamp(g.now()))
	if err != nil {
		return nil, err
	}
	where = append(where, "trim(agent_name) != ''")
	sqlQuery := store.AppendWhere(`SELECT agent_name, COUNT(1),
		SUM(CASE WHEN state = 'active' THEN 1 ELSE 0 END)
		FROM sessions`, where)
	sqlQuery += " GROUP BY agent_name ORDER BY lower(agent_name), agent_name"

	// dynamic-sql: exclusion slices and catalog visibility filters alter the aggregate query structure.
	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query session agent counts: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("store: close session agent count rows: %w", closeErr))
		}
	}()

	counts = make([]store.SessionAgentCount, 0)
	for rows.Next() {
		var count store.SessionAgentCount
		if scanErr := rows.Scan(&count.AgentName, &count.Total, &count.Active); scanErr != nil {
			return nil, fmt.Errorf("store: scan session agent count: %w", scanErr)
		}
		count.AgentName = strings.TrimSpace(count.AgentName)
		counts = append(counts, count)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate session agent counts: %w", err)
	}
	return counts, nil
}

var _ store.SessionAgentCounter = (*SessionRepo)(nil)
