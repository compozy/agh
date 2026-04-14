package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

const (
	// WebhookTimestampHeader is the required HTTP header that carries the
	// signed webhook timestamp.
	WebhookTimestampHeader = "X-AGH-Webhook-Timestamp"
	// WebhookSignatureHeader is the required HTTP header that carries the HMAC
	// signature for webhook delivery.
	WebhookSignatureHeader = "X-AGH-Webhook-Signature"
	// WebhookDeliveryIDHeader identifies one webhook delivery so replayed
	// requests can be rejected inside the trigger engine.
	WebhookDeliveryIDHeader = "X-AGH-Webhook-Delivery-ID"

	maxWebhookPayloadSize = 1 << 20
)

// ListAutomationJobs returns the filtered automation job list.
func (h *BaseHandlers) ListAutomationJobs(c *gin.Context) {
	manager, ok := h.requireAutomationManager(c)
	if !ok {
		return
	}

	query, err := ParseAutomationJobListQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, NewAutomationValidationError(err))
		return
	}

	jobs, err := manager.ListJobs(c.Request.Context(), query)
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}

	nextRunByID, err := h.automationNextRunByJobID(c.Request.Context(), manager)
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.JobsResponse{Jobs: JobPayloadsFromJobs(jobs, nextRunByID)})
}

// CreateAutomationJob stores a new dynamic automation job.
func (h *BaseHandlers) CreateAutomationJob(c *gin.Context) {
	manager, ok := h.requireAutomationManager(c)
	if !ok {
		return
	}

	var req contract.CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, NewAutomationValidationError(fmt.Errorf("%s: decode automation job create request: %w", h.transportName(), err)))
		return
	}

	job := jobFromCreateRequest(req)
	if err := job.Validate("job"); err != nil {
		h.respondError(c, http.StatusBadRequest, NewAutomationValidationError(err))
		return
	}

	created, err := manager.CreateJob(c.Request.Context(), job)
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}

	nextRunByID := h.automationNextRunByJobIDBestEffort(c.Request.Context(), manager, "create_job")

	c.JSON(http.StatusCreated, contract.JobResponse{Job: JobPayloadFromJob(created, timePointerFromMap(nextRunByID, created.ID))})
}

// GetAutomationJob returns one automation job by id.
func (h *BaseHandlers) GetAutomationJob(c *gin.Context) {
	manager, ok := h.requireAutomationManager(c)
	if !ok {
		return
	}

	job, err := manager.GetJob(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}

	nextRunByID, err := h.automationNextRunByJobID(c.Request.Context(), manager)
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.JobResponse{Job: JobPayloadFromJob(job, timePointerFromMap(nextRunByID, job.ID))})
}

// UpdateAutomationJob patches one automation job definition.
func (h *BaseHandlers) UpdateAutomationJob(c *gin.Context) {
	manager, ok := h.requireAutomationManager(c)
	if !ok {
		return
	}

	var req contract.UpdateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, NewAutomationValidationError(fmt.Errorf("%s: decode automation job update request: %w", h.transportName(), err)))
		return
	}
	if !req.HasChanges() {
		h.respondError(c, http.StatusBadRequest, NewAutomationValidationError(errors.New("automation job update must include at least one field")))
		return
	}

	current, err := manager.GetJob(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}

	var updated automationpkg.Job
	switch current.Source {
	case automationpkg.JobSourceConfig:
		if err := validateConfigJobUpdate(req); err != nil {
			h.respondError(c, http.StatusBadRequest, NewAutomationValidationError(err))
			return
		}
		updated, err = manager.SetJobEnabled(c.Request.Context(), current.ID, *req.Enabled)
	default:
		next := applyJobPatch(current, req)
		if err := next.Validate("job"); err != nil {
			h.respondError(c, http.StatusBadRequest, NewAutomationValidationError(err))
			return
		}
		updated, err = manager.UpdateJob(c.Request.Context(), next)
	}
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}

	nextRunByID := h.automationNextRunByJobIDBestEffort(c.Request.Context(), manager, "update_job")

	c.JSON(http.StatusOK, contract.JobResponse{Job: JobPayloadFromJob(updated, timePointerFromMap(nextRunByID, updated.ID))})
}

