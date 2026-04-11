package globaldb

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/channels"
	"github.com/pedronauck/agh/internal/testutil"
	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

func TestOpenGlobalDBCreatesChannelTables(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)

	assertTablesPresent(t, globalDB.db, "channel_instances", "channel_secret_bindings", "channel_routes", "channel_ingest_dedup")
	assertTableColumns(t, globalDB.db, "channel_instances", []string{
		"id",
		"scope",
		"workspace_id",
		"platform",
		"extension_name",
		"display_name",
		"enabled",
		"status",
		"routing_policy",
		"delivery_defaults",
		"created_at",
		"updated_at",
	})
	assertTableColumns(t, globalDB.db, "channel_secret_bindings", []string{
		"channel_instance_id",
		"binding_name",
		"vault_ref",
		"kind",
		"created_at",
		"updated_at",
	})
	assertTableColumns(t, globalDB.db, "channel_routes", []string{
		"routing_key_hash",
		"scope",
		"workspace_id",
		"channel_instance_id",
		"peer_id",
		"thread_id",
		"group_id",
		"session_id",
		"agent_name",
		"last_activity_at",
		"created_at",
		"updated_at",
	})
	assertTableColumns(t, globalDB.db, "channel_ingest_dedup", []string{
		"idempotency_key",
		"channel_instance_id",
		"received_at",
		"expires_at",
	})
}

func TestGlobalDBChannelGuardClauses(t *testing.T) {
	t.Parallel()

	var nilDB *GlobalDB
	if err := nilDB.InsertChannelInstance(testutil.Context(t), channels.ChannelInstance{}); err == nil {
		t.Fatal("InsertChannelInstance(nil receiver) error = nil, want non-nil")
	}
	if _, err := nilDB.GetChannelInstance(testutil.Context(t), "chan-1"); err == nil {
		t.Fatal("GetChannelInstance(nil receiver) error = nil, want non-nil")
	}
	if err := nilDB.PutChannelSecretBinding(testutil.Context(t), channels.ChannelSecretBinding{}); err == nil {
		t.Fatal("PutChannelSecretBinding(nil receiver) error = nil, want non-nil")
	}
	if err := nilDB.PutChannelRoute(testutil.Context(t), channels.ChannelRoute{}); err == nil {
		t.Fatal("PutChannelRoute(nil receiver) error = nil, want non-nil")
	}
	if err := nilDB.PutChannelIngestDedup(testutil.Context(t), channels.IngestDedupRecord{}); err == nil {
		t.Fatal("PutChannelIngestDedup(nil receiver) error = nil, want non-nil")
	}

	globalDB := openTestGlobalDB(t)
	if err := globalDB.InsertChannelInstance(nilGlobalContext(), channels.ChannelInstance{}); err == nil {
		t.Fatal("InsertChannelInstance(nil ctx) error = nil, want non-nil")
	}
	if _, err := globalDB.GetChannelInstance(nilGlobalContext(), "chan-1"); err == nil {
		t.Fatal("GetChannelInstance(nil ctx) error = nil, want non-nil")
	}
	if err := globalDB.PutChannelSecretBinding(nilGlobalContext(), channels.ChannelSecretBinding{}); err == nil {
		t.Fatal("PutChannelSecretBinding(nil ctx) error = nil, want non-nil")
	}
	if err := globalDB.PutChannelRoute(nilGlobalContext(), channels.ChannelRoute{}); err == nil {
		t.Fatal("PutChannelRoute(nil ctx) error = nil, want non-nil")
	}
	if err := globalDB.PutChannelIngestDedup(nilGlobalContext(), channels.IngestDedupRecord{}); err == nil {
		t.Fatal("PutChannelIngestDedup(nil ctx) error = nil, want non-nil")
	}
}

