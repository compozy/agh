package network

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
		t.Parallel()

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

func TestAuditWriterPersistsTimelineMessagesForRenderableEnvelopes(t *testing.T) {
	t.Parallel()

	recordedAt := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		sessionID string
		envelope  func(*testing.T) Envelope
		record    func(context.Context, *FileAuditWriter, string, Envelope) error
		assert    func(*testing.T, store.NetworkMessageEntry)
	}{
		{
			name:      "Should persist sent say envelopes to the timeline store",
			sessionID: "sess-audit",
			envelope:  testSayAuditEnvelope,
			record: func(ctx context.Context, writer *FileAuditWriter, sessionID string, envelope Envelope) error {
				return writer.RecordSent(ctx, sessionID, envelope)
			},
			assert: func(t *testing.T, entry store.NetworkMessageEntry) {
				t.Helper()

				if got, want := entry.Direction, AuditDirectionSent; got != want {
					t.Fatalf("messages[0].Direction = %q, want %q", got, want)
				}
				if got, want := entry.MessageID, "msg_say_01"; got != want {
					t.Fatalf("messages[0].MessageID = %q, want %q", got, want)
				}
				if got, want := entry.Intent, "announce"; got != want {
					t.Fatalf("messages[0].Intent = %q, want %q", got, want)
				}
				if got, want := entry.Text, "  hello builders  \n"; got != want {
					t.Fatalf("messages[0].Text = %q, want %q", got, want)
				}
			},
		},
		{
			name:      "Should persist received say envelopes to the timeline store",
			sessionID: "sess-remote",
			envelope:  testSayAuditEnvelope,
			record: func(ctx context.Context, writer *FileAuditWriter, sessionID string, envelope Envelope) error {
				return writer.RecordReceived(ctx, sessionID, envelope)
			},
			assert: func(t *testing.T, entry store.NetworkMessageEntry) {
				t.Helper()

				if got, want := entry.Direction, AuditDirectionReceived; got != want {
					t.Fatalf("messages[0].Direction = %q, want %q", got, want)
				}
				if got, want := entry.SessionID, "sess-remote"; got != want {
					t.Fatalf("messages[0].SessionID = %q, want %q", got, want)
				}
				if got, want := entry.MessageID, "msg_say_01"; got != want {
					t.Fatalf("messages[0].MessageID = %q, want %q", got, want)
				}
			},
		},
		{
			name:      "Should persist sent direct envelopes with addressing metadata",
			sessionID: "sess-audit",
			envelope:  testAuditEnvelope,
			record: func(ctx context.Context, writer *FileAuditWriter, sessionID string, envelope Envelope) error {
				return writer.RecordSent(ctx, sessionID, envelope)
			},
			assert: func(t *testing.T, entry store.NetworkMessageEntry) {
				t.Helper()

				if got, want := entry.Direction, AuditDirectionSent; got != want {
					t.Fatalf("messages[0].Direction = %q, want %q", got, want)
				}
				if got, want := entry.PeerFrom, "coder.sess-audit"; got != want {
					t.Fatalf("messages[0].PeerFrom = %q, want %q", got, want)
				}
				if got, want := entry.PeerTo, "reviewer.sess-xyz"; got != want {
					t.Fatalf("messages[0].PeerTo = %q, want %q", got, want)
				}
			},
		},
		{
			name:      "Should persist received direct envelopes with addressing metadata",
			sessionID: "sess-audit",
			envelope:  testReceivedDirectAuditEnvelope,
			record: func(ctx context.Context, writer *FileAuditWriter, sessionID string, envelope Envelope) error {
				return writer.RecordReceived(ctx, sessionID, envelope)
			},
			assert: func(t *testing.T, entry store.NetworkMessageEntry) {
				t.Helper()

				if got, want := entry.Direction, AuditDirectionReceived; got != want {
					t.Fatalf("messages[0].Direction = %q, want %q", got, want)
				}
				if got, want := entry.PeerFrom, "reviewer.sess-xyz"; got != want {
					t.Fatalf("messages[0].PeerFrom = %q, want %q", got, want)
				}
				if got, want := entry.PeerTo, "coder.sess-audit"; got != want {
					t.Fatalf("messages[0].PeerTo = %q, want %q", got, want)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			storeSink := &recordingAuditStore{}
			writer, err := NewAuditWriter("", storeSink)
			if err != nil {
				t.Fatalf("NewAuditWriter() error = %v", err)
			}
			writer.now = func() time.Time { return recordedAt }

			if err := tt.record(context.Background(), writer, tt.sessionID, tt.envelope(t)); err != nil {
				t.Fatalf("record envelope error = %v", err)
			}

			if got, want := len(storeSink.messages), 1; got != want {
				t.Fatalf("len(store messages) = %d, want %d", got, want)
			}

			tt.assert(t, storeSink.messages[0])
		})
	}
}

