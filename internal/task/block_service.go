package task

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/compozy/agh/internal/diagnostics"
	hookspkg "github.com/compozy/agh/internal/hooks"
)

const taskBlockIDPrefix = "block"

// BlockTask creates a runtime-declared task block and parks the active run when supplied.
func (m *Service) BlockTask(ctx context.Context, req BlockRequest, actor ActorContext) (TaskBlock, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return TaskBlock{}, err
	}
	block, runID, claimToken, err := m.taskBlockFromRequest(req, actor)
	if err != nil {
		return TaskBlock{}, err
	}
	if runID != "" || claimToken != "" {
		if runID == "" {
			return TaskBlock{}, fmt.Errorf("%w: task_block.run_id is required when claim_token is set", ErrValidation)
		}
		if claimToken == "" {
			return TaskBlock{}, fmt.Errorf("%w: task_block.claim_token is required when run_id is set", ErrValidation)
		}
		result, blockErr := m.store.BlockTaskAndReleaseRun(ctx, BlockTaskAndReleaseRunMutation{
			Block:           block,
			RunID:           runID,
			ClaimToken:      claimToken,
			Now:             block.CreatedAt,
			RecurrenceLimit: m.blockRecurrenceLimit,
			Actor:           actor,
		})
		if blockErr != nil {
			return TaskBlock{}, blockErr
		}
		reconciled, reconcileErr := m.reconcileTaskCascade(ctx, result.Block.TaskID, actor)
		if reconcileErr != nil {
			return TaskBlock{}, reconcileErr
		}
		m.recordTaskBlockCreated(ctx, result.Block, reconciled, actor, &result)
		m.recordReleasedRunEvent(ctx, &result, reconciled, actor)
		m.dispatchTaskBlocked(ctx, result.Block, reconciled, actor, &result)
		m.dispatchBlockedWake(ctx, reconciled, result.Block, actor, &result)
		m.recordTaskNeedsAttention(
			ctx,
			result.Block,
			result.EscalatedTask,
			reconciled,
			actor,
			&result,
		)
		m.dispatchTaskRunReleased(ctx, result.Run, reconciled, actor, result.PreviousRun, result.ReleaseReason)
		return result.Block, nil
	}

	if err := m.requireAgentSessionTaskLease(ctx, block.TaskID, actor); err != nil {
		return TaskBlock{}, err
	}

	created, err := m.store.CreateTaskBlock(ctx, CreateTaskBlockMutation{
		Block:           block,
		RecurrenceLimit: m.blockRecurrenceLimit,
		Actor:           actor,
	})
	if err != nil {
		return TaskBlock{}, err
	}
	reconciled, err := m.reconcileTaskCascade(ctx, created.Block.TaskID, actor)
	if err != nil {
		return TaskBlock{}, err
	}
	m.recordTaskBlockCreated(ctx, created.Block, reconciled, actor, nil)
	m.dispatchTaskBlocked(ctx, created.Block, reconciled, actor, nil)
	m.dispatchBlockedWake(ctx, reconciled, created.Block, actor, nil)
	m.recordTaskNeedsAttention(ctx, created.Block, created.EscalatedTask, reconciled, actor, nil)
	return created.Block, nil
}

func (m *Service) recordReleasedRunEvent(
	ctx context.Context,
	result *BlockTaskAndReleaseRunResult,
	reconciled Task,
	actor ActorContext,
) {
	if result == nil {
		return
	}
	if eventErr := m.recordTaskEvent(
		ctx,
		result.Run.TaskID,
		result.Run.ID,
		taskEventRunReleased,
		actor,
		releasedRunPayload{
			PreviousStatus: result.PreviousRun.Status,
			Status:         result.Run.Status,
			TaskStatus:     reconciled.Status,
			Reason:         result.ReleaseReason,
			SessionID:      result.PreviousRun.SessionID,
		},
	); eventErr != nil {
		slog.Error(
			"task: run release event failed after committed block-and-release",
			"error", eventErr,
			"event_type", taskEventRunReleased,
			"task_id", result.Run.TaskID,
			"run_id", result.Run.ID,
		)
	}
}

