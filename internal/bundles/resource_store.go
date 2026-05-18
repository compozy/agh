package bundles

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/heartbeat"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/soul"
)

// ResourceStore persists bundles, activations, and owned activation fan-out through canonical resources.
type ResourceStore struct {
	bundles         resources.Store[BundleResourceSpec]
	bundleCodec     resources.KindCodec[BundleResourceSpec]
	activations     resources.Store[ActivationResourceSpec]
	activationCodec resources.KindCodec[ActivationResourceSpec]
	agents          resources.Store[aghconfig.AgentDef]
	agentCodec      resources.KindCodec[aghconfig.AgentDef]
	souls           resources.Store[soul.ResourceSpec]
	soulCodec       resources.KindCodec[soul.ResourceSpec]
	heartbeats      resources.Store[heartbeat.ResourceSpec]
	heartbeatCodec  resources.KindCodec[heartbeat.ResourceSpec]
	jobs            resources.Store[automationpkg.Job]
	jobCodec        resources.KindCodec[automationpkg.Job]
	triggers        resources.Store[automationpkg.Trigger]
	triggerCodec    resources.KindCodec[automationpkg.Trigger]
	bridges         resources.Store[bridgepkg.BridgeInstanceSpec]
	bridgeCodec     resources.KindCodec[bridgepkg.BridgeInstanceSpec]
	actor           resources.MutationActor
	trigger         func(context.Context, resources.ResourceKind, resources.ReconcileReason) error
	now             func() time.Time
}

// ResourceStoreConfig groups the typed stores used by bundle resource projection.
type ResourceStoreConfig struct {
	Bundles         resources.Store[BundleResourceSpec]
	BundleCodec     resources.KindCodec[BundleResourceSpec]
	Activations     resources.Store[ActivationResourceSpec]
	ActivationCodec resources.KindCodec[ActivationResourceSpec]
	Agents          resources.Store[aghconfig.AgentDef]
	AgentCodec      resources.KindCodec[aghconfig.AgentDef]
	Souls           resources.Store[soul.ResourceSpec]
	SoulCodec       resources.KindCodec[soul.ResourceSpec]
	Heartbeats      resources.Store[heartbeat.ResourceSpec]
	HeartbeatCodec  resources.KindCodec[heartbeat.ResourceSpec]
	Jobs            resources.Store[automationpkg.Job]
	JobCodec        resources.KindCodec[automationpkg.Job]
	Triggers        resources.Store[automationpkg.Trigger]
	TriggerCodec    resources.KindCodec[automationpkg.Trigger]
	Bridges         resources.Store[bridgepkg.BridgeInstanceSpec]
	BridgeCodec     resources.KindCodec[bridgepkg.BridgeInstanceSpec]
	Actor           resources.MutationActor
	Trigger         func(context.Context, resources.ResourceKind, resources.ReconcileReason) error
	Now             func() time.Time
}

var _ Store = (*ResourceStore)(nil)

var bundleActivationOwnedKindAllowlist = map[resources.ResourceKind]struct{}{
	aghconfig.AgentResourceKind:          {},
	soul.ResourceKind:                    {},
	heartbeat.ResourceKind:               {},
	automationpkg.JobResourceKind:        {},
	automationpkg.TriggerResourceKind:    {},
	bridgepkg.BridgeInstanceResourceKind: {},
}

func bundleActivationOwnedKindAllowed(kind resources.ResourceKind) bool {
	_, ok := bundleActivationOwnedKindAllowlist[kind.Normalize()]
	return ok
}

// NewResourceStore constructs a resource-backed bundle store.
func NewResourceStore(cfg ResourceStoreConfig) (*ResourceStore, error) {
	if cfg.Bundles == nil {
		return nil, errors.New("bundles: bundle resource store is required")
	}
	if cfg.BundleCodec == nil {
		return nil, errors.New("bundles: bundle resource codec is required")
	}
	if cfg.Activations == nil {
		return nil, errors.New("bundles: activation resource store is required")
	}
	if cfg.ActivationCodec == nil {
		return nil, errors.New("bundles: activation resource codec is required")
	}
	if cfg.Agents == nil || cfg.AgentCodec == nil {
		return nil, errors.New("bundles: agent resource store and codec are required")
	}
	if cfg.Souls == nil || cfg.SoulCodec == nil {
		return nil, errors.New("bundles: soul resource store and codec are required")
	}
	if cfg.Heartbeats == nil || cfg.HeartbeatCodec == nil {
		return nil, errors.New("bundles: heartbeat resource store and codec are required")
	}
	if cfg.Jobs == nil || cfg.JobCodec == nil {
		return nil, errors.New("bundles: automation job resource store and codec are required")
	}
	if cfg.Triggers == nil || cfg.TriggerCodec == nil {
		return nil, errors.New("bundles: automation trigger resource store and codec are required")
	}
	if cfg.Bridges == nil || cfg.BridgeCodec == nil {
		return nil, errors.New("bundles: bridge instance resource store and codec are required")
	}
	if cfg.Actor.Kind == "" {
		cfg.Actor = defaultBundleResourceActor()
	}
	if cfg.Now == nil {
		cfg.Now = func() time.Time { return time.Now().UTC() }
	}
	return &ResourceStore{
		bundles:         cfg.Bundles,
		bundleCodec:     cfg.BundleCodec,
		activations:     cfg.Activations,
		activationCodec: cfg.ActivationCodec,
		agents:          cfg.Agents,
		agentCodec:      cfg.AgentCodec,
		souls:           cfg.Souls,
		soulCodec:       cfg.SoulCodec,
		heartbeats:      cfg.Heartbeats,
		heartbeatCodec:  cfg.HeartbeatCodec,
		jobs:            cfg.Jobs,
		jobCodec:        cfg.JobCodec,
		triggers:        cfg.Triggers,
		triggerCodec:    cfg.TriggerCodec,
		bridges:         cfg.Bridges,
		bridgeCodec:     cfg.BridgeCodec,
		actor:           cfg.Actor,
		trigger:         cfg.Trigger,
		now:             cfg.Now,
	}, nil
}

