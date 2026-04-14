package network

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pedronauck/agh/internal/store"
)

const (
	// AuditDirectionSent records a successful outbound publish.
	AuditDirectionSent = "sent"
	// AuditDirectionReceived records an accepted inbound delivery.
	AuditDirectionReceived = "received"
	// AuditDirectionRejected records a rejected envelope.
	AuditDirectionRejected = "rejected"
	// AuditDirectionDelivered records a completed local delivery.
	AuditDirectionDelivered = "delivered"
)

// AuditStore is the persistence surface consumed by the network audit writer.
type AuditStore interface {
	WriteNetworkAudit(ctx context.Context, entry store.NetworkAuditEntry) error
}

// MessageStore is the persistence surface consumed by the network timeline writer.
type MessageStore interface {
	WriteNetworkMessage(ctx context.Context, entry store.NetworkMessageEntry) error
}

// AuditWriter records network activity into the configured sinks.
type AuditWriter interface {
	RecordSent(ctx context.Context, sessionID string, envelope Envelope) error
	RecordReceived(ctx context.Context, sessionID string, envelope Envelope) error
	RecordRejected(ctx context.Context, sessionID string, envelope Envelope, reason string) error
	RecordDelivered(ctx context.Context, sessionID string, envelope Envelope) error
}

// FileAuditWriter writes normalized network audit records to a JSONL file and
// optionally mirrors them into a persistent store.
type FileAuditWriter struct {
	path         string
	store        AuditStore
	messageStore MessageStore
	now          func() time.Time

	mu sync.Mutex
}

// NewAuditWriter constructs the dual-path network audit writer.
func NewAuditWriter(path string, auditStore AuditStore) (*FileAuditWriter, error) {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" && auditStore == nil {
		return nil, errors.New("network: audit sink is required")
	}

	return &FileAuditWriter{
		path:  cleanPath,
		store: auditStore,
		now: func() time.Time {
			return time.Now().UTC()
		},
		messageStore: messageStoreFromAuditStore(auditStore),
	}, nil
}

func messageStoreFromAuditStore(auditStore AuditStore) MessageStore {
	if auditStore == nil {
		return nil
	}
	messageStore, ok := auditStore.(MessageStore)
	if !ok {
		return nil
	}
	return messageStore
}

var _ AuditWriter = (*FileAuditWriter)(nil)

// RecordSent stores a sent network audit record.
func (w *FileAuditWriter) RecordSent(ctx context.Context, sessionID string, envelope Envelope) error {
	return w.record(ctx, sessionID, AuditDirectionSent, envelope, "")
}

// RecordReceived stores a received network audit record.
func (w *FileAuditWriter) RecordReceived(ctx context.Context, sessionID string, envelope Envelope) error {
	return w.record(ctx, sessionID, AuditDirectionReceived, envelope, "")
}

// RecordRejected stores a rejected network audit record.
func (w *FileAuditWriter) RecordRejected(ctx context.Context, sessionID string, envelope Envelope, reason string) error {
	return w.record(ctx, sessionID, AuditDirectionRejected, envelope, reason)
}

// RecordDelivered stores a delivered network audit record.
func (w *FileAuditWriter) RecordDelivered(ctx context.Context, sessionID string, envelope Envelope) error {
	return w.record(ctx, sessionID, AuditDirectionDelivered, envelope, "")
}

func (w *FileAuditWriter) record(ctx context.Context, sessionID string, direction string, envelope Envelope, reason string) error {
	if ctx == nil {
		return errors.New("network: audit context is required")
	}
	if w == nil {
		return errors.New("network: audit writer is required")
	}

	entry, err := NormalizeAuditEntry(sessionID, direction, envelope, reason, w.now())
	if err != nil {
		return err
	}

	var recordErr error
	if w.path != "" {
		recordErr = errors.Join(recordErr, w.appendFile(entry))
	}
	if w.store != nil {
		recordErr = errors.Join(recordErr, w.store.WriteNetworkAudit(ctx, entry))
	}
	if messageEntry, ok, messageErr := normalizeTimelineMessageEntry(sessionID, direction, envelope, entry.Timestamp); messageErr != nil {
		recordErr = errors.Join(recordErr, messageErr)
	} else if ok && w.messageStore != nil {
		recordErr = errors.Join(recordErr, w.messageStore.WriteNetworkMessage(ctx, messageEntry))
	}

	return recordErr
}

// NormalizeAuditEntry derives a consistent audit row from envelope metadata.
func NormalizeAuditEntry(sessionID string, direction string, envelope Envelope, reason string, at time.Time) (store.NetworkAuditEntry, error) {
	canonicalEnvelope, err := json.Marshal(envelope)
	if err != nil {
		return store.NetworkAuditEntry{}, fmt.Errorf("network: marshal audit envelope: %w", err)
	}

	peerTo := ""
	if envelope.To != nil {
		peerTo = strings.TrimSpace(*envelope.To)
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}

	entry := store.NetworkAuditEntry{
		ID:        store.NewID("naud"),
		SessionID: strings.TrimSpace(sessionID),
		Direction: strings.TrimSpace(direction),
		Kind:      strings.TrimSpace(string(envelope.Kind)),
		Channel:   strings.TrimSpace(envelope.Channel),
		PeerFrom:  strings.TrimSpace(envelope.From),
		PeerTo:    peerTo,
		MessageID: strings.TrimSpace(envelope.ID),
		Reason:    strings.TrimSpace(reason),
		Size:      len(canonicalEnvelope),
		Timestamp: at.UTC(),
	}
	if err := entry.Validate(); err != nil {
		return store.NetworkAuditEntry{}, err
	}

	return entry, nil
}

func normalizeTimelineMessageEntry(sessionID string, direction string, envelope Envelope, at time.Time) (store.NetworkMessageEntry, bool, error) {
	if envelope.Kind != KindSay {
		return store.NetworkMessageEntry{}, false, nil
	}
	switch strings.TrimSpace(direction) {
	case AuditDirectionSent, AuditDirectionReceived:
	default:
		return store.NetworkMessageEntry{}, false, nil
	}

	body, err := envelope.DecodeBody()
	if err != nil {
		return store.NetworkMessageEntry{}, false, fmt.Errorf("network: decode timeline envelope body: %w", err)
	}
	sayBody, ok := body.(SayBody)
	if !ok {
		return store.NetworkMessageEntry{}, false, fmt.Errorf("network: unexpected timeline body type for %q", envelope.ID)
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}

	entry := store.NetworkMessageEntry{
		MessageID: strings.TrimSpace(envelope.ID),
		SessionID: strings.TrimSpace(sessionID),
		Channel:   strings.TrimSpace(envelope.Channel),
		PeerFrom:  strings.TrimSpace(envelope.From),
		Kind:      strings.TrimSpace(string(envelope.Kind)),
		Intent:    strings.TrimSpace(sayBody.Intent),
		Text:      sayBody.Text,
		Timestamp: at.UTC(),
	}
	if err := entry.Validate(); err != nil {
		return store.NetworkMessageEntry{}, false, err
	}
	return entry, true, nil
}

func (w *FileAuditWriter) appendFile(entry store.NetworkAuditEntry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(w.path), 0o755); err != nil {
		return fmt.Errorf("network: create audit log directory: %w", err)
	}

	file, err := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("network: open audit log file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	payload, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("network: marshal audit log entry: %w", err)
	}
	payload = append(payload, '\n')

	if _, err := file.Write(payload); err != nil {
		return fmt.Errorf("network: append audit log entry: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("network: sync audit log file: %w", err)
	}

	return nil
}
