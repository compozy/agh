package skills

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/pedronauck/agh/internal/filesnap"
)

func TestParseFrontmatterValidCases(t *testing.T) {
	t.Parallel()

	longBody := strings.Repeat("abc123", 9_000)
	tests := []struct {
		name           string
		content        string
		wantMeta       SkillMeta
		wantBody       string
		wantBodyLength int
	}{
		{
			name: "all fields",
			content: strings.Join([]string{
				"---",
				"name: code-review",
				"description: Review code carefully",
				"version: 1.2.3",
				"metadata:",
				"  agh:",
				"    category: quality",
				"---",
				"# Heading",
				"",
				"Review body",
			}, "\n"),
			wantMeta: SkillMeta{
				Name:        "code-review",
				Description: "Review code carefully",
				Version:     "1.2.3",
				Metadata: map[string]any{
					"agh": map[string]any{
						"category": "quality",
					},
				},
			},
			wantBody: "# Heading\n\nReview body",
		},
		{
			name: "required fields only",
			content: strings.Join([]string{
				"---",
				"name: debug",
				"description: Diagnose runtime failures",
				"---",
				"Debug this system",
			}, "\n"),
			wantMeta: SkillMeta{
				Name:        "debug",
				Description: "Diagnose runtime failures",
			},
			wantBody: "Debug this system",
		},
		{
			name: "empty body",
			content: strings.Join([]string{
				"---",
				"name: empty",
				"description: Body optional",
				"---",
			}, "\n"),
			wantMeta: SkillMeta{
				Name:        "empty",
				Description: "Body optional",
			},
			wantBody: "",
		},
		{
			name: "long body",
			content: strings.Join([]string{
				"---",
				"name: long-body",
				"description: Long content",
				"---",
				longBody,
			}, "\n"),
			wantMeta: SkillMeta{
				Name:        "long-body",
				Description: "Long content",
			},
			wantBodyLength: len(longBody),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotMeta, gotBody, err := parseFrontmatter(tt.content)
			if err != nil {
				t.Fatalf("parseFrontmatter() error = %v", err)
			}

			if !reflect.DeepEqual(gotMeta, tt.wantMeta) {
				t.Fatalf("parseFrontmatter() meta mismatch\nwant: %#v\ngot:  %#v", tt.wantMeta, gotMeta)
			}

			switch {
			case tt.wantBodyLength > 0 && len(gotBody) != tt.wantBodyLength:
				t.Fatalf("parseFrontmatter() body length = %d, want %d", len(gotBody), tt.wantBodyLength)
			case tt.wantBodyLength == 0 && gotBody != tt.wantBody:
				t.Fatalf("parseFrontmatter() body = %q, want %q", gotBody, tt.wantBody)
			}
		})
	}
}

func TestParseFrontmatterErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		wantErr error
	}{
		{
			name:    "delimiter only",
			content: "---",
			wantErr: errFrontmatterUnterminated,
		},
		{
			name: "missing opening delimiter",
			content: strings.Join([]string{
				"name: invalid",
				"description: missing delimiters",
			}, "\n"),
			wantErr: errFrontmatterMissing,
		},
		{
			name: "unterminated frontmatter",
			content: strings.Join([]string{
				"---",
				"name: invalid",
				"description: missing close",
			}, "\n"),
			wantErr: errFrontmatterUnterminated,
		},
		{
			name: "malformed yaml",
			content: strings.Join([]string{
				"---",
				"name: [broken",
				"description: invalid yaml",
				"---",
			}, "\n"),
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := parseFrontmatter(tt.content)
			if err == nil {
				t.Fatal("parseFrontmatter() error = nil, want error")
			}

			if tt.wantErr != nil && !strings.Contains(err.Error(), tt.wantErr.Error()) {
				t.Fatalf("parseFrontmatter() error = %v, want containing %q", err, tt.wantErr)
			}
			if tt.wantErr == nil && !strings.Contains(err.Error(), "decode YAML frontmatter") {
				t.Fatalf("parseFrontmatter() error = %v, want YAML decode error", err)
			}
		})
	}
}

