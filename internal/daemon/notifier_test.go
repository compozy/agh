package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/session"
	skillspkg "github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/testutil"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestNotifierFanoutRunsHookPhaseAfterBuiltInNotifiers(t *testing.T) {
	t.Parallel()

	order := make([]string, 0, 4)
	fanout := notifierFanout{
		notifiers: []session.Notifier{
			notifierFunc{
				onCreated: func(context.Context, *session.Session) {
					order = append(order, "notifier-created")
				},
				onStopped: func(context.Context, *session.Session) {
					order = append(order, "notifier-stopped")
				},
			},
		},
		hookPhase: hookPhaseRecorder{
			onCreated: func(context.Context, *session.Session) {
				order = append(order, "hook-created")
			},
			onStopped: func(context.Context, *session.Session) {
				order = append(order, "hook-stopped")
			},
		},
	}

	fanout.OnSessionCreated(testutil.Context(t), &session.Session{ID: "sess-created"})
	fanout.OnSessionStopped(testutil.Context(t), &session.Session{ID: "sess-stopped"})

	want := []string{"notifier-created", "hook-created", "notifier-stopped", "hook-stopped"}
	if !testutil.EqualStringSlices(order, want) {
		t.Fatalf("fanout order = %#v, want %#v", order, want)
	}
}

func TestSkillsHookDispatcherUsesResolvedWorkspaceForLookupAndPayload(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	rootDir := filepath.Join(workDir, "workspace")
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(workspace) error = %v", err)
	}

	scriptPath := writeHookScript(t, workDir, "capture.sh", "#!/bin/sh\ncat > \"$1\"\n")
	outputPath := filepath.Join(workDir, "created.json")

	registry := &hookDispatcherRegistry{
		skills: []*skillspkg.Skill{
			{
				Source: skillspkg.SourceWorkspace,
				Meta:   skillspkg.SkillMeta{Name: "hook-skill"},
				Hooks: []hookspkg.HookDecl{
					{
						Name:        "hook-skill",
						Event:       hookspkg.HookSessionPostCreate,
						Source:      hookspkg.HookSourceSkill,
						Mode:        hookspkg.HookModeSync,
						Command:     scriptPath,
						Args:        []string{outputPath},
						SkillSource: hookspkg.HookSkillSourceWorkspace,
					},
				},
			},
		},
	}
	resolver := &hookDispatcherWorkspaceResolver{
		resolved: workspacepkg.ResolvedWorkspace{
			Workspace: workspacepkg.Workspace{
				ID:      "ws-1",
				RootDir: rootDir,
				Name:    "workspace",
			},
		},
	}

	dispatcher := newSkillsHookDispatcher(registry, aghconfig.SkillsConfig{}, resolver, discardLogger())
	dispatcher.OnSessionCreated(testutil.Context(t), &session.Session{
		ID:          "sess-1",
		AgentName:   "coder",
		WorkspaceID: "ws-1",
		Workspace:   filepath.Join(workDir, "non-canonical"),
	})

	if got := resolver.callCount(); got != 1 {
		t.Fatalf("workspace resolver call count = %d, want 1", got)
	}
	if got := resolver.call(0); got != "ws-1" {
		t.Fatalf("workspace resolver call = %q, want %q", got, "ws-1")
	}
	if got := registry.callCount(); got != 1 {
		t.Fatalf("registry call count = %d, want 1", got)
	}
	if got := registry.call(0).RootDir; got != rootDir {
		t.Fatalf("registry workspace root = %q, want %q", got, rootDir)
	}

	payloadBytes, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", outputPath, err)
	}
	var payload hookspkg.SessionPostCreatePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		t.Fatalf("json.Unmarshal(payload) error = %v", err)
	}
	if payload.SessionID != "sess-1" {
		t.Fatalf("payload.SessionID = %q, want %q", payload.SessionID, "sess-1")
	}
	if payload.AgentName != "coder" {
		t.Fatalf("payload.AgentName = %q, want %q", payload.AgentName, "coder")
	}
	if payload.Workspace != rootDir {
		t.Fatalf("payload.Workspace = %q, want %q", payload.Workspace, rootDir)
	}
	if payload.Event != hookspkg.HookSessionPostCreate {
		t.Fatalf("payload.Event = %q, want %q", payload.Event, hookspkg.HookSessionPostCreate)
	}
}

