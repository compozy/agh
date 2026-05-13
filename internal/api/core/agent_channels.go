package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/agentidentity"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
)

const (
	agentActionContext      = "agent.context"
	agentActionChannelList  = "agent.ch.list"
	agentActionChannelRecv  = "agent.ch.recv"
	agentActionChannelSend  = "agent.ch.send"
	agentActionChannelReply = "agent.ch.reply"

	agentCoordinationExtKey = "coordination"
	agentChannelThreadID    = "thread_agent_channel"
)

// AgentContext returns the bounded situation payload for the validated caller session.
func (h *BaseHandlers) AgentContext(c *gin.Context) {
	caller, ok := h.requireAgentCaller(c, agentActionContext)
	if !ok {
		return
	}
	if h.AgentContextService == nil {
		h.respondError(c, http.StatusServiceUnavailable, errors.New("api: agent context service is not configured"))
		return
	}

	payload, err := h.AgentContextService.ContextForSession(c.Request.Context(), sessionInfoFromAgentCaller(caller))
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, contract.AgentContextResponse{
		Context: contract.NormalizeAgentContextPayload(&payload),
	})
}

// AgentChannels lists discoverable coordination channels for the validated caller.
func (h *BaseHandlers) AgentChannels(c *gin.Context) {
	caller, ok := h.requireAgentCaller(c, agentActionChannelList)
	if !ok {
		return
	}
	service, err := h.networkServiceRequired()
	if err != nil {
		h.respondError(c, http.StatusServiceUnavailable, err)
		return
	}

	channels, err := h.agentChannelPayloads(c.Request.Context(), caller, service)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.AgentChannelsResponse{Channels: channels})
}

// AgentChannelRecv returns queued channel messages for the validated caller, optionally waiting.
func (h *BaseHandlers) AgentChannelRecv(c *gin.Context) {
	caller, ok := h.requireAgentCaller(c, agentActionChannelRecv)
	if !ok {
		return
	}
	service, err := h.networkServiceRequired()
	if err != nil {
		h.respondError(c, http.StatusServiceUnavailable, err)
		return
	}
	channel := strings.TrimSpace(c.Param("channel"))
	if channel == "" {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(errors.New("channel is required")))
		return
	}
	if err := network.ValidateChannel(channel); err != nil {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(err))
		return
	}
	wait, err := parseBoolQuery(c, "wait")
	if err != nil {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(err))
		return
	}
	limit, err := parsePositiveIntQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(err))
		return
	}

	envelopes, err := agentChannelInbox(
		c.Request.Context(),
		service,
		caller.Session.ID,
		channel,
		wait,
	)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	messages := agentChannelMessagesFromEnvelopes(envelopes, channel, limit)
	c.JSON(http.StatusOK, contract.AgentChannelMessagesResponse{Messages: messages})
}

// AgentChannelSend sends one coordination message using the validated caller identity.
func (h *BaseHandlers) AgentChannelSend(c *gin.Context) {
	caller, ok := h.requireAgentCaller(c, agentActionChannelSend)
	if !ok {
		return
	}
	service, err := h.networkServiceRequired()
	if err != nil {
		h.respondError(c, http.StatusServiceUnavailable, err)
		return
	}
	channel := strings.TrimSpace(c.Param("channel"))
	if channel == "" {
		h.respondError(c, http.StatusBadRequest, NewNetworkValidationError(errors.New("channel is required")))
		return
	}

	var req contract.AgentChannelSendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode agent channel send request: %w", h.transportName(), err),
		)
		return
	}
	if err := validateAgentChannelRequest(req.Body, req.Metadata, req); err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}

	ext, err := coordinationExt(req.Metadata)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	sendReq := network.SendRequest{
		SessionID: strings.TrimSpace(caller.Session.ID),
		Channel:   channel,
		Surface:   networkSurfacePtr(network.SurfaceThread),
		ThreadID:  ptrString(agentChannelThreadID),
		Kind:      network.KindSay,
		Body:      cloneRawMessage(req.Body),
		Ext:       ext,
	}
	if idempotencyKey := strings.TrimSpace(req.IdempotencyKey); idempotencyKey != "" {
		sendReq.ID = ptrString(idempotencyKey)
	}

	messageID, err := service.Send(c.Request.Context(), sendReq)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	c.JSON(http.StatusAccepted, contract.AgentChannelMessageResponse{
		Message: agentChannelMessageFromRequest(
			messageID,
			channel,
			caller.Session.ID,
			"",
			req.Body,
			req.Metadata,
			h.nowUTC(),
		),
	})
}

