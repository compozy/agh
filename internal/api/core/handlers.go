package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/events"
	"github.com/pedronauck/agh/internal/memory"
	authproviders "github.com/pedronauck/agh/internal/providers"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/transcript"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const (
	handlersErrorKey = "error"
)

const defaultPollInterval = 100 * time.Millisecond

const (
	defaultSessionAttachTTL      = 15 * time.Minute
	maxSessionAttachTTL          = 24 * time.Hour
	defaultSessionRecapLimit     = 20
	maxSessionRecapLimit         = 100
	recapConsistencyReadSnapshot = "read_snapshot"
)

var errCreateAgentRequestInvalid = errors.New("api: invalid create agent request")

// TaskActorContextResolver derives the trusted task-domain actor envelope for one API request.
type TaskActorContextResolver func(c *gin.Context, action string) (taskpkg.ActorContext, error)

// BaseHandlerConfig configures a shared handler set for one transport.
type BaseHandlerConfig struct {
	TransportName                string
	MaskInternalErrors           bool
	IncludeSessionWorkspaceInSSE bool
	Sessions                     SessionManager
	SessionCatalog               SessionCatalog
	Network                      NetworkService
	NetworkStore                 NetworkStore
	Observer                     Observer
	Resources                    ResourceService
	Tools                        ToolRegistry
	Toolsets                     ToolsetRegistry
	ToolApprovals                ToolApprovalIssuer
	Automation                   AutomationManager
	Tasks                        TaskService
	Bridges                      BridgeService
	Bundles                      BundleService
	SupportBundles               SupportBundleService
	Settings                     SettingsService
	SettingsRestart              SettingsRestartController
	SettingsUpdate               SettingsUpdateController
	Vault                        VaultService
	Workspaces                   WorkspaceService
	AgentCatalog                 AgentCatalog
	ModelCatalog                 ModelCatalogService
	ProviderAuthRunner           authproviders.ProviderAuthCommandRunner
	AgentContextService          AgentContextService
	SoulAuthoring                SoulAuthoringService
	SoulRefresher                SoulRefresher
	HeartbeatAuthoring           HeartbeatAuthoringService
	HeartbeatStatus              HeartbeatStatusService
	HeartbeatWake                HeartbeatWakeService
	SessionHealth                SessionHealthReader
	HeartbeatWakeEvents          HeartbeatWakeEventReader
	CoordinatorConfig            CoordinatorConfigResolver
	SkillsRegistry               SkillsRegistry
	SkillMarketplace             SkillMarketplaceService
	TaskActorContextResolver     TaskActorContextResolver
	MemoryStore                  *memory.Store
	DreamTrigger                 DreamTrigger
	MemoryExtractor              MemoryExtractorService
	MemoryProviders              MemoryProviderService
	MemorySessionLedger          MemorySessionLedgerService
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
	SessionCatalog               SessionCatalog
	Network                      NetworkService
	NetworkStore                 NetworkStore
	Observer                     Observer
	Resources                    ResourceService
	Tools                        ToolRegistry
	Toolsets                     ToolsetRegistry
	ToolApprovals                ToolApprovalIssuer
	Automation                   AutomationManager
	Tasks                        TaskService
	Bridges                      BridgeService
	Bundles                      BundleService
	SupportBundles               SupportBundleService
	Settings                     SettingsService
	SettingsRestart              SettingsRestartController
	SettingsUpdate               SettingsUpdateController
	Vault                        VaultService
	Workspaces                   WorkspaceService
	AgentCatalog                 AgentCatalog
	ModelCatalog                 ModelCatalogService
	ProviderAuthRunner           authproviders.ProviderAuthCommandRunner
	AgentContextService          AgentContextService
	SoulAuthoring                SoulAuthoringService
	SoulRefresher                SoulRefresher
	HeartbeatAuthoring           HeartbeatAuthoringService
	HeartbeatStatus              HeartbeatStatusService
	HeartbeatWake                HeartbeatWakeService
	SessionHealth                SessionHealthReader
	HeartbeatWakeEvents          HeartbeatWakeEventReader
	CoordinatorConfig            CoordinatorConfigResolver
	SkillsRegistry               SkillsRegistry
	SkillMarketplace             SkillMarketplaceService
	TaskActorContextResolver     TaskActorContextResolver
	MemoryStore                  *memory.Store
	DreamTrigger                 DreamTrigger
	MemoryExtractor              MemoryExtractorService
	MemoryProviders              MemoryProviderService
	MemorySessionLedger          MemorySessionLedgerService
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
func NewBaseHandlers(cfg *BaseHandlerConfig) *BaseHandlers {
	if cfg == nil {
		cfg = &BaseHandlerConfig{}
	}
	defaults := normalizeBaseHandlerConfig(cfg)

	handlers := &BaseHandlers{
		TransportName:                strings.TrimSpace(cfg.TransportName),
		MaskInternalErrors:           cfg.MaskInternalErrors,
		IncludeSessionWorkspaceInSSE: cfg.IncludeSessionWorkspaceInSSE,
		Sessions:                     cfg.Sessions,
		SessionCatalog:               cfg.SessionCatalog,
		Network:                      cfg.Network,
		NetworkStore:                 cfg.NetworkStore,
		Observer:                     cfg.Observer,
		Resources:                    cfg.Resources,
		Tools:                        cfg.Tools,
		Toolsets:                     cfg.Toolsets,
		ToolApprovals:                cfg.ToolApprovals,
		Automation:                   cfg.Automation,
		Tasks:                        cfg.Tasks,
		Bridges:                      cfg.Bridges,
		Bundles:                      cfg.Bundles,
		SupportBundles:               cfg.SupportBundles,
		Settings:                     cfg.Settings,
		SettingsRestart:              cfg.SettingsRestart,
		SettingsUpdate:               cfg.SettingsUpdate,
		Vault:                        cfg.Vault,
		Workspaces:                   cfg.Workspaces,
		AgentCatalog:                 cfg.AgentCatalog,
		ModelCatalog:                 cfg.ModelCatalog,
		ProviderAuthRunner:           defaults.providerAuthRunner,
		AgentContextService:          cfg.AgentContextService,
		CoordinatorConfig:            cfg.CoordinatorConfig,
		SkillsRegistry:               cfg.SkillsRegistry,
		SkillMarketplace:             cfg.SkillMarketplace,
		TaskActorContextResolver:     cfg.TaskActorContextResolver,
		MemoryStore:                  cfg.MemoryStore,
		DreamTrigger:                 cfg.DreamTrigger,
		MemoryExtractor:              cfg.MemoryExtractor,
		MemoryProviders:              cfg.MemoryProviders,
		MemorySessionLedger:          cfg.MemorySessionLedger,
		HomePaths:                    cfg.HomePaths,
		Config:                       cfg.Config,
		Logger:                       defaults.logger,
		StartedAt:                    cfg.StartedAt,
		Now:                          defaults.now,
		PollInterval:                 cfg.PollInterval,
		AgentLoader:                  defaults.agentLoader,
		PID:                          defaults.pid,
	}
	handlers.applyAuthoredContextConfig(cfg)
	handlers.streamDone = cfg.StreamDone
	handlers.httpPort.Store(int64(cfg.HTTPPort))
	return handlers
}

type baseHandlerDefaults struct {
	logger             *slog.Logger
	now                func() time.Time
	agentLoader        AgentLoader
	pid                func() int
	providerAuthRunner authproviders.ProviderAuthCommandRunner
}

func normalizeBaseHandlerConfig(cfg *BaseHandlerConfig) baseHandlerDefaults {
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
		pid = os.Getpid
	}
	providerAuthRunner := cfg.ProviderAuthRunner
	if providerAuthRunner == nil {
		providerAuthRunner = authproviders.DefaultProviderAuthCommandRunner
	}
	if cfg.StreamDone == nil {
		logger.Warn(
			"api: stream shutdown bridge not provided; streaming handlers will rely on caller context " +
				"until a transport installs one",
		)
		cfg.StreamDone = make(chan struct{})
	}
	return baseHandlerDefaults{
		logger:             logger,
		now:                now,
		agentLoader:        agentLoader,
		pid:                pid,
		providerAuthRunner: providerAuthRunner,
	}
}

