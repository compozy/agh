package daemon

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/session"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

const (
	harnessDetachedMetadataSchema  = "agh.harness.detached.v1"
	harnessDetachedTaskMetadataKey = "harness_detached_task"
	harnessDetachedRunMetadataKey  = "harness_detached_run"
	harnessDetachedActorRefPrefix  = "harness-detached"
	harnessDetachedOriginRefPrefix = "daemon.harness.detached"
	defaultDetachedHarnessSummary  = "Detached harness work"
)

type detachedHarnessWakeTargetInput struct {
	SessionID string
}

type detachedHarnessWakeTarget struct {
	SessionID   string `json:"session_id"`
	SessionType string `json:"session_type,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	Channel     string `json:"channel,omitempty"`
}

type detachedHarnessTaskMetadata struct {
	Schema               string                    `json:"schema"`
	Kind                 string                    `json:"kind"`
	SubmissionKey        string                    `json:"submission_key"`
	Summary              string                    `json:"summary,omitempty"`
	SubmissionTurnSource string                    `json:"submission_turn_source,omitempty"`
	OwnerSessionID       string                    `json:"owner_session_id"`
	OwnerSessionType     string                    `json:"owner_session_type,omitempty"`
	OwnerWorkspaceID     string                    `json:"owner_workspace_id,omitempty"`
	OwnerChannel         string                    `json:"owner_channel,omitempty"`
	WakeTarget           detachedHarnessWakeTarget `json:"wake_target"`
}

type detachedHarnessRunMetadata struct {
	Schema               string                    `json:"schema"`
	Kind                 string                    `json:"kind"`
	SubmissionKey        string                    `json:"submission_key"`
	Summary              string                    `json:"summary,omitempty"`
	SubmissionTurnSource string                    `json:"submission_turn_source,omitempty"`
	OwnerSessionID       string                    `json:"owner_session_id"`
	OwnerSessionType     string                    `json:"owner_session_type,omitempty"`
	OwnerWorkspaceID     string                    `json:"owner_workspace_id,omitempty"`
	OwnerChannel         string                    `json:"owner_channel,omitempty"`
	WakeTarget           detachedHarnessWakeTarget `json:"wake_target"`
	Reentry              *detachedHarnessReentry   `json:"reentry,omitempty"`
}

type detachedHarnessReentry struct {
	Outcome     string    `json:"outcome,omitempty"`
	Reason      string    `json:"reason,omitempty"`
	ProcessedAt time.Time `json:"processed_at"`
}

type detachedHarnessSubmitRequest struct {
	SubmissionKey  string
	OwnerSessionID string
	Scope          taskpkg.Scope
	WorkspaceID    string
	Summary        string
	Description    string
	NetworkChannel string
	TurnSource     session.TurnSource
	WakeTarget     detachedHarnessWakeTargetInput
}

type detachedHarnessSubmission struct {
	Task         taskpkg.Task
	Run          taskpkg.Run
	ExistingTask bool
	ExistingRun  bool
}

type harnessDetachedSessionReader interface {
	Status(ctx context.Context, id string) (*session.Info, error)
}

type harnessDetachedWorkBridge struct {
	tasks    taskpkg.Manager
	store    taskStore
	sessions harnessDetachedSessionReader
}

type normalizedDetachedHarnessSubmitRequest struct {
	TaskID           string
	SubmissionKey    string
	Scope            taskpkg.Scope
	WorkspaceID      string
	Summary          string
	Description      string
	NetworkChannel   string
	TurnSource       session.TurnSource
	OwnerSessionID   string
	OwnerSessionType string
	OwnerWorkspaceID string
	OwnerChannel     string
	WakeTarget       detachedHarnessWakeTarget
}

func newHarnessDetachedWorkBridge(
	tasks taskpkg.Manager,
	store taskStore,
	sessions harnessDetachedSessionReader,
) (*harnessDetachedWorkBridge, error) {
	if tasks == nil {
		return nil, errors.New("daemon: harness detached work bridge requires task manager")
	}
	if store == nil {
		return nil, errors.New("daemon: harness detached work bridge requires task store")
	}
	if sessions == nil {
		return nil, errors.New("daemon: harness detached work bridge requires session reader")
	}
	return &harnessDetachedWorkBridge{
		tasks:    tasks,
		store:    store,
		sessions: sessions,
	}, nil
}

func (b *harnessDetachedWorkBridge) submit(
	ctx context.Context,
	req detachedHarnessSubmitRequest,
) (*detachedHarnessSubmission, error) {
	if ctx == nil {
		return nil, errors.New("daemon: detached harness submit context is required")
	}
	if b == nil {
		return nil, errors.New("daemon: detached harness work bridge is required")
	}

	normalized, err := b.normalizeSubmitRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	actor, err := detachedHarnessActorContext(normalized.OwnerSessionID)
	if err != nil {
		return nil, err
	}

	taskMetadata := buildDetachedHarnessTaskMetadata(normalized)
	runMetadata := buildDetachedHarnessRunMetadata(normalized)
	taskMetadataJSON, err := marshalDetachedHarnessMetadata(taskMetadata)
	if err != nil {
		return nil, err
	}
	runMetadataJSON, err := marshalDetachedHarnessMetadata(runMetadata)
	if err != nil {
		return nil, err
	}

	existingRun, existingRunFound, err := b.lookupExistingRun(ctx, normalized, actor.Origin, runMetadata)
	if err != nil {
		return nil, err
	}

	taskRecord, existingTask, err := b.ensureTask(ctx, normalized, actor, taskMetadata, taskMetadataJSON)
	if err != nil {
		return nil, err
	}
	if existingRunFound && strings.TrimSpace(existingRun.TaskID) != taskRecord.ID {
		return nil, fmt.Errorf(
			"%w: detached harness submission %q is already bound to task %q",
			taskpkg.ErrValidation,
			normalized.SubmissionKey,
			existingRun.TaskID,
		)
	}

	run, err := b.tasks.EnqueueRun(ctx, taskpkg.EnqueueRun{
		TaskID:         taskRecord.ID,
		IdempotencyKey: normalized.SubmissionKey,
		NetworkChannel: normalized.NetworkChannel,
		Metadata:       runMetadataJSON,
	}, actor)
	if err != nil {
		return nil, err
	}
	if run == nil || strings.TrimSpace(run.ID) == "" {
		return nil, errors.New("daemon: detached harness submit returned empty task run")
	}

	return &detachedHarnessSubmission{
		Task:         taskRecord,
		Run:          *run,
		ExistingTask: existingTask,
		ExistingRun:  existingRunFound,
	}, nil
}

func (b *harnessDetachedWorkBridge) normalizeSubmitRequest(
	ctx context.Context,
	req detachedHarnessSubmitRequest,
) (normalizedDetachedHarnessSubmitRequest, error) {
	submissionKey := strings.TrimSpace(req.SubmissionKey)
	if submissionKey == "" {
		return normalizedDetachedHarnessSubmitRequest{}, fmt.Errorf(
			"%w: detached harness submission key is required",
			taskpkg.ErrValidation,
		)
	}

	ownerInfo, err := b.lookupDetachedHarnessSession(
		ctx,
		strings.TrimSpace(req.OwnerSessionID),
		"owner_session_id",
	)
	if err != nil {
		return normalizedDetachedHarnessSubmitRequest{}, err
	}
	wakeInfo, err := b.lookupDetachedHarnessSession(
		ctx,
		strings.TrimSpace(req.WakeTarget.SessionID),
		"wake_target.session_id",
	)
	if err != nil {
		return normalizedDetachedHarnessSubmitRequest{}, err
	}

	scope := req.Scope.Normalize()
	workspaceID, err := normalizeDetachedHarnessWorkspace(
		scope,
		strings.TrimSpace(req.WorkspaceID),
		ownerInfo,
		wakeInfo,
	)
	if err != nil {
		return normalizedDetachedHarnessSubmitRequest{}, err
	}

	return normalizedDetachedHarnessSubmitRequest{
		TaskID:           detachedHarnessTaskID(ownerInfo.ID, submissionKey),
		SubmissionKey:    submissionKey,
		Scope:            scope,
		WorkspaceID:      workspaceID,
		Summary:          detachedHarnessSummary(req.Summary),
		Description:      strings.TrimSpace(req.Description),
		NetworkChannel:   detachedHarnessChannel(req.NetworkChannel, ownerInfo.Channel),
		TurnSource:       normalizeDetachedHarnessTurnSource(req.TurnSource),
		OwnerSessionID:   strings.TrimSpace(ownerInfo.ID),
		OwnerSessionType: string(ownerInfo.Type),
		OwnerWorkspaceID: strings.TrimSpace(ownerInfo.WorkspaceID),
		OwnerChannel:     strings.TrimSpace(ownerInfo.Channel),
		WakeTarget: detachedHarnessWakeTarget{
			SessionID:   strings.TrimSpace(wakeInfo.ID),
			SessionType: string(wakeInfo.Type),
			WorkspaceID: strings.TrimSpace(wakeInfo.WorkspaceID),
			Channel:     strings.TrimSpace(wakeInfo.Channel),
		},
	}, nil
}

func (b *harnessDetachedWorkBridge) lookupDetachedHarnessSession(
	ctx context.Context,
	sessionID string,
	field string,
) (*session.Info, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("%w: detached harness %s is required", taskpkg.ErrValidation, field)
	}

	info, err := b.sessions.Status(ctx, sessionID)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			return nil, fmt.Errorf("%w: detached harness %s %q was not found", taskpkg.ErrValidation, field, sessionID)
		}
		return nil, err
	}
	if info == nil {
		return nil, fmt.Errorf("%w: detached harness %s %q was not found", taskpkg.ErrValidation, field, sessionID)
	}
	return info, nil
}

func normalizeDetachedHarnessWorkspace(
	scope taskpkg.Scope,
	workspaceID string,
	ownerInfo *session.Info,
	wakeInfo *session.Info,
) (string, error) {
	switch scope {
	case taskpkg.ScopeWorkspace:
		wakeWorkspaceID := strings.TrimSpace(wakeInfo.WorkspaceID)
		ownerWorkspaceID := strings.TrimSpace(ownerInfo.WorkspaceID)
		if wakeWorkspaceID == "" {
			wakeWorkspaceID = ownerWorkspaceID
		}
		if wakeWorkspaceID == "" {
			return "", fmt.Errorf(
				"%w: detached harness workspace scope requires a workspace-bound wake target",
				taskpkg.ErrValidation,
			)
		}
		if ownerWorkspaceID != "" && ownerWorkspaceID != wakeWorkspaceID {
			return "", fmt.Errorf(
				"%w: detached harness owner session %q and wake target %q must share one workspace",
				taskpkg.ErrValidation,
				strings.TrimSpace(ownerInfo.ID),
				strings.TrimSpace(wakeInfo.ID),
			)
		}
		if workspaceID != "" && workspaceID != wakeWorkspaceID {
			return "", fmt.Errorf(
				"%w: detached harness workspace %q does not match wake target workspace %q",
				taskpkg.ErrValidation,
				workspaceID,
				wakeWorkspaceID,
			)
		}
		return wakeWorkspaceID, nil
	case taskpkg.ScopeGlobal:
		if workspaceID != "" {
			return "", fmt.Errorf(
				"%w: detached harness global scope cannot include workspace_id",
				taskpkg.ErrValidation,
			)
		}
		return "", nil
	default:
		return "", fmt.Errorf("%w: unsupported detached harness scope %q", taskpkg.ErrValidation, scope)
	}
}

func (b *harnessDetachedWorkBridge) lookupExistingRun(
	ctx context.Context,
	req normalizedDetachedHarnessSubmitRequest,
	origin taskpkg.Origin,
	expected detachedHarnessRunMetadata,
) (taskpkg.Run, bool, error) {
	run, err := b.store.GetTaskRunByIdempotencyKey(ctx, req.SubmissionKey, origin)
	switch {
	case errors.Is(err, taskpkg.ErrTaskRunIdempotencyNotFound):
		return taskpkg.Run{}, false, nil
	case err != nil:
		return taskpkg.Run{}, false, err
	}
	if err := validateDetachedHarnessRunMatch(run, req, origin, expected); err != nil {
		return taskpkg.Run{}, false, err
	}
	return run, true, nil
}

func (b *harnessDetachedWorkBridge) ensureTask(
	ctx context.Context,
	req normalizedDetachedHarnessSubmitRequest,
	actor taskpkg.ActorContext,
	expected detachedHarnessTaskMetadata,
	expectedJSON json.RawMessage,
) (taskpkg.Task, bool, error) {
	if current, err := b.store.GetTask(ctx, req.TaskID); err == nil {
		if err := validateDetachedHarnessTaskMatch(current, req, actor, expected); err != nil {
			return taskpkg.Task{}, false, err
		}
		return current, true, nil
	} else if !errors.Is(err, taskpkg.ErrTaskNotFound) {
		return taskpkg.Task{}, false, err
	}

	created, err := b.tasks.CreateTask(ctx, taskpkg.CreateTask{
		ID:             req.TaskID,
		Scope:          req.Scope,
		WorkspaceID:    req.WorkspaceID,
		NetworkChannel: req.NetworkChannel,
		Title:          req.Summary,
		Description:    req.Description,
		Owner: &taskpkg.Ownership{
			Kind: taskpkg.OwnerKindAgentSession,
			Ref:  req.OwnerSessionID,
		},
		Metadata: expectedJSON,
	}, actor)
	if err == nil {
		if created == nil {
			return taskpkg.Task{}, false, errors.New("daemon: detached harness submit returned empty task")
		}
		return *created, false, nil
	}

	current, getErr := b.store.GetTask(ctx, req.TaskID)
	if getErr != nil {
		return taskpkg.Task{}, false, err
	}
	if err := validateDetachedHarnessTaskMatch(current, req, actor, expected); err != nil {
		return taskpkg.Task{}, false, err
	}
	return current, true, nil
}

func buildDetachedHarnessTaskMetadata(req normalizedDetachedHarnessSubmitRequest) detachedHarnessTaskMetadata {
	return detachedHarnessTaskMetadata{
		Schema:               harnessDetachedMetadataSchema,
		Kind:                 harnessDetachedTaskMetadataKey,
		SubmissionKey:        req.SubmissionKey,
		Summary:              req.Summary,
		SubmissionTurnSource: string(req.TurnSource),
		OwnerSessionID:       req.OwnerSessionID,
		OwnerSessionType:     req.OwnerSessionType,
		OwnerWorkspaceID:     req.OwnerWorkspaceID,
		OwnerChannel:         req.OwnerChannel,
		WakeTarget:           req.WakeTarget,
	}
}

func buildDetachedHarnessRunMetadata(req normalizedDetachedHarnessSubmitRequest) detachedHarnessRunMetadata {
	return detachedHarnessRunMetadata{
		Schema:               harnessDetachedMetadataSchema,
		Kind:                 harnessDetachedRunMetadataKey,
		SubmissionKey:        req.SubmissionKey,
		Summary:              req.Summary,
		SubmissionTurnSource: string(req.TurnSource),
		OwnerSessionID:       req.OwnerSessionID,
		OwnerSessionType:     req.OwnerSessionType,
		OwnerWorkspaceID:     req.OwnerWorkspaceID,
		OwnerChannel:         req.OwnerChannel,
		WakeTarget:           req.WakeTarget,
		Reentry:              nil,
	}
}

func marshalDetachedHarnessMetadata(value any) (json.RawMessage, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("daemon: marshal detached harness metadata: %w", err)
	}
	return payload, nil
}

func validateDetachedHarnessTaskMatch(
	record taskpkg.Task,
	req normalizedDetachedHarnessSubmitRequest,
	actor taskpkg.ActorContext,
	expected detachedHarnessTaskMetadata,
) error {
	if record.Scope != req.Scope ||
		record.WorkspaceID != req.WorkspaceID ||
		record.NetworkChannel != req.NetworkChannel ||
		record.Title != req.Summary ||
		record.Description != req.Description ||
		record.CreatedBy != actor.Actor ||
		record.Origin != actor.Origin {
		return fmt.Errorf(
			"%w: detached harness submission %q is already bound to a different task payload",
			taskpkg.ErrValidation,
			req.SubmissionKey,
		)
	}
	switch {
	case record.Owner == nil:
		return fmt.Errorf(
			"%w: detached harness task %q is missing owner session metadata",
			taskpkg.ErrValidation,
			record.ID,
		)
	case record.Owner.Kind != taskpkg.OwnerKindAgentSession || record.Owner.Ref != req.OwnerSessionID:
		return fmt.Errorf(
			"%w: detached harness task %q is already bound to owner %q/%q",
			taskpkg.ErrValidation,
			record.ID,
			record.Owner.Kind,
			record.Owner.Ref,
		)
	}

	current, err := decodeDetachedHarnessTaskMetadata(record.Metadata)
	if err != nil {
		return err
	}
	if current != expected {
		return fmt.Errorf(
			"%w: detached harness task %q metadata does not match submission %q",
			taskpkg.ErrValidation,
			record.ID,
			req.SubmissionKey,
		)
	}
	return nil
}

func validateDetachedHarnessRunMatch(
	run taskpkg.Run,
	req normalizedDetachedHarnessSubmitRequest,
	origin taskpkg.Origin,
	expected detachedHarnessRunMetadata,
) error {
	if run.Origin != origin {
		return fmt.Errorf(
			"%w: detached harness run %q origin %q/%q does not match submission origin %q/%q",
			taskpkg.ErrValidation,
			run.ID,
			run.Origin.Kind,
			run.Origin.Ref,
			origin.Kind,
			origin.Ref,
		)
	}
	current, err := decodeDetachedHarnessRunMetadata(run.Metadata)
	if err != nil {
		return err
	}
	if current != expected {
		return fmt.Errorf(
			"%w: detached harness run %q metadata does not match submission %q",
			taskpkg.ErrValidation,
			run.ID,
			req.SubmissionKey,
		)
	}
	return nil
}

func decodeDetachedHarnessTaskMetadata(raw json.RawMessage) (detachedHarnessTaskMetadata, error) {
	var metadata detachedHarnessTaskMetadata
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return detachedHarnessTaskMetadata{}, fmt.Errorf(
			"%w: decode detached harness task metadata: %v",
			taskpkg.ErrValidation,
			err,
		)
	}
	if metadata.Schema != harnessDetachedMetadataSchema || metadata.Kind != harnessDetachedTaskMetadataKey {
		return detachedHarnessTaskMetadata{}, fmt.Errorf(
			"%w: detached harness task metadata has unsupported schema %q/%q",
			taskpkg.ErrValidation,
			metadata.Schema,
			metadata.Kind,
		)
	}
	return metadata, nil
}

func decodeDetachedHarnessRunMetadata(raw json.RawMessage) (detachedHarnessRunMetadata, error) {
	var metadata detachedHarnessRunMetadata
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return detachedHarnessRunMetadata{}, fmt.Errorf(
			"%w: decode detached harness run metadata: %v",
			taskpkg.ErrValidation,
			err,
		)
	}
	if metadata.Schema != harnessDetachedMetadataSchema || metadata.Kind != harnessDetachedRunMetadataKey {
		return detachedHarnessRunMetadata{}, fmt.Errorf(
			"%w: detached harness run metadata has unsupported schema %q/%q",
			taskpkg.ErrValidation,
			metadata.Schema,
			metadata.Kind,
		)
	}
	return metadata, nil
}

func maybeDecodeDetachedHarnessRunMetadata(raw json.RawMessage) (detachedHarnessRunMetadata, bool, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return detachedHarnessRunMetadata{}, false, nil
	}

	var probe struct {
		Schema string `json:"schema"`
		Kind   string `json:"kind"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return detachedHarnessRunMetadata{}, false, nil
	}
	if strings.TrimSpace(probe.Schema) != harnessDetachedMetadataSchema ||
		strings.TrimSpace(probe.Kind) != harnessDetachedRunMetadataKey {
		return detachedHarnessRunMetadata{}, false, nil
	}

	metadata, err := decodeDetachedHarnessRunMetadata(raw)
	if err != nil {
		return detachedHarnessRunMetadata{}, false, err
	}
	return metadata, true, nil
}

