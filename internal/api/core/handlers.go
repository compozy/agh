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
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/session"
	"github.com/compozy/agh/internal/store"
	workspacepkg "github.com/compozy/agh/internal/workspace"
	"github.com/gin-gonic/gin"
)

const (
	handlersErrorKey     = "error"
	handlersOperationKey = "operation"
)

const defaultPollInterval = 100 * time.Millisecond

const (
	defaultSessionAttachTTL        = 15 * time.Minute
	maxSessionAttachTTL            = 24 * time.Hour
	defaultSessionRecapLimit       = 20
	maxSessionRecapLimit           = 100
	pendingTranscriptMarkerLimit   = maxSessionRecapLimit * 5
	recapConsistencyPersistedReads = "persisted_reads"
)

var errCreateAgentRequestInvalid = errors.New("api: invalid create agent request")

// SetHTTPPort overrides the reported HTTP port for daemon status responses.
func (h *BaseHandlers) SetHTTPPort(port int) {
	if h == nil || port <= 0 {
		return
	}
	h.httpPort.Store(int64(port))
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
		h.respondError(c, statusForCreateSessionValidationError(err), err)
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
		ReasoningEffort: strings.TrimSpace(string(req.ReasoningEffort)),
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
	attacher, ok := h.Sessions.(SessionAttachManager)
	if !ok {
		h.respondError(c, http.StatusServiceUnavailable, errors.New("api: session attach manager is required"))
		return
	}
	_, sessionID, _, ok := h.routeSessionInWorkspace(c)
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
	attach, err := attacher.AttachSession(c.Request.Context(), store.SessionAttachRequest{
		SessionID:  sessionID,
		AttachedTo: attachedTo,
		Now:        h.Now(),
		TTL:        ttl,
	})
	if err != nil {
		h.respondError(c, StatusForSessionError(err), err)
		return
	}

	info, err := h.Sessions.Status(c.Request.Context(), sessionID)
	if err != nil {
		h.respondError(c, StatusForSessionError(err), err)
		return
	}
	payload := sessionPayloadFromInfoAt(info, attach.AttachedAt)
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
	if err := rejectTranscriptBackwardCursor(c); err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	query, err := ParseSessionEventQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	query, err = applyBoundedSessionReadDefault(query)
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
		h.logSessionReadError(
			"events",
			sessionID,
			err,
			"after_sequence",
			query.AfterSequence,
			"type",
			query.Type,
			"turn_id",
			query.TurnID,
		)
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
	if err := rejectTranscriptBackwardCursor(c); err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	query, err := ParseSessionEventQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	query, err = applyBoundedSessionReadDefault(query)
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
		h.logSessionReadError("history", sessionID, err)
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

func decodeStrictCreateAgentRequest(c *gin.Context, req *contract.CreateAgentRequest) error {
	return decodeStrictJSONBody(c, req)
}

func decodeStrictJSONBody(c *gin.Context, target any) error {
	if c == nil || c.Request == nil || c.Request.Body == nil {
		return io.EOF
	}
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
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
) (aghconfig.AgentDefinitionDraft, string, string, error) {
	draft, err := createAgentDraftFromRequest(req)
	if err != nil {
		return aghconfig.AgentDefinitionDraft{}, "", "", err
	}

	target, err := createAgentDefinitionTargetFor(ctx, req, h.HomePaths, h.Workspaces, h.transportName())
	if err != nil {
		return aghconfig.AgentDefinitionDraft{}, "", "", err
	}
	return draft, target.Path, target.WorkspaceID, nil
}

func (h *BaseHandlers) createAgentDefinitionPath(
	ctx context.Context,
	req contract.CreateAgentRequest,
) (string, error) {
	return createAgentDefinitionPathFor(ctx, req, h.HomePaths, h.Workspaces, h.transportName())
}

func (h *BaseHandlers) workspaceAgentEntriesWithDiagnostics(
	ctx context.Context,
	workspaceRef string,
) ([]AgentCatalogEntry, string, []workspacepkg.AgentDiagnostic, error) {
	if h.Workspaces == nil {
		return nil, "", nil, fmt.Errorf("api: %w", workspacepkg.ErrWorkspaceResolverUnavailable)
	}
	resolved, err := h.Workspaces.Resolve(ctx, workspaceRef)
	if err != nil {
		return nil, "", nil, err
	}
	entries, err := h.workspaceDetailAgentEntries(ctx, &resolved)
	if err != nil {
		return nil, "", nil, err
	}
	return entries,
		strings.TrimSpace(resolved.ID),
		append([]workspacepkg.AgentDiagnostic(nil), resolved.AgentDiagnostics...),
		nil
}

func (h *BaseHandlers) workspaceAgentDef(
	ctx context.Context,
	workspaceRef string,
	name string,
) (AgentCatalogEntry, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return AgentCatalogEntry{}, fmt.Errorf("api: agent name is required: %w", os.ErrNotExist)
	}

	entries, workspaceID, _, err := h.workspaceAgentEntriesWithDiagnostics(ctx, workspaceRef)
	if err != nil {
		return AgentCatalogEntry{}, err
	}
	for _, entry := range entries {
		if strings.TrimSpace(entry.Def.Name) == trimmedName {
			if entry.Origin == contract.AgentOriginWorkspace {
				entry.WorkspaceID = workspaceID
			}
			return entry, nil
		}
	}
	return AgentCatalogEntry{}, fmt.Errorf(
		"api: agent %q is not available in workspace %q: %w",
		trimmedName,
		strings.TrimSpace(workspaceRef),
		workspacepkg.ErrAgentNotAvailable,
	)
}

