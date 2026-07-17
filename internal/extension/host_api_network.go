package extensionpkg

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	apicontract "github.com/compozy/agh/internal/api/contract"
	extensioncontract "github.com/compozy/agh/internal/extension/contract"
	"github.com/compozy/agh/internal/network"
	"github.com/compozy/agh/internal/store"
)

func registerHostAPINetworkMethodHandlers(
	handler *HostAPIHandler,
	handlers map[string]hostAPIMethodFunc,
) {
	handlers[string(extensioncontract.HostAPIMethodNetworkStatus)] = handler.handleNetworkStatus
	handlers[string(extensioncontract.HostAPIMethodNetworkUsage)] = handler.handleNetworkUsage
	handlers[string(extensioncontract.HostAPIMethodNetworkChannels)] = handler.handleNetworkChannels
	handlers[string(extensioncontract.HostAPIMethodNetworkPeers)] = handler.handleNetworkPeers
	handlers[string(extensioncontract.HostAPIMethodNetworkThreads)] = handler.handleNetworkThreads
	handlers[string(extensioncontract.HostAPIMethodNetworkThreadGet)] = handler.handleNetworkThreadGet
	handlers[string(extensioncontract.HostAPIMethodNetworkThreadMessages)] = handler.handleNetworkThreadMessages
	handlers[string(extensioncontract.HostAPIMethodNetworkDirects)] = handler.handleNetworkDirects
	handlers[string(extensioncontract.HostAPIMethodNetworkDirectResolve)] = handler.handleNetworkDirectResolve
	handlers[string(extensioncontract.HostAPIMethodNetworkDirectMessages)] = handler.handleNetworkDirectMessages
	handlers[string(extensioncontract.HostAPIMethodNetworkWorkGet)] = handler.handleNetworkWorkGet
	handlers[string(extensioncontract.HostAPIMethodNetworkSend)] = handler.handleNetworkSend
}

func (h *HostAPIHandler) handleNetworkChannels(ctx context.Context, raw json.RawMessage) (any, error) {
	var params extensioncontract.NetworkChannelsParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	service, err := h.requireHostAPINetworkService()
	if err != nil {
		return nil, err
	}
	workspaceID, err := h.hostAPINetworkWorkspaceID(ctx, params.WorkspaceID)
	if err != nil {
		return nil, err
	}
	channels, err := service.ListChannels(ctx, workspaceID)
	if err != nil {
		return nil, mapHostAPINetworkRPCError(err)
	}
	return hostAPINetworkChannelPayloads(channels), nil
}

func (h *HostAPIHandler) handleNetworkPeers(ctx context.Context, raw json.RawMessage) (any, error) {
	var params extensioncontract.NetworkPeersParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	service, err := h.requireHostAPINetworkService()
	if err != nil {
		return nil, err
	}
	channel := strings.TrimSpace(params.Channel)
	if channel != "" {
		if err := network.ValidateChannel(channel); err != nil {
			return nil, invalidParamsRPCError(err)
		}
	}
	workspaceID, err := h.hostAPINetworkWorkspaceID(ctx, params.WorkspaceID)
	if err != nil {
		return nil, err
	}
	peers, err := service.ListPeers(ctx, workspaceID, channel)
	if err != nil {
		return nil, mapHostAPINetworkRPCError(err)
	}
	return hostAPINetworkPeerPayloads(peers), nil
}

func (h *HostAPIHandler) handleNetworkThreadGet(ctx context.Context, raw json.RawMessage) (any, error) {
	var params extensioncontract.NetworkThreadTargetParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	networkStore, err := h.requireHostAPINetworkStore()
	if err != nil {
		return nil, err
	}
	channel, err := hostAPINetworkChannel(params.Channel)
	if err != nil {
		return nil, err
	}
	threadID := strings.TrimSpace(params.ThreadID)
	if err := network.ValidateConversationID(threadID, "thread_id"); err != nil {
		return nil, invalidParamsRPCError(err)
	}
	workspaceID, err := h.hostAPINetworkWorkspaceID(ctx, params.WorkspaceID)
	if err != nil {
		return nil, err
	}
	thread, err := networkStore.GetThread(
		ctx,
		store.NetworkChannelRef{WorkspaceID: workspaceID, Channel: channel},
		threadID,
	)
	if err != nil {
		return nil, mapHostAPINetworkRPCError(err)
	}
	return hostAPINetworkThreadSummaryPayload(thread), nil
}

