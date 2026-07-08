package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/compozy/agh/internal/acp"
	aghconfig "github.com/compozy/agh/internal/config"
	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/network"
	"github.com/compozy/agh/internal/session"
	toolspkg "github.com/compozy/agh/internal/tools"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

func TestLoopRuntimeSessionChannelShouldMatchNetworkGrammar(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		workspaceID string
		suffix      string
		want        string
	}{
		{
			name:        "Should preserve generated workspace ids",
			workspaceID: "ws_524b04478f9c1062",
			suffix:      "main",
			want:        "loop_ws_524b04478f9c1062_main",
		},
		{
			name:        "Should normalize invalid separators",
			workspaceID: "WS:Alpha/Primary",
			suffix:      "Review Gate:Approval",
			want:        "loop_ws_alpha_primary_review_gate_approval",
		},
		{
			name:        "Should fall back for empty fragments",
			workspaceID: " ",
			suffix:      " ",
			want:        "loop_workspace_main",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := loopRuntimeSessionChannel(looppkg.WorkspaceID(tc.workspaceID), tc.suffix)
			if got != tc.want {
				t.Fatalf("loopRuntimeSessionChannel() = %q, want %q", got, tc.want)
			}
			if err := network.ValidateChannel(got); err != nil {
				t.Fatalf("ValidateChannel(%q) error = %v", got, err)
			}
		})
	}
}

func TestLoopRuntimeSessionChannelShouldBoundLongHandles(t *testing.T) {
	t.Parallel()

	t.Run("Should bound long handles and keep workspace scoped prefix", func(t *testing.T) {
		t.Parallel()

		got := loopRuntimeSessionChannel(
			looppkg.WorkspaceID("ws_524b04478f9c1062"),
			strings.Repeat("long-node-handle-", 8),
		)
		if len(got) > 64 {
			t.Fatalf("loopRuntimeSessionChannel() length = %d, want <= 64: %q", len(got), got)
		}
		if err := network.ValidateChannel(got); err != nil {
			t.Fatalf("ValidateChannel(%q) error = %v", got, err)
		}
		if !strings.HasPrefix(got, "loop_ws_524b04478f9c1062_") {
			t.Fatalf("loopRuntimeSessionChannel() = %q, want workspace-scoped prefix", got)
		}
	})
}

