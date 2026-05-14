package core

import (
	"bytes"
	"context"
	"encoding/json"
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
	toolspkg "github.com/pedronauck/agh/internal/tools"
)

const (
	apiCapabilityBriefExtKey   = "agh.capabilities_brief"
	apiCapabilityCatalogExtKey = "agh.capability_catalog"
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
	scope, ok := h.resolveWorkspaceScope(c)
	if !ok {
		return
	}

	peers, err := service.ListPeers(
		c.Request.Context(),
		scope.NetworkWorkspaceID(),
		strings.TrimSpace(c.Query("channel")),
	)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	sessionByID := h.networkPeerSessionInfoMap(c.Request.Context(), peers)
	payload := make([]contract.NetworkPeerPayload, 0, len(peers))
	for _, peer := range peers {
		payload = append(payload, networkPeerPayloadFromInfoWithSessions(peer, sessionByID))
	}
	sortNetworkPeerPayloads(payload)
	c.JSON(http.StatusOK, contract.NetworkPeersResponse{Peers: payload})
}

func (h *BaseHandlers) networkPeerSessionInfoMap(
	ctx context.Context,
	peers []network.PeerInfo,
) map[string]*session.Info {
	if h == nil || h.Sessions == nil || len(peers) == 0 {
		return nil
	}

	wanted := make(map[string]struct{}, len(peers))
	for _, peer := range peers {
		if peer.SessionID == nil {
			continue
		}

		sessionID := strings.TrimSpace(*peer.SessionID)
		if sessionID == "" {
			continue
		}
		wanted[sessionID] = struct{}{}
	}
	if len(wanted) == 0 {
		return nil
	}

	infos, err := h.Sessions.ListAll(ctx)
	if err != nil {
		if h.Logger != nil {
			h.Logger.Warn(
				h.transportName()+": skip network peer session enrichment",
				"error",
				err,
			)
		}
		return nil
	}

	sessionByID := make(map[string]*session.Info, len(wanted))
	for _, info := range infos {
		if info == nil {
			continue
		}
		sessionID := strings.TrimSpace(info.ID)
		if _, ok := wanted[sessionID]; ok {
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
	scope, ok := h.resolveWorkspaceScope(c)
	if !ok {
		return
	}

	channels, err := h.networkChannelPayloads(c.Request.Context(), service, scope.NetworkWorkspaceID())
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
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
	scope, ok := h.resolveWorkspaceScope(c)
	if !ok {
		return
	}

	var req contract.NetworkSendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode network send request: %w", h.transportName(), err),
		)
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
	req.WorkspaceID = scope.NetworkWorkspaceID()
	if strings.TrimSpace(req.SessionID) == "" {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(errors.New("session_id is required")))
		return
	}
	if _, err := h.requireSessionInWorkspace(
		c.Request.Context(),
		scope.SessionWorkspaceID(),
		req.SessionID,
	); err != nil {
		h.respondError(c, statusForWorkspaceScopedResourceError(err), err)
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
	scope, ok := h.resolveWorkspaceScope(c)
	if !ok {
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
	if _, err := h.requireSessionInWorkspace(c.Request.Context(), scope.SessionWorkspaceID(), sessionID); err != nil {
		h.respondError(c, statusForWorkspaceScopedResourceError(err), err)
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
func NetworkStatusPayloadFromStatus(status *network.Status) *contract.NetworkStatusPayload {
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
		OpenThreads:          status.OpenThreads,
		OpenDirectRooms:      status.OpenDirectRooms,
		OpenWorkItems:        status.OpenWorkItems,
		ConversationMessages: status.ConversationMessages,
		WorkTransitions:      status.WorkTransitions,
		DirectResolves:       status.DirectResolves,
		LastDisconnect:       strings.TrimSpace(status.LastDisconnect),
		KindMetrics:          kindMetrics,
	}
}

// NetworkSendRequestFromPayload validates and converts one shared send payload into the runtime request.
func NetworkSendRequestFromPayload(req contract.NetworkSendRequest) (network.SendRequest, error) {
	if strings.TrimSpace(req.SessionID) == "" {
		return network.SendRequest{}, NewNetworkValidationError(errors.New("session_id is required"))
	}
	if strings.TrimSpace(req.WorkspaceID) == "" {
		return network.SendRequest{}, NewNetworkValidationError(errors.New("workspace_id is required"))
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
	if err := validateNetworkSendNoRawClaimToken(req); err != nil {
		return network.SendRequest{}, err
	}
	if err := validateNetworkSendConversation(req); err != nil {
		return network.SendRequest{}, err
	}

	sendReq := network.SendRequest{
		SessionID:   strings.TrimSpace(req.SessionID),
		WorkspaceID: strings.TrimSpace(req.WorkspaceID),
		Channel:     strings.TrimSpace(req.Channel),
		Kind:        network.Kind(strings.TrimSpace(req.Kind)),
		Body:        cloneRawMessage(req.Body),
		ExpiresAt:   cloneInt64Ptr(req.ExpiresAt),
		Ext:         cloneRawMap(req.Ext),
	}
	if to := strings.TrimSpace(req.To); to != "" {
		sendReq.To = ptrString(to)
	}
	if surface := strings.TrimSpace(req.Surface); surface != "" {
		networkSurface := network.Surface(surface)
		sendReq.Surface = &networkSurface
	}
	if threadID := strings.TrimSpace(req.ThreadID); threadID != "" {
		sendReq.ThreadID = ptrString(threadID)
	}
	if directID := strings.TrimSpace(req.DirectID); directID != "" {
		sendReq.DirectID = ptrString(directID)
	}
	if workID := strings.TrimSpace(req.WorkID); workID != "" {
		sendReq.WorkID = ptrString(workID)
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

func validateNetworkSendConversation(req contract.NetworkSendRequest) error {
	kind := network.Kind(strings.TrimSpace(req.Kind))
	if err := kind.Validate(); err != nil {
		return NewNetworkValidationError(err)
	}

	surface := strings.TrimSpace(req.Surface)
	threadID := strings.TrimSpace(req.ThreadID)
	directID := strings.TrimSpace(req.DirectID)
	workID := strings.TrimSpace(req.WorkID)
	if kind == network.KindGreet || kind == network.KindWhois {
		if surface != "" || threadID != "" || directID != "" || workID != "" {
			return NewNetworkValidationError(fmt.Errorf(
				"%w: %s cannot carry conversation or work fields",
				network.ErrInvalidField,
				kind,
			))
		}
		return nil
	}

	if surface == "" {
		return NewNetworkValidationError(fmt.Errorf("%w: surface is required", network.ErrMissingField))
	}
	ref := network.ConversationRef{
		WorkspaceID: strings.TrimSpace(req.WorkspaceID),
		Channel:     strings.TrimSpace(req.Channel),
		Surface:     network.Surface(surface),
		ThreadID:    threadID,
		DirectID:    directID,
	}
	if err := ref.Validate(); err != nil {
		return NewNetworkValidationError(err)
	}
	if workID != "" {
		if err := network.ValidateWorkID(workID); err != nil {
			return NewNetworkValidationError(err)
		}
	}
	if kind == network.KindCapability || kind == network.KindReceipt || kind == network.KindTrace {
		if workID == "" {
			return NewNetworkValidationError(fmt.Errorf("%w: work_id is required", network.ErrMissingField))
		}
	}
	return nil
}

// validateNetworkSendNoRawClaimToken keeps raw claim_token fields out of client-controlled network payloads.
func validateNetworkSendNoRawClaimToken(req contract.NetworkSendRequest) error {
	payload := struct {
		Body json.RawMessage            `json:"body"`
		Ext  map[string]json.RawMessage `json:"ext,omitempty"`
	}{
		Body: req.Body,
		Ext:  req.Ext,
	}
	if err := contract.ValidateNoRawClaimTokenField(payload); err != nil {
		return NewNetworkValidationError(fmt.Errorf(
			"%s: raw claim_token fields are forbidden: %w",
			toolspkg.ReasonNetworkRawTokenRejected,
			err,
		))
	}
	return nil
}

// NetworkSendPayloadFromRequest builds the shared send response payload
// from the original request plus the assigned message id.
func NetworkSendPayloadFromRequest(id string, req contract.NetworkSendRequest) contract.NetworkSendPayload {
	return contract.NetworkSendPayload{
		ID:          strings.TrimSpace(id),
		WorkspaceID: strings.TrimSpace(req.WorkspaceID),
		SessionID:   strings.TrimSpace(req.SessionID),
		Channel:     strings.TrimSpace(req.Channel),
		Surface:     strings.TrimSpace(req.Surface),
		ThreadID:    strings.TrimSpace(req.ThreadID),
		DirectID:    strings.TrimSpace(req.DirectID),
		Kind:        strings.TrimSpace(req.Kind),
		To:          strings.TrimSpace(req.To),
		WorkID:      strings.TrimSpace(req.WorkID),
		ReplyTo:     strings.TrimSpace(req.ReplyTo),
		TraceID:     strings.TrimSpace(req.TraceID),
		CausationID: strings.TrimSpace(req.CausationID),
		ExpiresAt:   cloneInt64Ptr(req.ExpiresAt),
		Ext:         cloneRawMap(req.Ext),
	}
}

// NetworkPeerPayloadsFromInfos converts the visible peer snapshot into shared payloads.
func NetworkPeerPayloadsFromInfos(peers []network.PeerInfo) []contract.NetworkPeerPayload {
	payload := make([]contract.NetworkPeerPayload, 0, len(peers))
	for _, peer := range peers {
		payload = append(payload, NetworkPeerPayloadFromInfo(peer))
	}
	sortNetworkPeerPayloads(payload)
	return payload
}

func sortNetworkPeerPayloads(peers []contract.NetworkPeerPayload) {
	sort.Slice(peers, func(i int, j int) bool {
		if peers[i].Local != peers[j].Local {
			return peers[i].Local
		}

		left := networkPeerSortTimestamp(peers[i])
		right := networkPeerSortTimestamp(peers[j])
		switch {
		case left != nil && right != nil && !left.Equal(*right):
			return left.After(*right)
		case left != nil && right == nil:
			return true
		case left == nil && right != nil:
			return false
		}

		leftName := networkPeerSortName(peers[i])
		rightName := networkPeerSortName(peers[j])
		if leftName != rightName {
			return leftName < rightName
		}
		if peers[i].PeerID != peers[j].PeerID {
			return peers[i].PeerID < peers[j].PeerID
		}
		return peers[i].Channel < peers[j].Channel
	})
}

func networkPeerSortTimestamp(peer contract.NetworkPeerPayload) *time.Time {
	if peer.LastSeen != nil {
		return peer.LastSeen
	}
	return peer.JoinedAt
}

func networkPeerSortName(peer contract.NetworkPeerPayload) string {
	if value := strings.TrimSpace(peer.DisplayName); value != "" {
		return value
	}
	return strings.TrimSpace(peer.PeerID)
}

// NetworkPeerPayloadFromInfo converts one visible peer snapshot into the shared payload.
func NetworkPeerPayloadFromInfo(peer network.PeerInfo) contract.NetworkPeerPayload {
	displayName := peer.PeerID
	if peer.PeerCard.DisplayName != nil {
		if trimmed := strings.TrimSpace(*peer.PeerCard.DisplayName); trimmed != "" {
			displayName = trimmed
		}
	}
	return contract.NetworkPeerPayload{
		WorkspaceID: strings.TrimSpace(peer.WorkspaceID),
		SessionID:   peer.SessionID,
		PeerID:      peer.PeerID,
		DisplayName: displayName,
		Channel:     peer.Channel,
		Local:       peer.Local,
		PeerCard:    networkPeerCardPayload(peer),
		JoinedAt:    cloneTimePtr(peer.JoinedAt),
		LastSeen:    cloneTimePtr(peer.LastSeen),
		ExpiresAt:   cloneTimePtr(peer.ExpiresAt),
	}
}

func networkPeerCardPayload(peer network.PeerInfo) contract.NetworkPeerCardPayload {
	capabilityCatalog := peer.CapabilityCatalog
	if !peer.CapabilityCatalogKnown {
		capabilityCatalog = nil
	}

	return contract.NetworkPeerCardPayload{
		PeerID:              peer.PeerCard.PeerID,
		DisplayName:         peer.PeerCard.DisplayName,
		ProfilesSupported:   cloneNetworkStringSliceOrEmpty(peer.PeerCard.ProfilesSupported),
		Capabilities:        networkCapabilityBriefPayloads(peer.PeerCard, capabilityCatalog),
		ArtifactsSupported:  cloneNetworkStringSliceOrEmpty(peer.PeerCard.ArtifactsSupported),
		TrustModesSupported: cloneNetworkStringSliceOrEmpty(peer.PeerCard.TrustModesSupported),
		Ext:                 clonePeerCardExtWithoutCapabilityDiscovery(peer.PeerCard.Ext),
	}
}

func cloneNetworkStringSliceOrEmpty(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}

func networkCapabilityBriefPayloads(
	card network.PeerCard,
	capabilityCatalog []session.NetworkPeerCapability,
) []contract.NetworkCapabilityBriefPayload {
	summaries := decodeCapabilityBriefSummaries(card.Ext)
	if len(capabilityCatalog) > 0 {
		orderedIDs := card.Capabilities
		if len(orderedIDs) == 0 {
			orderedIDs = make([]string, 0, len(capabilityCatalog))
			for _, capability := range capabilityCatalog {
				orderedIDs = append(orderedIDs, capability.ID)
			}
		}

		if summaries == nil {
			summaries = make(map[string]string, len(capabilityCatalog))
		}
		for _, capability := range capabilityCatalog {
			id := strings.TrimSpace(capability.ID)
			if id == "" {
				continue
			}
			summaries[id] = strings.TrimSpace(capability.Summary)
		}
		return capabilityBriefPayloadsFromIDs(orderedIDs, summaries)
	}

	return capabilityBriefPayloadsFromIDs(card.Capabilities, summaries)
}

func decodeCapabilityBriefSummaries(ext network.ExtensionMap) map[string]string {
	if len(ext) == 0 {
		return nil
	}

	raw, ok := ext[apiCapabilityBriefExtKey]
	if !ok || len(raw) == 0 {
		return nil
	}

	var payload []contract.NetworkCapabilityBriefPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}

	summaries := make(map[string]string, len(payload))
	for _, capability := range payload {
		id := strings.TrimSpace(capability.ID)
		if id == "" {
			continue
		}
		summaries[id] = strings.TrimSpace(capability.Summary)
	}
	if len(summaries) == 0 {
		return nil
	}
	return summaries
}

func capabilityBriefPayloadsFromIDs(
	capabilityIDs []string,
	summaries map[string]string,
) []contract.NetworkCapabilityBriefPayload {
	brief := make([]contract.NetworkCapabilityBriefPayload, 0, len(capabilityIDs))
	for _, capabilityID := range capabilityIDs {
		id := strings.TrimSpace(capabilityID)
		if id == "" {
			continue
		}

		brief = append(brief, contract.NetworkCapabilityBriefPayload{
			ID:      id,
			Summary: strings.TrimSpace(summaries[id]),
		})
	}
	if len(brief) == 0 {
		return []contract.NetworkCapabilityBriefPayload{}
	}
	return brief
}

func clonePeerCardExtWithoutCapabilityDiscovery(
	ext network.ExtensionMap,
) map[string]json.RawMessage {
	cloned := cloneRawMap(ext)
	if len(cloned) == 0 {
		return nil
	}

	delete(cloned, apiCapabilityBriefExtKey)
	delete(cloned, apiCapabilityCatalogExtKey)
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}

func networkCapabilityCatalogPayload(
	peer network.PeerInfo,
) *contract.NetworkCapabilityCatalogPayload {
	if !peer.CapabilityCatalogKnown {
		return nil
	}

	payload := &contract.NetworkCapabilityCatalogPayload{
		Capabilities: make([]contract.NetworkCapabilityPayload, 0, len(peer.CapabilityCatalog)),
	}
	for _, capability := range peer.CapabilityCatalog {
		id := strings.TrimSpace(capability.ID)
		if id == "" {
			continue
		}

		payload.Capabilities = append(payload.Capabilities, contract.NetworkCapabilityPayload{
			ID:                id,
			Summary:           strings.TrimSpace(capability.Summary),
			Outcome:           strings.TrimSpace(capability.Outcome),
			Version:           strings.TrimSpace(capability.Version),
			Digest:            strings.TrimSpace(capability.Digest),
			ContextNeeded:     append([]string(nil), capability.ContextNeeded...),
			ArtifactsExpected: append([]string(nil), capability.ArtifactsExpected...),
			ExecutionOutline:  append([]string(nil), capability.ExecutionOutline...),
			Constraints:       append([]string(nil), capability.Constraints...),
			Examples:          append([]string(nil), capability.Examples...),
			Requirements:      append([]string(nil), capability.Requirements...),
		})
	}
	return payload
}

// NetworkChannelPayloadsFromInfos converts active channel summaries into shared payloads.
func NetworkChannelPayloadsFromInfos(channels []network.ChannelInfo) []contract.NetworkChannelPayload {
	payload := make([]contract.NetworkChannelPayload, 0, len(channels))
	for _, channel := range channels {
		payload = append(payload, contract.NetworkChannelPayload{
			WorkspaceID: strings.TrimSpace(channel.WorkspaceID),
			Channel:     channel.Channel,
			PeerCount:   channel.PeerCount,
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
		Protocol:    envelope.Protocol,
		ID:          envelope.ID,
		Kind:        string(envelope.Kind),
		WorkspaceID: strings.TrimSpace(envelope.WorkspaceID),
		Channel:     envelope.Channel,
		Surface:     cloneSurfacePtr(envelope.Surface),
		ThreadID:    cloneStringPtr(envelope.ThreadID),
		DirectID:    cloneStringPtr(envelope.DirectID),
		From:        envelope.From,
		To:          cloneStringPtr(envelope.To),
		WorkID:      cloneStringPtr(envelope.WorkID),
		ReplyTo:     cloneStringPtr(envelope.ReplyTo),
		TraceID:     cloneStringPtr(envelope.TraceID),
		CausationID: cloneStringPtr(envelope.CausationID),
		TS:          envelope.TS,
		ExpiresAt:   cloneInt64Ptr(envelope.ExpiresAt),
		Body:        cloneRawMessage(envelope.Body),
		Proof:       cloneProofPtr(envelope.Proof),
		Ext:         cloneRawMap(envelope.Ext),
	}
}

func cloneSurfacePtr(value *network.Surface) *string {
	if value == nil {
		return nil
	}
	copyValue := strings.TrimSpace(string(*value))
	return &copyValue
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
