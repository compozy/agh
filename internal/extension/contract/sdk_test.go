package contract

import (
	"testing"

	"github.com/pedronauck/agh/internal/hooks"
)

func TestHookContractsResolveAutonomyDescriptors(t *testing.T) {
	t.Parallel()

	contracts := HookContracts()
	if got, want := len(contracts), len(hooks.AllEventDescriptors()); got != want {
		t.Fatalf("len(HookContracts()) = %d, want %d", got, want)
	}

	byEvent := make(map[hooks.HookEvent]HookContractSpec, len(contracts))
	for _, contract := range contracts {
		byEvent[contract.Event] = contract
	}
	for _, event := range []hooks.HookEvent{
		hooks.HookCoordinatorPreSpawn,
		hooks.HookTaskRunPreClaim,
		hooks.HookSpawnPreCreate,
	} {
		contract, ok := byEvent[event]
		if !ok {
			t.Fatalf("HookContracts() missing %q", event)
		}
		if contract.Payload.Name == "" || contract.Patch.Name == "" {
			t.Fatalf("%s contract = %#v, want payload and patch names", event, contract)
		}
	}
}
