package globaldb

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

func TestOpenGlobalDBCreatesBridgeTables(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)

	assertTablesPresent(
		t,
		globalDB.db,
		"bridge_instances",
		"bridge_secret_bindings",
		"bridge_routes",
		"bridge_ingest_dedup",
	)
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
		"dm_policy",
		"routing_policy",
		"provider_config",
		"delivery_defaults",
		"degradation_reason",
		"degradation_message",
		"created_at",
		"updated_at",
	})
	assertTableColumns(t, globalDB.db, "bridge_secret_bindings", []string{
		"bridge_instance_id",
		"binding_name",
		"secret_ref",
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

	now := time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)
	if _, err := globalDB.db.ExecContext(
		testutil.Context(t),
		`INSERT INTO bridge_instances (
			id, scope, workspace_id, platform, extension_name, display_name, enabled, status, routing_policy, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"brg-default-source",
		string(bridges.ScopeGlobal),
		nil,
		"telegram",
		"telegram-adapter",
		"Default Source Bridge",
		true,
		string(bridges.BridgeStatusReady),
		`{"include_peer":true}`,
		store.FormatTimestamp(now),
		store.FormatTimestamp(now),
	); err != nil {
		t.Fatalf("ExecContext(insert legacy bridge row without source) error = %v", err)
	}

	loaded, err := globalDB.GetBridgeInstance(testutil.Context(t), "brg-default-source")
	if err != nil {
		t.Fatalf("GetBridgeInstance() error = %v", err)
	}
	if got, want := loaded.Source, bridges.BridgeInstanceSourceDynamic; got != want {
		t.Fatalf("loaded.Source = %q, want %q", got, want)
	}
	if loaded.Source == "" {
		t.Fatal("loaded.Source = empty, want default source value")
	}
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

func TestOpenGlobalDBMigratesLegacyBridgeSecretBindingsVaultRefColumn(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	dbPath := filepath.Join(t.TempDir(), GlobalDatabaseName)
	db, err := openSQLiteDatabase(ctx, dbPath, nil)
	if err != nil {
		t.Fatalf("openSQLiteDatabase() error = %v", err)
	}

	if err := store.RunMigrations(ctx, db, globalSchemaMigrations[:10]); err != nil {
		t.Fatalf("RunMigrations(v1-v10) error = %v", err)
	}

	now := time.Date(2026, 5, 1, 18, 30, 0, 0, time.UTC)
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO bridge_instances (
			id, scope, workspace_id, platform, extension_name, display_name, source, enabled, status, dm_policy, routing_policy, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"brg-legacy",
		string(bridges.ScopeGlobal),
		nil,
		"telegram",
		"telegram-adapter",
		"Legacy Bridge",
		string(bridges.BridgeInstanceSourceDynamic),
		true,
		string(bridges.BridgeStatusReady),
		string(bridges.BridgeDMPolicyOpen),
		`{"include_peer":true}`,
		store.FormatTimestamp(now),
		store.FormatTimestamp(now),
	); err != nil {
		t.Fatalf("ExecContext(insert bridge instance) error = %v", err)
	}

	legacyRef := "vault:bridges/brg-legacy/bot_token"
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO bridge_secret_bindings (
			bridge_instance_id, binding_name, secret_ref, kind, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?)`,
		"brg-legacy",
		"bot_token",
		legacyRef,
		"token",
		store.FormatTimestamp(now),
		store.FormatTimestamp(now),
	); err != nil {
		t.Fatalf("ExecContext(insert bridge binding) error = %v", err)
	}

	if _, err := db.ExecContext(
		ctx,
		`ALTER TABLE bridge_secret_bindings RENAME COLUMN secret_ref TO vault_ref`,
	); err != nil {
		t.Fatalf("ExecContext(rename secret_ref->vault_ref) error = %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close(legacy db setup) error = %v", err)
	}

	globalDB, err := OpenGlobalDB(ctx, dbPath)
	if err != nil {
		t.Fatalf("OpenGlobalDB(legacy bridge schema) error = %v", err)
	}
	t.Cleanup(func() {
		if err := globalDB.Close(testutil.Context(t)); err != nil {
			t.Fatalf("Close(migrated global db) error = %v", err)
		}
	})

	assertTableColumns(t, globalDB.db, "bridge_secret_bindings", []string{
		"bridge_instance_id",
		"binding_name",
		"secret_ref",
		"kind",
		"created_at",
		"updated_at",
	})

	binding, err := globalDB.GetBridgeSecretBinding(ctx, "brg-legacy", "bot_token")
	if err != nil {
		t.Fatalf("GetBridgeSecretBinding() error = %v", err)
	}
	if got, want := binding.SecretRef, legacyRef; got != want {
		t.Fatalf("binding.SecretRef = %q, want %q", got, want)
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

	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"bridge-workspace",
		filepath.Join(t.TempDir(), "bridge-workspace"),
	)
	instance := bridges.BridgeInstance{
		ID:             "brg-workspace",
		Scope:          bridges.ScopeWorkspace,
		WorkspaceID:    workspaceID,
		Platform:       "telegram",
		ExtensionName:  "telegram-adapter",
		DisplayName:    "Workspace Telegram",
		Enabled:        true,
		Status:         bridges.BridgeStatusReady,
		DMPolicy:       bridges.BridgeDMPolicyAllowlist,
		RoutingPolicy:  bridges.RoutingPolicy{IncludePeer: true, IncludeThread: true},
		ProviderConfig: []byte(`{"mode":"bot","tenant":"workspace-alpha"}`),
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
	if got, want := loaded.DMPolicy, bridges.BridgeDMPolicyAllowlist; got != want {
		t.Fatalf("loaded.DMPolicy = %q, want %q", got, want)
	}
	if got, want := string(loaded.ProviderConfig), `{"mode":"bot","tenant":"workspace-alpha"}`; got != want {
		t.Fatalf("loaded.ProviderConfig = %s, want %s", got, want)
	}

	loaded.DisplayName = "Workspace Telegram Updated"
	loaded.Enabled = false
	loaded.Status = bridges.BridgeStatusDisabled
	loaded.DMPolicy = bridges.BridgeDMPolicyOpen
	loaded.ProviderConfig = []byte(`{"mode":"bot","tenant":"workspace-beta"}`)
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
	if got, want := string(instances[0].ProviderConfig), `{"mode":"bot","tenant":"workspace-beta"}`; got != want {
		t.Fatalf("instances[0].ProviderConfig = %s, want %s", got, want)
	}

	binding := bridges.BridgeSecretBinding{
		BridgeInstanceID: instance.ID,
		BindingName:      "bot_token",
		SecretRef:        "vault:bridges/" + instance.ID + "/bot_token",
		Kind:             "token",
	}
	if err := globalDB.PutBridgeSecretBinding(testutil.Context(t), binding); err != nil {
		t.Fatalf("PutBridgeSecretBinding() error = %v", err)
	}

	gotBinding, err := globalDB.GetBridgeSecretBinding(
		testutil.Context(t),
		binding.BridgeInstanceID,
		binding.BindingName,
	)
	if err != nil {
		t.Fatalf("GetBridgeSecretBinding() error = %v", err)
	}
	if gotBinding.SecretRef != binding.SecretRef {
		t.Fatalf("GetBridgeSecretBinding().SecretRef = %q, want %q", gotBinding.SecretRef, binding.SecretRef)
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
	if _, err := globalDB.GetBridgeIngestDedup(
		testutil.Context(t),
		expired.IdempotencyKey,
		base.Add(time.Minute),
	); !errors.Is(
		err,
		bridges.ErrIngestDedupRecordNotFound,
	) {
		t.Fatalf("GetBridgeIngestDedup(expired) error = %v, want ErrIngestDedupRecordNotFound", err)
	}

	deleted, err := globalDB.DeleteExpiredBridgeIngestDedup(testutil.Context(t), base.Add(time.Minute))
	if err != nil {
		t.Fatalf("DeleteExpiredBridgeIngestDedup() error = %v", err)
	}
	if got, want := deleted, int64(1); got != want {
		t.Fatalf("DeleteExpiredBridgeIngestDedup() = %d, want %d", got, want)
	}

	if err := globalDB.DeleteBridgeSecretBinding(
		testutil.Context(t),
		binding.BridgeInstanceID,
		binding.BindingName,
	); err != nil {
		t.Fatalf("DeleteBridgeSecretBinding() error = %v", err)
	}
	if err := globalDB.DeleteBridgeInstance(testutil.Context(t), instance.ID); err != nil {
		t.Fatalf("DeleteBridgeInstance() error = %v", err)
	}
}

func TestGlobalDBReplaceBridgeInstancesAtomicallySwapsProjection(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	now := time.Date(2026, 4, 16, 13, 0, 0, 0, time.UTC)
	stale := bridges.BridgeInstance{
		ID:            "brg-stale",
		Scope:         bridges.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "ext-telegram",
		DisplayName:   "Stale Bridge",
		Source:        bridges.BridgeInstanceSourceDynamic,
		Enabled:       true,
		Status:        bridges.BridgeStatusReady,
		DMPolicy:      bridges.BridgeDMPolicyOpen,
		RoutingPolicy: bridges.RoutingPolicy{IncludePeer: true},
		CreatedAt:     now.Add(-time.Hour),
		UpdatedAt:     now.Add(-time.Hour),
	}
	keep := stale
	keep.ID = "brg-keep"
	keep.DisplayName = "Keep Bridge"
	if err := globalDB.InsertBridgeInstance(testutil.Context(t), stale); err != nil {
		t.Fatalf("InsertBridgeInstance(stale) error = %v", err)
	}
	if err := globalDB.InsertBridgeInstance(testutil.Context(t), keep); err != nil {
		t.Fatalf("InsertBridgeInstance(keep) error = %v", err)
	}

	keep.DisplayName = "Projected Bridge"
	keep.UpdatedAt = now
	added := bridges.BridgeInstance{
		ID:            "brg-added",
		Scope:         bridges.ScopeGlobal,
		Platform:      "slack",
		ExtensionName: "ext-slack",
		DisplayName:   "Added Bridge",
		Source:        bridges.BridgeInstanceSourceDynamic,
		Enabled:       false,
		Status:        bridges.BridgeStatusDisabled,
		DMPolicy:      bridges.BridgeDMPolicyPairing,
		RoutingPolicy: bridges.RoutingPolicy{IncludePeer: true},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := globalDB.ReplaceBridgeInstances(testutil.Context(t), []bridges.BridgeInstance{keep, added}); err != nil {
		t.Fatalf("ReplaceBridgeInstances() error = %v", err)
	}

	if _, err := globalDB.GetBridgeInstance(testutil.Context(t), stale.ID); !errors.Is(
		err,
		bridges.ErrBridgeInstanceNotFound,
	) {
		t.Fatalf("GetBridgeInstance(stale) error = %v, want ErrBridgeInstanceNotFound", err)
	}
	loaded, err := globalDB.GetBridgeInstance(testutil.Context(t), keep.ID)
	if err != nil {
		t.Fatalf("GetBridgeInstance(keep) error = %v", err)
	}
	if got, want := loaded.DisplayName, "Projected Bridge"; got != want {
		t.Fatalf("loaded.DisplayName = %q, want %q", got, want)
	}
	instances, err := globalDB.ListBridgeInstances(testutil.Context(t))
	if err != nil {
		t.Fatalf("ListBridgeInstances() error = %v", err)
	}
	if got, want := len(instances), 2; got != want {
		t.Fatalf("len(ListBridgeInstances()) = %d, want %d", got, want)
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
	if _, err := globalDB.GetBridgeRoute(
		testutil.Context(t),
		canonical.RoutingKeyHash,
	); !errors.Is(
		err,
		bridges.ErrBridgeRouteNotFound,
	) {
		t.Fatalf("GetBridgeRoute(after delete) error = %v, want ErrBridgeRouteNotFound", err)
	}
	if err := globalDB.DeleteBridgeRoute(
		testutil.Context(t),
		canonical.RoutingKeyHash,
	); !errors.Is(
		err,
		bridges.ErrBridgeRouteNotFound,
	) {
		t.Fatalf("DeleteBridgeRoute(after delete) error = %v, want ErrBridgeRouteNotFound", err)
	}
}

func TestMigrateBridgeInstanceColumnsAddsMissingColumns(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	if _, err := globalDB.db.ExecContext(testutil.Context(t), `DROP TABLE bridge_instances`); err != nil {
		t.Fatalf("drop bridge_instances error = %v", err)
	}
	if _, err := globalDB.db.ExecContext(testutil.Context(t), `CREATE TABLE bridge_instances (
		id TEXT PRIMARY KEY,
		scope TEXT NOT NULL,
		workspace_id TEXT,
		platform TEXT NOT NULL,
		extension_name TEXT NOT NULL,
		display_name TEXT NOT NULL,
		enabled BOOLEAN NOT NULL DEFAULT 1,
		status TEXT NOT NULL,
		routing_policy TEXT NOT NULL,
		delivery_defaults TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create legacy bridge_instances error = %v", err)
	}

	if err := migrateBridgeInstanceColumns(testutil.Context(t), globalDB.db); err != nil {
		t.Fatalf("migrateBridgeInstanceColumns() error = %v", err)
	}

	assertTableColumns(t, globalDB.db, "bridge_instances", []string{
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
		"dm_policy",
		"provider_config",
		"degradation_reason",
		"degradation_message",
	})
}

func TestNormalizeBridgeInstanceRecordEncodesProviderConfigAndDegradation(t *testing.T) {
	t.Parallel()

	instance := bridges.BridgeInstance{
		ID:               "brg-encode",
		Scope:            bridges.ScopeGlobal,
		Platform:         "slack",
		ExtensionName:    "slack-adapter",
		DisplayName:      "Slack",
		Enabled:          true,
		Status:           bridges.BridgeStatusDegraded,
		DMPolicy:         bridges.BridgeDMPolicyAllowlist,
		RoutingPolicy:    bridges.RoutingPolicy{IncludePeer: true},
		ProviderConfig:   json.RawMessage(`{"mode":"bot","tenant":"acme"}`),
		DeliveryDefaults: json.RawMessage(`{"mode":"reply","peer_id":"peer-1"}`),
		Degradation: &bridges.BridgeDegradation{
			Reason:  bridges.BridgeDegradationReasonRateLimited,
			Message: "provider throttled requests",
		},
	}

	normalized, routingPolicyJSON, providerConfig, deliveryDefaults, degradationReason, degradationMessage, err := normalizeBridgeInstanceRecord(
		instance,
	)
	if err != nil {
		t.Fatalf("normalizeBridgeInstanceRecord() error = %v", err)
	}
	if got, want := normalized.DMPolicy, bridges.BridgeDMPolicyAllowlist; got != want {
		t.Fatalf("normalized.DMPolicy = %q, want %q", got, want)
	}
	if got, want := providerConfig, any(`{"mode":"bot","tenant":"acme"}`); got != want {
		t.Fatalf("providerConfig = %#v, want %#v", got, want)
	}
	if got, want := deliveryDefaults, any(`{"mode":"reply","peer_id":"peer-1"}`); got != want {
		t.Fatalf("deliveryDefaults = %#v, want %#v", got, want)
	}
	if got, want := degradationReason, any("rate_limited"); got != want {
		t.Fatalf("degradationReason = %#v, want %#v", got, want)
	}
	if degradationMessage == nil {
		t.Fatal("degradationMessage = nil, want value")
	}
	if routingPolicyJSON == "" {
		t.Fatal("routingPolicyJSON = empty, want JSON")
	}
}

func TestGlobalDBBridgeDeleteAndRouteLookupNotFound(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)

	if err := globalDB.DeleteBridgeInstance(
		testutil.Context(t),
		"missing-bridge",
	); !errors.Is(
		err,
		bridges.ErrBridgeInstanceNotFound,
	) {
		t.Fatalf("DeleteBridgeInstance(missing) error = %v, want ErrBridgeInstanceNotFound", err)
	}
	if err := globalDB.DeleteBridgeSecretBinding(
		testutil.Context(t),
		"missing-bridge",
		"bot_token",
	); !errors.Is(
		err,
		bridges.ErrBridgeSecretBindingNotFound,
	) {
		t.Fatalf("DeleteBridgeSecretBinding(missing) error = %v, want ErrBridgeSecretBindingNotFound", err)
	}
	if _, err := globalDB.ResolveBridgeRoute(testutil.Context(t), bridges.RoutingKey{
		Scope:            bridges.ScopeGlobal,
		BridgeInstanceID: "missing-bridge",
		PeerID:           "peer-1",
	}); !errors.Is(err, bridges.ErrBridgeRouteNotFound) {
		t.Fatalf("ResolveBridgeRoute(missing) error = %v, want ErrBridgeRouteNotFound", err)
	}
	if err := globalDB.DeleteBridgeRoute(
		testutil.Context(t),
		"missing-route",
	); !errors.Is(
		err,
		bridges.ErrBridgeRouteNotFound,
	) {
		t.Fatalf("DeleteBridgeRoute(missing) error = %v, want ErrBridgeRouteNotFound", err)
	}
}

func TestGlobalDBBridgeMissingLookupsAndHelpers(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	if _, err := globalDB.GetBridgeInstance(
		testutil.Context(t),
		"missing",
	); !errors.Is(
		err,
		bridges.ErrBridgeInstanceNotFound,
	) {
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
	if err := globalDB.DeleteBridgeInstance(
		testutil.Context(t),
		"missing",
	); !errors.Is(
		err,
		bridges.ErrBridgeInstanceNotFound,
	) {
		t.Fatalf("DeleteBridgeInstance(missing) error = %v, want ErrBridgeInstanceNotFound", err)
	}
	if _, err := globalDB.GetBridgeSecretBinding(
		testutil.Context(t),
		"missing",
		"token",
	); !errors.Is(
		err,
		bridges.ErrBridgeSecretBindingNotFound,
	) {
		t.Fatalf("GetBridgeSecretBinding(missing) error = %v, want ErrBridgeSecretBindingNotFound", err)
	}
	if err := globalDB.DeleteBridgeSecretBinding(
		testutil.Context(t),
		"missing",
		"token",
	); !errors.Is(
		err,
		bridges.ErrBridgeSecretBindingNotFound,
	) {
		t.Fatalf("DeleteBridgeSecretBinding(missing) error = %v, want ErrBridgeSecretBindingNotFound", err)
	}
	if _, err := globalDB.GetBridgeRoute(
		testutil.Context(t),
		"missing",
	); !errors.Is(
		err,
		bridges.ErrBridgeRouteNotFound,
	) {
		t.Fatalf("GetBridgeRoute(missing) error = %v, want ErrBridgeRouteNotFound", err)
	}
	if _, err := globalDB.ResolveBridgeRoute(
		testutil.Context(t),
		bridges.RoutingKey{Scope: bridges.ScopeGlobal, BridgeInstanceID: "missing"},
	); !errors.Is(
		err,
		bridges.ErrBridgeRouteNotFound,
	) {
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
		SecretRef:        "vault:bridges/brg-missing/bot_token",
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
	if deleted, err := globalDB.DeleteExpiredBridgeIngestDedup(
		testutil.Context(t),
		time.Time{},
	); err != nil ||
		deleted != 0 {
		t.Fatalf("DeleteExpiredBridgeIngestDedup(default now) = (%d, %v), want (0, nil)", deleted, err)
	}
}

func TestBridgeConstraintMappers(t *testing.T) {
	t.Parallel()

	if err := mapBridgeInstanceConstraintError(
		errors.New("FOREIGN KEY constraint failed"),
	); !errors.Is(
		err,
		aghworkspace.ErrWorkspaceNotFound,
	) {
		t.Fatalf("mapBridgeInstanceConstraintError(fk) = %v, want workspace.ErrWorkspaceNotFound", err)
	}
	if err := mapBridgeInstanceConstraintError(
		errors.New("UNIQUE constraint failed"),
	); err == nil ||
		err.Error() != "UNIQUE constraint failed" {
		t.Fatalf("mapBridgeInstanceConstraintError(passthrough) = %v, want passthrough error", err)
	}
	if err := mapBridgeChildConstraintError(
		errors.New("FOREIGN KEY constraint failed"),
	); !errors.Is(
		err,
		bridges.ErrBridgeInstanceNotFound,
	) {
		t.Fatalf("mapBridgeChildConstraintError(fk) = %v, want ErrBridgeInstanceNotFound", err)
	}
	if err := mapBridgeChildConstraintError(
		errors.New("UNIQUE constraint failed"),
	); err == nil ||
		err.Error() != "UNIQUE constraint failed" {
		t.Fatalf("mapBridgeChildConstraintError(passthrough) = %v, want passthrough error", err)
	}
}
