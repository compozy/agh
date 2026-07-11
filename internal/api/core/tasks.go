package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/network"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
	"github.com/gin-gonic/gin"
)

const (
	defaultTaskActorRef              = "local-user"
	taskDesignationRollupDetailLimit = 20
	taskDesignationRollupCompleted   = "completed"
	taskDesignationRollupCanceled    = "canceled"
	// #nosec G101 -- This is an HTTP header name, not a credential value.
	taskClaimTokenHeader       = "X-AGH-Claim-Token"
	taskActionCreate           = "create"
	taskActionGet              = "get"
	taskActionInspect          = "inspect"
	taskActionDelete           = "delete"
	taskActionPublish          = "publish"
	taskActionStart            = "start"
	taskActionUpdate           = "update"
	taskActionCancel           = "cancel"
	taskActionBlock            = "block"
	taskActionListBlocks       = "list_blocks"
	taskActionClearBlock       = "clear_block"
	taskActionRecover          = "recover"
	taskActionCreateChild      = "create_child"
	taskActionAddDependency    = "add_dependency"
	taskActionRemoveDependency = "remove_dependency"
	taskActionListRuns         = "list_runs"
	taskActionGetRun           = "get_run"
	taskActionEnqueueRun       = "enqueue_run"
	taskActionFanOutRuns       = "fan_out_runs"
	taskActionClaimRun         = "claim_run"
	taskActionStartRun         = "start_run"
	taskActionAttachRun        = "attach_run_session"
	taskActionCompleteRun      = "complete_run"
	taskActionFailRun          = "fail_run"
	taskActionForceReleaseRun  = "force_release_run"
	taskActionForceFailRun     = "force_fail_run"
	taskActionRetryRun         = "retry_run"
	taskActionRecoverRun       = "recover_run"
	taskActionBulkReleaseRuns  = "bulk_release_runs"
	taskActionBulkFailRuns     = "bulk_fail_runs"
	taskActionCancelRun        = "cancel_run"
	taskActionTimeline         = "timeline"
	taskActionStream           = "stream"
	taskActionTree             = "tree"
	taskActionGetProfile       = "get_profile"
	taskActionSetProfile       = "set_profile"
	taskActionDeleteProfile    = "delete_profile"
	taskActionRequestReview    = "request_review"
	taskActionListReviews      = "list_reviews"
	taskActionGetReview        = "get_review"
	taskActionSubmitReview     = "submit_review"
	taskActionCreateBridgeSub  = "create_bridge_notification_subscription"
	taskActionListBridgeSubs   = "list_bridge_notification_subscriptions"
	taskActionGetBridgeSub     = "get_bridge_notification_subscription"
	taskActionDeleteBridgeSub  = "delete_bridge_notification_subscription"
	taskActionPromoteNetwork   = "promote_network_thread"
	taskActionDashboard        = "dashboard"
	taskActionInbox            = "inbox"
	taskActionApprove          = "approve"
	taskActionReject           = "reject"
	taskActionTriageRead       = "triage_read"
	taskActionTriageArchive    = "triage_archive"
	taskActionTriageDismiss    = "triage_dismiss"
	taskActionPauseTask        = "pause_task"
	taskActionResumeTask       = "resume_task"
	taskActionSchedulerStatus  = "scheduler_status"
	taskActionSchedulerPause   = "scheduler_pause"
	taskActionSchedulerResume  = "scheduler_resume"
	taskActionSchedulerDrain   = "scheduler_drain"
	taskActionSchedulerBacklog = "scheduler_backlog"
)

func (h *BaseHandlers) requireTaskManager(c *gin.Context) (TaskService, bool) {
	if h.Tasks == nil {
		h.respondError(
			c,
			http.StatusServiceUnavailable,
			fmt.Errorf("%s: task service is not configured", h.transportName()),
		)
		return nil, false
	}
	return h.Tasks, true
}

func (h *BaseHandlers) requireTaskObserver(c *gin.Context) (Observer, bool) {
	if h.Observer == nil {
		h.respondError(
			c,
			http.StatusServiceUnavailable,
			fmt.Errorf("%s: observe service is not configured", h.transportName()),
		)
		return nil, false
	}
	return h.Observer, true
}

func (h *BaseHandlers) taskActorContext(c *gin.Context, action string) (taskpkg.ActorContext, error) {
	return h.taskActorContextForWorkspace(c, action, "")
}

func (h *BaseHandlers) taskActorContextForWorkspace(
	c *gin.Context,
	action string,
	expectedWorkspaceID string,
) (taskpkg.ActorContext, error) {
	if h.TaskActorContextResolver != nil {
		return h.TaskActorContextResolver(c, action)
	}
	credentials := agentCallerCredentialsFromRequest(c)
	if hasAgentCallerIdentityCredentials(credentials) {
		caller, err := h.resolveAgentCallerForWorkspace(
			c.Request.Context(),
			credentials,
			"tasks."+strings.TrimSpace(action),
			expectedWorkspaceID,
		)
		if err != nil {
			return taskpkg.ActorContext{}, err
		}
		return caller.Actor, nil
	}
	return taskpkg.DeriveHumanActorContext(
		defaultTaskActorRef,
		taskOriginKindForTransport(h.transportName()),
		"tasks."+strings.TrimSpace(action),
	)
}

func taskOriginKindForTransport(name string) taskpkg.OriginKind {
	normalized := strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.Contains(normalized, "uds"):
		return taskpkg.OriginKindUDS
	case strings.Contains(normalized, "web"):
		return taskpkg.OriginKindWeb
	case strings.Contains(normalized, "cli"):
		return taskpkg.OriginKindCLI
	default:
		return taskpkg.OriginKindHTTP
	}
}

// ListTasks returns the filtered task list.
func (h *BaseHandlers) ListTasks(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	actor, err := h.taskActorContext(c, taskActionList)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	transportQuery, err := ParseTaskListQuery(c)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	query, err := h.taskListDomainQuery(c.Request.Context(), transportQuery)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	page, err := manager.ListTaskCatalog(c.Request.Context(), query, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, TaskCatalogResponseFromPage(page))
}

// CreateTask creates one new task.
func (h *BaseHandlers) CreateTask(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	var req contract.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode create task request: %w", h.transportName(), err)),
		)
		return
	}

	spec, err := h.createTaskSpecFromRequest(c.Request.Context(), req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	actor, err := h.taskActorContextForWorkspace(c, taskActionCreate, spec.WorkspaceID)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	record, err := manager.CreateTask(c.Request.Context(), spec, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusCreated, contract.TaskResponse{Task: TaskPayloadFromTask(record)})
}

// GetTask returns one expanded task view.
func (h *BaseHandlers) GetTask(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	actor, err := h.taskActorContext(c, taskActionGet)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	view, err := manager.GetTask(c.Request.Context(), taskID, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	payload, err := h.taskDetailPayload(c.Request.Context(), view)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.TaskDetailResponse{Task: payload})
}

// BlockTask creates one runtime-declared block for a task.
func (h *BaseHandlers) BlockTask(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.CreateTaskBlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode block task request: %w", h.transportName(), err)),
		)
		return
	}

	actor, err := h.taskActorContext(c, taskActionBlock)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	claimToken, err := h.taskBlockClaimToken(c, manager, req)
	if err != nil {
		h.respondError(c, statusForTaskBlockError(err), err)
		return
	}
	blockReq, err := createTaskBlockFromRequest(taskID, req, claimToken)
	if err != nil {
		h.respondError(c, statusForTaskBlockError(err), err)
		return
	}
	block, err := manager.BlockTask(c.Request.Context(), blockReq, actor)
	if err != nil {
		h.respondError(c, statusForTaskBlockError(err), err)
		return
	}

	c.JSON(http.StatusCreated, contract.TaskBlockResponse{Block: TaskBlockPayloadFromBlock(block)})
}

func (h *BaseHandlers) taskBlockClaimToken(
	c *gin.Context,
	manager TaskService,
	req contract.CreateTaskBlockRequest,
) (string, error) {
	claimToken := strings.TrimSpace(c.GetHeader(taskClaimTokenHeader))
	if claimToken != "" || strings.TrimSpace(req.RunID) == "" {
		return claimToken, nil
	}
	credentials := agentCallerCredentialsFromRequest(c)
	if !hasAgentCallerIdentityCredentials(credentials) {
		return "", nil
	}
	caller, err := h.resolveAgentCallerForWorkspace(
		c.Request.Context(),
		credentials,
		"tasks."+taskActionBlock,
		"",
	)
	if err != nil {
		return "", err
	}
	handle, err := h.lookupAgentTaskLease(c.Request.Context(), manager, caller, req.RunID)
	if err != nil {
		return "", err
	}
	return handle.ClaimToken, nil
}