func defaultBundleResourceActor() resources.MutationActor {
	return resources.MutationActor{
		Kind:     resources.MutationActorKindDaemon,
		ID:       "bundle-resource",
		Source:   resources.ResourceSource{Kind: resources.ResourceSourceKind("daemon"), ID: "bundle-resource"},
		MaxScope: resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
	}
}

func (s *ResourceStore) CreateBundleActivation(ctx context.Context, activation Activation) error {
	activation, err := activation.Validated()
	if err != nil {
		return err
	}
	if activation.CreatedAt.IsZero() {
		activation.CreatedAt = s.now().UTC()
	}
	if activation.UpdatedAt.IsZero() {
		activation.UpdatedAt = activation.CreatedAt
	}
	_, err = s.activations.Put(ctx, s.actor, resources.Draft[ActivationResourceSpec]{
		ID:              strings.TrimSpace(activation.ID),
		Scope:           resourceScopeForActivation(activation),
		ExpectedVersion: 0,
		Spec:            activationResourceSpecFromActivation(activation),
	})
	if err != nil {
		return fmt.Errorf("bundles: create activation resource %q: %w", activation.ID, err)
	}
	return nil
}

func (s *ResourceStore) UpdateBundleActivation(ctx context.Context, activation Activation) error {
	activation, err := activation.Validated()
	if err != nil {
		return err
	}
	current, err := s.activations.Get(ctx, s.actor, strings.TrimSpace(activation.ID))
	if err != nil {
		return mapActivationResourceError("get", activation.ID, err)
	}
	if activation.UpdatedAt.IsZero() {
		activation.UpdatedAt = s.now().UTC()
	}
	_, err = s.activations.Put(ctx, s.actor, resources.Draft[ActivationResourceSpec]{
		ID:              strings.TrimSpace(activation.ID),
		Scope:           resourceScopeForActivation(activation),
		ExpectedVersion: current.Version,
		Spec:            activationResourceSpecFromActivation(activation),
	})
	if err != nil {
		return fmt.Errorf("bundles: update activation resource %q: %w", activation.ID, err)
	}
	return nil
}

func (s *ResourceStore) DeleteBundleActivation(ctx context.Context, id string) error {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return errors.New("bundles: activation id is required")
	}
	current, err := s.activations.Get(ctx, s.actor, trimmed)
	if err != nil {
		return mapActivationResourceError("get", trimmed, err)
	}
	if err := s.activations.Delete(ctx, s.actor, trimmed, current.Version); err != nil {
		return mapActivationResourceError("delete", trimmed, err)
	}
	return nil
}

func (s *ResourceStore) GetBundleActivation(ctx context.Context, id string) (Activation, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return Activation{}, errors.New("bundles: activation id is required")
	}
	record, err := s.activations.Get(ctx, s.actor, trimmed)
	if err != nil {
		return Activation{}, mapActivationResourceError("get", trimmed, err)
	}
	return activationFromResourceRecord(record), nil
}

func (s *ResourceStore) ListBundleActivations(ctx context.Context) ([]Activation, error) {
	records, err := s.activations.List(ctx, s.actor, resources.ResourceFilter{
		Kind: BundleActivationResourceKind,
	})
	if err != nil {
		return nil, fmt.Errorf("bundles: list activation resources: %w", err)
	}
	activations := make([]Activation, 0, len(records))
	for _, record := range records {
		activations = append(activations, activationFromResourceRecord(record))
	}
	slices.SortFunc(activations, compareActivations)
	return activations, nil
}

func (s *ResourceStore) ListBundleResources(
	ctx context.Context,
) ([]resources.Record[BundleResourceSpec], error) {
	records, err := s.bundles.List(ctx, s.actor, resources.ResourceFilter{Kind: BundleResourceKind})
	if err != nil {
		return nil, fmt.Errorf("bundles: list bundle resources: %w", err)
	}
	slices.SortFunc(records, func(left, right resources.Record[BundleResourceSpec]) int {
		if cmp := strings.Compare(left.Spec.ExtensionName, right.Spec.ExtensionName); cmp != 0 {
			return cmp
		}
		return strings.Compare(left.Spec.Bundle.Name, right.Spec.Bundle.Name)
	})
	return records, nil
}

func (s *ResourceStore) ListAgentResources(
	ctx context.Context,
) ([]resources.Record[aghconfig.AgentDef], error) {
	records, err := s.agents.List(ctx, s.actor, resources.ResourceFilter{Kind: aghconfig.AgentResourceKind})
	if err != nil {
		return nil, fmt.Errorf("bundles: list agent resources: %w", err)
	}
	return records, nil
}

func (s *ResourceStore) ListBundleActivationInventory(
	ctx context.Context,
	activationID string,
) ([]InventoryItem, error) {
	owner := ownerForActivation(activationID)
	if err := owner.Validate("owner"); err != nil {
		return nil, err
	}
	items := make([]InventoryItem, 0)
	collectors := []func(context.Context, resources.ResourceOwner) ([]InventoryItem, error){
		s.listOwnedAgentInventory,
		s.listOwnedSoulInventory,
		s.listOwnedHeartbeatInventory,
		s.listOwnedJobInventory,
		s.listOwnedTriggerInventory,
		s.listOwnedBridgeInventory,
	}
	for _, collect := range collectors {
		collected, err := collect(ctx, owner)
		if err != nil {
			return nil, err
		}
		items = append(items, collected...)
	}
	slices.SortFunc(items, compareInventoryItems)
	return items, nil
}

func (s *ResourceStore) listOwnedAgentInventory(
	ctx context.Context,
	owner resources.ResourceOwner,
) ([]InventoryItem, error) {
	return listOwnedInventoryForKind(
		ctx,
		s.agents,
		s.actor,
		owner,
		aghconfig.AgentResourceKind,
		"agents",
		func(spec aghconfig.AgentDef) string { return spec.Name },
	)
}

