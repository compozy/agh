package daemon

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	aghconfig "github.com/pedronauck/agh/internal/config"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/resources"
	skillspkg "github.com/pedronauck/agh/internal/skills"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const (
	agentManagedIDPrefix = "daemon.sync.agent."
	skillManagedIDPrefix = "daemon.sync.skill."
)

type agentSkillPublisher interface {
	Sync(context.Context) error
}

type agentSkillPublisherFunc func(context.Context) error

func (f agentSkillPublisherFunc) Sync(ctx context.Context) error {
	if f == nil {
		return nil
	}
	return f(ctx)
}

type agentSkillDeclarationProvider func(context.Context) (agentSkillDesiredResources, error)

type agentPublicationInput struct {
	sourceKey string
	scope     resources.ResourceScope
	spec      aghconfig.AgentDef
}

type skillPublicationInput struct {
	sourceKey string
	scope     resources.ResourceScope
	spec      skillspkg.SkillResourceSpec
}

type agentSkillDesiredResources struct {
	agents     []agentPublicationInput
	skills     []skillPublicationInput
	mcpServers []mcpServerPublicationInput
}

type agentSkillSourceSyncer struct {
	agentStore resources.Store[aghconfig.AgentDef]
	agentCodec resources.KindCodec[aghconfig.AgentDef]
	skillStore resources.Store[skillspkg.SkillResourceSpec]
	skillCodec resources.KindCodec[skillspkg.SkillResourceSpec]
	mcpStore   resources.Store[aghconfig.MCPServer]
	mcpCodec   resources.KindCodec[aghconfig.MCPServer]
	actor      resources.MutationActor
	logger     *slog.Logger
	trigger    func(context.Context, resources.ResourceKind, resources.ReconcileReason) error
	providers  []agentSkillDeclarationProvider
}

type skillResourceProjectionPlan struct {
	revision int64
	records  []resources.Record[skillspkg.SkillResourceSpec]
}

func (p *skillResourceProjectionPlan) Kind() resources.ResourceKind {
	if p == nil {
		return ""
	}
	return skillspkg.SkillResourceKind
}

func (p *skillResourceProjectionPlan) Revision() int64 {
	if p == nil {
		return 0
	}
	return p.revision
}

func (p *skillResourceProjectionPlan) OperationCount() int {
	if p == nil {
		return 0
	}
	return len(p.records)
}

type skillResourceProjector struct {
	registry *skillspkg.Registry
}

func newAgentProjector(catalog *resourceCatalog[aghconfig.AgentDef]) resources.TypedProjector[aghconfig.AgentDef] {
	if catalog == nil {
		return nil
	}
	return &resourceCatalogProjector[aghconfig.AgentDef]{
		kind:      aghconfig.AgentResourceKind,
		catalog:   catalog,
		cloneSpec: cloneAgentDef,
	}
}

func newSkillProjector(registry *skillspkg.Registry) resources.TypedProjector[skillspkg.SkillResourceSpec] {
	if registry == nil {
		return nil
	}
	return &skillResourceProjector{registry: registry}
}

func (p *skillResourceProjector) Kind() resources.ResourceKind {
	return skillspkg.SkillResourceKind
}

func (p *skillResourceProjector) DependsOn() []resources.ResourceKind {
	return nil
}

func (p *skillResourceProjector) Build(
	_ context.Context,
	records []resources.Record[skillspkg.SkillResourceSpec],
) (resources.ProjectionPlan, error) {
	if p == nil || p.registry == nil {
		return nil, errors.New("daemon: skill resource projector is required")
	}
	var revision int64
	for _, record := range records {
		if record.Version > revision {
			revision = record.Version
		}
	}
	return &skillResourceProjectionPlan{
		revision: revision,
		records:  cloneResourceRecords(records, cloneSkillResourceSpec),
	}, nil
}

