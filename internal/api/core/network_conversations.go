package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/network"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
	"github.com/gin-gonic/gin"
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
	peerCosts, err := h.networkThreadPeerCosts(c.Request.Context(), scope.NetworkWorkspaceID(), channel, threadID)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	taskLinks, err := h.networkThreadTaskLinks(c.Request.Context(), scope.NetworkWorkspaceID(), channel, threadID)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.NetworkThreadResponse{
		Thread:    NetworkThreadSummaryPayloadFromStore(thread),
		PeerCosts: NetworkThreadPeerCostPayloadsFromStore(peerCosts),
		TaskLinks: NetworkTaskThreadOriginPayloadsFromStore(taskLinks),
	})
}

// PromoteNetworkThreadTask creates a durable task from a public-thread message.
func (h *BaseHandlers) PromoteNetworkThreadTask(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}
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
	var req contract.PromoteNetworkThreadTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode promote network thread task request: %w", h.transportName(), err),
		)
		return
	}
	originMessageID := strings.TrimSpace(req.OriginMessageID)
	if originMessageID == "" {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(errors.New("origin_message_id is required")))
		return
	}
	source, err := h.networkThreadPromotionSource(
		c.Request.Context(),
		scope.NetworkWorkspaceID(),
		channel,
		threadID,
		originMessageID,
	)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	actor, err := h.taskActorContextForWorkspace(c, taskActionPromoteNetwork, scope.NetworkWorkspaceID())
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	taskRecord, err := h.createPromotedNetworkThreadTask(c.Request.Context(), manager, actor, source, req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	origin, err := h.writePromotedNetworkThreadOrigin(c.Request.Context(), taskRecord.ID, source)
	if err != nil {
		if cleanupErr := manager.DeleteTask(c.Request.Context(), taskRecord.ID, actor); cleanupErr != nil {
			err = errors.Join(err, fmt.Errorf("rollback promoted network task %q: %w", taskRecord.ID, cleanupErr))
		}
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	c.JSON(http.StatusCreated, contract.PromoteNetworkThreadTaskResponse{
		Task:   TaskPayloadFromTask(taskRecord),
		Origin: NetworkTaskThreadOriginPayloadFromStore(origin),
	})
}

type networkThreadPromotionSource struct {
	workspaceID      string
	channel          string
	threadID         string
	originMessageID  string
	digest           string
	sourceMessageIDs []string
	thread           store.NetworkThreadSummary
}

func (h *BaseHandlers) networkThreadPromotionSource(
	ctx context.Context,
	workspaceID string,
	channel string,
	threadID string,
	originMessageID string,
) (networkThreadPromotionSource, error) {
	ref := store.NetworkConversationRef{
		WorkspaceID: workspaceID,
		Channel:     channel,
		Surface:     store.NetworkSurfaceThread,
		ThreadID:    threadID,
	}
	messages, err := h.NetworkStore.ListConversationMessages(
		ctx,
		ref,
		store.NetworkConversationMessageQuery{Limit: 200},
	)
	if err != nil {
		return networkThreadPromotionSource{}, err
	}
	digest, sourceMessageIDs, found := networkThreadPromotionDigest(messages, originMessageID)
	if !found {
		return networkThreadPromotionSource{}, NewNetworkValidationError(
			fmt.Errorf("origin_message_id %q is not part of thread %q", originMessageID, threadID),
		)
	}
	channelRef := store.NetworkChannelRef{WorkspaceID: workspaceID, Channel: channel}
	thread, err := h.NetworkStore.GetThread(ctx, channelRef, threadID)
	if err != nil {
		return networkThreadPromotionSource{}, err
	}
	return networkThreadPromotionSource{
		workspaceID:      workspaceID,
		channel:          channel,
		threadID:         threadID,
		originMessageID:  originMessageID,
		digest:           digest,
		sourceMessageIDs: sourceMessageIDs,
		thread:           thread,
	}, nil
}

func (h *BaseHandlers) createPromotedNetworkThreadTask(
	ctx context.Context,
	manager TaskService,
	actor taskpkg.ActorContext,
	source networkThreadPromotionSource,
	req contract.PromoteNetworkThreadTaskRequest,
) (*taskpkg.Task, error) {
	spec := taskpkg.CreateTask{
		Scope:          taskpkg.ScopeWorkspace,
		WorkspaceID:    source.workspaceID,
		NetworkChannel: source.channel,
		Title:          promotedNetworkThreadTaskTitle(req, source),
		Description:    firstNonEmpty(strings.TrimSpace(req.Description), source.digest),
		Priority:       taskpkg.Priority(strings.TrimSpace(req.Priority)).Normalize(),
		Metadata:       cloneRawMessage(req.Metadata),
	}
	if err := spec.Validate("promote_network_thread"); err != nil {
		return nil, err
	}
	return manager.CreateTask(ctx, spec, actor)
}

func promotedNetworkThreadTaskTitle(
	req contract.PromoteNetworkThreadTaskRequest,
	source networkThreadPromotionSource,
) string {
	return firstNonEmpty(
		strings.TrimSpace(req.Title),
		strings.TrimSpace(source.thread.Title),
		"Network thread "+source.threadID,
	)
}

func (h *BaseHandlers) writePromotedNetworkThreadOrigin(
	ctx context.Context,
	taskID string,
	source networkThreadPromotionSource,
) (store.NetworkTaskThreadOrigin, error) {
	now := h.nowUTC()
	origin := store.NetworkTaskThreadOrigin{
		TaskID:           taskID,
		WorkspaceID:      source.workspaceID,
		Channel:          source.channel,
		ThreadID:         source.threadID,
		OriginMessageID:  source.originMessageID,
		Digest:           source.digest,
		SourceMessageIDs: source.sourceMessageIDs,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := origin.Validate(); err != nil {
		return store.NetworkTaskThreadOrigin{}, err
	}
	return origin, h.NetworkStore.PutNetworkTaskThreadOrigin(ctx, origin)
}

// NetworkSubscriptions returns peer delivery preferences for a channel or thread.
func (h *BaseHandlers) NetworkSubscriptions(c *gin.Context) {
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
	limit, err := parsePositiveIntQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(err))
		return
	}
	if limit == 0 {
		limit = 100
	}
	query := store.NetworkSubscriptionQuery{
		WorkspaceID: scope.NetworkWorkspaceID(),
		Channel:     channel,
		ThreadID:    strings.TrimSpace(c.Query("thread_id")),
		PeerID:      strings.TrimSpace(c.Query("peer_id")),
		Limit:       limit,
	}
	if err := query.Validate(); err != nil {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(err))
		return
	}
	subscriptions, err := h.NetworkStore.ListNetworkSubscriptions(c.Request.Context(), query)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.NetworkSubscriptionsResponse{
		Subscriptions: NetworkSubscriptionPayloadsFromStore(subscriptions),
	})
}