// ListTaskBlocks returns task block rows for one task.
func (h *BaseHandlers) ListTaskBlocks(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	includeCleared, err := parseBoolQuery(c, "include_cleared")
	if err != nil {
		h.respondError(c, http.StatusBadRequest, NewTaskValidationError(err))
		return
	}

	actor, err := h.taskActorContext(c, taskActionListBlocks)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	blocks, err := manager.ListTaskBlocks(c.Request.Context(), taskID, includeCleared, actor)
	if err != nil {
		h.respondError(c, statusForTaskBlockError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskBlocksResponse{Blocks: TaskBlockPayloadsFromBlocks(blocks)})
}

// ClearTaskBlock clears one open task block.
func (h *BaseHandlers) ClearTaskBlock(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	blockID, err := requiredPathID(c.Param("block_id"), "block id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.ClearTaskBlockRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode clear task block request: %w", h.transportName(), err)),
		)
		return
	}

	actor, err := h.taskActorContext(c, taskActionClearBlock)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	block, err := manager.ClearTaskBlock(c.Request.Context(), taskID, blockID, strings.TrimSpace(req.Note), actor)
	if err != nil {
		h.respondError(c, statusForTaskBlockError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskBlockResponse{Block: TaskBlockPayloadFromBlock(block)})
}

// RecoverTask clears task-level needs_attention state.
func (h *BaseHandlers) RecoverTask(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.RecoverTaskRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode recover task request: %w", h.transportName(), err)),
		)
		return
	}

	actor, err := h.taskActorContext(c, taskActionRecover)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	record, err := manager.RecoverTask(c.Request.Context(), taskID, strings.TrimSpace(req.Note), actor)
	if err != nil {
		h.respondError(c, statusForTaskBlockError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskResponse{Task: TaskPayloadFromTask(record)})
}

func (h *BaseHandlers) taskDetailPayload(ctx context.Context, view *taskpkg.View) (contract.TaskDetailPayload, error) {
	payload := TaskDetailPayloadFromView(view)
	if h == nil || h.NetworkStore == nil || view == nil || strings.TrimSpace(view.Task.ID) == "" {
		return payload, nil
	}
	rollups, err := h.NetworkStore.ListTaskDesignationRollups(ctx, store.TaskDesignationRollupQuery{
		TaskID: strings.TrimSpace(view.Task.ID),
		Limit:  taskDesignationRollupDetailLimit,
	})
	if err != nil {
		return contract.TaskDetailPayload{}, fmt.Errorf("api: list task designation rollups: %w", err)
	}
	payload.DesignationRollups = TaskDesignationRollupPayloadsFromStore(rollups)
	return payload, nil
}

// InspectTask returns a diagnostic snapshot for one task.
func (h *BaseHandlers) InspectTask(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	actor, err := h.taskActorContext(c, taskActionInspect)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	view, err := manager.InspectTask(c.Request.Context(), taskID, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskInspectResponse{Inspect: TaskInspectPayloadFromView(view)})
}

// InspectRun returns a diagnostic snapshot rooted at one run.
func (h *BaseHandlers) InspectRun(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	runID, err := requiredPathID(c.Param("id"), "run id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	actor, err := h.taskActorContext(c, taskActionInspect)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	view, err := manager.InspectRun(c.Request.Context(), runID, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskInspectResponse{Inspect: TaskInspectPayloadFromView(view)})
}

// DeleteTask removes one task record and any cascade-owned child rows.
func (h *BaseHandlers) DeleteTask(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	actor, err := h.taskActorContext(c, taskActionDelete)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	if err := manager.DeleteTask(c.Request.Context(), taskID, actor); err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.Status(http.StatusNoContent)
}

// UpdateTask patches one mutable task surface.
func (h *BaseHandlers) UpdateTask(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode update task request: %w", h.transportName(), err)),
		)
		return
	}
	if !req.HasChanges() {
		err := NewTaskValidationError(errors.New("task update must include at least one mutable field"))
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	actor, err := h.taskActorContext(c, taskActionUpdate)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	patch, err := taskPatchFromRequest(req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	record, err := manager.UpdateTask(c.Request.Context(), taskID, patch, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskResponse{Task: TaskPayloadFromTask(record)})
}

// GetTaskExecutionProfile returns one task-owned execution profile.
func (h *BaseHandlers) GetTaskExecutionProfile(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	actor, err := h.taskActorContext(c, taskActionGetProfile)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	profile, err := manager.GetExecutionProfile(c.Request.Context(), taskID, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskExecutionProfileResponse{Profile: profile})
}

// SetTaskExecutionProfile replaces one task-owned execution profile.
func (h *BaseHandlers) SetTaskExecutionProfile(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.SetTaskExecutionProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode task execution profile request: %w", h.transportName(), err)),
		)
		return
	}

	profile, err := taskExecutionProfileFromRequest(taskID, &req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	actor, err := h.taskActorContext(c, taskActionSetProfile)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	stored, err := manager.SetExecutionProfile(c.Request.Context(), taskID, profile, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskExecutionProfileResponse{Profile: stored})
}

// DeleteTaskExecutionProfile removes one persisted task-owned execution profile.
func (h *BaseHandlers) DeleteTaskExecutionProfile(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	actor, err := h.taskActorContext(c, taskActionDeleteProfile)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	if err := manager.DeleteExecutionProfile(c.Request.Context(), taskID, actor); err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.Status(http.StatusNoContent)
}

// PublishTask publishes one draft task into the canonical runnable lifecycle.
func (h *BaseHandlers) PublishTask(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.TaskExecutionRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode publish task request: %w", h.transportName(), err)),
		)
		return
	}

	actor, err := h.taskActorContext(c, taskActionPublish)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	executionReq, err := taskExecutionRequestFromRequest(req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	execution, err := manager.PublishTask(c.Request.Context(), taskID, executionReq, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, TaskExecutionResponseFromExecution(execution))
}

// StartTask explicitly enqueues one executable run for an existing task.
func (h *BaseHandlers) StartTask(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.TaskExecutionRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode start task request: %w", h.transportName(), err)),
		)
		return
	}

	actor, err := h.taskActorContext(c, taskActionStart)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	executionReq, err := taskExecutionRequestFromRequest(req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	execution, err := manager.StartTask(c.Request.Context(), taskID, executionReq, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusCreated, TaskExecutionResponseFromExecution(execution))
}

// CancelTask requests cancellation for one task tree.
func (h *BaseHandlers) CancelTask(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.CancelTaskRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode cancel task request: %w", h.transportName(), err)),
		)
		return
	}

	actor, err := h.taskActorContext(c, taskActionCancel)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	cancelReq, err := cancelTaskFromRequest(req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	record, err := manager.CancelTask(c.Request.Context(), taskID, cancelReq, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskResponse{Task: TaskPayloadFromTask(record)})
}

// CreateChildTask creates one child task beneath the supplied parent.
func (h *BaseHandlers) CreateChildTask(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	parentTaskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.CreateTaskChildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode create child task request: %w", h.transportName(), err)),
		)
		return
	}

	actor, err := h.taskActorContext(c, taskActionCreateChild)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	spec, err := h.createChildTaskSpecFromRequest(c.Request.Context(), req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	record, err := manager.CreateChildTask(c.Request.Context(), parentTaskID, spec, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusCreated, contract.TaskResponse{Task: TaskPayloadFromTask(record)})
}

// AddTaskDependency adds one blocking dependency edge.
func (h *BaseHandlers) AddTaskDependency(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.AddTaskDependencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode add dependency request: %w", h.transportName(), err)),
		)
		return
	}

	actor, err := h.taskActorContext(c, taskActionAddDependency)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	spec, err := addTaskDependencyFromRequest(taskID, req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	if err := manager.AddDependency(c.Request.Context(), spec, actor); err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	view, err := manager.GetTask(c.Request.Context(), taskID, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	payload, err := h.taskDetailPayload(c.Request.Context(), view)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.TaskDetailResponse{Task: payload})
}

// RemoveTaskDependency removes one blocking dependency edge.
func (h *BaseHandlers) RemoveTaskDependency(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	dependsOnID, err := requiredPathID(c.Param("depends_on_id"), "depends_on_id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	actor, err := h.taskActorContext(c, taskActionRemoveDependency)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	if err := manager.RemoveDependency(c.Request.Context(), taskID, dependsOnID, actor); err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	view, err := manager.GetTask(c.Request.Context(), taskID, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	payload, err := h.taskDetailPayload(c.Request.Context(), view)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.TaskDetailResponse{Task: payload})
}

// ListTaskRuns returns the filtered run list for one task.
func (h *BaseHandlers) ListTaskRuns(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	actor, err := h.taskActorContext(c, taskActionListRuns)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	transportQuery, err := ParseTaskRunListQuery(c)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	query, err := taskRunListDomainQuery(transportQuery)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	runs, err := manager.ListTaskRuns(c.Request.Context(), taskID, query, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskRunsResponse{Runs: TaskRunPayloadsFromRuns(runs)})
}

// GetTaskRun returns one run-detail view.
func (h *BaseHandlers) GetTaskRun(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	runID, err := requiredPathID(c.Param("id"), "run id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	actor, err := h.taskActorContext(c, taskActionGetRun)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	view, err := manager.RunDetail(c.Request.Context(), runID, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskRunDetailResponse{Run: TaskRunDetailPayloadFromView(view)})
}

// TaskTimeline returns the task-native live timeline for one task.
func (h *BaseHandlers) TaskTimeline(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	actor, err := h.taskActorContext(c, taskActionTimeline)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	transportQuery, err := ParseTaskTimelineQuery(c)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	query, err := taskTimelineDomainQuery(transportQuery)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	items, err := manager.Timeline(c.Request.Context(), taskID, query, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskTimelineResponse{Timeline: TaskTimelineItemPayloadsFromItems(items)})
}

// StreamTask streams task-native live events over SSE.
func (h *BaseHandlers) StreamTask(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	actor, err := h.taskActorContext(c, taskActionStream)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	transportQuery, err := ParseTaskStreamQuery(c)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	query, err := h.taskStreamDomainQuery(c, transportQuery)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	stream, err := manager.Stream(c.Request.Context(), taskID, query, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	writer, err := PrepareSSE(c)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-h.StreamDoneChannel():
			return
		case event, ok := <-stream:
			if !ok {
				return
			}
			if err := WriteTaskStreamEvent(writer, event); err != nil {
				h.logSSEWriteFailure(event.Type, err)
				return
			}
		}
	}
}

// TaskTree returns one task-tree live view.
func (h *BaseHandlers) TaskTree(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	actor, err := h.taskActorContext(c, taskActionTree)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	view, err := manager.Tree(c.Request.Context(), taskID, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskTreeResponse{Tree: TaskTreePayloadFromView(view)})
}

// TaskDashboard returns the observer-backed task dashboard view.
func (h *BaseHandlers) TaskDashboard(c *gin.Context) {
	observer, ok := h.requireTaskObserver(c)
	if !ok {
		return
	}

	transportQuery, err := ParseTaskDashboardQuery(c)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	query, err := h.taskDashboardDomainQuery(c.Request.Context(), transportQuery)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	view, err := observer.QueryTaskDashboard(c.Request.Context(), query)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskDashboardResponse{Dashboard: TaskDashboardPayloadFromView(&view)})
}