func (m *Service) recordTaskBlockCreated(
	ctx context.Context,
	block TaskBlock,
	reconciled Task,
	actor ActorContext,
	release *BlockTaskAndReleaseRunResult,
) {
	runID := ""
	claimTokenHash := ""
	if release != nil {
		runID = strings.TrimSpace(release.Run.ID)
		claimTokenHash = strings.TrimSpace(release.ClaimTokenHash)
	}
	if eventErr := m.recordTaskEvent(ctx, block.TaskID, runID, taskEventBlockCreated, actor, taskBlockCreatedPayload{
		Status:         reconciled.Status,
		BlockID:        strings.TrimSpace(block.ID),
		BlockKind:      block.Kind.Normalize(),
		Reason:         redactTaskSecretText(strings.TrimSpace(block.Reason)),
		ExpiresAt:      block.ExpiresAt,
		ClaimTokenHash: claimTokenHash,
	}); eventErr != nil {
		m.logTaskBlockEventFailure(eventErr, taskEventBlockCreated, block, runID)
	}
}

func (m *Service) recordTaskBlockCleared(
	ctx context.Context,
	block TaskBlock,
	reconciled Task,
	actor ActorContext,
) {
	if eventErr := m.recordTaskEvent(ctx, block.TaskID, "", taskEventBlockCleared, actor, taskBlockClearedPayload{
		Status:    reconciled.Status,
		BlockID:   strings.TrimSpace(block.ID),
		BlockKind: block.Kind.Normalize(),
		Reason:    redactTaskSecretText(strings.TrimSpace(block.Reason)),
		ClearNote: redactTaskSecretText(strings.TrimSpace(block.ClearNote)),
		ClearedAt: block.ClearedAt,
	}); eventErr != nil {
		m.logTaskBlockEventFailure(eventErr, taskEventBlockCleared, block, "")
	}
}

func (m *Service) recordTaskBlockExpired(
	ctx context.Context,
	block TaskBlock,
	reconciled Task,
	actor ActorContext,
) {
	if eventErr := m.recordTaskEvent(ctx, block.TaskID, "", taskEventBlockExpired, actor, taskBlockExpiredPayload{
		Status:    reconciled.Status,
		BlockID:   strings.TrimSpace(block.ID),
		BlockKind: block.Kind.Normalize(),
		Reason:    redactTaskSecretText(strings.TrimSpace(block.Reason)),
		ExpiresAt: block.ExpiresAt,
		ClearedAt: block.ClearedAt,
	}); eventErr != nil {
		m.logTaskBlockEventFailure(eventErr, taskEventBlockExpired, block, "")
	}
}

func (m *Service) logTaskBlockEventFailure(err error, eventType string, block TaskBlock, runID string) {
	slog.Error(
		"task: task block event failed after committed mutation",
		"error", err,
		"event_type", eventType,
		"task_id", block.TaskID,
		"run_id", strings.TrimSpace(runID),
		"block_id", block.ID,
	)
}

