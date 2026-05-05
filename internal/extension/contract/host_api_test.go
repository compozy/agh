package contract

import (
	"encoding/json"
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

func TestSkillsListParamsUseForAgentWireField(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(SkillsListParams{
		Workspace: "ws-alpha",
		ForAgent:  "coder",
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got, want := decoded["for_agent"], "coder"; got != want {
		t.Fatalf("decoded[for_agent] = %#v, want %q", got, want)
	}
	if _, ok := decoded["agent_name"]; ok {
		t.Fatalf("decoded unexpectedly contains legacy agent_name key: %#v", decoded)
	}
}
