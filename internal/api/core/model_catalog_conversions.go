package core

import (
	"time"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/modelcatalog"
)

const (
	taskActionList = "list"
)

func ProviderModelListPayloadFromModels(models []modelcatalog.Model) contract.ProviderModelListResponse {
	payload := contract.ProviderModelListResponse{
		Models: make([]contract.ProviderModelPayload, 0, len(models)),
	}
	for _, model := range models {
		payload.Models = append(payload.Models, ProviderModelPayloadFromModel(model))
	}
	return payload
}

func ProviderModelPayloadFromModel(model modelcatalog.Model) contract.ProviderModelPayload {
	payload := contract.ProviderModelPayload{
		ProviderID:             model.ProviderID,
		ModelID:                model.ModelID,
		DisplayName:            model.DisplayName,
		Sources:                SourceRefPayloadsFromRefs(model.Sources),
		Available:              model.Available,
		AvailabilityState:      string(model.AvailabilityState),
		Stale:                  model.Stale,
		RefreshedAt:            modelCatalogTimeString(model.RefreshedAt),
		ContextWindow:          model.ContextWindow,
		MaxInputTokens:         model.MaxInputTokens,
		MaxOutputTokens:        model.MaxOutputTokens,
		SupportsTools:          model.SupportsTools,
		SupportsReasoning:      model.SupportsReasoning,
		ReasoningEfforts:       reasoningEffortStrings(model.ReasoningEfforts),
		DefaultReasoningEffort: reasoningEffortStringPtr(model.DefaultReasoningEffort),
		LastError:              modelcatalog.RedactString(model.LastError),
	}
	if model.CostInputPerMillion != nil || model.CostOutputPerMillion != nil {
		payload.Cost = &contract.ModelCatalogCostPayload{
			InputPerMillion:  model.CostInputPerMillion,
			OutputPerMillion: model.CostOutputPerMillion,
		}
	}
	return payload
}

func SourceRefPayloadsFromRefs(refs []modelcatalog.SourceRef) []contract.ModelCatalogSourceRefPayload {
	payloads := make([]contract.ModelCatalogSourceRefPayload, 0, len(refs))
	for _, ref := range refs {
		payloads = append(payloads, contract.ModelCatalogSourceRefPayload{
			SourceID:    ref.SourceID,
			SourceKind:  string(ref.SourceKind),
			Priority:    ref.Priority,
			RefreshedAt: modelCatalogTimeString(ref.RefreshedAt),
			Stale:       ref.Stale,
			LastError:   modelcatalog.RedactString(ref.LastError),
		})
	}
	return payloads
}

func SourceStatusPayloadsFromStatuses(
	statuses []modelcatalog.SourceStatus,
) []contract.ModelCatalogSourceStatusPayload {
	payloads := make([]contract.ModelCatalogSourceStatusPayload, 0, len(statuses))
	for _, status := range statuses {
		payloads = append(payloads, contract.ModelCatalogSourceStatusPayload{
			SourceID:     status.SourceID,
			SourceKind:   string(status.SourceKind),
			ProviderID:   status.ProviderID,
			Priority:     status.Priority,
			LastRefresh:  modelCatalogTimeString(status.LastRefresh),
			NextRefresh:  modelCatalogTimeString(status.NextRefresh),
			LastSuccess:  modelCatalogTimeString(status.LastSuccess),
			LastError:    modelcatalog.RedactString(status.LastError),
			RefreshState: string(status.RefreshState),
			RowCount:     status.RowCount,
			Stale:        status.Stale,
		})
	}
	return payloads
}

func OpenAIModelListPayloadFromModels(models []modelcatalog.Model) contract.OpenAIModelListResponse {
	payload := contract.OpenAIModelListResponse{
		Object: taskActionList,
		Data:   make([]contract.OpenAIModelPayload, 0, len(models)),
	}
	for _, model := range models {
		payload.Data = append(payload.Data, OpenAIModelPayloadFromModel(model))
	}
	return payload
}

func OpenAIModelPayloadFromModel(model modelcatalog.Model) contract.OpenAIModelPayload {
	return contract.OpenAIModelPayload{
		ID:      model.ModelID,
		Object:  "model",
		Created: 0,
		OwnedBy: model.ProviderID,
		AGH: contract.OpenAIModelAGHPayload{
			ProviderID:             model.ProviderID,
			ModelID:                model.ModelID,
			DisplayName:            model.DisplayName,
			Sources:                sourceIDsFromRefs(model.Sources),
			Available:              model.Available,
			AvailabilityState:      string(model.AvailabilityState),
			Stale:                  model.Stale,
			RefreshedAt:            modelCatalogTimeString(model.RefreshedAt),
			ContextWindow:          model.ContextWindow,
			MaxInputTokens:         model.MaxInputTokens,
			MaxOutputTokens:        model.MaxOutputTokens,
			SupportsTools:          model.SupportsTools,
			SupportsReasoning:      model.SupportsReasoning,
			ReasoningEfforts:       reasoningEffortStrings(model.ReasoningEfforts),
			DefaultReasoningEffort: reasoningEffortStringPtr(model.DefaultReasoningEffort),
			Cost:                   costPayloadFromModel(model),
			LastError:              modelcatalog.RedactString(model.LastError),
		},
	}
}

func costPayloadFromModel(model modelcatalog.Model) *contract.ModelCatalogCostPayload {
	if model.CostInputPerMillion == nil && model.CostOutputPerMillion == nil {
		return nil
	}
	return &contract.ModelCatalogCostPayload{
		InputPerMillion:  model.CostInputPerMillion,
		OutputPerMillion: model.CostOutputPerMillion,
	}
}

func sourceIDsFromRefs(refs []modelcatalog.SourceRef) []string {
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		ids = append(ids, ref.SourceID)
	}
	return ids
}

func reasoningEffortStrings(efforts []modelcatalog.ReasoningEffort) []string {
	values := make([]string, 0, len(efforts))
	for _, effort := range efforts {
		values = append(values, string(effort))
	}
	return values
}

func reasoningEffortStringPtr(effort *modelcatalog.ReasoningEffort) *string {
	if effort == nil {
		return nil
	}
	value := string(*effort)
	return &value
}

func modelCatalogTimeString(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