func (m *Service) requireAgentSessionTaskLease(ctx context.Context, taskID string, actor ActorContext) error {
	if actor.Actor.Kind.Normalize() != ActorKindAgentSession {
		return nil
	}
	sessionID := strings.TrimSpace(actor.Actor.Ref)
	normalizedTaskID := strings.TrimSpace(taskID)
	if sessionID == "" {
		return autonomyError(AutonomySessionRequired, ErrPermissionDenied, "agent session identity is required")
	}
	if normalizedTaskID == "" {
		return fmt.Errorf("%w: task.id is required", ErrValidation)
	}
	leaseStore, ok := m.store.(AutonomyLeaseStore)
	if !ok {
		return errors.New("task: autonomy lease lookup store is unavailable")
	}
	handles, err := leaseStore.ListAutonomyLeaseHandles(ctx, sessionID)
	if err != nil {
		return err
	}
	activeCount := 0
	matched := false
	now := m.now().UTC()
	for _, handle := range handles {
		normalized := normalizeAutonomyLeaseHandle(handle)
		if !isScopedTaskLeaseActive(normalized, sessionID, now) {
			continue
		}
		activeCount++
		if normalized.TaskID == normalizedTaskID {
			matched = true
		}
	}
	switch {
	case activeCount > 1:
		return autonomyError(
			AutonomyLeaseAlreadyHeld,
			ErrActiveRunLease,
			"session %q owns multiple active task-run leases",
			sessionID,
		)
	case matched:
		return nil
	case activeCount == 1:
		return autonomyError(
			AutonomyForeignRun,
			ErrPermissionDenied,
			"task %q is not leased by session %q",
			normalizedTaskID,
			sessionID,
		)
	default:
		return autonomyError(
			AutonomyNoActiveLease,
			ErrInvalidClaimToken,
			"session %q has no active task-run lease",
			sessionID,
		)
	}
}

func isScopedTaskLeaseActive(handle AutonomyLeaseHandle, sessionID string, now time.Time) bool {
	return handle.SessionID == sessionID &&
		isAutonomyLeaseStatusActive(handle.Status) &&
		!handle.LeaseUntil.IsZero() &&
		handle.LeaseUntil.After(now) &&
		handle.ClaimTokenHash != ""
}

// ClearTaskBlock clears one open task block through the service-owned transition.
func (m *Service) ClearTaskBlock(
	ctx context.Context,
	taskID string,
	blockID string,
	note string,
	actor ActorContext,
) (TaskBlock, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return TaskBlock{}, err
	}
	normalizedTaskID := strings.TrimSpace(taskID)
	normalizedBlockID := strings.TrimSpace(blockID)
	if normalizedTaskID == "" {
		return TaskBlock{}, fmt.Errorf("%w: task_block.task_id is required", ErrValidation)
	}
	if normalizedBlockID == "" {
		return TaskBlock{}, fmt.Errorf("%w: task_block.id is required", ErrValidation)
	}
	normalizedNote := strings.TrimSpace(note)
	if err := rejectTaskSecretText("task_block.clear_note", normalizedNote); err != nil {
		return TaskBlock{}, err
	}
	if err := m.requireAgentSessionTaskLease(ctx, normalizedTaskID, actor); err != nil {
		return TaskBlock{}, err
	}

	cleared, err := m.store.ClearTaskBlock(ctx, ClearTaskBlockMutation{
		TaskID:    normalizedTaskID,
		BlockID:   normalizedBlockID,
		ClearedBy: actor.Actor,
		ClearedAt: m.now().UTC(),
		ClearNote: normalizedNote,
		Actor:     actor,
	})
	if err != nil {
		return TaskBlock{}, err
	}
	reconciled, err := m.reconcileTaskCascade(ctx, cleared.TaskID, actor)
	if err != nil {
		return TaskBlock{}, err
	}
	m.recordTaskBlockCleared(ctx, cleared, reconciled, actor)
	m.dispatchTaskUnblocked(ctx, cleared, reconciled, actor)
	m.autoEnqueueReadyTaskDetached(ctx, cleared.TaskID, autoEnqueueTrigger{
		Kind: autoEnqueueTriggerBlockClear,
		Ref:  cleared.ID,
	}, actor)
	return cleared, nil
}

