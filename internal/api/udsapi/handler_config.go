package udsapi

import (
	"log/slog"
	"time"

	"github.com/compozy/agh/internal/api/core"
	aghconfig "github.com/compozy/agh/internal/config"
	mcppkg "github.com/compozy/agh/internal/mcp"
	"github.com/compozy/agh/internal/memory"
	"github.com/compozy/agh/internal/store"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

type handlerConfig struct {
	sessions                core.SessionManager
	sessionCatalog          core.SessionCatalog
	tasks                   core.TaskService
	network                 core.NetworkService
	networkStore            core.NetworkStore
	networkUsage            store.NetworkUsageStore
	coordinationSettings    workspacepkg.CoordinationSettings
	coordinationInvitations workspacepkg.CoordinationInvitations
	observer                core.Observer
	schemaStreams           core.SchemaStreamStatusReader
	resources               core.ResourceService
	automation              core.AutomationManager
	loops                   core.LoopService
	bridges                 core.BridgeService
	notifications           core.NotificationPresetService
	bundles                 core.BundleService
	supportBundles          core.SupportBundleService
	tools                   core.ToolRegistry
	toolsets                core.ToolsetRegistry
	toolApprovals           core.ToolApprovalIssuer
	settings                core.SettingsService
	settingsRestart         core.SettingsRestartController
	settingsUpdate          core.SettingsUpdateController
	vault                   core.VaultService
	workspaces              core.WorkspaceService
	onboarding              core.OnboardingStore
	agentCatalog            core.AgentCatalog
	agentSync               core.AgentDefinitionSync
	modelCatalog            core.ModelCatalogService
	marketplaceCatalog      core.MarketplaceCatalogService
	agentContext            core.AgentContextService
	soulAuthoring           core.SoulAuthoringService
	soulHistoryPurger       core.SoulHistoryPurger
	soulRefresher           core.SoulRefresher
	heartbeatAuthor         core.HeartbeatAuthoringService
	heartbeatPurger         core.HeartbeatHistoryPurger
	heartbeatStatus         core.HeartbeatStatusService
	heartbeatWake           core.HeartbeatWakeService
	sessionHealth           core.SessionHealthReader
	wakeEvents              core.HeartbeatWakeEventReader
	coordinatorConfig       core.CoordinatorConfigResolver
	skillsRegistry          core.SkillsRegistry
	skillResources          core.SkillResourceSyncer
	memoryStore             *memory.Store
	dreamTrigger            core.DreamTrigger
	memoryExtractor         core.MemoryExtractorService
	memoryProviders         core.MemoryProviderService
	memoryLedger            core.MemorySessionLedgerService
	homePaths               aghconfig.HomePaths
	config                  aghconfig.Config
	logger                  *slog.Logger
	startedAt               time.Time
	now                     func() time.Time
	pollInterval            time.Duration
	agentLoader             core.AgentLoader
	extensions              ExtensionService
	hostedMCP               *mcppkg.HostedService
}