func (s *ResourceStore) listOwnedSoulInventory(
	ctx context.Context,
	owner resources.ResourceOwner,
) ([]InventoryItem, error) {
	return listOwnedInventoryForKind(
		ctx,
		s.souls,
		s.actor,
		owner,
		soul.ResourceKind,
		"soul resources",
		func(spec soul.ResourceSpec) string { return spec.AgentName },
	)
}

func (s *ResourceStore) listOwnedHeartbeatInventory(
	ctx context.Context,
	owner resources.ResourceOwner,
) ([]InventoryItem, error) {
	return listOwnedInventoryForKind(
		ctx,
		s.heartbeats,
		s.actor,
		owner,
		heartbeat.ResourceKind,
		"heartbeat resources",
		func(spec heartbeat.ResourceSpec) string { return spec.AgentName },
	)
}

func (s *ResourceStore) listOwnedJobInventory(
	ctx context.Context,
	owner resources.ResourceOwner,
) ([]InventoryItem, error) {
	return listOwnedInventoryForKind(
		ctx,
		s.jobs,
		s.actor,
		owner,
		automationpkg.JobResourceKind,
		"automation jobs",
		func(spec automationpkg.Job) string { return spec.Name },
	)
}

func (s *ResourceStore) listOwnedTriggerInventory(
	ctx context.Context,
	owner resources.ResourceOwner,
) ([]InventoryItem, error) {
	return listOwnedInventoryForKind(
		ctx,
		s.triggers,
		s.actor,
		owner,
		automationpkg.TriggerResourceKind,
		"automation triggers",
		func(spec automationpkg.Trigger) string { return spec.Name },
	)
}

func (s *ResourceStore) listOwnedBridgeInventory(
	ctx context.Context,
	owner resources.ResourceOwner,
) ([]InventoryItem, error) {
	return listOwnedInventoryForKind(
		ctx,
		s.bridges,
		s.actor,
		owner,
		bridgepkg.BridgeInstanceResourceKind,
		"bridge instances",
		func(spec bridgepkg.BridgeInstanceSpec) string { return spec.DisplayName },
	)
}

func listOwnedInventoryForKind[T any](
	ctx context.Context,
	store resources.Store[T],
	actor resources.MutationActor,
	owner resources.ResourceOwner,
	kind resources.ResourceKind,
	label string,
	resourceName func(T) string,
) ([]InventoryItem, error) {
	records, err := store.List(ctx, actor, resources.ResourceFilter{Kind: kind, Owner: &owner})
	if err != nil {
		return nil, fmt.Errorf("bundles: list owned %s: %w", label, err)
	}
	items := make([]InventoryItem, 0, len(records))
	for _, record := range records {
		items = append(items, InventoryItem{
			ActivationID:  owner.ID,
			ResourceKind:  string(kind),
			ResourceID:    record.ID,
			ResourceName:  resourceName(record.Spec),
			RecordedAtUTC: record.UpdatedAt,
		})
	}
	return items, nil
}

func (s *ResourceStore) ApplyBundleActivationResources(
	ctx context.Context,
	plan BundleActivationResourcePlan,
) error {
	snapshot, err := s.snapshotOwnedBundleActivationResources(ctx)
	if err != nil {
		return err
	}
	changed := make(map[resources.ResourceKind]struct{}, 6)
	if err := s.applyBundleActivationResourcePlan(ctx, plan, changed); err != nil {
		rollbackCtx := context.WithoutCancel(ctx)
		if deadline, ok := ctx.Deadline(); ok {
			var cancel context.CancelFunc
			rollbackCtx, cancel = context.WithDeadline(rollbackCtx, deadline)
			defer cancel()
		}
		if rollbackErr := s.restoreOwnedBundleActivationResources(rollbackCtx, snapshot); rollbackErr != nil {
			return errors.Join(err, fmt.Errorf("bundles: rollback owned activation resources: %w", rollbackErr))
		}
		return err
	}
	return s.triggerChangedKinds(ctx, changed)
}

func (s *ResourceStore) applyBundleActivationResourcePlan(
	ctx context.Context,
	plan BundleActivationResourcePlan,
	changed map[resources.ResourceKind]struct{},
) error {
	if err := s.syncOwnedAgentResources(ctx, plan, changed); err != nil {
		return err
	}
	if err := s.syncOwnedSoulResources(ctx, plan, changed); err != nil {
		return err
	}
	if err := s.syncOwnedHeartbeatResources(ctx, plan, changed); err != nil {
		return err
	}
	if err := s.syncOwnedJobResources(ctx, plan, changed); err != nil {
		return err
	}
	if err := s.syncOwnedTriggerResources(ctx, plan, changed); err != nil {
		return err
	}
	return s.syncOwnedBridgeResources(ctx, plan, changed)
}

type bundleActivationOwnedSnapshot struct {
	agents     map[string]resources.Record[aghconfig.AgentDef]
	souls      map[string]resources.Record[soul.ResourceSpec]
	heartbeats map[string]resources.Record[heartbeat.ResourceSpec]
	jobs       map[string]resources.Record[automationpkg.Job]
	triggers   map[string]resources.Record[automationpkg.Trigger]
	bridges    map[string]resources.Record[bridgepkg.BridgeInstanceSpec]
}

