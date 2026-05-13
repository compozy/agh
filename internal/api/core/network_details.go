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

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
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
	lastPresenceAt             *time.Time
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
	createdAt   *time.Time
	purpose     string
	workspaceID string
	createdBy   string
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
	if bodyWorkspaceID := strings.TrimSpace(req.WorkspaceID); bodyWorkspaceID != "" && bodyWorkspaceID != scope.ID {
		h.respondError(
			c,
			http.StatusBadRequest,
			NewNetworkValidationError(errors.New("workspace_id does not match path")),
		)
		return
	}

	channel, purpose, resolved, agentNames, err := h.resolveCreateNetworkChannelRequest(
		c.Request.Context(),
		req,
		&scope.Resolved,
	)
	if err != nil {
		h.respondError(c, statusForCreateNetworkChannelError(err), err)
		return
	}

	createdIDs, err := h.createNetworkChannelSessions(
		c.Request.Context(),
		channel,
		resolved.WorkspaceID,
		agentNames,
	)
	if err != nil {
		h.respondError(c, StatusForSessionError(err), err)
		return
	}

	detail, err := h.finalizeCreatedNetworkChannel(
		c.Request.Context(),
		service,
		networkStore,
		store.NetworkChannelEntry{
			Channel:     channel,
			WorkspaceID: resolved.WorkspaceID,
			Purpose:     purpose,
			CreatedBy:   agentNames[0],
		},
		createdIDs,
	)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusCreated, contract.CreateNetworkChannelResponse{Channel: detail})
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

	detail, err := h.networkChannelDetailPayload(c.Request.Context(), service, scope.ID, channel)
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
	peers, err := service.ListPeers(c.Request.Context(), scope.ID, channel)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}

	query.WorkspaceID = scope.ID
	query.Channel = channel
	if err := query.Validate(); err != nil {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(err))
		return
	}
	rawMessages, messages, err := h.loadPublicChannelTimeline(c.Request.Context(), networkStore, query)
	if err != nil {
		h.respondNetworkMessageError(c, err)
		return
	}

	metadata, err := h.loadNetworkChannelMetadata(c.Request.Context(), networkStore, scope.NetworkChannelRef(channel))
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	if len(rawMessages) == 0 && !networkChannelExists(sessions, peers, metadata, scope.ID, channel) {
		notFoundErr := fmt.Errorf("%w: %s", errNetworkChannelNotFound, channel)
		h.respondError(c, http.StatusNotFound, notFoundErr)
		return
	}

	sessionByID := sessionInfoMapByID(sessions)
	peerByID := peerInfoMapByID(peers)
	payload, err := networkTimelinePayloads(
		messages,
		sessionByID,
		peerByID,
		query,
		h.networkPresenceWindow(),
	)
	if err != nil {
		h.respondNetworkMessageError(c, err)
		return
	}

	c.JSON(http.StatusOK, contract.NetworkChannelMessagesResponse{Messages: payload})
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
	peers, err := service.ListPeers(c.Request.Context(), scope.ID, "")
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

	query.WorkspaceID = scope.ID
	query.PeerID = peerID
	query.DirectedOnly = !query.IncludePresence
	if err := query.Validate(); err != nil {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(err))
		return
	}
	messages, err := h.loadVisiblePeerMessages(c.Request.Context(), networkStore, query)
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
		query,
		h.networkPresenceWindow(),
	)
	if err != nil {
		h.respondNetworkMessageError(c, err)
		return
	}

	c.JSON(http.StatusOK, contract.NetworkPeerMessagesResponse{Messages: payload})
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
	peers, err := service.ListPeers(c.Request.Context(), scope.ID, "")
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
) (string, string, workspacepkg.ResolvedWorkspace, []string, error) {
	_ = ctx
	if resolved == nil {
		return "", "", workspacepkg.ResolvedWorkspace{}, nil, NewNetworkValidationError(
			errors.New("workspace is required"),
		)
	}
	channel, err := normalizeNetworkChannel(req.Channel)
	if err != nil {
		return "", "", workspacepkg.ResolvedWorkspace{}, nil, err
	}
	purpose, err := normalizeNetworkChannelPurpose(req.Purpose)
	if err != nil {
		return "", "", workspacepkg.ResolvedWorkspace{}, nil, err
	}

	workspaceID := strings.TrimSpace(resolved.WorkspaceID)
	if workspaceID == "" {
		return "", "", workspacepkg.ResolvedWorkspace{}, nil, NewNetworkValidationError(
			errors.New("workspace_id is required"),
		)
	}

	agentNames, err := normalizeNetworkAgentNames(req.AgentNames)
	if err != nil {
		return "", "", workspacepkg.ResolvedWorkspace{}, nil, err
	}
	available := make(map[string]struct{}, len(resolved.Agents))
	for _, agent := range resolved.Agents {
		available[strings.TrimSpace(agent.Name)] = struct{}{}
	}
	for _, agentName := range agentNames {
		if _, ok := available[agentName]; ok {
			continue
		}
		return "", "", workspacepkg.ResolvedWorkspace{}, nil, fmt.Errorf(
			"%w: %s",
			workspacepkg.ErrAgentNotAvailable,
			agentName,
		)
	}

	return channel, purpose, *resolved, agentNames, nil
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
	messages, err := networkStore.ListNetworkMessages(ctx, store.NetworkMessageQuery{WorkspaceID: trimmedWorkspaceID})
	if err != nil {
		return nil, fmt.Errorf("api: list network messages: %w", err)
	}

	aggregates := make(map[string]*networkChannelAggregate)
	applyNetworkChannelMetadata(aggregates, trimmedWorkspaceID, channelMetadata)
	applyNetworkChannelSessions(aggregates, trimmedWorkspaceID, sessions)
	applyNetworkChannelPeers(aggregates, runtimePeers)
	applyNetworkChannelMessages(aggregates, messages)
	return aggregates, nil
}

