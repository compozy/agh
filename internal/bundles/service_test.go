package bundles

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"maps"
	"path/filepath"
	"strings"
	"testing"
	"time"

	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/testutil"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func discardBundleTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type memoryStore struct {
	activations map[string]Activation
	inventory   map[string][]InventoryItem
	bundles     []resources.Record[BundleResourceSpec]
	applied     []BundleActivationResourcePlan
	applyErr    error

	createBundleActivationHook func(Activation) error
	updateBundleActivationHook func(Activation) error
	deleteBundleActivationHook func(string) error
	listBundleActivationsHook  func() ([]Activation, error)
	listBundleInventoryHook    func(string) ([]InventoryItem, error)
	listBundleResourcesHook    func() ([]resources.Record[BundleResourceSpec], error)
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		activations: make(map[string]Activation),
		inventory:   make(map[string][]InventoryItem),
	}
}

func copyStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	next := make(map[string]string, len(values))
	maps.Copy(next, values)
	return next
}

func (s *memoryStore) CreateBundleActivation(_ context.Context, activation Activation) error {
	if s.createBundleActivationHook != nil {
		if err := s.createBundleActivationHook(activation); err != nil {
			return err
		}
	}
	if _, exists := s.activations[activation.ID]; exists {
		return errors.New("duplicate activation")
	}
	s.activations[activation.ID] = activation
	return nil
}

func (s *memoryStore) UpdateBundleActivation(_ context.Context, activation Activation) error {
	if s.updateBundleActivationHook != nil {
		if err := s.updateBundleActivationHook(activation); err != nil {
			return err
		}
	}
	if _, exists := s.activations[activation.ID]; !exists {
		return ErrActivationNotFound
	}
	s.activations[activation.ID] = activation
	return nil
}

func (s *memoryStore) DeleteBundleActivation(_ context.Context, id string) error {
	if s.deleteBundleActivationHook != nil {
		if err := s.deleteBundleActivationHook(id); err != nil {
			return err
		}
	}
	if _, exists := s.activations[id]; !exists {
		return ErrActivationNotFound
	}
	delete(s.activations, id)
	delete(s.inventory, id)
	return nil
}

func (s *memoryStore) GetBundleActivation(_ context.Context, id string) (Activation, error) {
	activation, exists := s.activations[id]
	if !exists {
		return Activation{}, ErrActivationNotFound
	}
	return activation, nil
}

func (s *memoryStore) ListBundleActivations(_ context.Context) ([]Activation, error) {
	if s.listBundleActivationsHook != nil {
		return s.listBundleActivationsHook()
	}
	items := make([]Activation, 0, len(s.activations))
	for _, activation := range s.activations {
		items = append(items, activation)
	}
	return items, nil
}

func (s *memoryStore) ListBundleActivationInventory(_ context.Context, activationID string) ([]InventoryItem, error) {
	if s.listBundleInventoryHook != nil {
		return s.listBundleInventoryHook(activationID)
	}
	return append([]InventoryItem(nil), s.inventory[activationID]...), nil
}

func (s *memoryStore) ListBundleResources(
	_ context.Context,
) ([]resources.Record[BundleResourceSpec], error) {
	if s.listBundleResourcesHook != nil {
		return s.listBundleResourcesHook()
	}
	return append([]resources.Record[BundleResourceSpec](nil), s.bundles...), nil
}

