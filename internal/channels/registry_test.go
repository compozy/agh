package channels_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	channelspkg "github.com/pedronauck/agh/internal/channels"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/store/globaldb"
	"github.com/pedronauck/agh/internal/testutil"
	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

type stubRegistryStore struct {
	insertChannelInstanceFn func(context.Context, channelspkg.ChannelInstance) error
	updateChannelInstanceFn func(context.Context, channelspkg.ChannelInstance) error
	getChannelInstanceFn    func(context.Context, string) (channelspkg.ChannelInstance, error)
	listChannelInstancesFn  func(context.Context) ([]channelspkg.ChannelInstance, error)
	putChannelRouteFn       func(context.Context, channelspkg.ChannelRoute) error
	resolveChannelRouteFn   func(context.Context, channelspkg.RoutingKey) (channelspkg.ChannelRoute, error)
	listChannelRoutesFn     func(context.Context, string) ([]channelspkg.ChannelRoute, error)
}

func (s stubRegistryStore) InsertChannelInstance(ctx context.Context, instance channelspkg.ChannelInstance) error {
	if s.insertChannelInstanceFn != nil {
		return s.insertChannelInstanceFn(ctx, instance)
	}
	return nil
}

func (s stubRegistryStore) UpdateChannelInstance(ctx context.Context, instance channelspkg.ChannelInstance) error {
	if s.updateChannelInstanceFn != nil {
		return s.updateChannelInstanceFn(ctx, instance)
	}
	return nil
}

func (s stubRegistryStore) GetChannelInstance(ctx context.Context, id string) (channelspkg.ChannelInstance, error) {
	if s.getChannelInstanceFn != nil {
		return s.getChannelInstanceFn(ctx, id)
	}
	return channelspkg.ChannelInstance{}, channelspkg.ErrChannelInstanceNotFound
}

func (s stubRegistryStore) ListChannelInstances(ctx context.Context) ([]channelspkg.ChannelInstance, error) {
	if s.listChannelInstancesFn != nil {
		return s.listChannelInstancesFn(ctx)
	}
	return nil, nil
}

func (s stubRegistryStore) PutChannelRoute(ctx context.Context, route channelspkg.ChannelRoute) error {
	if s.putChannelRouteFn != nil {
		return s.putChannelRouteFn(ctx, route)
	}
	return nil
}

func (s stubRegistryStore) ResolveChannelRoute(ctx context.Context, key channelspkg.RoutingKey) (channelspkg.ChannelRoute, error) {
	if s.resolveChannelRouteFn != nil {
		return s.resolveChannelRouteFn(ctx, key)
	}
	return channelspkg.ChannelRoute{}, channelspkg.ErrChannelRouteNotFound
}

func (s stubRegistryStore) ListChannelRoutes(ctx context.Context, channelInstanceID string) ([]channelspkg.ChannelRoute, error) {
	if s.listChannelRoutesFn != nil {
		return s.listChannelRoutesFn(ctx, channelInstanceID)
	}
	return nil, nil
}

func TestBuildRoutingKeyAppliesPeerOnlyPolicy(t *testing.T) {
	t.Parallel()

	instance := channelspkg.ChannelInstance{
		ID:            "chan-peer-only",
		Scope:         channelspkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Peer Only",
		Enabled:       true,
		Status:        channelspkg.ChannelStatusReady,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
	}

	key, err := channelspkg.BuildRoutingKey(instance, channelspkg.RoutingDimensions{
		PeerID:   "peer-1",
		ThreadID: "thread-1",
		GroupID:  "group-1",
	})
	if err != nil {
		t.Fatalf("BuildRoutingKey() error = %v", err)
	}
	if key.PeerID != "peer-1" {
		t.Fatalf("BuildRoutingKey().PeerID = %q, want peer-1", key.PeerID)
	}
	if key.ThreadID != "" {
		t.Fatalf("BuildRoutingKey().ThreadID = %q, want empty", key.ThreadID)
	}
	if key.GroupID != "" {
		t.Fatalf("BuildRoutingKey().GroupID = %q, want empty", key.GroupID)
	}
}

