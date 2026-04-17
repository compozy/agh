//go:build integration && !windows

package daemon

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	aghcontract "github.com/pedronauck/agh/internal/api/contract"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	aghconfig "github.com/pedronauck/agh/internal/config"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	extensiontest "github.com/pedronauck/agh/internal/extensiontest"
	"github.com/pedronauck/agh/internal/testutil/acpmock"
	e2etest "github.com/pedronauck/agh/internal/testutil/e2e"
)

const bridgeIngressFixtureAgentName = "mock-bridge-runner"

func TestDaemonE2EBridgeIngressCreatesAndReusesRouteThroughTelegramExtension(t *testing.T) {
	skipWithoutNode(t)

	repoRoot := daemonBridgeRuntimeRepoRoot(t)
	extensionDir := prepareDaemonTelegramReferenceExtension(t, repoRoot)

	markers := extensiontest.NewMarkerPaths(filepath.Join(t.TempDir(), "markers"))
	env := markers.Env()
	delete(env, extensiontest.EnvCrashOncePath)
	env["AGH_TEST_TELEGRAM_TOKEN"] = "telegram-bot-token"

	harness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{
		ConfigSeed: e2etest.ConfigSeedOptions{
			DefaultAgent:   bridgeIngressFixtureAgentName,
			PermissionMode: aghconfig.PermissionModeApproveAll,
		},
		MockAgents: []e2etest.MockAgentSpec{{
			FixturePath:  mockFixturePath(t, "bridge_ingress_fixture.json"),
			FixtureAgent: "bridge-runner",
			AgentName:    bridgeIngressFixtureAgentName,
		}},
		Env: env,
	})

	registration, ok := harness.MockAgentRegistration(bridgeIngressFixtureAgentName)
	if !ok {
		t.Fatalf("MockAgentRegistration(%q) = missing, want present", bridgeIngressFixtureAgentName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	var (
		bridgeID  string
		sessionID string
	)
	registerBridgeExtensionArtifacts(t, harness, markers, registration, &bridgeID, &sessionID)

	checksum, err := extensionpkg.ComputeDirectoryChecksum(extensionDir)
	if err != nil {
		t.Fatalf("ComputeDirectoryChecksum(%q) error = %v", extensionDir, err)
	}

	installed, err := harness.InstallExtension(ctx, aghcontract.InstallExtensionRequest{
		Path:     extensionDir,
		Checksum: checksum,
	})
	if err != nil {
		t.Fatalf("InstallExtension(%q) error = %v", extensionDir, err)
	}
	if got, want := installed.Name, "telegram-reference"; got != want {
		t.Fatalf("installed.Name = %q, want %q", got, want)
	}

	waitForRuntimeCondition(t, "telegram extension registered", 10*time.Second, func() bool {
		ext, err := harness.GetExtension(ctx, "telegram-reference")
		return err == nil && ext.Enabled
	})

	createdBridge, err := harness.CreateBridge(ctx, aghcontract.CreateBridgeRequest{
		Scope:         bridgepkg.ScopeWorkspace,
		WorkspaceID:   harness.WorkspaceID,
		Platform:      "telegram",
		ExtensionName: "telegram-reference",
		DisplayName:   "Telegram Runtime E2E",
		Enabled:       false,
		Status:        bridgepkg.BridgeStatusDisabled,
		RoutingPolicy: bridgepkg.RoutingPolicy{
			IncludePeer:   true,
			IncludeThread: true,
		},
	})
	if err != nil {
		t.Fatalf("CreateBridge() error = %v", err)
	}
	bridgeID = createdBridge.Bridge.ID
	if got, want := createdBridge.Bridge.Status.Normalize(), bridgepkg.BridgeStatusDisabled; got != want {
		t.Fatalf("created bridge status = %q, want %q", got, want)
	}
	if createdBridge.Bridge.Enabled {
		t.Fatalf("created bridge enabled = %v, want false before explicit start", createdBridge.Bridge.Enabled)
	}

	binding, err := harness.PutBridgeSecretBinding(
		ctx,
		bridgeID,
		"bot_token",
		aghcontract.PutBridgeSecretBindingRequest{
			VaultRef: "env:AGH_TEST_TELEGRAM_TOKEN",
			Kind:     "token",
		},
	)
	if err != nil {
		t.Fatalf("PutBridgeSecretBinding() error = %v", err)
	}
	if got, want := binding.BindingName, "bot_token"; got != want {
		t.Fatalf("binding.BindingName = %q, want %q", got, want)
	}

	if _, err := harness.EnableBridge(ctx, bridgeID); err != nil {
		t.Fatalf("EnableBridge(%q) error = %v", bridgeID, err)
	}

	handshake := extensiontest.WaitForHandshakeMarker(t, markers, 15*time.Second)
	if handshake.Request.Runtime.Bridge == nil {
		t.Fatal("initialize runtime.bridge = nil, want bridge runtime metadata")
	}
	managed, ok := handshake.Request.Runtime.Bridge.ManagedInstance(bridgeID)
	if !ok || managed == nil {
		t.Fatalf("initialize runtime missing managed instance %q", bridgeID)
	}
	if got, want := managed.Instance.ExtensionName, "telegram-reference"; got != want {
		t.Fatalf("managed.Instance.ExtensionName = %q, want %q", got, want)
	}
	if len(managed.BoundSecrets) == 0 {
		t.Fatal("managed.BoundSecrets = empty, want resolved bot_token binding")
	}
	if got, want := managed.BoundSecrets[0].BindingName, "bot_token"; got != want {
		t.Fatalf("managed.BoundSecrets[0].BindingName = %q, want %q", got, want)
	}

	states := extensiontest.WaitForStateMarkers(t, markers, 15*time.Second, func(records []extensiontest.StateRecord) bool {
		return len(records) > 0 && records[len(records)-1].Status.Normalize() == bridgepkg.BridgeStatusReady
	})
	if got, want := states[len(states)-1].BridgeInstanceID, bridgeID; got != want {
		t.Fatalf("last state bridge_instance_id = %q, want %q", got, want)
	}

	waitForRuntimeCondition(t, "telegram extension active", 10*time.Second, func() bool {
		ext, err := harness.GetExtension(ctx, "telegram-reference")
		return err == nil && ext.State == "active" && ext.Health == "healthy"
	})

	waitForRuntimeCondition(t, "bridge ready before ingress", 10*time.Second, func() bool {
		bridge, err := harness.GetBridge(ctx, bridgeID)
		return err == nil &&
			bridge.Health.Status.Normalize() == bridgepkg.BridgeStatusReady &&
			bridge.Health.RouteCount == 0
	})

	extensiontest.AppendInboundUpdateMarker(
		t,
		markers,
		telegramRuntimeInboundUpdate(time.Now().UTC(), 9001, 321, "Need a runtime bridge summary"),
	)

	firstIngests := extensiontest.WaitForIngestMarkers(t, markers, 15*time.Second, func(records []extensiontest.IngestRecord) bool {
		return len(records) >= 1 && strings.TrimSpace(records[0].Result.SessionID) != ""
	})
	firstIngest := firstIngests[0]
	sessionID = firstIngest.Result.SessionID

	firstHandling, err := classifyBridgeSessionHandling(firstIngest.Result, "")
	if err != nil {
		t.Fatalf("classifyBridgeSessionHandling(first) error = %v", err)
	}
	if got, want := firstHandling, bridgeSessionHandlingCreated; got != want {
		t.Fatalf("first bridge session handling = %q, want %q", got, want)
	}

	firstDeliveries := extensiontest.WaitForDeliveryMarkers(t, markers, 15*time.Second, func(records []extensiontest.DeliveryRecord) bool {
		return countDeliveryEvents(records, bridgepkg.DeliveryEventTypeFinal) >= 1
	})
	if !hasDeliveryEventType(firstDeliveries, bridgepkg.DeliveryEventTypeStart) {
		t.Fatalf("first deliveries = %#v, want start event", firstDeliveries)
	}

	waitForRuntimeCondition(t, "first bridge transcript", 15*time.Second, func() bool {
		return sessionTranscriptHasNeedle(ctx, harness, sessionID, "Need a runtime bridge summary") &&
			sessionTranscriptHasNeedle(ctx, harness, sessionID, "Bridge summary: initial route handled.")
	})

	waitForRuntimeCondition(t, "bridge route created after first ingress", 15*time.Second, func() bool {
		bridge, err := harness.GetBridge(ctx, bridgeID)
		return err == nil &&
			bridge.Health.RouteCount == 1 &&
			bridge.Health.LastSuccessAt != nil
	})

	extensiontest.AppendInboundUpdateMarker(
		t,
		markers,
		telegramRuntimeInboundUpdate(time.Now().UTC(), 9002, 322, "Provide a follow-up runtime bridge summary"),
	)

	ingests := extensiontest.WaitForIngestMarkers(t, markers, 15*time.Second, func(records []extensiontest.IngestRecord) bool {
		return len(records) >= 2 && strings.TrimSpace(records[len(records)-1].Result.SessionID) != ""
	})
	secondIngest := ingests[len(ingests)-1]

	secondHandling, err := classifyBridgeSessionHandling(secondIngest.Result, firstIngest.Result.SessionID)
	if err != nil {
		t.Fatalf("classifyBridgeSessionHandling(second) error = %v", err)
	}
	if got, want := secondHandling, bridgeSessionHandlingReused; got != want {
		t.Fatalf("second bridge session handling = %q, want %q", got, want)
	}
	if got, want := secondIngest.Result.SessionID, firstIngest.Result.SessionID; got != want {
		t.Fatalf("second ingest session_id = %q, want reused %q", got, want)
	}
	if got, want := secondIngest.Result.RoutingKey, firstIngest.Result.RoutingKey; got != want {
		t.Fatalf("second ingest routing_key = %#v, want %#v", got, want)
	}

	deliveries := extensiontest.WaitForDeliveryMarkers(t, markers, 15*time.Second, func(records []extensiontest.DeliveryRecord) bool {
		return countDeliveryEvents(records, bridgepkg.DeliveryEventTypeFinal) >= 2
	})
	if got, want := uniqueDeliveryCount(deliveries), 2; got < want {
		t.Fatalf("unique delivery IDs = %d, want at least %d", got, want)
	}
	if !hasDeliveryEventType(deliveries, bridgepkg.DeliveryEventTypeStart) {
		t.Fatalf("deliveries = %#v, want start event", deliveries)
	}
	if !hasDeliveryEventType(deliveries, bridgepkg.DeliveryEventTypeFinal) {
		t.Fatalf("deliveries = %#v, want final event", deliveries)
	}

	routes, err := harness.ListBridgeRoutes(ctx, bridgeID)
	if err != nil {
		t.Fatalf("ListBridgeRoutes(%q) error = %v", bridgeID, err)
	}
	if got, want := len(routes), 1; got != want {
		t.Fatalf("len(routes) = %d, want %d", got, want)
	}
	route := routes[0]
	if got, want := route.SessionID, firstIngest.Result.SessionID; got != want {
		t.Fatalf("route.SessionID = %q, want %q", got, want)
	}
	if got, want := route.AgentName, bridgeIngressFixtureAgentName; got != want {
		t.Fatalf("route.AgentName = %q, want %q", got, want)
	}
	if route.RoutingKey() != firstIngest.Result.RoutingKey {
		t.Fatalf("route.RoutingKey() = %#v, want %#v", route.RoutingKey(), firstIngest.Result.RoutingKey)
	}

	bindings, err := harness.ListBridgeSecretBindings(ctx, bridgeID)
	if err != nil {
		t.Fatalf("ListBridgeSecretBindings(%q) error = %v", bridgeID, err)
	}
	if got, want := len(bindings), 1; got != want {
		t.Fatalf("len(bindings) = %d, want %d", got, want)
	}
	if got, want := bindings[0].VaultRef, "env:AGH_TEST_TELEGRAM_TOKEN"; got != want {
		t.Fatalf("bindings[0].VaultRef = %q, want %q", got, want)
	}

	bridge, err := harness.GetBridge(ctx, bridgeID)
	if err != nil {
		t.Fatalf("GetBridge(%q) error = %v", bridgeID, err)
	}
	if got, want := bridge.Health.RouteCount, 1; got != want {
		t.Fatalf("bridge.Health.RouteCount = %d, want %d", got, want)
	}
	if bridge.Health.LastSuccessAt == nil {
		t.Fatal("bridge.Health.LastSuccessAt = nil, want successful delivery progression")
	}

	transcript := mustSessionTranscript(t, ctx, harness, sessionID)
	transcriptContent := joinTranscriptContent(transcript.Messages)
	for _, needle := range []string{
		"Need a runtime bridge summary",
		"Provide a follow-up runtime bridge summary",
		"Bridge summary: initial route handled.",
		"Bridge summary: follow-up reused the existing session.",
	} {
		if !strings.Contains(transcriptContent, needle) {
			t.Fatalf("transcript = %q, want %q", transcriptContent, needle)
		}
	}

	report := extensiontest.ReportFromMarkers(t, markers)
	if err := extensiontest.ValidateConformance(report, extensiontest.ConformanceExpectation{
		Provider:                  "telegram-reference",
		Platform:                  "telegram",
		RequireOwnedInstanceList:  true,
		RequireOwnedInstanceFetch: true,
		RequireStateReport:        true,
		RequireDelivery:           true,
		ManagedInstances: []extensiontest.ManagedInstanceExpectation{{
			InstanceID:          bridgeID,
			ExtensionName:       "telegram-reference",
			BoundSecretNames:    []string{"bot_token"},
			ExpectedFinalStatus: bridgepkg.BridgeStatusReady,
		}},
	}); err != nil {
		t.Fatalf("ValidateConformance() error = %v", err)
	}

	if report.Ownership == nil {
		t.Fatal("report.Ownership = nil, want provider ownership markers")
	}
	if got, want := len(report.Ownership.Listed), 1; got != want {
		t.Fatalf("len(report.Ownership.Listed) = %d, want %d", got, want)
	}
	if got, want := report.Ownership.Listed[0].ID, bridgeID; got != want {
		t.Fatalf("report.Ownership.Listed[0].ID = %q, want %q", got, want)
	}
	if got, want := len(report.Ownership.Fetched), 1; got != want {
		t.Fatalf("len(report.Ownership.Fetched) = %d, want %d", got, want)
	}
	if got, want := report.Ownership.Fetched[0].ID, bridgeID; got != want {
		t.Fatalf("report.Ownership.Fetched[0].ID = %q, want %q", got, want)
	}
	if got, want := len(report.Ingests), 2; got < want {
		t.Fatalf("len(report.Ingests) = %d, want at least %d", got, want)
	}
	if report.Ingests[0].Result.SessionID != route.SessionID || report.Ingests[len(report.Ingests)-1].Result.SessionID != route.SessionID {
		t.Fatalf("ingest session IDs = %#v, want route session %q", report.Ingests, route.SessionID)
	}
	if got, want := uniqueDeliveryCount(report.Deliveries), 2; got < want {
		t.Fatalf("unique delivery IDs = %d, want at least %d", got, want)
	}
	if !hasDeliveryEventType(report.Deliveries, bridgepkg.DeliveryEventTypeStart) {
		t.Fatalf("report.Deliveries = %#v, want start event", report.Deliveries)
	}
	if !hasDeliveryEventType(report.Deliveries, bridgepkg.DeliveryEventTypeFinal) {
		t.Fatalf("report.Deliveries = %#v, want final event", report.Deliveries)
	}
	if report.Deliveries[len(report.Deliveries)-1].Request.Event.BridgeInstanceID != bridgeID {
		t.Fatalf(
			"last delivery bridge_instance_id = %q, want %q",
			report.Deliveries[len(report.Deliveries)-1].Request.Event.BridgeInstanceID,
			bridgeID,
		)
	}
}

func registerBridgeExtensionArtifacts(
	t testing.TB,
	harness *e2etest.RuntimeHarness,
	markers extensiontest.MarkerPaths,
	registration acpmock.Registration,
	bridgeID *string,
	sessionID *string,
) {
	t.Helper()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := harness.CaptureBridgeHealth(ctx); err != nil {
			t.Logf("CaptureBridgeHealth() error = %v", err)
		}

		trimmedBridgeID := strings.TrimSpace(derefStringValue(bridgeID))
		if trimmedBridgeID != "" {
			if err := harness.CaptureBridgeRoutes(ctx, trimmedBridgeID); err != nil {
				t.Logf("CaptureBridgeRoutes(%q) error = %v", trimmedBridgeID, err)
			}
			if err := harness.CaptureBridgeDeliveryState(ctx, trimmedBridgeID); err != nil {
				t.Logf("CaptureBridgeDeliveryState(%q) error = %v", trimmedBridgeID, err)
			}
			if err := harness.CaptureBridgeSecretBindings(ctx, trimmedBridgeID); err != nil {
				t.Logf("CaptureBridgeSecretBindings(%q) error = %v", trimmedBridgeID, err)
			}
		}

		trimmedSessionID := strings.TrimSpace(derefStringValue(sessionID))
		if trimmedSessionID != "" {
			if err := harness.CaptureSessionTranscript(ctx, trimmedSessionID); err != nil {
				t.Logf("CaptureSessionTranscript(%q) error = %v", trimmedSessionID, err)
			}
			if err := harness.CaptureSessionEvents(ctx, trimmedSessionID); err != nil {
				t.Logf("CaptureSessionEvents(%q) error = %v", trimmedSessionID, err)
			}
			if err := harness.CaptureSessionEnvironment(ctx, trimmedSessionID); err != nil {
				t.Logf("CaptureSessionEnvironment(%q) error = %v", trimmedSessionID, err)
			}
		}

		payload := map[string]any{
			"provider_markers": extensiontest.ReportFromMarkers(t, markers),
		}
		if records, err := acpmock.ReadDiagnostics(registration.DiagnosticsPath); err == nil {
			payload["mock_agents"] = map[string]any{
				registration.AgentName: records,
			}
		} else {
			t.Logf("ReadDiagnostics(%q) error = %v", registration.AgentName, err)
		}
		if err := harness.CaptureProviderCallsJSON(payload); err != nil {
			t.Logf("CaptureProviderCallsJSON() error = %v", err)
		}
	})
}

