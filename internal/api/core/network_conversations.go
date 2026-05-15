package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/store"
)

// NetworkThreads returns public-thread summaries for one channel.
func (h *BaseHandlers) NetworkThreads(c *gin.Context) {
	if _, ok := h.requireNetworkReadDependencies(c); !ok {
		return
	}
	networkStore := h.NetworkStore
	channel, err := normalizeNetworkChannel(c.Param("channel"))
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	scope, ok := h.resolveWorkspaceScope(c)
	if !ok {
		return
	}
	query, err := parseNetworkThreadQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	threads, err := networkStore.ListThreads(c.Request.Context(), scope.NetworkChannelRef(channel), query)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.NetworkThreadsResponse{
		Threads: NetworkThreadSummaryPayloadsFromStore(threads),
	})
}

// NetworkThread returns one public-thread summary.
func (h *BaseHandlers) NetworkThread(c *gin.Context) {
	if _, ok := h.requireNetworkReadDependencies(c); !ok {
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
	threadID := strings.TrimSpace(c.Param("thread_id"))
	if err := network.ValidateConversationID(threadID, "thread_id"); err != nil {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(err))
		return
	}

	thread, err := h.NetworkStore.GetThread(c.Request.Context(), scope.NetworkChannelRef(channel), threadID)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.NetworkThreadResponse{
		Thread: NetworkThreadSummaryPayloadFromStore(thread),
	})
}

// NetworkThreadMessages returns messages isolated to one public thread.
func (h *BaseHandlers) NetworkThreadMessages(c *gin.Context) {
	if _, ok := h.requireNetworkReadDependencies(c); !ok {
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
	threadID := strings.TrimSpace(c.Param("thread_id"))
	ref := store.NetworkConversationRef{
		WorkspaceID: scope.NetworkWorkspaceID(),
		Channel:     channel,
		Surface:     store.NetworkSurfaceThread,
		ThreadID:    threadID,
	}
	h.respondNetworkConversationMessages(c, ref)
}

// NetworkDirectRooms returns direct-room summaries for one channel.
func (h *BaseHandlers) NetworkDirectRooms(c *gin.Context) {
	if _, ok := h.requireNetworkReadDependencies(c); !ok {
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
	query, err := parseNetworkDirectRoomQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	directs, err := h.NetworkStore.ListDirectRooms(c.Request.Context(), scope.NetworkChannelRef(channel), query)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.NetworkDirectRoomsResponse{
		Directs: NetworkDirectRoomPayloadsFromStore(directs),
	})
}

// ResolveNetworkDirectRoom creates or returns a deterministic two-party direct room.
func (h *BaseHandlers) ResolveNetworkDirectRoom(c *gin.Context) {
	service, ok := h.requireNetworkReadDependencies(c)
	if !ok {
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

	var req contract.NetworkDirectResolveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode network direct resolve request: %w", h.transportName(), err),
		)
		return
	}
	sessionID := strings.TrimSpace(req.SessionID)
	peerID := strings.TrimSpace(req.PeerID)
	if sessionID == "" {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(errors.New("session_id is required")))
		return
	}
	if err := network.ValidatePeerID(peerID); err != nil {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(err))
		return
	}

	localPeer, remotePeer, err := h.resolveDirectRoomPeers(
		c.Request.Context(),
		service,
		scope.NetworkWorkspaceID(),
		channel,
		sessionID,
		peerID,
	)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	networkWorkspaceID := scope.NetworkWorkspaceID()
	directID, peerA, peerB, err := network.DirectRoomIdentity(
		networkWorkspaceID,
		channel,
		localPeer.PeerID,
		remotePeer.PeerID,
	)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	now := h.nowUTC()
	direct, err := h.NetworkStore.ResolveDirectRoom(c.Request.Context(), store.NetworkDirectRoomEntry{
		WorkspaceID:    networkWorkspaceID,
		Channel:        channel,
		DirectID:       directID,
		PeerA:          peerA,
		PeerB:          peerB,
		OpenedAt:       now,
		LastActivityAt: now,
	})
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.NetworkDirectRoomResponse{
		Direct: NetworkDirectRoomPayloadFromStore(direct),
	})
}

