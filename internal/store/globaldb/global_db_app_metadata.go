package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
)

// GetAppMetadata returns the value stored under key and whether the key exists.
func (g *GlobalDB) GetAppMetadata(ctx context.Context, key string) (string, bool, error) {
	if err := g.checkReady(ctx, "get app metadata"); err != nil {
		return "", false, err
	}
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return "", false, errors.New("store: app metadata key is required")
	}

	var value string
	err := g.db.QueryRowContext(
		ctx,
		`SELECT value FROM app_metadata WHERE key = ?`,
		trimmed,
	).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("store: query app metadata %q: %w", trimmed, err)
	}
	return value, true, nil
}

// SetAppMetadata upserts the value stored under key.
func (g *GlobalDB) SetAppMetadata(ctx context.Context, key string, value string) error {
	if err := g.checkReady(ctx, "set app metadata"); err != nil {
		return err
	}
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return errors.New("store: app metadata key is required")
	}

	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO app_metadata (key, value, updated_at)
			VALUES (?, ?, ?)
			ON CONFLICT(key) DO UPDATE SET
				value = excluded.value,
				updated_at = excluded.updated_at`,
		trimmed,
		value,
		store.FormatTimestamp(g.now()),
	); err != nil {
		return fmt.Errorf("store: upsert app metadata %q: %w", trimmed, err)
	}
	return nil
}

// DeleteAppMetadata removes the value stored under key. Missing keys are a no-op.
func (g *GlobalDB) DeleteAppMetadata(ctx context.Context, key string) error {
	if err := g.checkReady(ctx, "delete app metadata"); err != nil {
		return err
	}
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return errors.New("store: app metadata key is required")
	}

	if _, err := g.db.ExecContext(ctx, `DELETE FROM app_metadata WHERE key = ?`, trimmed); err != nil {
		return fmt.Errorf("store: delete app metadata %q: %w", trimmed, err)
	}
	return nil
}
