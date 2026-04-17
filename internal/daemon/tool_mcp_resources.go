package daemon

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"

	aghconfig "github.com/pedronauck/agh/internal/config"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	"github.com/pedronauck/agh/internal/resources"
	toolspkg "github.com/pedronauck/agh/internal/tools"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const (
	toolManagedIDPrefix      = "daemon.sync.tool."
	mcpServerManagedIDPrefix = "daemon.sync.mcp_server."
)

type toolMCPPublisher interface {
	Sync(context.Context) error
}

type toolMCPPublisherFunc func(context.Context) error

func (f toolMCPPublisherFunc) Sync(ctx context.Context) error {
	if f == nil {
		return nil
	}
	return f(ctx)
}

type resourceCatalog[T any] struct {
	mu        sync.RWMutex
	revision  int64
	records   []resources.Record[T]
	cloneSpec func(T) T
}

func newResourceCatalog[T any](cloneSpec func(T) T) *resourceCatalog[T] {
	return &resourceCatalog[T]{cloneSpec: cloneSpec}
}

func (c *resourceCatalog[T]) Replace(revision int64, records []resources.Record[T]) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.revision = revision
	c.records = cloneResourceRecords(records, c.cloneSpec)
}

func (c *resourceCatalog[T]) Snapshot() []resources.Record[T] {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return cloneResourceRecords(c.records, c.cloneSpec)
}

func (c *resourceCatalog[T]) Revision() int64 {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.revision
}

type resourceCatalogProjectionPlan[T any] struct {
	kind       resources.ResourceKind
	revision   int64
	operations int
	records    []resources.Record[T]
}

func (p *resourceCatalogProjectionPlan[T]) Kind() resources.ResourceKind {
	if p == nil {
		return ""
	}
	return p.kind
}

func (p *resourceCatalogProjectionPlan[T]) Revision() int64 {
	if p == nil {
		return 0
	}
	return p.revision
}

func (p *resourceCatalogProjectionPlan[T]) OperationCount() int {
	if p == nil {
		return 0
	}
	return p.operations
}

type resourceCatalogProjector[T any] struct {
	kind      resources.ResourceKind
	catalog   *resourceCatalog[T]
	cloneSpec func(T) T
}

func newToolProjector(catalog *resourceCatalog[toolspkg.Tool]) resources.TypedProjector[toolspkg.Tool] {
	if catalog == nil {
		return nil
	}
	return &resourceCatalogProjector[toolspkg.Tool]{
		kind:      toolspkg.ToolResourceKind,
		catalog:   catalog,
		cloneSpec: cloneToolSpec,
	}
}

func newMCPServerProjector(
	catalog *resourceCatalog[aghconfig.MCPServer],
) resources.TypedProjector[aghconfig.MCPServer] {
	if catalog == nil {
		return nil
	}
	return &resourceCatalogProjector[aghconfig.MCPServer]{
		kind:      aghconfig.MCPServerResourceKind,
		catalog:   catalog,
		cloneSpec: cloneDaemonMCPServer,
	}
}

func (p *resourceCatalogProjector[T]) Kind() resources.ResourceKind {
	if p == nil {
		return ""
	}
	return p.kind
}

func (p *resourceCatalogProjector[T]) DependsOn() []resources.ResourceKind {
	return nil
}

func (p *resourceCatalogProjector[T]) Build(
	_ context.Context,
	records []resources.Record[T],
) (resources.ProjectionPlan, error) {
	if p == nil || p.catalog == nil {
		return nil, errors.New("daemon: resource catalog projector is required")
	}

	var revision int64
	for _, record := range records {
		if record.Version > revision {
			revision = record.Version
		}
	}

	return &resourceCatalogProjectionPlan[T]{
		kind:       p.kind,
		revision:   revision,
		operations: len(records),
		records:    cloneResourceRecords(records, p.cloneSpec),
	}, nil
}

func (p *resourceCatalogProjector[T]) Apply(ctx context.Context, plan resources.ProjectionPlan) error {
	if p == nil || p.catalog == nil {
		return errors.New("daemon: resource catalog projector is required")
	}
	if ctx == nil {
		return errors.New("daemon: resource catalog projector apply context is required")
	}

	typed, ok := plan.(*resourceCatalogProjectionPlan[T])
	if !ok {
		return fmt.Errorf("daemon: resource catalog projector plan has type %T", plan)
	}
	p.catalog.Replace(typed.revision, typed.records)
	return nil
}

