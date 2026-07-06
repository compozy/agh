package loop

import (
	"context"
	"errors"
	"fmt"

	"github.com/compozy/agh/internal/loop/dsl"
)

func (r *CoordinatorRunner) resolveCoordinatorEffectiveConfig(
	ctx context.Context,
	run Run,
	resolved *ResolvedDefinition,
) (EffectiveConfig, error) {
	stored, err := r.store.GetLoopConfig(ctx, run.WorkspaceID, run.LoopName)
	if err != nil && !errors.Is(err, ErrConfigNotFound) {
		return EffectiveConfig{}, err
	}
	if errors.Is(err, ErrConfigNotFound) {
		stored = nil
	}
	defaults := DefaultLoopDefaults()
	if r.defaultsResolver != nil {
		defaults, err = r.defaultsResolver(ctx, run.WorkspaceID)
		if err != nil {
			return EffectiveConfig{}, fmt.Errorf("resolve loop defaults: %w", err)
		}
	}
	effective, err := ResolveEffectiveConfig(resolved, defaults, stored, LoopConfig{})
	if err != nil {
		return EffectiveConfig{}, err
	}
	return effective, nil
}

func coordinatorResolvedWithEffectiveConfig(
	resolved *ResolvedDefinition,
	effective EffectiveConfig,
) *ResolvedDefinition {
	if resolved == nil {
		return nil
	}
	next := *resolved
	nodes := append([]dsl.Node(nil), resolved.Definition.Graph.Nodes...)
	for index, node := range nodes {
		if node.Class == dsl.NodeClassControl && dsl.ControlKind(node.Kind) == dsl.ControlGate {
			node.MaxRevisions = effective.GateMaxRevisions
			nodes[index] = node
		}
	}
	next.Definition.Graph.Nodes = nodes
	return &next
}

func coordinatorFanOutWidth(effective EffectiveConfig) int {
	if effective.FanOutWidth <= 0 {
		return 1
	}
	return effective.FanOutWidth
}
