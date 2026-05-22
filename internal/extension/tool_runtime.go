package extensionpkg

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	extensionprotocol "github.com/compozy/agh/internal/extension/protocol"
	toolspkg "github.com/compozy/agh/internal/tools"
)

var _ toolspkg.ExtensionToolInvoker = (*Manager)(nil)

// ProvideTools calls the negotiated runtime descriptor endpoint for one
// tool-provider extension.
func (m *Manager) ProvideTools(
	ctx context.Context,
	extensionName string,
) ([]toolspkg.ExtensionToolRuntimeDescriptor, error) {
	process, name, err := m.extensionServiceProcess(
		ctx,
		extensionName,
		extensionprotocol.ExtensionServiceMethodProvideTools,
	)
	if err != nil {
		return nil, err
	}

	var response toolspkg.ExtensionProvideToolsResponse
	if err := process.Call(
		ctx,
		string(extensionprotocol.ExtensionServiceMethodProvideTools),
		struct{}{},
		&response,
	); err != nil {
		return nil, fmt.Errorf("extension: provide tools via %q: %w", name, err)
	}
	return cloneRuntimeToolDescriptors(response.Tools), nil
}

// CallTool invokes one reconciled extension-host tool through the existing
// subprocess JSON-RPC transport.
func (m *Manager) CallTool(
	ctx context.Context,
	extensionName string,
	req toolspkg.ExtensionToolCallRequest,
) (toolspkg.ToolResult, error) {
	if err := req.ToolID.Validate(); err != nil {
		return toolspkg.ToolResult{}, err
	}
	if strings.TrimSpace(req.Handler) == "" {
		return toolspkg.ToolResult{}, toolspkg.NewValidationError(
			"handler",
			toolspkg.ReasonHandlerMissing,
			"handler is required",
		)
	}

	process, name, err := m.extensionServiceProcess(
		ctx,
		extensionName,
		extensionprotocol.ExtensionServiceMethodToolsCall,
	)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}

	req.Handler = strings.TrimSpace(req.Handler)
	req.Input = cloneRawMessage(req.Input)
	var response toolspkg.ExtensionToolCallResponse
	if err := process.Call(ctx, string(extensionprotocol.ExtensionServiceMethodToolsCall), req, &response); err != nil {
		return toolspkg.ToolResult{}, fmt.Errorf("extension: call tool via %q: %w", name, err)
	}
	return response.Result, nil
}

func (m *Manager) extensionServiceProcess(
	ctx context.Context,
	extensionName string,
	method extensionprotocol.ExtensionServiceMethod,
) (processHandle, string, error) {
	if ctx == nil {
		return nil, "", ErrContextRequired
	}
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	if m == nil {
		return nil, "", ErrManagerRequired
	}
	name := strings.TrimSpace(extensionName)
	if name == "" {
		return nil, "", errors.New("extension: extension name is required")
	}
	methodName := string(method)
	if strings.TrimSpace(methodName) == "" {
		return nil, "", errors.New("extension: service method is required")
	}

	m.mu.RLock()
	ext := m.extensions[name]
	if ext == nil || ext.process == nil || !ext.active {
		m.mu.RUnlock()
		return nil, name, fmt.Errorf("extension: extension %q is not active: %w", name, toolspkg.ErrToolUnavailable)
	}
	grantedMethods := extensionprotocol.CapabilityServiceMethods(ext.info.Capabilities.Provides)
	if ext.manifest != nil {
		manifestMethods := extensionprotocol.CapabilityServiceMethods(ext.manifest.Capabilities.Provides)
		if len(manifestMethods) > 0 {
			grantedMethods = manifestMethods
		}
	}
	process := ext.process
	initialize := cloneInitializeResponse(ext.initialize)
	m.mu.RUnlock()

	if !slices.Contains(grantedMethods, methodName) {
		return nil, name, fmt.Errorf(
			"extension: extension %q is not granted service method %q: %w",
			name,
			methodName,
			toolspkg.ErrToolUnavailable,
		)
	}
	if initialize == nil || !slices.Contains(initialize.ImplementedMethods, methodName) {
		return nil, name, fmt.Errorf(
			"extension: extension %q does not implement %q: %w",
			name,
			methodName,
			toolspkg.ErrToolUnavailable,
		)
	}
	return process, name, nil
}

func cloneRuntimeToolDescriptors(
	src []toolspkg.ExtensionToolRuntimeDescriptor,
) []toolspkg.ExtensionToolRuntimeDescriptor {
	if len(src) == 0 {
		return nil
	}
	cloned := make([]toolspkg.ExtensionToolRuntimeDescriptor, len(src))
	for index := range src {
		cloned[index] = src[index]
		cloned[index].Capabilities = slices.Clone(src[index].Capabilities)
	}
	return cloned
}
