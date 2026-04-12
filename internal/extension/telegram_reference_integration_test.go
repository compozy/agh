//go:build integration

package extension_test

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	channelspkg "github.com/pedronauck/agh/internal/channels"
	extensiontest "github.com/pedronauck/agh/internal/extensiontest"
	observepkg "github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/subprocess"
)

var (
	buildTelegramReferenceOnce sync.Once
	buildTelegramReferenceErr  error
)

func TestTelegramReferenceAdapterLaunchNegotiatesChannelRuntime(t *testing.T) {
	repoRoot := telegramReferenceRepoRoot(t)
	buildTelegramReferenceAdapter(t, repoRoot)

	harness := extensiontest.NewHarness(t, extensiontest.HarnessConfig{
		ExtensionDir: telegramReferenceExtensionDir(repoRoot),
		BoundSecrets: []subprocess.InitializeChannelBoundSecret{
			{BindingName: "bot_token", Kind: "token", Value: "telegram-bot-token"},
		},
		StartTime: time.Date(2026, 4, 11, 6, 0, 0, 0, time.UTC),
	})

	handshake := harness.WaitForHandshake(t, 10*time.Second)
	states := harness.WaitForStates(t, 10*time.Second, func(states []extensiontest.StateRecord) bool {
		return len(states) > 0
	})
	if got, want := states[len(states)-1].Status.Normalize(), channelspkg.ChannelStatusReady; got != want {
		t.Fatalf("last adapter state = %q (error=%q), want %q", got, states[len(states)-1].Error, want)
	}
	report := harness.Report(t)

	if err := extensiontest.ValidateConformance(report, extensiontest.ConformanceExpectation{
		InstanceID:          harness.Instance.ID,
		ExtensionName:       "telegram-reference",
		BoundSecretNames:    []string{"bot_token"},
		RequireStateReport:  true,
		ExpectedFinalStatus: channelspkg.ChannelStatusReady,
	}); err != nil {
		t.Fatalf("ValidateConformance() error = %v", err)
	}

	if handshake.Request.Runtime.Channel == nil {
		t.Fatal("initialize runtime.channel = nil, want bound channel launch metadata")
	}
	if got, want := handshake.Request.Runtime.Channel.Instance.ID, harness.Instance.ID; got != want {
		t.Fatalf("initialize runtime channel instance = %q, want %q", got, want)
	}
	if got, want := handshake.Request.Runtime.Channel.Instance.ExtensionName, "telegram-reference"; got != want {
		t.Fatalf("initialize runtime channel extension = %q, want %q", got, want)
	}
	if got, want := strings.TrimSpace(handshake.Request.Runtime.Channel.BoundSecrets[0].Value), "telegram-bot-token"; got != want {
		t.Fatalf("initialize bound bot token = %q, want %q", got, want)
	}
	if report.Instance == nil {
		t.Fatal("instance marker = nil, want channel instance metadata")
	}
	if got, want := report.Instance.ID, harness.Instance.ID; got != want {
		t.Fatalf("instance marker id = %q, want %q", got, want)
	}

	row := waitForChannelHealth(t, 10*time.Second, harness, func(health observepkg.ChannelInstanceHealth) bool {
		return health.Status.Normalize() == channelspkg.ChannelStatusReady
	})
	if got, want := row.RouteCount, 0; got != want {
		t.Fatalf("channel health route_count = %d, want %d before ingress", got, want)
	}

	health := harness.ObserveHealth(t)
	if got, want := health.Channels.StatusCounts.Ready, 1; got != want {
		t.Fatalf("observe.Health().Channels.StatusCounts.Ready = %d, want %d", got, want)
	}
}