func sortedNetworkChannelPayloads(
	aggregates map[string]*networkChannelAggregate,
) []contract.NetworkChannelPayload {
	channels := make([]contract.NetworkChannelPayload, 0, len(aggregates))
	for _, aggregate := range aggregates {
		if aggregate == nil {
			continue
		}
		channels = append(channels, networkChannelPayloadFromAggregate(aggregate))
	}
	sort.Slice(channels, func(i int, j int) bool {
		left := networkChannelSortTimestamp(channels[i])
		right := networkChannelSortTimestamp(channels[j])
		switch {
		case left != nil && right != nil && !left.Equal(*right):
			return left.After(*right)
		case left != nil && right == nil:
			return true
		case left == nil && right != nil:
			return false
		case channels[i].MessageCount != channels[j].MessageCount:
			return channels[i].MessageCount > channels[j].MessageCount
		default:
			return channels[i].Channel < channels[j].Channel
		}
	})
	return channels
}

func networkChannelSortTimestamp(channel contract.NetworkChannelPayload) *time.Time {
	switch {
	case channel.LastActivityAt == nil:
		return channel.LastPresenceAt
	case channel.LastPresenceAt == nil:
		return channel.LastActivityAt
	case channel.LastPresenceAt.After(*channel.LastActivityAt):
		return channel.LastPresenceAt
	default:
		return channel.LastActivityAt
	}
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
		aggregate := ensureNetworkChannelAggregate(aggregates, workspaceID, info.Channel)
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
		aggregate.recordHistoricalParticipant(message.PeerFrom)
		aggregate.recordHistoricalParticipant(message.PeerTo)
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
			AgentName: agentName,
			Provider:  "",
			Workspace: workspaceID,
			Channel:   channel,
			Type:      session.SessionTypeUser,
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

func (h *BaseHandlers) finalizeCreatedNetworkChannel(
	ctx context.Context,
	service NetworkService,
	networkStore NetworkStore,
	entry store.NetworkChannelEntry,
	createdIDs []string,
) (contract.NetworkChannelDetailPayload, error) {
	if err := networkStore.WriteNetworkChannel(ctx, entry); err != nil {
		return contract.NetworkChannelDetailPayload{}, rollbackCreatedNetworkChannel(
			ctx,
			h.Sessions,
			networkStore,
			store.NetworkChannelRef{
				WorkspaceID: strings.TrimSpace(entry.WorkspaceID),
				Channel:     strings.TrimSpace(entry.Channel),
			},
			createdIDs,
			err,
			false,
		)
	}

	detail, err := h.networkChannelDetailPayload(ctx, service, entry.WorkspaceID, entry.Channel)
	if err != nil {
		return contract.NetworkChannelDetailPayload{}, rollbackCreatedNetworkChannel(
			ctx,
			h.Sessions,
			networkStore,
			store.NetworkChannelRef{
				WorkspaceID: strings.TrimSpace(entry.WorkspaceID),
				Channel:     strings.TrimSpace(entry.Channel),
			},
			createdIDs,
			err,
			true,
		)
	}
	return detail, nil
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
	messages, err := networkStore.ListNetworkMessages(ctx, store.NetworkMessageQuery{
		WorkspaceID: trimmedWorkspaceID,
		Channel:     channel,
	})
	if err != nil {
		return contract.NetworkChannelDetailPayload{}, err
	}
	history := summarizeNetworkMessageHistory(filterPublicChannelTimelineMessages(messages), h.networkPresenceWindow())
	messageCount := len(history.conversation)
	if len(filteredSessions) == 0 && len(peers) == 0 && len(messages) == 0 && metadata == nil {
		return contract.NetworkChannelDetailPayload{}, fmt.Errorf("%w: %s", errNetworkChannelNotFound, channel)
	}

	payloadPeers, localPeerCount := networkChannelPeerPayloads(peers, filteredSessions)

	metadataFields := networkChannelMetadataPayloadFields(metadata)
	var (
		lastActivityAt     *time.Time
		lastPresenceAt     *time.Time
		lastMessagePreview string
	)
	kindCounts := summarizeNetworkChannelKindCounts(history.conversation)
	if messageCount > 0 {
		lastActivityAt = laterTimePtr(lastActivityAt, history.conversation[messageCount-1].Timestamp)
		lastMessagePreview = networkMessagePreview(history.conversation[messageCount-1])
	}
	lastPresenceAt = cloneTimePtr(history.lastPresenceAt)

	return contract.NetworkChannelDetailPayload{
		Channel:                    channel,
		WorkspaceID:                firstNonEmpty(metadataFields.workspaceID, trimmedWorkspaceID),
		Purpose:                    metadataFields.purpose,
		CreatedBy:                  metadataFields.createdBy,
		CreatedAt:                  metadataFields.createdAt,
		PeerCount:                  len(peers),
		LocalPeerCount:             localPeerCount,
		RemotePeerCount:            len(peers) - localPeerCount,
		SessionCount:               len(filteredSessions),
		MessageCount:               messageCount,
		PresenceCount:              history.presenceCount,
		HistoricalParticipantCount: summarizeHistoricalParticipantCount(messages),
		LastActivityAt:             cloneTimePtr(lastActivityAt),
		LastPresenceAt:             lastPresenceAt,
		LastMessagePreview:         lastMessagePreview,
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
		createdAt:   cloneTimePtr(&metadata.CreatedAt),
		purpose:     strings.TrimSpace(metadata.Purpose),
		workspaceID: strings.TrimSpace(metadata.WorkspaceID),
		createdBy:   strings.TrimSpace(metadata.CreatedBy),
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
			strings.TrimSpace(info.Channel) != channel {
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
			strings.TrimSpace(info.Channel) == channel {
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
	return strings.TrimSpace(info.Channel) != ""
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
	payload.CreatedBy = strings.TrimSpace(aggregate.metadata.CreatedBy)
	payload.CreatedAt = cloneTimePtr(&aggregate.metadata.CreatedAt)
	return payload
}

func summarizeNetworkChannelKindCounts(
	messages []store.NetworkMessageEntry,
) []contract.NetworkChannelKindCountPayload {
	if len(messages) == 0 {
		return nil
	}
	counts := make(map[string]int, len(messages))
	for _, item := range messages {
		kind := strings.TrimSpace(item.Kind)
		if kind == "" {
			continue
		}
		counts[kind]++
	}
	if len(counts) == 0 {
		return nil
	}
	payload := make([]contract.NetworkChannelKindCountPayload, 0, len(counts))
	for kind, count := range counts {
		payload = append(payload, contract.NetworkChannelKindCountPayload{
			Kind:  kind,
			Count: count,
		})
	}
	sort.Slice(payload, func(i int, j int) bool {
		leftRank := networkKindSortRank(payload[i].Kind)
		rightRank := networkKindSortRank(payload[j].Kind)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		return payload[i].Kind < payload[j].Kind
	})
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

func parseNetworkMessageQuery(c *gin.Context) (store.NetworkMessageQuery, error) {
	limit, err := ParseOptionalInt(c.Query("limit"))
	if err != nil {
		return store.NetworkMessageQuery{}, err
	}
	includePresence, err := ParseOptionalBool(c.Query("include_presence"))
	if err != nil {
		return store.NetworkMessageQuery{}, err
	}
	query := store.NetworkMessageQuery{
		BeforeMessageID: strings.TrimSpace(c.Query("before")),
		AfterMessageID:  strings.TrimSpace(c.Query("after")),
		IncludePresence: includePresence,
		Limit:           limit,
	}
	if strings.TrimSpace(query.BeforeMessageID) != "" && strings.TrimSpace(query.AfterMessageID) != "" {
		return store.NetworkMessageQuery{}, NewNetworkValidationError(
			fmt.Errorf(
				"%w: network message query cannot specify both before and after cursors",
				network.ErrInvalidField,
			),
		)
	}
	return query, nil
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
		recordHistoricalParticipant(participants, message.PeerFrom)
		recordHistoricalParticipant(participants, message.PeerTo)
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

func summarizeHistoricalParticipantCount(messages []store.NetworkMessageEntry) int {
	participants := make(map[string]struct{})
	for _, message := range messages {
		recordHistoricalParticipant(participants, message.PeerFrom)
		recordHistoricalParticipant(participants, message.PeerTo)
	}
	return len(participants)
}

func (h *BaseHandlers) loadPublicChannelTimeline(
	ctx context.Context,
	networkStore NetworkStore,
	query store.NetworkMessageQuery,
) ([]store.NetworkMessageEntry, []store.NetworkMessageEntry, error) {
	rawMessages, err := listTimelineRawMessages(ctx, networkStore, query)
	if err != nil {
		return nil, nil, err
	}
	return rawMessages, filterVisiblePublicChannelMessages(rawMessages, query.IncludePresence), nil
}

func (h *BaseHandlers) loadVisiblePeerMessages(
	ctx context.Context,
	networkStore NetworkStore,
	query store.NetworkMessageQuery,
) ([]store.NetworkMessageEntry, error) {
	rawMessages, err := listTimelineRawMessages(ctx, networkStore, query)
	if err != nil {
		return nil, err
	}
	return filterVisiblePeerMessages(rawMessages, query.IncludePresence), nil
}

func listTimelineRawMessages(
	ctx context.Context,
	networkStore NetworkStore,
	query store.NetworkMessageQuery,
) ([]store.NetworkMessageEntry, error) {
	rawQuery := query
	rawQuery.Limit = 0
	rawQuery.BeforeMessageID = ""
	rawQuery.AfterMessageID = ""
	return networkStore.ListNetworkMessages(ctx, rawQuery)
}

func (h *BaseHandlers) respondNetworkMessageError(c *gin.Context, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(errors.New("message cursor not found")))
		return
	}
	h.respondError(c, http.StatusInternalServerError, err)
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
		Body:        cloneRawMessage(entry.Body),
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
	return paginated[:query.Limit], nil
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
		switch strings.TrimSpace(entry.Direction) {
		case network.AuditDirectionSent:
			metrics.Sent++
		case network.AuditDirectionReceived:
			metrics.Received++
		case network.AuditDirectionRejected:
			metrics.Rejected++
		case network.AuditDirectionDelivered:
			metrics.Delivered++
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
	payload := contract.NetworkPeerDetailPayload{
		SessionID:         peer.SessionID,
		PeerID:            peer.PeerID,
		DisplayName:       networkPeerDisplayName(peer, sessionsByID),
		Channel:           peer.Channel,
		Local:             peer.Local,
		PeerCard:          NetworkPeerPayloadFromInfo(peer).PeerCard,
		CapabilityCatalog: networkCapabilityCatalogPayload(peer),
		JoinedAt:          cloneTimePtr(peer.JoinedAt),
		LastSeen:          cloneTimePtr(peer.LastSeen),
		ExpiresAt:         cloneTimePtr(peer.ExpiresAt),
		Metrics:           metrics,
	}
	return payload
}
