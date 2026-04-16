//go:build integration

package daemon

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pedronauck/agh/internal/environment"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestDaemonEnvironmentReconcileIntegrationBootFinalizeReattachesBeforeObserverReconcile(t *testing.T) {
	daemon, state, provider, _ := newEnvironmentReconcileHarness(t)
	writeEnvironmentReconcileMeta(t, daemon, environmentReconcileMeta{
		id:     "sess-crashed-active",
		state:  session.StateActive,
		env:    remoteMeta("env-crashed-active", environment.BackendDaytona, "daytona", "sandbox-crashed-active"),
		agent:  "coder",
		worker: "ws-crashed-active",
	})

	resourceReconcile := &fakeResourceReconcileDriver{
		onRunBoot: func() {
			if got := len(provider.prepareRequests); got != 0 {
				t.Fatalf("Prepare calls during resource RunBoot = %d, want 0", got)
			}
		},
	}
	observer := &fakeObserver{
		onReconcile: func() {
			if got := len(provider.prepareRequests); got != 1 {
				t.Fatalf("Prepare calls before observer Reconcile = %d, want 1", got)
			}
		},
	}
	state.resourceReconcile = resourceReconcile
	state.observer = observer

	if err := daemon.bootFinalize(testutil.Context(t), state); err != nil {
		t.Fatalf("bootFinalize() error = %v", err)
	}

	if resourceReconcile.runBootCalls != 1 {
		t.Fatalf("resource RunBoot calls = %d, want 1", resourceReconcile.runBootCalls)
	}
	if !observer.reconciled {
		t.Fatal("observer Reconcile was not called")
	}
	req := provider.prepareRequests[0]
	if req.EnvironmentID != "env-crashed-active" ||
		req.InstanceID != "sandbox-crashed-active" ||
		string(req.ProviderState) == "" {
		t.Fatalf("PrepareRequest = %#v, want persisted environment identity and state", req)
	}
}

func TestDaemonEnvironmentReconcileIntegrationPartialCreateFoundByEnvironmentID(t *testing.T) {
	daemon, state, provider, _ := newEnvironmentReconcileHarness(t)
	provider.findState = environment.SessionState{
		EnvironmentID: "env-timeout",
		Backend:       environment.BackendDaytona,
		Profile:       "daytona",
		State:         "found",
		InstanceID:    "sandbox-timeout",
		ProviderState: json.RawMessage(`{"sandbox":"timeout"}`),
	}
	writeEnvironmentReconcileMeta(t, daemon, environmentReconcileMeta{
		id:     "sess-timeout",
		state:  session.StateActive,
		env:    remoteMeta("env-timeout", environment.BackendDaytona, "daytona", ""),
		agent:  "coder",
		worker: "ws-timeout",
	})

	daemon.reconcileDaemonEnvironments(testutil.Context(t), state)

	if got := provider.findRequests[0].Labels["agh_environment_id"]; got != "env-timeout" {
		t.Fatalf("agh_environment_id lookup = %q, want env-timeout", got)
	}
	if got := provider.prepareRequests[0].InstanceID; got != "sandbox-timeout" {
		t.Fatalf("PrepareRequest.InstanceID = %q, want sandbox-timeout", got)
	}
}

func TestDaemonEnvironmentReconcileIntegrationUnrecoverableSandboxDestroyLogged(t *testing.T) {
	daemon, state, provider, logs := newEnvironmentReconcileHarness(t)
	provider.prepareErr = errIntegrationReconcileProviderFailure{}
	writeEnvironmentReconcileMeta(t, daemon, environmentReconcileMeta{
		id:     "sess-unrecoverable",
		state:  session.StateActive,
		env:    remoteMeta("env-unrecoverable", environment.BackendDaytona, "daytona", "sandbox-unrecoverable"),
		agent:  "coder",
		worker: "ws-unrecoverable",
	})

	daemon.reconcileDaemonEnvironments(testutil.Context(t), state)

	if got := len(provider.destroyStates); got != 1 {
		t.Fatalf("Destroy calls = %d, want 1", got)
	}
	if !strings.Contains(logs.String(), "daemon: environment reattach failed") ||
		!strings.Contains(logs.String(), "daemon: environment destroy complete") {
		t.Fatalf("logs missing reattach failure and cleanup: %s", logs.String())
	}
}

func TestDaemonEnvironmentReconcileIntegrationStoppedRemoteSessionDoesNotReattach(t *testing.T) {
	daemon, state, provider, _ := newEnvironmentReconcileHarness(t)
	writeEnvironmentReconcileMeta(t, daemon, environmentReconcileMeta{
		id:     "sess-stopped-integration",
		state:  session.StateStopped,
		env:    remoteMeta("env-stopped-integration", environment.BackendDaytona, "daytona", "sandbox-stopped-integration"),
		agent:  "coder",
		worker: "ws-stopped-integration",
	})

	daemon.reconcileDaemonEnvironments(testutil.Context(t), state)

	if got := len(provider.prepareRequests); got != 0 {
		t.Fatalf("Prepare calls = %d, want 0", got)
	}
	if got := len(provider.destroyStates); got != 1 {
		t.Fatalf("Destroy calls = %d, want 1 cleanup attempt", got)
	}
}

type errIntegrationReconcileProviderFailure struct{}

func (errIntegrationReconcileProviderFailure) Error() string {
	return "reattach failed"
}
