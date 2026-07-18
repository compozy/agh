package daemon

import (
	core "github.com/compozy/agh/internal/api/core"
	aghconfig "github.com/compozy/agh/internal/config"
	extensionpkg "github.com/compozy/agh/internal/extension"
	memorypkg "github.com/compozy/agh/internal/memory"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
	toolspkg "github.com/compozy/agh/internal/tools"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

type daemonNativeToolsDeps struct {
	Registry                   func() toolspkg.Registry
	Config                     aghconfig.Config
	Skills                     core.SkillsRegistry
	Sessions                   core.SessionManager
	Workspaces                 core.WorkspaceService
	WorkspaceResolver          workspacepkg.RuntimeResolver
	ModelCatalog               core.ModelCatalogService
	MarketplaceCatalog         core.MarketplaceCatalogService
	MarketplaceSkills          core.SkillMarketplaceService
	MarketplaceInstalledSkills core.InstalledSkillMarketplaceService
	Settings                   func() core.SettingsService
	Network                    core.NetworkService
	NetworkStore               core.NetworkStore
	NetworkUsage               store.NetworkUsageStore
	Tasks                      taskpkg.Manager
	MemoryStore                *memorypkg.Store
	MemoryToolWrites           memoryToolWriteRecorder
	DreamTrigger               core.DreamTrigger
	MemoryExtractor            core.MemoryExtractorService
	MemoryProviders            core.MemoryProviderService
	MemorySessionLedger        core.MemorySessionLedgerService
	Bridges                    core.BridgeService
	HomePaths                  aghconfig.HomePaths
	Observer                   core.Observer
	HookBindings               hookBindingPublisher
	AgentCatalog               core.AgentCatalog
	HeartbeatStatus            core.HeartbeatStatusService
	HeartbeatWake              core.HeartbeatWakeService
	SessionHealth              core.SessionHealthReader
	WakeEvents                 core.HeartbeatWakeEventReader
	Automation                 core.AutomationManager
	AutomationRuntime          func() core.AutomationManager
	ExtensionRegistry          *extensionpkg.Registry
	Extensions                 func() core.ExtensionService
	ExtensionRuntime           func() extensionRuntime
	ExtensionMarket            aghconfig.ExtensionsMarketplaceConfig
	ExtensionSources           extensionMarketplaceSourceLoader
	ExtensionEvents            store.EventSummaryStore
	AgentSkills                agentSkillPublisher
	ToolMCP                    toolMCPPublisher
	MCPAuth                    func() toolspkg.MCPAuthStatusProvider
	BundleResources            bundleResourcePublisher
	LoopResources              loopResourcePublisher
	BundleService              func() core.BundleService
	Loops                      func() core.LoopService
	Resources                  core.ResourceService
}
