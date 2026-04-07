package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/store"
)

// PrepareSSE configures a Gin response for SSE streaming.
func PrepareSSE(c *gin.Context) (FlushWriter, error) {
	writer, ok := c.Writer.(FlushWriter)
	if !ok {
		return nil, errors.New("response writer does not support flushing")
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

// WriteSSE writes one SSE message with JSON-encoded data.
func WriteSSE(writer FlushWriter, msg SSEMessage) error {
	if writer == nil {
		return errors.New("sse writer is required")
	}

	payload, err := json.Marshal(msg.Data)
	if err != nil {
		return fmt.Errorf("marshal sse payload: %w", err)
	}
	if len(payload) == 0 {
		payload = []byte("null")
	}

	return WriteSSERaw(writer, msg.ID, string(payload), msg.Name)
}

// WriteSSERaw writes one SSE message using a pre-encoded payload.
func WriteSSERaw(writer FlushWriter, id string, raw string, names ...string) error {
	if writer == nil {
		return errors.New("sse writer is required")
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

// EmitObserveEvents writes observe events newer than the supplied cursor.
func EmitObserveEvents(writer FlushWriter, events []store.EventSummary, cursor ObserveCursor) ObserveCursor {
	next := cursor
	for _, event := range events {
		if !ObserveEventAfterCursor(event, next) {
			continue
		}
		next = ObserveCursor{
			Timestamp: event.Timestamp.UTC(),
			ID:        event.ID,
		}
		if err := WriteSSE(writer, SSEMessage{
			ID:   ObserveEventID(event),
			Name: event.Type,
			Data: ObserveEventPayloadFromEvent(event),
		}); err != nil {
			return next
		}
	}
	return next
}

// ObserveEventAfterCursor reports whether an observe event should be emitted after the cursor.
func ObserveEventAfterCursor(event store.EventSummary, cursor ObserveCursor) bool {
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

// ObserveEventID builds a stable Last-Event-ID value for observe streaming.
func ObserveEventID(event store.EventSummary) string {
	return event.Timestamp.UTC().Format(timeRFC3339Nano) + "|" + event.ID
}
