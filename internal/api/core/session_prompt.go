package core

import (
	"strings"

	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/session"
)

// PromptResultPayloadFromSession converts runtime prompt admission state into the shared DTO.
func PromptResultPayloadFromSession(result session.SendPromptResult) contract.SendPromptResultPayload {
	payload := contract.SendPromptResultPayload{
		Status:                strings.TrimSpace(result.Status),
		Mode:                  contract.PromptMode(strings.TrimSpace(string(result.Mode))),
		Queued:                result.Queued,
		Staged:                result.Staged,
		Interrupted:           result.Interrupted,
		QueueEntryID:          strings.TrimSpace(result.QueueEntryID),
		QueuePosition:         result.QueuePosition,
		QueueGeneration:       result.QueueGeneration,
		EstimatedSendAt:       result.EstimatedSendAt,
		PreviousTurnID:        strings.TrimSpace(result.PreviousTurnID),
		NewTurnID:             strings.TrimSpace(result.NewTurnID),
		CanceledQueuedEntries: result.CanceledQueuedEntries,
	}
	if fallbackMode := strings.TrimSpace(result.FallbackModeIfNoToolResult); fallbackMode != "" {
		payload.FallbackModeIfNoToolResult = contract.PromptMode(fallbackMode)
	}
	return payload
}
