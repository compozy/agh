package core

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

// PauseTask marks one task as paused for future claims.
func (h *BaseHandlers) PauseTask(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}
	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	var req contract.PauseTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode pause task request: %w", h.transportName(), err)),
		)
		return
	}
	actor, err := h.taskActorContext(c, taskActionPauseTask)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	taskRecord, err := manager.PauseTask(
		c.Request.Context(),
		taskID,
		taskpkg.PauseTaskRequest{Reason: req.Reason, Metadata: req.Metadata},
		actor,
	)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.TaskResponse{Task: TaskPayloadFromTask(taskRecord)})
}

// ResumeTask clears one task pause for future claims.
func (h *BaseHandlers) ResumeTask(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}
	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	var req contract.ResumeTaskRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode resume task request: %w", h.transportName(), err)),
		)
		return
	}
	actor, err := h.taskActorContext(c, taskActionResumeTask)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	taskRecord, err := manager.ResumeTask(
		c.Request.Context(),
		taskID,
		taskpkg.ResumeTaskRequest{Metadata: req.Metadata},
		actor,
	)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.TaskResponse{Task: TaskPayloadFromTask(taskRecord)})
}

// GetScheduler returns scheduler-wide pause state and queue pressure.
func (h *BaseHandlers) GetScheduler(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}
	actor, err := h.taskActorContext(c, taskActionSchedulerStatus)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	status, err := manager.SchedulerStatus(c.Request.Context(), actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.SchedulerStatusResponse{Scheduler: SchedulerStatusPayloadFromDomain(status)})
}

// PauseScheduler marks the scheduler as paused for new dispatch and claims.
func (h *BaseHandlers) PauseScheduler(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}
	var req contract.SchedulerPauseRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode scheduler pause request: %w", h.transportName(), err)),
		)
		return
	}
	actor, err := h.taskActorContext(c, taskActionSchedulerPause)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	status, err := manager.PauseScheduler(
		c.Request.Context(),
		taskpkg.SchedulerPauseRequest{Reason: req.Reason},
		actor,
	)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.SchedulerStatusResponse{Scheduler: SchedulerStatusPayloadFromDomain(status)})
}

// ResumeScheduler clears scheduler-wide pause state.
func (h *BaseHandlers) ResumeScheduler(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}
	var req contract.SchedulerResumeRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode scheduler resume request: %w", h.transportName(), err)),
		)
		return
	}
	actor, err := h.taskActorContext(c, taskActionSchedulerResume)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	status, err := manager.ResumeScheduler(
		c.Request.Context(),
		taskpkg.SchedulerResumeRequest{Reason: req.Reason},
		actor,
	)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.SchedulerStatusResponse{Scheduler: SchedulerStatusPayloadFromDomain(status)})
}

// DrainScheduler pauses the scheduler and waits for active claims to drain.
func (h *BaseHandlers) DrainScheduler(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}
	var req contract.SchedulerDrainRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode scheduler drain request: %w", h.transportName(), err)),
		)
		return
	}
	timeout := 60 * time.Second
	if req.TimeoutSeconds != nil {
		if *req.TimeoutSeconds < 0 {
			h.respondError(
				c,
				http.StatusBadRequest,
				NewTaskValidationError(fmt.Errorf("scheduler timeout_seconds must be non-negative")),
			)
			return
		}
		timeout = time.Duration(*req.TimeoutSeconds) * time.Second
	}
	actor, err := h.taskActorContext(c, taskActionSchedulerDrain)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	result, err := manager.DrainScheduler(
		c.Request.Context(),
		taskpkg.SchedulerDrainRequest{
			Reason:  req.Reason,
			Timeout: timeout,
		},
		actor,
	)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	c.JSON(http.StatusOK, SchedulerDrainResponseFromDomain(result))
}

// GetSchedulerBacklog returns queued scheduler backlog rows.
func (h *BaseHandlers) GetSchedulerBacklog(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}
	var req contract.SchedulerBacklogQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode scheduler backlog query: %w", h.transportName(), err)),
		)
		return
	}
	actor, err := h.taskActorContext(c, taskActionSchedulerBacklog)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	backlog, err := manager.SchedulerBacklog(c.Request.Context(), contract.SchedulerBacklogDomainQuery(req), actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.SchedulerBacklogResponse{Backlog: SchedulerBacklogPayloadFromDomain(backlog)})
}