// UpsertNetworkSubscription sets one peer delivery preference for a channel or thread.
func (h *BaseHandlers) UpsertNetworkSubscription(c *gin.Context) {
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
	var req contract.NetworkSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode network subscription request: %w", h.transportName(), err),
		)
		return
	}
	now := h.nowUTC()
	entry := store.NetworkSubscriptionEntry{
		WorkspaceID:    scope.NetworkWorkspaceID(),
		Channel:        channel,
		ThreadID:       strings.TrimSpace(req.ThreadID),
		PeerID:         strings.TrimSpace(req.PeerID),
		Mode:           strings.TrimSpace(req.Mode),
		KeywordFilters: cloneTrimmedStrings(req.KeywordFilters),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := entry.Validate(); err != nil {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(err))
		return
	}
	channelMetadata, err := h.networkChannelMetadataForUpdate(
		c.Request.Context(),
		h.NetworkStore,
		store.NetworkChannelRef{WorkspaceID: entry.WorkspaceID, Channel: entry.Channel},
	)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	if err := h.NetworkStore.WriteNetworkChannel(c.Request.Context(), channelMetadata); err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	if err := h.NetworkStore.PutNetworkSubscription(c.Request.Context(), entry); err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.NetworkSubscriptionResponse{
		Subscription: NetworkSubscriptionPayloadFromStore(entry),
	})
}

// DeleteNetworkSubscription removes one peer delivery preference.
func (h *BaseHandlers) DeleteNetworkSubscription(c *gin.Context) {
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
	ref := store.NetworkSubscriptionRef{
		WorkspaceID: scope.NetworkWorkspaceID(),
		Channel:     channel,
		ThreadID:    strings.TrimSpace(c.Query("thread_id")),
		PeerID:      strings.TrimSpace(c.Param("peer_id")),
	}
	if err := ref.Validate(); err != nil {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(err))
		return
	}
	if err := h.NetworkStore.DeleteNetworkSubscription(c.Request.Context(), ref); err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	c.Status(http.StatusNoContent)
}

