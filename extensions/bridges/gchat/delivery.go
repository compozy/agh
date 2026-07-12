package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	bridgepkg "github.com/compozy/agh/internal/bridges"
	"github.com/compozy/agh/internal/bridgesdk"
)

const gchatMaxMessageBytes = 32_000

type gchatResolvedTarget struct {
	SpaceName  string
	ThreadName string
}

type deliveryState struct {
	LastSeq                int64
	RemoteMessageID        string
	ReplaceRemoteMessageID string
	Progress               *bridgesdk.ProgressDispatcher
}

func (p *gchatProvider) handleBridgesDeliver(
	ctx context.Context,
	session *bridgesdk.Session,
	request bridgepkg.DeliveryRequest,
) (bridgepkg.DeliveryAck, error) {
	marker := deliveryMarker{
		PID:     os.Getpid(),
		Request: request,
	}

	cfg, err := p.waitForInstanceConfig(strings.TrimSpace(request.Event.BridgeInstanceID), 500*time.Millisecond)
	if err != nil {
		marker.Error = err.Error()
		p.reportSideEffectError("write failed delivery marker", appendJSONLine(p.env.deliveryPath, marker))
		p.setLastError(err)
		return bridgepkg.DeliveryAck{}, err
	}
	if cfg.configError != nil {
		err = cfg.configError
		marker.Error = err.Error()
		p.reportSideEffectError("write failed delivery marker", appendJSONLine(p.env.deliveryPath, marker))
		p.setLastError(err)
		return bridgepkg.DeliveryAck{}, err
	}

	if shouldCrashOnce(p.env.crashOncePath) {
		p.reportSideEffectError("write pre-crash delivery marker", appendJSONLine(p.env.deliveryPath, marker))
		p.reportSideEffectError("write crash marker", writeJSONFile(p.env.crashOncePath, map[string]any{
			"crashed":            true,
			"pid":                os.Getpid(),
			"delivery_id":        strings.TrimSpace(request.Event.DeliveryID),
			"bridge_instance_id": cfg.instanceID,
		}))
		os.Exit(23)
	}

	state := p.deliveryState(cfg.instanceID, request.Event.DeliveryID)
	if state.Progress != nil {
		if err := state.Progress.Flush(ctx); err != nil {
			return p.failGChatDelivery(ctx, session, &cfg, marker, err)
		}
	}

	ack, state, err := executeGChatDelivery(
		ctx,
		p.apiFactory(&cfg),
		request,
		state,
	)
	if err != nil {
		return p.failGChatDelivery(ctx, session, &cfg, marker, err)
	}

	dispatcher := state.Progress
	var cleanupErr error
	if dispatcher != nil {
		cleanupErr = dispatcher.OnContent(ctx)
		if isTerminalGChatDeliveryEvent(request.Event) {
			state.Progress = nil
		}
	}
	p.storeDeliveryState(cfg.instanceID, request.Event.DeliveryID, request.Event, state)
	if dispatcher != nil && state.Progress == nil {
		dispatcher.Close()
	}
	readyErr := p.reportReadyIfNeeded(ctx, session, cfg.instanceID)
	switch {
	case readyErr != nil:
		p.setLastError(readyErr)
	case cleanupErr != nil:
		p.recordGChatProgressCleanupError(cleanupErr)
	default:
		p.clearLastError()
	}

	marker.Ack = &ack
	p.reportSideEffectError("write delivery marker", appendJSONLine(p.env.deliveryPath, marker))
	return ack, nil
}

func (p *gchatProvider) failGChatDelivery(
	ctx context.Context,
	session *bridgesdk.Session,
	cfg *resolvedInstanceConfig,
	marker deliveryMarker,
	err error,
) (bridgepkg.DeliveryAck, error) {
	marker.Error = err.Error()
	p.reportSideEffectError("write failed delivery marker", appendJSONLine(p.env.deliveryPath, marker))
	classified := bridgesdk.ClassifyError(err)
	_, _, reportErr := session.ReportClassifiedError(ctx, cfg.instanceID, classified)
	if reportErr != nil {
		p.setLastError(reportErr)
	} else {
		p.setLastError(err)
	}
	return bridgepkg.DeliveryAck{}, err
}

