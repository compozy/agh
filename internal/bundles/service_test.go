package bundles

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	"github.com/pedronauck/agh/internal/testutil"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

type memoryStore struct {
	activations map[string]Activation
	inventory   map[string][]InventoryItem
	bridges     map[string]bridgepkg.BridgeInstance

	createBundleActivationHook func(Activation) error
	updateBundleActivationHook func(Activation) error
	deleteBundleActivationHook func(string) error
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		activations: make(map[string]Activation),
		inventory:   make(map[string][]InventoryItem),
		bridges:     make(map[string]bridgepkg.BridgeInstance),
	}
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
	items := make([]Activation, 0, len(s.activations))
	for _, activation := range s.activations {
		items = append(items, activation)
	}
	return items, nil
}

func (s *memoryStore) ReplaceBundleActivationInventory(_ context.Context, activationID string, items []InventoryItem) error {
	s.inventory[activationID] = append([]InventoryItem(nil), items...)
	return nil
}

func (s *memoryStore) ListBundleActivationInventory(_ context.Context, activationID string) ([]InventoryItem, error) {
	return append([]InventoryItem(nil), s.inventory[activationID]...), nil
}

func (s *memoryStore) ListBridgeInstances(_ context.Context) ([]bridgepkg.BridgeInstance, error) {
	items := make([]bridgepkg.BridgeInstance, 0, len(s.bridges))
	for _, instance := range s.bridges {
		items = append(items, instance)
	}
	return items, nil
}

func (s *memoryStore) InsertBridgeInstance(_ context.Context, instance bridgepkg.BridgeInstance) error {
	s.bridges[instance.ID] = instance
	return nil
}

func (s *memoryStore) UpdateBridgeInstance(_ context.Context, instance bridgepkg.BridgeInstance) error {
	s.bridges[instance.ID] = instance
	return nil
}

func (s *memoryStore) DeleteBridgeInstance(_ context.Context, id string) error {
	delete(s.bridges, id)
	return nil
}

type recordingAutomationSyncer struct {
	source   automationpkg.JobSource
	jobs     []automationpkg.Job
	triggers []automationpkg.Trigger
	calls    int
	err      error
}

func (s *recordingAutomationSyncer) SyncManagedDefinitions(
	_ context.Context,
	source automationpkg.JobSource,
	desiredJobs []automationpkg.Job,
	desiredTriggers []automationpkg.Trigger,
	_ map[string]string,
) (automationpkg.SyncStats, error) {
	s.calls++
	s.source = source
	s.jobs = append([]automationpkg.Job(nil), desiredJobs...)
	s.triggers = append([]automationpkg.Trigger(nil), desiredTriggers...)
	if s.err != nil {
		return automationpkg.SyncStats{}, s.err
	}
	return automationpkg.SyncStats{
		JobsSynced:     len(desiredJobs),
		TriggersSynced: len(desiredTriggers),
	}, nil
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

func (r memoryWorkspaceResolver) ResolveOrRegister(ctx context.Context, path string) (workspacepkg.ResolvedWorkspace, error) {
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

func newMarketingService(store *memoryStore, automation *recordingAutomationSyncer, opts ...Option) *Service {
	ext := newMarketingExtension()
	options := []Option{
		WithAutomation(automation),
		WithBridges(bridgepkg.NewManagedSyncer(store, bridgepkg.WithManagedSyncNow(func() time.Time {
			return time.Date(2026, 4, 14, 22, 0, 0, 0, time.UTC)
		}))),
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
	automation := &recordingAutomationSyncer{}
	service := newMarketingService(store, automation)

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
	if got, want := automation.source, automationpkg.JobSourcePackage; got != want {
		t.Fatalf("automation source = %q, want %q", got, want)
	}
	if got, want := len(automation.jobs), 1; got != want {
		t.Fatalf("len(automation.jobs) = %d, want %d", got, want)
	}
	if got, want := len(automation.triggers), 1; got != want {
		t.Fatalf("len(automation.triggers) = %d, want %d", got, want)
	}
	if got, want := len(store.bridges), 1; got != want {
		t.Fatalf("len(store.bridges) = %d, want %d", got, want)
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
	automation := &recordingAutomationSyncer{}
	service := newMarketingService(store, automation)

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
	if got := len(store.bridges); got != 0 {
		t.Fatalf("len(store.bridges) = %d, want 0", got)
	}
	if got, want := automation.calls, 2; got != want {
		t.Fatalf("automation calls = %d, want %d", got, want)
	}
	if got := len(automation.jobs); got != 0 {
		t.Fatalf("len(automation.jobs) after deactivate = %d, want 0", got)
	}
	if got := len(automation.triggers); got != 0 {
		t.Fatalf("len(automation.triggers) after deactivate = %d, want 0", got)
	}
}

func TestServiceUpdateActivationRestoresRecordOnReconcileFailure(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	automation := &recordingAutomationSyncer{}
	service := newMarketingService(store, automation)

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

	automation.err = errors.New("sync failed")
	_, err = service.UpdateActivation(testutil.Context(t), UpdateActivationRequest{
		ID:                          preview.Activation.ID,
		BindPrimaryChannelAsDefault: true,
	})
	if err == nil || !strings.Contains(err.Error(), "sync failed") {
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
	automation := &recordingAutomationSyncer{}
	service := newMarketingService(store, automation)

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

	automation.err = errors.New("sync failed")
	store.createBundleActivationHook = func(activation Activation) error {
		if activation.ID == preview.Activation.ID {
			return errors.New("recreate failed")
		}
		return nil
	}

	err = service.Deactivate(testutil.Context(t), preview.Activation.ID)
	if err == nil {
		t.Fatal("Deactivate() error = nil, want reconcile + rollback failure")
	}
	if !strings.Contains(err.Error(), "sync failed") {
		t.Fatalf("Deactivate() error = %v, want sync failure", err)
	}
	if !strings.Contains(err.Error(), "recreate failed") {
		t.Fatalf("Deactivate() error = %v, want rollback failure", err)
	}
}

func TestServiceActivateWorkspaceScopedResources(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	automation := &recordingAutomationSyncer{}
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
	service := newMarketingService(store, automation, WithWorkspaceResolver(resolver))

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
	if got, want := automation.jobs[0].Scope, automationpkg.AutomationScopeWorkspace; got != want {
		t.Fatalf("job scope = %q, want %q", got, want)
	}
	if got, want := automation.jobs[0].WorkspaceID, "ws-marketing"; got != want {
		t.Fatalf("job workspace = %q, want %q", got, want)
	}

	var storedBridge bridgepkg.BridgeInstance
	for _, instance := range store.bridges {
		storedBridge = instance
	}
	if got, want := storedBridge.Scope, bridgepkg.ScopeWorkspace; got != want {
		t.Fatalf("bridge scope = %q, want %q", got, want)
	}
	if got, want := storedBridge.WorkspaceID, "ws-marketing"; got != want {
		t.Fatalf("bridge workspace = %q, want %q", got, want)
	}
}