func (s *memoryStore) ApplyBundleActivationResources(
	_ context.Context,
	plan BundleActivationResourcePlan,
) error {
	if s.applyErr != nil {
		return s.applyErr
	}
	next := plan
	next.activeActivationIDs = cloneStringSet(plan.activeActivationIDs)
	next.desiredJobs = cloneJobsForBundle(plan.desiredJobs)
	next.desiredTriggers = cloneTriggersForBundle(plan.desiredTriggers)
	next.desiredBridges = cloneBridgeInstancesForBundle(plan.desiredBridges)
	next.jobOwners = copyStringMap(plan.jobOwners)
	next.triggerOwners = copyStringMap(plan.triggerOwners)
	next.bridgeOwners = copyStringMap(plan.bridgeOwners)
	next.declaredChannels = append([]DeclaredChannel(nil), plan.declaredChannels...)
	s.applied = append(s.applied, next)
	s.inventory = make(map[string][]InventoryItem)
	for _, job := range plan.desiredJobs {
		activationID := strings.TrimSpace(plan.jobOwners[strings.TrimSpace(job.ID)])
		s.inventory[activationID] = append(s.inventory[activationID], InventoryItem{
			ActivationID: activationID,
			ResourceKind: string(automationpkg.JobResourceKind),
			ResourceID:   job.ID,
			ResourceName: job.Name,
		})
	}
	for _, trigger := range plan.desiredTriggers {
		activationID := strings.TrimSpace(plan.triggerOwners[strings.TrimSpace(trigger.ID)])
		s.inventory[activationID] = append(s.inventory[activationID], InventoryItem{
			ActivationID: activationID,
			ResourceKind: string(automationpkg.TriggerResourceKind),
			ResourceID:   trigger.ID,
			ResourceName: trigger.Name,
		})
	}
	for _, instance := range plan.desiredBridges {
		activationID := strings.TrimSpace(plan.bridgeOwners[strings.TrimSpace(instance.ID)])
		s.inventory[activationID] = append(s.inventory[activationID], InventoryItem{
			ActivationID: activationID,
			ResourceKind: string(bridgepkg.BridgeInstanceResourceKind),
			ResourceID:   instance.ID,
			ResourceName: instance.DisplayName,
		})
	}
	return nil
}

type staticExtensionLister struct {
	items []extensionpkg.ExtensionInfo
}

func (l staticExtensionLister) List() ([]extensionpkg.ExtensionInfo, error) {
	return append([]extensionpkg.ExtensionInfo(nil), l.items...), nil
}

type memoryWorkspaceResolver struct {
	resolveFn           func(context.Context, string) (workspacepkg.ResolvedWorkspace, error)
	resolveOrRegisterFn func(context.Context, string) (workspacepkg.ResolvedWorkspace, error)
}

func (r memoryWorkspaceResolver) Resolve(ctx context.Context, idOrPath string) (workspacepkg.ResolvedWorkspace, error) {
	if r.resolveFn != nil {
		return r.resolveFn(ctx, idOrPath)
	}
	return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceNotFound
}

func (r memoryWorkspaceResolver) ResolveOrRegister(
	ctx context.Context,
	path string,
) (workspacepkg.ResolvedWorkspace, error) {
	if r.resolveOrRegisterFn != nil {
		return r.resolveOrRegisterFn(ctx, path)
	}
	return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceNotFound
}

func newMarketingExtension() *extensionpkg.Extension {
	return &extensionpkg.Extension{
		Info: extensionpkg.ExtensionInfo{Name: "marketing-team"},
		Manifest: &extensionpkg.Manifest{
			Name: "marketing-team",
			Bridge: extensionpkg.BridgeConfig{
				Platform:    "telegram",
				DisplayName: "Telegram",
			},
		},
		Bundles: []extensionpkg.BundleSpec{{
			Name:        "marketing",
			Description: "Marketing team bundle",
			Profiles: []extensionpkg.BundleProfile{{
				Name: "default",
				Channels: extensionpkg.BundleChannelsConfig{
					Primary: "marketing",
					Items: []extensionpkg.BundleChannel{{
						Name:        "marketing",
						Description: "Primary marketing channel",
					}},
				},
				Jobs: []extensionpkg.BundleJob{{
					Name:      "daily-sync",
					AgentName: "planner",
					Prompt:    "Summarize campaign status",
					Schedule:  automationpkg.ScheduleSpec{Mode: automationpkg.ScheduleModeEvery, Interval: "1h"},
					Enabled:   true,
					Retry:     automationpkg.DefaultRetryConfig(),
					FireLimit: automationpkg.DefaultFireLimitConfig(),
				}},
				Triggers: []extensionpkg.BundleTrigger{{
					Name:      "session-opened",
					AgentName: "planner",
					Prompt:    "React to session open",
					Event:     "session.created",
					Enabled:   true,
					Retry:     automationpkg.DefaultRetryConfig(),
					FireLimit: automationpkg.DefaultFireLimitConfig(),
				}},
				Bridges: []extensionpkg.BundleBridgePreset{{
					Name:        "telegram-main",
					DisplayName: "Marketing Telegram",
					RoutingPolicy: bridgepkg.RoutingPolicy{
						IncludePeer: true,
					},
					SecretSlots: []extensionpkg.BundleBridgeSecretSlot{{
						Name: "bot_token",
						Kind: "token",
					}},
				}},
			}},
		}},
	}
}

