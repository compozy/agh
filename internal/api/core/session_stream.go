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

type sessionStreamOptions struct {
	frameMode      string
	replaySnapshot bool
}

func parseSessionStreamOptions(c *gin.Context) (sessionStreamOptions, error) {
	frameMode := strings.TrimSpace(c.Query("frames"))
	if frameMode == "" {
		frameMode = contract.SessionStreamFrameTranscript
	}
	switch frameMode {
	case contract.SessionStreamFrameRaw, contract.SessionStreamFrameTranscript:
	default:
		return sessionStreamOptions{}, fmt.Errorf("invalid frames query: %q", frameMode)
	}

	replay := strings.TrimSpace(c.Query("replay"))
	replaySnapshot := replay == contract.SessionStreamReplaySnapshot
	if replay != "" && !replaySnapshot {
		return sessionStreamOptions{}, fmt.Errorf("invalid replay query: %q", replay)
	}
	if strings.TrimSpace(c.GetHeader("Last-Event-ID")) == "" {
		replaySnapshot = true
	}

	return sessionStreamOptions{
		frameMode:      frameMode,
		replaySnapshot: replaySnapshot,
	}, nil
}

func parseLastEventID(lastEventID string, transportName string) (int64, error) {
	trimmed := strings.TrimSpace(lastEventID)
	if trimmed == "" {
		return 0, nil
	}

	after, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid Last-Event-ID %q: %w", transportName, trimmed, err)
	}
	if after < 0 {
		return 0, fmt.Errorf("%s: invalid Last-Event-ID %q: sequence must be non-negative", transportName, trimmed)
	}
	return after, nil
}

func (h *BaseHandlers) streamSessionWithMode(
	c *gin.Context,
	writer FlushWriter,
	sessionID string,
	info *session.Info,
	query store.EventQuery,
	initial []store.SessionEvent,
	options sessionStreamOptions,
	subscription sessionEventStreamSubscription,
) {
	switch options.frameMode {
	case contract.SessionStreamFrameRaw:
		h.streamRawSessionEvents(c, writer, sessionID, info, query, initial, subscription)
	default:
		h.streamTranscriptSessionEvents(c, writer, sessionID, info, query, initial, options, subscription)
	}
}

func (h *BaseHandlers) streamRawSessionEvents(
	c *gin.Context,
	writer FlushWriter,
	sessionID string,
	info *session.Info,
	query store.EventQuery,
	initial []store.SessionEvent,
	subscription sessionEventStreamSubscription,
) {
	defer subscription.cancelIfActive()

	afterSequence := query.AfterSequence
	nextSequence, err := h.writeSessionEventBatch(writer, initial, info)
	if err != nil {
		return
	}
	if nextSequence > afterSequence {
		afterSequence = nextSequence
	}

	pollQuery := query
	pollQuery.Limit = 0
	if subscription.active() {
		h.pushAndStreamSessionEvents(c, writer, sessionID, info, pollQuery, afterSequence, subscription)
		return
	}
	h.pollAndStreamSessionEvents(c, writer, sessionID, info, pollQuery, afterSequence)
}

