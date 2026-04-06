package workspace_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/workspace"
)

func TestWorkspaceErrorsMatchViaErrorsIs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		sentinel error
	}{
		{name: "not found", sentinel: workspace.ErrWorkspaceNotFound},
		{name: "root missing", sentinel: workspace.ErrWorkspaceRootMissing},
		{name: "agent unavailable", sentinel: workspace.ErrAgentNotAvailable},
		{name: "name taken", sentinel: workspace.ErrWorkspaceNameTaken},
		{name: "path taken", sentinel: workspace.ErrWorkspacePathTaken},
		{name: "has sessions", sentinel: workspace.ErrWorkspaceHasSessions},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := fmt.Errorf("workspace api: %w", tt.sentinel)
			if !errors.Is(err, tt.sentinel) {
				t.Fatalf("errors.Is(%v, %v) = false, want true", err, tt.sentinel)
			}
		})
	}
}

func TestWorkspaceErrorsAreDistinct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		left error
		want error
	}{
		{
			name: "not found does not match root missing",
			left: workspace.ErrWorkspaceNotFound,
			want: workspace.ErrWorkspaceRootMissing,
		},
		{
			name: "not found does not match agent unavailable",
			left: workspace.ErrWorkspaceNotFound,
			want: workspace.ErrAgentNotAvailable,
		},
		{
			name: "name taken does not match path taken",
			left: workspace.ErrWorkspaceNameTaken,
			want: workspace.ErrWorkspacePathTaken,
		},
		{
			name: "path taken does not match has sessions",
			left: workspace.ErrWorkspacePathTaken,
			want: workspace.ErrWorkspaceHasSessions,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := fmt.Errorf("wrapped: %w", tt.left)
			if errors.Is(err, tt.want) {
				t.Fatalf("errors.Is(%v, %v) = true, want false", err, tt.want)
			}
		})
	}
}

func TestUniqueWorkspaceName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rootDir string
		taken   map[string]struct{}
		want    string
	}{
		{
			name:    "uses base directory name",
			rootDir: "/tmp/project",
			taken:   map[string]struct{}{},
			want:    "project",
		},
		{
			name:    "deduplicates taken name",
			rootDir: "/tmp/project",
			taken:   map[string]struct{}{"project": {}},
			want:    "project-2",
		},
		{
			name:    "falls back for blankish path",
			rootDir: " / ",
			taken:   map[string]struct{}{"workspace": {}},
			want:    "workspace-2",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := workspace.UniqueWorkspaceName(tt.rootDir, tt.taken); got != tt.want {
				t.Fatalf("UniqueWorkspaceName(%q) = %q, want %q", tt.rootDir, got, tt.want)
			}
		})
	}
}

func TestWorkspaceZeroValues(t *testing.T) {
	t.Parallel()

	var ws workspace.Workspace
	if ws.ID != "" {
		t.Fatalf("Workspace.ID = %q, want empty", ws.ID)
	}
	if ws.RootDir != "" {
		t.Fatalf("Workspace.RootDir = %q, want empty", ws.RootDir)
	}
	if ws.AdditionalDirs != nil {
		t.Fatalf("Workspace.AdditionalDirs = %#v, want nil", ws.AdditionalDirs)
	}
	if ws.Name != "" {
		t.Fatalf("Workspace.Name = %q, want empty", ws.Name)
	}
	if ws.DefaultAgent != "" {
		t.Fatalf("Workspace.DefaultAgent = %q, want empty", ws.DefaultAgent)
	}
	if !ws.CreatedAt.IsZero() {
		t.Fatalf("Workspace.CreatedAt = %v, want zero", ws.CreatedAt)
	}
	if !ws.UpdatedAt.IsZero() {
		t.Fatalf("Workspace.UpdatedAt = %v, want zero", ws.UpdatedAt)
	}
}