// NetworkDirectRoom returns one direct-room summary.
func (h *BaseHandlers) NetworkDirectRoom(c *gin.Context) {
	if _, ok := h.requireNetworkReadDependencies(c); !ok {
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
	directID := strings.TrimSpace(c.Param("direct_id"))
	if err := network.ValidateConversationID(directID, "direct_id"); err != nil {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(err))
		return
	}

	direct, err := h.NetworkStore.GetDirectRoom(c.Request.Context(), scope.NetworkChannelRef(channel), directID)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.NetworkDirectRoomResponse{
		Direct: NetworkDirectRoomPayloadFromStore(direct),
	})
}

// NetworkDirectRoomMessages returns messages isolated to one direct room.
func (h *BaseHandlers) NetworkDirectRoomMessages(c *gin.Context) {
	if _, ok := h.requireNetworkReadDependencies(c); !ok {
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
	directID := strings.TrimSpace(c.Param("direct_id"))
	ref := store.NetworkConversationRef{
		WorkspaceID: scope.NetworkWorkspaceID(),
		Channel:     channel,
		Surface:     store.NetworkSurfaceDirect,
		DirectID:    directID,
	}
	h.respondNetworkConversationMessages(c, ref)
}

// NetworkWork returns one network work row by work_id.
func (h *BaseHandlers) NetworkWork(c *gin.Context) {
	if _, ok := h.requireNetworkReadDependencies(c); !ok {
		return
	}
	workID := strings.TrimSpace(c.Param("work_id"))
	if err := network.ValidateWorkID(workID); err != nil {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(err))
		return
	}
	scope, ok := h.resolveWorkspaceScope(c)
	if !ok {
		return
	}
	work, err := h.NetworkStore.GetWork(c.Request.Context(), scope.NetworkWorkspaceID(), workID)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.NetworkWorkResponse{Work: NetworkWorkPayloadFromStore(work)})
}

func (h *BaseHandlers) requireNetworkReadDependencies(c *gin.Context) (NetworkService, bool) {
	service, err := h.networkServiceRequired()
	if err != nil {
		h.respondError(c, http.StatusServiceUnavailable, err)
		return nil, false
	}
	if _, err := h.networkStoreRequired(); err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return nil, false
	}
	return service, true
}

func (h *BaseHandlers) respondNetworkConversationMessages(c *gin.Context, ref store.NetworkConversationRef) {
	if err := ref.Validate(); err != nil {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(err))
		return
	}
	query, err := parseNetworkConversationMessageQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	messages, err := h.NetworkStore.ListConversationMessages(c.Request.Context(), ref, query)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	payload := NetworkConversationMessagePayloadsFromStore(messages)
	switch ref.Surface {
	case store.NetworkSurfaceThread:
		c.JSON(http.StatusOK, contract.NetworkThreadMessagesResponse{Messages: payload})
	case store.NetworkSurfaceDirect:
		c.JSON(http.StatusOK, contract.NetworkDirectRoomMessagesResponse{Messages: payload})
	default:
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(errors.New("surface is required")))
	}
}

func (h *BaseHandlers) resolveDirectRoomPeers(
	ctx context.Context,
	service NetworkService,
	workspaceID string,
	channel string,
	sessionID string,
	peerID string,
) (network.PeerInfo, network.PeerInfo, error) {
	peers, err := service.ListPeers(ctx, workspaceID, channel)
	if err != nil {
		return network.PeerInfo{}, network.PeerInfo{}, err
	}
	var local network.PeerInfo
	localFound := false
	remote, remoteFound := findPeerInfo(peers, peerID)
	for _, peer := range peers {
		if peer.SessionID == nil || strings.TrimSpace(*peer.SessionID) != sessionID {
			continue
		}
		if !peer.Local {
			continue
		}
		local = peer
		localFound = true
		break
	}
	if !localFound {
		return network.PeerInfo{}, network.PeerInfo{}, fmt.Errorf(
			"%w: session=%q channel=%q",
			network.ErrLocalPeerNotFound,
			sessionID,
			channel,
		)
	}
	if !remoteFound {
		return network.PeerInfo{}, network.PeerInfo{}, fmt.Errorf(
			"%w: peer_id=%q channel=%q",
			network.ErrTargetPeerNotFound,
			peerID,
			channel,
		)
	}
	return local, remote, nil
}

