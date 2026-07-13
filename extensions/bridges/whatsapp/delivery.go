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

const whatsappMaxMessageRunes = 4_096

type whatsappSendMessageRequest struct {
	MessagingProduct string `json:"messaging_product"`
	RecipientType    string `json:"recipient_type,omitempty"`
	To               string `json:"to"`
	Type             string `json:"type"`
	Text             struct {
		Body       string `json:"body"`
		PreviewURL bool   `json:"preview_url"`
	} `json:"text"`
}

type whatsappSendMessageResponse struct {
	Messages []struct {
		ID string `json:"id,omitempty"`
	} `json:"messages,omitempty"`
}

type deliveryState struct {
	LastSeq                int64
	RemoteMessageID        string
	ReplaceRemoteMessageID string
	Progress               *bridgesdk.ProgressDispatcher
}

func (p *whatsappProvider) handleBridgesDeliver(
	ctx context.Context,
	session *bridgesdk.Session,
	request bridgepkg.DeliveryRequest,
) (bridgepkg.DeliveryAck, error) {
	marker := bridgesdk.DeliveryMarker{
		PID:     os.Getpid(),
		Request: request,
	}

	cfg, err := p.waitForInstanceConfig(
		strings.TrimSpace(request.Event.BridgeInstanceID),
		500*time.Millisecond,
	)
	if err != nil {
		marker.Error = err.Error()
		p.markers.RecordDelivery(marker)
		p.setInstanceError(request.Event.BridgeInstanceID, err)
		return bridgepkg.DeliveryAck{}, err
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
			return p.failWhatsAppDelivery(ctx, session, cfg, marker, err)
		}
	}

	api := p.apiFactory(cfg)
	ack, state, err := executeWhatsAppDelivery(
		ctx,
		api,
		cfg,
		request,
		state,
	)
	if err != nil {
		return p.failWhatsAppDelivery(ctx, session, cfg, marker, err)
	}
	return p.completeWhatsAppDelivery(ctx, session, cfg, request, marker, ack, state)
}

