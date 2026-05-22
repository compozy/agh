package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	presetspkg "github.com/compozy/agh/internal/notifications/presets"
	"github.com/compozy/agh/internal/store"
)

var _ presetspkg.Store = (*GlobalDB)(nil)

type notificationPresetScanner interface {
	Scan(dest ...any) error
}

func (g *GlobalDB) ListPresets(
	ctx context.Context,
	query presetspkg.Query,
) (items []presetspkg.Preset, err error) {
	if err := g.checkReady(ctx, "list notification presets"); err != nil {
		return nil, err
	}
	normalized := query.Normalize()
	args := make([]any, 0, 3)
	clauses := make([]string, 0, 3)
	if normalized.Enabled != nil {
		clauses = append(clauses, "enabled = ?")
		args = append(args, *normalized.Enabled)
	}
	if normalized.BuiltIn != nil {
		clauses = append(clauses, "built_in = ?")
		args = append(args, *normalized.BuiltIn)
	}
	if normalized.Name != "" {
		clauses = append(clauses, "name = ?")
		args = append(args, normalized.Name)
	}
	where := ""
	if len(clauses) > 0 {
		where = " WHERE " + strings.Join(clauses, " AND ")
	}
	limit := ""
	if normalized.Limit > 0 {
		limit = " LIMIT ?"
		args = append(args, normalized.Limit)
	}
	rows, err := g.db.QueryContext(
		ctx,
		`SELECT name, events, targets, filter, enabled, built_in, default_version,
		       default_hash, user_modified, default_update_available, created_at, updated_at
		  FROM notification_presets`+where+`
		 ORDER BY built_in DESC, name ASC`+limit,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("store: query notification presets: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("store: close notification preset rows: %w", closeErr)
		}
	}()
	items = make([]presetspkg.Preset, 0)
	for rows.Next() {
		preset, scanErr := scanNotificationPreset(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, preset)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate notification presets: %w", err)
	}
	return items, nil
}

func (g *GlobalDB) GetPreset(ctx context.Context, name string) (presetspkg.Preset, error) {
	if err := g.checkReady(ctx, "get notification preset"); err != nil {
		return presetspkg.Preset{}, err
	}
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return presetspkg.Preset{}, fmt.Errorf("%w: name is required", presetspkg.ErrInvalidPreset)
	}
	return g.getNotificationPreset(ctx, g.db, trimmed)
}

func (g *GlobalDB) CreatePreset(
	ctx context.Context,
	preset presetspkg.Preset,
) (presetspkg.Preset, error) {
	if err := g.checkReady(ctx, "create notification preset"); err != nil {
		return presetspkg.Preset{}, err
	}
	normalized := preset.Normalize()
	if err := normalized.Validate(); err != nil {
		return presetspkg.Preset{}, err
	}
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = g.now()
	}
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = normalized.CreatedAt
	}
	eventsJSON, targetsJSON, err := encodeNotificationPresetLists(normalized)
	if err != nil {
		return presetspkg.Preset{}, err
	}
	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO notification_presets (
			name, events, targets, filter, enabled, built_in, default_version, default_hash,
			user_modified, default_update_available, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		normalized.Name,
		eventsJSON,
		targetsJSON,
		normalized.Filter,
		normalized.Enabled,
		normalized.BuiltIn,
		normalized.DefaultVersion,
		normalized.DefaultHash,
		normalized.UserModified,
		normalized.DefaultUpdateAvailable,
		store.FormatTimestamp(normalized.CreatedAt),
		store.FormatTimestamp(normalized.UpdatedAt),
	); err != nil {
		if isNotificationPresetDuplicateNameError(err) {
			return presetspkg.Preset{}, presetspkg.ErrPresetDuplicateName
		}
		return presetspkg.Preset{}, fmt.Errorf(
			"store: insert notification preset %q: %w",
			normalized.Name,
			err,
		)
	}
	return normalized, nil
}