func TestLoopActionSessionBinderShouldApplyPolicyGate(t *testing.T) {
	t.Parallel()

	t.Run("Should bind run-agent sessions with workspace policy and allowed tools", func(t *testing.T) {
		t.Parallel()

		allowedTools := []string{
			string(toolspkg.ToolIDTaskRead),
			string(toolspkg.ToolIDTaskUpdate),
		}
		resolved := loopActionBinderWorkspace(t, []aghconfig.AgentDef{{
			Name:        "task-worker",
			Provider:    "mock",
			Prompt:      "Handle the loop node.",
			Tools:       append([]string(nil), allowedTools...),
			Permissions: string(aghconfig.PermissionModeDenyAll),
		}})
		sessions := &loopActionBinderSessionManager{sessionID: "sess-loop-policy"}
		binder := &loopActionSessionBinder{
			sessions: sessions,
			workspaceResolver: loopActionBinderWorkspaceResolver{
				byID: map[string]workspacepkg.ResolvedWorkspace{"ws-loop": resolved},
			},
		}

		binding, err := binder.BindActionSession(context.Background(), looppkg.ActionSessionBindRequest{
			WorkspaceID:   looppkg.WorkspaceID("ws-loop"),
			Agent:         "task-worker",
			Handle:        "execute_task",
			AllowedTools:  []string{allowedTools[1], allowedTools[0], allowedTools[0]},
			ContractBlock: "Follow the loop contract.",
		})
		if err != nil {
			t.Fatalf("BindActionSession() error = %v", err)
		}
		if got, want := binding.SessionID, "sess-loop-policy"; got != want {
			t.Fatalf("binding.SessionID = %q, want %q", got, want)
		}
		createCall := sessions.singleCreateCall(t)
		if got, want := createCall.SandboxRef, "evidence-lab"; got != want {
			t.Fatalf("CreateOpts.SandboxRef = %q, want %q", got, want)
		}
		if got, want := createCall.Permissions, aghconfig.PermissionModeDenyAll; got != want {
			t.Fatalf("CreateOpts.Permissions = %q, want %q", got, want)
		}
		if got, want := createCall.Workspace, "ws-loop"; got != want {
			t.Fatalf("CreateOpts.Workspace = %q, want %q", got, want)
		}
		if !slices.Equal(createCall.AllowedToolsOverride, allowedTools) {
			t.Fatalf(
				"CreateOpts.AllowedToolsOverride = %#v, want %#v",
				createCall.AllowedToolsOverride,
				allowedTools,
			)
		}
	})

	t.Run("Should propagate allowed-tools widening failures without a session binding", func(t *testing.T) {
		t.Parallel()

		readTool := string(toolspkg.ToolIDTaskRead)
		updateTool := string(toolspkg.ToolIDTaskUpdate)
		resolved := loopActionBinderWorkspace(t, []aghconfig.AgentDef{{
			Name:     "task-worker",
			Provider: "mock",
			Prompt:   "Handle the loop node.",
			Tools:    []string{readTool},
		}})
		wideningErr := fmt.Errorf(
			"%w: allowed_tools includes %q which widens agent profile",
			session.ErrValidation,
			updateTool,
		)
		sessions := &loopActionBinderSessionManager{createErr: wideningErr}
		binder := &loopActionSessionBinder{
			sessions: sessions,
			workspaceResolver: loopActionBinderWorkspaceResolver{
				byID: map[string]workspacepkg.ResolvedWorkspace{"ws-loop": resolved},
			},
		}

		_, err := binder.BindActionSession(context.Background(), looppkg.ActionSessionBindRequest{
			WorkspaceID:   looppkg.WorkspaceID("ws-loop"),
			Agent:         "task-worker",
			Handle:        "execute_task",
			AllowedTools:  []string{updateTool},
			ContractBlock: "Follow the loop contract.",
		})
		if !errors.Is(err, session.ErrValidation) {
			t.Fatalf("BindActionSession() error = %v, want session.ErrValidation", err)
		}
		if !strings.Contains(err.Error(), updateTool) {
			t.Fatalf("BindActionSession() error = %v, want rejected tool id", err)
		}
		if got, want := sessions.createCount(), 1; got != want {
			t.Fatalf("Create call count = %d, want %d validation attempt", got, want)
		}
	})

	t.Run("Should fail deterministically when the target agent cannot resolve", func(t *testing.T) {
		t.Parallel()

		resolved := loopActionBinderWorkspace(t, nil)
		sessions := &loopActionBinderSessionManager{sessionID: "unused"}
		binder := &loopActionSessionBinder{
			sessions: sessions,
			workspaceResolver: loopActionBinderWorkspaceResolver{
				byID: map[string]workspacepkg.ResolvedWorkspace{"ws-loop": resolved},
			},
		}

		_, err := binder.BindActionSession(context.Background(), looppkg.ActionSessionBindRequest{
			WorkspaceID: looppkg.WorkspaceID("ws-loop"),
			Agent:       "missing-worker",
			Handle:      "execute_task",
		})
		if !errors.Is(err, workspacepkg.ErrAgentNotAvailable) {
			t.Fatalf("BindActionSession() error = %v, want ErrAgentNotAvailable", err)
		}
		if got := sessions.createCount(); got != 0 {
			t.Fatalf("Create call count = %d, want 0", got)
		}
	})
}

func TestCollectLoopPromptResultShouldNotTreatProtocolRawAsStructuredOutput(t *testing.T) {
	t.Parallel()

	t.Run("Should ignore ACP protocol raw and preserve agent text", func(t *testing.T) {
		t.Parallel()

		manager := loopPromptResultSessionManager{
			events: []acp.AgentEvent{{
				Text: "```json\n{\"summary\":\"done\",\"message\":\"loop channel result\"}\n```",
				Raw:  json.RawMessage(`{"session_update":"agent_message_chunk"}`),
			}},
		}
		result, err := collectLoopPromptResult(
			context.Background(),
			manager,
			"sess-loop",
			looppkg.ActionPromptRequest{Message: "loop event probe"},
		)
		if err != nil {
			t.Fatalf("collectLoopPromptResult() error = %v", err)
		}
		if strings.TrimSpace(result.Text) == "" || !strings.Contains(result.Text, `"summary":"done"`) {
			t.Fatalf("collectLoopPromptResult() text = %q, want agent text", result.Text)
		}
		if len(result.Structured) != 0 {
			t.Fatalf("collectLoopPromptResult() structured = %s, want empty protocol raw ignored", result.Structured)
		}
	})
}