// AgentChannelReply replies to one queued or persisted message using the validated caller identity.
func (h *BaseHandlers) AgentChannelReply(c *gin.Context) {
	caller, ok := h.requireAgentCaller(c, agentActionChannelReply)
	if !ok {
		return
	}
	service, err := h.networkServiceRequired()
	if err != nil {
		h.respondError(c, http.StatusServiceUnavailable, err)
		return
	}

	req, err := decodeAgentChannelReplyRequest(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	source, sourceMetadata, err := h.resolveAgentReplySource(
		c.Request.Context(),
		service,
		caller,
		req.ReplyToMessageID,
	)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	metadata, err := agentChannelReplyMetadata(req.Metadata, sourceMetadata)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	if err := validateAgentChannelRequest(req.Body, metadata, struct {
		Body     json.RawMessage                             `json:"body"`
		Metadata contract.CoordinationMessageMetadataPayload `json:"metadata"`
	}{Body: req.Body, Metadata: metadata}); err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}

	ext, err := coordinationExt(metadata)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	targetPeer := strings.TrimSpace(source.From)
	sendReq, err := agentChannelReplySendRequest(caller, source, req, ext)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	if idempotencyKey := strings.TrimSpace(req.IdempotencyKey); idempotencyKey != "" {
		sendReq.ID = ptrString(idempotencyKey)
	}

	messageID, err := service.Send(c.Request.Context(), sendReq)
	if err != nil {
		h.respondError(c, StatusForNetworkError(err), err)
		return
	}
	c.JSON(http.StatusAccepted, contract.AgentChannelMessageResponse{
		Message: agentChannelMessageFromRequest(
			messageID,
			source.Channel,
			caller.Session.ID,
			targetPeer,
			req.Body,
			metadata,
			h.nowUTC(),
		),
	})
}

func agentChannelReplyMetadata(
	requestMetadata contract.CoordinationMessageMetadataPayload,
	sourceMetadata sourceCoordinationMetadata,
) (contract.CoordinationMessageMetadataPayload, error) {
	if zeroCoordinationMetadata(requestMetadata) {
		if !sourceMetadata.ok {
			return contract.CoordinationMessageMetadataPayload{}, NewNetworkValidationError(errors.New(
				"metadata is required when the source message has no coordination metadata",
			))
		}
		metadata := sourceMetadata.metadata
		metadata.MessageKind = contract.CoordinationMessageReply
		return metadata, nil
	}
	if requestMetadata.MessageKind != contract.CoordinationMessageReply {
		return contract.CoordinationMessageMetadataPayload{}, NewNetworkValidationError(errors.New(
			"metadata.message_kind must be reply for agent channel replies",
		))
	}
	return requestMetadata, nil
}

func agentChannelReplySendRequest(
	caller agentidentity.Caller,
	source network.Envelope,
	req decodedAgentChannelReplyRequest,
	ext map[string]json.RawMessage,
) (network.SendRequest, error) {
	callerPeerID := agentCallerPeerID(caller)
	workspaceID := strings.TrimSpace(caller.Session.WorkspaceID)
	directID, _, _, err := network.DirectRoomIdentity(workspaceID, source.Channel, callerPeerID, source.From)
	if err != nil {
		return network.SendRequest{}, err
	}
	sendReq := network.SendRequest{
		SessionID:   strings.TrimSpace(caller.Session.ID),
		WorkspaceID: workspaceID,
		Channel:     strings.TrimSpace(source.Channel),
		Surface:     networkSurfacePtr(network.SurfaceDirect),
		DirectID:    ptrString(directID),
		Kind:        network.KindSay,
		To:          ptrString(strings.TrimSpace(source.From)),
		ReplyTo:     ptrString(req.ReplyToMessageID),
		Body:        cloneRawMessage(req.Body),
		Ext:         ext,
	}
	if source.WorkID != nil && strings.TrimSpace(*source.WorkID) != "" {
		sendReq.WorkID = ptrString(*source.WorkID)
	}
	if source.TraceID != nil && strings.TrimSpace(*source.TraceID) != "" {
		sendReq.TraceID = ptrString(*source.TraceID)
	}
	return sendReq, nil
}