// TaskInbox returns the observer-backed task inbox view.
func (h *BaseHandlers) TaskInbox(c *gin.Context) {
	observer, ok := h.requireTaskObserver(c)
	if !ok {
		return
	}

	actor, err := h.taskActorContext(c, taskActionInbox)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	transportQuery, err := ParseTaskInboxQuery(c)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	query, err := h.taskInboxDomainQuery(c.Request.Context(), transportQuery)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	view, err := observer.QueryTaskInbox(c.Request.Context(), query, actor.Actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskInboxResponse{Inbox: TaskInboxPayloadFromView(view)})
}

// ApproveTask records one approval decision for an approval-gated task.
func (h *BaseHandlers) ApproveTask(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.TaskExecutionRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode approve task request: %w", h.transportName(), err)),
		)
		return
	}

	actor, err := h.taskActorContext(c, taskActionApprove)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	executionReq, err := taskExecutionRequestFromRequest(req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	execution, err := manager.ApproveTask(c.Request.Context(), taskID, executionReq, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusCreated, TaskExecutionResponseFromExecution(execution))
}

// RejectTask records one rejection decision for an approval-gated task.
func (h *BaseHandlers) RejectTask(c *gin.Context) {
	h.mutateTaskApproval(c, taskActionReject, func(
		ctx context.Context,
		manager TaskService,
		taskID string,
		actor taskpkg.ActorContext,
	) (*taskpkg.Task, error) {
		return manager.RejectTask(ctx, taskID, actor)
	})
}

// MarkTaskRead marks one task triage record as read for the current actor.
func (h *BaseHandlers) MarkTaskRead(c *gin.Context) {
	h.mutateTaskTriage(c, taskActionTriageRead, func(
		ctx context.Context,
		manager TaskService,
		taskID string,
		actor taskpkg.ActorContext,
	) (taskpkg.TriageState, error) {
		return manager.MarkTaskRead(ctx, taskID, actor)
	})
}

// ArchiveTask archives one task triage record for the current actor.
func (h *BaseHandlers) ArchiveTask(c *gin.Context) {
	h.mutateTaskTriage(c, taskActionTriageArchive, func(
		ctx context.Context,
		manager TaskService,
		taskID string,
		actor taskpkg.ActorContext,
	) (taskpkg.TriageState, error) {
		return manager.ArchiveTask(ctx, taskID, actor)
	})
}

// DismissTask dismisses one task triage record for the current actor.
func (h *BaseHandlers) DismissTask(c *gin.Context) {
	h.mutateTaskTriage(c, taskActionTriageDismiss, func(
		ctx context.Context,
		manager TaskService,
		taskID string,
		actor taskpkg.ActorContext,
	) (taskpkg.TriageState, error) {
		return manager.DismissTask(ctx, taskID, actor)
	})
}

// EnqueueTaskRun creates one new queue-first run for the supplied task.
func (h *BaseHandlers) EnqueueTaskRun(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.EnqueueTaskRunRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode enqueue run request: %w", h.transportName(), err)),
		)
		return
	}

	actor, err := h.taskActorContext(c, taskActionEnqueueRun)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	spec, err := enqueueTaskRunFromRequest(taskID, req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	run, err := manager.EnqueueRun(c.Request.Context(), spec, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusCreated, contract.TaskRunResponse{Run: TaskRunPayloadFromRun(run)})
}

// FanOutTaskRuns creates designated sibling runs for one task.
func (h *BaseHandlers) FanOutTaskRuns(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}
	networkStore, err := h.networkStoreRequired()
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	var req contract.FanOutTaskRunsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode fan-out task runs request: %w", h.transportName(), err)),
		)
		return
	}
	maxDesignations := h.Config.Task.Orchestration.DesignatedRunMax
	if maxDesignations <= 0 {
		maxDesignations = aghconfig.DefaultTaskDesignatedRunMax
	}
	prepared, err := prepareFanOutTaskRunsRequest(req, maxDesignations)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	actor, err := h.taskActorContext(c, taskActionFanOutRuns)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	groupID := store.NewID("tdg")
	runs, err := enqueueFanOutTaskRuns(c.Request.Context(), manager, actor, taskID, groupID, req, prepared)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	now := h.nowUTC()
	if err := networkStore.PutTaskDesignationRollup(
		c.Request.Context(),
		store.TaskDesignationRollup{
			DesignationGroupID: groupID,
			TaskID:             taskID,
			SummaryJSON:        fanOutDesignationRollupJSON(runs, now),
			CreatedAt:          now,
		},
	); err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	c.JSON(http.StatusCreated, contract.FanOutTaskRunsResponse{
		DesignationGroupID: groupID,
		Runs:               TaskRunPayloadsFromRuns(runs),
	})
}

type preparedFanOutDesignation struct {
	idempotencyKey string
	metadata       json.RawMessage
}

func prepareFanOutTaskRunsRequest(
	req contract.FanOutTaskRunsRequest,
	maxDesignations int,
) ([]preparedFanOutDesignation, error) {
	if len(req.Designations) == 0 {
		return nil, NewTaskValidationError(errors.New("designations are required"))
	}
	if len(req.Designations) > maxDesignations {
		return nil, NewTaskValidationError(fmt.Errorf("designations cannot exceed %d", maxDesignations))
	}
	if err := validateTaskChannel("fan_out_runs.network_channel", req.NetworkChannel); err != nil {
		return nil, err
	}
	prepared := make([]preparedFanOutDesignation, 0, len(req.Designations))
	for index, designation := range req.Designations {
		metadata, err := fanOutDesignationMetadata(index, designation)
		if err != nil {
			return nil, err
		}
		idempotencyKey := fanOutDesignationIdempotencyKey(req.IdempotencyKey, designation, index)
		if idempotencyKey == "" {
			return nil, NewTaskValidationError(fmt.Errorf(
				"designations[%d].idempotency_key is required when fan_out_runs.idempotency_key is empty",
				index,
			))
		}
		prepared = append(prepared, preparedFanOutDesignation{
			idempotencyKey: idempotencyKey,
			metadata:       metadata,
		})
	}
	return prepared, nil
}

func enqueueFanOutTaskRuns(
	ctx context.Context,
	manager TaskService,
	actor taskpkg.ActorContext,
	taskID string,
	groupID string,
	req contract.FanOutTaskRunsRequest,
	prepared []preparedFanOutDesignation,
) ([]taskpkg.Run, error) {
	runs := make([]taskpkg.Run, 0, len(req.Designations))
	for index := range req.Designations {
		run, err := manager.EnqueueRun(ctx, taskpkg.EnqueueRun{
			TaskID:             taskID,
			IdempotencyKey:     prepared[index].idempotencyKey,
			NetworkChannel:     strings.TrimSpace(req.NetworkChannel),
			DesignationGroupID: groupID,
			Metadata:           prepared[index].metadata,
		}, actor)
		if err != nil {
			return nil, err
		}
		runs = append(runs, *run)
	}
	return runs, nil
}

// ClaimTaskRun claims one queued run.
func (h *BaseHandlers) ClaimTaskRun(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	runID, err := requiredPathID(c.Param("id"), "run id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.ClaimTaskRunRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode claim run request: %w", h.transportName(), err)),
		)
		return
	}

	actor, err := h.taskActorContext(c, taskActionClaimRun)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	claim, err := claimTaskRunFromRequest(req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	run, err := manager.ClaimRun(c.Request.Context(), runID, claim, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskRunResponse{Run: TaskRunPayloadFromRun(run)})
}

