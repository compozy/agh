package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	bridgepkg "github.com/compozy/agh/internal/bridges"
	"github.com/compozy/agh/internal/bridgesdk"
)

const (
	telegramMaxMessageLen          = 4_096
	telegramChunkEscapeHeadroom    = 2
	telegramChunkIndicatorOpen     = "\n\n("
	telegramEscapedIndicatorPrefix = "\n\n\\("
)

type deliveryState struct {
	LastSeq                int64
	LastContent            string
	RemoteMessageID        string
	ReplaceRemoteMessageID string
	PreviewOnly            bool
	Progress               *bridgesdk.ProgressDispatcher
}

type telegramMarkdownParseError struct {
	err error
}

func (e *telegramMarkdownParseError) Error() string {
	return e.err.Error()
}

func (e *telegramMarkdownParseError) Unwrap() error {
	return e.err
}

func executeDelivery(
	ctx context.Context,
	api telegramAPI,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	if err := request.Validate(); err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}

	event := request.Event
	if event.EventType != bridgepkg.DeliveryEventTypeResume && event.Seq <= state.LastSeq {
		return bridgepkg.DeliveryAck{}, state, fmt.Errorf(
			"telegram: out-of-order delivery seq %d after %d",
			event.Seq,
			state.LastSeq,
		)
	}
	state = applyTelegramDeliverySnapshot(state, event, request.Snapshot)

	targetChatID, targetThreadID, err := resolveDeliveryTarget(event)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}

	switch {
	case isTelegramDeleteDelivery(event):
		return executeTelegramDelete(ctx, api, event, request.Snapshot, state)
	case shouldPostNewMessage(event, state, request):
		return executeTelegramPost(ctx, api, event, targetChatID, targetThreadID, state)
	default:
		return executeTelegramEdit(ctx, api, event, request.Snapshot, state, targetThreadID)
	}
}

func applyTelegramDeliverySnapshot(
	state deliveryState,
	event bridgepkg.DeliveryEvent,
	snapshot *bridgepkg.DeliverySnapshot,
) deliveryState {
	if event.EventType != bridgepkg.DeliveryEventTypeResume || snapshot == nil {
		return state
	}
	state.LastSeq = snapshot.LastAckedSeq
	if snapshot.LastAckedSeq >= snapshot.LatestSeq {
		state.LastContent = snapshot.CurrentContent.Text
		state.PreviewOnly = !snapshot.Final && telegramContentNeedsChunks(snapshot.CurrentContent.Text)
	} else {
		state.LastContent = ""
		state.PreviewOnly = false
	}
	state.RemoteMessageID = strings.TrimSpace(snapshot.RemoteMessageID)
	state.ReplaceRemoteMessageID = strings.TrimSpace(snapshot.ReplaceRemoteMessageID)
	return state
}

func isTelegramDeleteDelivery(event bridgepkg.DeliveryEvent) bool {
	return event.Operation.Normalize() == bridgepkg.DeliveryOperationDelete ||
		normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeDelete
}

func isTerminalTelegramDeliveryEvent(event bridgepkg.DeliveryEvent) bool {
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

func executeTelegramDelete(
	ctx context.Context,
	api telegramAPI,
	event bridgepkg.DeliveryEvent,
	snapshot *bridgepkg.DeliverySnapshot,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	remoteID := firstNonEmpty(referenceRemoteMessageID(event.Reference), state.RemoteMessageID)
	if remoteID == "" && snapshot != nil {
		remoteID = strings.TrimSpace(snapshot.RemoteMessageID)
	}
	if remoteID == "" {
		return bridgepkg.DeliveryAck{}, state, errors.New(
			"telegram: delete delivery requires a remote message id",
		)
	}

	chatID, messageID, err := decodeRemoteMessageID(remoteID)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	if err := api.DeleteMessage(ctx, telegramDeleteMessageRequest{
		ChatID:    chatID,
		MessageID: messageID,
	}); err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}

	ack := newTelegramDeliveryAck(event, remoteID, firstNonEmpty(state.RemoteMessageID, remoteID))
	state.LastSeq = event.Seq
	state.LastContent = ""
	state.RemoteMessageID = remoteID
	state.ReplaceRemoteMessageID = ack.ReplaceRemoteMessageID
	state.PreviewOnly = false
	return ack, state, ack.ValidateFor(event)
}