func (h *BaseHandlers) respondAgentDefs(
	c *gin.Context,
	agentDefs []aghconfig.AgentDef,
	workspaceID string,
	diagnostics ...[]workspacepkg.AgentDiagnostic,
) {
	entries := make([]AgentCatalogEntry, 0, len(agentDefs))
	for _, agent := range agentDefs {
		entries = append(entries, h.agentCatalogEntryFromDef(agent, workspaceID))
	}
	h.respondAgentEntries(c, entries, workspaceID, diagnostics...)
}

func (h *BaseHandlers) respondAgentEntries(
	c *gin.Context,
	entries []AgentCatalogEntry,
	diagnosticWorkspaceID string,
	diagnostics ...[]workspacepkg.AgentDiagnostic,
) {
	diagnosticCount := 0
	for _, group := range diagnostics {
		diagnosticCount += len(group)
	}
	agents := make([]contract.AgentPayload, 0, len(entries)+diagnosticCount)
	for _, entry := range entries {
		if !aghconfig.IsPublicAgentDef(entry.Def) {
			continue
		}
		agents = append(agents, AgentPayloadFromEntry(entry))
	}
	for _, group := range diagnostics {
		for _, diagnostic := range group {
			if aghconfig.IsInternalManagedAgentName(diagnostic.Name) {
				continue
			}
			agents = append(agents, AgentPayloadFromDiagnostic(diagnostic, diagnosticWorkspaceID))
		}
	}
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Name < agents[j].Name
	})
	c.JSON(http.StatusOK, contract.AgentsResponse{Agents: agents})
}

func (h *BaseHandlers) agentCatalogEntryFromDef(
	agent aghconfig.AgentDef,
	workspaceID string,
) AgentCatalogEntry {
	return AgentCatalogEntryFromDef(h.HomePaths, agent, strings.TrimSpace(workspaceID))
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
	if h.Observer == nil {
		h.respondError(c, http.StatusServiceUnavailable, errors.New("api: observer is required"))
		return
	}
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
	userHomeDir, err := aghconfig.ResolveOperatorHomeDir(h.HomePaths)
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

func (h *BaseHandlers) logSessionReadError(operation string, sessionID string, err error, attrs ...any) {
	if err == nil {
		return
	}
	logger := slog.Default()
	if h != nil && h.Logger != nil {
		logger = h.Logger
	}
	logAttrs := []any{
		handlersOperationKey,
		operation,
		"session_id",
		strings.TrimSpace(sessionID),
	}
	logAttrs = append(logAttrs, attrs...)
	logAttrs = append(
		logAttrs,
		handlersErrorKey,
		err,
	)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		logger.Debug("api: session read canceled", logAttrs...)
		return
	}
	logger.Warn("api: session read failed", logAttrs...)
}

func (h *BaseHandlers) transportName() string {
	if strings.TrimSpace(h.TransportName) == "" {
		return "apicore"
	}
	return h.TransportName
}
