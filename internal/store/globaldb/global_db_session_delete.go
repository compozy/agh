package globaldb

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/store/globaldb/sqlcgen"
)

// DeleteSession removes one session and its non-cascading dependent rows.
func (g *SessionRepo) DeleteSession(ctx context.Context, id string) error {
	if err := g.checkReady(ctx, "delete session"); err != nil {
		return err
	}
	target := strings.TrimSpace(id)
	if target == "" {
		return errors.New("store: session id is required")
	}

	if err := store.ExecuteWrite(ctx, g.db, func(ctx context.Context, tx *store.WriteTx) error {
		queries := sqlcgen.New(tx)
		if err := queries.DeletePermissionLogsBySession(ctx, target); err != nil {
			return fmt.Errorf("store: delete permission logs for session %q: %w", target, err)
		}
		if err := queries.DeleteTokenStatsBySession(ctx, target); err != nil {
			return fmt.Errorf("store: delete token stats for session %q: %w", target, err)
		}

		affected, err := queries.DeleteSession(ctx, target)
		if err != nil {
			return fmt.Errorf("store: delete session row %q: %w", target, err)
		}
		if affected == 0 {
			return fmt.Errorf("%w: %s", store.ErrSessionNotFound, target)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("store: delete session %q: %w", target, err)
	}
	return nil
}
