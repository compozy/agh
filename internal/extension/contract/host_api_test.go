package contract

import (
	"encoding/json"
	"reflect"
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

func TestSkillsListParamsUseForAgentWireField(t *testing.T) {
	t.Parallel()

	t.Run("Should use for_agent wire field", func(t *testing.T) {
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
	})
}

func TestEventSinceFiltersAreOptionalContractFields(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		value     any
		fieldName string
	}{
		{
			name:      "Should mark session event since filter optional",
			value:     SessionEventsParams{},
			fieldName: "Since",
		},
		{
			name:      "Should mark observe event since filter optional",
			value:     ObserveEventsParams{},
			fieldName: "Since",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			field, ok := reflect.TypeOf(tc.value).FieldByName(tc.fieldName)
			if !ok {
				t.Fatalf("%T.%s field missing", tc.value, tc.fieldName)
			}
			if got, want := field.Tag.Get("json"), "since,omitzero"; got != want {
				t.Fatalf("%T.%s json tag = %q, want %q", tc.value, tc.fieldName, got, want)
			}
		})
	}
}
