package modelcatalog

import (
	"slices"
	"testing"

	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/testutil"
)

func TestProviderConfigSources(t *testing.T) {
	t.Parallel()

	t.Run("Should expose manual default outside curated list", func(t *testing.T) {
		t.Parallel()

		source := NewConfigSource(map[string]aghconfig.ProviderConfig{
			"codex": {
				Models: aghconfig.ProviderModelsConfig{
					Default: "manual-model",
					Curated: []aghconfig.ProviderModelConfig{
						{ID: "curated-model", DisplayName: "Curated Model"},
					},
				},
			},
		})
		rows, err := source.ListModels(testutil.Context(t), ListOptions{ProviderID: "codex", Now: testTime(0)})
		if err != nil {
			t.Fatalf("ListModels() error = %v", err)
		}
		if got, want := rowModelIDs(rows), []string{"manual-model", "curated-model"}; !slices.Equal(got, want) {
			t.Fatalf("row ids = %#v, want %#v", got, want)
		}
		if rows[0].DisplayName != "" {
			t.Fatalf("default DisplayName = %q, want empty metadata for manual default", rows[0].DisplayName)
		}
	})

	t.Run("Should expose canonical model ids for configured aliases", func(t *testing.T) {
		t.Parallel()

		source := NewConfigSource(map[string]aghconfig.ProviderConfig{
			"claude": {
				Models: aghconfig.ProviderModelsConfig{Default: "sonnet"},
			},
		})
		rows, err := source.ListModels(testutil.Context(t), ListOptions{ProviderID: "claude", Now: testTime(0)})
		if err != nil {
			t.Fatalf("ListModels() error = %v", err)
		}
		if got, want := rowModelIDs(rows), []string{"claude-sonnet-4-6"}; !slices.Equal(got, want) {
			t.Fatalf("row ids = %#v, want %#v", got, want)
		}
	})

	t.Run("Should preserve explicit curated ids before applying aliases", func(t *testing.T) {
		t.Parallel()

		source := NewConfigSource(map[string]aghconfig.ProviderConfig{
			"codex": {
				Models: aghconfig.ProviderModelsConfig{
					Default: "gpt-5",
					Curated: []aghconfig.ProviderModelConfig{
						{ID: "gpt-5", DisplayName: "GPT-5"},
					},
				},
			},
		})
		rows, err := source.ListModels(testutil.Context(t), ListOptions{ProviderID: "codex", Now: testTime(0)})
		if err != nil {
			t.Fatalf("ListModels() error = %v", err)
		}
		if got, want := rowModelIDs(rows), []string{"gpt-5"}; !slices.Equal(got, want) {
			t.Fatalf("row ids = %#v, want %#v", got, want)
		}
	})

	t.Run("Should convert curated config metadata into rows", func(t *testing.T) {
		t.Parallel()

		supportsTools := true
		contextWindow := int64(128000)
		defaultEffort := ReasoningEffortHigh
		source := NewConfigSource(map[string]aghconfig.ProviderConfig{
			"codex": {
				Models: aghconfig.ProviderModelsConfig{
					Default: "gpt-5.4",
					Curated: []aghconfig.ProviderModelConfig{
						{
							ID:                     "gpt-5.4",
							DisplayName:            "GPT-5.4",
							ContextWindow:          &contextWindow,
							SupportsTools:          &supportsTools,
							ReasoningEfforts:       []string{"low", "high"},
							DefaultReasoningEffort: string(defaultEffort),
						},
					},
				},
			},
		})
		rows, err := source.ListModels(testutil.Context(t), ListOptions{ProviderID: "codex", Now: testTime(0)})
		if err != nil {
			t.Fatalf("ListModels() error = %v", err)
		}
		if len(rows) != 1 {
			t.Fatalf("len(rows) = %d, want 1: %#v", len(rows), rows)
		}
		row := rows[0]
		if row.DisplayName != "GPT-5.4" {
			t.Fatalf("DisplayName = %q, want GPT-5.4", row.DisplayName)
		}
		if row.ContextWindow == nil || *row.ContextWindow != contextWindow {
			t.Fatalf("ContextWindow = %v, want %d", row.ContextWindow, contextWindow)
		}
		if row.SupportsTools == nil || !*row.SupportsTools {
			t.Fatalf("SupportsTools = %v, want true", row.SupportsTools)
		}
		if !slices.Equal(row.ReasoningEfforts, []ReasoningEffort{ReasoningEffortLow, ReasoningEffortHigh}) {
			t.Fatalf("ReasoningEfforts = %#v, want low/high", row.ReasoningEfforts)
		}
		if row.DefaultReasoningEffort == nil || *row.DefaultReasoningEffort != defaultEffort {
			t.Fatalf("DefaultReasoningEffort = %v, want high", row.DefaultReasoningEffort)
		}
	})

	t.Run("Should snapshot provider configs at construction", func(t *testing.T) {
		t.Parallel()

		contextWindow := int64(128000)
		supportsTools := true
		providers := map[string]aghconfig.ProviderConfig{
			"codex": {
				Models: aghconfig.ProviderModelsConfig{
					Curated: []aghconfig.ProviderModelConfig{
						{
							ID:               "gpt-5.4",
							DisplayName:      "GPT-5.4",
							ContextWindow:    &contextWindow,
							SupportsTools:    &supportsTools,
							ReasoningEfforts: []string{"low", "high"},
						},
					},
				},
			},
		}

		source := NewConfigSource(providers)

		cfg := providers["codex"]
		cfg.Models.Curated[0].DisplayName = "Mutated"
		cfg.Models.Curated[0].ReasoningEfforts[0] = "xhigh"
		providers["codex"] = cfg
		contextWindow = 1
		supportsTools = false

		rows, err := source.ListModels(testutil.Context(t), ListOptions{ProviderID: "codex", Now: testTime(0)})
		if err != nil {
			t.Fatalf("ListModels() error = %v", err)
		}
		if len(rows) != 1 {
			t.Fatalf("len(rows) = %d, want 1: %#v", len(rows), rows)
		}
		row := rows[0]
		if row.DisplayName != "GPT-5.4" {
			t.Fatalf("DisplayName = %q, want GPT-5.4 snapshot", row.DisplayName)
		}
		if row.ContextWindow == nil || *row.ContextWindow != 128000 {
			t.Fatalf("ContextWindow = %v, want 128000 snapshot", row.ContextWindow)
		}
		if row.SupportsTools == nil || !*row.SupportsTools {
			t.Fatalf("SupportsTools = %v, want true snapshot", row.SupportsTools)
		}
		if !slices.Equal(row.ReasoningEfforts, []ReasoningEffort{ReasoningEffortLow, ReasoningEffortHigh}) {
			t.Fatalf("ReasoningEfforts = %#v, want low/high snapshot", row.ReasoningEfforts)
		}
	})

	t.Run("Should expose builtin provider model defaults", func(t *testing.T) {
		t.Parallel()

		source := NewBuiltinSource()
		rows, err := source.ListModels(testutil.Context(t), ListOptions{ProviderID: "codex", Now: testTime(0)})
		if err != nil {
			t.Fatalf("ListModels() error = %v", err)
		}
		if len(rows) == 0 {
			t.Fatal("len(rows) = 0, want builtin codex models")
		}
		if rows[0].SourceID != SourceIDBuiltin || rows[0].SourceKind != SourceKindBuiltin ||
			rows[0].Priority != PriorityBuiltin {
			t.Fatalf("row source = %#v, want builtin source metadata", rows[0])
		}
	})
}

func rowModelIDs(rows []ModelRow) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ModelID)
	}
	return ids
}
