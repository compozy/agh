package daemon

import (
	"context"
	"time"

	"github.com/compozy/agh/internal/api/core"
	"github.com/compozy/agh/internal/memory"
	"github.com/compozy/agh/internal/memory/consolidation"
)

func (d *Daemon) runtimeDeps(
	ctx context.Context,
	state *bootState,
	sessions SessionManager,
) RuntimeDeps {
	if state != nil && state.dreamSvc != nil {
		lockPath := memory.ConsolidationLockPath(state.globalMemoryDir)
		state.dreamRuntime = consolidation.NewRuntime(
			state.cfg.Memory.Dream.Enabled,
			state.dreamSvc,
			consolidation.NewSessionSpawner(
				sessions,
				state.workspaceResolver,
				&state.cfg,
			),
			state.cfg.Memory.Dream.CheckInterval,
			state.logger,
			func() (time.Time, error) {
				return memory.NewConsolidationLock(lockPath).LastConsolidatedAt()
			},
		)
	}
	authoredContext := authoredContextRuntimeDeps(ctx, state, sessions)
	var memoryProviders core.MemoryProviderService
	if state.memoryProviderRegistry != nil {
		memoryProviders = daemonMemoryProviderService{registry: state.memoryProviderRegistry}
	}

	return RuntimeDeps{
		Config:              state.cfg,
		HomePaths:           d.homePaths,
		Logger:              state.logger,
		Sessions:            sessions,
		Bridges:             state.bridges,
		Notifications:       state.notificationPresets,
		Registry:            state.registry,
		MemoryStore:         state.memoryStore,
		MemoryExtractor:     state.memoryExtractor,
		MemoryProviders:     memoryProviders,
		MemorySessionLedger: newDaemonMemorySessionLedgerService(state, d.now),
		WorkspaceResolver:   state.workspaceResolver,
		WorkspaceService:    state.workspaceResolver,
		ModelCatalog:        state.modelCatalog,
		AgentCatalog: agentCatalogDependency(state.agentCatalog, agentSidecarCatalogs{
			soul:      state.soulCatalog,
			heartbeat: state.heartbeatCatalog,
		}),
		AgentContext:        state.situationContext,
		AgentDefinitionSync: state.agentSkillResources,
		SoulAuthoring:       authoredContext.SoulAuthoring,
		SoulHistoryPurger:   authoredContext.SoulHistoryPurger,
		SoulRefresher:       authoredContext.SoulRefresher,
		HeartbeatAuthor:     authoredContext.HeartbeatAuthoring,
		HeartbeatPurger:     authoredContext.HeartbeatPurger,
		HeartbeatStatus:     authoredContext.HeartbeatStatus,
		HeartbeatWake:       authoredContext.HeartbeatWake,
		SessionHealth:       authoredContext.SessionHealth,
		WakeEvents:          authoredContext.WakeEvents,
		CoordinatorConfig: newCoordinatorConfigResolver(
			&state.cfg,
			state.workspaceResolver,
			agentCatalogDependency(state.agentCatalog, agentSidecarCatalogs{
				soul:      state.soulCatalog,
				heartbeat: state.heartbeatCatalog,
			}),
		),
		SkillsRegistry: skillsRegistryAPI(state.skillsRegistry),
		ToolRegistry:   state.toolRegistry,
		Toolsets:       state.toolsets,
		ToolApprovals:  state.toolApprovals,
		HostedMCP:      state.hostedMCP,
		DreamTrigger:   dreamTriggerFromRuntime(state.dreamRuntime),
		Vault:          state.providerVault,
		StartedAt:      state.startedAt,
	}
}