func newMarketingService(store *memoryStore, opts ...Option) *Service {
	ext := newMarketingExtension()
	store.bundles = []resources.Record[BundleResourceSpec]{{
		Kind:  BundleResourceKind,
		ID:    BundleResourceID(ext.Info.Name, ext.Bundles[0].Name),
		Scope: resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
		Spec: BundleResourceSpec{
			ExtensionName:              ext.Info.Name,
			Bundle:                     ext.Bundles[0],
			OwnerBridgePlatform:        ext.Manifest.Bridge.Platform,
			OwnerProvidesBridgeAdapter: true,
		},
	}}
	options := []Option{
		WithConfiguredDefaultChannel("default"),
		WithNow(func() time.Time {
			return time.Date(2026, 4, 14, 22, 0, 0, 0, time.UTC)
		}),
	}
	options = append(options, opts...)
	return NewService(
		store,
		staticExtensionLister{items: []extensionpkg.ExtensionInfo{{Name: "marketing-team"}}},
		func(name string) (*extensionpkg.Extension, error) {
			if name != "marketing-team" {
				return nil, extensionpkg.ErrExtensionNotFound
			}
			return ext, nil
		},
		options...,
	)
}

func TestServiceActivateMaterializesManagedResources(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	service := newMarketingService(store)

	preview, err := service.Activate(testutil.Context(t), ActivateRequest{
		ExtensionName:               "marketing-team",
		BundleName:                  "marketing",
		ProfileName:                 "default",
		Scope:                       ScopeGlobal,
		BindPrimaryChannelAsDefault: true,
	})
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}

	if got, want := len(preview.Inventory), 3; got != want {
		t.Fatalf("len(preview.Inventory) = %d, want %d", got, want)
	}
	if got, want := len(store.applied), 1; got != want {
		t.Fatalf("len(applied plans) = %d, want %d", got, want)
	}
	plan := store.applied[0]
	if got, want := len(plan.desiredJobs), 1; got != want {
		t.Fatalf("len(plan.desiredJobs) = %d, want %d", got, want)
	}
	if got, want := plan.desiredJobs[0].Source, automationpkg.JobSourcePackage; got != want {
		t.Fatalf("plan.desiredJobs[0].Source = %q, want %q", got, want)
	}
	if got, want := len(plan.desiredTriggers), 1; got != want {
		t.Fatalf("len(plan.desiredTriggers) = %d, want %d", got, want)
	}
	if got, want := len(plan.desiredBridges), 1; got != want {
		t.Fatalf("len(plan.desiredBridges) = %d, want %d", got, want)
	}
	if strings.TrimSpace(preview.Activation.SpecContentHash) == "" {
		t.Fatal("preview.Activation.SpecContentHash = empty, want persisted hash")
	}

	settings, err := service.NetworkSettings(testutil.Context(t))
	if err != nil {
		t.Fatalf("NetworkSettings() error = %v", err)
	}
	if got, want := settings.EffectiveDefaultChannel, "marketing"; got != want {
		t.Fatalf("EffectiveDefaultChannel = %q, want %q", got, want)
	}
	if got, want := len(settings.DeclaredChannels), 1; got != want {
		t.Fatalf("len(DeclaredChannels) = %d, want %d", got, want)
	}
	if !settings.DeclaredChannels[0].Primary {
		t.Fatal("DeclaredChannels[0].Primary = false, want true")
	}
}