func TestGlobalDBChannelPersistenceHelpers(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	base := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	callCount := 0
	globalDB.now = func() time.Time {
		callCount++
		return base.Add(time.Duration(callCount) * time.Minute)
	}

	workspaceID := registerWorkspaceForGlobalTests(t, globalDB, "channel-workspace", filepath.Join(t.TempDir(), "channel-workspace"))
	instance := channels.ChannelInstance{
		ID:            "chan-workspace",
		Scope:         channels.ScopeWorkspace,
		WorkspaceID:   workspaceID,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Workspace Telegram",
		Enabled:       true,
		Status:        channels.ChannelStatusReady,
		RoutingPolicy: channels.RoutingPolicy{IncludePeer: true, IncludeThread: true},
	}
	if err := globalDB.InsertChannelInstance(testutil.Context(t), instance); err != nil {
		t.Fatalf("InsertChannelInstance() error = %v", err)
	}

	loaded, err := globalDB.GetChannelInstance(testutil.Context(t), instance.ID)
	if err != nil {
		t.Fatalf("GetChannelInstance() error = %v", err)
	}
	if loaded.WorkspaceID != workspaceID || loaded.Status != channels.ChannelStatusReady {
		t.Fatalf("loaded channel instance = %#v", loaded)
	}

	loaded.DisplayName = "Workspace Telegram Updated"
	loaded.Enabled = false
	loaded.Status = channels.ChannelStatusDisabled
	if err := globalDB.UpdateChannelInstance(testutil.Context(t), loaded); err != nil {
		t.Fatalf("UpdateChannelInstance() error = %v", err)
	}

	instances, err := globalDB.ListChannelInstances(testutil.Context(t))
	if err != nil {
		t.Fatalf("ListChannelInstances() error = %v", err)
	}
	if got, want := len(instances), 1; got != want {
		t.Fatalf("len(instances) = %d, want %d", got, want)
	}
	if got, want := instances[0].DisplayName, "Workspace Telegram Updated"; got != want {
		t.Fatalf("instances[0].DisplayName = %q, want %q", got, want)
	}

	binding := channels.ChannelSecretBinding{
		ChannelInstanceID: instance.ID,
		BindingName:       "bot_token",
		VaultRef:          "vault://telegram-bot-token",
		Kind:              "token",
	}
	if err := globalDB.PutChannelSecretBinding(testutil.Context(t), binding); err != nil {
		t.Fatalf("PutChannelSecretBinding() error = %v", err)
	}

	gotBinding, err := globalDB.GetChannelSecretBinding(testutil.Context(t), binding.ChannelInstanceID, binding.BindingName)
	if err != nil {
		t.Fatalf("GetChannelSecretBinding() error = %v", err)
	}
	if gotBinding.VaultRef != binding.VaultRef {
		t.Fatalf("GetChannelSecretBinding().VaultRef = %q, want %q", gotBinding.VaultRef, binding.VaultRef)
	}

	bindings, err := globalDB.ListChannelSecretBindings(testutil.Context(t), binding.ChannelInstanceID)
	if err != nil {
		t.Fatalf("ListChannelSecretBindings() error = %v", err)
	}
	if got, want := len(bindings), 1; got != want {
		t.Fatalf("len(bindings) = %d, want %d", got, want)
	}

	record := channels.IngestDedupRecord{
		IdempotencyKey:    "idem-live",
		ChannelInstanceID: instance.ID,
		ReceivedAt:        base,
		ExpiresAt:         base.Add(5 * time.Minute),
	}
	if err := globalDB.PutChannelIngestDedup(testutil.Context(t), record); err != nil {
		t.Fatalf("PutChannelIngestDedup() error = %v", err)
	}

	expired := channels.IngestDedupRecord{
		IdempotencyKey:    "idem-expired",
		ChannelInstanceID: instance.ID,
		ReceivedAt:        base.Add(-2 * time.Minute),
		ExpiresAt:         base.Add(-time.Minute),
	}
	if err := globalDB.PutChannelIngestDedup(testutil.Context(t), expired); err != nil {
		t.Fatalf("PutChannelIngestDedup(expired) error = %v", err)
	}

	liveRecord, err := globalDB.GetChannelIngestDedup(testutil.Context(t), record.IdempotencyKey, base.Add(time.Minute))
	if err != nil {
		t.Fatalf("GetChannelIngestDedup(live) error = %v", err)
	}
	if liveRecord.ChannelInstanceID != instance.ID {
		t.Fatalf("GetChannelIngestDedup(live) = %#v", liveRecord)
	}
	if _, err := globalDB.GetChannelIngestDedup(testutil.Context(t), expired.IdempotencyKey, base.Add(time.Minute)); !errors.Is(err, channels.ErrIngestDedupRecordNotFound) {
		t.Fatalf("GetChannelIngestDedup(expired) error = %v, want ErrIngestDedupRecordNotFound", err)
	}

	deleted, err := globalDB.DeleteExpiredChannelIngestDedup(testutil.Context(t), base.Add(time.Minute))
	if err != nil {
		t.Fatalf("DeleteExpiredChannelIngestDedup() error = %v", err)
	}
	if got, want := deleted, int64(1); got != want {
		t.Fatalf("DeleteExpiredChannelIngestDedup() = %d, want %d", got, want)
	}

	if err := globalDB.DeleteChannelSecretBinding(testutil.Context(t), binding.ChannelInstanceID, binding.BindingName); err != nil {
		t.Fatalf("DeleteChannelSecretBinding() error = %v", err)
	}
	if err := globalDB.DeleteChannelInstance(testutil.Context(t), instance.ID); err != nil {
		t.Fatalf("DeleteChannelInstance() error = %v", err)
	}
}

