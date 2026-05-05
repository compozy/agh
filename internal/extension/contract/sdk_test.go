package contract

import (
	"testing"

	"github.com/pedronauck/agh/internal/hooks"
)

func TestHookContractsResolveDescriptors(t *testing.T) {
	t.Parallel()

	contracts := HookContracts()
	if got, want := len(contracts), len(hooks.AllEventDescriptors()); got != want {
		t.Fatalf("len(HookContracts()) = %d, want %d", got, want)
	}

	byEvent := make(map[hooks.HookEvent]HookContractSpec, len(contracts))
	for _, contract := range contracts {
		byEvent[contract.Event] = contract
	}
	for _, tc := range []struct {
		event   hooks.HookEvent
		payload string
		patch   string
	}{
		{
			event:   hooks.HookCoordinatorPreSpawn,
			payload: "CoordinatorPreSpawnPayload",
			patch:   "CoordinatorSpawnPatch",
		},
		{
			event:   hooks.HookTaskRunPreClaim,
			payload: "TaskRunPreClaimPayload",
			patch:   "TaskRunPreClaimPatch",
		},
		{
			event:   hooks.HookSpawnPreCreate,
			payload: "SpawnPreCreatePayload",
			patch:   "SpawnCreatePatch",
		},
		{
			event:   hooks.HookNetworkMessagePersisted,
			payload: "NetworkMessagePersistedPayload",
			patch:   "NetworkObservationPatch",
		},
		{
			event:   hooks.HookNetworkWorkClosed,
			payload: "NetworkWorkClosedPayload",
			patch:   "NetworkObservationPatch",
		},
	} {
		contract, ok := byEvent[tc.event]
		if !ok {
			t.Fatalf("HookContracts() missing %q", tc.event)
		}
		if contract.Payload.Name != tc.payload || contract.Patch.Name != tc.patch {
			t.Fatalf(
				"%s contract payload/patch = %q/%q, want %q/%q",
				tc.event,
				contract.Payload.Name,
				contract.Patch.Name,
				tc.payload,
				tc.patch,
			)
		}
	}
}