func (h *HostAPIHandler) handleNetworkThreadMessages(ctx context.Context, raw json.RawMessage) (any, error) {
	var params extensioncontract.NetworkThreadMessagesParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	ref := store.NetworkConversationRef{
		WorkspaceID: strings.TrimSpace(params.WorkspaceID),
		Channel:     strings.TrimSpace(params.Channel),
		Surface:     store.NetworkSurfaceThread,
		ThreadID:    strings.TrimSpace(params.ThreadID),
	}
	workspaceID, err := h.hostAPINetworkWorkspaceID(ctx, params.WorkspaceID)
	if err != nil {
		return nil, err
	}
	ref.WorkspaceID = workspaceID
	query, err := hostAPINetworkConversationMessageQuery(
		params.Limit,
		params.Before,
		params.After,
		params.Kind,
		params.WorkID,
	)
	if err != nil {
		return nil, err
	}
	payload, page, err := h.hostAPINetworkConversationMessages(ctx, ref, query)
	if err != nil {
		return nil, err
	}
	return apicontract.NetworkThreadMessagesResponse{Messages: payload, Page: page}, nil
}

func (h *HostAPIHandler) handleNetworkDirectResolve(ctx context.Context, raw json.RawMessage) (any, error) {
	var params extensioncontract.NetworkDirectResolveParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	service, err := h.requireHostAPINetworkService()
	if err != nil {
		return nil, err
	}
	networkStore, err := h.requireHostAPINetworkStore()
	if err != nil {
		return nil, err
	}
	channel, err := hostAPINetworkChannel(params.Channel)
	if err != nil {
		return nil, err
	}
	sessionID := strings.TrimSpace(params.SessionID)
	if sessionID == "" {
		return nil, invalidParamsRPCError(errors.New("session_id is required"))
	}
	peerID := strings.TrimSpace(params.PeerID)
	if err := network.ValidatePeerID(peerID); err != nil {
		return nil, invalidParamsRPCError(err)
	}
	workspaceID, err := h.hostAPINetworkWorkspaceID(ctx, params.WorkspaceID)
	if err != nil {
		return nil, err
	}
	if err := h.requireHostAPINetworkParticipant(ctx, workspaceID, sessionID, channel); err != nil {
		return nil, err
	}
	local, remote, err := h.resolveHostAPIDirectPeers(ctx, service, workspaceID, channel, sessionID, peerID)
	if err != nil {
		return nil, mapHostAPINetworkRPCError(err)
	}
	directID, sessionA, sessionB, err := store.NetworkDirectRoomIdentity(
		workspaceID,
		channel,
		*local.SessionID,
		*remote.SessionID,
	)
	if err != nil {
		return nil, mapHostAPINetworkRPCError(err)
	}
	now := h.now()
	direct, err := networkStore.ResolveDirectRoom(ctx, store.NetworkDirectRoomEntry{
		WorkspaceID:    workspaceID,
		Channel:        channel,
		DirectID:       directID,
		SessionA:       sessionA,
		SessionB:       sessionB,
		OpenedAt:       now,
		LastActivityAt: now,
	})
	if err != nil {
		return nil, mapHostAPINetworkRPCError(err)
	}
	return hostAPINetworkDirectRoomPayload(direct), nil
}