type toolPublicationInput struct {
	sourceKey string
	scope     resources.ResourceScope
	spec      toolspkg.Tool
}

type mcpServerPublicationInput struct {
	sourceKey string
	scope     resources.ResourceScope
	spec      aghconfig.MCPServer
}

type toolMCPDesiredResources struct {
	tools      []toolPublicationInput
	mcpServers []mcpServerPublicationInput
}

type toolMCPDeclarationProvider func(context.Context) (toolMCPDesiredResources, error)

type toolMCPSourceSyncer struct {
	toolStore resources.Store[toolspkg.Tool]
	toolCodec resources.KindCodec[toolspkg.Tool]
	mcpStore  resources.Store[aghconfig.MCPServer]
	mcpCodec  resources.KindCodec[aghconfig.MCPServer]
	actor     resources.MutationActor
	logger    *slog.Logger
	trigger   func(context.Context, resources.ResourceKind, resources.ReconcileReason) error
	providers []toolMCPDeclarationProvider
}

func newToolMCPSourceSyncer(
	toolStore resources.Store[toolspkg.Tool],
	toolCodec resources.KindCodec[toolspkg.Tool],
	mcpStore resources.Store[aghconfig.MCPServer],
	mcpCodec resources.KindCodec[aghconfig.MCPServer],
	actor resources.MutationActor,
	logger *slog.Logger,
	trigger func(context.Context, resources.ResourceKind, resources.ReconcileReason) error,
	providers ...toolMCPDeclarationProvider,
) toolMCPPublisher {
	if toolStore == nil || toolCodec == nil || mcpStore == nil || mcpCodec == nil {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &toolMCPSourceSyncer{
		toolStore: toolStore,
		toolCodec: toolCodec,
		mcpStore:  mcpStore,
		mcpCodec:  mcpCodec,
		actor:     actor,
		logger:    logger,
		trigger:   trigger,
		providers: append([]toolMCPDeclarationProvider(nil), providers...),
	}
}

func toolMCPSyncActor() resources.MutationActor {
	return resources.MutationActor{
		Kind: resources.MutationActorKindDaemon,
		ID:   "tool-mcp-sync",
		Source: resources.ResourceSource{
			Kind: resources.ResourceSourceKind("daemon"),
			ID:   "tool-mcp-sync",
		},
		MaxScope: resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
	}
}

func (s *toolMCPSourceSyncer) Sync(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("daemon: tool/mcp sync context is required")
	}

	desired, err := s.desiredResources(ctx)
	if err != nil {
		return err
	}

	toolChanged, err := s.syncTools(ctx, desired.tools)
	if err != nil {
		return err
	}
	mcpChanged, err := s.syncMCPServers(ctx, desired.mcpServers)
	if err != nil {
		return err
	}

	if toolChanged && s.trigger != nil {
		if err := s.trigger(ctx, toolspkg.ToolResourceKind, resources.ReconcileReasonWrite); err != nil {
			return err
		}
	}
	if mcpChanged && s.trigger != nil {
		if err := s.trigger(ctx, aghconfig.MCPServerResourceKind, resources.ReconcileReasonWrite); err != nil {
			return err
		}
	}

	return nil
}

type desiredToolResource struct {
	id      string
	scope   resources.ResourceScope
	spec    toolspkg.Tool
	encoded []byte
}

type desiredMCPServerResource struct {
	id      string
	scope   resources.ResourceScope
	spec    aghconfig.MCPServer
	encoded []byte
}

func (s *toolMCPSourceSyncer) desiredResources(ctx context.Context) (struct {
	tools      map[string]desiredToolResource
	mcpServers map[string]desiredMCPServerResource
}, error) {
	desired := struct {
		tools      map[string]desiredToolResource
		mcpServers map[string]desiredMCPServerResource
	}{
		tools:      make(map[string]desiredToolResource),
		mcpServers: make(map[string]desiredMCPServerResource),
	}

	for _, provider := range s.providers {
		if provider == nil {
			continue
		}
		items, err := provider(ctx)
		if err != nil {
			return desired, err
		}

		for _, item := range items.tools {
			spec, encoded, err := validateAndEncodeTool(ctx, s.toolCodec, item.scope, item.spec)
			if err != nil {
				return desired, err
			}
			id := managedResourceID(toolManagedIDPrefix, item.scope.Normalize(), item.sourceKey, encoded)
			desired.tools[id] = desiredToolResource{
				id:      id,
				scope:   item.scope.Normalize(),
				spec:    spec,
				encoded: encoded,
			}
		}
		for _, item := range items.mcpServers {
			spec, encoded, err := validateAndEncodeMCPServer(ctx, s.mcpCodec, item.scope, item.spec)
			if err != nil {
				return desired, err
			}
			id := managedResourceID(mcpServerManagedIDPrefix, item.scope.Normalize(), item.sourceKey, encoded)
			desired.mcpServers[id] = desiredMCPServerResource{
				id:      id,
				scope:   item.scope.Normalize(),
				spec:    spec,
				encoded: encoded,
			}
		}
	}

	return desired, nil
}

