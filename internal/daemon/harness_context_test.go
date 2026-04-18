package daemon

import (
	"context"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/skills/bundled"
)

func TestHarnessContextResolverMatrix(t *testing.T) {
	t.Parallel()

	resolver := NewHarnessContextResolver(HarnessRuntimeSignals{
		MemoryPromptSectionEnabled: true,
		SkillsPromptSectionEnabled: true,
		DurableMemoryAugmenter:     true,
		SyntheticTurnsEnabled:      true,
		DetachedTaskRuntimeEnabled: true,
	})

	testCases := []struct {
		name           string
		input          HarnessResolutionInput
		wantSections   []HarnessPromptSection
		wantAugmenters []HarnessAugmenter
		wantReentry    ReentryMode
		wantDetached   DetachedRunMode
		wantLabel      string
		wantTags       map[string]string
	}{
		{
			name: "user session plus user turn resolves baseline policy",
			input: HarnessResolutionInput{
				Surface: ResolutionSurfaceTurn,
				Session: HarnessSessionInput{
					Type: session.SessionTypeUser,
				},
				Turn: HarnessTurnRequest{
					Source: session.TurnSourceUser,
				},
			},
			wantSections:   []HarnessPromptSection{HarnessPromptSectionMemory, HarnessPromptSectionSkills},
			wantAugmenters: []HarnessAugmenter{HarnessAugmenterDurableMemory},
			wantReentry:    ReentryModeNone,
			wantDetached:   DetachedRunModeNone,
			wantLabel:      "interactive.user",
			wantTags: map[string]string{
				"harness.surface":          "turn",
				"harness.session_type":     "user",
				"harness.session_class":    "interactive",
				"harness.turn_origin":      "user",
				"harness.channel_bound":    "false",
				"harness.diagnostic_label": "interactive.user",
			},
		},
		{
			name: "channel-bound user session plus network turn resolves network-aware policy",
			input: HarnessResolutionInput{
				Surface: ResolutionSurfaceTurn,
				Session: HarnessSessionInput{
					Type:    session.SessionTypeUser,
					Channel: "builders",
				},
				Turn: HarnessTurnRequest{
					Source: session.TurnSourceNetwork,
					PromptMeta: acp.PromptMeta{
						Network: &acp.PromptNetworkMeta{
							Channel: "builders",
							From:    "ops.peer",
						},
					},
				},
			},
			wantSections: []HarnessPromptSection{
				HarnessPromptSectionMemory,
				HarnessPromptSectionSkills,
				HarnessPromptSectionNetwork,
			},
			wantAugmenters: nil,
			wantReentry:    ReentryModeNone,
			wantDetached:   DetachedRunModeNone,
			wantLabel:      "interactive.channel.network",
			wantTags: map[string]string{
				"harness.surface":          "turn",
				"harness.session_type":     "user",
				"harness.session_class":    "interactive",
				"harness.turn_origin":      "network",
				"harness.channel_bound":    "true",
				"harness.diagnostic_label": "interactive.channel.network",
			},
		},
		{
			name: "system session plus synthetic turn requires metadata and resolves reentry policy",
			input: HarnessResolutionInput{
				Surface: ResolutionSurfaceTurn,
				Session: HarnessSessionInput{
					Type: session.SessionTypeSystem,
				},
				Turn: HarnessTurnRequest{
					Source: session.TurnSourceSynthetic,
					Synthetic: &SyntheticTurnMetadata{
						Reason:  "task_complete",
						Trigger: "task.run.completed",
					},
					Detached: &DetachedRunMetadata{
						TaskRunID: "run-123",
					},
				},
			},
			wantSections:   []HarnessPromptSection{HarnessPromptSectionMemory, HarnessPromptSectionSkills},
			wantAugmenters: nil,
			wantReentry:    ReentryModeSynthetic,
			wantDetached:   DetachedRunModeTaskRuntime,
			wantLabel:      "system.synthetic.reentry",
			wantTags: map[string]string{
				"harness.surface":           "turn",
				"harness.session_type":      "system",
				"harness.session_class":     "system",
				"harness.turn_origin":       "synthetic",
				"harness.channel_bound":     "false",
				"harness.diagnostic_label":  "system.synthetic.reentry",
				"harness.synthetic_reason":  "task_complete",
				"harness.synthetic_trigger": "task.run.completed",
				"harness.task_run_id":       "run-123",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolver.Resolve(tc.input)
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}

			if !slices.Equal(got.Policy.IncludeSections, tc.wantSections) {
				t.Fatalf("IncludeSections = %#v, want %#v", got.Policy.IncludeSections, tc.wantSections)
			}
			if !slices.Equal(got.Policy.EnableAugmenters, tc.wantAugmenters) {
				t.Fatalf("EnableAugmenters = %#v, want %#v", got.Policy.EnableAugmenters, tc.wantAugmenters)
			}
			if got.Policy.ReentryMode != tc.wantReentry {
				t.Fatalf("ReentryMode = %q, want %q", got.Policy.ReentryMode, tc.wantReentry)
			}
			if got.Policy.DetachedRunMode != tc.wantDetached {
				t.Fatalf("DetachedRunMode = %q, want %q", got.Policy.DetachedRunMode, tc.wantDetached)
			}
			if got.Policy.DiagnosticLabel != tc.wantLabel {
				t.Fatalf("DiagnosticLabel = %q, want %q", got.Policy.DiagnosticLabel, tc.wantLabel)
			}
			if !maps.Equal(got.Policy.ObservabilityTags, tc.wantTags) {
				t.Fatalf("ObservabilityTags = %#v, want %#v", got.Policy.ObservabilityTags, tc.wantTags)
			}
		})
	}
}