// DeleteAutomationJob removes one dynamic automation job definition.
func (h *BaseHandlers) DeleteAutomationJob(c *gin.Context) {
	manager, ok := h.requireAutomationManager(c)
	if !ok {
		return
	}

	current, err := manager.GetJob(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}
	if current.Source == automationpkg.JobSourceConfig {
		h.respondError(c, http.StatusBadRequest, NewAutomationValidationError(errors.New("config-backed automation jobs cannot be deleted")))
		return
	}

	if err := manager.DeleteJob(c.Request.Context(), current.ID); err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}
	c.Status(http.StatusNoContent)
}

// TriggerAutomationJob forces one immediate manual automation run.
func (h *BaseHandlers) TriggerAutomationJob(c *gin.Context) {
	manager, ok := h.requireAutomationManager(c)
	if !ok {
		return
	}

	run, err := manager.TriggerJob(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.RunResponse{Run: RunPayloadFromRun(run)})
}

// AutomationJobRuns returns run history for one automation job.
func (h *BaseHandlers) AutomationJobRuns(c *gin.Context) {
	manager, ok := h.requireAutomationManager(c)
	if !ok {
		return
	}

	job, err := manager.GetJob(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}

	query, err := ParseAutomationRunQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, NewAutomationValidationError(err))
		return
	}
	query.JobID = job.ID
	query.TriggerID = ""

	runs, err := manager.ListRuns(c.Request.Context(), query)
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.RunsResponse{Runs: RunPayloadsFromRuns(runs)})
}

// ListAutomationTriggers returns the filtered automation trigger list.
func (h *BaseHandlers) ListAutomationTriggers(c *gin.Context) {
	manager, ok := h.requireAutomationManager(c)
	if !ok {
		return
	}

	query, err := ParseAutomationTriggerListQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, NewAutomationValidationError(err))
		return
	}

	triggers, err := manager.ListTriggers(c.Request.Context(), query)
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TriggersResponse{Triggers: TriggerPayloadsFromTriggers(triggers)})
}

// CreateAutomationTrigger stores a new dynamic automation trigger definition.
func (h *BaseHandlers) CreateAutomationTrigger(c *gin.Context) {
	manager, ok := h.requireAutomationManager(c)
	if !ok {
		return
	}

	var req contract.CreateTriggerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, NewAutomationValidationError(fmt.Errorf("%s: decode automation trigger create request: %w", h.transportName(), err)))
		return
	}

	trigger := triggerFromCreateRequest(req)
	if err := trigger.Validate("trigger"); err != nil {
		h.respondError(c, http.StatusBadRequest, NewAutomationValidationError(err))
		return
	}

	created, err := manager.CreateTrigger(c.Request.Context(), trigger, req.WebhookSecret)
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}

	c.JSON(http.StatusCreated, contract.TriggerResponse{Trigger: TriggerPayloadFromTrigger(created)})
}

// GetAutomationTrigger returns one automation trigger by id.
func (h *BaseHandlers) GetAutomationTrigger(c *gin.Context) {
	manager, ok := h.requireAutomationManager(c)
	if !ok {
		return
	}

	trigger, err := manager.GetTrigger(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.TriggerResponse{Trigger: TriggerPayloadFromTrigger(trigger)})
}

// UpdateAutomationTrigger patches one automation trigger definition.
func (h *BaseHandlers) UpdateAutomationTrigger(c *gin.Context) {
	manager, ok := h.requireAutomationManager(c)
	if !ok {
		return
	}

	var req contract.UpdateTriggerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, NewAutomationValidationError(fmt.Errorf("%s: decode automation trigger update request: %w", h.transportName(), err)))
		return
	}
	if !req.HasChanges() {
		h.respondError(c, http.StatusBadRequest, NewAutomationValidationError(errors.New("automation trigger update must include at least one field")))
		return
	}

	current, err := manager.GetTrigger(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}

	var updated automationpkg.Trigger
	switch current.Source {
	case automationpkg.JobSourceConfig:
		if err := validateConfigTriggerUpdate(req); err != nil {
			h.respondError(c, http.StatusBadRequest, NewAutomationValidationError(err))
			return
		}
		updated, err = manager.SetTriggerEnabled(c.Request.Context(), current.ID, *req.Enabled)
	default:
		next := applyTriggerPatch(current, req)
		if err := next.Validate("trigger"); err != nil {
			h.respondError(c, http.StatusBadRequest, NewAutomationValidationError(err))
			return
		}
		updated, err = manager.UpdateTrigger(c.Request.Context(), next, req.WebhookSecret)
	}
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TriggerResponse{Trigger: TriggerPayloadFromTrigger(updated)})
}

