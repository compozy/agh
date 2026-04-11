package globaldb

import (
	"errors"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestOpenGlobalDBCreatesNetworkAuditLogSchema(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)

	assertTablesPresent(t, globalDB.db, "network_audit_log")
	assertTableColumns(t, globalDB.db, "network_audit_log", []string{
		"id",
		"session_id",
		"direction",
		"kind",
		"space",
		"peer_from",
		"peer_to",
		"message_id",
		"reason",
		"size",
		"timestamp",
	})
}

func TestGlobalDBWriteAndListNetworkAudit(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	registerSessionForGlobalTests(t, globalDB, "sess-network-audit")

	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	globalDB.now = func() time.Time { return now }

	if err := globalDB.WriteNetworkAudit(testutil.Context(t), store.NetworkAuditEntry{
		SessionID: "sess-network-audit",
		Direction: "sent",
		Kind:      "direct",
		Space:     "builders",
		PeerFrom:  "coder.sess-network-audit",
		PeerTo:    "reviewer.sess-xyz",
		MessageID: "msg_direct_01",
		Size:      128,
	}); err != nil {
		t.Fatalf("WriteNetworkAudit(sent) error = %v", err)
	}

	if err := globalDB.WriteNetworkAudit(testutil.Context(t), store.NetworkAuditEntry{
		SessionID: "sess-network-audit",
		Direction: "rejected",
		Kind:      "receipt",
		Space:     "builders",
		PeerFrom:  "reviewer.sess-xyz",
		MessageID: "msg_receipt_01",
		Reason:    "not_found",
		Size:      64,
		Timestamp: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("WriteNetworkAudit(rejected) error = %v", err)
	}

	entries, err := globalDB.ListNetworkAudit(testutil.Context(t), store.NetworkAuditQuery{
		SessionID: "sess-network-audit",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListNetworkAudit() error = %v", err)
	}
	if got, want := len(entries), 2; got != want {
		t.Fatalf("len(entries) = %d, want %d", got, want)
	}

	if got, want := entries[0].Direction, "sent"; got != want {
		t.Fatalf("entries[0].Direction = %q, want %q", got, want)
	}
	if got, want := entries[0].Timestamp, now; !got.Equal(want) {
		t.Fatalf("entries[0].Timestamp = %s, want %s", got, want)
	}
	if got, want := entries[0].PeerTo, "reviewer.sess-xyz"; got != want {
		t.Fatalf("entries[0].PeerTo = %q, want %q", got, want)
	}

	if got, want := entries[1].Direction, "rejected"; got != want {
		t.Fatalf("entries[1].Direction = %q, want %q", got, want)
	}
	if got, want := entries[1].Reason, "not_found"; got != want {
		t.Fatalf("entries[1].Reason = %q, want %q", got, want)
	}
}

func TestGlobalDBNetworkAuditGuardClauses(t *testing.T) {
	t.Parallel()

	var nilDB *GlobalDB
	if err := nilDB.WriteNetworkAudit(testutil.Context(t), store.NetworkAuditEntry{}); err == nil {
		t.Fatal("WriteNetworkAudit(nil receiver) error = nil, want non-nil")
	}
	if _, err := nilDB.ListNetworkAudit(testutil.Context(t), store.NetworkAuditQuery{}); err == nil {
		t.Fatal("ListNetworkAudit(nil receiver) error = nil, want non-nil")
	}

	globalDB := openTestGlobalDB(t)
	if err := globalDB.WriteNetworkAudit(nilGlobalContext(), store.NetworkAuditEntry{}); err == nil {
		t.Fatal("WriteNetworkAudit(nil ctx) error = nil, want non-nil")
	}
	if _, err := globalDB.ListNetworkAudit(nilGlobalContext(), store.NetworkAuditQuery{}); err == nil {
		t.Fatal("ListNetworkAudit(nil ctx) error = nil, want non-nil")
	}
	if err := globalDB.Close(testutil.Context(t)); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := globalDB.WriteNetworkAudit(testutil.Context(t), store.NetworkAuditEntry{}); !errors.Is(err, store.ErrClosed) {
		t.Fatalf("WriteNetworkAudit(after close) error = %v, want ErrClosed", err)
	}
}
