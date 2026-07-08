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

	var epoch int64
	if err := g.db.QueryRowContext(
		ctx,
		`UPDATE sessions
		 SET `+sessionTranscriptEpochColumn+` = ?, updated_at = ?
		 WHERE id = ? AND `+sessionTranscriptEpochColumn+` < ?
		 RETURNING `+sessionTranscriptEpochColumn,
		update.Minimum,
		store.FormatTimestamp(g.now()),
		update.SessionID,
		update.Minimum,
	).Scan(&epoch); err == nil {
		return epoch, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("store: ensure session transcript epoch %q: %w", update.SessionID, err)
	}

	return g.SessionTranscriptEpoch(ctx, update.SessionID)
}