func TestBuildRoutingKeySeparatesThreadsWhenPolicyIncludesThread(t *testing.T) {
	t.Parallel()

	instance := channelspkg.ChannelInstance{
		ID:            "chan-peer-thread",
		Scope:         channelspkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Peer Thread",
		Enabled:       true,
		Status:        channelspkg.ChannelStatusReady,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true, IncludeThread: true},
	}

	first, err := channelspkg.BuildRoutingKey(instance, channelspkg.RoutingDimensions{PeerID: "peer-1", ThreadID: "thread-a"})
	if err != nil {
		t.Fatalf("BuildRoutingKey(first) error = %v", err)
	}
	second, err := channelspkg.BuildRoutingKey(instance, channelspkg.RoutingDimensions{PeerID: "peer-1", ThreadID: "thread-b"})
	if err != nil {
		t.Fatalf("BuildRoutingKey(second) error = %v", err)
	}
	if first.ThreadID == second.ThreadID {
		t.Fatalf("BuildRoutingKey() threads = %q and %q, want distinct", first.ThreadID, second.ThreadID)
	}

	firstHash, err := first.Hash()
	if err != nil {
		t.Fatalf("first.Hash() error = %v", err)
	}
	secondHash, err := second.Hash()
	if err != nil {
		t.Fatalf("second.Hash() error = %v", err)
	}
	if firstHash == secondHash {
		t.Fatalf("Hash() = %q for both routes, want distinct hashes", firstHash)
	}
}

func TestValidateInstanceStateTransitionRejectsReadyFromDisabledWithoutEnablePath(t *testing.T) {
	t.Parallel()

	current := channelspkg.ChannelInstance{
		ID:            "chan-disabled",
		Scope:         channelspkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Disabled",
		Enabled:       false,
		Status:        channelspkg.ChannelStatusDisabled,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
	}

	err := channelspkg.ValidateInstanceStateTransition(current, true, channelspkg.ChannelStatusReady)
	if !errors.Is(err, channelspkg.ErrInvalidChannelStateTransition) {
		t.Fatalf("ValidateInstanceStateTransition() error = %v, want ErrInvalidChannelStateTransition", err)
	}
}

func TestPlatformDimensionMappingValidate(t *testing.T) {
	t.Parallel()

	mapping := channelspkg.PlatformDimensionMapping{
		Platform:        "telegram",
		PeerIDConcept:   "chat or user id",
		ThreadIDConcept: "forum topic id",
		GroupIDConcept:  "group or channel id",
	}
	if err := mapping.Validate(); err != nil {
		t.Fatalf("PlatformDimensionMapping.Validate(valid) error = %v", err)
	}

	if err := (channelspkg.PlatformDimensionMapping{Platform: "telegram"}).Validate(); err == nil {
		t.Fatal("PlatformDimensionMapping.Validate(no concepts) error = nil, want non-nil")
	}
}

func TestBuildRoutingKeyRequiresConfiguredDimensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		policy   channelspkg.RoutingPolicy
		dims     channelspkg.RoutingDimensions
		wantText string
	}{
		{
			name:     "peer required",
			policy:   channelspkg.RoutingPolicy{IncludePeer: true},
			dims:     channelspkg.RoutingDimensions{},
			wantText: "peer id",
		},
		{
			name:     "thread required",
			policy:   channelspkg.RoutingPolicy{IncludePeer: true, IncludeThread: true},
			dims:     channelspkg.RoutingDimensions{PeerID: "peer-1"},
			wantText: "thread id",
		},
		{
			name:     "group required",
			policy:   channelspkg.RoutingPolicy{IncludeGroup: true},
			dims:     channelspkg.RoutingDimensions{},
			wantText: "group id",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := channelspkg.BuildRoutingKey(channelspkg.ChannelInstance{
				ID:            "chan-required",
				Scope:         channelspkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "telegram-adapter",
				DisplayName:   "Required",
				Enabled:       true,
				Status:        channelspkg.ChannelStatusReady,
				RoutingPolicy: tt.policy,
			}, tt.dims)
			if err == nil || !strings.Contains(err.Error(), tt.wantText) {
				t.Fatalf("BuildRoutingKey() error = %v, want text %q", err, tt.wantText)
			}
		})
	}
}

