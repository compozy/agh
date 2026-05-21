package testutil

import (
	"context"

	core "github.com/pedronauck/agh/internal/api/core"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

type StubTaskManager struct {
	CreateTaskFn      func(context.Context, taskpkg.CreateTask, taskpkg.ActorContext) (*taskpkg.Task, error)
	CreateChildTaskFn func(
		context.Context,
		string,
		taskpkg.CreateTask,
		taskpkg.ActorContext,
	) (*taskpkg.Task, error)
	DeleteTaskFn  func(context.Context, string, taskpkg.ActorContext) error
	UpdateTaskFn  func(context.Context, string, taskpkg.Patch, taskpkg.ActorContext) (*taskpkg.Task, error)
	PublishTaskFn func(
		context.Context,
		string,
		taskpkg.ExecutionRequest,
		taskpkg.ActorContext,
	) (*taskpkg.Execution, error)
	StartTaskFn func(
		context.Context,
		string,
		taskpkg.ExecutionRequest,
		taskpkg.ActorContext,
	) (*taskpkg.Execution, error)
	ApproveTaskFn func(
		context.Context,
		string,
		taskpkg.ExecutionRequest,
		taskpkg.ActorContext,
	) (*taskpkg.Execution, error)
	RejectTaskFn func(context.Context, string, taskpkg.ActorContext) (*taskpkg.Task, error)
	CancelTaskFn func(
		context.Context,
		string,
		taskpkg.CancelTask,
		taskpkg.ActorContext,
	) (*taskpkg.Task, error)
	MarkTaskReadFn        func(context.Context, string, taskpkg.ActorContext) (taskpkg.TriageState, error)
	ArchiveTaskFn         func(context.Context, string, taskpkg.ActorContext) (taskpkg.TriageState, error)
	DismissTaskFn         func(context.Context, string, taskpkg.ActorContext) (taskpkg.TriageState, error)
	GetExecutionProfileFn func(
		context.Context,
		string,
		taskpkg.ActorContext,
	) (taskpkg.ExecutionProfile, error)
	SetExecutionProfileFn func(
		context.Context,
		string,
		*taskpkg.ExecutionProfile,
		taskpkg.ActorContext,
	) (taskpkg.ExecutionProfile, error)
	DeleteExecutionProfileFn func(context.Context, string, taskpkg.ActorContext) error
	RequestRunReviewFn       func(
		context.Context,
		taskpkg.RunReviewRequest,
		taskpkg.ActorContext,
	) (taskpkg.RunReview, bool, error)
	GetRunReviewFn    func(context.Context, string, taskpkg.ActorContext) (taskpkg.RunReview, error)
	RecordRunReviewFn func(
		context.Context,
		taskpkg.RecordRunReviewRequest,
		taskpkg.ActorContext,
	) (taskpkg.RunReviewResult, error)
	BindRunReviewSessionFn func(
		context.Context,
		taskpkg.BindRunReviewSessionRequest,
		taskpkg.ActorContext,
	) (taskpkg.RunReviewBinding, error)
	LookupRunReviewForSessionFn func(
		context.Context,
		string,
		taskpkg.ActorContext,
	) (taskpkg.RunReviewBinding, error)
	ListRunReviewsFn   func(context.Context, taskpkg.RunReviewQuery, taskpkg.ActorContext) ([]taskpkg.RunReview, error)
	AddDependencyFn    func(context.Context, taskpkg.AddDependency, taskpkg.ActorContext) error
	RemoveDependencyFn func(context.Context, string, string, taskpkg.ActorContext) error
	EnqueueRunFn       func(context.Context, taskpkg.EnqueueRun, taskpkg.ActorContext) (*taskpkg.Run, error)
	ClaimNextRunFn     func(
		context.Context,
		taskpkg.ClaimCriteria,
		taskpkg.ActorContext,
	) (*taskpkg.ClaimResult, error)
	ClaimRunFn                  func(context.Context, string, taskpkg.ClaimRun, taskpkg.ActorContext) (*taskpkg.Run, error)
	StartRunFn                  func(context.Context, string, taskpkg.StartRun, taskpkg.ActorContext) (*taskpkg.Run, error)
	AttachRunSessionFn          func(context.Context, string, string, taskpkg.ActorContext) (*taskpkg.Run, error)
	HeartbeatRunLeaseFn         func(context.Context, taskpkg.LeaseHeartbeat, taskpkg.ActorContext) (*taskpkg.Run, error)
	ReleaseRunLeaseFn           func(context.Context, taskpkg.LeaseRelease, taskpkg.ActorContext) (*taskpkg.Run, error)
	CompleteRunLeaseFn          func(context.Context, taskpkg.LeaseCompletion, taskpkg.ActorContext) (*taskpkg.Run, error)
	FailRunLeaseFn              func(context.Context, taskpkg.LeaseFailure, taskpkg.ActorContext) (*taskpkg.Run, error)
	LookupActiveRunForSessionFn func(
		context.Context,
		string,
		string,
	) (taskpkg.AutonomyLeaseHandle, error)
	CompleteRunFn             func(context.Context, string, taskpkg.RunResult, taskpkg.ActorContext) (*taskpkg.Run, error)
	FailRunFn                 func(context.Context, string, taskpkg.RunFailure, taskpkg.ActorContext) (*taskpkg.Run, error)
	CancelRunFn               func(context.Context, string, taskpkg.CancelRun, taskpkg.ActorContext) (*taskpkg.Run, error)
	RecoverExpiredRunLeasesFn func(
		context.Context,
		taskpkg.ExpiredLeaseRecovery,
		taskpkg.ActorContext,
	) ([]taskpkg.ExpiredLeaseRecoveryResult, error)
	GetTaskFn      func(context.Context, string, taskpkg.ActorContext) (*taskpkg.View, error)
	InspectTaskFn  func(context.Context, string, taskpkg.ActorContext) (*taskpkg.InspectView, error)
	InspectRunFn   func(context.Context, string, taskpkg.ActorContext) (*taskpkg.InspectView, error)
	ListTaskRunsFn func(context.Context, string, taskpkg.RunQuery, taskpkg.ActorContext) ([]taskpkg.Run, error)
	ListTasksFn    func(context.Context, taskpkg.Query, taskpkg.ActorContext) ([]taskpkg.Summary, error)
	TimelineFn     func(
		context.Context,
		string,
		taskpkg.TimelineQuery,
		taskpkg.ActorContext,
	) ([]taskpkg.TimelineItem, error)
	StreamFn func(
		context.Context,
		string,
		taskpkg.StreamQuery,
		taskpkg.ActorContext,
	) (<-chan taskpkg.StreamEvent, error)
	TreeFn      func(context.Context, string, taskpkg.ActorContext) (*taskpkg.TreeView, error)
	RunDetailFn func(context.Context, string, taskpkg.ActorContext) (*taskpkg.RunDetailView, error)
}

func (s StubTaskManager) CreateTask(
	ctx context.Context,
	spec taskpkg.CreateTask,
	actor taskpkg.ActorContext,
) (*taskpkg.Task, error) {
	if s.CreateTaskFn != nil {
		return s.CreateTaskFn(ctx, spec, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) CreateChildTask(
	ctx context.Context,
	parentTaskID string,
	spec taskpkg.CreateTask,
	actor taskpkg.ActorContext,
) (*taskpkg.Task, error) {
	if s.CreateChildTaskFn != nil {
		return s.CreateChildTaskFn(ctx, parentTaskID, spec, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) DeleteTask(
	ctx context.Context,
	id string,
	actor taskpkg.ActorContext,
) error {
	if s.DeleteTaskFn != nil {
		return s.DeleteTaskFn(ctx, id, actor)
	}
	return taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) UpdateTask(
	ctx context.Context,
	id string,
	patch taskpkg.Patch,
	actor taskpkg.ActorContext,
) (*taskpkg.Task, error) {
	if s.UpdateTaskFn != nil {
		return s.UpdateTaskFn(ctx, id, patch, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) PublishTask(
	ctx context.Context,
	id string,
	req taskpkg.ExecutionRequest,
	actor taskpkg.ActorContext,
) (*taskpkg.Execution, error) {
	if s.PublishTaskFn != nil {
		return s.PublishTaskFn(ctx, id, req, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) StartTask(
	ctx context.Context,
	id string,
	req taskpkg.ExecutionRequest,
	actor taskpkg.ActorContext,
) (*taskpkg.Execution, error) {
	if s.StartTaskFn != nil {
		return s.StartTaskFn(ctx, id, req, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) ApproveTask(
	ctx context.Context,
	id string,
	req taskpkg.ExecutionRequest,
	actor taskpkg.ActorContext,
) (*taskpkg.Execution, error) {
	if s.ApproveTaskFn != nil {
		return s.ApproveTaskFn(ctx, id, req, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) RejectTask(
	ctx context.Context,
	id string,
	actor taskpkg.ActorContext,
) (*taskpkg.Task, error) {
	if s.RejectTaskFn != nil {
		return s.RejectTaskFn(ctx, id, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) CancelTask(
	ctx context.Context,
	id string,
	req taskpkg.CancelTask,
	actor taskpkg.ActorContext,
) (*taskpkg.Task, error) {
	if s.CancelTaskFn != nil {
		return s.CancelTaskFn(ctx, id, req, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) MarkTaskRead(
	ctx context.Context,
	id string,
	actor taskpkg.ActorContext,
) (taskpkg.TriageState, error) {
	if s.MarkTaskReadFn != nil {
		return s.MarkTaskReadFn(ctx, id, actor)
	}
	return taskpkg.TriageState{}, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) ArchiveTask(
	ctx context.Context,
	id string,
	actor taskpkg.ActorContext,
) (taskpkg.TriageState, error) {
	if s.ArchiveTaskFn != nil {
		return s.ArchiveTaskFn(ctx, id, actor)
	}
	return taskpkg.TriageState{}, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) DismissTask(
	ctx context.Context,
	id string,
	actor taskpkg.ActorContext,
) (taskpkg.TriageState, error) {
	if s.DismissTaskFn != nil {
		return s.DismissTaskFn(ctx, id, actor)
	}
	return taskpkg.TriageState{}, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) GetExecutionProfile(
	ctx context.Context,
	taskID string,
	actor taskpkg.ActorContext,
) (taskpkg.ExecutionProfile, error) {
	if s.GetExecutionProfileFn != nil {
		return s.GetExecutionProfileFn(ctx, taskID, actor)
	}
	return taskpkg.ExecutionProfile{}, taskpkg.ErrExecutionProfileNotFound
}

func (s StubTaskManager) SetExecutionProfile(
	ctx context.Context,
	taskID string,
	profile *taskpkg.ExecutionProfile,
	actor taskpkg.ActorContext,
) (taskpkg.ExecutionProfile, error) {
	if s.SetExecutionProfileFn != nil {
		return s.SetExecutionProfileFn(ctx, taskID, profile, actor)
	}
	return taskpkg.ExecutionProfile{}, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) DeleteExecutionProfile(
	ctx context.Context,
	taskID string,
	actor taskpkg.ActorContext,
) error {
	if s.DeleteExecutionProfileFn != nil {
		return s.DeleteExecutionProfileFn(ctx, taskID, actor)
	}
	return taskpkg.ErrExecutionProfileNotFound
}

func (s StubTaskManager) RequestRunReview(
	ctx context.Context,
	req taskpkg.RunReviewRequest,
	actor taskpkg.ActorContext,
) (taskpkg.RunReview, bool, error) {
	if s.RequestRunReviewFn != nil {
		return s.RequestRunReviewFn(ctx, req, actor)
	}
	return taskpkg.RunReview{}, false, taskpkg.ErrRunReviewNotFound
}

func (s StubTaskManager) GetRunReview(
	ctx context.Context,
	reviewID string,
	actor taskpkg.ActorContext,
) (taskpkg.RunReview, error) {
	if s.GetRunReviewFn != nil {
		return s.GetRunReviewFn(ctx, reviewID, actor)
	}
	return taskpkg.RunReview{}, taskpkg.ErrRunReviewNotFound
}

func (s StubTaskManager) RecordRunReview(
	ctx context.Context,
	req taskpkg.RecordRunReviewRequest,
	actor taskpkg.ActorContext,
) (taskpkg.RunReviewResult, error) {
	if s.RecordRunReviewFn != nil {
		return s.RecordRunReviewFn(ctx, req, actor)
	}
	return taskpkg.RunReviewResult{}, taskpkg.ErrRunReviewNotFound
}

func (s StubTaskManager) BindRunReviewSession(
	ctx context.Context,
	req taskpkg.BindRunReviewSessionRequest,
	actor taskpkg.ActorContext,
) (taskpkg.RunReviewBinding, error) {
	if s.BindRunReviewSessionFn != nil {
		return s.BindRunReviewSessionFn(ctx, req, actor)
	}
	return taskpkg.RunReviewBinding{}, taskpkg.ErrRunReviewNotFound
}

func (s StubTaskManager) LookupRunReviewForSession(
	ctx context.Context,
	sessionID string,
	actor taskpkg.ActorContext,
) (taskpkg.RunReviewBinding, error) {
	if s.LookupRunReviewForSessionFn != nil {
		return s.LookupRunReviewForSessionFn(ctx, sessionID, actor)
	}
	return taskpkg.RunReviewBinding{}, taskpkg.ErrRunReviewNotFound
}

func (s StubTaskManager) ListRunReviews(
	ctx context.Context,
	query taskpkg.RunReviewQuery,
	actor taskpkg.ActorContext,
) ([]taskpkg.RunReview, error) {
	if s.ListRunReviewsFn != nil {
		return s.ListRunReviewsFn(ctx, query, actor)
	}
	return nil, nil
}

func (s StubTaskManager) AddDependency(
	ctx context.Context,
	spec taskpkg.AddDependency,
	actor taskpkg.ActorContext,
) error {
	if s.AddDependencyFn != nil {
		return s.AddDependencyFn(ctx, spec, actor)
	}
	return taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) RemoveDependency(
	ctx context.Context,
	taskID string,
	dependsOnID string,
	actor taskpkg.ActorContext,
) error {
	if s.RemoveDependencyFn != nil {
		return s.RemoveDependencyFn(ctx, taskID, dependsOnID, actor)
	}
	return taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) EnqueueRun(
	ctx context.Context,
	spec taskpkg.EnqueueRun,
	actor taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	if s.EnqueueRunFn != nil {
		return s.EnqueueRunFn(ctx, spec, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) ClaimNextRun(
	ctx context.Context,
	criteria taskpkg.ClaimCriteria,
	actor taskpkg.ActorContext,
) (*taskpkg.ClaimResult, error) {
	if s.ClaimNextRunFn != nil {
		return s.ClaimNextRunFn(ctx, criteria, actor)
	}
	return nil, taskpkg.ErrNoClaimableRun
}

func (s StubTaskManager) ClaimRun(
	ctx context.Context,
	runID string,
	claim taskpkg.ClaimRun,
	actor taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	if s.ClaimRunFn != nil {
		return s.ClaimRunFn(ctx, runID, claim, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (s StubTaskManager) StartRun(
	ctx context.Context,
	runID string,
	req taskpkg.StartRun,
	actor taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	if s.StartRunFn != nil {
		return s.StartRunFn(ctx, runID, req, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (s StubTaskManager) AttachRunSession(
	ctx context.Context,
	runID string,
	sessionID string,
	actor taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	if s.AttachRunSessionFn != nil {
		return s.AttachRunSessionFn(ctx, runID, sessionID, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (s StubTaskManager) HeartbeatRunLease(
	ctx context.Context,
	heartbeat taskpkg.LeaseHeartbeat,
	actor taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	if s.HeartbeatRunLeaseFn != nil {
		return s.HeartbeatRunLeaseFn(ctx, heartbeat, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (s StubTaskManager) ReleaseRunLease(
	ctx context.Context,
	release taskpkg.LeaseRelease,
	actor taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	if s.ReleaseRunLeaseFn != nil {
		return s.ReleaseRunLeaseFn(ctx, release, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (s StubTaskManager) CompleteRunLease(
	ctx context.Context,
	completion taskpkg.LeaseCompletion,
	actor taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	if s.CompleteRunLeaseFn != nil {
		return s.CompleteRunLeaseFn(ctx, completion, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (s StubTaskManager) FailRunLease(
	ctx context.Context,
	failure taskpkg.LeaseFailure,
	actor taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	if s.FailRunLeaseFn != nil {
		return s.FailRunLeaseFn(ctx, failure, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (s StubTaskManager) LookupActiveRunForSession(
	ctx context.Context,
	sessionID string,
	runID string,
) (taskpkg.AutonomyLeaseHandle, error) {
	if s.LookupActiveRunForSessionFn != nil {
		return s.LookupActiveRunForSessionFn(ctx, sessionID, runID)
	}
	return taskpkg.AutonomyLeaseHandle{}, taskpkg.ErrTaskRunNotFound
}

func (s StubTaskManager) CompleteRun(
	ctx context.Context,
	runID string,
	result taskpkg.RunResult,
	actor taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	if s.CompleteRunFn != nil {
		return s.CompleteRunFn(ctx, runID, result, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (s StubTaskManager) FailRun(
	ctx context.Context,
	runID string,
	failure taskpkg.RunFailure,
	actor taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	if s.FailRunFn != nil {
		return s.FailRunFn(ctx, runID, failure, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (s StubTaskManager) CancelRun(
	ctx context.Context,
	runID string,
	req taskpkg.CancelRun,
	actor taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	if s.CancelRunFn != nil {
		return s.CancelRunFn(ctx, runID, req, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (s StubTaskManager) RecoverExpiredRunLeases(
	ctx context.Context,
	recovery taskpkg.ExpiredLeaseRecovery,
	actor taskpkg.ActorContext,
) ([]taskpkg.ExpiredLeaseRecoveryResult, error) {
	if s.RecoverExpiredRunLeasesFn != nil {
		return s.RecoverExpiredRunLeasesFn(ctx, recovery, actor)
	}
	return nil, nil
}

func (s StubTaskManager) GetTask(
	ctx context.Context,
	id string,
	actor taskpkg.ActorContext,
) (*taskpkg.View, error) {
	if s.GetTaskFn != nil {
		return s.GetTaskFn(ctx, id, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) InspectTask(
	ctx context.Context,
	taskID string,
	actor taskpkg.ActorContext,
) (*taskpkg.InspectView, error) {
	if s.InspectTaskFn != nil {
		return s.InspectTaskFn(ctx, taskID, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) InspectRun(
	ctx context.Context,
	runID string,
	actor taskpkg.ActorContext,
) (*taskpkg.InspectView, error) {
	if s.InspectRunFn != nil {
		return s.InspectRunFn(ctx, runID, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (s StubTaskManager) ListTaskRuns(
	ctx context.Context,
	taskID string,
	query taskpkg.RunQuery,
	actor taskpkg.ActorContext,
) ([]taskpkg.Run, error) {
	if s.ListTaskRunsFn != nil {
		return s.ListTaskRunsFn(ctx, taskID, query, actor)
	}
	return nil, nil
}

func (s StubTaskManager) ListTasks(
	ctx context.Context,
	query taskpkg.Query,
	actor taskpkg.ActorContext,
) ([]taskpkg.Summary, error) {
	if s.ListTasksFn != nil {
		return s.ListTasksFn(ctx, query, actor)
	}
	return nil, nil
}

func (s StubTaskManager) Timeline(
	ctx context.Context,
	taskID string,
	query taskpkg.TimelineQuery,
	actor taskpkg.ActorContext,
) ([]taskpkg.TimelineItem, error) {
	if s.TimelineFn != nil {
		return s.TimelineFn(ctx, taskID, query, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) Stream(
	ctx context.Context,
	taskID string,
	query taskpkg.StreamQuery,
	actor taskpkg.ActorContext,
) (<-chan taskpkg.StreamEvent, error) {
	if s.StreamFn != nil {
		return s.StreamFn(ctx, taskID, query, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) Tree(
	ctx context.Context,
	taskID string,
	actor taskpkg.ActorContext,
) (*taskpkg.TreeView, error) {
	if s.TreeFn != nil {
		return s.TreeFn(ctx, taskID, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) RunDetail(
	ctx context.Context,
	runID string,
	actor taskpkg.ActorContext,
) (*taskpkg.RunDetailView, error) {
	if s.RunDetailFn != nil {
		return s.RunDetailFn(ctx, runID, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

var _ core.TaskService = (*StubTaskManager)(nil)
