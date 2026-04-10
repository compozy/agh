// Package sse provides shared server-sent event decoding helpers.
package sse

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const maxLineBytes = 1024 * 1024

// Event is one parsed server-sent event frame.
type Event struct {
	ID    string
	Event string
	Data  json.RawMessage
}

// Handler consumes parsed SSE frames.
type Handler func(Event) error

// ErrStop stops SSE decoding without surfacing an error.
var ErrStop = errors.New("sse: stop stream")

// Decode reads one SSE stream from body until EOF, context cancellation, or a
// handler error.
func Decode(ctx context.Context, body io.Reader, handler Handler) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineBytes)

	event := Event{}
	dataLines := make([]string, 0, 4)
	emit := func() error {
		if event.ID == "" && event.Event == "" && len(dataLines) == 0 {
			return nil
		}
		if len(dataLines) > 0 {
			event.Data = json.RawMessage(strings.Join(dataLines, "\n"))
		}
		err := handler(event)
		event = Event{}
		dataLines = dataLines[:0]
		return err
	}

	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}

		line := scanner.Text()
		if line == "" {
			if err := emit(); err != nil {
				if errors.Is(err, ErrStop) {
					return nil
				}
				return err
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}

		field, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		value = strings.TrimPrefix(value, " ")

		switch field {
		case "id":
			event.ID = value
		case "event":
			event.Event = value
		case "data":
			dataLines = append(dataLines, value)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("sse: read stream: %w", err)
	}
	if err := emit(); err != nil {
		if errors.Is(err, ErrStop) {
			return nil
		}
		return err
	}
	return nil
}