func (h *BaseHandlers) applyAuthoredContextConfig(cfg *BaseHandlerConfig) {
	if h == nil || cfg == nil {
		return
	}
	h.SoulAuthoring = cfg.SoulAuthoring
	h.SoulRefresher = cfg.SoulRefresher
	h.HeartbeatAuthoring = cfg.HeartbeatAuthoring
	h.HeartbeatStatus = cfg.HeartbeatStatus
	h.HeartbeatWake = cfg.HeartbeatWake
	h.SessionHealth = cfg.SessionHealth
	h.HeartbeatWakeEvents = cfg.HeartbeatWakeEvents
}

// SetStreamDone updates the transport shutdown bridge used by streaming handlers.
func (h *BaseHandlers) SetStreamDone(done <-chan struct{}) {
	if h == nil {
		return
	}
	if done == nil {
		if h.Logger != nil {
			h.Logger.Warn(
				"api: stream shutdown bridge cleared; streaming handlers will rely on caller context " +
					"until a transport installs one",
			)
		}
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
	workspaceFilter := strings.TrimSpace(c.Query("workspace"))
	resumable, err := parseBoolQuery(c, "resumable")
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	if resumable {
		h.listResumableSessions(c, workspaceFilter)
		return
	}
	infos, err := h.Sessions.ListAll(c.Request.Context())
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	if workspaceFilter != "" {
		workspaceID, lookupErr := h.lookupWorkspaceID(c.Request.Context(), workspaceFilter)
		if lookupErr != nil {
			h.respondError(c, StatusForWorkspaceError(lookupErr), lookupErr)
			return
		}
		infos = filterSessionInfosByWorkspaceIDInternal(infos, workspaceID)
	}
	includeHealth, err := parseBoolQuery(c, "include_health")
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	payloads, err := h.sessionPayloadsWithOptionalHealth(c.Request.Context(), infos, includeHealth)
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.SessionsResponse{Sessions: payloads})
}

