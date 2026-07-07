package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/compozy/agh/internal/api/contract"
	core "github.com/compozy/agh/internal/api/core"
	"github.com/compozy/agh/internal/loop/dsl"
	taskpkg "github.com/compozy/agh/internal/task"
	toolspkg "github.com/compozy/agh/internal/tools"
)

func TestDaemonNativeLoopTools(t *testing.T) {
	t.Parallel()

	t.Run("Should route dry run through loop service with native tool actor", func(t *testing.T) {
		t.Parallel()

		var capturedStartKind dsl.StartKind
		var capturedActor taskpkg.ActorContext
		var capturedDry bool
		registry := newDaemonNativeRegistry(t, &daemonNativeToolsDeps{
			Loops: func() core.LoopService {
				return &nativeLoopServiceStub{
					runLoopFn: func(
						_ context.Context,
						workspaceID string,
						name string,
						request contract.RunLoopRequest,
						startKind dsl.StartKind,
						actor taskpkg.ActorContext,
						dry bool,
					) (contract.RunLoopResponse, error) {
						if workspaceID != "ws-alpha" || name != "release" {
							t.Fatalf("RunLoop target = %s/%s, want ws-alpha/release", workspaceID, name)
						}
						if request.Inputs["target"] != "prod" {
							t.Fatalf("RunLoop inputs = %#v, want target prod", request.Inputs)
						}
						capturedStartKind = startKind
						capturedActor = actor
						capturedDry = dry
						return contract.RunLoopResponse{
							DryRun: &contract.LoopPlanPayload{LoopName: "release", ResolvedInputs: request.Inputs},
						}, nil
					},
				}
			},
		}, nativeApproveAllPolicyInputs())

		result, err := registry.Call(
			t.Context(),
			toolspkg.Scope{SessionID: "sess-caller"},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDLoopRun,
				Input: json.RawMessage(
					`{"workspace_id":"ws-alpha","name":"release","inputs":{"target":"prod"},"dry":true}`,
				),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(loop_run) error = %v", err)
		}

		if capturedStartKind != dsl.StartNativeTool {
			t.Fatalf("startKind = %q, want native_tool", capturedStartKind)
		}
		if capturedActor.Actor.Kind != taskpkg.ActorKindAgentSession || capturedActor.Actor.Ref != "sess-caller" {
			t.Fatalf("actor = %#v, want agent session sess-caller", capturedActor.Actor)
		}
		if !capturedDry {
			t.Fatal("dry = false, want true")
		}
		requireNativeStructuredContains(t, result, []byte(`"dry_run"`))
		requireNativeStructuredContains(t, result, []byte(`"release"`))
	})

	t.Run("Should keep loop tools unavailable until service is ready", func(t *testing.T) {
		t.Parallel()

		var loopSvc core.LoopService
		registry := newDaemonNativeRegistry(t, &daemonNativeToolsDeps{
			Loops: func() core.LoopService { return loopSvc },
		}, nativeApproveAllPolicyInputs())
		scope := toolspkg.Scope{Operator: true}

		views, err := registry.OperatorProjection(t.Context(), scope)
		if err != nil {
			t.Fatalf("OperatorProjection(before service) error = %v", err)
		}
		requireNativeToolUnavailableReason(t, views, toolspkg.ToolIDLoopList)
		requireNativeToolUnavailableReason(t, views, toolspkg.ToolIDLoopApprove)

		loopSvc = &nativeLoopServiceStub{
			listLoopsFn: func(context.Context, string) (contract.LoopsResponse, error) {
				return contract.LoopsResponse{}, nil
			},
		}
		views, err = registry.OperatorProjection(t.Context(), scope)
		if err != nil {
			t.Fatalf("OperatorProjection(after service) error = %v", err)
		}
		requireNativeToolAvailable(t, views, toolspkg.ToolIDLoopList)
	})

	t.Run("Should surface shared service self approval denial", func(t *testing.T) {
		t.Parallel()

		approveCalled := false
		registry := newDaemonNativeRegistry(t, &daemonNativeToolsDeps{
			Loops: func() core.LoopService {
				return &nativeLoopServiceStub{
					approveLoopRunFn: func(context.Context, string, string, contract.ApproveLoopRunRequest, taskpkg.ActorContext) error {
						approveCalled = true
						return taskpkg.ErrPermissionDenied
					},
				}
			},
		}, nativeApproveAllPolicyInputs())

		_, err := registry.Call(
			t.Context(),
			toolspkg.Scope{SessionID: "sess-author"},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDLoopApprove,
				Input: json.RawMessage(
					`{"workspace_id":"ws-alpha","run_id":"looprun-1","gate_id":"human",` +
						`"decision":"approve","approval_token_hash":"sha256:` +
						strings.Repeat("a", 64) +
						`"}`,
				),
			},
		)

		var toolErr *toolspkg.ToolError
		if !errors.As(err, &toolErr) {
			t.Fatalf("Registry.Call(loop_approve) error = %v, want ToolError", err)
		}
		if !slices.Contains(toolErr.ReasonCodes, toolspkg.ReasonApprovalSelfDenied) ||
			!slices.Contains(toolErr.ReasonCodes, toolspkg.ReasonPolicyDenied) {
			t.Fatalf("ReasonCodes = %#v, want self approval and policy denial", toolErr.ReasonCodes)
		}
		if !approveCalled {
			t.Fatal("ApproveLoopRun was not called")
		}
	})

	t.Run("Should reject raw approval tokens without echoing them", func(t *testing.T) {
		t.Parallel()

		registry := newDaemonNativeRegistry(t, &daemonNativeToolsDeps{
			Loops: func() core.LoopService { return &nativeLoopServiceStub{} },
		}, nativeApproveAllPolicyInputs())

		_, err := registry.Call(
			t.Context(),
			toolspkg.Scope{SessionID: "sess-reviewer"},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDLoopApprove,
				Input: json.RawMessage(
					`{"workspace_id":"ws-alpha","run_id":"looprun-1","gate_id":"human","decision":"approve","approval_token_hash":"raw-secret-token"}`,
				),
			},
		)

		var toolErr *toolspkg.ToolError
		if !errors.As(err, &toolErr) {
			t.Fatalf("Registry.Call(loop_approve raw token) error = %v, want ToolError", err)
		}
		if toolErr.Code != toolspkg.ErrorCodeInvalidInput ||
			!slices.Contains(toolErr.ReasonCodes, toolspkg.ReasonSchemaInvalid) {
			t.Fatalf("tool error = %#v, want invalid input schema error", toolErr)
		}
		if strings.Contains(toolErr.Error(), "raw-secret-token") {
			t.Fatalf("tool error leaked raw token: %v", toolErr)
		}
	})
}

