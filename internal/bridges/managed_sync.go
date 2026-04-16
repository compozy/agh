package bridges

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/resources"
)

// ManagedSyncStore is the persistence surface required to reconcile one
// daemon-managed bridge-instance source.
type ManagedSyncStore interface {
	InsertBridgeInstance(ctx context.Context, instance BridgeInstance) error
	UpdateBridgeInstance(ctx context.Context, instance BridgeInstance) error
	DeleteBridgeInstance(ctx context.Context, id string) error
	ListBridgeInstances(ctx context.Context) ([]BridgeInstance, error)
}

// ManagedSyncer reconciles the desired set of bridge instances for one managed
// source such as extension bundles.
type ManagedSyncer interface {
	SyncManagedInstances(
		ctx context.Context,
		source BridgeInstanceSource,
		desired []BridgeInstance,
	) (ManagedSyncStats, error)
}

// ManagedSyncStats summarizes one managed bridge reconcile pass.
type ManagedSyncStats struct {
	InstancesSynced  int
	InstancesRemoved int
	SyncedAt         time.Time
}

// ManagedResourceSyncService reconciles one managed bridge source into canonical resources.
type ManagedResourceSyncService struct {
	store   resources.Store[BridgeInstanceSpec]
	actor   resources.MutationActor
	trigger func(context.Context, resources.ResourceKind, resources.ReconcileReason) error
	now     func() time.Time
}

// ManagedResourceSyncOption customizes ManagedResourceSyncService construction.
type ManagedResourceSyncOption func(*ManagedResourceSyncService)

var _ ManagedSyncer = (*ManagedResourceSyncService)(nil)

// NewManagedResourceSyncer constructs a managed bridge reconciler over canonical bridge.instance resources.
func NewManagedResourceSyncer(
	store resources.Store[BridgeInstanceSpec],
	actor resources.MutationActor,
	trigger func(context.Context, resources.ResourceKind, resources.ReconcileReason) error,
	opts ...ManagedResourceSyncOption,
) *ManagedResourceSyncService {
	service := &ManagedResourceSyncService{
		store:   store,
		actor:   actor,
		trigger: trigger,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(service)
		}
	}
	return service
}

// WithManagedResourceSyncNow overrides the resource sync clock in tests.
func WithManagedResourceSyncNow(now func() time.Time) ManagedResourceSyncOption {
	return func(service *ManagedResourceSyncService) {
		if now != nil {
			service.now = now
		}
	}
}

// ManagedSyncService reconciles one managed bridge source directly against the
// persisted bridge-instance catalog.
type ManagedSyncService struct {
	store ManagedSyncStore
	now   func() time.Time
}

// ManagedSyncOption customizes ManagedSyncService construction.
type ManagedSyncOption func(*ManagedSyncService)

var _ ManagedSyncer = (*ManagedSyncService)(nil)

// NewManagedSyncer constructs a managed bridge reconciler over the supplied store.
func NewManagedSyncer(store ManagedSyncStore, opts ...ManagedSyncOption) *ManagedSyncService {
	service := &ManagedSyncService{
		store: store,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(service)
		}
	}
	return service
}

// WithManagedSyncNow overrides the reconcile clock in tests.
func WithManagedSyncNow(now func() time.Time) ManagedSyncOption {
	return func(service *ManagedSyncService) {
		if now != nil {
			service.now = now
		}
	}
}

// SyncManagedInstances reconciles managed bridge resources for one source exactly.
func (s *ManagedResourceSyncService) SyncManagedInstances(
	ctx context.Context,
	source BridgeInstanceSource,
	desired []BridgeInstance,
) (ManagedSyncStats, error) {
	actor, normalizedSource, err := s.validateResourceSyncInputs(ctx, source)
	if err != nil {
		return ManagedSyncStats{}, err
	}

	existing, err := s.listManagedResourceInstances(ctx, actor)
	if err != nil {
		return ManagedSyncStats{}, err
	}
	desiredByID, synced, err := s.syncDesiredManagedResources(ctx, actor, normalizedSource, existing, desired)
	if err != nil {
		return ManagedSyncStats{}, err
	}
	removed, err := s.deleteStaleManagedResources(ctx, actor, existing, desiredByID)
	if err != nil {
		return ManagedSyncStats{}, err
	}
	if synced > 0 || removed > 0 {
		if err := s.triggerResourceReconcile(ctx); err != nil {
			return ManagedSyncStats{}, err
		}
	}
	return ManagedSyncStats{
		InstancesSynced:  synced,
		InstancesRemoved: removed,
		SyncedAt:         s.now().UTC(),
	}, nil
}

