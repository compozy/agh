package modelcatalog

import (
	"sort"
	"strings"
	"time"
)

// MergeRows computes deterministic model projections from source rows.
func MergeRows(rows []ModelRow) []Model {
	if len(rows) == 0 {
		return nil
	}
	grouped := make(map[string][]ModelRow)
	for _, row := range rows {
		if strings.TrimSpace(row.ProviderID) == "" || strings.TrimSpace(row.ModelID) == "" {
			continue
		}
		key := row.ProviderID + "\x00" + row.ModelID
		grouped[key] = append(grouped[key], row)
	}
	models := make([]Model, 0, len(grouped))
	for _, group := range grouped {
		sortModelRows(group)
		models = append(models, mergeModelGroup(group))
	}
	sort.SliceStable(models, func(i, j int) bool {
		if models[i].ProviderID != models[j].ProviderID {
			return models[i].ProviderID < models[j].ProviderID
		}
		return models[i].ModelID < models[j].ModelID
	})
	return models
}

func mergeModelGroup(rows []ModelRow) Model {
	first := rows[0]
	model := Model{
		ProviderID:        first.ProviderID,
		ModelID:           first.ModelID,
		AvailabilityState: AvailabilityStateUnknown,
		RefreshedAt:       first.RefreshedAt,
		Sources:           make([]SourceRef, 0, len(rows)),
	}
	for _, row := range rows {
		model.Sources = append(model.Sources, SourceRef{
			SourceID:    row.SourceID,
			SourceKind:  row.SourceKind,
			Priority:    row.Priority,
			RefreshedAt: row.RefreshedAt,
			Stale:       row.Stale,
			LastError:   RedactString(row.LastError),
		})
		if model.DisplayName == "" {
			model.DisplayName = row.DisplayName
		}
		if model.ContextWindow == nil {
			model.ContextWindow = row.ContextWindow
		}
		if model.MaxInputTokens == nil {
			model.MaxInputTokens = row.MaxInputTokens
		}
		if model.MaxOutputTokens == nil {
			model.MaxOutputTokens = row.MaxOutputTokens
		}
		if model.SupportsTools == nil {
			model.SupportsTools = row.SupportsTools
		}
		if model.SupportsReasoning == nil {
			model.SupportsReasoning = row.SupportsReasoning
		}
		if len(model.ReasoningEfforts) == 0 && len(row.ReasoningEfforts) > 0 {
			model.ReasoningEfforts = append([]ReasoningEffort(nil), row.ReasoningEfforts...)
		}
		if model.DefaultReasoningEffort == nil {
			model.DefaultReasoningEffort = row.DefaultReasoningEffort
		}
		if model.CostInputPerMillion == nil {
			model.CostInputPerMillion = row.CostInputPerMillion
		}
		if model.CostOutputPerMillion == nil {
			model.CostOutputPerMillion = row.CostOutputPerMillion
		}
		if model.LastError == "" {
			model.LastError = RedactString(row.LastError)
		}
		if row.Stale {
			model.Stale = true
		}
	}
	applyAvailability(&model, rows)
	return model
}

func applyAvailability(model *Model, rows []ModelRow) {
	for _, row := range rows {
		if row.Available == nil || !availabilityAuthority(row.SourceKind) {
			continue
		}
		model.Available = row.Available
		model.Stale = row.Stale
		switch {
		case *row.Available && row.Stale:
			model.AvailabilityState = AvailabilityStateAvailableStale
		case *row.Available:
			model.AvailabilityState = AvailabilityStateAvailableLive
		case row.Stale:
			model.AvailabilityState = AvailabilityStateUnavailableStale
		default:
			model.AvailabilityState = AvailabilityStateUnavailableLive
		}
		return
	}
	model.AvailabilityState = AvailabilityStateUnknown
	model.Available = nil
}

func availabilityAuthority(kind SourceKind) bool {
	return kind == SourceKindProviderLive || kind == SourceKindExtension
}

func sortModelRows(rows []ModelRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		left := rows[i]
		right := rows[j]
		if left.Priority != right.Priority {
			return left.Priority > right.Priority
		}
		if !left.RefreshedAt.Equal(right.RefreshedAt) {
			return left.RefreshedAt.After(right.RefreshedAt)
		}
		return left.SourceID < right.SourceID
	})
}

func defaultNow(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}
