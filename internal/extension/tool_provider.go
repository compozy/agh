package extensionpkg

import (
	"context"
	"fmt"
	"slices"
	"strings"

	toolspkg "github.com/pedronauck/agh/internal/tools"
)

const extensionToolProviderOwner = "extensions"

// ExtensionToolRuntime is the live runtime surface needed by extension-host
// tool handles.
type ExtensionToolRuntime interface {
	Get(name string) (*Extension, error)
	toolspkg.ExtensionToolInvoker
}

// ExtensionToolRuntimeResolver returns the current live extension runtime.
type ExtensionToolRuntimeResolver func() ExtensionToolRuntime

// ExtensionToolProviderOption configures an extension-host tool provider.
type ExtensionToolProviderOption func(*ExtensionToolProvider)

// ExtensionToolProvider lists manifest-authored extension tools and resolves
// executable handles through the live subprocess runtime.
type ExtensionToolProvider struct {
	registry *Registry
	runtime  ExtensionToolRuntimeResolver
	source   toolspkg.SourceRef
}

var _ toolspkg.Provider = (*ExtensionToolProvider)(nil)

// NewExtensionToolProvider creates the extension_host provider for the central
// tool registry.
func NewExtensionToolProvider(
	registry *Registry,
	runtime ExtensionToolRuntimeResolver,
	opts ...ExtensionToolProviderOption,
) (*ExtensionToolProvider, error) {
	if registry == nil {
		return nil, toolspkg.NewValidationError(
			"registry",
			toolspkg.ReasonDependencyMissing,
			"extension registry is required",
		)
	}
	provider := &ExtensionToolProvider{
		registry: registry,
		runtime:  runtime,
		source: toolspkg.SourceRef{
			Kind:  toolspkg.SourceExtension,
			Owner: extensionToolProviderOwner,
		},
	}
	for _, opt := range opts {
		opt(provider)
	}
	if err := provider.source.Validate("source"); err != nil {
		return nil, err
	}
	return provider, nil
}

// ID returns the aggregate extension-provider provenance.
func (p *ExtensionToolProvider) ID() toolspkg.SourceRef {
	if p == nil {
		return toolspkg.SourceRef{}
	}
	return p.source
}

// List returns manifest-authoritative extension-host tool descriptors.
func (p *ExtensionToolProvider) List(ctx context.Context, _ toolspkg.Scope) ([]toolspkg.Descriptor, error) {
	if err := extensionProviderContextErr(ctx); err != nil {
		return nil, err
	}
	manifestTools, err := p.manifestTools()
	if err != nil {
		return nil, err
	}
	descriptors := make([]toolspkg.Descriptor, 0, len(manifestTools))
	for i := range manifestTools {
		descriptors = append(descriptors, manifestTools[i].descriptor.Tool.Descriptor())
	}
	return descriptors, nil
}

// Resolve returns a handle that reconciles one manifest descriptor against the
// live extension runtime before allowing execution.
func (p *ExtensionToolProvider) Resolve(
	ctx context.Context,
	scope toolspkg.Scope,
	id toolspkg.ToolID,
) (toolspkg.Handle, bool, error) {
	if err := extensionProviderContextErr(ctx); err != nil {
		return nil, false, err
	}
	if err := id.Validate(); err != nil {
		return nil, false, err
	}
	manifestTool, found, err := p.manifestTool(id)
	if err != nil || !found {
		return nil, found, err
	}
	return &extensionToolHandle{
		manifest: manifestTool,
		runtime:  p.runtime,
		scope:    scope,
	}, true, nil
}

type extensionManifestTool struct {
	info       ExtensionInfo
	descriptor ManifestToolDescriptor
}

type extensionToolHandle struct {
	manifest extensionManifestTool
	runtime  ExtensionToolRuntimeResolver
	scope    toolspkg.Scope
}

var _ toolspkg.Handle = (*extensionToolHandle)(nil)

func (h *extensionToolHandle) Descriptor() toolspkg.Descriptor {
	if h == nil {
		return toolspkg.Descriptor{}
	}
	return h.manifest.descriptor.Tool.Descriptor()
}

func (h *extensionToolHandle) Availability(ctx context.Context, _ toolspkg.Scope) toolspkg.Availability {
	if h == nil {
		return toolspkg.Unavailable(toolspkg.ReasonBackendNotExecutable)
	}
	state, runtime := h.runtimeState()
	if runtime == nil {
		return ReconcileManifestToolRuntime(&h.manifest.descriptor, nil, state)
	}
	descriptors, err := runtime.ProvideTools(ctx, h.extensionID())
	if err != nil {
		availability := ReconcileManifestToolRuntime(&h.manifest.descriptor, nil, state)
		availability.ReasonCodes = appendToolReason(availability.ReasonCodes, toolspkg.ReasonBackendUnhealthy)
		return availability
	}
	runtimeDescriptor, duplicate := runtimeDescriptorForTool(descriptors, h.manifest.descriptor.Tool.ID)
	availability := ReconcileManifestToolRuntime(&h.manifest.descriptor, runtimeDescriptor, state)
	if duplicate {
		availability.ReasonCodes = appendToolReason(
			availability.ReasonCodes,
			toolspkg.ReasonRuntimeDescriptorMismatch,
		)
		availability.Available = false
		availability.Executable = false
	}
	return availability
}

