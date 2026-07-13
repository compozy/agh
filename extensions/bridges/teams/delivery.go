package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	bridgepkg "github.com/compozy/agh/internal/bridges/contract"
	"github.com/compozy/agh/internal/bridgesdk"
)

const teamsMaxMessageLen = 28_000

func (p *teamsProvider) handleBridgesDeliver(
	ctx context.Context,
	session *bridgesdk.Session,
	request bridgepkg.DeliveryRequest,
) (bridgepkg.DeliveryAck, error) {
	marker := bridgesdk.DeliveryMarker{PID: os.Getpid(), Request: request}
	cfg, err := p.waitForInstanceConfig(
		strings.TrimSpace(request.Event.BridgeInstanceID),
		500*time.Millisecond,
	)
	if err != nil {
		return p.failTeamsDelivery(ctx, session, resolvedInstanceConfig{}, marker, err)
	}

	if p.markers.ShouldCrashOnce() {
		p.markers.RecordDelivery(marker)
		p.markers.RecordCrash(map[string]any{
			"crashed":            true,
			"pid":                os.Getpid(),
			"delivery_id":        strings.TrimSpace(request.Event.DeliveryID),
			"bridge_instance_id": cfg.instanceID,
		})
		os.Exit(23)
	}

	state := p.deliveryState(cfg.instanceID, request.Event.DeliveryID)
	if state.Progress != nil {
		if err := state.Progress.Flush(ctx); err != nil {
			return p.failTeamsDelivery(ctx, session, cfg, marker, err)
		}
	}

	ack, state, err := executeTeamsDelivery(
		ctx,
		p.apiFactory(cfg),
		cfg,
		request,
		state,
		func(deliveryID string) (deliveryState, bool) {
			referencedState := p.deliveryState(cfg.instanceID, deliveryID)
			return referencedState, strings.TrimSpace(referencedState.RemoteMessageID) != ""
		},
		p.userContext,
	)
	if err != nil {
		p.storeDeliveryRetryState(cfg.instanceID, request.Event.DeliveryID, state)
		return p.failTeamsDelivery(ctx, session, cfg, marker, err)
	}

	dispatcher := state.Progress
	var cleanupErr error
	if dispatcher != nil {
		cleanupErr = dispatcher.OnContent(ctx)
		if isTerminalTeamsDeliveryEvent(request.Event) {
			state.Progress = nil
		}
	}
	p.storeDeliveryState(cfg.instanceID, request.Event.DeliveryID, state)
	if dispatcher != nil && state.Progress == nil {
		dispatcher.Close()
	}

	readyErr := p.lifecycle.Host().ReportReadyIfNeeded(ctx, session, cfg.instanceID)
	switch {
	case readyErr != nil:
		p.setLastError(readyErr)
	case cleanupErr != nil:
		p.recordTeamsProgressCleanupError(cleanupErr)
	default:
		p.clearLastError()
	}

	marker.Ack = &ack
	p.markers.RecordDelivery(marker)
	return ack, nil
}

func (p *teamsProvider) failTeamsDelivery(
	ctx context.Context,
	session *bridgesdk.Session,
	cfg resolvedInstanceConfig,
	marker bridgesdk.DeliveryMarker,
	err error,
) (bridgepkg.DeliveryAck, error) {
	marker.Error = err.Error()
	p.markers.RecordDelivery(marker)
	if strings.TrimSpace(cfg.instanceID) == "" {
		p.setLastError(err)
		return bridgepkg.DeliveryAck{}, err
	}
	classified := bridgesdk.ClassifyError(err)
	_, _, reportErr := session.ReportClassifiedError(ctx, cfg.instanceID, classified)
	if reportErr != nil {
		p.setLastError(reportErr)
	} else {
		p.setLastError(err)
	}
	return bridgepkg.DeliveryAck{}, err
}