func (h *BaseHandlers) listResumableSessions(c *gin.Context, workspaceFilter string) {
	if h.SessionCatalog == nil {
		h.respondError(c, http.StatusServiceUnavailable, errors.New("api: session catalog is required"))
		return
	}
	workspaceID := ""
	if workspaceFilter != "" {
		resolved, err := h.lookupWorkspaceID(c.Request.Context(), workspaceFilter)
		if err != nil {
			h.respondError(c, StatusForWorkspaceError(err), err)
			return
		}
		workspaceID = resolved
	}
	limit, err := parseOptionalPositiveIntQuery(c, "limit", 0, maxSessionRecapLimit)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	sortKey := strings.TrimSpace(c.Query("sort"))
	infos, err := h.SessionCatalog.ListSessions(c.Request.Context(), store.SessionListQuery{
		WorkspaceID: workspaceID,
		Resumable:   true,
		Sort:        sortKey,
		Limit:       limit,
	})
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	payloads := make([]contract.SessionPayload, 0, len(infos))
	for _, info := range infos {
		payloads = append(payloads, SessionPayloadFromStoreInfo(info))
	}
	c.JSON(http.StatusOK, contract.SessionsResponse{Sessions: payloads})
}

// CreateSession creates a new runtime session.
func (h *BaseHandlers) CreateSession(c *gin.Context) {
	var req contract.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode create session request: %w", h.transportName(), err),
		)
		return
	}
	if err := h.validateCreateSessionRequest(req); err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	channel, err := h.defaultSessionChannel(c.Request.Context(), strings.TrimSpace(req.Channel))
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	sess, err := h.Sessions.Create(c.Request.Context(), session.CreateOpts{
		AgentName:       req.AgentName,
		Provider:        strings.TrimSpace(req.Provider),
		Model:           strings.TrimSpace(req.Model),
		ReasoningEffort: strings.TrimSpace(req.ReasoningEffort),
		Name:            req.Name,
		Workspace:       strings.TrimSpace(req.Workspace),
		WorkspacePath:   strings.TrimSpace(req.WorkspacePath),
		Channel:         channel,
		Type:            session.SessionTypeUser,
	})
	if err != nil {
		h.respondError(c, StatusForSessionError(err), err)
		return
	}

	c.JSON(http.StatusCreated, contract.SessionResponse{Session: SessionPayloadFromInfo(sess.Info())})
}

// GetSession returns one session snapshot.
func (h *BaseHandlers) GetSession(c *gin.Context) {
	_, _, info, ok := h.routeSessionInWorkspace(c)
	if !ok {
		return
	}
	includeHealth, err := parseBoolQuery(c, "include_health")
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	payload, err := h.sessionPayloadWithOptionalHealth(c.Request.Context(), info, includeHealth)
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.SessionResponse{Session: payload})
}

// DeleteSession removes one session from the runtime catalog and persisted history.
func (h *BaseHandlers) DeleteSession(c *gin.Context) {
	_, sessionID, _, ok := h.routeSessionInWorkspace(c)
	if !ok {
		return
	}
	if err := h.Sessions.Delete(c.Request.Context(), sessionID); err != nil {
		h.respondError(c, StatusForSessionError(err), err)
		return
	}

	c.Status(http.StatusNoContent)
}