func TestResolvedWorkspaceZeroValue(t *testing.T) {
	t.Parallel()

	var resolved workspace.ResolvedWorkspace
	if !reflect.DeepEqual(resolved.Workspace, workspace.Workspace{}) {
		t.Fatalf("ResolvedWorkspace.Workspace = %#v, want zero Workspace", resolved.Workspace)
	}
	if !reflect.DeepEqual(resolved.Config, aghconfig.Config{}) {
		t.Fatalf("ResolvedWorkspace.Config = %#v, want zero Config", resolved.Config)
	}
	if resolved.Agents != nil {
		t.Fatalf("ResolvedWorkspace.Agents = %#v, want nil", resolved.Agents)
	}
	if resolved.Skills != nil {
		t.Fatalf("ResolvedWorkspace.Skills = %#v, want nil", resolved.Skills)
	}
	if !resolved.ResolvedAt.IsZero() {
		t.Fatalf("ResolvedWorkspace.ResolvedAt = %v, want zero", resolved.ResolvedAt)
	}
}

func TestWorkspaceStructSurface(t *testing.T) {
	t.Parallel()

	type fieldSpec struct {
		name      string
		fieldType reflect.Type
		tag       string
		embedded  bool
	}

	tests := []struct {
		name   string
		target reflect.Type
		fields []fieldSpec
	}{
		{
			name:   "Workspace",
			target: reflect.TypeOf(workspace.Workspace{}),
			fields: []fieldSpec{
				{name: "ID", fieldType: reflect.TypeOf("")},
				{name: "RootDir", fieldType: reflect.TypeOf("")},
				{name: "AdditionalDirs", fieldType: reflect.TypeOf([]string(nil))},
				{name: "Name", fieldType: reflect.TypeOf("")},
				{name: "DefaultAgent", fieldType: reflect.TypeOf("")},
				{name: "CreatedAt", fieldType: reflect.TypeOf(time.Time{})},
				{name: "UpdatedAt", fieldType: reflect.TypeOf(time.Time{})},
			},
		},
		{
			name:   "ResolvedWorkspace",
			target: reflect.TypeOf(workspace.ResolvedWorkspace{}),
			fields: []fieldSpec{
				{name: "Workspace", fieldType: reflect.TypeOf(workspace.Workspace{}), embedded: true},
				{name: "Config", fieldType: reflect.TypeOf(aghconfig.Config{})},
				{name: "Agents", fieldType: reflect.TypeOf([]aghconfig.AgentDef(nil))},
				{name: "Skills", fieldType: reflect.TypeOf([]workspace.SkillPath(nil))},
				{name: "ResolvedAt", fieldType: reflect.TypeOf(time.Time{})},
			},
		},
		{
			name:   "SkillPath",
			target: reflect.TypeOf(workspace.SkillPath{}),
			fields: []fieldSpec{
				{name: "Dir", fieldType: reflect.TypeOf("")},
				{name: "Source", fieldType: reflect.TypeOf("")},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got, want := tt.target.NumField(), len(tt.fields); got != want {
				t.Fatalf("%s field count = %d, want %d", tt.name, got, want)
			}

			for idx, wantField := range tt.fields {
				field := tt.target.Field(idx)
				if field.Name != wantField.name {
					t.Fatalf("%s field %d name = %q, want %q", tt.name, idx, field.Name, wantField.name)
				}
				if field.Type != wantField.fieldType {
					t.Fatalf("%s field %q type = %v, want %v", tt.name, field.Name, field.Type, wantField.fieldType)
				}
				if field.Tag != reflect.StructTag(wantField.tag) {
					t.Fatalf("%s field %q tag = %q, want %q", tt.name, field.Name, field.Tag, wantField.tag)
				}
				if field.Anonymous != wantField.embedded {
					t.Fatalf("%s field %q embedded = %t, want %t", tt.name, field.Name, field.Anonymous, wantField.embedded)
				}
			}
		})
	}
}
