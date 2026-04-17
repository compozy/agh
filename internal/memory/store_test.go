package memory

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/goccy/go-yaml"
)

type testMemoryMeta struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Type        Type   `yaml:"type"`
	AgentName   string `yaml:"agent_name,omitempty"`
}

func TestStoreWriteReadRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		scope        Scope
		filename     string
		meta         testMemoryMeta
		body         string
		wantLocation func(*testStoreEnv) string
	}{
		{
			name:     "global scope",
			scope:    ScopeGlobal,
			filename: "user_preferences.md",
			meta: testMemoryMeta{
				Name:        "User Preferences",
				Description: "Preferred working style",
				Type:        MemoryTypeUser,
			},
			body: "Prefers explicit error handling.\n",
			wantLocation: func(env *testStoreEnv) string {
				return filepath.Join(env.store.globalDir, "user_preferences.md")
			},
		},
		{
			name:     "workspace scope",
			scope:    ScopeWorkspace,
			filename: "project_auth.md",
			meta: testMemoryMeta{
				Name:        "Auth Rewrite",
				Description: "JWT rollout plan",
				Type:        MemoryTypeProject,
				AgentName:   "claude",
			},
			body: "Workspace uses JWT-based auth.\n",
			wantLocation: func(env *testStoreEnv) string {
				return filepath.Join(env.store.workspaceDir, "project_auth.md")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := newTestStoreEnv(t)
			payload := mustMemoryContent(t, tt.meta, tt.body)

			if err := env.store.Write(tt.scope, tt.filename, payload); err != nil {
				t.Fatalf("Store.Write() error = %v", err)
			}

			got, err := env.store.Read(tt.scope, tt.filename)
			if err != nil {
				t.Fatalf("Store.Read() error = %v", err)
			}
			if !bytes.Equal(got, payload) {
				t.Fatalf("Store.Read() = %q, want %q", string(got), string(payload))
			}

			if _, err := os.Stat(tt.wantLocation(env)); err != nil {
				t.Fatalf("os.Stat(written path) error = %v", err)
			}
		})
	}
}