func TestParseFrontmatterWarnsOnUnknownFields(t *testing.T) {
	original := slog.Default()
	var logs bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() {
		slog.SetDefault(original)
	})

	content := strings.Join([]string{
		"---",
		"name: warning-test",
		"description: unknown fields are allowed",
		"extra: true",
		"---",
		"body",
	}, "\n")

	meta, body, err := parseFrontmatter(content)
	if err != nil {
		t.Fatalf("parseFrontmatter() error = %v", err)
	}
	if meta.Name != "warning-test" {
		t.Fatalf("parseFrontmatter() meta.Name = %q, want %q", meta.Name, "warning-test")
	}
	if body != "body" {
		t.Fatalf("parseFrontmatter() body = %q, want %q", body, "body")
	}
	if !strings.Contains(logs.String(), "unknown frontmatter field") || !strings.Contains(logs.String(), "extra") {
		t.Fatalf("expected unknown field warning in logs, got %q", logs.String())
	}
}

func TestParseSkillFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := writeSkillFile(t, root, filepath.Join("quality", skillFileName), strings.Join([]string{
		"---",
		"name: quality",
		"description: Validate output",
		"version: 2.0.0",
		"---",
		"Check every requirement.",
	}, "\n"))

	skill, err := ParseSkillFile(path)
	if err != nil {
		t.Fatalf("ParseSkillFile() error = %v", err)
	}

	if skill.Meta.Name != "quality" {
		t.Fatalf("ParseSkillFile() meta.Name = %q, want %q", skill.Meta.Name, "quality")
	}
	if skill.Content != "Check every requirement." {
		t.Fatalf("ParseSkillFile() content = %q, want %q", skill.Content, "Check every requirement.")
	}
	if skill.FilePath != path {
		t.Fatalf("ParseSkillFile() FilePath = %q, want %q", skill.FilePath, path)
	}
	if skill.Dir != filepath.Dir(path) {
		t.Fatalf("ParseSkillFile() Dir = %q, want %q", skill.Dir, filepath.Dir(path))
	}
	if !skill.Enabled {
		t.Fatal("ParseSkillFile() Enabled = false, want true")
	}
}

func TestParseSkillFileMissingName(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := writeSkillFile(t, root, filepath.Join("missing-name", skillFileName), strings.Join([]string{
		"---",
		"description: No name present",
		"---",
		"body",
	}, "\n"))

	_, err := ParseSkillFile(path)
	if err == nil {
		t.Fatal("ParseSkillFile() error = nil, want error")
	}
	if !strings.Contains(err.Error(), errSkillNameRequired.Error()) {
		t.Fatalf("ParseSkillFile() error = %v, want containing %q", err, errSkillNameRequired)
	}
}

func TestParseSkillFileWarnsOnMissingDescription(t *testing.T) {
	original := slog.Default()
	var logs bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() {
		slog.SetDefault(original)
	})

	root := t.TempDir()
	path := writeSkillFile(t, root, filepath.Join("no-description", skillFileName), strings.Join([]string{
		"---",
		"name: no-description",
		"---",
		"body",
	}, "\n"))

	skill, err := ParseSkillFile(path)
	if err != nil {
		t.Fatalf("ParseSkillFile() error = %v", err)
	}
	if skill.Meta.Name != "no-description" {
		t.Fatalf("ParseSkillFile() meta.Name = %q, want %q", skill.Meta.Name, "no-description")
	}
	if !strings.Contains(logs.String(), "parsed skill without description") {
		t.Fatalf("expected missing description warning in logs, got %q", logs.String())
	}
}

