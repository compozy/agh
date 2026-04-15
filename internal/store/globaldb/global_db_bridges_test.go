package globaldb

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/testutil"
	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

func TestOpenGlobalDBCreatesBridgeTables(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)

	assertTablesPresent(t, globalDB.db, "bridge_instances", "bridge_secret_bindings", "bridge_routes", "bridge_ingest_dedup")
	assertTableColumns(t, globalDB.db, "bridge_instances", []string{
		"id",
		"scope",
		"workspace_id",
		"platform",
		"extension_name",
		"display_name",
		"source",
		"enabled",
		"status",
		"routing_policy",
		"delivery_defaults",
		"created_at",
		"updated_at",
	})
	assertTableColumns(t, globalDB.db, "bridge_secret_bindings", []string{
		"bridge_instance_id",
		"binding_name",
		"vault_ref",
		"kind",
		"created_at",
		"updated_at",
	})
	assertTableColumns(t, globalDB.db, "bridge_routes", []string{
		"routing_key_hash",
		"scope",
		"workspace_id",
		"bridge_instance_id",
		"peer_id",
		"thread_id",
		"group_id",
		"session_id",
		"agent_name",
		"last_activity_at",
		"created_at",
		"updated_at",
	})
	assertTableColumns(t, globalDB.db, "bridge_ingest_dedup", []string{
		"idempotency_key",
		"bridge_instance_id",
		"received_at",
		"expires_at",
	})
}

func TestGlobalDBBridgeGuardClauses(t *testing.T) {
	t.Parallel()

	var nilDB *GlobalDB
	if err := nilDB.InsertBridgeInstance(testutil.Context(t), bridges.BridgeInstance{}); err == nil {
		t.Fatal("InsertBridgeInstance(nil receiver) error = nil, want non-nil")
	}
	if _, err := nilDB.GetBridgeInstance(testutil.Context(t), "brg-1"); err == nil {
		t.Fatal("GetBridgeInstance(nil receiver) error = nil, want non-nil")
	}
	if err := nilDB.PutBridgeSecretBinding(testutil.Context(t), bridges.BridgeSecretBinding{}); err == nil {
		t.Fatal("PutBridgeSecretBinding(nil receiver) error = nil, want non-nil")
	}
	if err := nilDB.PutBridgeRoute(testutil.Context(t), bridges.BridgeRoute{}); err == nil {
		t.Fatal("PutBridgeRoute(nil receiver) error = nil, want non-nil")
	}
	if err := nilDB.PutBridgeIngestDedup(testutil.Context(t), bridges.IngestDedupRecord{}); err == nil {
		t.Fatal("PutBridgeIngestDedup(nil receiver) error = nil, want non-nil")
	}

	globalDB := openTestGlobalDB(t)
	if err := globalDB.InsertBridgeInstance(nilGlobalContext(), bridges.BridgeInstance{}); err == nil {
		t.Fatal("InsertBridgeInstance(nil ctx) error = nil, want non-nil")
	}
	if _, err := globalDB.GetBridgeInstance(nilGlobalContext(), "brg-1"); err == nil {
		t.Fatal("GetBridgeInstance(nil ctx) error = nil, want non-nil")
	}
	if err := globalDB.PutBridgeSecretBinding(nilGlobalContext(), bridges.BridgeSecretBinding{}); err == nil {
		t.Fatal("PutBridgeSecretBinding(nil ctx) error = nil, want non-nil")
	}
	if err := globalDB.PutBridgeRoute(nilGlobalContext(), bridges.BridgeRoute{}); err == nil {
		t.Fatal("PutBridgeRoute(nil ctx) error = nil, want non-nil")
	}
	if err := globalDB.PutBridgeIngestDedup(nilGlobalContext(), bridges.IngestDedupRecord{}); err == nil {
		t.Fatal("PutBridgeIngestDedup(nil ctx) error = nil, want non-nil")
	}
}

