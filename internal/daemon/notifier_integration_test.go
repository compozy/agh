//go:build integration

package daemon

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/session"
	skillspkg "github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/testutil"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestNotifierFanoutExecutesCreatedAndStoppedHooks(t *testing.T) {
	workDir := t.TempDir()
	rootDir := filepath.Join(workDir, "workspace")
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(workspace) error = %v", err)
	}

	scriptPath := writeIntegrationHookScript(t, workDir, "capture.sh", "#!/bin/sh\ncat > \"$1\"\n")
	createdOutput := filepath.Join(workDir, "created.json")
	stoppedOutput := filepath.Join(workDir, "stopped.json")

	registry := &integrationHookRegistry{
		skills: []*skillspkg.Skill{
			{
				Source: skillspkg.SourceWorkspace,
				Meta:   skillspkg.SkillMeta{Name: "hook-skill"},
				Hooks: []skillspkg.HookDecl{
					{
						Event:   skillspkg.HookSessionCreated,
						Command: scriptPath,
						Args:    []string{createdOutput},
					},
					{
						Event:   skillspkg.HookSessionStopped,
						Command: scriptPath,
						Args:    []string{stoppedOutput},
					},
				},
			},
		},
	}
	resolver := &integrationHookWorkspaceResolver{
		resolved: workspacepkg.ResolvedWorkspace{
			Workspace: workspacepkg.Workspace{
				ID:      "ws-1",
				RootDir: rootDir,
				Name:    "workspace",
			},
		},
	}

	fanout := notifierFanout{
		notifiers: []session.Notifier{&recordingNotifier{}},
		hookPhase: newSkillsHookDispatcher(registry, skillspkg.NewHookRunner(aghconfig.SkillsConfig{}, discardLogger()), resolver, discardLogger()),
	}
	sess := &session.Session{
		ID:          "sess-1",
		AgentName:   "coder",
		WorkspaceID: "ws-1",
		Workspace:   filepath.Join(workDir, "non-canonical"),
	}

	fanout.OnSessionCreated(testutil.Context(t), sess)
	fanout.OnSessionStopped(testutil.Context(t), sess)

	assertHookPayload(t, createdOutput, skillspkg.HookPayload{
		SessionID: "sess-1",
		AgentName: "coder",
		Workspace: rootDir,
		Event:     string(skillspkg.HookSessionCreated),
	})
	assertHookPayload(t, stoppedOutput, skillspkg.HookPayload{
		SessionID: "sess-1",
		AgentName: "coder",
		Workspace: rootDir,
		Event:     string(skillspkg.HookSessionStopped),
	})
}

func TestNotifierFanoutHookFailureDoesNotBlockLifecycle(t *testing.T) {
	workDir := t.TempDir()
	rootDir := filepath.Join(workDir, "workspace")
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(workspace) error = %v", err)
	}

	scriptPath := writeIntegrationHookScript(t, workDir, "fail.sh", "#!/bin/sh\nexit 7\n")
	registry := &integrationHookRegistry{
		skills: []*skillspkg.Skill{
			{
				Source: skillspkg.SourceWorkspace,
				Meta:   skillspkg.SkillMeta{Name: "failing-hook-skill"},
				Hooks: []skillspkg.HookDecl{
					{
						Event:   skillspkg.HookSessionCreated,
						Command: scriptPath,
					},
					{
						Event:   skillspkg.HookSessionStopped,
						Command: scriptPath,
					},
				},
			},
		},
	}
	resolver := &integrationHookWorkspaceResolver{
		resolved: workspacepkg.ResolvedWorkspace{
			Workspace: workspacepkg.Workspace{
				ID:      "ws-1",
				RootDir: rootDir,
				Name:    "workspace",
			},
		},
	}
	notifier := &recordingNotifier{}
	fanout := notifierFanout{
		notifiers: []session.Notifier{notifier},
		hookPhase: newSkillsHookDispatcher(registry, skillspkg.NewHookRunner(aghconfig.SkillsConfig{}, discardLogger()), resolver, discardLogger()),
	}

	sess := &session.Session{ID: "sess-1", AgentName: "coder", WorkspaceID: "ws-1"}
	fanout.OnSessionCreated(testutil.Context(t), sess)
	fanout.OnSessionStopped(testutil.Context(t), sess)

	if got, want := notifier.events, []string{"created", "stopped"}; !testutil.EqualStringSlices(got, want) {
		t.Fatalf("built-in notifier events = %#v, want %#v", got, want)
	}
}

type integrationHookRegistry struct {
	skills []*skillspkg.Skill
}

func (r *integrationHookRegistry) ForWorkspace(context.Context, workspacepkg.ResolvedWorkspace) ([]*skillspkg.Skill, error) {
	return append([]*skillspkg.Skill(nil), r.skills...), nil
}

type integrationHookWorkspaceResolver struct {
	resolved workspacepkg.ResolvedWorkspace
}

func (r *integrationHookWorkspaceResolver) Resolve(context.Context, string) (workspacepkg.ResolvedWorkspace, error) {
	return r.resolved, nil
}

func (r *integrationHookWorkspaceResolver) ResolveOrRegister(context.Context, string) (workspacepkg.ResolvedWorkspace, error) {
	return workspacepkg.ResolvedWorkspace{}, nil
}

func writeIntegrationHookScript(t *testing.T, dir string, name string, contents string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
	return path
}

func assertHookPayload(t *testing.T, path string, want skillspkg.HookPayload) {
	t.Helper()

	payloadBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}
	var got skillspkg.HookPayload
	if err := json.Unmarshal(payloadBytes, &got); err != nil {
		t.Fatalf("json.Unmarshal(%q) error = %v", path, err)
	}
	if got != want {
		t.Fatalf("hook payload = %#v, want %#v", got, want)
	}
}
