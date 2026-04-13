package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/session"
)

const defaultPollInterval = 100 * time.Millisecond

// BaseHandlerConfig configures a shared handler set for one transport.
type BaseHandlerConfig struct {
	TransportName                string
	MaskInternalErrors           bool
	IncludeSessionWorkspaceInSSE bool
	Sessions                     SessionManager
	Network                      NetworkService
	Observer                     Observer
	Automation                   AutomationManager
	Bridges                      BridgeService
	Workspaces                   WorkspaceService
	SkillsRegistry               SkillsRegistry
	MemoryStore                  *memory.Store
	DreamTrigger                 DreamTrigger
	HomePaths                    aghconfig.HomePaths
	Config                       aghconfig.Config
	Logger                       *slog.Logger
	StartedAt                    time.Time
	Now                          func() time.Time
	PollInterval                 time.Duration
	AgentLoader                  AgentLoader
	StreamDone                   <-chan struct{}
	HTTPPort                     int
	PID                          func() int
}

// BaseHandlers contains the shared transport-independent API handler logic.
type BaseHandlers struct {
	TransportName                string
	MaskInternalErrors           bool
	IncludeSessionWorkspaceInSSE bool
	Sessions                     SessionManager
	Network                      NetworkService
	Observer                     Observer
	Automation                   AutomationManager
	Bridges                      BridgeService
	Workspaces                   WorkspaceService
	SkillsRegistry               SkillsRegistry
	MemoryStore                  *memory.Store
	DreamTrigger                 DreamTrigger
	HomePaths                    aghconfig.HomePaths
	Config                       aghconfig.Config
	Logger                       *slog.Logger
	StartedAt                    time.Time
	Now                          func() time.Time
	PollInterval                 time.Duration
	AgentLoader                  AgentLoader
	PID                          func() int

	settingsMu sync.RWMutex
	streamDone <-chan struct{}
	httpPort   atomic.Int64
}

// NewBaseHandlers builds a shared handler set with transport-specific defaults applied.
func NewBaseHandlers(cfg BaseHandlerConfig) *BaseHandlers {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	now := cfg.Now
	if now == nil {
		now = func() time.Time {
			return time.Now().UTC()
		}
	}
	agentLoader := cfg.AgentLoader
	if agentLoader == nil {
		agentLoader = aghconfig.LoadAgentDef
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = defaultPollInterval
	}
	if cfg.StartedAt.IsZero() {
		cfg.StartedAt = now()
	}
	pid := cfg.PID
	if pid == nil {
		pid = func() int {
			return os.Getpid()
		}
	}

	if cfg.StreamDone == nil {
		logger.Warn("api: stream shutdown bridge not provided; streaming handlers will rely on caller context until a transport installs one")
		cfg.StreamDone = make(chan struct{})
	}

	handlers := &BaseHandlers{
		TransportName:                strings.TrimSpace(cfg.TransportName),
		MaskInternalErrors:           cfg.MaskInternalErrors,
		IncludeSessionWorkspaceInSSE: cfg.IncludeSessionWorkspaceInSSE,
		Sessions:                     cfg.Sessions,
		Network:                      cfg.Network,
		Observer:                     cfg.Observer,
		Automation:                   cfg.Automation,
		Bridges:                      cfg.Bridges,
		Workspaces:                   cfg.Workspaces,
		SkillsRegistry:               cfg.SkillsRegistry,
		MemoryStore:                  cfg.MemoryStore,
		DreamTrigger:                 cfg.DreamTrigger,
		HomePaths:                    cfg.HomePaths,
		Config:                       cfg.Config,
		Logger:                       logger,
		StartedAt:                    cfg.StartedAt,
		Now:                          now,
		PollInterval:                 cfg.PollInterval,
		AgentLoader:                  agentLoader,
		PID:                          pid,
	}
	handlers.streamDone = cfg.StreamDone
	handlers.httpPort.Store(int64(cfg.HTTPPort))
	return handlers
}

