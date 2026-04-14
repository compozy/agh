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

// TaskIngressAudit captures one task-domain ingress decision originating from a
// validated network peer.
type TaskIngressAudit struct {
	Action    string
	Direction string
	PeerID    string
	Channel   string
	RequestID string
	Reason    string
	Payload   any
}

// TaskIngressAuditWriter is the optional audit extension used by task-aware
// network ingress. Existing protocol-message auditing remains unchanged.
type TaskIngressAuditWriter interface {
	RecordTaskIngress(ctx context.Context, audit TaskIngressAudit) error
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
var _ TaskIngressAuditWriter = (*FileAuditWriter)(nil)

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

// RecordTaskIngress stores one accepted or rejected task-ingress audit record
// using the existing network audit sinks.
func (w *FileAuditWriter) RecordTaskIngress(ctx context.Context, audit TaskIngressAudit) error {
	if w == nil {
		return errors.New("network: audit writer is required")
	}
	if ctx == nil {
		return errors.New("network: audit context is required")
	}
	if w.path == "" && w.store == nil {
		return errors.New("network: audit sink is required")
	}

	now := w.now
	if now == nil {
		now = func() time.Time {
			return time.Now().UTC()
		}
	}

	entry, err := normalizeTaskIngressAuditEntry(audit, now())
	if err != nil {
		return fmt.Errorf("network: normalize task ingress audit entry: %w", err)
	}

	var recordErr error
	if w.path != "" {
		if err := w.appendFile(entry); err != nil {
			recordErr = errors.Join(recordErr, fmt.Errorf("network: append file audit entry: %w", err))
		}
	}
	if w.store != nil {
		if err := w.store.WriteNetworkAudit(ctx, entry); err != nil {
			recordErr = errors.Join(recordErr, fmt.Errorf("network: persist audit entry: %w", err))
		}
	}

	return recordErr
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
	var auditWriteErr error
	if w.store != nil {
		auditWriteErr = w.store.WriteNetworkAudit(ctx, entry)
		recordErr = errors.Join(recordErr, auditWriteErr)
	}
	if w.messageStore == nil || auditWriteErr != nil {
		return recordErr
	}

	messageEntry, ok, messageErr := normalizeTimelineMessageEntry(sessionID, direction, envelope, entry.Timestamp)
	if messageErr != nil {
		recordErr = errors.Join(recordErr, messageErr)
		return recordErr
	}
	if ok {
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

func normalizeTaskIngressAuditEntry(audit TaskIngressAudit, at time.Time) (store.NetworkAuditEntry, error) {
	payloadSize := 0
	if audit.Payload != nil {
		payload, err := json.Marshal(audit.Payload)
		if err != nil {
			return store.NetworkAuditEntry{}, fmt.Errorf("network: marshal task ingress audit payload: %w", err)
		}
		payloadSize = len(payload)
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}

	entry := store.NetworkAuditEntry{
		ID:        store.NewID("naud"),
		SessionID: "netpeer:" + strings.TrimSpace(audit.PeerID),
		Direction: strings.TrimSpace(audit.Direction),
		Kind:      strings.TrimSpace(audit.Action),
		Channel:   strings.TrimSpace(audit.Channel),
		PeerFrom:  strings.TrimSpace(audit.PeerID),
		MessageID: strings.TrimSpace(audit.RequestID),
		Reason:    strings.TrimSpace(audit.Reason),
		Size:      payloadSize,
		Timestamp: at.UTC(),
	}
	if err := entry.Validate(); err != nil {
		return store.NetworkAuditEntry{}, fmt.Errorf("network: validate audit entry: %w", err)
	}
	return entry, nil
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
