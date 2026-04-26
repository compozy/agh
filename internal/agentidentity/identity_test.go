package agentidentity

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/session"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

func TestResolveValidatesAgentCallerIdentity(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)
	active := SessionSnapshot{
		ID:            "sess-1",
		Name:          "worker",
		AgentName:     "coder",
		Provider:      "test-provider",
		WorkspaceID:   "ws-1",
		WorkspacePath: "/workspace",
		State:         session.StateActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	tests := []struct {
		name              string
		credentials       Credentials
		session           SessionSnapshot
		lookupErr         error
		expectedWorkspace string
		originKind        taskpkg.OriginKind
		wantErr           error
		wantExit          int
		wantOrigin        taskpkg.OriginKind
	}{
		{
			name: "missing session id",
			credentials: Credentials{
				AgentName: "coder",
			},
			wantErr:  ErrIdentityRequired,
			wantExit: ExitIdentityRequired,
		},
		{
			name: "missing agent name",
			credentials: Credentials{
				SessionID: "sess-1",
			},
			wantErr:  ErrIdentityRequired,
			wantExit: ExitIdentityRequired,
		},
		{
			name: "unknown session",
			credentials: Credentials{
				SessionID: "missing",
				AgentName: "coder",
			},
			lookupErr: errors.New("not found"),
			wantErr:   ErrIdentityStale,
			wantExit:  ExitIdentityInvalid,
		},
		{
			name: "stopped session",
			credentials: Credentials{
				SessionID: "sess-1",
				AgentName: "coder",
			},
			session: func() SessionSnapshot {
				s := active
				s.State = session.StateStopped
				return s
			}(),
			wantErr:  ErrIdentityStale,
			wantExit: ExitIdentityInvalid,
		},
		{
			name: "agent mismatch",
			credentials: Credentials{
				SessionID: "sess-1",
				AgentName: "reviewer",
			},
			session:  active,
			wantErr:  ErrIdentityMismatch,
			wantExit: ExitIdentityInvalid,
		},
		{
			name: "workspace mismatch",
			credentials: Credentials{
				SessionID: "sess-1",
				AgentName: "coder",
			},
			session:           active,
			expectedWorkspace: "ws-2",
			wantErr:           ErrIdentityUnauthorized,
			wantExit:          ExitUnauthorized,
		},
		{
			name: "valid cli identity",
			credentials: Credentials{
				SessionID: " sess-1 ",
				AgentName: " coder ",
			},
			session:    active,
			originKind: taskpkg.OriginKindCLI,
			wantOrigin: taskpkg.OriginKindCLI,
		},
		{
			name: "valid uds identity",
			credentials: Credentials{
				SessionID:   "sess-1",
				AgentName:   "coder",
				WorkspaceID: "ws-1",
			},
			session:    active,
			originKind: taskpkg.OriginKindUDS,
			wantOrigin: taskpkg.OriginKindUDS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lookup := func(_ context.Context, sessionID string) (SessionSnapshot, error) {
				if tt.lookupErr != nil {
					return SessionSnapshot{}, tt.lookupErr
				}
				if tt.session.ID == "" {
					return active, nil
				}
				if strings.TrimSpace(sessionID) != tt.session.ID {
					t.Fatalf("lookup sessionID = %q, want %q", sessionID, tt.session.ID)
				}
				return tt.session, nil
			}

			caller, err := Resolve(context.Background(), ResolveOptions{
				Credentials:         tt.credentials,
				Lookup:              lookup,
				ExpectedWorkspaceID: tt.expectedWorkspace,
				OriginKind:          tt.originKind,
				OriginRef:           "agent.test",
			})
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Resolve() error = %v, want %v", err, tt.wantErr)
				}
				if got := ExitCodeForError(err); got != tt.wantExit {
					t.Fatalf("ExitCodeForError() = %d, want %d", got, tt.wantExit)
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if caller.Session.ID != "sess-1" || caller.Session.AgentName != "coder" {
				t.Fatalf("caller.Session = %#v, want validated session", caller.Session)
			}
			if caller.Actor.Actor.Kind != taskpkg.ActorKindAgentSession || caller.Actor.Actor.Ref != "sess-1" {
				t.Fatalf("caller.Actor.Actor = %#v, want agent session sess-1", caller.Actor.Actor)
			}
			if caller.Actor.Origin.Kind != tt.wantOrigin || caller.Actor.Origin.Ref != "agent.test" {
				t.Fatalf("caller.Actor.Origin = %#v, want %s agent.test", caller.Actor.Origin, tt.wantOrigin)
			}
		})
	}
}

