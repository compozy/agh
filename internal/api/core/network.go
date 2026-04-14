package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/session"
)

func (h *BaseHandlers) networkServiceRequired() (NetworkService, error) {
	if !h.Config.Network.Enabled {
		return nil, errors.New("api: network is disabled")
	}
	if h.Network == nil {
		return nil, errors.New("api: network service is required when network is enabled")
	}
	return h.Network, nil
}

// NetworkStatus returns the current daemon-owned network runtime status.
func (h *BaseHandlers) NetworkStatus(c *gin.Context) {
	payload, err := h.networkStatusPayload(c.Request.Context())
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, contract.NetworkStatusResponse{Network: *payload})
}

// NetworkPeers returns the current visible peers, optionally filtered by channel.
func (h *BaseHandlers) NetworkPeers(c *gin.Context) {
	service, err := h.networkServiceRequired()
	if err != nil {
		h.respondError(c, http.StatusServiceUnavailable, err)
		return
	}

	peers, err := service.ListPeers(c.Request.Context(), strings.TrimSpace(c.Query("channel")))
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	sessionByID := h.networkPeerSessionInfoMap(c.Request.Context(), peers)
	payload := make([]contract.NetworkPeerPayload, 0, len(peers))
	for _, peer := range peers {
		payload = append(payload, networkPeerPayloadFromInfoWithSessions(peer, sessionByID))
	}
	c.JSON(http.StatusOK, contract.NetworkPeersResponse{Peers: payload})
}

func (h *BaseHandlers) networkPeerSessionInfoMap(
	ctx context.Context,
	peers []network.PeerInfo,
) map[string]*session.SessionInfo {
	if h == nil || h.Sessions == nil || len(peers) == 0 {
		return nil
	}

	sessionByID := make(map[string]*session.SessionInfo, len(peers))
	for _, peer := range peers {
		if peer.SessionID == nil {
			continue
		}

		sessionID := strings.TrimSpace(*peer.SessionID)
		if sessionID == "" {
			continue
		}
		if _, seen := sessionByID[sessionID]; seen {
			continue
		}

		info, err := h.Sessions.Status(ctx, sessionID)
		if err != nil {
			if h.Logger != nil {
				h.Logger.Warn(
					h.transportName()+": skip network peer session enrichment",
					"session_id",
					sessionID,
					"peer_id",
					strings.TrimSpace(peer.PeerID),
					"error",
					err,
				)
			}
			continue
		}
		if info != nil {
			sessionByID[sessionID] = info
		}
	}
	if len(sessionByID) == 0 {
		return nil
	}
	return sessionByID
}

// NetworkChannels returns the active runtime channels.
func (h *BaseHandlers) NetworkChannels(c *gin.Context) {
	service, err := h.networkServiceRequired()
	if err != nil {
		h.respondError(c, http.StatusServiceUnavailable, err)
		return
	}

	channels, err := h.networkChannelPayloads(c.Request.Context(), service)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrNetworkValidation) ||
			errors.Is(err, network.ErrLocalPeerNotFound) ||
			errors.Is(err, network.ErrTargetPeerNotFound) ||
			errors.Is(err, network.ErrMissingField) ||
			errors.Is(err, network.ErrInvalidField) ||
			errors.Is(err, network.ErrInvalidKind) ||
			errors.Is(err, network.ErrInvalidBody) ||
			errors.Is(err, network.ErrExpired) ||
			errors.Is(err, network.ErrReplayTooOld) {
			status = StatusForNetworkError(err)
		}
		h.respondError(c, status, err)
		return
	}
	c.JSON(http.StatusOK, contract.NetworkChannelsResponse{Channels: channels})
}

// NetworkSend validates and forwards one outbound network send request.
func (h *BaseHandlers) NetworkSend(c *gin.Context) {
	service, err := h.networkServiceRequired()
	if err != nil {
		h.respondError(c, http.StatusServiceUnavailable, err)
		return
	}

	var req contract.NetworkSendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, fmt.Errorf("%s: decode network send request: %w", h.transportName(), err))
		return
	}

	sendReq, err := NetworkSendRequestFromPayload(req)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}

	id, err := service.Send(c.Request.Context(), sendReq)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.NetworkSendResponse{Message: NetworkSendPayloadFromRequest(id, req)})
}

// NetworkInbox returns the queued inbound envelopes for one local session.
func (h *BaseHandlers) NetworkInbox(c *gin.Context) {
	service, err := h.networkServiceRequired()
	if err != nil {
		h.respondError(c, http.StatusServiceUnavailable, err)
		return
	}

	sessionID := strings.TrimSpace(c.Query("session_id"))
	if sessionID == "" {
		sessionID = strings.TrimSpace(c.Query("session"))
	}
	if sessionID == "" {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(errors.New("session_id query is required")))
		return
	}

	messages, err := service.Inbox(c.Request.Context(), sessionID)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.NetworkInboxResponse{Messages: NetworkEnvelopePayloadsFromEnvelopes(messages)})
}