// StartTaskRun starts one claimed run.
func (h *BaseHandlers) StartTaskRun(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	runID, err := requiredPathID(c.Param("id"), "run id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.StartTaskRunRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode start run request: %w", h.transportName(), err)),
		)
		return
	}

	actor, err := h.taskActorContext(c, taskActionStartRun)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	startReq, err := startTaskRunFromRequest(req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	run, err := manager.StartRun(c.Request.Context(), runID, startReq, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskRunResponse{Run: TaskRunPayloadFromRun(run)})
}

// AttachTaskRunSession binds one existing session to a run.
func (h *BaseHandlers) AttachTaskRunSession(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	runID, err := requiredPathID(c.Param("id"), "run id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.AttachTaskRunSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode attach run session request: %w", h.transportName(), err)),
		)
		return
	}

	actor, err := h.taskActorContext(c, taskActionAttachRun)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	sessionID, err := attachTaskRunSessionIDFromRequest(req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	run, err := manager.AttachRunSession(c.Request.Context(), runID, sessionID, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskRunResponse{Run: TaskRunPayloadFromRun(run)})
}

// CompleteTaskRun marks one running run as completed.
func (h *BaseHandlers) CompleteTaskRun(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	runID, err := requiredPathID(c.Param("id"), "run id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.CompleteTaskRunRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode complete run request: %w", h.transportName(), err)),
		)
		return
	}

	actor, err := h.taskActorContext(c, taskActionCompleteRun)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	result, err := completeTaskRunFromRequest(req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	run, err := manager.CompleteRun(c.Request.Context(), runID, result, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskRunResponse{Run: TaskRunPayloadFromRun(run)})
}

// FailTaskRun marks one run as failed.
func (h *BaseHandlers) FailTaskRun(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	runID, err := requiredPathID(c.Param("id"), "run id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.FailTaskRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode fail run request: %w", h.transportName(), err)),
		)
		return
	}

	actor, err := h.taskActorContext(c, taskActionFailRun)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	failure, err := failTaskRunFromRequest(req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	run, err := manager.FailRun(c.Request.Context(), runID, failure, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskRunResponse{Run: TaskRunPayloadFromRun(run)})
}

// ForceReleaseTaskRun force releases one claimed run without requiring the raw claim token.
func (h *BaseHandlers) ForceReleaseTaskRun(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}
	runID, err := requiredPathID(c.Param("id"), "run id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	var req contract.ForceReleaseTaskRunRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode force release run request: %w", h.transportName(), err)),
		)
		return
	}
	actor, err := h.taskActorContext(c, taskActionForceReleaseRun)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	run, err := manager.ForceReleaseRun(
		c.Request.Context(),
		runID,
		taskpkg.ForceReleaseRun{Reason: req.Reason, Metadata: req.Metadata},
		actor,
	)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.TaskRunResponse{Run: TaskRunPayloadFromRun(run)})
}

// ForceFailTaskRun force fails one queued or claimed run without requiring the raw claim token.
func (h *BaseHandlers) ForceFailTaskRun(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}
	runID, err := requiredPathID(c.Param("id"), "run id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	var req contract.ForceFailTaskRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode force fail run request: %w", h.transportName(), err)),
		)
		return
	}
	actor, err := h.taskActorContext(c, taskActionForceFailRun)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	run, err := manager.ForceFailRun(
		c.Request.Context(),
		runID,
		taskpkg.ForceFailRun{Reason: req.Reason, Metadata: req.Metadata},
		actor,
	)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.TaskRunResponse{Run: TaskRunPayloadFromRun(run)})
}

// RetryTaskRun enqueues a new run linked to one failed source run.
func (h *BaseHandlers) RetryTaskRun(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}
	runID, err := requiredPathID(c.Param("id"), "run id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	var req contract.RetryTaskRunRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode retry run request: %w", h.transportName(), err)),
		)
		return
	}
	actor, err := h.taskActorContext(c, taskActionRetryRun)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	result, err := manager.RetryRun(
		c.Request.Context(),
		runID,
		taskpkg.RetryRunRequest{Metadata: req.Metadata},
		actor,
	)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	c.JSON(http.StatusCreated, RetryTaskRunResponseFromResult(result))
}

// RecoverTaskRun terminalizes one needs_attention run and queues a fresh child to resume work.
func (h *BaseHandlers) RecoverTaskRun(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}
	runID, err := requiredPathID(c.Param("id"), "run id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	var req contract.RecoverTaskRunRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode recover run request: %w", h.transportName(), err)),
		)
		return
	}
	actor, err := h.taskActorContext(c, taskActionRecoverRun)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	result, err := manager.RecoverRun(
		c.Request.Context(),
		runID,
		taskpkg.RecoverRunRequest{Reason: req.Reason, Metadata: req.Metadata},
		actor,
	)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	c.JSON(http.StatusCreated, RetryTaskRunResponseFromResult(result))
}

// BulkForceReleaseTaskRuns force releases a bounded set of runs.
func (h *BaseHandlers) BulkForceReleaseTaskRuns(c *gin.Context) {
	h.bulkForceTaskRuns(c, taskActionBulkReleaseRuns, false)
}

// BulkForceFailTaskRuns force fails a bounded set of runs.
func (h *BaseHandlers) BulkForceFailTaskRuns(c *gin.Context) {
	h.bulkForceTaskRuns(c, taskActionBulkFailRuns, true)
}

func (h *BaseHandlers) bulkForceTaskRuns(c *gin.Context, action string, fail bool) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}
	var req contract.BulkForceTaskRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode bulk force run request: %w", h.transportName(), err)),
		)
		return
	}
	actor, err := h.taskActorContext(c, action)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	domainReq := taskpkg.BulkForceRunRequest{
		RunIDs:   req.RunIDs,
		Reason:   req.Reason,
		Metadata: req.Metadata,
	}
	var result taskpkg.BulkForceRunResult
	if fail {
		result, err = manager.BulkForceFailRuns(c.Request.Context(), domainReq, actor)
	} else {
		result, err = manager.BulkForceReleaseRuns(c.Request.Context(), domainReq, actor)
	}
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	c.JSON(http.StatusOK, BulkForceTaskRunResponseFromResult(result, h.MaskInternalErrors))
}

// CancelTaskRun cancels one non-terminal run.
func (h *BaseHandlers) CancelTaskRun(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	runID, err := requiredPathID(c.Param("id"), "run id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.CancelTaskRunRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewTaskValidationError(fmt.Errorf("%s: decode cancel run request: %w", h.transportName(), err)),
		)
		return
	}

	actor, err := h.taskActorContext(c, taskActionCancelRun)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	cancelReq, err := cancelTaskRunFromRequest(req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	run, err := manager.CancelRun(c.Request.Context(), runID, cancelReq, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskRunResponse{Run: TaskRunPayloadFromRun(run)})
}

func (h *BaseHandlers) mutateTaskApproval(
	c *gin.Context,
	action string,
	mutate func(context.Context, TaskService, string, taskpkg.ActorContext) (*taskpkg.Task, error),
) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	actor, err := h.taskActorContext(c, action)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	record, err := mutate(c.Request.Context(), manager, taskID, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskResponse{Task: TaskPayloadFromTask(record)})
}

func (h *BaseHandlers) mutateTaskTriage(
	c *gin.Context,
	action string,
	mutate func(context.Context, TaskService, string, taskpkg.ActorContext) (taskpkg.TriageState, error),
) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	actor, err := h.taskActorContext(c, action)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	state, err := mutate(c.Request.Context(), manager, taskID, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskTriageStateResponse{Triage: TaskTriageStatePayloadFromState(state)})
}

func (h *BaseHandlers) createTaskSpecFromRequest(
	ctx context.Context,
	req contract.CreateTaskRequest,
) (taskpkg.CreateTask, error) {
	scope := req.Scope.Normalize()
	workspaceID, err := h.resolveTaskWorkspaceBinding(ctx, scope, req.Workspace, "create_task")
	if err != nil {
		return taskpkg.CreateTask{}, err
	}
	if err := validateTaskChannel("create_task.network_channel", req.NetworkChannel); err != nil {
		return taskpkg.CreateTask{}, err
	}

	spec := taskpkg.CreateTask{
		ID:                 strings.TrimSpace(req.ID),
		Identifier:         strings.TrimSpace(req.Identifier),
		Scope:              scope,
		WorkspaceID:        workspaceID,
		NetworkChannel:     strings.TrimSpace(req.NetworkChannel),
		Title:              strings.TrimSpace(req.Title),
		Description:        strings.TrimSpace(req.Description),
		Priority:           req.Priority.Normalize(),
		MaxAttempts:        req.MaxAttempts,
		AutoEnqueueOnReady: req.AutoEnqueueOnReady,
		Draft:              req.Draft,
		ApprovalPolicy:     req.ApprovalPolicy.Normalize(),
		Owner:              cloneOwnership(req.Owner),
		WakeCreator:        cloneBoolPtr(req.WakeCreator),
		Metadata:           cloneRawMessage(req.Metadata),
	}
	if err := spec.Validate("create_task"); err != nil {
		return taskpkg.CreateTask{}, err
	}
	return spec, nil
}

