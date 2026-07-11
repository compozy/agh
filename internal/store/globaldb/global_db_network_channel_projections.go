package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
)

// ListNetworkChannelProjections returns one materialized row per persisted
// channel timeline without scanning the underlying message log.
func (g *GlobalDB) ListNetworkChannelProjections(
	ctx context.Context,
	query store.NetworkChannelProjectionQuery,
) (projections []store.NetworkChannelProjection, err error) {
	if err := g.checkReady(ctx, "list network channel projections"); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, fmt.Errorf("store: validate network channel projection query: %w", err)
	}

	statement := `SELECT
		workspace_id, channel, message_count, presence_count, historical_participant_count,
		last_activity_at, last_activity_sequence, last_presence_at, last_presence_sequence,
		last_message_sequence, last_message_preview
	FROM network_channel_stats
	WHERE workspace_id = ?`
	args := []any{strings.TrimSpace(query.WorkspaceID)}
	if channel := strings.TrimSpace(query.Channel); channel != "" {
		statement += " AND channel = ?"
		args = append(args, channel)
	}
	statement += " ORDER BY channel ASC"
	statement, args = store.AppendLimit(statement, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query network channel projections: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			closeErr = fmt.Errorf("store: close network channel projection rows: %w", closeErr)
			err = errors.Join(err, closeErr)
		}
	}()

	projections = make([]store.NetworkChannelProjection, 0)
	for rows.Next() {
		projection, scanErr := scanNetworkChannelProjection(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		projections = append(projections, projection)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate network channel projections: %w", err)
	}
	return projections, nil
}

// ListNetworkChannelKindCounts returns materialized public-message totals for
// one workspace-qualified channel.
func (g *GlobalDB) ListNetworkChannelKindCounts(
	ctx context.Context,
	ref store.NetworkChannelRef,
) (counts []store.NetworkChannelKindCount, err error) {
	if err := g.checkReady(ctx, "list network channel kind counts"); err != nil {
		return nil, err
	}
	if err := ref.Validate(); err != nil {
		return nil, fmt.Errorf("store: validate network channel kind count ref: %w", err)
	}
	rows, err := g.db.QueryContext(
		ctx,
		`SELECT workspace_id, channel, kind, message_count
		 FROM network_channel_kind_counts
		 WHERE workspace_id = ? AND channel = ?
		 ORDER BY kind ASC`,
		strings.TrimSpace(ref.WorkspaceID),
		strings.TrimSpace(ref.Channel),
	)
	if err != nil {
		return nil, fmt.Errorf("store: query network channel kind counts: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			closeErr = fmt.Errorf("store: close network channel kind count rows: %w", closeErr)
			err = errors.Join(err, closeErr)
		}
	}()

	counts = make([]store.NetworkChannelKindCount, 0)
	for rows.Next() {
		var count store.NetworkChannelKindCount
		if scanErr := rows.Scan(&count.WorkspaceID, &count.Channel, &count.Kind, &count.Count); scanErr != nil {
			return nil, fmt.Errorf("store: scan network channel kind count: %w", scanErr)
		}
		counts = append(counts, count)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate network channel kind counts: %w", err)
	}
	return counts, nil
}

func scanNetworkChannelProjection(scanner rowScanner) (store.NetworkChannelProjection, error) {
	var (
		projection      store.NetworkChannelProjection
		lastActivityRaw sql.NullString
		lastPresenceRaw sql.NullString
	)
	if err := scanner.Scan(
		&projection.WorkspaceID,
		&projection.Channel,
		&projection.MessageCount,
		&projection.PresenceCount,
		&projection.HistoricalParticipantCount,
		&lastActivityRaw,
		&projection.LastActivitySequence,
		&lastPresenceRaw,
		&projection.LastPresenceSequence,
		&projection.LastMessageSequence,
		&projection.LastMessagePreview,
	); err != nil {
		return store.NetworkChannelProjection{}, fmt.Errorf("store: scan network channel projection: %w", err)
	}
	if lastActivityRaw.Valid {
		value, err := store.ParseTimestamp(lastActivityRaw.String)
		if err != nil {
			return store.NetworkChannelProjection{}, fmt.Errorf("store: parse channel last activity: %w", err)
		}
		projection.LastActivityAt = &value
	}
	if lastPresenceRaw.Valid {
		value, err := store.ParseTimestamp(lastPresenceRaw.String)
		if err != nil {
			return store.NetworkChannelProjection{}, fmt.Errorf("store: parse channel last presence: %w", err)
		}
		projection.LastPresenceAt = &value
	}
	return projection, nil
}