func isTerminalTeamsDeliveryEvent(event bridgepkg.DeliveryEvent) bool {
	if event.Operation.Normalize() == bridgepkg.DeliveryOperationDelete {
		return true
	}
	switch normalizeDeliveryEventType(event.EventType) {
	case bridgepkg.DeliveryEventTypeFinal,
		bridgepkg.DeliveryEventTypeError,
		bridgepkg.DeliveryEventTypeDelete:
		return true
	default:
		return false
	}
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
	if err := deleteTeamsDeliveryActivity(ctx, api, ref.ServiceURL, ref.ConversationID, ref.ActivityID); err != nil {
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
	target, err = ensureTeamsConversation(ctx, api, cfg, target)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}

	baseConversationID, replyToID := splitTeamsConversationTarget(
		target.ConversationID,
	)
	if target.ReplyToID != "" {
		replyToID = target.ReplyToID
	}

	chunks := teamsDeliveryChunks(event)
	state.Chunks = bridgesdk.BeginDeliveryChunks(
		state.Chunks,
		event.Seq,
		bridgesdk.DeliveryChunkModeCreate,
		len(chunks),
		event.Content.Text,
		"",
		state.RemoteMessageID,
	)
	for index := state.Chunks.NextChunk(); index < len(chunks); index++ {
		chunk := chunks[index]
		sent, sendErr := sendTeamsDeliveryActivity(
			ctx,
			api,
			target.ServiceURL,
			baseConversationID,
			replyToID,
			teamsOutboundActivity{
				Type:       providerMessageKey,
				Text:       chunk,
				TextFormat: teamsTextFormatMarkdown,
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
		remoteID := encodeRemoteMessageID(teamsRemoteMessageRef{
			ConversationID: baseConversationID,
			ServiceURL:     target.ServiceURL,
			ActivityID:     strings.TrimSpace(sent.ID),
		})
		state.Chunks = state.Chunks.Advance(remoteID)
	}

	remoteID := state.Chunks.LastRemoteMessageID()
	replaceRemoteID := state.Chunks.ReplaceRemoteMessageID()
	state.Chunks = bridgesdk.DeliveryChunkCursor{}
	ack := newTeamsDeliveryAck(event, remoteID, replaceRemoteID)
	state.LastSeq = event.Seq
	state.ReplaceRemoteMessageID = replaceRemoteID
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
	replaceRemoteID := firstNonEmpty(state.RemoteMessageID, remoteID)
	state.Chunks = bridgesdk.BeginDeliveryChunks(
		state.Chunks,
		event.Seq,
		bridgesdk.DeliveryChunkModeUpdate,
		len(chunks),
		event.Content.Text,
		remoteID,
		replaceRemoteID,
	)
	for index := state.Chunks.NextChunk(); index < len(chunks); index++ {
		chunk := chunks[index]
		if index == 0 {
			if remoteID != state.RemoteMessageID || chunk != state.LastContent {
				if err := updateTeamsDeliveryActivity(
					ctx,
					api,
					ref.ServiceURL,
					ref.ConversationID,
					ref.ActivityID,
					teamsOutboundActivity{
						Type:       providerMessageKey,
						Text:       chunk,
						TextFormat: teamsTextFormatMarkdown,
					},
				); err != nil {
					return bridgepkg.DeliveryAck{}, state, fmt.Errorf("teams: update activity: %w", err)
				}
			}
			state.Chunks = state.Chunks.Advance(remoteID)
			continue
		}

		sent, sendErr := sendTeamsDeliveryActivity(
			ctx,
			api,
			ref.ServiceURL,
			ref.ConversationID,
			continuationReplyToID,
			teamsOutboundActivity{
				Type:       providerMessageKey,
				Text:       chunk,
				TextFormat: teamsTextFormatMarkdown,
			},
		)
		if sendErr != nil {
			return bridgepkg.DeliveryAck{}, state, fmt.Errorf("teams: send continuation %d: %w", index, sendErr)
		}
		if sent == nil || strings.TrimSpace(sent.ID) == "" {
			return bridgepkg.DeliveryAck{}, state, &bridgesdk.TransientError{
				Err: errors.New("teams: send activity response omitted id"),
			}
		}
		state.Chunks = state.Chunks.Advance(encodeRemoteMessageID(teamsRemoteMessageRef{
			ConversationID: ref.ConversationID,
			ServiceURL:     ref.ServiceURL,
			ActivityID:     strings.TrimSpace(sent.ID),
		}))
	}

	lastRemoteID := state.Chunks.LastRemoteMessageID()
	replaceRemoteID = state.Chunks.ReplaceRemoteMessageID()
	state.Chunks = bridgesdk.DeliveryChunkCursor{}
	ack := newTeamsDeliveryAck(event, lastRemoteID, replaceRemoteID)
	state.LastSeq = event.Seq
	state.RemoteMessageID = lastRemoteID
	state.ReplaceRemoteMessageID = replaceRemoteID
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