func TestCanonicalizeRoutingKeyRejectsBaseMismatch(t *testing.T) {
	t.Parallel()

	instance := channelspkg.ChannelInstance{
		ID:            "chan-base",
		Scope:         channelspkg.ScopeWorkspace,
		WorkspaceID:   "ws-1",
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Base",
		Enabled:       true,
		Status:        channelspkg.ChannelStatusReady,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
	}

	_, err := channelspkg.CanonicalizeRoutingKey(instance, channelspkg.RoutingKey{
		Scope:             channelspkg.ScopeGlobal,
		WorkspaceID:       "ws-2",
		ChannelInstanceID: instance.ID,
		PeerID:            "peer-1",
	})
	if err == nil {
		t.Fatal("CanonicalizeRoutingKey() error = nil, want non-nil")
	}
}

func TestValidateInstanceStateTransitionAllowedPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		current     channelspkg.ChannelInstance
		nextEnabled bool
		nextStatus  channelspkg.ChannelStatus
	}{
		{
			name: "starting to ready",
			current: channelspkg.ChannelInstance{
				ID:            "chan-starting",
				Scope:         channelspkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "telegram-adapter",
				DisplayName:   "Starting",
				Enabled:       true,
				Status:        channelspkg.ChannelStatusStarting,
				RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
			},
			nextEnabled: true,
			nextStatus:  channelspkg.ChannelStatusReady,
		},
		{
			name: "ready to starting",
			current: channelspkg.ChannelInstance{
				ID:            "chan-ready",
				Scope:         channelspkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "telegram-adapter",
				DisplayName:   "Ready",
				Enabled:       true,
				Status:        channelspkg.ChannelStatusReady,
				RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
			},
			nextEnabled: true,
			nextStatus:  channelspkg.ChannelStatusStarting,
		},
		{
			name: "degraded to ready",
			current: channelspkg.ChannelInstance{
				ID:            "chan-degraded",
				Scope:         channelspkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "telegram-adapter",
				DisplayName:   "Degraded",
				Enabled:       true,
				Status:        channelspkg.ChannelStatusDegraded,
				RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
			},
			nextEnabled: true,
			nextStatus:  channelspkg.ChannelStatusReady,
		},
		{
			name: "auth required to starting",
			current: channelspkg.ChannelInstance{
				ID:            "chan-auth",
				Scope:         channelspkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "telegram-adapter",
				DisplayName:   "Auth",
				Enabled:       true,
				Status:        channelspkg.ChannelStatusAuthRequired,
				RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
			},
			nextEnabled: true,
			nextStatus:  channelspkg.ChannelStatusStarting,
		},
		{
			name: "error to starting",
			current: channelspkg.ChannelInstance{
				ID:            "chan-error",
				Scope:         channelspkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "telegram-adapter",
				DisplayName:   "Error",
				Enabled:       true,
				Status:        channelspkg.ChannelStatusError,
				RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
			},
			nextEnabled: true,
			nextStatus:  channelspkg.ChannelStatusStarting,
		},
		{
			name: "ready to disabled",
			current: channelspkg.ChannelInstance{
				ID:            "chan-disable",
				Scope:         channelspkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "telegram-adapter",
				DisplayName:   "Disable",
				Enabled:       true,
				Status:        channelspkg.ChannelStatusReady,
				RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
			},
			nextEnabled: false,
			nextStatus:  channelspkg.ChannelStatusDisabled,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := channelspkg.ValidateInstanceStateTransition(tt.current, tt.nextEnabled, tt.nextStatus); err != nil {
				t.Fatalf("ValidateInstanceStateTransition() error = %v", err)
			}
		})
	}
}

func TestCreateInstanceRequestValidate(t *testing.T) {
	t.Parallel()

	req := channelspkg.CreateInstanceRequest{
		Scope:         channelspkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Validate",
		Enabled:       true,
		Status:        channelspkg.ChannelStatusStarting,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("CreateInstanceRequest.Validate() error = %v", err)
	}
}