// StopSession stops a running session without deleting persisted history.
func (h *BaseHandlers) StopSession(c *gin.Context) {
	_, sessionID, _, ok := h.routeSessionInWorkspace(c)
	if !ok {
		return
	}
	if err := h.Sessions.Stop(c.Request.Context(), sessionID); err != nil {
		h.respondError(c, StatusForSessionError(err), err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ResumeSession attaches a caller to an eligible live session.
func (h *BaseHandlers) ResumeSession(c *gin.Context) {
	h.AttachSession(c)
}

// AttachSession acquires a short-lived attach lease without starting a new runtime authority.
func (h *BaseHandlers) AttachSession(c *gin.Context) {
	if h.SessionCatalog == nil {
		h.respondError(c, http.StatusServiceUnavailable, errors.New("api: session catalog is required"))
		return
	}
	_, sessionID, info, ok := h.routeSessionInWorkspace(c)
	if !ok {
		return
	}
	var req contract.AttachSessionRequest
	if err := decodeOptionalJSONBody(c, &req); err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	ttl := defaultSessionAttachTTL
	if req.TTLSeconds > 0 {
		ttl = time.Duration(req.TTLSeconds) * time.Second
	}
	if ttl > maxSessionAttachTTL {
		h.respondError(c, http.StatusBadRequest, fmt.Errorf("attach ttl must be <= %s", maxSessionAttachTTL))
		return
	}
	attachedTo := strings.TrimSpace(req.AttachedTo)
	if attachedTo == "" {
		attachedTo = fmt.Sprintf("%s:%d", h.transportName(), h.PID())
	}
	attach, err := h.SessionCatalog.AttachSession(c.Request.Context(), store.SessionAttachRequest{
		SessionID:  sessionID,
		AttachedTo: attachedTo,
		Now:        h.Now(),
		TTL:        ttl,
	})
	if err != nil {
		h.respondError(c, StatusForSessionError(err), err)
		return
	}

	payload := SessionPayloadFromInfo(info)
	payload.AttachedTo = attach.AttachedTo
	payload.AttachExpiresAt = &attach.AttachExpiresAt
	c.JSON(http.StatusOK, contract.SessionAttachResponse{
		Session: payload,
		Attach: contract.SessionAttachPayload{
			SessionID:       attach.SessionID,
			AttachedTo:      attach.AttachedTo,
			AttachExpiresAt: attach.AttachExpiresAt,
			AttachedAt:      attach.AttachedAt,
		},
	})
}

func decodeOptionalJSONBody(c *gin.Context, dest any) error {
	if c.Request.Body == nil {
		return nil
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return fmt.Errorf("read request body: %w", err)
	}
	if strings.TrimSpace(string(body)) == "" {
		return nil
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("decode request body: %w", err)
	}
	return nil
}

// RepairSession inspects and optionally repairs an interrupted persisted session transcript.
func (h *BaseHandlers) RepairSession(c *gin.Context) {
	_, sessionID, _, ok := h.routeSessionInWorkspace(c)
	if !ok {
		return
	}
	dryRun, err := repairBoolQuery(c, "dry_run", "dry-run")
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	force, err := repairBoolQuery(c, "force")
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.Sessions.RepairSession(c.Request.Context(), session.RepairOpts{
		SessionID: sessionID,
		DryRun:    dryRun,
		Force:     force,
	})
	if err != nil {
		h.respondError(c, StatusForSessionError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.SessionRepairResponse{Repair: SessionRepairPayloadFromResult(result)})
}

// ClearSessionConversation clears persisted conversation history and restarts the
// session with a fresh ACP conversation context while preserving the same id.
func (h *BaseHandlers) ClearSessionConversation(c *gin.Context) {
	_, sessionID, _, ok := h.routeSessionInWorkspace(c)
	if !ok {
		return
	}
	sess, err := h.Sessions.ClearConversation(c.Request.Context(), sessionID)
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
	if lastEventID := strings.TrimSpace(c.GetHeader("Last-Event-ID")); lastEventID != "" {
		query.AfterSequence, err = parseLastEventID(lastEventID, h.transportName())
		if err != nil {
			h.respondError(c, http.StatusBadRequest, err)
			return
		}
	}

	_, sessionID, info, ok := h.routeSessionInWorkspace(c)
	if !ok {
		return
	}
	if !h.IncludeSessionWorkspaceInSSE {
		info = nil
	}

	events, err := h.Sessions.Events(c.Request.Context(), sessionID, query)
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

	_, sessionID, info, ok := h.routeSessionInWorkspace(c)
	if !ok {
		return
	}
	if !h.IncludeSessionWorkspaceInSSE {
		info = nil
	}

	history, err := h.Sessions.History(c.Request.Context(), sessionID, query)
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
	_, sessionID, _, ok := h.routeSessionInWorkspace(c)
	if !ok {
		return
	}
	messages, err := h.Sessions.Transcript(c.Request.Context(), sessionID)
	if err != nil {
		h.respondError(c, StatusForSessionError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.SessionTranscriptResponse{Messages: messages})
}

// SessionRecap returns a deterministic recap composed from persisted session state.
func (h *BaseHandlers) SessionRecap(c *gin.Context) {
	_, sessionID, info, ok := h.routeSessionInWorkspace(c)
	if !ok {
		return
	}
	limit, err := parseOptionalPositiveIntQuery(c, "limit", defaultSessionRecapLimit, maxSessionRecapLimit)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	eventsList, err := h.Sessions.Events(c.Request.Context(), sessionID, store.EventQuery{Limit: maxSessionRecapLimit * 5})
	if err != nil {
		h.respondError(c, StatusForSessionError(err), err)
		return
	}
	messages, err := transcript.ToUIMessages(eventsList)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	markers, err := h.recentTranscriptMarkers(c.Request.Context(), sessionID, eventsList, 5)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	eventCursor := maxSessionEventSequence(eventsList)
	payload := contract.RecapPayload{
		Session:        SessionPayloadFromInfo(info),
		RecentMarkers:  markers,
		RecentMessages: recentUIMessages(messages, limit),
		PendingInputs:  0,
		PendingMarkers: 0,
		Snapshot: contract.RecapSnapshotPayload{
			GeneratedAt:      h.Now().UTC(),
			EventCursor:      eventCursor,
			TranscriptCursor: eventCursor,
			QueueGeneration:  0,
			Consistency:      recapConsistencyReadSnapshot,
		},
	}
	c.JSON(http.StatusOK, contract.SessionRecapResponse{Recap: payload})
}

func (h *BaseHandlers) recentTranscriptMarkers(
	ctx context.Context,
	sessionID string,
	eventsList []store.SessionEvent,
	limit int,
) ([]contract.TranscriptMarkerPayload, error) {
	if limit <= 0 {
		return []contract.TranscriptMarkerPayload{}, nil
	}
	if h.Observer != nil {
		summaries, err := h.Observer.QueryEvents(ctx, store.EventSummaryQuery{
			SessionID: sessionID,
			Type:      events.TranscriptMarkerCreated,
			Limit:     limit,
		})
		if err != nil {
			return nil, fmt.Errorf("api: query transcript marker summaries: %w", err)
		}
		markers := markerPayloadsFromSummaries(summaries)
		if len(markers) > 0 {
			return markers, nil
		}
	}
	return markerPayloadsFromEvents(eventsList, limit), nil
}

func markerPayloadsFromSummaries(summaries []store.EventSummary) []contract.TranscriptMarkerPayload {
	markers := make([]contract.TranscriptMarkerPayload, 0, len(summaries))
	for _, summary := range summaries {
		marker, ok := transcript.ParseMarker(summary.Content)
		if !ok {
			continue
		}
		markers = append(markers, transcriptMarkerPayload(marker))
	}
	return markers
}

func markerPayloadsFromEvents(eventsList []store.SessionEvent, limit int) []contract.TranscriptMarkerPayload {
	markers := make([]contract.TranscriptMarkerPayload, 0, limit)
	for index := len(eventsList) - 1; index >= 0 && len(markers) < limit; index-- {
		event := eventsList[index]
		if event.Type != events.TranscriptMarkerCreated && event.Type != events.TranscriptMarkerRedacted {
			continue
		}
		agentEvent, err := transcript.UnmarshalAgentEvent(event.Content)
		if err != nil {
			continue
		}
		marker, ok := transcript.ParseMarker(agentEvent.Raw)
		if !ok {
			continue
		}
		markers = append(markers, transcriptMarkerPayload(marker))
	}
	return markers
}

func transcriptMarkerPayload(marker transcript.Marker) contract.TranscriptMarkerPayload {
	normalized := marker.Normalize()
	return contract.TranscriptMarkerPayload{
		Kind:       normalized.Kind,
		OccurredAt: normalized.OccurredAt,
		Summary:    normalized.Summary,
		Evidence:   normalized.Evidence,
		Diagnostic: normalized.Diagnostic,
	}
}

func recentUIMessages(messages []transcript.UIMessage, limit int) []transcript.UIMessage {
	if len(messages) == 0 || limit == 0 {
		return []transcript.UIMessage{}
	}
	if limit < 0 || limit >= len(messages) {
		return append([]transcript.UIMessage(nil), messages...)
	}
	return append([]transcript.UIMessage(nil), messages[len(messages)-limit:]...)
}

func maxSessionEventSequence(eventsList []store.SessionEvent) int64 {
	var maxSequence int64
	for _, event := range eventsList {
		if event.Sequence > maxSequence {
			maxSequence = event.Sequence
		}
	}
	return maxSequence
}

func repairBoolQuery(c *gin.Context, names ...string) (bool, error) {
	var (
		value bool
		seen  bool
	)
	for _, name := range names {
		raw, ok := c.GetQuery(name)
		if !ok {
			continue
		}
		parsed, err := ParseOptionalBool(raw)
		if err != nil {
			return false, fmt.Errorf("invalid %s query: %w", name, err)
		}
		if seen && parsed != value {
			return false, fmt.Errorf(
				"conflicting boolean query values for %s",
				strings.Join(names, ", "),
			)
		}
		value = parsed
		seen = true
	}
	return value, nil
}

func parseOptionalPositiveIntQuery(c *gin.Context, name string, fallback int, maxValue int) (int, error) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s query: %w", name, err)
	}
	if value < 0 {
		return 0, fmt.Errorf("invalid %s query: must be zero or positive", name)
	}
	if maxValue > 0 && value > maxValue {
		return 0, fmt.Errorf("invalid %s query: must be <= %d", name, maxValue)
	}
	return value, nil
}

// StreamSession streams session events over SSE.
func (h *BaseHandlers) StreamSession(c *gin.Context) {
	_, sessionID, info, ok := h.routeSessionInWorkspace(c)
	if !ok {
		return
	}
	if !h.IncludeSessionWorkspaceInSSE {
		info = nil
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

	initial, err := h.Sessions.Events(c.Request.Context(), sessionID, query)
	if err != nil {
		h.respondError(c, StatusForSessionError(err), err)
		return
	}

	writer, err := PrepareSSE(c)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	afterSequence := query.AfterSequence
	nextSequence, err := h.writeSessionEventBatch(writer, initial, info)
	if err != nil {
		return
	}
	if nextSequence > afterSequence {
		afterSequence = nextSequence
	}

	pollQuery := query
	pollQuery.Limit = 0
	h.pollAndStreamSessionEvents(c, writer, sessionID, info, pollQuery, afterSequence)
}

// ListAgents returns all readable agent definitions in home paths.
func (h *BaseHandlers) ListAgents(c *gin.Context) {
	if workspaceRef := strings.TrimSpace(c.Query("workspace")); workspaceRef != "" {
		agentDefs, diagnostics, err := h.workspaceAgentDefsWithDiagnostics(c.Request.Context(), workspaceRef)
		if err != nil {
			h.respondError(c, statusForAgentWorkspaceError(err), err)
			return
		}
		h.respondAgentDefs(c, agentDefs, diagnostics)
		return
	}

	if h.AgentCatalog != nil {
		agentDefs, err := h.AgentCatalog.ListAgents(c.Request.Context())
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				c.JSON(http.StatusOK, contract.AgentsResponse{Agents: []contract.AgentPayload{}})
				return
			}
			h.respondError(c, http.StatusInternalServerError, err)
			return
		}
		h.respondAgentDefs(c, agentDefs)
		return
	}

	entries, err := os.ReadDir(h.HomePaths.AgentsDir)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		c.JSON(http.StatusOK, contract.AgentsResponse{Agents: []contract.AgentPayload{}})
		return
	default:
		h.respondError(
			c,
			http.StatusInternalServerError,
			fmt.Errorf("%s: read agents directory %q: %w", h.transportName(), h.HomePaths.AgentsDir, err),
		)
		return
	}

	agentDefs := make([]aghconfig.AgentDef, 0, len(entries))
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
			h.Logger.Warn(
				h.transportName()+": skip unreadable agent definition",
				"agent_name",
				name,
				handlersErrorKey,
				loadErr,
			)
			continue
		}
		agentDefs = append(agentDefs, agent)
	}

	h.respondAgentDefs(c, agentDefs)
}