// NetworkStatusPayloadFromStatus converts the runtime network status snapshot into the shared payload.
func NetworkStatusPayloadFromStatus(status *network.NetworkStatus) *contract.NetworkStatusPayload {
	if status == nil {
		return nil
	}

	kindMetrics := make([]contract.NetworkKindMetricPayload, 0, len(status.KindMetrics))
	for _, metric := range status.KindMetrics {
		kindMetrics = append(kindMetrics, contract.NetworkKindMetricPayload{
			Kind:      string(metric.Kind),
			Sent:      metric.Sent,
			Received:  metric.Received,
			Rejected:  metric.Rejected,
			Delivered: metric.Delivered,
		})
	}

	return &contract.NetworkStatusPayload{
		Enabled:              status.Enabled,
		Status:               strings.TrimSpace(status.Status),
		ListenerHost:         strings.TrimSpace(status.ListenerHost),
		ListenerPort:         status.ListenerPort,
		LocalPeers:           status.LocalPeers,
		RemotePeers:          status.RemotePeers,
		Channels:             status.Channels,
		QueuedMessages:       status.QueuedMessages,
		QueuedSessions:       status.QueuedSessions,
		DeliveryWorkers:      status.DeliveryWorkers,
		MessagesSent:         status.MessagesSent,
		MessagesReceived:     status.MessagesReceived,
		MessagesRejected:     status.MessagesRejected,
		MessagesDelivered:    status.MessagesDelivered,
		WorkflowTaggedEvents: status.WorkflowTaggedEvents,
		HandoffTaggedEvents:  status.HandoffTaggedEvents,
		LastDisconnect:       strings.TrimSpace(status.LastDisconnect),
		KindMetrics:          kindMetrics,
	}
}

// NetworkSendRequestFromPayload validates and converts one shared send payload into the runtime request.
func NetworkSendRequestFromPayload(req contract.NetworkSendRequest) (network.SendRequest, error) {
	if strings.TrimSpace(req.SessionID) == "" {
		return network.SendRequest{}, NewNetworkValidationError(errors.New("session_id is required"))
	}
	if strings.TrimSpace(req.Channel) == "" {
		return network.SendRequest{}, NewNetworkValidationError(errors.New("channel is required"))
	}
	if strings.TrimSpace(req.Kind) == "" {
		return network.SendRequest{}, NewNetworkValidationError(errors.New("kind is required"))
	}
	if len(bytes.TrimSpace(req.Body)) == 0 {
		return network.SendRequest{}, NewNetworkValidationError(errors.New("body is required"))
	}
	if !json.Valid(req.Body) {
		return network.SendRequest{}, NewNetworkValidationError(errors.New("body must be valid JSON"))
	}

	sendReq := network.SendRequest{
		SessionID: strings.TrimSpace(req.SessionID),
		Channel:   strings.TrimSpace(req.Channel),
		Kind:      network.Kind(strings.TrimSpace(req.Kind)),
		Body:      cloneRawMessage(req.Body),
		ExpiresAt: cloneInt64Ptr(req.ExpiresAt),
		Ext:       cloneRawMap(req.Ext),
	}
	if to := strings.TrimSpace(req.To); to != "" {
		sendReq.To = ptrString(to)
	}
	if interactionID := strings.TrimSpace(req.InteractionID); interactionID != "" {
		sendReq.InteractionID = ptrString(interactionID)
	}
	if replyTo := strings.TrimSpace(req.ReplyTo); replyTo != "" {
		sendReq.ReplyTo = ptrString(replyTo)
	}
	if traceID := strings.TrimSpace(req.TraceID); traceID != "" {
		sendReq.TraceID = ptrString(traceID)
	}
	if causationID := strings.TrimSpace(req.CausationID); causationID != "" {
		sendReq.CausationID = ptrString(causationID)
	}
	if id := strings.TrimSpace(req.ID); id != "" {
		sendReq.ID = ptrString(id)
	}

	return sendReq, nil
}

// NetworkSendPayloadFromRequest builds the shared send response payload from the original request plus the assigned message id.
func NetworkSendPayloadFromRequest(id string, req contract.NetworkSendRequest) contract.NetworkSendPayload {
	return contract.NetworkSendPayload{
		ID:            strings.TrimSpace(id),
		SessionID:     strings.TrimSpace(req.SessionID),
		Channel:       strings.TrimSpace(req.Channel),
		Kind:          strings.TrimSpace(req.Kind),
		To:            strings.TrimSpace(req.To),
		InteractionID: strings.TrimSpace(req.InteractionID),
		ReplyTo:       strings.TrimSpace(req.ReplyTo),
		TraceID:       strings.TrimSpace(req.TraceID),
		CausationID:   strings.TrimSpace(req.CausationID),
		ExpiresAt:     cloneInt64Ptr(req.ExpiresAt),
		Ext:           cloneRawMap(req.Ext),
	}
}

