package core

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/session"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/workref"
	"github.com/gin-gonic/gin"
)

const (
	sessionStreamErrorKey = "error"
)

func parseLastEventID(lastEventID string, transportName string) (int64, error) {
	trimmed := strings.TrimSpace(lastEventID)
	if trimmed == "" {
		return 0, nil
	}

	after, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid Last-Event-ID %q: %w", transportName, trimmed, err)
	}
	return after, nil
}

func (h *BaseHandlers) writeSessionEventBatch(
	writer FlushWriter,
	events []store.SessionEvent,
	info *session.Info,
) (int64, error) {
	var afterSequence int64
	for _, event := range events {
		afterSequence = event.Sequence
		if err := WriteSSE(writer, SSEMessage{
			ID:   strconv.FormatInt(event.Sequence, 10),
			Name: event.Type,
			Data: SessionEventPayloadFromEvent(event, info),
		}); err != nil {
			return afterSequence, err
		}
	}
	return afterSequence, nil
}

func (h *BaseHandlers) writeSessionStoppedEvent(writer FlushWriter, latest *session.Info) error {
	if latest == nil || latest.State != session.StateStopped {
		return nil
	}

	ref := workref.NewPath(latest.WorkspaceID, latest.Workspace)
	return WriteSSE(writer, SSEMessage{
		Name: session.EventTypeSessionStopped,
		Data: contract.SessionEventPayload{
			SessionID:     latest.ID,
			Type:          session.EventTypeSessionStopped,
			WorkspaceID:   ref.WorkspaceID,
			WorkspacePath: ref.WorkspacePath,
			StopReason:    latest.StopReason,
			StopDetail:    latest.StopDetail,
			Failure:       SessionFailurePayloadFromStore(latest.Failure),
			Timestamp:     latest.UpdatedAt,
		},
	})
}

func (h *BaseHandlers) pollAndStreamSessionEvents(
	c *gin.Context,
	writer FlushWriter,
	sessionID string,
	info *session.Info,
	pollQuery store.EventQuery,
	afterSequence int64,
) {
	ticker := time.NewTicker(h.PollInterval)
	defer ticker.Stop()

	currentInfo := info
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-h.StreamDoneChannel():
			return
		case <-ticker.C:
			var done bool
			afterSequence, currentInfo, done = h.pollSessionStreamTick(
				c,
				writer,
				sessionID,
				currentInfo,
				pollQuery,
				afterSequence,
			)
			if done {
				return
			}
		}
	}
}

func (h *BaseHandlers) pollSessionStreamTick(
	c *gin.Context,
	writer FlushWriter,
	sessionID string,
	info *session.Info,
	pollQuery store.EventQuery,
	afterSequence int64,
) (int64, *session.Info, bool) {
	pollQuery.AfterSequence = afterSequence

	events, pollErr := h.Sessions.Events(c.Request.Context(), sessionID, pollQuery)
	if pollErr != nil {
		// Best-effort notification; the SSE client may already be disconnected.
		h.writeSSEBestEffort(writer, SSEMessage{
			Name: sessionStreamErrorKey,
			Data: ErrorPayloadForError(pollErr),
		})
		return afterSequence, info, true
	}

	nextSequence, err := h.writeSessionEventBatch(writer, events, info)
	if err != nil {
		return nextSequence, info, true
	}
	if nextSequence > afterSequence {
		return nextSequence, info, false
	}

	latest, statusErr := h.Sessions.Status(c.Request.Context(), sessionID)
	if statusErr != nil {
		// Best-effort notification; the SSE client may already be disconnected.
		h.writeSSEBestEffort(writer, SSEMessage{
			Name: sessionStreamErrorKey,
			Data: ErrorPayloadForError(statusErr),
		})
		return afterSequence, info, true
	}
	if latest != nil && latest.State == session.StateStopped {
		// Best-effort terminal event; there is nothing else to do if the stream is closed.
		h.logSSEWriteFailure("session_stopped", h.writeSessionStoppedEvent(writer, latest))
		return afterSequence, latest, true
	}
	if h.IncludeSessionWorkspaceInSSE {
		info = latest
	}

	return afterSequence, info, false
}
