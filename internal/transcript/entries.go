package transcript

import (
	"encoding/json"

	"github.com/compozy/agh/internal/store"
)

// Entry pairs a UI message with the event sequence that last shaped it.
type Entry struct {
	Message   UIMessage `json:"message"`
	Sequence  int64     `json:"sequence"`
	EventType string    `json:"-"`
	Marker    *Marker   `json:"-"`
}

// ToUIEntries projects persisted session events into sequence-bearing AI SDK messages.
func ToUIEntries(events []store.SessionEvent) ([]Entry, error) {
	fold := NewFold()
	if _, err := fold.Apply(events); err != nil {
		return nil, err
	}
	return fold.Entries(), nil
}

// MessagesFromEntries returns the UI messages carried by sequence-bearing entries.
func MessagesFromEntries(entries []Entry) []UIMessage {
	messages := make([]UIMessage, 0, len(entries))
	for _, entry := range entries {
		messages = append(messages, entry.Message)
	}
	return messages
}

// CloneEntries returns an isolated copy of sequence-bearing UI messages.
func CloneEntries(entries []Entry) []Entry {
	cloned := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		cloned = append(cloned, Entry{
			Message:   cloneUIMessage(entry.Message),
			Sequence:  entry.Sequence,
			EventType: entry.EventType,
			Marker:    cloneMarker(entry.Marker),
		})
	}
	return cloned
}

func cloneMarker(marker *Marker) *Marker {
	if marker == nil {
		return nil
	}
	cloned := marker.Normalize()
	return &cloned
}

func cloneUIMessage(message UIMessage) UIMessage {
	return UIMessage{
		ID:       message.ID,
		Role:     message.Role,
		Metadata: cloneRawMessage(message.Metadata),
		Parts:    cloneUIMessageParts(message.Parts),
	}
}

func cloneRawMessage(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}
