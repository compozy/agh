package daemon

import (
	"reflect"
	"strings"
	"testing"

	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/session"
	taskpkg "github.com/compozy/agh/internal/task"
	toolspkg "github.com/compozy/agh/internal/tools"
)

func TestSessionPolicyGateAppliesSandboxPolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		policy  SessionPolicy
		wantRef string
		wantOff bool
	}{
		{
			name:    "Should disable sandbox for none mode",
			policy:  SessionPolicy{Sandbox: SessionSandboxPolicy{Mode: SessionSandboxModeNone, SandboxRef: "ignored"}},
			wantOff: true,
		},
		{
			name: "Should propagate sandbox ref for ref mode",
			policy: SessionPolicy{Sandbox: SessionSandboxPolicy{
				Mode:       SessionSandboxModeRef,
				SandboxRef: " evidence-lab ",
			}},
			wantRef: "evidence-lab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := session.CreateOpts{SandboxRef: "preexisting"}
			applySessionSandboxPolicy(&opts, tt.policy)

			if got := opts.DisableSandbox; got != tt.wantOff {
				t.Fatalf("DisableSandbox = %v, want %v", got, tt.wantOff)
			}
			if got := opts.SandboxRef; got != tt.wantRef {
				t.Fatalf("SandboxRef = %q, want %q", got, tt.wantRef)
			}
		})
	}
}

func TestSessionPolicyGateAppliesEvidencePermissionPolicy(t *testing.T) {
	t.Parallel()

	t.Run("Should auto approve evidence mode only with sandbox ref", func(t *testing.T) {
		t.Parallel()

		opts := session.CreateOpts{Permissions: aghconfig.PermissionModeDenyAll}
		applySessionPermissionPolicy(&opts, SessionPolicy{
			Sandbox: SessionSandboxPolicy{Mode: SessionSandboxModeRef, SandboxRef: "evidence-lab"},
			Runtime: SessionRuntimePolicy{Mode: SessionRuntimeModeEvidence},
		})

		if got, want := opts.Permissions, aghconfig.PermissionModeApproveAll; got != want {
			t.Fatalf("Permissions = %q, want %q", got, want)
		}
		if !strings.Contains(opts.PromptOverlay, "Runtime evidence mode is enabled") {
			t.Fatalf("PromptOverlay = %q, want evidence guidance", opts.PromptOverlay)
		}
	})

	t.Run("Should preserve configured permission mode without sandbox ref", func(t *testing.T) {
		t.Parallel()

		opts := session.CreateOpts{Permissions: aghconfig.PermissionModeDenyAll}
		applySessionPermissionPolicy(&opts, SessionPolicy{
			Runtime: SessionRuntimePolicy{Mode: SessionRuntimeModeEvidence},
		})

		if got, want := opts.Permissions, aghconfig.PermissionModeDenyAll; got != want {
			t.Fatalf("Permissions = %q, want configured %q", got, want)
		}
		if !strings.Contains(opts.PromptOverlay, "did not select a sandbox") {
			t.Fatalf("PromptOverlay = %q, want sandbox warning", opts.PromptOverlay)
		}
	})
}

func TestSessionPolicyGateAppliesAllowedToolsNarrowing(t *testing.T) {
	t.Parallel()

	t.Run("Should normalize concrete allowed tools", func(t *testing.T) {
		t.Parallel()

		opts := session.CreateOpts{}
		err := applyAllowedToolsNarrowing(&opts, []string{
			" " + toolspkg.ToolIDTaskUpdate.String() + " ",
			toolspkg.ToolIDTaskRead.String(),
			toolspkg.ToolIDTaskRead.String(),
		})
		if err != nil {
			t.Fatalf("applyAllowedToolsNarrowing() error = %v", err)
		}
		want := []string{toolspkg.ToolIDTaskRead.String(), toolspkg.ToolIDTaskUpdate.String()}
		if !reflect.DeepEqual(opts.AllowedToolsOverride, want) {
			t.Fatalf("AllowedToolsOverride = %#v, want %#v", opts.AllowedToolsOverride, want)
		}
	})

	t.Run("Should reject non canonical requested tools", func(t *testing.T) {
		t.Parallel()

		err := applyAllowedToolsNarrowing(&session.CreateOpts{}, []string{"Read"})
		if err == nil || !strings.Contains(err.Error(), "allowed_tools[0]") {
			t.Fatalf("applyAllowedToolsNarrowing() error = %v, want indexed validation error", err)
		}
	})
}