func TestServiceCatalogPreviewListAndGetUseCanonicalResources(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	service := newMarketingService(store, WithLogger(discardBundleTestLogger()))

	catalog, err := service.Catalog(testutil.Context(t))
	if err != nil {
		t.Fatalf("Catalog() error = %v", err)
	}
	if got, want := len(catalog), 1; got != want {
		t.Fatalf("len(Catalog()) = %d, want %d", got, want)
	}
	preview, err := service.PreviewActivation(testutil.Context(t), ActivateRequest{
		ExtensionName: "marketing-team",
		BundleName:    "marketing",
		ProfileName:   "default",
		Scope:         ScopeGlobal,
	})
	if err != nil {
		t.Fatalf("PreviewActivation() error = %v", err)
	}
	if got, want := len(preview.Inventory), 3; got != want {
		t.Fatalf("len(preview.Inventory) = %d, want %d", got, want)
	}
	activated, err := service.Activate(testutil.Context(t), ActivateRequest{
		ExtensionName: "marketing-team",
		BundleName:    "marketing",
		ProfileName:   "default",
		Scope:         ScopeGlobal,
	})
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	listed, err := service.ListActivations(testutil.Context(t))
	if err != nil {
		t.Fatalf("ListActivations() error = %v", err)
	}
	if got, want := len(listed), 1; got != want {
		t.Fatalf("len(ListActivations()) = %d, want %d", got, want)
	}
	loaded, err := service.GetActivation(testutil.Context(t), activated.Activation.ID)
	if err != nil {
		t.Fatalf("GetActivation() error = %v", err)
	}
	if got, want := len(loaded.Inventory), 3; got != want {
		t.Fatalf("len(GetActivation().Inventory) = %d, want %d", got, want)
	}
}

func TestServiceReadMethodsValidateInputs(t *testing.T) {
	t.Parallel()

	var nilService *Service
	if _, err := nilService.NetworkSettings(testutil.Context(t)); err == nil {
		t.Fatal("nil NetworkSettings() error = nil, want failure")
	}
	service := newMarketingService(newMemoryStore())
	var nilCtx context.Context
	if _, err := service.NetworkSettings(nilCtx); err == nil {
		t.Fatal("NetworkSettings(nil) error = nil, want failure")
	}
	if _, err := service.GetActivation(testutil.Context(t), ""); err == nil {
		t.Fatal("GetActivation(empty) error = nil, want failure")
	}
	if _, err := service.ListActivations(nilCtx); err == nil {
		t.Fatal("ListActivations(nil) error = nil, want failure")
	}
}

func TestServiceListActivationsWrapsErrorsWithContext(t *testing.T) {
	t.Parallel()

	t.Run("ShouldWrapBundleResourceListError", func(t *testing.T) {
		t.Parallel()

		store := newMemoryStore()
		store.activations["activation-1"] = Activation{ID: "activation-1"}
		resourceErr := errors.New("resource store offline")
		store.listBundleResourcesHook = func() ([]resources.Record[BundleResourceSpec], error) {
			return nil, resourceErr
		}
		service := newMarketingService(store, WithLogger(discardBundleTestLogger()))

		_, err := service.ListActivations(testutil.Context(t))
		if err == nil {
			t.Fatal("ListActivations() error = nil, want non-nil")
		}
		if !errors.Is(err, resourceErr) {
			t.Fatalf("ListActivations() error = %v, want wrapped %v", err, resourceErr)
		}
		if !strings.Contains(err.Error(), "list bundle resources for activations") {
			t.Fatalf("ListActivations() error = %v, want wrapped bundle resource context", err)
		}
	})

	t.Run("ShouldWrapActivationInventoryError", func(t *testing.T) {
		t.Parallel()

		store := newMemoryStore()
		service := newMarketingService(store, WithLogger(discardBundleTestLogger()))

		preview, err := service.Activate(testutil.Context(t), ActivateRequest{
			ExtensionName: "marketing-team",
			BundleName:    "marketing",
			ProfileName:   "default",
			Scope:         ScopeGlobal,
		})
		if err != nil {
			t.Fatalf("Activate() error = %v", err)
		}

		inventoryErr := errors.New("inventory unavailable")
		store.listBundleInventoryHook = func(activationID string) ([]InventoryItem, error) {
			if activationID != preview.Activation.ID {
				t.Fatalf("ListBundleActivationInventory() id = %q, want %q", activationID, preview.Activation.ID)
			}
			return nil, inventoryErr
		}

		_, err = service.ListActivations(testutil.Context(t))
		if !errors.Is(err, inventoryErr) {
			t.Fatalf("ListActivations() error = %v, want %v", err, inventoryErr)
		}
		if !strings.Contains(err.Error(), `list activation inventory for "`+preview.Activation.ID+`"`) {
			t.Fatalf("ListActivations() error = %v, want wrapped inventory context", err)
		}
	})
}