func TestGlobalDBChannelRouteCRUD(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	instance := channels.ChannelInstance{
		ID:            "chan-route-unit",
		Scope:         channels.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Route Unit",
		Enabled:       true,
		Status:        channels.ChannelStatusReady,
		RoutingPolicy: channels.RoutingPolicy{IncludePeer: true, IncludeThread: true},
	}
	if err := globalDB.InsertChannelInstance(testutil.Context(t), instance); err != nil {
		t.Fatalf("InsertChannelInstance() error = %v", err)
	}

	route := channels.ChannelRoute{
		Scope:             channels.ScopeGlobal,
		ChannelInstanceID: instance.ID,
		PeerID:            "peer-1",
		ThreadID:          "thread-1",
		GroupID:           "group-1",
		SessionID:         "sess-route-unit",
		AgentName:         "coder",
		LastActivityAt:    time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
		CreatedAt:         time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
		UpdatedAt:         time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
	}
	if err := globalDB.PutChannelRoute(testutil.Context(t), route); err != nil {
		t.Fatalf("PutChannelRoute() error = %v", err)
	}

	canonical, err := route.Canonicalize()
	if err != nil {
		t.Fatalf("route.Canonicalize() error = %v", err)
	}
	loaded, err := globalDB.GetChannelRoute(testutil.Context(t), canonical.RoutingKeyHash)
	if err != nil {
		t.Fatalf("GetChannelRoute() error = %v", err)
	}
	if loaded.ThreadID != route.ThreadID || loaded.GroupID != route.GroupID {
		t.Fatalf("GetChannelRoute() = %#v, want thread/group from %#v", loaded, route)
	}

	resolved, err := globalDB.ResolveChannelRoute(testutil.Context(t), route.RoutingKey())
	if err != nil {
		t.Fatalf("ResolveChannelRoute() error = %v", err)
	}
	if resolved.RoutingKeyHash != canonical.RoutingKeyHash {
		t.Fatalf("ResolveChannelRoute().RoutingKeyHash = %q, want %q", resolved.RoutingKeyHash, canonical.RoutingKeyHash)
	}

	routes, err := globalDB.ListChannelRoutes(testutil.Context(t), instance.ID)
	if err != nil {
		t.Fatalf("ListChannelRoutes() error = %v", err)
	}
	if got, want := len(routes), 1; got != want {
		t.Fatalf("len(routes) = %d, want %d", got, want)
	}

	if err := globalDB.DeleteChannelRoute(testutil.Context(t), canonical.RoutingKeyHash); err != nil {
		t.Fatalf("DeleteChannelRoute() error = %v", err)
	}
	if _, err := globalDB.GetChannelRoute(testutil.Context(t), canonical.RoutingKeyHash); !errors.Is(err, channels.ErrChannelRouteNotFound) {
		t.Fatalf("GetChannelRoute(after delete) error = %v, want ErrChannelRouteNotFound", err)
	}
	if err := globalDB.DeleteChannelRoute(testutil.Context(t), canonical.RoutingKeyHash); !errors.Is(err, channels.ErrChannelRouteNotFound) {
		t.Fatalf("DeleteChannelRoute(after delete) error = %v, want ErrChannelRouteNotFound", err)
	}
}

