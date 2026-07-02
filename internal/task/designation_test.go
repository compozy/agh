package task

import (
	"encoding/json"
	"testing"
)

func TestApplyRunDesignationSummary(t *testing.T) {
	t.Parallel()

	t.Run("Should project run designation metadata into summaries", func(t *testing.T) {
		t.Parallel()

		summary := &RunSummary{DesignationGroupID: "stale-group"}
		run := Run{
			DesignationGroupID: "designation-group",
			Metadata:           json.RawMessage(`{"designation":{"index":3,"brief":"Review network fanout"}}`),
		}

		ApplyRunDesignationSummary(summary, run)

		if got, want := summary.DesignationGroupID, "designation-group"; got != want {
			t.Fatalf("DesignationGroupID = %q, want %q", got, want)
		}
		if summary.Designation == nil {
			t.Fatal("Designation = nil, want summary")
		}
		if got, want := summary.Designation.Index, 3; got != want {
			t.Fatalf("Designation.Index = %d, want %d", got, want)
		}
		if got, want := summary.Designation.Brief, "Review network fanout"; got != want {
			t.Fatalf("Designation.Brief = %q, want %q", got, want)
		}
	})

	t.Run("Should leave summaries unchanged when the run is not designated", func(t *testing.T) {
		t.Parallel()

		summary := &RunSummary{DesignationGroupID: "existing-group", Designation: &RunDesignationSummary{Index: 1}}

		ApplyRunDesignationSummary(summary, Run{})

		if got, want := summary.DesignationGroupID, "existing-group"; got != want {
			t.Fatalf("DesignationGroupID = %q, want %q", got, want)
		}
		if summary.Designation == nil || summary.Designation.Index != 1 {
			t.Fatalf("Designation = %#v, want existing summary", summary.Designation)
		}
	})
}