func (h *HostAPIHandler) handleNetworkDirectMessages(ctx context.Context, raw json.RawMessage) (any, error) {
	var params extensioncontract.NetworkDirectMessagesParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	ref := store.NetworkConversationRef{
		WorkspaceID: strings.TrimSpace(params.WorkspaceID),
		Channel:     strings.TrimSpace(params.Channel),
		Surface:     store.NetworkSurfaceDirect,
		DirectID:    strings.TrimSpace(params.DirectID),
	}
	workspaceID, err := h.hostAPINetworkWorkspaceID(ctx, params.WorkspaceID)
	if err != nil {
		return nil, err
	}
	ref.WorkspaceID = workspaceID
	query, err := hostAPINetworkConversationMessageQuery(
		params.Limit,
		params.Before,
		params.After,
		params.Kind,
		params.WorkID,
	)
	if err != nil {
		return nil, err
	}
	payload, page, err := h.hostAPINetworkConversationMessages(ctx, ref, query)
	if err != nil {
		return nil, err
	}
	return apicontract.NetworkDirectRoomMessagesResponse{Messages: payload, Page: page}, nil
}

func (h *HostAPIHandler) handleNetworkWorkGet(ctx context.Context, raw json.RawMessage) (any, error) {
	var params extensioncontract.NetworkWorkGetParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	networkStore, err := h.requireHostAPINetworkStore()
	if err != nil {
		return nil, err
	}
	workID := strings.TrimSpace(params.WorkID)
	if err := network.ValidateWorkID(workID); err != nil {
		return nil, invalidParamsRPCError(err)
	}
	workspaceID, err := h.hostAPINetworkWorkspaceID(ctx, params.WorkspaceID)
	if err != nil {
		return nil, err
	}
	work, err := networkStore.GetWork(ctx, workspaceID, workID)
	if err != nil {
		return nil, mapHostAPINetworkRPCError(err)
	}
	return hostAPINetworkWorkPayload(work), nil
}

func (h *HostAPIHandler) handleNetworkSend(ctx context.Context, raw json.RawMessage) (any, error) {
	var params extensioncontract.NetworkSendParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	service, err := h.requireHostAPINetworkService()
	if err != nil {
		return nil, err
	}
	workspaceID, err := h.hostAPINetworkWorkspaceID(ctx, params.WorkspaceID)
	if err != nil {
		return nil, err
	}
	params.WorkspaceID = workspaceID
	sendReq, err := hostAPINetworkSendRequestFromPayload(params)
	if err != nil {
		return nil, mapHostAPINetworkRPCError(err)
	}
	if err := h.requireHostAPINetworkParticipant(
		ctx,
		workspaceID,
		params.SessionID,
		params.Channel,
	); err != nil {
		return nil, err
	}
	id, err := service.Send(ctx, sendReq)
	if err != nil {
		return nil, mapHostAPINetworkRPCError(err)
	}
	return hostAPINetworkSendPayloadFromRequest(id, params), nil
}

func (h *HostAPIHandler) requireHostAPINetworkService() (hostAPINetworkService, error) {
	if h == nil || h.network == nil {
		return nil, unavailableRPCError(errors.New("extension: network service is not configured"))
	}
	return h.network, nil
}

func (h *HostAPIHandler) requireHostAPINetworkStore() (store.NetworkConversationStore, error) {
	if h == nil || h.networkStore == nil {
		return nil, unavailableRPCError(errors.New("extension: network store is not configured"))
	}
	return h.networkStore, nil
}

