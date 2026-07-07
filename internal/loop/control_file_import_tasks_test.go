package loop

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/compozy/agh/internal/loop/dsl"
)

func TestFileImportMDTasksShouldLoadCompozyTaskManifest(t *testing.T) {
	t.Parallel()

	t.Run("Should import pending tasks in topological numeric order with hydrated bodies", func(t *testing.T) {
		t.Parallel()

		tasksDir := t.TempDir()
		writeMDTasksManifest(t, tasksDir, []string{
			"    - from: task_02",
			"      to: task_03",
			"    - from: task_01",
			"      to: task_03",
		})
		writeMDTaskFile(t, tasksDir, "task_01.md", "pending", "First task", "# First\n\nDo one.")
		writeMDTaskFile(t, tasksDir, "task_02.md", "completed", "Second task", "# Second\n\nAlready done.")
		writeMDTaskFile(t, tasksDir, "task_03.md", "pending", "Third task", "# Third\n\nDo three.")

		ref, empty, err := fileImportOutputRef(
			dsl.FileParseMDTasks,
			filepath.Join(tasksDir, "task_*.md"),
		)
		if err != nil {
			t.Fatalf("fileImportOutputRef(md_tasks) error = %v", err)
		}
		if empty {
			t.Fatal("fileImportOutputRef(md_tasks) empty = true, want false")
		}

		var payload struct {
			Tasks []markdownTaskPayload `json:"tasks"`
		}
		if err := json.Unmarshal([]byte(ref), &payload); err != nil {
			t.Fatalf("Unmarshal(file import ref) error = %v", err)
		}
		if len(payload.Tasks) != 2 {
			t.Fatalf("len(tasks) = %d, want 2 pending tasks", len(payload.Tasks))
		}
		if got, want := payload.Tasks[0].ID, "task_01"; got != want {
			t.Fatalf("tasks[0].id = %q, want %q", got, want)
		}
		if payload.Tasks[0].Blocks == nil || len(payload.Tasks[0].Blocks) != 0 {
			t.Fatalf("tasks[0].blocks = %#v, want empty JSON array", payload.Tasks[0].Blocks)
		}
		third := payload.Tasks[1]
		if third.ID != "task_03" || third.Number != 3 || third.Title != "Third task" {
			t.Fatalf("third task = %#v, want task_03 metadata", third)
		}
		if !strings.Contains(third.Body, "Do three.") {
			t.Fatalf("third body = %q, want hydrated markdown body", third.Body)
		}
		if got, want := third.BodyRef, OutputRefForPayload([]byte(third.Body)); got != want {
			t.Fatalf("third body_ref = %q, want %q", got, want)
		}
		if got, want := strings.Join(third.Blocks, ","), "task_01,task_02"; got != want {
			t.Fatalf("third blocks = %q, want %q", got, want)
		}
	})

	t.Run("Should return no work for an empty pending set", func(t *testing.T) {
		t.Parallel()

		tasksDir := t.TempDir()
		writeMDTasksManifest(t, tasksDir, nil)
		writeMDTaskFile(t, tasksDir, "task_01.md", "done", "First task", "# First\n")
		writeMDTaskFile(t, tasksDir, "task_02.md", "finished", "Second task", "# Second\n")
		writeMDTaskFile(t, tasksDir, "task_03.md", "completed", "Third task", "# Third\n")

		ref, empty, err := fileImportOutputRef(
			dsl.FileParseMDTasks,
			filepath.Join(tasksDir, "task_*.md"),
		)
		if err != nil {
			t.Fatalf("fileImportOutputRef(md_tasks) error = %v", err)
		}
		if !empty {
			t.Fatal("fileImportOutputRef(md_tasks) empty = false, want true")
		}
		if got, want := ref, `{"tasks":[]}`; got != want {
			t.Fatalf("fileImportOutputRef(md_tasks) = %q, want %q", got, want)
		}
	})

	t.Run("Should reject dependencies drift in task frontmatter", func(t *testing.T) {
		t.Parallel()

		tasksDir := t.TempDir()
		writeMDTasksManifest(t, tasksDir, nil)
		writeMDTaskFileWithFrontmatter(t, tasksDir, "task_01.md", []string{
			"status: pending",
			"title: First task",
			"dependencies: [task_02]",
		}, "# First\n")
		writeMDTaskFile(t, tasksDir, "task_02.md", "pending", "Second task", "# Second\n")
		writeMDTaskFile(t, tasksDir, "task_03.md", "pending", "Third task", "# Third\n")

		_, _, err := fileImportOutputRef(dsl.FileParseMDTasks, filepath.Join(tasksDir, "task_*.md"))
		if !errors.Is(err, ErrValidation) || !strings.Contains(err.Error(), "dependencies must live in _tasks.md") {
			t.Fatalf("fileImportOutputRef(md_tasks) error = %v, want dependencies validation", err)
		}
	})

	t.Run("Should reject invalid graph edges", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name    string
			edges   []string
			wantErr string
		}{
			{
				name: "self-edge",
				edges: []string{
					"    - from: task_03",
					"      to: task_03",
				},
				wantErr: "self-edge",
			},
			{
				name: "unknown source",
				edges: []string{
					"    - from: task_99",
					"      to: task_03",
				},
				wantErr: `edge source "task_99" is not a graph node`,
			},
			{
				name: "unknown target",
				edges: []string{
					"    - from: task_01",
					"      to: task_99",
				},
				wantErr: `edge target "task_99" is not a graph node`,
			},
			{
				name: "duplicate edge",
				edges: []string{
					"    - from: task_01",
					"      to: task_03",
					"    - from: task_01",
					"      to: task_03",
				},
				wantErr: `duplicate md_tasks edge "task_01" -> "task_03"`,
			},
		}

		for _, tc := range cases {
			t.Run("Should reject "+tc.name, func(t *testing.T) {
				t.Parallel()

				tasksDir := t.TempDir()
				writeMDTasksManifest(t, tasksDir, tc.edges)
				writeMDTaskFile(t, tasksDir, "task_01.md", "pending", "First task", "# First\n")
				writeMDTaskFile(t, tasksDir, "task_02.md", "pending", "Second task", "# Second\n")
				writeMDTaskFile(t, tasksDir, "task_03.md", "pending", "Third task", "# Third\n")

				_, _, err := fileImportOutputRef(dsl.FileParseMDTasks, filepath.Join(tasksDir, "task_*.md"))
				if !errors.Is(err, ErrValidation) || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("fileImportOutputRef(md_tasks) error = %v, want %q validation", err, tc.wantErr)
				}
			})
		}
	})

	t.Run("Should reject cyclic task graphs", func(t *testing.T) {
		t.Parallel()

		tasksDir := t.TempDir()
		writeMDTasksManifest(t, tasksDir, []string{
			"    - from: task_01",
			"      to: task_02",
			"    - from: task_02",
			"      to: task_01",
		})
		writeMDTaskFile(t, tasksDir, "task_01.md", "pending", "First task", "# First\n")
		writeMDTaskFile(t, tasksDir, "task_02.md", "pending", "Second task", "# Second\n")
		writeMDTaskFile(t, tasksDir, "task_03.md", "pending", "Third task", "# Third\n")

		_, _, err := fileImportOutputRef(dsl.FileParseMDTasks, filepath.Join(tasksDir, "task_*.md"))
		if !errors.Is(err, ErrValidation) || !strings.Contains(err.Error(), "cycle") {
			t.Fatalf("fileImportOutputRef(md_tasks) error = %v, want cycle validation", err)
		}
	})
}