// CreateAgent writes a new global or workspace-local AGENT.md definition.
func (h *BaseHandlers) CreateAgent(c *gin.Context) {
	var req contract.CreateAgentRequest
	if err := decodeStrictCreateAgentRequest(c, &req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode create agent request: %w", h.transportName(), err),
		)
		return
	}

	draft, path, err := h.createAgentDraftAndPath(c.Request.Context(), req)
	if err != nil {
		h.respondError(c, statusForCreateAgentError(err), err)
		return
	}

	agent, err := aghconfig.CreateAgentDefFile(path, draft, false)
	if err != nil {
		h.respondError(c, statusForCreateAgentError(err), err)
		return
	}
	c.JSON(http.StatusCreated, contract.AgentResponse{Agent: AgentPayloadFromDef(agent)})
}

// GetAgent returns one agent definition by name.
func (h *BaseHandlers) GetAgent(c *gin.Context) {
	if workspaceRef := strings.TrimSpace(c.Query("workspace")); workspaceRef != "" {
		agent, err := h.workspaceAgentDef(c.Request.Context(), workspaceRef, c.Param("name"))
		if err != nil {
			h.respondError(c, statusForAgentWorkspaceError(err), err)
			return
		}
		c.JSON(http.StatusOK, contract.AgentResponse{Agent: AgentPayloadFromDef(agent)})
		return
	}

	if h.AgentCatalog != nil {
		agent, err := h.AgentCatalog.GetAgent(c.Request.Context(), c.Param("name"))
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, os.ErrNotExist) {
				status = http.StatusNotFound
			}
			h.respondError(c, status, err)
			return
		}
		c.JSON(http.StatusOK, contract.AgentResponse{Agent: AgentPayloadFromDef(agent)})
		return
	}

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

