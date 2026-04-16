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
	"github.com/pedronauck/agh/internal/resources"
)

// ResourceStore persists bundles, activations, and owned activation fan-out through canonical resources.
type ResourceStore struct {
	bundles         resources.Store[BundleResourceSpec]
	bundleCodec     resources.KindCodec[BundleResourceSpec]
	activations     resources.Store[ActivationResourceSpec]
	activationCodec resources.KindCodec[ActivationResourceSpec]
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
	if err := activation.Validate(); err != nil {
		return err
	}
	if activation.CreatedAt.IsZero() {
		activation.CreatedAt = s.now().UTC()
	}
	if activation.UpdatedAt.IsZero() {
		activation.UpdatedAt = activation.CreatedAt
	}
	_, err := s.activations.Put(ctx, s.actor, resources.Draft[ActivationResourceSpec]{
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
	if err := activation.Validate(); err != nil {
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

func (s *ResourceStore) ListBundleActivationInventory(
	ctx context.Context,
	activationID string,
) ([]InventoryItem, error) {
	owner := ownerForActivation(activationID)
	if err := owner.Validate("owner"); err != nil {
		return nil, err
	}
	items := make([]InventoryItem, 0)
	jobRecords, err := s.jobs.List(ctx, s.actor, resources.ResourceFilter{
		Kind:  automationpkg.JobResourceKind,
		Owner: &owner,
	})
	if err != nil {
		return nil, fmt.Errorf("bundles: list owned automation jobs: %w", err)
	}
	for _, record := range jobRecords {
		items = append(items, InventoryItem{
			ActivationID:  owner.ID,
			ResourceKind:  string(automationpkg.JobResourceKind),
			ResourceID:    record.ID,
			ResourceName:  record.Spec.Name,
			RecordedAtUTC: record.UpdatedAt,
		})
	}
	triggerRecords, err := s.triggers.List(ctx, s.actor, resources.ResourceFilter{
		Kind:  automationpkg.TriggerResourceKind,
		Owner: &owner,
	})
	if err != nil {
		return nil, fmt.Errorf("bundles: list owned automation triggers: %w", err)
	}
	for _, record := range triggerRecords {
		items = append(items, InventoryItem{
			ActivationID:  owner.ID,
			ResourceKind:  string(automationpkg.TriggerResourceKind),
			ResourceID:    record.ID,
			ResourceName:  record.Spec.Name,
			RecordedAtUTC: record.UpdatedAt,
		})
	}
	bridgeRecords, err := s.bridges.List(ctx, s.actor, resources.ResourceFilter{
		Kind:  bridgepkg.BridgeInstanceResourceKind,
		Owner: &owner,
	})
	if err != nil {
		return nil, fmt.Errorf("bundles: list owned bridge instances: %w", err)
	}
	for _, record := range bridgeRecords {
		items = append(items, InventoryItem{
			ActivationID:  owner.ID,
			ResourceKind:  string(bridgepkg.BridgeInstanceResourceKind),
			ResourceID:    record.ID,
			ResourceName:  record.Spec.DisplayName,
			RecordedAtUTC: record.UpdatedAt,
		})
	}
	slices.SortFunc(items, compareInventoryItems)
	return items, nil
}

func (s *ResourceStore) ApplyBundleActivationResources(
	ctx context.Context,
	plan BundleActivationResourcePlan,
) error {
	changed := make(map[resources.ResourceKind]struct{}, 3)
	if err := s.syncOwnedJobResources(ctx, plan, changed); err != nil {
		return err
	}
	if err := s.syncOwnedTriggerResources(ctx, plan, changed); err != nil {
		return err
	}
	if err := s.syncOwnedBridgeResources(ctx, plan, changed); err != nil {
		return err
	}
	return s.triggerChangedKinds(ctx, changed)
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