func (s *toolMCPSourceSyncer) syncTools(ctx context.Context, desired map[string]desiredToolResource) (bool, error) {
	source := s.actor.Source
	current, err := s.toolStore.List(ctx, s.actor, resources.ResourceFilter{Source: &source})
	if err != nil {
		return false, fmt.Errorf("daemon: list managed tools: %w", err)
	}

	currentByID := make(map[string]resources.Record[toolspkg.Tool], len(current))
	for _, record := range current {
		currentByID[record.ID] = record
	}

	changed := false
	for id, desiredTool := range desired {
		existing, ok := currentByID[id]
		if ok && s.sameTool(existing, desiredTool.scope, desiredTool.encoded) {
			delete(currentByID, id)
			continue
		}

		expectedVersion := int64(0)
		if ok {
			expectedVersion = existing.Version
		}
		if _, err := s.toolStore.Put(ctx, s.actor, resources.Draft[toolspkg.Tool]{
			ID:              desiredTool.id,
			Scope:           desiredTool.scope,
			ExpectedVersion: expectedVersion,
			Spec:            desiredTool.spec,
		}); err != nil {
			return false, fmt.Errorf("daemon: sync tool %q: %w", id, err)
		}
		changed = true
		delete(currentByID, id)
	}

	for _, stale := range currentByID {
		if err := s.toolStore.Delete(ctx, s.actor, stale.ID, stale.Version); err != nil {
			return false, fmt.Errorf("daemon: delete stale tool %q: %w", stale.ID, err)
		}
		changed = true
	}

	return changed, nil
}

func (s *toolMCPSourceSyncer) syncMCPServers(
	ctx context.Context,
	desired map[string]desiredMCPServerResource,
) (bool, error) {
	source := s.actor.Source
	current, err := s.mcpStore.List(ctx, s.actor, resources.ResourceFilter{Source: &source})
	if err != nil {
		return false, fmt.Errorf("daemon: list managed mcp servers: %w", err)
	}

	currentByID := make(map[string]resources.Record[aghconfig.MCPServer], len(current))
	for _, record := range current {
		currentByID[record.ID] = record
	}

	changed := false
	for id, desiredServer := range desired {
		existing, ok := currentByID[id]
		if ok && s.sameMCPServer(existing, desiredServer.scope, desiredServer.encoded) {
			delete(currentByID, id)
			continue
		}

		expectedVersion := int64(0)
		if ok {
			expectedVersion = existing.Version
		}
		if _, err := s.mcpStore.Put(ctx, s.actor, resources.Draft[aghconfig.MCPServer]{
			ID:              desiredServer.id,
			Scope:           desiredServer.scope,
			ExpectedVersion: expectedVersion,
			Spec:            desiredServer.spec,
		}); err != nil {
			return false, fmt.Errorf("daemon: sync mcp server %q: %w", id, err)
		}
		changed = true
		delete(currentByID, id)
	}

	for _, stale := range currentByID {
		if err := s.mcpStore.Delete(ctx, s.actor, stale.ID, stale.Version); err != nil {
			return false, fmt.Errorf("daemon: delete stale mcp server %q: %w", stale.ID, err)
		}
		changed = true
	}

	return changed, nil
}

func (s *toolMCPSourceSyncer) sameTool(
	record resources.Record[toolspkg.Tool],
	scope resources.ResourceScope,
	encoded []byte,
) bool {
	if record.Scope != scope {
		return false
	}

	currentEncoded, err := s.toolCodec.Encode(record.Spec)
	if err != nil {
		return false
	}
	return bytes.Equal(currentEncoded, encoded)
}