func TestSkillsHookDispatcherSkipsUntrustedMarketplaceHooks(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	rootDir := filepath.Join(workDir, "workspace")
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(workspace) error = %v", err)
	}

	scriptPath := writeHookScript(t, workDir, "capture.sh", "#!/bin/sh\ncat > \"$1\"\n")
	outputPath := filepath.Join(workDir, "blocked.json")

	registry := &hookDispatcherRegistry{
		skills: []*skillspkg.Skill{
			{
				Source: skillspkg.SourceMarketplace,
				Meta:   skillspkg.SkillMeta{Name: "marketplace-skill"},
				Hooks: []hookspkg.HookDecl{
					{
						Name:        "marketplace-skill",
						Event:       hookspkg.HookSessionPostCreate,
						Source:      hookspkg.HookSourceSkill,
						Mode:        hookspkg.HookModeSync,
						Command:     scriptPath,
						Args:        []string{outputPath},
						SkillSource: hookspkg.HookSkillSourceMarketplace,
					},
				},
			},
		},
	}
	resolver := &hookDispatcherWorkspaceResolver{
		resolved: workspacepkg.ResolvedWorkspace{
			Workspace: workspacepkg.Workspace{
				ID:      "ws-1",
				RootDir: rootDir,
				Name:    "workspace",
			},
		},
	}

	dispatcher := newSkillsHookDispatcher(registry, aghconfig.SkillsConfig{}, resolver, discardLogger())
	dispatcher.OnSessionCreated(testutil.Context(t), &session.Session{
		ID:          "sess-1",
		AgentName:   "coder",
		WorkspaceID: "ws-1",
	})

	if _, err := os.Stat(outputPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("os.Stat(%q) error = %v, want os.ErrNotExist", outputPath, err)
	}
}

type notifierFunc struct {
	onCreated func(context.Context, *session.Session)
	onStopped func(context.Context, *session.Session)
}

func (n notifierFunc) OnSessionCreated(ctx context.Context, sess *session.Session) {
	if n.onCreated != nil {
		n.onCreated(ctx, sess)
	}
}

func (n notifierFunc) OnSessionStopped(ctx context.Context, sess *session.Session) {
	if n.onStopped != nil {
		n.onStopped(ctx, sess)
	}
}

func (n notifierFunc) OnAgentEvent(context.Context, string, any) {}

type hookPhaseRecorder struct {
	onCreated func(context.Context, *session.Session)
	onStopped func(context.Context, *session.Session)
}

func (h hookPhaseRecorder) OnSessionCreated(ctx context.Context, sess *session.Session) {
	if h.onCreated != nil {
		h.onCreated(ctx, sess)
	}
}

func (h hookPhaseRecorder) OnSessionStopped(ctx context.Context, sess *session.Session) {
	if h.onStopped != nil {
		h.onStopped(ctx, sess)
	}
}

type hookDispatcherRegistry struct {
	skills []*skillspkg.Skill
	calls  []workspacepkg.ResolvedWorkspace
	err    error
}

func (r *hookDispatcherRegistry) ForWorkspace(_ context.Context, resolved workspacepkg.ResolvedWorkspace) ([]*skillspkg.Skill, error) {
	r.calls = append(r.calls, resolved)
	if r.err != nil {
		return nil, r.err
	}
	return append([]*skillspkg.Skill(nil), r.skills...), nil
}

func (r *hookDispatcherRegistry) callCount() int {
	return len(r.calls)
}

func (r *hookDispatcherRegistry) call(index int) workspacepkg.ResolvedWorkspace {
	return r.calls[index]
}

type hookDispatcherWorkspaceResolver struct {
	resolved workspacepkg.ResolvedWorkspace
	calls    []string
	err      error
}

func (r *hookDispatcherWorkspaceResolver) Resolve(_ context.Context, idOrPath string) (workspacepkg.ResolvedWorkspace, error) {
	r.calls = append(r.calls, idOrPath)
	if r.err != nil {
		return workspacepkg.ResolvedWorkspace{}, r.err
	}
	return r.resolved, nil
}

func (r *hookDispatcherWorkspaceResolver) ResolveOrRegister(context.Context, string) (workspacepkg.ResolvedWorkspace, error) {
	return workspacepkg.ResolvedWorkspace{}, errors.New("unexpected ResolveOrRegister call")
}

func (r *hookDispatcherWorkspaceResolver) callCount() int {
	return len(r.calls)
}

func (r *hookDispatcherWorkspaceResolver) call(index int) string {
	return r.calls[index]
}

func writeHookScript(t *testing.T, dir string, name string, contents string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
	return path
}