// SetStreamDone updates the transport shutdown bridge used by streaming handlers.
func (h *BaseHandlers) SetStreamDone(done <-chan struct{}) {
	if h == nil {
		return
	}
	if done == nil {
		h.Logger.Warn("api: stream shutdown bridge cleared; streaming handlers will rely on caller context until a transport installs one")
		done = make(chan struct{})
	}
	h.settingsMu.Lock()
	h.streamDone = done
	h.settingsMu.Unlock()
}

// SetHTTPPort overrides the reported HTTP port for daemon status responses.
func (h *BaseHandlers) SetHTTPPort(port int) {
	if h == nil || port <= 0 {
		return
	}
	h.httpPort.Store(int64(port))
}

// ListSessions returns the visible session list.
func (h *BaseHandlers) ListSessions(c *gin.Context) {
	infos, err := h.Sessions.ListAll(c.Request.Context())
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	workspaceFilter := strings.TrimSpace(c.Query("workspace"))
	if workspaceFilter != "" {
		workspaceID, lookupErr := h.lookupWorkspaceID(c.Request.Context(), workspaceFilter)
		if lookupErr != nil {
			h.respondError(c, StatusForWorkspaceError(lookupErr), lookupErr)
			return
		}
		infos = filterSessionInfosByWorkspaceIDInternal(infos, workspaceID)
	}

	c.JSON(http.StatusOK, contract.SessionsResponse{Sessions: SessionPayloadsFromInfos(infos)})
}

// CreateSession creates a new runtime session.
func (h *BaseHandlers) CreateSession(c *gin.Context) {
	var req contract.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, fmt.Errorf("%s: decode create session request: %w", h.transportName(), err))
		return
	}
	if err := h.validateCreateSessionRequest(req); err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	sess, err := h.Sessions.Create(c.Request.Context(), session.CreateOpts{
		AgentName:     req.AgentName,
		Name:          req.Name,
		Workspace:     strings.TrimSpace(req.Workspace),
		WorkspacePath: strings.TrimSpace(req.WorkspacePath),
		Channel:       strings.TrimSpace(req.Channel),
		Type:          session.SessionTypeUser,
	})
	if err != nil {
		h.respondError(c, StatusForSessionError(err), err)
		return
	}

	c.JSON(http.StatusCreated, contract.SessionResponse{Session: SessionPayloadFromInfo(sess.Info())})
}

