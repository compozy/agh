package daemon

import (
	"context"

	aghconfig "github.com/compozy/agh/internal/config"
	looppkg "github.com/compozy/agh/internal/loop"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

func newGoalRunPolicyResolver(
	homePaths aghconfig.HomePaths,
	workspaceResolver workspacepkg.RuntimeResolver,
) looppkg.GoalRunPolicyResolver {
	return looppkg.GoalRunPolicyResolverFunc(func(
		ctx context.Context,
		workspaceID looppkg.WorkspaceID,
	) (*looppkg.GoalRunPolicy, error) {
		cfg, err := resolveLoopServiceConfig(ctx, homePaths, workspaceResolver, workspaceID)
		if err != nil {
			return nil, err
		}
		return &looppkg.GoalRunPolicy{
			ContextNudgeRatio: cfg.Goals.ContextNudgeRatio,
		}, nil
	})
}
