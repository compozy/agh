package httpapi

import (
	"io/fs"
	"log/slog"
	"strings"
	"time"

	"github.com/compozy/agh/internal/api/core"
	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/memory"
	"github.com/gin-gonic/gin"
)

const (
	handlersLocalhostKey = "localhost"
)

type handlerConfig struct {
	sessions          core.SessionManager
	sessionCatalog    core.SessionCatalog
	tasks             core.TaskService
	network           core.NetworkService
	networkStore      core.NetworkStore
	observer          core.Observer
	resources         core.ResourceService
	automation        core.AutomationManager
	bridges           core.BridgeService
	notifications     core.NotificationPresetService
	bundles           core.BundleService
	supportBundles    core.SupportBundleService
	tools             core.ToolRegistry
	toolsets          core.ToolsetRegistry
	toolApprovals     core.ToolApprovalIssuer
	settings          core.SettingsService
	settingsRestart   core.SettingsRestartController
	settingsUpdate    core.SettingsUpdateController
	vault             core.VaultService
	workspaces        core.WorkspaceService
	onboarding        core.OnboardingStore
	agentCatalog      core.AgentCatalog
	modelCatalog      core.ModelCatalogService
	agentContext      core.AgentContextService
	coordinatorConfig core.CoordinatorConfigResolver
	soulAuthoring     core.SoulAuthoringService
	soulRefresher     core.SoulRefresher
	heartbeatAuthor   core.HeartbeatAuthoringService
	heartbeatStatus   core.HeartbeatStatusService
	heartbeatWake     core.HeartbeatWakeService
	sessionHealth     core.SessionHealthReader
	wakeEvents        core.HeartbeatWakeEventReader
	skillsRegistry    core.SkillsRegistry
	memoryStore       *memory.Store
	dreamTrigger      core.DreamTrigger
	memoryExtractor   core.MemoryExtractorService
	memoryProviders   core.MemoryProviderService
	memoryLedger      core.MemorySessionLedgerService
	staticFS          fs.FS
	homePaths         aghconfig.HomePaths
	config            aghconfig.Config
	boundHost         string
	logger            *slog.Logger
	startedAt         time.Time
	now               func() time.Time
	pollInterval      time.Duration
	agentLoader       core.AgentLoader
	httpPort          int
	resourceAuth      []gin.HandlerFunc
	extensions        ExtensionService
}

// Handlers expose request/response and SSE endpoints for the AGH API.
type Handlers struct {
	*core.BaseHandlers
	staticFS     fs.FS
	resourceAuth []gin.HandlerFunc
	Extensions   ExtensionService
	boundHost    string
}

func newHandlers(cfg *handlerConfig) *Handlers {
	if cfg == nil {
		cfg = &handlerConfig{}
	}

	if cfg.pollInterval <= 0 {
		cfg.pollInterval = defaultPollInterval
	}
	if cfg.httpPort <= 0 {
		cfg.httpPort = cfg.config.HTTP.Port
	}
	boundHost := strings.TrimSpace(cfg.boundHost)
	if boundHost == "" {
		boundHost = strings.TrimSpace(cfg.config.HTTP.Host)
	}
	if boundHost == "" {
		boundHost = handlersLocalhostKey
	}

	return &Handlers{
		BaseHandlers: core.NewBaseHandlers(&core.BaseHandlerConfig{
			TransportName:                "httpapi",
			MaskInternalErrors:           true,
			IncludeSessionWorkspaceInSSE: true,
			Sessions:                     cfg.sessions,
			SessionCatalog:               cfg.sessionCatalog,
			Tasks:                        cfg.tasks,
			Network:                      cfg.network,
			NetworkStore:                 cfg.networkStore,
			Observer:                     cfg.observer,
			Resources:                    cfg.resources,
			Extensions:                   cfg.extensions,
			Automation:                   cfg.automation,
			Bridges:                      cfg.bridges,
			Notifications:                cfg.notifications,
			Bundles:                      cfg.bundles,
			SupportBundles:               cfg.supportBundles,
			Tools:                        cfg.tools,
			Toolsets:                     cfg.toolsets,
			ToolApprovals:                cfg.toolApprovals,
			Settings:                     cfg.settings,
			SettingsRestart:              cfg.settingsRestart,
			SettingsUpdate:               cfg.settingsUpdate,
			Vault:                        cfg.vault,
			Workspaces:                   cfg.workspaces,
			Onboarding:                   cfg.onboarding,
			AgentCatalog:                 cfg.agentCatalog,
			ModelCatalog:                 cfg.modelCatalog,
			AgentContextService:          cfg.agentContext,
			CoordinatorConfig:            cfg.coordinatorConfig,
			SoulAuthoring:                cfg.soulAuthoring,
			SoulRefresher:                cfg.soulRefresher,
			HeartbeatAuthoring:           cfg.heartbeatAuthor,
			HeartbeatStatus:              cfg.heartbeatStatus,
			HeartbeatWake:                cfg.heartbeatWake,
			SessionHealth:                cfg.sessionHealth,
			HeartbeatWakeEvents:          cfg.wakeEvents,
			SkillsRegistry:               cfg.skillsRegistry,
			MemoryStore:                  cfg.memoryStore,
			DreamTrigger:                 cfg.dreamTrigger,
			MemoryExtractor:              cfg.memoryExtractor,
			MemoryProviders:              cfg.memoryProviders,
			MemorySessionLedger:          cfg.memoryLedger,
			HomePaths:                    cfg.homePaths,
			Config:                       cfg.config,
			Logger:                       cfg.logger,
			StartedAt:                    cfg.startedAt,
			Now:                          cfg.now,
			PollInterval:                 cfg.pollInterval,
			AgentLoader:                  cfg.agentLoader,
			HTTPPort:                     cfg.httpPort,
		}),
		staticFS:     cfg.staticFS,
		resourceAuth: append([]gin.HandlerFunc(nil), cfg.resourceAuth...),
		Extensions:   cfg.extensions,
		boundHost:    boundHost,
	}
}

func (h *Handlers) setStreamDone(done <-chan struct{}) {
	if h != nil && h.BaseHandlers != nil {
		h.SetStreamDone(done)
	}
}

func (h *Handlers) setHTTPPort(port int) {
	if h != nil && h.BaseHandlers != nil {
		h.SetHTTPPort(port)
	}
}

func (h *Handlers) resourceAuthMiddleware() []gin.HandlerFunc {
	if h == nil || len(h.resourceAuth) == 0 {
		return nil
	}
	return append([]gin.HandlerFunc(nil), h.resourceAuth...)
}

func (h *Handlers) privilegedMutationGuard() gin.HandlerFunc {
	boundHost := ""
	if h != nil {
		boundHost = h.boundHost
	}
	return loopbackMutationGuard(boundHost)
}