func TestAuditWriterRecordsCapabilityTransfersAsCapabilityAudits(t *testing.T) {
	t.Parallel()

	t.Run("ShouldRecordCapabilityTransfersAsCapabilityAudits", func(t *testing.T) {
		t.Parallel()

		storeSink := &recordingAuditStore{}
		writer, err := NewAuditWriter("", storeSink)
		if err != nil {
			t.Fatalf("NewAuditWriter() error = %v", err)
		}
		recordedAt := time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)
		writer.now = func() time.Time { return recordedAt }

		if err := writer.RecordReceived(
			context.Background(),
			"sess-audit",
			testCapabilityAuditEnvelope(t),
		); err != nil {
			t.Fatalf("RecordReceived(capability) error = %v", err)
		}

		if got, want := len(storeSink.entries), 1; got != want {
			t.Fatalf("len(store entries) = %d, want %d", got, want)
		}
		entry := storeSink.entries[0]
		if got, want := entry.Kind, string(KindCapability); got != want {
			t.Fatalf("entry.Kind = %q, want %q", got, want)
		}
		if got, want := entry.Direction, AuditDirectionReceived; got != want {
			t.Fatalf("entry.Direction = %q, want %q", got, want)
		}
		if got, want := len(storeSink.messages), 1; got != want {
			t.Fatalf("len(store timeline messages) = %d, want %d", got, want)
		}
		if got, want := storeSink.messages[0].Kind, string(KindCapability); got != want {
			t.Fatalf("messages[0].Kind = %q, want %q", got, want)
		}
		if got, want := storeSink.messages[0].PreviewText, "Review fix flow"; got != want {
			t.Fatalf("messages[0].PreviewText = %q, want %q", got, want)
		}
	})
}

func TestAuditWriterCoalescesRepeatedGreetHeartbeatsInTimeline(t *testing.T) {
	t.Run("Should coalesce repeated greet heartbeats in the operator timeline", func(t *testing.T) {
		t.Parallel()

		recordedAt := time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)
		storeSink := &recordingAuditStore{}
		writer, err := NewAuditWriter(
			"",
			storeSink,
			WithAuditWriterPresenceWindow(60*time.Second),
		)
		if err != nil {
			t.Fatalf("NewAuditWriter() error = %v", err)
		}
		callTimes := []time.Time{
			recordedAt,
			recordedAt.Add(30 * time.Second),
			recordedAt.Add(2 * time.Minute),
		}
		callIndex := 0
		writer.now = func() time.Time {
			value := callTimes[callIndex]
			callIndex++
			return value
		}

		first := testGreetAuditEnvelope(t, recordedAt, "msg_greet_01", "")
		second := testGreetAuditEnvelope(t, recordedAt.Add(30*time.Second), "msg_greet_02", "")
		third := testGreetAuditEnvelope(t, recordedAt.Add(2*time.Minute), "msg_greet_03", "")

		if err := writer.RecordSent(context.Background(), "sess-audit", first); err != nil {
			t.Fatalf("RecordSent(first greet) error = %v", err)
		}
		if err := writer.RecordSent(context.Background(), "sess-audit", second); err != nil {
			t.Fatalf("RecordSent(second greet) error = %v", err)
		}
		if err := writer.RecordSent(context.Background(), "sess-audit", third); err != nil {
			t.Fatalf("RecordSent(third greet) error = %v", err)
		}

		if got, want := len(storeSink.entries), 3; got != want {
			t.Fatalf("len(store audit entries) = %d, want %d", got, want)
		}
		if got, want := len(storeSink.messages), 2; got != want {
			t.Fatalf("len(store timeline messages) = %d, want %d", got, want)
		}
		if got, want := storeSink.messages[0].MessageID, "msg_greet_01"; got != want {
			t.Fatalf("messages[0].MessageID = %q, want %q", got, want)
		}
		if got, want := storeSink.messages[1].MessageID, "msg_greet_03"; got != want {
			t.Fatalf("messages[1].MessageID = %q, want %q", got, want)
		}
	})
}