func TestTelegramReferenceAdapterIngressAndDeliveryConformance(t *testing.T) {
	repoRoot := telegramReferenceRepoRoot(t)
	buildTelegramReferenceAdapter(t, repoRoot)

	startTime := time.Date(2026, 4, 11, 6, 5, 0, 0, time.UTC)
	harness := extensiontest.NewHarness(t, extensiontest.HarnessConfig{
		ExtensionDir: telegramReferenceExtensionDir(repoRoot),
		BoundSecrets: []subprocess.InitializeChannelBoundSecret{
			{BindingName: "bot_token", Kind: "token", Value: "telegram-bot-token"},
		},
		Driver: extensiontest.NewScriptedPromptDriver(startTime, []extensiontest.ScriptedPromptEvent{
			{Type: acp.EventTypeAgentMessage, Text: "hello"},
			{Type: acp.EventTypeAgentMessage, Text: " world"},
			{Type: acp.EventTypeDone},
		}),
		StartTime: startTime,
	})

	harness.WaitForHandshake(t, 10*time.Second)
	states := harness.WaitForStates(t, 10*time.Second, func(states []extensiontest.StateRecord) bool {
		return len(states) > 0
	})
	if got, want := states[len(states)-1].Status.Normalize(), channelspkg.ChannelStatusReady; got != want {
		t.Fatalf("last adapter state = %q (error=%q), want %q", got, states[len(states)-1].Error, want)
	}
	harness.AppendInboundUpdate(t, telegramInboundUpdate(startTime))

	ingests := harness.WaitForIngests(t, 10*time.Second, func(records []extensiontest.IngestRecord) bool {
		return len(records) > 0 && strings.TrimSpace(records[len(records)-1].Result.SessionID) != ""
	})
	deliveries := harness.WaitForDeliveries(t, 10*time.Second, func(records []extensiontest.DeliveryRecord) bool {
		return len(records) > 0 && normalizeDeliveryEventType(records[len(records)-1].Request.Event.EventType) == channelspkg.DeliveryEventTypeFinal
	})
	report := harness.Report(t)

	if err := extensiontest.ValidateConformance(report, extensiontest.ConformanceExpectation{
		InstanceID:          harness.Instance.ID,
		ExtensionName:       "telegram-reference",
		BoundSecretNames:    []string{"bot_token"},
		RequireStateReport:  true,
		RequireDelivery:     true,
		ExpectedFinalStatus: channelspkg.ChannelStatusReady,
	}); err != nil {
		t.Fatalf("ValidateConformance() error = %v", err)
	}

	if got, want := len(ingests), 1; got != want {
		t.Fatalf("len(ingests) = %d, want %d", got, want)
	}
	if got, want := ingests[0].Envelope.Content.Text, "Need a summary"; got != want {
		t.Fatalf("ingest envelope text = %q, want %q", got, want)
	}
	if len(deliveries) < 2 {
		t.Fatalf("len(deliveries) = %d, want at least 2", len(deliveries))
	}
	if got, want := normalizeDeliveryEventType(deliveries[0].Request.Event.EventType), channelspkg.DeliveryEventTypeStart; got != want {
		t.Fatalf("first delivery event type = %q, want %q", got, want)
	}
	if got, want := normalizeDeliveryEventType(deliveries[len(deliveries)-1].Request.Event.EventType), channelspkg.DeliveryEventTypeFinal; got != want {
		t.Fatalf("last delivery event type = %q, want %q", got, want)
	}

	row := waitForChannelHealth(t, 10*time.Second, harness, func(health observepkg.ChannelInstanceHealth) bool {
		return health.Status.Normalize() == channelspkg.ChannelStatusReady && health.RouteCount == 1
	})
	if got, want := row.RouteCount, 1; got != want {
		t.Fatalf("channel health route_count = %d, want %d", got, want)
	}
}

func TestTelegramReferenceAdapterRestartResumesActiveDelivery(t *testing.T) {
	repoRoot := telegramReferenceRepoRoot(t)
	buildTelegramReferenceAdapter(t, repoRoot)

	startTime := time.Date(2026, 4, 11, 6, 10, 0, 0, time.UTC)
	harness := extensiontest.NewHarness(t, extensiontest.HarnessConfig{
		ExtensionDir: telegramReferenceExtensionDir(repoRoot),
		BoundSecrets: []subprocess.InitializeChannelBoundSecret{
			{BindingName: "bot_token", Kind: "token", Value: "telegram-bot-token"},
		},
		Driver: extensiontest.NewScriptedPromptDriver(startTime, []extensiontest.ScriptedPromptEvent{
			{Type: acp.EventTypeAgentMessage, Text: "hello"},
			{Type: acp.EventTypeDone},
		}),
		StartTime:                startTime,
		CrashOnceOnFirstDelivery: true,
		BrokerOptions: []channelspkg.DeliveryBrokerOption{
			channelspkg.WithDeliveryBrokerRetryDelay(20 * time.Millisecond),
		},
	})

	harness.WaitForHandshake(t, 10*time.Second)
	states := harness.WaitForStates(t, 10*time.Second, func(states []extensiontest.StateRecord) bool {
		return len(states) > 0
	})
	if got, want := states[len(states)-1].Status.Normalize(), channelspkg.ChannelStatusReady; got != want {
		t.Fatalf("last adapter state = %q (error=%q), want %q", got, states[len(states)-1].Error, want)
	}
	harness.AppendInboundUpdate(t, telegramInboundUpdate(startTime))

	deliveries := harness.WaitForDeliveries(t, 10*time.Second, func(records []extensiontest.DeliveryRecord) bool {
		for _, record := range records {
			if normalizeDeliveryEventType(record.Request.Event.EventType) == channelspkg.DeliveryEventTypeResume {
				return true
			}
		}
		return false
	})
	report := harness.Report(t)

	if err := extensiontest.ValidateConformance(report, extensiontest.ConformanceExpectation{
		InstanceID:          harness.Instance.ID,
		ExtensionName:       "telegram-reference",
		BoundSecretNames:    []string{"bot_token"},
		RequireStateReport:  true,
		RequireDelivery:     true,
		RequireResume:       true,
		ExpectedFinalStatus: channelspkg.ChannelStatusReady,
	}); err != nil {
		t.Fatalf("ValidateConformance() error = %v", err)
	}

	if len(deliveries) < 2 {
		t.Fatalf("len(deliveries) = %d, want at least 2", len(deliveries))
	}
	resume := findDeliveryRecord(t, deliveries, channelspkg.DeliveryEventTypeResume)
	if resume.Request.Snapshot == nil {
		t.Fatal("resume delivery snapshot = nil, want resumable state")
	}
	if resume.PID == deliveries[0].PID {
		t.Fatalf("resume pid = %d, want a restarted adapter process different from %d", resume.PID, deliveries[0].PID)
	}
}

