package marketplace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathInsideRoot(t *testing.T) {
	t.Parallel()

	t.Run("Should reject targets that escape the root through symlinks", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		outside := t.TempDir()
		linkPath := filepath.Join(root, "escape")
		if err := os.Symlink(outside, linkPath); err != nil {
			t.Fatalf("os.Symlink() error = %v", err)
		}
		outsideSkill := filepath.Join(outside, "SKILL.md")
		if err := os.WriteFile(outsideSkill, []byte("outside"), 0o644); err != nil {
			t.Fatalf("os.WriteFile() error = %v", err)
		}

		_, err := PathInsideRoot(root, filepath.Join(linkPath, "SKILL.md"))
		if err == nil {
			t.Fatal("PathInsideRoot() error = nil, want path escape error")
		}
	})

	t.Run("Should preserve lexical targets that stay within the root", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		targetDir := filepath.Join(root, "review")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatalf("os.MkdirAll() error = %v", err)
		}
		targetPath := filepath.Join(targetDir, "SKILL.md")
		if err := os.WriteFile(targetPath, []byte("inside"), 0o644); err != nil {
			t.Fatalf("os.WriteFile() error = %v", err)
		}

		resolved, err := PathInsideRoot(root, targetPath)
		if err != nil {
			t.Fatalf("PathInsideRoot() error = %v", err)
		}
		if got := resolved; got != targetPath {
			t.Fatalf("PathInsideRoot() = %q, want %q", got, targetPath)
		}
	})

	t.Run("Should allow missing targets beneath the resolved root", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		targetPath := filepath.Join(root, "review", "SKILL.md")

		resolved, err := PathInsideRoot(root, targetPath)
		if err != nil {
			t.Fatalf("PathInsideRoot() error = %v", err)
		}
		if got := resolved; got != targetPath {
			t.Fatalf("PathInsideRoot() = %q, want %q", got, targetPath)
		}
	})
}