// DeleteAutomationTrigger removes one dynamic automation trigger definition.
func (h *BaseHandlers) DeleteAutomationTrigger(c *gin.Context) {
	manager, ok := h.requireAutomationManager(c)
	if !ok {
		return
	}

	current, err := manager.GetTrigger(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}
	if current.Source == automationpkg.JobSourceConfig {
		h.respondError(c, http.StatusBadRequest, NewAutomationValidationError(errors.New("config-backed automation triggers cannot be deleted")))
		return
	}

	if err := manager.DeleteTrigger(c.Request.Context(), current.ID); err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}
	c.Status(http.StatusNoContent)
}

// AutomationTriggerRuns returns run history for one automation trigger.
func (h *BaseHandlers) AutomationTriggerRuns(c *gin.Context) {
	manager, ok := h.requireAutomationManager(c)
	if !ok {
		return
	}

	trigger, err := manager.GetTrigger(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}

	query, err := ParseAutomationRunQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, NewAutomationValidationError(err))
		return
	}
	query.JobID = ""
	query.TriggerID = trigger.ID

	runs, err := manager.ListRuns(c.Request.Context(), query)
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.RunsResponse{Runs: RunPayloadsFromRuns(runs)})
}

// ListAutomationRuns returns filtered automation run history.
func (h *BaseHandlers) ListAutomationRuns(c *gin.Context) {
	manager, ok := h.requireAutomationManager(c)
	if !ok {
		return
	}

	query, err := ParseAutomationRunQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, NewAutomationValidationError(err))
		return
	}

	runs, err := manager.ListRuns(c.Request.Context(), query)
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.RunsResponse{Runs: RunPayloadsFromRuns(runs)})
}

// GetAutomationRun returns one automation run by id.
func (h *BaseHandlers) GetAutomationRun(c *gin.Context) {
	manager, ok := h.requireAutomationManager(c)
	if !ok {
		return
	}

	run, err := manager.GetRun(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.RunResponse{Run: RunPayloadFromRun(run)})
}

// DeliverGlobalWebhook handles external webhook delivery for global triggers.
func (h *BaseHandlers) DeliverGlobalWebhook(c *gin.Context) {
	h.deliverWebhook(c, automationpkg.AutomationScopeGlobal)
}

// DeliverWorkspaceWebhook handles external webhook delivery for workspace-scoped triggers.
func (h *BaseHandlers) DeliverWorkspaceWebhook(c *gin.Context) {
	h.deliverWebhook(c, automationpkg.AutomationScopeWorkspace)
}

func (h *BaseHandlers) deliverWebhook(c *gin.Context, scope automationpkg.AutomationScope) {
	manager, ok := h.requireAutomationManager(c)
	if !ok {
		return
	}

	request, err := webhookRequestFromHTTP(c, scope)
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}

	result, err := manager.HandleWebhook(c.Request.Context(), request)
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.WebhookDeliveryResponse{Result: WebhookDeliveryPayloadFromResult(result)})
}

func (h *BaseHandlers) requireAutomationManager(c *gin.Context) (AutomationManager, bool) {
	if h.Automation == nil {
		h.respondError(c, http.StatusServiceUnavailable, fmt.Errorf("%s: automation manager is not configured", h.transportName()))
		return nil, false
	}
	return h.Automation, true
}

func (h *BaseHandlers) automationNextRunByJobID(ctx context.Context, manager AutomationManager) (map[string]*time.Time, error) {
	if manager == nil {
		return nil, nil
	}

	status, err := manager.Status(ctx)
	if err != nil {
		return nil, err
	}

	nextRunByID := make(map[string]*time.Time, len(status.ScheduledJobs))
	for _, scheduled := range status.ScheduledJobs {
		if scheduled.NextRun == nil {
			continue
		}
		next := scheduled.NextRun.UTC()
		nextRunByID[scheduled.JobID] = &next
	}
	return nextRunByID, nil
}

func (h *BaseHandlers) automationNextRunByJobIDBestEffort(ctx context.Context, manager AutomationManager, operation string) map[string]*time.Time {
	nextRunByID, err := h.automationNextRunByJobID(ctx, manager)
	if err == nil {
		return nextRunByID
	}

	h.Logger.Warn(
		"api: automation next_run enrichment failed",
		"transport", h.transportName(),
		"operation", strings.TrimSpace(operation),
		"error", err,
	)
	return nil
}