func TestErrorOutputConventionsRenderStableJSONAndJSONL(t *testing.T) {
	t.Parallel()

	err := &Error{
		Code:    "identity_required",
		Message: EnvSessionID + " is required for agent commands",
		Action:  "run this command from an AGH-managed agent session",
		Err:     ErrIdentityRequired,
	}

	jsonPayload, jsonErr := MarshalErrorJSON(err)
	if jsonErr != nil {
		t.Fatalf("MarshalErrorJSON() error = %v", jsonErr)
	}
	var jsonObject struct {
		Error ErrorPayload `json:"error"`
	}
	if unmarshalErr := json.Unmarshal(jsonPayload, &jsonObject); unmarshalErr != nil {
		t.Fatalf("json.Unmarshal(JSON) error = %v", unmarshalErr)
	}
	if jsonObject.Error.Code != "identity_required" || jsonObject.Error.ExitCode != ExitIdentityRequired {
		t.Fatalf("JSON error = %#v, want stable identity error payload", jsonObject.Error)
	}

	jsonlPayload, jsonlErr := MarshalErrorJSONL(err)
	if jsonlErr != nil {
		t.Fatalf("MarshalErrorJSONL() error = %v", jsonlErr)
	}
	if strings.Contains(string(jsonlPayload), "\n") {
		t.Fatalf("JSONL payload contains embedded newline: %q", jsonlPayload)
	}
	var jsonlObject struct {
		Type  string       `json:"type"`
		Error ErrorPayload `json:"error"`
	}
	if unmarshalErr := json.Unmarshal(jsonlPayload, &jsonlObject); unmarshalErr != nil {
		t.Fatalf("json.Unmarshal(JSONL) error = %v", unmarshalErr)
	}
	if jsonlObject.Type != "error" || jsonlObject.Error.Action == "" {
		t.Fatalf("JSONL object = %#v, want error frame with actionable payload", jsonlObject)
	}
}

func TestResolveRejectsUnavailableAndMalformedLookupResults(t *testing.T) {
	t.Parallel()

	creds := Credentials{
		SessionID: "sess-1",
		AgentName: "coder",
	}

	tests := []struct {
		name    string
		ctx     context.Context
		lookup  SessionLookup
		wantErr error
	}{
		{
			name:    "nil context",
			wantErr: ErrIdentityStale,
		},
		{
			name:    "nil lookup",
			ctx:     context.Background(),
			wantErr: ErrIdentityStale,
		},
		{
			name: "empty returned session id",
			ctx:  context.Background(),
			lookup: func(_ context.Context, _ string) (SessionSnapshot, error) {
				return SessionSnapshot{
					AgentName: "coder",
					State:     session.StateActive,
				}, nil
			},
			wantErr: ErrIdentityStale,
		},
		{
			name: "different returned session id",
			ctx:  context.Background(),
			lookup: func(_ context.Context, _ string) (SessionSnapshot, error) {
				return SessionSnapshot{
					ID:        "sess-2",
					AgentName: "coder",
					State:     session.StateActive,
				}, nil
			},
			wantErr: ErrIdentityMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := Resolve(tt.ctx, ResolveOptions{
				Credentials: creds,
				Lookup:      tt.lookup,
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Resolve() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestResolveDefaultsAgentSessionOrigin(t *testing.T) {
	t.Parallel()

	caller, err := Resolve(context.Background(), ResolveOptions{
		Credentials: Credentials{
			SessionID: "sess-1",
			AgentName: "coder",
		},
		Lookup: func(_ context.Context, _ string) (SessionSnapshot, error) {
			return SessionSnapshot{
				ID:        " sess-1 ",
				AgentName: " coder ",
				State:     session.StateActive,
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if caller.Actor.Origin.Kind != taskpkg.OriginKindAgentSession || caller.Actor.Origin.Ref != "agent.session" {
		t.Fatalf("caller.Actor.Origin = %#v, want default agent_session origin", caller.Actor.Origin)
	}
}

func TestSessionSnapshotFromInfo(t *testing.T) {
	t.Parallel()

	if got := SessionSnapshotFromInfo(nil); got != (SessionSnapshot{}) {
		t.Fatalf("SessionSnapshotFromInfo(nil) = %#v, want empty snapshot", got)
	}

	now := time.Date(2026, 4, 26, 11, 0, 0, 0, time.UTC)
	info := &session.Info{
		ID:          "sess-1",
		Name:        "worker",
		AgentName:   "coder",
		Provider:    "provider",
		WorkspaceID: "ws-1",
		Workspace:   "/workspace",
		Channel:     "main",
		Type:        session.SessionTypeUser,
		State:       session.StateActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	got := SessionSnapshotFromInfo(info)
	if got.ID != info.ID || got.WorkspacePath != info.Workspace || got.State != info.State ||
		!got.CreatedAt.Equal(now) {
		t.Fatalf("SessionSnapshotFromInfo() = %#v, want fields copied from session.Info", got)
	}
}

func TestErrorPayloadFallbacksAndExitCodes(t *testing.T) {
	t.Parallel()

	if got := ExitCodeForError(nil); got != ExitOK {
		t.Fatalf("ExitCodeForError(nil) = %d, want %d", got, ExitOK)
	}

	nilPayload := ErrorPayloadFor(nil)
	if nilPayload.Code != "agent_error" || nilPayload.Message != agentCommandFailedMessage ||
		nilPayload.ExitCode != ExitOK {
		t.Fatalf("ErrorPayloadFor(nil) = %#v, want default successful payload", nilPayload)
	}

	genericPayload := ErrorPayloadFor(errors.New("daemon unavailable"))
	if genericPayload.Code != "agent_error" ||
		genericPayload.Message != "daemon unavailable" ||
		genericPayload.Action != "inspect the daemon error and retry" ||
		genericPayload.ExitCode != ExitUnavailable {
		t.Fatalf("ErrorPayloadFor(generic) = %#v, want fallback agent error payload", genericPayload)
	}

	emptyIdentityPayload := ErrorPayloadFor(&Error{Err: ErrIdentityRequired})
	if emptyIdentityPayload.Message != agentCommandFailedMessage ||
		emptyIdentityPayload.Action != "inspect the daemon error and retry" ||
		emptyIdentityPayload.ExitCode != ExitIdentityRequired {
		t.Fatalf(
			"ErrorPayloadFor(empty identity error) = %#v, want fallback text with identity exit code",
			emptyIdentityPayload,
		)
	}
}
