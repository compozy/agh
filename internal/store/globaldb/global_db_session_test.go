package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/testutil"
)

func TestScanSessionInfoReadsStopFields(t *testing.T) {
	t.Parallel()

	t.Run("Should read stop fields and Soul provenance", func(t *testing.T) {
		t.Parallel()

		db := openScanSessionInfoDB(t)
		subprocessStartedAt := time.Date(2026, 4, 3, 12, 3, 0, 0, time.UTC)
		lastUpdateAt := time.Date(2026, 4, 3, 12, 4, 0, 0, time.UTC)
		row := db.QueryRowContext(context.Background(), `
		SELECT
			'sess-scan',
			'Demo',
			'coder',
			'claude',
			'ws-1',
			'builders',
			'user',
			NULL,
			NULL,
			0,
			NULL,
			NULL,
			false,
			'{}',
			'{}',
			'stopped',
			'acp-123',
			'timeout',
			'deadline exceeded',
			'process_exit',
			'redacted summary',
			'/tmp/crash.json',
			42,
			?,
			?,
			'stalled',
			'activity_timeout',
			'',
			'',
			NULL,
			'snap-scan',
			'sha256:scan',
			'sha256:parent',
			'env-scan',
			'local',
			'local',
			'instance-scan',
			'prepared',
			'{"local":true}',
			?,
			'sync failed',
			?,
			?`,
			formatTimestamp(subprocessStartedAt),
			formatTimestamp(lastUpdateAt),
			formatTimestamp(time.Date(2026, 4, 3, 12, 4, 30, 0, time.UTC)),
			formatTimestamp(time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)),
			formatTimestamp(time.Date(2026, 4, 3, 12, 5, 0, 0, time.UTC)),
		)

		info, err := scanSessionInfo(row)
		if err != nil {
			t.Fatalf("scanSessionInfo() error = %v", err)
		}
		if got, want := info.StopReason, store.StopTimeout; got != want {
			t.Fatalf("info.StopReason = %q, want %q", got, want)
		}
		if got, want := info.StopDetail, "deadline exceeded"; got != want {
			t.Fatalf("info.StopDetail = %q, want %q", got, want)
		}
		if info.Failure == nil {
			t.Fatal("info.Failure = nil, want failure")
		}
		if got, want := info.Failure.Kind, store.FailureProcess; got != want {
			t.Fatalf("info.Failure.Kind = %q, want %q", got, want)
		}
		if got, want := info.Failure.Summary, "redacted summary"; got != want {
			t.Fatalf("info.Failure.Summary = %q, want %q", got, want)
		}
		if got, want := info.Provider, "claude"; got != want {
			t.Fatalf("info.Provider = %q, want %q", got, want)
		}
		if got, want := info.Channel, "builders"; got != want {
			t.Fatalf("info.Channel = %q, want %q", got, want)
		}
		if info.ACPSessionID == nil || *info.ACPSessionID != "acp-123" {
			t.Fatalf("info.ACPSessionID = %#v, want acp-123", info.ACPSessionID)
		}
		if got, want := info.SoulSnapshotID, "snap-scan"; got != want {
			t.Fatalf("info.SoulSnapshotID = %q, want %q", got, want)
		}
		if got, want := info.SoulDigest, "sha256:scan"; got != want {
			t.Fatalf("info.SoulDigest = %q, want %q", got, want)
		}
		if got, want := info.ParentSoulDigest, "sha256:parent"; got != want {
			t.Fatalf("info.ParentSoulDigest = %q, want %q", got, want)
		}
		if info.Sandbox == nil {
			t.Fatal("info.Sandbox = nil, want sandbox metadata")
		}
		if info.Liveness == nil {
			t.Fatal("info.Liveness = nil, want liveness metadata")
		}
		if got, want := info.Liveness.SubprocessPID, 42; got != want {
			t.Fatalf("info.Liveness.SubprocessPID = %d, want %d", got, want)
		}
		if info.Liveness.SubprocessStartedAt == nil || !info.Liveness.SubprocessStartedAt.Equal(subprocessStartedAt) {
			t.Fatalf(
				"info.Liveness.SubprocessStartedAt = %#v, want %s",
				info.Liveness.SubprocessStartedAt,
				subprocessStartedAt,
			)
		}
		if info.Liveness.LastUpdateAt == nil || !info.Liveness.LastUpdateAt.Equal(lastUpdateAt) {
			t.Fatalf("info.Liveness.LastUpdateAt = %#v, want %s", info.Liveness.LastUpdateAt, lastUpdateAt)
		}
		if got, want := info.Liveness.StallState, "stalled"; got != want {
			t.Fatalf("info.Liveness.StallState = %q, want %q", got, want)
		}
		if got, want := info.Liveness.StallReason, "activity_timeout"; got != want {
			t.Fatalf("info.Liveness.StallReason = %q, want %q", got, want)
		}
		if got, want := info.Sandbox.SandboxID, "env-scan"; got != want {
			t.Fatalf("info.Sandbox.SandboxID = %q, want %q", got, want)
		}
		if got, want := info.Sandbox.LastSyncError, "sync failed"; got != want {
			t.Fatalf("info.Sandbox.LastSyncError = %q, want %q", got, want)
		}
	})
}