func networkSurfacePtr(value network.Surface) *network.Surface {
	return &value
}

func agentCallerPeerID(caller agentidentity.Caller) string {
	agentName := strings.ToLower(strings.TrimSpace(caller.Session.AgentName))
	sessionID := strings.TrimSpace(caller.Session.ID)
	if agentName == "" {
		return sessionID
	}
	return agentName + "." + sessionID
}

type sourceCoordinationMetadata struct {
	metadata contract.CoordinationMessageMetadataPayload
	ok       bool
}

type decodedAgentChannelReplyRequest struct {
	ReplyToMessageID string
	Body             json.RawMessage
	Metadata         contract.CoordinationMessageMetadataPayload
	IdempotencyKey   string
}

func (h *BaseHandlers) enrichAgentMePayload(
	ctx context.Context,
	caller agentidentity.Caller,
	payload *contract.AgentMePayload,
) {
	if payload == nil {
		return
	}
	if h != nil && h.AgentContextService != nil {
		contextPayload, err := h.AgentContextService.ContextForSession(ctx, sessionInfoFromAgentCaller(caller))
		if err == nil {
			payload.Workspace = contextPayload.Workspace
			payload.Capabilities = contextPayload.Capabilities.Capabilities
			payload.Limits = contextPayload.Limits
			if contextPayload.Task.Lease != nil {
				payload.ActiveTaskLeases = []contract.TaskRunLeaseSummaryPayload{*contextPayload.Task.Lease}
			}
			if contextPayload.CoordinationChannel.Channel != nil {
				payload.Channels = append(payload.Channels, *contextPayload.CoordinationChannel.Channel)
			}
		}
	}
	if h == nil {
		return
	}
	if coordinatorPayload, err := h.agentCoordinatorConfigPayload(ctx, caller.Session.WorkspaceID); err == nil {
		payload.Coordinator = coordinatorPayload
	}
	service, err := h.networkServiceRequired()
	if err != nil {
		if callerChannel := strings.TrimSpace(
			caller.Session.Channel,
		); callerChannel != "" &&
			len(payload.Channels) == 0 {
			payload.Channels = []contract.CoordinationChannelPayload{
				contract.NormalizeCoordinationChannelPayload(coordinationChannelFromNetwork(
					callerChannel,
					caller.Session.WorkspaceID,
					store.NetworkChannelEntry{},
				)),
			}
		}
		return
	}
	channels, err := h.agentChannelPayloads(ctx, caller, service)
	if err == nil {
		payload.Channels = mergeCoordinationChannels(payload.Channels, channels)
	}
}

func (h *BaseHandlers) agentChannelPayloads(
	ctx context.Context,
	caller agentidentity.Caller,
	service NetworkService,
) ([]contract.CoordinationChannelPayload, error) {
	infos, err := service.ListChannels(ctx, strings.TrimSpace(caller.Session.WorkspaceID))
	if err != nil {
		return nil, err
	}

	metadata := h.agentChannelMetadata(ctx, caller.Session.WorkspaceID)
	payloadByID := make(map[string]contract.CoordinationChannelPayload, len(infos)+len(metadata))
	for _, info := range infos {
		channel := strings.TrimSpace(info.Channel)
		if channel == "" {
			continue
		}
		entry, hasEntry := metadata[channel]
		if len(metadata) > 0 && !hasEntry && channel != strings.TrimSpace(caller.Session.Channel) {
			continue
		}
		payloadByID[channel] = coordinationChannelFromNetwork(channel, caller.Session.WorkspaceID, entry)
	}
	for channel, entry := range metadata {
		if _, ok := payloadByID[channel]; ok {
			continue
		}
		payloadByID[channel] = coordinationChannelFromNetwork(channel, caller.Session.WorkspaceID, entry)
	}
	if callerChannel := strings.TrimSpace(caller.Session.Channel); callerChannel != "" {
		if _, ok := payloadByID[callerChannel]; !ok {
			payloadByID[callerChannel] = coordinationChannelFromNetwork(
				callerChannel,
				caller.Session.WorkspaceID,
				store.NetworkChannelEntry{},
			)
		}
	}

	channels := make([]contract.CoordinationChannelPayload, 0, len(payloadByID))
	for _, payload := range payloadByID {
		channels = append(channels, contract.NormalizeCoordinationChannelPayload(payload))
	}
	sortCoordinationChannels(channels)
	return channels, nil
}