func (s *toolMCPSourceSyncer) sameMCPServer(
	record resources.Record[aghconfig.MCPServer],
	scope resources.ResourceScope,
	encoded []byte,
) bool {
	if record.Scope != scope {
		return false
	}

	currentEncoded, err := s.mcpCodec.Encode(record.Spec)
	if err != nil {
		return false
	}
	return bytes.Equal(currentEncoded, encoded)
}

func (d *Daemon) newToolMCPPublisher(
	state *bootState,
	registry *extensionpkg.Registry,
) (toolMCPPublisher, error) {
	publisher := toolMCPPublisher(toolMCPPublisherFunc(func(context.Context) error { return nil }))
	if state == nil {
		return publisher, nil
	}
	if state.resourceKernel == nil || state.resourceCodecs == nil {
		return publisher, nil
	}

	toolCodec, err := resources.ResolveCodec[toolspkg.Tool](state.resourceCodecs, toolspkg.ToolResourceKind)
	if err != nil {
		return nil, fmt.Errorf("daemon: resolve tool codec: %w", err)
	}
	toolStore, err := resources.NewStore(state.resourceKernel, toolCodec)
	if err != nil {
		return nil, fmt.Errorf("daemon: create tool store: %w", err)
	}

	mcpCodec, err := resources.ResolveCodec[aghconfig.MCPServer](state.resourceCodecs, aghconfig.MCPServerResourceKind)
	if err != nil {
		return nil, fmt.Errorf("daemon: resolve mcp server codec: %w", err)
	}
	mcpStore, err := resources.NewStore(state.resourceKernel, mcpCodec)
	if err != nil {
		return nil, fmt.Errorf("daemon: create mcp server store: %w", err)
	}

	return newToolMCPSourceSyncer(
		toolStore,
		toolCodec,
		mcpStore,
		mcpCodec,
		toolMCPSyncActor(),
		state.logger,
		func(ctx context.Context, kind resources.ResourceKind, reason resources.ReconcileReason) error {
			if state.resourceReconcile == nil {
				return nil
			}
			return state.resourceReconcile.Trigger(ctx, kind, reason)
		},
		daemonConfigMCPDeclarationProvider(&state.cfg, state.registry, state.workspaceResolver, state.logger),
		extensionManifestToolMCPDeclarationProvider(registry, state.currentExtensionRuntime, d.getenv, state.logger),
	), nil
}

func daemonConfigMCPDeclarationProvider(
	cfg *aghconfig.Config,
	registry Registry,
	workspaceResolver workspacepkg.RuntimeResolver,
	logger *slog.Logger,
) toolMCPDeclarationProvider {
	return func(ctx context.Context) (toolMCPDesiredResources, error) {
		desired := toolMCPDesiredResources{}
		if cfg == nil {
			return desired, nil
		}
		globalScope := resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal}
		for _, server := range cfg.MCPServers {
			desired.mcpServers = append(desired.mcpServers, mcpServerPublicationInput{
				sourceKey: "config/global/" + strings.TrimSpace(server.Name),
				scope:     globalScope,
				spec:      cloneDaemonMCPServer(server),
			})
		}

		workspaces, err := registeredWorkspaces(ctx, registry, workspaceResolver, logger)
		if err != nil {
			return toolMCPDesiredResources{}, err
		}
		for idx := range workspaces {
			resolved := &workspaces[idx]
			scope := resources.ResourceScope{
				Kind: resources.ResourceScopeKindWorkspace,
				ID:   strings.TrimSpace(resolved.ID),
			}
			for _, server := range resolved.Config.MCPServers {
				desired.mcpServers = append(desired.mcpServers, mcpServerPublicationInput{
					sourceKey: "config/workspace/" + scope.ID + "/" + strings.TrimSpace(server.Name),
					scope:     scope,
					spec:      cloneDaemonMCPServer(server),
				})
			}
		}

		return desired, nil
	}
}