func (h *BaseHandlers) automationHealth(ctx context.Context) (contract.AutomationHealthPayload, error) {
	if h.Automation == nil {
		return contract.AutomationHealthPayload{
			Enabled:          false,
			Jobs:             contract.AutomationResourceStatusPayload{},
			Triggers:         contract.AutomationResourceStatusPayload{},
			SchedulerRunning: false,
		}, nil
	}

	status, err := h.Automation.Status(ctx)
	if err != nil {
		return contract.AutomationHealthPayload{}, err
	}
	return AutomationHealthPayloadFromStatus(h.Config.Automation.Enabled, status), nil
}

// ParseAutomationJobListQuery parses the shared automation job list filters.
func ParseAutomationJobListQuery(c *gin.Context) (automationpkg.JobListQuery, error) {
	limit, err := ParseOptionalInt(c.Query("limit"))
	if err != nil {
		return automationpkg.JobListQuery{}, err
	}

	query := automationpkg.JobListQuery{
		WorkspaceID: strings.TrimSpace(c.Query("workspace_id")),
		Limit:       limit,
	}
	if rawScope := strings.TrimSpace(c.Query("scope")); rawScope != "" {
		query.Scope = automationpkg.AutomationScope(rawScope)
		if err := query.Scope.Validate("scope"); err != nil {
			return automationpkg.JobListQuery{}, err
		}
	}
	if rawSource := strings.TrimSpace(c.Query("source")); rawSource != "" {
		query.Source = automationpkg.JobSource(rawSource)
		if err := query.Source.Validate("source"); err != nil {
			return automationpkg.JobListQuery{}, err
		}
	}
	return query, nil
}

// ParseAutomationTriggerListQuery parses the shared automation trigger list filters.
func ParseAutomationTriggerListQuery(c *gin.Context) (automationpkg.TriggerListQuery, error) {
	limit, err := ParseOptionalInt(c.Query("limit"))
	if err != nil {
		return automationpkg.TriggerListQuery{}, err
	}

	query := automationpkg.TriggerListQuery{
		WorkspaceID: strings.TrimSpace(c.Query("workspace_id")),
		Event:       strings.TrimSpace(c.Query("event")),
		Limit:       limit,
	}
	if rawScope := strings.TrimSpace(c.Query("scope")); rawScope != "" {
		query.Scope = automationpkg.AutomationScope(rawScope)
		if err := query.Scope.Validate("scope"); err != nil {
			return automationpkg.TriggerListQuery{}, err
		}
	}
	if rawSource := strings.TrimSpace(c.Query("source")); rawSource != "" {
		query.Source = automationpkg.JobSource(rawSource)
		if err := query.Source.Validate("source"); err != nil {
			return automationpkg.TriggerListQuery{}, err
		}
	}
	return query, nil
}

// ParseAutomationRunQuery parses the shared automation run list filters.
func ParseAutomationRunQuery(c *gin.Context) (automationpkg.RunQuery, error) {
	limit, err := ParseOptionalInt(c.Query("limit"))
	if err != nil {
		return automationpkg.RunQuery{}, err
	}
	since, err := ParseOptionalTime(c.Query("since"))
	if err != nil {
		return automationpkg.RunQuery{}, err
	}
	until, err := ParseOptionalTime(c.Query("until"))
	if err != nil {
		return automationpkg.RunQuery{}, err
	}

	query := automationpkg.RunQuery{
		JobID:     strings.TrimSpace(c.Query("job_id")),
		TriggerID: strings.TrimSpace(c.Query("trigger_id")),
		Since:     since,
		Until:     until,
		Limit:     limit,
	}
	if rawStatus := strings.TrimSpace(c.Query("status")); rawStatus != "" {
		query.Status = automationpkg.RunStatus(rawStatus)
		if err := query.Status.Validate("status"); err != nil {
			return automationpkg.RunQuery{}, err
		}
	}
	return query, nil
}