func writeMDTasksManifest(t *testing.T, tasksDir string, edges []string) {
	t.Helper()
	lines := []string{
		"---",
		`schema_version: "compozy.tasks/v2"`,
		"workflow: demo",
		"graph:",
		"  nodes:",
		"    - id: task_01",
		"      file: task_01.md",
		"    - id: task_02",
		"      file: task_02.md",
		"    - id: task_03",
		"      file: task_03.md",
	}
	if len(edges) == 0 {
		lines = append(lines, "  edges: []")
	} else {
		lines = append(lines, "  edges:")
		lines = append(lines, edges...)
	}
	lines = append(lines, "---", "", "# Demo tasks", "")
	writeMDTasksFile(t, tasksDir, "_tasks.md", strings.Join(lines, "\n"))
}

func writeMDTaskFile(t *testing.T, tasksDir string, name string, status string, title string, body string) {
	t.Helper()
	writeMDTaskFileWithFrontmatter(t, tasksDir, name, []string{
		"status: " + status,
		"title: " + title,
	}, body)
}

func writeMDTaskFileWithFrontmatter(
	t *testing.T,
	tasksDir string,
	name string,
	metadata []string,
	body string,
) {
	t.Helper()
	lines := append([]string{"---"}, metadata...)
	lines = append(lines, "---", "", body)
	writeMDTasksFile(t, tasksDir, name, strings.Join(lines, "\n"))
}

func writeMDTasksFile(t *testing.T, tasksDir string, name string, content string) {
	t.Helper()
	path := filepath.Join(tasksDir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}
