package contract

import (
	"testing"

	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
)

func TestHostAPIMethodSpecsFollowProtocolWireOrder(t *testing.T) {
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
}
