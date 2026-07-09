package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	toolspkg "github.com/compozy/agh/internal/tools"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

const devCycleImportTasksToolID toolspkg.ToolID = "ext__dev_cycle__import_tasks"

type daemonExtensionToolProvider struct {
	inner             toolspkg.Provider
	workspaceResolver workspacepkg.RuntimeResolver
}

var _ toolspkg.Provider = (*daemonExtensionToolProvider)(nil)

func newDaemonScopedExtensionToolProvider(
	inner toolspkg.Provider,
	workspaceResolver workspacepkg.RuntimeResolver,
) toolspkg.Provider {
	if inner == nil {
		return nil
	}
	return &daemonExtensionToolProvider{
		inner:             inner,
		workspaceResolver: workspaceResolver,
	}
}

func (p *daemonExtensionToolProvider) ID() toolspkg.SourceRef {
	return p.inner.ID()
}

func (p *daemonExtensionToolProvider) List(
	ctx context.Context,
	scope toolspkg.Scope,
) ([]toolspkg.Descriptor, error) {
	return p.inner.List(ctx, scope)
}

func (p *daemonExtensionToolProvider) Resolve(
	ctx context.Context,
	scope toolspkg.Scope,
	id toolspkg.ToolID,
) (toolspkg.Handle, bool, error) {
	handle, ok, err := p.inner.Resolve(ctx, scope, id)
	if err != nil || !ok || handle == nil {
		return handle, ok, err
	}
	return &daemonExtensionToolHandle{
		inner:             handle,
		workspaceResolver: p.workspaceResolver,
	}, true, nil
}

type daemonExtensionToolHandle struct {
	inner             toolspkg.Handle
	workspaceResolver workspacepkg.RuntimeResolver
}

var _ toolspkg.Handle = (*daemonExtensionToolHandle)(nil)

func (h *daemonExtensionToolHandle) Descriptor() toolspkg.Descriptor {
	return h.inner.Descriptor()
}

func (h *daemonExtensionToolHandle) Availability(
	ctx context.Context,
	scope toolspkg.Scope,
) toolspkg.Availability {
	return h.inner.Availability(ctx, scope)
}

func (h *daemonExtensionToolHandle) Call(
	ctx context.Context,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	patched, err := h.workspaceScopedCallRequest(ctx, req)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	return h.inner.Call(ctx, patched)
}

func (h *daemonExtensionToolHandle) workspaceScopedCallRequest(
	ctx context.Context,
	req toolspkg.CallRequest,
) (toolspkg.CallRequest, error) {
	if req.ToolID != devCycleImportTasksToolID || h.workspaceResolver == nil {
		return req, nil
	}
	workspaceID := strings.TrimSpace(req.WorkspaceID)
	if workspaceID == "" {
		return req, nil
	}
	var input struct {
		Pattern string `json:"pattern"`
	}
	if err := json.Unmarshal(req.Input, &input); err != nil {
		return toolspkg.CallRequest{}, toolspkg.NewToolError(
			toolspkg.ErrorCodeInvalidInput,
			req.ToolID,
			fmt.Sprintf("tool %q input is invalid", req.ToolID),
			fmt.Errorf("%w: decode input: %w", toolspkg.ErrToolInvalidInput, err),
			toolspkg.ReasonSchemaInvalid,
		)
	}
	pattern := strings.TrimSpace(input.Pattern)
	if pattern == "" || filepath.IsAbs(pattern) {
		return req, nil
	}
	resolved, err := h.workspaceResolver.Resolve(ctx, workspaceID)
	if err != nil {
		return toolspkg.CallRequest{}, toolspkg.NewToolError(
			toolspkg.ErrorCodeInvalidInput,
			req.ToolID,
			fmt.Sprintf("tool %q workspace %q is invalid", req.ToolID, workspaceID),
			fmt.Errorf("%w: resolve workspace: %w", toolspkg.ErrToolInvalidInput, err),
			toolspkg.ReasonScopeMismatch,
		)
	}
	root := strings.TrimSpace(resolved.RootDir)
	if root == "" {
		return toolspkg.CallRequest{}, toolspkg.NewToolError(
			toolspkg.ErrorCodeInvalidInput,
			req.ToolID,
			fmt.Sprintf("tool %q workspace %q has no root directory", req.ToolID, workspaceID),
			toolspkg.ErrToolInvalidInput,
			toolspkg.ReasonScopeMismatch,
		)
	}
	absolute, err := workspaceRelativePattern(root, pattern)
	if err != nil {
		return toolspkg.CallRequest{}, toolspkg.NewToolError(
			toolspkg.ErrorCodeInvalidInput,
			req.ToolID,
			fmt.Sprintf("tool %q pattern escapes workspace root", req.ToolID),
			fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
			toolspkg.ReasonScopeMismatch,
		)
	}
	input.Pattern = absolute
	encoded, err := json.Marshal(input)
	if err != nil {
		return toolspkg.CallRequest{}, fmt.Errorf("daemon: encode workspace-scoped extension tool input: %w", err)
	}
	req.Input = encoded
	return req, nil
}

func workspaceRelativePattern(root string, pattern string) (string, error) {
	rootAbs, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return "", fmt.Errorf("workspace root: %w", err)
	}
	absolute := filepath.Join(rootAbs, strings.TrimSpace(pattern))
	patternAbs, err := filepath.Abs(absolute)
	if err != nil {
		return "", fmt.Errorf("pattern: %w", err)
	}
	rel, err := filepath.Rel(rootAbs, patternAbs)
	if err != nil {
		return "", fmt.Errorf("relative pattern: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("relative pattern %q escapes %q", pattern, rootAbs)
	}
	return patternAbs, nil
}