func (g *GlobalDB) UpdatePreset(
	ctx context.Context,
	name string,
	req presetspkg.UpdateRequest,
) (presetspkg.Preset, error) {
	if err := g.checkReady(ctx, "update notification preset"); err != nil {
		return presetspkg.Preset{}, err
	}
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return presetspkg.Preset{}, fmt.Errorf("%w: name is required", presetspkg.ErrInvalidPreset)
	}
	if !req.HasMutableField() {
		return presetspkg.Preset{}, fmt.Errorf(
			"%w: update requires at least one mutable field",
			presetspkg.ErrInvalidPreset,
		)
	}
	current, err := g.getNotificationPreset(ctx, g.db, trimmed)
	if err != nil {
		return presetspkg.Preset{}, err
	}
	updated := current.Normalize()
	if req.Events != nil {
		updated.Events = append([]string(nil), (*req.Events)...)
	}
	if req.Targets != nil {
		updated.Targets = append([]presetspkg.Target(nil), (*req.Targets)...)
	}
	if req.Filter != nil {
		updated.Filter = strings.TrimSpace(*req.Filter)
	}
	if req.Enabled != nil {
		updated.Enabled = *req.Enabled
	}
	updated.UpdatedAt = req.Now
	if updated.UpdatedAt.IsZero() {
		updated.UpdatedAt = g.now()
	}
	updated = presetspkg.ApplyDefaultDrift(updated)
	if err := updated.Validate(); err != nil {
		return presetspkg.Preset{}, err
	}
	eventsJSON, targetsJSON, err := encodeNotificationPresetLists(updated)
	if err != nil {
		return presetspkg.Preset{}, err
	}
	result, err := g.db.ExecContext(
		ctx,
		`UPDATE notification_presets
		    SET events = ?, targets = ?, filter = ?, enabled = ?, user_modified = ?,
		        default_update_available = ?, updated_at = ?
		  WHERE name = ?`,
		eventsJSON,
		targetsJSON,
		updated.Filter,
		updated.Enabled,
		updated.UserModified,
		updated.DefaultUpdateAvailable,
		store.FormatTimestamp(updated.UpdatedAt),
		updated.Name,
	)
	if err != nil {
		return presetspkg.Preset{}, fmt.Errorf(
			"store: update notification preset %q: %w",
			updated.Name,
			err,
		)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return presetspkg.Preset{}, fmt.Errorf(
			"store: rows affected for notification preset %q: %w",
			updated.Name,
			err,
		)
	}
	if affected == 0 {
		return presetspkg.Preset{}, presetspkg.ErrPresetNotFound
	}
	return updated, nil
}

func (g *GlobalDB) DeletePreset(ctx context.Context, name string) error {
	if err := g.checkReady(ctx, "delete notification preset"); err != nil {
		return err
	}
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("%w: name is required", presetspkg.ErrInvalidPreset)
	}
	current, err := g.getNotificationPreset(ctx, g.db, trimmed)
	if err != nil {
		return err
	}
	if current.BuiltIn {
		return presetspkg.ErrPresetBuiltIn
	}
	result, err := g.db.ExecContext(ctx, `DELETE FROM notification_presets WHERE name = ?`, trimmed)
	if err != nil {
		return fmt.Errorf("store: delete notification preset %q: %w", trimmed, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: rows affected for notification preset %q: %w", trimmed, err)
	}
	if affected == 0 {
		return presetspkg.ErrPresetNotFound
	}
	return nil
}

// EnsureBuiltInPresets inserts disabled built-ins and records default drift without overwriting user edits.
func (g *GlobalDB) EnsureBuiltInPresets(
	ctx context.Context,
	defaults []presetspkg.Preset,
) (err error) {
	if err := g.checkReady(ctx, "ensure notification preset defaults"); err != nil {
		return err
	}
	conn, err := g.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("store: open notification preset default transaction: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf(
				"store: close notification preset default transaction connection: %w",
				closeErr,
			)
		}
	}()
	rollbackCtx, rollbackCancel := notificationPresetRollbackContext(ctx)
	defer rollbackCancel()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("store: begin notification preset default seed: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			joinCleanupError(
				&err,
				rollbackImmediate(rollbackCtx, conn, "notification preset default seed"),
			)
		}
	}()
	for _, defaultPreset := range defaults {
		if seedErr := seedNotificationPresetDefault(ctx, conn, defaultPreset.Normalize(), g.now()); seedErr != nil {
			return seedErr
		}
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("store: commit notification preset default seed: %w", err)
	}
	committed = true
	return nil
}

func (g *GlobalDB) getNotificationPreset(
	ctx context.Context,
	queryer interface {
		QueryRowContext(context.Context, string, ...any) *sql.Row
	},
	name string,
) (presetspkg.Preset, error) {
	row := queryer.QueryRowContext(
		ctx,
		`SELECT name, events, targets, filter, enabled, built_in, default_version,
		       default_hash, user_modified, default_update_available, created_at, updated_at
		  FROM notification_presets
		 WHERE name = ?`,
		name,
	)
	preset, err := scanNotificationPreset(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return presetspkg.Preset{}, presetspkg.ErrPresetNotFound
		}
		return presetspkg.Preset{}, err
	}
	return preset, nil
}

func encodeNotificationPresetLists(preset presetspkg.Preset) (string, string, error) {
	eventsJSON, err := json.Marshal(preset.Events)
	if err != nil {
		return "", "", fmt.Errorf("store: encode notification preset events: %w", err)
	}
	targetsJSON, err := json.Marshal(preset.Targets)
	if err != nil {
		return "", "", fmt.Errorf("store: encode notification preset targets: %w", err)
	}
	return string(eventsJSON), string(targetsJSON), nil
}

