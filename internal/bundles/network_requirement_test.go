package bundles

import (
	"testing"

	"github.com/compozy/agh/internal/resources"
)

func TestActivationResourceSpecNetworkRequirementFields(t *testing.T) {
	t.Parallel()

	t.Run("Should round-trip confirmation fields on ActivationResourceSpec", func(t *testing.T) {
		t.Parallel()

		activation := Activation{
			ID:                       "act_live",
			ExtensionName:            "ext-live",
			BundleName:               "bundle",
			ProfileName:              "default",
			Scope:                    ScopeGlobal,
			NetworkRequirementDigest: "digest-v1",
			ConfirmedBy:              "operator",
			ConfirmedAt:              "2026-07-14T12:00:00Z",
		}
		spec := activationResourceSpecFromActivation(activation)
		if spec.NetworkRequirementDigest != "digest-v1" {
			t.Fatalf("digest = %q, want digest-v1", spec.NetworkRequirementDigest)
		}
		if spec.ConfirmedBy != "operator" || spec.ConfirmedAt != "2026-07-14T12:00:00Z" {
			t.Fatalf("confirmation = (%q, %q)", spec.ConfirmedBy, spec.ConfirmedAt)
		}
		restored := activationFromResourceRecord(resources.Record[ActivationResourceSpec]{
			ID:   activation.ID,
			Kind: BundleActivationResourceKind,
			Spec: spec,
		})
		if restored.NetworkRequirementDigest != "digest-v1" ||
			restored.ConfirmedBy != "operator" ||
			restored.ConfirmedAt != "2026-07-14T12:00:00Z" {
			t.Fatalf("restored = %#v", restored)
		}
	})

	t.Run("Should clear confirmation preview when digest is empty", func(t *testing.T) {
		t.Parallel()

		got := previewNetworkRequirement(
			Activation{
				NetworkRequirementDigest: "stale",
				ConfirmedBy:              "operator",
				ConfirmedAt:              "2026-07-14T12:00:00Z",
			},
			"",
		)
		if got.NetworkRequirementDigest != "" || got.ConfirmedBy != "" || got.ConfirmedAt != "" {
			t.Fatalf("preview = %#v, want cleared confirmation", got)
		}
	})
}
