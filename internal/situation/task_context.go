package situation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pedronauck/agh/internal/api/contract"
	aghconfig "github.com/pedronauck/agh/internal/config"
	taskpkg "github.com/pedronauck/agh/internal/task"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func (s *Service) BundleForActiveLease(
	ctx context.Context,
	req taskpkg.ContextRequest,
) (taskpkg.ContextBundle, error) {
	if err := checkContext(ctx); err != nil {
		return taskpkg.ContextBundle{}, err
	}
	store := s.taskStoreValue()
	if store == nil {
		return taskpkg.ContextBundle{}, nil
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return taskpkg.ContextBundle{}, fmt.Errorf("%w: session_id is required", taskpkg.ErrPermissionDenied)
	}
	run, ok, err := s.activeRunForSession(ctx, sessionID)
	if err != nil || !ok {
		return taskpkg.ContextBundle{}, err
	}
	if runID := strings.TrimSpace(req.RunID); runID != "" && runID != strings.TrimSpace(run.ID) {
		return taskpkg.ContextBundle{}, fmt.Errorf(
			"%w: run %q is not owned by session %q",
			taskpkg.ErrPermissionDenied,
			runID,
			sessionID,
		)
	}
	taskRecord, err := store.GetTask(ctx, run.TaskID)
	if err != nil {
		return taskpkg.ContextBundle{}, err
	}
	workspaceSnapshot, err := s.resolveWorkspace(ctx, taskRecord.WorkspaceID, "", nil)
	if err != nil {
		return taskpkg.ContextBundle{}, err
	}
	return s.bundleForRun(ctx, taskRecord, run, workspaceSnapshot, nil)
}

func (s *Service) BundleForOperatorTask(
	ctx context.Context,
	req taskpkg.OperatorTaskContextRequest,
) (taskpkg.ContextBundle, error) {
	if err := checkContext(ctx); err != nil {
		return taskpkg.ContextBundle{}, err
	}
	store := s.taskStoreValue()
	if store == nil {
		return taskpkg.ContextBundle{}, nil
	}
	taskID := strings.TrimSpace(req.TaskID)
	if taskID == "" {
		return taskpkg.ContextBundle{}, fmt.Errorf("%w: task_id is required", taskpkg.ErrValidation)
	}
	taskRecord, err := store.GetTask(ctx, taskID)
	if err != nil {
		return taskpkg.ContextBundle{}, err
	}
	workspaceSnapshot, err := s.resolveWorkspace(ctx, taskRecord.WorkspaceID, "", nil)
	if err != nil {
		return taskpkg.ContextBundle{}, err
	}
	runs, err := store.ListTaskRuns(ctx, taskpkg.RunQuery{TaskID: taskID})
	if err != nil {
		return taskpkg.ContextBundle{}, err
	}
	run, ok := selectTaskContextRun(taskRecord, runs)
	if !ok {
		return s.bundleForTaskOnly(ctx, taskRecord, workspaceSnapshot)
	}
	return s.bundleForRun(ctx, taskRecord, run, workspaceSnapshot, nil)
}

func (s *Service) TaskRunPromptOverlay(
	ctx context.Context,
	taskRecord taskpkg.Task,
	run taskpkg.Run,
	profile *taskpkg.ExecutionProfile,
) (string, error) {
	if s == nil {
		return "", nil
	}
	workspaceSnapshot, err := s.resolveWorkspace(ctx, taskRecord.WorkspaceID, "", nil)
	if err != nil {
		return "", err
	}
	bundle, err := s.bundleForRun(ctx, taskRecord, run, workspaceSnapshot, profile)
	if err != nil {
		return "", err
	}
	return renderTaskBundlePrompt(bundle)
}

