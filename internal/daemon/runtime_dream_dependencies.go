package daemon

import (
	"context"
	"fmt"
	"strings"
	"time"

	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/memory"
	"github.com/compozy/agh/internal/memory/consolidation"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

func initializeDreamRuntime(state *bootState, sessions SessionManager) {
	if state == nil || state.dreamSvc == nil {
		return
	}
	lockPath := memory.ConsolidationLockPath(state.globalMemoryDir)
	roles := roleResolverForState(state)
	state.dreamRuntime = consolidation.NewRuntime(
		func() bool { return state.cfg.Roles.Dream.Enabled },
		state.dreamSvc,
		consolidation.NewSessionSpawner(
			sessions,
			state.workspaceResolver,
			state.cfg.Memory.Enabled,
			func(ctx context.Context, workspaceID string) (consolidation.SessionRoute, error) {
				correlation := roleInvocationCorrelation{
					WorkspaceID: strings.TrimSpace(workspaceID),
					Event: store.EventCorrelation{
						SchedulerReason: "dream-consolidation",
						ActorKind:       string(taskpkg.ActorKindDaemon),
						ActorID:         "dream-runtime",
					},
				}
				roleCtx := withRoleInvocationCorrelation(ctx, correlation)
				resolved, err := roles.Resolve(roleCtx, workspaceID, aghconfig.RoleDream)
				if err != nil {
					return consolidation.SessionRoute{}, fmt.Errorf("resolve dream role: %w", err)
				}
				return consolidation.SessionRoute{
					Enabled:         resolved.Enabled,
					AgentName:       resolved.AgentName,
					Provider:        resolved.Provider,
					Model:           resolved.Model,
					ReasoningEffort: resolved.ReasoningEffort,
					Fallbacks:       append([]aghconfig.RoleFallback(nil), resolved.Fallbacks...),
					BeforeFallback: func(
						fallbackCtx context.Context,
						attempt int,
						fallback aghconfig.RoleFallback,
					) error {
						return recordRoleFallbackEvent(fallbackCtx, resolved, correlation, attempt, roleAttemptRoute{
							AgentName:       resolved.AgentName,
							Provider:        fallback.Provider,
							Model:           fallback.Model,
							ReasoningEffort: fallback.ReasoningEffort,
						})
					},
				}, nil
			},
		),
		state.cfg.Memory.Dream.CheckInterval,
		state.logger,
		func() (time.Time, error) {
			return memory.NewConsolidationLock(lockPath).LastConsolidatedAt()
		},
	)
}