func TestTelegramReferenceAdapterAuthRequiredHealthSurface(t *testing.T) {
	repoRoot := telegramReferenceRepoRoot(t)
	buildTelegramReferenceAdapter(t, repoRoot)

	startTime := time.Date(2026, 4, 11, 6, 15, 0, 0, time.UTC)
	harness := extensiontest.NewHarness(t, extensiontest.HarnessConfig{
		ExtensionDir: telegramReferenceExtensionDir(repoRoot),
		StartTime:    startTime,
	})

	harness.WaitForHandshake(t, 10*time.Second)
	states := harness.WaitForStates(t, 10*time.Second, func(states []extensiontest.StateRecord) bool {
		return len(states) > 0
	})
	if got, want := states[len(states)-1].Status.Normalize(), channelspkg.ChannelStatusAuthRequired; got != want {
		t.Fatalf("last adapter state = %q (error=%q), want %q", got, states[len(states)-1].Error, want)
	}
	report := harness.Report(t)

	if err := extensiontest.ValidateConformance(report, extensiontest.ConformanceExpectation{
		InstanceID:          harness.Instance.ID,
		ExtensionName:       "telegram-reference",
		RequireStateReport:  true,
		ExpectedFinalStatus: channelspkg.ChannelStatusAuthRequired,
	}); err != nil {
		t.Fatalf("ValidateConformance() error = %v", err)
	}

	row := waitForChannelHealth(t, 10*time.Second, harness, func(health observepkg.ChannelInstanceHealth) bool {
		return health.Status.Normalize() == channelspkg.ChannelStatusAuthRequired && health.AuthFailuresTotal > 0
	})
	if row.AuthFailuresTotal <= 0 {
		t.Fatalf("channel auth_failures_total = %d, want > 0", row.AuthFailuresTotal)
	}

	health := harness.ObserveHealth(t)
	if got, want := health.Channels.StatusCounts.AuthRequired, 1; got != want {
		t.Fatalf("observe.Health().Channels.StatusCounts.AuthRequired = %d, want %d", got, want)
	}
	if health.Channels.AuthFailuresTotal <= 0 {
		t.Fatalf("observe.Health().Channels.AuthFailuresTotal = %d, want > 0", health.Channels.AuthFailuresTotal)
	}
}

func telegramReferenceRepoRoot(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}

func telegramReferenceExtensionDir(repoRoot string) string {
	return filepath.Join(repoRoot, "sdk", "examples", "telegram-reference")
}

func buildTelegramReferenceAdapter(t *testing.T, repoRoot string) {
	t.Helper()

	buildTelegramReferenceOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		cmd := exec.CommandContext(
			ctx,
			"go",
			"build",
			"-o",
			"./sdk/examples/telegram-reference/bin/telegram-reference",
			"./sdk/examples/telegram-reference",
		)
		cmd.Dir = repoRoot
		output, err := cmd.CombinedOutput()
		if err != nil {
			buildTelegramReferenceErr = fmt.Errorf("go build telegram-reference: %w\n%s", err, string(output))
		}
	})
	if buildTelegramReferenceErr != nil {
		t.Fatal(buildTelegramReferenceErr)
	}
}

func telegramInboundUpdate(now time.Time) map[string]any {
	return map[string]any{
		"update_id": 9001,
		"message": map[string]any{
			"message_id":        321,
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
			"text": "Need a summary",
		},
	}
}

func waitForChannelHealth(
	t *testing.T,
	timeout time.Duration,
	harness *extensiontest.Harness,
	predicate func(observepkg.ChannelInstanceHealth) bool,
) observepkg.ChannelInstanceHealth {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		rows := harness.QueryChannelHealth(t)
		for _, row := range rows {
			if row.ChannelInstanceID == harness.Instance.ID && predicate(row) {
				return row
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("channel health for %q did not satisfy predicate before timeout", harness.Instance.ID)
	return observepkg.ChannelInstanceHealth{}
}

func findDeliveryRecord(
	t *testing.T,
	records []extensiontest.DeliveryRecord,
	eventType string,
) extensiontest.DeliveryRecord {
	t.Helper()

	want := normalizeDeliveryEventType(eventType)
	for _, record := range records {
		if normalizeDeliveryEventType(record.Request.Event.EventType) == want {
			return record
		}
	}
	t.Fatalf("delivery records did not contain event type %q", eventType)
	return extensiontest.DeliveryRecord{}
}

func normalizeDeliveryEventType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
