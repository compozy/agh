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

	bundlepkg "github.com/pedronauck/agh/internal/bundles"
	aghconfig "github.com/pedronauck/agh/internal/config"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	"github.com/pedronauck/agh/internal/heartbeat"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/session"
	skillspkg "github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/soul"
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
	raw        resources.RawStore
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

func newSoulProjector(catalog *resourceCatalog[soul.ResourceSpec]) resources.TypedProjector[soul.ResourceSpec] {
	if catalog == nil {
		return nil
	}
	return &resourceCatalogProjector[soul.ResourceSpec]{
		kind:      soul.ResourceKind,
		catalog:   catalog,
		cloneSpec: cloneSoulResourceSpec,
	}
}

func newHeartbeatProjector(
	catalog *resourceCatalog[heartbeat.ResourceSpec],
) resources.TypedProjector[heartbeat.ResourceSpec] {
	if catalog == nil {
		return nil
	}
	return &resourceCatalogProjector[heartbeat.ResourceSpec]{
		kind:      heartbeat.ResourceKind,
		catalog:   catalog,
		cloneSpec: cloneHeartbeatResourceSpec,
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
	catalog          *resourceCatalog[aghconfig.AgentDef]
	soulCatalog      *resourceCatalog[soul.ResourceSpec]
	heartbeatCatalog *resourceCatalog[heartbeat.ResourceSpec]
}

var _ session.AgentArtifactResolver = (*resourceAgentCatalog)(nil)
var _ heartbeat.PolicyResolver = (*resourceAgentCatalog)(nil)

type agentSidecarCatalogs struct {
	soul      *resourceCatalog[soul.ResourceSpec]
	heartbeat *resourceCatalog[heartbeat.ResourceSpec]
}

func agentCatalogDependency(
	catalog *resourceCatalog[aghconfig.AgentDef],
	sidecars ...agentSidecarCatalogs,
) *resourceAgentCatalog {
	if catalog == nil {
		return nil
	}
	dependency := &resourceAgentCatalog{catalog: catalog}
	if len(sidecars) > 0 {
		dependency.soulCatalog = sidecars[0].soul
		dependency.heartbeatCatalog = sidecars[0].heartbeat
	}
	return dependency
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
	if record, ok := c.lookupAgentRecord(target, resolved); ok {
		return cloneAgentDef(record.Spec), nil
	}
	if resolved != nil {
		return resolveAgentFromWorkspaceSnapshot(target, resolved)
	}
	return aghconfig.AgentDef{}, fmt.Errorf("%w: %s", workspacepkg.ErrAgentNotAvailable, target)
}

func (c *resourceAgentCatalog) lookupAgentRecord(
	target string,
	resolved *workspacepkg.ResolvedWorkspace,
) (resources.Record[aghconfig.AgentDef], bool) {
	if c == nil || c.catalog == nil {
		return resources.Record[aghconfig.AgentDef]{}, false
	}

	workspaceID := ""
	if resolved != nil {
		workspaceID = strings.TrimSpace(resolved.ID)
	}

	var (
		globalKey      string
		globalAgent    resources.Record[aghconfig.AgentDef]
		globalFound    bool
		workspaceKey   string
		workspaceAgent resources.Record[aghconfig.AgentDef]
		workspaceFound bool
	)

	c.catalog.mu.RLock()
	defer c.catalog.mu.RUnlock()
	for _, record := range c.catalog.records {
		if strings.TrimSpace(record.Spec.Name) != target {
			continue
		}

		sortKey := agentRecordSortKey(record)
		switch record.Scope.Kind.Normalize() {
		case resources.ResourceScopeKindGlobal:
			if !globalFound || sortKey > globalKey {
				globalKey = sortKey
				globalAgent = record
				globalFound = true
			}
		case resources.ResourceScopeKindWorkspace:
			if workspaceID == "" || strings.TrimSpace(record.Scope.ID) != workspaceID {
				continue
			}
			if !workspaceFound || sortKey > workspaceKey {
				workspaceKey = sortKey
				workspaceAgent = record
				workspaceFound = true
			}
		}
	}

	if workspaceFound {
		return c.catalog.cloneRecord(workspaceAgent), true
	}
	if globalFound {
		return c.catalog.cloneRecord(globalAgent), true
	}
	return resources.Record[aghconfig.AgentDef]{}, false
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

func (c *resourceAgentCatalog) ResolveAgentArtifacts(
	name string,
	resolved *workspacepkg.ResolvedWorkspace,
) (session.AgentArtifacts, error) {
	target := strings.TrimSpace(name)
	if target == "" {
		return session.AgentArtifacts{}, errors.New("session: agent name is required")
	}
	if c == nil || c.catalog == nil {
		agent, err := resolveAgentFromWorkspaceSnapshot(target, resolved)
		if err != nil {
			return session.AgentArtifacts{}, err
		}
		return session.AgentArtifacts{Agent: agent}, nil
	}
	record, ok := c.lookupAgentRecord(target, resolved)
	if !ok {
		agent, err := resolveAgentFromWorkspaceSnapshot(target, resolved)
		if err != nil {
			return session.AgentArtifacts{}, err
		}
		return session.AgentArtifacts{Agent: agent}, nil
	}
	artifacts := session.AgentArtifacts{
		Agent:        cloneAgentDef(record.Spec),
		ResourceID:   strings.TrimSpace(record.ID),
		OwnerKind:    string(record.Owner.Kind.Normalize()),
		OwnerID:      strings.TrimSpace(record.Owner.ID),
		Scope:        record.Scope.Normalize(),
		PackageOwned: record.Owner.Kind.Normalize() == bundlepkg.BundleActivationOwnerKind,
	}
	if c.soulCatalog != nil {
		if spec, ok := c.lookupSoulForAgent(record); ok {
			artifacts.SoulSourcePath = spec.SourcePath
			artifacts.SoulBody = spec.Body
		}
	}
	if c.heartbeatCatalog != nil {
		if spec, ok := c.lookupHeartbeatForAgent(record); ok {
			artifacts.HeartbeatSourcePath = spec.SourcePath
			artifacts.HeartbeatBody = spec.Body
		}
	}
	return artifacts, nil
}

func (c *resourceAgentCatalog) ResolveHeartbeatPolicy(
	ctx context.Context,
	target heartbeat.AuthoringTarget,
) (heartbeat.ResolvedPolicy, bool, error) {
	if ctx == nil {
		return heartbeat.ResolvedPolicy{}, false, errors.New("daemon: heartbeat policy context is required")
	}
	if err := ctx.Err(); err != nil {
		return heartbeat.ResolvedPolicy{}, false, err
	}
	config := target.Config
	if config == (aghconfig.HeartbeatConfig{}) {
		config = aghconfig.DefaultHeartbeatConfig()
	}
	workspace := &workspacepkg.ResolvedWorkspace{
		Workspace: workspacepkg.Workspace{
			ID:      strings.TrimSpace(target.WorkspaceID),
			RootDir: strings.TrimSpace(target.WorkspaceRoot),
		},
		Config: aghconfig.Config{
			Agents: aghconfig.AgentsConfig{Heartbeat: config},
		},
	}
	artifacts, err := c.ResolveAgentArtifacts(target.AgentName, workspace)
	if err != nil {
		if errors.Is(err, workspacepkg.ErrAgentNotAvailable) {
			return heartbeat.ResolvedPolicy{}, false, nil
		}
		return heartbeat.ResolvedPolicy{}, false, err
	}
	if !artifacts.PackageOwned {
		return heartbeat.ResolvedPolicy{}, false, nil
	}
	sourcePath := strings.TrimSpace(artifacts.HeartbeatSourcePath)
	if strings.TrimSpace(artifacts.HeartbeatBody) == "" {
		policy, emptyErr := heartbeat.Empty(config, sourcePath)
		return policy, true, emptyErr
	}
	policy, parseErr := heartbeat.Parse(ctx, heartbeat.ParseRequest{
		SourcePath:    sourcePath,
		WorkspaceRoot: strings.TrimSpace(target.WorkspaceRoot),
		Content:       []byte(artifacts.HeartbeatBody),
		Config:        config,
	})
	return policy, true, parseErr
}

func (c *resourceAgentCatalog) lookupSoulForAgent(
	agent resources.Record[aghconfig.AgentDef],
) (soul.ResourceSpec, bool) {
	if c == nil || c.soulCatalog == nil {
		return soul.ResourceSpec{}, false
	}
	var (
		bestKey string
		best    soul.ResourceSpec
		found   bool
	)
	for _, record := range c.soulCatalog.Snapshot() {
		if !sidecarRecordMatchesAgent(record.Scope, record.Owner, record.Spec.AgentResourceID, agent) {
			continue
		}
		sortKey := sidecarRecordSortKey(record)
		if !found || sortKey > bestKey {
			bestKey = sortKey
			best = cloneSoulResourceSpec(record.Spec)
			found = true
		}
	}
	return best, found
}

func (c *resourceAgentCatalog) lookupHeartbeatForAgent(
	agent resources.Record[aghconfig.AgentDef],
) (heartbeat.ResourceSpec, bool) {
	if c == nil || c.heartbeatCatalog == nil {
		return heartbeat.ResourceSpec{}, false
	}
	var (
		bestKey string
		best    heartbeat.ResourceSpec
		found   bool
	)
	for _, record := range c.heartbeatCatalog.Snapshot() {
		if !sidecarRecordMatchesAgent(record.Scope, record.Owner, record.Spec.AgentResourceID, agent) {
			continue
		}
		sortKey := sidecarRecordSortKey(record)
		if !found || sortKey > bestKey {
			bestKey = sortKey
			best = cloneHeartbeatResourceSpec(record.Spec)
			found = true
		}
	}
	return best, found
}

func sidecarRecordMatchesAgent(
	scope resources.ResourceScope,
	owner resources.ResourceOwner,
	agentResourceID string,
	agent resources.Record[aghconfig.AgentDef],
) bool {
	return strings.TrimSpace(agentResourceID) == strings.TrimSpace(agent.ID) &&
		scope.Normalize() == agent.Scope.Normalize() &&
		owner.Normalize() == agent.Owner.Normalize()
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
	raw resources.RawStore,
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
	if raw == nil || agentStore == nil || agentCodec == nil || skillStore == nil || skillCodec == nil ||
		mcpStore == nil || mcpCodec == nil {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &agentSkillSourceSyncer{
		raw:        raw,
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
	current, err := s.raw.ListRaw(ctx, s.actor, resources.ResourceFilter{
		Kind:   aghconfig.AgentResourceKind,
		Source: &source,
	})
	if err != nil {
		return false, fmt.Errorf("daemon: list managed agents: %w", err)
	}
	currentByID := make(map[string]resources.RawRecord, len(current))
	for _, record := range current {
		currentByID[record.ID] = record
	}

	changed := false
	for id, desiredAgent := range desired {
		existing, ok := currentByID[id]
		if ok && sameManagedRawRecord(existing, desiredAgent.scope, desiredAgent.encoded) {
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
	current, err := s.raw.ListRaw(ctx, s.actor, resources.ResourceFilter{
		Kind:   skillspkg.SkillResourceKind,
		Source: &source,
	})
	if err != nil {
		return false, fmt.Errorf("daemon: list managed skills: %w", err)
	}
	currentByID := make(map[string]resources.RawRecord, len(current))
	for _, record := range current {
		currentByID[record.ID] = record
	}

	changed := false
	for id, desiredSkill := range desired {
		existing, ok := currentByID[id]
		if ok && sameManagedRawRecord(existing, desiredSkill.scope, desiredSkill.encoded) {
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
	current, err := s.raw.ListRaw(ctx, s.actor, resources.ResourceFilter{
		Kind:   aghconfig.MCPServerResourceKind,
		Source: &source,
	})
	if err != nil {
		return false, fmt.Errorf("daemon: list agent/skill mcp servers: %w", err)
	}
	currentByID := make(map[string]resources.RawRecord, len(current))
	for _, record := range current {
		currentByID[record.ID] = record
	}

	changed := false
	for id, desiredServer := range desired {
		existing, ok := currentByID[id]
		if ok && sameManagedRawRecord(existing, desiredServer.scope, desiredServer.encoded) {
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

func sameManagedRawRecord(
	record resources.RawRecord,
	scope resources.ResourceScope,
	encoded []byte,
) bool {
	if record.Scope != scope {
		return false
	}
	return bytes.Equal(record.SpecJSON, encoded)
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
		state.resourceKernel,
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
	return func(ctx context.Context) (agentSkillDesiredResources, error) {
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
			ext, err := loadExtensionSnapshot(ctx, registry, manager, logger, info.Name)
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
	return aghconfig.CloneAgentDef(agent)
}

func cloneSoulResourceSpec(spec soul.ResourceSpec) soul.ResourceSpec {
	return soul.ResourceSpec{
		AgentName:       strings.TrimSpace(spec.AgentName),
		AgentResourceID: strings.TrimSpace(spec.AgentResourceID),
		SourcePath:      strings.TrimSpace(spec.SourcePath),
		Body:            spec.Body,
	}
}

func cloneHeartbeatResourceSpec(spec heartbeat.ResourceSpec) heartbeat.ResourceSpec {
	return heartbeat.ResourceSpec{
		AgentName:       strings.TrimSpace(spec.AgentName),
		AgentResourceID: strings.TrimSpace(spec.AgentResourceID),
		SourcePath:      strings.TrimSpace(spec.SourcePath),
		Body:            spec.Body,
	}
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

func sidecarRecordSortKey[T any](record resources.Record[T]) string {
	return string(record.Scope.Kind.Normalize()) + "\x00" +
		strings.TrimSpace(record.Scope.ID) + "\x00" +
		string(record.Owner.Kind.Normalize()) + "\x00" +
		strings.TrimSpace(record.Owner.ID) + "\x00" +
		strings.TrimSpace(record.ID)
}
