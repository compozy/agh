//go:build integration

package globaldb

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/channels"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestGlobalDBChannelInstanceRoundTripAcrossReopen(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), store.GlobalDatabaseName)
	first, err := OpenGlobalDB(testutil.Context(t), dbPath)
	if err != nil {
		t.Fatalf("OpenGlobalDB(first) error = %v", err)
	}

	workspaceID := registerWorkspaceForGlobalTests(t, first, "integration-channel-instance", filepath.Join(t.TempDir(), "integration-channel-instance"))
	instance := channels.ChannelInstance{
		ID:            "chan-integration",
		Scope:         channels.ScopeWorkspace,
		WorkspaceID:   workspaceID,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Integration Telegram",
		Enabled:       true,
		Status:        channels.ChannelStatusReady,
		RoutingPolicy: channels.RoutingPolicy{IncludePeer: true},
		CreatedAt:     time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
	}
	if err := first.InsertChannelInstance(testutil.Context(t), instance); err != nil {
		t.Fatalf("InsertChannelInstance() error = %v", err)
	}
	if err := first.Close(testutil.Context(t)); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}

	second, err := OpenGlobalDB(testutil.Context(t), dbPath)
	if err != nil {
		t.Fatalf("OpenGlobalDB(second) error = %v", err)
	}
	t.Cleanup(func() {
		if err := second.Close(testutil.Context(t)); err != nil {
			t.Fatalf("Close(second) error = %v", err)
		}
	})

	assertTablesPresent(t, second.db, "channel_instances", "channel_secret_bindings", "channel_routes", "channel_ingest_dedup")

	loaded, err := second.GetChannelInstance(testutil.Context(t), instance.ID)
	if err != nil {
		t.Fatalf("GetChannelInstance() error = %v", err)
	}
	if loaded.Scope != channels.ScopeWorkspace || loaded.WorkspaceID != workspaceID {
		t.Fatalf("loaded channel instance = %#v", loaded)
	}
}

func TestGlobalDBChannelRouteSurvivesReopen(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), store.GlobalDatabaseName)
	first, err := OpenGlobalDB(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("OpenGlobalDB(first) error = %v", err)
	}

	workspaceID := registerWorkspaceForGlobalTests(t, first, "integration-channel-route", filepath.Join(t.TempDir(), "integration-channel-route"))
	registerSessionForGlobalTests(t, first, "sess-channel-route")
	instance := channels.ChannelInstance{
		ID:            "chan-route",
		Scope:         channels.ScopeWorkspace,
		WorkspaceID:   workspaceID,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Route Telegram",
		Enabled:       true,
		Status:        channels.ChannelStatusReady,
		RoutingPolicy: channels.RoutingPolicy{IncludePeer: true, IncludeThread: true},
		CreatedAt:     time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
	}
	if err := first.InsertChannelInstance(testutil.Context(t), instance); err != nil {
		t.Fatalf("InsertChannelInstance() error = %v", err)
	}

	route := channels.ChannelRoute{
		Scope:             channels.ScopeWorkspace,
		WorkspaceID:       workspaceID,
		ChannelInstanceID: instance.ID,
		PeerID:            "peer-1",
		ThreadID:          "thread-9",
		SessionID:         "sess-channel-route",
		AgentName:         "coder",
		LastActivityAt:    time.Date(2026, 4, 10, 12, 5, 0, 0, time.UTC),
		CreatedAt:         time.Date(2026, 4, 10, 12, 5, 0, 0, time.UTC),
		UpdatedAt:         time.Date(2026, 4, 10, 12, 5, 0, 0, time.UTC),
	}
	if err := first.PutChannelRoute(testutil.Context(t), route); err != nil {
		t.Fatalf("PutChannelRoute() error = %v", err)
	}
	if err := first.Close(testutil.Context(t)); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}

	second, err := OpenGlobalDB(testutil.Context(t), dbPath)
	if err != nil {
		t.Fatalf("OpenGlobalDB(second) error = %v", err)
	}
	t.Cleanup(func() {
		if err := second.Close(testutil.Context(t)); err != nil {
			t.Fatalf("Close(second) error = %v", err)
		}
	})

	resolved, err := second.ResolveChannelRoute(testutil.Context(t), route.RoutingKey())
	if err != nil {
		t.Fatalf("ResolveChannelRoute() error = %v", err)
	}
	if resolved.SessionID != route.SessionID || resolved.ThreadID != route.ThreadID || resolved.PeerID != route.PeerID {
		t.Fatalf("resolved route = %#v, want session/thread/peer from %#v", resolved, route)
	}
}