func (s *ManagedResourceSyncService) validateResourceSyncInputs(
	ctx context.Context,
	source BridgeInstanceSource,
) (resources.MutationActor, BridgeInstanceSource, error) {
	if s == nil {
		return resources.MutationActor{}, "", errors.New("bridges: managed resource sync service is required")
	}
	if ctx == nil {
		return resources.MutationActor{}, "", errors.New("bridges: managed resource sync context is required")
	}
	normalizedSource := source.Normalize()
	if err := normalizedSource.Validate(); err != nil {
		return resources.MutationActor{}, "", err
	}
	if s.store == nil {
		return resources.MutationActor{}, "", errors.New("bridges: managed resource sync store is required")
	}
	return bridgeResourceActorForSource(s.actor, normalizedSource), normalizedSource, nil
}

func (s *ManagedResourceSyncService) listManagedResourceInstances(
	ctx context.Context,
	actor resources.MutationActor,
) (map[string]resources.Record[BridgeInstanceSpec], error) {
	source := actor.Source.Normalize()
	records, err := s.store.List(ctx, actor, resources.ResourceFilter{
		Kind:   BridgeInstanceResourceKind,
		Source: &source,
	})
	if err != nil {
		return nil, fmt.Errorf("bridges: reconcile list managed bridge resources: %w", err)
	}
	byID := make(map[string]resources.Record[BridgeInstanceSpec], len(records))
	for _, record := range records {
		byID[record.ID] = record
	}
	return byID, nil
}

func (s *ManagedResourceSyncService) syncDesiredManagedResources(
	ctx context.Context,
	actor resources.MutationActor,
	source BridgeInstanceSource,
	existingByID map[string]resources.Record[BridgeInstanceSpec],
	desired []BridgeInstance,
) (map[string]BridgeInstanceSpec, int, error) {
	desiredByID := make(map[string]BridgeInstanceSpec, len(desired))
	synced := 0
	for _, instance := range desired {
		id, spec, err := s.prepareDesiredManagedResource(source, desiredByID, instance)
		if err != nil {
			return nil, 0, err
		}
		current, exists := existingByID[id]
		if exists && current.Scope == ResourceScopeForBridge(spec.Scope, spec.WorkspaceID) &&
			sameBridgeInstanceSpec(current.Spec, spec) {
			desiredByID[id] = spec
			synced++
			continue
		}

		expectedVersion := int64(0)
		if exists {
			expectedVersion = current.Version
		}
		if _, err := s.store.Put(ctx, actor, resources.Draft[BridgeInstanceSpec]{
			ID:              id,
			Scope:           ResourceScopeForBridge(spec.Scope, spec.WorkspaceID),
			ExpectedVersion: expectedVersion,
			Spec:            spec,
		}); err != nil {
			return nil, 0, fmt.Errorf(
				"bridges: reconcile upsert %q bridge resource %q: %w",
				source,
				strings.TrimSpace(id),
				err,
			)
		}
		desiredByID[id] = spec
		synced++
	}
	return desiredByID, synced, nil
}

func (s *ManagedResourceSyncService) prepareDesiredManagedResource(
	source BridgeInstanceSource,
	desiredByID map[string]BridgeInstanceSpec,
	instance BridgeInstance,
) (string, BridgeInstanceSpec, error) {
	next := instance
	next.Source = source
	next.CreatedAt = time.Time{}
	next.UpdatedAt = time.Time{}
	if strings.TrimSpace(next.ID) == "" {
		return "", BridgeInstanceSpec{}, errors.New("bridges: managed bridge resource id is required")
	}
	if _, exists := desiredByID[next.ID]; exists {
		return "", BridgeInstanceSpec{}, fmt.Errorf(
			"bridges: duplicate desired managed bridge resource %q",
			strings.TrimSpace(next.ID),
		)
	}
	spec := BridgeInstanceSpecFromInstance(next)
	return strings.TrimSpace(next.ID), spec, nil
}