func mergeCoordinationChannels(
	left []contract.CoordinationChannelPayload,
	right []contract.CoordinationChannelPayload,
) []contract.CoordinationChannelPayload {
	mergedByID := make(map[string]contract.CoordinationChannelPayload, len(left)+len(right))
	for _, channel := range left {
		normalized := contract.NormalizeCoordinationChannelPayload(channel)
		id := firstNonEmpty(normalized.ID, normalized.Channel)
		if id != "" {
			mergedByID[id] = normalized
		}
	}
	for _, channel := range right {
		normalized := contract.NormalizeCoordinationChannelPayload(channel)
		id := firstNonEmpty(normalized.ID, normalized.Channel)
		if id != "" {
			mergedByID[id] = normalized
		}
	}
	merged := make([]contract.CoordinationChannelPayload, 0, len(mergedByID))
	for _, channel := range mergedByID {
		merged = append(merged, channel)
	}
	sortCoordinationChannels(merged)
	return merged
}

func (h *BaseHandlers) agentChannelMetadata(
	ctx context.Context,
	workspaceID string,
) map[string]store.NetworkChannelEntry {
	if h == nil || h.NetworkStore == nil || strings.TrimSpace(workspaceID) == "" {
		return nil
	}
	entries, err := h.NetworkStore.ListNetworkChannels(ctx, store.NetworkChannelQuery{
		WorkspaceID: strings.TrimSpace(workspaceID),
	})
	if err != nil {
		if h.Logger != nil {
			h.Logger.Warn("api: skip agent channel metadata", "error", err)
		}
		return nil
	}
	metadata := make(map[string]store.NetworkChannelEntry, len(entries))
	for _, entry := range entries {
		channel := strings.TrimSpace(entry.Channel)
		if channel != "" {
			metadata[channel] = entry
		}
	}
	return metadata
}

func coordinationChannelFromNetwork(
	channel string,
	workspaceID string,
	entry store.NetworkChannelEntry,
) contract.CoordinationChannelPayload {
	channel = strings.TrimSpace(channel)
	workspaceID = firstNonEmpty(strings.TrimSpace(entry.WorkspaceID), strings.TrimSpace(workspaceID))
	payload := contract.CoordinationChannelPayload{
		ID:          channel,
		Channel:     channel,
		DisplayName: channel,
		Purpose:     firstNonEmpty(strings.TrimSpace(entry.Purpose), "network_channel"),
		WorkspaceID: workspaceID,
	}
	if !entry.UpdatedAt.IsZero() {
		updatedAt := entry.UpdatedAt.UTC()
		payload.LastActivityAt = &updatedAt
	}
	return payload
}

func agentChannelInbox(
	ctx context.Context,
	service NetworkService,
	sessionID string,
	channel string,
	wait bool,
) ([]network.Envelope, error) {
	if !wait {
		envelopes, err := service.Inbox(ctx, strings.TrimSpace(sessionID))
		if err != nil {
			return nil, err
		}
		return envelopes, nil
	}
	envelopes, err := service.WaitInbox(
		ctx,
		strings.TrimSpace(sessionID),
		strings.TrimSpace(channel),
	)
	if err != nil {
		return nil, err
	}
	return envelopes, nil
}

func (h *BaseHandlers) resolveAgentReplySource(
	ctx context.Context,
	service NetworkService,
	caller agentidentity.Caller,
	messageID string,
) (network.Envelope, sourceCoordinationMetadata, error) {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return network.Envelope{}, sourceCoordinationMetadata{}, NewNetworkValidationError(
			errors.New("reply_to_message_id is required"),
		)
	}

	envelopes, err := service.Inbox(ctx, strings.TrimSpace(caller.Session.ID))
	if err != nil {
		return network.Envelope{}, sourceCoordinationMetadata{}, err
	}
	for _, envelope := range envelopes {
		if strings.TrimSpace(envelope.ID) != messageID {
			continue
		}
		metadata, ok := coordinationMetadataFromEnvelope(envelope)
		return envelope, sourceCoordinationMetadata{metadata: metadata, ok: ok}, validateReplySource(envelope)
	}

	if h != nil && h.NetworkStore != nil {
		entries, lookupErr := h.NetworkStore.ListNetworkMessages(ctx, store.NetworkMessageQuery{
			WorkspaceID: strings.TrimSpace(caller.Session.WorkspaceID),
			SessionID:   strings.TrimSpace(caller.Session.ID),
			MessageID:   messageID,
			Limit:       1,
		})
		if lookupErr != nil {
			return network.Envelope{}, sourceCoordinationMetadata{}, lookupErr
		}
		if len(entries) > 0 {
			envelope := envelopeFromNetworkMessage(entries[0])
			return envelope, sourceCoordinationMetadata{}, validateReplySource(envelope)
		}
	}

	return network.Envelope{}, sourceCoordinationMetadata{}, fmt.Errorf(
		"%w: message_id=%q",
		network.ErrTargetPeerNotFound,
		messageID,
	)
}

