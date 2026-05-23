package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
)

// WriteNetworkChannel upserts durable network channel metadata.
func (g *GlobalDB) WriteNetworkChannel(ctx context.Context, entry store.NetworkChannelEntry) error {
	entry.Channel = strings.TrimSpace(entry.Channel)
	entry.CreatedBy = strings.TrimSpace(entry.CreatedBy)
	if err := g.checkReady(ctx, "write network channel"); err != nil {
		return err
	}
	if err := entry.Validate(); err != nil {
		return fmt.Errorf("store: validate network channel entry: %w", err)
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = g.now()
	}
	if entry.UpdatedAt.IsZero() {
		entry.UpdatedAt = entry.CreatedAt
	}

	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO network_channels (
			channel,
			workspace_id,
			purpose,
			created_by,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(workspace_id, channel) DO UPDATE SET
				purpose = excluded.purpose,
				updated_at = excluded.updated_at,
				created_by = CASE
				WHEN TRIM(network_channels.created_by) = '' THEN excluded.created_by
				ELSE network_channels.created_by
			END`,
		entry.Channel,
		entry.WorkspaceID,
		entry.Purpose,
		entry.CreatedBy,
		store.FormatTimestamp(entry.CreatedAt),
		store.FormatTimestamp(entry.UpdatedAt),
	); err != nil {
		return fmt.Errorf("store: insert network channel entry: %w", err)
	}

	return nil
}

// GetNetworkChannel returns one persisted network channel metadata row.
func (g *GlobalDB) GetNetworkChannel(
	ctx context.Context,
	ref store.NetworkChannelRef,
) (store.NetworkChannelEntry, error) {
	if err := g.checkReady(ctx, "get network channel"); err != nil {
		return store.NetworkChannelEntry{}, err
	}
	normalized := store.NetworkChannelRef{
		WorkspaceID: strings.TrimSpace(ref.WorkspaceID),
		Channel:     strings.TrimSpace(ref.Channel),
	}
	if err := normalized.Validate(); err != nil {
		return store.NetworkChannelEntry{}, err
	}

	row := g.db.QueryRowContext(
		ctx,
		`SELECT channel, workspace_id, purpose, created_by, created_at, updated_at
		FROM network_channels
		WHERE workspace_id = ? AND channel = ?`,
		normalized.WorkspaceID,
		normalized.Channel,
	)

	entry, err := scanNetworkChannel(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.NetworkChannelEntry{}, err
		}
		return store.NetworkChannelEntry{}, err
	}
	return entry, nil
}

// ListNetworkChannels returns persisted network channel metadata rows.
func (g *GlobalDB) ListNetworkChannels(
	ctx context.Context,
	query store.NetworkChannelQuery,
) (entries []store.NetworkChannelEntry, err error) {
	if err := g.checkReady(ctx, "list network channels"); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, fmt.Errorf("store: validate network channel query: %w", err)
	}

	sqlQuery := `SELECT channel, workspace_id, purpose, created_by, created_at, updated_at FROM network_channels`
	where, args := store.BuildClauses(
		store.StringClause("channel", query.Channel),
		store.StringClause("workspace_id", query.WorkspaceID),
	)
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY updated_at DESC, channel ASC"
	sqlQuery, args = store.AppendLimit(sqlQuery, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query network channels: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			closeErr = fmt.Errorf("store: close network channels rows: %w", closeErr)
			if err != nil {
				err = errors.Join(err, closeErr)
				return
			}
			err = closeErr
		}
	}()

	entries = make([]store.NetworkChannelEntry, 0)
	for rows.Next() {
		entry, scanErr := scanNetworkChannel(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate network channels: %w", err)
	}

	return entries, nil
}

// DeleteNetworkChannel removes one persisted channel metadata row.
func (g *GlobalDB) DeleteNetworkChannel(ctx context.Context, ref store.NetworkChannelRef) error {
	if err := g.checkReady(ctx, "delete network channel"); err != nil {
		return err
	}
	normalized := store.NetworkChannelRef{
		WorkspaceID: strings.TrimSpace(ref.WorkspaceID),
		Channel:     strings.TrimSpace(ref.Channel),
	}
	if err := normalized.Validate(); err != nil {
		return err
	}

	if _, err := g.db.ExecContext(
		ctx,
		`DELETE FROM network_channels WHERE workspace_id = ? AND channel = ?`,
		normalized.WorkspaceID,
		normalized.Channel,
	); err != nil {
		return fmt.Errorf("store: delete network channel: %w", err)
	}
	return nil
}

func scanNetworkChannel(scanner rowScanner) (store.NetworkChannelEntry, error) {
	var (
		entry        store.NetworkChannelEntry
		createdBy    sql.NullString
		createdAtRaw string
		updatedAtRaw string
	)
	if err := scanner.Scan(
		&entry.Channel,
		&entry.WorkspaceID,
		&entry.Purpose,
		&createdBy,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.NetworkChannelEntry{}, err
		}
		return store.NetworkChannelEntry{}, fmt.Errorf("store: scan network channel: %w", err)
	}

	if value := store.NullString(createdBy); value != nil {
		entry.CreatedBy = *value
	}

	createdAt, err := store.ParseTimestamp(createdAtRaw)
	if err != nil {
		return store.NetworkChannelEntry{}, fmt.Errorf("store: parse network channel created_at: %w", err)
	}
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return store.NetworkChannelEntry{}, fmt.Errorf("store: parse network channel updated_at: %w", err)
	}
	entry.CreatedAt = createdAt
	entry.UpdatedAt = updatedAt
	return entry, nil
}