func TestUpdateInstanceRequestValidate(t *testing.T) {
	t.Parallel()

	displayName := "Updated"
	deliveryDefaults := json.RawMessage(`{"peer_id":"peer-default","mode":"reply"}`)
	req := channelspkg.UpdateInstanceRequest{
		ID:               "chan-update",
		DisplayName:      &displayName,
		RoutingPolicy:    &channelspkg.RoutingPolicy{IncludePeer: true, IncludeThread: true},
		DeliveryDefaults: &deliveryDefaults,
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("UpdateInstanceRequest.Validate() error = %v", err)
	}
}

func TestRegistryCreateGetAndUpdateInstanceState(t *testing.T) {
	t.Parallel()

	registry, _ := newRegistryTestHarness(t)
	created, err := registry.CreateInstance(testutil.Context(t), channelspkg.CreateInstanceRequest{
		ID:            "chan-state",
		Scope:         channelspkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Lifecycle",
		Enabled:       true,
		Status:        channelspkg.ChannelStatusStarting,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
	})
	if err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}
	if created.Status != channelspkg.ChannelStatusStarting {
		t.Fatalf("CreateInstance().Status = %q, want starting", created.Status)
	}

	loaded, err := registry.GetInstance(testutil.Context(t), created.ID)
	if err != nil {
		t.Fatalf("GetInstance() error = %v", err)
	}
	if loaded.ID != created.ID {
		t.Fatalf("GetInstance().ID = %q, want %q", loaded.ID, created.ID)
	}

	updated, err := registry.UpdateInstanceState(testutil.Context(t), channelspkg.UpdateInstanceStateRequest{
		ID:      created.ID,
		Enabled: true,
		Status:  channelspkg.ChannelStatusReady,
	})
	if err != nil {
		t.Fatalf("UpdateInstanceState() error = %v", err)
	}
	if updated.Status != channelspkg.ChannelStatusReady {
		t.Fatalf("UpdateInstanceState().Status = %q, want ready", updated.Status)
	}

	instances, err := registry.ListInstances(testutil.Context(t))
	if err != nil {
		t.Fatalf("ListInstances() error = %v", err)
	}
	if got, want := len(instances), 1; got != want {
		t.Fatalf("len(instances) = %d, want %d", got, want)
	}
}

func TestRegistryUpdateInstanceMutatesDisplayNameRoutingPolicyAndDefaults(t *testing.T) {
	t.Parallel()

	registry, _ := newRegistryTestHarness(t)
	instance := createTestChannelInstance(t, registry, channelspkg.CreateInstanceRequest{
		ID:            "chan-update",
		Scope:         channelspkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Original",
		Enabled:       true,
		Status:        channelspkg.ChannelStatusReady,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
	})

	displayName := "Updated"
	deliveryDefaults := json.RawMessage(`{"peer_id":"peer-default","mode":"reply"}`)
	updated, err := registry.UpdateInstance(testutil.Context(t), channelspkg.UpdateInstanceRequest{
		ID:               instance.ID,
		DisplayName:      &displayName,
		RoutingPolicy:    &channelspkg.RoutingPolicy{IncludePeer: true, IncludeThread: true},
		DeliveryDefaults: &deliveryDefaults,
	})
	if err != nil {
		t.Fatalf("UpdateInstance() error = %v", err)
	}
	if updated.DisplayName != "Updated" {
		t.Fatalf("UpdateInstance().DisplayName = %q, want Updated", updated.DisplayName)
	}
	if !updated.RoutingPolicy.IncludeThread {
		t.Fatalf("UpdateInstance().RoutingPolicy = %#v, want thread routing enabled", updated.RoutingPolicy)
	}
	if got := string(updated.DeliveryDefaults); got != `{"peer_id":"peer-default","mode":"reply"}` {
		t.Fatalf("UpdateInstance().DeliveryDefaults = %s, want compact JSON", got)
	}
}

