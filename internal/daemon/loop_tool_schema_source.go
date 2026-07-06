package daemon

import (
	"context"
	"encoding/json"
	"strings"

	looppkg "github.com/compozy/agh/internal/loop"
	toolspkg "github.com/compozy/agh/internal/tools"
)

type loopToolSchemaSource struct {
	registry toolspkg.Registry
}

var _ looppkg.ToolSchemaSource = loopToolSchemaSource{}

func newLoopToolSchemaSource(registry toolspkg.Registry) looppkg.ToolSchemaSource {
	if registry == nil {
		return nil
	}
	return loopToolSchemaSource{registry: registry}
}

func (s loopToolSchemaSource) Snapshot(toolID string) (looppkg.ToolSchemaSnapshot, bool) {
	if s.registry == nil {
		return looppkg.ToolSchemaSnapshot{}, false
	}
	id := toolspkg.ToolID(strings.TrimSpace(toolID))
	if err := id.Validate(); err != nil {
		return looppkg.ToolSchemaSnapshot{}, false
	}
	view, err := s.registry.Get(context.Background(), toolspkg.Scope{Operator: true}, id)
	if err != nil {
		return looppkg.ToolSchemaSnapshot{}, false
	}
	descriptor := view.Descriptor
	return looppkg.ToolSchemaSnapshot{
		ToolID:             descriptor.ID.String(),
		InputSchema:        cloneLoopSchemaRaw(descriptor.InputSchema),
		OutputSchema:       cloneLoopSchemaRaw(descriptor.OutputSchema),
		InputSchemaDigest:  strings.TrimSpace(descriptor.InputSchemaDigest),
		OutputSchemaDigest: strings.TrimSpace(descriptor.OutputSchemaDigest),
	}, true
}

func newLoopCompilerWithSchemaSource(source looppkg.ToolSchemaSource) *looppkg.Compiler {
	if source == nil {
		return looppkg.NewCompiler()
	}
	return looppkg.NewCompiler(looppkg.WithCompilerToolSchemaSource(source))
}

func newLoopLinterWithSchemaSource(source looppkg.ToolSchemaSource) looppkg.Linter {
	if source == nil {
		return looppkg.NewLinter()
	}
	return looppkg.NewLinter(looppkg.WithToolSchemaSource(source))
}

func cloneLoopSchemaRaw(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}