// RecoverTask clears task-level needs_attention escalation through the service-owned transition.
func (m *Service) RecoverTask(ctx context.Context, id string, note string, actor ActorContext) (*Task, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return nil, err
	}
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: task.id is required", ErrValidation)
	}
	if actor.Actor.Kind.Normalize() == ActorKindAgentSession {
		return nil, autonomyError(
			AutonomyForeignRun,
			ErrPermissionDenied,
			"agent session %q cannot recover task %q without operator scope",
			strings.TrimSpace(actor.Actor.Ref),
			trimmedID,
		)
	}
	normalizedNote := strings.TrimSpace(note)
	if err := rejectTaskSecretText("task.recover_note", normalizedNote); err != nil {
		return nil, err
	}
	recoveredAt := m.now().UTC()
	cleared, err := m.store.ClearTaskNeedsAttention(ctx, NeedsAttentionClearMutation{
		TaskID:    trimmedID,
		ClearedBy: actor.Actor,
		ClearedAt: recoveredAt,
		Origin:    actor.Origin,
	})
	if err != nil {
		return nil, err
	}
	reconciled, err := m.reconcileTaskCascade(ctx, cleared.ID, actor)
	if err != nil {
		return nil, err
	}
	m.dispatchTaskRecovered(ctx, reconciled, actor, normalizedNote, recoveredAt)
	m.autoEnqueueReadyTaskDetached(ctx, reconciled.ID, autoEnqueueTrigger{
		Kind: autoEnqueueTriggerRecover,
		Ref:  reconciled.ID,
	}, actor)
	return &reconciled, nil
}

// ExpireTaskBlocks finalizes expired transient blocks through service-owned transitions.
func (m *Service) ExpireTaskBlocks(
	ctx context.Context,
	now time.Time,
	actor ActorContext,
) (ExpireTaskBlocksResult, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return ExpireTaskBlocksResult{}, err
	}
	expireAt := now.UTC()
	if expireAt.IsZero() {
		expireAt = m.now().UTC()
	}
	result, err := m.store.ExpireTaskBlocks(ctx, ExpireTaskBlocksMutation{
		Now:       expireAt,
		ClearedBy: actor.Actor,
	})
	if err != nil {
		return ExpireTaskBlocksResult{}, err
	}
	blocksByTask := make(map[string][]TaskBlock)
	taskIDs := make([]string, 0)
	for _, block := range result.Blocks {
		if _, ok := blocksByTask[block.TaskID]; !ok {
			taskIDs = append(taskIDs, block.TaskID)
		}
		blocksByTask[block.TaskID] = append(blocksByTask[block.TaskID], block)
	}
	for _, taskID := range taskIDs {
		blocks := blocksByTask[taskID]
		reconciled, reconcileErr := m.reconcileTaskCascade(ctx, taskID, actor)
		if reconcileErr != nil {
			return ExpireTaskBlocksResult{}, reconcileErr
		}
		for _, block := range blocks {
			m.recordTaskBlockExpired(ctx, block, reconciled, actor)
			m.dispatchTaskUnblocked(ctx, block, reconciled, actor)
		}
		m.autoEnqueueReadyTaskDetached(ctx, taskID, autoEnqueueTrigger{
			Kind: autoEnqueueTriggerTransientExpiry,
			Ref:  transientExpiryTriggerRef(blocks),
		}, actor)
	}
	return result, nil
}

func transientExpiryTriggerRef(blocks []TaskBlock) string {
	if len(blocks) == 1 {
		return blocks[0].ID
	}
	ids := make([]string, 0, len(blocks))
	for _, block := range blocks {
		ids = append(ids, block.ID)
	}
	sort.Strings(ids)
	return strings.Join(ids, ",")
}

// ListTaskBlocks returns task block rows after enforcing read authority and task existence.
func (m *Service) ListTaskBlocks(
	ctx context.Context,
	taskID string,
	includeCleared bool,
	actor ActorContext,
) ([]TaskBlock, error) {
	if err := requireReadAuthority(actor); err != nil {
		return nil, err
	}
	normalizedTaskID := strings.TrimSpace(taskID)
	if normalizedTaskID == "" {
		return nil, fmt.Errorf("%w: task_block.task_id is required", ErrValidation)
	}
	if err := m.requireAgentSessionTaskLease(ctx, normalizedTaskID, actor); err != nil {
		return nil, err
	}
	if _, err := m.store.GetTask(ctx, normalizedTaskID); err != nil {
		return nil, err
	}
	return m.store.ListTaskBlocks(ctx, normalizedTaskID, includeCleared)
}