func (s *Service) TaskRunPromptOverlayByID(ctx context.Context, taskID string, runID string) (string, error) {
	if s == nil {
		return "", nil
	}
	if err := checkContext(ctx); err != nil {
		return "", err
	}
	store := s.taskStoreValue()
	if store == nil {
		return "", nil
	}
	taskRecord, err := store.GetTask(ctx, strings.TrimSpace(taskID))
	if err != nil {
		return "", err
	}
	run, err := store.GetTaskRun(ctx, strings.TrimSpace(runID))
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(run.TaskID) != strings.TrimSpace(taskRecord.ID) {
		return "", fmt.Errorf(
			"%w: run %q belongs to task %q, not task %q",
			taskpkg.ErrValidation,
			run.ID,
			run.TaskID,
			taskRecord.ID,
		)
	}
	return s.TaskRunPromptOverlay(ctx, taskRecord, run, nil)
}

func (s *Service) activeRunForSession(
	ctx context.Context,
	sessionID string,
) (taskpkg.Run, bool, error) {
	store := s.taskStoreValue()
	if store == nil {
		return taskpkg.Run{}, false, nil
	}
	runs, err := store.ListTaskRuns(ctx, taskpkg.RunQuery{SessionID: strings.TrimSpace(sessionID)})
	if err != nil {
		return taskpkg.Run{}, false, err
	}
	run, ok := selectActiveRun(runs)
	return run, ok, nil
}

func (s *Service) bundleForTaskOnly(
	ctx context.Context,
	taskRecord taskpkg.Task,
	workspaceSnapshot *workspacepkg.ResolvedWorkspace,
) (taskpkg.ContextBundle, error) {
	cfg := taskContextConfig(workspaceSnapshot)
	profile, err := s.loadTaskExecutionProfile(ctx, taskRecord.ID)
	if err != nil {
		return taskpkg.ContextBundle{}, err
	}
	bundle := taskpkg.ContextBundle{
		Task:             taskReference(taskRecord),
		LatestEventSeq:   taskRecord.LatestEventSeq,
		Limits:           taskContextRuntimeLimits(cfg),
		ExecutionProfile: &profile,
		PriorAttempts:    []taskpkg.RunSummary{},
		RecentEvents:     []taskpkg.TimelineItem{},
		ReviewHistory:    []taskpkg.RunReviewSummary{},
	}
	return s.enforceTaskContextBudget(taskpkg.NormalizeContextBundle(bundle), cfg.ContextBodyMaxBytes)
}

func (s *Service) bundleForRun(
	ctx context.Context,
	taskRecord taskpkg.Task,
	run taskpkg.Run,
	workspaceSnapshot *workspacepkg.ResolvedWorkspace,
	profileOverride *taskpkg.ExecutionProfile,
) (taskpkg.ContextBundle, error) {
	cfg := taskContextConfig(workspaceSnapshot)
	profile, err := s.loadTaskExecutionProfile(ctx, taskRecord.ID)
	if err != nil {
		return taskpkg.ContextBundle{}, err
	}
	if profileOverride != nil {
		profile = *profileOverride
	}
	priorAttempts, err := s.priorRunSummaries(ctx, taskRecord, run, cfg.ContextPriorAttempts)
	if err != nil {
		return taskpkg.ContextBundle{}, err
	}
	recentEvents, err := s.recentTaskEvents(ctx, taskRecord, cfg.ContextRecentEvents)
	if err != nil {
		return taskpkg.ContextBundle{}, err
	}
	reviewHistory, err := s.reviewHistory(ctx, taskRecord, cfg)
	if err != nil {
		return taskpkg.ContextBundle{}, err
	}
	reviewContinuation, err := s.reviewContinuation(ctx, run, cfg)
	if err != nil {
		return taskpkg.ContextBundle{}, err
	}

	currentRun := runSummaryFromTaskRun(run, taskRecord.MaxAttempts)
	bundle := taskpkg.ContextBundle{
		Task:               taskReference(taskRecord),
		LatestEventSeq:     taskRecord.LatestEventSeq,
		CurrentRun:         &currentRun,
		PriorAttempts:      priorAttempts,
		RecentEvents:       recentEvents,
		HandoffSummary:     handoffSummary(run, reviewContinuation, cfg.ContextBodyMaxBytes),
		Limits:             taskContextRuntimeLimits(cfg),
		ExecutionProfile:   &profile,
		ReviewContinuation: reviewContinuation,
		ReviewHistory:      reviewHistory,
	}
	return s.enforceTaskContextBudget(taskpkg.NormalizeContextBundle(bundle), cfg.ContextBodyMaxBytes)
}