func TestAuditWriterPrunesExpiredPresenceKeys(t *testing.T) {
	t.Run("Should evict stale presence keys while tracking greet suppression", func(t *testing.T) {
		t.Parallel()

		recordedAt := time.Date(2026, 4, 20, 12, 2, 0, 0, time.UTC)
		writer, err := NewAuditWriter(
			"",
			&recordingAuditStore{},
			WithAuditWriterPresenceWindow(time.Minute),
		)
		if err != nil {
			t.Fatalf("NewAuditWriter() error = %v", err)
		}

		writer.presence.lastSeen["stale"] = recordedAt.Add(-2 * time.Minute)
		writer.presence.lastSeen["fresh"] = recordedAt.Add(-30 * time.Second)

		entry := store.NetworkMessageEntry{
			Kind:      string(KindGreet),
			Direction: AuditDirectionReceived,
			Channel:   "builders",
			PeerFrom:  "reviewer.sess-audit",
			Timestamp: recordedAt,
		}
		if !writer.shouldWriteTimelineMessage(entry) {
			t.Fatal("shouldWriteTimelineMessage() = false, want true for first greet in window")
		}
		if _, ok := writer.presence.lastSeen["stale"]; ok {
			t.Fatalf("presence.lastSeen still contains stale key: %#v", writer.presence.lastSeen)
		}
		if _, ok := writer.presence.lastSeen["fresh"]; !ok {
			t.Fatalf("presence.lastSeen missing fresh key: %#v", writer.presence.lastSeen)
		}
	})
}