func TestGlobalDBBridgePersistenceHelpers(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	base := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	callCount := 0
	globalDB.now = func() time.Time {
		callCount++
		return base.Add(time.Duration(callCount) * time.Minute)
	}

	workspaceID := registerWorkspaceForGlobalTests(t, globalDB, "bridge-workspace", filepath.Join(t.TempDir(), "bridge-workspace"))
	instance := bridges.BridgeInstance{
		ID:            "brg-workspace",
		Scope:         bridges.ScopeWorkspace,
		WorkspaceID:   workspaceID,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Workspace Telegram",
		Enabled:       true,
		Status:        bridges.BridgeStatusReady,
		RoutingPolicy: bridges.RoutingPolicy{IncludePeer: true, IncludeThread: true},
	}
	if err := globalDB.InsertBridgeInstance(testutil.Context(t), instance); err != nil {
		t.Fatalf("InsertBridgeInstance() error = %v", err)
	}

	loaded, err := globalDB.GetBridgeInstance(testutil.Context(t), instance.ID)
	if err != nil {
		t.Fatalf("GetBridgeInstance() error = %v", err)
	}
	if loaded.WorkspaceID != workspaceID || loaded.Status != bridges.BridgeStatusReady {
		t.Fatalf("loaded bridge instance = %#v", loaded)
	}

	loaded.DisplayName = "Workspace Telegram Updated"
	loaded.Enabled = false
	loaded.Status = bridges.BridgeStatusDisabled
	if err := globalDB.UpdateBridgeInstance(testutil.Context(t), loaded); err != nil {
		t.Fatalf("UpdateBridgeInstance() error = %v", err)
	}

	instances, err := globalDB.ListBridgeInstances(testutil.Context(t))
	if err != nil {
		t.Fatalf("ListBridgeInstances() error = %v", err)
	}
	if got, want := len(instances), 1; got != want {
		t.Fatalf("len(instances) = %d, want %d", got, want)
	}
	if got, want := instances[0].DisplayName, "Workspace Telegram Updated"; got != want {
		t.Fatalf("instances[0].DisplayName = %q, want %q", got, want)
	}

	binding := bridges.BridgeSecretBinding{
		BridgeInstanceID: instance.ID,
		BindingName:      "bot_token",
		VaultRef:         "vault://telegram-bot-token",
		Kind:             "token",
	}
	if err := globalDB.PutBridgeSecretBinding(testutil.Context(t), binding); err != nil {
		t.Fatalf("PutBridgeSecretBinding() error = %v", err)
	}

	gotBinding, err := globalDB.GetBridgeSecretBinding(testutil.Context(t), binding.BridgeInstanceID, binding.BindingName)
	if err != nil {
		t.Fatalf("GetBridgeSecretBinding() error = %v", err)
	}
	if gotBinding.VaultRef != binding.VaultRef {
		t.Fatalf("GetBridgeSecretBinding().VaultRef = %q, want %q", gotBinding.VaultRef, binding.VaultRef)
	}

	bindings, err := globalDB.ListBridgeSecretBindings(testutil.Context(t), binding.BridgeInstanceID)
	if err != nil {
		t.Fatalf("ListBridgeSecretBindings() error = %v", err)
	}
	if got, want := len(bindings), 1; got != want {
		t.Fatalf("len(bindings) = %d, want %d", got, want)
	}

	record := bridges.IngestDedupRecord{
		IdempotencyKey:   "idem-live",
		BridgeInstanceID: instance.ID,
		ReceivedAt:       base,
		ExpiresAt:        base.Add(5 * time.Minute),
	}
	if err := globalDB.PutBridgeIngestDedup(testutil.Context(t), record); err != nil {
		t.Fatalf("PutBridgeIngestDedup() error = %v", err)
	}

	expired := bridges.IngestDedupRecord{
		IdempotencyKey:   "idem-expired",
		BridgeInstanceID: instance.ID,
		ReceivedAt:       base.Add(-2 * time.Minute),
		ExpiresAt:        base.Add(-time.Minute),
	}
	if err := globalDB.PutBridgeIngestDedup(testutil.Context(t), expired); err != nil {
		t.Fatalf("PutBridgeIngestDedup(expired) error = %v", err)
	}

	liveRecord, err := globalDB.GetBridgeIngestDedup(testutil.Context(t), record.IdempotencyKey, base.Add(time.Minute))
	if err != nil {
		t.Fatalf("GetBridgeIngestDedup(live) error = %v", err)
	}
	if liveRecord.BridgeInstanceID != instance.ID {
		t.Fatalf("GetBridgeIngestDedup(live) = %#v", liveRecord)
	}
	if _, err := globalDB.GetBridgeIngestDedup(testutil.Context(t), expired.IdempotencyKey, base.Add(time.Minute)); !errors.Is(err, bridges.ErrIngestDedupRecordNotFound) {
		t.Fatalf("GetBridgeIngestDedup(expired) error = %v, want ErrIngestDedupRecordNotFound", err)
	}

	deleted, err := globalDB.DeleteExpiredBridgeIngestDedup(testutil.Context(t), base.Add(time.Minute))
	if err != nil {
		t.Fatalf("DeleteExpiredBridgeIngestDedup() error = %v", err)
	}
	if got, want := deleted, int64(1); got != want {
		t.Fatalf("DeleteExpiredBridgeIngestDedup() = %d, want %d", got, want)
	}

	if err := globalDB.DeleteBridgeSecretBinding(testutil.Context(t), binding.BridgeInstanceID, binding.BindingName); err != nil {
		t.Fatalf("DeleteBridgeSecretBinding() error = %v", err)
	}
	if err := globalDB.DeleteBridgeInstance(testutil.Context(t), instance.ID); err != nil {
		t.Fatalf("DeleteBridgeInstance() error = %v", err)
	}
}