func TestServiceListActivationsReturnsEmptyWithoutLoadingBundleResources(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	store.listBundleResourcesHook = func() ([]resources.Record[BundleResourceSpec], error) {
		return nil, errors.New("bundle resources should not be loaded for empty activations")
	}
	service := newMarketingService(store, WithLogger(discardBundleTestLogger()))

	got, err := service.ListActivations(testutil.Context(t))
	if err != nil {
		t.Fatalf("ListActivations() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len(ListActivations()) = %d, want 0", len(got))
	}
	if got == nil {
		t.Fatal("ListActivations() = nil, want empty slice")
	}
}

func TestFindBundleResourceRecordIndexedNormalizesLookupKeys(t *testing.T) {
	t.Parallel()

	record := resources.Record[BundleResourceSpec]{
		ID: BundleResourceID("Marketing-Team", "Launch"),
		Spec: BundleResourceSpec{
			ExtensionName: " Marketing-Team ",
			Bundle: extensionpkg.BundleSpec{
				Name: " Launch ",
			},
		},
	}
	lookup := newBundleRecordLookup([]resources.Record[BundleResourceSpec]{record})
	lookup.records = nil

	got, ok := findBundleResourceRecordIndexed(lookup, "marketing-team", "launch")
	if !ok {
		t.Fatal("findBundleResourceRecordIndexed() ok = false, want true")
	}
	if got.ID != record.ID {
		t.Fatalf("findBundleResourceRecordIndexed().ID = %q, want %q", got.ID, record.ID)
	}
}

func TestServiceRejectsMultipleDefaultChannelClaims(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	ext := &extensionpkg.Extension{
		Info: extensionpkg.ExtensionInfo{Name: "ops-team"},
		Bundles: []extensionpkg.BundleSpec{{
			Name: "ops",
			Profiles: []extensionpkg.BundleProfile{
				{
					Name: "alpha",
					Channels: extensionpkg.BundleChannelsConfig{
						Primary: "ops-alpha",
						Items:   []extensionpkg.BundleChannel{{Name: "ops-alpha"}},
					},
				},
				{
					Name: "beta",
					Channels: extensionpkg.BundleChannelsConfig{
						Primary: "ops-beta",
						Items:   []extensionpkg.BundleChannel{{Name: "ops-beta"}},
					},
				},
			},
		}},
	}
	store.bundles = []resources.Record[BundleResourceSpec]{{
		Kind:  BundleResourceKind,
		ID:    BundleResourceID(ext.Info.Name, ext.Bundles[0].Name),
		Scope: resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
		Spec: BundleResourceSpec{
			ExtensionName: ext.Info.Name,
			Bundle:        ext.Bundles[0],
		},
	}}

	service := NewService(
		store,
		staticExtensionLister{items: []extensionpkg.ExtensionInfo{{Name: "ops-team"}}},
		func(name string) (*extensionpkg.Extension, error) {
			if name != "ops-team" {
				return nil, extensionpkg.ErrExtensionNotFound
			}
			return ext, nil
		},
		WithConfiguredDefaultChannel("default"),
	)

	if _, err := service.Activate(testutil.Context(t), ActivateRequest{
		ExtensionName:               "ops-team",
		BundleName:                  "ops",
		ProfileName:                 "alpha",
		Scope:                       ScopeGlobal,
		BindPrimaryChannelAsDefault: true,
	}); err != nil {
		t.Fatalf("Activate(alpha) error = %v", err)
	}

	if _, err := service.Activate(testutil.Context(t), ActivateRequest{
		ExtensionName:               "ops-team",
		BundleName:                  "ops",
		ProfileName:                 "beta",
		Scope:                       ScopeGlobal,
		BindPrimaryChannelAsDefault: true,
	}); !errors.Is(err, ErrDefaultChannelBusy) {
		t.Fatalf("Activate(beta) error = %v, want ErrDefaultChannelBusy", err)
	}
}

