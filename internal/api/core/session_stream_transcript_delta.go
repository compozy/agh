package core

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/session"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/transcript"
)

func (h *BaseHandlers) writeTranscriptDeltasForEvents(
	ctx context.Context,
	writer FlushWriter,
	sessionID string,
	info *session.Info,
	events []store.SessionEvent,
	afterSequence int64,
) (int64, error) {
	nextSequence := maxSessionEventSequence(events)
	if nextSequence <= afterSequence {
		return afterSequence, nil
	}
	startedAt := time.Now()
	entries, err := h.Sessions.Transcript(ctx, sessionID, store.EventQuery{AfterSequence: afterSequence})
	h.logTranscriptAssembly(
		ctx,
		sessionID,
		info,
		"delta",
		afterSequence,
		len(events),
		len(entries),
		time.Since(startedAt),
		err,
	)
	if err != nil {
		return afterSequence, fmt.Errorf("query transcript deltas: %w", err)
	}
	emittedSequence, err := h.writeTranscriptDeltaEntries(writer, sessionID, info, entries, afterSequence)
	if err != nil {
		return afterSequence, err
	}
	if emittedSequence > nextSequence {
		return emittedSequence, nil
	}
	return nextSequence, nil
}

func (h *BaseHandlers) writeTranscriptDeltaEntries(
	writer FlushWriter,
	sessionID string,
	info *session.Info,
	entries []transcript.Entry,
	afterSequence int64,
) (int64, error) {
	workspaceID, workspacePath := streamWorkspaceFields(info)
	emittedSequence := afterSequence
	for _, entry := range entries {
		if entry.Sequence <= afterSequence {
			continue
		}
		payload := contract.TranscriptDeltaPayload{
			SessionID:     sessionID,
			WorkspaceID:   workspaceID,
			WorkspacePath: workspacePath,
			Epoch:         transcriptEpoch(info),
			Entry:         entry,
			Sequence:      entry.Sequence,
		}
		if err := WriteSSE(writer, SSEMessage{
			ID:   strconv.FormatInt(entry.Sequence, 10),
			Name: contract.SessionStreamEventTranscriptDelta,
			Data: payload,
		}); err != nil {
			return afterSequence, err
		}
		if entry.Sequence > emittedSequence {
			emittedSequence = entry.Sequence
		}
	}
	return emittedSequence, nil
}