func daemonBridgeRuntimeRepoRoot(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}

func daemonTelegramReferenceExtensionDir(repoRoot string) string {
	return filepath.Join(repoRoot, "sdk", "examples", "telegram-reference")
}

func prepareDaemonTelegramReferenceExtension(t *testing.T, repoRoot string) string {
	t.Helper()

	sourceDir := daemonTelegramReferenceExtensionDir(repoRoot)
	targetDir := filepath.Join(t.TempDir(), "telegram-reference")
	if err := copyDirectory(sourceDir, targetDir); err != nil {
		t.Fatalf("copyDirectory(%q, %q) error = %v", sourceDir, targetDir, err)
	}
	buildDaemonTelegramReferenceAdapter(t, repoRoot, targetDir)
	return targetDir
}

func buildDaemonTelegramReferenceAdapter(t *testing.T, repoRoot string, extensionDir string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		"go",
		"build",
		"-o", filepath.Join(extensionDir, "bin", "telegram-reference"),
		"./sdk/examples/telegram-reference",
	)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf(
			"go build telegram-reference from %q into %q: %v\n%s",
			repoRoot,
			extensionDir,
			err,
			string(output),
		)
	}
}

func copyDirectory(sourceDir string, targetDir string) error {
	return filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relativePath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("rel %q from %q: %w", path, sourceDir, err)
		}
		targetPath := filepath.Join(targetDir, relativePath)

		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("stat %q: %w", path, err)
		}

		if entry.IsDir() {
			return os.MkdirAll(targetPath, info.Mode().Perm())
		}

		bytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %q: %w", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("mkdir %q: %w", filepath.Dir(targetPath), err)
		}
		if err := os.WriteFile(targetPath, bytes, info.Mode().Perm()); err != nil {
			return fmt.Errorf("write %q: %w", targetPath, err)
		}
		return nil
	})
}

