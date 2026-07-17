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

	"github.com/compozy/agh/internal/agentidentity"
	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/network"
	"github.com/compozy/agh/internal/session"
	"github.com/compozy/agh/internal/store"
	"github.com/gin-gonic/gin"
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
	if err := decodeStrictJSONBody(c, &req); err != nil {
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
		Surface:   new(network.SurfaceThread),
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
			caller.Session.NetworkSpecSnapshot().ChannelID,
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
		if len(metadata) > 0 && !hasEntry &&
			channel != strings.TrimSpace(caller.Session.NetworkSpecSnapshot().ChannelID) {
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
	if callerChannel := strings.TrimSpace(caller.Session.NetworkSpecSnapshot().ChannelID); callerChannel != "" {
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
		id := strings.TrimSpace(normalized.ID)
		if id != "" {
			mergedByID[id] = normalized
		}
	}
	for _, channel := range right {
		normalized := contract.NormalizeCoordinationChannelPayload(channel)
		id := strings.TrimSpace(normalized.ID)
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
				strings.TrimSpace(metadata.ChannelID),
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
		ChannelID:     firstNonEmpty(strings.TrimSpace(metadata.ChannelID), strings.TrimSpace(channel)),
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

func envelopeFromNetworkMessage(entry store.NetworkMessageEntry) (network.Envelope, error) {
	envelope := network.Envelope{
		Protocol:    network.ProtocolV0,
		ID:          strings.TrimSpace(entry.MessageID),
		Kind:        network.Kind(strings.TrimSpace(entry.Kind)),
		WorkspaceID: strings.TrimSpace(entry.WorkspaceID),
		Channel:     strings.TrimSpace(entry.Channel),
		From:        strings.TrimSpace(entry.PeerFrom),
		Mentions:    cloneTrimmedStrings(entry.Mentions),
		WorkID:      optionalStringPtr(entry.WorkID),
		ReplyTo:     optionalStringPtr(entry.ReplyTo),
		TraceID:     optionalStringPtr(entry.TraceID),
		CausationID: optionalStringPtr(entry.CausationID),
		TS:          entry.Timestamp.Unix(),
		Body:        cloneRawMessage(entry.Body),
	}
	ext, err := extensionMapFromNetworkMessage(entry)
	if err != nil {
		return network.Envelope{}, err
	}
	if len(ext) > 0 {
		envelope.Ext = ext
	}
	if to := strings.TrimSpace(entry.PeerTo); to != "" {
		envelope.To = &to
	}
	return envelope, nil
}

func sessionInfoFromAgentCaller(caller agentidentity.Caller) *session.Info {
	return &session.Info{
		ID:                   strings.TrimSpace(caller.Session.ID),
		Name:                 strings.TrimSpace(caller.Session.Name),
		AgentName:            strings.TrimSpace(caller.Session.AgentName),
		Provider:             strings.TrimSpace(caller.Session.Provider),
		Model:                strings.TrimSpace(caller.Session.Model),
		WorkspaceID:          strings.TrimSpace(caller.Session.WorkspaceID),
		Workspace:            strings.TrimSpace(caller.Session.WorkspacePath),
		NetworkParticipation: caller.Session.NetworkSpecSnapshot(),
		Type:                 caller.Session.Type,
		Lineage:              store.CloneSessionLineage(caller.Session.Lineage),
		State:                caller.Session.State,
		SoulSnapshotID:       strings.TrimSpace(caller.Session.SoulSnapshotID),
		SoulDigest:           strings.TrimSpace(caller.Session.SoulDigest),
		ParentSoulDigest:     strings.TrimSpace(caller.Session.ParentSoulDigest),
		CreatedAt:            caller.Session.CreatedAt,
		UpdatedAt:            caller.Session.UpdatedAt,
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
		strings.TrimSpace(metadata.ChannelID) == "" &&
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
