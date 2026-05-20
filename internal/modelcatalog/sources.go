package modelcatalog

import (
	"context"
	"maps"
	"sort"
	"strings"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

type providerConfigSource struct {
	id        string
	kind      SourceKind
	priority  int
	providers map[string]aghconfig.ProviderConfig
}

var _ Source = (*providerConfigSource)(nil)

// NewBuiltinSource creates the offline bootstrap source from AGH built-ins.
func NewBuiltinSource() Source {
	return newProviderConfigSource(
		SourceIDBuiltin,
		SourceKindBuiltin,
		PriorityBuiltin,
		aghconfig.BuiltinProviders(),
	)
}

// NewConfigSource creates the operator config model source.
func NewConfigSource(providers map[string]aghconfig.ProviderConfig) Source {
	return newProviderConfigSource(SourceIDConfig, SourceKindConfig, PriorityConfig, providers)
}

func newProviderConfigSource(
	id string,
	kind SourceKind,
	priority int,
	providers map[string]aghconfig.ProviderConfig,
) Source {
	return &providerConfigSource{
		id:        id,
		kind:      kind,
		priority:  priority,
		providers: cloneConfigProviders(providers),
	}
}

func (s *providerConfigSource) ID() string {
	return s.id
}

func (s *providerConfigSource) Kind() SourceKind {
	return s.kind
}

func (s *providerConfigSource) Priority() int {
	return s.priority
}

func (s *providerConfigSource) ProviderIDs() []string {
	providers := make([]string, 0, len(s.providers))
	for providerID := range s.providers {
		providers = append(providers, providerID)
	}
	sort.Strings(providers)
	return providers
}

func (s *providerConfigSource) ListModels(
	_ context.Context,
	opts ListOptions,
) ([]ModelRow, error) {
	now := defaultNow(opts.Now)
	providers := s.ProviderIDs()
	rows := make([]ModelRow, 0)
	for _, providerID := range providers {
		if opts.ProviderID != "" && opts.ProviderID != providerID {
			continue
		}
		provider := s.providers[providerID]
		rows = append(rows, providerModelRows(providerID, provider.Models, s.id, s.kind, s.priority, now)...)
	}
	return rows, nil
}

func providerModelRows(
	providerID string,
	models aghconfig.ProviderModelsConfig,
	sourceID string,
	kind SourceKind,
	priority int,
	now time.Time,
) []ModelRow {
	byID := make(map[string]ModelRow)
	order := make([]string, 0, len(models.Curated)+1)
	addModel := func(modelID string) ModelRow {
		trimmed := strings.TrimSpace(modelID)
		row, ok := byID[trimmed]
		if ok {
			return row
		}
		row = ModelRow{
			ProviderID:  providerID,
			ModelID:     trimmed,
			SourceID:    sourceID,
			SourceKind:  kind,
			Priority:    priority,
			RefreshedAt: now,
		}
		byID[trimmed] = row
		order = append(order, trimmed)
		return row
	}
	if defaultModel := strings.TrimSpace(models.Default); defaultModel != "" {
		if providerConfigHasCuratedModel(models, defaultModel) {
			addModel(defaultModel)
		} else {
			addModel(aghconfig.CanonicalProviderModelName(providerID, defaultModel))
		}
	}
	for _, curated := range models.Curated {
		modelID := strings.TrimSpace(curated.ID)
		if modelID == "" {
			continue
		}
		row := addModel(modelID)
		enrichRowFromProviderModel(&row, curated)
		byID[modelID] = row
	}
	rows := make([]ModelRow, 0, len(order))
	for _, modelID := range order {
		rows = append(rows, byID[modelID])
	}
	return rows
}

func providerConfigHasCuratedModel(models aghconfig.ProviderModelsConfig, modelID string) bool {
	for _, curated := range models.Curated {
		if strings.TrimSpace(curated.ID) == modelID {
			return true
		}
	}
	return false
}

func enrichRowFromProviderModel(row *ModelRow, model aghconfig.ProviderModelConfig) {
	row.DisplayName = strings.TrimSpace(model.DisplayName)
	row.ContextWindow = model.ContextWindow
	row.MaxInputTokens = model.MaxInputTokens
	row.MaxOutputTokens = model.MaxOutputTokens
	row.SupportsTools = model.SupportsTools
	row.SupportsReasoning = model.SupportsReasoning
	row.CostInputPerMillion = model.CostInputPerMillion
	row.CostOutputPerMillion = model.CostOutputPerMillion
	if len(model.ReasoningEfforts) > 0 {
		row.ReasoningEfforts = make([]ReasoningEffort, 0, len(model.ReasoningEfforts))
		for _, effort := range model.ReasoningEfforts {
			trimmed := strings.TrimSpace(effort)
			if trimmed != "" {
				row.ReasoningEfforts = append(row.ReasoningEfforts, ReasoningEffort(trimmed))
			}
		}
	}
	if effort := strings.TrimSpace(model.DefaultReasoningEffort); effort != "" {
		defaultEffort := ReasoningEffort(effort)
		row.DefaultReasoningEffort = &defaultEffort
	}
}

func cloneConfigProviders(src map[string]aghconfig.ProviderConfig) map[string]aghconfig.ProviderConfig {
	if src == nil {
		return map[string]aghconfig.ProviderConfig{}
	}
	cloned := make(map[string]aghconfig.ProviderConfig, len(src))
	for providerID, cfg := range src {
		cloned[providerID] = cloneProviderConfig(cfg)
	}
	return cloned
}

func cloneProviderConfig(src aghconfig.ProviderConfig) aghconfig.ProviderConfig {
	cloned := src
	cloned.Models = cloneProviderModelsConfig(src.Models)
	cloned.SessionMCP = clonePtr(src.SessionMCP)
	cloned.Aliases = cloneStrings(src.Aliases)
	cloned.CredentialSlots = cloneProviderCredentialSlots(src.CredentialSlots)
	cloned.MCPServers = cloneMCPServers(src.MCPServers)
	return cloned
}

func cloneProviderModelsConfig(src aghconfig.ProviderModelsConfig) aghconfig.ProviderModelsConfig {
	cloned := src
	cloned.Curated = cloneProviderModelConfigs(src.Curated)
	return cloned
}

func cloneProviderModelConfigs(src []aghconfig.ProviderModelConfig) []aghconfig.ProviderModelConfig {
	if src == nil {
		return nil
	}
	cloned := make([]aghconfig.ProviderModelConfig, len(src))
	for index, cfg := range src {
		cloned[index] = cfg
		cloned[index].ContextWindow = clonePtr(cfg.ContextWindow)
		cloned[index].MaxInputTokens = clonePtr(cfg.MaxInputTokens)
		cloned[index].MaxOutputTokens = clonePtr(cfg.MaxOutputTokens)
		cloned[index].SupportsTools = clonePtr(cfg.SupportsTools)
		cloned[index].SupportsReasoning = clonePtr(cfg.SupportsReasoning)
		cloned[index].ReasoningEfforts = cloneStrings(cfg.ReasoningEfforts)
		cloned[index].CostInputPerMillion = clonePtr(cfg.CostInputPerMillion)
		cloned[index].CostOutputPerMillion = clonePtr(cfg.CostOutputPerMillion)
	}
	return cloned
}

func cloneProviderCredentialSlots(src []aghconfig.ProviderCredentialSlot) []aghconfig.ProviderCredentialSlot {
	if src == nil {
		return nil
	}
	return append([]aghconfig.ProviderCredentialSlot(nil), src...)
}

func cloneMCPServers(src []aghconfig.MCPServer) []aghconfig.MCPServer {
	if src == nil {
		return nil
	}
	cloned := make([]aghconfig.MCPServer, len(src))
	for index, server := range src {
		cloned[index] = server
		cloned[index].Args = cloneStrings(server.Args)
		cloned[index].Env = cloneStringMap(server.Env)
		cloned[index].SecretEnv = cloneStringMap(server.SecretEnv)
		cloned[index].Auth.Scopes = cloneStrings(server.Auth.Scopes)
	}
	return cloned
}

func cloneStrings(src []string) []string {
	if src == nil {
		return nil
	}
	return append([]string(nil), src...)
}

func cloneStringMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	cloned := make(map[string]string, len(src))
	maps.Copy(cloned, src)
	return cloned
}

func clonePtr[T any](value *T) *T {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
