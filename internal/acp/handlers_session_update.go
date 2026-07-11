package acp

import (
	"encoding/json"
	"strings"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/compozy/agh/internal/store"
)

func translateSessionUpdate(
	notification acpsdk.SessionNotification,
	rawUpdate json.RawMessage,
	turnID string,
) AgentEvent {
	event := AgentEvent{
		SessionID: string(notification.SessionId),
		TurnID:    turnID,
		Timestamp: timeNowUTC(),
		Raw:       CloneRawMessage(rawUpdate),
	}

	switch {
	case notification.Update.UserMessageChunk != nil:
		event.Type = EventTypeUserMessage
		event.Text = extractContentText(notification.Update.UserMessageChunk.Content)
	case notification.Update.AgentMessageChunk != nil:
		event.Type = EventTypeAgentMessage
		event.Text = extractContentText(notification.Update.AgentMessageChunk.Content)
	case notification.Update.AgentThoughtChunk != nil:
		event.Type = EventTypeThought
		event.Text = extractContentText(notification.Update.AgentThoughtChunk.Content)
	case notification.Update.ToolCall != nil:
		toolCall := notification.Update.ToolCall
		event.Type = EventTypeToolCall
		event.Title = toolCall.Title
		event.ToolCallID = string(toolCall.ToolCallId)
	case notification.Update.ToolCallUpdate != nil:
		translateToolCallUpdate(&event, notification.Update.ToolCallUpdate)
	case notification.Update.Plan != nil:
		event.Type = EventTypePlan
	case notification.Update.AvailableCommandsUpdate != nil:
		event.Type = EventTypeAvailableCommands
		event.Title = SystemEventTitleAvailableCommandsUpdate
		event.AvailableCommands = NewAvailableCommandSet(availableCommandNames(
			notification.Update.AvailableCommandsUpdate.AvailableCommands,
		))
	case notification.Update.CurrentModeUpdate != nil:
		event.Type = EventTypeSystem
		event.Title = "current_mode_update"
	case notification.Update.ConfigOptionUpdate != nil:
		event.Type = EventTypeSystem
		event.Title = sessionUpdateConfigOption
	default:
		event.Type = EventTypeSystem
	}

	return event
}

func translateToolCallUpdate(event *AgentEvent, update *acpsdk.SessionToolCallUpdate) {
	event.ToolCallID = string(update.ToolCallId)
	if update.Title != nil {
		event.Title = *update.Title
	}
	if update.Status != nil &&
		(*update.Status == acpsdk.ToolCallStatusCompleted || *update.Status == acpsdk.ToolCallStatusFailed) {
		event.Type = EventTypeToolResult
		return
	}
	event.Type = EventTypeToolCall
}

func availableCommandNames(commands []acpsdk.AvailableCommand) []store.SessionAdvertisedCommand {
	result := make([]store.SessionAdvertisedCommand, 0, len(commands))
	for _, command := range commands {
		name := strings.TrimSpace(command.Name)
		if name == "" {
			continue
		}
		next := store.SessionAdvertisedCommand{
			Name:        name,
			Description: strings.TrimSpace(command.Description),
		}
		if command.Input != nil && command.Input.Unstructured != nil {
			next.Input = &store.SessionAdvertisedCommandInput{
				Hint: strings.TrimSpace(command.Input.Unstructured.Hint),
			}
		}
		result = append(result, next)
	}
	return result
}