func TestGlobalDBGlobalAndWorkspaceInstancesCoexist(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(t, globalDB, "integration-coexist", filepath.Join(t.TempDir(), "integration-coexist"))

	globalInstance := channels.ChannelInstance{
		ID:            "chan-global",
		Scope:         channels.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Global Telegram",
		Enabled:       true,
		Status:        channels.ChannelStatusReady,
		RoutingPolicy: channels.RoutingPolicy{IncludePeer: true},
	}
	workspaceInstance := channels.ChannelInstance{
		ID:            "chan-workspace",
		Scope:         channels.ScopeWorkspace,
		WorkspaceID:   workspaceID,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Workspace Telegram",
		Enabled:       true,
		Status:        channels.ChannelStatusReady,
		RoutingPolicy: channels.RoutingPolicy{IncludePeer: true},
	}

	if err := globalDB.InsertChannelInstance(testutil.Context(t), globalInstance); err != nil {
		t.Fatalf("InsertChannelInstance(global) error = %v", err)
	}
	if err := globalDB.InsertChannelInstance(testutil.Context(t), workspaceInstance); err != nil {
		t.Fatalf("InsertChannelInstance(workspace) error = %v", err)
	}

	instances, err := globalDB.ListChannelInstances(testutil.Context(t))
	if err != nil {
		t.Fatalf("ListChannelInstances() error = %v", err)
	}
	if got, want := len(instances), 2; got != want {
		t.Fatalf("len(instances) = %d, want %d", got, want)
	}
}

func TestGlobalDBExpiredDedupRecordsExcluded(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	instance := channels.ChannelInstance{
		ID:            "chan-dedup",
		Scope:         channels.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Dedup Telegram",
		Enabled:       true,
		Status:        channels.ChannelStatusReady,
		RoutingPolicy: channels.RoutingPolicy{IncludePeer: true},
	}
	if err := globalDB.InsertChannelInstance(testutil.Context(t), instance); err != nil {
		t.Fatalf("InsertChannelInstance() error = %v", err)
	}

	base := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	live := channels.IngestDedupRecord{
		IdempotencyKey:    "idem-live",
		ChannelInstanceID: instance.ID,
		ReceivedAt:        base,
		ExpiresAt:         base.Add(5 * time.Minute),
	}
	expired := channels.IngestDedupRecord{
		IdempotencyKey:    "idem-expired",
		ChannelInstanceID: instance.ID,
		ReceivedAt:        base.Add(-2 * time.Minute),
		ExpiresAt:         base.Add(-time.Minute),
	}
	if err := globalDB.PutChannelIngestDedup(testutil.Context(t), live); err != nil {
		t.Fatalf("PutChannelIngestDedup(live) error = %v", err)
	}
	if err := globalDB.PutChannelIngestDedup(testutil.Context(t), expired); err != nil {
		t.Fatalf("PutChannelIngestDedup(expired) error = %v", err)
	}

	if _, err := globalDB.GetChannelIngestDedup(testutil.Context(t), live.IdempotencyKey, base.Add(time.Minute)); err != nil {
		t.Fatalf("GetChannelIngestDedup(live) error = %v", err)
	}
	if _, err := globalDB.GetChannelIngestDedup(testutil.Context(t), expired.IdempotencyKey, base.Add(time.Minute)); !errors.Is(err, channels.ErrIngestDedupRecordNotFound) {
		t.Fatalf("GetChannelIngestDedup(expired) error = %v, want ErrIngestDedupRecordNotFound", err)
	}
}