func executeTelegramPost(
	ctx context.Context,
	api telegramAPI,
	event bridgepkg.DeliveryEvent,
	targetChatID string,
	targetThreadID string,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	messageThreadID, err := resolveTelegramThreadID(targetThreadID, targetChatID)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	chunks, previewOnly := telegramDeliveryChunks(event)
	lastRemoteID := ""
	for _, chunk := range chunks {
		sent, err := sendTelegramOutbound(ctx, api, telegramSendMessageRequest{
			ChatID:          targetChatID,
			MessageThreadID: messageThreadID,
		}, chunk)
		if err != nil {
			return bridgepkg.DeliveryAck{}, state, err
		}
		lastRemoteID = encodeRemoteMessageID(targetChatID, sent.MessageID)
	}

	ack := newTelegramDeliveryAck(event, lastRemoteID, state.RemoteMessageID)
	state.LastSeq = event.Seq
	state.LastContent = event.Content.Text
	state.ReplaceRemoteMessageID = state.RemoteMessageID
	state.RemoteMessageID = lastRemoteID
	state.PreviewOnly = previewOnly
	return ack, state, ack.ValidateFor(event)
}

func executeTelegramEdit(
	ctx context.Context,
	api telegramAPI,
	event bridgepkg.DeliveryEvent,
	snapshot *bridgepkg.DeliverySnapshot,
	state deliveryState,
	targetThreadID string,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	remoteID := referenceRemoteMessageID(event.Reference)
	if remoteID == "" {
		remoteID = state.RemoteMessageID
	}
	if remoteID == "" && snapshot != nil {
		remoteID = strings.TrimSpace(snapshot.RemoteMessageID)
	}
	if remoteID == "" {
		return bridgepkg.DeliveryAck{}, state, errors.New(
			"telegram: edit delivery requires a remote message id",
		)
	}

	ack := newTelegramDeliveryAck(event, remoteID, firstNonEmpty(state.RemoteMessageID, remoteID))
	if event.Content.Text == state.LastContent && !state.PreviewOnly {
		state.LastSeq = event.Seq
		state.RemoteMessageID = remoteID
		state.ReplaceRemoteMessageID = ack.ReplaceRemoteMessageID
		return ack, state, ack.ValidateFor(event)
	}

	chatID, messageID, err := decodeRemoteMessageID(remoteID)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	messageThreadID, err := resolveTelegramThreadID(targetThreadID, chatID)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	chunks, previewOnly := telegramDeliveryChunks(event)
	if err := editTelegramOutbound(ctx, api, telegramEditMessageTextRequest{
		ChatID:    chatID,
		MessageID: messageID,
	}, chunks[0]); err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}

	lastRemoteID := remoteID
	for _, chunk := range chunks[1:] {
		sent, err := sendTelegramOutbound(ctx, api, telegramSendMessageRequest{
			ChatID:          chatID,
			MessageThreadID: messageThreadID,
		}, chunk)
		if err != nil {
			return bridgepkg.DeliveryAck{}, state, err
		}
		lastRemoteID = encodeRemoteMessageID(chatID, sent.MessageID)
	}

	ack = newTelegramDeliveryAck(event, lastRemoteID, firstNonEmpty(state.RemoteMessageID, remoteID))
	state.LastSeq = event.Seq
	state.LastContent = event.Content.Text
	state.RemoteMessageID = lastRemoteID
	state.ReplaceRemoteMessageID = ack.ReplaceRemoteMessageID
	state.PreviewOnly = previewOnly
	return ack, state, ack.ValidateFor(event)
}

func sendTelegramOutbound(
	ctx context.Context,
	api telegramAPI,
	request telegramSendMessageRequest,
	outbound telegramOutboundText,
) (*telegramSentMessage, error) {
	request.Text = outbound.Text
	request.ParseMode = outbound.ParseMode
	sent, err := api.SendMessage(ctx, request)
	if err == nil {
		return validatedTelegramSentMessage(sent)
	}
	var parseErr *telegramMarkdownParseError
	if outbound.ParseMode == "" || !errors.As(err, &parseErr) {
		return nil, fmt.Errorf("telegram: send outbound text: %w", err)
	}
	request.Text = outbound.PlainText
	request.ParseMode = ""
	sent, err = api.SendMessage(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("telegram: send plain-text fallback: %w", err)
	}
	return validatedTelegramSentMessage(sent)
}

func validatedTelegramSentMessage(message *telegramSentMessage) (*telegramSentMessage, error) {
	if message == nil {
		return nil, &bridgesdk.TransientError{
			Err: errors.New("telegram: send message returned no response"),
		}
	}
	if message.MessageID <= 0 {
		return nil, &bridgesdk.TransientError{
			Err: fmt.Errorf("telegram: send message returned invalid message id %d", message.MessageID),
		}
	}
	return message, nil
}

