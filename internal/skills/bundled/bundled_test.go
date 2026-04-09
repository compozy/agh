package bundled_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/skills/bundled"
)

var bundledSkillFixtures = []struct {
	path string
	name string
}{
	{
		path: "skills/agh-agent-setup/SKILL.md",
		name: "agh-agent-setup",
	},
	{
		path: "skills/agh-memory-guide/SKILL.md",
		name: "agh-memory-guide",
	},
	{
		path: "skills/agh-session-guide/SKILL.md",
		name: "agh-session-guide",
	},
}

func TestBundledFSContainsExpectedSkills(t *testing.T) {
	t.Parallel()

	fsys := bundled.FS()

	gotPaths, err := walkSkillPaths(fsys)
	if err != nil {
		t.Fatalf("walk bundled FS: %v", err)
	}

	wantPaths := make([]string, 0, len(bundledSkillFixtures))
	for _, fixture := range bundledSkillFixtures {
		wantPaths = append(wantPaths, fixture.path)

		content, err := fs.ReadFile(fsys, fixture.path)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", fixture.path, err)
		}
		if strings.TrimSpace(string(content)) == "" {
			t.Fatalf("ReadFile(%q) returned empty content", fixture.path)
		}
	}

	slices.Sort(wantPaths)
	if !slices.Equal(gotPaths, wantPaths) {
		t.Fatalf("bundled skill paths = %#v, want %#v", gotPaths, wantPaths)
	}
}

func TestBundledSkillsParseWithLoader(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fsys := bundled.FS()

	for _, fixture := range bundledSkillFixtures {
		fixture := fixture
		t.Run(fixture.name, func(t *testing.T) {
			t.Parallel()

			skillPath := materializeSkillFile(t, fsys, root, fixture.path)

			parsed, err := skills.ParseSkillFile(skillPath)
			if err != nil {
				t.Fatalf("ParseSkillFile(%q) error = %v", skillPath, err)
			}
			if parsed.Meta.Name != fixture.name {
				t.Fatalf("ParseSkillFile(%q) name = %q, want %q", skillPath, parsed.Meta.Name, fixture.name)
			}
			if strings.TrimSpace(parsed.Meta.Description) == "" {
				t.Fatalf("ParseSkillFile(%q) description is empty", skillPath)
			}
			if !parsed.Enabled {
				t.Fatalf("ParseSkillFile(%q) Enabled = false, want true", skillPath)
			}

			content, err := skills.ReadSkillContent(skillPath)
			if err != nil {
				t.Fatalf("ReadSkillContent(%q) error = %v", skillPath, err)
			}
			if strings.TrimSpace(content) == "" {
				t.Fatalf("ReadSkillContent(%q) returned empty content", skillPath)
			}
		})
	}
}

func walkSkillPaths(fsys fs.FS) ([]string, error) {
	paths := make([]string, 0, len(bundledSkillFixtures))
	err := fs.WalkDir(fsys, ".", func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Base(current) != "SKILL.md" {
			return nil
		}

		paths = append(paths, current)
		return nil
	})
	if err != nil {
		return nil, err
	}

	slices.Sort(paths)
	return paths, nil
}

func materializeSkillFile(t *testing.T, fsys fs.FS, root, bundledPath string) string {
	t.Helper()

	content, err := fs.ReadFile(fsys, bundledPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", bundledPath, err)
	}

	targetPath := filepath.Join(root, bundledPath)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(targetPath), err)
	}
	if err := os.WriteFile(targetPath, content, 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", targetPath, err)
	}

	return targetPath
}