func TestSessionPolicyGateKeepsTaskRoleCreateOptsParity(t *testing.T) {
	t.Parallel()

	t.Run("Should preserve task-role create options through the shared gate", func(t *testing.T) {
		t.Parallel()

		activation := taskRoleActivation{
			TaskID:        "task-parity",
			RunID:         "run-parity",
			Scope:         taskpkg.ScopeWorkspace,
			WorkspaceID:   "ws-parity",
			AgentName:     "frontend-engineer",
			Provider:      "claude",
			Model:         "sonnet",
			Channel:       "design-review",
			Title:         "Parity task",
			Capabilities:  []string{"frontend"},
			Profile:       legacyTaskRoleParityProfile(),
			WorkspacePath: "/unused",
		}

		got, err := taskRoleCreateOpts(activation)
		if err != nil {
			t.Fatalf("taskRoleCreateOpts() error = %v", err)
		}
		want, err := legacyTaskRoleCreateOptsForTest(activation)
		if err != nil {
			t.Fatalf("legacyTaskRoleCreateOptsForTest() error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("taskRoleCreateOpts() = %#v, want legacy parity %#v", got, want)
		}
	})
}

func legacyTaskRoleParityProfile() *taskpkg.ExecutionProfile {
	return &taskpkg.ExecutionProfile{
		TaskID: "task-parity",
		Worker: taskpkg.WorkerProfile{
			Mode:      taskpkg.WorkerModeSelect,
			AgentName: "frontend-engineer",
			Provider:  "claude",
			Model:     "sonnet",
		},
		Sandbox: taskpkg.SandboxPolicy{
			Mode:       taskpkg.SandboxModeRef,
			SandboxRef: "evidence-lab",
		},
		Runtime: taskpkg.RuntimePolicy{Mode: taskpkg.RuntimeModeEvidence},
	}
}

func legacyTaskRoleCreateOptsForTest(activation taskRoleActivation) (session.CreateOpts, error) {
	opts := session.CreateOpts{
		AgentName:     activation.AgentName,
		Provider:      activation.Provider,
		Model:         activation.Model,
		Name:          taskRoleSessionName(activation),
		Channel:       activation.Channel,
		PromptOverlay: taskRolePromptOverlay(activation),
		Type:          session.SessionTypeSystem,
	}
	legacyApplyTaskSessionSandboxProfileForTest(&opts, activation.Profile)
	legacyApplyTaskSessionRuntimeProfileForTest(&opts, activation.Profile)
	switch activation.Scope {
	case taskpkg.ScopeWorkspace:
		opts.Workspace = activation.WorkspaceID
	case taskpkg.ScopeGlobal:
		opts.WorkspacePath = activation.WorkspacePath
	default:
		return session.CreateOpts{}, taskpkg.ErrValidation
	}
	return opts, nil
}

func legacyApplyTaskSessionSandboxProfileForTest(opts *session.CreateOpts, profile *taskpkg.ExecutionProfile) {
	if opts == nil || profile == nil {
		return
	}
	switch profile.Sandbox.Mode.Normalize() {
	case taskpkg.SandboxModeNone:
		opts.DisableSandbox = true
		opts.SandboxRef = ""
	case taskpkg.SandboxModeRef:
		opts.DisableSandbox = false
		opts.SandboxRef = strings.TrimSpace(profile.Sandbox.SandboxRef)
	default:
		return
	}
}

func legacyApplyTaskSessionRuntimeProfileForTest(opts *session.CreateOpts, profile *taskpkg.ExecutionProfile) {
	if opts == nil || profile == nil {
		return
	}
	if profile.Runtime.Mode.Normalize() != taskpkg.RuntimeModeEvidence {
		return
	}
	guidance := "Runtime evidence mode is enabled for this task. You may boot local app runtimes, " +
		"run browser or simulator validation, and capture runtime evidence artifacts required by the task."
	if legacyTaskRuntimeEvidenceCanAutoApproveForTest(profile) {
		opts.Permissions = aghconfig.PermissionModeApproveAll
	} else {
		guidance += " AGH keeps the configured permission mode because the task profile did not select a sandbox."
	}
	opts.PromptOverlay = joinPromptOverlays(opts.PromptOverlay, guidance)
}

func legacyTaskRuntimeEvidenceCanAutoApproveForTest(profile *taskpkg.ExecutionProfile) bool {
	if profile == nil {
		return false
	}
	return profile.Sandbox.Mode.Normalize() == taskpkg.SandboxModeRef &&
		strings.TrimSpace(profile.Sandbox.SandboxRef) != ""
}
