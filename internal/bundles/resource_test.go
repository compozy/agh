package bundles

import (
	"context"
	"testing"

	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/resources"
)

func TestBundleActivationOwnedKindAllowlist(t *testing.T) {
	t.Parallel()

	for _, kind := range []resources.ResourceKind{
		automationpkg.JobResourceKind,
		automationpkg.TriggerResourceKind,
		bridgepkg.BridgeInstanceResourceKind,
	} {
		if !bundleActivationOwnedKindAllowed(kind) {
			t.Fatalf("bundleActivationOwnedKindAllowed(%q) = false, want true", kind)
		}
	}
	if bundleActivationOwnedKindAllowed(resources.ResourceKind("tool")) {
		t.Fatal("bundleActivationOwnedKindAllowed(tool) = true, want false")
	}
}

func TestBundleActivationBuildComposesTypedBundleDependency(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	service := newMarketingService(store)
	activation := Activation{
		ID:            ActivationResourceID("marketing-team", "marketing", "default", ScopeGlobal, ""),
		ExtensionName: "marketing-team",
		BundleName:    "marketing",
		ProfileName:   "default",
		Scope:         ScopeGlobal,
	}

	plan, err := service.Build(
		context.Background(),
		[]resources.Record[ActivationResourceSpec]{{
			Kind:    BundleActivationResourceKind,
			ID:      activation.ID,
			Version: 3,
			Scope:   resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
			Spec:    activationResourceSpecFromActivation(activation),
		}},
		store.bundles,
	)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if got, want := plan.Kind(), BundleActivationResourceKind; got != want {
		t.Fatalf("plan.Kind() = %q, want %q", got, want)
	}
	if got, want := plan.Revision(), int64(3); got != want {
		t.Fatalf("plan.Revision() = %d, want %d", got, want)
	}
	if got, want := plan.OperationCount(), 3; got != want {
		t.Fatalf("plan.OperationCount() = %d, want %d", got, want)
	}
	if err := service.Apply(context.Background(), plan); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got, want := len(store.applied), 1; got != want {
		t.Fatalf("len(applied plans) = %d, want %d", got, want)
	}
	if err := service.Apply(context.Background(), nonBundleActivationPlan{}); err == nil {
		t.Fatal("Apply(wrong plan type) error = nil, want failure")
	}
}

type nonBundleActivationPlan struct{}

func (nonBundleActivationPlan) Kind() resources.ResourceKind {
	return BundleActivationResourceKind
}

func (nonBundleActivationPlan) Revision() int64 {
	return 0
}

func (nonBundleActivationPlan) OperationCount() int {
	return 0
}
