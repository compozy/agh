package core_test

import (
	"strings"
	"testing"

	"github.com/compozy/agh/internal/api/core"
	bundlepkg "github.com/compozy/agh/internal/bundles"
	aghconfig "github.com/compozy/agh/internal/config"
	extensionpkg "github.com/compozy/agh/internal/extension"
)

func TestBundleActivationPayloadCategoryPath(t *testing.T) {
	t.Parallel()

	t.Run("Should expose category path on bundle agents", func(t *testing.T) {
		t.Parallel()

		payload := core.BundleActivationPayload(bundlepkg.ActivationPreview{
			Activation: bundlepkg.Activation{
				ID:            "act_marketing",
				ExtensionName: "marketing-team",
				BundleName:    "marketing",
				ProfileName:   "default",
				Scope:         bundlepkg.ScopeGlobal,
			},
			Bundle: extensionpkg.BundleSpec{Name: "marketing"},
			Profile: extensionpkg.BundleProfile{
				Name: "default",
				Agents: []extensionpkg.BundleAgent{{
					Path: "agents/planner",
					Agent: aghconfig.AgentDef{
						Name:         "planner",
						Model:        "sonnet",
						CategoryPath: []string{"Marketing", "Planning"},
						Prompt:       "Plan campaign work.",
					},
				}},
			},
		})
		if len(payload.Agents) != 1 {
			t.Fatalf("len(payload.Agents) = %d, want 1", len(payload.Agents))
		}
		if got, want := strings.Join(payload.Agents[0].CategoryPath, ","), "Marketing,Planning"; got != want {
			t.Fatalf("payload.Agents[0].CategoryPath = %#v, want %q", payload.Agents[0].CategoryPath, want)
		}
	})
}