func validateReplySource(envelope network.Envelope) error {
	if strings.TrimSpace(envelope.WorkspaceID) == "" {
		return NewNetworkValidationError(errors.New("source message workspace_id is required"))
	}
	if strings.TrimSpace(envelope.Channel) == "" {
		return NewNetworkValidationError(errors.New("source message channel is required"))
	}
	if strings.TrimSpace(envelope.From) == "" {
		return NewNetworkValidationError(errors.New("source message sender is required"))
	}
	return nil
}

func decodeAgentChannelReplyRequest(c *gin.Context) (decodedAgentChannelReplyRequest, error) {
	if c == nil || c.Request == nil || c.Request.Body == nil {
		return decodedAgentChannelReplyRequest{}, NewNetworkValidationError(
			errors.New("reply request body is required"),
		)
	}
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(c.Request.Body).Decode(&raw); err != nil {
		return decodedAgentChannelReplyRequest{}, fmt.Errorf("decode agent channel reply request: %w", err)
	}
	if err := contract.ValidateNoRawClaimTokenField(raw); err != nil {
		return decodedAgentChannelReplyRequest{}, NewNetworkValidationError(err)
	}

	var req decodedAgentChannelReplyRequest
	if err := decodeRawString(raw["reply_to_message_id"], &req.ReplyToMessageID); err != nil {
		return decodedAgentChannelReplyRequest{}, NewNetworkValidationError(fmt.Errorf("reply_to_message_id: %w", err))
	}
	if err := decodeRawString(raw["idempotency_key"], &req.IdempotencyKey); err != nil {
		return decodedAgentChannelReplyRequest{}, NewNetworkValidationError(fmt.Errorf("idempotency_key: %w", err))
	}
	req.Body = cloneRawMessage(raw["body"])
	if len(bytes.TrimSpace(req.Body)) == 0 {
		return decodedAgentChannelReplyRequest{}, NewNetworkValidationError(errors.New("body is required"))
	}

	if metadataRaw := bytes.TrimSpace(raw["metadata"]); len(metadataRaw) > 0 &&
		!bytes.Equal(metadataRaw, []byte("null")) &&
		!bytes.Equal(metadataRaw, []byte("{}")) {
		metadata, ok, err := decodeOptionalCoordinationMetadata(metadataRaw)
		if err != nil {
			return decodedAgentChannelReplyRequest{}, NewNetworkValidationError(fmt.Errorf("metadata: %w", err))
		}
		if ok {
			req.Metadata = metadata
		}
	}
	return req, nil
}