func TestServiceDeactivateCleansUpManagedResources(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	service := newMarketingService(store)

	preview, err := service.Activate(testutil.Context(t), ActivateRequest{
		ExtensionName:               "marketing-team",
		BundleName:                  "marketing",
		ProfileName:                 "default",
		Scope:                       ScopeGlobal,
		BindPrimaryChannelAsDefault: true,
	})
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}

	if err := service.Deactivate(testutil.Context(t), preview.Activation.ID); err != nil {
		t.Fatalf("Deactivate() error = %v", err)
	}

	if got := len(store.activations); got != 0 {
		t.Fatalf("len(store.activations) = %d, want 0", got)
	}
	if got := len(store.inventory); got != 0 {
		t.Fatalf("len(store.inventory) = %d, want 0", got)
	}
	if got, want := len(store.applied), 2; got != want {
		t.Fatalf("len(applied plans) = %d, want %d", got, want)
	}
	last := store.applied[len(store.applied)-1]
	if got := len(last.activeActivationIDs); got != 0 {
		t.Fatalf("len(last.activeActivationIDs) = %d, want 0", got)
	}
	if got := len(last.desiredJobs); got != 0 {
		t.Fatalf("len(last.desiredJobs) after deactivate = %d, want 0", got)
	}
	if got := len(last.desiredTriggers); got != 0 {
		t.Fatalf("len(last.desiredTriggers) after deactivate = %d, want 0", got)
	}
	if got := len(last.desiredBridges); got != 0 {
		t.Fatalf("len(last.desiredBridges) after deactivate = %d, want 0", got)
	}
}

func TestServiceUpdateActivationRestoresRecordOnReconcileFailure(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	service := newMarketingService(store)

	preview, err := service.Activate(testutil.Context(t), ActivateRequest{
		ExtensionName:               "marketing-team",
		BundleName:                  "marketing",
		ProfileName:                 "default",
		Scope:                       ScopeGlobal,
		BindPrimaryChannelAsDefault: false,
	})
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}

	syncErr := errors.New("sync failed")
	store.applyErr = syncErr
	_, err = service.UpdateActivation(testutil.Context(t), UpdateActivationRequest{
		ID:                          preview.Activation.ID,
		BindPrimaryChannelAsDefault: true,
	})
	if !errors.Is(err, syncErr) {
		t.Fatalf("UpdateActivation() error = %v, want sync failure", err)
	}

	stored, getErr := store.GetBundleActivation(testutil.Context(t), preview.Activation.ID)
	if getErr != nil {
		t.Fatalf("GetBundleActivation() error = %v", getErr)
	}
	if stored.BindPrimaryChannelAsDefault {
		t.Fatal("stored.BindPrimaryChannelAsDefault = true, want rollback to false")
	}
}

func TestServiceDeactivateReturnsRollbackFailureWhenRestoreFails(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	service := newMarketingService(store)

	preview, err := service.Activate(testutil.Context(t), ActivateRequest{
		ExtensionName:               "marketing-team",
		BundleName:                  "marketing",
		ProfileName:                 "default",
		Scope:                       ScopeGlobal,
		BindPrimaryChannelAsDefault: false,
	})
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}

	syncErr := errors.New("sync failed")
	recreateErr := errors.New("recreate failed")
	store.applyErr = syncErr
	store.createBundleActivationHook = func(activation Activation) error {
		if activation.ID == preview.Activation.ID {
			return recreateErr
		}
		return nil
	}

	err = service.Deactivate(testutil.Context(t), preview.Activation.ID)
	if err == nil {
		t.Fatal("Deactivate() error = nil, want reconcile + rollback failure")
	}
	if !errors.Is(err, syncErr) {
		t.Fatalf("Deactivate() error = %v, want sync failure", err)
	}
	if !errors.Is(err, recreateErr) {
		t.Fatalf("Deactivate() error = %v, want rollback failure", err)
	}
}

