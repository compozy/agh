package filesnap

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFromPath(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "demo.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}

	snapshot, err := FromPath(path)
	if err != nil {
		t.Fatalf("FromPath() error = %v", err)
	}
	if snapshot.Size != int64(len("hello")) {
		t.Fatalf("FromPath().Size = %d, want %d", snapshot.Size, len("hello"))
	}
	if snapshot.ModTime.IsZero() {
		t.Fatal("FromPath().ModTime = zero, want populated")
	}

	if _, err := FromPath(filepath.Join(t.TempDir(), "missing.txt")); err == nil {
		t.Fatal("FromPath(missing) error = nil, want non-nil")
	}
}

func TestEqual(t *testing.T) {
	t.Parallel()

	modTime := time.Date(2026, 4, 6, 22, 30, 0, 0, time.UTC)
	left := map[string]Snapshot{
		"a": {ModTime: modTime, Size: 1},
		"b": {ModTime: modTime.Add(time.Second), Size: 2},
	}

	if !Equal(left, Clone(left)) {
		t.Fatal("Equal(clone) = false, want true")
	}

	if Equal(left, map[string]Snapshot{"a": left["a"]}) {
		t.Fatal("Equal(different sizes) = true, want false")
	}

	right := Clone(left)
	right["b"] = Snapshot{ModTime: modTime.Add(2 * time.Second), Size: 2}
	if Equal(left, right) {
		t.Fatal("Equal(different values) = true, want false")
	}
}

func TestCloneReturnsIndependentCopy(t *testing.T) {
	t.Parallel()

	modTime := time.Date(2026, 4, 6, 22, 45, 0, 0, time.UTC)
	original := map[string]Snapshot{
		"skill.md": {ModTime: modTime, Size: 10},
	}

	cloned := Clone(original)
	cloned["skill.md"] = Snapshot{ModTime: modTime.Add(time.Minute), Size: 42}

	if original["skill.md"].Size != 10 {
		t.Fatalf("original snapshot size = %d, want 10", original["skill.md"].Size)
	}
	if original["skill.md"].ModTime != modTime {
		t.Fatalf("original snapshot mod time = %v, want %v", original["skill.md"].ModTime, modTime)
	}

	if got := Clone(nil); got == nil || len(got) != 0 {
		t.Fatalf("Clone(nil) = %#v, want empty map", got)
	}
}
