package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/network"
	"github.com/compozy/agh/internal/network/participation"
	"github.com/compozy/agh/internal/session"
	"github.com/compozy/agh/internal/store"
	workspacepkg "github.com/compozy/agh/internal/workspace"
	"github.com/gin-gonic/gin"
)

type networkChannelAggregate struct {
	workspaceID                string
	channel                    string
	metadata                   *store.NetworkChannelEntry
	peerCount                  int
	localPeerCount             int
	remotePeerCount            int
	sessionCount               int
	messageCount               int
	presenceCount              int
	lastActivityAt             *time.Time
	lastActivitySequence       int64
	lastPresenceAt             *time.Time
	lastPresenceSequence       int64
	lastMessageAt              *time.Time
	lastMessagePreview         string
	historicalParticipantCount int
	historicalParticipants     map[string]struct{}
}

type networkTimelineMessageView struct {
	entry              store.NetworkMessageEntry
	presenceCount      int
	presenceStartedAt  *time.Time
	presenceLastSeenAt *time.Time
}

type networkMessageHistorySummary struct {
	conversation               []store.NetworkMessageEntry
	presenceEpisodes           []networkTimelineMessageView
	presenceCount              int
	lastPresenceAt             *time.Time
	historicalParticipantCount int
}

type networkChannelMetadataFields struct {
	createdAt         *time.Time
	purpose           string
	fanoutPolicy      string
	coordinatorPeerID string
	workspaceID       string
	createdBy         string
}

type networkPresenceEpisodeKey struct {
	workspaceID string
	direction   string
	channel     string
	surface     string
	threadID    string
	directID    string
	workID      string
	peerFrom    string
	peerTo      string
}

var errNetworkChannelNotFound = errors.New("api: network channel not found")

func (h *BaseHandlers) networkStoreRequired() (NetworkStore, error) {
	if h == nil || h.NetworkStore == nil {
		return nil, errors.New("api: network store is required")
	}
	return h.NetworkStore, nil
}

// CreateNetworkChannel validates and creates one new channel by starting a new session per selected agent.
func (h *BaseHandlers) CreateNetworkChannel(c *gin.Context) {
	service, err := h.networkServiceRequired()
	if err != nil {
		h.respondError(c, http.StatusServiceUnavailable, err)
		return
	}
	networkStore, err := h.networkStoreRequired()
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	var req contract.CreateNetworkChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode create network channel request: %w", h.transportName(), err),
		)
		return
	}
	scope, ok := h.resolveWorkspaceScope(c)
	if !ok {
		return
	}
	if !scope.BodyWorkspaceIDMatches(req.WorkspaceID) {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewNetworkValidationError(errors.New("workspace_id does not match path")),
		)
		return
	}
	networkWorkspaceID := scope.NetworkWorkspaceID()
	req.WorkspaceID = networkWorkspaceID

	channel, purpose, agentNames, err := h.resolveCreateNetworkChannelRequest(
		c.Request.Context(),
		req,
		&scope.Resolved,
	)
	if err != nil {
		h.respondError(c, statusForCreateNetworkChannelError(err), err)
		return
	}
	fanoutPolicy, coordinatorPeerID, ok := h.createNetworkChannelFanoutFields(c, req)
	if !ok {
		return
	}

	entry := store.NetworkChannelEntry{
		Channel:           channel,
		WorkspaceID:       networkWorkspaceID,
		Purpose:           purpose,
		FanoutPolicy:      fanoutPolicy,
		CoordinatorPeerID: coordinatorPeerID,
		CreatedBy:         agentNames[0],
	}
	detail, status, err := h.provisionNetworkChannel(
		c.Request.Context(),
		service,
		networkStore,
		entry,
		agentNames,
	)
	if err != nil {
		h.respondError(c, status, err)
		return
	}

	c.JSON(http.StatusCreated, contract.CreateNetworkChannelResponse{Channel: detail})
}

func (h *BaseHandlers) createNetworkChannelFanoutFields(
	c *gin.Context,
	req contract.CreateNetworkChannelRequest,
) (string, string, bool) {
	fanoutPolicy := store.NormalizeNetworkFanoutPolicy(req.FanoutPolicy)
	coordinatorPeerID := strings.TrimSpace(req.CoordinatorPeerID)
	if err := store.ValidateNetworkChannelFanoutConfiguration(fanoutPolicy, coordinatorPeerID); err != nil {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(err))
		return "", "", false
	}
	return fanoutPolicy, coordinatorPeerID, true
}

// NetworkChannel returns one network channel detail payload.
func (h *BaseHandlers) NetworkChannel(c *gin.Context) {
	service, err := h.networkServiceRequired()
	if err != nil {
		h.respondError(c, http.StatusServiceUnavailable, err)
		return
	}

	channel, err := normalizeNetworkChannel(c.Param("channel"))
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	scope, ok := h.resolveWorkspaceScope(c)
	if !ok {
		return
	}

	detail, err := h.networkChannelDetailPayload(c.Request.Context(), service, scope.NetworkWorkspaceID(), channel)
	if err != nil {
		if isNetworkChannelNotFound(err) {
			h.respondError(c, http.StatusNotFound, err)
			return
		}
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, contract.NetworkChannelResponse{Channel: detail})
}

// UpdateNetworkChannel mutates the channel metadata and delivery policy.
func (h *BaseHandlers) UpdateNetworkChannel(c *gin.Context) {
	service, err := h.networkServiceRequired()
	if err != nil {
		h.respondError(c, http.StatusServiceUnavailable, err)
		return
	}
	networkStore, err := h.networkStoreRequired()
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	channel, err := normalizeNetworkChannel(c.Param("channel"))
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	scope, ok := h.resolveWorkspaceScope(c)
	if !ok {
		return
	}
	var req contract.UpdateNetworkChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode update network channel request: %w", h.transportName(), err),
		)
		return
	}
	if req.Purpose == nil && req.FanoutPolicy == nil && req.CoordinatorPeerID == nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewNetworkValidationError(errors.New("network channel update must include at least one field")),
		)
		return
	}
	ref := scope.NetworkChannelRef(channel)
	patch := store.NetworkChannelPatch{UpdatedAt: h.nowUTC()}
	if req.Purpose != nil {
		purpose := strings.TrimSpace(*req.Purpose)
		patch.Purpose = &purpose
	}
	if req.FanoutPolicy != nil {
		fanoutPolicy := store.NormalizeNetworkFanoutPolicy(*req.FanoutPolicy)
		if err := store.ValidateNetworkFanoutPolicy(fanoutPolicy); err != nil {
			h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(err))
			return
		}
		patch.FanoutPolicy = &fanoutPolicy
	}
	if req.CoordinatorPeerID != nil {
		coordinatorPeerID := strings.TrimSpace(*req.CoordinatorPeerID)
		patch.CoordinatorPeerID = &coordinatorPeerID
	}
	if err := networkStore.PatchNetworkChannel(c.Request.Context(), ref, patch); err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	detail, err := h.networkChannelDetailPayload(c.Request.Context(), service, ref.WorkspaceID, ref.Channel)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, contract.NetworkChannelResponse{Channel: detail})
}