type nativeLoopServiceStub struct {
	listLoopsFn         func(context.Context, string) (contract.LoopsResponse, error)
	createLoopFn        func(context.Context, string, contract.CreateLoopRequest) (contract.LoopResponse, error)
	getLoopFn           func(context.Context, string, string) (contract.LoopResponse, error)
	patchLoopFn         func(context.Context, string, string, contract.PatchLoopRequest) (contract.LoopResponse, error)
	validateLoopFn      func(context.Context, string, string, contract.ValidateLoopRequest) (contract.LoopValidationResponse, error)
	deleteLoopFn        func(context.Context, string, string) error
	runLoopFn           func(context.Context, string, string, contract.RunLoopRequest, dsl.StartKind, taskpkg.ActorContext, bool) (contract.RunLoopResponse, error)
	getLoopConfigFn     func(context.Context, string, string) (contract.LoopConfigResponse, error)
	putLoopConfigFn     func(context.Context, string, string, contract.PutLoopConfigRequest) (contract.LoopConfigResponse, error)
	listLoopRunsFn      func(context.Context, string, core.LoopRunListQuery) (contract.LoopRunsResponse, error)
	getLoopRunFn        func(context.Context, string, string) (contract.LoopRunResponse, error)
	stopLoopRunFn       func(context.Context, string, string) error
	pauseLoopRunFn      func(context.Context, string, string) error
	resumeLoopRunFn     func(context.Context, string, string) error
	approveLoopRunFn    func(context.Context, string, string, contract.ApproveLoopRunRequest, taskpkg.ActorContext) error
	listLoopRunEventsFn func(context.Context, string, string, int64) ([]contract.LoopRunEventPayload, error)
}

var _ core.LoopService = (*nativeLoopServiceStub)(nil)

func (s *nativeLoopServiceStub) ListLoops(ctx context.Context, workspaceID string) (contract.LoopsResponse, error) {
	if s.listLoopsFn != nil {
		return s.listLoopsFn(ctx, workspaceID)
	}
	return contract.LoopsResponse{}, errors.New("unexpected ListLoops call")
}

func (s *nativeLoopServiceStub) CreateLoop(
	ctx context.Context,
	workspaceID string,
	req contract.CreateLoopRequest,
) (contract.LoopResponse, error) {
	if s.createLoopFn != nil {
		return s.createLoopFn(ctx, workspaceID, req)
	}
	return contract.LoopResponse{}, errors.New("unexpected CreateLoop call")
}

func (s *nativeLoopServiceStub) GetLoop(
	ctx context.Context,
	workspaceID string,
	name string,
) (contract.LoopResponse, error) {
	if s.getLoopFn != nil {
		return s.getLoopFn(ctx, workspaceID, name)
	}
	return contract.LoopResponse{}, errors.New("unexpected GetLoop call")
}

func (s *nativeLoopServiceStub) PatchLoop(
	ctx context.Context,
	workspaceID string,
	name string,
	req contract.PatchLoopRequest,
) (contract.LoopResponse, error) {
	if s.patchLoopFn != nil {
		return s.patchLoopFn(ctx, workspaceID, name, req)
	}
	return contract.LoopResponse{}, errors.New("unexpected PatchLoop call")
}

