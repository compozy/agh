package contract

import (
	"reflect"
	"testing"

	apicontract "github.com/compozy/agh/internal/api/contract"
	extensionprotocol "github.com/compozy/agh/internal/extension/protocol"
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

func TestHostAPIMethodSpecsPagedCollectionResults(t *testing.T) {
	t.Parallel()

	t.Run("Should expose the pagination envelopes returned by collection handlers", func(t *testing.T) {
		t.Parallel()

		want := map[HostAPIMethod]NamedType{
			HostAPIMethodAutomationJobs: {
				Name:  "AutomationJobsResult",
				Value: AutomationJobsResult{},
			},
			HostAPIMethodAutomationTriggers: {
				Name:  "AutomationTriggersResult",
				Value: AutomationTriggersResult{},
			},
			HostAPIMethodTasks: {
				Name:  "TasksResponse",
				Value: apicontract.TasksResponse{},
			},
		}

		for _, spec := range HostAPIMethodSpecs() {
			wantResult, ok := want[spec.Method]
			if !ok {
				continue
			}
			if spec.Result.Name != wantResult.Name {
				t.Fatalf(
					"HostAPIMethodSpecs()[%q].Result.Name = %q, want %q",
					spec.Method,
					spec.Result.Name,
					wantResult.Name,
				)
			}
			if got, expected := reflect.TypeOf(spec.Result.Value), reflect.TypeOf(wantResult.Value); got != expected {
				t.Fatalf("HostAPIMethodSpecs()[%q].Result.Value type = %v, want %v", spec.Method, got, expected)
			}
			delete(want, spec.Method)
		}

		if len(want) != 0 {
			t.Fatalf("HostAPIMethodSpecs() missing paged methods: %#v", want)
		}
	})
}