func (p *whatsappProvider) completeWhatsAppDelivery(
	ctx context.Context,
	session *bridgesdk.Session,
	cfg resolvedInstanceConfig,
	request bridgepkg.DeliveryRequest,
	marker bridgesdk.DeliveryMarker,
	ack bridgepkg.DeliveryAck,
	state deliveryState,
) (bridgepkg.DeliveryAck, error) {
	dispatcher := state.Progress
	var cleanupErr error
	if dispatcher != nil {
		cleanupErr = dispatcher.OnContent(ctx)
		if isTerminalWhatsAppDeliveryEvent(request.Event) {
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
		p.setInstanceError(cfg.instanceID, readyErr)
	case cleanupErr != nil:
		p.recordWhatsAppProgressCleanupError(cfg.instanceID, cleanupErr)
	default:
		p.clearInstanceError(cfg.instanceID)
	}

	marker.Ack = &ack
	p.markers.RecordDelivery(marker)
	return ack, nil
}

func (p *whatsappProvider) failWhatsAppDelivery(
	ctx context.Context,
	session *bridgesdk.Session,
	cfg resolvedInstanceConfig,
	marker bridgesdk.DeliveryMarker,
	err error,
) (bridgepkg.DeliveryAck, error) {
	marker.Error = err.Error()
	p.markers.RecordDelivery(marker)
	classified := bridgesdk.ClassifyError(err)
	_, _, reportErr := session.ReportClassifiedError(ctx, cfg.instanceID, classified)
	if reportErr != nil {
		p.setInstanceError(cfg.instanceID, reportErr)
	} else {
		p.setInstanceError(cfg.instanceID, err)
	}
	return bridgepkg.DeliveryAck{}, err
}

func isTerminalWhatsAppDeliveryEvent(event bridgepkg.DeliveryEvent) bool {
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

func (p *whatsappProvider) deliveryState(instanceID string, deliveryID string) deliveryState {
	state, _ := p.deliveries.Load(deliveryStateKey(instanceID, deliveryID))
	return state
}

func (p *whatsappProvider) storeDeliveryState(
	instanceID string,
	deliveryID string,
	state deliveryState,
) {
	p.deliveries.Store(deliveryStateKey(instanceID, deliveryID), state)
}

func executeWhatsAppDelivery(
	ctx context.Context,
	api whatsappAPI,
	cfg resolvedInstanceConfig,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	if err := request.Validate(); err != nil {
		return bridgepkg.DeliveryAck{}, state, fmt.Errorf("whatsapp: validate delivery request: %w", err)
	}

	event := request.Event
	if event.EventType != bridgepkg.DeliveryEventTypeResume && event.Seq <= state.LastSeq {
		return bridgepkg.DeliveryAck{}, state, fmt.Errorf(
			"whatsapp: out-of-order delivery seq %d after %d",
			event.Seq,
			state.LastSeq,
		)
	}
	if event.EventType == bridgepkg.DeliveryEventTypeResume && request.Snapshot != nil {
		state.LastSeq = request.Snapshot.LastAckedSeq
		state.RemoteMessageID = strings.TrimSpace(request.Snapshot.RemoteMessageID)
		state.ReplaceRemoteMessageID = strings.TrimSpace(request.Snapshot.ReplaceRemoteMessageID)
	}

	if isWhatsAppDeleteDelivery(event) {
		return bridgepkg.DeliveryAck{}, state, &bridgesdk.PermanentError{
			Err: errors.New("whatsapp: delete delivery is not supported by the Cloud API"),
		}
	}

	targetUserID, err := resolveDeliveryTarget(event)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, fmt.Errorf("whatsapp: resolve delivery target: %w", err)
	}
	text := event.Content.Text
	if strings.TrimSpace(text) == "" {
		return bridgepkg.DeliveryAck{}, state, &bridgesdk.PermanentError{
			Err: errors.New("whatsapp: text delivery content is required"),
		}
	}

	if ack, resumed, ok := resumeWhatsAppDelivery(event, request.Snapshot, state); ok {
		if err := ack.ValidateFor(event); err != nil {
			return bridgepkg.DeliveryAck{}, state, fmt.Errorf("whatsapp: validate resume ack: %w", err)
		}
		return ack, resumed, nil
	}

	remoteID, err := sendWhatsAppDeliveryChunks(ctx, api, cfg.phoneNumberID, targetUserID, text)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	replaceRemoteID := whatsappReplacementRemoteID(event, request.Snapshot, state)
	ack := bridgepkg.DeliveryAck{
		DeliveryID:      event.DeliveryID,
		Seq:             event.Seq,
		RemoteMessageID: remoteID,
	}
	if event.Seq > 1 || replaceRemoteID != "" {
		ack.ReplaceRemoteMessageID = replaceRemoteID
	}
	state.LastSeq = event.Seq
	state.ReplaceRemoteMessageID = replaceRemoteID
	state.RemoteMessageID = remoteID
	if err := ack.ValidateFor(event); err != nil {
		return bridgepkg.DeliveryAck{}, state, fmt.Errorf("whatsapp: validate delivery ack: %w", err)
	}
	return ack, state, nil
}

func whatsappReplacementRemoteID(
	event bridgepkg.DeliveryEvent,
	snapshot *bridgepkg.DeliverySnapshot,
	state deliveryState,
) string {
	referenceRemoteID := ""
	if event.Reference != nil {
		referenceRemoteID = strings.TrimSpace(event.Reference.RemoteMessageID)
	}
	snapshotRemoteID := ""
	if snapshot != nil {
		snapshotRemoteID = strings.TrimSpace(snapshot.RemoteMessageID)
	}
	return firstNonEmpty(referenceRemoteID, state.RemoteMessageID, snapshotRemoteID)
}

func isWhatsAppDeleteDelivery(event bridgepkg.DeliveryEvent) bool {
	return event.Operation.Normalize() == bridgepkg.DeliveryOperationDelete ||
		normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeDelete
}

func resumeWhatsAppDelivery(
	event bridgepkg.DeliveryEvent,
	snapshot *bridgepkg.DeliverySnapshot,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, bool) {
	if normalizeDeliveryEventType(event.EventType) != bridgepkg.DeliveryEventTypeResume ||
		snapshot == nil {
		return bridgepkg.DeliveryAck{}, state, false
	}
	remoteMessageID := strings.TrimSpace(snapshot.RemoteMessageID)
	if remoteMessageID == "" {
		return bridgepkg.DeliveryAck{}, state, false
	}

	ack := bridgepkg.DeliveryAck{
		DeliveryID:             event.DeliveryID,
		Seq:                    event.Seq,
		RemoteMessageID:        remoteMessageID,
		ReplaceRemoteMessageID: strings.TrimSpace(snapshot.ReplaceRemoteMessageID),
	}
	state.LastSeq = event.Seq
	state.RemoteMessageID = ack.RemoteMessageID
	state.ReplaceRemoteMessageID = ack.ReplaceRemoteMessageID
	return ack, state, true
}

func sendWhatsAppDeliveryChunks(
	ctx context.Context,
	api whatsappAPI,
	phoneNumberID string,
	targetUserID string,
	text string,
) (string, error) {
	remoteID := ""
	chunks := bridgesdk.ChunkMessage(text, whatsappMaxMessageRunes, nil)
	for index, chunk := range chunks {
		req := whatsappSendMessageRequest{
			MessagingProduct: providerWhatsappKey,
			RecipientType:    whatsappRecipientTypeIndividual,
			To:               targetUserID,
			Type:             providerTextKey,
		}
		req.Text.Body = chunk
		req.Text.PreviewURL = false

		response, err := api.SendTextMessage(ctx, phoneNumberID, req)
		if err != nil {
			return "", fmt.Errorf("whatsapp: send text chunk %d: %w", index+1, err)
		}
		remoteID, err = lastWhatsAppResponseMessageID(response)
		if err != nil {
			return "", err
		}
	}
	return remoteID, nil
}

func lastWhatsAppResponseMessageID(response *whatsappSendMessageResponse) (string, error) {
	if response == nil || len(response.Messages) == 0 {
		return "", &bridgesdk.TransientError{
			Err: errors.New("whatsapp: send message response omitted a message id"),
		}
	}
	messageID := strings.TrimSpace(response.Messages[len(response.Messages)-1].ID)
	if messageID == "" {
		return "", &bridgesdk.TransientError{
			Err: errors.New("whatsapp: send message response omitted a message id"),
		}
	}
	return messageID, nil
}