func (p *skillResourceProjector) Apply(ctx context.Context, plan resources.ProjectionPlan) error {
	if p == nil || p.registry == nil {
		return errors.New("daemon: skill resource projector is required")
	}
	if ctx == nil {
		return errors.New("daemon: skill resource projector apply context is required")
	}
	typed, ok := plan.(*skillResourceProjectionPlan)
	if !ok {
		return fmt.Errorf("daemon: skill resource projector plan has type %T", plan)
	}
	return p.registry.ApplyResourceRecords(typed.revision, typed.records)
}

type resourceAgentCatalog struct {
	catalog *resourceCatalog[aghconfig.AgentDef]
}

func agentCatalogDependency(catalog *resourceCatalog[aghconfig.AgentDef]) *resourceAgentCatalog {
	if catalog == nil {
		return nil
	}
	return &resourceAgentCatalog{catalog: catalog}
}

func (c *resourceAgentCatalog) ResolveAgent(
	name string,
	resolved *workspacepkg.ResolvedWorkspace,
) (aghconfig.AgentDef, error) {
	target := strings.TrimSpace(name)
	if target == "" {
		return aghconfig.AgentDef{}, errors.New("session: agent name is required")
	}
	if c == nil || c.catalog == nil {
		return resolveAgentFromWorkspaceSnapshot(target, resolved)
	}
	if agent, ok := c.lookupAgent(target, resolved); ok {
		return agent, nil
	}
	if resolved != nil {
		return resolveAgentFromWorkspaceSnapshot(target, resolved)
	}
	return aghconfig.AgentDef{}, fmt.Errorf("%w: %s", workspacepkg.ErrAgentNotAvailable, target)
}

func (c *resourceAgentCatalog) lookupAgent(
	target string,
	resolved *workspacepkg.ResolvedWorkspace,
) (aghconfig.AgentDef, bool) {
	if c == nil || c.catalog == nil {
		return aghconfig.AgentDef{}, false
	}

	workspaceID := ""
	if resolved != nil {
		workspaceID = strings.TrimSpace(resolved.ID)
	}

	var (
		globalKey      string
		globalAgent    aghconfig.AgentDef
		globalFound    bool
		workspaceKey   string
		workspaceAgent aghconfig.AgentDef
		workspaceFound bool
	)

	for _, record := range c.catalog.Snapshot() {
		if strings.TrimSpace(record.Spec.Name) != target {
			continue
		}

		sortKey := agentRecordSortKey(record)
		switch record.Scope.Kind.Normalize() {
		case resources.ResourceScopeKindGlobal:
			if !globalFound || sortKey > globalKey {
				globalKey = sortKey
				globalAgent = cloneAgentDef(record.Spec)
				globalFound = true
			}
		case resources.ResourceScopeKindWorkspace:
			if workspaceID == "" || strings.TrimSpace(record.Scope.ID) != workspaceID {
				continue
			}
			if !workspaceFound || sortKey > workspaceKey {
				workspaceKey = sortKey
				workspaceAgent = cloneAgentDef(record.Spec)
				workspaceFound = true
			}
		}
	}

	if workspaceFound {
		return workspaceAgent, true
	}
	if globalFound {
		return globalAgent, true
	}
	return aghconfig.AgentDef{}, false
}

func resolveAgentFromWorkspaceSnapshot(
	target string,
	resolved *workspacepkg.ResolvedWorkspace,
) (aghconfig.AgentDef, error) {
	if resolved == nil {
		return aghconfig.AgentDef{}, errors.New("session: resolved workspace is required")
	}
	for _, agent := range resolved.Agents {
		if strings.TrimSpace(agent.Name) == target {
			return cloneAgentDef(agent), nil
		}
	}
	return aghconfig.AgentDef{}, fmt.Errorf("%w: %s", workspacepkg.ErrAgentNotAvailable, target)
}