func parseNetworkThreadQuery(c *gin.Context) (store.NetworkThreadQuery, error) {
	limit, err := parsePositiveIntQuery(c)
	if err != nil {
		return store.NetworkThreadQuery{}, NewNetworkValidationError(err)
	}
	query := store.NetworkThreadQuery{
		Limit: limit,
		After: strings.TrimSpace(c.Query("after")),
	}
	if err := query.Validate(); err != nil {
		return store.NetworkThreadQuery{}, NewNetworkValidationError(err)
	}
	return query, nil
}

func parseNetworkDirectRoomQuery(c *gin.Context) (store.NetworkDirectRoomQuery, error) {
	limit, err := parsePositiveIntQuery(c)
	if err != nil {
		return store.NetworkDirectRoomQuery{}, NewNetworkValidationError(err)
	}
	query := store.NetworkDirectRoomQuery{
		PeerID: strings.TrimSpace(c.Query("peer_id")),
		Limit:  limit,
		After:  strings.TrimSpace(c.Query("after")),
	}
	if err := query.Validate(); err != nil {
		return store.NetworkDirectRoomQuery{}, NewNetworkValidationError(err)
	}
	return query, nil
}

func parseNetworkConversationMessageQuery(c *gin.Context) (store.NetworkConversationMessageQuery, error) {
	limit, err := parsePositiveIntQuery(c)
	if err != nil {
		return store.NetworkConversationMessageQuery{}, NewNetworkValidationError(err)
	}
	query := store.NetworkConversationMessageQuery{
		BeforeMessageID: strings.TrimSpace(c.Query("before")),
		AfterMessageID:  strings.TrimSpace(c.Query("after")),
		Kind:            strings.TrimSpace(c.Query("kind")),
		WorkID:          strings.TrimSpace(c.Query("work_id")),
		Limit:           limit,
	}
	if err := query.Validate(); err != nil {
		return store.NetworkConversationMessageQuery{}, NewNetworkValidationError(err)
	}
	return query, nil
}

// NetworkThreadSummaryPayloadsFromStore converts stored thread summaries into public payloads.
func NetworkThreadSummaryPayloadsFromStore(
	threads []store.NetworkThreadSummary,
) []contract.NetworkThreadSummaryPayload {
	payload := make([]contract.NetworkThreadSummaryPayload, 0, len(threads))
	for _, thread := range threads {
		payload = append(payload, NetworkThreadSummaryPayloadFromStore(thread))
	}
	return payload
}

// NetworkThreadSummaryPayloadFromStore converts one stored thread summary into a public payload.
func NetworkThreadSummaryPayloadFromStore(thread store.NetworkThreadSummary) contract.NetworkThreadSummaryPayload {
	return contract.NetworkThreadSummaryPayload{
		WorkspaceID:        strings.TrimSpace(thread.WorkspaceID),
		Channel:            strings.TrimSpace(thread.Channel),
		ThreadID:           strings.TrimSpace(thread.ThreadID),
		RootMessageID:      strings.TrimSpace(thread.RootMessageID),
		Title:              strings.TrimSpace(thread.Title),
		OpenedByPeerID:     strings.TrimSpace(thread.OpenedByPeerID),
		OpenedSessionID:    strings.TrimSpace(thread.OpenedSessionID),
		OpenedAt:           cloneTimePtr(&thread.OpenedAt),
		LastActivityAt:     cloneTimePtr(&thread.LastActivityAt),
		MessageCount:       thread.MessageCount,
		ParticipantCount:   thread.ParticipantCount,
		OpenWorkCount:      thread.OpenWorkCount,
		LastMessagePreview: strings.TrimSpace(thread.LastMessagePreview),
	}
}

// NetworkDirectRoomPayloadsFromStore converts stored direct rooms into public payloads.
func NetworkDirectRoomPayloadsFromStore(
	directs []store.NetworkDirectRoomSummary,
) []contract.NetworkDirectRoomPayload {
	payload := make([]contract.NetworkDirectRoomPayload, 0, len(directs))
	for _, direct := range directs {
		payload = append(payload, NetworkDirectRoomPayloadFromStore(direct))
	}
	return payload
}

