package store

import (
	"context"
	cryptorand "crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

const (
	defaultWriteMaxAttempts       = 15
	defaultWriteMinRetryDelay     = 20 * time.Millisecond
	defaultWriteMaxRetryDelay     = 150 * time.Millisecond
	defaultWriteCheckpointEvery   = 64
	defaultWriteRollbackTimeout   = 5 * time.Second
	sqlitePrimaryResultCodeMask   = 0xff
	sqliteBeginImmediateStatement = "BEGIN IMMEDIATE"
	sqliteCommitStatement         = "COMMIT"
	sqliteRollbackStatement       = "ROLLBACK"
)

var executeWriteSuccesses atomic.Uint64

// WriteTx is the single-connection transaction handle passed to ExecuteWrite callbacks.
type WriteTx struct {
	conn *sql.Conn
}

// ExecContext executes a statement inside the active write transaction.
func (tx *WriteTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if tx == nil || tx.conn == nil {
		return nil, errors.New("store: write transaction is closed")
	}
	return tx.conn.ExecContext(ctx, query, args...)
}

// QueryContext executes a query inside the active write transaction.
func (tx *WriteTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if tx == nil || tx.conn == nil {
		return nil, errors.New("store: write transaction is closed")
	}
	return tx.conn.QueryContext(ctx, query, args...)
}

// QueryRowContext executes a single-row query inside the active write transaction.
func (tx *WriteTx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return tx.conn.QueryRowContext(ctx, query, args...)
}

// ExecuteWrite runs fn inside a BEGIN IMMEDIATE transaction with bounded SQLITE_BUSY retries.
func ExecuteWrite(ctx context.Context, db *sql.DB, fn func(context.Context, *WriteTx) error) error {
	return executeWrite(ctx, db, defaultExecuteWriteConfig(), fn)
}

type executeWriteConfig struct {
	maxAttempts     int
	minRetryDelay   time.Duration
	maxRetryDelay   time.Duration
	checkpointEvery uint64
	jitter          func(time.Duration, time.Duration) time.Duration
	checkpoint      func(context.Context, *sql.DB) error
}

func defaultExecuteWriteConfig() executeWriteConfig {
	return executeWriteConfig{
		maxAttempts:     defaultWriteMaxAttempts,
		minRetryDelay:   defaultWriteMinRetryDelay,
		maxRetryDelay:   defaultWriteMaxRetryDelay,
		checkpointEvery: defaultWriteCheckpointEvery,
		jitter:          randomWriteRetryDelay,
		checkpoint:      Checkpoint,
	}
}

func executeWrite(
	ctx context.Context,
	db *sql.DB,
	cfg executeWriteConfig,
	fn func(context.Context, *WriteTx) error,
) error {
	if ctx == nil {
		return errors.New("store: execute write context is required")
	}
	if db == nil {
		return errors.New("store: execute write database is required")
	}
	if fn == nil {
		return errors.New("store: execute write callback is required")
	}
	cfg = normalizeExecuteWriteConfig(cfg)

	var lastErr error
	for attempt := 1; attempt <= cfg.maxAttempts; attempt++ {
		err := executeWriteAttempt(ctx, db, fn)
		if err == nil {
			return maybeCheckpointAfterWrite(ctx, db, cfg)
		}
		lastErr = err
		if !isSQLiteBusy(err) || attempt == cfg.maxAttempts {
			return err
		}
		if waitErr := waitForWriteRetry(ctx, cfg.jitter(cfg.minRetryDelay, cfg.maxRetryDelay)); waitErr != nil {
			return errors.Join(err, waitErr)
		}
	}

	return lastErr
}

func normalizeExecuteWriteConfig(cfg executeWriteConfig) executeWriteConfig {
	defaults := defaultExecuteWriteConfig()
	if cfg.maxAttempts <= 0 {
		cfg.maxAttempts = defaults.maxAttempts
	}
	if cfg.minRetryDelay <= 0 {
		cfg.minRetryDelay = defaults.minRetryDelay
	}
	if cfg.maxRetryDelay < cfg.minRetryDelay {
		cfg.maxRetryDelay = cfg.minRetryDelay
	}
	if cfg.jitter == nil {
		cfg.jitter = defaults.jitter
	}
	if cfg.checkpoint == nil {
		cfg.checkpoint = defaults.checkpoint
	}
	return cfg
}

func executeWriteAttempt(
	ctx context.Context,
	db *sql.DB,
	fn func(context.Context, *WriteTx) error,
) (err error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("store: acquire sqlite write connection: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			closeErr = fmt.Errorf("store: close sqlite write connection: %w", closeErr)
			if err == nil {
				err = closeErr
				return
			}
			err = errors.Join(err, closeErr)
		}
	}()

	if _, err := conn.ExecContext(ctx, sqliteBeginImmediateStatement); err != nil {
		return fmt.Errorf("store: begin immediate sqlite write: %w", err)
	}
	active := true
	defer func() {
		if !active {
			return
		}
		if rollbackErr := rollbackWriteTx(context.WithoutCancel(ctx), conn); rollbackErr != nil {
			if err == nil {
				err = rollbackErr
				return
			}
			err = errors.Join(err, rollbackErr)
		}
	}()

	tx := &WriteTx{conn: conn}
	if err := fn(ctx, tx); err != nil {
		return fmt.Errorf("store: execute sqlite write callback: %w", err)
	}
	if _, err := conn.ExecContext(ctx, sqliteCommitStatement); err != nil {
		return fmt.Errorf("store: commit sqlite write: %w", err)
	}
	active = false
	return nil
}

func rollbackWriteTx(ctx context.Context, conn *sql.Conn) error {
	rollbackCtx, cancel := context.WithTimeout(ctx, defaultWriteRollbackTimeout)
	defer cancel()
	if _, err := conn.ExecContext(rollbackCtx, sqliteRollbackStatement); err != nil {
		return fmt.Errorf("store: rollback sqlite write: %w", err)
	}
	return nil
}

func maybeCheckpointAfterWrite(ctx context.Context, db *sql.DB, cfg executeWriteConfig) error {
	if cfg.checkpointEvery == 0 {
		return nil
	}
	if executeWriteSuccesses.Add(1)%cfg.checkpointEvery != 0 {
		return nil
	}
	if err := cfg.checkpoint(ctx, db); err != nil {
		return fmt.Errorf("store: checkpoint after sqlite write: %w", err)
	}
	return nil
}

func waitForWriteRetry(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("store: wait for sqlite write retry: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}

func randomWriteRetryDelay(minDelay time.Duration, maxDelay time.Duration) time.Duration {
	if maxDelay <= minDelay {
		return minDelay
	}
	span := maxDelay - minDelay
	offset, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(span)+1))
	if err != nil {
		return minDelay
	}
	return minDelay + time.Duration(offset.Int64())
}

func isSQLiteBusy(err error) bool {
	if err == nil {
		return false
	}
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	code := sqliteErr.Code() & sqlitePrimaryResultCodeMask
	return code == sqlite3.SQLITE_BUSY || code == sqlite3.SQLITE_LOCKED
}
