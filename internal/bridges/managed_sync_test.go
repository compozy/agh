package bridges_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestManagedSyncerReconcilesCreateUpdateDelete(t *testing.T) {
	t.Parallel()

	store := stubRegistryStore{
		listBridgeInstancesFn: func(_ context.Context) ([]bridgepkg.BridgeInstance, error) {
			return []bridgepkg.BridgeInstance{{
				ID:            "brg-existing",
				Scope:         bridgepkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "telegram-adapter",
				DisplayName:   "Old Name",
				Source:        bridgepkg.BridgeInstanceSourcePackage,
				Enabled:       false,
				Status:        bridgepkg.BridgeStatusDisabled,
				RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
				CreatedAt:     time.Date(2026, 4, 14, 18, 0, 0, 0, time.UTC),
				UpdatedAt:     time.Date(2026, 4, 14, 18, 0, 0, 0, time.UTC),
			}, {
				ID:            "brg-remove",
				Scope:         bridgepkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "telegram-adapter",
				DisplayName:   "Remove",
				Source:        bridgepkg.BridgeInstanceSourcePackage,
				Enabled:       false,
				Status:        bridgepkg.BridgeStatusDisabled,
				RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
			}}, nil
		},
	}

	var (
		inserted []bridgepkg.BridgeInstance
		updated  []bridgepkg.BridgeInstance
		deleted  []string
	)
	store.insertBridgeInstanceFn = func(_ context.Context, instance bridgepkg.BridgeInstance) error {
		inserted = append(inserted, instance)
		return nil
	}
	store.updateBridgeInstanceFn = func(_ context.Context, instance bridgepkg.BridgeInstance) error {
		updated = append(updated, instance)
		return nil
	}
	store.deleteBridgeInstanceFn = func(_ context.Context, id string) error {
		deleted = append(deleted, id)
		return nil
	}

	syncer := bridgepkg.NewManagedSyncer(store, bridgepkg.WithManagedSyncNow(func() time.Time {
		return time.Date(2026, 4, 14, 19, 0, 0, 0, time.UTC)
	}))
	stats, err := syncer.SyncManagedInstances(testutil.Context(t), bridgepkg.BridgeInstanceSourcePackage, []bridgepkg.BridgeInstance{{
		ID:            "brg-existing",
		Scope:         bridgepkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "New Name",
		Enabled:       false,
		Status:        bridgepkg.BridgeStatusDisabled,
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	}, {
		ID:            "brg-new",
		Scope:         bridgepkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "New Bridge",
		Enabled:       false,
		Status:        bridgepkg.BridgeStatusDisabled,
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	}})
	if err != nil {
		t.Fatalf("SyncManagedInstances() error = %v", err)
	}

	if got, want := stats.InstancesSynced, 2; got != want {
		t.Fatalf("InstancesSynced = %d, want %d", got, want)
	}
	if got, want := stats.InstancesRemoved, 1; got != want {
		t.Fatalf("InstancesRemoved = %d, want %d", got, want)
	}
	if got, want := len(inserted), 1; got != want {
		t.Fatalf("len(inserted) = %d, want %d", got, want)
	}
	if got, want := len(updated), 1; got != want {
		t.Fatalf("len(updated) = %d, want %d", got, want)
	}
	if got, want := len(deleted), 1; got != want {
		t.Fatalf("len(deleted) = %d, want %d", got, want)
	}
	if got, want := updated[0].CreatedAt, time.Date(2026, 4, 14, 18, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("updated[0].CreatedAt = %s, want %s", got, want)
	}
}

func TestManagedSyncerWrapsStoreErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		store     stubRegistryStore
		desired   []bridgepkg.BridgeInstance
		wantError string
	}{
		{
			name: "ShouldWrapListFailuresWithSourceContext",
			store: stubRegistryStore{
				listBridgeInstancesFn: func(context.Context) ([]bridgepkg.BridgeInstance, error) {
					return nil, errors.New("list failed")
				},
			},
			wantError: `bridges: reconcile list "package" instances: list failed`,
		},
		{
			name: "ShouldWrapUpdateFailuresWithInstanceContext",
			store: stubRegistryStore{
				listBridgeInstancesFn: func(context.Context) ([]bridgepkg.BridgeInstance, error) {
					return []bridgepkg.BridgeInstance{{
						ID:            "brg-existing",
						Scope:         bridgepkg.ScopeGlobal,
						Platform:      "telegram",
						ExtensionName: "telegram-adapter",
						DisplayName:   "Old Name",
						Source:        bridgepkg.BridgeInstanceSourcePackage,
						Enabled:       false,
						Status:        bridgepkg.BridgeStatusDisabled,
						RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
						CreatedAt:     time.Date(2026, 4, 14, 18, 0, 0, 0, time.UTC),
					}}, nil
				},
				updateBridgeInstanceFn: func(context.Context, bridgepkg.BridgeInstance) error {
					return errors.New("update failed")
				},
			},
			desired: []bridgepkg.BridgeInstance{{
				ID:            "brg-existing",
				Scope:         bridgepkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "telegram-adapter",
				DisplayName:   "New Name",
				Status:        bridgepkg.BridgeStatusDisabled,
				RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
			}},
			wantError: `bridges: reconcile update "package" instance "brg-existing": update failed`,
		},
		{
			name: "ShouldWrapDeleteFailuresWithInstanceContext",
			store: stubRegistryStore{
				listBridgeInstancesFn: func(context.Context) ([]bridgepkg.BridgeInstance, error) {
					return []bridgepkg.BridgeInstance{{
						ID:            "brg-remove",
						Scope:         bridgepkg.ScopeGlobal,
						Platform:      "telegram",
						ExtensionName: "telegram-adapter",
						DisplayName:   "Remove",
						Source:        bridgepkg.BridgeInstanceSourcePackage,
						Enabled:       false,
						Status:        bridgepkg.BridgeStatusDisabled,
						RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
					}}, nil
				},
				deleteBridgeInstanceFn: func(_ context.Context, id string) error {
					if id != "brg-remove" {
						t.Fatalf("DeleteBridgeInstance() id = %q, want brg-remove", id)
					}
					return errors.New("delete failed")
				},
			},
			wantError: `bridges: reconcile delete "package" instance "brg-remove": delete failed`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			syncer := bridgepkg.NewManagedSyncer(tc.store)
			_, err := syncer.SyncManagedInstances(testutil.Context(t), bridgepkg.BridgeInstanceSourcePackage, tc.desired)
			if err == nil || !containsText(err, tc.wantError) {
				t.Fatalf("SyncManagedInstances() error = %v, want substring %q", err, tc.wantError)
			}
		})
	}
}

func TestManagedSyncerRejectsDuplicateDesiredIDs(t *testing.T) {
	t.Parallel()

	syncer := bridgepkg.NewManagedSyncer(stubRegistryStore{})
	_, err := syncer.SyncManagedInstances(testutil.Context(t), bridgepkg.BridgeInstanceSourcePackage, []bridgepkg.BridgeInstance{
		{
			ID:            "brg-dup",
			Scope:         bridgepkg.ScopeGlobal,
			Platform:      "telegram",
			ExtensionName: "telegram-adapter",
			DisplayName:   "First",
			Status:        bridgepkg.BridgeStatusDisabled,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
		},
		{
			ID:            "brg-dup",
			Scope:         bridgepkg.ScopeGlobal,
			Platform:      "telegram",
			ExtensionName: "telegram-adapter",
			DisplayName:   "Second",
			Status:        bridgepkg.BridgeStatusDisabled,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
		},
	})
	if err == nil || !containsText(err, "duplicate desired managed instance") {
		t.Fatalf("SyncManagedInstances() error = %v, want duplicate-id failure", err)
	}
}

func TestManagedSyncerReturnsExplicitErrorForNilStoreConstruction(t *testing.T) {
	t.Parallel()

	syncer := bridgepkg.NewManagedSyncer(nil)
	if syncer == nil {
		t.Fatal("NewManagedSyncer(nil) = nil, want non-nil service")
	}

	_, err := syncer.SyncManagedInstances(testutil.Context(t), bridgepkg.BridgeInstanceSourcePackage, nil)
	if err == nil || !containsText(err, "managed sync store is required") {
		t.Fatalf("SyncManagedInstances(nil store) error = %v, want explicit missing-store failure", err)
	}
}

func containsText(err error, text string) bool {
	return err != nil && text != "" && strings.Contains(err.Error(), text)
}