func (h *HostAPIHandler) resolveHostAPIDirectPeers(
	ctx context.Context,
	service hostAPINetworkService,
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
	remote, remoteFound := hostAPINetworkFindPeer(peers, peerID)
	for _, peer := range peers {
		if !peer.Local || peer.SessionID == nil {
			continue
		}
		if strings.TrimSpace(*peer.SessionID) != sessionID {
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
	if remote.SessionID == nil || strings.TrimSpace(*remote.SessionID) == "" {
		return network.PeerInfo{}, network.PeerInfo{}, fmt.Errorf(
			"%w: peer_id=%q has no participating session",
			network.ErrTargetPeerNotFound,
			peerID,
		)
	}
	return local, remote, nil
}

func hostAPINetworkFindPeer(peers []network.PeerInfo, peerID string) (network.PeerInfo, bool) {
	wanted := strings.TrimSpace(peerID)
	for _, peer := range peers {
		if strings.TrimSpace(peer.PeerID) == wanted {
			return peer, true
		}
	}
	return network.PeerInfo{}, false
}

func (h *HostAPIHandler) hostAPINetworkWorkspaceID(ctx context.Context, raw string) (string, error) {
	return h.resolveRequiredWorkspaceID(ctx, raw)
}

func mapHostAPINetworkRPCError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, network.ErrLocalPeerNotFound),
		errors.Is(err, network.ErrTargetPeerNotFound),
		errors.Is(err, store.ErrNetworkConversationNotFound),
		errors.Is(err, sql.ErrNoRows):
		return notFoundRPCError("network", "", err)
	case errors.Is(err, store.ErrNetworkDirectRoomCollision),
		errors.Is(err, store.ErrNetworkWorkContainerMismatch),
		errors.Is(err, store.ErrNetworkWorkClosed):
		return hostAPIStatusRPCError(409, "Conflict", map[string]string{extensionStateError: err.Error()})
	case errors.Is(err, network.ErrMissingField),
		errors.Is(err, store.ErrNetworkCursorInvalid),
		errors.Is(err, network.ErrInvalidField),
		errors.Is(err, network.ErrInvalidKind),
		errors.Is(err, network.ErrInvalidBody),
		errors.Is(err, network.ErrEnvelopeTooLarge),
		errors.Is(err, network.ErrExpired),
		errors.Is(err, network.ErrReplayTooOld),
		errors.Is(err, network.ErrLegacyFieldRejected):
		return invalidParamsRPCError(err)
	default:
		return err
	}
}

func hostAPINetworkStatusPayload(status *network.Status) apicontract.NetworkStatusPayload {
	if status == nil {
		return apicontract.NetworkStatusPayload{}
	}
	kindMetrics := make([]apicontract.NetworkKindMetricPayload, 0, len(status.KindMetrics))
	for _, metric := range status.KindMetrics {
		kindMetrics = append(kindMetrics, apicontract.NetworkKindMetricPayload{
			Kind:      string(metric.Kind),
			Sent:      metric.Sent,
			Received:  metric.Received,
			Rejected:  metric.Rejected,
			Delivered: metric.Delivered,
		})
	}
	return apicontract.NetworkStatusPayload{
		Enabled:              status.Enabled,
		Status:               strings.TrimSpace(status.Status),
		LocalPeers:           status.LocalPeers,
		Channels:             status.Channels,
		MessagesSent:         status.MessagesSent,
		MessagesReceived:     status.MessagesReceived,
		MessagesRejected:     status.MessagesRejected,
		MessagesDelivered:    status.MessagesDelivered,
		WorkflowTaggedEvents: status.WorkflowTaggedEvents,
		HandoffTaggedEvents:  status.HandoffTaggedEvents,
		OpenThreads:          status.OpenThreads,
		OpenDirectRooms:      status.OpenDirectRooms,
		OpenWorkItems:        status.OpenWorkItems,
		ConversationMessages: status.ConversationMessages,
		WorkTransitions:      status.WorkTransitions,
		DirectResolves:       status.DirectResolves,
		KindMetrics:          kindMetrics,
	}
}

func hostAPINetworkChannelPayloads(channels []network.ChannelInfo) []apicontract.NetworkChannelPayload {
	payload := make([]apicontract.NetworkChannelPayload, 0, len(channels))
	for _, channel := range channels {
		payload = append(payload, apicontract.NetworkChannelPayload{
			WorkspaceID: strings.TrimSpace(channel.WorkspaceID),
			Channel:     strings.TrimSpace(channel.Channel),
			PeerCount:   channel.PeerCount,
		})
	}
	sort.Slice(payload, func(i int, j int) bool {
		return payload[i].Channel < payload[j].Channel
	})
	return payload
}

func hostAPINetworkPeerPayloads(peers []network.PeerInfo) []apicontract.NetworkPeerPayload {
	payload := make([]apicontract.NetworkPeerPayload, 0, len(peers))
	for _, peer := range peers {
		payload = append(payload, hostAPINetworkPeerPayload(peer))
	}
	sort.Slice(payload, func(i int, j int) bool {
		if payload[i].Local != payload[j].Local {
			return payload[i].Local
		}
		if payload[i].PeerID != payload[j].PeerID {
			return payload[i].PeerID < payload[j].PeerID
		}
		return payload[i].Channel < payload[j].Channel
	})
	return payload
}