func (c *resourceAgentCatalog) ListAgents(ctx context.Context) ([]aghconfig.AgentDef, error) {
	if ctx == nil {
		return nil, errors.New("daemon: list agent catalog context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c == nil || c.catalog == nil {
		return nil, nil
	}
	return c.agentsForWorkspace(nil), nil
}

func (c *resourceAgentCatalog) GetAgent(ctx context.Context, name string) (aghconfig.AgentDef, error) {
	if ctx == nil {
		return aghconfig.AgentDef{}, errors.New("daemon: get agent catalog context is required")
	}
	if err := ctx.Err(); err != nil {
		return aghconfig.AgentDef{}, err
	}
	target := strings.TrimSpace(name)
	if target == "" {
		return aghconfig.AgentDef{}, errors.New("agent name is required")
	}
	for _, agent := range c.agentsForWorkspace(nil) {
		if strings.TrimSpace(agent.Name) == target {
			return cloneAgentDef(agent), nil
		}
	}
	return aghconfig.AgentDef{}, fmt.Errorf("%w: %s", os.ErrNotExist, target)
}

func (c *resourceAgentCatalog) agentsForWorkspace(resolved *workspacepkg.ResolvedWorkspace) []aghconfig.AgentDef {
	if c == nil || c.catalog == nil {
		return nil
	}
	records := c.catalog.Snapshot()
	slices.SortFunc(records, func(left, right resources.Record[aghconfig.AgentDef]) int {
		return strings.Compare(agentRecordSortKey(left), agentRecordSortKey(right))
	})
	merged := make(map[string]aghconfig.AgentDef)
	for _, record := range records {
		if record.Scope.Kind.Normalize() != resources.ResourceScopeKindGlobal {
			continue
		}
		name := strings.TrimSpace(record.Spec.Name)
		if name != "" {
			merged[name] = cloneAgentDef(record.Spec)
		}
	}
	workspaceID := ""
	if resolved != nil {
		workspaceID = strings.TrimSpace(resolved.ID)
	}
	if workspaceID != "" {
		for _, record := range records {
			if record.Scope.Kind.Normalize() != resources.ResourceScopeKindWorkspace ||
				strings.TrimSpace(record.Scope.ID) != workspaceID {
				continue
			}
			name := strings.TrimSpace(record.Spec.Name)
			if name != "" {
				merged[name] = cloneAgentDef(record.Spec)
			}
		}
	}
	names := make([]string, 0, len(merged))
	for name := range merged {
		names = append(names, name)
	}
	slices.Sort(names)
	agents := make([]aghconfig.AgentDef, 0, len(names))
	for _, name := range names {
		agents = append(agents, cloneAgentDef(merged[name]))
	}
	return agents
}

func newAgentSkillSourceSyncer(
	agentStore resources.Store[aghconfig.AgentDef],
	agentCodec resources.KindCodec[aghconfig.AgentDef],
	skillStore resources.Store[skillspkg.SkillResourceSpec],
	skillCodec resources.KindCodec[skillspkg.SkillResourceSpec],
	mcpStore resources.Store[aghconfig.MCPServer],
	mcpCodec resources.KindCodec[aghconfig.MCPServer],
	actor resources.MutationActor,
	logger *slog.Logger,
	trigger func(context.Context, resources.ResourceKind, resources.ReconcileReason) error,
	providers ...agentSkillDeclarationProvider,
) agentSkillPublisher {
	if agentStore == nil || agentCodec == nil || skillStore == nil || skillCodec == nil ||
		mcpStore == nil || mcpCodec == nil {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &agentSkillSourceSyncer{
		agentStore: agentStore,
		agentCodec: agentCodec,
		skillStore: skillStore,
		skillCodec: skillCodec,
		mcpStore:   mcpStore,
		mcpCodec:   mcpCodec,
		actor:      actor,
		logger:     logger,
		trigger:    trigger,
		providers:  append([]agentSkillDeclarationProvider(nil), providers...),
	}
}

func agentSkillSyncActor() resources.MutationActor {
	return resources.MutationActor{
		Kind: resources.MutationActorKindDaemon,
		ID:   "agent-skill-sync",
		Source: resources.ResourceSource{
			Kind: resources.ResourceSourceKind("daemon"),
			ID:   "agent-skill-sync",
		},
		MaxScope: resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
	}
}

func (s *agentSkillSourceSyncer) Sync(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("daemon: agent/skill sync context is required")
	}

	desired, err := s.desiredResources(ctx)
	if err != nil {
		return err
	}
	agentChanged, err := s.syncAgents(ctx, desired.agents)
	if err != nil {
		return err
	}
	skillChanged, err := s.syncSkills(ctx, desired.skills)
	if err != nil {
		return err
	}
	mcpChanged, err := s.syncMCPServers(ctx, desired.mcpServers)
	if err != nil {
		return err
	}

	if agentChanged && s.trigger != nil {
		if err := s.trigger(ctx, aghconfig.AgentResourceKind, resources.ReconcileReasonWrite); err != nil {
			return err
		}
	}
	if skillChanged && s.trigger != nil {
		if err := s.trigger(ctx, skillspkg.SkillResourceKind, resources.ReconcileReasonWrite); err != nil {
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

type desiredAgentResource struct {
	id      string
	scope   resources.ResourceScope
	spec    aghconfig.AgentDef
	encoded []byte
}

type desiredSkillResource struct {
	id      string
	scope   resources.ResourceScope
	spec    skillspkg.SkillResourceSpec
	encoded []byte
}

func (s *agentSkillSourceSyncer) desiredResources(ctx context.Context) (struct {
	agents     map[string]desiredAgentResource
	skills     map[string]desiredSkillResource
	mcpServers map[string]desiredMCPServerResource
}, error) {
	desired := struct {
		agents     map[string]desiredAgentResource
		skills     map[string]desiredSkillResource
		mcpServers map[string]desiredMCPServerResource
	}{
		agents:     make(map[string]desiredAgentResource),
		skills:     make(map[string]desiredSkillResource),
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
		for _, item := range items.agents {
			spec, encoded, err := validateAndEncodeAgent(ctx, s.agentCodec, item.scope, item.spec)
			if err != nil {
				return desired, err
			}
			id := managedResourceID(agentManagedIDPrefix, item.scope.Normalize(), item.sourceKey, encoded)
			desired.agents[id] = desiredAgentResource{
				id:      id,
				scope:   item.scope.Normalize(),
				spec:    spec,
				encoded: encoded,
			}
		}
		for _, item := range items.skills {
			spec, encoded, err := validateAndEncodeSkill(ctx, s.skillCodec, item.scope, item.spec)
			if err != nil {
				return desired, err
			}
			id := managedResourceID(skillManagedIDPrefix, item.scope.Normalize(), item.sourceKey, encoded)
			desired.skills[id] = desiredSkillResource{
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

func (s *agentSkillSourceSyncer) syncAgents(
	ctx context.Context,
	desired map[string]desiredAgentResource,
) (bool, error) {
	source := s.actor.Source
	current, err := s.agentStore.List(ctx, s.actor, resources.ResourceFilter{Source: &source})
	if err != nil {
		return false, fmt.Errorf("daemon: list managed agents: %w", err)
	}
	currentByID := make(map[string]resources.Record[aghconfig.AgentDef], len(current))
	for _, record := range current {
		currentByID[record.ID] = record
	}

	changed := false
	for id, desiredAgent := range desired {
		existing, ok := currentByID[id]
		if ok && s.sameAgent(existing, desiredAgent.scope, desiredAgent.encoded) {
			delete(currentByID, id)
			continue
		}
		expectedVersion := int64(0)
		if ok {
			expectedVersion = existing.Version
		}
		if _, err := s.agentStore.Put(ctx, s.actor, resources.Draft[aghconfig.AgentDef]{
			ID:              desiredAgent.id,
			Scope:           desiredAgent.scope,
			ExpectedVersion: expectedVersion,
			Spec:            desiredAgent.spec,
		}); err != nil {
			return false, fmt.Errorf("daemon: sync agent %q: %w", id, err)
		}
		changed = true
		delete(currentByID, id)
	}
	for _, stale := range currentByID {
		if err := s.agentStore.Delete(ctx, s.actor, stale.ID, stale.Version); err != nil {
			return false, fmt.Errorf("daemon: delete stale agent %q: %w", stale.ID, err)
		}
		changed = true
	}
	return changed, nil
}

func (s *agentSkillSourceSyncer) syncSkills(
	ctx context.Context,
	desired map[string]desiredSkillResource,
) (bool, error) {
	source := s.actor.Source
	current, err := s.skillStore.List(ctx, s.actor, resources.ResourceFilter{Source: &source})
	if err != nil {
		return false, fmt.Errorf("daemon: list managed skills: %w", err)
	}
	currentByID := make(map[string]resources.Record[skillspkg.SkillResourceSpec], len(current))
	for _, record := range current {
		currentByID[record.ID] = record
	}

	changed := false
	for id, desiredSkill := range desired {
		existing, ok := currentByID[id]
		if ok && s.sameSkill(existing, desiredSkill.scope, desiredSkill.encoded) {
			delete(currentByID, id)
			continue
		}
		expectedVersion := int64(0)
		if ok {
			expectedVersion = existing.Version
		}
		if _, err := s.skillStore.Put(ctx, s.actor, resources.Draft[skillspkg.SkillResourceSpec]{
			ID:              desiredSkill.id,
			Scope:           desiredSkill.scope,
			ExpectedVersion: expectedVersion,
			Spec:            desiredSkill.spec,
		}); err != nil {
			return false, fmt.Errorf("daemon: sync skill %q: %w", id, err)
		}
		changed = true
		delete(currentByID, id)
	}
	for _, stale := range currentByID {
		if err := s.skillStore.Delete(ctx, s.actor, stale.ID, stale.Version); err != nil {
			return false, fmt.Errorf("daemon: delete stale skill %q: %w", stale.ID, err)
		}
		changed = true
	}
	return changed, nil
}

func (s *agentSkillSourceSyncer) syncMCPServers(
	ctx context.Context,
	desired map[string]desiredMCPServerResource,
) (bool, error) {
	source := s.actor.Source
	current, err := s.mcpStore.List(ctx, s.actor, resources.ResourceFilter{Source: &source})
	if err != nil {
		return false, fmt.Errorf("daemon: list agent/skill mcp servers: %w", err)
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
			return false, fmt.Errorf("daemon: sync agent/skill mcp server %q: %w", id, err)
		}
		changed = true
		delete(currentByID, id)
	}
	for _, stale := range currentByID {
		if err := s.mcpStore.Delete(ctx, s.actor, stale.ID, stale.Version); err != nil {
			return false, fmt.Errorf("daemon: delete stale agent/skill mcp server %q: %w", stale.ID, err)
		}
		changed = true
	}
	return changed, nil
}

func (s *agentSkillSourceSyncer) sameAgent(
	record resources.Record[aghconfig.AgentDef],
	scope resources.ResourceScope,
	encoded []byte,
) bool {
	if record.Scope != scope {
		return false
	}
	currentEncoded, err := s.agentCodec.Encode(record.Spec)
	if err != nil {
		return false
	}
	return bytes.Equal(currentEncoded, encoded)
}

func (s *agentSkillSourceSyncer) sameSkill(
	record resources.Record[skillspkg.SkillResourceSpec],
	scope resources.ResourceScope,
	encoded []byte,
) bool {
	if record.Scope != scope {
		return false
	}
	currentEncoded, err := s.skillCodec.Encode(record.Spec)
	if err != nil {
		return false
	}
	return bytes.Equal(currentEncoded, encoded)
}

func (s *agentSkillSourceSyncer) sameMCPServer(
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

func (d *Daemon) newAgentSkillPublisher(
	state *bootState,
	registry *extensionpkg.Registry,
) (agentSkillPublisher, error) {
	publisher := agentSkillPublisher(agentSkillPublisherFunc(func(context.Context) error { return nil }))
	if state == nil {
		return publisher, nil
	}
	if state.resourceKernel == nil || state.resourceCodecs == nil {
		return publisher, nil
	}

	agentCodec, err := resources.ResolveCodec[aghconfig.AgentDef](state.resourceCodecs, aghconfig.AgentResourceKind)
	if err != nil {
		return nil, fmt.Errorf("daemon: resolve agent codec: %w", err)
	}
	agentStore, err := resources.NewStore(state.resourceKernel, agentCodec)
	if err != nil {
		return nil, fmt.Errorf("daemon: create agent store: %w", err)
	}
	skillCodec, err := resources.ResolveCodec[skillspkg.SkillResourceSpec](
		state.resourceCodecs,
		skillspkg.SkillResourceKind,
	)
	if err != nil {
		return nil, fmt.Errorf("daemon: resolve skill codec: %w", err)
	}
	skillStore, err := resources.NewStore(state.resourceKernel, skillCodec)
	if err != nil {
		return nil, fmt.Errorf("daemon: create skill store: %w", err)
	}
	mcpCodec, err := resources.ResolveCodec[aghconfig.MCPServer](state.resourceCodecs, aghconfig.MCPServerResourceKind)
	if err != nil {
		return nil, fmt.Errorf("daemon: resolve mcp server codec for agent/skill sync: %w", err)
	}
	mcpStore, err := resources.NewStore(state.resourceKernel, mcpCodec)
	if err != nil {
		return nil, fmt.Errorf("daemon: create mcp server store for agent/skill sync: %w", err)
	}

	return newAgentSkillSourceSyncer(
		agentStore,
		agentCodec,
		skillStore,
		skillCodec,
		mcpStore,
		mcpCodec,
		agentSkillSyncActor(),
		state.logger,
		func(ctx context.Context, kind resources.ResourceKind, reason resources.ReconcileReason) error {
			if state.resourceReconcile == nil {
				return nil
			}
			return state.resourceReconcile.Trigger(ctx, kind, reason)
		},
		daemonAgentSkillDeclarationProvider(
			d.homePaths,
			state.registry,
			state.workspaceResolver,
			state.skillsRegistry,
			state.logger,
		),
		extensionAgentSkillDeclarationProvider(registry, state.currentExtensionRuntime, state.logger),
	), nil
}

func daemonAgentSkillDeclarationProvider(
	homePaths aghconfig.HomePaths,
	registry Registry,
	workspaceResolver workspacepkg.RuntimeResolver,
	skillsRegistry *skillspkg.Registry,
	logger *slog.Logger,
) agentSkillDeclarationProvider {
	return func(ctx context.Context) (agentSkillDesiredResources, error) {
		desired := agentSkillDesiredResources{}
		globalScope := resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal}
		globalAgents, err := aghconfig.LoadWorkspaceAgentDefs("", nil, homePaths)
		if err != nil {
			return agentSkillDesiredResources{}, fmt.Errorf("daemon: discover global agents: %w", err)
		}
		appendAgentResources(&desired, globalScope, "config/global", globalAgents)

		if skillsRegistry != nil {
			globalSkills, _, err := skillsRegistry.DiscoverGlobal(ctx)
			if err != nil {
				return agentSkillDesiredResources{}, fmt.Errorf("daemon: discover global skills: %w", err)
			}
			appendSkillResources(&desired, globalScope, "skills/global", globalSkills)
		}

		workspaces, err := registeredWorkspaces(ctx, registry, workspaceResolver, logger)
		if err != nil {
			return agentSkillDesiredResources{}, err
		}
		for idx := range workspaces {
			resolved := &workspaces[idx]
			scope := resources.ResourceScope{
				Kind: resources.ResourceScopeKindWorkspace,
				ID:   strings.TrimSpace(resolved.ID),
			}
			appendAgentResources(&desired, scope, "config/workspace/"+scope.ID, resolved.Agents)
			if skillsRegistry == nil {
				continue
			}
			workspaceSkills, _, err := skillsRegistry.DiscoverWorkspace(ctx, resolved)
			if err != nil {
				return agentSkillDesiredResources{}, fmt.Errorf(
					"daemon: discover workspace %q skills: %w",
					scope.ID,
					err,
				)
			}
			appendSkillResources(&desired, scope, "skills/workspace/"+scope.ID, workspaceSkills)
		}

		return desired, nil
	}
}

func extensionAgentSkillDeclarationProvider(
	registry *extensionpkg.Registry,
	runtime func() extensionRuntime,
	logger *slog.Logger,
) agentSkillDeclarationProvider {
	return func(_ context.Context) (agentSkillDesiredResources, error) {
		if registry == nil || runtime == nil {
			return agentSkillDesiredResources{}, nil
		}
		manager := runtime()
		if manager == nil {
			return agentSkillDesiredResources{}, nil
		}

		infos, err := registry.List()
		if err != nil {
			return agentSkillDesiredResources{}, fmt.Errorf("daemon: list extensions for agent/skill sync: %w", err)
		}
		slices.SortFunc(infos, func(left, right extensionpkg.ExtensionInfo) int {
			return strings.Compare(left.Name, right.Name)
		})

		desired := agentSkillDesiredResources{}
		globalScope := resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal}
		for _, info := range infos {
			if !info.Enabled {
				continue
			}
			ext, err := loadExtensionSnapshot(registry, manager, logger, info.Name)
			if err != nil {
				return agentSkillDesiredResources{}, fmt.Errorf(
					"daemon: load extension %q for agent/skill sync: %w",
					info.Name,
					err,
				)
			}
			if ext == nil || ext.Manifest == nil || !ext.Status.Registered {
				continue
			}
			appendAgentResources(&desired, globalScope, "extension/"+ext.Info.Name+"/agents", ext.Agents)
			appendSkillResources(&desired, globalScope, "extension/"+ext.Info.Name+"/skills", ext.Skills)
		}

		return desired, nil
	}
}

func appendAgentResources(
	desired *agentSkillDesiredResources,
	scope resources.ResourceScope,
	sourcePrefix string,
	agents []aghconfig.AgentDef,
) {
	if desired == nil {
		return
	}
	for _, agent := range agents {
		name := strings.TrimSpace(agent.Name)
		if name == "" {
			continue
		}
		desired.agents = append(desired.agents, agentPublicationInput{
			sourceKey: sourcePrefix + "/agent/" + name,
			scope:     scope,
			spec:      cloneAgentDef(agent),
		})
		for _, server := range agent.MCPServers {
			serverName := strings.TrimSpace(server.Name)
			if serverName == "" {
				continue
			}
			desired.mcpServers = append(desired.mcpServers, mcpServerPublicationInput{
				sourceKey: sourcePrefix + "/agent/" + name + "/mcp_server/" + serverName,
				scope:     scope,
				spec:      cloneDaemonMCPServer(server),
			})
		}
	}
}

func appendSkillResources(
	desired *agentSkillDesiredResources,
	scope resources.ResourceScope,
	sourcePrefix string,
	skills []*skillspkg.Skill,
) {
	if desired == nil {
		return
	}
	for _, skill := range skills {
		if skill == nil {
			continue
		}
		name := strings.TrimSpace(skill.Meta.Name)
		if name == "" {
			continue
		}
		desired.skills = append(desired.skills, skillPublicationInput{
			sourceKey: sourcePrefix + "/skill/" + name,
			scope:     scope,
			spec:      skillspkg.SkillToResourceSpec(skill),
		})
		for _, server := range skill.MCPServers {
			serverName := strings.TrimSpace(server.Name)
			if serverName == "" {
				continue
			}
			desired.mcpServers = append(desired.mcpServers, mcpServerPublicationInput{
				sourceKey: sourcePrefix + "/skill/" + name + "/mcp_server/" + serverName,
				scope:     scope,
				spec:      mcpServerFromSkillDecl(server),
			})
		}
	}
}

func mcpServerFromSkillDecl(decl skillspkg.MCPServerDecl) aghconfig.MCPServer {
	return aghconfig.MCPServer{
		Name:    strings.TrimSpace(decl.Name),
		Command: strings.TrimSpace(decl.Command),
		Args:    slices.Clone(decl.Args),
		Env:     cloneStringMap(decl.Env),
	}
}

func validateAndEncodeAgent(
	ctx context.Context,
	codec resources.KindCodec[aghconfig.AgentDef],
	scope resources.ResourceScope,
	spec aghconfig.AgentDef,
) (aghconfig.AgentDef, []byte, error) {
	encoded, err := codec.Encode(spec)
	if err != nil {
		return aghconfig.AgentDef{}, nil, err
	}
	validated, err := codec.DecodeAndValidate(ctx, scope.Normalize(), encoded)
	if err != nil {
		return aghconfig.AgentDef{}, nil, err
	}
	canonical, err := codec.Encode(validated)
	if err != nil {
		return aghconfig.AgentDef{}, nil, err
	}
	return validated, canonical, nil
}

func validateAndEncodeSkill(
	ctx context.Context,
	codec resources.KindCodec[skillspkg.SkillResourceSpec],
	scope resources.ResourceScope,
	spec skillspkg.SkillResourceSpec,
) (skillspkg.SkillResourceSpec, []byte, error) {
	encoded, err := codec.Encode(spec)
	if err != nil {
		return skillspkg.SkillResourceSpec{}, nil, err
	}
	validated, err := codec.DecodeAndValidate(ctx, scope.Normalize(), encoded)
	if err != nil {
		return skillspkg.SkillResourceSpec{}, nil, err
	}
	canonical, err := codec.Encode(validated)
	if err != nil {
		return skillspkg.SkillResourceSpec{}, nil, err
	}
	return validated, canonical, nil
}

func cloneAgentDef(agent aghconfig.AgentDef) aghconfig.AgentDef {
	return aghconfig.AgentDef{
		Name:         strings.TrimSpace(agent.Name),
		Provider:     strings.TrimSpace(agent.Provider),
		Command:      strings.TrimSpace(agent.Command),
		Model:        strings.TrimSpace(agent.Model),
		Tools:        slices.Clone(agent.Tools),
		Toolsets:     slices.Clone(agent.Toolsets),
		DenyTools:    slices.Clone(agent.DenyTools),
		Permissions:  strings.TrimSpace(agent.Permissions),
		MCPServers:   cloneMCPServers(agent.MCPServers),
		Hooks:        cloneHookDecls(agent.Hooks),
		Capabilities: agent.Capabilities.Clone(),
		Prompt:       strings.TrimSpace(agent.Prompt),
	}
}

func cloneMCPServers(src []aghconfig.MCPServer) []aghconfig.MCPServer {
	if len(src) == 0 {
		return nil
	}
	cloned := make([]aghconfig.MCPServer, 0, len(src))
	for _, server := range src {
		cloned = append(cloned, cloneDaemonMCPServer(server))
	}
	return cloned
}

func cloneHookDecls(src []hookspkg.HookDecl) []hookspkg.HookDecl {
	if len(src) == 0 {
		return nil
	}
	cloned := make([]hookspkg.HookDecl, 0, len(src))
	for _, decl := range src {
		next := decl
		next.Args = slices.Clone(decl.Args)
		next.Env = cloneStringMap(decl.Env)
		next.Metadata = cloneStringMap(decl.Metadata)
		if decl.Matcher.ToolReadOnly != nil {
			value := *decl.Matcher.ToolReadOnly
			next.Matcher.ToolReadOnly = &value
		}
		cloned = append(cloned, next)
	}
	return cloned
}

func cloneSkillResourceSpec(src skillspkg.SkillResourceSpec) skillspkg.SkillResourceSpec {
	skill, err := skillspkg.SkillFromResourceSpec(src)
	if err != nil {
		return src
	}
	return skillspkg.SkillToResourceSpec(skill)
}

func agentRecordSortKey(record resources.Record[aghconfig.AgentDef]) string {
	return string(record.Scope.Kind.Normalize()) + "\x00" +
		strings.TrimSpace(record.Scope.ID) + "\x00" +
		string(record.Source.Kind.Normalize()) + "\x00" +
		strings.TrimSpace(record.Source.ID) + "\x00" +
		strings.TrimSpace(record.ID)
}
