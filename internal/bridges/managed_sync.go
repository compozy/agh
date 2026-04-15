package bridges

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
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
	SyncManagedInstances(ctx context.Context, source BridgeInstanceSource, desired []BridgeInstance) (ManagedSyncStats, error)
}

// ManagedSyncStats summarizes one managed bridge reconcile pass.
type ManagedSyncStats struct {
	InstancesSynced  int
	InstancesRemoved int
	SyncedAt         time.Time
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

// SyncManagedInstances reconciles all persisted bridge instances for one source
// so they match the desired set exactly.
func (s *ManagedSyncService) SyncManagedInstances(
	ctx context.Context,
	source BridgeInstanceSource,
	desired []BridgeInstance,
) (ManagedSyncStats, error) {
	if s == nil {
		return ManagedSyncStats{}, errors.New("bridges: managed sync service is required")
	}
	if ctx == nil {
		return ManagedSyncStats{}, errors.New("bridges: managed sync context is required")
	}
	normalizedSource := source.Normalize()
	if err := normalizedSource.Validate(); err != nil {
		return ManagedSyncStats{}, err
	}
	if s.store == nil {
		return ManagedSyncStats{}, errors.New("bridges: managed sync store is required")
	}

	existing, err := s.store.ListBridgeInstances(ctx)
	if err != nil {
		return ManagedSyncStats{}, fmt.Errorf("bridges: reconcile list %q instances: %w", normalizedSource, err)
	}

	existingByID := make(map[string]BridgeInstance)
	for _, instance := range existing {
		if instance.Source.Normalize() == normalizedSource {
			existingByID[instance.ID] = instance
		}
	}

	desiredByID := make(map[string]BridgeInstance, len(desired))
	synced := 0
	for _, instance := range desired {
		next := instance
		next.Source = normalizedSource
		if err := next.Validate(); err != nil {
			return ManagedSyncStats{}, fmt.Errorf("bridges: sync managed instance %q: %w", strings.TrimSpace(next.ID), err)
		}
		if _, exists := desiredByID[next.ID]; exists {
			return ManagedSyncStats{}, fmt.Errorf("bridges: duplicate desired managed instance %q", strings.TrimSpace(next.ID))
		}
		desiredByID[next.ID] = next

		current, exists := existingByID[next.ID]
		switch {
		case !exists:
			if err := s.store.InsertBridgeInstance(ctx, next); err != nil {
				return ManagedSyncStats{}, fmt.Errorf("bridges: reconcile insert %q instance %q: %w", normalizedSource, strings.TrimSpace(next.ID), err)
			}
		case !sameManagedInstance(current, next):
			next.CreatedAt = current.CreatedAt
			if next.CreatedAt.IsZero() {
				next.CreatedAt = s.now().UTC()
			}
			if err := s.store.UpdateBridgeInstance(ctx, next); err != nil {
				return ManagedSyncStats{}, fmt.Errorf("bridges: reconcile update %q instance %q: %w", normalizedSource, strings.TrimSpace(next.ID), err)
			}
		}
		synced++
	}

	removed := 0
	for id := range existingByID {
		if _, ok := desiredByID[id]; ok {
			continue
		}
		if err := s.store.DeleteBridgeInstance(ctx, id); err != nil {
			return ManagedSyncStats{}, fmt.Errorf("bridges: reconcile delete %q instance %q: %w", normalizedSource, strings.TrimSpace(id), err)
		}
		removed++
	}

	return ManagedSyncStats{
		InstancesSynced:  synced,
		InstancesRemoved: removed,
		SyncedAt:         s.now().UTC(),
	}, nil
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
		left.RoutingPolicy == right.RoutingPolicy &&
		managedSyncJSONEqual(left.DeliveryDefaults, right.DeliveryDefaults)
}

func managedSyncJSONEqual(left json.RawMessage, right json.RawMessage) bool {
	return strings.TrimSpace(string(left)) == strings.TrimSpace(string(right))
}
