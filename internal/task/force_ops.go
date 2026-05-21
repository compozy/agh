package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	diagnosticcontract "github.com/pedronauck/agh/internal/diagnosticcontract"
	diagnosticitems "github.com/pedronauck/agh/internal/diagnostics"
)

const forceOpsDocURL = "/runtime/core/autonomy/task-runs-and-leases#force-operations"

type inputQueueGenerationStore interface {
	AdvanceSessionInputGeneration(ctx context.Context, sessionID string, now time.Time) (int64, error)
	CancelPendingSessionInputs(ctx context.Context, sessionID string, generation int64, now time.Time) (int, error)
}

type forceRunRateLimiter struct {
	mu      sync.Mutex
	windows map[string]forceRunRateWindow
}

type forceRunRateWindow struct {
	start time.Time
	count int
}

func newForceRunRateLimiter() *forceRunRateLimiter {
	return &forceRunRateLimiter{windows: make(map[string]forceRunRateWindow)}
}

func normalizeForceRecoveryOptions(options ForceRecoveryOptions) ForceRecoveryOptions {
	if options.RateLimitPerMinute <= 0 {
		options.RateLimitPerMinute = DefaultForceRunRateLimitPerMinute
	}
	return options
}

func (l *forceRunRateLimiter) allow(actor ActorIdentity, taskID string, now time.Time, limit int) bool {
	if l == nil || limit <= 0 {
		return true
	}
	key := string(actor.Kind.Normalize()) + ":" + strings.TrimSpace(actor.Ref) + ":" + strings.TrimSpace(taskID)
	l.mu.Lock()
	defer l.mu.Unlock()

	window := l.windows[key]
	if window.start.IsZero() || now.Sub(window.start) >= time.Minute {
		l.windows[key] = forceRunRateWindow{start: now, count: 1}
		return true
	}
	if window.count >= limit {
		return false
	}
	window.count++
	l.windows[key] = window
	return true
}

// ForceReleaseRun releases one claimed run without requiring the raw claim token.
func (m *Service) ForceReleaseRun(
	ctx context.Context,
	runID string,
	release ForceReleaseRun,
	actor ActorContext,
) (*Run, error) {
	if err := m.requireForceRunAuthority(actor); err != nil {
		return nil, err
	}
	normalized, err := normalizeForceReleaseRun(release)
	if err != nil {
		return nil, err
	}
	previous, taskRecord, err := m.loadRunWithTask(ctx, runID)
	if err != nil {
		return nil, err
	}
	if err := m.requireForceRunRate(actor, taskRecord.ID); err != nil {
		return nil, err
	}
	if previous.Status.Normalize() != TaskRunStatusClaimed {
		return nil, forceRunDiagnosticError(
			diagnosticcontract.CodeTaskRunNotReleasable,
			"Task run cannot be force released",
			fmt.Sprintf(
				"Run %s is %s; only claimed runs can be force released.",
				previous.ID,
				previous.Status.Normalize(),
			),
			diagnosticcontract.SeverityError,
			fmt.Sprintf("agh task inspect %s", previous.ID),
			map[string]any{runEvidenceIDKey: previous.ID, leaseStatusKey: string(previous.Status.Normalize())},
			ErrInvalidStatusTransition,
		)
	}

	mutation, err := m.store.ForceReleaseTaskRun(ctx, ForceReleaseRunMutation{
		RunID: previous.ID,
		Now:   m.now().UTC(),
	})
	if err != nil {
		return nil, err
	}
	queueGeneration, canceledInputs, err := m.invalidateForceRunInputs(ctx, mutation.Previous)
	if err != nil {
		return nil, err
	}
	reconciledTask, err := m.reconcileTaskCascade(ctx, taskRecord.ID)
	if err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(
		ctx,
		mutation.Run.TaskID,
		mutation.Run.ID,
		taskEventRunReleased,
		actor,
		releasedRunPayload{
			Manual:                          true,
			ActorKind:                       actor.Actor.Kind.Normalize(),
			ActorID:                         actor.Actor.Ref,
			PreviousStatus:                  mutation.Previous.Status,
			Status:                          mutation.Run.Status,
			TaskStatus:                      reconciledTask.Status,
			Reason:                          normalized.Reason,
			SessionID:                       mutation.Previous.SessionID,
			PreviousSessionID:               mutation.Previous.SessionID,
			PreviousClaimTokenHashTruncated: truncateClaimTokenHash(mutation.Previous.ClaimTokenHash),
			PreviousLeaseUntil:              optionalPayloadTime(mutation.Previous.LeaseUntil),
			QueueGeneration:                 queueGeneration,
			CanceledQueuedInputs:            canceledInputs,
		},
	); err != nil {
		return nil, err
	}
	m.dispatchTaskRunReleased(ctx, mutation.Run, reconciledTask, actor, mutation.Previous, normalized.Reason)
	return &mutation.Run, nil
}