func seedNotificationPresetDefault(
	ctx context.Context,
	execer interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
	},
	defaultPreset presetspkg.Preset,
	now time.Time,
) error {
	normalized := defaultPreset.Normalize()
	if !normalized.BuiltIn || normalized.DefaultHash == "" {
		return fmt.Errorf("%w: built-in preset default is incomplete", presetspkg.ErrInvalidPreset)
	}
	eventsJSON, targetsJSON, err := encodeNotificationPresetLists(normalized)
	if err != nil {
		return err
	}
	timestamp := store.FormatTimestamp(now.UTC())
	_, err = execer.ExecContext(
		ctx,
		`INSERT INTO notification_presets (
			name, events, targets, filter, enabled, built_in, default_version, default_hash,
			user_modified, default_update_available, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, 1, ?, ?, 0, 0, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			events = CASE
				WHEN notification_presets.built_in = 1 AND notification_presets.user_modified = 0
				THEN excluded.events ELSE notification_presets.events END,
			targets = CASE
				WHEN notification_presets.built_in = 1 AND notification_presets.user_modified = 0
				THEN excluded.targets ELSE notification_presets.targets END,
			filter = CASE
				WHEN notification_presets.built_in = 1 AND notification_presets.user_modified = 0
				THEN excluded.filter ELSE notification_presets.filter END,
			enabled = CASE
				WHEN notification_presets.built_in = 1 AND notification_presets.user_modified = 0
				THEN excluded.enabled ELSE notification_presets.enabled END,
			built_in = CASE
				WHEN notification_presets.built_in = 1 THEN 1 ELSE notification_presets.built_in END,
			default_version = CASE
				WHEN notification_presets.built_in = 1 THEN excluded.default_version
				ELSE notification_presets.default_version END,
			default_hash = CASE
				WHEN notification_presets.built_in = 1 THEN excluded.default_hash
				ELSE notification_presets.default_hash END,
			default_update_available = CASE
				WHEN notification_presets.built_in = 1
				 AND notification_presets.user_modified = 1
				 AND notification_presets.default_hash <> excluded.default_hash
				THEN 1
				WHEN notification_presets.built_in = 1 AND notification_presets.user_modified = 0
				THEN 0
				ELSE notification_presets.default_update_available END,
			updated_at = CASE
				WHEN notification_presets.built_in = 1 THEN excluded.updated_at
				ELSE notification_presets.updated_at END`,
		normalized.Name,
		eventsJSON,
		targetsJSON,
		normalized.Filter,
		normalized.Enabled,
		normalized.DefaultVersion,
		normalized.DefaultHash,
		timestamp,
		timestamp,
	)
	if err != nil {
		return fmt.Errorf("store: seed notification preset default %q: %w", normalized.Name, err)
	}
	return nil
}

func scanNotificationPreset(scanner notificationPresetScanner) (presetspkg.Preset, error) {
	var (
		preset       presetspkg.Preset
		eventsRaw    string
		targetsRaw   string
		createdAtRaw string
		updatedAtRaw string
	)
	if err := scanner.Scan(
		&preset.Name,
		&eventsRaw,
		&targetsRaw,
		&preset.Filter,
		&preset.Enabled,
		&preset.BuiltIn,
		&preset.DefaultVersion,
		&preset.DefaultHash,
		&preset.UserModified,
		&preset.DefaultUpdateAvailable,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return presetspkg.Preset{}, fmt.Errorf("store: scan notification preset: %w", err)
	}
	if err := json.Unmarshal([]byte(eventsRaw), &preset.Events); err != nil {
		return presetspkg.Preset{}, fmt.Errorf("store: decode notification preset events: %w", err)
	}
	if err := json.Unmarshal([]byte(targetsRaw), &preset.Targets); err != nil {
		return presetspkg.Preset{}, fmt.Errorf("store: decode notification preset targets: %w", err)
	}
	createdAt, err := store.ParseTimestamp(createdAtRaw)
	if err != nil {
		return presetspkg.Preset{}, err
	}
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return presetspkg.Preset{}, err
	}
	preset.CreatedAt = createdAt
	preset.UpdatedAt = updatedAt
	preset = presetspkg.ApplyDefaultDrift(preset)
	if err := preset.Validate(); err != nil {
		return presetspkg.Preset{}, err
	}
	return preset, nil
}

func notificationPresetRollbackContext(
	parent context.Context,
) (context.Context, context.CancelFunc) {
	if parent == nil {
		return context.WithTimeout(context.Background(), notificationCursorRollbackTimeout)
	}
	return context.WithTimeout(context.WithoutCancel(parent), notificationCursorRollbackTimeout)
}

func isNotificationPresetDuplicateNameError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique constraint failed") &&
		strings.Contains(message, "notification_presets.name")
}