func extensionManifestToolMCPDeclarationProvider(
	registry *extensionpkg.Registry,
	runtime func() extensionRuntime,
	getenv func(string) string,
	logger *slog.Logger,
) toolMCPDeclarationProvider {
	return func(_ context.Context) (toolMCPDesiredResources, error) {
		if registry == nil || runtime == nil {
			return toolMCPDesiredResources{}, nil
		}

		manager := runtime()
		if manager == nil {
			return toolMCPDesiredResources{}, nil
		}

		infos, err := registry.List()
		if err != nil {
			return toolMCPDesiredResources{}, fmt.Errorf("daemon: list extensions for tool/mcp sync: %w", err)
		}
		slices.SortFunc(infos, func(left, right extensionpkg.ExtensionInfo) int {
			return strings.Compare(left.Name, right.Name)
		})

		desired := toolMCPDesiredResources{}
		globalScope := resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal}
		for _, info := range infos {
			if !info.Enabled {
				continue
			}

			ext, err := loadExtensionSnapshot(registry, manager, logger, info.Name)
			if err != nil {
				return toolMCPDesiredResources{}, fmt.Errorf(
					"daemon: load extension %q for tool/mcp sync: %w",
					info.Name,
					err,
				)
			}
			if ext == nil || ext.Manifest == nil || !ext.Status.Registered {
				continue
			}

			for _, tool := range extensionpkg.ResolveManifestToolResources(ext.Manifest) {
				desired.tools = append(desired.tools, toolPublicationInput{
					sourceKey: "extension/" + ext.Info.Name + "/tool/" + strings.TrimSpace(tool.Name),
					scope:     globalScope,
					spec:      cloneToolSpec(tool),
				})
			}

			servers, err := extensionpkg.ResolveManifestMCPServerResources(ext.RootDir, ext.Manifest, getenv)
			if err != nil {
				return toolMCPDesiredResources{}, fmt.Errorf(
					"daemon: resolve extension %q mcp servers: %w",
					ext.Info.Name,
					err,
				)
			}
			for _, server := range servers {
				desired.mcpServers = append(desired.mcpServers, mcpServerPublicationInput{
					sourceKey: "extension/" + ext.Info.Name + "/mcp_server/" + strings.TrimSpace(server.Name),
					scope:     globalScope,
					spec:      cloneDaemonMCPServer(server),
				})
			}
		}

		return desired, nil
	}
}

func validateAndEncodeTool(
	ctx context.Context,
	codec resources.KindCodec[toolspkg.Tool],
	scope resources.ResourceScope,
	spec toolspkg.Tool,
) (toolspkg.Tool, []byte, error) {
	encoded, err := codec.Encode(spec)
	if err != nil {
		return toolspkg.Tool{}, nil, err
	}
	validated, err := codec.DecodeAndValidate(ctx, scope.Normalize(), encoded)
	if err != nil {
		return toolspkg.Tool{}, nil, err
	}
	canonical, err := codec.Encode(validated)
	if err != nil {
		return toolspkg.Tool{}, nil, err
	}
	return validated, canonical, nil
}

func validateAndEncodeMCPServer(
	ctx context.Context,
	codec resources.KindCodec[aghconfig.MCPServer],
	scope resources.ResourceScope,
	spec aghconfig.MCPServer,
) (aghconfig.MCPServer, []byte, error) {
	encoded, err := codec.Encode(spec)
	if err != nil {
		return aghconfig.MCPServer{}, nil, err
	}
	validated, err := codec.DecodeAndValidate(ctx, scope.Normalize(), encoded)
	if err != nil {
		return aghconfig.MCPServer{}, nil, err
	}
	canonical, err := codec.Encode(validated)
	if err != nil {
		return aghconfig.MCPServer{}, nil, err
	}
	return validated, canonical, nil
}

func managedResourceID(
	prefix string,
	scope resources.ResourceScope,
	sourceKey string,
	encoded []byte,
) string {
	sum := sha256.Sum256([]byte(
		string(scope.Kind) + "\x00" + scope.ID + "\x00" + strings.TrimSpace(sourceKey) + "\x00" + string(encoded),
	))
	return prefix + hex.EncodeToString(sum[:12])
}

func cloneResourceRecords[T any](records []resources.Record[T], cloneSpec func(T) T) []resources.Record[T] {
	if len(records) == 0 {
		return nil
	}
	cloned := make([]resources.Record[T], len(records))
	for idx := range records {
		cloned[idx] = records[idx]
		cloned[idx].Spec = cloneSpec(records[idx].Spec)
	}
	return cloned
}

func cloneToolSpec(src toolspkg.Tool) toolspkg.Tool {
	cloned := src
	if len(src.InputSchema) > 0 {
		cloned.InputSchema = append([]byte(nil), src.InputSchema...)
	}
	return cloned
}

func cloneDaemonMCPServer(src aghconfig.MCPServer) aghconfig.MCPServer {
	return aghconfig.MCPServer{
		Name:    src.Name,
		Command: src.Command,
		Args:    slices.Clone(src.Args),
		Env:     cloneStringMap(src.Env),
	}
}
