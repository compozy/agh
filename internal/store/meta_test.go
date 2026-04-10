package store

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestWriteSessionMetaAndReadBack(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), SessionMetaName)
	stopReason := StopHookStopped
	meta := SessionMeta{
		ID:          "sess-meta",
		Name:        "Session Meta",
		AgentName:   "coder",
		WorkspaceID: "ws-meta",
		SessionType: "system",
		State:       "stopped",
		StopReason:  &stopReason,
		StopDetail:  "hook denied continuation",
		CreatedAt:   time.Date(2026, 4, 3, 17, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 3, 17, 1, 0, 0, time.UTC),
	}

	if err := WriteSessionMeta(path, meta); err != nil {
		t.Fatalf("WriteSessionMeta() error = %v", err)
	}

	readBack, err := ReadSessionMeta(path)
	if err != nil {
		t.Fatalf("ReadSessionMeta() error = %v", err)
	}
	if readBack.ID != meta.ID ||
		readBack.AgentName != meta.AgentName ||
		readBack.WorkspaceID != meta.WorkspaceID ||
		readBack.State != meta.State ||
		readBack.SessionType != meta.SessionType ||
		readBack.StopDetail != meta.StopDetail {
		t.Fatalf("ReadSessionMeta() = %#v, want %#v", readBack, meta)
	}
	if readBack.StopReason == nil {
		t.Fatal("ReadSessionMeta().StopReason = nil, want non-nil")
	}
	if *readBack.StopReason != *meta.StopReason {
		t.Fatalf("ReadSessionMeta().StopReason = %q, want %q", *readBack.StopReason, *meta.StopReason)
	}
}

func TestWriteSessionMetaConcurrentWritesDoNotCorruptFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), SessionMetaName)
	base := SessionMeta{
		ID:          "sess-meta-concurrent",
		AgentName:   "coder",
		WorkspaceID: "ws-meta-concurrent",
		State:       "active",
		CreatedAt:   time.Date(2026, 4, 3, 18, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 3, 18, 0, 0, 0, time.UTC),
	}

	var wg sync.WaitGroup
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			meta := base
			meta.Name = filepath.Base(filepath.Join("name", time.Date(2026, 4, 3, 18, 0, i, 0, time.UTC).Format(time.RFC3339Nano)))
			meta.UpdatedAt = base.UpdatedAt.Add(time.Duration(i) * time.Second)
			if err := WriteSessionMeta(path, meta); err != nil {
				t.Errorf("WriteSessionMeta() error = %v", err)
			}
		}(i)
	}
	wg.Wait()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if len(data) == 0 {
		t.Fatal("meta file is empty after concurrent writes")
	}

	meta, err := ReadSessionMeta(path)
	if err != nil {
		t.Fatalf("ReadSessionMeta() error = %v", err)
	}
	if meta.ID != base.ID || meta.AgentName != base.AgentName || meta.WorkspaceID != base.WorkspaceID {
		t.Fatalf(
			"ReadSessionMeta() = %#v, want id=%q agent=%q workspace_id=%q",
			meta,
			base.ID,
			base.AgentName,
			base.WorkspaceID,
		)
	}
}

func TestReadSessionMetaLegacyStopFieldsOmitted(t *testing.T) {
	t.Run("Should handle legacy stop fields omitted", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), SessionMetaName)
		payload := []byte(`{
  "id": "sess-legacy",
  "name": "Legacy Session",
  "agent_name": "coder",
  "workspace_id": "ws-legacy",
  "session_type": "user",
  "state": "stopped",
  "created_at": "2026-04-03T17:00:00Z",
  "updated_at": "2026-04-03T17:01:00Z"
}
`)
		if err := os.WriteFile(path, payload, 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		meta, err := ReadSessionMeta(path)
		if err != nil {
			t.Fatalf("ReadSessionMeta() error = %v", err)
		}
		if meta.StopReason != nil {
			t.Fatalf("ReadSessionMeta().StopReason = %v, want nil", *meta.StopReason)
		}
		if meta.StopDetail != "" {
			t.Fatalf("ReadSessionMeta().StopDetail = %q, want empty", meta.StopDetail)
		}
	})
}