// NetworkChannelMessages returns the read-only message timeline for one network channel.
func (h *BaseHandlers) NetworkChannelMessages(c *gin.Context) {
	service, err := h.networkServiceRequired()
	if err != nil {
		h.respondError(c, http.StatusServiceUnavailable, err)
		return
	}
	networkStore, err := h.networkStoreRequired()
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	channel, err := normalizeNetworkChannel(c.Param("channel"))
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	scope, ok := h.resolveWorkspaceScope(c)
	if !ok {
		return
	}
	query, err := parseNetworkMessageQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	sessions, err := h.Sessions.ListAll(c.Request.Context())
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	networkWorkspaceID := scope.NetworkWorkspaceID()
	peers, err := service.ListPeers(c.Request.Context(), networkWorkspaceID, channel)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}

	query.WorkspaceID = networkWorkspaceID
	query.Channel = channel
	if err := query.Validate(); err != nil {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(err))
		return
	}
	storeQuery := overfetchNetworkMessageQuery(query)
	rawMessages, messages, err := h.loadPublicChannelTimeline(c.Request.Context(), networkStore, storeQuery)
	if err != nil {
		h.respondNetworkMessageError(c, err)
		return
	}

	if err := h.ensureNetworkChannelTimelineExists(
		c.Request.Context(),
		networkStore,
		scope.NetworkChannelRef(channel),
		sessions,
		peers,
		rawMessages,
	); err != nil {
		if isNetworkChannelNotFound(err) {
			h.respondError(c, http.StatusNotFound, err)
			return
		}
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	response, err := h.networkChannelMessagesResponse(messages, sessions, peers, storeQuery, query)
	if err != nil {
		h.respondNetworkMessageError(c, err)
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *BaseHandlers) networkChannelMessagesResponse(
	messages []store.NetworkMessageEntry,
	sessions []*session.Info,
	peers []network.PeerInfo,
	storeQuery store.NetworkMessageQuery,
	requestQuery store.NetworkMessageQuery,
) (contract.NetworkChannelMessagesResponse, error) {
	payload, err := networkTimelinePayloads(
		messages,
		sessionInfoMapByID(sessions),
		peerInfoMapByID(peers),
		networkTimelinePayloadQuery(storeQuery),
		h.networkPresenceWindow(),
	)
	if err != nil {
		return contract.NetworkChannelMessagesResponse{}, err
	}
	payload, page := trimNetworkMessagePayloadPage(
		payload,
		requestQuery.AfterMessageID,
		requestQuery.Limit,
	)
	return contract.NetworkChannelMessagesResponse{Messages: payload, Page: page}, nil
}

func (h *BaseHandlers) ensureNetworkChannelTimelineExists(
	ctx context.Context,
	networkStore NetworkStore,
	ref store.NetworkChannelRef,
	sessions []*session.Info,
	peers []network.PeerInfo,
	messages []store.NetworkMessageEntry,
) error {
	metadata, err := h.loadNetworkChannelMetadata(ctx, networkStore, ref)
	if err != nil {
		return fmt.Errorf("load network channel metadata: %w", err)
	}
	if len(messages) > 0 || networkChannelExists(sessions, peers, metadata, ref.WorkspaceID, ref.Channel) {
		return nil
	}
	hasHistory, err := networkChannelHasPersistedProjection(ctx, networkStore, ref.WorkspaceID, ref.Channel)
	if err != nil {
		return err
	}
	if hasHistory {
		return nil
	}
	return fmt.Errorf("%w: %s", errNetworkChannelNotFound, ref.Channel)
}

func (h *BaseHandlers) networkChannelMetadataForUpdate(
	ctx context.Context,
	networkStore NetworkStore,
	ref store.NetworkChannelRef,
) (store.NetworkChannelEntry, error) {
	metadata, err := h.loadNetworkChannelMetadata(ctx, networkStore, ref)
	if err != nil {
		return store.NetworkChannelEntry{}, err
	}
	if metadata != nil {
		return *metadata, nil
	}
	now := h.nowUTC()
	return store.NetworkChannelEntry{
		WorkspaceID:  strings.TrimSpace(ref.WorkspaceID),
		Channel:      strings.TrimSpace(ref.Channel),
		Purpose:      "network_channel",
		FanoutPolicy: store.NetworkFanoutPolicyCapabilityMatch,
		CreatedBy:    "api",
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// NetworkPeerMessages returns the directed message timeline for one network peer.
func (h *BaseHandlers) NetworkPeerMessages(c *gin.Context) {
	service, err := h.networkServiceRequired()
	if err != nil {
		h.respondError(c, http.StatusServiceUnavailable, err)
		return
	}
	networkStore, err := h.networkStoreRequired()
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	peerID := strings.TrimSpace(c.Param("peer_id"))
	if peerID == "" {
		err := NewNetworkValidationError(errors.New("peer_id path is required"))
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	query, err := parseNetworkMessageQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	scope, ok := h.resolveWorkspaceScope(c)
	if !ok {
		return
	}
	networkWorkspaceID := scope.NetworkWorkspaceID()
	peers, err := service.ListPeers(c.Request.Context(), networkWorkspaceID, "")
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	if _, ok := findPeerInfo(peers, peerID); !ok {
		h.respondError(c, http.StatusNotFound, fmt.Errorf("api: network peer not found: %s", peerID))
		return
	}

	sessions, err := h.Sessions.ListAll(c.Request.Context())
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	query.WorkspaceID = networkWorkspaceID
	query.PeerID = peerID
	query.DirectedOnly = !query.IncludePresence
	if err := query.Validate(); err != nil {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(err))
		return
	}
	storeQuery := overfetchNetworkMessageQuery(query)
	messages, err := h.loadVisiblePeerMessages(c.Request.Context(), networkStore, storeQuery)
	if err != nil {
		h.respondNetworkMessageError(c, err)
		return
	}

	sessionByID := sessionInfoMapByID(sessions)
	peerByID := peerInfoMapByID(peers)
	payload, err := networkTimelinePayloads(
		messages,
		sessionByID,
		peerByID,
		networkTimelinePayloadQuery(storeQuery),
		h.networkPresenceWindow(),
	)
	if err != nil {
		h.respondNetworkMessageError(c, err)
		return
	}

	payload, page := trimNetworkMessagePayloadPage(
		payload,
		query.AfterMessageID,
		query.Limit,
	)
	c.JSON(http.StatusOK, contract.NetworkPeerMessagesResponse{Messages: payload, Page: page})
}

// NetworkPeer returns one selected peer detail payload.
func (h *BaseHandlers) NetworkPeer(c *gin.Context) {
	service, err := h.networkServiceRequired()
	if err != nil {
		h.respondError(c, http.StatusServiceUnavailable, err)
		return
	}
	networkStore, err := h.networkStoreRequired()
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	peerID := strings.TrimSpace(c.Param("peer_id"))
	if peerID == "" {
		err := NewNetworkValidationError(errors.New("peer_id path is required"))
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	scope, ok := h.resolveWorkspaceScope(c)
	if !ok {
		return
	}
	peers, err := service.ListPeers(c.Request.Context(), scope.NetworkWorkspaceID(), "")
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	peer, ok := findPeerInfo(peers, peerID)
	if !ok {
		h.respondError(c, http.StatusNotFound, fmt.Errorf("api: network peer not found: %s", peerID))
		return
	}

	sessions, err := h.Sessions.ListAll(c.Request.Context())
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	auditEntries, err := h.loadPeerAuditEntries(c.Request.Context(), networkStore, peer)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	payload := NetworkPeerDetailPayloadFromInfo(
		peer,
		sessionInfoMapByID(sessions),
		summarizePeerMetrics(peer, auditEntries),
	)
	c.JSON(http.StatusOK, contract.NetworkPeerResponse{Peer: payload})
}

func (h *BaseHandlers) resolveCreateNetworkChannelRequest(
	ctx context.Context,
	req contract.CreateNetworkChannelRequest,
	resolved *workspacepkg.ResolvedWorkspace,
) (string, string, []string, error) {
	_ = ctx
	if resolved == nil {
		return "", "", nil, NewNetworkValidationError(
			errors.New("workspace is required"),
		)
	}
	channel, err := normalizeNetworkChannel(req.Channel)
	if err != nil {
		return "", "", nil, err
	}
	purpose, err := normalizeNetworkChannelPurpose(req.Purpose)
	if err != nil {
		return "", "", nil, err
	}

	workspaceID := strings.TrimSpace(resolved.ID)
	if workspaceID == "" {
		return "", "", nil, NewNetworkValidationError(
			errors.New("workspace_id is required"),
		)
	}

	agentNames, err := normalizeNetworkAgentNames(req.AgentNames)
	if err != nil {
		return "", "", nil, err
	}
	available := make(map[string]struct{}, len(resolved.Agents))
	for _, agent := range resolved.Agents {
		available[strings.TrimSpace(agent.Name)] = struct{}{}
	}
	for _, agentName := range agentNames {
		if _, ok := available[agentName]; ok {
			continue
		}
		return "", "", nil, fmt.Errorf(
			"%w: %s",
			workspacepkg.ErrAgentNotAvailable,
			agentName,
		)
	}

	return channel, purpose, agentNames, nil
}

func normalizeNetworkChannel(channel string) (string, error) {
	trimmed := strings.TrimSpace(channel)
	if trimmed == "" {
		return "", NewNetworkValidationError(errors.New("channel is required"))
	}
	if err := network.ValidateChannel(trimmed); err != nil {
		return "", err
	}
	return trimmed, nil
}

func normalizeNetworkAgentNames(agentNames []string) ([]string, error) {
	if len(agentNames) == 0 {
		return nil, NewNetworkValidationError(errors.New("agent_names is required"))
	}

	normalized := make([]string, 0, len(agentNames))
	seen := make(map[string]struct{}, len(agentNames))
	for _, raw := range agentNames {
		name := strings.TrimSpace(raw)
		if name == "" {
			return nil, NewNetworkValidationError(errors.New("agent_names entries are required"))
		}
		if _, ok := seen[name]; ok {
			return nil, NewNetworkValidationError(fmt.Errorf("agent_names contains duplicate entry %q", name))
		}
		seen[name] = struct{}{}
		normalized = append(normalized, name)
	}
	return normalized, nil
}

func normalizeNetworkChannelPurpose(purpose string) (string, error) {
	trimmed := strings.TrimSpace(purpose)
	if trimmed == "" {
		return "", NewNetworkValidationError(errors.New("purpose is required"))
	}
	return trimmed, nil
}

func rollbackCreatedNetworkSessions(ctx context.Context, sessions SessionManager, sessionIDs []string) error {
	if len(sessionIDs) == 0 {
		return nil
	}

	var rollbackErr error
	for _, sessionID := range sessionIDs {
		if strings.TrimSpace(sessionID) == "" {
			continue
		}
		rollbackErr = errors.Join(
			rollbackErr,
			sessions.StopWithCause(ctx, sessionID, session.CauseFailed, "rollback network channel creation"),
		)
	}
	return rollbackErr
}

func (h *BaseHandlers) networkChannelPayloads(
	ctx context.Context,
	service NetworkService,
	workspaceID string,
) ([]contract.NetworkChannelPayload, error) {
	if h == nil {
		return nil, errors.New("api: handlers are required")
	}
	return NetworkChannelPayloads(ctx, service, h.Sessions, h.NetworkStore, workspaceID)
}

// NetworkChannelPayloads builds the shared runtime channel projection used by transports and tools.
func NetworkChannelPayloads(
	ctx context.Context,
	service NetworkService,
	sessionsManager SessionManager,
	networkStore NetworkStore,
	workspaceID string,
) ([]contract.NetworkChannelPayload, error) {
	aggregates, err := networkChannelAggregates(ctx, service, sessionsManager, networkStore, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("api: build network channel aggregates: %w", err)
	}
	return sortedNetworkChannelPayloads(aggregates), nil
}

// networkChannelAggregates merges live peers with persisted channel/session/message state for one projection pass.
func networkChannelAggregates(
	ctx context.Context,
	service NetworkService,
	sessionsManager SessionManager,
	networkStore NetworkStore,
	workspaceID string,
) (map[string]*networkChannelAggregate, error) {
	if service == nil {
		return nil, errors.New("api: network service is required")
	}
	if networkStore == nil {
		return nil, errors.New("api: network store is required")
	}
	if sessionsManager == nil {
		return nil, errors.New("api: sessions are required")
	}
	trimmedWorkspaceID := strings.TrimSpace(workspaceID)
	if trimmedWorkspaceID == "" {
		return nil, errors.New("api: network workspace_id is required")
	}
	runtimePeers, err := service.ListPeers(ctx, trimmedWorkspaceID, "")
	if err != nil {
		return nil, fmt.Errorf("api: list network peers: %w", err)
	}
	sessions, err := sessionsManager.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("api: list sessions: %w", err)
	}
	channelMetadata, err := networkStore.ListNetworkChannels(
		ctx,
		store.NetworkChannelQuery{WorkspaceID: trimmedWorkspaceID},
	)
	if err != nil {
		return nil, fmt.Errorf("api: list network channels: %w", err)
	}
	projections, err := listNetworkChannelProjections(ctx, networkStore, trimmedWorkspaceID, "")
	if err != nil {
		return nil, fmt.Errorf("api: list network channel projections: %w", err)
	}

	aggregates := make(map[string]*networkChannelAggregate)
	applyNetworkChannelMetadata(aggregates, trimmedWorkspaceID, channelMetadata)
	applyNetworkChannelSessions(aggregates, trimmedWorkspaceID, sessions)
	applyNetworkChannelPeers(aggregates, runtimePeers)
	applyNetworkChannelProjections(aggregates, projections)
	return aggregates, nil
}

func applyNetworkChannelMetadata(
	aggregates map[string]*networkChannelAggregate,
	workspaceID string,
	metadataEntries []store.NetworkChannelEntry,
) {
	for _, metadata := range metadataEntries {
		metadataCopy := metadata
		aggregate := ensureNetworkChannelAggregate(aggregates, workspaceID, metadata.Channel)
		aggregate.metadata = &metadataCopy
	}
}

func applyNetworkChannelSessions(
	aggregates map[string]*networkChannelAggregate,
	workspaceID string,
	sessions []*session.Info,
) {
	for _, info := range sessions {
		if !networkChannelSessionVisible(info) || strings.TrimSpace(info.WorkspaceID) != workspaceID {
			continue
		}
		aggregate := ensureNetworkChannelAggregate(
			aggregates,
			workspaceID,
			info.NetworkParticipation.ChannelID,
		)
		aggregate.sessionCount++
	}
}

func applyNetworkChannelPeers(
	aggregates map[string]*networkChannelAggregate,
	peers []network.PeerInfo,
) {
	for _, peer := range peers {
		aggregate := ensureNetworkChannelAggregate(aggregates, peer.WorkspaceID, peer.Channel)
		aggregate.peerCount++
		if peer.Local {
			aggregate.localPeerCount++
			continue
		}
		aggregate.remotePeerCount++
	}
}

func applyNetworkChannelMessages(
	aggregates map[string]*networkChannelAggregate,
	messages []store.NetworkMessageEntry,
) {
	for _, message := range messages {
		aggregate := ensureNetworkChannelAggregate(aggregates, message.WorkspaceID, message.Channel)
		recordNetworkMessageParticipants(aggregate.recordHistoricalParticipant, message)
		if !isPublicChannelTimelineMessage(message) {
			continue
		}
		if isPresenceMessage(message) {
			aggregate.presenceCount++
			aggregate.lastPresenceAt = laterTimePtr(aggregate.lastPresenceAt, message.Timestamp)
			continue
		}
		aggregate.messageCount++
		aggregate.lastActivityAt = laterTimePtr(aggregate.lastActivityAt, message.Timestamp)
		aggregate.lastMessageAt = laterTimePtr(aggregate.lastMessageAt, message.Timestamp)
		if preview := networkMessagePreview(message); preview != "" && aggregateMessageIsLatest(aggregate, message) {
			aggregate.lastMessagePreview = preview
		}
	}
}

func recordNetworkMessageParticipants(record func(string), message store.NetworkMessageEntry) {
	record(message.PeerFrom)
	record(message.PeerTo)
	for _, peerID := range message.Mentions {
		record(peerID)
	}
}

func (a *networkChannelAggregate) recordHistoricalParticipant(peerID string) {
	if a == nil {
		return
	}
	trimmed := strings.TrimSpace(peerID)
	if trimmed == "" {
		return
	}
	if a.historicalParticipants == nil {
		a.historicalParticipants = make(map[string]struct{})
	}
	if _, exists := a.historicalParticipants[trimmed]; exists {
		return
	}
	a.historicalParticipants[trimmed] = struct{}{}
	a.historicalParticipantCount = len(a.historicalParticipants)
}

func aggregateMessageIsLatest(
	aggregate *networkChannelAggregate,
	message store.NetworkMessageEntry,
) bool {
	return aggregate != nil &&
		aggregate.lastMessageAt != nil &&
		message.Timestamp.Equal(aggregate.lastMessageAt.UTC())
}

func statusForCreateNetworkChannelError(err error) int {
	switch {
	case errors.Is(err, workspacepkg.ErrWorkspaceNotFound),
		errors.Is(err, workspacepkg.ErrWorkspaceRootMissing):
		return StatusForWorkspaceError(err)
	case errors.Is(err, workspacepkg.ErrAgentNotAvailable):
		return StatusForSessionError(err)
	case errors.Is(err, store.ErrNetworkChannelExists):
		return http.StatusConflict
	case errors.Is(err, network.ErrInvalidField):
		return StatusForNetworkError(err)
	default:
		return http.StatusBadRequest
	}
}

func (h *BaseHandlers) createNetworkChannelSessions(
	ctx context.Context,
	channel string,
	workspaceID string,
	agentNames []string,
) ([]string, error) {
	createdIDs := make([]string, 0, len(agentNames))
	for _, agentName := range agentNames {
		sess, err := h.Sessions.Create(ctx, session.CreateOpts{
			AgentName:            agentName,
			Provider:             "",
			Workspace:            workspaceID,
			NetworkParticipation: namedParticipationRequest(channel),
			Type:                 session.SessionTypeUser,
		})
		if err != nil {
			if rollbackErr := rollbackCreatedNetworkSessions(ctx, h.Sessions, createdIDs); rollbackErr != nil {
				err = errors.Join(err, rollbackErr)
			}
			return nil, err
		}
		if sess != nil && sess.Info() != nil {
			createdIDs = append(createdIDs, sess.Info().ID)
		}
	}
	return createdIDs, nil
}

func rollbackCreatedNetworkChannel(
	ctx context.Context,
	sessions SessionManager,
	networkStore NetworkStore,
	ref store.NetworkChannelRef,
	createdIDs []string,
	baseErr error,
	deleteChannel bool,
) error {
	if rollbackErr := rollbackCreatedNetworkSessions(ctx, sessions, createdIDs); rollbackErr != nil {
		baseErr = errors.Join(baseErr, rollbackErr)
	}
	if deleteChannel {
		if rollbackErr := networkStore.DeleteNetworkChannel(ctx, ref); rollbackErr != nil {
			baseErr = errors.Join(baseErr, rollbackErr)
		}
	}
	return baseErr
}

func (h *BaseHandlers) networkChannelDetailPayload(
	ctx context.Context,
	service NetworkService,
	workspaceID string,
	channel string,
) (contract.NetworkChannelDetailPayload, error) {
	networkStore, err := h.networkStoreRequired()
	if err != nil {
		return contract.NetworkChannelDetailPayload{}, err
	}
	trimmedWorkspaceID := strings.TrimSpace(workspaceID)
	if trimmedWorkspaceID == "" {
		return contract.NetworkChannelDetailPayload{}, errors.New("api: network workspace_id is required")
	}
	peers, err := service.ListPeers(ctx, trimmedWorkspaceID, channel)
	if err != nil {
		return contract.NetworkChannelDetailPayload{}, err
	}
	sessions, err := h.Sessions.ListAll(ctx)
	if err != nil {
		return contract.NetworkChannelDetailPayload{}, err
	}

	filteredSessions := sessionsForChannel(sessions, trimmedWorkspaceID, channel)
	metadata, err := h.loadNetworkChannelMetadata(ctx, networkStore, store.NetworkChannelRef{
		WorkspaceID: trimmedWorkspaceID,
		Channel:     channel,
	})
	if err != nil {
		return contract.NetworkChannelDetailPayload{}, err
	}
	projections, err := listNetworkChannelProjections(ctx, networkStore, trimmedWorkspaceID, channel)
	if err != nil {
		return contract.NetworkChannelDetailPayload{}, err
	}
	if len(filteredSessions) == 0 && len(peers) == 0 && len(projections) == 0 && metadata == nil {
		return contract.NetworkChannelDetailPayload{}, fmt.Errorf("%w: %s", errNetworkChannelNotFound, channel)
	}

	payloadPeers, localPeerCount := networkChannelPeerPayloads(peers, filteredSessions)

	metadataFields := networkChannelMetadataPayloadFields(metadata)
	projection := firstNetworkChannelProjection(projections)
	kindCounts, err := listNetworkChannelKindCountPayloads(ctx, networkStore, store.NetworkChannelRef{
		WorkspaceID: trimmedWorkspaceID,
		Channel:     channel,
	})
	if err != nil {
		return contract.NetworkChannelDetailPayload{}, err
	}

	return contract.NetworkChannelDetailPayload{
		Channel:                    channel,
		WorkspaceID:                firstNonEmpty(metadataFields.workspaceID, trimmedWorkspaceID),
		Purpose:                    metadataFields.purpose,
		FanoutPolicy:               metadataFields.fanoutPolicy,
		CoordinatorPeerID:          metadataFields.coordinatorPeerID,
		CreatedBy:                  metadataFields.createdBy,
		CreatedAt:                  metadataFields.createdAt,
		PeerCount:                  len(peers),
		LocalPeerCount:             localPeerCount,
		RemotePeerCount:            len(peers) - localPeerCount,
		SessionCount:               len(filteredSessions),
		MessageCount:               projection.MessageCount,
		PresenceCount:              projection.PresenceCount,
		HistoricalParticipantCount: projection.HistoricalParticipantCount,
		LastActivityAt:             cloneTimePtr(projection.LastActivityAt),
		LastPresenceAt:             cloneTimePtr(projection.LastPresenceAt),
		LastMessagePreview:         strings.TrimSpace(projection.LastMessagePreview),
		KindCounts:                 kindCounts,
		Sessions:                   SessionPayloadsFromInfos(filteredSessions),
		Peers:                      payloadPeers,
	}, nil
}

func networkChannelPeerPayloads(
	peers []network.PeerInfo,
	sessions []*session.Info,
) ([]contract.NetworkPeerPayload, int) {
	sessionByID := sessionInfoMapByID(sessions)
	payloads := make([]contract.NetworkPeerPayload, 0, len(peers))
	localPeerCount := 0
	for _, peer := range peers {
		if peer.Local {
			localPeerCount++
		}
		payloads = append(payloads, networkPeerPayloadFromInfoWithSessions(peer, sessionByID))
	}
	sortNetworkPeerPayloads(payloads)
	return payloads, localPeerCount
}

func networkChannelMetadataPayloadFields(metadata *store.NetworkChannelEntry) networkChannelMetadataFields {
	if metadata == nil {
		return networkChannelMetadataFields{}
	}
	return networkChannelMetadataFields{
		createdAt:         cloneTimePtr(&metadata.CreatedAt),
		purpose:           strings.TrimSpace(metadata.Purpose),
		fanoutPolicy:      strings.TrimSpace(metadata.FanoutPolicy),
		coordinatorPeerID: strings.TrimSpace(metadata.CoordinatorPeerID),
		workspaceID:       strings.TrimSpace(metadata.WorkspaceID),
		createdBy:         strings.TrimSpace(metadata.CreatedBy),
	}
}

func ensureNetworkChannelAggregate(
	aggregates map[string]*networkChannelAggregate,
	workspaceID string,
	channel string,
) *networkChannelAggregate {
	trimmed := strings.TrimSpace(channel)
	aggregate, ok := aggregates[trimmed]
	if ok && aggregate != nil {
		return aggregate
	}
	aggregate = &networkChannelAggregate{
		workspaceID: strings.TrimSpace(workspaceID),
		channel:     trimmed,
	}
	aggregates[trimmed] = aggregate
	return aggregate
}

func sessionsForChannel(sessions []*session.Info, workspaceID string, channel string) []*session.Info {
	filtered := make([]*session.Info, 0, len(sessions))
	for _, info := range sessions {
		if !networkChannelSessionVisible(info) ||
			strings.TrimSpace(info.WorkspaceID) != strings.TrimSpace(workspaceID) ||
			strings.TrimSpace(info.NetworkParticipation.ChannelID) != channel {
			continue
		}
		filtered = append(filtered, info)
	}
	return filtered
}

func networkChannelExists(
	sessions []*session.Info,
	peers []network.PeerInfo,
	metadata *store.NetworkChannelEntry,
	workspaceID string,
	channel string,
) bool {
	if metadata != nil {
		return true
	}
	for _, info := range sessions {
		if networkChannelSessionVisible(info) &&
			strings.TrimSpace(info.WorkspaceID) == strings.TrimSpace(workspaceID) &&
			strings.TrimSpace(info.NetworkParticipation.ChannelID) == channel {
			return true
		}
	}
	for _, peer := range peers {
		if strings.TrimSpace(peer.WorkspaceID) == strings.TrimSpace(workspaceID) &&
			strings.TrimSpace(peer.Channel) == channel {
			return true
		}
	}
	return false
}

func networkChannelSessionVisible(info *session.Info) bool {
	if info == nil {
		return false
	}
	if info.State == session.StateStopped {
		return false
	}
	return info.NetworkParticipation.Mode == participation.ModeLive
}

func namedParticipationRequest(channelID string) *participation.Request {
	live := participation.ModeLive
	named := participation.StrategyNamed
	trimmed := strings.TrimSpace(channelID)
	return &participation.Request{
		Mode:            &live,
		ChannelStrategy: &named,
		ChannelID:       &trimmed,
	}
}

func isNetworkChannelNotFound(err error) bool {
	return errors.Is(err, errNetworkChannelNotFound)
}

func sessionInfoMapByID(sessions []*session.Info) map[string]*session.Info {
	index := make(map[string]*session.Info, len(sessions))
	for _, info := range sessions {
		if info == nil {
			continue
		}
		index[strings.TrimSpace(info.ID)] = info
	}
	return index
}

func peerInfoMapByID(peers []network.PeerInfo) map[string]network.PeerInfo {
	index := make(map[string]network.PeerInfo, len(peers))
	for _, peer := range peers {
		index[strings.TrimSpace(peer.PeerID)] = peer
	}
	return index
}

func networkChannelPayloadFromAggregate(
	aggregate *networkChannelAggregate,
) contract.NetworkChannelPayload {
	payload := contract.NetworkChannelPayload{
		Channel:                    aggregate.channel,
		WorkspaceID:                strings.TrimSpace(aggregate.workspaceID),
		PeerCount:                  aggregate.peerCount,
		LocalPeerCount:             aggregate.localPeerCount,
		RemotePeerCount:            aggregate.remotePeerCount,
		SessionCount:               aggregate.sessionCount,
		MessageCount:               aggregate.messageCount,
		PresenceCount:              aggregate.presenceCount,
		HistoricalParticipantCount: aggregate.historicalParticipantCount,
		LastActivityAt:             cloneTimePtr(aggregate.lastActivityAt),
		LastPresenceAt:             cloneTimePtr(aggregate.lastPresenceAt),
		LastMessagePreview:         strings.TrimSpace(aggregate.lastMessagePreview),
	}
	if aggregate.metadata == nil {
		return payload
	}
	payload.WorkspaceID = firstNonEmpty(
		strings.TrimSpace(aggregate.metadata.WorkspaceID),
		strings.TrimSpace(aggregate.workspaceID),
	)
	payload.Purpose = strings.TrimSpace(aggregate.metadata.Purpose)
	payload.FanoutPolicy = strings.TrimSpace(aggregate.metadata.FanoutPolicy)
	payload.CoordinatorPeerID = strings.TrimSpace(aggregate.metadata.CoordinatorPeerID)
	payload.CreatedBy = strings.TrimSpace(aggregate.metadata.CreatedBy)
	payload.CreatedAt = cloneTimePtr(&aggregate.metadata.CreatedAt)
	return payload
}

func networkKindSortRank(kind string) int {
	switch strings.TrimSpace(kind) {
	case string(network.KindSay):
		return 0
	case string(network.KindReceipt):
		return 1
	case string(network.KindCapability):
		return 2
	case string(network.KindGreet):
		return 3
	case string(network.KindWhois):
		return 4
	case string(network.KindTrace):
		return 5
	default:
		return 100
	}
}

func networkMessagePreview(entry store.NetworkMessageEntry) string {
	if preview := strings.TrimSpace(entry.PreviewText); preview != "" {
		return preview
	}
	if text := strings.TrimSpace(entry.Text); text != "" {
		return text
	}
	return network.PreviewTextForRawBody(network.Kind(strings.TrimSpace(entry.Kind)), entry.Body)
}

func (h *BaseHandlers) networkPresenceWindow() time.Duration {
	if h == nil {
		return 0
	}
	window := 2 * h.Config.Network.GreetIntervalDuration()
	if window <= 0 {
		return 0
	}
	return window
}

func (h *BaseHandlers) loadNetworkChannelMetadata(
	ctx context.Context,
	networkStore NetworkStore,
	ref store.NetworkChannelRef,
) (*store.NetworkChannelEntry, error) {
	entry, err := networkStore.GetNetworkChannel(ctx, ref)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &entry, nil
}

func findPeerInfo(peers []network.PeerInfo, peerID string) (network.PeerInfo, bool) {
	target := strings.TrimSpace(peerID)
	for _, peer := range peers {
		if strings.TrimSpace(peer.PeerID) == target {
			return peer, true
		}
	}
	return network.PeerInfo{}, false
}

func laterTimePtr(current *time.Time, candidate time.Time) *time.Time {
	if candidate.IsZero() {
		return cloneTimePtr(current)
	}
	if current == nil || candidate.After(current.UTC()) {
		value := candidate.UTC()
		return &value
	}
	return cloneTimePtr(current)
}

func networkPeerPayloadFromInfoWithSessions(
	peer network.PeerInfo,
	sessionsByID map[string]*session.Info,
) contract.NetworkPeerPayload {
	payload := NetworkPeerPayloadFromInfo(peer)
	payload.DisplayName = networkPeerDisplayName(peer, sessionsByID)
	return payload
}

func networkPeerDisplayName(peer network.PeerInfo, sessionsByID map[string]*session.Info) string {
	if peer.PeerCard.DisplayName != nil {
		if value := strings.TrimSpace(*peer.PeerCard.DisplayName); value != "" {
			return value
		}
	}
	if peer.SessionID != nil && sessionsByID != nil {
		if info, ok := sessionsByID[strings.TrimSpace(*peer.SessionID)]; ok && info != nil {
			if value := strings.TrimSpace(info.Name); value != "" {
				return value
			}
			if value := strings.TrimSpace(info.AgentName); value != "" {
				return value
			}
		}
	}
	return strings.TrimSpace(peer.PeerID)
}

// NetworkConversationMessagePayloadFromEntry converts one persisted timeline row into the shared payload.
func NetworkConversationMessagePayloadFromEntry(
	entry store.NetworkMessageEntry,
	sessionsByID map[string]*session.Info,
	peersByID map[string]network.PeerInfo,
) contract.NetworkConversationMessagePayload {
	return NetworkConversationMessagePayloadFromView(networkTimelineMessageView{entry: entry}, sessionsByID, peersByID)
}

func NetworkConversationMessagePayloadFromView(
	view networkTimelineMessageView,
	sessionsByID map[string]*session.Info,
	peersByID map[string]network.PeerInfo,
) contract.NetworkConversationMessagePayload {
	entry := view.entry
	storedSessionID := strings.TrimSpace(entry.SessionID)
	displayName := strings.TrimSpace(entry.PeerFrom)
	local := strings.TrimSpace(entry.Direction) == network.AuditDirectionSent
	payloadSessionID := ""

	if peer, ok := peersByID[strings.TrimSpace(entry.PeerFrom)]; ok {
		displayName = networkPeerDisplayName(peer, sessionsByID)
	}

	if local {
		payloadSessionID = storedSessionID
	}

	if local && payloadSessionID != "" {
		if info, ok := sessionsByID[payloadSessionID]; ok && info != nil {
			if value := strings.TrimSpace(info.Name); value != "" {
				displayName = value
			} else if value := strings.TrimSpace(info.AgentName); value != "" {
				displayName = value
			}
		}
	}

	return contract.NetworkConversationMessagePayload{
		MessageID:          strings.TrimSpace(entry.MessageID),
		WorkspaceID:        strings.TrimSpace(entry.WorkspaceID),
		Channel:            strings.TrimSpace(entry.Channel),
		Surface:            strings.TrimSpace(entry.Surface),
		ThreadID:           strings.TrimSpace(entry.ThreadID),
		DirectID:           strings.TrimSpace(entry.DirectID),
		Kind:               strings.TrimSpace(entry.Kind),
		Direction:          strings.TrimSpace(entry.Direction),
		PeerFrom:           strings.TrimSpace(entry.PeerFrom),
		PeerTo:             strings.TrimSpace(entry.PeerTo),
		Mentions:           cloneTrimmedStrings(entry.Mentions),
		DisplayName:        displayName,
		SessionID:          payloadSessionID,
		Local:              local,
		WorkID:             strings.TrimSpace(entry.WorkID),
		ReplyTo:            strings.TrimSpace(entry.ReplyTo),
		TraceID:            strings.TrimSpace(entry.TraceID),
		CausationID:        strings.TrimSpace(entry.CausationID),
		Intent:             strings.TrimSpace(entry.Intent),
		Text:               strings.TrimSpace(entry.Text),
		PreviewText:        networkMessagePreview(entry),
		SizeBytes:          entry.SizeBytes,
		PresenceCount:      view.presenceCount,
		PresenceStartedAt:  cloneTimePtr(view.presenceStartedAt),
		PresenceLastSeenAt: cloneTimePtr(view.presenceLastSeenAt),
		Body:               cloneRawMessage(entry.Body),
		Timestamp:          entry.Timestamp.UTC(),
	}
}

func summarizeNetworkMessageHistory(
	messages []store.NetworkMessageEntry,
	presenceWindow time.Duration,
) networkMessageHistorySummary {
	summary := networkMessageHistorySummary{
		conversation:     make([]store.NetworkMessageEntry, 0, len(messages)),
		presenceEpisodes: make([]networkTimelineMessageView, 0),
	}
	if len(messages) == 0 {
		return summary
	}

	participants := make(map[string]struct{})
	openEpisodes := make(map[networkPresenceEpisodeKey]int)

	for _, message := range messages {
		recordNetworkMessageParticipants(func(peerID string) {
			recordHistoricalParticipant(participants, peerID)
		}, message)
		if isPresenceMessage(message) {
			summary.presenceCount++
			summary.lastPresenceAt = laterTimePtr(summary.lastPresenceAt, message.Timestamp)
			key := networkPresenceEpisodeKeyForMessage(message)
			if index, ok := openEpisodes[key]; ok &&
				canExtendPresenceEpisode(summary.presenceEpisodes[index], message, presenceWindow) {
				extendPresenceEpisode(&summary.presenceEpisodes[index], message)
				continue
			}
			currentEpisode := networkTimelineMessageView{
				entry: cloneNetworkMessageEntry(message),
			}
			startedAt := message.Timestamp.UTC()
			lastSeenAt := message.Timestamp.UTC()
			currentEpisode.presenceCount = 1
			currentEpisode.presenceStartedAt = &startedAt
			currentEpisode.presenceLastSeenAt = &lastSeenAt
			currentEpisode.entry.PreviewText = networkMessagePreview(currentEpisode.entry)
			summary.presenceEpisodes = append(summary.presenceEpisodes, currentEpisode)
			openEpisodes[key] = len(summary.presenceEpisodes) - 1
			continue
		}

		summary.conversation = append(summary.conversation, cloneNetworkMessageEntry(message))
	}
	summary.historicalParticipantCount = len(participants)
	return summary
}

func networkTimelinePayloads(
	messages []store.NetworkMessageEntry,
	sessionByID map[string]*session.Info,
	peerByID map[string]network.PeerInfo,
	query store.NetworkMessageQuery,
	presenceWindow time.Duration,
) ([]contract.NetworkConversationMessagePayload, error) {
	history := summarizeNetworkMessageHistory(messages, presenceWindow)
	views, err := paginateNetworkTimelineViews(history.timelineViews(query.IncludePresence), query)
	if err != nil {
		return nil, err
	}
	payload := make([]contract.NetworkConversationMessagePayload, 0, len(views))
	for _, view := range views {
		payload = append(payload, NetworkConversationMessagePayloadFromView(view, sessionByID, peerByID))
	}
	return payload, nil
}

func (s networkMessageHistorySummary) timelineViews(includePresence bool) []networkTimelineMessageView {
	if !includePresence {
		views := make([]networkTimelineMessageView, 0, len(s.conversation))
		for _, entry := range s.conversation {
			views = append(views, networkTimelineMessageView{entry: entry})
		}
		return views
	}

	views := make([]networkTimelineMessageView, 0, len(s.conversation)+len(s.presenceEpisodes))
	for _, entry := range s.conversation {
		views = append(views, networkTimelineMessageView{entry: entry})
	}
	views = append(views, s.presenceEpisodes...)
	sort.SliceStable(views, func(i int, j int) bool {
		left := views[i].entry.Timestamp.UTC()
		right := views[j].entry.Timestamp.UTC()
		if !left.Equal(right) {
			return left.Before(right)
		}
		return strings.TrimSpace(views[i].entry.MessageID) < strings.TrimSpace(views[j].entry.MessageID)
	})
	return views
}

func canExtendPresenceEpisode(
	current networkTimelineMessageView,
	next store.NetworkMessageEntry,
	window time.Duration,
) bool {
	if current.presenceCount <= 0 || window <= 0 {
		return false
	}
	if !isPresenceMessage(current.entry) || !isPresenceMessage(next) {
		return false
	}
	if strings.TrimSpace(current.entry.Direction) != strings.TrimSpace(next.Direction) {
		return false
	}
	if strings.TrimSpace(current.entry.Channel) != strings.TrimSpace(next.Channel) {
		return false
	}
	if strings.TrimSpace(current.entry.Surface) != strings.TrimSpace(next.Surface) {
		return false
	}
	if strings.TrimSpace(current.entry.ThreadID) != strings.TrimSpace(next.ThreadID) {
		return false
	}
	if strings.TrimSpace(current.entry.DirectID) != strings.TrimSpace(next.DirectID) {
		return false
	}
	if strings.TrimSpace(current.entry.WorkID) != strings.TrimSpace(next.WorkID) {
		return false
	}
	if strings.TrimSpace(current.entry.PeerFrom) != strings.TrimSpace(next.PeerFrom) {
		return false
	}
	if strings.TrimSpace(current.entry.PeerTo) != strings.TrimSpace(next.PeerTo) {
		return false
	}
	return next.Timestamp.UTC().Sub(current.entry.Timestamp.UTC()) <= window
}

func networkPresenceEpisodeKeyForMessage(message store.NetworkMessageEntry) networkPresenceEpisodeKey {
	return networkPresenceEpisodeKey{
		workspaceID: strings.TrimSpace(message.WorkspaceID),
		direction:   strings.TrimSpace(message.Direction),
		channel:     strings.TrimSpace(message.Channel),
		surface:     strings.TrimSpace(message.Surface),
		threadID:    strings.TrimSpace(message.ThreadID),
		directID:    strings.TrimSpace(message.DirectID),
		workID:      strings.TrimSpace(message.WorkID),
		peerFrom:    strings.TrimSpace(message.PeerFrom),
		peerTo:      strings.TrimSpace(message.PeerTo),
	}
}

func extendPresenceEpisode(current *networkTimelineMessageView, next store.NetworkMessageEntry) {
	if current == nil {
		return
	}
	nextCopy := cloneNetworkMessageEntry(next)
	nextCopy.PreviewText = networkMessagePreview(nextCopy)
	lastSeenAt := nextCopy.Timestamp.UTC()
	current.entry = nextCopy
	current.presenceCount++
	current.presenceLastSeenAt = &lastSeenAt
}

func cloneNetworkMessageEntry(entry store.NetworkMessageEntry) store.NetworkMessageEntry {
	return store.NetworkMessageEntry{
		MessageID:   strings.TrimSpace(entry.MessageID),
		WorkspaceID: strings.TrimSpace(entry.WorkspaceID),
		SessionID:   strings.TrimSpace(entry.SessionID),
		Channel:     strings.TrimSpace(entry.Channel),
		Surface:     strings.TrimSpace(entry.Surface),
		ThreadID:    strings.TrimSpace(entry.ThreadID),
		DirectID:    strings.TrimSpace(entry.DirectID),
		Direction:   strings.TrimSpace(entry.Direction),
		PeerFrom:    strings.TrimSpace(entry.PeerFrom),
		PeerTo:      strings.TrimSpace(entry.PeerTo),
		Kind:        strings.TrimSpace(entry.Kind),
		WorkID:      strings.TrimSpace(entry.WorkID),
		ReplyTo:     strings.TrimSpace(entry.ReplyTo),
		TraceID:     strings.TrimSpace(entry.TraceID),
		CausationID: strings.TrimSpace(entry.CausationID),
		Intent:      strings.TrimSpace(entry.Intent),
		Text:        entry.Text,
		PreviewText: strings.TrimSpace(entry.PreviewText),
		Mentions:    cloneTrimmedStrings(entry.Mentions),
		Body:        cloneRawMessage(entry.Body),
		SizeBytes:   entry.SizeBytes,
		Timestamp:   entry.Timestamp.UTC(),
	}
}

func isPresenceMessage(entry store.NetworkMessageEntry) bool {
	return strings.TrimSpace(entry.Kind) == string(network.KindGreet)
}

func recordHistoricalParticipant(target map[string]struct{}, peerID string) {
	trimmed := strings.TrimSpace(peerID)
	if trimmed == "" {
		return
	}
	target[trimmed] = struct{}{}
}

func filterPeerTimelineMessages(messages []store.NetworkMessageEntry) []store.NetworkMessageEntry {
	filtered := make([]store.NetworkMessageEntry, 0, len(messages))
	for _, message := range messages {
		if isPresenceMessage(message) || isDirectedChannelMessage(message) {
			filtered = append(filtered, message)
		}
	}
	return filtered
}

func filterVisiblePublicChannelMessages(
	messages []store.NetworkMessageEntry,
	includePresence bool,
) []store.NetworkMessageEntry {
	if includePresence {
		return filterPublicChannelTimelineMessages(messages)
	}

	filtered := make([]store.NetworkMessageEntry, 0, len(messages))
	for _, message := range messages {
		if isPublicConversationMessage(message) {
			filtered = append(filtered, message)
		}
	}
	return filtered
}

func filterPublicChannelTimelineMessages(messages []store.NetworkMessageEntry) []store.NetworkMessageEntry {
	filtered := make([]store.NetworkMessageEntry, 0, len(messages))
	for _, message := range messages {
		if isPublicChannelTimelineMessage(message) {
			filtered = append(filtered, message)
		}
	}
	return filtered
}

func isPublicChannelTimelineMessage(message store.NetworkMessageEntry) bool {
	return isPresenceMessage(message) || !isDirectedChannelMessage(message)
}

func isPublicConversationMessage(message store.NetworkMessageEntry) bool {
	return !isPresenceMessage(message) && !isDirectedChannelMessage(message)
}

func isDirectedChannelMessage(message store.NetworkMessageEntry) bool {
	if strings.TrimSpace(message.PeerTo) != "" {
		return true
	}
	if strings.TrimSpace(message.DirectID) != "" {
		return true
	}
	return strings.TrimSpace(message.Surface) == string(network.SurfaceDirect)
}

func filterVisiblePeerMessages(messages []store.NetworkMessageEntry, includePresence bool) []store.NetworkMessageEntry {
	if includePresence {
		return filterPeerTimelineMessages(messages)
	}

	filtered := make([]store.NetworkMessageEntry, 0, len(messages))
	for _, message := range messages {
		if isDirectedChannelMessage(message) {
			filtered = append(filtered, message)
		}
	}
	return filtered
}

func paginateNetworkTimelineViews(
	views []networkTimelineMessageView,
	query store.NetworkMessageQuery,
) ([]networkTimelineMessageView, error) {
	paginated := views
	if before := strings.TrimSpace(query.BeforeMessageID); before != "" {
		index := indexNetworkTimelineViewByMessageID(paginated, before)
		if index < 0 {
			return nil, fmt.Errorf("network timeline before cursor: %w", sql.ErrNoRows)
		}
		paginated = paginated[:index]
	}
	if after := strings.TrimSpace(query.AfterMessageID); after != "" {
		index := indexNetworkTimelineViewByMessageID(paginated, after)
		if index < 0 {
			return nil, fmt.Errorf("network timeline after cursor: %w", sql.ErrNoRows)
		}
		paginated = paginated[index+1:]
	}
	if query.Limit <= 0 || len(paginated) <= query.Limit {
		return paginated, nil
	}
	if strings.TrimSpace(query.BeforeMessageID) != "" {
		return paginated[len(paginated)-query.Limit:], nil
	}
	if strings.TrimSpace(query.AfterMessageID) != "" {
		return paginated[:query.Limit], nil
	}
	return paginated[len(paginated)-query.Limit:], nil
}

func indexNetworkTimelineViewByMessageID(views []networkTimelineMessageView, messageID string) int {
	target := strings.TrimSpace(messageID)
	for index, view := range views {
		if strings.TrimSpace(view.entry.MessageID) == target {
			return index
		}
	}
	return -1
}

func (h *BaseHandlers) loadPeerAuditEntries(
	ctx context.Context,
	networkStore NetworkStore,
	peer network.PeerInfo,
) ([]store.NetworkAuditEntry, error) {
	if peer.SessionID != nil {
		return networkStore.ListNetworkAudit(ctx, store.NetworkAuditQuery{
			WorkspaceID: strings.TrimSpace(peer.WorkspaceID),
			SessionID:   strings.TrimSpace(*peer.SessionID),
		})
	}

	entries, err := networkStore.ListNetworkAudit(ctx, store.NetworkAuditQuery{
		WorkspaceID: strings.TrimSpace(peer.WorkspaceID),
		Channel:     strings.TrimSpace(peer.Channel),
	})
	if err != nil {
		return nil, err
	}

	filtered := make([]store.NetworkAuditEntry, 0, len(entries))
	for _, entry := range entries {
		if networkAuditMatchesPeer(peer, entry) {
			filtered = append(filtered, entry)
		}
	}
	return filtered, nil
}

func networkAuditMatchesPeer(peer network.PeerInfo, entry store.NetworkAuditEntry) bool {
	targetPeerID := strings.TrimSpace(peer.PeerID)
	if targetPeerID == "" {
		return false
	}
	if peer.SessionID != nil && strings.TrimSpace(entry.SessionID) == strings.TrimSpace(*peer.SessionID) {
		return true
	}
	return strings.TrimSpace(entry.PeerFrom) == targetPeerID || strings.TrimSpace(entry.PeerTo) == targetPeerID
}

func summarizePeerMetrics(peer network.PeerInfo, entries []store.NetworkAuditEntry) contract.NetworkPeerMetricsPayload {
	metrics := contract.NetworkPeerMetricsPayload{}
	for _, entry := range entries {
		if !networkAuditMatchesPeer(peer, entry) {
			continue
		}
		entrySize := int64(entry.Size)
		metrics.TotalSizeBytes += entrySize
		switch strings.TrimSpace(entry.Direction) {
		case network.AuditDirectionSent:
			metrics.Sent++
			metrics.SentSizeBytes += entrySize
		case network.AuditDirectionReceived:
			metrics.Received++
			metrics.ReceivedSizeBytes += entrySize
		case network.AuditDirectionRejected:
			metrics.Rejected++
			metrics.RejectedSizeBytes += entrySize
		case network.AuditDirectionDelivered:
			metrics.Delivered++
			metrics.DeliveredSizeBytes += entrySize
		}
	}
	return metrics
}

// NetworkPeerDetailPayloadFromInfo converts one peer info plus metrics into the shared detail payload.
func NetworkPeerDetailPayloadFromInfo(
	peer network.PeerInfo,
	sessionsByID map[string]*session.Info,
	metrics contract.NetworkPeerMetricsPayload,
) contract.NetworkPeerDetailPayload {
	presenceState, lastSeenAgeSeconds := networkPresenceFields(peer)
	payload := contract.NetworkPeerDetailPayload{
		SessionID:          peer.SessionID,
		PeerID:             peer.PeerID,
		DisplayName:        networkPeerDisplayName(peer, sessionsByID),
		Channel:            peer.Channel,
		Local:              peer.Local,
		PeerCard:           NetworkPeerPayloadFromInfo(peer).PeerCard,
		CapabilityCatalog:  networkCapabilityCatalogPayload(peer),
		JoinedAt:           cloneTimePtr(peer.JoinedAt),
		LastSeen:           cloneTimePtr(peer.LastSeen),
		ExpiresAt:          cloneTimePtr(peer.ExpiresAt),
		PresenceState:      presenceState,
		LastSeenAgeSeconds: lastSeenAgeSeconds,
		Metrics:            metrics,
	}
	return payload
}
