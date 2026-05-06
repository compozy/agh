package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	defaultRunReviewAttempt = 1
	defaultRunReviewRound   = 1

	maxRunReviewReasonBytes        = 4096
	maxRunReviewGuidanceBytes      = 8192
	maxRunReviewTextBytes          = MaxResultBytes
	maxRunReviewSelectorFieldBytes = 256
)

var defaultMissingWorkJSON = json.RawMessage("[]")

// ReviewPolicy identifies when a terminal task run needs a review gate.
type ReviewPolicy string

const (
	// ReviewPolicyNone disables review gating.
	ReviewPolicyNone ReviewPolicy = "none"
	// ReviewPolicyOnSuccess requests review only after a successful run.
	ReviewPolicyOnSuccess ReviewPolicy = "on_success"
	// ReviewPolicyOnFailure requests review only after a failed or canceled run.
	ReviewPolicyOnFailure ReviewPolicy = "on_failure"
	// ReviewPolicyAlways requests review after every terminal run.
	ReviewPolicyAlways ReviewPolicy = "always"
)

// RunReviewStatus identifies the lifecycle state of one run review request.
type RunReviewStatus string

const (
	// RunReviewStatusRequested reports a persisted review request awaiting routing.
	RunReviewStatusRequested RunReviewStatus = "requested"
	// RunReviewStatusRouted reports a review that has been routed but not yet bound.
	RunReviewStatusRouted RunReviewStatus = "routed"
	// RunReviewStatusInReview reports a review bound to a reviewer session.
	RunReviewStatusInReview RunReviewStatus = "in_review"
	// RunReviewStatusRecorded reports a persisted terminal reviewer verdict.
	RunReviewStatusRecorded RunReviewStatus = "recorded"
	// RunReviewStatusCircuitOpened reports review routing stopped by circuit policy.
	RunReviewStatusCircuitOpened RunReviewStatus = "circuit_opened"
	// RunReviewStatusCanceled reports an explicitly canceled review request.
	RunReviewStatusCanceled RunReviewStatus = "canceled"
)

// RunReviewOutcome identifies the authoritative reviewer verdict.
type RunReviewOutcome string

const (
	// RunReviewOutcomeApproved accepts the run result.
	RunReviewOutcomeApproved RunReviewOutcome = "approved"
	// RunReviewOutcomeRejected rejects the run and may request continuation work.
	RunReviewOutcomeRejected RunReviewOutcome = "rejected"
	// RunReviewOutcomeBlocked reports the review could not continue due to external blockers.
	RunReviewOutcomeBlocked RunReviewOutcome = "blocked"
	// RunReviewOutcomeError reports reviewer/tool execution failed.
	RunReviewOutcomeError RunReviewOutcome = "error"
	// RunReviewOutcomeTimeout reports the review exceeded its deadline.
	RunReviewOutcomeTimeout RunReviewOutcome = "timeout"
	// RunReviewOutcomeInvalidOutput reports the reviewer returned a malformed verdict.
	RunReviewOutcomeInvalidOutput RunReviewOutcome = "invalid_output"
)