func decodeStrictCreateAgentRequest(c *gin.Context, req *contract.CreateAgentRequest) error {
	if c == nil || c.Request == nil || c.Request.Body == nil {
		return io.EOF
	}
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(req); err != nil {
		return err
	}
	var extra struct{}
	if err := decoder.Decode(&extra); err != nil && !errors.Is(err, io.EOF) {
		return err
	} else if err == nil {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func (h *BaseHandlers) createAgentDraftAndPath(
	ctx context.Context,
	req contract.CreateAgentRequest,
) (aghconfig.AgentDefinitionDraft, string, error) {
	draft, err := createAgentDraftFromRequest(req)
	if err != nil {
		return aghconfig.AgentDefinitionDraft{}, "", err
	}

	path, err := h.createAgentDefinitionPath(ctx, req)
	if err != nil {
		return aghconfig.AgentDefinitionDraft{}, "", err
	}
	return draft, path, nil
}

func createAgentDraftFromRequest(req contract.CreateAgentRequest) (aghconfig.AgentDefinitionDraft, error) {
	agent := req.Agent
	if strings.TrimSpace(agent.Name) == "" {
		return aghconfig.AgentDefinitionDraft{}, errors.Join(
			errCreateAgentRequestInvalid,
			errors.New("agent.name is required"),
		)
	}
	if strings.TrimSpace(agent.Provider) == "" {
		return aghconfig.AgentDefinitionDraft{}, errors.Join(
			errCreateAgentRequestInvalid,
			errors.New("agent.provider is required"),
		)
	}
	if strings.TrimSpace(agent.Prompt) == "" {
		return aghconfig.AgentDefinitionDraft{}, errors.Join(
			errCreateAgentRequestInvalid,
			errors.New("agent.prompt is required"),
		)
	}
	disabledSkills := []string(nil)
	if agent.Skills != nil {
		disabledSkills = append([]string(nil), agent.Skills.Disabled...)
	}
	return aghconfig.AgentDefinitionDraft{
		Name:         agent.Name,
		Provider:     agent.Provider,
		Command:      agent.Command,
		Model:        agent.Model,
		Tools:        append([]string(nil), agent.Tools...),
		Toolsets:     append([]string(nil), agent.Toolsets...),
		DenyTools:    append([]string(nil), agent.DenyTools...),
		Permissions:  string(agent.Permissions),
		Skills:       aghconfig.AgentSkillsConfig{Disabled: disabledSkills},
		CategoryPath: append([]string(nil), agent.CategoryPath...),
		Prompt:       agent.Prompt,
	}, nil
}

func (h *BaseHandlers) createAgentDefinitionPath(
	ctx context.Context,
	req contract.CreateAgentRequest,
) (string, error) {
	name := aghconfig.NormalizeAgentName(req.Agent.Name)
	switch req.Scope {
	case contract.AgentCreateScopeGlobal:
		return filepath.Join(h.HomePaths.AgentsDir, name, aghconfig.AgentDefinitionFileName), nil
	case contract.AgentCreateScopeWorkspace:
		workspaceRef := strings.TrimSpace(req.Workspace)
		if workspaceRef == "" {
			return "", errors.Join(
				errCreateAgentRequestInvalid,
				errors.New("workspace is required for workspace-scoped agents"),
			)
		}
		if h.Workspaces == nil {
			return "", fmt.Errorf("%s: %w", h.transportName(), workspacepkg.ErrWorkspaceResolverUnavailable)
		}
		resolved, err := h.Workspaces.Resolve(ctx, workspaceRef)
		if err != nil {
			return "", err
		}
		rootDir := strings.TrimSpace(resolved.RootDir)
		if rootDir == "" {
			return "", fmt.Errorf("%s: %w", h.transportName(), workspacepkg.ErrWorkspaceRootMissing)
		}
		return filepath.Join(
			rootDir,
			aghconfig.DirName,
			aghconfig.AgentsDirName,
			name,
			aghconfig.AgentDefinitionFileName,
		), nil
	default:
		return "", errors.Join(
			errCreateAgentRequestInvalid,
			fmt.Errorf(
				"scope must be %q or %q",
				contract.AgentCreateScopeWorkspace,
				contract.AgentCreateScopeGlobal,
			),
		)
	}
}

func (h *BaseHandlers) workspaceAgentDefs(ctx context.Context, workspaceRef string) ([]aghconfig.AgentDef, error) {
	agents, _, err := h.workspaceAgentDefsWithDiagnostics(ctx, workspaceRef)
	return agents, err
}

func (h *BaseHandlers) workspaceAgentDefsWithDiagnostics(
	ctx context.Context,
	workspaceRef string,
) ([]aghconfig.AgentDef, []workspacepkg.AgentDiagnostic, error) {
	if h.Workspaces == nil {
		return nil, nil, fmt.Errorf("%s: %w", h.transportName(), workspacepkg.ErrWorkspaceResolverUnavailable)
	}
	resolved, err := h.Workspaces.Resolve(ctx, workspaceRef)
	if err != nil {
		return nil, nil, err
	}
	agents, err := h.workspaceDetailAgents(ctx, &resolved)
	if err != nil {
		return nil, nil, err
	}
	return agents, append([]workspacepkg.AgentDiagnostic(nil), resolved.AgentDiagnostics...), nil
}

func (h *BaseHandlers) workspaceAgentDef(
	ctx context.Context,
	workspaceRef string,
	name string,
) (aghconfig.AgentDef, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return aghconfig.AgentDef{}, fmt.Errorf("%s: agent name is required: %w", h.transportName(), os.ErrNotExist)
	}

	agents, err := h.workspaceAgentDefs(ctx, workspaceRef)
	if err != nil {
		return aghconfig.AgentDef{}, err
	}
	for _, agent := range agents {
		if strings.TrimSpace(agent.Name) == trimmedName {
			return agent, nil
		}
	}
	return aghconfig.AgentDef{}, fmt.Errorf(
		"%s: agent %q is not available in workspace %q: %w",
		h.transportName(),
		trimmedName,
		strings.TrimSpace(workspaceRef),
		workspacepkg.ErrAgentNotAvailable,
	)
}

func (h *BaseHandlers) respondAgentDefs(
	c *gin.Context,
	agentDefs []aghconfig.AgentDef,
	diagnostics ...[]workspacepkg.AgentDiagnostic,
) {
	diagnosticCount := 0
	for _, group := range diagnostics {
		diagnosticCount += len(group)
	}
	agents := make([]contract.AgentPayload, 0, len(agentDefs)+diagnosticCount)
	for _, agent := range agentDefs {
		agents = append(agents, AgentPayloadFromDef(agent))
	}
	for _, group := range diagnostics {
		for _, diagnostic := range group {
			agents = append(agents, AgentPayloadFromDiagnostic(diagnostic))
		}
	}
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Name < agents[j].Name
	})
	c.JSON(http.StatusOK, contract.AgentsResponse{Agents: agents})
}