func (h *BaseHandlers) createChildTaskSpecFromRequest(
	ctx context.Context,
	req contract.CreateTaskChildRequest,
) (taskpkg.CreateTask, error) {
	scope := req.Scope.Normalize()
	workspaceID, err := h.resolveTaskWorkspaceBinding(ctx, scope, req.Workspace, "create_child_task")
	if err != nil {
		return taskpkg.CreateTask{}, err
	}
	if err := validateTaskChannel("create_child_task.network_channel", req.NetworkChannel); err != nil {
		return taskpkg.CreateTask{}, err
	}

	spec := taskpkg.CreateTask{
		ID:                 strings.TrimSpace(req.ID),
		Identifier:         strings.TrimSpace(req.Identifier),
		Scope:              scope,
		WorkspaceID:        workspaceID,
		NetworkChannel:     strings.TrimSpace(req.NetworkChannel),
		Title:              strings.TrimSpace(req.Title),
		Description:        strings.TrimSpace(req.Description),
		Priority:           req.Priority.Normalize(),
		MaxAttempts:        req.MaxAttempts,
		AutoEnqueueOnReady: req.AutoEnqueueOnReady,
		Draft:              req.Draft,
		ApprovalPolicy:     req.ApprovalPolicy.Normalize(),
		Owner:              cloneOwnership(req.Owner),
		WakeCreator:        cloneBoolPtr(req.WakeCreator),
		Metadata:           cloneRawMessage(req.Metadata),
	}
	if err := spec.Validate("create_child_task"); err != nil {
		return taskpkg.CreateTask{}, err
	}
	return spec, nil
}

func taskPatchFromRequest(req contract.UpdateTaskRequest) (taskpkg.Patch, error) {
	if req.NetworkChannel != nil {
		if err := validateTaskChannel("task_patch.network_channel", *req.NetworkChannel); err != nil {
			return taskpkg.Patch{}, err
		}
	}

	patch := taskpkg.Patch{
		Title:              trimStringPtr(req.Title),
		Description:        trimStringPtr(req.Description),
		Priority:           normalizePriorityPtr(req.Priority),
		MaxAttempts:        req.MaxAttempts,
		AutoEnqueueOnReady: req.AutoEnqueueOnReady,
		ApprovalPolicy:     normalizeApprovalPolicyPtr(req.ApprovalPolicy),
		Metadata:           cloneRawMessagePtr(req.Metadata),
		NetworkChannel:     trimStringPtr(req.NetworkChannel),
		Owner:              cloneOwnership(req.Owner),
		ClearOwner:         req.ClearOwner,
	}
	if err := patch.Validate("task_patch"); err != nil {
		return taskpkg.Patch{}, err
	}
	return patch, nil
}

func taskExecutionProfileFromRequest(
	taskID string,
	req *contract.SetTaskExecutionProfileRequest,
) (*taskpkg.ExecutionProfile, error) {
	trimmedID := strings.TrimSpace(taskID)
	if trimmedID == "" {
		return nil, NewTaskValidationError(errors.New("task id is required"))
	}
	if req == nil {
		return nil, NewTaskValidationError(errors.New("task execution profile request is required"))
	}
	if strings.TrimSpace(req.TaskID) != "" && strings.TrimSpace(req.TaskID) != trimmedID {
		return nil, NewTaskValidationError(fmt.Errorf(
			"task_execution_profile.task_id must match task id %q",
			trimmedID,
		))
	}
	profile := *req
	profile.TaskID = trimmedID
	profile.CreatedAt = time.Time{}
	profile.UpdatedAt = time.Time{}
	return &profile, nil
}

func cancelTaskFromRequest(req contract.CancelTaskRequest) (taskpkg.CancelTask, error) {
	cancelReq := taskpkg.CancelTask{
		Reason:   strings.TrimSpace(req.Reason),
		Metadata: cloneRawMessage(req.Metadata),
	}
	if err := cancelReq.Validate("cancel_task"); err != nil {
		return taskpkg.CancelTask{}, err
	}
	return cancelReq, nil
}

func addTaskDependencyFromRequest(taskID string, req contract.AddTaskDependencyRequest) (taskpkg.AddDependency, error) {
	kind := req.Kind.Normalize()
	if kind == "" {
		kind = taskpkg.DependencyKindBlocks
	}

	spec := taskpkg.AddDependency{
		TaskID:          strings.TrimSpace(taskID),
		DependsOnTaskID: strings.TrimSpace(req.DependsOnTaskID),
		Kind:            kind,
	}
	if err := spec.Validate("add_dependency"); err != nil {
		return taskpkg.AddDependency{}, err
	}
	return spec, nil
}

func enqueueTaskRunFromRequest(taskID string, req contract.EnqueueTaskRunRequest) (taskpkg.EnqueueRun, error) {
	if err := validateTaskChannel("enqueue_run.network_channel", req.NetworkChannel); err != nil {
		return taskpkg.EnqueueRun{}, err
	}

	spec := taskpkg.EnqueueRun{
		TaskID:         strings.TrimSpace(taskID),
		IdempotencyKey: strings.TrimSpace(req.IdempotencyKey),
		NetworkChannel: strings.TrimSpace(req.NetworkChannel),
		Metadata:       append(json.RawMessage(nil), req.Metadata...),
	}
	if err := spec.Validate("enqueue_run"); err != nil {
		return taskpkg.EnqueueRun{}, err
	}
	return spec, nil
}

type fanOutDesignationMetadataPayload struct {
	Designation fanOutDesignationMetadataDetail `json:"designation"`
	Metadata    json.RawMessage                 `json:"metadata,omitempty"`
}

type fanOutDesignationMetadataDetail struct {
	Index int    `json:"index"`
	Brief string `json:"brief"`
}

func fanOutDesignationMetadata(
	index int,
	designation contract.TaskFanOutRunDesignationRequest,
) (json.RawMessage, error) {
	brief := strings.TrimSpace(designation.Brief)
	if brief == "" {
		return nil, NewTaskValidationError(fmt.Errorf("designations[%d].brief is required", index))
	}
	metadata := cloneRawMessage(designation.Metadata)
	if len(metadata) > 0 && !json.Valid(metadata) {
		return nil, NewTaskValidationError(fmt.Errorf("designations[%d].metadata must be valid JSON", index))
	}
	payload := fanOutDesignationMetadataPayload{
		Designation: fanOutDesignationMetadataDetail{Index: index, Brief: brief},
		Metadata:    metadata,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("task fan-out metadata: %w", err)
	}
	return encoded, nil
}

func fanOutDesignationIdempotencyKey(
	requestKey string,
	designation contract.TaskFanOutRunDesignationRequest,
	index int,
) string {
	if key := strings.TrimSpace(designation.IdempotencyKey); key != "" {
		return key
	}
	if key := strings.TrimSpace(requestKey); key != "" {
		return fmt.Sprintf("%s:%d", key, index)
	}
	return ""
}

func fanOutDesignationRollupJSON(runs []taskpkg.Run, now time.Time) json.RawMessage {
	runIDs := make([]string, 0, len(runs))
	statuses := make(map[string]int)
	var taskID string
	var groupID string
	completed := 0
	failed := 0
	canceled := 0
	running := 0
	queued := 0
	needsAttention := 0
	terminalCount := 0
	for _, run := range runs {
		if id := strings.TrimSpace(run.ID); id != "" {
			runIDs = append(runIDs, id)
		}
		taskID = firstNonEmpty(taskID, strings.TrimSpace(run.TaskID))
		groupID = firstNonEmpty(groupID, strings.TrimSpace(run.DesignationGroupID))
		status := run.Status.Normalize()
		statuses[status.String()]++
		switch status {
		case taskpkg.TaskRunStatusCompleted:
			completed++
			terminalCount++
		case taskpkg.TaskRunStatusFailed:
			failed++
			terminalCount++
		case taskpkg.TaskRunStatusCanceled:
			canceled++
			terminalCount++
		case taskpkg.TaskRunStatusQueued:
			queued++
		case taskpkg.TaskRunStatusNeedsAttention:
			needsAttention++
		default:
			running++
		}
	}
	encoded, err := json.Marshal(map[string]any{
		"designation_group_id":         groupID,
		"task_id":                      taskID,
		"run_ids":                      runIDs,
		"total":                        len(runIDs),
		"terminal_count":               terminalCount,
		taskDesignationRollupCompleted: completed,
		"failed":                       failed,
		taskDesignationRollupCanceled:  canceled,
		"running":                      running,
		"queued":                       queued,
		"needs_attention":              needsAttention,
		"statuses":                     statuses,
		"complete":                     len(runIDs) > 0 && terminalCount == len(runIDs),
		"updated_at":                   now.UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return json.RawMessage(`{"run_ids":[],"total":0,"terminal_count":0,"complete":false}`)
	}
	return encoded
}

func taskExecutionRequestFromRequest(req contract.TaskExecutionRequest) (taskpkg.ExecutionRequest, error) {
	if err := validateTaskChannel("task_execution.network_channel", req.NetworkChannel); err != nil {
		return taskpkg.ExecutionRequest{}, err
	}
	spec := taskpkg.ExecutionRequest{
		IdempotencyKey: strings.TrimSpace(req.IdempotencyKey),
		NetworkChannel: strings.TrimSpace(req.NetworkChannel),
		Metadata:       append(json.RawMessage(nil), req.Metadata...),
	}
	if err := spec.Validate("task_execution"); err != nil {
		return taskpkg.ExecutionRequest{}, err
	}
	return spec, nil
}

func claimTaskRunFromRequest(req contract.ClaimTaskRunRequest) (taskpkg.ClaimRun, error) {
	claim := taskpkg.ClaimRun{IdempotencyKey: strings.TrimSpace(req.IdempotencyKey)}
	if err := claim.Validate("claim_run"); err != nil {
		return taskpkg.ClaimRun{}, err
	}
	return claim, nil
}

func startTaskRunFromRequest(req contract.StartTaskRunRequest) (taskpkg.StartRun, error) {
	startReq := taskpkg.StartRun{IdempotencyKey: strings.TrimSpace(req.IdempotencyKey)}
	if err := startReq.Validate("start_run"); err != nil {
		return taskpkg.StartRun{}, err
	}
	return startReq, nil
}

func attachTaskRunSessionIDFromRequest(req contract.AttachTaskRunSessionRequest) (string, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return "", NewTaskValidationError(errors.New("session_id is required"))
	}
	return sessionID, nil
}