func decodeOptionalCoordinationMetadata(
	raw json.RawMessage,
) (contract.CoordinationMessageMetadataPayload, bool, error) {
	var decoded struct {
		TaskID                string                           `json:"task_id"`
		RunID                 string                           `json:"run_id"`
		WorkflowID            string                           `json:"workflow_id,omitempty"`
		CoordinationChannelID string                           `json:"coordination_channel_id"`
		MessageKind           contract.CoordinationMessageKind `json:"message_kind"`
		CorrelationID         string                           `json:"correlation_id"`
		Ext                   map[string]json.RawMessage       `json:"ext,omitempty"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return contract.CoordinationMessageMetadataPayload{}, false, err
	}
	metadata := contract.CoordinationMessageMetadataPayload{
		TaskID:                decoded.TaskID,
		RunID:                 decoded.RunID,
		WorkflowID:            decoded.WorkflowID,
		CoordinationChannelID: decoded.CoordinationChannelID,
		MessageKind:           decoded.MessageKind,
		CorrelationID:         decoded.CorrelationID,
		Ext:                   decoded.Ext,
	}
	if zeroCoordinationMetadata(metadata) {
		return contract.CoordinationMessageMetadataPayload{}, false, nil
	}
	if err := metadata.Validate(); err != nil {
		return contract.CoordinationMessageMetadataPayload{}, false, err
	}
	return metadata, true, nil
}

func decodeRawString(raw json.RawMessage, target *string) error {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	*target = strings.TrimSpace(value)
	return nil
}

func validateAgentChannelRequest(
	body json.RawMessage,
	metadata contract.CoordinationMessageMetadataPayload,
	fullPayload any,
) error {
	if len(bytes.TrimSpace(body)) == 0 {
		return NewNetworkValidationError(errors.New("body is required"))
	}
	if !json.Valid(body) {
		return NewNetworkValidationError(errors.New("body must be valid JSON"))
	}
	if err := metadata.Validate(); err != nil {
		return NewNetworkValidationError(err)
	}
	if err := contract.ValidateNoRawClaimTokenField(fullPayload); err != nil {
		return NewNetworkValidationError(err)
	}
	return nil
}

func coordinationExt(
	metadata contract.CoordinationMessageMetadataPayload,
) (map[string]json.RawMessage, error) {
	raw, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("api: marshal coordination metadata: %w", err)
	}
	return map[string]json.RawMessage{agentCoordinationExtKey: raw}, nil
}

func agentChannelMessagesFromEnvelopes(
	envelopes []network.Envelope,
	channel string,
	limit int,
) []contract.AgentChannelMessagePayload {
	filtered := filterAgentChannelEnvelopes(envelopes, channel)
	messages := make([]contract.AgentChannelMessagePayload, 0, len(filtered))
	for _, envelope := range filtered {
		metadata, ok := coordinationMetadataFromEnvelope(envelope)
		if !ok {
			continue
		}
		if err := contract.ValidateNoRawClaimTokenField(struct {
			Body     json.RawMessage                             `json:"body"`
			Metadata contract.CoordinationMessageMetadataPayload `json:"metadata"`
		}{Body: envelope.Body, Metadata: metadata}); err != nil {
			continue
		}
		messages = append(messages, contract.AgentChannelMessagePayload{
			MessageID: strings.TrimSpace(envelope.ID),
			ChannelID: firstNonEmpty(
				strings.TrimSpace(metadata.CoordinationChannelID),
				strings.TrimSpace(envelope.Channel),
			),
			FromSessionID: strings.TrimSpace(envelope.From),
			ToSessionID:   stringPtrValue(envelope.To),
			Body:          cloneRawMessage(envelope.Body),
			Metadata:      metadata,
			Timestamp:     envelopeTime(envelope),
		})
	}
	sort.SliceStable(messages, func(left, right int) bool {
		if !messages[left].Timestamp.Equal(messages[right].Timestamp) {
			return messages[left].Timestamp.Before(messages[right].Timestamp)
		}
		return messages[left].MessageID < messages[right].MessageID
	})
	if limit > 0 && len(messages) > limit {
		return messages[:limit]
	}
	return messages
}

func agentChannelMessageFromRequest(
	messageID string,
	channel string,
	fromSessionID string,
	toSessionID string,
	body json.RawMessage,
	metadata contract.CoordinationMessageMetadataPayload,
	timestamp time.Time,
) contract.AgentChannelMessagePayload {
	return contract.AgentChannelMessagePayload{
		MessageID:     strings.TrimSpace(messageID),
		ChannelID:     firstNonEmpty(strings.TrimSpace(metadata.CoordinationChannelID), strings.TrimSpace(channel)),
		FromSessionID: strings.TrimSpace(fromSessionID),
		ToSessionID:   strings.TrimSpace(toSessionID),
		Body:          cloneRawMessage(body),
		Metadata:      metadata,
		Timestamp:     timestamp.UTC(),
	}
}

func filterAgentChannelEnvelopes(envelopes []network.Envelope, channel string) []network.Envelope {
	channel = strings.TrimSpace(channel)
	filtered := make([]network.Envelope, 0, len(envelopes))
	for _, envelope := range envelopes {
		if channel == "" || strings.TrimSpace(envelope.Channel) == channel {
			filtered = append(filtered, envelope)
		}
	}
	return filtered
}

func coordinationMetadataFromEnvelope(
	envelope network.Envelope,
) (contract.CoordinationMessageMetadataPayload, bool) {
	for _, key := range []string{agentCoordinationExtKey, "coordination_metadata", "agh_coordination", "metadata"} {
		if raw, ok := envelope.Ext[key]; ok {
			var metadata contract.CoordinationMessageMetadataPayload
			if err := json.Unmarshal(raw, &metadata); err == nil {
				return metadata, true
			}
		}
	}
	return contract.CoordinationMessageMetadataPayload{}, false
}

func envelopeFromNetworkMessage(entry store.NetworkMessageEntry) network.Envelope {
	envelope := network.Envelope{
		Protocol:    network.ProtocolV2,
		ID:          strings.TrimSpace(entry.MessageID),
		Kind:        network.Kind(strings.TrimSpace(entry.Kind)),
		WorkspaceID: strings.TrimSpace(entry.WorkspaceID),
		Channel:     strings.TrimSpace(entry.Channel),
		From:        strings.TrimSpace(entry.PeerFrom),
		WorkID:      optionalStringPtr(entry.WorkID),
		ReplyTo:     optionalStringPtr(entry.ReplyTo),
		TraceID:     optionalStringPtr(entry.TraceID),
		CausationID: optionalStringPtr(entry.CausationID),
		TS:          entry.Timestamp.Unix(),
		Body:        cloneRawMessage(entry.Body),
	}
	if to := strings.TrimSpace(entry.PeerTo); to != "" {
		envelope.To = &to
	}
	return envelope
}

func sessionInfoFromAgentCaller(caller agentidentity.Caller) *session.Info {
	return &session.Info{
		ID:               strings.TrimSpace(caller.Session.ID),
		Name:             strings.TrimSpace(caller.Session.Name),
		AgentName:        strings.TrimSpace(caller.Session.AgentName),
		Provider:         strings.TrimSpace(caller.Session.Provider),
		Model:            strings.TrimSpace(caller.Session.Model),
		WorkspaceID:      strings.TrimSpace(caller.Session.WorkspaceID),
		Workspace:        strings.TrimSpace(caller.Session.WorkspacePath),
		Channel:          strings.TrimSpace(caller.Session.Channel),
		Type:             caller.Session.Type,
		Lineage:          store.CloneSessionLineage(caller.Session.Lineage),
		State:            caller.Session.State,
		SoulSnapshotID:   strings.TrimSpace(caller.Session.SoulSnapshotID),
		SoulDigest:       strings.TrimSpace(caller.Session.SoulDigest),
		ParentSoulDigest: strings.TrimSpace(caller.Session.ParentSoulDigest),
		CreatedAt:        caller.Session.CreatedAt,
		UpdatedAt:        caller.Session.UpdatedAt,
	}
}

func sortCoordinationChannels(channels []contract.CoordinationChannelPayload) {
	sort.SliceStable(channels, func(left, right int) bool {
		if channels[left].WorkspaceID != channels[right].WorkspaceID {
			return channels[left].WorkspaceID < channels[right].WorkspaceID
		}
		return channels[left].ID < channels[right].ID
	})
}

func parseBoolQuery(c *gin.Context, key string) (bool, error) {
	if c == nil {
		return false, nil
	}
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return false, nil
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("query parameter %q must be a boolean: %w", key, err)
	}
	return parsed, nil
}

func parsePositiveIntQuery(c *gin.Context) (int, error) {
	const key = "limit"
	if c == nil {
		return 0, nil
	}
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("query parameter %q must be a positive integer: %w", key, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("query parameter %q must be a positive integer: %d", key, parsed)
	}
	return parsed, nil
}

func (h *BaseHandlers) nowUTC() time.Time {
	if h == nil || h.Now == nil {
		return time.Now().UTC()
	}
	return h.Now().UTC()
}

func envelopeTime(envelope network.Envelope) time.Time {
	if envelope.TS <= 0 {
		return time.Time{}
	}
	return time.Unix(envelope.TS, 0).UTC()
}

func zeroCoordinationMetadata(metadata contract.CoordinationMessageMetadataPayload) bool {
	return strings.TrimSpace(metadata.TaskID) == "" &&
		strings.TrimSpace(metadata.RunID) == "" &&
		strings.TrimSpace(metadata.WorkflowID) == "" &&
		strings.TrimSpace(metadata.CoordinationChannelID) == "" &&
		strings.TrimSpace(string(metadata.MessageKind)) == "" &&
		strings.TrimSpace(metadata.CorrelationID) == "" &&
		len(metadata.Ext) == 0
}

func optionalStringPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
