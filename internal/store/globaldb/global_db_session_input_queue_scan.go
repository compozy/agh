package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/store"
)

func getSessionInputQueueEntry(
	ctx context.Context,
	exec globalSQLExecutor,
	sessionID string,
	entryID string,
) (store.SessionInputQueueEntry, error) {
	entry, err := scanSessionInputQueueEntry(exec.QueryRowContext(ctx, `
		SELECT `+sessionInputQueueColumns+`
		FROM session_input_queue
		WHERE session_id = ? AND id = ?`,
		sessionID,
		entryID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return store.SessionInputQueueEntry{}, fmt.Errorf("%w: %s", store.ErrSessionInputQueueEntryNotFound, entryID)
	}
	return entry, err
}

func scanSessionInputQueueEntry(row rowScanner) (store.SessionInputQueueEntry, error) {
	var entry store.SessionInputQueueEntry
	var fields sessionInputQueueScanFields
	if err := row.Scan(
		&entry.ID,
		&entry.SessionID,
		&entry.Status,
		&entry.Mode,
		&entry.Text,
		&entry.SessionGeneration,
		&entry.TaskRunID,
		&fields.runGeneration,
		&entry.AttemptCount,
		&fields.enqueuedAtRaw,
		&fields.dispatchStartedAt,
		&fields.sentAt,
		&fields.failedAt,
		&entry.FailureSummary,
		&fields.canceledAt,
		&fields.updatedAtRaw,
		&fields.loopRunID,
		&fields.ownerKind,
		&fields.ownerEpoch,
		&fields.bindingEpoch,
		&fields.promptID,
		&fields.promptKind,
		&fields.usageBase,
		&entry.PromptAttempt,
		&fields.dispatchable,
		&fields.activatedAt,
		&fields.dispatchTokenHash,
		&fields.fenceKind,
		&fields.fenceDisposition,
		&fields.fenceReason,
		&fields.fencedAt,
		&fields.terminalStart,
		&fields.terminalEnd,
		&fields.terminalKind,
		&fields.terminalStop,
		&fields.terminalDisposition,
		&fields.terminalReason,
		&fields.terminalTokensReported,
		&fields.terminalTokens,
		&fields.terminalAt,
	); err != nil {
		return store.SessionInputQueueEntry{}, fmt.Errorf("store: scan session input queue entry: %w", err)
	}
	fields.applyValues(&entry)
	if err := parseSessionInputQueueTimes(
		&entry,
		fields.enqueuedAtRaw,
		fields.updatedAtRaw,
		fields.dispatchStartedAt,
		fields.sentAt,
		fields.failedAt,
		fields.canceledAt,
		fields.activatedAt,
		fields.fencedAt,
		fields.terminalAt,
	); err != nil {
		return store.SessionInputQueueEntry{}, fmt.Errorf("store: parse session input queue entry times: %w", err)
	}
	return entry, nil
}

type sessionInputQueueScanFields struct {
	runGeneration          sql.NullInt64
	ownerEpoch             sql.NullInt64
	bindingEpoch           sql.NullInt64
	usageBase              sql.NullInt64
	terminalStart          sql.NullInt64
	terminalEnd            sql.NullInt64
	terminalTokens         sql.NullInt64
	loopRunID              sql.NullString
	ownerKind              sql.NullString
	promptID               sql.NullString
	promptKind             sql.NullString
	dispatchTokenHash      sql.NullString
	fenceKind              sql.NullString
	fenceDisposition       sql.NullString
	fenceReason            sql.NullString
	terminalKind           sql.NullString
	terminalStop           sql.NullString
	terminalDisposition    sql.NullString
	terminalReason         sql.NullString
	dispatchStartedAt      sql.NullString
	sentAt                 sql.NullString
	failedAt               sql.NullString
	canceledAt             sql.NullString
	activatedAt            sql.NullString
	fencedAt               sql.NullString
	terminalAt             sql.NullString
	enqueuedAtRaw          string
	updatedAtRaw           string
	dispatchable           int
	terminalTokensReported int
}

func (fields *sessionInputQueueScanFields) applyValues(entry *store.SessionInputQueueEntry) {
	entry.RunGeneration = nullableInt64Pointer(fields.runGeneration)
	entry.LoopRunID = strings.TrimSpace(fields.loopRunID.String)
	entry.OwnerKind = strings.TrimSpace(fields.ownerKind.String)
	entry.OwnerEpoch = nullableInt64Pointer(fields.ownerEpoch)
	entry.BindingEpoch = nullableInt64Pointer(fields.bindingEpoch)
	entry.PromptID = strings.TrimSpace(fields.promptID.String)
	entry.PromptKind = strings.TrimSpace(fields.promptKind.String)
	entry.DispatchTokenHash = strings.TrimSpace(fields.dispatchTokenHash.String)
	entry.OperationUsageBaseTokens = nullableInt64Pointer(fields.usageBase)
	entry.Dispatchable = fields.dispatchable != 0
	entry.FenceKind = strings.TrimSpace(fields.fenceKind.String)
	entry.FenceDisposition = strings.TrimSpace(fields.fenceDisposition.String)
	entry.FenceReasonCode = strings.TrimSpace(fields.fenceReason.String)
	entry.TerminalEventStartSeq = nullableInt64Pointer(fields.terminalStart)
	entry.TerminalEventEndSeq = nullableInt64Pointer(fields.terminalEnd)
	entry.TerminalKind = strings.TrimSpace(fields.terminalKind.String)
	entry.TerminalStopReason = strings.TrimSpace(fields.terminalStop.String)
	entry.TerminalDisposition = strings.TrimSpace(fields.terminalDisposition.String)
	entry.TerminalReasonCode = strings.TrimSpace(fields.terminalReason.String)
	entry.TerminalTokensReported = fields.terminalTokensReported != 0
	entry.TerminalTokensUsed = nullableInt64Pointer(fields.terminalTokens)
}

func parseSessionInputQueueTimes(
	entry *store.SessionInputQueueEntry,
	enqueuedAtRaw string,
	updatedAtRaw string,
	dispatchStartedAt sql.NullString,
	sentAt sql.NullString,
	failedAt sql.NullString,
	canceledAt sql.NullString,
	activatedAt sql.NullString,
	fencedAt sql.NullString,
	terminalAt sql.NullString,
) error {
	parsedEnqueuedAt, err := store.ParseTimestamp(enqueuedAtRaw)
	if err != nil {
		return fmt.Errorf("store: parse session input enqueued_at: %w", err)
	}
	entry.EnqueuedAt = parsedEnqueuedAt
	parsedUpdatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return fmt.Errorf("store: parse session input updated_at: %w", err)
	}
	entry.UpdatedAt = parsedUpdatedAt
	for name, target := range map[string]struct {
		raw sql.NullString
		set func(*time.Time)
	}{
		"dispatch_started_at": {dispatchStartedAt, func(value *time.Time) { entry.DispatchStartedAt = value }},
		"sent_at":             {sentAt, func(value *time.Time) { entry.SentAt = value }},
		"failed_at":           {failedAt, func(value *time.Time) { entry.FailedAt = value }},
		"canceled_at":         {canceledAt, func(value *time.Time) { entry.CanceledAt = value }},
		"activated_at":        {activatedAt, func(value *time.Time) { entry.ActivatedAt = value }},
		"fenced_at":           {fencedAt, func(value *time.Time) { entry.FencedAt = value }},
		"terminal_at":         {terminalAt, func(value *time.Time) { entry.TerminalAt = value }},
	} {
		value, err := parseOptionalSessionInputTimestamp(target.raw)
		if err != nil {
			return fmt.Errorf("store: parse session input %s: %w", name, err)
		}
		target.set(value)
	}
	return nil
}

func nullableInt64Pointer(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	result := value.Int64
	return &result
}

func parseOptionalSessionInputTimestamp(value sql.NullString) (*time.Time, error) {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil, nil
	}
	parsed, err := store.ParseTimestamp(value.String)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func requireSessionInputRowsAffected(result sql.Result, action string, id string) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: rows affected for %s %q: %w", action, id, err)
	}
	if rows == 0 {
		return fmt.Errorf("%w: %s", store.ErrSessionInputQueueEntryNotFound, id)
	}
	return nil
}