func (m *Service) taskBlockFromRequest(req BlockRequest, actor ActorContext) (TaskBlock, string, string, error) {
	now := m.now().UTC()
	taskID := strings.TrimSpace(req.TaskID)
	if taskID == "" {
		return TaskBlock{}, "", "", fmt.Errorf("%w: task_block.task_id is required", ErrValidation)
	}
	kind := req.Kind.Normalize()
	if err := kind.Validate("task_block.kind"); err != nil {
		return TaskBlock{}, "", "", err
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return TaskBlock{}, "", "", fmt.Errorf("%w: task_block.reason is required", ErrValidation)
	}
	if err := rejectTaskSecretText("task_block.reason", reason); err != nil {
		return TaskBlock{}, "", "", err
	}
	details := cloneRawJSON(req.Details)
	if err := rejectTaskSecretJSON("task_block.details", details); err != nil {
		return TaskBlock{}, "", "", err
	}
	expiresAt := req.ExpiresAt
	if !expiresAt.IsZero() {
		expiresAt = expiresAt.UTC()
		if kind != BlockKindTransient {
			return TaskBlock{}, "", "", fmt.Errorf(
				"%w: task_block.expires_at is only valid for %q blocks",
				ErrValidation,
				BlockKindTransient,
			)
		}
	}
	runID := strings.TrimSpace(req.RunID)
	claimToken := strings.TrimSpace(req.ClaimToken)

	return TaskBlock{
		ID:        m.newID(taskBlockIDPrefix),
		TaskID:    taskID,
		Kind:      kind,
		Reason:    reason,
		Details:   details,
		CreatedBy: ActorIdentity{Kind: actor.Actor.Kind.Normalize(), Ref: strings.TrimSpace(actor.Actor.Ref)},
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}, runID, claimToken, nil
}

func rejectTaskSecretText(path string, value string) error {
	if redactTaskSecretText(value) != value {
		return fmt.Errorf("%w: %s must not embed raw secret material", ErrValidation, path)
	}
	return nil
}

func rejectTaskSecretJSON(path string, value []byte) error {
	if len(value) == 0 {
		return nil
	}
	decoded, ok := decodeTaskSecretJSON(value)
	if !ok {
		if redactTaskSecretText(string(value)) != string(value) {
			return fmt.Errorf("%w: %s must not embed raw secret material", ErrValidation, path)
		}
		return nil
	}
	if taskSecretValueContainsSecret(decoded) {
		return fmt.Errorf("%w: %s must not embed raw secret material", ErrValidation, path)
	}
	return nil
}

func redactTaskSecretText(value string) string { return diagnostics.Redact(RedactClaimTokens(value)) }

func redactTaskSecretJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	decoded, ok := decodeTaskSecretJSON(raw)
	if !ok {
		return json.RawMessage(redactTaskSecretText(string(raw)))
	}
	encoded, err := json.Marshal(redactTaskSecretValue(decoded))
	if err != nil {
		return json.RawMessage(redactTaskSecretText(string(raw)))
	}
	return json.RawMessage(encoded)
}

func decodeTaskSecretJSON(raw []byte) (any, bool) {
	var decoded any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&decoded); err != nil {
		return nil, false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, false
	}
	return decoded, true
}

func taskSecretValueContainsSecret(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			if taskSecretKeyCarriesSecret(key) || taskSecretValueContainsSecret(nested) {
				return true
			}
		}
	case []any:
		return slices.ContainsFunc(typed, taskSecretValueContainsSecret)
	case string:
		return redactTaskSecretText(typed) != typed
	}
	return false
}

func redactTaskSecretValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		redacted := make(map[string]any, len(typed))
		for key, nested := range typed {
			if taskSecretKeyCarriesSecret(key) {
				redacted[key] = "[REDACTED]"
				continue
			}
			redacted[key] = redactTaskSecretValue(nested)
		}
		return redacted
	case []any:
		redacted := make([]any, 0, len(typed))
		for _, nested := range typed {
			redacted = append(redacted, redactTaskSecretValue(nested))
		}
		return redacted
	case string:
		return redactTaskSecretText(typed)
	default:
		return typed
	}
}

