package core

import (
	"context"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/loop/dsl"
	taskpkg "github.com/compozy/agh/internal/task"
)

// LoopService exposes the daemon-owned Loop public API facade to transports.
type LoopService interface {
	ListLoops(ctx context.Context, workspaceID string) (contract.LoopsResponse, error)
	CreateLoop(ctx context.Context, workspaceID string, req contract.CreateLoopRequest) (contract.LoopResponse, error)
	GetLoop(ctx context.Context, workspaceID string, name string) (contract.LoopResponse, error)
	PatchLoop(
		ctx context.Context,
		workspaceID string,
		name string,
		req contract.PatchLoopRequest,
	) (contract.LoopResponse, error)
	ValidateLoop(
		ctx context.Context,
		workspaceID string,
		name string,
		req contract.ValidateLoopRequest,
	) (contract.LoopValidationResponse, error)
	DeleteLoop(ctx context.Context, workspaceID string, name string) error
	RunLoop(
		ctx context.Context,
		workspaceID string,
		name string,
		req contract.RunLoopRequest,
		startKind dsl.StartKind,
		actor taskpkg.ActorContext,
		dry bool,
	) (contract.RunLoopResponse, error)
	GetLoopConfig(ctx context.Context, workspaceID string, name string) (contract.LoopConfigResponse, error)
	PutLoopConfig(
		ctx context.Context,
		workspaceID string,
		name string,
		req contract.PutLoopConfigRequest,
	) (contract.LoopConfigResponse, error)
	GetLoopAnnotations(ctx context.Context, workspaceID string, name string) (contract.LoopAnnotationsResponse, error)
	PutLoopAnnotations(
		ctx context.Context,
		workspaceID string,
		name string,
		req contract.PutLoopAnnotationsRequest,
	) (contract.LoopAnnotationsResponse, error)
	ListLoopRuns(ctx context.Context, workspaceID string, query LoopRunListQuery) (contract.LoopRunsResponse, error)
	GetLoopRun(ctx context.Context, workspaceID string, runID string) (contract.LoopRunResponse, error)
	StopLoopRun(ctx context.Context, workspaceID string, runID string) error
	PauseLoopRun(ctx context.Context, workspaceID string, runID string) error
	ResumeLoopRun(ctx context.Context, workspaceID string, runID string) error
	ApproveLoopRun(
		ctx context.Context,
		workspaceID string,
		runID string,
		req contract.ApproveLoopRunRequest,
		actor taskpkg.ActorContext,
	) error
	ListLoopRunEvents(
		ctx context.Context,
		workspaceID string,
		runID string,
		afterSeq int64,
	) ([]contract.LoopRunEventPayload, error)
}

// LoopRunListQuery contains HTTP/UDS list filters for loop runs.
type LoopRunListQuery struct {
	LoopName string
	Status   string
	Limit    int
}
