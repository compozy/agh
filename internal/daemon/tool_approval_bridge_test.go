package daemon

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/compozy/agh/internal/acp"
	toolspkg "github.com/compozy/agh/internal/tools"
)

func TestToolApprovalBridgeDeterministicErrors(t *testing.T) {
	t.Parallel()

	view := toolApprovalTestView()

	t.Run("Should return approval_unreachable without a permission channel", func(t *testing.T) {
		t.Parallel()

		bridge := newToolApprovalBridge(nil, time.Second)
		err := bridge.RequestToolApproval(
			t.Context(),
			toolspkg.Scope{SessionID: "sess-1"},
			toolspkg.CallRequest{ToolID: view.Descriptor.ID, Input: []byte(`{}`)},
			&view,
		)
		requireToolApprovalReason(t, err, toolspkg.ReasonApprovalUnreachable)
	})

	t.Run("Should return approval_timed_out when ACP permission request exceeds timeout", func(t *testing.T) {
		t.Parallel()

		requester := &recordingPermissionRequester{
			fn: func(ctx context.Context, _ string, _ acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
				<-ctx.Done()
				return acp.RequestPermissionResponse{}, ctx.Err()
			},
		}
		bridge := newToolApprovalBridge(func() sessionPermissionRequester { return requester }, time.Nanosecond)
		err := bridge.RequestToolApproval(
			t.Context(),
			toolspkg.Scope{SessionID: "sess-1"},
			toolspkg.CallRequest{ToolID: view.Descriptor.ID, Input: []byte(`{}`)},
			&view,
		)
		requireToolApprovalReason(t, err, toolspkg.ReasonApprovalTimedOut)
	})

	t.Run("Should return approval_canceled when caller context is canceled", func(t *testing.T) {
		t.Parallel()

		requester := &recordingPermissionRequester{
			fn: func(ctx context.Context, _ string, _ acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
				return acp.RequestPermissionResponse{}, ctx.Err()
			},
		}
		bridge := newToolApprovalBridge(func() sessionPermissionRequester { return requester }, time.Second)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		err := bridge.RequestToolApproval(
			ctx,
			toolspkg.Scope{SessionID: "sess-1"},
			toolspkg.CallRequest{ToolID: view.Descriptor.ID, Input: []byte(`{}`)},
			&view,
		)
		requireToolApprovalReason(t, err, toolspkg.ReasonApprovalCanceled)
	})

	t.Run("Should return approval_canceled when ACP returns canceled outcome", func(t *testing.T) {
		t.Parallel()

		requester := &recordingPermissionRequester{
			response: acp.RequestPermissionResponse{Outcome: acpsdk.NewRequestPermissionOutcomeCancelled()},
		}
		bridge := newToolApprovalBridge(func() sessionPermissionRequester { return requester }, time.Second)
		err := bridge.RequestToolApproval(
			t.Context(),
			toolspkg.Scope{SessionID: "sess-1"},
			toolspkg.CallRequest{ToolID: view.Descriptor.ID, Input: []byte(`{}`)},
			&view,
		)
		requireToolApprovalReason(t, err, toolspkg.ReasonApprovalCanceled)
	})
}

func TestToolApprovalBridgeRoutesAllowAndRejectOutcomes(t *testing.T) {
	t.Parallel()

	view := toolApprovalTestView()

	t.Run("Should allow selected allow option", func(t *testing.T) {
		t.Parallel()

		requester := &recordingPermissionRequester{
			response: acp.RequestPermissionResponse{
				Outcome: acpsdk.NewRequestPermissionOutcomeSelected(toolApprovalAllowOnceID),
			},
		}
		bridge := newToolApprovalBridge(func() sessionPermissionRequester { return requester }, time.Second)
		err := bridge.RequestToolApproval(
			t.Context(),
			toolspkg.Scope{SessionID: "sess-1"},
			toolspkg.CallRequest{
				ToolID:      view.Descriptor.ID,
				ToolCallID:  "call-1",
				SessionID:   "sess-1",
				WorkspaceID: "ws-1",
				AgentName:   "codex",
				Input:       []byte(`{"message":"hello"}`),
			},
			&view,
		)
		if err != nil {
			t.Fatalf("RequestToolApproval() error = %v, want nil", err)
		}
		request := requester.lastRequest(t)
		if request.ToolCall.ToolCallId != "call-1" || request.SessionId != "sess-1" {
			t.Fatalf("permission request = %#v, want hosted call context", request)
		}
		if got, want := len(request.Options), 4; got != want {
			t.Fatalf("permission options = %#v, want %d options", request.Options, want)
		}
	})

	t.Run("Should reject selected reject option", func(t *testing.T) {
		t.Parallel()

		requester := &recordingPermissionRequester{
			response: acp.RequestPermissionResponse{
				Outcome: acpsdk.NewRequestPermissionOutcomeSelected(toolApprovalRejectOnceID),
			},
		}
		bridge := newToolApprovalBridge(func() sessionPermissionRequester { return requester }, time.Second)
		err := bridge.RequestToolApproval(
			t.Context(),
			toolspkg.Scope{SessionID: "sess-1"},
			toolspkg.CallRequest{ToolID: view.Descriptor.ID, Input: []byte(`{}`)},
			&view,
		)
		requireToolApprovalReason(t, err, toolspkg.ReasonApprovalRequired)
	})
}

func requireToolApprovalReason(t *testing.T, err error, want toolspkg.ReasonCode) {
	t.Helper()

	if !errors.Is(err, toolspkg.ErrToolApprovalRequired) {
		t.Fatalf("RequestToolApproval() error = %v, want ErrToolApprovalRequired", err)
	}
	var toolErr *toolspkg.ToolError
	if !errors.As(err, &toolErr) || !slices.Contains(toolErr.ReasonCodes, want) {
		t.Fatalf("approval error = %#v, want reason %q", err, want)
	}
}

func toolApprovalTestView() toolspkg.ToolView {
	return toolspkg.ToolView{
		Descriptor: toolspkg.Descriptor{
			ID:           "agh__approval_probe",
			Backend:      toolspkg.BackendRef{Kind: toolspkg.BackendNativeGo, NativeName: "approval_probe"},
			Description:  "approval probe",
			InputSchema:  []byte(`{"type":"object"}`),
			Source:       toolspkg.SourceRef{Kind: toolspkg.SourceBuiltin, Owner: "daemon"},
			Visibility:   toolspkg.VisibilityModel,
			Risk:         toolspkg.RiskMutating,
			Destructive:  false,
			ReadOnly:     false,
			OpenWorld:    false,
			DisplayTitle: "Approval Probe",
		},
		Decision: toolspkg.EffectiveToolDecision{
			VisibleToSession: true,
			Callable:         true,
			ApprovalRequired: true,
		},
	}
}

type recordingPermissionRequester struct {
	response acp.RequestPermissionResponse
	err      error
	fn       func(context.Context, string, acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error)
	requests []acp.RequestPermissionRequest
}

var _ sessionPermissionRequester = (*recordingPermissionRequester)(nil)

func (r *recordingPermissionRequester) RequestPermission(
	ctx context.Context,
	id string,
	req acp.RequestPermissionRequest,
) (acp.RequestPermissionResponse, error) {
	r.requests = append(r.requests, req)
	if r.fn != nil {
		return r.fn(ctx, id, req)
	}
	return r.response, r.err
}

func (r *recordingPermissionRequester) lastRequest(t *testing.T) acp.RequestPermissionRequest {
	t.Helper()

	if len(r.requests) == 0 {
		t.Fatal("RequestPermission was not invoked")
	}
	return r.requests[len(r.requests)-1]
}
