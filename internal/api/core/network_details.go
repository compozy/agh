package core

import (
	"context"
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
	channel         string
	peerCount       int
	localPeerCount  int
	remotePeerCount int
	sessionCount    int
	messageCount    int
	lastMessageAt   *time.Time
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

	var req contract.CreateNetworkChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode create network channel request: %w", h.transportName(), err),
		)
		return
	}

	channel, resolved, agentNames, err := h.resolveCreateNetworkChannelRequest(c.Request.Context(), req)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, workspacepkg.ErrWorkspaceNotFound),
			errors.Is(err, workspacepkg.ErrWorkspaceRootMissing):
			status = StatusForWorkspaceError(err)
		case errors.Is(err, workspacepkg.ErrAgentNotAvailable):
			status = StatusForSessionError(err)
		case errors.Is(err, network.ErrInvalidField):
			status = StatusForNetworkError(err)
		}
		h.respondError(c, status, err)
		return
	}

	createdIDs := make([]string, 0, len(agentNames))
	for _, agentName := range agentNames {
		sess, createErr := h.Sessions.Create(c.Request.Context(), session.CreateOpts{
			AgentName: agentName,
			Workspace: resolved.ID,
			Channel:   channel,
			Type:      session.SessionTypeUser,
		})
		if createErr != nil {
			if rollbackErr := rollbackCreatedNetworkSessions(
				c.Request.Context(),
				h.Sessions,
				createdIDs,
			); rollbackErr != nil {
				createErr = errors.Join(createErr, rollbackErr)
			}
			h.respondError(c, StatusForSessionError(createErr), createErr)
			return
		}
		if sess != nil && sess.Info() != nil {
			createdIDs = append(createdIDs, sess.Info().ID)
		}
	}

	detail, detailErr := h.networkChannelDetailPayload(c.Request.Context(), service, channel)
	if detailErr != nil {
		if rollbackErr := rollbackCreatedNetworkSessions(
			c.Request.Context(),
			h.Sessions,
			createdIDs,
		); rollbackErr != nil {
			detailErr = errors.Join(detailErr, rollbackErr)
		}
		h.respondError(c, http.StatusInternalServerError, detailErr)
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

	detail, err := h.networkChannelDetailPayload(c.Request.Context(), service, channel)
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
	limit, err := ParseOptionalInt(c.Query("limit"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	sessions, err := h.Sessions.ListAll(c.Request.Context())
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	peers, err := service.ListPeers(c.Request.Context(), channel)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}

	messages, err := networkStore.ListNetworkMessages(c.Request.Context(), store.NetworkMessageQuery{
		Channel: channel,
		Limit:   limit,
	})
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	if len(messages) == 0 && !networkChannelExists(sessions, peers, channel) {
		notFoundErr := fmt.Errorf("%w: %s", errNetworkChannelNotFound, channel)
		h.respondError(c, http.StatusNotFound, notFoundErr)
		return
	}

	directionByMessageID := map[string]string{}
	if len(messages) > 0 {
		auditEntries, auditErr := networkStore.ListNetworkAudit(c.Request.Context(), store.NetworkAuditQuery{
			Channel: channel,
		})
		if auditErr != nil {
			h.respondError(c, http.StatusInternalServerError, auditErr)
			return
		}
		directionByMessageID = networkMessageDirectionMap(auditEntries, networkMessageIDSet(messages))
	}

	sessionByID := sessionInfoMapByID(sessions)
	peerByID := peerInfoMapByID(peers)
	payload := make([]contract.NetworkChannelMessagePayload, 0, len(messages))
	for _, entry := range messages {
		payload = append(payload, NetworkChannelMessagePayloadFromEntry(
			entry,
			directionByMessageID[strings.TrimSpace(entry.MessageID)],
			sessionByID,
			peerByID,
		))
	}

	c.JSON(http.StatusOK, contract.NetworkChannelMessagesResponse{Messages: payload})
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

	peers, err := service.ListPeers(c.Request.Context(), "")
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
) (string, workspacepkg.ResolvedWorkspace, []string, error) {
	channel, err := normalizeNetworkChannel(req.Channel)
	if err != nil {
		return "", workspacepkg.ResolvedWorkspace{}, nil, err
	}

	workspaceID := strings.TrimSpace(req.WorkspaceID)
	if workspaceID == "" {
		return "", workspacepkg.ResolvedWorkspace{}, nil, NewNetworkValidationError(
			errors.New("workspace_id is required"),
		)
	}

	resolved, err := h.Workspaces.Resolve(ctx, workspaceID)
	if err != nil {
		return "", workspacepkg.ResolvedWorkspace{}, nil, err
	}

	agentNames, err := normalizeNetworkAgentNames(req.AgentNames)
	if err != nil {
		return "", workspacepkg.ResolvedWorkspace{}, nil, err
	}
	available := make(map[string]struct{}, len(resolved.Agents))
	for _, agent := range resolved.Agents {
		available[strings.TrimSpace(agent.Name)] = struct{}{}
	}
	for _, agentName := range agentNames {
		if _, ok := available[agentName]; ok {
			continue
		}
		return "", workspacepkg.ResolvedWorkspace{}, nil, fmt.Errorf(
			"%w: %s",
			workspacepkg.ErrAgentNotAvailable,
			agentName,
		)
	}

	return channel, resolved, agentNames, nil
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
) ([]contract.NetworkChannelPayload, error) {
	runtimePeers, err := service.ListPeers(ctx, "")
	if err != nil {
		return nil, err
	}
	sessions, err := h.Sessions.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	aggregates := make(map[string]*networkChannelAggregate)
	for _, info := range sessions {
		if !networkChannelSessionVisible(info) {
			continue
		}
		channel := strings.TrimSpace(info.Channel)
		aggregate := ensureNetworkChannelAggregate(aggregates, channel)
		aggregate.sessionCount++
	}
	for _, peer := range runtimePeers {
		aggregate := ensureNetworkChannelAggregate(aggregates, peer.Channel)
		aggregate.peerCount++
		if peer.Local {
			aggregate.localPeerCount++
		} else {
			aggregate.remotePeerCount++
		}
	}

	if h != nil && h.NetworkStore != nil {
		messages, msgErr := h.NetworkStore.ListNetworkMessages(ctx, store.NetworkMessageQuery{})
		if msgErr != nil {
			return nil, msgErr
		}
		for _, message := range messages {
			aggregate := ensureNetworkChannelAggregate(aggregates, message.Channel)
			aggregate.messageCount++
			aggregate.lastMessageAt = laterTimePtr(aggregate.lastMessageAt, message.Timestamp)
		}
	}

	channels := make([]contract.NetworkChannelPayload, 0, len(aggregates))
	for _, aggregate := range aggregates {
		if aggregate == nil {
			continue
		}
		channels = append(channels, contract.NetworkChannelPayload{
			Channel:         aggregate.channel,
			PeerCount:       aggregate.peerCount,
			LocalPeerCount:  aggregate.localPeerCount,
			RemotePeerCount: aggregate.remotePeerCount,
			SessionCount:    aggregate.sessionCount,
			MessageCount:    aggregate.messageCount,
			LastMessageAt:   cloneTimePtr(aggregate.lastMessageAt),
		})
	}
	sort.Slice(channels, func(i int, j int) bool {
		return channels[i].Channel < channels[j].Channel
	})
	return channels, nil
}

func (h *BaseHandlers) networkChannelDetailPayload(
	ctx context.Context,
	service NetworkService,
	channel string,
) (contract.NetworkChannelDetailPayload, error) {
	peers, err := service.ListPeers(ctx, channel)
	if err != nil {
		return contract.NetworkChannelDetailPayload{}, err
	}
	sessions, err := h.Sessions.ListAll(ctx)
	if err != nil {
		return contract.NetworkChannelDetailPayload{}, err
	}

	filteredSessions := sessionsForChannel(sessions, channel)
	messageCount := 0
	var lastMessageAt *time.Time
	if h != nil && h.NetworkStore != nil {
		messages, msgErr := h.NetworkStore.ListNetworkMessages(ctx, store.NetworkMessageQuery{Channel: channel})
		if msgErr != nil {
			return contract.NetworkChannelDetailPayload{}, msgErr
		}
		messageCount = len(messages)
		if messageCount > 0 {
			lastMessageAt = laterTimePtr(nil, messages[len(messages)-1].Timestamp)
		}
	}
	if len(filteredSessions) == 0 && len(peers) == 0 && messageCount == 0 {
		return contract.NetworkChannelDetailPayload{}, fmt.Errorf("%w: %s", errNetworkChannelNotFound, channel)
	}

	sessionByID := sessionInfoMapByID(filteredSessions)
	payloadPeers := make([]contract.NetworkPeerPayload, 0, len(peers))
	localPeerCount := 0
	for _, peer := range peers {
		if peer.Local {
			localPeerCount++
		}
		payloadPeers = append(payloadPeers, networkPeerPayloadFromInfoWithSessions(peer, sessionByID))
	}

	return contract.NetworkChannelDetailPayload{
		Channel:         channel,
		PeerCount:       len(peers),
		LocalPeerCount:  localPeerCount,
		RemotePeerCount: len(peers) - localPeerCount,
		SessionCount:    len(filteredSessions),
		MessageCount:    messageCount,
		LastMessageAt:   cloneTimePtr(lastMessageAt),
		Sessions:        SessionPayloadsFromInfos(filteredSessions),
		Peers:           payloadPeers,
	}, nil
}

func ensureNetworkChannelAggregate(
	aggregates map[string]*networkChannelAggregate,
	channel string,
) *networkChannelAggregate {
	trimmed := strings.TrimSpace(channel)
	aggregate, ok := aggregates[trimmed]
	if ok && aggregate != nil {
		return aggregate
	}
	aggregate = &networkChannelAggregate{channel: trimmed}
	aggregates[trimmed] = aggregate
	return aggregate
}

func sessionsForChannel(sessions []*session.Info, channel string) []*session.Info {
	filtered := make([]*session.Info, 0, len(sessions))
	for _, info := range sessions {
		if !networkChannelSessionVisible(info) || strings.TrimSpace(info.Channel) != channel {
			continue
		}
		filtered = append(filtered, info)
	}
	return filtered
}

func networkChannelExists(sessions []*session.Info, peers []network.PeerInfo, channel string) bool {
	for _, info := range sessions {
		if networkChannelSessionVisible(info) && strings.TrimSpace(info.Channel) == channel {
			return true
		}
	}
	for _, peer := range peers {
		if strings.TrimSpace(peer.Channel) == channel {
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

func networkMessageIDSet(messages []store.NetworkMessageEntry) map[string]struct{} {
	ids := make(map[string]struct{}, len(messages))
	for _, message := range messages {
		messageID := strings.TrimSpace(message.MessageID)
		if messageID == "" {
			continue
		}
		ids[messageID] = struct{}{}
	}
	return ids
}

func networkMessageDirectionMap(
	entries []store.NetworkAuditEntry,
	messageIDs map[string]struct{},
) map[string]string {
	directions := make(map[string]string, len(messageIDs))
	for _, entry := range entries {
		messageID := strings.TrimSpace(entry.MessageID)
		if messageID == "" {
			continue
		}
		if _, ok := messageIDs[messageID]; !ok {
			continue
		}
		direction := strings.TrimSpace(entry.Direction)
		if direction != network.AuditDirectionSent && direction != network.AuditDirectionReceived {
			continue
		}
		if _, seen := directions[messageID]; seen {
			continue
		}
		directions[messageID] = direction
	}
	return directions
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

// NetworkChannelMessagePayloadFromEntry converts one persisted timeline row into the shared payload.
func NetworkChannelMessagePayloadFromEntry(
	entry store.NetworkMessageEntry,
	auditDirection string,
	sessionsByID map[string]*session.Info,
	peersByID map[string]network.PeerInfo,
) contract.NetworkChannelMessagePayload {
	storedSessionID := strings.TrimSpace(entry.SessionID)
	displayName := strings.TrimSpace(entry.PeerFrom)
	local := false
	payloadSessionID := ""

	if peer, ok := peersByID[strings.TrimSpace(entry.PeerFrom)]; ok {
		displayName = networkPeerDisplayName(peer, sessionsByID)
		local = peer.Local
	}

	switch strings.TrimSpace(auditDirection) {
	case network.AuditDirectionSent:
		local = true
		payloadSessionID = storedSessionID
	case network.AuditDirectionReceived:
		local = false
	default:
		if local {
			payloadSessionID = storedSessionID
		}
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

	return contract.NetworkChannelMessagePayload{
		MessageID:   strings.TrimSpace(entry.MessageID),
		Channel:     strings.TrimSpace(entry.Channel),
		PeerID:      strings.TrimSpace(entry.PeerFrom),
		DisplayName: displayName,
		SessionID:   payloadSessionID,
		Local:       local,
		Intent:      strings.TrimSpace(entry.Intent),
		Text:        strings.TrimSpace(entry.Text),
		Timestamp:   entry.Timestamp.UTC(),
	}
}

func (h *BaseHandlers) loadPeerAuditEntries(
	ctx context.Context,
	networkStore NetworkStore,
	peer network.PeerInfo,
) ([]store.NetworkAuditEntry, error) {
	if peer.SessionID != nil {
		return networkStore.ListNetworkAudit(ctx, store.NetworkAuditQuery{
			SessionID: strings.TrimSpace(*peer.SessionID),
		})
	}

	entries, err := networkStore.ListNetworkAudit(ctx, store.NetworkAuditQuery{
		Channel: strings.TrimSpace(peer.Channel),
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
