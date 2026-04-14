package network

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/store"
)

type recordingAuditStore struct {
	entries  []store.NetworkAuditEntry
	messages []store.NetworkMessageEntry
}

func (s *recordingAuditStore) WriteNetworkAudit(_ context.Context, entry store.NetworkAuditEntry) error {
	s.entries = append(s.entries, entry)
	return nil
}

func (s *recordingAuditStore) WriteNetworkMessage(_ context.Context, entry store.NetworkMessageEntry) error {
	s.messages = append(s.messages, entry)
	return nil
}

type failingAuditStore struct {
	recordingAuditStore
	auditErr     error
	auditCalls   int
	messageCalls int
}

func (s *failingAuditStore) WriteNetworkAudit(_ context.Context, entry store.NetworkAuditEntry) error {
	s.auditCalls++
	if s.auditErr != nil {
		return s.auditErr
	}
	return s.recordingAuditStore.WriteNetworkAudit(context.Background(), entry)
}

func (s *failingAuditStore) WriteNetworkMessage(_ context.Context, entry store.NetworkMessageEntry) error {
	s.messageCalls++
	return s.recordingAuditStore.WriteNetworkMessage(context.Background(), entry)
}

func TestNewAuditWriterRequiresSink(t *testing.T) {
	t.Parallel()

	if _, err := NewAuditWriter("", nil); err == nil {
		t.Fatal("NewAuditWriter() error = nil, want non-nil")
	}
}

func TestNormalizeAuditEntryRejectsRejectedWithoutReason(t *testing.T) {
	t.Parallel()

	_, err := NormalizeAuditEntry(
		"sess-audit",
		AuditDirectionRejected,
		testAuditEnvelope(t),
		"",
		time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
	)
	if err == nil {
		t.Fatal("NormalizeAuditEntry() error = nil, want non-nil")
	}
}

func TestAuditWriterNormalizesRecordsConsistentlyAcrossSinks(t *testing.T) {
	t.Parallel()

	auditPath := filepath.Join(t.TempDir(), "logs", "network.audit")
	storeSink := &recordingAuditStore{}

	writer, err := NewAuditWriter(auditPath, storeSink)
	if err != nil {
		t.Fatalf("NewAuditWriter() error = %v", err)
	}

	recordingTime := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	writer.now = func() time.Time { return recordingTime }

	if err := writer.RecordReceived(context.Background(), "sess-audit", testAuditEnvelope(t)); err != nil {
		t.Fatalf("RecordReceived() error = %v", err)
	}

	if got, want := len(storeSink.entries), 1; got != want {
		t.Fatalf("len(store entries) = %d, want %d", got, want)
	}

	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", auditPath, err)
	}

	var fileEntry store.NetworkAuditEntry
	if err := json.Unmarshal(data, &fileEntry); err != nil {
		t.Fatalf("json.Unmarshal(file entry) error = %v", err)
	}

	if !reflect.DeepEqual(fileEntry, storeSink.entries[0]) {
		t.Fatalf("file entry = %#v, want %#v", fileEntry, storeSink.entries[0])
	}
}

func TestAuditWriterRecordSentAndRejected(t *testing.T) {
	t.Parallel()

	storeSink := &recordingAuditStore{}
	writer, err := NewAuditWriter("", storeSink)
	if err != nil {
		t.Fatalf("NewAuditWriter() error = %v", err)
	}
	writer.now = func() time.Time {
		return time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	}

	if err := writer.RecordSent(context.Background(), "sess-audit", testAuditEnvelope(t)); err != nil {
		t.Fatalf("RecordSent() error = %v", err)
	}
	if err := writer.RecordRejected(context.Background(), "sess-audit", testAuditEnvelope(t), "not_found"); err != nil {
		t.Fatalf("RecordRejected() error = %v", err)
	}

	if got, want := len(storeSink.entries), 2; got != want {
		t.Fatalf("len(store entries) = %d, want %d", got, want)
	}
	if got, want := storeSink.entries[0].Direction, AuditDirectionSent; got != want {
		t.Fatalf("entries[0].Direction = %q, want %q", got, want)
	}
	if got, want := storeSink.entries[1].Direction, AuditDirectionRejected; got != want {
		t.Fatalf("entries[1].Direction = %q, want %q", got, want)
	}
}