func (s *ResourceStore) snapshotOwnedBundleActivationResources(
	ctx context.Context,
) (bundleActivationOwnedSnapshot, error) {
	agents, err := listBundleActivationOwnedRecords(
		ctx,
		s.agents,
		s.actor,
		aghconfig.AgentResourceKind,
	)
	if err != nil {
		return bundleActivationOwnedSnapshot{}, err
	}
	soulsByID, err := listBundleActivationOwnedRecords(
		ctx,
		s.souls,
		s.actor,
		soul.ResourceKind,
	)
	if err != nil {
		return bundleActivationOwnedSnapshot{}, err
	}
	heartbeatsByID, err := listBundleActivationOwnedRecords(
		ctx,
		s.heartbeats,
		s.actor,
		heartbeat.ResourceKind,
	)
	if err != nil {
		return bundleActivationOwnedSnapshot{}, err
	}
	jobsByID, err := listBundleActivationOwnedRecords(
		ctx,
		s.jobs,
		s.actor,
		automationpkg.JobResourceKind,
	)
	if err != nil {
		return bundleActivationOwnedSnapshot{}, err
	}
	triggersByID, err := listBundleActivationOwnedRecords(
		ctx,
		s.triggers,
		s.actor,
		automationpkg.TriggerResourceKind,
	)
	if err != nil {
		return bundleActivationOwnedSnapshot{}, err
	}
	bridgesByID, err := listBundleActivationOwnedRecords(
		ctx,
		s.bridges,
		s.actor,
		bridgepkg.BridgeInstanceResourceKind,
	)
	if err != nil {
		return bundleActivationOwnedSnapshot{}, err
	}
	return bundleActivationOwnedSnapshot{
		agents:     agents,
		souls:      soulsByID,
		heartbeats: heartbeatsByID,
		jobs:       jobsByID,
		triggers:   triggersByID,
		bridges:    bridgesByID,
	}, nil
}

func listBundleActivationOwnedRecords[T any](
	ctx context.Context,
	store resources.Store[T],
	actor resources.MutationActor,
	kind resources.ResourceKind,
) (map[string]resources.Record[T], error) {
	records, err := store.List(ctx, actor, resources.ResourceFilter{Kind: kind})
	if err != nil {
		return nil, fmt.Errorf("bundles: list owned %s resources: %w", kind, err)
	}
	owned := make(map[string]resources.Record[T], len(records))
	for _, record := range records {
		if record.Owner.Kind != BundleActivationOwnerKind {
			continue
		}
		owned[record.ID] = record
	}
	return owned, nil
}