func TestScanSessionInfoHandlesNullStopReason(t *testing.T) {
	t.Parallel()

	t.Run("Should handle null stop reason and empty Soul provenance", func(t *testing.T) {
		t.Parallel()

		db := openScanSessionInfoDB(t)
		row := db.QueryRowContext(context.Background(), `
		SELECT
			'sess-null',
			NULL,
			'coder',
			'',
			'ws-1',
			'',
			'user',
			NULL,
			NULL,
			0,
			NULL,
			NULL,
			false,
			'{}',
			'{}',
			'active',
			NULL,
			NULL,
			NULL,
			NULL,
			'',
			'',
				0,
				NULL,
				NULL,
				'',
				'',
				'',
				'',
				NULL,
				NULL,
				'',
				'',
				'',
				'',
				'',
				'',
				'',
				'',
				NULL,
				'',
			?,
			?`,
			formatTimestamp(time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)),
			formatTimestamp(time.Date(2026, 4, 3, 12, 5, 0, 0, time.UTC)),
		)

		info, err := scanSessionInfo(row)
		if err != nil {
			t.Fatalf("scanSessionInfo() error = %v", err)
		}
		if info.StopReason != "" {
			t.Fatalf("info.StopReason = %q, want empty", info.StopReason)
		}
		if info.StopDetail != "" {
			t.Fatalf("info.StopDetail = %q, want empty", info.StopDetail)
		}
		if info.Provider != "" {
			t.Fatalf("info.Provider = %q, want empty", info.Provider)
		}
		if info.Channel != "" {
			t.Fatalf("info.Channel = %q, want empty", info.Channel)
		}
		if info.ACPSessionID != nil {
			t.Fatalf("info.ACPSessionID = %#v, want nil", info.ACPSessionID)
		}
		if info.SoulSnapshotID != "" || info.SoulDigest != "" || info.ParentSoulDigest != "" {
			t.Fatalf(
				"Soul provenance = %#v/%q/%q, want empty",
				info.SoulSnapshotID,
				info.SoulDigest,
				info.ParentSoulDigest,
			)
		}
	})
}

func TestScanSessionInfoRejectsInvalidSandboxLastSyncAt(t *testing.T) {
	t.Parallel()

	t.Run("Should reject invalid sandbox last sync timestamps", func(t *testing.T) {
		t.Parallel()

		db := openScanSessionInfoDB(t)
		row := db.QueryRowContext(context.Background(), `
		SELECT
			'sess-invalid-last-sync',
			'Demo',
			'coder',
			'claude',
			'ws-1',
			'builders',
			'user',
			NULL,
			NULL,
			0,
			NULL,
			NULL,
			false,
			'{}',
			'{}',
			'active',
			NULL,
			NULL,
			NULL,
			NULL,
			'',
			'',
				0,
				NULL,
				NULL,
				'',
				'',
				'',
				'',
				NULL,
				NULL,
				'',
				'',
				'env-invalid',
				'local',
			'local',
			'instance-invalid',
			'prepared',
			'{"local":true}',
			'not-a-timestamp',
			'',
			?,
			?`,
			formatTimestamp(time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)),
			formatTimestamp(time.Date(2026, 4, 3, 12, 5, 0, 0, time.UTC)),
		)

		_, err := scanSessionInfo(row)
		if err == nil {
			t.Fatal("scanSessionInfo() error = nil, want invalid sandbox_last_sync_at failure")
		}
		if got, want := err.Error(), `store: parse timestamp "not-a-timestamp"`; !strings.Contains(got, want) {
			t.Fatalf("scanSessionInfo() error = %v, want substring %q", err, want)
		}
	})
}

func TestScanSessionInfoRejectsStallStateWithoutReason(t *testing.T) {
	t.Parallel()

	t.Run("Should reject stall state without a reason", func(t *testing.T) {
		t.Parallel()

		db := openScanSessionInfoDB(t)
		row := db.QueryRowContext(context.Background(), `
		SELECT
			'sess-invalid-stall',
			'Demo',
			'coder',
			'claude',
			'ws-1',
			'builders',
			'user',
			NULL,
			NULL,
			0,
			NULL,
			NULL,
			false,
			'{}',
			'{}',
			'active',
			NULL,
			NULL,
			NULL,
			NULL,
			'',
			'',
			42,
			?,
			?,
			'stalled',
			'',
			'',
			'',
			NULL,
			NULL,
			'',
			'',
			'',
			'local',
			'',
			'',
			'',
			'',
			NULL,
			'',
			?,
			?`,
			formatTimestamp(time.Date(2026, 4, 3, 12, 3, 0, 0, time.UTC)),
			formatTimestamp(time.Date(2026, 4, 3, 12, 4, 0, 0, time.UTC)),
			formatTimestamp(time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)),
			formatTimestamp(time.Date(2026, 4, 3, 12, 5, 0, 0, time.UTC)),
		)

		_, err := scanSessionInfo(row)
		if err == nil {
			t.Fatal("scanSessionInfo() error = nil, want invalid stall reason failure")
		}
		if got, want := err.Error(), "store: session stall reason required when stall state is set"; !strings.Contains(
			got,
			want,
		) {
			t.Fatalf("scanSessionInfo() error = %v, want substring %q", err, want)
		}
	})
}

