package coordinator

import (
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/store"
)

var testBenchmarkTime = time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)

var (
	promptOverlaySink     string
	permissionPolicySink  store.SessionPermissionPolicy
	sessionLineagePtrSink *store.SessionLineage
)

func BenchmarkPromptOverlay(b *testing.B) {
	input := PromptInput{
		WorkspaceID:           "workspace-123",
		TaskID:                "task-456",
		RunID:                 "run-789",
		WorkflowID:            "workflow-012",
		CoordinationChannelID: "channel-345",
	}

	b.ReportAllocs()
	for b.Loop() {
		promptOverlaySink = PromptOverlay(input)
	}
}

func BenchmarkPermissionPolicy(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		permissionPolicySink = PermissionPolicy("channel-1", "channel-1", "channel-2", " ")
	}
}

func BenchmarkLineage(b *testing.B) {
	cfg := aghconfig.DefaultCoordinatorConfig()
	cfg.Enabled = true
	policy := PermissionPolicy("channel-1")

	b.ReportAllocs()
	for b.Loop() {
		sessionLineagePtrSink = Lineage(testBenchmarkTime, cfg, policy)
	}
}
