package session

import (
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/heartbeat"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/subprocess"
	"github.com/compozy/agh/internal/testutil"
)

func TestPromptActivitySupervisorHealthContract(t *testing.T) {
	t.Parallel()

	t.Run("Should persist unhealthy process diagnostics in session health", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
		healthStore := newFakeSessionHealthStore()
		h := newHarness(t,
			WithNow(func() time.Time { return now }),
			WithSessionHealthStore(healthStore),
		)
		session := createSession(t, h)
		t.Cleanup(func() {
			if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
				t.Errorf("Stop(%q) error = %v", session.ID, err)
			}
		})

		supervisor := newPromptActivitySupervisor(
			testutil.Context(t),
			h.manager,
			session,
			newPromptTurnDispatchState(session, "turn-unhealthy", TurnSourceUser, "hello"),
			testSupervisionConfig(),
		)
		supervisor.touch(now, runtimeActivityKindPromptStarted, "prompt started")

		proc := h.driver.lastProcess()
		if proc == nil {
			t.Fatal("lastProcess() = nil, want process")
		}
		proc.setHealth(subprocess.HealthState{
			Healthy:             false,
			LastCheckedAt:       now.Add(5 * time.Second),
			ConsecutiveFailures: 2,
			Message:             "provider health probe failed",
			LastError:           "health_check: context deadline exceeded",
		})

		supervisor.evaluate(now.Add(6 * time.Second))

		meta := readMeta(t, session.MetaPath())
		if meta.Liveness == nil {
			t.Fatal("meta.Liveness = nil, want stalled liveness")
		}
		if got, want := meta.Liveness.StallState, store.SessionStallStateDetected; got != want {
			t.Fatalf("meta.Liveness.StallState = %q, want %q", got, want)
		}

		health, err := h.manager.storedSessionHealth(testutil.Context(t), session.ID)
		if err != nil {
			t.Fatalf("storedSessionHealth() error = %v", err)
		}
		if health.State != heartbeat.SessionHealthStatePrompting || health.Health != heartbeat.SessionHealthDegraded {
			t.Fatalf("storedSessionHealth() = %#v, want prompting/degraded", health)
		}
		if !strings.Contains(health.LastError, "provider health probe failed") ||
			!strings.Contains(health.LastError, "health_check: context deadline exceeded") {
			t.Fatalf("storedSessionHealth().LastError = %q, want unhealthy process diagnostic", health.LastError)
		}
	})
}
