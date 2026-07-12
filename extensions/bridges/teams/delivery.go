package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	bridgepkg "github.com/compozy/agh/internal/bridges"
	"github.com/compozy/agh/internal/bridgesdk"
)

const teamsMaxMessageLen = 28_000

type deliveryState struct {
	LastSeq                int64
	RemoteMessageID        string
	ReplaceRemoteMessageID string
	LastContent            string
}

type teamsDeliveryStateLookup func(deliveryID string) (deliveryState, bool)

func (p *teamsProvider) deliveryState(instanceID string, deliveryID string) deliveryState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.deliveries[deliveryStateKey(instanceID, deliveryID)]
}

func (p *teamsProvider) storeDeliveryState(
	instanceID string,
	deliveryID string,
	state deliveryState,
) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deliveries[deliveryStateKey(instanceID, deliveryID)] = state
}

func executeTeamsDelivery(
	ctx context.Context,
	api teamsAPI,
	cfg resolvedInstanceConfig,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
	referenceStateLookup teamsDeliveryStateLookup,
	userContextLookup func(string, string) (teamsUserContext, bool),
) (bridgepkg.DeliveryAck, deliveryState, error) {
	if err := request.Validate(); err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}

	event := request.Event
	if event.EventType != bridgepkg.DeliveryEventTypeResume && event.Seq <= state.LastSeq {
		return bridgepkg.DeliveryAck{}, state, fmt.Errorf(
			"teams: out-of-order delivery seq %d after %d",
			event.Seq,
			state.LastSeq,
		)
	}
	if event.EventType == bridgepkg.DeliveryEventTypeResume && request.Snapshot != nil {
		state.LastSeq = request.Snapshot.LastAckedSeq
		state.RemoteMessageID = strings.TrimSpace(request.Snapshot.RemoteMessageID)
		state.ReplaceRemoteMessageID = strings.TrimSpace(request.Snapshot.ReplaceRemoteMessageID)
	}

	switch {
	case isTeamsDeleteDelivery(event):
		return executeTeamsDeleteDelivery(ctx, api, event, request.Snapshot, state, referenceStateLookup)
	case shouldPostTeamsMessage(event, state, request):
		return executeTeamsPostDelivery(ctx, api, cfg, event, state, userContextLookup)
	default:
		return executeTeamsEditDelivery(
			ctx,
			api,
			cfg,
			event,
			request.Snapshot,
			state,
			referenceStateLookup,
			userContextLookup,
		)
	}
}

func isTeamsDeleteDelivery(event bridgepkg.DeliveryEvent) bool {
	return event.Operation.Normalize() == bridgepkg.DeliveryOperationDelete ||
		normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeDelete
}

func executeTeamsDeleteDelivery(
	ctx context.Context,
	api teamsAPI,
	event bridgepkg.DeliveryEvent,
	snapshot *bridgepkg.DeliverySnapshot,
	state deliveryState,
	referenceStateLookup teamsDeliveryStateLookup,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	remoteID := resolveTeamsReferencedRemoteMessageID(event.Reference, snapshot, state, referenceStateLookup)
	if remoteID == "" {
		return bridgepkg.DeliveryAck{}, state, errors.New(
			"teams: delete delivery requires a remote message id",
		)
	}
	ref, err := decodeRemoteMessageID(remoteID)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	if err := api.DeleteActivity(ctx, ref.ServiceURL, ref.ConversationID, ref.ActivityID); err != nil {
		return bridgepkg.DeliveryAck{}, state, fmt.Errorf("teams: delete activity: %w", err)
	}

	ack := newTeamsDeliveryAck(event, remoteID, firstNonEmpty(state.RemoteMessageID, remoteID))
	state.LastSeq = event.Seq
	state.RemoteMessageID = remoteID
	state.ReplaceRemoteMessageID = ack.ReplaceRemoteMessageID
	return ack, state, ack.ValidateFor(event)
}