func TestGlobalDBBridgeRouteCRUD(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	instance := bridges.BridgeInstance{
		ID:            "brg-route-unit",
		Scope:         bridges.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Route Unit",
		Enabled:       true,
		Status:        bridges.BridgeStatusReady,
		RoutingPolicy: bridges.RoutingPolicy{IncludePeer: true, IncludeThread: true},
	}
	if err := globalDB.InsertBridgeInstance(testutil.Context(t), instance); err != nil {
		t.Fatalf("InsertBridgeInstance() error = %v", err)
	}

	route := bridges.BridgeRoute{
		Scope:            bridges.ScopeGlobal,
		BridgeInstanceID: instance.ID,
		PeerID:           "peer-1",
		ThreadID:         "thread-1",
		GroupID:          "group-1",
		SessionID:        "sess-route-unit",
		AgentName:        "coder",
		LastActivityAt:   time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
		CreatedAt:        time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
		UpdatedAt:        time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
	}
	if err := globalDB.PutBridgeRoute(testutil.Context(t), route); err != nil {
		t.Fatalf("PutBridgeRoute() error = %v", err)
	}

	canonical, err := route.Canonicalize()
	if err != nil {
		t.Fatalf("route.Canonicalize() error = %v", err)
	}
	loaded, err := globalDB.GetBridgeRoute(testutil.Context(t), canonical.RoutingKeyHash)
	if err != nil {
		t.Fatalf("GetBridgeRoute() error = %v", err)
	}
	if loaded.ThreadID != route.ThreadID || loaded.GroupID != route.GroupID {
		t.Fatalf("GetBridgeRoute() = %#v, want thread/group from %#v", loaded, route)
	}

	resolved, err := globalDB.ResolveBridgeRoute(testutil.Context(t), route.RoutingKey())
	if err != nil {
		t.Fatalf("ResolveBridgeRoute() error = %v", err)
	}
	if resolved.RoutingKeyHash != canonical.RoutingKeyHash {
		t.Fatalf("ResolveBridgeRoute().RoutingKeyHash = %q, want %q", resolved.RoutingKeyHash, canonical.RoutingKeyHash)
	}

	routes, err := globalDB.ListBridgeRoutes(testutil.Context(t), instance.ID)
	if err != nil {
		t.Fatalf("ListBridgeRoutes() error = %v", err)
	}
	if got, want := len(routes), 1; got != want {
		t.Fatalf("len(routes) = %d, want %d", got, want)
	}

	if err := globalDB.DeleteBridgeRoute(testutil.Context(t), canonical.RoutingKeyHash); err != nil {
		t.Fatalf("DeleteBridgeRoute() error = %v", err)
	}
	if _, err := globalDB.GetBridgeRoute(testutil.Context(t), canonical.RoutingKeyHash); !errors.Is(err, bridges.ErrBridgeRouteNotFound) {
		t.Fatalf("GetBridgeRoute(after delete) error = %v, want ErrBridgeRouteNotFound", err)
	}
	if err := globalDB.DeleteBridgeRoute(testutil.Context(t), canonical.RoutingKeyHash); !errors.Is(err, bridges.ErrBridgeRouteNotFound) {
		t.Fatalf("DeleteBridgeRoute(after delete) error = %v, want ErrBridgeRouteNotFound", err)
	}
}