// NetworkPeerPayloadsFromInfos converts the visible peer snapshot into shared payloads.
func NetworkPeerPayloadsFromInfos(peers []network.PeerInfo) []contract.NetworkPeerPayload {
	payload := make([]contract.NetworkPeerPayload, 0, len(peers))
	for _, peer := range peers {
		payload = append(payload, NetworkPeerPayloadFromInfo(peer))
	}
	return payload
}

// NetworkPeerPayloadFromInfo converts one visible peer snapshot into the shared payload.
func NetworkPeerPayloadFromInfo(peer network.PeerInfo) contract.NetworkPeerPayload {
	displayName := peer.PeerID
	if peer.PeerCard.DisplayName != nil {
		displayName = strings.TrimSpace(*peer.PeerCard.DisplayName)
	}
	return contract.NetworkPeerPayload{
		SessionID:   peer.SessionID,
		PeerID:      peer.PeerID,
		DisplayName: displayName,
		Channel:     peer.Channel,
		Local:       peer.Local,
		PeerCard: contract.NetworkPeerCardPayload{
			PeerID:              peer.PeerCard.PeerID,
			DisplayName:         peer.PeerCard.DisplayName,
			ProfilesSupported:   append([]string(nil), peer.PeerCard.ProfilesSupported...),
			Capabilities:        append([]string(nil), peer.PeerCard.Capabilities...),
			ArtifactsSupported:  append([]string(nil), peer.PeerCard.ArtifactsSupported...),
			TrustModesSupported: append([]string(nil), peer.PeerCard.TrustModesSupported...),
			Ext:                 cloneRawMap(peer.PeerCard.Ext),
		},
		JoinedAt:  cloneTimePtr(peer.JoinedAt),
		LastSeen:  cloneTimePtr(peer.LastSeen),
		ExpiresAt: cloneTimePtr(peer.ExpiresAt),
	}
}

// NetworkChannelPayloadsFromInfos converts active channel summaries into shared payloads.
func NetworkChannelPayloadsFromInfos(channels []network.ChannelInfo) []contract.NetworkChannelPayload {
	payload := make([]contract.NetworkChannelPayload, 0, len(channels))
	for _, channel := range channels {
		payload = append(payload, contract.NetworkChannelPayload{
			Channel:   channel.Channel,
			PeerCount: channel.PeerCount,
		})
	}
	return payload
}

// NetworkEnvelopePayloadsFromEnvelopes converts surfaced envelopes into shared payloads.
func NetworkEnvelopePayloadsFromEnvelopes(envelopes []network.Envelope) []contract.NetworkEnvelopePayload {
	payload := make([]contract.NetworkEnvelopePayload, 0, len(envelopes))
	for _, envelope := range envelopes {
		payload = append(payload, NetworkEnvelopePayloadFromEnvelope(envelope))
	}
	return payload
}

// NetworkEnvelopePayloadFromEnvelope converts one surfaced envelope into the shared payload.
func NetworkEnvelopePayloadFromEnvelope(envelope network.Envelope) contract.NetworkEnvelopePayload {
	return contract.NetworkEnvelopePayload{
		Protocol:      envelope.Protocol,
		ID:            envelope.ID,
		Kind:          string(envelope.Kind),
		Channel:       envelope.Channel,
		From:          envelope.From,
		To:            cloneStringPtr(envelope.To),
		InteractionID: cloneStringPtr(envelope.InteractionID),
		ReplyTo:       cloneStringPtr(envelope.ReplyTo),
		TraceID:       cloneStringPtr(envelope.TraceID),
		CausationID:   cloneStringPtr(envelope.CausationID),
		TS:            envelope.TS,
		ExpiresAt:     cloneInt64Ptr(envelope.ExpiresAt),
		Body:          cloneRawMessage(envelope.Body),
		Proof:         cloneProofPtr(envelope.Proof),
		Ext:           cloneRawMap(envelope.Ext),
	}
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	copyValue := strings.TrimSpace(*value)
	return &copyValue
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := value.UTC()
	return &copyValue
}

func ptrString(value string) *string {
	copyValue := strings.TrimSpace(value)
	return &copyValue
}

func cloneInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func cloneRawMessage(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}

func cloneRawMap[T ~map[string]json.RawMessage](source T) map[string]json.RawMessage {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]json.RawMessage, len(source))
	for key, value := range source {
		cloned[key] = cloneRawMessage(value)
	}
	return cloned
}

func cloneProofPtr(source *network.Proof) map[string]json.RawMessage {
	if source == nil {
		return nil
	}
	return cloneRawMap(*source)
}