func executeTeamsPostDelivery(
	ctx context.Context,
	api teamsAPI,
	cfg resolvedInstanceConfig,
	event bridgepkg.DeliveryEvent,
	state deliveryState,
	userContextLookup func(string, string) (teamsUserContext, bool),
) (bridgepkg.DeliveryAck, deliveryState, error) {
	target, err := resolveTeamsDeliveryTarget(cfg, event, userContextLookup)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}

	conversationID := target.ConversationID
	serviceURL := target.ServiceURL
	if conversationID == "" {
		createReq := teamsCreateConversationRequest{
			Bot:      teamsChannelAccount{ID: cfg.appID},
			Members:  []teamsChannelAccount{{ID: target.UserID}},
			IsGroup:  false,
			TenantID: target.TenantID,
			ChannelData: map[string]any{
				"tenant": map[string]any{"id": target.TenantID},
			},
		}
		created, createErr := api.CreateConversation(ctx, serviceURL, createReq)
		if createErr != nil {
			return bridgepkg.DeliveryAck{}, state, fmt.Errorf("teams: create conversation: %w", createErr)
		}
		if created == nil || strings.TrimSpace(created.ID) == "" {
			return bridgepkg.DeliveryAck{}, state, &bridgesdk.TransientError{
				Err: errors.New("teams: create conversation response omitted id"),
			}
		}
		conversationID = strings.TrimSpace(created.ID)
	}

	baseConversationID, replyToID := splitTeamsConversationTarget(
		firstNonEmpty(conversationID, target.ConversationID),
	)
	if target.ReplyToID != "" {
		replyToID = target.ReplyToID
	}

	chunks := teamsDeliveryChunks(event)
	remoteID := ""
	for index, chunk := range chunks {
		sent, sendErr := api.SendActivity(
			ctx,
			serviceURL,
			baseConversationID,
			replyToID,
			teamsOutboundActivity{
				Type:       providerMessageKey,
				Text:       chunk,
				TextFormat: "markdown",
			},
		)
		if sendErr != nil {
			return bridgepkg.DeliveryAck{}, state, fmt.Errorf("teams: send activity chunk %d: %w", index+1, sendErr)
		}
		if sent == nil || strings.TrimSpace(sent.ID) == "" {
			return bridgepkg.DeliveryAck{}, state, &bridgesdk.TransientError{
				Err: errors.New("teams: send activity response omitted id"),
			}
		}
		remoteID = encodeRemoteMessageID(teamsRemoteMessageRef{
			ConversationID: baseConversationID,
			ServiceURL:     serviceURL,
			ActivityID:     strings.TrimSpace(sent.ID),
		})
	}

	ack := newTeamsDeliveryAck(event, remoteID, state.RemoteMessageID)
	state.LastSeq = event.Seq
	state.ReplaceRemoteMessageID = state.RemoteMessageID
	state.RemoteMessageID = remoteID
	state.LastContent = chunks[len(chunks)-1]
	return ack, state, ack.ValidateFor(event)
}