// ForceFailRun marks one queued or claimed run failed without requiring the raw claim token.
func (m *Service) ForceFailRun(
	ctx context.Context,
	runID string,
	failure ForceFailRun,
	actor ActorContext,
) (*Run, error) {
	if err := m.requireForceRunAuthority(actor); err != nil {
		return nil, err
	}
	normalized, err := normalizeForceFailRun(failure)
	if err != nil {
		return nil, err
	}
	previous, taskRecord, err := m.loadRunWithTask(ctx, runID)
	if err != nil {
		return nil, err
	}
	if err := m.requireForceRunRate(actor, taskRecord.ID); err != nil {
		return nil, err
	}
	if err := requireForceFailStatus(previous); err != nil {
		return nil, err
	}

	mutation, err := m.store.ForceFailTaskRun(ctx, ForceFailRunMutation{
		RunID:  previous.ID,
		Reason: normalized.Reason,
		Now:    m.now().UTC(),
	})
	if err != nil {
		return nil, err
	}
	queueGeneration, canceledInputs, err := m.invalidateForceRunInputs(ctx, mutation.Previous)
	if err != nil {
		return nil, err
	}
	reconciledTask, err := m.reconcileTaskCascade(ctx, taskRecord.ID)
	if err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(
		ctx,
		mutation.Run.TaskID,
		mutation.Run.ID,
		taskEventRunOperatorForcedFail,
		actor,
		operatorForcedFailPayload{
			Manual:               true,
			ActorKind:            actor.Actor.Kind.Normalize(),
			ActorID:              actor.Actor.Ref,
			PreviousStatus:       mutation.Previous.Status,
			Status:               mutation.Run.Status,
			TaskStatus:           reconciledTask.Status,
			Reason:               normalized.Reason,
			SessionID:            mutation.Previous.SessionID,
			QueueGeneration:      queueGeneration,
			CanceledQueuedInputs: canceledInputs,
			Metadata:             cloneRawJSON(normalized.Metadata),
		},
	); err != nil {
		return nil, err
	}
	return &mutation.Run, nil
}

// RetryRun creates one new queued run linked to a failed source run.
func (m *Service) RetryRun(
	ctx context.Context,
	runID string,
	retry RetryRunRequest,
	actor ActorContext,
) (*RetryRunResult, error) {
	if err := m.requireForceRunAuthority(actor); err != nil {
		return nil, err
	}
	normalized, err := normalizeRetryRunRequest(retry)
	if err != nil {
		return nil, err
	}
	source, taskRecord, err := m.loadRunWithTask(ctx, runID)
	if err != nil {
		return nil, err
	}
	if err := m.requireForceRunRate(actor, taskRecord.ID); err != nil {
		return nil, err
	}
	if source.Status.Normalize() != TaskRunStatusFailed {
		return nil, forceRunDiagnosticError(
			diagnosticcontract.CodeTaskRunStillActive,
			"Task run cannot be retried",
			fmt.Sprintf("Run %s is %s; only failed runs can be retried.", source.ID, source.Status.Normalize()),
			diagnosticcontract.SeverityError,
			fmt.Sprintf("agh task inspect %s", source.ID),
			map[string]any{runEvidenceIDKey: source.ID, leaseStatusKey: string(source.Status.Normalize())},
			ErrInvalidStatusTransition,
		)
	}
	if err := m.requireRetryChainDepth(ctx, source); err != nil {
		return nil, err
	}

	result, err := m.store.RetryTaskRun(ctx, RetryRunMutation{
		SourceRunID: source.ID,
		NewRunID:    m.newID("run"),
		Origin:      actor.Origin,
		Metadata:    normalized.Metadata,
		QueuedAt:    m.now().UTC(),
	})
	if err != nil {
		return nil, err
	}
	reconciledTask, err := m.reconcileTaskCascade(ctx, taskRecord.ID)
	if err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(
		ctx,
		result.Run.TaskID,
		result.Run.ID,
		taskEventRunOperatorRetry,
		actor,
		operatorRetryPayload{
			Manual:      true,
			ActorKind:   actor.Actor.Kind.Normalize(),
			ActorID:     actor.Actor.Ref,
			SourceRunID: result.PreviousRun.ID,
			NewRunID:    result.Run.ID,
			TaskStatus:  reconciledTask.Status,
		},
	); err != nil {
		return nil, err
	}
	return &result, nil
}