func TestScanDirectoryHonorsDepthAndSkips(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	expected := []string{
		writeSkillFile(t, root, filepath.Join("depth1", skillFileName), defaultSkillContent("depth1")),
		writeSkillFile(t, root, filepath.Join("a", "depth2", skillFileName), defaultSkillContent("depth2")),
		writeSkillFile(t, root, filepath.Join("a", "b", "depth3", skillFileName), defaultSkillContent("depth3")),
		writeSkillFile(t, root, filepath.Join("a", "b", "c", "depth4", skillFileName), defaultSkillContent("depth4")),
		writeSkillFile(t, root, filepath.Join(".agents", "shared", skillFileName), defaultSkillContent("agents")),
		writeSkillFile(t, root, filepath.Join(".agh", "workspace", skillFileName), defaultSkillContent("workspace")),
	}
	writeSkillFile(t, root, filepath.Join("a", "b", "c", "d", "too-deep", skillFileName), defaultSkillContent("depth5"))
	writeSkillFile(t, root, filepath.Join(".git", "ignored", skillFileName), defaultSkillContent("git"))
	writeSkillFile(t, root, filepath.Join("node_modules", "pkg", skillFileName), defaultSkillContent("node"))
	writeSkillFile(t, root, filepath.Join(".hidden", "ignored", skillFileName), defaultSkillContent("hidden"))

	got, err := scanDirectory(root)
	if err != nil {
		t.Fatalf("scanDirectory() error = %v", err)
	}

	slices.Sort(expected)
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("scanDirectory() mismatch\nwant: %#v\ngot:  %#v", expected, got)
	}
}

func TestScanDirectoryCandidateLimit(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	for i := range maxScanCandidates + 5 {
		writeSkillFile(t, root, filepath.Join(fmt.Sprintf("skill-%03d", i), skillFileName), defaultSkillContent(fmt.Sprintf("skill-%03d", i)))
	}

	got, err := scanDirectory(root)
	if err != nil {
		t.Fatalf("scanDirectory() error = %v", err)
	}
	if len(got) != maxScanCandidates {
		t.Fatalf("scanDirectory() len = %d, want %d", len(got), maxScanCandidates)
	}
}

func TestScanDirectoryMissingRoot(t *testing.T) {
	t.Parallel()

	got, err := scanDirectory(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("scanDirectory() error = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Fatalf("scanDirectory() len = %d, want 0", len(got))
	}
}

func TestScanDirectoryRejectsInvalidRoots(t *testing.T) {
	t.Parallel()

	if _, err := scanDirectory("   "); err == nil {
		t.Fatal("scanDirectory() error = nil, want error for blank root")
	}

	fileRoot := filepath.Join(t.TempDir(), "SKILL.md")
	if err := os.WriteFile(fileRoot, []byte(defaultSkillContent("not-a-dir")), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", fileRoot, err)
	}

	if _, err := scanDirectory(fileRoot); err == nil {
		t.Fatal("scanDirectory() error = nil, want error for non-directory root")
	}
}

func TestSnapshotFile(t *testing.T) {
	t.Parallel()

	path := writeSkillFile(t, t.TempDir(), filepath.Join("skill", skillFileName), defaultSkillContent("snapshot"))

	snapshot, err := filesnap.FromPath(path)
	if err != nil {
		t.Fatalf("filesnap.FromPath() error = %v", err)
	}
	if snapshot.Size <= 0 {
		t.Fatalf("filesnap.FromPath() size = %d, want > 0", snapshot.Size)
	}
	if snapshot.ModTime.IsZero() {
		t.Fatal("filesnap.FromPath() mod time = zero, want populated")
	}

	if _, err := filesnap.FromPath(filepath.Join(t.TempDir(), "missing", skillFileName)); err == nil {
		t.Fatal("filesnap.FromPath() error = nil, want error for missing path")
	}
}

func writeSkillFile(t *testing.T, root, relPath, content string) string {
	t.Helper()

	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}

	return path
}

func defaultSkillContent(name string) string {
	return strings.Join([]string{
		"---",
		"name: " + name,
		"description: test skill",
		"---",
		"body",
	}, "\n")
}
