package network

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/store"
)

type recordingAuditStore struct {
	entries []store.NetworkAuditEntry
}

func (s *recordingAuditStore) WriteNetworkAudit(_ context.Context, entry store.NetworkAuditEntry) error {
	s.entries = append(s.entries, entry)
	return nil
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

func testAuditEnvelope(t *testing.T) Envelope {
	t.Helper()

	return Envelope{
		Protocol:      ProtocolV0,
		ID:            "msg_direct_01",
		Kind:          KindDirect,
		Space:         "builders",
		From:          "coder.sess-audit",
		To:            stringPtr("reviewer.sess-xyz"),
		InteractionID: stringPtr("int_patch_42"),
		TS:            time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC).Unix(),
		Body:          mustRawJSON(t, map[string]any{"text": "Please inspect auth.go"}),
	}
}