func webhookRequestFromHTTP(c *gin.Context, scope automationpkg.AutomationScope) (automationpkg.WebhookRequest, error) {
	workspaceID := strings.TrimSpace(c.Param("workspace_id"))
	if err := scope.Validate("scope"); err != nil {
		return automationpkg.WebhookRequest{}, NewAutomationValidationError(err)
	}
	if err := automationpkg.ValidateScopeBinding(scope, workspaceID, "webhook", "workspace_id"); err != nil {
		return automationpkg.WebhookRequest{}, NewAutomationValidationError(err)
	}

	endpoint := strings.TrimSpace(c.Param("endpoint"))
	if _, err := automationpkg.ParseWebhookEndpoint(endpoint); err != nil {
		return automationpkg.WebhookRequest{}, NewAutomationValidationError(err)
	}

	timestamp, err := parseWebhookTimestampHeader(c.GetHeader(WebhookTimestampHeader))
	if err != nil {
		return automationpkg.WebhookRequest{}, NewAutomationValidationError(err)
	}

	signature := strings.TrimSpace(c.GetHeader(WebhookSignatureHeader))
	if signature == "" {
		return automationpkg.WebhookRequest{}, NewAutomationValidationError(errors.New("webhook signature header is required"))
	}
	deliveryID := strings.TrimSpace(c.GetHeader(WebhookDeliveryIDHeader))
	if deliveryID == "" {
		return automationpkg.WebhookRequest{}, NewAutomationValidationError(errors.New("webhook delivery id header is required"))
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxWebhookPayloadSize)
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return automationpkg.WebhookRequest{}, NewAutomationValidationError(fmt.Errorf("webhook request body exceeds %d bytes: %w", maxWebhookPayloadSize, err))
		}
		return automationpkg.WebhookRequest{}, fmt.Errorf("%s: read webhook request body: %w", c.FullPath(), err)
	}

	request := automationpkg.WebhookRequest{
		Scope:       scope,
		WorkspaceID: workspaceID,
		Endpoint:    endpoint,
		DeliveryID:  deliveryID,
		Timestamp:   timestamp,
		Signature:   signature,
		Payload:     payload,
		Data:        decodeWebhookPayloadData(payload),
	}
	if err := request.Validate("webhook"); err != nil {
		return automationpkg.WebhookRequest{}, NewAutomationValidationError(err)
	}
	return request, nil
}

func parseWebhookTimestampHeader(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, errors.New("webhook timestamp header is required")
	}

	if parsed, err := ParseOptionalTime(value); err == nil {
		return parsed, nil
	}

	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid webhook timestamp %q", value)
	}
	return time.Unix(seconds, 0).UTC(), nil
}

func decodeWebhookPayloadData(payload []byte) map[string]any {
	trimmed := strings.TrimSpace(string(payload))
	if trimmed == "" {
		return nil
	}

	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil
	}
	if len(data) == 0 {
		return nil
	}
	return data
}

func jobFromCreateRequest(req contract.CreateJobRequest) automationpkg.Job {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	retry := automationpkg.DefaultRetryConfig()
	if req.Retry != nil {
		retry = *req.Retry
	}

	fireLimit := automationpkg.DefaultFireLimitConfig()
	if req.FireLimit != nil {
		fireLimit = *req.FireLimit
	}

	schedule := req.Schedule
	taskConfig := cloneAutomationJobTaskConfig(req.Task)
	return automationpkg.Job{
		Scope:       req.Scope,
		Name:        strings.TrimSpace(req.Name),
		AgentName:   strings.TrimSpace(req.AgentName),
		WorkspaceID: strings.TrimSpace(req.WorkspaceID),
		Prompt:      strings.TrimSpace(req.Prompt),
		Schedule:    &schedule,
		Task:        taskConfig,
		Enabled:     enabled,
		Retry:       retry,
		FireLimit:   fireLimit,
		Source:      automationpkg.JobSourceDynamic,
	}
}

func applyJobPatch(current automationpkg.Job, req contract.UpdateJobRequest) automationpkg.Job {
	next := current
	if req.Name != nil {
		next.Name = strings.TrimSpace(*req.Name)
	}
	if req.AgentName != nil {
		next.AgentName = strings.TrimSpace(*req.AgentName)
	}
	if req.WorkspaceID != nil {
		next.WorkspaceID = strings.TrimSpace(*req.WorkspaceID)
	}
	if req.Prompt != nil {
		next.Prompt = strings.TrimSpace(*req.Prompt)
	}
	if req.Schedule != nil {
		schedule := *req.Schedule
		next.Schedule = &schedule
	}
	if req.Task != nil {
		next.Task = cloneAutomationJobTaskConfig(req.Task)
	}
	if req.Enabled != nil {
		next.Enabled = *req.Enabled
	}
	if req.Retry != nil {
		next.Retry = *req.Retry
	}
	if req.FireLimit != nil {
		next.FireLimit = *req.FireLimit
	}
	return next
}