func executeTeamsEditDelivery(
	ctx context.Context,
	api teamsAPI,
	cfg resolvedInstanceConfig,
	event bridgepkg.DeliveryEvent,
	snapshot *bridgepkg.DeliverySnapshot,
	state deliveryState,
	referenceStateLookup teamsDeliveryStateLookup,
	userContextLookup func(string, string) (teamsUserContext, bool),
) (bridgepkg.DeliveryAck, deliveryState, error) {
	remoteID := resolveTeamsReferencedRemoteMessageID(event.Reference, snapshot, state, referenceStateLookup)
	if remoteID == "" {
		return bridgepkg.DeliveryAck{}, state, errors.New(
			"teams: edit delivery requires a remote message id",
		)
	}
	ref, err := decodeRemoteMessageID(remoteID)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}

	chunks := teamsDeliveryChunks(event)
	continuationReplyToID := ""
	if len(chunks) > 1 {
		target, targetErr := resolveTeamsDeliveryTarget(cfg, event, userContextLookup)
		if targetErr != nil {
			return bridgepkg.DeliveryAck{}, state, fmt.Errorf("teams: resolve continuation target: %w", targetErr)
		}
		if target.ConversationID == ref.ConversationID && target.ServiceURL == ref.ServiceURL {
			continuationReplyToID = target.ReplyToID
		}
	}
	if remoteID != state.RemoteMessageID || chunks[0] != state.LastContent {
		if err := api.UpdateActivity(ctx, ref.ServiceURL, ref.ConversationID, ref.ActivityID, teamsOutboundActivity{
			Type:       providerMessageKey,
			Text:       chunks[0],
			TextFormat: "markdown",
		}); err != nil {
			return bridgepkg.DeliveryAck{}, state, fmt.Errorf("teams: update activity: %w", err)
		}
	}

	lastRemoteID := remoteID
	for index, chunk := range chunks[1:] {
		sent, sendErr := api.SendActivity(
			ctx,
			ref.ServiceURL,
			ref.ConversationID,
			continuationReplyToID,
			teamsOutboundActivity{
				Type:       providerMessageKey,
				Text:       chunk,
				TextFormat: "markdown",
			},
		)
		if sendErr != nil {
			return bridgepkg.DeliveryAck{}, state, fmt.Errorf("teams: send continuation %d: %w", index+1, sendErr)
		}
		if sent == nil || strings.TrimSpace(sent.ID) == "" {
			return bridgepkg.DeliveryAck{}, state, &bridgesdk.TransientError{
				Err: errors.New("teams: send activity response omitted id"),
			}
		}
		lastRemoteID = encodeRemoteMessageID(teamsRemoteMessageRef{
			ConversationID: ref.ConversationID,
			ServiceURL:     ref.ServiceURL,
			ActivityID:     strings.TrimSpace(sent.ID),
		})
	}

	ack := newTeamsDeliveryAck(event, lastRemoteID, remoteID)
	state.LastSeq = event.Seq
	state.RemoteMessageID = lastRemoteID
	state.ReplaceRemoteMessageID = remoteID
	state.LastContent = chunks[len(chunks)-1]
	return ack, state, ack.ValidateFor(event)
}

func teamsDeliveryChunks(event bridgepkg.DeliveryEvent) []string {
	chunks := bridgesdk.ChunkMessage(event.Content.Text, teamsMaxMessageLen, nil)
	if !event.Final && len(chunks) > 1 {
		return chunks[:1]
	}
	return chunks
}

func resolveTeamsReferencedRemoteMessageID(
	reference *bridgepkg.DeliveryMessageReference,
	snapshot *bridgepkg.DeliverySnapshot,
	state deliveryState,
	referenceStateLookup teamsDeliveryStateLookup,
) string {
	if remoteID := referenceRemoteMessageID(reference); remoteID != "" {
		return remoteID
	}
	if deliveryID := referenceDeliveryID(reference); deliveryID != "" {
		if referenceStateLookup == nil {
			return ""
		}
		referencedState, ok := referenceStateLookup(deliveryID)
		if !ok {
			return ""
		}
		return strings.TrimSpace(referencedState.RemoteMessageID)
	}
	if remoteID := strings.TrimSpace(state.RemoteMessageID); remoteID != "" {
		return remoteID
	}
	if snapshot != nil {
		return strings.TrimSpace(snapshot.RemoteMessageID)
	}
	return ""
}

func newTeamsDeliveryAck(
	event bridgepkg.DeliveryEvent,
	remoteMessageID string,
	replaceRemoteMessageID string,
) bridgepkg.DeliveryAck {
	ack := bridgepkg.DeliveryAck{
		DeliveryID:      event.DeliveryID,
		Seq:             event.Seq,
		RemoteMessageID: remoteMessageID,
	}
	if strings.TrimSpace(replaceRemoteMessageID) != "" {
		ack.ReplaceRemoteMessageID = strings.TrimSpace(replaceRemoteMessageID)
	}
	return ack
}

func shouldPostTeamsMessage(
	event bridgepkg.DeliveryEvent,
	state deliveryState,
	request bridgepkg.DeliveryRequest,
) bool {
	if event.Operation.Normalize() == bridgepkg.DeliveryOperationEdit {
		return false
	}
	if normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeStart {
		return true
	}
	if normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeResume {
		if request.Snapshot == nil {
			return state.RemoteMessageID == ""
		}
		return strings.TrimSpace(request.Snapshot.RemoteMessageID) == ""
	}
	return strings.TrimSpace(state.RemoteMessageID) == ""
}