func TestRegistryCreateInstanceWrapsInsertErrors(t *testing.T) {
	t.Parallel()

	insertErr := errors.New("insert failed")
	registry := channelspkg.NewRegistry(stubRegistryStore{
		insertChannelInstanceFn: func(context.Context, channelspkg.ChannelInstance) error {
			return insertErr
		},
	})

	_, err := registry.CreateInstance(testutil.Context(t), channelspkg.CreateInstanceRequest{
		ID:            "chan-wrap-create",
		Scope:         channelspkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Wrapped Create",
		Enabled:       true,
		Status:        channelspkg.ChannelStatusStarting,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
	})
	if !errors.Is(err, insertErr) {
		t.Fatalf("CreateInstance() error = %v, want wrapped %v", err, insertErr)
	}
	if !strings.Contains(err.Error(), "create channel instance") || !strings.Contains(err.Error(), "insert") {
		t.Fatalf("CreateInstance() error = %q, want contextual insert failure", err)
	}
}

func TestRegistryListInstancesReturnsClonedDeliveryDefaults(t *testing.T) {
	t.Parallel()

	stored := []channelspkg.ChannelInstance{{
		ID:               "chan-list-clone",
		Scope:            channelspkg.ScopeGlobal,
		Platform:         "telegram",
		ExtensionName:    "telegram-adapter",
		DisplayName:      "List Clone",
		Enabled:          true,
		Status:           channelspkg.ChannelStatusReady,
		RoutingPolicy:    channelspkg.RoutingPolicy{IncludePeer: true},
		DeliveryDefaults: json.RawMessage(`{"mode":"reply"}`),
	}}
	registry := channelspkg.NewRegistry(stubRegistryStore{
		listChannelInstancesFn: func(context.Context) ([]channelspkg.ChannelInstance, error) {
			return stored, nil
		},
	})

	first, err := registry.ListInstances(testutil.Context(t))
	if err != nil {
		t.Fatalf("ListInstances(first) error = %v", err)
	}
	first[0].DeliveryDefaults[0] = 'x'
	if got := string(stored[0].DeliveryDefaults); got != `{"mode":"reply"}` {
		t.Fatalf("stored delivery defaults = %s, want original JSON", got)
	}

	second, err := registry.ListInstances(testutil.Context(t))
	if err != nil {
		t.Fatalf("ListInstances(second) error = %v", err)
	}
	if got := string(second[0].DeliveryDefaults); got != `{"mode":"reply"}` {
		t.Fatalf("ListInstances(second).DeliveryDefaults = %s, want original JSON", got)
	}
}

func TestRegistryListRoutesReturnsClonedRoutes(t *testing.T) {
	t.Parallel()

	stored := []channelspkg.ChannelRoute{{
		Scope:             channelspkg.ScopeGlobal,
		ChannelInstanceID: "chan-route-clone",
		PeerID:            "peer-1",
		SessionID:         "sess-1",
		AgentName:         "coder",
	}}
	registry := channelspkg.NewRegistry(stubRegistryStore{
		getChannelInstanceFn: func(context.Context, string) (channelspkg.ChannelInstance, error) {
			return channelspkg.ChannelInstance{
				ID:            "chan-route-clone",
				Scope:         channelspkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "telegram-adapter",
				DisplayName:   "Route Clone",
				Enabled:       true,
				Status:        channelspkg.ChannelStatusReady,
				RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
			}, nil
		},
		listChannelRoutesFn: func(context.Context, string) ([]channelspkg.ChannelRoute, error) {
			return stored, nil
		},
	})

	first, err := registry.ListRoutes(testutil.Context(t), "chan-route-clone")
	if err != nil {
		t.Fatalf("ListRoutes(first) error = %v", err)
	}
	first[0].SessionID = "sess-mutated"
	if got, want := stored[0].SessionID, "sess-1"; got != want {
		t.Fatalf("stored route session id = %q, want %q", got, want)
	}

	second, err := registry.ListRoutes(testutil.Context(t), "chan-route-clone")
	if err != nil {
		t.Fatalf("ListRoutes(second) error = %v", err)
	}
	if got, want := second[0].SessionID, "sess-1"; got != want {
		t.Fatalf("ListRoutes(second).SessionID = %q, want %q", got, want)
	}
}

