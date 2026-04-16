package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pedronauck/agh/internal/resources"
)

const (
	// ToolResourceKind is the canonical desired-state resource kind for tool records.
	ToolResourceKind     resources.ResourceKind = "tool"
	toolResourceMaxBytes                        = 256 << 10
)

// NewResourceCodec builds the canonical tool resource codec.
func NewResourceCodec() (resources.KindCodec[Tool], error) {
	return resources.NewJSONCodec(ToolResourceKind, toolResourceMaxBytes, validateToolSpec)
}

func validateToolSpec(_ context.Context, scope resources.ResourceScope, spec Tool) (Tool, error) {
	normalizedScope := scope.Normalize()
	if err := normalizedScope.Validate("scope"); err != nil {
		return Tool{}, err
	}

	normalized := Tool{
		Name:        strings.TrimSpace(spec.Name),
		Description: strings.TrimSpace(spec.Description),
		ReadOnly:    spec.ReadOnly,
		Source:      spec.Source,
	}
	if normalized.Name == "" {
		return Tool{}, fmt.Errorf("%w: tool.name is required", resources.ErrValidation)
	}
	if err := normalized.Source.Validate(); err != nil {
		return Tool{}, err
	}

	if len(spec.InputSchema) > 0 {
		var decoded any
		if err := json.Unmarshal(spec.InputSchema, &decoded); err != nil {
			return Tool{}, fmt.Errorf("%w: tool.input_schema: %v", resources.ErrValidation, err)
		}
		if decoded == nil {
			return normalized, nil
		}
		if _, ok := decoded.(map[string]any); !ok {
			return Tool{}, fmt.Errorf("%w: tool.input_schema must be a JSON object", resources.ErrValidation)
		}
		canonical, err := json.Marshal(decoded)
		if err != nil {
			return Tool{}, fmt.Errorf("tools: canonicalize input schema: %w", err)
		}
		normalized.InputSchema = append(json.RawMessage(nil), canonical...)
	}

	return normalized, nil
}