func editTelegramOutbound(
	ctx context.Context,
	api telegramAPI,
	request telegramEditMessageTextRequest,
	outbound telegramOutboundText,
) error {
	request.Text = outbound.Text
	request.ParseMode = outbound.ParseMode
	err := api.EditMessageText(ctx, request)
	if err == nil {
		return nil
	}
	var parseErr *telegramMarkdownParseError
	if outbound.ParseMode == "" || !errors.As(err, &parseErr) {
		return fmt.Errorf("telegram: edit outbound text: %w", err)
	}
	request.Text = outbound.PlainText
	request.ParseMode = ""
	if err := api.EditMessageText(ctx, request); err != nil {
		return fmt.Errorf("telegram: edit plain-text fallback: %w", err)
	}
	return nil
}

func telegramDeliveryChunks(event bridgepkg.DeliveryEvent) ([]telegramOutboundText, bool) {
	chunks := telegramOutboundChunks(event.Content.Text)
	previewOnly := !materializeAllTelegramChunks(event) && len(chunks) > 1
	if previewOnly {
		return chunks[:1], true
	}
	return chunks, false
}

func telegramOutboundChunks(content string) []telegramOutboundText {
	formatted := formatOutbound(content)
	if formatted.ParseMode == "" {
		return plainTelegramChunks(content)
	}
	if bridgesdk.UTF16Len(formatted.Text) <= telegramMaxMessageLen {
		return []telegramOutboundText{formatted}
	}
	if telegramContainsInlineCode(content) {
		return plainTelegramChunks(content)
	}

	chunks := bridgesdk.ChunkMessage(
		formatted.Text,
		telegramMaxMessageLen-telegramChunkEscapeHeadroom,
		bridgesdk.UTF16Len,
	)
	result := make([]telegramOutboundText, len(chunks))
	for index, chunk := range chunks {
		chunk = escapeTelegramChunkIndicator(chunk)
		result[index] = telegramOutboundText{
			Text:      chunk,
			PlainText: stripTelegramMarkdownV2(chunk),
			ParseMode: telegramMarkdownV2,
		}
	}
	return result
}

func plainTelegramChunks(content string) []telegramOutboundText {
	chunks := bridgesdk.ChunkMessage(content, telegramMaxMessageLen, bridgesdk.UTF16Len)
	result := make([]telegramOutboundText, len(chunks))
	for index, chunk := range chunks {
		result[index] = telegramOutboundText{Text: chunk, PlainText: chunk}
	}
	return result
}

func telegramContentNeedsChunks(content string) bool {
	return len(telegramOutboundChunks(content)) > 1
}

func materializeAllTelegramChunks(event bridgepkg.DeliveryEvent) bool {
	return event.Final
}

func telegramContainsInlineCode(value string) bool {
	inFence := false
	for index := 0; index < len(value); index++ {
		if value[index] == '\\' {
			index++
			continue
		}
		if strings.HasPrefix(value[index:], "```") {
			inFence = !inFence
			index += 2
			continue
		}
		if !inFence && value[index] == '`' {
			return true
		}
	}
	return false
}

func escapeTelegramChunkIndicator(value string) string {
	start := strings.LastIndex(value, telegramChunkIndicatorOpen)
	if start < 0 || !strings.HasSuffix(value, ")") {
		return value
	}
	indicator := value[start+len(telegramChunkIndicatorOpen) : len(value)-1]
	return value[:start] + telegramEscapedIndicatorPrefix + indicator + "\\)"
}

func stripTelegramMarkdownV2(value string) string {
	var plain strings.Builder
	plain.Grow(len(value))
	inFence := false
	inInlineCode := false
	for index := 0; index < len(value); index++ {
		current := value[index]
		if current == '\\' && index+1 < len(value) {
			index++
			plain.WriteByte(value[index])
			continue
		}
		if !inInlineCode && strings.HasPrefix(value[index:], "```") {
			plain.WriteString("```")
			inFence = !inFence
			index += 2
			continue
		}
		if !inFence && current == '`' {
			plain.WriteByte(current)
			inInlineCode = !inInlineCode
			continue
		}
		switch current {
		case '*', '_', '~':
			if !inFence && !inInlineCode {
				continue
			}
			plain.WriteByte(current)
		default:
			plain.WriteByte(current)
		}
	}
	return plain.String()
}

func telegramMarkdownParseFailure(message string) bool {
	normalized := strings.ToLower(strings.TrimSpace(message))
	return strings.Contains(normalized, "can't parse entities") ||
		strings.Contains(normalized, "cant parse entities") ||
		strings.Contains(normalized, "can't find end of") ||
		strings.Contains(normalized, "character '") && strings.Contains(normalized, "is reserved")
}

func newTelegramDeliveryAck(
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

func shouldPostNewMessage(
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