func (s *Service) loadTaskExecutionProfile(ctx context.Context, taskID string) (taskpkg.ExecutionProfile, error) {
	store := s.taskStoreValue()
	if store == nil {
		return taskpkg.DefaultExecutionProfile(taskID), nil
	}
	profile, err := store.GetExecutionProfile(ctx, taskID)
	switch {
	case errors.Is(err, taskpkg.ErrExecutionProfileNotFound):
		return taskpkg.DefaultExecutionProfile(taskID), nil
	case err != nil:
		return taskpkg.ExecutionProfile{}, err
	default:
		return profile, nil
	}
}

func (s *Service) priorRunSummaries(
	ctx context.Context,
	taskRecord taskpkg.Task,
	current taskpkg.Run,
	limit int,
) ([]taskpkg.RunSummary, error) {
	if limit <= 0 {
		return []taskpkg.RunSummary{}, nil
	}
	store := s.taskStoreValue()
	if store == nil {
		return []taskpkg.RunSummary{}, nil
	}
	runs, err := store.ListTaskRuns(ctx, taskpkg.RunQuery{TaskID: taskRecord.ID})
	if err != nil {
		return nil, err
	}
	prior := make([]taskpkg.Run, 0, len(runs))
	for _, run := range runs {
		if strings.TrimSpace(run.ID) == strings.TrimSpace(current.ID) {
			continue
		}
		if run.Attempt > 0 && current.Attempt > 0 && run.Attempt >= current.Attempt {
			continue
		}
		prior = append(prior, run)
	}
	sortRunsByAttemptAndActivity(prior)
	summaries := make([]taskpkg.RunSummary, 0, min(len(prior), limit))
	for _, run := range prior {
		if len(summaries) == limit {
			break
		}
		summaries = append(summaries, runSummaryFromTaskRun(run, taskRecord.MaxAttempts))
	}
	return summaries, nil
}

func (s *Service) recentTaskEvents(
	ctx context.Context,
	taskRecord taskpkg.Task,
	limit int,
) ([]taskpkg.TimelineItem, error) {
	if limit <= 0 {
		return []taskpkg.TimelineItem{}, nil
	}
	store := s.taskStoreValue()
	if store == nil {
		return []taskpkg.TimelineItem{}, nil
	}
	records, err := store.ListTaskEventRecords(ctx, taskpkg.EventRecordQuery{
		TaskID:     taskRecord.ID,
		Limit:      limit,
		Descending: true,
	})
	if err != nil {
		return nil, err
	}
	runs, err := store.ListTaskRuns(ctx, taskpkg.RunQuery{TaskID: taskRecord.ID})
	if err != nil {
		return nil, err
	}
	runsByID := make(map[string]taskpkg.Run, len(runs))
	for _, run := range runs {
		runsByID[strings.TrimSpace(run.ID)] = run
	}
	items := make([]taskpkg.TimelineItem, 0, len(records))
	for _, record := range records {
		event := record.Event
		var runSummary *taskpkg.RunSummary
		if run, ok := runsByID[strings.TrimSpace(event.RunID)]; ok {
			summary := runSummaryFromTaskRun(run, taskRecord.MaxAttempts)
			runSummary = &summary
		}
		payload := cloneRawJSON(event.Payload)
		redactedPayload, err := redactTaskContextPayload(payload)
		if err != nil {
			return nil, err
		}
		items = append(items, taskpkg.TimelineItem{
			Sequence:  record.Sequence,
			EventID:   strings.TrimSpace(event.ID),
			Task:      taskReference(taskRecord),
			Run:       runSummary,
			EventType: strings.TrimSpace(event.EventType),
			Actor:     event.Actor,
			Origin:    event.Origin,
			Payload:   redactedPayload,
			Timestamp: event.Timestamp.UTC(),
		})
	}
	slices.Reverse(items)
	return items, nil
}