func TestStoreWriteRejectsInvalidFrontmatter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name: "missing name",
			content: `---
type: user
---
Body
`,
			wantErr: "memory name is required",
		},
		{
			name: "missing type",
			content: `---
name: Missing Type
---
Body
`,
			wantErr: "memory type is required",
		},
		{
			name: "unknown type",
			content: `---
name: Unknown Type
type: archive
---
Body
`,
			wantErr: `unsupported memory type "archive"`,
		},
		{
			name:    "missing frontmatter",
			content: "Body only\n",
			wantErr: "missing YAML frontmatter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := newTestStoreEnv(t)
			err := env.store.Write(ScopeGlobal, "invalid.md", []byte(tt.content))
			if err == nil {
				t.Fatal("Store.Write() error = nil, want non-nil")
			}
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("Store.Write() error = %v, want ErrValidation", err)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Store.Write() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestStoreWriteRejectsInvalidFilename(t *testing.T) {
	t.Parallel()
	payload := mustMemoryContent(t, testMemoryMeta{
		Name:        "Valid",
		Description: "Validation test",
		Type:        MemoryTypeUser,
	}, "Body\n")

	tests := []struct {
		name     string
		filename string
		wantErr  string
	}{
		{name: "path separator", filename: "nested/file.md", wantErr: "must not include path separators"},
		{name: "backslash separator", filename: `nested\file.md`, wantErr: "must not include path separators"},
		{name: "dot", filename: ".", wantErr: `filename "." is invalid`},
		{name: "dotdot", filename: "..", wantErr: `filename ".." is invalid`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := newTestStoreEnv(t)
			err := env.store.Write(ScopeGlobal, tt.filename, payload)
			if err == nil {
				t.Fatal("Store.Write() error = nil, want non-nil")
			}
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("Store.Write() error = %v, want ErrValidation", err)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Store.Write() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestStoreReadMissingFile(t *testing.T) {
	t.Parallel()

	env := newTestStoreEnv(t)

	_, err := env.store.Read(ScopeGlobal, "missing.md")
	if err == nil {
		t.Fatal("Store.Read() error = nil, want non-nil")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Store.Read() error = %v, want os.ErrNotExist", err)
	}
}

func TestStoreDeleteRemovesFileAndIndexEntry(t *testing.T) {
	t.Parallel()

	env := newTestStoreEnv(t)
	payload := mustMemoryContent(t, testMemoryMeta{
		Name:        "User Preferences",
		Description: "Preferred tools",
		Type:        MemoryTypeUser,
	}, "Prefers rg over grep.\n")

	if err := env.store.Write(ScopeGlobal, "user_preferences.md", payload); err != nil {
		t.Fatalf("Store.Write() error = %v", err)
	}

	indexContent := strings.Join([]string{
		"- [User Preferences](user_preferences.md) - Preferred tools",
		"- [Other](other.md) - Another note",
		"",
	}, "\n")
	if err := os.WriteFile(
		filepath.Join(env.store.globalDir, indexFilename),
		[]byte(indexContent),
		filePerm,
	); err != nil {
		t.Fatalf("write index file: %v", err)
	}

	if err := env.store.Delete(ScopeGlobal, "user_preferences.md"); err != nil {
		t.Fatalf("Store.Delete() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(env.store.globalDir, "user_preferences.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("os.Stat(deleted file) error = %v, want os.ErrNotExist", err)
	}

	indexBytes, err := os.ReadFile(filepath.Join(env.store.globalDir, indexFilename))
	if err != nil {
		t.Fatalf("os.ReadFile(index) error = %v", err)
	}
	if strings.Contains(string(indexBytes), "(user_preferences.md)") {
		t.Fatalf("index content still references deleted file: %q", string(indexBytes))
	}
	if !strings.Contains(string(indexBytes), "(other.md)") {
		t.Fatalf("index content removed unrelated entry: %q", string(indexBytes))
	}
}

func TestStoreDeletePreservesLinesThatOnlyMentionFilenameInDescription(t *testing.T) {
	t.Parallel()

	env := newTestStoreEnv(t)
	payload := mustMemoryContent(t, testMemoryMeta{
		Name:        "User Preferences",
		Description: "Preferred tools",
		Type:        MemoryTypeUser,
	}, "Prefers rg over grep.\n")

	if err := env.store.Write(ScopeGlobal, "user_preferences.md", payload); err != nil {
		t.Fatalf("Store.Write() error = %v", err)
	}

	indexContent := strings.Join([]string{
		"- [User Preferences](user_preferences.md) - Preferred tools",
		"- [Related Notes](other.md) - Mirrors notes from (user_preferences.md)",
		"",
	}, "\n")
	if err := os.WriteFile(
		filepath.Join(env.store.globalDir, indexFilename),
		[]byte(indexContent),
		filePerm,
	); err != nil {
		t.Fatalf("write index file: %v", err)
	}

	if err := env.store.Delete(ScopeGlobal, "user_preferences.md"); err != nil {
		t.Fatalf("Store.Delete() error = %v", err)
	}

	indexBytes, err := os.ReadFile(filepath.Join(env.store.globalDir, indexFilename))
	if err != nil {
		t.Fatalf("os.ReadFile(index) error = %v", err)
	}
	got := string(indexBytes)
	if strings.Contains(got, "- [User Preferences](user_preferences.md)") {
		t.Fatalf("index content still references deleted file: %q", got)
	}
	if !strings.Contains(got, "- [Related Notes](other.md) - Mirrors notes from (user_preferences.md)") {
		t.Fatalf("index content removed unrelated descriptive line: %q", got)
	}
}

func TestStoreDeleteRemovesIndexEntryForFilenameWithParentheses(t *testing.T) {
	t.Parallel()

	env := newTestStoreEnv(t)
	filename := "user(preferences).md"
	payload := mustMemoryContent(t, testMemoryMeta{
		Name:        "User Preferences",
		Description: "Preferred tools",
		Type:        MemoryTypeUser,
	}, "Prefers rg over grep.\n")

	if err := env.store.Write(ScopeGlobal, filename, payload); err != nil {
		t.Fatalf("Store.Write() error = %v", err)
	}

	indexContent := strings.Join([]string{
		"- [User Preferences](user(preferences).md) - Preferred tools",
		"- [Other](other.md) - Another note",
		"",
	}, "\n")
	if err := os.WriteFile(
		filepath.Join(env.store.globalDir, indexFilename),
		[]byte(indexContent),
		filePerm,
	); err != nil {
		t.Fatalf("write index file: %v", err)
	}

	if err := env.store.Delete(ScopeGlobal, filename); err != nil {
		t.Fatalf("Store.Delete() error = %v", err)
	}

	indexBytes, err := os.ReadFile(filepath.Join(env.store.globalDir, indexFilename))
	if err != nil {
		t.Fatalf("os.ReadFile(index) error = %v", err)
	}
	got := string(indexBytes)
	if strings.Contains(got, "- [User Preferences](user(preferences).md)") {
		t.Fatalf("index content still references deleted file: %q", got)
	}
	if !strings.Contains(got, "- [Other](other.md) - Another note") {
		t.Fatalf("index content removed unrelated line: %q", got)
	}
}

func TestStoreDeleteMissingFile(t *testing.T) {
	t.Parallel()

	env := newTestStoreEnv(t)

	err := env.store.Delete(ScopeWorkspace, "missing.md")
	if err == nil {
		t.Fatal("Store.Delete() error = nil, want non-nil")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Store.Delete() error = %v, want os.ErrNotExist", err)
	}
}

func TestStoreScanReturnsNewestFirst(t *testing.T) {
	t.Parallel()

	env := newTestStoreEnv(t)
	base := time.Now().Add(-3 * time.Hour)

	files := []struct {
		filename string
		name     string
		modTime  time.Time
		agent    string
	}{
		{filename: "older.md", name: "Older", modTime: base, agent: "claude"},
		{filename: "newest.md", name: "Newest", modTime: base.Add(2 * time.Hour), agent: "codex"},
		{filename: "middle.md", name: "Middle", modTime: base.Add(1 * time.Hour)},
	}

	for _, file := range files {
		payload := mustMemoryContent(t, testMemoryMeta{
			Name:        file.name,
			Description: file.name + " description",
			Type:        MemoryTypeProject,
			AgentName:   file.agent,
		}, file.name+" body\n")
		if err := env.store.Write(ScopeWorkspace, file.filename, payload); err != nil {
			t.Fatalf("Store.Write(%q) error = %v", file.filename, err)
		}

		path, err := env.store.pathFor(ScopeWorkspace, file.filename)
		if err != nil {
			t.Fatalf("pathFor(%q) error = %v", file.filename, err)
		}
		if err := os.Chtimes(path, file.modTime, file.modTime); err != nil {
			t.Fatalf("os.Chtimes(%q) error = %v", path, err)
		}
	}

	headers, err := env.store.Scan(ScopeWorkspace)
	if err != nil {
		t.Fatalf("Store.Scan() error = %v", err)
	}

	if got, want := len(headers), 3; got != want {
		t.Fatalf("len(headers) = %d, want %d", got, want)
	}

	wantOrder := []string{"newest.md", "middle.md", "older.md"}
	for idx, want := range wantOrder {
		if headers[idx].Filename != want {
			t.Fatalf("headers[%d].Filename = %q, want %q", idx, headers[idx].Filename, want)
		}
		if headers[idx].FilePath == "" {
			t.Fatalf("headers[%d].FilePath = empty, want populated path", idx)
		}
	}

	if headers[0].AgentName != "codex" {
		t.Fatalf("headers[0].AgentName = %q, want %q", headers[0].AgentName, "codex")
	}
	if headers[1].AgentName != "" {
		t.Fatalf("headers[1].AgentName = %q, want empty string", headers[1].AgentName)
	}
}

func TestStoreScanCapsAtTwoHundredFiles(t *testing.T) {
	t.Parallel()

	env := newTestStoreEnv(t)
	base := time.Now().Add(-205 * time.Minute)

	for idx := range 205 {
		filename := fmt.Sprintf("%03d.md", idx)
		payload := mustMemoryContent(t, testMemoryMeta{
			Name:        fmt.Sprintf("Memory %03d", idx),
			Description: "Cap test",
			Type:        MemoryTypeReference,
		}, "Reference entry\n")
		if err := env.store.Write(ScopeWorkspace, filename, payload); err != nil {
			t.Fatalf("Store.Write(%q) error = %v", filename, err)
		}

		path, err := env.store.pathFor(ScopeWorkspace, filename)
		if err != nil {
			t.Fatalf("pathFor(%q) error = %v", filename, err)
		}
		modTime := base.Add(time.Duration(idx) * time.Minute)
		if err := os.Chtimes(path, modTime, modTime); err != nil {
			t.Fatalf("os.Chtimes(%q) error = %v", path, err)
		}
	}

	headers, err := env.store.Scan(ScopeWorkspace)
	if err != nil {
		t.Fatalf("Store.Scan() error = %v", err)
	}

	if got, want := len(headers), 200; got != want {
		t.Fatalf("len(headers) = %d, want %d", got, want)
	}
	if headers[0].Filename != "204.md" {
		t.Fatalf("headers[0].Filename = %q, want %q", headers[0].Filename, "204.md")
	}
	if headers[len(headers)-1].Filename != "005.md" {
		t.Fatalf("headers[last].Filename = %q, want %q", headers[len(headers)-1].Filename, "005.md")
	}
}

func TestStoreScanCapsAtTwoHundredFilesAfterSkippingMalformedNewestEntries(t *testing.T) {
	t.Parallel()

	env := newTestStoreEnv(t)
	base := time.Now().Add(-205 * time.Minute)

	for idx := range 205 {
		filename := fmt.Sprintf("%03d.md", idx)
		payload := mustMemoryContent(t, testMemoryMeta{
			Name:        fmt.Sprintf("Memory %03d", idx),
			Description: "Cap test",
			Type:        MemoryTypeReference,
		}, "Reference entry\n")
		if err := env.store.Write(ScopeWorkspace, filename, payload); err != nil {
			t.Fatalf("Store.Write(%q) error = %v", filename, err)
		}

		path, err := env.store.pathFor(ScopeWorkspace, filename)
		if err != nil {
			t.Fatalf("pathFor(%q) error = %v", filename, err)
		}
		modTime := base.Add(time.Duration(idx) * time.Minute)
		if err := os.Chtimes(path, modTime, modTime); err != nil {
			t.Fatalf("os.Chtimes(%q) error = %v", path, err)
		}
	}

	for idx := range 3 {
		filename := fmt.Sprintf("broken-%d.md", idx)
		path, err := env.store.pathFor(ScopeWorkspace, filename)
		if err != nil {
			t.Fatalf("pathFor(%q) error = %v", filename, err)
		}
		if err := os.WriteFile(path, []byte("not frontmatter\n"), filePerm); err != nil {
			t.Fatalf("write malformed file %q: %v", filename, err)
		}
		modTime := base.Add(time.Duration(205+idx) * time.Minute)
		if err := os.Chtimes(path, modTime, modTime); err != nil {
			t.Fatalf("os.Chtimes(%q) error = %v", path, err)
		}
	}

	headers, err := env.store.Scan(ScopeWorkspace)
	if err != nil {
		t.Fatalf("Store.Scan() error = %v", err)
	}

	if got, want := len(headers), 200; got != want {
		t.Fatalf("len(headers) = %d, want %d", got, want)
	}
	if headers[0].Filename != "204.md" {
		t.Fatalf("headers[0].Filename = %q, want %q", headers[0].Filename, "204.md")
	}
	if headers[len(headers)-1].Filename != "005.md" {
		t.Fatalf("headers[last].Filename = %q, want %q", headers[len(headers)-1].Filename, "005.md")
	}
}

func TestStoreScanSkipsMalformedFilesAndLogsWarning(t *testing.T) {
	t.Parallel()

	env := newTestStoreEnv(t)
	var logs bytes.Buffer
	env.store.logger = slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelWarn}))

	if err := env.store.Write(ScopeGlobal, "valid.md", mustMemoryContent(t, testMemoryMeta{
		Name:        "Valid",
		Description: "Valid memory",
		Type:        MemoryTypeFeedback,
	}, "Valid body\n")); err != nil {
		t.Fatalf("Store.Write(valid) error = %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(env.store.globalDir, "broken.md"),
		[]byte("not frontmatter\n"),
		filePerm,
	); err != nil {
		t.Fatalf("write malformed file: %v", err)
	}

	headers, err := env.store.Scan(ScopeGlobal)
	if err != nil {
		t.Fatalf("Store.Scan() error = %v", err)
	}
	if got, want := len(headers), 1; got != want {
		t.Fatalf("len(headers) = %d, want %d", got, want)
	}
	if !strings.Contains(logs.String(), "skip malformed memory file") {
		t.Fatalf("scan logs = %q, want malformed warning", logs.String())
	}
}

func TestStoreLoadIndex(t *testing.T) {
	t.Parallel()

	t.Run("returns full content", func(t *testing.T) {
		t.Parallel()

		env := newTestStoreEnv(t)
		want := strings.Join([]string{
			"- [Auth Rewrite](project_auth.md) - JWT rollout",
			"- [Reference](reference_api.md) - API docs",
			"",
		}, "\n")
		if err := os.WriteFile(
			filepath.Join(env.store.workspaceDir, indexFilename),
			[]byte(want),
			filePerm,
		); err != nil {
			t.Fatalf("write index: %v", err)
		}

		got, truncated, err := env.store.LoadIndex(ScopeWorkspace)
		if err != nil {
			t.Fatalf("Store.LoadIndex() error = %v", err)
		}
		if truncated {
			t.Fatal("Store.LoadIndex() truncated = true, want false")
		}
		if got != want {
			t.Fatalf("Store.LoadIndex() = %q, want %q", got, want)
		}
	})

	t.Run("truncates by line count", func(t *testing.T) {
		t.Parallel()

		env := newTestStoreEnv(t)

		lines := make([]string, 0, 205)
		for idx := range 205 {
			lines = append(lines, fmt.Sprintf("- [Line %03d](line-%03d.md) - Description", idx, idx))
		}
		index := strings.Join(lines, "\n") + "\n"
		if err := os.WriteFile(filepath.Join(env.store.globalDir, indexFilename), []byte(index), filePerm); err != nil {
			t.Fatalf("write index: %v", err)
		}

		got, truncated, err := env.store.LoadIndex(ScopeGlobal)
		if err != nil {
			t.Fatalf("Store.LoadIndex() error = %v", err)
		}
		if !truncated {
			t.Fatal("Store.LoadIndex() truncated = false, want true")
		}

		gotLines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
		if gotCount, wantCount := len(gotLines), 200; gotCount != wantCount {
			t.Fatalf("len(gotLines) = %d, want %d", gotCount, wantCount)
		}
		if gotLines[199] != "- [Line 199](line-199.md) - Description" {
			t.Fatalf("last retained line = %q, want line 199", gotLines[199])
		}
	})

	t.Run("truncates by byte count and respects utf8", func(t *testing.T) {
		t.Parallel()

		env := newTestStoreEnv(t)

		line := strings.Repeat("é", defaultIndexBytes) + "\n"
		if err := os.WriteFile(filepath.Join(env.store.globalDir, indexFilename), []byte(line), filePerm); err != nil {
			t.Fatalf("write index: %v", err)
		}

		got, truncated, err := env.store.LoadIndex(ScopeGlobal)
		if err != nil {
			t.Fatalf("Store.LoadIndex() error = %v", err)
		}
		if !truncated {
			t.Fatal("Store.LoadIndex() truncated = false, want true")
		}
		if len(got) > env.store.maxIndexBytes {
			t.Fatalf("len(got) = %d, want <= %d", len(got), env.store.maxIndexBytes)
		}
		if !utf8.ValidString(got) {
			t.Fatalf("Store.LoadIndex() returned invalid UTF-8: %q", got)
		}
	})

	t.Run("missing index returns empty content", func(t *testing.T) {
		t.Parallel()

		env := newTestStoreEnv(t)

		got, truncated, err := env.store.LoadIndex(ScopeWorkspace)
		if err != nil {
			t.Fatalf("Store.LoadIndex() error = %v", err)
		}
		if truncated {
			t.Fatal("Store.LoadIndex() truncated = true, want false")
		}
		if got != "" {
			t.Fatalf("Store.LoadIndex() = %q, want empty", got)
		}
	})
}

func TestStoreEnsureDirs(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	store := NewStore(filepath.Join(baseDir, "home", "memory")).ForWorkspace(filepath.Join(baseDir, "workspace"))

	if err := store.EnsureDirs(); err != nil {
		t.Fatalf("Store.EnsureDirs() error = %v", err)
	}

	for _, dir := range []string{store.globalDir, store.workspaceDir} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("os.Stat(%q) error = %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("%q is not a directory", dir)
		}
	}

	baseStore := NewStore(filepath.Join(baseDir, "only-global"))
	if err := baseStore.EnsureDirs(); err != nil {
		t.Fatalf("Store.EnsureDirs() with global-only store error = %v", err)
	}
}

func TestStoreRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		run     func(*testStoreEnv) error
		wantErr string
		verify  func(*testing.T, *testStoreEnv)
	}{
		{
			name: "invalid scope on scan",
			run: func(env *testStoreEnv) error {
				_, err := env.store.Scan(Scope("sideways"))
				return err
			},
			wantErr: `unsupported scope "sideways"`,
		},
		{
			name: "invalid scope on load index",
			run: func(env *testStoreEnv) error {
				_, _, err := env.store.LoadIndex(Scope("sideways"))
				return err
			},
			wantErr: `unsupported scope "sideways"`,
		},
		{
			name: "missing workspace directory",
			run: func(env *testStoreEnv) error {
				_, err := NewStore(env.store.globalDir).Scan(ScopeWorkspace)
				return err
			},
			wantErr: "workspace directory is required",
		},
		{
			name: "path traversal filename on read",
			run: func(env *testStoreEnv) error {
				_, err := env.store.Read(ScopeGlobal, "nested/file.md")
				return err
			},
			wantErr: "must not include path separators",
		},
		{
			name: "empty filename on delete",
			run: func(env *testStoreEnv) error {
				return env.store.Delete(ScopeGlobal, " ")
			},
			wantErr: "filename is required",
		},
		{
			name: "normalized memory type",
			run: func(env *testStoreEnv) error {
				return env.store.Write(ScopeGlobal, "normalized.md", []byte(`---
name: Normalized Type
type: "  PROJECT "
---
Body
`))
			},
			wantErr: "",
			verify: func(t *testing.T, env *testStoreEnv) {
				t.Helper()

				headers, err := env.store.Scan(ScopeGlobal)
				if err != nil {
					t.Fatalf("Store.Scan() error = %v", err)
				}
				if got, want := len(headers), 1; got != want {
					t.Fatalf("len(headers) = %d, want %d", got, want)
				}
				if headers[0].Type != MemoryTypeProject {
					t.Fatalf("headers[0].Type = %q, want %q", headers[0].Type, MemoryTypeProject)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := newTestStoreEnv(t)
			err := tt.run(env)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("operation error = %v, want nil", err)
				}
				if tt.verify != nil {
					tt.verify(t, env)
				}
				return
			}
			if err == nil {
				t.Fatal("operation error = nil, want non-nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("operation error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestStoreScanMissingDirectoryReturnsEmpty(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	store := NewStore(filepath.Join(baseDir, "global")).ForWorkspace(filepath.Join(baseDir, "workspace"))

	headers, err := store.Scan(ScopeWorkspace)
	if err != nil {
		t.Fatalf("Store.Scan() error = %v", err)
	}
	if len(headers) != 0 {
		t.Fatalf("len(headers) = %d, want 0", len(headers))
	}
}

func TestStalenessHelpers(t *testing.T) {
	t.Parallel()

	location := time.FixedZone("UTC-3", -3*60*60)
	now := time.Date(2026, 4, 4, 10, 0, 0, 0, location)

	today := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, location)
	yesterday := today.Add(-24 * time.Hour)
	threeDaysAgo := today.Add(-72 * time.Hour)

	testCases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "Should return zero days for today",
			run: func(t *testing.T) {
				t.Parallel()
				if got := ageDays(today, now); got != 0 {
					t.Fatalf("ageDays(today) = %d, want 0", got)
				}
			},
		},
		{
			name: "Should return one day for yesterday",
			run: func(t *testing.T) {
				t.Parallel()
				if got := ageDays(yesterday, now); got != 1 {
					t.Fatalf("ageDays(yesterday) = %d, want 1", got)
				}
			},
		},
		{
			name: "Should render today age text",
			run: func(t *testing.T) {
				t.Parallel()
				if got := ageText(today, now); got != "today" {
					t.Fatalf("ageText(today) = %q, want %q", got, "today")
				}
			},
		},
		{
			name: "Should render yesterday age text",
			run: func(t *testing.T) {
				t.Parallel()
				if got := ageText(yesterday, now); got != "yesterday" {
					t.Fatalf("ageText(yesterday) = %q, want %q", got, "yesterday")
				}
			},
		},
		{
			name: "Should render multi-day age text",
			run: func(t *testing.T) {
				t.Parallel()
				if got := ageText(threeDaysAgo, now); got != "3 days ago" {
					t.Fatalf("ageText(threeDaysAgo) = %q, want %q", got, "3 days ago")
				}
			},
		},
		{
			name: "Should omit freshness warning for today",
			run: func(t *testing.T) {
				t.Parallel()
				if got := freshnessWarning(today, now); got != "" {
					t.Fatalf("freshnessWarning(today) = %q, want empty", got)
				}
			},
		},
		{
			name: "Should omit freshness warning for yesterday",
			run: func(t *testing.T) {
				t.Parallel()
				if got := freshnessWarning(yesterday, now); got != "" {
					t.Fatalf("freshnessWarning(yesterday) = %q, want empty", got)
				}
			},
		},
		{
			name: "Should warn for stale memories",
			run: func(t *testing.T) {
				t.Parallel()
				if got := freshnessWarning(threeDaysAgo, now); !strings.Contains(got, "3 days old") {
					t.Fatalf("freshnessWarning(threeDaysAgo) = %q, want age caveat", got)
				}
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}

func TestStoreExists(t *testing.T) {
	t.Parallel()

	env := newTestStoreEnv(t)
	content := mustMemoryContent(t, testMemoryMeta{
		Name:        "User Memory",
		Description: "desc",
		Type:        MemoryTypeUser,
	}, "hello")
	if err := env.store.Write(ScopeWorkspace, "exists.md", content); err != nil {
		t.Fatalf("Store.Write() error = %v", err)
	}

	exists, err := env.store.Exists(ScopeWorkspace, "exists.md")
	if err != nil {
		t.Fatalf("Store.Exists(exists.md) error = %v", err)
	}
	if !exists {
		t.Fatal("Store.Exists(exists.md) = false, want true")
	}

	missing, err := env.store.Exists(ScopeWorkspace, "missing.md")
	if err != nil {
		t.Fatalf("Store.Exists(missing.md) error = %v", err)
	}
	if missing {
		t.Fatal("Store.Exists(missing.md) = true, want false")
	}
}

type testStoreEnv struct {
	store *Store
}

func newTestStoreEnv(t *testing.T) *testStoreEnv {
	t.Helper()

	baseDir := t.TempDir()
	workspaceRoot := filepath.Join(baseDir, "workspace")
	store := NewStore(filepath.Join(baseDir, "global")).ForWorkspace(workspaceRoot)
	if err := store.EnsureDirs(); err != nil {
		t.Fatalf("Store.EnsureDirs() error = %v", err)
	}

	return &testStoreEnv{store: store}
}

func mustMemoryContent(t *testing.T, meta testMemoryMeta, body string) []byte {
	t.Helper()

	metaBytes, err := yaml.Marshal(meta)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}

	return []byte("---\n" + strings.TrimRight(string(metaBytes), "\n") + "\n---\n" + body)
}