// BulkForceReleaseRuns applies force release one row at a time to preserve per-row preconditions.
func (m *Service) BulkForceReleaseRuns(
	ctx context.Context,
	req BulkForceRunRequest,
	actor ActorContext,
) (BulkForceRunResult, error) {
	normalized, err := normalizeBulkForceRunRequest(req, false)
	if err != nil {
		return BulkForceRunResult{}, err
	}
	result := BulkForceRunResult{Items: make([]BulkForceRunItem, 0, len(normalized.RunIDs))}
	for _, id := range normalized.RunIDs {
		run, err := m.ForceReleaseRun(ctx, id, ForceReleaseRun{Reason: normalized.Reason}, actor)
		result.Items = append(result.Items, bulkForceRunItem(id, run, err))
	}
	return result, nil
}

// BulkForceFailRuns applies forced failure one row at a time to preserve per-row preconditions.
func (m *Service) BulkForceFailRuns(
	ctx context.Context,
	req BulkForceRunRequest,
	actor ActorContext,
) (BulkForceRunResult, error) {
	normalized, err := normalizeBulkForceRunRequest(req, true)
	if err != nil {
		return BulkForceRunResult{}, err
	}
	result := BulkForceRunResult{Items: make([]BulkForceRunItem, 0, len(normalized.RunIDs))}
	for _, id := range normalized.RunIDs {
		run, err := m.ForceFailRun(
			ctx,
			id,
			ForceFailRun{Reason: normalized.Reason, Metadata: normalized.Metadata},
			actor,
		)
		result.Items = append(result.Items, bulkForceRunItem(id, run, err))
	}
	return result, nil
}

func bulkForceRunItem(id string, run *Run, err error) BulkForceRunItem {
	item := BulkForceRunItem{RunID: strings.TrimSpace(id), OK: err == nil, Err: err}
	if run != nil {
		runCopy := *run
		item.Run = &runCopy
	}
	return item
}

func (m *Service) requireForceRunAuthority(actor ActorContext) error {
	if err := requireWriteAuthority(actor); err != nil {
		return err
	}
	if m.forceRecovery.AllowAgentForce || actor.Actor.Kind.Normalize() != ActorKindAgentSession {
		return nil
	}
	return forceRunDiagnosticError(
		diagnosticcontract.CodeForbiddenOperatorAction,
		"Force operation is disabled for agents",
		"task.recovery.allow_agent_force is false, so only non-agent operators can run this recovery action.",
		diagnosticcontract.SeverityError,
		"agh config set task.recovery.allow_agent_force true",
		map[string]any{"actor_kind": string(actor.Actor.Kind.Normalize()), "actor_id": actor.Actor.Ref},
		ErrForbiddenOperatorAction,
	)
}

func (m *Service) requireForceRunRate(actor ActorContext, taskID string) error {
	if actor.Actor.Kind.Normalize() != ActorKindAgentSession {
		return nil
	}
	now := m.now().UTC()
	if m.forceRateLimiter.allow(actor.Actor, taskID, now, m.forceRecovery.RateLimitPerMinute) {
		return nil
	}
	return forceRunDiagnosticError(
		diagnosticcontract.CodeForceOpRateLimited,
		"Force operation rate limit exceeded",
		fmt.Sprintf(
			"Actor %s exceeded %d force operations per minute for task %s.",
			actor.Actor.Ref,
			m.forceRecovery.RateLimitPerMinute,
			taskID,
		),
		diagnosticcontract.SeverityWarn,
		"agh task inspect "+taskID,
		map[string]any{
			"actor_kind":      string(actor.Actor.Kind.Normalize()),
			"actor_id":        actor.Actor.Ref,
			taskEvidenceIDKey: taskID,
			"limit":           m.forceRecovery.RateLimitPerMinute,
		},
		ErrForceOpRateLimited,
	)
}