func TestAuditWriterFallsBackToDeterministicGreetSummaries(t *testing.T) {
	t.Run("Should use capability briefs for deterministic greet summaries", func(t *testing.T) {
		t.Parallel()

		recordedAt := time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)
		storeSink := &recordingAuditStore{}
		writer, err := NewAuditWriter("", storeSink)
		if err != nil {
			t.Fatalf("NewAuditWriter() error = %v", err)
		}

		if err := writer.RecordSent(
			context.Background(),
			"sess-audit",
			testGreetAuditEnvelope(t, recordedAt, "msg_greet_summary", ""),
		); err != nil {
			t.Fatalf("RecordSent(greet) error = %v", err)
		}

		if got, want := len(storeSink.messages), 1; got != want {
			t.Fatalf("len(store timeline messages) = %d, want %d", got, want)
		}
		if got := storeSink.messages[0].PreviewText; got == "" {
			t.Fatal("messages[0].PreviewText = empty, want fallback greet summary")
		}
		if got, want := storeSink.messages[0].PreviewText, "Reviewer ready for Review pull requests +1 more"; got != want {
			t.Fatalf("messages[0].PreviewText = %q, want %q", got, want)
		}
	})

	t.Run("Should ignore blank capability entries when counting remaining raw capabilities", func(t *testing.T) {
		t.Parallel()

		recordedAt := time.Date(2026, 4, 20, 12, 5, 0, 0, time.UTC)
		displayName := "Reviewer"
		storeSink := &recordingAuditStore{}
		writer, err := NewAuditWriter("", storeSink)
		if err != nil {
			t.Fatalf("NewAuditWriter() error = %v", err)
		}

		envelope := Envelope{
			Protocol: ProtocolV0,
			ID:       "msg_greet_summary_blank_caps",
			Kind:     KindGreet,
			Channel:  "builders",
			From:     "reviewer.sess-audit",
			TS:       recordedAt.Unix(),
			Body: mustRawJSON(t, GreetBody{
				PeerCard: PeerCard{
					PeerID:              "reviewer.sess-audit",
					DisplayName:         &displayName,
					ArtifactsSupported:  []string{string(KindCapability)},
					ProfilesSupported:   []string{ProtocolV0},
					TrustModesSupported: []string{"unverified"},
					Capabilities:        []string{"review-pr", " ", "\t", "draft-spec"},
				},
			}),
		}

		if err := writer.RecordSent(context.Background(), "sess-audit", envelope); err != nil {
			t.Fatalf("RecordSent(greet) error = %v", err)
		}
		if got, want := len(storeSink.messages), 1; got != want {
			t.Fatalf("len(store timeline messages) = %d, want %d", got, want)
		}
		if got, want := storeSink.messages[0].PreviewText, "Reviewer ready for review-pr +1 more"; got != want {
			t.Fatalf("messages[0].PreviewText = %q, want %q", got, want)
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

	t.Run("Should record task ingress", func(t *testing.T) {
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
	})

	t.Run("Should reject task ingress when no audit sink is configured", func(t *testing.T) {
		writer := &FileAuditWriter{}

		err := writer.RecordTaskIngress(context.Background(), TaskIngressAudit{
			Action:    networkTaskActionEnqueue,
			Direction: AuditDirectionRejected,
			PeerID:    "reviewer.sess-ops",
			Channel:   "ops",
			RequestID: "req-enqueue-2",
			Reason:    "channel_mismatch",
		})
		if err == nil || !strings.Contains(err.Error(), "audit sink is required") {
			t.Fatalf("RecordTaskIngress(no sink) error = %v, want audit sink validation", err)
		}
	})

	t.Run("Should fall back to the current time when the writer clock is unset", func(t *testing.T) {
		storeSink := &recordingAuditStore{}
		writer := &FileAuditWriter{store: storeSink}

		if err := writer.RecordTaskIngress(context.Background(), TaskIngressAudit{
			Action:    networkTaskActionEnqueue,
			Direction: AuditDirectionRejected,
			PeerID:    "reviewer.sess-ops",
			Channel:   "ops",
			RequestID: "req-enqueue-3",
			Reason:    "channel_mismatch",
		}); err != nil {
			t.Fatalf("RecordTaskIngress(nil now) error = %v", err)
		}
		if got, want := len(storeSink.entries), 1; got != want {
			t.Fatalf("len(store entries) = %d, want %d", got, want)
		}
		if storeSink.entries[0].Timestamp.IsZero() {
			t.Fatal("entry.Timestamp = zero, want fallback timestamp")
		}
	})

	t.Run("Should wrap task ingress normalization failures with operation context", func(t *testing.T) {
		writer := &FileAuditWriter{store: &recordingAuditStore{}}

		err := writer.RecordTaskIngress(context.Background(), TaskIngressAudit{
			Direction: AuditDirectionRejected,
			PeerID:    "reviewer.sess-ops",
			Channel:   "ops",
			RequestID: "req-enqueue-4",
		})
		if err == nil || !strings.Contains(err.Error(), "network: normalize task ingress audit entry") {
			t.Fatalf("RecordTaskIngress(normalize) error = %v, want normalize context", err)
		}
		if !strings.Contains(err.Error(), "network: validate audit entry") {
			t.Fatalf("RecordTaskIngress(normalize) error = %v, want validate context", err)
		}
	})

	t.Run("Should wrap task ingress sink failures with operation context", func(t *testing.T) {
		storeErr := errors.New("audit store unavailable")
		storeSink := &failingAuditStore{auditErr: storeErr}
		writer := &FileAuditWriter{
			path:  filepath.Join(t.TempDir(), "audit-dir"),
			store: storeSink,
			now: func() time.Time {
				return time.Date(2026, 4, 14, 18, 15, 0, 0, time.UTC)
			},
		}
		if err := os.MkdirAll(writer.path, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", writer.path, err)
		}

		err := writer.RecordTaskIngress(context.Background(), TaskIngressAudit{
			Action:    networkTaskActionEnqueue,
			Direction: AuditDirectionRejected,
			PeerID:    "reviewer.sess-ops",
			Channel:   "ops",
			RequestID: "req-enqueue-5",
			Reason:    "channel_mismatch",
		})
		if err == nil {
			t.Fatal("RecordTaskIngress(sink failures) error = nil, want joined error")
		}
		if !errors.Is(err, storeErr) {
			t.Fatalf("RecordTaskIngress(sink failures) error = %v, want wrapped store error", err)
		}
		if !strings.Contains(err.Error(), "network: append file audit entry") {
			t.Fatalf("RecordTaskIngress(sink failures) error = %v, want append context", err)
		}
		if !strings.Contains(err.Error(), "network: persist audit entry") {
			t.Fatalf("RecordTaskIngress(sink failures) error = %v, want persist context", err)
		}
	})
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
		Protocol: ProtocolV0,
		ID:       "msg_direct_01",
		Kind:     KindSay,
		Channel:  "builders",
		Surface:  surfacePtr(SurfaceDirect),
		DirectID: stringPtr("direct_0123456789abcdef0123456789abcdef"),
		From:     "coder.sess-audit",
		To:       stringPtr("reviewer.sess-xyz"),
		WorkID:   stringPtr("int_patch_42"),
		TS:       time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC).Unix(),
		Body:     mustRawJSON(t, map[string]any{"text": "Please inspect auth.go"}),
	}
}