func (s *Service) reviewContinuation(
	ctx context.Context,
	run taskpkg.Run,
	cfg aghconfig.TaskOrchestrationConfig,
) (*taskpkg.ReviewContinuation, error) {
	if run.Review == nil ||
		strings.TrimSpace(run.Review.ReviewID) == "" ||
		run.Review.ContinuationReason == "" {
		return nil, nil
	}
	store := s.taskStoreValue()
	if store == nil {
		return nil, nil
	}
	review, err := store.GetRunReview(ctx, run.Review.ReviewID)
	if err != nil {
		return nil, err
	}
	missingWorkRaw := run.Review.MissingWork
	if len(missingWorkRaw) == 0 {
		missingWorkRaw = review.MissingWork
	}
	missingWork, err := boundedMissingWork(
		missingWorkRaw,
		cfg.Review.MissingWorkMaxItems,
		cfg.Review.MissingWorkItemMaxBytes,
	)
	if err != nil {
		return nil, err
	}
	reviewRound := run.Review.ReviewRound
	if reviewRound == 0 {
		reviewRound = review.ReviewRound
	}
	return &taskpkg.ReviewContinuation{
		ReviewID:      strings.TrimSpace(run.Review.ReviewID),
		ReviewedRunID: firstTrimmed(run.Review.ParentRunID, review.RunID),
		ReviewRound:   reviewRound,
		Outcome:       string(review.Outcome.Normalize()),
		Reason: safeTaskContextText(
			firstTrimmed(review.Reason, run.Review.ContinuationReason),
			cfg.Review.ReasonMaxBytes,
		),
		MissingWork: missingWork,
		NextRoundGuidance: safeTaskContextText(
			firstTrimmed(run.Review.NextRoundGuidance, review.NextRoundGuidance),
			cfg.Review.NextRoundGuidanceMaxBytes,
		),
	}, nil
}

func (s *Service) reviewHistory(
	ctx context.Context,
	taskRecord taskpkg.Task,
	cfg aghconfig.TaskOrchestrationConfig,
) ([]taskpkg.RunReviewSummary, error) {
	limit := cfg.ContextPriorAttempts
	if limit <= 0 {
		return []taskpkg.RunReviewSummary{}, nil
	}
	store := s.taskStoreValue()
	if store == nil {
		return []taskpkg.RunReviewSummary{}, nil
	}
	reviews, err := store.ListRunReviews(ctx, taskpkg.RunReviewQuery{TaskID: taskRecord.ID, Limit: limit})
	if err != nil {
		return nil, err
	}
	summaries := make([]taskpkg.RunReviewSummary, 0, len(reviews))
	for _, review := range reviews {
		summaries = append(summaries, runReviewSummary(review, cfg))
	}
	return summaries, nil
}

func (s *Service) enforceTaskContextBudget(
	bundle taskpkg.ContextBundle,
	maxBytes int,
) (taskpkg.ContextBundle, error) {
	if maxBytes <= 0 {
		return taskpkg.NormalizeContextBundle(bundle), nil
	}
	bundle = taskpkg.NormalizeContextBundle(bundle)
	for {
		over, err := taskContextOverBudget(bundle, maxBytes)
		if err != nil {
			return taskpkg.ContextBundle{}, err
		}
		if !over {
			return bundle, nil
		}
		switch {
		case len(bundle.RecentEvents) > 0:
			bundle.RecentEvents = bundle.RecentEvents[1:]
		case len(bundle.PriorAttempts) > 0:
			bundle.PriorAttempts = bundle.PriorAttempts[:len(bundle.PriorAttempts)-1]
		case len(bundle.ReviewHistory) > 0:
			bundle.ReviewHistory = bundle.ReviewHistory[:len(bundle.ReviewHistory)-1]
		case strings.TrimSpace(bundle.HandoffSummary) != "":
			bundle.HandoffSummary = truncateUTF8Bytes(bundle.HandoffSummary, len(bundle.HandoffSummary)/2)
		case bundle.ReviewContinuation != nil && strings.TrimSpace(bundle.ReviewContinuation.NextRoundGuidance) != "":
			continuation := *bundle.ReviewContinuation
			continuation.NextRoundGuidance = truncateUTF8Bytes(
				continuation.NextRoundGuidance,
				len(continuation.NextRoundGuidance)/2,
			)
			bundle.ReviewContinuation = &continuation
		default:
			return taskpkg.ContextBundle{}, fmt.Errorf(
				"%w: task context bundle exceeds %d bytes after trimming",
				taskpkg.ErrPayloadTooLarge,
				maxBytes,
			)
		}
	}
}