func (s *ResourceStore) restoreOwnedBundleActivationResources(
	ctx context.Context,
	snapshot bundleActivationOwnedSnapshot,
) error {
	var errs []error
	if err := restoreOwnedBundleActivationRecords(
		ctx,
		s.agents,
		s.actor,
		aghconfig.AgentResourceKind,
		snapshot.agents,
		s.sameAgent,
	); err != nil {
		errs = append(errs, err)
	}
	if err := restoreOwnedBundleActivationRecords(
		ctx,
		s.souls,
		s.actor,
		soul.ResourceKind,
		snapshot.souls,
		s.sameSoul,
	); err != nil {
		errs = append(errs, err)
	}
	if err := restoreOwnedBundleActivationRecords(
		ctx,
		s.heartbeats,
		s.actor,
		heartbeat.ResourceKind,
		snapshot.heartbeats,
		s.sameHeartbeat,
	); err != nil {
		errs = append(errs, err)
	}
	if err := restoreOwnedBundleActivationRecords(
		ctx,
		s.jobs,
		s.actor,
		automationpkg.JobResourceKind,
		snapshot.jobs,
		s.sameJob,
	); err != nil {
		errs = append(errs, err)
	}
	if err := restoreOwnedBundleActivationRecords(
		ctx,
		s.triggers,
		s.actor,
		automationpkg.TriggerResourceKind,
		snapshot.triggers,
		s.sameTrigger,
	); err != nil {
		errs = append(errs, err)
	}
	if err := restoreOwnedBundleActivationRecords(
		ctx,
		s.bridges,
		s.actor,
		bridgepkg.BridgeInstanceResourceKind,
		snapshot.bridges,
		s.sameBridge,
	); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func restoreOwnedBundleActivationRecords[T any](
	ctx context.Context,
	store resources.Store[T],
	actor resources.MutationActor,
	kind resources.ResourceKind,
	snapshot map[string]resources.Record[T],
	same func(resources.Record[T], T) bool,
) error {
	current, err := listBundleActivationOwnedRecords(ctx, store, actor, kind)
	if err != nil {
		return err
	}
	for id, record := range current {
		if _, ok := snapshot[id]; ok {
			continue
		}
		if err := store.Delete(
			ctx,
			activationResourceActor(actor, record.Owner.ID),
			record.ID,
			record.Version,
		); err != nil {
			return fmt.Errorf("bundles: restore owned %s %q delete: %w", kind, record.ID, err)
		}
	}
	for id, record := range snapshot {
		existing, ok := current[id]
		if ok && existing.Owner == record.Owner && existing.Scope == record.Scope && same(existing, record.Spec) {
			continue
		}
		expectedVersion := int64(0)
		if ok {
			expectedVersion = existing.Version
		}
		if _, err := store.Put(ctx, activationResourceActor(actor, record.Owner.ID), resources.Draft[T]{
			ID:              id,
			Scope:           record.Scope,
			ExpectedVersion: expectedVersion,
			Spec:            record.Spec,
		}); err != nil {
			return fmt.Errorf("bundles: restore owned %s %q upsert: %w", kind, record.ID, err)
		}
	}
	return nil
}

func (s *ResourceStore) syncOwnedAgentResources(
	ctx context.Context,
	plan BundleActivationResourcePlan,
	changed map[resources.ResourceKind]struct{},
) error {
	desired := make(map[string]map[string]ownedAgentResource)
	for _, agent := range plan.desiredAgents {
		ownerID := strings.TrimSpace(plan.agentOwners[strings.TrimSpace(agent.ID)])
		if ownerID == "" {
			return fmt.Errorf("bundles: owned agent %q has no activation owner", agent.ID)
		}
		if desired[ownerID] == nil {
			desired[ownerID] = make(map[string]ownedAgentResource)
		}
		desired[ownerID][agent.ID] = agent
	}
	return syncOwnedResources(
		aghconfig.AgentResourceKind,
		plan.activeActivationIDs,
		changed,
		func(owner resources.ResourceOwner) ([]resources.Record[aghconfig.AgentDef], error) {
			return s.agents.List(ctx, s.actor, resources.ResourceFilter{
				Kind:  aghconfig.AgentResourceKind,
				Owner: &owner,
			})
		},
		func(ownerID string, current map[string]resources.Record[aghconfig.AgentDef]) error {
			return s.upsertOwnedAgents(ctx, ownerID, current, desired[ownerID], changed)
		},
		func(ownerID string) resources.MutationActor { return activationResourceActor(s.actor, ownerID) },
		func(actor resources.MutationActor, stale resources.Record[aghconfig.AgentDef]) error {
			return s.agents.Delete(ctx, actor, stale.ID, stale.Version)
		},
		func() ([]resources.Record[aghconfig.AgentDef], error) {
			return s.agents.List(ctx, s.actor, resources.ResourceFilter{Kind: aghconfig.AgentResourceKind})
		},
	)
}

func (s *ResourceStore) syncOwnedSoulResources(
	ctx context.Context,
	plan BundleActivationResourcePlan,
	changed map[resources.ResourceKind]struct{},
) error {
	desired := make(map[string]map[string]ownedSoulResource)
	for _, spec := range plan.desiredSouls {
		ownerID := strings.TrimSpace(plan.soulOwners[strings.TrimSpace(spec.ID)])
		if ownerID == "" {
			return fmt.Errorf("bundles: owned soul resource %q has no activation owner", spec.ID)
		}
		if desired[ownerID] == nil {
			desired[ownerID] = make(map[string]ownedSoulResource)
		}
		desired[ownerID][spec.ID] = spec
	}
	return syncOwnedResources(
		soul.ResourceKind,
		plan.activeActivationIDs,
		changed,
		func(owner resources.ResourceOwner) ([]resources.Record[soul.ResourceSpec], error) {
			return s.souls.List(ctx, s.actor, resources.ResourceFilter{Kind: soul.ResourceKind, Owner: &owner})
		},
		func(ownerID string, current map[string]resources.Record[soul.ResourceSpec]) error {
			return s.upsertOwnedSouls(ctx, ownerID, current, desired[ownerID], changed)
		},
		func(ownerID string) resources.MutationActor { return activationResourceActor(s.actor, ownerID) },
		func(actor resources.MutationActor, stale resources.Record[soul.ResourceSpec]) error {
			return s.souls.Delete(ctx, actor, stale.ID, stale.Version)
		},
		func() ([]resources.Record[soul.ResourceSpec], error) {
			return s.souls.List(ctx, s.actor, resources.ResourceFilter{Kind: soul.ResourceKind})
		},
	)
}

func (s *ResourceStore) syncOwnedHeartbeatResources(
	ctx context.Context,
	plan BundleActivationResourcePlan,
	changed map[resources.ResourceKind]struct{},
) error {
	desired := make(map[string]map[string]ownedHeartbeatResource)
	for _, spec := range plan.desiredHeartbeats {
		ownerID := strings.TrimSpace(plan.heartbeatOwners[strings.TrimSpace(spec.ID)])
		if ownerID == "" {
			return fmt.Errorf("bundles: owned heartbeat resource %q has no activation owner", spec.ID)
		}
		if desired[ownerID] == nil {
			desired[ownerID] = make(map[string]ownedHeartbeatResource)
		}
		desired[ownerID][spec.ID] = spec
	}
	return syncOwnedResources(
		heartbeat.ResourceKind,
		plan.activeActivationIDs,
		changed,
		func(owner resources.ResourceOwner) ([]resources.Record[heartbeat.ResourceSpec], error) {
			return s.heartbeats.List(
				ctx,
				s.actor,
				resources.ResourceFilter{Kind: heartbeat.ResourceKind, Owner: &owner},
			)
		},
		func(ownerID string, current map[string]resources.Record[heartbeat.ResourceSpec]) error {
			return s.upsertOwnedHeartbeats(ctx, ownerID, current, desired[ownerID], changed)
		},
		func(ownerID string) resources.MutationActor { return activationResourceActor(s.actor, ownerID) },
		func(actor resources.MutationActor, stale resources.Record[heartbeat.ResourceSpec]) error {
			return s.heartbeats.Delete(ctx, actor, stale.ID, stale.Version)
		},
		func() ([]resources.Record[heartbeat.ResourceSpec], error) {
			return s.heartbeats.List(ctx, s.actor, resources.ResourceFilter{Kind: heartbeat.ResourceKind})
		},
	)
}

func (s *ResourceStore) syncOwnedJobResources(
	ctx context.Context,
	plan BundleActivationResourcePlan,
	changed map[resources.ResourceKind]struct{},
) error {
	desired := make(map[string]map[string]automationpkg.Job)
	for _, job := range plan.desiredJobs {
		ownerID := strings.TrimSpace(plan.jobOwners[strings.TrimSpace(job.ID)])
		if ownerID == "" {
			return fmt.Errorf("bundles: owned automation job %q has no activation owner", job.ID)
		}
		if desired[ownerID] == nil {
			desired[ownerID] = make(map[string]automationpkg.Job)
		}
		desired[ownerID][job.ID] = job
	}
	return syncOwnedResources(
		automationpkg.JobResourceKind,
		plan.activeActivationIDs,
		changed,
		func(owner resources.ResourceOwner) ([]resources.Record[automationpkg.Job], error) {
			return s.jobs.List(
				ctx,
				s.actor,
				resources.ResourceFilter{Kind: automationpkg.JobResourceKind, Owner: &owner},
			)
		},
		func(ownerID string, current map[string]resources.Record[automationpkg.Job]) error {
			return s.upsertOwnedJobs(ctx, ownerID, current, desired[ownerID], changed)
		},
		func(ownerID string) resources.MutationActor { return activationResourceActor(s.actor, ownerID) },
		func(actor resources.MutationActor, stale resources.Record[automationpkg.Job]) error {
			return s.jobs.Delete(ctx, actor, stale.ID, stale.Version)
		},
		func() ([]resources.Record[automationpkg.Job], error) {
			return s.jobs.List(ctx, s.actor, resources.ResourceFilter{Kind: automationpkg.JobResourceKind})
		},
	)
}

func (s *ResourceStore) syncOwnedTriggerResources(
	ctx context.Context,
	plan BundleActivationResourcePlan,
	changed map[resources.ResourceKind]struct{},
) error {
	desired := make(map[string]map[string]automationpkg.Trigger)
	for _, trigger := range plan.desiredTriggers {
		ownerID := strings.TrimSpace(plan.triggerOwners[strings.TrimSpace(trigger.ID)])
		if ownerID == "" {
			return fmt.Errorf("bundles: owned automation trigger %q has no activation owner", trigger.ID)
		}
		if desired[ownerID] == nil {
			desired[ownerID] = make(map[string]automationpkg.Trigger)
		}
		desired[ownerID][trigger.ID] = trigger
	}
	return syncOwnedResources(
		automationpkg.TriggerResourceKind,
		plan.activeActivationIDs,
		changed,
		func(owner resources.ResourceOwner) ([]resources.Record[automationpkg.Trigger], error) {
			return s.triggers.List(ctx, s.actor, resources.ResourceFilter{
				Kind:  automationpkg.TriggerResourceKind,
				Owner: &owner,
			})
		},
		func(ownerID string, current map[string]resources.Record[automationpkg.Trigger]) error {
			return s.upsertOwnedTriggers(ctx, ownerID, current, desired[ownerID], changed)
		},
		func(ownerID string) resources.MutationActor { return activationResourceActor(s.actor, ownerID) },
		func(actor resources.MutationActor, stale resources.Record[automationpkg.Trigger]) error {
			return s.triggers.Delete(ctx, actor, stale.ID, stale.Version)
		},
		func() ([]resources.Record[automationpkg.Trigger], error) {
			return s.triggers.List(ctx, s.actor, resources.ResourceFilter{Kind: automationpkg.TriggerResourceKind})
		},
	)
}

func (s *ResourceStore) syncOwnedBridgeResources(
	ctx context.Context,
	plan BundleActivationResourcePlan,
	changed map[resources.ResourceKind]struct{},
) error {
	desired := make(map[string]map[string]bridgepkg.BridgeInstanceSpec)
	for _, instance := range plan.desiredBridges {
		ownerID := strings.TrimSpace(plan.bridgeOwners[strings.TrimSpace(instance.ID)])
		if ownerID == "" {
			return fmt.Errorf("bundles: owned bridge instance %q has no activation owner", instance.ID)
		}
		if desired[ownerID] == nil {
			desired[ownerID] = make(map[string]bridgepkg.BridgeInstanceSpec)
		}
		desired[ownerID][instance.ID] = bridgepkg.BridgeInstanceSpecFromInstance(instance)
	}
	return syncOwnedResources(
		bridgepkg.BridgeInstanceResourceKind,
		plan.activeActivationIDs,
		changed,
		func(owner resources.ResourceOwner) ([]resources.Record[bridgepkg.BridgeInstanceSpec], error) {
			return s.bridges.List(ctx, s.actor, resources.ResourceFilter{
				Kind:  bridgepkg.BridgeInstanceResourceKind,
				Owner: &owner,
			})
		},
		func(ownerID string, current map[string]resources.Record[bridgepkg.BridgeInstanceSpec]) error {
			return s.upsertOwnedBridges(ctx, ownerID, current, desired[ownerID], changed)
		},
		func(ownerID string) resources.MutationActor { return activationResourceActor(s.actor, ownerID) },
		func(actor resources.MutationActor, stale resources.Record[bridgepkg.BridgeInstanceSpec]) error {
			return s.bridges.Delete(ctx, actor, stale.ID, stale.Version)
		},
		func() ([]resources.Record[bridgepkg.BridgeInstanceSpec], error) {
			return s.bridges.List(ctx, s.actor, resources.ResourceFilter{Kind: bridgepkg.BridgeInstanceResourceKind})
		},
	)
}

func (s *ResourceStore) upsertOwnedJobs(
	ctx context.Context,
	ownerID string,
	current map[string]resources.Record[automationpkg.Job],
	desired map[string]automationpkg.Job,
	changed map[resources.ResourceKind]struct{},
) error {
	actor := activationResourceActor(s.actor, ownerID)
	for id, job := range desired {
		existing, ok := current[id]
		if ok && existing.Scope == automationpkg.ResourceScopeForAutomation(job.Scope, job.WorkspaceID) &&
			s.sameJob(existing, job) {
			delete(current, id)
			continue
		}
		expectedVersion := int64(0)
		if ok {
			expectedVersion = existing.Version
		}
		if _, err := s.jobs.Put(ctx, actor, resources.Draft[automationpkg.Job]{
			ID:              id,
			Scope:           automationpkg.ResourceScopeForAutomation(job.Scope, job.WorkspaceID),
			ExpectedVersion: expectedVersion,
			Spec:            job,
		}); err != nil {
			return fmt.Errorf("bundles: upsert owned automation job %q: %w", id, err)
		}
		changed[automationpkg.JobResourceKind] = struct{}{}
		delete(current, id)
	}
	return deleteStaleOwnedRecords(ctx, actor, current, changed, automationpkg.JobResourceKind, s.jobs.Delete)
}

func (s *ResourceStore) upsertOwnedAgents(
	ctx context.Context,
	ownerID string,
	current map[string]resources.Record[aghconfig.AgentDef],
	desired map[string]ownedAgentResource,
	changed map[resources.ResourceKind]struct{},
) error {
	actor := activationResourceActor(s.actor, ownerID)
	for id, desiredAgent := range desired {
		spec := desiredAgent.Spec
		scope := desiredAgent.Scope.Normalize()
		existing, ok := current[id]
		if ok && existing.Scope == scope && s.sameAgent(existing, spec) {
			delete(current, id)
			continue
		}
		expectedVersion := int64(0)
		if ok {
			expectedVersion = existing.Version
		}
		if _, err := s.agents.Put(ctx, actor, resources.Draft[aghconfig.AgentDef]{
			ID:              id,
			Scope:           scope,
			ExpectedVersion: expectedVersion,
			Spec:            spec,
		}); err != nil {
			return fmt.Errorf("bundles: upsert owned agent %q: %w", id, err)
		}
		changed[aghconfig.AgentResourceKind] = struct{}{}
		delete(current, id)
	}
	return deleteStaleOwnedRecords(ctx, actor, current, changed, aghconfig.AgentResourceKind, s.agents.Delete)
}

func (s *ResourceStore) upsertOwnedSouls(
	ctx context.Context,
	ownerID string,
	current map[string]resources.Record[soul.ResourceSpec],
	desired map[string]ownedSoulResource,
	changed map[resources.ResourceKind]struct{},
) error {
	actor := activationResourceActor(s.actor, ownerID)
	for id, desiredSoul := range desired {
		spec := desiredSoul.Spec
		scope := desiredSoul.Scope.Normalize()
		existing, ok := current[id]
		if ok && existing.Scope == scope && s.sameSoul(existing, spec) {
			delete(current, id)
			continue
		}
		expectedVersion := int64(0)
		if ok {
			expectedVersion = existing.Version
		}
		if _, err := s.souls.Put(ctx, actor, resources.Draft[soul.ResourceSpec]{
			ID:              id,
			Scope:           scope,
			ExpectedVersion: expectedVersion,
			Spec:            spec,
		}); err != nil {
			return fmt.Errorf("bundles: upsert owned soul resource %q: %w", id, err)
		}
		changed[soul.ResourceKind] = struct{}{}
		delete(current, id)
	}
	return deleteStaleOwnedRecords(ctx, actor, current, changed, soul.ResourceKind, s.souls.Delete)
}

func (s *ResourceStore) upsertOwnedHeartbeats(
	ctx context.Context,
	ownerID string,
	current map[string]resources.Record[heartbeat.ResourceSpec],
	desired map[string]ownedHeartbeatResource,
	changed map[resources.ResourceKind]struct{},
) error {
	actor := activationResourceActor(s.actor, ownerID)
	for id, desiredHeartbeat := range desired {
		spec := desiredHeartbeat.Spec
		scope := desiredHeartbeat.Scope.Normalize()
		existing, ok := current[id]
		if ok && existing.Scope == scope && s.sameHeartbeat(existing, spec) {
			delete(current, id)
			continue
		}
		expectedVersion := int64(0)
		if ok {
			expectedVersion = existing.Version
		}
		if _, err := s.heartbeats.Put(ctx, actor, resources.Draft[heartbeat.ResourceSpec]{
			ID:              id,
			Scope:           scope,
			ExpectedVersion: expectedVersion,
			Spec:            spec,
		}); err != nil {
			return fmt.Errorf("bundles: upsert owned heartbeat resource %q: %w", id, err)
		}
		changed[heartbeat.ResourceKind] = struct{}{}
		delete(current, id)
	}
	return deleteStaleOwnedRecords(ctx, actor, current, changed, heartbeat.ResourceKind, s.heartbeats.Delete)
}

func (s *ResourceStore) upsertOwnedTriggers(
	ctx context.Context,
	ownerID string,
	current map[string]resources.Record[automationpkg.Trigger],
	desired map[string]automationpkg.Trigger,
	changed map[resources.ResourceKind]struct{},
) error {
	actor := activationResourceActor(s.actor, ownerID)
	for id, trigger := range desired {
		existing, ok := current[id]
		if ok && existing.Scope == automationpkg.ResourceScopeForAutomation(trigger.Scope, trigger.WorkspaceID) &&
			s.sameTrigger(existing, trigger) {
			delete(current, id)
			continue
		}
		expectedVersion := int64(0)
		if ok {
			expectedVersion = existing.Version
		}
		if _, err := s.triggers.Put(ctx, actor, resources.Draft[automationpkg.Trigger]{
			ID:              id,
			Scope:           automationpkg.ResourceScopeForAutomation(trigger.Scope, trigger.WorkspaceID),
			ExpectedVersion: expectedVersion,
			Spec:            trigger,
		}); err != nil {
			return fmt.Errorf("bundles: upsert owned automation trigger %q: %w", id, err)
		}
		changed[automationpkg.TriggerResourceKind] = struct{}{}
		delete(current, id)
	}
	return deleteStaleOwnedRecords(ctx, actor, current, changed, automationpkg.TriggerResourceKind, s.triggers.Delete)
}

func (s *ResourceStore) upsertOwnedBridges(
	ctx context.Context,
	ownerID string,
	current map[string]resources.Record[bridgepkg.BridgeInstanceSpec],
	desired map[string]bridgepkg.BridgeInstanceSpec,
	changed map[resources.ResourceKind]struct{},
) error {
	actor := activationResourceActor(s.actor, ownerID)
	for id, spec := range desired {
		existing, ok := current[id]
		if ok && existing.Scope == bridgepkg.ResourceScopeForBridge(spec.Scope, spec.WorkspaceID) &&
			s.sameBridge(existing, spec) {
			delete(current, id)
			continue
		}
		expectedVersion := int64(0)
		if ok {
			expectedVersion = existing.Version
		}
		if _, err := s.bridges.Put(ctx, actor, resources.Draft[bridgepkg.BridgeInstanceSpec]{
			ID:              id,
			Scope:           bridgepkg.ResourceScopeForBridge(spec.Scope, spec.WorkspaceID),
			ExpectedVersion: expectedVersion,
			Spec:            spec,
		}); err != nil {
			return fmt.Errorf("bundles: upsert owned bridge instance %q: %w", id, err)
		}
		changed[bridgepkg.BridgeInstanceResourceKind] = struct{}{}
		delete(current, id)
	}
	return deleteStaleOwnedRecords(ctx, actor, current, changed, bridgepkg.BridgeInstanceResourceKind, s.bridges.Delete)
}

func syncOwnedResources[T any](
	kind resources.ResourceKind,
	active map[string]struct{},
	changed map[resources.ResourceKind]struct{},
	listByOwner func(resources.ResourceOwner) ([]resources.Record[T], error),
	upsert func(string, map[string]resources.Record[T]) error,
	actorForOwner func(string) resources.MutationActor,
	deleteOne func(resources.MutationActor, resources.Record[T]) error,
	listAll func() ([]resources.Record[T], error),
) error {
	if !bundleActivationOwnedKindAllowed(kind) {
		return fmt.Errorf("bundles: bundle activation cannot own resource kind %q", kind)
	}
	for ownerID := range active {
		owner := ownerForActivation(ownerID)
		records, err := listByOwner(owner)
		if err != nil {
			return err
		}
		current := make(map[string]resources.Record[T], len(records))
		for _, record := range records {
			current[record.ID] = record
		}
		if err := upsert(ownerID, current); err != nil {
			return err
		}
	}

	records, err := listAll()
	if err != nil {
		return err
	}
	for _, record := range records {
		if record.Owner.Kind != BundleActivationOwnerKind {
			continue
		}
		ownerID := strings.TrimSpace(record.Owner.ID)
		if _, ok := active[ownerID]; ok {
			continue
		}
		if err := deleteOne(actorForOwner(ownerID), record); err != nil {
			return fmt.Errorf("bundles: delete stale owned %s %q: %w", kind, record.ID, err)
		}
		changed[kind] = struct{}{}
	}
	return nil
}

func deleteStaleOwnedRecords[T any](
	ctx context.Context,
	actor resources.MutationActor,
	current map[string]resources.Record[T],
	changed map[resources.ResourceKind]struct{},
	kind resources.ResourceKind,
	deleteFunc func(context.Context, resources.MutationActor, string, int64) error,
) error {
	for _, stale := range current {
		if err := deleteFunc(ctx, actor, stale.ID, stale.Version); err != nil {
			return fmt.Errorf("bundles: delete stale owned %s %q: %w", kind, stale.ID, err)
		}
		changed[kind] = struct{}{}
	}
	return nil
}

func (s *ResourceStore) sameJob(record resources.Record[automationpkg.Job], desired automationpkg.Job) bool {
	return sameEncodedSpec(s.jobCodec, record.Scope, record.Spec, desired)
}

func (s *ResourceStore) sameAgent(
	record resources.Record[aghconfig.AgentDef],
	desired aghconfig.AgentDef,
) bool {
	return sameEncodedSpec(s.agentCodec, record.Scope, record.Spec, desired)
}

func (s *ResourceStore) sameSoul(
	record resources.Record[soul.ResourceSpec],
	desired soul.ResourceSpec,
) bool {
	return sameEncodedSpec(s.soulCodec, record.Scope, record.Spec, desired)
}

func (s *ResourceStore) sameHeartbeat(
	record resources.Record[heartbeat.ResourceSpec],
	desired heartbeat.ResourceSpec,
) bool {
	return sameEncodedSpec(s.heartbeatCodec, record.Scope, record.Spec, desired)
}

func (s *ResourceStore) sameTrigger(
	record resources.Record[automationpkg.Trigger],
	desired automationpkg.Trigger,
) bool {
	return sameEncodedSpec(s.triggerCodec, record.Scope, record.Spec, desired)
}

func (s *ResourceStore) sameBridge(
	record resources.Record[bridgepkg.BridgeInstanceSpec],
	desired bridgepkg.BridgeInstanceSpec,
) bool {
	return sameEncodedSpec(s.bridgeCodec, record.Scope, record.Spec, desired)
}

func sameEncodedSpec[T any](
	codec resources.KindCodec[T],
	scope resources.ResourceScope,
	current T,
	desired T,
) bool {
	currentJSON, err := codec.Encode(current)
	if err != nil {
		return false
	}
	desiredJSON, err := codec.Encode(desired)
	if err != nil {
		return false
	}
	return bytes.Equal(currentJSON, desiredJSON) && scope == scope.Normalize()
}

func (s *ResourceStore) triggerChangedKinds(
	ctx context.Context,
	changed map[resources.ResourceKind]struct{},
) error {
	if s.trigger == nil || len(changed) == 0 {
		return nil
	}
	kinds := make([]resources.ResourceKind, 0, len(changed))
	for kind := range changed {
		kinds = append(kinds, kind)
	}
	slices.Sort(kinds)
	var errs []error
	for _, kind := range kinds {
		if err := s.trigger(ctx, kind, resources.ReconcileReasonWrite); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func mapActivationResourceError(action string, id string, err error) error {
	if errors.Is(err, resources.ErrNotFound) {
		return ErrActivationNotFound
	}
	return fmt.Errorf("bundles: %s activation resource %q: %w", action, strings.TrimSpace(id), err)
}

func compareActivations(left Activation, right Activation) int {
	if cmp := strings.Compare(left.ExtensionName, right.ExtensionName); cmp != 0 {
		return cmp
	}
	if cmp := strings.Compare(left.BundleName, right.BundleName); cmp != 0 {
		return cmp
	}
	if cmp := strings.Compare(left.ProfileName, right.ProfileName); cmp != 0 {
		return cmp
	}
	if cmp := strings.Compare(string(left.Scope), string(right.Scope)); cmp != 0 {
		return cmp
	}
	if cmp := strings.Compare(left.WorkspaceID, right.WorkspaceID); cmp != 0 {
		return cmp
	}
	return strings.Compare(left.ID, right.ID)
}

func compareInventoryItems(left InventoryItem, right InventoryItem) int {
	if cmp := strings.Compare(left.ResourceKind, right.ResourceKind); cmp != 0 {
		return cmp
	}
	if cmp := strings.Compare(left.ResourceName, right.ResourceName); cmp != 0 {
		return cmp
	}
	return strings.Compare(left.ResourceID, right.ResourceID)
}