func validateConfigJobUpdate(req contract.UpdateJobRequest) error {
	switch {
	case req.Enabled == nil:
		return errors.New("config-backed automation jobs only accept enabled updates")
	case req.Name != nil || req.AgentName != nil || req.WorkspaceID != nil || req.Prompt != nil || req.Schedule != nil || req.Task != nil || req.Retry != nil || req.FireLimit != nil:
		return errors.New("config-backed automation jobs only accept enabled updates")
	default:
		return nil
	}
}

func cloneAutomationJobTaskConfig(config *automationpkg.JobTaskConfig) *automationpkg.JobTaskConfig {
	if config == nil {
		return nil
	}
	cloned := *config
	cloned.Title = strings.TrimSpace(cloned.Title)
	cloned.Description = strings.TrimSpace(cloned.Description)
	cloned.NetworkChannel = strings.TrimSpace(cloned.NetworkChannel)
	if config.Owner != nil {
		owner := *config.Owner
		owner.Kind = taskpkg.OwnerKind(strings.TrimSpace(string(owner.Kind)))
		owner.Ref = strings.TrimSpace(owner.Ref)
		cloned.Owner = &owner
	}
	return &cloned
}

func triggerFromCreateRequest(req contract.CreateTriggerRequest) automationpkg.Trigger {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	retry := automationpkg.DefaultRetryConfig()
	if req.Retry != nil {
		retry = *req.Retry
	}

	fireLimit := automationpkg.DefaultFireLimitConfig()
	if req.FireLimit != nil {
		fireLimit = *req.FireLimit
	}

	return automationpkg.Trigger{
		Scope:        req.Scope,
		Name:         strings.TrimSpace(req.Name),
		AgentName:    strings.TrimSpace(req.AgentName),
		WorkspaceID:  strings.TrimSpace(req.WorkspaceID),
		Prompt:       strings.TrimSpace(req.Prompt),
		Event:        strings.TrimSpace(req.Event),
		Filter:       cloneAutomationFilter(req.Filter),
		Enabled:      enabled,
		Retry:        retry,
		FireLimit:    fireLimit,
		Source:       automationpkg.JobSourceDynamic,
		WebhookID:    strings.TrimSpace(req.WebhookID),
		EndpointSlug: strings.TrimSpace(req.EndpointSlug),
	}
}

func applyTriggerPatch(current automationpkg.Trigger, req contract.UpdateTriggerRequest) automationpkg.Trigger {
	next := current
	if req.Name != nil {
		next.Name = strings.TrimSpace(*req.Name)
	}
	if req.AgentName != nil {
		next.AgentName = strings.TrimSpace(*req.AgentName)
	}
	if req.WorkspaceID != nil {
		next.WorkspaceID = strings.TrimSpace(*req.WorkspaceID)
	}
	if req.Prompt != nil {
		next.Prompt = strings.TrimSpace(*req.Prompt)
	}
	if req.Event != nil {
		next.Event = strings.TrimSpace(*req.Event)
	}
	if req.Filter != nil {
		next.Filter = cloneAutomationFilter(req.Filter)
	}
	if req.Enabled != nil {
		next.Enabled = *req.Enabled
	}
	if req.Retry != nil {
		next.Retry = *req.Retry
	}
	if req.FireLimit != nil {
		next.FireLimit = *req.FireLimit
	}

	event := strings.TrimSpace(next.Event)
	if req.WebhookID != nil {
		next.WebhookID = strings.TrimSpace(*req.WebhookID)
	} else if !strings.EqualFold(event, "webhook") {
		next.WebhookID = ""
	}
	if req.EndpointSlug != nil {
		next.EndpointSlug = strings.TrimSpace(*req.EndpointSlug)
	} else if !strings.EqualFold(event, "webhook") {
		next.EndpointSlug = ""
	}

	return next
}

func validateConfigTriggerUpdate(req contract.UpdateTriggerRequest) error {
	switch {
	case req.Enabled == nil:
		return errors.New("config-backed automation triggers only accept enabled updates")
	case req.Name != nil || req.AgentName != nil || req.WorkspaceID != nil || req.Prompt != nil || req.Event != nil || req.Filter != nil || req.Retry != nil || req.FireLimit != nil || req.WebhookID != nil || req.EndpointSlug != nil || req.WebhookSecret != nil:
		return errors.New("config-backed automation triggers only accept enabled updates")
	default:
		return nil
	}
}

func cloneAutomationFilter(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}
