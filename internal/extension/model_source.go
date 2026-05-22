package extensionpkg

import (
	"context"
	"fmt"
	"strings"
	"time"

	apicontract "github.com/compozy/agh/internal/api/contract"
	extensioncontract "github.com/compozy/agh/internal/extension/contract"
	extensionprotocol "github.com/compozy/agh/internal/extension/protocol"
	"github.com/compozy/agh/internal/modelcatalog"
)

// ModelSourceRuntime calls AGH-to-extension model source services.
type ModelSourceRuntime interface {
	ListModelSourceRows(
		ctx context.Context,
		extensionName string,
		params extensioncontract.ModelSourceListParams,
	) ([]extensioncontract.ModelSourceRow, error)
}

// ModelSourceRuntimeResolver returns the current extension runtime.
type ModelSourceRuntimeResolver func() ModelSourceRuntime

// ModelSource adapts one extension into a daemon-owned model catalog source.
type ModelSource struct {
	info     ExtensionInfo
	sourceID string
	resolver ModelSourceRuntimeResolver
}

var _ modelcatalog.Source = (*ModelSource)(nil)
var _ ModelSourceRuntime = (*Manager)(nil)

// NewExtensionModelSources creates sources for installed extensions that provide model.source.
func NewExtensionModelSources(registry *Registry, resolver ModelSourceRuntimeResolver) ([]modelcatalog.Source, error) {
	if registry == nil {
		return nil, nil
	}
	infos, err := registry.List()
	if err != nil {
		return nil, fmt.Errorf("extension: list model source extensions: %w", err)
	}
	sources := make([]modelcatalog.Source, 0, len(infos))
	for _, info := range infos {
		if !providesCapability(info.Capabilities.Provides, extensionprotocol.CapabilityProvideModelSource) {
			continue
		}
		source, err := NewExtensionModelSource(info, resolver)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	return sources, nil
}

// NewExtensionModelSource creates a daemon model catalog source for one extension.
func NewExtensionModelSource(info ExtensionInfo, resolver ModelSourceRuntimeResolver) (*ModelSource, error) {
	sourceID, err := modelcatalog.SourceKindExtensionID(info.Name)
	if err != nil {
		return nil, fmt.Errorf("extension: create model source for %q: %w", info.Name, err)
	}
	return &ModelSource{
		info:     cloneExtensionInfo(info),
		sourceID: sourceID,
		resolver: resolver,
	}, nil
}

// ID returns the stable extension source id.
func (s *ModelSource) ID() string {
	if s == nil {
		return ""
	}
	return s.sourceID
}

// Kind returns extension.
func (s *ModelSource) Kind() modelcatalog.SourceKind {
	return modelcatalog.SourceKindExtension
}

// Priority returns the extension merge priority.
func (s *ModelSource) Priority() int {
	return modelcatalog.PriorityExtension
}

// ListModels calls the extension models/list service and validates rows before persistence.
func (s *ModelSource) ListModels(ctx context.Context, opts modelcatalog.ListOptions) ([]modelcatalog.ModelRow, error) {
	if ctx == nil {
		return nil, fmt.Errorf("extension: model source context is required")
	}
	if s == nil {
		return nil, fmt.Errorf("extension: model source is required")
	}
	if !s.info.Enabled {
		return nil, modelcatalog.ErrSourceDisabled
	}
	if !providesCapability(s.info.Capabilities.Provides, extensionprotocol.CapabilityProvideModelSource) {
		return nil, fmt.Errorf(
			"extension: model source %q is missing %q capability",
			s.info.Name,
			extensionprotocol.CapabilityProvideModelSource,
		)
	}
	if s.resolver == nil {
		return nil, fmt.Errorf("extension: model source runtime is unavailable")
	}
	runtime := s.resolver()
	if runtime == nil {
		return nil, fmt.Errorf("extension: model source runtime is unavailable")
	}
	rows, err := runtime.ListModelSourceRows(ctx, s.info.Name, extensioncontract.ModelSourceListParams{
		ProviderID:   strings.TrimSpace(opts.ProviderID),
		Refresh:      opts.Refresh,
		IncludeStale: opts.IncludeStale,
	})
	if err != nil {
		return nil, fmt.Errorf("extension: list model source %q: %w", s.info.Name, err)
	}
	return s.validateRows(rows, opts)
}

// ListModelSourceRows calls one extension's negotiated models/list service.
func (m *Manager) ListModelSourceRows(
	ctx context.Context,
	extensionName string,
	params extensioncontract.ModelSourceListParams,
) ([]extensioncontract.ModelSourceRow, error) {
	process, name, err := m.extensionServiceProcess(
		ctx,
		extensionName,
		extensionprotocol.ExtensionServiceMethodModelsList,
	)
	if err != nil {
		return nil, err
	}

	var response extensioncontract.ModelSourceListResponse
	if err := process.Call(
		ctx,
		string(extensionprotocol.ExtensionServiceMethodModelsList),
		params,
		&response,
	); err != nil {
		return nil, fmt.Errorf("extension: list models via %q: %w", name, err)
	}
	return cloneModelSourceRows(response.Rows), nil
}

func (s *ModelSource) validateRows(
	rows []extensioncontract.ModelSourceRow,
	opts modelcatalog.ListOptions,
) ([]modelcatalog.ModelRow, error) {
	if len(rows) == 0 {
		return nil, nil
	}
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	providerFilter := strings.TrimSpace(opts.ProviderID)
	validated := make([]modelcatalog.ModelRow, 0, len(rows))
	for index, row := range rows {
		modelRow, include, err := s.validateRow(index, row, providerFilter, now)
		if err != nil {
			return nil, err
		}
		if include {
			validated = append(validated, modelRow)
		}
	}
	return validated, nil
}

func (s *ModelSource) validateRow(
	index int,
	row extensioncontract.ModelSourceRow,
	providerFilter string,
	now time.Time,
) (modelcatalog.ModelRow, bool, error) {
	sourceID, providerID, modelID, err := s.validateRowIdentity(index, row, providerFilter)
	if err != nil {
		return modelcatalog.ModelRow{}, false, err
	}
	priority := row.Priority
	if priority == 0 {
		priority = modelcatalog.PriorityExtension
	}
	if priority != modelcatalog.PriorityExtension {
		return modelcatalog.ModelRow{}, false, fmt.Errorf(
			"extension: model source row %d priority %d must equal %d",
			index,
			priority,
			modelcatalog.PriorityExtension,
		)
	}
	if err := validateModelSourceMetadata(index, row); err != nil {
		return modelcatalog.ModelRow{}, false, err
	}
	efforts, defaultEffort, err := modelSourceReasoning(index, row.ReasoningEfforts, row.DefaultReasoningEffort)
	if err != nil {
		return modelcatalog.ModelRow{}, false, err
	}
	refreshedAt := row.RefreshedAt
	if refreshedAt.IsZero() {
		refreshedAt = now
	}
	modelRow := modelcatalog.ModelRow{
		ProviderID:             providerID,
		ModelID:                modelID,
		DisplayName:            strings.TrimSpace(row.DisplayName),
		SourceID:               sourceID,
		SourceKind:             modelcatalog.SourceKindExtension,
		Priority:               modelcatalog.PriorityExtension,
		Available:              row.Available,
		Stale:                  row.Stale,
		RefreshedAt:            refreshedAt.UTC(),
		ExpiresAt:              row.ExpiresAt.UTC(),
		ContextWindow:          row.ContextWindow,
		MaxInputTokens:         row.MaxInputTokens,
		MaxOutputTokens:        row.MaxOutputTokens,
		SupportsTools:          row.SupportsTools,
		SupportsReasoning:      row.SupportsReasoning,
		ReasoningEfforts:       efforts,
		DefaultReasoningEffort: defaultEffort,
		LastError:              strings.TrimSpace(row.LastError),
	}
	if row.Cost != nil {
		modelRow.CostInputPerMillion = row.Cost.InputPerMillion
		modelRow.CostOutputPerMillion = row.Cost.OutputPerMillion
	}
	return modelRow, true, nil
}

func (s *ModelSource) validateRowIdentity(
	index int,
	row extensioncontract.ModelSourceRow,
	providerFilter string,
) (string, string, string, error) {
	sourceID := strings.TrimSpace(row.SourceID)
	if sourceID == "" {
		return "", "", "", fmt.Errorf("extension: model source row %d source_id is required", index)
	}
	if sourceID != s.sourceID {
		return "", "", "", fmt.Errorf(
			"extension: model source row %d source_id %q must equal %q",
			index,
			sourceID,
			s.sourceID,
		)
	}
	if err := modelcatalog.ValidateSourceIdentity(sourceID, modelcatalog.SourceKindExtension); err != nil {
		return "", "", "", fmt.Errorf("extension: model source row %d: %w", index, err)
	}
	providerID := strings.TrimSpace(row.ProviderID)
	if providerID == "" {
		return "", "", "", fmt.Errorf(
			"extension: model source row %d provider_id is required",
			index,
		)
	}
	if providerFilter != "" && providerID != providerFilter {
		return "", "", "", fmt.Errorf(
			"extension: model source row %d provider_id %q is outside requested provider %q",
			index,
			providerID,
			providerFilter,
		)
	}
	modelID := strings.TrimSpace(row.ModelID)
	if modelID == "" {
		return "", "", "", fmt.Errorf("extension: model source row %d model_id is required", index)
	}
	return sourceID, providerID, modelID, nil
}

func validateModelSourceMetadata(index int, row extensioncontract.ModelSourceRow) error {
	for _, check := range []struct {
		field string
		value *int64
	}{
		{field: "context_window", value: row.ContextWindow},
		{field: "max_input_tokens", value: row.MaxInputTokens},
		{field: "max_output_tokens", value: row.MaxOutputTokens},
	} {
		if check.value != nil && *check.value < 0 {
			return fmt.Errorf("extension: model source row %d %s must be non-negative", index, check.field)
		}
	}
	if row.Cost != nil {
		if err := validateModelSourceCost(index, *row.Cost); err != nil {
			return err
		}
	}
	return nil
}

func validateModelSourceCost(index int, cost apicontract.ModelCatalogCostPayload) error {
	for _, check := range []struct {
		field string
		value *float64
	}{
		{field: "cost.input_per_million", value: cost.InputPerMillion},
		{field: "cost.output_per_million", value: cost.OutputPerMillion},
	} {
		if check.value != nil && *check.value < 0 {
			return fmt.Errorf("extension: model source row %d %s must be non-negative", index, check.field)
		}
	}
	return nil
}

func modelSourceReasoning(
	index int,
	values []string,
	defaultValue *string,
) ([]modelcatalog.ReasoningEffort, *modelcatalog.ReasoningEffort, error) {
	efforts := make([]modelcatalog.ReasoningEffort, 0, len(values))
	seen := make(map[modelcatalog.ReasoningEffort]struct{}, len(values))
	for _, value := range values {
		effort, err := parseModelSourceReasoningEffort(value)
		if err != nil {
			return nil, nil, fmt.Errorf("extension: model source row %d: %w", index, err)
		}
		if _, exists := seen[effort]; exists {
			return nil, nil, fmt.Errorf(
				"extension: model source row %d reasoning_efforts contains duplicate %q",
				index,
				effort,
			)
		}
		seen[effort] = struct{}{}
		efforts = append(efforts, effort)
	}
	if defaultValue == nil {
		return efforts, nil, nil
	}
	defaultEffort, err := parseModelSourceReasoningEffort(*defaultValue)
	if err != nil {
		return nil, nil, fmt.Errorf("extension: model source row %d default_reasoning_effort: %w", index, err)
	}
	if len(seen) > 0 {
		if _, ok := seen[defaultEffort]; !ok {
			return nil, nil, fmt.Errorf(
				"extension: model source row %d default_reasoning_effort %q is not in reasoning_efforts",
				index,
				defaultEffort,
			)
		}
	}
	return efforts, &defaultEffort, nil
}

func parseModelSourceReasoningEffort(value string) (modelcatalog.ReasoningEffort, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	switch modelcatalog.ReasoningEffort(trimmed) {
	case modelcatalog.ReasoningEffortMinimal,
		modelcatalog.ReasoningEffortLow,
		modelcatalog.ReasoningEffortMedium,
		modelcatalog.ReasoningEffortHigh,
		modelcatalog.ReasoningEffortXHigh:
		return modelcatalog.ReasoningEffort(trimmed), nil
	default:
		return "", fmt.Errorf("reasoning effort %q is not supported", value)
	}
}

func cloneModelSourceRows(src []extensioncontract.ModelSourceRow) []extensioncontract.ModelSourceRow {
	if len(src) == 0 {
		return nil
	}
	cloned := make([]extensioncontract.ModelSourceRow, len(src))
	for index := range src {
		cloned[index] = src[index]
		cloned[index].Available = cloneModelSourceBoolPointer(src[index].Available)
		cloned[index].ContextWindow = cloneModelSourceInt64Pointer(src[index].ContextWindow)
		cloned[index].MaxInputTokens = cloneModelSourceInt64Pointer(src[index].MaxInputTokens)
		cloned[index].MaxOutputTokens = cloneModelSourceInt64Pointer(src[index].MaxOutputTokens)
		cloned[index].SupportsTools = cloneModelSourceBoolPointer(src[index].SupportsTools)
		cloned[index].SupportsReasoning = cloneModelSourceBoolPointer(src[index].SupportsReasoning)
		cloned[index].ReasoningEfforts = append([]string(nil), src[index].ReasoningEfforts...)
		if src[index].DefaultReasoningEffort != nil {
			value := *src[index].DefaultReasoningEffort
			cloned[index].DefaultReasoningEffort = &value
		}
		if src[index].Cost != nil {
			cloned[index].Cost = &apicontract.ModelCatalogCostPayload{
				InputPerMillion:  cloneModelSourceFloat64Pointer(src[index].Cost.InputPerMillion),
				OutputPerMillion: cloneModelSourceFloat64Pointer(src[index].Cost.OutputPerMillion),
			}
		}
	}
	return cloned
}

func cloneModelSourceBoolPointer(src *bool) *bool {
	if src == nil {
		return nil
	}
	value := *src
	return &value
}

func cloneModelSourceInt64Pointer(src *int64) *int64 {
	if src == nil {
		return nil
	}
	value := *src
	return &value
}

func cloneModelSourceFloat64Pointer(src *float64) *float64 {
	if src == nil {
		return nil
	}
	value := *src
	return &value
}