func TestServiceReconcileReturnsBeforeSyncWhenActivationResolutionFails(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	goodActivation := Activation{
		ID:            stableID("act", "marketing-team", "marketing", "default", string(ScopeGlobal), ""),
		ExtensionName: "marketing-team",
		BundleName:    "marketing",
		ProfileName:   "default",
		Scope:         ScopeGlobal,
	}
	badActivation := Activation{
		ID:            stableID("act", "broken-team", "marketing", "default", string(ScopeGlobal), ""),
		ExtensionName: "broken-team",
		BundleName:    "marketing",
		ProfileName:   "default",
		Scope:         ScopeGlobal,
	}
	store.activations[goodActivation.ID] = goodActivation
	store.activations[badActivation.ID] = badActivation
	marketing := newMarketingExtension()
	store.bundles = []resources.Record[BundleResourceSpec]{{
		Kind:  BundleResourceKind,
		ID:    BundleResourceID(marketing.Info.Name, marketing.Bundles[0].Name),
		Scope: resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
		Spec: BundleResourceSpec{
			ExtensionName:              marketing.Info.Name,
			Bundle:                     marketing.Bundles[0],
			OwnerBridgePlatform:        marketing.Manifest.Bridge.Platform,
			OwnerProvidesBridgeAdapter: true,
		},
	}}
	service := NewService(
		store,
		staticExtensionLister{items: []extensionpkg.ExtensionInfo{{Name: "marketing-team"}, {Name: "broken-team"}}},
		func(name string) (*extensionpkg.Extension, error) {
			switch name {
			case "marketing-team":
				return marketing, nil
			default:
				return nil, extensionpkg.ErrExtensionNotFound
			}
		},
	)

	err := service.Reconcile(testutil.Context(t))
	if !errors.Is(err, ErrBundleNotFound) {
		t.Fatalf("Reconcile() error = %v, want ErrBundleNotFound", err)
	}
	if got := len(store.applied); got != 0 {
		t.Fatalf("len(applied plans) = %d, want 0 after failed resolve", got)
	}
}

func TestServicePreviewRejectsWebhookTriggers(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	ext := &extensionpkg.Extension{
		Info: extensionpkg.ExtensionInfo{Name: "webhook-team"},
		Bundles: []extensionpkg.BundleSpec{{
			Name: "webhook",
			Profiles: []extensionpkg.BundleProfile{{
				Name: "default",
				Triggers: []extensionpkg.BundleTrigger{{
					Name:      "webhook",
					AgentName: "planner",
					Prompt:    "Handle webhook",
					Event:     "webhook",
					Enabled:   true,
					Retry:     automationpkg.DefaultRetryConfig(),
					FireLimit: automationpkg.DefaultFireLimitConfig(),
				}},
			}},
		}},
	}
	store.bundles = []resources.Record[BundleResourceSpec]{{
		Kind:  BundleResourceKind,
		ID:    BundleResourceID(ext.Info.Name, ext.Bundles[0].Name),
		Scope: resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
		Spec: BundleResourceSpec{
			ExtensionName: ext.Info.Name,
			Bundle:        ext.Bundles[0],
		},
	}}
	service := NewService(
		store,
		staticExtensionLister{items: []extensionpkg.ExtensionInfo{{Name: ext.Info.Name}}},
		func(name string) (*extensionpkg.Extension, error) {
			if name != ext.Info.Name {
				return nil, extensionpkg.ErrExtensionNotFound
			}
			return ext, nil
		},
	)

	_, err := service.PreviewActivation(testutil.Context(t), ActivateRequest{
		ExtensionName: ext.Info.Name,
		BundleName:    "webhook",
		ProfileName:   "default",
		Scope:         ScopeGlobal,
	})
	if !errors.Is(err, ErrWebhookUnsupported) {
		t.Fatalf("PreviewActivation() error = %v, want ErrWebhookUnsupported", err)
	}
}

