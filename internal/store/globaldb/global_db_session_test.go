package globaldb

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/store"
)

func TestScanSessionInfoReadsStopFields(t *testing.T) {
	t.Parallel()

	db := openScanSessionInfoDB(t)
	subprocessStartedAt := time.Date(2026, 4, 3, 12, 3, 0, 0, time.UTC)
	lastUpdateAt := time.Date(2026, 4, 3, 12, 4, 0, 0, time.UTC)
	row := db.QueryRowContext(context.Background(), `
		SELECT
			'sess-scan',
			'Demo',
			'coder',
			'ws-1',
			'builders',
			'user',
			'stopped',
			'acp-123',
			'timeout',
			'deadline exceeded',
			42,
			?,
			?,
			'stalled',
			'activity_timeout',
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
	if got, want := info.Channel, "builders"; got != want {
		t.Fatalf("info.Channel = %q, want %q", got, want)
	}
	if info.ACPSessionID == nil || *info.ACPSessionID != "acp-123" {
		t.Fatalf("info.ACPSessionID = %#v, want acp-123", info.ACPSessionID)
	}
	if info.Environment == nil {
		t.Fatal("info.Environment = nil, want environment metadata")
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
	if got, want := info.Environment.EnvironmentID, "env-scan"; got != want {
		t.Fatalf("info.Environment.EnvironmentID = %q, want %q", got, want)
	}
	if got, want := info.Environment.LastSyncError, "sync failed"; got != want {
		t.Fatalf("info.Environment.LastSyncError = %q, want %q", got, want)
	}
}

func TestScanSessionInfoHandlesNullStopReason(t *testing.T) {
	t.Parallel()

	db := openScanSessionInfoDB(t)
	row := db.QueryRowContext(context.Background(), `
		SELECT
			'sess-null',
			NULL,
			'coder',
			'ws-1',
			'',
			'user',
			'active',
			NULL,
			NULL,
			NULL,
			0,
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
	if info.Channel != "" {
		t.Fatalf("info.Channel = %q, want empty", info.Channel)
	}
	if info.ACPSessionID != nil {
		t.Fatalf("info.ACPSessionID = %#v, want nil", info.ACPSessionID)
	}
}

func TestScanSessionInfoRejectsInvalidEnvironmentLastSyncAt(t *testing.T) {
	t.Parallel()

	db := openScanSessionInfoDB(t)
	row := db.QueryRowContext(context.Background(), `
		SELECT
			'sess-invalid-last-sync',
			'Demo',
			'coder',
			'ws-1',
			'builders',
			'user',
			'active',
			NULL,
			NULL,
			NULL,
			0,
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
		t.Fatal("scanSessionInfo() error = nil, want invalid environment_last_sync_at failure")
	}
	if got, want := err.Error(), `store: parse timestamp "not-a-timestamp"`; !strings.Contains(got, want) {
		t.Fatalf("scanSessionInfo() error = %v, want substring %q", err, want)
	}
}

func TestScanSessionInfoRejectsStallStateWithoutReason(t *testing.T) {
	t.Parallel()

	db := openScanSessionInfoDB(t)
	row := db.QueryRowContext(context.Background(), `
		SELECT
			'sess-invalid-stall',
			'Demo',
			'coder',
			'ws-1',
			'builders',
			'user',
			'active',
			NULL,
			NULL,
			NULL,
			42,
			?,
			?,
			'stalled',
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
}

func openScanSessionInfoDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open(sqliteDriverName, sqliteDSN(t.TempDir()+"/scan.db"))
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}
