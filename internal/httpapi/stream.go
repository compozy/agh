package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/apisupport"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
)

const timeRFC3339Nano = time.RFC3339Nano

type errorPayload struct {
	Error string `json:"error"`
}

type sseMessage struct {
	ID   string
	Name string
	Data any
}

type observeCursor struct {
	Timestamp time.Time
	ID        string
}

type flushWriter interface {
	io.Writer
	Flush()
}

func (h *Handlers) streamSession(c *gin.Context) {
	if _, err := h.sessions.Status(c.Request.Context(), c.Param("id")); err != nil {
		respondError(c, statusForSessionError(err), err)
		return
	}

	query, err := parseSessionEventQuery(c)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	if lastEventID := strings.TrimSpace(c.GetHeader("Last-Event-ID")); lastEventID != "" {
		after, parseErr := strconv.ParseInt(lastEventID, 10, 64)
		if parseErr != nil {
			respondError(c, http.StatusBadRequest, fmt.Errorf("httpapi: invalid Last-Event-ID %q: %w", lastEventID, parseErr))
			return
		}
		query.AfterSequence = after
	}

	initial, err := h.sessions.Events(c.Request.Context(), c.Param("id"), query)
	if err != nil {
		respondError(c, statusForSessionError(err), err)
		return
	}

	writer, err := prepareSSE(c)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	afterSequence := query.AfterSequence
	for _, event := range initial {
		afterSequence = event.Sequence
		if err := writeSSE(writer, sseMessage{
			ID:   strconv.FormatInt(event.Sequence, 10),
			Name: event.Type,
			Data: sessionEventPayloadFromEvent(event),
		}); err != nil {
			return
		}
	}

	pollQuery := query
	pollQuery.Limit = 0
	pollQuery.AfterSequence = afterSequence

	ticker := time.NewTicker(h.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-h.streamDone:
			return
		case <-ticker.C:
			pollQuery.AfterSequence = afterSequence
			events, err := h.sessions.Events(c.Request.Context(), c.Param("id"), pollQuery)
			if err != nil {
				_ = writeSSE(writer, sseMessage{
					Name: "error",
					Data: errorPayload{Error: err.Error()},
				})
				return
			}
			for _, event := range events {
				afterSequence = event.Sequence
				if err := writeSSE(writer, sseMessage{
					ID:   strconv.FormatInt(event.Sequence, 10),
					Name: event.Type,
					Data: sessionEventPayloadFromEvent(event),
				}); err != nil {
					return
				}
			}
			if len(events) == 0 {
				info, err := h.sessions.Status(c.Request.Context(), c.Param("id"))
				if err != nil {
					_ = writeSSE(writer, sseMessage{
						Name: "error",
						Data: errorPayload{Error: err.Error()},
					})
					return
				}
				if info != nil && info.State == session.StateStopped {
					_ = writeSSE(writer, sseMessage{
						Name: session.EventTypeSessionStopped,
						Data: sessionEventPayload{
							SessionID:     info.ID,
							Type:          session.EventTypeSessionStopped,
							WorkspaceID:   info.WorkspaceID,
							WorkspacePath: info.Workspace,
							Timestamp:     info.UpdatedAt,
						},
					})
					return
				}
			}
		}
	}
}

func (h *Handlers) streamObserveEvents(c *gin.Context) {
	query, err := parseObserveEventQuery(c)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	cursor, err := parseObserveCursor(c.GetHeader("Last-Event-ID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	if !cursor.Timestamp.IsZero() {
		query.Since = cursor.Timestamp
	}

	initial, err := h.observer.QueryEvents(c.Request.Context(), query)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	writer, err := prepareSSE(c)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	cursor = emitObserveEvents(writer, initial, cursor)

	pollQuery := query
	pollQuery.Limit = 0
	if !cursor.Timestamp.IsZero() {
		pollQuery.Since = cursor.Timestamp
	}

	ticker := time.NewTicker(h.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-h.streamDone:
			return
		case <-ticker.C:
			if !cursor.Timestamp.IsZero() {
				pollQuery.Since = cursor.Timestamp
			}
			events, err := h.observer.QueryEvents(c.Request.Context(), pollQuery)
			if err != nil {
				_ = writeSSE(writer, sseMessage{
					Name: "error",
					Data: errorPayload{Error: err.Error()},
				})
				return
			}
			cursor = emitObserveEvents(writer, events, cursor)
		}
	}
}