func TestHarnessContextResolverValidation(t *testing.T) {
	t.Parallel()

	resolver := NewHarnessContextResolver(HarnessRuntimeSignals{
		MemoryPromptSectionEnabled: true,
		SkillsPromptSectionEnabled: true,
		DurableMemoryAugmenter:     true,
		SyntheticTurnsEnabled:      true,
		DetachedTaskRuntimeEnabled: true,
	})

	testCases := []struct {
		name    string
		input   HarnessResolutionInput
		wantErr string
	}{
		{
			name: "unknown turn origin fails validation",
			input: HarnessResolutionInput{
				Surface: ResolutionSurfaceTurn,
				Session: HarnessSessionInput{Type: session.SessionTypeUser},
				Turn: HarnessTurnRequest{
					Source: session.TurnSource("mystery"),
				},
			},
			wantErr: `invalid harness turn origin "mystery"`,
		},
		{
			name: "mismatched prompt metadata fails validation",
			input: HarnessResolutionInput{
				Surface: ResolutionSurfaceTurn,
				Session: HarnessSessionInput{Type: session.SessionTypeUser},
				Turn: HarnessTurnRequest{
					Source: session.TurnSourceUser,
					PromptMeta: acp.PromptMeta{
						TurnSource: acp.PromptTurnSourceNetwork,
						Network: &acp.PromptNetworkMeta{
							Channel: "builders",
						},
					},
				},
			},
			wantErr: `does not match prompt metadata turn_source "network"`,
		},
		{
			name: "synthetic turn without metadata fails validation",
			input: HarnessResolutionInput{
				Surface: ResolutionSurfaceTurn,
				Session: HarnessSessionInput{Type: session.SessionTypeSystem},
				Turn: HarnessTurnRequest{
					Source: session.TurnSourceSynthetic,
				},
			},
			wantErr: "synthetic harness turns require runtime metadata",
		},
		{
			name: "synthetic turn on non-system session fails validation",
			input: HarnessResolutionInput{
				Surface: ResolutionSurfaceTurn,
				Session: HarnessSessionInput{Type: session.SessionTypeUser},
				Turn: HarnessTurnRequest{
					Source: session.TurnSourceSynthetic,
					Synthetic: &SyntheticTurnMetadata{
						Reason:  "task_complete",
						Trigger: "task.run.completed",
					},
				},
			},
			wantErr: `synthetic harness turns require a system session`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := resolver.Resolve(tc.input)
			if err == nil {
				t.Fatal("Resolve() error = nil, want validation failure")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Resolve() error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestHarnessContextResolverDiagnosticLabelsAreStable(t *testing.T) {
	t.Parallel()

	resolver := NewHarnessContextResolver(HarnessRuntimeSignals{
		MemoryPromptSectionEnabled: true,
		SkillsPromptSectionEnabled: true,
		DurableMemoryAugmenter:     true,
		SyntheticTurnsEnabled:      true,
		DetachedTaskRuntimeEnabled: true,
	})

	input := HarnessResolutionInput{
		Surface: ResolutionSurfaceTurn,
		Session: HarnessSessionInput{
			Type:    session.SessionTypeUser,
			Channel: "builders",
		},
		Turn: HarnessTurnRequest{
			Source: session.TurnSourceNetwork,
		},
	}

	first, err := resolver.Resolve(input)
	if err != nil {
		t.Fatalf("Resolve(first) error = %v", err)
	}
	second, err := resolver.Resolve(input)
	if err != nil {
		t.Fatalf("Resolve(second) error = %v", err)
	}

	if first.Policy.DiagnosticLabel != second.Policy.DiagnosticLabel {
		t.Fatalf(
			"DiagnosticLabel mismatch: first=%q second=%q",
			first.Policy.DiagnosticLabel,
			second.Policy.DiagnosticLabel,
		)
	}
}

func TestHarnessStartupPromptOverlayAppendsBundledNetworkSkill(t *testing.T) {
	t.Parallel()

	resolver := NewHarnessContextResolver(HarnessRuntimeSignals{})
	overlay := newHarnessStartupPromptOverlay(resolver)

	got, err := overlay.Apply(context.Background(), session.StartupPromptContext{
		SessionType: session.SessionTypeUser,
		Channel:     "builders",
	}, "Base prompt.")
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	networkSkill, err := bundled.LoadContent(bundledNetworkSkillName)
	if err != nil {
		t.Fatalf("LoadContent(%q) error = %v", bundledNetworkSkillName, err)
	}
	if !strings.Contains(got, networkSkill) {
		t.Fatalf("overlay prompt = %q, want bundled network skill content", got)
	}
	if strings.Count(got, networkSkill) != 1 {
		t.Fatalf("overlay network skill occurrences = %d, want 1", strings.Count(got, networkSkill))
	}

	plain, err := overlay.Apply(context.Background(), session.StartupPromptContext{
		SessionType: session.SessionTypeUser,
	}, " Base prompt. ")
	if err != nil {
		t.Fatalf("Apply(no channel) error = %v", err)
	}
	if plain != "Base prompt." {
		t.Fatalf("Apply(no channel) = %q, want %q", plain, "Base prompt.")
	}
}
