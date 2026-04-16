package bridges

import (
	"bytes"
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
		return "", fmt.Errorf("bridges: validate managed sync source %q: %w", normalizedSource, err)
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
		left.RoutingPolicy == right.RoutingPolicy &&
		managedSyncJSONEqual(left.DeliveryDefaults, right.DeliveryDefaults)
}

func managedSyncJSONEqual(left json.RawMessage, right json.RawMessage) bool {
	leftNormalized, leftErr := normalizeRawJSON(left, "bridge instance delivery defaults")
	rightNormalized, rightErr := normalizeRawJSON(right, "bridge instance delivery defaults")
	if leftErr != nil || rightErr != nil {
		return strings.TrimSpace(string(left)) == strings.TrimSpace(string(right))
	}
	return bytes.Equal(leftNormalized, rightNormalized)
}