// GetSession returns one session snapshot.
func (h *BaseHandlers) GetSession(c *gin.Context) {
	info, err := h.Sessions.Status(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondError(c, StatusForSessionError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.SessionResponse{Session: SessionPayloadFromInfo(info)})
}

// StopSession stops a running session.
func (h *BaseHandlers) StopSession(c *gin.Context) {
	if err := h.Sessions.Stop(c.Request.Context(), c.Param("id")); err != nil {
		h.respondError(c, StatusForSessionError(err), err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ResumeSession resumes a stopped session.
func (h *BaseHandlers) ResumeSession(c *gin.Context) {
	sess, err := h.Sessions.Resume(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondError(c, StatusForSessionError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.SessionResponse{Session: SessionPayloadFromInfo(sess.Info())})
}

// SessionEvents returns the filtered session event list.
func (h *BaseHandlers) SessionEvents(c *gin.Context) {
	query, err := ParseSessionEventQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	info, err := h.sessionEventInfo(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondError(c, StatusForSessionError(err), err)
		return
	}

	events, err := h.Sessions.Events(c.Request.Context(), c.Param("id"), query)
	if err != nil {
		h.respondError(c, StatusForSessionError(err), err)
		return
	}

	payload := make([]contract.SessionEventPayload, 0, len(events))
	for _, event := range events {
		payload = append(payload, SessionEventPayloadFromEvent(event, info))
	}

	c.JSON(http.StatusOK, contract.SessionEventsResponse{Events: payload})
}

// SessionHistory returns the grouped turn history for a session.
func (h *BaseHandlers) SessionHistory(c *gin.Context) {
	query, err := ParseSessionEventQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	info, err := h.sessionEventInfo(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondError(c, StatusForSessionError(err), err)
		return
	}

	history, err := h.Sessions.History(c.Request.Context(), c.Param("id"), query)
	if err != nil {
		h.respondError(c, StatusForSessionError(err), err)
		return
	}

	payload := make([]contract.TurnHistoryPayload, 0, len(history))
	for _, turn := range history {
		events := make([]contract.SessionEventPayload, 0, len(turn.Events))
		for _, event := range turn.Events {
			events = append(events, SessionEventPayloadFromEvent(event, info))
		}
		payload = append(payload, contract.TurnHistoryPayload{
			TurnID: turn.TurnID,
			Events: events,
		})
	}

	c.JSON(http.StatusOK, contract.SessionHistoryResponse{History: payload})
}

// SessionTranscript returns the stored transcript for a session.
func (h *BaseHandlers) SessionTranscript(c *gin.Context) {
	messages, err := h.Sessions.Transcript(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondError(c, StatusForSessionError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.SessionTranscriptResponse{Messages: messages})
}

// StreamSession streams session events over SSE.
func (h *BaseHandlers) StreamSession(c *gin.Context) {
	info, err := h.streamSessionInfo(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondError(c, StatusForSessionError(err), err)
		return
	}

	query, err := ParseSessionEventQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	if query.AfterSequence, err = parseLastEventID(c.GetHeader("Last-Event-ID"), h.transportName()); err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	initial, err := h.Sessions.Events(c.Request.Context(), c.Param("id"), query)
	if err != nil {
		h.respondError(c, StatusForSessionError(err), err)
		return
	}

	writer, err := PrepareSSE(c)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	afterSequence, err := h.writeSessionEventBatch(writer, initial, info)
	if err != nil {
		return
	}

	pollQuery := query
	pollQuery.Limit = 0
	h.pollAndStreamSessionEvents(c, writer, c.Param("id"), info, pollQuery, afterSequence)
}

// ListAgents returns all readable agent definitions in home paths.
func (h *BaseHandlers) ListAgents(c *gin.Context) {
	entries, err := os.ReadDir(h.HomePaths.AgentsDir)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		c.JSON(http.StatusOK, contract.AgentsResponse{Agents: []contract.AgentPayload{}})
		return
	default:
		h.respondError(c, http.StatusInternalServerError, fmt.Errorf("%s: read agents directory %q: %w", h.transportName(), h.HomePaths.AgentsDir, err))
		return
	}

	agents := make([]contract.AgentPayload, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := strings.TrimSpace(entry.Name())
		if name == "" {
			continue
		}

		agent, loadErr := h.AgentLoader(name, h.HomePaths)
		if loadErr != nil {
			h.Logger.Warn(h.transportName()+": skip unreadable agent definition", "agent_name", name, "error", loadErr)
			continue
		}
		agents = append(agents, AgentPayloadFromDef(agent))
	}

	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Name < agents[j].Name
	})

	c.JSON(http.StatusOK, contract.AgentsResponse{Agents: agents})
}

// GetAgent returns one agent definition by name.
func (h *BaseHandlers) GetAgent(c *gin.Context) {
	agent, err := h.AgentLoader(c.Param("name"), h.HomePaths)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
		}
		h.respondError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, contract.AgentResponse{Agent: AgentPayloadFromDef(agent)})
}

// HookCatalog returns the resolved hook catalog for the supplied workspace and agent view.
func (h *BaseHandlers) HookCatalog(c *gin.Context) {
	filter, err := ParseHookCatalogFilter(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	if workspaceRef := strings.TrimSpace(c.Query("workspace")); workspaceRef != "" {
		resolved, err := h.Workspaces.Resolve(c.Request.Context(), workspaceRef)
		if err != nil {
			h.respondError(c, StatusForWorkspaceError(err), err)
			return
		}
		filter.WorkspaceID = strings.TrimSpace(resolved.ID)
		filter.WorkspaceRoot = strings.TrimSpace(resolved.RootDir)
	}

	entries, err := h.Observer.QueryHookCatalog(c.Request.Context(), filter)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, contract.HookCatalogResponse{Hooks: HookCatalogPayloadsFromEntries(entries)})
}

// HookRuns returns persisted hook execution history for a session.
func (h *BaseHandlers) HookRuns(c *gin.Context) {
	query, err := ParseHookRunsQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(query.SessionID) == "" {
		h.respondError(c, http.StatusBadRequest, fmt.Errorf("%s: session query is required", h.transportName()))
		return
	}

	if _, err := h.Sessions.Status(c.Request.Context(), query.SessionID); err != nil {
		h.respondError(c, StatusForSessionError(err), err)
		return
	}

	records, err := h.Observer.QueryHookRuns(c.Request.Context(), query)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, contract.HookRunsResponse{Runs: HookRunPayloadsFromRecords(records)})
}

