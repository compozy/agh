package cli

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/session"
)

func TestParseSinceFlagRFC3339(t *testing.T) {
	t.Parallel()

	got, err := parseSinceFlag("2026-04-03T11:55:00Z", func() time.Time { return fixedTestNow })
	if err != nil {
		t.Fatalf("parseSinceFlag(RFC3339) error = %v", err)
	}
	if want := time.Date(2026, 4, 3, 11, 55, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("parseSinceFlag(RFC3339) = %s, want %s", got, want)
	}
}

func TestParseSinceFlagRelativeDuration(t *testing.T) {
	t.Parallel()

	got, err := parseSinceFlag("5m", func() time.Time { return fixedTestNow })
	if err != nil {
		t.Fatalf("parseSinceFlag(5m) error = %v", err)
	}
	if want := fixedTestNow.Add(-5 * time.Minute); !got.Equal(want) {
		t.Fatalf("parseSinceFlag(5m) = %s, want %s", got, want)
	}
}

func TestSessionNewUsesConfigDefaultWhenAgentFlagIsOmitted(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, stubClient{
		createSessionFn: func(_ context.Context, request CreateSessionRequest) (SessionRecord, error) {
			if request.AgentName != "" {
				t.Fatalf("CreateSession() AgentName = %q, want empty", request.AgentName)
			}
			if request.WorkspacePath != "/workspace/project" || request.Workspace != "" {
				t.Fatalf("CreateSession() request = %#v, want workspace_path only", request)
			}
			return SessionRecord{
				ID:            "sess-1",
				AgentName:     "general",
				WorkspaceID:   "ws-1",
				WorkspacePath: request.WorkspacePath,
				State:         string(session.StateActive),
				CreatedAt:     fixedTestNow,
				UpdatedAt:     fixedTestNow,
			}, nil
		},
	})

	stdout, _, err := executeRootCommand(t, deps, "session", "new", "-o", "json")
	if err != nil {
		t.Fatalf("executeRootCommand(session new) error = %v", err)
	}

	var decoded SessionRecord
	if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(session new) error = %v", err)
	}
	if decoded.AgentName != "general" {
		t.Fatalf("decoded.AgentName = %q, want %q", decoded.AgentName, "general")
	}
}

func TestSessionNewWorkspaceOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		request CreateSessionRequest
	}{
		{
			name: "registered workspace",
			args: []string{"session", "new", "--workspace", "ws_abc", "-o", "json"},
			request: CreateSessionRequest{
				Workspace: "ws_abc",
			},
		},
		{
			name: "explicit cwd",
			args: []string{"session", "new", "--cwd", "/tmp/proj", "-o", "json"},
			request: CreateSessionRequest{
				WorkspacePath: "/tmp/proj",
			},
		},
		{
			name: "default cwd fallback",
			args: []string{"session", "new", "-o", "json"},
			request: CreateSessionRequest{
				WorkspacePath: "/workspace/project",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			deps := newTestDeps(t, stubClient{
				createSessionFn: func(_ context.Context, request CreateSessionRequest) (SessionRecord, error) {
					if request.Workspace != tt.request.Workspace || request.WorkspacePath != tt.request.WorkspacePath {
						t.Fatalf("CreateSession() request = %#v, want %#v", request, tt.request)
					}
					return SessionRecord{
						ID:            "sess-1",
						AgentName:     "general",
						WorkspaceID:   "ws-1",
						WorkspacePath: request.WorkspacePath,
						State:         string(session.StateActive),
						CreatedAt:     fixedTestNow,
						UpdatedAt:     fixedTestNow,
					}, nil
				},
			})

			stdout, _, err := executeRootCommand(t, deps, tt.args...)
			if err != nil {
				t.Fatalf("executeRootCommand(%v) error = %v", tt.args, err)
			}

			var decoded SessionRecord
			if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
				t.Fatalf("json.Unmarshal(session new) error = %v", err)
			}
			if decoded.ID != "sess-1" {
				t.Fatalf("decoded.ID = %q, want %q", decoded.ID, "sess-1")
			}
		})
	}
}

