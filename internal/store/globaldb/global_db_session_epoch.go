package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
)

const sessionTranscriptEpochColumn = "transcript_epoch"

// SessionTranscriptEpoch returns the durable transcript reset epoch for one session.
func (g *GlobalDB) SessionTranscriptEpoch(ctx context.Context, sessionID string) (int64, error) {
	if err := g.checkReady(ctx, "read session transcript epoch"); err != nil {
		return 0, err
	}
	target := strings.TrimSpace(sessionID)
	if target == "" {
		return 0, errors.New("store: session id is required")
	}

	var epoch int64
	if err := g.db.QueryRowContext(
		ctx,
		`SELECT `+sessionTranscriptEpochColumn+` FROM sessions WHERE id = ?`,
		target,
	).Scan(&epoch); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("%w: %s", store.ErrSessionNotFound, target)
		}
		return 0, fmt.Errorf("store: read session transcript epoch %q: %w", target, err)
	}
	return epoch, nil
}

// BumpSessionTranscriptEpoch advances one session transcript epoch at least to update.Minimum.
func (g *GlobalDB) BumpSessionTranscriptEpoch(
	ctx context.Context,
	update store.SessionTranscriptEpochUpdate,
) (int64, error) {
	if err := g.checkReady(ctx, "bump session transcript epoch"); err != nil {
		return 0, err
	}
	if err := update.Validate(); err != nil {
		return 0, err
	}

	result, err := g.db.ExecContext(
		ctx,
		`UPDATE sessions
		 SET `+sessionTranscriptEpochColumn+` = CASE
			 WHEN `+sessionTranscriptEpochColumn+` < ? THEN ?
			 ELSE `+sessionTranscriptEpochColumn+` + 1
		 END,
		 updated_at = ?
		 WHERE id = ?`,
		update.Minimum,
		update.Minimum,
		store.FormatTimestamp(g.now()),
		update.SessionID,
	)
	if err != nil {
		return 0, fmt.Errorf("store: bump session transcript epoch %q: %w", update.SessionID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("store: rows affected for transcript epoch %q: %w", update.SessionID, err)
	}
	if affected == 0 {
		return 0, fmt.Errorf("%w: %s", store.ErrSessionNotFound, update.SessionID)
	}

	var epoch int64
	if err := g.db.QueryRowContext(
		ctx,
		`SELECT `+sessionTranscriptEpochColumn+` FROM sessions WHERE id = ?`,
		update.SessionID,
	).Scan(&epoch); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("%w: %s", store.ErrSessionNotFound, update.SessionID)
		}
		return 0, fmt.Errorf("store: read session transcript epoch %q: %w", update.SessionID, err)
	}
	return epoch, nil
}

// EnsureSessionTranscriptEpoch raises one session transcript epoch to update.Minimum without incrementing past it.
func (g *GlobalDB) EnsureSessionTranscriptEpoch(
	ctx context.Context,
	update store.SessionTranscriptEpochUpdate,
) (int64, error) {
	if err := g.checkReady(ctx, "ensure session transcript epoch"); err != nil {
		return 0, err
	}
	if err := update.Validate(); err != nil {
		return 0, err
	}

	if _, err := g.db.ExecContext(
		ctx,
		`UPDATE sessions
		 SET `+sessionTranscriptEpochColumn+` = ?, updated_at = ?
		 WHERE id = ? AND `+sessionTranscriptEpochColumn+` < ?`,
		update.Minimum,
		store.FormatTimestamp(g.now()),
		update.SessionID,
		update.Minimum,
	); err != nil {
		return 0, fmt.Errorf("store: ensure session transcript epoch %q: %w", update.SessionID, err)
	}

	var epoch int64
	if err := g.db.QueryRowContext(
		ctx,
		`SELECT `+sessionTranscriptEpochColumn+` FROM sessions WHERE id = ?`,
		update.SessionID,
	).Scan(&epoch); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("%w: %s", store.ErrSessionNotFound, update.SessionID)
		}
		return 0, fmt.Errorf("store: read ensured session transcript epoch %q: %w", update.SessionID, err)
	}
	return epoch, nil
}