func (s *ManagedResourceSyncService) deleteStaleManagedResources(
	ctx context.Context,
	actor resources.MutationActor,
	existingByID map[string]resources.Record[BridgeInstanceSpec],
	desiredByID map[string]BridgeInstanceSpec,
) (int, error) {
	removed := 0
	for id, stale := range existingByID {
		if _, ok := desiredByID[id]; ok {
			continue
		}
		if err := s.store.Delete(ctx, actor, stale.ID, stale.Version); err != nil {
			return 0, fmt.Errorf("bridges: reconcile delete managed bridge resource %q: %w", id, err)
		}
		removed++
	}
	return removed, nil
}

func (s *ManagedResourceSyncService) triggerResourceReconcile(ctx context.Context) error {
	if s == nil || s.trigger == nil {
		return nil
	}
	return s.trigger(ctx, BridgeInstanceResourceKind, resources.ReconcileReasonWrite)
}

func bridgeResourceActorForSource(
	base resources.MutationActor,
	source BridgeInstanceSource,
) resources.MutationActor {
	actor := base
	if actor.Kind == "" {
		actor.Kind = resources.MutationActorKindDaemon
	}
	actor.ID = "bridge." + strings.TrimSpace(string(source.Normalize()))
	actor.Source = resources.ResourceSource{
		Kind: resources.ResourceSourceKind("daemon"),
		ID:   actor.ID,
	}
	if actor.MaxScope.Kind == "" {
		actor.MaxScope = resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal}
	}
	return actor
}

// SyncManagedInstances reconciles all persisted bridge instances for one source
// so they match the desired set exactly.
func (s *ManagedSyncService) SyncManagedInstances(
	ctx context.Context,
	source BridgeInstanceSource,
	desired []BridgeInstance,
) (ManagedSyncStats, error) {
	normalizedSource, err := s.validateSyncInputs(ctx, source)
	if err != nil {
		return ManagedSyncStats{}, err
	}

	existingByID, err := s.listManagedInstancesByID(ctx, normalizedSource)
	if err != nil {
		return ManagedSyncStats{}, err
	}

	desiredByID, synced, err := s.syncDesiredManagedInstances(
		ctx,
		normalizedSource,
		existingByID,
		desired,
	)
	if err != nil {
		return ManagedSyncStats{}, err
	}

	removed, err := s.deleteStaleManagedInstances(ctx, normalizedSource, existingByID, desiredByID)
	if err != nil {
		return ManagedSyncStats{}, err
	}

	return ManagedSyncStats{
		InstancesSynced:  synced,
		InstancesRemoved: removed,
		SyncedAt:         s.now().UTC(),
	}, nil
}

func (s *ManagedSyncService) validateSyncInputs(
	ctx context.Context,
	source BridgeInstanceSource,
) (BridgeInstanceSource, error) {
	if s == nil {
		return "", errors.New("bridges: managed sync service is required")
	}
	if ctx == nil {
		return "", errors.New("bridges: managed sync context is required")
	}
	normalizedSource := source.Normalize()
	if err := normalizedSource.Validate(); err != nil {
		return "", err
	}
	if s.store == nil {
		return "", errors.New("bridges: managed sync store is required")
	}
	return normalizedSource, nil
}

func (s *ManagedSyncService) listManagedInstancesByID(
	ctx context.Context,
	source BridgeInstanceSource,
) (map[string]BridgeInstance, error) {
	existing, err := s.store.ListBridgeInstances(ctx)
	if err != nil {
		return nil, fmt.Errorf("bridges: reconcile list %q instances: %w", source, err)
	}

	existingByID := make(map[string]BridgeInstance)
	for _, instance := range existing {
		if instance.Source.Normalize() == source {
			existingByID[instance.ID] = instance
		}
	}

	return existingByID, nil
}

func (s *ManagedSyncService) syncDesiredManagedInstances(
	ctx context.Context,
	source BridgeInstanceSource,
	existingByID map[string]BridgeInstance,
	desired []BridgeInstance,
) (map[string]BridgeInstance, int, error) {
	desiredByID := make(map[string]BridgeInstance, len(desired))
	synced := 0
	for _, instance := range desired {
		next, err := s.prepareDesiredManagedInstance(source, desiredByID, instance)
		if err != nil {
			return nil, 0, err
		}
		current, exists := existingByID[next.ID]
		if err := s.upsertManagedInstance(ctx, source, current, exists, next); err != nil {
			return nil, 0, err
		}
		desiredByID[next.ID] = next
		synced++
	}
	return desiredByID, synced, nil
}

