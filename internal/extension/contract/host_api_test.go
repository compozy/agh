package contract

import (
	"testing"

	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
)

func TestHostAPIMethodSpecsFollowProtocolWireOrder(t *testing.T) {
	t.Parallel()

	t.Run("Should follow protocol wire order", func(t *testing.T) {
		t.Parallel()

		specs := HostAPIMethodSpecs()
		wantOrder := extensionprotocol.AllHostAPIMethods()
		if len(specs) != len(wantOrder) {
			t.Fatalf("len(HostAPIMethodSpecs()) = %d, want %d", len(specs), len(wantOrder))
		}

		for idx := range wantOrder {
			if specs[idx].Method != wantOrder[idx] {
				t.Fatalf("HostAPIMethodSpecs()[%d].Method = %q, want %q", idx, specs[idx].Method, wantOrder[idx])
			}
		}
	})
}

func TestHostAPIMethodSpecsDefensiveCopy(t *testing.T) {
	t.Parallel()

	t.Run("Should isolate returned spec slice mutations", func(t *testing.T) {
		t.Parallel()

		specs := HostAPIMethodSpecs()
		if len(specs) == 0 {
			t.Fatal("HostAPIMethodSpecs() returned no specs")
		}

		original := specs[0].Method
		specs[0].Method = HostAPIMethod("mutated")

		next := HostAPIMethodSpecs()
		if next[0].Method != original {
			t.Fatalf("HostAPIMethodSpecs()[0].Method = %q after mutation, want %q", next[0].Method, original)
		}
	})
}