func TestSessionNewRejectsInvalidWorkspaceFlags(t *testing.T) {
	t.Parallel()

	code, _, stderr := executeRootCommandWithExit(t, newTestDeps(t, stubClient{}),
		"session", "new", "--workspace", "ws_abc", "--cwd", "/tmp/proj",
	)
	if code != 1 {
		t.Fatalf("executeRootCommandWithExit() code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "--workspace and --cwd are mutually exclusive") {
		t.Fatalf("stderr = %q, want workspace flag validation message", stderr)
	}
}

func TestSessionListPassesWorkspaceFilter(t *testing.T) {
	t.Parallel()

	var seenQuery SessionListQuery

	deps := newTestDeps(t, stubClient{
		listSessionsFn: func(_ context.Context, query SessionListQuery) ([]SessionRecord, error) {
			seenQuery = query
			return []SessionRecord{{
				ID:            "sess-1",
				AgentName:     "general",
				WorkspaceID:   "ws-filtered",
				WorkspacePath: "/workspace/project",
				State:         string(session.StateActive),
				CreatedAt:     fixedTestNow,
				UpdatedAt:     fixedTestNow,
			}}, nil
		},
	})

	stdout, _, err := executeRootCommand(t, deps, "session", "list", "--workspace", "ws-filtered", "--all", "-o", "json")
	if err != nil {
		t.Fatalf("executeRootCommand(session list) error = %v", err)
	}
	if seenQuery.Workspace != "ws-filtered" {
		t.Fatalf("seenQuery.Workspace = %q, want %q", seenQuery.Workspace, "ws-filtered")
	}

	var decoded []SessionRecord
	if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(session list) error = %v", err)
	}
	if len(decoded) != 1 || decoded[0].WorkspaceID != "ws-filtered" {
		t.Fatalf("decoded = %#v, want filtered session", decoded)
	}
}

func TestSessionEventsFollowUsesSSE(t *testing.T) {
	t.Parallel()

	var (
		streamCalled bool
		querySeen    SessionEventQuery
	)

	deps := newTestDeps(t, stubClient{
		streamSessionFn: func(_ context.Context, id string, query SessionEventQuery, _ string, handler SSEHandler) error {
			streamCalled = true
			querySeen = query
			return handler(SSEEvent{
				ID:    "5",
				Event: session.EventTypeSessionStopped,
				Data: mustJSON(t, SessionEventRecord{
					ID:        "evt-5",
					SessionID: id,
					Sequence:  5,
					TurnID:    "turn-1",
					Type:      session.EventTypeSessionStopped,
					AgentName: "coder",
					Timestamp: fixedTestNow,
				}),
			})
		},
		sessionEventsFn: func(context.Context, string, SessionEventQuery) ([]SessionEventRecord, error) {
			t.Fatal("SessionEvents should not be called when --follow is set")
			return nil, nil
		},
	})

	stdout, _, err := executeRootCommand(t, deps, "session", "events", "sess-1", "--type", "tool_call", "--follow", "-o", "json")
	if err != nil {
		t.Fatalf("executeRootCommand() error = %v", err)
	}
	if !streamCalled {
		t.Fatal("StreamSessionEvents was not called")
	}
	if querySeen.Type != "tool_call" {
		t.Fatalf("querySeen.Type = %q, want %q", querySeen.Type, "tool_call")
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 1 {
		t.Fatalf("follow stdout lines = %d, want 1", len(lines))
	}
	var decoded SessionEventRecord
	if err := json.Unmarshal([]byte(lines[0]), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(stream line) error = %v", err)
	}
	if decoded.Type != session.EventTypeSessionStopped {
		t.Fatalf("decoded.Type = %q, want %q", decoded.Type, session.EventTypeSessionStopped)
	}
}

func TestSessionWaitReturnsImmediatelyForStoppedSession(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, stubClient{
		getSessionFn: func(context.Context, string) (SessionRecord, error) {
			return SessionRecord{
				ID:            "sess-1",
				AgentName:     "coder",
				WorkspaceID:   "ws-1",
				WorkspacePath: "/workspace/project",
				State:         string(session.StateStopped),
				CreatedAt:     fixedTestNow,
				UpdatedAt:     fixedTestNow,
			}, nil
		},
		streamSessionFn: func(context.Context, string, SessionEventQuery, string, SSEHandler) error {
			t.Fatal("StreamSessionEvents should not be called for an already stopped session")
			return nil
		},
	})

	stdout, _, err := executeRootCommand(t, deps, "session", "wait", "sess-1", "-o", "json")
	if err != nil {
		t.Fatalf("executeRootCommand() error = %v", err)
	}

	var decoded SessionRecord
	if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decoded.State != string(session.StateStopped) {
		t.Fatalf("decoded.State = %q, want %q", decoded.State, session.StateStopped)
	}
}