func (s *nativeLoopServiceStub) ValidateLoop(
	ctx context.Context,
	workspaceID string,
	name string,
	req contract.ValidateLoopRequest,
) (contract.LoopValidationResponse, error) {
	if s.validateLoopFn != nil {
		return s.validateLoopFn(ctx, workspaceID, name, req)
	}
	return contract.LoopValidationResponse{}, errors.New("unexpected ValidateLoop call")
}

func (s *nativeLoopServiceStub) DeleteLoop(ctx context.Context, workspaceID string, name string) error {
	if s.deleteLoopFn != nil {
		return s.deleteLoopFn(ctx, workspaceID, name)
	}
	return errors.New("unexpected DeleteLoop call")
}

func (s *nativeLoopServiceStub) RunLoop(
	ctx context.Context,
	workspaceID string,
	name string,
	req contract.RunLoopRequest,
	startKind dsl.StartKind,
	actor taskpkg.ActorContext,
	dry bool,
) (contract.RunLoopResponse, error) {
	if s.runLoopFn != nil {
		return s.runLoopFn(ctx, workspaceID, name, req, startKind, actor, dry)
	}
	return contract.RunLoopResponse{}, errors.New("unexpected RunLoop call")
}

func (s *nativeLoopServiceStub) GetLoopConfig(
	ctx context.Context,
	workspaceID string,
	name string,
) (contract.LoopConfigResponse, error) {
	if s.getLoopConfigFn != nil {
		return s.getLoopConfigFn(ctx, workspaceID, name)
	}
	return contract.LoopConfigResponse{}, errors.New("unexpected GetLoopConfig call")
}

func (s *nativeLoopServiceStub) PutLoopConfig(
	ctx context.Context,
	workspaceID string,
	name string,
	req contract.PutLoopConfigRequest,
) (contract.LoopConfigResponse, error) {
	if s.putLoopConfigFn != nil {
		return s.putLoopConfigFn(ctx, workspaceID, name, req)
	}
	return contract.LoopConfigResponse{}, errors.New("unexpected PutLoopConfig call")
}

func (s *nativeLoopServiceStub) GetLoopAnnotations(
	context.Context,
	string,
	string,
) (contract.LoopAnnotationsResponse, error) {
	return contract.LoopAnnotationsResponse{}, errors.New("unexpected GetLoopAnnotations call")
}

func (s *nativeLoopServiceStub) PutLoopAnnotations(
	context.Context,
	string,
	string,
	contract.PutLoopAnnotationsRequest,
) (contract.LoopAnnotationsResponse, error) {
	return contract.LoopAnnotationsResponse{}, errors.New("unexpected PutLoopAnnotations call")
}

func (s *nativeLoopServiceStub) ListLoopRuns(
	ctx context.Context,
	workspaceID string,
	query core.LoopRunListQuery,
) (contract.LoopRunsResponse, error) {
	if s.listLoopRunsFn != nil {
		return s.listLoopRunsFn(ctx, workspaceID, query)
	}
	return contract.LoopRunsResponse{}, errors.New("unexpected ListLoopRuns call")
}

func (s *nativeLoopServiceStub) GetLoopRun(
	ctx context.Context,
	workspaceID string,
	runID string,
) (contract.LoopRunResponse, error) {
	if s.getLoopRunFn != nil {
		return s.getLoopRunFn(ctx, workspaceID, runID)
	}
	return contract.LoopRunResponse{}, errors.New("unexpected GetLoopRun call")
}

func (s *nativeLoopServiceStub) StopLoopRun(ctx context.Context, workspaceID string, runID string) error {
	if s.stopLoopRunFn != nil {
		return s.stopLoopRunFn(ctx, workspaceID, runID)
	}
	return errors.New("unexpected StopLoopRun call")
}

func (s *nativeLoopServiceStub) PauseLoopRun(ctx context.Context, workspaceID string, runID string) error {
	if s.pauseLoopRunFn != nil {
		return s.pauseLoopRunFn(ctx, workspaceID, runID)
	}
	return errors.New("unexpected PauseLoopRun call")
}

func (s *nativeLoopServiceStub) ResumeLoopRun(ctx context.Context, workspaceID string, runID string) error {
	if s.resumeLoopRunFn != nil {
		return s.resumeLoopRunFn(ctx, workspaceID, runID)
	}
	return errors.New("unexpected ResumeLoopRun call")
}

func (s *nativeLoopServiceStub) ApproveLoopRun(
	ctx context.Context,
	workspaceID string,
	runID string,
	req contract.ApproveLoopRunRequest,
	actor taskpkg.ActorContext,
) error {
	if s.approveLoopRunFn != nil {
		return s.approveLoopRunFn(ctx, workspaceID, runID, req, actor)
	}
	return errors.New("unexpected ApproveLoopRun call")
}

func (s *nativeLoopServiceStub) ListLoopRunEvents(
	ctx context.Context,
	workspaceID string,
	runID string,
	afterSeq int64,
) ([]contract.LoopRunEventPayload, error) {
	if s.listLoopRunEventsFn != nil {
		return s.listLoopRunEventsFn(ctx, workspaceID, runID, afterSeq)
	}
	return nil, errors.New("unexpected ListLoopRunEvents call")
}
