package extensionpkg

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	automationpkg "github.com/pedronauck/agh/internal/automation"
)

func TestLoadBundleSpecsLoadsMixedFormatsAndSorts(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	bundlesDir := filepath.Join(rootDir, "bundles")
	if err := os.MkdirAll(bundlesDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	if err := os.WriteFile(filepath.Join(bundlesDir, "zeta.toml"), []byte(`
name = " Zeta "
description = " Team bundle "

[[profiles]]
name = " default "

[profiles.channels]
primary = " ops "

[[profiles.channels.items]]
name = " ops "
description = " Operations "
`), 0o644); err != nil {
		t.Fatalf("os.WriteFile(zeta.toml) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundlesDir, "alpha.json"), []byte(`{
		"bundle": {
			"name": " Alpha ",
			"description": " Alerts bundle ",
			"profiles": [{
				"name": " default ",
				"channels": {
					"primary": " alerts ",
					"items": [{
						"name": " alerts ",
						"description": " Alerts channel "
					}]
				}
			}]
		}
	}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile(alpha.json) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundlesDir, "ignore.txt"), []byte("skip"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(ignore.txt) error = %v", err)
	}

	bundles, err := LoadBundleSpecs(rootDir, &Manifest{
		Name: "bundle-loader",
		Resources: ResourcesConfig{
			Bundles: []string{"bundles"},
		},
	})
	if err != nil {
		t.Fatalf("LoadBundleSpecs() error = %v", err)
	}
	if len(bundles) != 2 {
		t.Fatalf("len(bundles) = %d, want 2", len(bundles))
	}

	gotNames := []string{bundles[0].Name, bundles[1].Name}
	if !slices.Equal(gotNames, []string{"Alpha", "Zeta"}) {
		t.Fatalf("bundle names = %#v, want sorted trimmed names", gotNames)
	}
	if bundles[0].Profiles[0].Channels.Primary != "alerts" {
		t.Fatalf("alpha primary channel = %q, want alerts", bundles[0].Profiles[0].Channels.Primary)
	}
	if bundles[1].Profiles[0].Channels.Items[0].Description != "Operations" {
		t.Fatalf(
			"zeta channel description = %q, want Operations",
			bundles[1].Profiles[0].Channels.Items[0].Description,
		)
	}
}

func TestBundleDocumentToBundleSpecNormalizesValuesAndDefaults(t *testing.T) {
	t.Parallel()

	disabled := false
	doc := bundleDocument{
		Bundle: bundleRawSpec{
			Name:        " Marketing ",
			Description: " Team bundle ",
			Profiles: []bundleRawProfile{{
				Name:        " default ",
				Description: " Primary profile ",
				Channels: BundleChannelsConfig{
					Primary: " ops ",
					Items: []BundleChannel{{
						Name:        " ops ",
						Description: " Operations ",
					}},
				},
				Jobs: []bundleRawJob{{
					Name:      " daily-digest ",
					AgentName: " planner ",
					Prompt:    " summarize incidents ",
					Schedule: automationpkg.ScheduleSpec{
						Mode:     automationpkg.ScheduleModeEvery,
						Interval: "1m",
					},
					Task:      &automationpkg.JobTaskConfig{NetworkChannel: "ops"},
					Retry:     automationpkg.DefaultRetryConfig(),
					FireLimit: automationpkg.DefaultFireLimitConfig(),
				}, {
					Name:      " disabled-job ",
					AgentName: " planner ",
					Prompt:    " summarize incidents ",
					Enabled:   &disabled,
					Schedule: automationpkg.ScheduleSpec{
						Mode:     automationpkg.ScheduleModeEvery,
						Interval: "5m",
					},
					Retry:     automationpkg.DefaultRetryConfig(),
					FireLimit: automationpkg.DefaultFireLimitConfig(),
				}},
				Triggers: []bundleRawTrigger{{
					Name:         " mention-alert ",
					AgentName:    " planner ",
					Prompt:       " triage this ",
					Event:        "message.created",
					Filter:       map[string]string{"team": "ops"},
					Retry:        automationpkg.DefaultRetryConfig(),
					FireLimit:    automationpkg.DefaultFireLimitConfig(),
					EndpointSlug: " /alerts ",
				}},
				Bridges: []BundleBridgePreset{{
					Name:             " telegram-main ",
					ExtensionName:    " bundled.bridge ",
					DisplayName:      " Marketing Bridge ",
					DeliveryDefaults: json.RawMessage(`{"mode":"safe"}`),
					SecretSlots: []BundleBridgeSecretSlot{{
						Name:        " bot_token ",
						Kind:        " api_token ",
						Description: " Bot token ",
					}},
				}},
			}},
		},
	}

	spec, err := doc.toBundleSpec()
	if err != nil {
		t.Fatalf("toBundleSpec() error = %v", err)
	}
	if spec.Name != "Marketing" {
		t.Fatalf("spec.Name = %q, want Marketing", spec.Name)
	}
	if spec.Description != "Team bundle" {
		t.Fatalf("spec.Description = %q, want Team bundle", spec.Description)
	}

	profile := spec.Profiles[0]
	if profile.Name != "default" {
		t.Fatalf("profile.Name = %q, want default", profile.Name)
	}
	if profile.Channels.Primary != "ops" {
		t.Fatalf("profile.Channels.Primary = %q, want ops", profile.Channels.Primary)
	}
	if !profile.Jobs[0].Enabled {
		t.Fatalf("jobs[0].Enabled = false, want true default")
	}
	if profile.Jobs[1].Enabled {
		t.Fatalf("jobs[1].Enabled = true, want explicit false")
	}
	if !profile.Triggers[0].Enabled {
		t.Fatalf("triggers[0].Enabled = false, want true default")
	}
	if profile.Triggers[0].EndpointSlug != "/alerts" {
		t.Fatalf("triggers[0].EndpointSlug = %q, want /alerts", profile.Triggers[0].EndpointSlug)
	}
	if profile.Bridges[0].SecretSlots[0].Kind != "api_token" {
		t.Fatalf("bridges[0].SecretSlots[0].Kind = %q, want api_token", profile.Bridges[0].SecretSlots[0].Kind)
	}

	profile.Jobs[0].Task.NetworkChannel = "changed"
	if doc.Bundle.Profiles[0].Jobs[0].Task.NetworkChannel != "ops" {
		t.Fatalf("raw job task mutated to %#v", doc.Bundle.Profiles[0].Jobs[0].Task)
	}

	profile.Triggers[0].Filter["team"] = "security"
	if doc.Bundle.Profiles[0].Triggers[0].Filter["team"] != "ops" {
		t.Fatalf("raw trigger filter mutated to %#v", doc.Bundle.Profiles[0].Triggers[0].Filter)
	}

	profile.Bridges[0].SecretSlots[0].Name = "changed"
	if doc.Bundle.Profiles[0].Bridges[0].SecretSlots[0].Name != " bot_token " {
		t.Fatalf("raw bridge secret slot mutated to %#v", doc.Bundle.Profiles[0].Bridges[0].SecretSlots)
	}
}

func TestBundleDocumentToBundleSpecRejectsConflictingProfileDeclarations(t *testing.T) {
	t.Parallel()

	_, err := (bundleDocument{
		Profiles: []bundleRawProfile{{Name: "root"}},
		Bundle: bundleRawSpec{
			Profiles: []bundleRawProfile{{Name: "bundle"}},
		},
	}).toBundleSpec()
	if !errors.Is(err, ErrBundleInvalid) {
		t.Fatalf("toBundleSpec() error = %v, want ErrBundleInvalid", err)
	}
}
