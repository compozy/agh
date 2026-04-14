package registry

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type tarEntry struct {
	name     string
	content  string
	typeflag byte
	linkname string
}

func TestExtractArchive_ValidArchiveProducesDirectoryStructure(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	archive := mustTarGz(t, []tarEntry{
		{name: "review/assets", typeflag: tar.TypeDir},
		{name: "review/SKILL.md", content: "name: review\n"},
		{name: "review/docs/guide.md", content: "guide"},
		{name: "review/scripts/run.sh", content: "echo ok\n"},
	})

	if err := ExtractArchive(bytes.NewReader(archive), root); err != nil {
		t.Fatalf("ExtractArchive() error = %v", err)
	}

	checks := []struct {
		path    string
		content string
	}{
		{path: filepath.Join(root, "review", "SKILL.md"), content: "name: review\n"},
		{path: filepath.Join(root, "review", "docs", "guide.md"), content: "guide"},
		{path: filepath.Join(root, "review", "scripts", "run.sh"), content: "echo ok\n"},
	}

	for _, check := range checks {
		data, err := os.ReadFile(check.path)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", check.path, err)
		}
		if got := string(data); got != check.content {
			t.Fatalf("ReadFile(%q) = %q, want %q", check.path, got, check.content)
		}
	}
}

func TestExtractArchive_EnforcesLimitsAndRejectsUnsafeEntries(t *testing.T) {
	t.Parallel()

	t.Run("decompressed-size", func(t *testing.T) {
		root := t.TempDir()
		archive := mustTarGz(t, []tarEntry{
			{name: "review/SKILL.md", content: "0123456789"},
		})

		err := extractArchive(bytes.NewReader(archive), root, extractLimits{
			maxDecompressedSize: 5,
			maxFileCount:        DefaultMaxFileCount,
		})
		if !errors.Is(err, errArchiveTooLarge) {
			t.Fatalf("extractArchive(size limit) error = %v, want %v", err, errArchiveTooLarge)
		}

		target := filepath.Join(root, "review", "SKILL.md")
		info, statErr := os.Stat(target)
		switch {
		case statErr == nil && info.Size() > 5:
			t.Fatalf("partial file size = %d, want <= 5", info.Size())
		case statErr != nil && !errors.Is(statErr, os.ErrNotExist):
			t.Fatalf("Stat(%q) error = %v", target, statErr)
		}
	})

	t.Run("file-count", func(t *testing.T) {
		root := t.TempDir()
		archive := mustTarGz(t, []tarEntry{
			{name: "one.txt", content: "one"},
			{name: "two.txt", content: "two"},
			{name: "three.txt", content: "three"},
		})

		err := extractArchive(bytes.NewReader(archive), root, extractLimits{
			maxDecompressedSize: DefaultMaxDecompressedSize,
			maxFileCount:        2,
		})
		if !errors.Is(err, errArchiveTooManyFiles) {
			t.Fatalf("extractArchive(file count) error = %v, want %v", err, errArchiveTooManyFiles)
		}
	})

	t.Run("symlink", func(t *testing.T) {
		archive := mustTarGz(t, []tarEntry{
			{name: "review/link", typeflag: tar.TypeSymlink, linkname: "../target"},
		})

		err := ExtractArchive(bytes.NewReader(archive), t.TempDir())
		if err == nil {
			t.Fatal("ExtractArchive(symlink) error = nil, want failure")
		}
		if !strings.Contains(err.Error(), "unsupported archive entry type") {
			t.Fatalf("ExtractArchive(symlink) error = %v, want unsupported entry context", err)
		}
	})

	t.Run("path-traversal", func(t *testing.T) {
		archive := mustTarGz(t, []tarEntry{
			{name: "../escape.txt", content: "nope"},
		})

		err := ExtractArchive(bytes.NewReader(archive), t.TempDir())
		if err == nil {
			t.Fatal("ExtractArchive(traversal) error = nil, want failure")
		}
		if !strings.Contains(err.Error(), "escapes the extraction root") {
			t.Fatalf("ExtractArchive(traversal) error = %v, want traversal context", err)
		}
	})

	t.Run("empty-destination", func(t *testing.T) {
		err := ExtractArchive(bytes.NewReader(mustTarGz(t, []tarEntry{
			{name: "review/SKILL.md", content: "name: review\n"},
		})), "")
		if err == nil {
			t.Fatal("ExtractArchive(empty dest) error = nil, want failure")
		}
		if !strings.Contains(err.Error(), "destination root is required") {
			t.Fatalf("ExtractArchive(empty dest) error = %v, want destination validation", err)
		}
	})

	t.Run("invalid-gzip", func(t *testing.T) {
		err := ExtractArchive(strings.NewReader("not-a-gzip-stream"), t.TempDir())
		if err == nil {
			t.Fatal("ExtractArchive(invalid gzip) error = nil, want failure")
		}
		if !strings.Contains(err.Error(), "open gzip stream") {
			t.Fatalf("ExtractArchive(invalid gzip) error = %v, want gzip-open context", err)
		}
	})
}

func TestPathWithinRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	target, err := PathWithinRoot(root, filepath.Join("review", "SKILL.md"))
	if err != nil {
		t.Fatalf("PathWithinRoot(valid) error = %v", err)
	}
	if !strings.HasPrefix(target, root+string(filepath.Separator)) {
		t.Fatalf("PathWithinRoot(valid) = %q, want path under %q", target, root)
	}

	if _, err := PathWithinRoot(root, filepath.Join("..", "escape")); err == nil {
		t.Fatal("PathWithinRoot(escape) error = nil, want failure")
	}
}

func TestCleanArchiveEntryPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		entry string
		want  string
		ok    bool
	}{
		{name: "valid", entry: "review\\SKILL.md", want: "review/SKILL.md", ok: true},
		{name: "empty", entry: ""},
		{name: "absolute", entry: "/tmp/skill.md"},
		{name: "traversal", entry: "../escape.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CleanArchiveEntryPath(tt.entry)
			if tt.ok {
				if err != nil {
					t.Fatalf("CleanArchiveEntryPath(%q) error = %v", tt.entry, err)
				}
				if got != tt.want {
					t.Fatalf("CleanArchiveEntryPath(%q) = %q, want %q", tt.entry, got, tt.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("CleanArchiveEntryPath(%q) error = nil, want failure", tt.entry)
			}
		})
	}
}

