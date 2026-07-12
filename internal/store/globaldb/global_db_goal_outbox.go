package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/goal"
	"github.com/compozy/agh/internal/store"
)

var _ goal.SessionOutboxStore = (*GlobalDB)(nil)

const goalSessionOutboxSelectColumns = `
	id, event_id, workspace_id, origin_session_id, loop_run_id, bound_session_id,
	cause, created_at, delivered_at`

// EnqueueGoalSessionOutbox inserts or returns one byte-identical projection event.
func (g *GlobalDB) EnqueueGoalSessionOutbox(
	ctx context.Context,
	request goal.EnqueueSessionOutboxRequest,
) (goal.SessionOutboxEvent, error) {
	if err := g.checkReady(ctx, "enqueue goal session outbox"); err != nil {
		return goal.SessionOutboxEvent{}, err
	}

	var event goal.SessionOutboxEvent
	err := g.withImmediateTransaction(ctx, "enqueue goal session outbox", func(exec globalSQLExecutor) error {
		var enqueueErr error
		event, enqueueErr = enqueueGoalSessionOutboxWithExecutor(ctx, exec, request)
		return enqueueErr
	})
	if err != nil {
		return goal.SessionOutboxEvent{}, err
	}
	return event, nil
}

// ClaimGoalSessionOutbox lists the oldest pending projection events for retryable relay.
func (g *GlobalDB) ClaimGoalSessionOutbox(
	ctx context.Context,
	limit int,
) (events []goal.SessionOutboxEvent, err error) {
	if err := g.checkReady(ctx, "claim goal session outbox"); err != nil {
		return nil, err
	}
	if limit < 0 || limit > 200 {
		return nil, fmt.Errorf("%w: goal outbox claim limit must be between 0 and 200", looppkg.ErrValidation)
	}
	if limit == 0 {
		limit = 50
	}
	rows, err := g.db.QueryContext(
		ctx,
		`SELECT `+goalSessionOutboxSelectColumns+`
		 FROM loop_goal_session_outbox
		 WHERE delivered_at IS NULL
		 ORDER BY id ASC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: claim goal session outbox: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("store: close goal session outbox rows: %w", closeErr))
		}
	}()

	events = make([]goal.SessionOutboxEvent, 0, limit)
	for rows.Next() {
		event, scanErr := scanGoalSessionOutbox(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("store: scan goal session outbox claim: %w", scanErr)
		}
		events = append(events, event)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("store: iterate goal session outbox claim: %w", rowsErr)
	}
	return events, nil
}

// AcknowledgeGoalSessionOutbox records the first successful delivery timestamp.
func (g *GlobalDB) AcknowledgeGoalSessionOutbox(
	ctx context.Context,
	eventID string,
	deliveredAt time.Time,
) error {
	if err := g.checkReady(ctx, "acknowledge goal session outbox"); err != nil {
		return err
	}
	normalizedEventID := strings.TrimSpace(eventID)
	if normalizedEventID == "" {
		return fmt.Errorf("%w: goal outbox event_id is required", looppkg.ErrValidation)
	}
	if deliveredAt.IsZero() {
		return fmt.Errorf("%w: goal outbox delivered_at is required", looppkg.ErrValidation)
	}
	deliveredAt = deliveredAt.UTC()
	return g.withImmediateTransaction(ctx, "acknowledge goal session outbox", func(exec globalSQLExecutor) error {
		event, err := loadGoalSessionOutboxByEventID(ctx, exec, normalizedEventID)
		if err != nil {
			return err
		}
		if deliveredAt.Before(event.CreatedAt) {
			return fmt.Errorf("%w: goal outbox delivered_at precedes created_at", looppkg.ErrValidation)
		}
		if event.DeliveredAt != nil {
			return nil
		}
		if _, err := exec.ExecContext(
			ctx,
			`UPDATE loop_goal_session_outbox SET delivered_at = ?
			 WHERE event_id = ? AND delivered_at IS NULL`,
			deliveredAt,
			normalizedEventID,
		); err != nil {
			return fmt.Errorf("store: acknowledge goal session outbox event %q: %w", normalizedEventID, err)
		}
		return nil
	})
}

func enqueueGoalSessionOutboxWithExecutor(
	ctx context.Context,
	exec globalSQLExecutor,
	request goal.EnqueueSessionOutboxRequest,
) (goal.SessionOutboxEvent, error) {
	request = normalizeGoalSessionOutboxRequest(request)
	if err := request.Validate(); err != nil {
		return goal.SessionOutboxEvent{}, err
	}
	if err := validateGoalOutboxTarget(ctx, exec, request); err != nil {
		return goal.SessionOutboxEvent{}, err
	}
	if _, err := exec.ExecContext(
		ctx,
		`INSERT INTO loop_goal_session_outbox (
			event_id, workspace_id, origin_session_id, loop_run_id,
			bound_session_id, cause, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(event_id) DO NOTHING`,
		request.EventID,
		string(request.WorkspaceID),
		request.OriginSessionID,
		string(request.LoopRunID),
		nullableGoalOutboxBoundSessionID(request.BoundSessionID),
		string(request.Cause),
		request.CreatedAt,
	); err != nil {
		return goal.SessionOutboxEvent{}, fmt.Errorf(
			"store: insert goal session outbox event %q: %w",
			request.EventID,
			err,
		)
	}
	loaded, err := loadGoalSessionOutboxByEventID(ctx, exec, request.EventID)
	if err != nil {
		return goal.SessionOutboxEvent{}, err
	}
	if !goalSessionOutboxMatchesRequest(loaded, request) {
		return goal.SessionOutboxEvent{}, fmt.Errorf(
			"%w: goal outbox event %q already has a different payload",
			looppkg.ErrTransitionConflict,
			request.EventID,
		)
	}
	return loaded, nil
}

func normalizeGoalSessionOutboxRequest(
	request goal.EnqueueSessionOutboxRequest,
) goal.EnqueueSessionOutboxRequest {
	request.EventID = strings.TrimSpace(request.EventID)
	request.WorkspaceID = looppkg.WorkspaceID(strings.TrimSpace(string(request.WorkspaceID)))
	request.OriginSessionID = strings.TrimSpace(request.OriginSessionID)
	request.LoopRunID = looppkg.RunID(strings.TrimSpace(string(request.LoopRunID)))
	if request.BoundSessionID != nil {
		normalizedBoundSessionID := strings.TrimSpace(*request.BoundSessionID)
		request.BoundSessionID = &normalizedBoundSessionID
	}
	request.Cause = goal.SessionOutboxCause(strings.TrimSpace(string(request.Cause)))
	request.CreatedAt = request.CreatedAt.UTC()
	return request
}

func validateGoalOutboxTarget(
	ctx context.Context,
	exec globalSQLExecutor,
	request goal.EnqueueSessionOutboxRequest,
) error {
	key := goal.TurnKey{
		WorkspaceID: request.WorkspaceID,
		LoopRunID:   request.LoopRunID,
		Generation:  1,
		NodeID:      "session-outbox",
	}
	if err := validateGoalRunWorkspace(ctx, exec, key); err != nil {
		return err
	}
	var originKind string
	var originSessionID sql.NullString
	if err := exec.QueryRowContext(
		ctx,
		`SELECT origin_kind, origin_session_id FROM loop_runs WHERE id = ? AND workspace_id = ?`,
		string(request.LoopRunID),
		string(request.WorkspaceID),
	).Scan(&originKind, &originSessionID); err != nil {
		return fmt.Errorf("store: load goal outbox run origin %q: %w", request.LoopRunID, err)
	}
	if originKind != goalRunOriginKindSession || !originSessionID.Valid ||
		strings.TrimSpace(originSessionID.String) != request.OriginSessionID {
		return fmt.Errorf(
			"%w: goal outbox origin does not match session-origin run %q",
			looppkg.ErrTransitionConflict,
			request.LoopRunID,
		)
	}
	if err := validateGoalOutboxSessionWorkspace(
		ctx,
		exec,
		request.OriginSessionID,
		request.WorkspaceID,
	); err != nil {
		return err
	}
	if request.BoundSessionID != nil {
		if err := validateGoalOutboxSessionWorkspace(
			ctx,
			exec,
			*request.BoundSessionID,
			request.WorkspaceID,
		); err != nil {
			return err
		}
	}
	return nil
}

func validateGoalOutboxSessionWorkspace(
	ctx context.Context,
	exec globalSQLExecutor,
	sessionID string,
	workspaceID looppkg.WorkspaceID,
) error {
	var exists int
	err := exec.QueryRowContext(
		ctx,
		`SELECT 1 FROM sessions WHERE id = ? AND workspace_id = ?`,
		sessionID,
		string(workspaceID),
	).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: %s", store.ErrSessionNotFound, sessionID)
	}
	if err != nil {
		return fmt.Errorf("store: validate goal outbox session %q: %w", sessionID, err)
	}
	return nil
}

func loadGoalSessionOutboxByEventID(
	ctx context.Context,
	exec globalSQLExecutor,
	eventID string,
) (goal.SessionOutboxEvent, error) {
	event, err := scanGoalSessionOutbox(exec.QueryRowContext(
		ctx,
		`SELECT `+goalSessionOutboxSelectColumns+`
		 FROM loop_goal_session_outbox WHERE event_id = ?`,
		eventID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return goal.SessionOutboxEvent{}, fmt.Errorf("%w: %s", goal.ErrOutboxEventNotFound, eventID)
	}
	if err != nil {
		return goal.SessionOutboxEvent{}, fmt.Errorf("store: load goal session outbox event %q: %w", eventID, err)
	}
	return event, nil
}

func scanGoalSessionOutbox(scanner rowScanner) (goal.SessionOutboxEvent, error) {
	var event goal.SessionOutboxEvent
	var boundSessionID sql.NullString
	var createdAtRaw any
	var deliveredAtRaw any
	if err := scanner.Scan(
		&event.ID,
		&event.EventID,
		&event.WorkspaceID,
		&event.OriginSessionID,
		&event.LoopRunID,
		&boundSessionID,
		&event.Cause,
		&createdAtRaw,
		&deliveredAtRaw,
	); err != nil {
		return goal.SessionOutboxEvent{}, err
	}
	if boundSessionID.Valid {
		value := boundSessionID.String
		event.BoundSessionID = &value
	}
	var err error
	event.CreatedAt, err = parseGoalTimestampValue(createdAtRaw, "outbox created_at")
	if err != nil {
		return goal.SessionOutboxEvent{}, err
	}
	event.DeliveredAt, err = parseOptionalGoalTimestampValue(deliveredAtRaw, "outbox delivered_at")
	if err != nil {
		return goal.SessionOutboxEvent{}, err
	}
	return event, nil
}

func goalSessionOutboxMatchesRequest(
	event goal.SessionOutboxEvent,
	request goal.EnqueueSessionOutboxRequest,
) bool {
	return event.EventID == request.EventID &&
		event.WorkspaceID == request.WorkspaceID &&
		event.OriginSessionID == request.OriginSessionID &&
		event.LoopRunID == request.LoopRunID &&
		goalOptionalStringEqual(event.BoundSessionID, request.BoundSessionID) &&
		event.Cause == request.Cause &&
		event.CreatedAt.Equal(request.CreatedAt)
}

func nullableGoalOutboxBoundSessionID(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func goalOptionalStringEqual(left *string, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