func taskSecretKeyCarriesSecret(key string) bool {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return false
	}
	probe := trimmed + "=value"
	return diagnostics.Redact(probe) != probe
}

func (m *Service) recordTaskNeedsAttention(
	ctx context.Context,
	block TaskBlock,
	escalatedTask *Task,
	reconciled Task,
	actor ActorContext,
	release *BlockTaskAndReleaseRunResult,
) {
	if escalatedTask == nil {
		return
	}
	needsAttention := reconciled.NeedsAttention
	if needsAttention == nil {
		needsAttention = escalatedTask.NeedsAttention
	}
	if needsAttention == nil {
		slog.Error(
			"task: recurrence escalation missing needs_attention metadata",
			"task_id", reconciled.ID,
			"block_id", block.ID,
		)
		return
	}
	m.dispatchNeedsAttentionWake(ctx, reconciled, block, needsAttention.Reason, actor, release)
	m.dispatchTaskNeedsAttention(ctx, reconciled, actor, needsAttention.Reason, needsAttention.At, release)
}

func (m *Service) dispatchTaskBlocked(
	ctx context.Context,
	block TaskBlock,
	taskRecord Task,
	actor ActorContext,
	release *BlockTaskAndReleaseRunResult,
) {
	payload := hookspkg.TaskBlockedPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookTaskBlocked,
			Timestamp: m.now().UTC(),
		},
		TaskContext: m.taskHookContext(taskRecord, actor, release),
		BlockID:     strings.TrimSpace(block.ID),
		Kind:        string(block.Kind.Normalize()),
		Reason:      redactTaskSecretText(strings.TrimSpace(block.Reason)),
		Details:     redactTaskSecretJSON(cloneRawJSON(block.Details)),
	}
	_, err := m.taskHooks.DispatchTaskBlocked(taskRunObservationHookContext(ctx), payload)
	m.reportTaskHookFailure(hookspkg.HookTaskBlocked, err, taskRecord)
}

func (m *Service) dispatchTaskUnblocked(
	ctx context.Context,
	block TaskBlock,
	taskRecord Task,
	actor ActorContext,
) {
	payload := hookspkg.TaskUnblockedPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookTaskUnblocked,
			Timestamp: m.now().UTC(),
		},
		TaskContext: m.taskHookContext(taskRecord, actor, nil),
		BlockID:     strings.TrimSpace(block.ID),
		Kind:        string(block.Kind.Normalize()),
		Reason:      redactTaskSecretText(strings.TrimSpace(block.Reason)),
		Details:     redactTaskSecretJSON(cloneRawJSON(block.Details)),
		ClearedAt:   block.ClearedAt,
		ClearNote:   redactTaskSecretText(strings.TrimSpace(block.ClearNote)),
	}
	_, err := m.taskHooks.DispatchTaskUnblocked(taskRunObservationHookContext(ctx), payload)
	m.reportTaskHookFailure(hookspkg.HookTaskUnblocked, err, taskRecord)
}

func (m *Service) dispatchTaskNeedsAttention(
	ctx context.Context,
	taskRecord Task,
	actor ActorContext,
	reason string,
	at time.Time,
	release *BlockTaskAndReleaseRunResult,
) {
	payload := hookspkg.TaskNeedsAttentionPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookTaskNeedsAttention,
			Timestamp: m.now().UTC(),
		},
		TaskContext: m.taskHookContext(taskRecord, actor, release),
		Reason:      redactTaskSecretText(strings.TrimSpace(reason)),
		At:          at,
	}
	_, err := m.taskHooks.DispatchTaskNeedsAttention(taskRunObservationHookContext(ctx), payload)
	m.reportTaskHookFailure(hookspkg.HookTaskNeedsAttention, err, taskRecord)
}