// SchedulerStatusPayloadFromDomain converts scheduler status into the shared payload.
func SchedulerStatusPayloadFromDomain(status taskpkg.SchedulerStatus) contract.SchedulerStatusPayload {
	return contract.SchedulerStatusPayload{
		Paused:           status.Paused,
		PausedBy:         status.PausedBy,
		PausedAt:         optionalTime(status.PausedAt),
		PausedReason:     status.PausedReason,
		ActiveClaimCount: status.ActiveClaimCount,
		QueuedRunCount:   status.QueuedRunCount,
		PausedTaskCount:  status.PausedTaskCount,
		DrainInProgress:  status.DrainInProgress,
		DrainStartedAt:   optionalTime(status.DrainStartedAt),
		AsOf:             status.AsOf,
	}
}

// SchedulerDrainResponseFromDomain converts one drain result into the shared response payload.
func SchedulerDrainResponseFromDomain(result taskpkg.SchedulerDrainResult) contract.SchedulerDrainResponse {
	return contract.SchedulerDrainResponse{
		Scheduler:       SchedulerStatusPayloadFromDomain(result.Status),
		Completed:       result.Completed,
		TimedOut:        result.TimedOut,
		RemainingClaims: result.RemainingClaims,
		StartedAt:       result.StartedAt,
		CompletedAt:     result.CompletedAt,
	}
}

// SchedulerBacklogPayloadFromDomain converts queued scheduler backlog rows into the shared payload.
func SchedulerBacklogPayloadFromDomain(backlog taskpkg.SchedulerBacklog) contract.SchedulerBacklogPayload {
	runs := make([]contract.SchedulerBacklogRunPayload, 0, len(backlog.Runs))
	for idx := range backlog.Runs {
		item := &backlog.Runs[idx]
		taskPayload := TaskSummaryPayloadFromTask(&item.Task)
		taskPayload.EffectivePaused = item.EffectivePaused
		taskPayload.PausedByTaskID = item.PausedByTaskID
		runs = append(runs, contract.SchedulerBacklogRunPayload{
			Task: taskPayload,
			Run:  TaskRunPayloadFromRun(&item.Run),
		})
	}
	return contract.SchedulerBacklogPayload{Runs: runs, Total: backlog.Total}
}

// TaskSummaryPayloadFromTask converts one durable task into a summary-shaped payload.
func TaskSummaryPayloadFromTask(record *taskpkg.Task) contract.TaskSummaryPayload {
	if record == nil {
		return contract.TaskSummaryPayload{}
	}
	return contract.TaskSummaryPayload{
		ID:              record.ID,
		Identifier:      record.Identifier,
		Scope:           record.Scope,
		WorkspaceID:     record.WorkspaceID,
		ParentTaskID:    record.ParentTaskID,
		NetworkChannel:  record.NetworkChannel,
		Title:           record.Title,
		Priority:        record.Priority,
		MaxAttempts:     record.MaxAttempts,
		Status:          record.Status,
		ApprovalPolicy:  record.ApprovalPolicy,
		ApprovalState:   record.ApprovalState,
		Draft:           record.Status.Normalize() == taskpkg.TaskStatusDraft,
		Owner:           cloneOwnership(record.Owner),
		CurrentRunID:    record.CurrentRunID,
		LatestEventSeq:  record.LatestEventSeq,
		Paused:          record.Paused,
		PausedBy:        record.PausedBy,
		PausedAt:        optionalTime(record.PausedAt),
		PausedReason:    record.PausedReason,
		EffectivePaused: record.Paused,
		PausedByTaskID: func() string {
			if record.Paused {
				return record.ID
			}
			return ""
		}(),
		CreatedBy:      record.CreatedBy,
		Origin:         record.Origin,
		CreatedAt:      record.CreatedAt,
		UpdatedAt:      record.UpdatedAt,
		ClosedAt:       optionalTime(record.ClosedAt),
		LastActivityAt: optionalTime(record.UpdatedAt),
	}
}