func testReceivedDirectAuditEnvelope(t *testing.T) Envelope {
	t.Helper()

	return Envelope{
		Protocol: ProtocolV0,
		ID:       "msg_direct_02",
		Kind:     KindSay,
		Channel:  "builders",
		Surface:  surfacePtr(SurfaceDirect),
		DirectID: stringPtr("direct_0123456789abcdef0123456789abcdef"),
		From:     "reviewer.sess-xyz",
		To:       stringPtr("coder.sess-audit"),
		WorkID:   stringPtr("int_patch_43"),
		TS:       time.Date(2026, 4, 10, 12, 1, 0, 0, time.UTC).Unix(),
		Body:     mustRawJSON(t, map[string]any{"text": "Please confirm the auth fix landed"}),
	}
}

func testSayAuditEnvelope(t *testing.T) Envelope {
	t.Helper()

	return Envelope{
		Protocol: ProtocolV0,
		ID:       "msg_say_01",
		Kind:     KindSay,
		Channel:  "builders",
		Surface:  surfacePtr(SurfaceThread),
		ThreadID: stringPtr("thread_patch_42"),
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
		Surface:  surfacePtr(SurfaceThread),
		ThreadID: stringPtr("thread_patch_42"),
		From:     "coder.sess-audit",
		TS:       time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC).Unix(),
		Body:     mustRawJSON(t, []string{"not", "an", "object"}),
	}
}

func testCapabilityAuditEnvelope(t *testing.T) Envelope {
	t.Helper()

	return Envelope{
		Protocol: ProtocolV0,
		ID:       "msg_capability_01",
		Kind:     KindCapability,
		Channel:  "builders",
		Surface:  surfacePtr(SurfaceDirect),
		DirectID: stringPtr("direct_0123456789abcdef0123456789abcdef"),
		From:     "coder.sess-audit",
		To:       stringPtr("reviewer.sess-xyz"),
		WorkID:   stringPtr("int_capability_42"),
		TS:       time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC).Unix(),
		Body: mustCapabilityBodyJSON(t, CapabilityEnvelopePayload{
			ID:               "review-fix",
			Summary:          "Review fix flow",
			Outcome:          "A reusable review fix workflow.",
			Version:          "1.0.0",
			ExecutionOutline: []string{"Inspect the issue", "Draft the fix"},
			Requirements:     []string{"workspace-write"},
		}),
	}
}

func testGreetAuditEnvelope(t *testing.T, recordedAt time.Time, messageID string, summary string) Envelope {
	t.Helper()

	displayName := "Reviewer"
	return Envelope{
		Protocol: ProtocolV0,
		ID:       messageID,
		Kind:     KindGreet,
		Channel:  "builders",
		From:     "reviewer.sess-audit",
		TS:       recordedAt.Unix(),
		Body: mustRawJSON(t, GreetBody{
			PeerCard: PeerCard{
				PeerID:              "reviewer.sess-audit",
				DisplayName:         &displayName,
				ProfilesSupported:   []string{ProtocolV0},
				Capabilities:        []string{"review-pr", "draft-spec"},
				ArtifactsSupported:  []string{string(KindCapability)},
				TrustModesSupported: []string{},
				Ext: ExtensionMap{
					capabilityBriefExtKey: mustRawJSON(t, []capabilityBrief{
						{ID: "review-pr", Summary: "Review pull requests"},
						{ID: "draft-spec", Summary: "Draft technical specs"},
					}),
				},
			},
			Summary: summary,
		}),
	}
}