func TestAuditWriterRecordsDeliveredDirection(t *testing.T) {
	t.Parallel()

	t.Run("Should record delivered direction in the audit sink", func(t *testing.T) {
		storeSink := &recordingAuditStore{}
		writer, err := NewAuditWriter("", storeSink)
		if err != nil {
			t.Fatalf("NewAuditWriter() error = %v", err)
		}

		if err := writer.RecordDelivered(context.Background(), "sess-audit", testAuditEnvelope(t)); err != nil {
			t.Fatalf("RecordDelivered() error = %v", err)
		}
		if got, want := len(storeSink.entries), 1; got != want {
			t.Fatalf("len(store entries) = %d, want %d", got, want)
		}
		if got, want := storeSink.entries[0].Direction, AuditDirectionDelivered; got != want {
			t.Fatalf("entries[0].Direction = %q, want %q", got, want)
		}
	})
}

func TestAuditWriterPersistsTimelineMessagesForSayEnvelopesOnly(t *testing.T) {
	t.Parallel()

	t.Run("Should persist sent say envelopes to the timeline store", func(t *testing.T) {
		storeSink := &recordingAuditStore{}
		writer, err := NewAuditWriter("", storeSink)
		if err != nil {
			t.Fatalf("NewAuditWriter() error = %v", err)
		}
		recordedAt := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
		writer.now = func() time.Time { return recordedAt }

		if err := writer.RecordSent(context.Background(), "sess-audit", testSayAuditEnvelope(t)); err != nil {
			t.Fatalf("RecordSent(say) error = %v", err)
		}

		if got, want := len(storeSink.messages), 1; got != want {
			t.Fatalf("len(store messages) = %d, want %d", got, want)
		}
		if got, want := storeSink.messages[0].MessageID, "msg_say_01"; got != want {
			t.Fatalf("messages[0].MessageID = %q, want %q", got, want)
		}
		if got, want := storeSink.messages[0].Intent, "announce"; got != want {
			t.Fatalf("messages[0].Intent = %q, want %q", got, want)
		}
		if got, want := storeSink.messages[0].Text, "  hello builders  \n"; got != want {
			t.Fatalf("messages[0].Text = %q, want %q", got, want)
		}
	})

	t.Run("Should persist received say envelopes to the timeline store", func(t *testing.T) {
		storeSink := &recordingAuditStore{}
		writer, err := NewAuditWriter("", storeSink)
		if err != nil {
			t.Fatalf("NewAuditWriter() error = %v", err)
		}
		recordedAt := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
		writer.now = func() time.Time { return recordedAt }

		if err := writer.RecordReceived(context.Background(), "sess-remote", testSayAuditEnvelope(t)); err != nil {
			t.Fatalf("RecordReceived(say) error = %v", err)
		}

		if got, want := len(storeSink.messages), 1; got != want {
			t.Fatalf("len(store messages) = %d, want %d", got, want)
		}
		if got, want := storeSink.messages[0].SessionID, "sess-remote"; got != want {
			t.Fatalf("messages[0].SessionID = %q, want %q", got, want)
		}
		if got, want := storeSink.messages[0].MessageID, "msg_say_01"; got != want {
			t.Fatalf("messages[0].MessageID = %q, want %q", got, want)
		}
	})

	t.Run("Should ignore non-say envelopes when writing timeline messages", func(t *testing.T) {
		storeSink := &recordingAuditStore{}
		writer, err := NewAuditWriter("", storeSink)
		if err != nil {
			t.Fatalf("NewAuditWriter() error = %v", err)
		}

		if err := writer.RecordSent(context.Background(), "sess-audit", testAuditEnvelope(t)); err != nil {
			t.Fatalf("RecordSent(direct) error = %v", err)
		}
		if err := writer.RecordReceived(context.Background(), "sess-audit", testAuditEnvelope(t)); err != nil {
			t.Fatalf("RecordReceived(direct) error = %v", err)
		}

		if got := len(storeSink.messages); got != 0 {
			t.Fatalf("len(store messages) = %d, want 0", got)
		}
	})
}

func TestAuditWriterSkipsTimelineWriteWhenAuditStoreFails(t *testing.T) {
	t.Parallel()

	t.Run("Should not persist timeline rows after audit write failures", func(t *testing.T) {
		storeErr := errors.New("audit store unavailable")
		storeSink := &failingAuditStore{auditErr: storeErr}
		writer, err := NewAuditWriter("", storeSink)
		if err != nil {
			t.Fatalf("NewAuditWriter() error = %v", err)
		}

		err = writer.RecordSent(context.Background(), "sess-audit", testSayAuditEnvelope(t))
		if !errors.Is(err, storeErr) {
			t.Fatalf("RecordSent() error = %v, want wrapped store error", err)
		}
		if got, want := storeSink.auditCalls, 1; got != want {
			t.Fatalf("audit calls = %d, want %d", got, want)
		}
		if got := storeSink.messageCalls; got != 0 {
			t.Fatalf("message calls = %d, want 0", got)
		}
		if got := len(storeSink.messages); got != 0 {
			t.Fatalf("len(store messages) = %d, want 0", got)
		}
	})
}