func completeTaskRunFromRequest(req contract.CompleteTaskRunRequest) (taskpkg.RunResult, error) {
	result := taskpkg.RunResult{Value: cloneRawMessage(req.Result)}
	if err := result.Validate("run_result"); err != nil {
		return taskpkg.RunResult{}, err
	}
	return result, nil
}

func createTaskBlockFromRequest(
	taskID string,
	req contract.CreateTaskBlockRequest,
	claimToken string,
) (taskpkg.BlockRequest, error) {
	blockReq := taskpkg.BlockRequest{
		TaskID:     strings.TrimSpace(taskID),
		Kind:       req.Kind.Normalize(),
		Reason:     strings.TrimSpace(req.Reason),
		Details:    cloneRawMessage(req.Details),
		RunID:      strings.TrimSpace(req.RunID),
		ClaimToken: strings.TrimSpace(claimToken),
	}
	if req.ExpiresAt != nil {
		blockReq.ExpiresAt = req.ExpiresAt.UTC()
	}
	if blockReq.TaskID == "" {
		return taskpkg.BlockRequest{}, fmt.Errorf("%w: task_block.task_id is required", taskpkg.ErrValidation)
	}
	if err := blockReq.Kind.Validate("task_block.kind"); err != nil {
		return taskpkg.BlockRequest{}, err
	}
	if blockReq.Reason == "" {
		return taskpkg.BlockRequest{}, fmt.Errorf("%w: task_block.reason is required", taskpkg.ErrValidation)
	}
	if blockReq.ExpiresAt.IsZero() {
		return blockReq, nil
	}
	if blockReq.Kind != taskpkg.BlockKindTransient {
		return taskpkg.BlockRequest{}, fmt.Errorf(
			"%w: task_block.expires_at is only valid for %q blocks",
			taskpkg.ErrValidation,
			taskpkg.BlockKindTransient,
		)
	}
	return blockReq, nil
}

func statusForTaskBlockError(err error) int {
	if errors.Is(err, taskpkg.ErrValidation) {
		return http.StatusUnprocessableEntity
	}
	return StatusForTaskError(err)
}

func failTaskRunFromRequest(req contract.FailTaskRunRequest) (taskpkg.RunFailure, error) {
	failure := taskpkg.RunFailure{
		Error:    strings.TrimSpace(req.Error),
		Metadata: cloneRawMessage(req.Metadata),
	}
	if err := failure.Validate("run_failure"); err != nil {
		return taskpkg.RunFailure{}, err
	}
	return failure, nil
}

func cancelTaskRunFromRequest(req contract.CancelTaskRunRequest) (taskpkg.CancelRun, error) {
	cancelReq := taskpkg.CancelRun{
		Reason:   strings.TrimSpace(req.Reason),
		Metadata: cloneRawMessage(req.Metadata),
	}
	if err := cancelReq.Validate("cancel_run"); err != nil {
		return taskpkg.CancelRun{}, err
	}
	return cancelReq, nil
}

func (h *BaseHandlers) resolveTaskWorkspaceBinding(
	ctx context.Context,
	scope taskpkg.Scope,
	workspaceRef string,
	path string,
) (string, error) {
	trimmed := strings.TrimSpace(workspaceRef)
	if err := taskpkg.ValidateScopeBinding(scope, trimmed, path, "workspace"); err != nil {
		return "", err
	}
	if scope.Normalize() != taskpkg.ScopeWorkspace {
		return "", nil
	}
	return h.lookupWorkspaceID(ctx, trimmed)
}

func validateTaskChannel(path string, channel string) error {
	trimmed := strings.TrimSpace(channel)
	if trimmed == "" {
		return nil
	}
	if err := network.ValidateChannel(trimmed); err != nil {
		return NewTaskValidationError(fmt.Errorf("%s: %w", path, err))
	}
	return nil
}

func requiredPathID(raw string, field string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", NewTaskValidationError(fmt.Errorf("%s is required", field))
	}
	return trimmed, nil
}

