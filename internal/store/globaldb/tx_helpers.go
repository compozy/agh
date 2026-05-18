package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type networkSQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type globalSQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func rollbackTx(tx *sql.Tx, action string) error {
	if tx == nil {
		return nil
	}
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		return fmt.Errorf("store: rollback %s transaction: %w", action, err)
	}
	return nil
}

func rollbackImmediate(ctx context.Context, conn *sql.Conn, action string) error {
	if conn == nil {
		return nil
	}
	if _, err := conn.ExecContext(ctx, "ROLLBACK"); err != nil {
		return fmt.Errorf("store: rollback %s transaction: %w", action, err)
	}
	return nil
}

func restoreForeignKeys(ctx context.Context, conn *sql.Conn) error {
	if conn == nil {
		return nil
	}
	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("store: restore sqlite foreign keys: %w", err)
	}
	return nil
}

func joinCleanupError(target *error, cleanupErr error) {
	if cleanupErr == nil || target == nil {
		return
	}
	if *target == nil {
		*target = cleanupErr
		return
	}
	*target = errors.Join(*target, cleanupErr)
}

func (g *GlobalDB) withNetworkImmediateTransaction(
	ctx context.Context,
	action string,
	run func(exec networkSQLExecutor) error,
) (err error) {
	conn, err := g.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("store: open connection for %s: %w", action, err)
	}
	defer func() {
		_ = conn.Close()
	}()

	rollbackCtx := context.WithoutCancel(ctx)
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("store: begin immediate %s transaction: %w", action, err)
	}

	finished := false
	defer func() {
		if !finished {
			joinCleanupError(&err, rollbackImmediate(rollbackCtx, conn, action))
		}
	}()

	if err := run(conn); err != nil {
		return err
	}
	if _, err = conn.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("store: commit %s transaction: %w", action, err)
	}

	finished = true
	return nil
}
