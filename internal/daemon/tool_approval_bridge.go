package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/pedronauck/agh/internal/acp"
	toolspkg "github.com/pedronauck/agh/internal/tools"
)

const (
	toolApprovalAllowOnceID    acpsdk.PermissionOptionId = "allow_once"
	toolApprovalAllowAlwaysID  acpsdk.PermissionOptionId = "allow_always"
	toolApprovalRejectOnceID   acpsdk.PermissionOptionId = "reject_once"
	toolApprovalRejectAlwaysID acpsdk.PermissionOptionId = "reject_always"
)

type sessionPermissionRequester interface {
	RequestPermission(
		ctx context.Context,
		id string,
		req acp.RequestPermissionRequest,
	) (acp.RequestPermissionResponse, error)
}

type toolApprovalBridge struct {
	sessions  func() sessionPermissionRequester
	timeout   time.Duration
	approvals toolspkg.ApprovalTokenConsumer
}

var _ toolspkg.ApprovalBridge = (*toolApprovalBridge)(nil)

func newToolApprovalBridge(
	sessions func() sessionPermissionRequester,
	timeout time.Duration,
	approvals ...toolspkg.ApprovalTokenConsumer,
) *toolApprovalBridge {
	bridge := &toolApprovalBridge{sessions: sessions, timeout: timeout}
	if len(approvals) > 0 {
		bridge.approvals = approvals[0]
	}
	return bridge
}

func (b *toolApprovalBridge) RequestToolApproval(
	ctx context.Context,
	scope toolspkg.Scope,
	call toolspkg.CallRequest,
	view *toolspkg.ToolView,
) error {
	toolID := toolApprovalID(call, view)
	if handled, err := b.consumeLocalToolApproval(ctx, scope, call); handled {
		return err
	}
	if b == nil || b.sessions == nil {
		return toolApprovalError(
			toolID,
			"tool approval channel is unreachable",
			toolspkg.ReasonApprovalUnreachable,
		)
	}
	sessions := b.sessions()
	if sessions == nil {
		return toolApprovalError(
			toolID,
			"tool approval channel is unreachable",
			toolspkg.ReasonApprovalUnreachable,
		)
	}
	sessionID := strings.TrimSpace(call.SessionID)
	if sessionID == "" {
		sessionID = strings.TrimSpace(scope.SessionID)
	}
	if sessionID == "" {
		return toolApprovalError(
			toolID,
			"tool approval session is unavailable",
			toolspkg.ReasonApprovalUnreachable,
		)
	}
	descriptor := toolApprovalDescriptor(call, view)
	response, err := b.requestSessionToolApproval(ctx, sessions, sessionID, call, descriptor, view)
	if err != nil {
		return err
	}
	return toolApprovalOutcome(toolID, response.Outcome)
}

func (b *toolApprovalBridge) requestSessionToolApproval(
	ctx context.Context,
	sessions sessionPermissionRequester,
	sessionID string,
	call toolspkg.CallRequest,
	descriptor toolspkg.Descriptor,
	view *toolspkg.ToolView,
) (acp.RequestPermissionResponse, error) {
	toolID := toolApprovalID(call, view)
	timeout := b.timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	approvalCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	response, err := sessions.RequestPermission(
		approvalCtx,
		sessionID,
		acp.RequestPermissionRequest{
			SessionId: acpsdk.SessionId(sessionID),
			ToolCall: acpsdk.RequestPermissionToolCall{
				ToolCallId: acpsdk.ToolCallId(toolApprovalCallID(call, view)),
				Title:      acpsdk.Ptr(toolApprovalTitle(descriptor)),
				Kind:       acpsdk.Ptr(toolApprovalKind(descriptor)),
				RawInput:   toolApprovalRawInput(call.Input),
				Status:     acpsdk.Ptr(acpsdk.ToolCallStatusPending),
			},
			Options: toolApprovalOptions(),
		},
	)
	if err != nil {
		switch {
		case errors.Is(approvalCtx.Err(), context.DeadlineExceeded):
			return acp.RequestPermissionResponse{}, toolApprovalError(
				toolID,
				"tool approval timed out",
				toolspkg.ReasonApprovalTimedOut,
			)
		case errors.Is(ctx.Err(), context.Canceled), errors.Is(err, context.Canceled):
			return acp.RequestPermissionResponse{}, toolApprovalError(
				toolID,
				"tool approval was canceled",
				toolspkg.ReasonApprovalCanceled,
			)
		default:
			return acp.RequestPermissionResponse{}, toolApprovalError(
				toolID,
				fmt.Sprintf("tool approval channel is unreachable: %v", err),
				toolspkg.ReasonApprovalUnreachable,
			)
		}
	}
	return response, nil
}