func telegramRuntimeInboundUpdate(now time.Time, updateID int64, messageID int64, text string) map[string]any {
	return map[string]any{
		"update_id": updateID,
		"message": map[string]any{
			"message_id":        messageID,
			"message_thread_id": 654,
			"date":              now.Unix(),
			"chat": map[string]any{
				"id":    777,
				"type":  "supergroup",
				"title": "ops",
			},
			"from": map[string]any{
				"id":         888,
				"username":   "alice",
				"first_name": "Alice",
				"last_name":  "Example",
			},
			"text": text,
		},
	}
}

func countDeliveryEvents(records []extensiontest.DeliveryRecord, want string) int {
	total := 0
	for _, record := range records {
		if normalizeBridgeDeliveryEventType(record.Request.Event.EventType) == normalizeBridgeDeliveryEventType(want) {
			total++
		}
	}
	return total
}

func hasDeliveryEventType(records []extensiontest.DeliveryRecord, want string) bool {
	return countDeliveryEvents(records, want) > 0
}

func uniqueDeliveryCount(records []extensiontest.DeliveryRecord) int {
	ids := make(map[string]struct{}, len(records))
	for _, record := range records {
		deliveryID := strings.TrimSpace(record.Request.Event.DeliveryID)
		if deliveryID == "" {
			continue
		}
		ids[deliveryID] = struct{}{}
	}
	return len(ids)
}

func normalizeBridgeDeliveryEventType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func derefStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