func TestGlobalDBBridgeMissingLookupsAndHelpers(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	if _, err := globalDB.GetBridgeInstance(testutil.Context(t), "missing"); !errors.Is(err, bridges.ErrBridgeInstanceNotFound) {
		t.Fatalf("GetBridgeInstance(missing) error = %v, want ErrBridgeInstanceNotFound", err)
	}
	if err := globalDB.UpdateBridgeInstance(testutil.Context(t), bridges.BridgeInstance{
		ID:            "missing",
		Scope:         bridges.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Missing",
		Enabled:       true,
		Status:        bridges.BridgeStatusReady,
		RoutingPolicy: bridges.RoutingPolicy{IncludePeer: true},
	}); !errors.Is(err, bridges.ErrBridgeInstanceNotFound) {
		t.Fatalf("UpdateBridgeInstance(missing) error = %v, want ErrBridgeInstanceNotFound", err)
	}
	if err := globalDB.DeleteBridgeInstance(testutil.Context(t), "missing"); !errors.Is(err, bridges.ErrBridgeInstanceNotFound) {
		t.Fatalf("DeleteBridgeInstance(missing) error = %v, want ErrBridgeInstanceNotFound", err)
	}
	if _, err := globalDB.GetBridgeSecretBinding(testutil.Context(t), "missing", "token"); !errors.Is(err, bridges.ErrBridgeSecretBindingNotFound) {
		t.Fatalf("GetBridgeSecretBinding(missing) error = %v, want ErrBridgeSecretBindingNotFound", err)
	}
	if err := globalDB.DeleteBridgeSecretBinding(testutil.Context(t), "missing", "token"); !errors.Is(err, bridges.ErrBridgeSecretBindingNotFound) {
		t.Fatalf("DeleteBridgeSecretBinding(missing) error = %v, want ErrBridgeSecretBindingNotFound", err)
	}
	if _, err := globalDB.GetBridgeRoute(testutil.Context(t), "missing"); !errors.Is(err, bridges.ErrBridgeRouteNotFound) {
		t.Fatalf("GetBridgeRoute(missing) error = %v, want ErrBridgeRouteNotFound", err)
	}
	if _, err := globalDB.ResolveBridgeRoute(testutil.Context(t), bridges.RoutingKey{Scope: bridges.ScopeGlobal, BridgeInstanceID: "missing"}); !errors.Is(err, bridges.ErrBridgeRouteNotFound) {
		t.Fatalf("ResolveBridgeRoute(missing) error = %v, want ErrBridgeRouteNotFound", err)
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

func TestGlobalDBBridgeConstraintFailuresAndDefaultDedupLookupTime(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	base := time.Date(2026, 4, 10, 15, 0, 0, 0, time.UTC)
	globalDB.now = func() time.Time { return base }

	if err := globalDB.InsertBridgeInstance(testutil.Context(t), bridges.BridgeInstance{
		ID:            "brg-missing-workspace",
		Scope:         bridges.ScopeWorkspace,
		WorkspaceID:   "ws-missing",
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Missing Workspace",
		Enabled:       true,
		Status:        bridges.BridgeStatusReady,
		RoutingPolicy: bridges.RoutingPolicy{IncludePeer: true},
	}); !errors.Is(err, aghworkspace.ErrWorkspaceNotFound) {
		t.Fatalf("InsertBridgeInstance(missing workspace) error = %v, want ErrWorkspaceNotFound", err)
	}

	if err := globalDB.PutBridgeSecretBinding(testutil.Context(t), bridges.BridgeSecretBinding{
		BridgeInstanceID: "brg-missing",
		BindingName:      "bot_token",
		VaultRef:         "vault://token",
		Kind:             "token",
	}); !errors.Is(err, bridges.ErrBridgeInstanceNotFound) {
		t.Fatalf("PutBridgeSecretBinding(missing instance) error = %v, want ErrBridgeInstanceNotFound", err)
	}

	if err := globalDB.PutBridgeRoute(testutil.Context(t), bridges.BridgeRoute{
		Scope:            bridges.ScopeGlobal,
		BridgeInstanceID: "brg-missing",
		PeerID:           "peer-1",
		SessionID:        "sess-1",
		AgentName:        "coder",
	}); !errors.Is(err, bridges.ErrBridgeInstanceNotFound) {
		t.Fatalf("PutBridgeRoute(missing instance) error = %v, want ErrBridgeInstanceNotFound", err)
	}

	if err := globalDB.PutBridgeIngestDedup(testutil.Context(t), bridges.IngestDedupRecord{
		IdempotencyKey:   "idem-missing",
		BridgeInstanceID: "brg-missing",
		ReceivedAt:       base.Add(-time.Minute),
		ExpiresAt:        base.Add(time.Minute),
	}); !errors.Is(err, bridges.ErrBridgeInstanceNotFound) {
		t.Fatalf("PutBridgeIngestDedup(missing instance) error = %v, want ErrBridgeInstanceNotFound", err)
	}

	instance := bridges.BridgeInstance{
		ID:            "brg-live-default-lookup",
		Scope:         bridges.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Live Lookup",
		Enabled:       true,
		Status:        bridges.BridgeStatusReady,
		RoutingPolicy: bridges.RoutingPolicy{IncludePeer: true},
	}
	if err := globalDB.InsertBridgeInstance(testutil.Context(t), instance); err != nil {
		t.Fatalf("InsertBridgeInstance(valid) error = %v", err)
	}
	if err := globalDB.PutBridgeIngestDedup(testutil.Context(t), bridges.IngestDedupRecord{
		IdempotencyKey:   "idem-default-lookup",
		BridgeInstanceID: instance.ID,
		ReceivedAt:       base.Add(-time.Minute),
		ExpiresAt:        base.Add(time.Minute),
	}); err != nil {
		t.Fatalf("PutBridgeIngestDedup(valid) error = %v", err)
	}
	if _, err := globalDB.GetBridgeIngestDedup(testutil.Context(t), "idem-default-lookup", time.Time{}); err != nil {
		t.Fatalf("GetBridgeIngestDedup(default lookup time) error = %v", err)
	}
	if deleted, err := globalDB.DeleteExpiredBridgeIngestDedup(testutil.Context(t), time.Time{}); err != nil || deleted != 0 {
		t.Fatalf("DeleteExpiredBridgeIngestDedup(default now) = (%d, %v), want (0, nil)", deleted, err)
	}
}

func TestBridgeConstraintMappers(t *testing.T) {
	t.Parallel()

	if err := mapBridgeInstanceConstraintError(errors.New("FOREIGN KEY constraint failed")); !errors.Is(err, aghworkspace.ErrWorkspaceNotFound) {
		t.Fatalf("mapBridgeInstanceConstraintError(fk) = %v, want workspace.ErrWorkspaceNotFound", err)
	}
	if err := mapBridgeInstanceConstraintError(errors.New("UNIQUE constraint failed")); err == nil || err.Error() != "UNIQUE constraint failed" {
		t.Fatalf("mapBridgeInstanceConstraintError(passthrough) = %v, want passthrough error", err)
	}
	if err := mapBridgeChildConstraintError(errors.New("FOREIGN KEY constraint failed")); !errors.Is(err, bridges.ErrBridgeInstanceNotFound) {
		t.Fatalf("mapBridgeChildConstraintError(fk) = %v, want ErrBridgeInstanceNotFound", err)
	}
	if err := mapBridgeChildConstraintError(errors.New("UNIQUE constraint failed")); err == nil || err.Error() != "UNIQUE constraint failed" {
		t.Fatalf("mapBridgeChildConstraintError(passthrough) = %v, want passthrough error", err)
	}
}
