package daemon

import (
	"strings"
	"testing"

	"github.com/pedronauck/agh/internal/resources"
)

type automationRuntimeOnlyStub struct {
	automationRuntime
}

type automationRuntimeTargetStub struct {
	automationRuntime
	automationResourceProjectorTarget
}

type rawStoreStub struct {
	resources.RawStore
}

var _ automationRuntime = automationRuntimeOnlyStub{}
var _ automationRuntime = automationRuntimeTargetStub{}
var _ automationResourceProjectorTarget = automationRuntimeTargetStub{}
var _ resources.RawStore = rawStoreStub{}

func TestAutomationResourceTargetReturnsProjectorTargetOnlyWhenSupported(t *testing.T) {
	t.Parallel()

	if got := automationResourceTarget(nil); got != nil {
		t.Fatalf("automationResourceTarget(nil) = %#v, want nil", got)
	}
	if got := automationResourceTarget(automationRuntimeOnlyStub{}); got != nil {
		t.Fatalf("automationResourceTarget(runtime without projector target) = %#v, want nil", got)
	}
	if got := automationResourceTarget(automationRuntimeTargetStub{}); got == nil {
		t.Fatal("automationResourceTarget(runtime with projector target) = nil, want target")
	}
}

func TestAutomationResourceStoresRejectsPartialWiring(t *testing.T) {
	t.Parallel()

	if jobs, triggers, err := automationResourceStores(nil, nil); err != nil || jobs != nil || triggers != nil {
		t.Fatalf(
			"automationResourceStores(nil, nil) = (%#v, %#v, %v), want (nil, nil, nil)",
			jobs,
			triggers,
			err,
		)
	}

	if _, _, err := automationResourceStores(nil, resources.NewCodecRegistry()); err == nil {
		t.Fatal("automationResourceStores(nil, codecs) error = nil, want missing raw store failure")
	} else if !strings.Contains(err.Error(), "raw store is required") {
		t.Fatalf("automationResourceStores(nil, codecs) error = %v, want raw store context", err)
	}

	if _, _, err := automationResourceStores(rawStoreStub{}, nil); err == nil {
		t.Fatal("automationResourceStores(raw, nil) error = nil, want missing codec registry failure")
	} else if !strings.Contains(err.Error(), "codec registry is required") {
		t.Fatalf("automationResourceStores(raw, nil) error = %v, want codec registry context", err)
	}
}
