package globaldb

import (
	"context"
	"fmt"
	"log/slog"

	taskpkg "github.com/compozy/agh/internal/task"
)

type taskTransactionExecutor struct {
	taskSQLExecutor
	committedEvents []taskpkg.EventRecord
}

func (e *taskTransactionExecutor) collectTaskEvent(record taskpkg.EventRecord) {
	e.committedEvents = append(e.committedEvents, record)
}

type taskEventCollector interface {
	collectTaskEvent(record taskpkg.EventRecord)
}

func collectTaskEvent(exec taskSQLExecutor, record taskpkg.EventRecord) {
	collector, ok := exec.(taskEventCollector)
	if !ok {
		return
	}
	collector.collectTaskEvent(record)
}

func (g *GlobalDB) withTaskImmediateTransaction(
	ctx context.Context,
	action string,
	run func(exec taskSQLExecutor) error,
) (err error) {
	conn, err := g.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("store: open connection for %s: %w", action, err)
	}

	transaction := &taskTransactionExecutor{taskSQLExecutor: conn}
	committed := false
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			slog.Warn("store: close task transaction connection", "action", action, "error", closeErr)
		}
		if committed && err == nil {
			g.notifyCommittedTaskEvents(context.WithoutCancel(ctx), transaction.committedEvents)
		}
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

	if err := run(transaction); err != nil {
		return err
	}

	if _, err = conn.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("store: commit %s transaction: %w", action, err)
	}

	finished = true
	committed = true
	return nil
}