func TestGlobalDBChannelMissingLookupsAndHelpers(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	if _, err := globalDB.GetChannelInstance(testutil.Context(t), "missing"); !errors.Is(err, channels.ErrChannelInstanceNotFound) {
		t.Fatalf("GetChannelInstance(missing) error = %v, want ErrChannelInstanceNotFound", err)
	}
	if err := globalDB.UpdateChannelInstance(testutil.Context(t), channels.ChannelInstance{
		ID:            "missing",
		Scope:         channels.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Missing",
		Enabled:       true,
		Status:        channels.ChannelStatusReady,
		RoutingPolicy: channels.RoutingPolicy{IncludePeer: true},
	}); !errors.Is(err, channels.ErrChannelInstanceNotFound) {
		t.Fatalf("UpdateChannelInstance(missing) error = %v, want ErrChannelInstanceNotFound", err)
	}
	if err := globalDB.DeleteChannelInstance(testutil.Context(t), "missing"); !errors.Is(err, channels.ErrChannelInstanceNotFound) {
		t.Fatalf("DeleteChannelInstance(missing) error = %v, want ErrChannelInstanceNotFound", err)
	}
	if _, err := globalDB.GetChannelSecretBinding(testutil.Context(t), "missing", "token"); !errors.Is(err, channels.ErrChannelSecretBindingNotFound) {
		t.Fatalf("GetChannelSecretBinding(missing) error = %v, want ErrChannelSecretBindingNotFound", err)
	}
	if err := globalDB.DeleteChannelSecretBinding(testutil.Context(t), "missing", "token"); !errors.Is(err, channels.ErrChannelSecretBindingNotFound) {
		t.Fatalf("DeleteChannelSecretBinding(missing) error = %v, want ErrChannelSecretBindingNotFound", err)
	}
	if _, err := globalDB.GetChannelRoute(testutil.Context(t), "missing"); !errors.Is(err, channels.ErrChannelRouteNotFound) {
		t.Fatalf("GetChannelRoute(missing) error = %v, want ErrChannelRouteNotFound", err)
	}
	if _, err := globalDB.ResolveChannelRoute(testutil.Context(t), channels.RoutingKey{Scope: channels.ScopeGlobal, ChannelInstanceID: "missing"}); !errors.Is(err, channels.ErrChannelRouteNotFound) {
		t.Fatalf("ResolveChannelRoute(missing) error = %v, want ErrChannelRouteNotFound", err)
	}

	if got := (*GlobalDB)(nil).DB(); got != nil {
		t.Fatalf("nil GlobalDB.DB() = %#v, want nil", got)
	}
	if got := globalDB.DB(); got == nil {
		t.Fatal("GlobalDB.DB() = nil, want non-nil")
	}

	raw, err := normalizeOptionalRawJSON([]byte(`{"ok":true}`))
	if err != nil {
		t.Fatalf("normalizeOptionalRawJSON(valid) error = %v", err)
	}
	if raw == nil {
		t.Fatal("normalizeOptionalRawJSON(valid) = nil, want non-nil")
	}
	if got, err := normalizeOptionalRawJSON(nil); err != nil || got != nil {
		t.Fatalf("normalizeOptionalRawJSON(nil) = (%v, %v), want (nil, nil)", got, err)
	}
	if _, err := normalizeOptionalRawJSON([]byte(`{`)); err == nil {
		t.Fatal("normalizeOptionalRawJSON(invalid) error = nil, want non-nil")
	}
}