func (m *Service) invalidateForceRunInputs(ctx context.Context, previous Run) (int64, int, error) {
	sessionID := strings.TrimSpace(previous.SessionID)
	if sessionID == "" {
		return 0, 0, nil
	}
	queueStore, ok := m.store.(inputQueueGenerationStore)
	if !ok {
		return 0, 0, nil
	}
	now := m.now().UTC()
	generation, err := queueStore.AdvanceSessionInputGeneration(ctx, sessionID, now)
	if err != nil {
		return 0, 0, fmt.Errorf("task: advance input generation for force operation on session %q: %w", sessionID, err)
	}
	canceled, err := queueStore.CancelPendingSessionInputs(ctx, sessionID, generation, now)
	if err != nil {
		return 0, 0, fmt.Errorf(
			"task: cancel stale input generation for force operation on session %q: %w",
			sessionID,
			err,
		)
	}
	return generation, canceled, nil
}

func requireForceFailStatus(run Run) error {
	switch run.Status.Normalize() {
	case TaskRunStatusQueued, TaskRunStatusClaimed:
		return nil
	case TaskRunStatusCompleted, TaskRunStatusFailed, TaskRunStatusCanceled:
		return forceRunDiagnosticError(
			diagnosticcontract.CodeTaskRunAlreadyTerminal,
			"Task run is already terminal",
			fmt.Sprintf("Run %s is already %s and cannot be force failed.", run.ID, run.Status.Normalize()),
			diagnosticcontract.SeverityInfo,
			fmt.Sprintf("agh task inspect %s", run.ID),
			map[string]any{runEvidenceIDKey: run.ID, leaseStatusKey: string(run.Status.Normalize())},
			ErrInvalidStatusTransition,
		)
	default:
		return forceRunDiagnosticError(
			diagnosticcontract.CodeTaskRunStillActive,
			"Task run is still active",
			fmt.Sprintf(
				"Run %s is %s; active runs must be stopped before forced failure.",
				run.ID,
				run.Status.Normalize(),
			),
			diagnosticcontract.SeverityError,
			fmt.Sprintf("agh task cancel %s --reason %q", run.ID, "stop before force fail"),
			map[string]any{runEvidenceIDKey: run.ID, leaseStatusKey: string(run.Status.Normalize())},
			ErrInvalidStatusTransition,
		)
	}
}

func (m *Service) requireRetryChainDepth(ctx context.Context, source Run) error {
	runs, err := m.store.ListTaskRuns(ctx, RunQuery{TaskID: source.TaskID})
	if err != nil {
		return err
	}
	byID := make(map[string]Run, len(runs))
	for _, run := range runs {
		byID[run.ID] = run
	}
	depth := 0
	for current := source; strings.TrimSpace(current.PreviousRunID) != ""; {
		depth++
		if depth >= MaxRetryRunChainDepth {
			return forceRunDiagnosticError(
				diagnosticcontract.CodeRetryChainTooDeep,
				"Retry chain is too deep",
				fmt.Sprintf("Run %s already has retry depth %d.", source.ID, depth),
				diagnosticcontract.SeverityError,
				fmt.Sprintf("agh task inspect %s", source.TaskID),
				map[string]any{runEvidenceIDKey: source.ID, taskEvidenceIDKey: source.TaskID, "depth": depth},
				ErrRetryChainTooDeep,
			)
		}
		previous, ok := byID[strings.TrimSpace(current.PreviousRunID)]
		if !ok {
			break
		}
		current = previous
	}
	return nil
}

func normalizeForceReleaseRun(req ForceReleaseRun) (ForceReleaseRun, error) {
	req.Reason = strings.TrimSpace(req.Reason)
	req.Metadata = normalizeRawJSON(req.Metadata)
	if err := ValidateMetadataSize(req.Metadata, "force_release.metadata"); err != nil {
		return ForceReleaseRun{}, err
	}
	return req, nil
}

func normalizeForceFailRun(req ForceFailRun) (ForceFailRun, error) {
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Reason == "" {
		return ForceFailRun{}, forceRunDiagnosticError(
			diagnosticcontract.CodeForceOpRequiresReason,
			"Force fail requires a reason",
			"Provide --reason so the recovery audit event explains why the run was failed.",
			diagnosticcontract.SeverityError,
			"agh task fail <run-id> --reason \"operator recovery\"",
			nil,
			ErrForceOpRequiresReason,
		)
	}
	req.Metadata = normalizeRawJSON(req.Metadata)
	if err := ValidateMetadataSize(req.Metadata, "force_fail.metadata"); err != nil {
		return ForceFailRun{}, err
	}
	return req, nil
}