func renderTaskBundlePrompt(bundle taskpkg.ContextBundle) (string, error) {
	payload := contract.AgentContextPayload{
		Task: contract.AgentTaskContextPayload{
			Available: true,
			Bundle:    &bundle,
		},
	}
	return RenderPrompt(&payload)
}

func taskContextOverBudget(bundle taskpkg.ContextBundle, maxBytes int) (bool, error) {
	content, err := json.Marshal(bundle)
	if err != nil {
		return false, fmt.Errorf("situation: marshal task context bundle: %w", err)
	}
	return len(content) > maxBytes, nil
}

func taskContextConfig(workspaceSnapshot *workspacepkg.ResolvedWorkspace) aghconfig.TaskOrchestrationConfig {
	defaults := aghconfig.DefaultTaskConfig().Orchestration
	if workspaceSnapshot == nil {
		return defaults
	}
	cfg := workspaceSnapshot.Config.Task.Orchestration
	if err := cfg.Validate("task.orchestration"); err != nil {
		return defaults
	}
	return cfg
}

func taskContextRuntimeLimits(cfg aghconfig.TaskOrchestrationConfig) taskpkg.RuntimeLimits {
	return taskpkg.RuntimeLimits{
		MaxRuntimeSeconds: int64(cfg.DefaultMaxRuntime.Seconds()),
		SummaryMaxBytes:   cfg.SummaryMaxBytes,
		ContextMaxBytes:   cfg.ContextBodyMaxBytes,
	}
}

func taskReference(taskRecord taskpkg.Task) taskpkg.Reference {
	return taskpkg.Reference{
		ID:             strings.TrimSpace(taskRecord.ID),
		Identifier:     safeTaskContextText(taskRecord.Identifier, maxReviewReasonFallback),
		Title:          safeTaskContextText(taskRecord.Title, maxReviewReasonFallback),
		Status:         taskRecord.Status,
		Priority:       taskRecord.Priority,
		Owner:          cloneOwnership(taskRecord.Owner),
		Scope:          taskRecord.Scope,
		WorkspaceID:    strings.TrimSpace(taskRecord.WorkspaceID),
		LatestEventSeq: taskRecord.LatestEventSeq,
	}
}

func runSummaryFromTaskRun(run taskpkg.Run, maxAttempts int) taskpkg.RunSummary {
	return taskpkg.RunSummary{
		ID:                    strings.TrimSpace(run.ID),
		TaskID:                strings.TrimSpace(run.TaskID),
		Status:                run.Status.Normalize(),
		Attempt:               run.Attempt,
		MaxAttempts:           maxAttempts,
		SessionID:             strings.TrimSpace(run.SessionID),
		ClaimedBy:             cloneActorIdentity(run.ClaimedBy),
		ClaimTokenHash:        strings.TrimSpace(run.ClaimTokenHash),
		LeaseUntil:            run.LeaseUntil.UTC(),
		HeartbeatAt:           run.HeartbeatAt.UTC(),
		CoordinationChannelID: strings.TrimSpace(run.CoordinationChannelID),
		QueuedAt:              run.QueuedAt.UTC(),
		ClaimedAt:             run.ClaimedAt.UTC(),
		StartedAt:             run.StartedAt.UTC(),
		EndedAt:               run.EndedAt.UTC(),
		Error:                 safeTaskContextText(run.Error, maxReviewReasonFallback),
	}
}

const maxReviewReasonFallback = 2048