func (h *extensionToolHandle) Call(ctx context.Context, req toolspkg.CallRequest) (toolspkg.ToolResult, error) {
	if h == nil {
		return toolspkg.ToolResult{}, toolspkg.NewToolError(
			toolspkg.ErrorCodeUnavailable,
			req.ToolID,
			"extension tool handle is unavailable",
			toolspkg.ErrToolUnavailable,
			toolspkg.ReasonBackendNotExecutable,
		)
	}
	availability := h.Availability(ctx, h.scope)
	if !availability.Executable {
		return toolspkg.ToolResult{}, toolspkg.NewToolError(
			toolspkg.ErrorCodeUnavailable,
			h.manifest.descriptor.Tool.ID,
			fmt.Sprintf("tool %q is unavailable", h.manifest.descriptor.Tool.ID),
			toolspkg.ErrToolUnavailable,
			availability.ReasonCodes...,
		)
	}
	runtime := h.runtimeInstance()
	if runtime == nil {
		return toolspkg.ToolResult{}, toolspkg.NewToolError(
			toolspkg.ErrorCodeUnavailable,
			h.manifest.descriptor.Tool.ID,
			fmt.Sprintf("tool %q runtime is unavailable", h.manifest.descriptor.Tool.ID),
			toolspkg.ErrToolUnavailable,
			toolspkg.ReasonExtensionInactive,
		)
	}
	return runtime.CallTool(ctx, h.extensionID(), toolspkg.ExtensionToolCallRequest{
		ToolID:    h.manifest.descriptor.Tool.ID,
		Handler:   h.manifest.descriptor.Tool.Backend.Handler,
		SessionID: req.SessionID,
		Input:     cloneRawMessage(req.Input),
	})
}

func (h *extensionToolHandle) runtimeState() (ExtensionToolRuntimeState, ExtensionToolRuntime) {
	state := ExtensionToolRuntimeState{
		Enabled: h.manifest.info.Enabled,
	}
	runtime := h.runtimeInstance()
	if runtime == nil {
		return state, nil
	}
	snapshot, err := runtime.Get(h.extensionID())
	if err != nil || snapshot == nil {
		return state, runtime
	}
	state.Enabled = snapshot.Info.Enabled
	state.Active = snapshot.Status.Active
	state.Healthy = snapshot.Status.Healthy
	if snapshot.InitializeResult != nil {
		state.ProvidedCapabilities = slices.Clone(snapshot.InitializeResult.AcceptedCapabilities.Provides)
	}
	return state, runtime
}

func (h *extensionToolHandle) runtimeInstance() ExtensionToolRuntime {
	if h == nil || h.runtime == nil {
		return nil
	}
	return h.runtime()
}

func (h *extensionToolHandle) extensionID() string {
	return strings.TrimSpace(h.manifest.descriptor.Tool.Backend.ExtensionID)
}

func (p *ExtensionToolProvider) manifestTool(
	id toolspkg.ToolID,
) (extensionManifestTool, bool, error) {
	manifestTools, err := p.manifestTools()
	if err != nil {
		return extensionManifestTool{}, false, err
	}
	for i := range manifestTools {
		if manifestTools[i].descriptor.Tool.ID == id {
			return manifestTools[i], true, nil
		}
	}
	return extensionManifestTool{}, false, nil
}

func (p *ExtensionToolProvider) manifestTools() ([]extensionManifestTool, error) {
	if p == nil || p.registry == nil {
		return nil, toolspkg.NewValidationError(
			"provider",
			toolspkg.ReasonDependencyMissing,
			"extension tool provider is required",
		)
	}
	infos, err := p.registry.List()
	if err != nil {
		return nil, fmt.Errorf("extension: list tool manifests: %w", err)
	}
	manifestTools := make([]extensionManifestTool, 0)
	for _, info := range infos {
		manifest, err := loadManifestAtPath(info.ManifestPath)
		if err != nil {
			return nil, fmt.Errorf("extension: load tool manifest %q: %w", info.Name, err)
		}
		descriptors, err := ResolveManifestToolDescriptors(manifest)
		if err != nil {
			return nil, fmt.Errorf("extension: resolve tool manifest %q: %w", info.Name, err)
		}
		for i := range descriptors {
			if descriptors[i].Tool.Backend.Kind != toolspkg.BackendExtensionHost {
				continue
			}
			manifestTools = append(manifestTools, extensionManifestTool{
				info:       cloneExtensionInfo(info),
				descriptor: cloneManifestToolDescriptor(&descriptors[i]),
			})
		}
	}
	slices.SortFunc(manifestTools, func(left, right extensionManifestTool) int {
		return strings.Compare(left.descriptor.Tool.ID.String(), right.descriptor.Tool.ID.String())
	})
	return manifestTools, nil
}

func runtimeDescriptorForTool(
	descriptors []toolspkg.ExtensionToolRuntimeDescriptor,
	id toolspkg.ToolID,
) (*toolspkg.ExtensionToolRuntimeDescriptor, bool) {
	var found *toolspkg.ExtensionToolRuntimeDescriptor
	for i := range descriptors {
		if descriptors[i].ID != id {
			continue
		}
		if found != nil {
			return found, true
		}
		descriptor := descriptors[i]
		descriptor.Capabilities = slices.Clone(descriptors[i].Capabilities)
		found = &descriptor
	}
	return found, false
}

func cloneManifestToolDescriptor(src *ManifestToolDescriptor) ManifestToolDescriptor {
	cloned := *src
	cloned.Tool = src.Tool.Descriptor().Tool()
	cloned.RuntimeDescriptor.Capabilities = slices.Clone(src.RuntimeDescriptor.Capabilities)
	return cloned
}

func extensionProviderContextErr(ctx context.Context) error {
	if ctx == nil {
		return ErrContextRequired
	}
	return ctx.Err()
}
