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
	presence     auditPresenceWindow

	fileMu sync.Mutex
}

type AuditWriterOption func(*FileAuditWriter)

type auditPresenceWindow struct {
	duration time.Duration
	lastSeen map[string]time.Time
	mu       sync.Mutex
}

// WithAuditWriterPresenceWindow suppresses repeated greet heartbeats from the
// operator timeline while leaving the raw audit trail untouched.
func WithAuditWriterPresenceWindow(window time.Duration) AuditWriterOption {
	return func(writer *FileAuditWriter) {
		if writer == nil {
			return
		}
		if window <= 0 {
			writer.presence.duration = 0
			writer.presence.lastSeen = nil
			return
		}
		writer.presence.duration = window
		if writer.presence.lastSeen == nil {
			writer.presence.lastSeen = make(map[string]time.Time)
		}
	}
}

// NewAuditWriter constructs the dual-path network audit writer.
func NewAuditWriter(path string, auditStore AuditStore, opts ...AuditWriterOption) (*FileAuditWriter, error) {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" && auditStore == nil {
		return nil, errors.New("network: audit sink is required")
	}

	writer := &FileAuditWriter{
		path:  cleanPath,
		store: auditStore,
		now: func() time.Time {
			return time.Now().UTC()
		},
		messageStore: messageStoreFromAuditStore(auditStore),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(writer)
		}
	}
	return writer, nil
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
func (w *FileAuditWriter) RecordRejected(
	ctx context.Context,
	sessionID string,
	envelope Envelope,
	reason string,
) error {
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

func (w *FileAuditWriter) record(
	ctx context.Context,
	sessionID string,
	direction string,
	envelope Envelope,
	reason string,
) error {
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
		if w.shouldWriteTimelineMessage(messageEntry) {
			recordErr = errors.Join(recordErr, w.messageStore.WriteNetworkMessage(ctx, messageEntry))
		}
	}

	return recordErr
}

func (w *FileAuditWriter) shouldWriteTimelineMessage(entry store.NetworkMessageEntry) bool {
	if strings.TrimSpace(entry.Kind) != string(KindGreet) {
		return true
	}
	if w == nil || w.presence.duration <= 0 {
		return true
	}

	key := strings.Join([]string{
		strings.TrimSpace(entry.Direction),
		strings.TrimSpace(entry.Channel),
		strings.TrimSpace(entry.PeerFrom),
		strings.TrimSpace(entry.PeerTo),
	}, "\x00")

	at := entry.Timestamp.UTC()
	w.presence.mu.Lock()
	defer w.presence.mu.Unlock()

	if w.presence.lastSeen == nil {
		w.presence.lastSeen = make(map[string]time.Time)
	}
	cutoff := at.Add(-w.presence.duration)
	for existingKey, seenAt := range w.presence.lastSeen {
		if seenAt.Before(cutoff) {
			delete(w.presence.lastSeen, existingKey)
		}
	}
	lastSeen, ok := w.presence.lastSeen[key]
	w.presence.lastSeen[key] = at
	if !ok {
		return true
	}
	return at.Sub(lastSeen) > w.presence.duration
}

// NormalizeAuditEntry derives a consistent audit row from envelope metadata.
func NormalizeAuditEntry(
	sessionID string,
	direction string,
	envelope Envelope,
	reason string,
	at time.Time,
) (store.NetworkAuditEntry, error) {
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

func normalizeTimelineMessageEntry(
	sessionID string,
	direction string,
	envelope Envelope,
	at time.Time,
) (store.NetworkMessageEntry, bool, error) {
	switch strings.TrimSpace(direction) {
	case AuditDirectionSent, AuditDirectionReceived:
	default:
		return store.NetworkMessageEntry{}, false, nil
	}

	body, err := envelope.DecodeBody()
	if err != nil {
		return store.NetworkMessageEntry{}, false, fmt.Errorf("network: decode timeline envelope body: %w", err)
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}

	peerTo := ""
	if envelope.To != nil {
		peerTo = strings.TrimSpace(*envelope.To)
	}
	entry := store.NetworkMessageEntry{
		MessageID:     strings.TrimSpace(envelope.ID),
		SessionID:     strings.TrimSpace(sessionID),
		Channel:       strings.TrimSpace(envelope.Channel),
		Direction:     strings.TrimSpace(direction),
		PeerFrom:      strings.TrimSpace(envelope.From),
		PeerTo:        peerTo,
		Kind:          strings.TrimSpace(string(envelope.Kind)),
		PreviewText:   previewForBody(body),
		Body:          cloneRawMessage(envelope.Body),
		Timestamp:     at.UTC(),
		InteractionID: trimmedPointerValue(envelope.InteractionID),
		ReplyTo:       trimmedPointerValue(envelope.ReplyTo),
		TraceID:       trimmedPointerValue(envelope.TraceID),
		CausationID:   trimmedPointerValue(envelope.CausationID),
	}
	switch value := body.(type) {
	case SayBody:
		entry.Intent = strings.TrimSpace(value.Intent)
		entry.Text = value.Text
	case DirectBody:
		entry.Intent = strings.TrimSpace(value.Intent)
		entry.Text = value.Text
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

func trimmedPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func (w *FileAuditWriter) appendFile(entry store.NetworkAuditEntry) error {
	w.fileMu.Lock()
	defer w.fileMu.Unlock()

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