func TestRegistryResolveOrCreateRouteReusesStoredSession(t *testing.T) {
	t.Parallel()

	registry, _ := newRegistryTestHarness(t)
	instance := createTestChannelInstance(t, registry, channelspkg.CreateInstanceRequest{
		ID:            "chan-route-reuse",
		Scope:         channelspkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Route Reuse",
		Enabled:       true,
		Status:        channelspkg.ChannelStatusReady,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true, IncludeThread: true},
	})

	first, created, err := registry.ResolveOrCreateRoute(testutil.Context(t), channelspkg.ChannelRoute{
		ChannelInstanceID: instance.ID,
		PeerID:            "peer-1",
		ThreadID:          "thread-1",
		SessionID:         "sess-1",
		AgentName:         "coder",
	})
	if err != nil {
		t.Fatalf("ResolveOrCreateRoute(first) error = %v", err)
	}
	if !created {
		t.Fatal("ResolveOrCreateRoute(first) created = false, want true")
	}

	second, created, err := registry.ResolveOrCreateRoute(testutil.Context(t), channelspkg.ChannelRoute{
		ChannelInstanceID: instance.ID,
		PeerID:            "peer-1",
		ThreadID:          "thread-1",
		SessionID:         "sess-2",
		AgentName:         "reviewer",
	})
	if err != nil {
		t.Fatalf("ResolveOrCreateRoute(second) error = %v", err)
	}
	if created {
		t.Fatal("ResolveOrCreateRoute(second) created = true, want false")
	}
	if second.SessionID != first.SessionID {
		t.Fatalf("ResolveOrCreateRoute(second).SessionID = %q, want %q", second.SessionID, first.SessionID)
	}

	routes, err := registry.ListRoutes(testutil.Context(t), instance.ID)
	if err != nil {
		t.Fatalf("ListRoutes() error = %v", err)
	}
	if got, want := len(routes), 1; got != want {
		t.Fatalf("len(routes) = %d, want %d", got, want)
	}
}

func TestRegistryBuildResolveAndUpsertRoute(t *testing.T) {
	t.Parallel()

	registry, db := newRegistryTestHarness(t)
	workspaceID := registerWorkspaceForChannelsTests(t, db, "ws-build-route", "build-route")
	instance := createTestChannelInstance(t, registry, channelspkg.CreateInstanceRequest{
		ID:            "chan-build-route",
		Scope:         channelspkg.ScopeWorkspace,
		WorkspaceID:   workspaceID,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Build Route",
		Enabled:       true,
		Status:        channelspkg.ChannelStatusReady,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true, IncludeThread: true},
	})

	key, err := registry.BuildRoutingKey(testutil.Context(t), channelspkg.RoutingKey{
		ChannelInstanceID: instance.ID,
		PeerID:            "peer-1",
		ThreadID:          "thread-1",
		GroupID:           "ignored-group",
	})
	if err != nil {
		t.Fatalf("BuildRoutingKey() error = %v", err)
	}
	if key.Scope != channelspkg.ScopeWorkspace || key.WorkspaceID != workspaceID {
		t.Fatalf("BuildRoutingKey() = %#v, want workspace scope %q", key, workspaceID)
	}
	if key.GroupID != "" {
		t.Fatalf("BuildRoutingKey().GroupID = %q, want empty", key.GroupID)
	}

	route, err := registry.UpsertRoute(testutil.Context(t), channelspkg.ChannelRoute{
		ChannelInstanceID: instance.ID,
		PeerID:            "peer-1",
		ThreadID:          "thread-1",
		SessionID:         "sess-1",
		AgentName:         "coder",
	})
	if err != nil {
		t.Fatalf("UpsertRoute(first) error = %v", err)
	}
	if route.SessionID != "sess-1" {
		t.Fatalf("UpsertRoute(first).SessionID = %q, want sess-1", route.SessionID)
	}

	resolved, err := registry.ResolveRoute(testutil.Context(t), channelspkg.RoutingKey{
		ChannelInstanceID: instance.ID,
		PeerID:            "peer-1",
		ThreadID:          "thread-1",
		GroupID:           "ignored-group",
	})
	if err != nil {
		t.Fatalf("ResolveRoute() error = %v", err)
	}
	if resolved.RoutingKeyHash != route.RoutingKeyHash {
		t.Fatalf("ResolveRoute().RoutingKeyHash = %q, want %q", resolved.RoutingKeyHash, route.RoutingKeyHash)
	}

	rebound, err := registry.UpsertRoute(testutil.Context(t), channelspkg.ChannelRoute{
		ChannelInstanceID: instance.ID,
		PeerID:            "peer-1",
		ThreadID:          "thread-1",
		SessionID:         "sess-2",
		AgentName:         "reviewer",
	})
	if err != nil {
		t.Fatalf("UpsertRoute(second) error = %v", err)
	}
	if rebound.SessionID != "sess-2" {
		t.Fatalf("UpsertRoute(second).SessionID = %q, want sess-2", rebound.SessionID)
	}
}