func (m *Service) dispatchTaskRecovered(
	ctx context.Context,
	taskRecord Task,
	actor ActorContext,
	note string,
	at time.Time,
) {
	payload := hookspkg.TaskRecoveredPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookTaskRecovered,
			Timestamp: m.now().UTC(),
		},
		TaskContext: m.taskHookContext(taskRecord, actor, nil),
		Note:        redactTaskSecretText(strings.TrimSpace(note)),
		At:          at,
	}
	_, err := m.taskHooks.DispatchTaskRecovered(taskRunObservationHookContext(ctx), payload)
	m.reportTaskHookFailure(hookspkg.HookTaskRecovered, err, taskRecord)
}

func (m *Service) taskHookContext(
	taskRecord Task,
	actor ActorContext,
	release *BlockTaskAndReleaseRunResult,
) hookspkg.TaskContext {
	contextPayload := hookspkg.TaskContext{
		TaskID:                strings.TrimSpace(taskRecord.ID),
		ParentTaskID:          strings.TrimSpace(taskRecord.ParentTaskID),
		WorkspaceID:           strings.TrimSpace(taskRecord.WorkspaceID),
		WorkflowID:            taskRunMetadataString(taskRecord.Metadata, "workflow_id"),
		CoordinationChannelID: taskRunMetadataString(taskRecord.Metadata, "coordination_channel_id"),
		NetworkChannel:        strings.TrimSpace(taskRecord.NetworkChannel),
		AgentName:             taskHookAgentName(taskRecord, actor),
		ActorKind:             string(actor.Actor.Kind.Normalize()),
		ActorID:               strings.TrimSpace(actor.Actor.Ref),
		OriginKind:            string(actor.Origin.Kind.Normalize()),
		OriginRef:             strings.TrimSpace(actor.Origin.Ref),
		TaskStatus:            string(taskRecord.Status.Normalize()),
	}
	if release != nil {
		contextPayload.RunID = strings.TrimSpace(release.Run.ID)
		contextPayload.ReleaseReason = strings.TrimSpace(release.ReleaseReason)
		contextPayload.ClaimTokenHash = strings.TrimSpace(release.ClaimTokenHash)
	}
	return contextPayload
}

func taskHookAgentName(taskRecord Task, actor ActorContext) string {
	if value := taskRunMetadataString(taskRecord.Metadata, "agent_name"); value != "" {
		return value
	}
	if actor.Actor.Kind.Normalize() == ActorKindAgentSession {
		return strings.TrimSpace(actor.Actor.Ref)
	}
	return ""
}

func (m *Service) reportTaskHookFailure(event hookspkg.HookEvent, err error, taskRecord Task) {
	if err == nil {
		return
	}
	slog.Error(
		"task: task lifecycle hook failed after committed mutation",
		"error", err,
		"hook_event", event,
		"task_id", taskRecord.ID,
		"task_status", taskRecord.Status,
	)
}

type taskBlockCreatedPayload struct {
	Status         Status    `json:"status"`
	BlockID        string    `json:"block_id"`
	BlockKind      BlockKind `json:"block_kind"`
	Reason         string    `json:"reason"`
	ExpiresAt      time.Time `json:"expires_at,omitzero"`
	ClaimTokenHash string    `json:"claim_token_hash,omitempty"`
}

type taskBlockClearedPayload struct {
	Status    Status    `json:"status"`
	BlockID   string    `json:"block_id"`
	BlockKind BlockKind `json:"block_kind"`
	Reason    string    `json:"reason"`
	ClearNote string    `json:"clear_note,omitempty"`
	ClearedAt time.Time `json:"cleared_at"`
}

type taskBlockExpiredPayload struct {
	Status    Status    `json:"status"`
	BlockID   string    `json:"block_id"`
	BlockKind BlockKind `json:"block_kind"`
	Reason    string    `json:"reason"`
	ExpiresAt time.Time `json:"expires_at"`
	ClearedAt time.Time `json:"cleared_at"`
}
