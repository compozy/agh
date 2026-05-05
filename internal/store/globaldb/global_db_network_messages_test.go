package globaldb

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestOpenGlobalDBCreatesNetworkTimelineLogSchema(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)

	assertTablesPresent(t, globalDB.db, "network_timeline_log")
	assertTableColumns(t, globalDB.db, "network_timeline_log", []string{
		"message_id",
		"session_id",
		"channel",
		"direction",
		"peer_from",
		"peer_to",
		"kind",
		"interaction_id",
		"reply_to",
		"trace_id",
		"causation_id",
		"intent",
		"text",
		"preview_text",
		"body_json",
		"timestamp",
	})
	assertTableHasNoForeignKeys(t, globalDB.db, "network_timeline_log")
}

func TestGlobalDBWriteAndListNetworkMessages(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	recordedAt := time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)
	globalDB.now = func() time.Time { return recordedAt }

	if err := globalDB.WriteNetworkMessage(testutil.Context(t), store.NetworkMessageEntry{
		MessageID:   "msg_say_01",
		SessionID:   "sess-audit",
		Channel:     "builders",
		Direction:   "sent",
		PeerFrom:    "coder.sess-audit",
		Kind:        "say",
		Intent:      "announce",
		Text:        "hello builders",
		PreviewText: "hello builders",
		Body:        []byte(`{"text":"hello builders","intent":"announce"}`),
	}); err != nil {
		t.Fatalf("WriteNetworkMessage(first) error = %v", err)
	}
	if err := globalDB.WriteNetworkMessage(testutil.Context(t), store.NetworkMessageEntry{
		MessageID:   "msg_say_01",
		Channel:     "builders",
		Direction:   "sent",
		PeerFrom:    "coder.sess-audit",
		Kind:        "say",
		Intent:      "announce",
		Text:        "hello builders",
		PreviewText: "hello builders",
		Body:        []byte(`{"text":"hello builders","intent":"announce"}`),
		Timestamp:   recordedAt.Add(time.Minute),
	}); err != nil {
		t.Fatalf("WriteNetworkMessage(duplicate) error = %v", err)
	}
	if err := globalDB.WriteNetworkMessage(testutil.Context(t), store.NetworkMessageEntry{
		MessageID:   "msg_say_02",
		Channel:     "builders",
		Direction:   "received",
		PeerFrom:    "reviewer.sess-remote",
		PeerTo:      "coder.sess-audit",
		Kind:        "direct",
		WorkID:      "ix-1",
		Text:        "review in progress",
		PreviewText: "review in progress",
		Body:        []byte(`{"text":"review in progress"}`),
		Timestamp:   recordedAt.Add(time.Minute),
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
	if got, want := entries[0].Direction, "sent"; got != want {
		t.Fatalf("entries[0].Direction = %q, want %q", got, want)
	}
	if got, want := entries[0].Timestamp, recordedAt; !got.Equal(want) {
		t.Fatalf("entries[0].Timestamp = %s, want %s", got, want)
	}
	if got, want := entries[1].PeerFrom, "reviewer.sess-remote"; got != want {
		t.Fatalf("entries[1].PeerFrom = %q, want %q", got, want)
	}
	if got, want := entries[1].PeerTo, "coder.sess-audit"; got != want {
		t.Fatalf("entries[1].PeerTo = %q, want %q", got, want)
	}
	if got, want := entries[1].WorkID, "ix-1"; got != want {
		t.Fatalf("entries[1].WorkID = %q, want %q", got, want)
	}
	if got, want := string(entries[1].Body), `{"text":"review in progress"}`; got != want {
		t.Fatalf("entries[1].Body = %q, want %q", got, want)
	}
}

func TestGlobalDBListNetworkMessagesSupportsMessageIDCursors(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	recordedAt := time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)

	entries := []store.NetworkMessageEntry{
		{
			MessageID:   "msg-1",
			Channel:     "builders",
			Direction:   "sent",
			PeerFrom:    "peer-a",
			Kind:        "say",
			PreviewText: "one",
			Body:        []byte(`{"text":"one"}`),
			Timestamp:   recordedAt,
		},
		{
			MessageID:   "msg-2a",
			Channel:     "builders",
			Direction:   "sent",
			PeerFrom:    "peer-a",
			Kind:        "say",
			PreviewText: "two a",
			Body:        []byte(`{"text":"two a"}`),
			Timestamp:   recordedAt.Add(time.Minute),
		},
		{
			MessageID:   "msg-2b",
			Channel:     "builders",
			Direction:   "sent",
			PeerFrom:    "peer-a",
			Kind:        "say",
			PreviewText: "two b",
			Body:        []byte(`{"text":"two b"}`),
			Timestamp:   recordedAt.Add(time.Minute),
		},
		{
			MessageID:   "msg-3",
			Channel:     "builders",
			Direction:   "received",
			PeerFrom:    "peer-b",
			PeerTo:      "peer-a",
			Kind:        "direct",
			PreviewText: "three",
			Body:        []byte(`{"text":"three"}`),
			Timestamp:   recordedAt.Add(2 * time.Minute),
		},
		{
			MessageID:   "msg-4",
			Channel:     "retro",
			Direction:   "received",
			PeerFrom:    "peer-c",
			PeerTo:      "peer-d",
			Kind:        "direct",
			PreviewText: "four",
			Body:        []byte(`{"text":"four"}`),
			Timestamp:   recordedAt.Add(3 * time.Minute),
		},
	}
	for _, entry := range entries {
		if err := globalDB.WriteNetworkMessage(testutil.Context(t), entry); err != nil {
			t.Fatalf("WriteNetworkMessage(%q) error = %v", entry.MessageID, err)
		}
	}

	before, err := globalDB.ListNetworkMessages(testutil.Context(t), store.NetworkMessageQuery{
		Channel:         "builders",
		BeforeMessageID: "msg-2b",
		Limit:           10,
	})
	if err != nil {
		t.Fatalf("ListNetworkMessages(before) error = %v", err)
	}
	if got, want := len(before), 2; got != want {
		t.Fatalf("len(before) = %d, want %d", got, want)
	}
	if got, want := before[0].MessageID, "msg-1"; got != want {
		t.Fatalf("before[0].MessageID = %q, want %q", got, want)
	}
	if got, want := before[1].MessageID, "msg-2a"; got != want {
		t.Fatalf("before[1].MessageID = %q, want %q", got, want)
	}

	after, err := globalDB.ListNetworkMessages(testutil.Context(t), store.NetworkMessageQuery{
		Channel:        "builders",
		AfterMessageID: "msg-2a",
		Limit:          10,
	})
	if err != nil {
		t.Fatalf("ListNetworkMessages(after) error = %v", err)
	}
	if got, want := len(after), 2; got != want {
		t.Fatalf("len(after) = %d, want %d", got, want)
	}
	if got, want := after[0].MessageID, "msg-2b"; got != want {
		t.Fatalf("after[0].MessageID = %q, want %q", got, want)
	}
	if got, want := after[1].MessageID, "msg-3"; got != want {
		t.Fatalf("after[1].MessageID = %q, want %q", got, want)
	}

	_, err = globalDB.ListNetworkMessages(testutil.Context(t), store.NetworkMessageQuery{
		Channel:         "builders",
		BeforeMessageID: "msg-4",
		Limit:           10,
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("ListNetworkMessages(cross-channel cursor) error = %v, want sql.ErrNoRows", err)
	}

	_, err = globalDB.ListNetworkMessages(testutil.Context(t), store.NetworkMessageQuery{
		PeerID:          "peer-a",
		DirectedOnly:    true,
		BeforeMessageID: "msg-4",
		Limit:           10,
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("ListNetworkMessages(cross-peer cursor) error = %v, want sql.ErrNoRows", err)
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
		`INSERT INTO network_timeline_log (
			message_id,
			session_id,
			channel,
			direction,
			peer_from,
			peer_to,
			kind,
			interaction_id,
			reply_to,
			trace_id,
			causation_id,
			intent,
			text,
			preview_text,
			body_json,
			timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"msg_bad_timestamp",
		nil,
		"builders",
		"sent",
		"coder.sess-audit",
		nil,
		"say",
		nil,
		nil,
		nil,
		nil,
		nil,
		"hello",
		"hello",
		`{"text":"hello"}`,
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

func TestGlobalDBWriteNetworkMessageRejectsNonCanonicalDirection(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	err := globalDB.WriteNetworkMessage(testutil.Context(t), store.NetworkMessageEntry{
		MessageID:   "msg_bad_direction",
		Channel:     "builders",
		Direction:   " sent ",
		PeerFrom:    "coder.sess-audit",
		Kind:        "say",
		PreviewText: "hello",
		Body:        []byte(`{"text":"hello"}`),
	})
	if err == nil {
		t.Fatal("WriteNetworkMessage() error = nil, want non-canonical direction rejection")
	}
	if !strings.Contains(err.Error(), `unsupported network message direction " sent "`) {
		t.Fatalf("WriteNetworkMessage() error = %v, want direction validation error", err)
	}
}
