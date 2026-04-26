package cli

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pedronauck/agh/internal/agentidentity"
	"github.com/pedronauck/agh/internal/session"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

func TestResolveAgentCallerFromEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "Should use daemon session lookup",
			run: func(t *testing.T) {
				t.Helper()

				deps := newTestDeps(t, &stubClient{})
				deps.getenv = func(key string) string {
					switch key {
					case agentidentity.EnvSessionID:
						return "sess-1"
					case agentidentity.EnvAgent:
						return "coder"
					default:
						return ""
					}
				}

				client := &stubClient{
					getSessionFn: func(_ context.Context, id string) (SessionRecord, error) {
						if id != "sess-1" {
							t.Fatalf("GetSession() id = %q, want sess-1", id)
						}
						return SessionRecord{
							ID:          "sess-1",
							AgentName:   "coder",
							WorkspaceID: "ws-1",
							State:       session.StateActive,
						}, nil
					},
				}

				caller, err := resolveAgentCallerFromEnv(context.Background(), deps, client, "ws-1", "agent.cli.test")
				if err != nil {
					t.Fatalf("resolveAgentCallerFromEnv() error = %v", err)
				}
				if caller.Actor.Actor.Kind != taskpkg.ActorKindAgentSession ||
					caller.Actor.Origin.Kind != taskpkg.OriginKindCLI {
					t.Fatalf("caller.Actor = %#v, want agent-session actor with CLI origin", caller.Actor)
				}
			},
		},
		{
			name: "Should reject missing identity before lookup",
			run: func(t *testing.T) {
				t.Helper()

				deps := newTestDeps(t, &stubClient{})
				client := &stubClient{
					getSessionFn: func(context.Context, string) (SessionRecord, error) {
						t.Fatal("GetSession() should not be called when env identity is missing")
						return SessionRecord{}, errors.New("unexpected")
					},
				}

				_, err := resolveAgentCallerFromEnv(context.Background(), deps, client, "", "agent.cli.test")
				if !errors.Is(err, agentidentity.ErrIdentityRequired) {
					t.Fatalf("resolveAgentCallerFromEnv() error = %v, want ErrIdentityRequired", err)
				}
				if got := cliExitCodeForError(err); got != agentidentity.ExitIdentityRequired {
					t.Fatalf("cliExitCodeForError() = %d, want %d", got, agentidentity.ExitIdentityRequired)
				}
			},
		},
		{
			name: "Should wrap daemon session lookup errors with session id",
			run: func(t *testing.T) {
				t.Helper()

				deps := newTestDeps(t, &stubClient{})
				deps.getenv = func(key string) string {
					switch key {
					case agentidentity.EnvSessionID:
						return "sess-missing"
					case agentidentity.EnvAgent:
						return "coder"
					default:
						return ""
					}
				}
				client := &stubClient{
					getSessionFn: func(context.Context, string) (SessionRecord, error) {
						return SessionRecord{}, session.ErrSessionNotFound
					},
				}

				lookup := agentSessionLookup(client)
				_, err := lookup(context.Background(), "sess-missing")
				if !errors.Is(err, session.ErrSessionNotFound) {
					t.Fatalf("agentSessionLookup() error = %v, want ErrSessionNotFound", err)
				}
				if !strings.Contains(err.Error(), "sess-missing") {
					t.Fatalf("agentSessionLookup() error = %v, want session id context", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}