func hostAPINetworkPeerPayload(peer network.PeerInfo) apicontract.NetworkPeerPayload {
	displayName := strings.TrimSpace(peer.PeerID)
	if peer.PeerCard.DisplayName != nil {
		if trimmed := strings.TrimSpace(*peer.PeerCard.DisplayName); trimmed != "" {
			displayName = trimmed
		}
	}
	return apicontract.NetworkPeerPayload{
		WorkspaceID: strings.TrimSpace(peer.WorkspaceID),
		SessionID:   hostAPICloneStringPtr(peer.SessionID),
		PeerID:      strings.TrimSpace(peer.PeerID),
		DisplayName: displayName,
		Channel:     strings.TrimSpace(peer.Channel),
		Local:       peer.Local,
		PeerCard: apicontract.NetworkPeerCardPayload{
			PeerID:              strings.TrimSpace(peer.PeerCard.PeerID),
			DisplayName:         hostAPICloneStringPtr(peer.PeerCard.DisplayName),
			ProfilesSupported:   append([]string(nil), peer.PeerCard.ProfilesSupported...),
			Capabilities:        hostAPINetworkCapabilityBriefPayloads(peer.PeerCard.Capabilities),
			ArtifactsSupported:  append([]string(nil), peer.PeerCard.ArtifactsSupported...),
			TrustModesSupported: append([]string(nil), peer.PeerCard.TrustModesSupported...),
			Ext:                 hostAPICloneRawMap(peer.PeerCard.Ext),
		},
		JoinedAt:      hostAPICloneTimePtr(peer.JoinedAt),
		PresenceState: apicontract.NetworkPresenceLocal,
	}
}

func hostAPINetworkCapabilityBriefPayloads(ids []string) []apicontract.NetworkCapabilityBriefPayload {
	payload := make([]apicontract.NetworkCapabilityBriefPayload, 0, len(ids))
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		payload = append(payload, apicontract.NetworkCapabilityBriefPayload{ID: trimmed})
	}
	if len(payload) == 0 {
		return []apicontract.NetworkCapabilityBriefPayload{}
	}
	return payload
}

func hostAPINetworkThreadSummaryPayloads(
	threads []store.NetworkThreadSummary,
) []apicontract.NetworkThreadSummaryPayload {
	payload := make([]apicontract.NetworkThreadSummaryPayload, 0, len(threads))
	for _, thread := range threads {
		payload = append(payload, hostAPINetworkThreadSummaryPayload(thread))
	}
	return payload
}

func hostAPINetworkThreadSummaryPayload(thread store.NetworkThreadSummary) apicontract.NetworkThreadSummaryPayload {
	return apicontract.NetworkThreadSummaryPayload{
		WorkspaceID:        strings.TrimSpace(thread.WorkspaceID),
		Channel:            strings.TrimSpace(thread.Channel),
		ThreadID:           strings.TrimSpace(thread.ThreadID),
		RootMessageID:      strings.TrimSpace(thread.RootMessageID),
		Title:              strings.TrimSpace(thread.Title),
		OpenedByPeerID:     strings.TrimSpace(thread.OpenedByPeerID),
		OpenedSessionID:    strings.TrimSpace(thread.OpenedSessionID),
		OpenedAt:           hostAPITimeValuePtr(thread.OpenedAt),
		LastActivityAt:     hostAPITimeValuePtr(thread.LastActivityAt),
		MessageCount:       thread.MessageCount,
		ParticipantCount:   thread.ParticipantCount,
		OpenWorkCount:      thread.OpenWorkCount,
		LastMessagePreview: strings.TrimSpace(thread.LastMessagePreview),
	}
}

func hostAPINetworkDirectRoomPayloads(
	directs []store.NetworkDirectRoomSummary,
) []apicontract.NetworkDirectRoomPayload {
	payload := make([]apicontract.NetworkDirectRoomPayload, 0, len(directs))
	for _, direct := range directs {
		payload = append(payload, hostAPINetworkDirectRoomPayload(direct))
	}
	return payload
}