func (s *ManagedSyncService) prepareDesiredManagedInstance(
	source BridgeInstanceSource,
	desiredByID map[string]BridgeInstance,
	instance BridgeInstance,
) (BridgeInstance, error) {
	next := instance
	next.Source = source
	if err := next.Validate(); err != nil {
		return BridgeInstance{}, fmt.Errorf(
			"bridges: sync managed instance %q: %w",
			strings.TrimSpace(next.ID),
			err,
		)
	}
	if _, exists := desiredByID[next.ID]; exists {
		return BridgeInstance{}, fmt.Errorf(
			"bridges: duplicate desired managed instance %q",
			strings.TrimSpace(next.ID),
		)
	}
	return next, nil
}

func (s *ManagedSyncService) upsertManagedInstance(
	ctx context.Context,
	source BridgeInstanceSource,
	current BridgeInstance,
	exists bool,
	next BridgeInstance,
) error {
	switch {
	case !exists:
		if err := s.store.InsertBridgeInstance(ctx, next); err != nil {
			return fmt.Errorf(
				"bridges: reconcile insert %q instance %q: %w",
				source,
				strings.TrimSpace(next.ID),
				err,
			)
		}
	case !sameManagedInstance(current, next):
		next.CreatedAt = current.CreatedAt
		if next.CreatedAt.IsZero() {
			next.CreatedAt = s.now().UTC()
		}
		if err := s.store.UpdateBridgeInstance(ctx, next); err != nil {
			return fmt.Errorf(
				"bridges: reconcile update %q instance %q: %w",
				source,
				strings.TrimSpace(next.ID),
				err,
			)
		}
	}
	return nil
}

func (s *ManagedSyncService) deleteStaleManagedInstances(
	ctx context.Context,
	source BridgeInstanceSource,
	existingByID map[string]BridgeInstance,
	desiredByID map[string]BridgeInstance,
) (int, error) {
	removed := 0
	for id := range existingByID {
		if _, ok := desiredByID[id]; ok {
			continue
		}
		if err := s.store.DeleteBridgeInstance(ctx, id); err != nil {
			return 0, fmt.Errorf(
				"bridges: reconcile delete %q instance %q: %w",
				source,
				strings.TrimSpace(id),
				err,
			)
		}
		removed++
	}
	return removed, nil
}

func sameManagedInstance(left BridgeInstance, right BridgeInstance) bool {
	return left.ID == right.ID &&
		left.Scope == right.Scope &&
		left.WorkspaceID == right.WorkspaceID &&
		left.Platform == right.Platform &&
		left.ExtensionName == right.ExtensionName &&
		left.DisplayName == right.DisplayName &&
		left.Source == right.Source &&
		left.Enabled == right.Enabled &&
		left.Status == right.Status &&
		left.DMPolicy == right.DMPolicy &&
		left.RoutingPolicy == right.RoutingPolicy &&
		managedSyncJSONEqual(left.ProviderConfig, right.ProviderConfig) &&
		managedSyncJSONEqual(left.DeliveryDefaults, right.DeliveryDefaults)
}

func sameBridgeInstanceSpec(left BridgeInstanceSpec, right BridgeInstanceSpec) bool {
	left = normalizeBridgeInstanceResourceSpec(left)
	right = normalizeBridgeInstanceResourceSpec(right)
	return left.Scope == right.Scope &&
		left.WorkspaceID == right.WorkspaceID &&
		left.Platform == right.Platform &&
		left.ExtensionName == right.ExtensionName &&
		left.DisplayName == right.DisplayName &&
		left.Source == right.Source &&
		left.Enabled == right.Enabled &&
		left.DMPolicy == right.DMPolicy &&
		left.RoutingPolicy == right.RoutingPolicy &&
		managedSyncJSONEqual(left.ProviderConfig, right.ProviderConfig) &&
		managedSyncJSONEqual(left.DeliveryDefaults, right.DeliveryDefaults) &&
		slicesEqualBridgeSecretSlots(left.SecretSlots, right.SecretSlots) &&
		sameBridgeProviderConfigSchema(left.ConfigSchema, right.ConfigSchema)
}

func slicesEqualBridgeSecretSlots(left []BridgeSecretSlot, right []BridgeSecretSlot) bool {
	left = normalizeBridgeSecretSlotsForResource(left)
	right = normalizeBridgeSecretSlotsForResource(right)
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}

func managedSyncJSONEqual(left json.RawMessage, right json.RawMessage) bool {
	return strings.TrimSpace(string(left)) == strings.TrimSpace(string(right))
}