func (p *gchatProvider) deliveryState(instanceID string, deliveryID string) deliveryState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.deliveries[deliveryStateKey(instanceID, deliveryID)]
}

func (p *gchatProvider) storeDeliveryState(
	instanceID string,
	deliveryID string,
	event bridgepkg.DeliveryEvent,
	state deliveryState,
) {
	p.mu.Lock()
	defer p.mu.Unlock()
	key := deliveryStateKey(instanceID, deliveryID)
	if isTerminalGChatDeliveryEvent(event) {
		delete(p.deliveries, key)
		return
	}
	p.deliveries[key] = state
}

func isTerminalGChatDeliveryEvent(event bridgepkg.DeliveryEvent) bool {
	if event.Operation.Normalize() == bridgepkg.DeliveryOperationDelete {
		return true
	}
	switch normalizeDeliveryEventType(event.EventType) {
	case bridgepkg.DeliveryEventTypeFinal, bridgepkg.DeliveryEventTypeError, bridgepkg.DeliveryEventTypeDelete:
		return true
	default:
		return false
	}
}

func executeGChatDelivery(
	ctx context.Context,
	api gchatAPI,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	if err := request.Validate(); err != nil {
		return bridgepkg.DeliveryAck{}, state, fmt.Errorf("gchat: validate delivery request: %w", err)
	}

	event := request.Event
	if event.EventType != bridgepkg.DeliveryEventTypeResume && event.Seq <= state.LastSeq {
		return bridgepkg.DeliveryAck{}, state, fmt.Errorf(
			"gchat: out-of-order delivery seq %d after %d",
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
	case isGChatDeleteEvent(event):
		return executeGChatDelete(ctx, api, request, state)
	case event.Operation.Normalize() == bridgepkg.DeliveryOperationEdit:
		return executeGChatUpdate(ctx, api, request, state)
	case shouldPostGChatMessage(event, state, request):
		return executeGChatCreate(ctx, api, event, state)
	default:
		return executeGChatUpdate(ctx, api, request, state)
	}
}

func isGChatDeleteEvent(event bridgepkg.DeliveryEvent) bool {
	return event.Operation.Normalize() == bridgepkg.DeliveryOperationDelete ||
		normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeDelete
}

func executeGChatDelete(
	ctx context.Context,
	api gchatAPI,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	event := request.Event
	messageName := gchatRemoteMessageIDFromRequest(request, state)
	if messageName == "" {
		return bridgepkg.DeliveryAck{}, state, errors.New("gchat: delete delivery requires a remote message id")
	}
	if err := api.DeleteMessage(ctx, messageName); err != nil {
		return bridgepkg.DeliveryAck{}, state, fmt.Errorf("gchat: delete message: %w", err)
	}
	state.LastSeq = event.Seq
	state.RemoteMessageID = messageName
	state.ReplaceRemoteMessageID = messageName
	return validateGChatDeliveryAck(event, state)
}

func executeGChatCreate(
	ctx context.Context,
	api gchatAPI,
	event bridgepkg.DeliveryEvent,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	text, err := gchatDeliveryText(event)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	target, err := resolveGChatDeliveryTarget(event)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, fmt.Errorf("gchat: resolve delivery target: %w", err)
	}

	chunks := gchatDeliveryChunks(text)
	if !event.Final {
		chunks = chunks[:1]
	}
	remoteID, err := postGChatChunks(ctx, api, target, chunks)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	previousRemoteID := strings.TrimSpace(state.RemoteMessageID)
	state.LastSeq = event.Seq
	state.RemoteMessageID = remoteID
	state.ReplaceRemoteMessageID = previousRemoteID
	return validateGChatDeliveryAck(event, state)
}