func normalizeRetryRunRequest(req RetryRunRequest) (RetryRunRequest, error) {
	req.Metadata = normalizeRawJSON(req.Metadata)
	if err := ValidateMetadataSize(req.Metadata, "retry_run.metadata"); err != nil {
		return RetryRunRequest{}, err
	}
	return req, nil
}

func normalizeBulkForceRunRequest(req BulkForceRunRequest, requireReason bool) (BulkForceRunRequest, error) {
	if len(req.RunIDs) == 0 {
		return BulkForceRunRequest{}, fmt.Errorf("%w: bulk force run_ids is required", ErrValidation)
	}
	if len(req.RunIDs) > MaxForceRunBulkIDs {
		return BulkForceRunRequest{}, forceRunDiagnosticError(
			diagnosticcontract.CodeBulkTooLarge,
			"Bulk force operation is too large",
			fmt.Sprintf("Bulk force operations accept at most %d run ids.", MaxForceRunBulkIDs),
			diagnosticcontract.SeverityError,
			"",
			map[string]any{"limit": MaxForceRunBulkIDs, "count": len(req.RunIDs)},
			ErrBulkTooLarge,
		)
	}
	seen := make(map[string]struct{}, len(req.RunIDs))
	ids := make([]string, 0, len(req.RunIDs))
	for _, id := range req.RunIDs {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			return BulkForceRunRequest{}, fmt.Errorf(
				"%w: bulk force run_ids must not contain empty values",
				ErrValidation,
			)
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		ids = append(ids, trimmed)
	}
	req.RunIDs = ids
	req.Reason = strings.TrimSpace(req.Reason)
	if requireReason && req.Reason == "" {
		return BulkForceRunRequest{}, forceRunDiagnosticError(
			diagnosticcontract.CodeForceOpRequiresReason,
			"Force fail requires a reason",
			"Provide a reason so every failed row has clear audit evidence.",
			diagnosticcontract.SeverityError,
			"agh task fail <run-id...> --reason \"operator recovery\"",
			nil,
			ErrForceOpRequiresReason,
		)
	}
	req.Metadata = normalizeRawJSON(req.Metadata)
	if err := ValidateMetadataSize(req.Metadata, "bulk_force.metadata"); err != nil {
		return BulkForceRunRequest{}, err
	}
	return req, nil
}

func forceRunDiagnosticError(
	code string,
	title string,
	message string,
	severity string,
	suggestedCommand string,
	evidence map[string]any,
	cause error,
) error {
	item := diagnosticitems.NewItem(
		"task.force."+code,
		code,
		diagnosticcontract.CategoryTask,
		title,
		message,
		severity,
		diagnosticcontract.FreshnessLive,
		diagnosticitems.WithDocURL(forceOpsDocURL),
		diagnosticitems.WithSuggestedCommand(suggestedCommand),
		diagnosticitems.WithEvidence(evidence),
	)
	return diagnosticitems.NewStructuredError(item, cause)
}

func optionalPayloadTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	normalized := value.UTC()
	return &normalized
}

type operatorForcedFailPayload struct {
	Manual               bool            `json:"manual"`
	ActorKind            ActorKind       `json:"actor_kind"`
	ActorID              string          `json:"actor_id"`
	PreviousStatus       RunStatus       `json:"previous_status"`
	Status               RunStatus       `json:"status"`
	TaskStatus           Status          `json:"task_status"`
	Reason               string          `json:"reason"`
	SessionID            string          `json:"session_id,omitempty"`
	QueueGeneration      int64           `json:"queue_generation,omitempty"`
	CanceledQueuedInputs int             `json:"canceled_queued_inputs,omitempty"`
	Metadata             json.RawMessage `json:"metadata,omitempty"`
}

type operatorRetryPayload struct {
	Manual      bool      `json:"manual"`
	ActorKind   ActorKind `json:"actor_kind"`
	ActorID     string    `json:"actor_id"`
	SourceRunID string    `json:"source_run_id"`
	NewRunID    string    `json:"new_run_id"`
	TaskStatus  Status    `json:"task_status"`
}