func (b *toolApprovalBridge) consumeLocalToolApproval(
	ctx context.Context,
	scope toolspkg.Scope,
	call toolspkg.CallRequest,
) (bool, error) {
	if b == nil || b.approvals == nil {
		return false, nil
	}
	if !scope.Operator && strings.TrimSpace(call.ApprovalToken) == "" {
		return false, nil
	}
	return true, b.approvals.ConsumeToolApproval(ctx, scope, call)
}

func toolApprovalID(call toolspkg.CallRequest, view *toolspkg.ToolView) toolspkg.ToolID {
	if view != nil && view.Descriptor.ID != "" {
		return view.Descriptor.ID
	}
	return call.ToolID
}

func toolApprovalDescriptor(call toolspkg.CallRequest, view *toolspkg.ToolView) toolspkg.Descriptor {
	if view != nil {
		return view.Descriptor
	}
	return toolspkg.Descriptor{ID: call.ToolID}
}

func toolApprovalOutcome(id toolspkg.ToolID, outcome acpsdk.RequestPermissionOutcome) error {
	if err := outcome.Validate(); err != nil {
		return toolApprovalError(id, "tool approval returned no outcome", toolspkg.ReasonApprovalUnreachable)
	}
	if outcome.Selected != nil {
		switch outcome.Selected.OptionId {
		case toolApprovalAllowOnceID, toolApprovalAllowAlwaysID:
			return nil
		case toolApprovalRejectOnceID, toolApprovalRejectAlwaysID:
			return toolApprovalError(id, "tool approval was rejected", toolspkg.ReasonApprovalRequired)
		default:
			return toolApprovalError(id, "tool approval selected an unknown option", toolspkg.ReasonApprovalUnreachable)
		}
	}
	return toolApprovalError(id, "tool approval was canceled", toolspkg.ReasonApprovalCanceled)
}

func toolApprovalOptions() []acpsdk.PermissionOption {
	return []acpsdk.PermissionOption{
		{Kind: acpsdk.PermissionOptionKindAllowOnce, Name: "Allow once", OptionId: toolApprovalAllowOnceID},
		{Kind: acpsdk.PermissionOptionKindAllowAlways, Name: "Allow always", OptionId: toolApprovalAllowAlwaysID},
		{Kind: acpsdk.PermissionOptionKindRejectOnce, Name: "Reject once", OptionId: toolApprovalRejectOnceID},
		{Kind: acpsdk.PermissionOptionKindRejectAlways, Name: "Reject always", OptionId: toolApprovalRejectAlwaysID},
	}
}

func toolApprovalCallID(call toolspkg.CallRequest, view *toolspkg.ToolView) string {
	toolID := toolApprovalID(call, view)
	for _, value := range []string{call.ToolCallID, call.CorrelationID, toolID.String()} {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return "hosted-tool-call"
}

func toolApprovalTitle(descriptor toolspkg.Descriptor) string {
	if title := strings.TrimSpace(descriptor.DisplayTitle); title != "" {
		return title
	}
	return descriptor.ID.String()
}

func toolApprovalKind(descriptor toolspkg.Descriptor) acpsdk.ToolKind {
	switch {
	case descriptor.ReadOnly && descriptor.Risk == toolspkg.RiskRead:
		return acpsdk.ToolKindRead
	case descriptor.Destructive || descriptor.Risk == toolspkg.RiskDestructive:
		return acpsdk.ToolKindDelete
	case descriptor.OpenWorld || descriptor.Risk == toolspkg.RiskOpenWorld:
		return acpsdk.ToolKindFetch
	default:
		return acpsdk.ToolKindExecute
	}
}

func toolApprovalRawInput(input json.RawMessage) any {
	if len(input) == 0 {
		return map[string]any{}
	}
	var value any
	if err := json.Unmarshal(input, &value); err != nil {
		return string(input)
	}
	return value
}

func toolApprovalError(id toolspkg.ToolID, message string, reason toolspkg.ReasonCode) error {
	return toolspkg.NewToolError(
		toolspkg.ErrorCodeApprovalRequired,
		id,
		message,
		toolspkg.ErrToolApprovalRequired,
		toolspkg.ReasonApprovalRequired,
		reason,
	)
}
