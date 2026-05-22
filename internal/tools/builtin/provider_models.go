package builtin

import toolspkg "github.com/pedronauck/agh/internal/tools"

const (
	providerModelsProvidersKey = "providers"
)

const (
	providerModelsModelsKey = "models"
)

var providerModelTools = []toolspkg.Descriptor{
	nativeDescriptor(
		toolspkg.ToolIDProviderModelsList,
		"provider_models_list",
		"Provider Models List",
		"List daemon provider model catalog entries.",
		providerModelsListInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDProviderModels},
		[]string{providerModelsProvidersKey, providerModelsModelsKey, descriptorKeywordCatalog},
		[]string{"provider models", "model catalog", "list models"},
	),
	nativeDescriptor(
		toolspkg.ToolIDProviderModelsRefresh,
		"provider_models_refresh",
		"Provider Models Refresh",
		"Refresh daemon provider model catalog sources.",
		providerModelsRefreshInputSchema,
		toolspkg.RiskMutating,
		false,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDProviderModels},
		[]string{providerModelsProvidersKey, providerModelsModelsKey, descriptorKeywordCatalog, "refresh"},
		[]string{"refresh provider models", "refresh model catalog"},
	),
	nativeDescriptor(
		toolspkg.ToolIDProviderModelsStatus,
		"provider_models_status",
		"Provider Models Status",
		"Read provider model catalog source status.",
		providerModelsStatusInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDProviderModels},
		[]string{
			providerModelsProvidersKey,
			providerModelsModelsKey,
			descriptorKeywordCatalog,
			descriptorKeywordStatus,
		},
		[]string{"provider model status", "model catalog status"},
	),
}

func providerModelsDescriptors() []toolspkg.Descriptor {
	return providerModelTools
}

const providerModelsListInputSchema = `{
	"type":"object",
		"properties":{
			"provider_id":{"type":"string"},
			"source_id":{"type":"string"},
			"include_stale":{"type":"boolean"}
		},
	"additionalProperties":false
}`

const providerModelsRefreshInputSchema = `{
	"type":"object",
	"properties":{
		"provider_id":{"type":"string"},
		"source_id":{"type":"string"},
		"force":{"type":"boolean"},
		"request_id":{"type":"string"}
	},
	"additionalProperties":false
}`

const providerModelsStatusInputSchema = `{
	"type":"object",
	"properties":{
		"provider_id":{"type":"string"}
	},
	"additionalProperties":false
}`
