package transcript

import (
	"github.com/compozy/agh/internal/acp"
	"github.com/compozy/agh/internal/store"
)

// Fold incrementally projects ordered session events into sequence-bearing UI messages.
type Fold struct {
	entries             []Entry
	assistant           *uiMessageBuilder
	assistantSequence   int64
	assistantEntryIndex int
	toolLifecycles      map[string]*uiToolLifecycle
	usedMessageIDs      map[string]struct{}
}

// NewFold returns an empty transcript fold.
func NewFold() *Fold {
	return &Fold{
		entries:             []Entry{},
		assistantEntryIndex: -1,
		toolLifecycles:      make(map[string]*uiToolLifecycle),
		usedMessageIDs:      make(map[string]struct{}),
	}
}

// Apply folds events newer than the caller's cached cursor and returns changed entries.
func (f *Fold) Apply(events []store.SessionEvent) ([]Entry, error) {
	if f == nil {
		return nil, nil
	}
	changed := make([]Entry, 0, len(events))
	for _, storedEvent := range sortedTranscriptEvents(events) {
		changed = f.applyEvent(storedEvent, changed)
	}
	return CloneEntries(changed), nil
}

// Entries returns the fold's accumulated transcript entries.
func (f *Fold) Entries() []Entry {
	if f == nil {
		return []Entry{}
	}
	return CloneEntries(f.entries)
}

func (f *Fold) applyEvent(storedEvent store.SessionEvent, changed []Entry) []Entry {
	decoded := decodeStoredEvent(storedEvent)
	if markerText := transcriptMarkerText(decoded.parsed); markerText != "" {
		changed = f.flushAssistant(changed)
		return f.appendMarkerEntry(decoded, markerText, storedEvent.Sequence, changed)
	}
	switch decoded.parsed.Type {
	case acp.EventTypeUserMessage:
		changed = f.flushAssistant(changed)
		if message := inputUIMessage(decoded, UIRoleUser); message != nil {
			changed = f.appendEntry(*message, storedEvent.Sequence, changed)
		}
	case acp.EventTypeSyntheticReentry:
		changed = f.flushAssistant(changed)
		if message := inputUIMessage(decoded, UIRoleSystem); message != nil {
			changed = f.appendEntry(*message, storedEvent.Sequence, changed)
		}
	default:
		assistantID := assistantMessageID(decoded)
		if f.assistant == nil || f.assistant.logicalID != assistantID {
			changed = f.flushAssistant(changed)
			f.assistant = newUIMessageBuilder(
				uniqueUIMessageID(assistantID, f.usedMessageIDs),
				assistantID,
				UIRoleAssistant,
				f.toolLifecycles,
			)
			f.assistantEntryIndex = -1
		}
		f.assistantSequence = storedEvent.Sequence
		applyDecodedEvent(f.assistant, decoded)
		changed = f.upsertAssistant(false, changed)
	}
	return changed
}

func (f *Fold) flushAssistant(changed []Entry) []Entry {
	if f.assistant == nil {
		return changed
	}
	changed = f.upsertAssistant(true, changed)
	f.assistant = nil
	f.assistantSequence = 0
	f.assistantEntryIndex = -1
	return changed
}

func (f *Fold) upsertAssistant(complete bool, changed []Entry) []Entry {
	if f.assistant == nil {
		return changed
	}
	message := f.assistant.build(complete || f.assistant.finished)
	if message == nil {
		return changed
	}
	entry := Entry{
		Message:  *message,
		Sequence: f.assistantSequence,
	}
	if f.assistantEntryIndex >= 0 {
		f.entries[f.assistantEntryIndex] = entry
		return appendChangedEntry(changed, entry)
	}
	f.usedMessageIDs[entry.Message.ID] = struct{}{}
	f.entries = append(f.entries, entry)
	f.assistantEntryIndex = len(f.entries) - 1
	return appendChangedEntry(changed, entry)
}

func (f *Fold) appendEntry(message UIMessage, sequence int64, changed []Entry) []Entry {
	return f.appendTranscriptEntry(Entry{
		Message:  message,
		Sequence: sequence,
	}, changed)
}

func (f *Fold) appendMarkerEntry(
	decoded *decodedStoredEvent,
	markerText string,
	sequence int64,
	changed []Entry,
) []Entry {
	marker := decoded.parsed.Marker.Normalize()
	return f.appendTranscriptEntry(Entry{
		Message:   runtimeMarkerUIMessage(decoded, markerText),
		Sequence:  sequence,
		EventType: decoded.parsed.Type,
		Marker:    &marker,
	}, changed)
}

func (f *Fold) appendTranscriptEntry(entry Entry, changed []Entry) []Entry {
	entry.Message.ID = uniqueUIMessageID(entry.Message.ID, f.usedMessageIDs)
	f.usedMessageIDs[entry.Message.ID] = struct{}{}
	f.entries = append(f.entries, entry)
	return appendChangedEntry(changed, entry)
}

func appendChangedEntry(entries []Entry, entry Entry) []Entry {
	for index := range entries {
		if entries[index].Message.ID == entry.Message.ID {
			entries[index] = entry
			return entries
		}
	}
	return append(entries, entry)
}