func decodeOptionalJSON(c *gin.Context, dest any) error {
	if err := c.ShouldBindJSON(dest); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

// TaskSummaryPayloadsFromSummaries converts task summaries into shared payloads.
func TaskSummaryPayloadsFromSummaries(tasks []taskpkg.Summary) []contract.TaskSummaryPayload {
	payloads := make([]contract.TaskSummaryPayload, 0, len(tasks))
	for idx := range tasks {
		payloads = append(payloads, TaskSummaryPayloadFromSummary(&tasks[idx]))
	}
	return payloads
}

// TaskSummaryPayloadFromSummary converts one task summary into the shared payload.
func TaskSummaryPayloadFromSummary(record *taskpkg.Summary) contract.TaskSummaryPayload {
	if record == nil {
		return contract.TaskSummaryPayload{}
	}

	return contract.TaskSummaryPayload{
		ID:                   record.ID,
		Identifier:           record.Identifier,
		Scope:                record.Scope,
		WorkspaceID:          record.WorkspaceID,
		ParentTaskID:         record.ParentTaskID,
		NetworkChannel:       record.NetworkChannel,
		Title:                taskpkg.RedactClaimTokens(strings.TrimSpace(record.Title)),
		Priority:             record.Priority,
		MaxAttempts:          record.MaxAttempts,
		AutoEnqueueOnReady:   record.AutoEnqueueOnReady,
		Status:               record.Status,
		ApprovalPolicy:       record.ApprovalPolicy,
		ApprovalState:        record.ApprovalState,
		Draft:                record.Draft,
		Owner:                cloneOwnership(record.Owner),
		CurrentRunID:         record.CurrentRunID,
		LatestEventSeq:       record.LatestEventSeq,
		Paused:               record.Paused,
		PausedBy:             record.PausedBy,
		PausedAt:             optionalTime(record.PausedAt),
		PausedReason:         taskpkg.RedactClaimTokens(strings.TrimSpace(record.PausedReason)),
		EffectivePaused:      record.EffectivePaused,
		PausedByTaskID:       record.PausedByTaskID,
		BlockedReasons:       blockedReasonsPayload(record.BlockedReasons),
		NeedsAttention:       recordNeedsAttention(record.NeedsAttention, record.Status),
		NeedsAttentionReason: needsAttentionReason(record.NeedsAttention),
		NeedsAttentionAt:     needsAttentionAt(record.NeedsAttention),
		NeedsAttentionBy:     needsAttentionBy(record.NeedsAttention),
		WakeCreator:          record.WakeCreator,
		CreatedBy:            record.CreatedBy,
		Origin:               record.Origin,
		CreatedAt:            record.CreatedAt,
		UpdatedAt:            record.UpdatedAt,
		ClosedAt:             optionalTime(record.ClosedAt),
		ChildCount:           int(record.ChildCount),
		DependencyCount:      int(record.DependencyCount),
		Dependencies:         TaskDependencyReferencePayloadsFromReferences(record.Dependencies),
		ActiveRun:            TaskRunSummaryPayloadFromSummary(record.ActiveRun),
		LastActivityAt:       optionalTime(record.LastActivityAt),
	}
}

// TaskPayloadFromTask converts one task record into the shared payload.
func TaskPayloadFromTask(record *taskpkg.Task) contract.TaskPayload {
	if record == nil {
		return contract.TaskPayload{}
	}

	return contract.TaskPayload{
		ID:                 record.ID,
		Identifier:         record.Identifier,
		Scope:              record.Scope,
		WorkspaceID:        record.WorkspaceID,
		ParentTaskID:       record.ParentTaskID,
		NetworkChannel:     record.NetworkChannel,
		Title:              taskpkg.RedactClaimTokens(strings.TrimSpace(record.Title)),
		Description:        taskpkg.RedactClaimTokens(strings.TrimSpace(record.Description)),
		Priority:           record.Priority,
		MaxAttempts:        record.MaxAttempts,
		AutoEnqueueOnReady: record.AutoEnqueueOnReady,
		Status:             record.Status,
		ApprovalPolicy:     record.ApprovalPolicy,
		ApprovalState:      record.ApprovalState,
		Draft:              record.Status.Normalize() == taskpkg.TaskStatusDraft,
		Owner:              cloneOwnership(record.Owner),
		CurrentRunID:       record.CurrentRunID,
		LatestEventSeq:     record.LatestEventSeq,
		Paused:             record.Paused,
		PausedBy:           record.PausedBy,
		PausedAt:           optionalTime(record.PausedAt),
		PausedReason:       taskpkg.RedactClaimTokens(strings.TrimSpace(record.PausedReason)),
		EffectivePaused:    record.Paused,
		PausedByTaskID: func() string {
			if record.Paused {
				return record.ID
			}
			return ""
		}(),
		NeedsAttention:       recordNeedsAttention(record.NeedsAttention, record.Status),
		NeedsAttentionReason: needsAttentionReason(record.NeedsAttention),
		NeedsAttentionAt:     needsAttentionAt(record.NeedsAttention),
		NeedsAttentionBy:     needsAttentionBy(record.NeedsAttention),
		WakeCreator:          record.WakeCreator,
		CreatedBy:            record.CreatedBy,
		Origin:               record.Origin,
		CreatedAt:            record.CreatedAt,
		UpdatedAt:            record.UpdatedAt,
		ClosedAt:             optionalTime(record.ClosedAt),
		Metadata:             redactRawClaimTokenFields(record.Metadata),
	}
}

func blockedReasonsPayload(reasons *[]taskpkg.BlockedReason) []taskpkg.BlockedReason {
	if reasons == nil || len(*reasons) == 0 {
		return nil
	}
	cloned := make([]taskpkg.BlockedReason, len(*reasons))
	for idx, reason := range *reasons {
		cloned[idx] = reason
		cloned[idx].Reason = taskpkg.RedactClaimTokens(strings.TrimSpace(reason.Reason))
	}
	return cloned
}

func recordNeedsAttention(attention *taskpkg.NeedsAttention, status taskpkg.Status) bool {
	return attention != nil || status.Normalize() == taskpkg.TaskStatusNeedsAttention
}

func needsAttentionReason(attention *taskpkg.NeedsAttention) string {
	if attention == nil {
		return ""
	}
	return taskpkg.RedactClaimTokens(strings.TrimSpace(attention.Reason))
}

func needsAttentionAt(attention *taskpkg.NeedsAttention) *time.Time {
	if attention == nil {
		return nil
	}
	return optionalTime(attention.At)
}

func needsAttentionBy(attention *taskpkg.NeedsAttention) *taskpkg.ActorIdentity {
	if attention == nil || attention.By.IsZero() {
		return nil
	}
	actor := attention.By
	return &actor
}

// TaskBlockPayloadsFromBlocks converts task-block records into shared payloads.
func TaskBlockPayloadsFromBlocks(blocks []taskpkg.TaskBlock) []contract.TaskBlockPayload {
	if len(blocks) == 0 {
		return nil
	}
	payloads := make([]contract.TaskBlockPayload, 0, len(blocks))
	for _, block := range blocks {
		payloads = append(payloads, TaskBlockPayloadFromBlock(block))
	}
	return payloads
}

// TaskBlockPayloadFromBlock converts one task-block record into the shared payload.
func TaskBlockPayloadFromBlock(block taskpkg.TaskBlock) contract.TaskBlockPayload {
	payload := contract.TaskBlockPayload{
		ID:          strings.TrimSpace(block.ID),
		TaskID:      strings.TrimSpace(block.TaskID),
		WorkspaceID: strings.TrimSpace(block.WorkspaceID),
		Kind:        block.Kind.Normalize(),
		Reason:      taskpkg.RedactClaimTokens(strings.TrimSpace(block.Reason)),
		Details:     redactRawClaimTokenFields(block.Details),
		CreatedAt:   block.CreatedAt,
		CreatedBy:   block.CreatedBy,
		ExpiresAt:   optionalTime(block.ExpiresAt),
		ClearedAt:   optionalTime(block.ClearedAt),
		ClearNote:   taskpkg.RedactClaimTokens(strings.TrimSpace(block.ClearNote)),
	}
	if !block.ClearedBy.IsZero() {
		clearedBy := block.ClearedBy
		payload.ClearedBy = &clearedBy
	}
	return payload
}

// TaskDependencyPayloadsFromDependencies converts dependency records into shared payloads.
func TaskDependencyPayloadsFromDependencies(dependencies []taskpkg.Dependency) []contract.TaskDependencyPayload {
	payloads := make([]contract.TaskDependencyPayload, 0, len(dependencies))
	for _, dependency := range dependencies {
		payloads = append(payloads, contract.TaskDependencyPayload{
			TaskID:          dependency.TaskID,
			DependsOnTaskID: dependency.DependsOnTaskID,
			Kind:            dependency.Kind,
			CreatedAt:       dependency.CreatedAt,
		})
	}
	return payloads
}

// TaskRunPayloadsFromRuns converts task runs into shared payloads.
func TaskRunPayloadsFromRuns(runs []taskpkg.Run) []contract.TaskRunPayload {
	payloads := make([]contract.TaskRunPayload, 0, len(runs))
	for _, run := range runs {
		payloads = append(payloads, TaskRunPayloadFromRun(&run))
	}
	return payloads
}

// TaskExecutionResponseFromExecution converts one task execution-boundary result.
func TaskExecutionResponseFromExecution(execution *taskpkg.Execution) contract.TaskExecutionResponse {
	if execution == nil {
		return contract.TaskExecutionResponse{}
	}
	return contract.TaskExecutionResponse{
		Task: TaskPayloadFromTask(&execution.Task),
		Run:  TaskRunPayloadFromRun(&execution.Run),
	}
}

// TaskRunPayloadFromRun converts one task run into the shared payload.
func TaskRunPayloadFromRun(run *taskpkg.Run) contract.TaskRunPayload {
	if run == nil {
		return contract.TaskRunPayload{}
	}

	return contract.TaskRunPayload{
		ID:                    run.ID,
		TaskID:                run.TaskID,
		Status:                run.Status,
		Attempt:               int(run.Attempt),
		PreviousRunID:         run.PreviousRunID,
		FailureKind:           run.FailureKind,
		ClaimedBy:             cloneActorIdentity(run.ClaimedBy),
		SessionID:             run.SessionID,
		Origin:                run.Origin,
		IdempotencyKey:        run.IdempotencyKey,
		NetworkChannel:        run.NetworkChannel,
		DesignationGroupID:    run.DesignationGroupID,
		ClaimTokenHash:        run.ClaimTokenHash,
		LeaseUntil:            optionalTime(run.LeaseUntil),
		HeartbeatAt:           optionalTime(run.HeartbeatAt),
		CoordinationChannelID: run.CoordinationChannelID,
		QueuedAt:              run.QueuedAt,
		ClaimedAt:             optionalTime(run.ClaimedAt),
		StartedAt:             optionalTime(run.StartedAt),
		EndedAt:               optionalTime(run.EndedAt),
		Error:                 run.Error,
		Metadata:              redactRawClaimTokenFields(run.Metadata),
		Result:                redactRawClaimTokenFields(run.Result),
	}
}

// RetryTaskRunResponseFromResult converts one retry result into the shared payload.
func RetryTaskRunResponseFromResult(result *taskpkg.RetryRunResult) contract.RetryTaskRunResponse {
	if result == nil {
		return contract.RetryTaskRunResponse{}
	}
	return contract.RetryTaskRunResponse{
		PreviousRun: TaskRunPayloadFromRun(&result.PreviousRun),
		Run:         TaskRunPayloadFromRun(&result.Run),
	}
}

// BulkForceTaskRunResponseFromResult converts per-row bulk force outcomes into shared payloads.
func BulkForceTaskRunResponseFromResult(
	result taskpkg.BulkForceRunResult,
	maskInternalErrors bool,
) contract.BulkForceTaskRunResponse {
	items := make([]contract.BulkForceTaskRunItemPayload, 0, len(result.Items))
	for _, item := range result.Items {
		payload := contract.BulkForceTaskRunItemPayload{
			RunID: item.RunID,
			OK:    item.OK,
			Run:   optionalTaskRunPayload(item.Run),
		}
		if item.Err != nil {
			errorPayload := ErrorPayloadForStatus(StatusForTaskError(item.Err), item.Err, maskInternalErrors)
			payload.Error = &errorPayload
		}
		items = append(items, payload)
	}
	return contract.BulkForceTaskRunResponse{Results: items}
}

func optionalTaskRunPayload(run *taskpkg.Run) *contract.TaskRunPayload {
	if run == nil {
		return nil
	}
	payload := TaskRunPayloadFromRun(run)
	return &payload
}

func redactRawClaimTokenFields(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return cloneRawMessage(raw)
	}
	redacted, changed := redactRawClaimTokenValue(decoded)
	if !changed {
		return cloneRawMessage(raw)
	}
	encoded, err := json.Marshal(redacted)
	if err != nil {
		return nil
	}
	return encoded
}

