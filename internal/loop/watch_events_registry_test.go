package loop_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/compozy/agh/internal/hooks"
	"github.com/compozy/agh/internal/loop"
)

const (
	watchEventsTestTaskStream       = "task_events"
	watchEventsTestLoopStream       = "loop_run_events"
	watchEventsTestAutomationStream = "automation_runs"
	watchEventsTestNetworkStream    = "network_timeline_log"

	loopRunEventTestStatusChanged = "status_changed"
	loopRunEventTestNodeSucceeded = "node_succeeded"
	loopRunEventTestNodeFailed    = "node_failed"
)

func TestSupportedWatchEventsShouldExposeSupportedContracts(t *testing.T) {
	t.Parallel()

	t.Run("Should expose exactly the shipped post-state watch event contracts", func(t *testing.T) {
		t.Parallel()

		contracts := loop.SupportedWatchEvents()
		if len(contracts) != 17 {
			t.Fatalf("SupportedWatchEvents() len = %d, want 17", len(contracts))
		}
		expected := map[hooks.HookEvent]struct {
			stream      string
			ledgerTypes []string
		}{
			hooks.HookTaskStatusChanged: {
				stream:      watchEventsTestTaskStream,
				ledgerTypes: []string{string(hooks.HookTaskStatusChanged)},
			},
			hooks.HookTaskBlocked: {
				stream:      watchEventsTestTaskStream,
				ledgerTypes: []string{string(hooks.HookTaskBlocked)},
			},
			hooks.HookTaskUnblocked: {
				stream:      watchEventsTestTaskStream,
				ledgerTypes: []string{string(hooks.HookTaskUnblocked)},
			},
			hooks.HookTaskNeedsAttention: {
				stream:      watchEventsTestTaskStream,
				ledgerTypes: []string{string(hooks.HookTaskNeedsAttention)},
			},
			hooks.HookTaskRecovered: {
				stream:      watchEventsTestTaskStream,
				ledgerTypes: []string{string(hooks.HookTaskRecovered)},
			},
			hooks.HookTaskRunCompleted: {
				stream:      watchEventsTestTaskStream,
				ledgerTypes: []string{string(hooks.HookTaskRunCompleted)},
			},
			hooks.HookTaskRunFailed: {
				stream:      watchEventsTestTaskStream,
				ledgerTypes: []string{string(hooks.HookTaskRunFailed)},
			},
			hooks.HookLoopTerminal: {
				stream:      watchEventsTestLoopStream,
				ledgerTypes: []string{loopRunEventTestStatusChanged},
			},
			hooks.HookLoopNodeTerminal: {
				stream:      watchEventsTestLoopStream,
				ledgerTypes: []string{loopRunEventTestNodeSucceeded, loopRunEventTestNodeFailed},
			},
			hooks.HookAutomationRunCompleted: {
				stream:      watchEventsTestAutomationStream,
				ledgerTypes: []string{string(hooks.HookAutomationRunCompleted)},
			},
			hooks.HookAutomationRunFailed: {
				stream:      watchEventsTestAutomationStream,
				ledgerTypes: []string{string(hooks.HookAutomationRunFailed)},
			},
			hooks.HookNetworkMessagePersisted: {
				stream:      watchEventsTestNetworkStream,
				ledgerTypes: []string{string(hooks.HookNetworkMessagePersisted)},
			},
			hooks.HookNetworkThreadOpened: {
				stream:      watchEventsTestNetworkStream,
				ledgerTypes: []string{string(hooks.HookNetworkThreadOpened)},
			},
			hooks.HookNetworkDirectRoomOpened: {
				stream:      watchEventsTestNetworkStream,
				ledgerTypes: []string{string(hooks.HookNetworkDirectRoomOpened)},
			},
			hooks.HookNetworkWorkOpened: {
				stream:      watchEventsTestNetworkStream,
				ledgerTypes: []string{string(hooks.HookNetworkWorkOpened)},
			},
			hooks.HookNetworkWorkTransitioned: {
				stream:      watchEventsTestNetworkStream,
				ledgerTypes: []string{string(hooks.HookNetworkWorkTransitioned)},
			},
			hooks.HookNetworkWorkClosed: {
				stream:      watchEventsTestNetworkStream,
				ledgerTypes: []string{string(hooks.HookNetworkWorkClosed)},
			},
		}
		catalog := hookCatalogForTest()
		for kind, contract := range contracts {
			if _, ok := catalog[kind]; !ok {
				t.Fatalf("SupportedWatchEvents()[%q] is not in AllHookEvents()", kind)
			}
			if strings.Contains(string(kind), ".pre") {
				t.Fatalf("SupportedWatchEvents()[%q] is a pre-state hook", kind)
			}
			if len(contract.RequiredVars) != 0 {
				t.Fatalf(
					"SupportedWatchEvents()[%q].RequiredVars = %#v, want empty",
					kind,
					contract.RequiredVars,
				)
			}
			if len(contract.PayloadFields) == 0 {
				t.Fatalf("SupportedWatchEvents()[%q].PayloadFields is empty", kind)
			}
			want, ok := expected[kind]
			if !ok {
				t.Fatalf("SupportedWatchEvents() contains unexpected kind %q", kind)
			}
			if contract.Stream != want.stream {
				t.Fatalf(
					"SupportedWatchEvents()[%q].Stream = %q, want %q",
					kind,
					contract.Stream,
					want.stream,
				)
			}
			if !reflect.DeepEqual(contract.LedgerTypes, want.ledgerTypes) {
				t.Fatalf(
					"SupportedWatchEvents()[%q].LedgerTypes = %#v, want %#v",
					kind,
					contract.LedgerTypes,
					want.ledgerTypes,
				)
			}
		}
		if _, ok := contracts[hooks.HookAutomationJobPreFire]; ok {
			t.Fatal("SupportedWatchEvents() contains automation.job.pre_fire, want unsupported")
		}
		if _, ok := contracts[hooks.HookAutomationJobPostFire]; ok {
			t.Fatal("SupportedWatchEvents() contains automation.job.post_fire, want unsupported")
		}
		if _, ok := contracts[hooks.HookNetworkPeerJoined]; ok {
			t.Fatal("SupportedWatchEvents() contains network.peer.joined, want unsupported")
		}
		if _, ok := contracts[hooks.HookNetworkPeerLeft]; ok {
			t.Fatal("SupportedWatchEvents() contains network.peer.left, want unsupported")
		}
		statusChanged := contracts[hooks.HookTaskStatusChanged]
		if !slices.Contains(statusChanged.PayloadFields, "to_status") {
			t.Fatalf(
				"task.status_changed PayloadFields = %#v, want to_status",
				statusChanged.PayloadFields,
			)
		}
		nodeTerminal := contracts[hooks.HookLoopNodeTerminal]
		if !slices.Contains(nodeTerminal.LedgerTypes, loopRunEventTestNodeSucceeded) ||
			!slices.Contains(nodeTerminal.LedgerTypes, loopRunEventTestNodeFailed) {
			t.Fatalf(
				"loop.node.terminal LedgerTypes = %#v, want succeeded and failed",
				nodeTerminal.LedgerTypes,
			)
		}
		networkWork := contracts[hooks.HookNetworkWorkTransitioned]
		if !slices.Contains(networkWork.PayloadFields, "work_state") {
			t.Fatalf(
				"network.work.transitioned PayloadFields = %#v, want work_state",
				networkWork.PayloadFields,
			)
		}
	})
}

func hookCatalogForTest() map[hooks.HookEvent]struct{} {
	catalog := make(map[hooks.HookEvent]struct{}, len(hooks.AllHookEvents()))
	for _, event := range hooks.AllHookEvents() {
		catalog[event] = struct{}{}
	}
	return catalog
}