func TestAuditWriterRecordTaskIngress(t *testing.T) {
	t.Parallel()

	storeSink := &recordingAuditStore{}
	writer, err := NewAuditWriter("", storeSink)
	if err != nil {
		t.Fatalf("NewAuditWriter() error = %v", err)
	}
	writer.now = func() time.Time {
		return time.Date(2026, 4, 14, 18, 15, 0, 0, time.UTC)
	}

	if err := writer.RecordTaskIngress(context.Background(), TaskIngressAudit{
		Action:    networkTaskActionEnqueue,
		Direction: AuditDirectionRejected,
		PeerID:    "reviewer.sess-ops",
		Channel:   "ops",
		RequestID: "req-enqueue-1",
		Reason:    "channel_mismatch",
		Payload: map[string]any{
			"task_id": "task-1",
		},
	}); err != nil {
		t.Fatalf("RecordTaskIngress() error = %v", err)
	}

	if got, want := len(storeSink.entries), 1; got != want {
		t.Fatalf("len(store entries) = %d, want %d", got, want)
	}
	entry := storeSink.entries[0]
	if got, want := entry.SessionID, "netpeer:reviewer.sess-ops"; got != want {
		t.Fatalf("entry.SessionID = %q, want %q", got, want)
	}
	if got, want := entry.Kind, networkTaskActionEnqueue; got != want {
		t.Fatalf("entry.Kind = %q, want %q", got, want)
	}
	if got, want := entry.Direction, AuditDirectionRejected; got != want {
		t.Fatalf("entry.Direction = %q, want %q", got, want)
	}
	if got, want := entry.Reason, "channel_mismatch"; got != want {
		t.Fatalf("entry.Reason = %q, want %q", got, want)
	}
	if entry.Size <= 0 {
		t.Fatalf("entry.Size = %d, want positive payload size", entry.Size)
	}
}

func TestAuditWriterAllowsFileOnlySinksWithoutTimelineNormalization(t *testing.T) {
	t.Parallel()

	t.Run("Should skip timeline normalization when no message store is configured", func(t *testing.T) {
		auditPath := filepath.Join(t.TempDir(), "logs", "network.audit")
		writer, err := NewAuditWriter(auditPath, nil)
		if err != nil {
			t.Fatalf("NewAuditWriter() error = %v", err)
		}

		if err := writer.RecordSent(context.Background(), "sess-audit", testInvalidSayAuditEnvelope(t)); err != nil {
			t.Fatalf("RecordSent(file-only invalid say) error = %v", err)
		}

		if _, err := os.Stat(auditPath); err != nil {
			t.Fatalf("Stat(%q) error = %v", auditPath, err)
		}
	})
}
func testAuditEnvelope(t *testing.T) Envelope {
	t.Helper()

	return Envelope{
		Protocol:      ProtocolV0,
		ID:            "msg_direct_01",
		Kind:          KindDirect,
		Channel:       "builders",
		From:          "coder.sess-audit",
		To:            stringPtr("reviewer.sess-xyz"),
		InteractionID: stringPtr("int_patch_42"),
		TS:            time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC).Unix(),
		Body:          mustRawJSON(t, map[string]any{"text": "Please inspect auth.go"}),
	}
}

func testSayAuditEnvelope(t *testing.T) Envelope {
	t.Helper()

	return Envelope{
		Protocol: ProtocolV0,
		ID:       "msg_say_01",
		Kind:     KindSay,
		Channel:  "builders",
		From:     "coder.sess-audit",
		TS:       time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC).Unix(),
		Body:     mustRawJSON(t, SayBody{Text: "  hello builders  \n", Intent: "announce"}),
	}
}

func testInvalidSayAuditEnvelope(t *testing.T) Envelope {
	t.Helper()

	return Envelope{
		Protocol: ProtocolV0,
		ID:       "msg_say_invalid_01",
		Kind:     KindSay,
		Channel:  "builders",
		From:     "coder.sess-audit",
		TS:       time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC).Unix(),
		Body:     mustRawJSON(t, []string{"not", "an", "object"}),
	}
}
