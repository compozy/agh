package daemon

import (
	"testing"

	aghconfig "github.com/compozy/agh/internal/config"
	loopdsl "github.com/compozy/agh/internal/loop/dsl"
)

func TestLoopDefaultsFromConfigShouldMapDeliveryAndWatchDefaults(t *testing.T) {
	t.Parallel()

	defaults := loopDefaultsFromConfig(aghconfig.DefaultLoopsConfig())

	assertIntPointer(t, "delivery iteration cap", defaults.Delivery.IterationCap, 50)
	assertIntPointer(t, "delivery no-progress window", defaults.Delivery.NoProgressWindow, 3)
	assertIntPointer(t, "delivery gate max revisions", defaults.Delivery.GateMaxRevisions, 10)
	assertIntPointer(t, "delivery budget tokens", defaults.Delivery.BudgetTokens, 0)
	assertIntPointer(t, "delivery budget wall", defaults.Delivery.BudgetWallSec, 0)
	if defaults.Delivery.BudgetOnExceeded == nil || *defaults.Delivery.BudgetOnExceeded != loopdsl.BudgetExceededHalt {
		t.Fatalf("delivery budget on exceeded = %#v, want halt", defaults.Delivery.BudgetOnExceeded)
	}
	assertIntPointer(t, "delivery fan out width", defaults.Delivery.FanOutWidth, 4)

	assertIntPointer(t, "watch iteration cap", defaults.Watch.IterationCap, 0)
	assertIntPointer(t, "watch no-progress window", defaults.Watch.NoProgressWindow, 2)
	if defaults.Watch.GateMaxRevisions != nil {
		t.Fatalf(
			"watch gate max revisions = %#v, want nil when config default is zero",
			defaults.Watch.GateMaxRevisions,
		)
	}
	assertIntPointer(t, "watch fan out width", defaults.Watch.FanOutWidth, 2)
}

func assertIntPointer(t *testing.T, label string, got *int, want int) {
	t.Helper()

	if got == nil {
		t.Fatalf("%s = nil, want %d", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %d, want %d", label, *got, want)
	}
}