type networkThreadPeerCostStore interface {
	ListNetworkThreadPeerTokenStats(
		ctx context.Context,
		query store.NetworkThreadPeerTokenStatsQuery,
	) ([]store.NetworkThreadPeerTokenStats, error)
}

type networkThreadTaskLinkStore interface {
	ListNetworkTaskThreadOrigins(
		ctx context.Context,
		query store.NetworkTaskThreadOriginQuery,
	) ([]store.NetworkTaskThreadOrigin, error)
}

func (h *BaseHandlers) networkThreadPeerCosts(
	ctx context.Context,
	workspaceID string,
	channel string,
	threadID string,
) ([]store.NetworkThreadPeerTokenStats, error) {
	costStore, ok := h.NetworkStore.(networkThreadPeerCostStore)
	if !ok {
		return nil, nil
	}
	return costStore.ListNetworkThreadPeerTokenStats(
		ctx,
		store.NetworkThreadPeerTokenStatsQuery{
			WorkspaceID: workspaceID,
			Channel:     channel,
			ThreadID:    threadID,
		},
	)
}

func (h *BaseHandlers) networkThreadTaskLinks(
	ctx context.Context,
	workspaceID string,
	channel string,
	threadID string,
) ([]store.NetworkTaskThreadOrigin, error) {
	linkStore, ok := h.NetworkStore.(networkThreadTaskLinkStore)
	if !ok {
		return nil, nil
	}
	return linkStore.ListNetworkTaskThreadOrigins(
		ctx,
		store.NetworkTaskThreadOriginQuery{
			WorkspaceID: strings.TrimSpace(workspaceID),
			Channel:     strings.TrimSpace(channel),
			ThreadID:    strings.TrimSpace(threadID),
		},
	)
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

func networkThreadPromotionDigest(
	messages []store.NetworkConversationMessage,
	originMessageID string,
) (string, []string, bool) {
	target := strings.TrimSpace(originMessageID)
	sourceIDs := make([]string, 0, len(messages))
	var builder strings.Builder
	found := false
	for _, message := range messages {
		messageID := strings.TrimSpace(message.MessageID)
		if messageID == "" {
			continue
		}
		sourceIDs = append(sourceIDs, messageID)
		if messageID == target {
			found = true
		}
		if builder.Len() >= 4000 {
			continue
		}
		line := networkThreadPromotionMessageLine(message)
		if line == "" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(line)
	}
	digest := strings.TrimSpace(builder.String())
	if len(digest) > 4000 {
		digest = truncateNetworkPromotionDigest(digest, 4000)
	}
	return digest, sourceIDs, found
}

func truncateNetworkPromotionDigest(value string, maxBytes int) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return strings.TrimSpace(value)
	}
	truncated := value[:maxBytes]
	for !utf8.ValidString(truncated) && truncated != "" {
		truncated = truncated[:len(truncated)-1]
	}
	return strings.TrimSpace(truncated)
}

func networkThreadPromotionMessageLine(message store.NetworkConversationMessage) string {
	preview := strings.TrimSpace(message.PreviewText)
	if preview == "" {
		preview = strings.TrimSpace(message.Text)
	}
	if preview == "" {
		return ""
	}
	peer := firstNonEmpty(strings.TrimSpace(message.PeerFrom), strings.TrimSpace(message.SessionID), unknownValue)
	return peer + ": " + preview
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
		CoordinationCost:   NetworkCoordinationCostPayloadFromStore(thread),
		LastMessagePreview: strings.TrimSpace(thread.LastMessagePreview),
	}
}

// NetworkCoordinationCostPayloadFromStore converts stored thread cost counters into public payloads.
func NetworkCoordinationCostPayloadFromStore(
	thread store.NetworkThreadSummary,
) *contract.NetworkCoordinationCostPayload {
	if thread.DeliveredCount == 0 && thread.PromptSizeBytes == 0 && thread.EstimatedPromptTokens == 0 {
		return nil
	}
	return &contract.NetworkCoordinationCostPayload{
		DeliveredCount:        thread.DeliveredCount,
		PromptSizeBytes:       thread.PromptSizeBytes,
		EstimatedPromptTokens: thread.EstimatedPromptTokens,
	}
}

// NetworkThreadPeerCostPayloadsFromStore converts stored thread peer cost rows into public payloads.
func NetworkThreadPeerCostPayloadsFromStore(
	stats []store.NetworkThreadPeerTokenStats,
) []contract.NetworkThreadPeerCostPayload {
	payload := make([]contract.NetworkThreadPeerCostPayload, 0, len(stats))
	for _, stat := range stats {
		payload = append(payload, NetworkThreadPeerCostPayloadFromStore(stat))
	}
	return payload
}

