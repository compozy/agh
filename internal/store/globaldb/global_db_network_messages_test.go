package globaldb

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestOpenGlobalDBCreatesNetworkMessageLogSchema(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)

	assertTablesPresent(t, globalDB.db, "network_message_log")
	assertTableColumns(t, globalDB.db, "network_message_log", []string{
		"message_id",
		"session_id",
		"channel",
		"peer_from",
		"kind",
		"intent",
		"text",
		"timestamp",
	})
	assertTableHasNoForeignKeys(t, globalDB.db, "network_message_log")
}

func TestGlobalDBWriteAndListNetworkMessages(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	recordedAt := time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)
	globalDB.now = func() time.Time { return recordedAt }

	if err := globalDB.WriteNetworkMessage(testutil.Context(t), store.NetworkMessageEntry{
		MessageID: "msg_say_01",
		SessionID: "sess-audit",
		Channel:   "builders",
		PeerFrom:  "coder.sess-audit",
		Kind:      "say",
		Intent:    "announce",
		Text:      "hello builders",
	}); err != nil {
		t.Fatalf("WriteNetworkMessage(first) error = %v", err)
	}
	if err := globalDB.WriteNetworkMessage(testutil.Context(t), store.NetworkMessageEntry{
		MessageID: "msg_say_01",
		SessionID: "",
		Channel:   "builders",
		PeerFrom:  "coder.sess-audit",
		Kind:      "say",
		Intent:    "announce",
		Text:      "hello builders",
		Timestamp: recordedAt.Add(time.Minute),
	}); err != nil {
		t.Fatalf("WriteNetworkMessage(duplicate) error = %v", err)
	}
	if err := globalDB.WriteNetworkMessage(testutil.Context(t), store.NetworkMessageEntry{
		MessageID: "msg_say_02",
		Channel:   "builders",
		PeerFrom:  "reviewer.sess-remote",
		Kind:      "say",
		Text:      "review in progress",
		Timestamp: recordedAt.Add(time.Minute),
	}); err != nil {
		t.Fatalf("WriteNetworkMessage(second) error = %v", err)
	}

	entries, err := globalDB.ListNetworkMessages(testutil.Context(t), store.NetworkMessageQuery{
		Channel: "builders",
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("ListNetworkMessages() error = %v", err)
	}
	if got, want := len(entries), 2; got != want {
		t.Fatalf("len(entries) = %d, want %d", got, want)
	}
	if got, want := entries[0].MessageID, "msg_say_01"; got != want {
		t.Fatalf("entries[0].MessageID = %q, want %q", got, want)
	}
	if got, want := entries[0].Intent, "announce"; got != want {
		t.Fatalf("entries[0].Intent = %q, want %q", got, want)
	}
	if got, want := entries[0].Timestamp, recordedAt; !got.Equal(want) {
		t.Fatalf("entries[0].Timestamp = %s, want %s", got, want)
	}
	if got, want := entries[1].PeerFrom, "reviewer.sess-remote"; got != want {
		t.Fatalf("entries[1].PeerFrom = %q, want %q", got, want)
	}
}

func TestGlobalDBNetworkMessageGuardClauses(t *testing.T) {
	t.Parallel()

	var nilDB *GlobalDB
	globalDB := openTestGlobalDB(t)
	if err := globalDB.Close(testutil.Context(t)); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	tests := []struct {
		name string
		run  func() error
		want error
	}{
		{
			name: "Should reject writes on a nil receiver",
			run: func() error {
				return nilDB.WriteNetworkMessage(testutil.Context(t), store.NetworkMessageEntry{})
			},
		},
		{
			name: "Should reject reads on a nil receiver",
			run: func() error {
				_, err := nilDB.ListNetworkMessages(testutil.Context(t), store.NetworkMessageQuery{})
				return err
			},
		},
		{
			name: "Should reject writes with a nil context",
			run: func() error {
				freshDB := openTestGlobalDB(t)
				defer func() {
					_ = freshDB.Close(testutil.Context(t))
				}()
				return freshDB.WriteNetworkMessage(nilGlobalContext(), store.NetworkMessageEntry{})
			},
		},
		{
			name: "Should reject reads with a nil context",
			run: func() error {
				freshDB := openTestGlobalDB(t)
				defer func() {
					_ = freshDB.Close(testutil.Context(t))
				}()
				_, err := freshDB.ListNetworkMessages(nilGlobalContext(), store.NetworkMessageQuery{})
				return err
			},
		},
		{
			name: "Should reject writes after the store is closed",
			run: func() error {
				return globalDB.WriteNetworkMessage(testutil.Context(t), store.NetworkMessageEntry{})
			},
			want: store.ErrClosed,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if tt.want != nil {
				if !errors.Is(err, tt.want) {
					t.Fatalf("error = %v, want %v", err, tt.want)
				}
				return
			}
			if err == nil {
				t.Fatal("error = nil, want non-nil")
			}
		})
	}
}

func TestGlobalDBListNetworkMessagesWrapsTimestampParseFailures(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	if _, err := globalDB.db.ExecContext(
		testutil.Context(t),
		`INSERT INTO network_message_log (
			message_id, session_id, channel, peer_from, kind, intent, text, timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"msg_bad_timestamp",
		nil,
		"builders",
		"coder.sess-audit",
		"say",
		nil,
		"hello",
		"not-a-timestamp",
	); err != nil {
		t.Fatalf("ExecContext(insert invalid network message) error = %v", err)
	}

	_, err := globalDB.ListNetworkMessages(testutil.Context(t), store.NetworkMessageQuery{Channel: "builders"})
	if err == nil {
		t.Fatal("ListNetworkMessages(invalid timestamp) error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "parse network message timestamp") {
		t.Fatalf("ListNetworkMessages(invalid timestamp) error = %v, want wrapped timestamp parse context", err)
	}
}
