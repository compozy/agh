package udsapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/agentidentity"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/session"
)

func TestAgentMeRejectsInvalidCallerIdentity(t *testing.T) {
	t.Parallel()

	active := &session.Info{
		ID:          "sess-1",
		AgentName:   "coder",
		Provider:    "test-provider",
		WorkspaceID: "ws-1",
		Workspace:   "/workspace",
		State:       session.StateActive,
		CreatedAt:   time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 26, 10, 0, 1, 0, time.UTC),
	}

	tests := []struct {
		name       string
		headers    map[string]string
		statusInfo *session.Info
		wantStatus int
	}{
		{
			name:       "missing env headers",
			headers:    map[string]string{},
			statusInfo: active,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "stopped session",
			headers: map[string]string{
				agentidentity.HeaderSessionID: "sess-1",
				agentidentity.HeaderAgent:     "coder",
			},
			statusInfo: func() *session.Info {
				info := *active
				info.State = session.StateStopped
				return &info
			}(),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "agent mismatch",
			headers: map[string]string{
				agentidentity.HeaderSessionID: "sess-1",
				agentidentity.HeaderAgent:     "reviewer",
			},
			statusInfo: active,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "workspace mismatch",
			headers: map[string]string{
				agentidentity.HeaderSessionID:   "sess-1",
				agentidentity.HeaderAgent:       "coder",
				agentidentity.HeaderWorkspaceID: "ws-2",
			},
			statusInfo: active,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			manager := stubSessionManager{
				StatusFn: func(_ context.Context, id string) (*session.Info, error) {
					if tt.statusInfo == nil {
						return nil, session.ErrSessionNotFound
					}
					if id != tt.statusInfo.ID {
						return nil, session.ErrSessionNotFound
					}
					return tt.statusInfo, nil
				},
			}
			engine := newTestRouter(t, newTestHandlers(t, manager, stubObserver{}, newTestHomePaths(t)))
			recorder := performAgentMeRequest(t, engine, tt.headers)
			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
			var payload contract.ErrorPayload
			decodeJSONResponse(t, recorder, &payload)
			if payload.Error == "" {
				t.Fatalf("error payload = %#v, want actionable error", payload)
			}
		})
	}
}

func TestAgentMeReturnsValidatedCallerIdentity(t *testing.T) {
	t.Parallel()

	manager := stubSessionManager{
		StatusFn: func(_ context.Context, id string) (*session.Info, error) {
			if id != "sess-1" {
				return nil, session.ErrSessionNotFound
			}
			now := time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)
			return &session.Info{
				ID:          "sess-1",
				Name:        "worker",
				AgentName:   "coder",
				Provider:    "test-provider",
				WorkspaceID: "ws-1",
				Workspace:   "/workspace",
				Channel:     "coord",
				Type:        session.SessionTypeUser,
				State:       session.StateActive,
				CreatedAt:   now,
				UpdatedAt:   now,
			}, nil
		},
	}
	engine := newTestRouter(t, newTestHandlers(t, manager, stubObserver{}, newTestHomePaths(t)))
	recorder := performAgentMeRequest(t, engine, map[string]string{
		agentidentity.HeaderSessionID:   "sess-1",
		agentidentity.HeaderAgent:       "coder",
		agentidentity.HeaderWorkspaceID: "ws-1",
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response contract.AgentMeResponse
	decodeJSONResponse(t, recorder, &response)
	if response.Me.Self.SessionID != "sess-1" || response.Me.Self.AgentName != "coder" {
		t.Fatalf("response.Me.Self = %#v, want validated caller", response.Me.Self)
	}
	if response.Me.Session.State != session.StateActive || response.Me.Workspace.ID != "ws-1" {
		encoded, _ := json.Marshal(response.Me)
		t.Fatalf("response.Me = %s, want active session in workspace ws-1", encoded)
	}
}

func performAgentMeRequest(t *testing.T, engine http.Handler, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/agent/me", http.NoBody)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	return recorder
}