func runReviewSummary(
	review taskpkg.RunReview,
	cfg aghconfig.TaskOrchestrationConfig,
) taskpkg.RunReviewSummary {
	return taskpkg.RunReviewSummary{
		ReviewID:      strings.TrimSpace(review.ReviewID),
		RunID:         strings.TrimSpace(review.RunID),
		ReviewRound:   review.ReviewRound,
		Attempt:       review.Attempt,
		Status:        string(review.Status.Normalize()),
		Outcome:       string(review.Outcome.Normalize()),
		Reason:        safeTaskContextText(review.Reason, cfg.Review.ReasonMaxBytes),
		ReviewedAt:    formatOptionalTime(review.ReviewedAt),
		ReviewerLabel: safeTaskContextText(reviewReviewerLabel(review), maxReviewReasonFallback),
	}
}

func reviewReviewerLabel(review taskpkg.RunReview) string {
	return firstTrimmed(review.ReviewerAgentName, review.ReviewerPeerID, review.ReviewerSessionID)
}

func formatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func boundedMissingWork(raw json.RawMessage, maxItems int, maxItemBytes int) ([]string, error) {
	if len(raw) == 0 {
		return []string{}, nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("situation: decode review missing work: %w", err)
	}
	if maxItems <= 0 || len(values) < maxItems {
		maxItems = len(values)
	}
	result := make([]string, 0, maxItems)
	for _, value := range values {
		if len(result) == maxItems {
			break
		}
		trimmed := safeTaskContextText(value, maxItemBytes)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result, nil
}

func handoffSummary(
	run taskpkg.Run,
	continuation *taskpkg.ReviewContinuation,
	contextMaxBytes int,
) string {
	if continuation != nil {
		return safeTaskContextText(continuation.NextRoundGuidance, contextMaxBytes/2)
	}
	return safeTaskContextText(run.Error, contextMaxBytes/2)
}

func selectTaskContextRun(taskRecord taskpkg.Task, runs []taskpkg.Run) (taskpkg.Run, bool) {
	currentRunID := strings.TrimSpace(taskRecord.CurrentRunID)
	if currentRunID != "" {
		for _, run := range runs {
			if strings.TrimSpace(run.ID) == currentRunID {
				return run, true
			}
		}
	}
	if run, ok := selectActiveRun(runs); ok {
		return run, true
	}
	if len(runs) == 0 {
		return taskpkg.Run{}, false
	}
	sorted := append([]taskpkg.Run(nil), runs...)
	sortRunsByAttemptAndActivity(sorted)
	return sorted[0], true
}

func sortRunsByAttemptAndActivity(runs []taskpkg.Run) {
	slices.SortStableFunc(runs, func(left, right taskpkg.Run) int {
		if left.Attempt != right.Attempt {
			return right.Attempt - left.Attempt
		}
		leftTime := runActivityTime(left)
		rightTime := runActivityTime(right)
		if !leftTime.Equal(rightTime) {
			if leftTime.After(rightTime) {
				return -1
			}
			return 1
		}
		return strings.Compare(left.ID, right.ID)
	})
}

func cloneRawJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}

func redactTaskContextPayload(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("situation: decode task context payload: %w", err)
	}
	cleaned := redactTaskContextJSONValue(decoded)
	content, err := json.Marshal(cleaned)
	if err != nil {
		return nil, fmt.Errorf("situation: encode task context payload: %w", err)
	}
	return json.RawMessage(content), nil
}

func redactTaskContextJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		cleaned := make(map[string]any, len(typed))
		for key, nested := range typed {
			if strings.EqualFold(strings.TrimSpace(key), "claim_token") {
				continue
			}
			cleaned[key] = redactTaskContextJSONValue(nested)
		}
		return cleaned
	case []any:
		cleaned := make([]any, len(typed))
		for idx, nested := range typed {
			cleaned[idx] = redactTaskContextJSONValue(nested)
		}
		return cleaned
	case string:
		return taskpkg.RedactClaimTokens(typed)
	default:
		return value
	}
}

func safeTaskContextText(value string, limit int) string {
	return truncateUTF8Bytes(taskpkg.RedactClaimTokens(singleLine(value)), limit)
}

func truncateUTF8Bytes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if len(value) <= limit {
		return value
	}
	if limit <= len("...") {
		return strings.Repeat(".", limit)
	}
	cut := limit - len("...")
	for cut > 0 && !utf8.ValidString(value[:cut]) {
		cut--
	}
	return strings.TrimSpace(value[:cut]) + "..."
}