func TestMoveInstalledDir(t *testing.T) {
	t.Parallel()

	t.Run("install-into-new-target", func(t *testing.T) {
		parent := t.TempDir()
		source := filepath.Join(parent, "source")
		target := filepath.Join(parent, "target")
		writeTestFile(t, filepath.Join(source, "SKILL.md"), "new")

		if err := MoveInstalledDir(source, target, false); err != nil {
			t.Fatalf("MoveInstalledDir(new target) error = %v", err)
		}

		data, err := os.ReadFile(filepath.Join(target, "SKILL.md"))
		if err != nil {
			t.Fatalf("ReadFile(target) error = %v", err)
		}
		if got := string(data); got != "new" {
			t.Fatalf("target content = %q, want new", got)
		}
	})

	t.Run("replace-into-new-target", func(t *testing.T) {
		parent := t.TempDir()
		source := filepath.Join(parent, "source")
		target := filepath.Join(parent, "target")
		writeTestFile(t, filepath.Join(source, "SKILL.md"), "new")

		if err := MoveInstalledDir(source, target, true); err != nil {
			t.Fatalf("MoveInstalledDir(replace new target) error = %v", err)
		}

		data, err := os.ReadFile(filepath.Join(target, "SKILL.md"))
		if err != nil {
			t.Fatalf("ReadFile(target) error = %v", err)
		}
		if got := string(data); got != "new" {
			t.Fatalf("target content = %q, want new", got)
		}
	})

	t.Run("replace-existing", func(t *testing.T) {
		parent := t.TempDir()
		source := filepath.Join(parent, "source")
		target := filepath.Join(parent, "target")
		writeTestFile(t, filepath.Join(source, "SKILL.md"), "new")
		writeTestFile(t, filepath.Join(target, "SKILL.md"), "old")

		if err := MoveInstalledDir(source, target, true); err != nil {
			t.Fatalf("MoveInstalledDir(replace) error = %v", err)
		}

		data, err := os.ReadFile(filepath.Join(target, "SKILL.md"))
		if err != nil {
			t.Fatalf("ReadFile(target) error = %v", err)
		}
		if got := string(data); got != "new" {
			t.Fatalf("target content = %q, want new", got)
		}
	})

	t.Run("reject-existing-when-replace-disabled", func(t *testing.T) {
		parent := t.TempDir()
		source := filepath.Join(parent, "source")
		target := filepath.Join(parent, "target")
		writeTestFile(t, filepath.Join(source, "SKILL.md"), "new")
		writeTestFile(t, filepath.Join(target, "SKILL.md"), "old")

		if err := MoveInstalledDir(source, target, false); err == nil {
			t.Fatal("MoveInstalledDir(no replace) error = nil, want failure")
		}
	})

	t.Run("restore-backup-when-replacement-fails", func(t *testing.T) {
		parent := t.TempDir()
		source := filepath.Join(parent, "missing-source")
		target := filepath.Join(parent, "target")
		writeTestFile(t, filepath.Join(target, "SKILL.md"), "old")

		if err := MoveInstalledDir(source, target, true); err == nil {
			t.Fatal("MoveInstalledDir(missing source) error = nil, want failure")
		}

		data, err := os.ReadFile(filepath.Join(target, "SKILL.md"))
		if err != nil {
			t.Fatalf("ReadFile(target after rollback) error = %v", err)
		}
		if got := string(data); got != "old" {
			t.Fatalf("target content after rollback = %q, want old", got)
		}
	})
}

func TestDefaultExtractLimits(t *testing.T) {
	t.Parallel()

	if got, want := DefaultMaxDecompressedSize, int64(500*1024*1024); got != want {
		t.Fatalf("DefaultMaxDecompressedSize = %d, want %d", got, want)
	}
	if got, want := DefaultMaxFileCount, 10000; got != want {
		t.Fatalf("DefaultMaxFileCount = %d, want %d", got, want)
	}
}

func TestCountingLimitWriter(t *testing.T) {
	t.Parallel()

	t.Run("nil-total", func(t *testing.T) {
		writer := &countingLimitWriter{limit: 10}
		if _, err := writer.Write([]byte("abc")); err == nil {
			t.Fatal("countingLimitWriter.Write(nil total) error = nil, want failure")
		}
	})

	t.Run("no-limit", func(t *testing.T) {
		var total int64
		writer := &countingLimitWriter{total: &total}
		if n, err := writer.Write([]byte("abcdef")); err != nil {
			t.Fatalf("countingLimitWriter.Write(no limit) error = %v", err)
		} else if n != 6 {
			t.Fatalf("countingLimitWriter.Write(no limit) = %d, want 6", n)
		}
		if total != 6 {
			t.Fatalf("total = %d, want 6", total)
		}
	})
}

func mustTarGz(t *testing.T, entries []tarEntry) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)

	for _, entry := range entries {
		typeflag := entry.typeflag
		if typeflag == 0 {
			typeflag = tar.TypeReg
		}

		header := &tar.Header{
			Name:     entry.name,
			Mode:     0o644,
			Typeflag: typeflag,
			Linkname: entry.linkname,
		}
		switch typeflag {
		case tar.TypeDir:
			header.Mode = 0o755
		case tar.TypeReg:
			header.Size = int64(len(entry.content))
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("WriteHeader(%q) error = %v", entry.name, err)
		}
		if typeflag == tar.TypeReg {
			if _, err := io.WriteString(tarWriter, entry.content); err != nil {
				t.Fatalf("Write(%q) error = %v", entry.name, err)
			}
		}
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("tarWriter.Close() error = %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("gzipWriter.Close() error = %v", err)
	}
	return buffer.Bytes()
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