// NetworkThreadPeerCostPayloadFromStore converts one stored thread peer cost row into a public payload.
func NetworkThreadPeerCostPayloadFromStore(
	stat store.NetworkThreadPeerTokenStats,
) contract.NetworkThreadPeerCostPayload {
	return contract.NetworkThreadPeerCostPayload{
		PeerID:                strings.TrimSpace(stat.PeerID),
		DeliveredCount:        stat.DeliveredCount,
		PromptSizeBytes:       stat.PromptSizeBytes,
		EstimatedPromptTokens: stat.EstimatedPromptTokens,
		FirstDeliveredAt:      cloneTimePtr(&stat.FirstDeliveredAt),
		LastDeliveredAt:       cloneTimePtr(&stat.LastDeliveredAt),
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
		Mentions:    cloneTrimmedStrings(message.Mentions),
		SessionID:   strings.TrimSpace(message.SessionID),
		WorkID:      strings.TrimSpace(message.WorkID),
		ReplyTo:     strings.TrimSpace(message.ReplyTo),
		TraceID:     strings.TrimSpace(message.TraceID),
		CausationID: strings.TrimSpace(message.CausationID),
		Intent:      strings.TrimSpace(message.Intent),
		Text:        strings.TrimSpace(message.Text),
		PreviewText: networkMessagePreview(message),
		SizeBytes:   message.SizeBytes,
		Body:        cloneRawMessage(message.Body),
		Timestamp:   message.Timestamp.UTC(),
	}
}

// NetworkSubscriptionPayloadsFromStore converts delivery preferences into public payloads.
func NetworkSubscriptionPayloadsFromStore(
	subscriptions []store.NetworkSubscriptionEntry,
) []contract.NetworkSubscriptionPayload {
	payload := make([]contract.NetworkSubscriptionPayload, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		payload = append(payload, NetworkSubscriptionPayloadFromStore(subscription))
	}
	return payload
}

// NetworkSubscriptionPayloadFromStore converts one delivery preference into a public payload.
func NetworkSubscriptionPayloadFromStore(
	subscription store.NetworkSubscriptionEntry,
) contract.NetworkSubscriptionPayload {
	return contract.NetworkSubscriptionPayload{
		WorkspaceID:    strings.TrimSpace(subscription.WorkspaceID),
		Channel:        strings.TrimSpace(subscription.Channel),
		ThreadID:       strings.TrimSpace(subscription.ThreadID),
		PeerID:         strings.TrimSpace(subscription.PeerID),
		Mode:           strings.TrimSpace(subscription.Mode),
		KeywordFilters: cloneTrimmedStrings(subscription.KeywordFilters),
		CreatedAt:      cloneTimePtr(&subscription.CreatedAt),
		UpdatedAt:      cloneTimePtr(&subscription.UpdatedAt),
	}
}

// NetworkTaskThreadOriginPayloadsFromStore converts promoted-task origin links into public payloads.
func NetworkTaskThreadOriginPayloadsFromStore(
	origins []store.NetworkTaskThreadOrigin,
) []contract.NetworkTaskThreadOriginPayload {
	payload := make([]contract.NetworkTaskThreadOriginPayload, 0, len(origins))
	for _, origin := range origins {
		payload = append(payload, NetworkTaskThreadOriginPayloadFromStore(origin))
	}
	return payload
}

// NetworkTaskThreadOriginPayloadFromStore converts one promoted-task origin link into a public payload.
func NetworkTaskThreadOriginPayloadFromStore(
	origin store.NetworkTaskThreadOrigin,
) contract.NetworkTaskThreadOriginPayload {
	return contract.NetworkTaskThreadOriginPayload{
		TaskID:           strings.TrimSpace(origin.TaskID),
		WorkspaceID:      strings.TrimSpace(origin.WorkspaceID),
		Channel:          strings.TrimSpace(origin.Channel),
		ThreadID:         strings.TrimSpace(origin.ThreadID),
		OriginMessageID:  strings.TrimSpace(origin.OriginMessageID),
		Digest:           strings.TrimSpace(origin.Digest),
		SourceMessageIDs: cloneTrimmedStrings(origin.SourceMessageIDs),
		CreatedAt:        cloneTimePtr(&origin.CreatedAt),
		UpdatedAt:        cloneTimePtr(&origin.UpdatedAt),
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