// NetworkDirectRoomPayloadFromStore converts one stored direct-room summary into a public payload.
func NetworkDirectRoomPayloadFromStore(direct store.NetworkDirectRoomSummary) contract.NetworkDirectRoomPayload {
	return contract.NetworkDirectRoomPayload{
		WorkspaceID:        strings.TrimSpace(direct.WorkspaceID),
		Channel:            strings.TrimSpace(direct.Channel),
		DirectID:           strings.TrimSpace(direct.DirectID),
		PeerA:              strings.TrimSpace(direct.PeerA),
		PeerB:              strings.TrimSpace(direct.PeerB),
		OpenedAt:           cloneTimePtr(&direct.OpenedAt),
		LastActivityAt:     cloneTimePtr(&direct.LastActivityAt),
		MessageCount:       direct.MessageCount,
		OpenWorkCount:      direct.OpenWorkCount,
		LastMessagePreview: strings.TrimSpace(direct.LastMessagePreview),
	}
}

// NetworkConversationMessagePayloadsFromStore converts stored messages into public payloads.
func NetworkConversationMessagePayloadsFromStore(
	messages []store.NetworkConversationMessage,
) []contract.NetworkConversationMessagePayload {
	payload := make([]contract.NetworkConversationMessagePayload, 0, len(messages))
	for _, message := range messages {
		payload = append(payload, NetworkConversationMessagePayloadFromStore(message))
	}
	return payload
}

// NetworkConversationMessagePayloadFromStore converts one stored message into a public payload.
func NetworkConversationMessagePayloadFromStore(
	message store.NetworkConversationMessage,
) contract.NetworkConversationMessagePayload {
	return contract.NetworkConversationMessagePayload{
		MessageID:   strings.TrimSpace(message.MessageID),
		WorkspaceID: strings.TrimSpace(message.WorkspaceID),
		Channel:     strings.TrimSpace(message.Channel),
		Surface:     strings.TrimSpace(message.Surface),
		ThreadID:    strings.TrimSpace(message.ThreadID),
		DirectID:    strings.TrimSpace(message.DirectID),
		Kind:        strings.TrimSpace(message.Kind),
		Direction:   strings.TrimSpace(message.Direction),
		PeerFrom:    strings.TrimSpace(message.PeerFrom),
		PeerTo:      strings.TrimSpace(message.PeerTo),
		SessionID:   strings.TrimSpace(message.SessionID),
		WorkID:      strings.TrimSpace(message.WorkID),
		ReplyTo:     strings.TrimSpace(message.ReplyTo),
		TraceID:     strings.TrimSpace(message.TraceID),
		CausationID: strings.TrimSpace(message.CausationID),
		Intent:      strings.TrimSpace(message.Intent),
		Text:        strings.TrimSpace(message.Text),
		PreviewText: networkMessagePreview(message),
		Body:        cloneRawMessage(message.Body),
		Timestamp:   message.Timestamp.UTC(),
	}
}

// NetworkWorkPayloadFromStore converts one network work row into a public payload.
func NetworkWorkPayloadFromStore(work store.NetworkWorkEntry) contract.NetworkWorkPayload {
	return contract.NetworkWorkPayload{
		WorkID:          strings.TrimSpace(work.WorkID),
		WorkspaceID:     strings.TrimSpace(work.WorkspaceID),
		Channel:         strings.TrimSpace(work.Channel),
		Surface:         strings.TrimSpace(work.Surface),
		ThreadID:        strings.TrimSpace(work.ThreadID),
		DirectID:        strings.TrimSpace(work.DirectID),
		OpenedByPeerID:  strings.TrimSpace(work.OpenedByPeerID),
		OpenedSessionID: strings.TrimSpace(work.OpenedSessionID),
		TargetPeerID:    strings.TrimSpace(work.TargetPeerID),
		State:           strings.TrimSpace(work.State),
		OpenedAt:        cloneTimePtr(&work.OpenedAt),
		LastActivityAt:  cloneTimePtr(&work.LastActivityAt),
		TerminalAt:      cloneTimePtr(work.TerminalAt),
	}
}
