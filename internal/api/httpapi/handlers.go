package httpapi

import (
	"io/fs"
	"log/slog"
	"time"

	"github.com/pedronauck/agh/internal/api/core"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
)

type handlerConfig struct {
	sessions       core.SessionManager
	observer       core.Observer
	automation     core.AutomationManager
	workspaces     core.WorkspaceService
	skillsRegistry core.SkillsRegistry
	memoryStore    *memory.Store
	dreamTrigger   core.DreamTrigger
	staticFS       fs.FS
	homePaths      aghconfig.HomePaths
	config         aghconfig.Config
	logger         *slog.Logger
	startedAt      time.Time
	now            func() time.Time
	pollInterval   time.Duration
	agentLoader    core.AgentLoader
	httpPort       int
}

// Handlers expose request/response and SSE endpoints for the AGH API.
type Handlers struct {
	*core.BaseHandlers
	staticFS fs.FS
}

func newHandlers(cfg handlerConfig) *Handlers {
	if cfg.pollInterval <= 0 {
		cfg.pollInterval = defaultPollInterval
	}
	if cfg.httpPort <= 0 {
		cfg.httpPort = cfg.config.HTTP.Port
	}

	return &Handlers{
		BaseHandlers: core.NewBaseHandlers(core.BaseHandlerConfig{
			TransportName:                "httpapi",
			MaskInternalErrors:           true,
			IncludeSessionWorkspaceInSSE: false,
			Sessions:                     cfg.sessions,
			Observer:                     cfg.observer,
			Automation:                   cfg.automation,
			Workspaces:                   cfg.workspaces,
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
		staticFS: cfg.staticFS,
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