func redactRawClaimTokenValue(value any) (any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		changed := false
		redacted := make(map[string]any, len(typed))
		for key, nested := range typed {
			if strings.EqualFold(strings.TrimSpace(key), "claim_token") {
				changed = true
				continue
			}
			next, nestedChanged := redactRawClaimTokenValue(nested)
			redacted[key] = next
			changed = changed || nestedChanged
		}
		return redacted, changed
	case []any:
		changed := false
		redacted := make([]any, len(typed))
		for idx, nested := range typed {
			next, nestedChanged := redactRawClaimTokenValue(nested)
			redacted[idx] = next
			changed = changed || nestedChanged
		}
		return redacted, changed
	case string:
		redacted := taskpkg.RedactClaimTokens(typed)
		return redacted, redacted != typed
	default:
		return value, false
	}
}

// TaskEventPayloadsFromEvents converts task events into shared payloads.
func TaskEventPayloadsFromEvents(events []taskpkg.Event) []contract.TaskEventPayload {
	payloads := make([]contract.TaskEventPayload, 0, len(events))
	for _, event := range events {
		payloads = append(payloads, contract.TaskEventPayload{
			ID:        event.ID,
			TaskID:    event.TaskID,
			RunID:     event.RunID,
			EventType: event.EventType,
			Actor:     event.Actor,
			Origin:    event.Origin,
			Payload:   cloneRawMessage(event.Payload),
			Timestamp: event.Timestamp,
		})
	}
	return payloads
}

// TaskDetailPayloadFromView converts one expanded task view into the shared payload.
func TaskDetailPayloadFromView(view *taskpkg.View) contract.TaskDetailPayload {
	if view == nil {
		return contract.TaskDetailPayload{}
	}

	summary := TaskSummaryPayloadFromSummary(&view.Summary)
	taskRecord := TaskPayloadFromTask(&view.Task)
	taskRecord.EffectivePaused = summary.EffectivePaused
	taskRecord.PausedByTaskID = summary.PausedByTaskID
	taskRecord.BlockedReasons = summary.BlockedReasons

	return contract.TaskDetailPayload{
		Summary:              summary,
		Task:                 taskRecord,
		Children:             TaskSummaryPayloadsFromSummaries(view.Children),
		Dependencies:         TaskDependencyPayloadsFromDependencies(view.Dependencies),
		DependencyReferences: TaskDependencyReferencePayloadsFromReferences(view.DependencyReferences),
		Runs:                 TaskRunPayloadsFromRuns(view.Runs),
		Events:               TaskEventPayloadsFromEvents(view.Events),
	}
}

// TaskDesignationRollupPayloadsFromStore converts persisted designation rollups into shared payloads.
func TaskDesignationRollupPayloadsFromStore(
	rollups []store.TaskDesignationRollup,
) []contract.TaskDesignationRollupPayload {
	if len(rollups) == 0 {
		return nil
	}
	payloads := make([]contract.TaskDesignationRollupPayload, 0, len(rollups))
	for _, rollup := range rollups {
		payloads = append(payloads, contract.TaskDesignationRollupPayload{
			DesignationGroupID: strings.TrimSpace(rollup.DesignationGroupID),
			TaskID:             strings.TrimSpace(rollup.TaskID),
			Summary:            cloneRawMessage(rollup.SummaryJSON),
			CreatedAt:          rollup.CreatedAt,
		})
	}
	return payloads
}

// TaskInspectPayloadFromView converts one inspect view into the shared payload.
func TaskInspectPayloadFromView(view *taskpkg.InspectView) contract.TaskInspectPayload {
	if view == nil {
		return contract.TaskInspectPayload{}
	}

	return contract.TaskInspectPayload{
		Target:       string(view.Target),
		Task:         TaskSummaryPayloadFromSummary(&view.Task),
		CurrentRun:   TaskInspectRunPayloadFromSummary(view.CurrentRun),
		BoundSession: TaskInspectSessionPayloadFromSummary(view.BoundSession),
		RecentRuns:   TaskInspectRunPayloadsFromSummaries(view.RecentRuns),
		RecentEvents: TaskInspectEventPayloadsFromSummaries(view.RecentEvents),
		Scheduler:    TaskInspectSchedulerPayloadFromState(view.Scheduler),
		Diagnostics:  append([]contract.DiagnosticItem(nil), view.Diagnostics...),
		NextAction:   string(view.NextAction),
		AsOf:         view.AsOf,
	}
}

// TaskInspectRunPayloadFromSummary converts an inspect run summary into the shared payload.
func TaskInspectRunPayloadFromSummary(summary *taskpkg.InspectRunSummary) *contract.TaskInspectRunPayload {
	if summary == nil {
		return nil
	}
	return &contract.TaskInspectRunPayload{
		RunID:                   summary.RunID,
		TaskID:                  summary.TaskID,
		Status:                  summary.Status,
		ClaimTokenHashTruncated: summary.ClaimTokenHashTruncated,
		LeaseUntil:              optionalTime(summary.LeaseUntil),
		HeartbeatAt:             optionalTime(summary.HeartbeatAt),
		HeartbeatAgeSeconds:     cloneInt64Ptr(summary.HeartbeatAgeSeconds),
		Retries:                 summary.Retries,
		LastErrorSummary:        summary.LastErrorSummary,
		FailureKind:             summary.FailureKind,
		BoundSessionID:          summary.BoundSessionID,
		StartedAt:               optionalTime(summary.StartedAt),
		EndedAt:                 optionalTime(summary.EndedAt),
		PreviousRunID:           summary.PreviousRunID,
		QueuedAt:                summary.QueuedAt,
		Attempt:                 summary.Attempt,
	}
}

// TaskInspectRunPayloadsFromSummaries converts inspect run summaries into shared payloads.
func TaskInspectRunPayloadsFromSummaries(
	summaries []taskpkg.InspectRunSummary,
) []contract.TaskInspectRunPayload {
	payloads := make([]contract.TaskInspectRunPayload, 0, len(summaries))
	for idx := range summaries {
		payload := TaskInspectRunPayloadFromSummary(&summaries[idx])
		if payload != nil {
			payloads = append(payloads, *payload)
		}
	}
	return payloads
}

// TaskInspectSessionPayloadFromSummary converts an inspect session summary into the shared payload.
func TaskInspectSessionPayloadFromSummary(
	summary *taskpkg.InspectSessionSummary,
) *contract.TaskInspectSessionPayload {
	if summary == nil {
		return nil
	}
	return &contract.TaskInspectSessionPayload{
		SessionID:      summary.SessionID,
		State:          summary.State,
		AgentName:      summary.AgentName,
		ProviderName:   summary.ProviderName,
		WorkspaceID:    summary.WorkspaceID,
		StartedAt:      optionalTime(summary.StartedAt),
		LastActivityAt: optionalTime(summary.LastActivityAt),
		StopReason:     summary.StopReason,
		FailureKind:    summary.FailureKind,
	}
}

// TaskInspectEventPayloadsFromSummaries converts inspect event summaries into shared payloads.
func TaskInspectEventPayloadsFromSummaries(
	summaries []taskpkg.InspectEventSummary,
) []contract.TaskInspectEventPayload {
	payloads := make([]contract.TaskInspectEventPayload, 0, len(summaries))
	for _, summary := range summaries {
		payloads = append(payloads, contract.TaskInspectEventPayload{
			ID:        summary.ID,
			Type:      summary.Type,
			SessionID: summary.SessionID,
			TaskID:    summary.TaskID,
			RunID:     summary.RunID,
			Outcome:   summary.Outcome,
			Summary:   summary.Summary,
			Timestamp: summary.Timestamp,
		})
	}
	return payloads
}

// TaskInspectSchedulerPayloadFromState converts scheduler state into the shared payload.
func TaskInspectSchedulerPayloadFromState(state taskpkg.InspectSchedulerState) contract.TaskInspectSchedulerPayload {
	return contract.TaskInspectSchedulerPayload{
		Paused:    state.Paused,
		PausedBy:  state.PausedBy,
		PausedAt:  optionalTime(state.PausedAt),
		Reason:    state.Reason,
		UpdatedAt: optionalTime(state.UpdatedAt),
	}
}

func cloneOwnership(source *taskpkg.Ownership) *taskpkg.Ownership {
	if source == nil {
		return nil
	}
	return &taskpkg.Ownership{
		Kind: source.Kind.Normalize(),
		Ref:  strings.TrimSpace(source.Ref),
	}
}

func cloneActorIdentity(source *taskpkg.ActorIdentity) *taskpkg.ActorIdentity {
	if source == nil {
		return nil
	}
	return &taskpkg.ActorIdentity{
		Kind: source.Kind.Normalize(),
		Ref:  strings.TrimSpace(source.Ref),
	}
}

func trimStringPtr(source *string) *string {
	if source == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*source)
	return &trimmed
}

func normalizePriorityPtr(source *taskpkg.Priority) *taskpkg.Priority {
	if source == nil {
		return nil
	}
	normalized := source.Normalize()
	return &normalized
}

func normalizeApprovalPolicyPtr(source *taskpkg.ApprovalPolicy) *taskpkg.ApprovalPolicy {
	if source == nil {
		return nil
	}
	normalized := source.Normalize()
	return &normalized
}

func optionalTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	cloned := value
	return &cloned
}

func cloneRawMessagePtr(source *json.RawMessage) *json.RawMessage {
	if source == nil {
		return nil
	}
	copyValue := cloneRawMessage(*source)
	return &copyValue
}
