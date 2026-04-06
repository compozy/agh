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
	meta := SessionMeta{
		ID:          "sess-meta",
		Name:        "Session Meta",
		AgentName:   "coder",
		WorkspaceID: "ws-meta",
		SessionType: "system",
		State:       "active",
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
		readBack.SessionType != meta.SessionType {
		t.Fatalf("ReadSessionMeta() = %#v, want %#v", readBack, meta)
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