func TestRegistryResolveRouteRejectsDisabledInstance(t *testing.T) {
	t.Parallel()

	registry, _ := newRegistryTestHarness(t)
	instance := createTestChannelInstance(t, registry, channelspkg.CreateInstanceRequest{
		ID:            "chan-disabled-route",
		Scope:         channelspkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Disabled Route",
		Enabled:       false,
		Status:        channelspkg.ChannelStatusDisabled,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
	})

	_, err := registry.ResolveRoute(testutil.Context(t), channelspkg.RoutingKey{
		ChannelInstanceID: instance.ID,
		PeerID:            "peer-1",
	})
	if !errors.Is(err, channelspkg.ErrChannelInstanceUnavailable) {
		t.Fatalf("ResolveRoute(disabled) error = %v, want ErrChannelInstanceUnavailable", err)
	}
}

func TestRegistryGuardClauses(t *testing.T) {
	t.Parallel()

	var nilRegistry *channelspkg.Service
	if _, err := nilRegistry.ListInstances(testutil.Context(t)); err == nil {
		t.Fatal("nilRegistry.ListInstances() error = nil, want non-nil")
	}

	missingStore := channelspkg.NewRegistry(nil)
	if _, err := missingStore.ListInstances(testutil.Context(t)); err == nil {
		t.Fatal("NewRegistry(nil).ListInstances() error = nil, want non-nil")
	}

	registry, _ := newRegistryTestHarness(t)
	if _, err := registry.ListInstances(nilContextForChannelsTests()); err == nil {
		t.Fatal("ListInstances(nil ctx) error = nil, want non-nil")
	}
}

func newRegistryTestHarness(t *testing.T) (*channelspkg.Service, *globaldb.GlobalDB) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), store.GlobalDatabaseName)
	db, err := globaldb.OpenGlobalDB(testutil.Context(t), dbPath)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(testutil.Context(t)); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	now := time.Date(2026, 4, 10, 16, 0, 0, 0, time.UTC)
	registry := channelspkg.NewRegistry(db, channelspkg.WithNow(func() time.Time { return now }))
	return registry, db
}

func createTestChannelInstance(t *testing.T, registry *channelspkg.Service, req channelspkg.CreateInstanceRequest) *channelspkg.ChannelInstance {
	t.Helper()

	instance, err := registry.CreateInstance(testutil.Context(t), req)
	if err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}
	return instance
}

func registerWorkspaceForChannelsTests(t *testing.T, db *globaldb.GlobalDB, id string, name string) string {
	t.Helper()

	workspace := aghworkspace.Workspace{
		ID:        id,
		RootDir:   filepath.Join(t.TempDir(), name),
		Name:      name,
		CreatedAt: time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
	}
	if err := db.InsertWorkspace(testutil.Context(t), workspace); err != nil {
		t.Fatalf("InsertWorkspace() error = %v", err)
	}
	return workspace.ID
}

func nilContextForChannelsTests() context.Context {
	return nil
}