func parseObserveEventQuery(c *gin.Context) (store.EventSummaryQuery, error) {
	since, err := parseOptionalTime(c.Query("since"))
	if err != nil {
		return store.EventSummaryQuery{}, err
	}
	limit, err := parseOptionalInt(c.Query("limit"))
	if err != nil {
		return store.EventSummaryQuery{}, err
	}

	return store.EventSummaryQuery{
		SessionID: strings.TrimSpace(c.Query("session_id")),
		AgentName: strings.TrimSpace(c.Query("agent_name")),
		Type:      strings.TrimSpace(c.Query("type")),
		Since:     since,
		Limit:     limit,
	}, nil
}

func parseObserveCursor(raw string) (observeCursor, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return observeCursor{}, nil
	}

	parts := strings.SplitN(value, "|", 2)
	if len(parts) != 2 {
		return observeCursor{}, fmt.Errorf("httpapi: invalid Last-Event-ID %q", value)
	}

	timestamp, err := time.Parse(timeRFC3339Nano, parts[0])
	if err != nil {
		return observeCursor{}, fmt.Errorf("httpapi: invalid Last-Event-ID timestamp %q: %w", parts[0], err)
	}

	return observeCursor{
		Timestamp: timestamp.UTC(),
		ID:        parts[1],
	}, nil
}

func emitObserveEvents(writer flushWriter, events []store.EventSummary, cursor observeCursor) observeCursor {
	next := cursor
	for _, event := range events {
		if !observeEventAfterCursor(event, next) {
			continue
		}
		next = observeCursor{
			Timestamp: event.Timestamp.UTC(),
			ID:        event.ID,
		}
		if err := writeSSE(writer, sseMessage{
			ID:   observeEventID(event),
			Name: event.Type,
			Data: observeEventPayloadFromEvent(event),
		}); err != nil {
			return next
		}
	}
	return next
}

func observeEventAfterCursor(event store.EventSummary, cursor observeCursor) bool {
	if cursor.Timestamp.IsZero() && strings.TrimSpace(cursor.ID) == "" {
		return true
	}

	timestamp := event.Timestamp.UTC()
	switch {
	case timestamp.After(cursor.Timestamp):
		return true
	case timestamp.Before(cursor.Timestamp):
		return false
	default:
		return event.ID > cursor.ID
	}
}

func observeEventID(event store.EventSummary) string {
	return event.Timestamp.UTC().Format(timeRFC3339Nano) + "|" + event.ID
}

func prepareSSE(c *gin.Context) (flushWriter, error) {
	writer, ok := c.Writer.(flushWriter)
	if !ok {
		return nil, errors.New("httpapi: response writer does not support flushing")
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	c.Writer.WriteHeaderNow()
	writer.Flush()

	return writer, nil
}

func writeSSE(writer flushWriter, msg sseMessage) error {
	if writer == nil {
		return errors.New("httpapi: sse writer is required")
	}

	payload, err := json.Marshal(msg.Data)
	if err != nil {
		return fmt.Errorf("httpapi: marshal sse payload: %w", err)
	}
	if len(payload) == 0 {
		payload = []byte("null")
	}

	return writeSSERaw(writer, msg.ID, string(payload), msg.Name)
}

func writeSSERaw(writer flushWriter, id string, raw string, names ...string) error {
	if writer == nil {
		return errors.New("httpapi: sse writer is required")
	}

	if id != "" {
		if _, err := io.WriteString(writer, "id: "+id+"\n"); err != nil {
			return err
		}
	}
	if len(names) > 0 && strings.TrimSpace(names[0]) != "" {
		if _, err := io.WriteString(writer, "event: "+names[0]+"\n"); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(writer, "data: "+raw+"\n\n"); err != nil {
		return err
	}
	writer.Flush()
	return nil
}

func respondError(c *gin.Context, status int, err error) {
	message := http.StatusText(status)
	if status >= http.StatusInternalServerError {
		if strings.TrimSpace(message) == "" {
			message = "internal server error"
		}
	} else if err != nil && strings.TrimSpace(err.Error()) != "" {
		message = err.Error()
	} else if strings.TrimSpace(message) == "" {
		message = "unknown error"
	}
	c.JSON(status, errorPayload{Error: message})
}

func statusForSessionError(err error) int {
	return apisupport.StatusForSessionError(err)
}

func payloadJSON(raw string) json.RawMessage {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return json.RawMessage("null")
	}
	if json.Valid([]byte(trimmed)) {
		return json.RawMessage(trimmed)
	}

	encoded, err := json.Marshal(trimmed)
	if err != nil {
		return json.RawMessage("null")
	}
	return json.RawMessage(encoded)
}
