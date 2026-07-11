package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func (g *GlobalDB) loadSessionIDs(
	ctx context.Context,
	tx *sql.Tx,
) (ids map[string]struct{}, err error) {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM sessions`)
	if err != nil {
		return nil, fmt.Errorf("store: query existing session ids: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("store: close existing session id rows: %w", closeErr))
		}
	}()

	ids = make(map[string]struct{})
	for rows.Next() {
		var id string
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, fmt.Errorf("store: scan existing session id: %w", scanErr)
		}
		ids[id] = struct{}{}
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("store: iterate existing session ids: %w", rowsErr)
	}
	return ids, nil
}
