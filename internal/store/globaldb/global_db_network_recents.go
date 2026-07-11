package globaldb

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
)

// ListNetworkRecents returns the newest thread/direct summaries across every
// channel in one workspace using the materialized conversation tables.
func (g *GlobalDB) ListNetworkRecents(
	ctx context.Context,
	query store.NetworkRecentQuery,
) (recents []store.NetworkRecentSummary, err error) {
	if err := g.checkReady(ctx, "list network recents"); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, fmt.Errorf("store: validate network recent query: %w", err)
	}
	workspaceID := strings.TrimSpace(query.WorkspaceID)
	statement := `SELECT
		workspace_id, channel, surface, container_id, last_activity_at, last_activity_sequence,
		last_message_preview, title, participant_count, peer_a, peer_b
	FROM (
		SELECT
			workspace_id, channel, 'thread' AS surface, thread_id AS container_id,
			last_activity_at, last_activity_sequence, last_message_preview, title, participant_count,
			'' AS peer_a, '' AS peer_b
		FROM network_threads
		WHERE workspace_id = ?
		UNION ALL
		SELECT
			workspace_id, channel, 'direct' AS surface, direct_id AS container_id,
			last_activity_at, last_activity_sequence, last_message_preview, '' AS title, 0 AS participant_count,
			peer_a, peer_b
		FROM network_direct_rooms
		WHERE workspace_id = ?
	)
	ORDER BY last_activity_sequence DESC, surface ASC, container_id ASC`
	args := []any{workspaceID, workspaceID}
	statement, args = store.AppendLimit(statement, args, query.Limit)
	rows, err := g.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query network recents: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			closeErr = fmt.Errorf("store: close network recent rows: %w", closeErr)
			err = errors.Join(err, closeErr)
		}
	}()

	recents = make([]store.NetworkRecentSummary, 0)
	for rows.Next() {
		var (
			recent      store.NetworkRecentSummary
			activityRaw string
		)
		if scanErr := rows.Scan(
			&recent.WorkspaceID,
			&recent.Channel,
			&recent.Surface,
			&recent.ContainerID,
			&activityRaw,
			&recent.LastActivitySequence,
			&recent.LastMessagePreview,
			&recent.Title,
			&recent.ParticipantCount,
			&recent.PeerA,
			&recent.PeerB,
		); scanErr != nil {
			return nil, fmt.Errorf("store: scan network recent: %w", scanErr)
		}
		activity, parseErr := store.ParseTimestamp(activityRaw)
		if parseErr != nil {
			return nil, fmt.Errorf("store: parse network recent activity: %w", parseErr)
		}
		recent.LastActivityAt = activity
		recents = append(recents, recent)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate network recents: %w", err)
	}
	return recents, nil
}