func executeGChatUpdate(
	ctx context.Context,
	api gchatAPI,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	event := request.Event
	text, err := gchatDeliveryText(event)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	messageName := gchatRemoteMessageIDFromRequest(request, state)
	if messageName == "" {
		return bridgepkg.DeliveryAck{}, state, errors.New("gchat: edit delivery requires a remote message id")
	}

	chunks := gchatDeliveryChunks(text)
	updated, err := api.UpdateMessage(ctx, gchatUpdateMessageRequest{
		MessageName: messageName,
		Text:        chunks[0],
	})
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, fmt.Errorf("gchat: update message: %w", err)
	}
	remoteID := messageName
	if updated != nil && strings.TrimSpace(updated.Name) != "" {
		remoteID = strings.TrimSpace(updated.Name)
	}
	if event.Final && len(chunks) > 1 {
		target, targetErr := resolveGChatDeliveryTarget(event)
		if targetErr != nil {
			return bridgepkg.DeliveryAck{}, state, fmt.Errorf("gchat: resolve continuation target: %w", targetErr)
		}
		remoteID, err = postGChatChunks(ctx, api, target, chunks[1:])
		if err != nil {
			return bridgepkg.DeliveryAck{}, state, err
		}
	}

	state.LastSeq = event.Seq
	state.RemoteMessageID = remoteID
	state.ReplaceRemoteMessageID = messageName
	return validateGChatDeliveryAck(event, state)
}

func postGChatChunks(
	ctx context.Context,
	api gchatAPI,
	target gchatResolvedTarget,
	chunks []string,
) (string, error) {
	remoteID := ""
	for index, chunk := range chunks {
		message, err := api.CreateMessage(ctx, gchatCreateMessageRequest{
			SpaceName:  target.SpaceName,
			ThreadName: target.ThreadName,
			Text:       chunk,
		})
		if err != nil {
			return "", fmt.Errorf("gchat: create message chunk %d: %w", index+1, err)
		}
		if message == nil || strings.TrimSpace(message.Name) == "" {
			return "", &bridgesdk.TransientError{
				Err: errors.New("gchat: create message response omitted name"),
			}
		}
		remoteID = strings.TrimSpace(message.Name)
	}
	return remoteID, nil
}

func validateGChatDeliveryAck(
	event bridgepkg.DeliveryEvent,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	ack := bridgepkg.DeliveryAck{
		DeliveryID:             event.DeliveryID,
		Seq:                    event.Seq,
		RemoteMessageID:        state.RemoteMessageID,
		ReplaceRemoteMessageID: state.ReplaceRemoteMessageID,
	}
	if err := ack.ValidateFor(event); err != nil {
		return bridgepkg.DeliveryAck{}, state, fmt.Errorf("gchat: validate delivery ack: %w", err)
	}
	return ack, state, nil
}

func gchatDeliveryChunks(text string) []string {
	return bridgesdk.ChunkMessage(text, gchatMaxMessageBytes, func(value string) int {
		return len(value)
	})
}

func gchatDeliveryText(event bridgepkg.DeliveryEvent) (string, error) {
	text := event.Content.Text
	if strings.TrimSpace(text) == "" {
		return "", &bridgesdk.PermanentError{Err: errors.New("gchat: text delivery content is required")}
	}
	return text, nil
}

func gchatRemoteMessageIDFromRequest(request bridgepkg.DeliveryRequest, state deliveryState) string {
	messageName := firstNonEmpty(referenceRemoteMessageID(request.Event.Reference), state.RemoteMessageID)
	if messageName == "" && request.Snapshot != nil {
		return strings.TrimSpace(request.Snapshot.RemoteMessageID)
	}
	return messageName
}

func shouldPostGChatMessage(
	event bridgepkg.DeliveryEvent,
	state deliveryState,
	request bridgepkg.DeliveryRequest,
) bool {
	if normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeStart {
		return true
	}
	if normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeResume {
		if request.Snapshot == nil {
			return strings.TrimSpace(state.RemoteMessageID) == ""
		}
		return strings.TrimSpace(request.Snapshot.RemoteMessageID) == ""
	}
	return strings.TrimSpace(state.RemoteMessageID) == ""
}