// RunReview is the task-domain record for one post-terminal review gate.
type RunReview struct {
	ReviewID          string           `json:"review_id"`
	TaskID            string           `json:"task_id"`
	RunID             string           `json:"run_id"`
	ParentReviewID    string           `json:"parent_review_id,omitempty"`
	Policy            ReviewPolicy     `json:"policy"`
	ReviewRound       int              `json:"review_round"`
	Attempt           int              `json:"attempt"`
	Status            RunReviewStatus  `json:"status"`
	Outcome           RunReviewOutcome `json:"outcome,omitempty"`
	Confidence        *float64         `json:"confidence,omitempty"`
	Reason            string           `json:"reason,omitempty"`
	DeliveryID        string           `json:"delivery_id,omitempty"`
	MissingWork       json.RawMessage  `json:"missing_work,omitempty"`
	NextRoundGuidance string           `json:"next_round_guidance,omitempty"`
	ReviewText        string           `json:"review_text,omitempty"`
	ReviewerSessionID string           `json:"reviewer_session_id,omitempty"`
	ReviewerAgentName string           `json:"reviewer_agent_name,omitempty"`
	ReviewerPeerID    string           `json:"reviewer_peer_id,omitempty"`
	ReviewerChannelID string           `json:"reviewer_channel_id,omitempty"`
	ReviewedBy        *ActorIdentity   `json:"reviewed_by,omitempty"`
	RequestedAt       time.Time        `json:"requested_at"`
	RoutedAt          time.Time        `json:"routed_at"`
	StartedAt         time.Time        `json:"started_at"`
	ReviewedAt        time.Time        `json:"reviewed_at"`
	DeadlineAt        time.Time        `json:"deadline_at"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
}

// RunReviewRequest captures one authoritative request to review a terminal run.
type RunReviewRequest struct {
	TaskID         string       `json:"task_id"`
	RunID          string       `json:"run_id"`
	ReviewRound    int          `json:"review_round,omitempty"`
	Attempt        int          `json:"attempt,omitempty"`
	Policy         ReviewPolicy `json:"policy,omitempty"`
	ParentReviewID string       `json:"parent_review_id,omitempty"`
	Reason         string       `json:"reason,omitempty"`
	DeadlineAt     time.Time    `json:"deadline_at"`
}

// RunReviewRequestedNotification is emitted after one review request is
// durably created and its audit event has been recorded.
type RunReviewRequestedNotification struct {
	Review RunReview
	Task   Task
	Run    Run
	Actor  ActorContext
}

// RunReviewRequestedObserver receives typed review-request wakeups without
// tailing task events or storage tables.
type RunReviewRequestedObserver interface {
	OnRunReviewRequested(ctx context.Context, notification *RunReviewRequestedNotification)
}

// BindRunReviewSessionRequest captures a reviewer-session binding.
type BindRunReviewSessionRequest struct {
	ReviewID          string `json:"review_id"`
	SessionID         string `json:"session_id"`
	ReviewerAgentName string `json:"reviewer_agent_name,omitempty"`
	ReviewerPeerID    string `json:"reviewer_peer_id,omitempty"`
	ReviewerChannelID string `json:"reviewer_channel_id,omitempty"`
}

// RunReviewBinding is the lookup shape consumed by reviewer-session tooling.
type RunReviewBinding struct {
	Review            RunReview `json:"review"`
	SessionID         string    `json:"session_id"`
	ReviewerAgentName string    `json:"reviewer_agent_name,omitempty"`
	ReviewerPeerID    string    `json:"reviewer_peer_id,omitempty"`
	ReviewerChannelID string    `json:"reviewer_channel_id,omitempty"`
}

// RunReviewQuery captures supported review read filters.
type RunReviewQuery struct {
	TaskID            string          `json:"task_id,omitempty"`
	RunID             string          `json:"run_id,omitempty"`
	Status            RunReviewStatus `json:"status,omitempty"`
	ReviewerSessionID string          `json:"reviewer_session_id,omitempty"`
	Limit             int             `json:"limit,omitempty"`
}

// RunReviewVerdict captures the terminal reviewer payload persisted by the task domain.
type RunReviewVerdict struct {
	Outcome           RunReviewOutcome `json:"outcome"`
	Confidence        *float64         `json:"confidence"`
	Reason            string           `json:"reason"`
	DeliveryID        string           `json:"delivery_id"`
	MissingWork       json.RawMessage  `json:"missing_work,omitempty"`
	NextRoundGuidance string           `json:"next_round_guidance,omitempty"`
	ReviewText        string           `json:"review_text,omitempty"`
}

// RecordRunReviewRequest captures an authoritative persisted review verdict write.
type RecordRunReviewRequest struct {
	ReviewID string           `json:"review_id"`
	RunID    string           `json:"run_id"`
	Verdict  RunReviewVerdict `json:"verdict"`
}

// RunReviewResult reports the stored verdict and optional rejected-review continuation run.
type RunReviewResult struct {
	Review          RunReview `json:"review"`
	ContinuationRun *Run      `json:"continuation_run,omitempty"`
	CircuitOpened   bool      `json:"circuit_opened,omitempty"`
}

// Normalize returns the normalized review policy value.
func (p ReviewPolicy) Normalize() ReviewPolicy {
	return ReviewPolicy(strings.ToLower(strings.TrimSpace(string(p))))
}

// Validate reports whether the review policy is one of the supported values.
func (p ReviewPolicy) Validate(path string) error {
	switch p.Normalize() {
	case ReviewPolicyNone, ReviewPolicyOnSuccess, ReviewPolicyOnFailure, ReviewPolicyAlways:
		return nil
	case "":
		return fmt.Errorf("%w: %s is required", ErrValidation, path)
	default:
		return fmt.Errorf(
			"%w: %s must be one of %q, %q, %q, or %q: %q",
			ErrValidation,
			path,
			ReviewPolicyNone,
			ReviewPolicyOnSuccess,
			ReviewPolicyOnFailure,
			ReviewPolicyAlways,
			p,
		)
	}
}

// MatchesRunStatus reports whether this policy applies to a terminal run status.
func (p ReviewPolicy) MatchesRunStatus(status RunStatus) bool {
	switch p.Normalize() {
	case ReviewPolicyOnSuccess:
		return status.Normalize() == TaskRunStatusCompleted
	case ReviewPolicyOnFailure:
		switch status.Normalize() {
		case TaskRunStatusFailed, TaskRunStatusCanceled:
			return true
		default:
			return false
		}
	case ReviewPolicyAlways:
		return IsTerminalRunStatus(status)
	default:
		return false
	}
}

// Normalize returns the normalized run-review status.
func (s RunReviewStatus) Normalize() RunReviewStatus {
	return RunReviewStatus(strings.ToLower(strings.TrimSpace(string(s))))
}

// Validate reports whether the run-review status is supported.
func (s RunReviewStatus) Validate(path string) error {
	switch s.Normalize() {
	case RunReviewStatusRequested,
		RunReviewStatusRouted,
		RunReviewStatusInReview,
		RunReviewStatusRecorded,
		RunReviewStatusCircuitOpened,
		RunReviewStatusCanceled:
		return nil
	case "":
		return fmt.Errorf("%w: %s is required", ErrValidation, path)
	default:
		return fmt.Errorf("%w: %s has unsupported value %q", ErrValidation, path, s)
	}
}

// Normalize returns the normalized run-review outcome.
func (o RunReviewOutcome) Normalize() RunReviewOutcome {
	return RunReviewOutcome(strings.ToLower(strings.TrimSpace(string(o))))
}

// Validate reports whether the run-review outcome is supported.
func (o RunReviewOutcome) Validate(path string) error {
	switch o.Normalize() {
	case RunReviewOutcomeApproved,
		RunReviewOutcomeRejected,
		RunReviewOutcomeBlocked,
		RunReviewOutcomeError,
		RunReviewOutcomeTimeout,
		RunReviewOutcomeInvalidOutput:
		return nil
	case "":
		return fmt.Errorf("%w: %s is required", ErrValidation, path)
	default:
		return fmt.Errorf("%w: %s has unsupported value %q", ErrValidation, path, o)
	}
}

// Normalize returns a canonical review record.
func (r *RunReview) Normalize(now time.Time) (RunReview, error) {
	if r == nil {
		return RunReview{}, fmt.Errorf("%w: task_run_review is required", ErrValidation)
	}
	normalized := *r
	normalized.ReviewID = strings.TrimSpace(normalized.ReviewID)
	normalized.TaskID = strings.TrimSpace(normalized.TaskID)
	normalized.RunID = strings.TrimSpace(normalized.RunID)
	normalized.ParentReviewID = strings.TrimSpace(normalized.ParentReviewID)
	normalized.Policy = normalized.Policy.Normalize()
	if normalized.Policy == "" {
		normalized.Policy = ReviewPolicyAlways
	}
	if normalized.ReviewRound == 0 {
		normalized.ReviewRound = defaultRunReviewRound
	}
	if normalized.Attempt == 0 {
		normalized.Attempt = defaultRunReviewAttempt
	}
	normalized.Status = normalized.Status.Normalize()
	if normalized.Status == "" {
		normalized.Status = RunReviewStatusRequested
	}
	normalized.Outcome = normalized.Outcome.Normalize()
	normalized.Reason = strings.TrimSpace(normalized.Reason)
	normalized.DeliveryID = strings.TrimSpace(normalized.DeliveryID)
	normalized.MissingWork = normalizeReviewMissingWork(normalized.MissingWork)
	normalized.NextRoundGuidance = strings.TrimSpace(normalized.NextRoundGuidance)
	normalized.ReviewText = strings.TrimSpace(normalized.ReviewText)
	normalized.ReviewerSessionID = strings.TrimSpace(normalized.ReviewerSessionID)
	normalized.ReviewerAgentName = strings.TrimSpace(normalized.ReviewerAgentName)
	normalized.ReviewerPeerID = strings.TrimSpace(normalized.ReviewerPeerID)
	normalized.ReviewerChannelID = strings.TrimSpace(normalized.ReviewerChannelID)
	if normalized.ReviewedBy != nil {
		reviewedBy := *normalized.ReviewedBy
		reviewedBy.Kind = reviewedBy.Kind.Normalize()
		reviewedBy.Ref = strings.TrimSpace(reviewedBy.Ref)
		normalized.ReviewedBy = &reviewedBy
	}
	normalized = normalizeRunReviewTimes(normalized, now)

	if err := (&normalized).Validate(); err != nil {
		return RunReview{}, err
	}
	return normalized, nil
}

// Validate reports whether the run-review record can be persisted.
func (r *RunReview) Validate() error {
	if r == nil {
		return fmt.Errorf("%w: task_run_review is required", ErrValidation)
	}
	if err := validateRunReviewIdentity(r); err != nil {
		return err
	}
	if err := validateRunReviewLifecycle(r); err != nil {
		return err
	}
	if err := validateRunReviewContent(r); err != nil {
		return err
	}
	return validateRunReviewActorsAndTimes(r)
}

func validateRunReviewIdentity(review *RunReview) error {
	if strings.TrimSpace(review.ReviewID) == "" {
		return fmt.Errorf("%w: task_run_review.review_id is required", ErrValidation)
	}
	if strings.TrimSpace(review.TaskID) == "" {
		return fmt.Errorf("%w: task_run_review.task_id is required", ErrValidation)
	}
	if strings.TrimSpace(review.RunID) == "" {
		return fmt.Errorf("%w: task_run_review.run_id is required", ErrValidation)
	}
	return nil
}

func validateRunReviewLifecycle(review *RunReview) error {
	if err := review.Policy.Validate("task_run_review.policy"); err != nil {
		return err
	}
	if review.Policy.Normalize() == ReviewPolicyNone && review.Status.Normalize() != RunReviewStatusCanceled {
		return fmt.Errorf("%w: task_run_review.policy cannot be %q for active reviews", ErrValidation, ReviewPolicyNone)
	}
	if review.ReviewRound <= 0 {
		return fmt.Errorf("%w: task_run_review.review_round must be positive: %d", ErrValidation, review.ReviewRound)
	}
	if review.Attempt <= 0 {
		return fmt.Errorf("%w: task_run_review.attempt must be positive: %d", ErrValidation, review.Attempt)
	}
	if err := review.Status.Validate("task_run_review.status"); err != nil {
		return err
	}
	if review.Outcome.Normalize() != "" {
		if err := review.Outcome.Validate("task_run_review.outcome"); err != nil {
			return err
		}
	}
	if review.Status.Normalize() == RunReviewStatusRecorded && review.Outcome.Normalize() == "" {
		return fmt.Errorf("%w: task_run_review.outcome is required when status is recorded", ErrValidation)
	}
	if review.Confidence != nil && (*review.Confidence < 0 || *review.Confidence > 1) {
		return fmt.Errorf("%w: task_run_review.confidence must be between 0 and 1", ErrValidation)
	}
	return nil
}

func validateRunReviewContent(review *RunReview) error {
	if err := validateBoundedReviewText(review.Reason, maxRunReviewReasonBytes, "task_run_review.reason"); err != nil {
		return err
	}
	if err := validateRunReviewMissingWork(review.MissingWork); err != nil {
		return err
	}
	if err := validateBoundedReviewText(
		review.NextRoundGuidance,
		maxRunReviewGuidanceBytes,
		"task_run_review.next_round_guidance",
	); err != nil {
		return err
	}
	if err := validateBoundedReviewText(
		review.ReviewText,
		maxRunReviewTextBytes,
		"task_run_review.review_text",
	); err != nil {
		return err
	}
	return validateRunReviewSelectors(review)
}

func validateRunReviewActorsAndTimes(review *RunReview) error {
	if review.ReviewedBy != nil {
		if err := review.ReviewedBy.Validate("task_run_review.reviewed_by"); err != nil {
			return err
		}
	}
	if review.RequestedAt.IsZero() {
		return fmt.Errorf("%w: task_run_review.requested_at is required", ErrValidation)
	}
	if review.CreatedAt.IsZero() {
		return fmt.Errorf("%w: task_run_review.created_at is required", ErrValidation)
	}
	if review.UpdatedAt.IsZero() {
		return fmt.Errorf("%w: task_run_review.updated_at is required", ErrValidation)
	}
	return nil
}

// Normalize returns a canonical review request.
func (r RunReviewRequest) Normalize() RunReviewRequest {
	normalized := r
	normalized.TaskID = strings.TrimSpace(normalized.TaskID)
	normalized.RunID = strings.TrimSpace(normalized.RunID)
	normalized.ParentReviewID = strings.TrimSpace(normalized.ParentReviewID)
	normalized.Policy = normalized.Policy.Normalize()
	if normalized.Policy == "" {
		normalized.Policy = ReviewPolicyAlways
	}
	if normalized.ReviewRound == 0 {
		normalized.ReviewRound = defaultRunReviewRound
	}
	if normalized.Attempt == 0 {
		normalized.Attempt = defaultRunReviewAttempt
	}
	normalized.Reason = strings.TrimSpace(normalized.Reason)
	if !normalized.DeadlineAt.IsZero() {
		normalized.DeadlineAt = normalized.DeadlineAt.UTC()
	}
	return normalized
}

// Validate reports whether the review request has enough data for a persisted request row.
func (r RunReviewRequest) Validate(path string) error {
	if strings.TrimSpace(r.TaskID) == "" {
		return fmt.Errorf("%w: %s is required", ErrValidation, nestedPath(path, "task_id"))
	}
	if strings.TrimSpace(r.RunID) == "" {
		return fmt.Errorf("%w: %s is required", ErrValidation, nestedPath(path, "run_id"))
	}
	if err := r.Policy.Validate(nestedPath(path, "policy")); err != nil {
		return err
	}
	if r.Policy.Normalize() == ReviewPolicyNone {
		return fmt.Errorf("%w: %s cannot be %q", ErrValidation, nestedPath(path, "policy"), ReviewPolicyNone)
	}
	if r.ReviewRound <= 0 {
		return fmt.Errorf(
			"%w: %s must be positive: %d",
			ErrValidation,
			nestedPath(path, "review_round"),
			r.ReviewRound,
		)
	}
	if r.Attempt <= 0 {
		return fmt.Errorf("%w: %s must be positive: %d", ErrValidation, nestedPath(path, "attempt"), r.Attempt)
	}
	return validateBoundedReviewText(r.Reason, maxRunReviewReasonBytes, nestedPath(path, "reason"))
}

// Normalize returns a canonical reviewer-session binding request.
func (r BindRunReviewSessionRequest) Normalize() BindRunReviewSessionRequest {
	normalized := r
	normalized.ReviewID = strings.TrimSpace(normalized.ReviewID)
	normalized.SessionID = strings.TrimSpace(normalized.SessionID)
	normalized.ReviewerAgentName = strings.TrimSpace(normalized.ReviewerAgentName)
	normalized.ReviewerPeerID = strings.TrimSpace(normalized.ReviewerPeerID)
	normalized.ReviewerChannelID = strings.TrimSpace(normalized.ReviewerChannelID)
	return normalized
}

// Validate reports whether the binding request can bind a reviewer session.
func (r BindRunReviewSessionRequest) Validate(path string) error {
	if strings.TrimSpace(r.ReviewID) == "" {
		return fmt.Errorf("%w: %s is required", ErrValidation, nestedPath(path, "review_id"))
	}
	if strings.TrimSpace(r.SessionID) == "" {
		return fmt.Errorf("%w: %s is required", ErrValidation, nestedPath(path, "session_id"))
	}
	return validateRunReviewSelectorFields(
		map[string]string{
			"reviewer_agent_name": r.ReviewerAgentName,
			"reviewer_peer_id":    r.ReviewerPeerID,
			"reviewer_channel_id": r.ReviewerChannelID,
		},
		path,
	)
}

// Validate reports whether review query filters are internally consistent.
func (q RunReviewQuery) Validate(path string) error {
	if q.Status.Normalize() != "" {
		if err := q.Status.Validate(nestedPath(path, "status")); err != nil {
			return err
		}
	}
	if q.Limit < 0 {
		return fmt.Errorf("%w: %s must be zero or positive: %d", ErrValidation, nestedPath(path, "limit"), q.Limit)
	}
	return nil
}

// Normalize returns a canonical verdict payload.
func (v RunReviewVerdict) Normalize() RunReviewVerdict {
	normalized := v
	normalized.Outcome = normalized.Outcome.Normalize()
	normalized.Reason = strings.TrimSpace(normalized.Reason)
	normalized.DeliveryID = strings.TrimSpace(normalized.DeliveryID)
	normalized.MissingWork = normalizeReviewMissingWork(normalized.MissingWork)
	normalized.NextRoundGuidance = strings.TrimSpace(normalized.NextRoundGuidance)
	normalized.ReviewText = strings.TrimSpace(normalized.ReviewText)
	return normalized
}

// Validate reports whether the verdict can be persisted as an authoritative reviewer decision.
func (v RunReviewVerdict) Validate(path string) error {
	if err := v.Outcome.Validate(nestedPath(path, "outcome")); err != nil {
		return err
	}
	if v.Confidence == nil {
		return fmt.Errorf("%w: %s is required", ErrValidation, nestedPath(path, "confidence"))
	}
	if *v.Confidence < 0 || *v.Confidence > 1 {
		return fmt.Errorf("%w: %s must be between 0 and 1", ErrValidation, nestedPath(path, "confidence"))
	}
	if strings.TrimSpace(v.Reason) == "" {
		return fmt.Errorf("%w: %s is required", ErrValidation, nestedPath(path, "reason"))
	}
	if strings.TrimSpace(v.DeliveryID) == "" {
		return fmt.Errorf("%w: %s is required", ErrValidation, nestedPath(path, "delivery_id"))
	}
	missingItems, err := decodeVerdictMissingWork(v.MissingWork, nestedPath(path, "missing_work"))
	if err != nil {
		return err
	}
	if v.Outcome.Normalize() == RunReviewOutcomeApproved && len(missingItems) > 0 {
		return fmt.Errorf(
			"%w: %s must be empty when outcome is %q",
			ErrValidation,
			nestedPath(path, "missing_work"),
			v.Outcome,
		)
	}
	if v.Outcome.Normalize() == RunReviewOutcomeRejected &&
		len(missingItems) == 0 &&
		strings.TrimSpace(v.NextRoundGuidance) == "" {
		return fmt.Errorf(
			"%w: rejected reviews require missing_work or next_round_guidance",
			ErrValidation,
		)
	}
	if err := validateBoundedReviewText(v.Reason, maxRunReviewReasonBytes, nestedPath(path, "reason")); err != nil {
		return err
	}
	if err := validateBoundedReviewText(
		v.NextRoundGuidance,
		maxRunReviewGuidanceBytes,
		nestedPath(path, "next_round_guidance"),
	); err != nil {
		return err
	}
	if err := validateBoundedReviewText(
		v.ReviewText,
		maxRunReviewTextBytes,
		nestedPath(path, "review_text"),
	); err != nil {
		return err
	}
	return validateRunReviewSelectorFields(map[string]string{"delivery_id": v.DeliveryID}, path)
}

// Normalize returns a canonical verdict-recording request.
func (r RecordRunReviewRequest) Normalize() RecordRunReviewRequest {
	normalized := r
	normalized.ReviewID = strings.TrimSpace(normalized.ReviewID)
	normalized.RunID = strings.TrimSpace(normalized.RunID)
	normalized.Verdict = normalized.Verdict.Normalize()
	return normalized
}

// Validate reports whether the verdict-recording request can identify a review and run.
func (r RecordRunReviewRequest) Validate(path string) error {
	if strings.TrimSpace(r.ReviewID) == "" {
		return fmt.Errorf("%w: %s is required", ErrValidation, nestedPath(path, "review_id"))
	}
	if strings.TrimSpace(r.RunID) == "" {
		return fmt.Errorf("%w: %s is required", ErrValidation, nestedPath(path, "run_id"))
	}
	return r.Verdict.Validate(nestedPath(path, "verdict"))
}

func runReviewFromRequest(reviewID string, req RunReviewRequest, now time.Time) RunReview {
	return RunReview{
		ReviewID:       strings.TrimSpace(reviewID),
		TaskID:         req.TaskID,
		RunID:          req.RunID,
		ParentReviewID: req.ParentReviewID,
		Policy:         req.Policy,
		ReviewRound:    req.ReviewRound,
		Attempt:        req.Attempt,
		Status:         RunReviewStatusRequested,
		Reason:         req.Reason,
		MissingWork:    defaultMissingWorkJSON,
		RequestedAt:    now,
		DeadlineAt:     req.DeadlineAt,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func runReviewBindingFromReview(review RunReview) RunReviewBinding {
	return RunReviewBinding{
		Review:            cloneRunReview(&review),
		SessionID:         review.ReviewerSessionID,
		ReviewerAgentName: review.ReviewerAgentName,
		ReviewerPeerID:    review.ReviewerPeerID,
		ReviewerChannelID: review.ReviewerChannelID,
	}
}

func cloneRunReview(review *RunReview) RunReview {
	if review == nil {
		return RunReview{}
	}
	cloned := *review
	cloned.MissingWork = cloneRawJSON(review.MissingWork)
	cloned.ReviewedBy = cloneActorIdentity(review.ReviewedBy)
	if review.Confidence != nil {
		confidence := *review.Confidence
		cloned.Confidence = &confidence
	}
	return cloned
}

func normalizeRunReviewTimes(review RunReview, now time.Time) RunReview {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	if review.RequestedAt.IsZero() {
		review.RequestedAt = now
	} else {
		review.RequestedAt = review.RequestedAt.UTC()
	}
	if !review.RoutedAt.IsZero() {
		review.RoutedAt = review.RoutedAt.UTC()
	}
	if !review.StartedAt.IsZero() {
		review.StartedAt = review.StartedAt.UTC()
	}
	if !review.ReviewedAt.IsZero() {
		review.ReviewedAt = review.ReviewedAt.UTC()
	}
	if !review.DeadlineAt.IsZero() {
		review.DeadlineAt = review.DeadlineAt.UTC()
	}
	if review.CreatedAt.IsZero() {
		review.CreatedAt = now
	} else {
		review.CreatedAt = review.CreatedAt.UTC()
	}
	if review.UpdatedAt.IsZero() {
		review.UpdatedAt = now
	} else {
		review.UpdatedAt = review.UpdatedAt.UTC()
	}
	return review
}

func normalizeReviewMissingWork(raw json.RawMessage) json.RawMessage {
	normalized := normalizeRawJSON(raw)
	if len(normalized) == 0 {
		return cloneRawJSON(defaultMissingWorkJSON)
	}
	return normalized
}

func validateRunReviewMissingWork(raw json.RawMessage) error {
	if err := ValidatePayloadSize(raw, "task_run_review.missing_work"); err != nil {
		return err
	}
	var decoded []json.RawMessage
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return fmt.Errorf("%w: task_run_review.missing_work must be a JSON array", ErrValidation)
	}
	return nil
}

func decodeVerdictMissingWork(raw json.RawMessage, path string) ([]string, error) {
	if err := ValidatePayloadSize(raw, path); err != nil {
		return nil, err
	}
	var decoded []string
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("%w: %s must be a JSON array of strings", ErrValidation, path)
	}
	items := make([]string, 0, len(decoded))
	for idx, item := range decoded {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: %s[%d] must not be empty", ErrValidation, path, idx)
		}
		items = append(items, trimmed)
	}
	return items, nil
}

func validateBoundedReviewText(value string, maxBytes int, path string) error {
	if len(value) > maxBytes {
		return fmt.Errorf("%w: %s exceeds %d bytes", ErrValidation, path, maxBytes)
	}
	return nil
}

func validateRunReviewSelectors(review *RunReview) error {
	return validateRunReviewSelectorFields(
		map[string]string{
			"reviewer_session_id": review.ReviewerSessionID,
			"reviewer_agent_name": review.ReviewerAgentName,
			"reviewer_peer_id":    review.ReviewerPeerID,
			"reviewer_channel_id": review.ReviewerChannelID,
			"parent_review_id":    review.ParentReviewID,
			"delivery_id":         review.DeliveryID,
		},
		"task_run_review",
	)
}

func validateRunReviewSelectorFields(values map[string]string, path string) error {
	for field, value := range values {
		if len(value) > maxRunReviewSelectorFieldBytes {
			return fmt.Errorf(
				"%w: %s exceeds %d bytes",
				ErrValidation,
				nestedPath(path, field),
				maxRunReviewSelectorFieldBytes,
			)
		}
	}
	return nil
}

func IsTerminalRunStatus(status RunStatus) bool {
	switch status.Normalize() {
	case TaskRunStatusCompleted, TaskRunStatusFailed, TaskRunStatusCanceled:
		return true
	default:
		return false
	}
}