func TestServiceMaterializesExternalBridgeProviderPlatform(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	consumer := &extensionpkg.Extension{
		Info: extensionpkg.ExtensionInfo{Name: "consumer-team"},
		Bundles: []extensionpkg.BundleSpec{{
			Name: "consumer",
			Profiles: []extensionpkg.BundleProfile{{
				Name: "default",
				Bridges: []extensionpkg.BundleBridgePreset{{
					Name:          "external",
					ExtensionName: "provider-team",
					DisplayName:   "Provider Bridge",
					RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
				}},
			}},
		}},
	}
	provider := &extensionpkg.Extension{
		Info: extensionpkg.ExtensionInfo{Name: "provider-team"},
		Manifest: &extensionpkg.Manifest{
			Name: "provider-team",
			Bridge: extensionpkg.BridgeConfig{
				Platform:    "slack",
				DisplayName: "Slack",
			},
		},
	}
	store.bundles = []resources.Record[BundleResourceSpec]{{
		Kind:  BundleResourceKind,
		ID:    BundleResourceID(consumer.Info.Name, consumer.Bundles[0].Name),
		Scope: resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
		Spec: BundleResourceSpec{
			ExtensionName: consumer.Info.Name,
			Bundle:        consumer.Bundles[0],
		},
	}}
	service := NewService(
		store,
		staticExtensionLister{items: []extensionpkg.ExtensionInfo{{Name: consumer.Info.Name}}},
		func(name string) (*extensionpkg.Extension, error) {
			switch name {
			case consumer.Info.Name:
				return consumer, nil
			case provider.Info.Name:
				return provider, nil
			default:
				return nil, extensionpkg.ErrExtensionNotFound
			}
		},
	)

	preview, err := service.PreviewActivation(testutil.Context(t), ActivateRequest{
		ExtensionName: consumer.Info.Name,
		BundleName:    "consumer",
		ProfileName:   "default",
		Scope:         ScopeGlobal,
	})
	if err != nil {
		t.Fatalf("PreviewActivation() error = %v", err)
	}
	if got, want := len(preview.Inventory), 1; got != want {
		t.Fatalf("len(preview.Inventory) = %d, want %d", got, want)
	}
	plan := store.applied
	if len(plan) != 0 {
		t.Fatalf("preview applied plans = %d, want 0", len(plan))
	}
}

func TestServiceActivateWorkspaceScopedResources(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	resolver := memoryWorkspaceResolver{
		resolveFn: func(_ context.Context, idOrPath string) (workspacepkg.ResolvedWorkspace, error) {
			return workspacepkg.ResolvedWorkspace{
				Workspace: workspacepkg.Workspace{
					ID:      "ws-marketing",
					RootDir: filepath.Join(t.TempDir(), "workspace"),
					Name:    idOrPath,
				},
			}, nil
		},
	}
	service := newMarketingService(store, WithWorkspaceResolver(resolver))

	preview, err := service.Activate(testutil.Context(t), ActivateRequest{
		ExtensionName:               "marketing-team",
		BundleName:                  "marketing",
		ProfileName:                 "default",
		Scope:                       ScopeWorkspace,
		Workspace:                   "marketing-workspace",
		BindPrimaryChannelAsDefault: false,
	})
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}

	if got, want := preview.Activation.WorkspaceID, "ws-marketing"; got != want {
		t.Fatalf("Activation.WorkspaceID = %q, want %q", got, want)
	}
	plan := store.applied[0]
	if got, want := plan.desiredJobs[0].Scope, automationpkg.AutomationScopeWorkspace; got != want {
		t.Fatalf("job scope = %q, want %q", got, want)
	}
	if got, want := plan.desiredJobs[0].WorkspaceID, "ws-marketing"; got != want {
		t.Fatalf("job workspace = %q, want %q", got, want)
	}

	storedBridge := plan.desiredBridges[0]
	if got, want := storedBridge.Scope, bridgepkg.ScopeWorkspace; got != want {
		t.Fatalf("bridge scope = %q, want %q", got, want)
	}
	if got, want := storedBridge.WorkspaceID, "ws-marketing"; got != want {
		t.Fatalf("bridge workspace = %q, want %q", got, want)
	}
}

func TestServiceActivateWorkspacePathUsesResolveOrRegister(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	called := false
	resolver := memoryWorkspaceResolver{
		resolveOrRegisterFn: func(_ context.Context, path string) (workspacepkg.ResolvedWorkspace, error) {
			called = true
			return workspacepkg.ResolvedWorkspace{
				Workspace: workspacepkg.Workspace{
					ID:      "ws-path",
					RootDir: path,
					Name:    "path-workspace",
				},
			}, nil
		},
	}
	service := newMarketingService(store, WithWorkspaceResolver(resolver))

	preview, err := service.Activate(testutil.Context(t), ActivateRequest{
		ExtensionName: "marketing-team",
		BundleName:    "marketing",
		ProfileName:   "default",
		Scope:         ScopeWorkspace,
		Workspace:     filepath.Join(t.TempDir(), "workspace"),
	})
	if err != nil {
		t.Fatalf("Activate(path workspace) error = %v", err)
	}
	if !called {
		t.Fatal("ResolveOrRegister was not called for path-like workspace ref")
	}
	if got, want := preview.Activation.WorkspaceID, "ws-path"; got != want {
		t.Fatalf("WorkspaceID = %q, want %q", got, want)
	}
}