// HookEvents returns the supported hook taxonomy metadata.
func (h *BaseHandlers) HookEvents(c *gin.Context) {
	filter, err := ParseHookEventFilter(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	events, err := h.Observer.QueryHookEvents(c.Request.Context(), filter)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, contract.HookEventsResponse{Events: HookEventPayloadsFromDescriptors(events)})
}

// ObserveEvents returns the filtered observe event list.
func (h *BaseHandlers) ObserveEvents(c *gin.Context) {
	query, err := ParseObserveEventQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	events, err := h.Observer.QueryEvents(c.Request.Context(), query)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	payload := make([]contract.ObserveEventPayload, 0, len(events))
	for _, event := range events {
		payload = append(payload, ObserveEventPayloadFromEvent(event))
	}

	c.JSON(http.StatusOK, contract.ObserveEventsResponse{Events: payload})
}

// StreamObserveEvents streams observe events over SSE.
func (h *BaseHandlers) StreamObserveEvents(c *gin.Context) {
	query, err := ParseObserveEventQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	cursor, err := ParseObserveCursor(c.GetHeader("Last-Event-ID"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	if !cursor.Timestamp.IsZero() {
		query.Since = cursor.Timestamp
	}

	initial, err := h.Observer.QueryEvents(c.Request.Context(), query)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	writer, err := PrepareSSE(c)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	cursor = EmitObserveEvents(writer, initial, cursor)

	pollQuery := query
	pollQuery.Limit = 0
	if !cursor.Timestamp.IsZero() {
		pollQuery.Since = cursor.Timestamp
	}

	ticker := time.NewTicker(h.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-h.StreamDoneChannel():
			return
		case <-ticker.C:
			if !cursor.Timestamp.IsZero() {
				pollQuery.Since = cursor.Timestamp
			}
			events, pollErr := h.Observer.QueryEvents(c.Request.Context(), pollQuery)
			if pollErr != nil {
				_ = WriteSSE(writer, SSEMessage{
					Name: "error",
					Data: contract.ErrorPayload{Error: pollErr.Error()},
				})
				return
			}
			cursor = EmitObserveEvents(writer, events, cursor)
		}
	}
}

// Health returns the daemon health snapshot plus memory health.
func (h *BaseHandlers) Health(c *gin.Context) {
	health, err := h.Observer.Health(c.Request.Context())
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	memoryHealth, err := h.memoryHealth(c)
	if err != nil {
		h.respondError(c, StatusForMemoryError(err), err)
		return
	}

	automationHealth, err := h.automationHealth(c.Request.Context())
	if err != nil {
		h.respondError(c, StatusForAutomationError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.HealthResponse{
		Health:     ObserveHealthPayloadFromHealth(health),
		Memory:     memoryHealth,
		Automation: automationHealth,
	})
}

// DaemonStatus returns the daemon status snapshot.
func (h *BaseHandlers) DaemonStatus(c *gin.Context) {
	health, err := h.Observer.Health(c.Request.Context())
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	sessions, err := h.Sessions.ListAll(c.Request.Context())
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	httpPort := h.HTTPPortValue()
	if httpPort <= 0 {
		httpPort = h.Config.HTTP.Port
	}
	networkStatus, err := h.networkStatusPayload(c.Request.Context())
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, contract.DaemonStatusResponse{
		Daemon: contract.DaemonStatusPayload{
			Status:         "running",
			PID:            h.PID(),
			StartedAt:      h.StartedAt,
			Socket:         h.Config.Daemon.Socket,
			HTTPHost:       h.Config.HTTP.Host,
			HTTPPort:       httpPort,
			UserHomeDir:    h.daemonUserHomeDir(),
			ActiveSessions: health.ActiveSessions,
			TotalSessions:  len(sessions),
			Version:        health.Version,
			Network:        networkStatus,
		},
	})
}

func (h *BaseHandlers) networkStatusPayload(ctx context.Context) (*contract.NetworkStatusPayload, error) {
	if !h.Config.Network.Enabled {
		return &contract.NetworkStatusPayload{
			Enabled: false,
			Status:  "disabled",
		}, nil
	}
	if h.Network == nil {
		return nil, errors.New("api: network service is required when network is enabled")
	}

	status, err := h.Network.Status(ctx)
	if err != nil {
		return nil, err
	}
	if status == nil {
		return nil, errors.New("api: network status is required")
	}

	return NetworkStatusPayloadFromStatus(status), nil
}

func (h *BaseHandlers) daemonUserHomeDir() string {
	userHomeDir, err := resolveUserHomeDir(h.HomePaths, os.UserHomeDir)
	if err == nil {
		return userHomeDir
	}

	logger := h.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.Warn("api: daemon status user home directory unavailable", "err", err)
	return ""
}

func resolveUserHomeDir(homePaths aghconfig.HomePaths, lookupHomeDir func() (string, error)) (string, error) {
	return resolveUserHomeDirWithResolver(homePaths, lookupHomeDir, aghconfig.ResolvePath)
}

func resolveUserHomeDirWithResolver(
	homePaths aghconfig.HomePaths,
	lookupHomeDir func() (string, error),
	resolvePath func(string) (string, error),
) (string, error) {
	if resolvePath == nil {
		resolvePath = aghconfig.ResolvePath
	}

	if lookupHomeDir != nil {
		userHomeDir, err := lookupHomeDir()
		if err == nil {
			resolvedUserHomeDir, resolveErr := resolvePath(userHomeDir)
			if resolveErr == nil && strings.TrimSpace(resolvedUserHomeDir) != "" {
				return resolvedUserHomeDir, nil
			}
			if fallback, ok := fallbackUserHomeDir(homePaths); ok {
				return fallback, nil
			}
			if resolveErr != nil {
				return "", fmt.Errorf("resolve user home directory: %w", resolveErr)
			}
			return "", nil
		}
		if fallback, ok := fallbackUserHomeDir(homePaths); ok {
			return fallback, nil
		}
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}

	if fallback, ok := fallbackUserHomeDir(homePaths); ok {
		return fallback, nil
	}
	return "", nil
}

func fallbackUserHomeDir(homePaths aghconfig.HomePaths) (string, bool) {
	homeDir := strings.TrimSpace(homePaths.HomeDir)
	if homeDir == "" || filepath.Base(homeDir) != aghconfig.DirName {
		return "", false
	}

	parent := filepath.Dir(homeDir)
	if parent == "." || parent == homeDir || strings.TrimSpace(parent) == "" {
		return "", false
	}
	return parent, true
}

// HTTPPortValue returns the configured HTTP port in a concurrency-safe way.
func (h *BaseHandlers) HTTPPortValue() int {
	if h == nil {
		return 0
	}
	return int(h.httpPort.Load())
}

// StreamDoneChannel returns the transport shutdown channel in a concurrency-safe way.
func (h *BaseHandlers) StreamDoneChannel() <-chan struct{} {
	if h == nil {
		return nil
	}
	h.settingsMu.RLock()
	defer h.settingsMu.RUnlock()
	return h.streamDone
}

func (h *BaseHandlers) respondError(c *gin.Context, status int, err error) {
	RespondError(c, status, err, h.MaskInternalErrors)
}

func (h *BaseHandlers) transportName() string {
	if strings.TrimSpace(h.TransportName) == "" {
		return "apicore"
	}
	return h.TransportName
}

func (h *BaseHandlers) sessionEventInfo(ctx context.Context, id string) (*session.SessionInfo, error) {
	if !h.IncludeSessionWorkspaceInSSE {
		return nil, nil
	}
	return h.Sessions.Status(ctx, id)
}

func (h *BaseHandlers) streamSessionInfo(ctx context.Context, id string) (*session.SessionInfo, error) {
	if h.IncludeSessionWorkspaceInSSE {
		return h.Sessions.Status(ctx, id)
	}
	_, err := h.Sessions.Status(ctx, id)
	return nil, err
}
