package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/compozy/agh/internal/store"
)

// GetSessionInputQueueEntryByID returns one exact internal queue entry by its globally unique ID.
func (g *GlobalDB) GetSessionInputQueueEntryByID(
	ctx context.Context,
	entryID string,
) (store.SessionInputQueueEntry, error) {
	if err := g.checkReady(ctx, "get session input queue entry by id"); err != nil {
		return store.SessionInputQueueEntry{}, err
	}
	target := strings.TrimSpace(entryID)
	if target == "" {
		return store.SessionInputQueueEntry{}, errors.New("store: session input queue entry id is required")
	}
	entry, err := scanSessionInputQueueEntry(g.db.QueryRowContext(
		ctx,
		`SELECT `+sessionInputQueueColumns+` FROM session_input_queue WHERE id = ?`,
		target,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return store.SessionInputQueueEntry{}, store.ErrSessionInputQueueEntryNotFound
	}
	if err != nil {
		return store.SessionInputQueueEntry{}, err
	}
	return entry, nil
}
