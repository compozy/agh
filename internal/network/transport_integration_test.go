//go:build integration

package network

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/store/globaldb"
	"github.com/pedronauck/agh/internal/testutil"
	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

func TestEmbeddedTransportLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(testutil.Context(t), 10*time.Second)
	defer cancel()

	transport, err := NewTransport(ctx, testNetworkConfig())
	if err != nil {
		t.Fatalf("NewTransport() error = %v", err)
	}

	subject, err := BroadcastSubject("builders")
	if err != nil {
		t.Fatalf("BroadcastSubject() error = %v", err)
	}

	received := make(chan string, 1)
	subscription, err := transport.Subscribe(subject, func(msg *nats.Msg) {
		received <- string(msg.Data)
	})
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	t.Cleanup(func() {
		_ = subscription.Unsubscribe()
	})

	if err := transport.Publish(ctx, subject, []byte("hello-network")); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	select {
	case got := <-received:
		if got != "hello-network" {
			t.Fatalf("received payload = %q, want %q", got, "hello-network")
		}
	case <-ctx.Done():
		t.Fatalf("timed out waiting for message: %v", ctx.Err())
	}

	if err := transport.Drain(ctx); err != nil {
		t.Fatalf("Drain() error = %v", err)
	}
	if err := transport.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestAuditWriterPersistsToDatabaseAndFileWithoutLeakingBrokerToken(t *testing.T) {
	ctx, cancel := context.WithTimeout(testutil.Context(t), 10*time.Second)
	defer cancel()

	transport, err := NewTransport(ctx, testNetworkConfig())
	if err != nil {
		t.Fatalf("NewTransport() error = %v", err)
	}
	t.Cleanup(func() {
		_ = transport.Shutdown(context.Background())
	})

	db, sessionID := openNetworkAuditIntegrationDB(t)
	auditPath := filepath.Join(t.TempDir(), "logs", "network.audit")

	writer, err := NewAuditWriter(auditPath, db)
	if err != nil {
		t.Fatalf("NewAuditWriter() error = %v", err)
	}
	writer.now = func() time.Time {
		return time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	}

	envelope := testAuditEnvelope(t)
	if err := writer.RecordSent(ctx, sessionID, envelope); err != nil {
		t.Fatalf("RecordSent() error = %v", err)
	}
	if err := writer.RecordRejected(ctx, sessionID, envelope, "busy"); err != nil {
		t.Fatalf("RecordRejected() error = %v", err)
	}

	entries, err := db.ListNetworkAudit(ctx, store.NetworkAuditQuery{SessionID: sessionID, Limit: 10})
	if err != nil {
		t.Fatalf("ListNetworkAudit() error = %v", err)
	}
	if got, want := len(entries), 2; got != want {
		t.Fatalf("len(entries) = %d, want %d", got, want)
	}

	fileData, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", auditPath, err)
	}
	if strings.Contains(string(fileData), transport.token) {
		t.Fatal("audit file leaked broker token")
	}

	for _, entry := range entries {
		combined := strings.Join([]string{
			entry.ID,
			entry.SessionID,
			entry.Direction,
			entry.Kind,
			entry.Channel,
			entry.PeerFrom,
			entry.PeerTo,
			entry.MessageID,
			entry.Reason,
		}, "|")
		if strings.Contains(combined, transport.token) {
			t.Fatalf("network audit entry leaked broker token: %#v", entry)
		}
	}
}

func openNetworkAuditIntegrationDB(t *testing.T) (*globaldb.GlobalDB, string) {
	t.Helper()

	ctx := testutil.Context(t)
	db, err := globaldb.OpenGlobalDB(ctx, filepath.Join(t.TempDir(), "agh.db"))
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close(context.Background())
	})

	workspaceRoot := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", workspaceRoot, err)
	}

	workspace := aghworkspace.Workspace{
		ID:        "ws-network-audit",
		RootDir:   workspaceRoot,
		Name:      "network-audit",
		CreatedAt: time.Date(2026, 4, 10, 11, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 10, 11, 0, 0, 0, time.UTC),
	}
	if err := db.InsertWorkspace(ctx, workspace); err != nil {
		t.Fatalf("InsertWorkspace() error = %v", err)
	}

	sessionID := "sess-network-audit"
	if err := db.RegisterSession(ctx, store.SessionInfo{
		ID:          sessionID,
		AgentName:   "coder",
		WorkspaceID: workspace.ID,
		State:       "active",
		CreatedAt:   time.Date(2026, 4, 10, 11, 30, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 10, 11, 30, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("RegisterSession() error = %v", err)
	}

	return db, sessionID
}