type loopActionBinderSessionManager struct {
	mu          sync.Mutex
	sessionID   string
	createErr   error
	createCalls []session.CreateOpts
}

func (m *loopActionBinderSessionManager) Create(
	_ context.Context,
	opts session.CreateOpts,
) (*session.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createCalls = append(m.createCalls, opts)
	if m.createErr != nil {
		return nil, m.createErr
	}
	sessionID := strings.TrimSpace(m.sessionID)
	if sessionID == "" {
		sessionID = "sess-loop-action"
	}
	return &session.Session{
		ID:          sessionID,
		Name:        opts.Name,
		AgentName:   opts.AgentName,
		Model:       opts.Model,
		WorkspaceID: opts.Workspace,
		Workspace:   firstNonEmpty(opts.Workspace, opts.WorkspacePath),
		Channel:     opts.Channel,
		Type:        opts.Type,
		State:       session.StateActive,
		CreatedAt:   time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC),
	}, nil
}

func (m *loopActionBinderSessionManager) Prompt(
	context.Context,
	string,
	string,
) (<-chan acp.AgentEvent, error) {
	events := make(chan acp.AgentEvent)
	close(events)
	return events, nil
}

func (m *loopActionBinderSessionManager) CancelPrompt(context.Context, string) error {
	return nil
}

func (m *loopActionBinderSessionManager) singleCreateCall(t *testing.T) session.CreateOpts {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.createCalls) != 1 {
		t.Fatalf("Create call count = %d, want 1", len(m.createCalls))
	}
	return m.createCalls[0]
}

func (m *loopActionBinderSessionManager) createCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.createCalls)
}

type loopActionBinderWorkspaceResolver struct {
	byID   map[string]workspacepkg.ResolvedWorkspace
	byPath map[string]workspacepkg.ResolvedWorkspace
}

func (r loopActionBinderWorkspaceResolver) Resolve(
	_ context.Context,
	idOrPath string,
) (workspacepkg.ResolvedWorkspace, error) {
	resolved, ok := r.byID[strings.TrimSpace(idOrPath)]
	if !ok {
		return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceNotFound
	}
	return resolved, nil
}

func (r loopActionBinderWorkspaceResolver) ResolveOrRegister(
	_ context.Context,
	path string,
) (workspacepkg.ResolvedWorkspace, error) {
	resolved, ok := r.byPath[strings.TrimSpace(path)]
	if !ok {
		return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceNotFound
	}
	return resolved, nil
}

func loopActionBinderWorkspace(t *testing.T, agents []aghconfig.AgentDef) workspacepkg.ResolvedWorkspace {
	t.Helper()
	cfg := aghconfig.DefaultWithHome(aghconfig.HomePaths{HomeDir: t.TempDir()})
	cfg.Defaults.Provider = "mock"
	cfg.Defaults.Sandbox = "evidence-lab"
	cfg.Providers["mock"] = aghconfig.ProviderConfig{Command: "mock-acp"}
	cfg.Sandboxes["evidence-lab"] = aghconfig.SandboxProfile{Backend: "local"}
	return workspacepkg.ResolvedWorkspace{
		Workspace: workspacepkg.Workspace{
			ID:         "ws-loop",
			RootDir:    t.TempDir(),
			SandboxRef: "evidence-lab",
		},
		WorkspaceID: "ws-loop",
		Config:      cfg,
		Agents:      append([]aghconfig.AgentDef(nil), agents...),
	}
}

type loopPromptResultSessionManager struct {
	events []acp.AgentEvent
}

func (m loopPromptResultSessionManager) Create(context.Context, session.CreateOpts) (*session.Session, error) {
	return nil, errors.New("unexpected Create call")
}

func (m loopPromptResultSessionManager) Prompt(
	ctx context.Context,
	_ string,
	_ string,
) (<-chan acp.AgentEvent, error) {
	events := make(chan acp.AgentEvent, len(m.events))
	for _, event := range m.events {
		select {
		case events <- event:
		case <-ctx.Done():
			close(events)
			return nil, ctx.Err()
		}
	}
	close(events)
	return events, nil
}

func (m loopPromptResultSessionManager) CancelPrompt(context.Context, string) error {
	return errors.New("unexpected CancelPrompt call")
}