func detachedHarnessActorContext(ownerSessionID string) (taskpkg.ActorContext, error) {
	suffix := strings.TrimSpace(ownerSessionID)
	return taskpkg.DeriveDaemonActorContext(
		harnessDetachedActorRefPrefix+":"+suffix,
		harnessDetachedOriginRefPrefix+"/"+suffix,
	)
}

func detachedHarnessTaskID(ownerSessionID string, submissionKey string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(ownerSessionID) + "\n" + strings.TrimSpace(submissionKey)))
	return "task-hdet-" + hex.EncodeToString(sum[:8])
}

func detachedHarnessSummary(summary string) string {
	trimmed := strings.TrimSpace(summary)
	if trimmed == "" {
		return defaultDetachedHarnessSummary
	}
	return trimmed
}

func detachedHarnessChannel(requested string, ownerChannel string) string {
	trimmed := strings.TrimSpace(requested)
	if trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(ownerChannel)
}

func normalizeDetachedHarnessTurnSource(source session.TurnSource) session.TurnSource {
	switch session.TurnSource(strings.TrimSpace(string(source))) {
	case "", session.TurnSourceUser:
		return session.TurnSourceUser
	case session.TurnSourceNetwork:
		return session.TurnSourceNetwork
	case session.TurnSourceSynthetic:
		return session.TurnSourceSynthetic
	default:
		return session.TurnSourceUser
	}
}