func TestGlobalDBAttachSessionRejectsStalledSessions(t *testing.T) {
	t.Parallel()

	t.Run("Should exclude stalled sessions from resumable attach paths", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		workspaceID := registerWorkspaceForGlobalTests(
			t,
			globalDB,
			"stalled-attach-workspace",
			filepath.Join(t.TempDir(), "stalled-attach-workspace"),
		)
		now := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
		if err := globalDB.RegisterSession(ctx, store.SessionInfo{
			ID:          "sess-stalled-attach",
			Name:        "Stalled Attach",
			AgentName:   "coder",
			Provider:    "claude",
			WorkspaceID: workspaceID,
			SessionType: defaultSessionType,
			State:       globalDBSessionStateActive,
			Liveness: &store.SessionLivenessMeta{
				SubprocessPID: 77,
				StallState:    store.SessionStallStateDetected,
				StallReason:   store.SessionStallReasonActivityTimeout,
			},
			CreatedAt: now,
			UpdatedAt: now,
		}); err != nil {
			t.Fatalf("RegisterSession() error = %v", err)
		}

		resumable, err := globalDB.ListSessions(ctx, store.SessionListQuery{Resumable: true})
		if err != nil {
			t.Fatalf("ListSessions(resumable) error = %v", err)
		}
		if len(resumable) != 0 {
			t.Fatalf("resumable sessions = %#v, want none for stalled session", resumable)
		}
		_, err = globalDB.AttachSession(ctx, store.SessionAttachRequest{
			SessionID:  "sess-stalled-attach",
			AttachedTo: "operator",
			Now:        now,
			TTL:        time.Minute,
		})
		if !errors.Is(err, store.ErrSessionNotAttachable) {
			t.Fatalf("AttachSession(stalled) error = %v, want ErrSessionNotAttachable", err)
		}
	})
}

func TestGlobalDBListSessionsSweepsExpiredAttachLocks(t *testing.T) {
	t.Parallel()

	t.Run("Should clear expired attach holder fields before returning sessions", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		workspaceID := registerWorkspaceForGlobalTests(
			t,
			globalDB,
			"expired-attach-workspace",
			filepath.Join(t.TempDir(), "expired-attach-workspace"),
		)
		now := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)
		expiredAt := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		if err := globalDB.RegisterSession(ctx, store.SessionInfo{
			ID:              "sess-expired-attach",
			Name:            "Expired Attach",
			AgentName:       "coder",
			Provider:        "claude",
			WorkspaceID:     workspaceID,
			SessionType:     defaultSessionType,
			State:           globalDBSessionStateActive,
			AttachedTo:      "operator-old",
			AttachExpiresAt: &expiredAt,
			CreatedAt:       now.Add(-time.Hour),
			UpdatedAt:       now.Add(-time.Hour),
		}); err != nil {
			t.Fatalf("RegisterSession() error = %v", err)
		}

		sessions, err := globalDB.ListSessions(ctx, store.SessionListQuery{ID: "sess-expired-attach"})
		if err != nil {
			t.Fatalf("ListSessions() error = %v", err)
		}
		if len(sessions) != 1 {
			t.Fatalf("len(sessions) = %d, want 1", len(sessions))
		}
		if sessions[0].AttachedTo != "" || sessions[0].AttachExpiresAt != nil {
			t.Fatalf(
				"session attach fields = %#v/%#v, want cleared",
				sessions[0].AttachedTo,
				sessions[0].AttachExpiresAt,
			)
		}
		cleared, err := globalDB.SweepExpiredSessionAttachLocks(ctx, now)
		if err != nil {
			t.Fatalf("SweepExpiredSessionAttachLocks(after list) error = %v", err)
		}
		if cleared != 0 {
			t.Fatalf("SweepExpiredSessionAttachLocks(after list) = %d, want 0", cleared)
		}
	})
}

func openScanSessionInfoDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open(sqliteDriverName, sqliteDSN(t.TempDir()+"/scan.db"))
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close(scan db) error = %v", err)
		}
	})
	return db
}