func statusForAgentWorkspaceError(err error) int {
	switch {
	case errors.Is(err, workspacepkg.ErrAgentNotAvailable), errors.Is(err, os.ErrNotExist):
		return http.StatusNotFound
	default:
		return StatusForWorkspaceError(err)
	}
}

func statusForCreateAgentError(err error) int {
	switch {
	case errors.Is(err, errCreateAgentRequestInvalid),
		errors.Is(err, aghconfig.ErrInvalidAgentDefinition):
		return http.StatusBadRequest
	case errors.Is(err, aghconfig.ErrAgentDefinitionExists):
		return http.StatusConflict
	case errors.Is(err, workspacepkg.ErrWorkspaceNotFound),
		errors.Is(err, workspacepkg.ErrWorkspaceRootMissing),
		errors.Is(err, workspacepkg.ErrWorkspaceNameTaken),
		errors.Is(err, workspacepkg.ErrWorkspacePathTaken),
		errors.Is(err, workspacepkg.ErrWorkspaceHasSessions),
		errors.Is(err, workspacepkg.ErrWorkspaceResolverUnavailable):
		return StatusForWorkspaceError(err)
	default:
		return http.StatusInternalServerError
	}
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
		filter.WorkspaceID = strings.TrimSpace(resolved.WorkspaceID)
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
		h.respondError(c, http.StatusBadRequest, errors.New("session query is required"))
		return
	}

	scope, ok := h.resolveWorkspaceScope(c)
	if !ok {
		return
	}
	if _, err := h.requireSessionInWorkspace(
		c.Request.Context(),
		scope.SessionWorkspaceID(),
		query.SessionID,
	); err != nil {
		h.respondError(c, statusForWorkspaceScopedResourceError(err), err)
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

// ListLogs returns the filtered runtime log list.
func (h *BaseHandlers) ListLogs(c *gin.Context) {
	query, err := ParseLogsQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	events, err := h.Observer.QueryEvents(c.Request.Context(), query)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	payload := make([]contract.LogEventPayload, 0, len(events))
	for _, event := range events {
		payload = append(payload, LogEventPayloadFromSummary(event))
	}

	c.JSON(http.StatusOK, contract.LogsListResponse{Events: payload})
}

// StreamLogs streams runtime logs over SSE.
func (h *BaseHandlers) StreamLogs(c *gin.Context) {
	query, err := ParseLogsQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	cursor, err := ParseLogsCursor(c.GetHeader("Last-Event-ID"))
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

	cursor = EmitLogs(writer, initial, cursor)

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
				h.writeSSEBestEffort(writer, SSEMessage{
					Name: handlersErrorKey,
					Data: ErrorPayloadForError(pollErr),
				})
				return
			}
			cursor = EmitLogs(writer, events, cursor)
		}
	}
}

func (h *BaseHandlers) networkStatusPayload(ctx context.Context) (*contract.NetworkStatusPayload, error) {
	if !h.Config.Network.Enabled {
		return &contract.NetworkStatusPayload{
			Enabled: false,
			Status:  memoryHealthStatusDisabled,
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

	payload := NetworkStatusPayloadFromStatus(status)
	if h.Bundles == nil {
		return payload, nil
	}

	settings, err := h.Bundles.NetworkSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("api: load bundle network settings: %w", err)
	}
	payload.ConfiguredDefaultChannel = strings.TrimSpace(settings.ConfiguredDefaultChannel)
	payload.EffectiveDefaultChannel = strings.TrimSpace(settings.EffectiveDefaultChannel)
	payload.EffectiveDefaultSource = strings.TrimSpace(settings.EffectiveDefaultSource)
	payload.DeclaredChannels = DeclaredNetworkChannelPayloads(settings.DeclaredChannels)
	return payload, nil
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