func hostAPINetworkDirectRoomPayload(direct store.NetworkDirectRoomSummary) apicontract.NetworkDirectRoomPayload {
	return apicontract.NetworkDirectRoomPayload{
		WorkspaceID:        strings.TrimSpace(direct.WorkspaceID),
		Channel:            strings.TrimSpace(direct.Channel),
		DirectID:           strings.TrimSpace(direct.DirectID),
		SessionA:           strings.TrimSpace(direct.SessionA),
		SessionB:           strings.TrimSpace(direct.SessionB),
		OpenedAt:           hostAPITimeValuePtr(direct.OpenedAt),
		LastActivityAt:     hostAPITimeValuePtr(direct.LastActivityAt),
		MessageCount:       direct.MessageCount,
		OpenWorkCount:      direct.OpenWorkCount,
		LastMessagePreview: strings.TrimSpace(direct.LastMessagePreview),
	}
}

func hostAPINetworkConversationMessagePayloads(
	messages []store.NetworkConversationMessage,
) []apicontract.NetworkConversationMessagePayload {
	payload := make([]apicontract.NetworkConversationMessagePayload, 0, len(messages))
	for _, message := range messages {
		payload = append(payload, hostAPINetworkConversationMessagePayload(message))
	}
	return payload
}

func hostAPINetworkConversationMessagePayload(
	message store.NetworkConversationMessage,
) apicontract.NetworkConversationMessagePayload {
	return apicontract.NetworkConversationMessagePayload{
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
		PreviewText: hostAPINetworkMessagePreview(message),
		Body:        hostAPICloneRawMessage(message.Body),
		Timestamp:   message.Timestamp.UTC(),
	}
}

func hostAPINetworkWorkPayload(work store.NetworkWorkEntry) apicontract.NetworkWorkPayload {
	return apicontract.NetworkWorkPayload{
		WorkID:          strings.TrimSpace(work.WorkID),
		WorkspaceID:     strings.TrimSpace(work.WorkspaceID),
		Channel:         strings.TrimSpace(work.Channel),
		Surface:         strings.TrimSpace(work.Surface),
		ThreadID:        strings.TrimSpace(work.ThreadID),
		DirectID:        strings.TrimSpace(work.DirectID),
		OpenedSessionID: strings.TrimSpace(work.OpenedBySessionID),
		State:           strings.TrimSpace(work.State),
		OpenedAt:        hostAPITimeValuePtr(work.OpenedAt),
		LastActivityAt:  hostAPITimeValuePtr(work.LastActivityAt),
		TerminalAt:      hostAPICloneTimePtr(work.TerminalAt),
	}
}

func hostAPINetworkSendPayloadFromRequest(
	id string,
	req apicontract.NetworkSendRequest,
) apicontract.NetworkSendPayload {
	return apicontract.NetworkSendPayload{
		ID:          strings.TrimSpace(id),
		WorkspaceID: strings.TrimSpace(req.WorkspaceID),
		SessionID:   strings.TrimSpace(req.SessionID),
		Channel:     strings.TrimSpace(req.Channel),
		Surface:     strings.TrimSpace(req.Surface),
		ThreadID:    strings.TrimSpace(req.ThreadID),
		DirectID:    strings.TrimSpace(req.DirectID),
		Kind:        strings.TrimSpace(req.Kind),
		To:          strings.TrimSpace(req.To),
		Mentions:    hostAPICloneTrimmedStrings(req.Mentions),
		WorkID:      strings.TrimSpace(req.WorkID),
		ReplyTo:     strings.TrimSpace(req.ReplyTo),
		TraceID:     strings.TrimSpace(req.TraceID),
		CausationID: strings.TrimSpace(req.CausationID),
		ExpiresAt:   hostAPICloneInt64Ptr(req.ExpiresAt),
		Ext:         hostAPICloneRawMap(req.Ext),
	}
}

func hostAPINetworkMessagePreview(message store.NetworkConversationMessage) string {
	if preview := strings.TrimSpace(message.PreviewText); preview != "" {
		return preview
	}
	if text := strings.TrimSpace(message.Text); text != "" {
		return text
	}
	return strings.TrimSpace(string(message.Body))
}