func (h *BaseHandlers) streamTranscriptSessionEvents(
	c *gin.Context,
	writer FlushWriter,
	sessionID string,
	info *session.Info,
	query store.EventQuery,
	initial []store.SessionEvent,
	options sessionStreamOptions,
	subscription sessionEventStreamSubscription,
) {
	defer subscription.cancelIfActive()

	afterSequence := query.AfterSequence
	if options.replaySnapshot {
		snapshot, err := h.writeTranscriptSnapshot(c.Request.Context(), writer, sessionID, info, afterSequence)
		if err != nil {
			h.logSSEWriteFailure(contract.SessionStreamEventTranscriptSnapshot, err)
			return
		}
		if snapshot.resetBelow || snapshot.maxSequence > afterSequence {
			afterSequence = snapshot.maxSequence
		}
	} else {
		nextSequence, err := h.writeTranscriptDeltasForEvents(
			c.Request.Context(),
			writer,
			sessionID,
			info,
			initial,
			afterSequence,
		)
		if err != nil {
			h.logSSEWriteFailure(contract.SessionStreamEventTranscriptDelta, err)
			return
		}
		if nextSequence > afterSequence {
			afterSequence = nextSequence
		}
	}

	pollQuery := query
	pollQuery.Limit = 0
	if subscription.active() {
		h.pushAndStreamSessionTranscript(c, writer, sessionID, info, pollQuery, afterSequence, subscription)
		return
	}
	h.pollAndStreamSessionTranscript(c, writer, sessionID, info, pollQuery, afterSequence)
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
	keepAlive := time.NewTicker(sessionStreamKeepAliveInterval)
	defer keepAlive.Stop()

	currentInfo := info
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-h.StreamDoneChannel():
			return
		case <-keepAlive.C:
			if !h.writeKeepAlive(writer) {
				return
			}
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

func (h *BaseHandlers) pushAndStreamSessionEvents(
	c *gin.Context,
	writer FlushWriter,
	sessionID string,
	info *session.Info,
	pollQuery store.EventQuery,
	afterSequence int64,
	subscription sessionEventStreamSubscription,
) {
	keepAlive := time.NewTicker(sessionStreamKeepAliveInterval)
	defer keepAlive.Stop()

	currentInfo := info
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-h.StreamDoneChannel():
			return
		case <-keepAlive.C:
			if !h.writeKeepAlive(writer) {
				return
			}
		case event, ok := <-subscription.events:
			if !ok {
				h.pollAndStreamSessionEvents(c, writer, sessionID, currentInfo, pollQuery, afterSequence)
				return
			}
			if event.Sequence <= afterSequence {
				continue
			}
			if event.Sequence > afterSequence+1 {
				h.pollAndStreamSessionEvents(c, writer, sessionID, currentInfo, pollQuery, afterSequence)
				return
			}
			nextSequence, err := h.writeSessionEventBatch(writer, []store.SessionEvent{event}, currentInfo)
			if err != nil {
				return
			}
			if nextSequence > afterSequence {
				afterSequence = nextSequence
			}
			if event.Type == session.EventTypeSessionStopped {
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

func (h *BaseHandlers) pollAndStreamSessionTranscript(
	c *gin.Context,
	writer FlushWriter,
	sessionID string,
	info *session.Info,
	pollQuery store.EventQuery,
	afterSequence int64,
) {
	ticker := time.NewTicker(h.PollInterval)
	defer ticker.Stop()
	keepAlive := time.NewTicker(sessionStreamKeepAliveInterval)
	defer keepAlive.Stop()

	currentInfo := info
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-h.StreamDoneChannel():
			return
		case <-keepAlive.C:
			if !h.writeKeepAlive(writer) {
				return
			}
		case <-ticker.C:
			var done bool
			afterSequence, currentInfo, done = h.pollSessionTranscriptTick(
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

func (h *BaseHandlers) pushAndStreamSessionTranscript(
	c *gin.Context,
	writer FlushWriter,
	sessionID string,
	info *session.Info,
	pollQuery store.EventQuery,
	afterSequence int64,
	subscription sessionEventStreamSubscription,
) {
	keepAlive := time.NewTicker(sessionStreamKeepAliveInterval)
	defer keepAlive.Stop()

	currentInfo := info
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-h.StreamDoneChannel():
			return
		case <-keepAlive.C:
			if !h.writeKeepAlive(writer) {
				return
			}
		case event, ok := <-subscription.events:
			if !ok {
				h.pollAndStreamSessionTranscript(c, writer, sessionID, currentInfo, pollQuery, afterSequence)
				return
			}
			if event.Sequence <= afterSequence {
				continue
			}
			if event.Sequence > afterSequence+1 {
				h.pollAndStreamSessionTranscript(c, writer, sessionID, currentInfo, pollQuery, afterSequence)
				return
			}
			nextSequence, err := h.writeTranscriptDeltasForEvents(
				c.Request.Context(),
				writer,
				sessionID,
				currentInfo,
				[]store.SessionEvent{event},
				afterSequence,
			)
			if err != nil {
				h.writeSSEBestEffort(writer, SSEMessage{
					Name: sessionStreamErrorKey,
					Data: ErrorPayloadForError(err),
				})
				return
			}
			if nextSequence > afterSequence {
				afterSequence = nextSequence
			}
			if event.Type == session.EventTypeSessionStopped {
				latest, statusErr := h.Sessions.Status(c.Request.Context(), sessionID)
				if statusErr != nil {
					h.writeSSEBestEffort(writer, SSEMessage{
						Name: sessionStreamErrorKey,
						Data: ErrorPayloadForError(statusErr),
					})
					return
				}
				h.logSSEWriteFailure("session_stopped", h.writeSessionStoppedEvent(writer, latest))
				return
			}
		}
	}
}

func (h *BaseHandlers) pollSessionTranscriptTick(
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
		h.writeSSEBestEffort(writer, SSEMessage{
			Name: sessionStreamErrorKey,
			Data: ErrorPayloadForError(pollErr),
		})
		return afterSequence, info, true
	}

	nextSequence, err := h.writeTranscriptDeltasForEvents(
		c.Request.Context(),
		writer,
		sessionID,
		info,
		events,
		afterSequence,
	)
	if err != nil {
		h.writeSSEBestEffort(writer, SSEMessage{
			Name: sessionStreamErrorKey,
			Data: ErrorPayloadForError(err),
		})
		return afterSequence, info, true
	}
	if nextSequence > afterSequence {
		return nextSequence, info, false
	}

	latest, statusErr := h.Sessions.Status(c.Request.Context(), sessionID)
	if statusErr != nil {
		h.writeSSEBestEffort(writer, SSEMessage{
			Name: sessionStreamErrorKey,
			Data: ErrorPayloadForError(statusErr),
		})
		return afterSequence, info, true
	}
	if latest != nil && latest.State == session.StateStopped {
		h.logSSEWriteFailure("session_stopped", h.writeSessionStoppedEvent(writer, latest))
		return afterSequence, latest, true
	}
	if h.IncludeSessionWorkspaceInSSE {
		info = latest
	}

	return afterSequence, info, false
}

func streamWorkspaceFields(info *session.Info) (string, string) {
	if info == nil {
		return "", ""
	}
	ref := workref.NewPath(info.WorkspaceID, info.Workspace)
	return ref.WorkspaceID, ref.WorkspacePath
}