func TestGlobalDBChannelConstraintFailuresAndDefaultDedupLookupTime(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	base := time.Date(2026, 4, 10, 15, 0, 0, 0, time.UTC)
	globalDB.now = func() time.Time { return base }

	if err := globalDB.InsertChannelInstance(testutil.Context(t), channels.ChannelInstance{
		ID:            "chan-missing-workspace",
		Scope:         channels.ScopeWorkspace,
		WorkspaceID:   "ws-missing",
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Missing Workspace",
		Enabled:       true,
		Status:        channels.ChannelStatusReady,
		RoutingPolicy: channels.RoutingPolicy{IncludePeer: true},
	}); !errors.Is(err, aghworkspace.ErrWorkspaceNotFound) {
		t.Fatalf("InsertChannelInstance(missing workspace) error = %v, want ErrWorkspaceNotFound", err)
	}

	if err := globalDB.PutChannelSecretBinding(testutil.Context(t), channels.ChannelSecretBinding{
		ChannelInstanceID: "chan-missing",
		BindingName:       "bot_token",
		VaultRef:          "vault://token",
		Kind:              "token",
	}); !errors.Is(err, channels.ErrChannelInstanceNotFound) {
		t.Fatalf("PutChannelSecretBinding(missing instance) error = %v, want ErrChannelInstanceNotFound", err)
	}

	if err := globalDB.PutChannelRoute(testutil.Context(t), channels.ChannelRoute{
		Scope:             channels.ScopeGlobal,
		ChannelInstanceID: "chan-missing",
		PeerID:            "peer-1",
		SessionID:         "sess-1",
		AgentName:         "coder",
	}); !errors.Is(err, channels.ErrChannelInstanceNotFound) {
		t.Fatalf("PutChannelRoute(missing instance) error = %v, want ErrChannelInstanceNotFound", err)
	}

	if err := globalDB.PutChannelIngestDedup(testutil.Context(t), channels.IngestDedupRecord{
		IdempotencyKey:    "idem-missing",
		ChannelInstanceID: "chan-missing",
		ReceivedAt:        base.Add(-time.Minute),
		ExpiresAt:         base.Add(time.Minute),
	}); !errors.Is(err, channels.ErrChannelInstanceNotFound) {
		t.Fatalf("PutChannelIngestDedup(missing instance) error = %v, want ErrChannelInstanceNotFound", err)
	}

	instance := channels.ChannelInstance{
		ID:            "chan-live-default-lookup",
		Scope:         channels.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Live Lookup",
		Enabled:       true,
		Status:        channels.ChannelStatusReady,
		RoutingPolicy: channels.RoutingPolicy{IncludePeer: true},
	}
	if err := globalDB.InsertChannelInstance(testutil.Context(t), instance); err != nil {
		t.Fatalf("InsertChannelInstance(valid) error = %v", err)
	}
	if err := globalDB.PutChannelIngestDedup(testutil.Context(t), channels.IngestDedupRecord{
		IdempotencyKey:    "idem-default-lookup",
		ChannelInstanceID: instance.ID,
		ReceivedAt:        base.Add(-time.Minute),
		ExpiresAt:         base.Add(time.Minute),
	}); err != nil {
		t.Fatalf("PutChannelIngestDedup(valid) error = %v", err)
	}
	if _, err := globalDB.GetChannelIngestDedup(testutil.Context(t), "idem-default-lookup", time.Time{}); err != nil {
		t.Fatalf("GetChannelIngestDedup(default lookup time) error = %v", err)
	}
	if deleted, err := globalDB.DeleteExpiredChannelIngestDedup(testutil.Context(t), time.Time{}); err != nil || deleted != 0 {
		t.Fatalf("DeleteExpiredChannelIngestDedup(default now) = (%d, %v), want (0, nil)", deleted, err)
	}
}

func TestChannelConstraintMappers(t *testing.T) {
	t.Parallel()

	if err := mapChannelInstanceConstraintError(errors.New("FOREIGN KEY constraint failed")); !errors.Is(err, aghworkspace.ErrWorkspaceNotFound) {
		t.Fatalf("mapChannelInstanceConstraintError(fk) = %v, want workspace.ErrWorkspaceNotFound", err)
	}
	if err := mapChannelInstanceConstraintError(errors.New("UNIQUE constraint failed")); err == nil || err.Error() != "UNIQUE constraint failed" {
		t.Fatalf("mapChannelInstanceConstraintError(passthrough) = %v, want passthrough error", err)
	}
	if err := mapChannelChildConstraintError(errors.New("FOREIGN KEY constraint failed")); !errors.Is(err, channels.ErrChannelInstanceNotFound) {
		t.Fatalf("mapChannelChildConstraintError(fk) = %v, want ErrChannelInstanceNotFound", err)
	}
	if err := mapChannelChildConstraintError(errors.New("UNIQUE constraint failed")); err == nil || err.Error() != "UNIQUE constraint failed" {
		t.Fatalf("mapChannelChildConstraintError(passthrough) = %v, want passthrough error", err)
	}
}
