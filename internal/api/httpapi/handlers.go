package httpapi

import (
	"io/fs"
	"log/slog"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/core"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
)

type handlerConfig struct {
	sessions        core.SessionManager
	tasks           core.TaskService
	network         core.NetworkService
	networkStore    core.NetworkStore
	observer        core.Observer
	resources       core.ResourceService
	automation      core.AutomationManager
	bridges         core.BridgeService
	bundles         core.BundleService
	settings        core.SettingsService
	settingsRestart core.SettingsRestartController
	workspaces      core.WorkspaceService
	agentCatalog    core.AgentCatalog
	skillsRegistry  core.SkillsRegistry
	memoryStore     *memory.Store
	dreamTrigger    core.DreamTrigger
	staticFS        fs.FS
	homePaths       aghconfig.HomePaths
	config          aghconfig.Config
	boundHost       string
	logger          *slog.Logger
	startedAt       time.Time
	now             func() time.Time
	pollInterval    time.Duration
	agentLoader     core.AgentLoader
	httpPort        int
	resourceAuth    []gin.HandlerFunc
	extensions      ExtensionService
}

// Handlers expose request/response and SSE endpoints for the AGH API.
type Handlers struct {
	*core.BaseHandlers
	staticFS      fs.FS
	resourceAuth  []gin.HandlerFunc
	Extensions    ExtensionService
	boundHost     string
	promptDrainWG sync.WaitGroup
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

	return &Handlers{
		BaseHandlers: core.NewBaseHandlers(&core.BaseHandlerConfig{
			TransportName:                "httpapi",
			MaskInternalErrors:           true,
			IncludeSessionWorkspaceInSSE: false,
			Sessions:                     cfg.sessions,
			Tasks:                        cfg.tasks,
			Network:                      cfg.network,
			NetworkStore:                 cfg.networkStore,
			Observer:                     cfg.observer,
			Resources:                    cfg.resources,
			Automation:                   cfg.automation,
			Bridges:                      cfg.bridges,
			Bundles:                      cfg.bundles,
			Settings:                     cfg.settings,
			SettingsRestart:              cfg.settingsRestart,
			Workspaces:                   cfg.workspaces,
			AgentCatalog:                 cfg.agentCatalog,
			SkillsRegistry:               cfg.skillsRegistry,
			MemoryStore:                  cfg.memoryStore,
			DreamTrigger:                 cfg.dreamTrigger,
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
		boundHost:    cfg.boundHost,
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
