package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

// GlobalDB owns the global session index and observability database.
type GlobalDB struct {
	db   *sql.DB
	path string
	now  func() time.Time
}

var _ SessionRegistry = (*GlobalDB)(nil)
var _ aghworkspace.WorkspaceStore = (*GlobalDB)(nil)

// OpenGlobalDB opens or creates the global AGH index database.
func OpenGlobalDB(ctx context.Context, path string) (*GlobalDB, error) {
	if ctx == nil {
		return nil, errors.New("store: open global database context is required")
	}

	db, err := openGlobalSQLite(ctx, path)
	if err != nil {
		return nil, err
	}

	return &GlobalDB{
		db:   db,
		path: strings.TrimSpace(path),
		now: func() time.Time {
			return time.Now().UTC()
		},
	}, nil
}

// Path reports the on-disk path for the global database file.
func (g *GlobalDB) Path() string {
	if g == nil {
		return ""
	}
	return g.path
}

// Close checkpoints the WAL and closes the database.
func (g *GlobalDB) Close(ctx context.Context) error {
	if g == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("store: close global database context is required")
	}

	checkpointErr := checkpoint(ctx, g.db)
	closeErr := g.db.Close()
	return errors.Join(checkpointErr, closeErr)
}
