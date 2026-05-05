package core_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/testutil"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/heartbeat"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/soul"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

type soulIfMatchTestAuthoring struct {
	putCalls      int
	deleteCalls   int
	rollbackCalls int
}

func (s *soulIfMatchTestAuthoring) Validate(context.Context, soul.ValidateRequest) (soul.ValidateResult, error) {
	return soul.ValidateResult{}, nil
}

func (s *soulIfMatchTestAuthoring) Put(context.Context, soul.PutRequest) (soul.MutationResult, error) {
	s.putCalls++
	return soul.MutationResult{}, nil
}

func (s *soulIfMatchTestAuthoring) Delete(context.Context, soul.DeleteRequest) (soul.MutationResult, error) {
	s.deleteCalls++
	return soul.MutationResult{}, nil
}

func (s *soulIfMatchTestAuthoring) History(context.Context, soul.HistoryRequest) (soul.HistoryResult, error) {
	return soul.HistoryResult{}, nil
}

func (s *soulIfMatchTestAuthoring) Rollback(context.Context, soul.RollbackRequest) (soul.MutationResult, error) {
	s.rollbackCalls++
	return soul.MutationResult{}, nil
}

type soulIfMatchTestRefresher struct {
	calls int
}

func (s *soulIfMatchTestRefresher) RefreshSoulWithExpectedDigest(
	context.Context,
	string,
	string,
) (session.SoulRefreshResult, error) {
	s.calls++
	return session.SoulRefreshResult{}, nil
}

type packageOwnedHeartbeatAuthoring struct {
	putCalls      int
	deleteCalls   int
	rollbackCalls int
}

func (h *packageOwnedHeartbeatAuthoring) Validate(
	context.Context,
	heartbeat.ValidateRequest,
) (heartbeat.ValidateResult, error) {
	return heartbeat.ValidateResult{}, nil
}

func (h *packageOwnedHeartbeatAuthoring) Put(
	context.Context,
	heartbeat.PutRequest,
) (heartbeat.MutationResult, error) {
	h.putCalls++
	return heartbeat.MutationResult{}, nil
}

func (h *packageOwnedHeartbeatAuthoring) Delete(
	context.Context,
	heartbeat.DeleteRequest,
) (heartbeat.MutationResult, error) {
	h.deleteCalls++
	return heartbeat.MutationResult{}, nil
}

func (h *packageOwnedHeartbeatAuthoring) History(
	context.Context,
	heartbeat.HistoryRequest,
) (heartbeat.HistoryResult, error) {
	return heartbeat.HistoryResult{}, nil
}

func (h *packageOwnedHeartbeatAuthoring) Rollback(
	context.Context,
	heartbeat.RollbackRequest,
) (heartbeat.MutationResult, error) {
	h.rollbackCalls++
	return heartbeat.MutationResult{}, nil
}

type packageOwnedAgentCatalog struct {
	artifacts session.AgentArtifacts
}

func (c packageOwnedAgentCatalog) ListAgents(context.Context) ([]aghconfig.AgentDef, error) {
	return []aghconfig.AgentDef{aghconfig.CloneAgentDef(c.artifacts.Agent)}, nil
}

func (c packageOwnedAgentCatalog) GetAgent(context.Context, string) (aghconfig.AgentDef, error) {
	return aghconfig.CloneAgentDef(c.artifacts.Agent), nil
}

func (c packageOwnedAgentCatalog) ResolveAgentArtifacts(
	string,
	*workspacepkg.ResolvedWorkspace,
) (session.AgentArtifacts, error) {
	return c.artifacts, nil
}

func TestAuthoredContextRejectsPackageOwnedSidecarMutations(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		method        string
		path          string
		body          []byte
		registerRoute func(handlerFixture)
		assertCalls   func(*testing.T, *soulIfMatchTestAuthoring, *packageOwnedHeartbeatAuthoring)
	}{
		{
			name:   "Should reject package-owned Soul writes",
			method: http.MethodPut,
			path:   "/agents/marketer/soul",
			body:   []byte(`{"workspace_id":"ws-1","body":"new soul"}`),
			registerRoute: func(fixture handlerFixture) {
				fixture.Engine.PUT("/agents/:agent_name/soul", fixture.Handlers.PutAgentSoul)
			},
			assertCalls: func(t *testing.T, soulAuthoring *soulIfMatchTestAuthoring, _ *packageOwnedHeartbeatAuthoring) {
				t.Helper()
				if soulAuthoring.putCalls != 0 {
					t.Fatalf("soul put calls = %d, want 0", soulAuthoring.putCalls)
				}
			},
		},
		{
			name:   "Should reject package-owned Soul deletes",
			method: http.MethodDelete,
			path:   "/agents/marketer/soul",
			body:   []byte(`{"workspace_id":"ws-1"}`),
			registerRoute: func(fixture handlerFixture) {
				fixture.Engine.DELETE("/agents/:agent_name/soul", fixture.Handlers.DeleteAgentSoul)
			},
			assertCalls: func(t *testing.T, soulAuthoring *soulIfMatchTestAuthoring, _ *packageOwnedHeartbeatAuthoring) {
				t.Helper()
				if soulAuthoring.deleteCalls != 0 {
					t.Fatalf("soul delete calls = %d, want 0", soulAuthoring.deleteCalls)
				}
			},
		},
		{
			name:   "Should reject package-owned Heartbeat writes",
			method: http.MethodPut,
			path:   "/agents/marketer/heartbeat",
			body:   []byte(`{"workspace_id":"ws-1","body":"new heartbeat"}`),
			registerRoute: func(fixture handlerFixture) {
				fixture.Engine.PUT("/agents/:agent_name/heartbeat", fixture.Handlers.PutAgentHeartbeat)
			},
			assertCalls: func(t *testing.T, _ *soulIfMatchTestAuthoring, heartbeatAuthoring *packageOwnedHeartbeatAuthoring) {
				t.Helper()
				if heartbeatAuthoring.putCalls != 0 {
					t.Fatalf("heartbeat put calls = %d, want 0", heartbeatAuthoring.putCalls)
				}
			},
		},
		{
			name:   "Should reject package-owned Heartbeat rollback",
			method: http.MethodPost,
			path:   "/agents/marketer/heartbeat/rollback",
			body:   []byte(`{"workspace_id":"ws-1","revision_id":"rev-hb-1"}`),
			registerRoute: func(fixture handlerFixture) {
				fixture.Engine.POST("/agents/:agent_name/heartbeat/rollback", fixture.Handlers.RollbackAgentHeartbeat)
			},
			assertCalls: func(t *testing.T, _ *soulIfMatchTestAuthoring, heartbeatAuthoring *packageOwnedHeartbeatAuthoring) {
				t.Helper()
				if heartbeatAuthoring.rollbackCalls != 0 {
					t.Fatalf("heartbeat rollback calls = %d, want 0", heartbeatAuthoring.rollbackCalls)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			workspaceRoot := t.TempDir()
			soulAuthoring := &soulIfMatchTestAuthoring{}
			heartbeatAuthoring := &packageOwnedHeartbeatAuthoring{}
			fixture := newHandlerFixture(
				t,
				testutil.StubSessionManager{},
				testutil.StubObserver{},
				testutil.StubWorkspaceService{
					ResolveFn: func(_ context.Context, ref string) (workspacepkg.ResolvedWorkspace, error) {
						if ref != "ws-1" {
							return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceNotFound
						}
						return workspacepkg.ResolvedWorkspace{
							Workspace: workspacepkg.Workspace{ID: "ws-1", RootDir: workspaceRoot},
							Config: aghconfig.Config{
								Agents: aghconfig.AgentsConfig{
									Soul:      aghconfig.DefaultSoulConfig(),
									Heartbeat: aghconfig.DefaultHeartbeatConfig(),
								},
							},
						}, nil
					},
				},
				nil,
				nil,
			)
			fixture.Handlers.SoulAuthoring = soulAuthoring
			fixture.Handlers.HeartbeatAuthoring = heartbeatAuthoring
			fixture.Handlers.AgentCatalog = packageOwnedAgentCatalog{
				artifacts: session.AgentArtifacts{
					Agent:               aghconfig.AgentDef{Name: "marketer", Prompt: "Run marketing workflows."},
					PackageOwned:        true,
					SoulSourcePath:      ".agh/bundles/act/agents/marketer/SOUL.md",
					SoulBody:            "Lead with campaign context.",
					HeartbeatSourcePath: ".agh/bundles/act/agents/marketer/HEARTBEAT.md",
					HeartbeatBody:       "Inspect campaigns and use AGH task APIs.",
				},
			}
			tc.registerRoute(fixture)

			req := httptest.NewRequestWithContext(context.Background(), tc.method, tc.path, bytes.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			fixture.Engine.ServeHTTP(recorder, req)

			if got, want := recorder.Code, http.StatusConflict; got != want {
				t.Fatalf(
					"%s %s status = %d, want %d body=%s",
					tc.method,
					tc.path,
					got,
					want,
					recorder.Body.String(),
				)
			}
			var payload contract.ErrorPayload
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("json.Unmarshal(error payload) error = %v", err)
			}
			if !strings.Contains(payload.Error, "package-owned") {
				t.Fatalf("payload.Error = %q, want package-owned context", payload.Error)
			}
			tc.assertCalls(t, soulAuthoring, heartbeatAuthoring)
		})
	}
}

func TestSoulHandlersRejectIfMatchHeader(t *testing.T) {
	testCases := []struct {
		name          string
		method        string
		path          string
		body          []byte
		registerRoute func(fixture handlerFixture)
		wantError     string
		assertCalls   func(t *testing.T, authoring *soulIfMatchTestAuthoring, refresher *soulIfMatchTestRefresher)
	}{
		{
			name:   "Should reject If-Match on soul write",
			method: http.MethodPut,
			path:   "/agents/coder/soul",
			body:   []byte(`{"body":"# Soul"}`),
			registerRoute: func(fixture handlerFixture) {
				fixture.Engine.PUT("/agents/:agent_name/soul", fixture.Handlers.PutAgentSoul)
			},
			wantError: "authored context validation error: soul_if_match_header_unsupported: use expected_digest in request body",
			assertCalls: func(t *testing.T, authoring *soulIfMatchTestAuthoring, refresher *soulIfMatchTestRefresher) {
				t.Helper()
				if authoring.putCalls != 0 || refresher.calls != 0 {
					t.Fatalf("write calls = put:%d refresh:%d, want zero", authoring.putCalls, refresher.calls)
				}
			},
		},
		{
			name:   "Should reject If-Match on soul delete",
			method: http.MethodDelete,
			path:   "/agents/coder/soul",
			body:   []byte(`{}`),
			registerRoute: func(fixture handlerFixture) {
				fixture.Engine.DELETE("/agents/:agent_name/soul", fixture.Handlers.DeleteAgentSoul)
			},
			wantError: "authored context validation error: soul_if_match_header_unsupported: use expected_digest in request body",
			assertCalls: func(t *testing.T, authoring *soulIfMatchTestAuthoring, refresher *soulIfMatchTestRefresher) {
				t.Helper()
				if authoring.deleteCalls != 0 || refresher.calls != 0 {
					t.Fatalf("delete calls = delete:%d refresh:%d, want zero", authoring.deleteCalls, refresher.calls)
				}
			},
		},
		{
			name:   "Should reject If-Match on soul rollback",
			method: http.MethodPost,
			path:   "/agents/coder/soul/rollback",
			body:   []byte(`{"revision_id":"rev_1"}`),
			registerRoute: func(fixture handlerFixture) {
				fixture.Engine.POST("/agents/:agent_name/soul/rollback", fixture.Handlers.RollbackAgentSoul)
			},
			wantError: "authored context validation error: soul_if_match_header_unsupported: use expected_digest in request body",
			assertCalls: func(t *testing.T, authoring *soulIfMatchTestAuthoring, refresher *soulIfMatchTestRefresher) {
				t.Helper()
				if authoring.rollbackCalls != 0 || refresher.calls != 0 {
					t.Fatalf(
						"rollback calls = rollback:%d refresh:%d, want zero",
						authoring.rollbackCalls,
						refresher.calls,
					)
				}
			},
		},
		{
			name:   "Should reject If-Match on session soul refresh",
			method: http.MethodPost,
			path:   "/sessions/sess_1/soul/refresh",
			body:   []byte(`{"expected_digest":"sha256:old"}`),
			registerRoute: func(fixture handlerFixture) {
				fixture.Engine.POST("/sessions/:id/soul/refresh", fixture.Handlers.RefreshSessionSoul)
			},
			wantError: "authored context validation error: soul_if_match_header_unsupported: use expected_digest in request body",
			assertCalls: func(t *testing.T, authoring *soulIfMatchTestAuthoring, refresher *soulIfMatchTestRefresher) {
				t.Helper()
				if authoring.putCalls != 0 || authoring.deleteCalls != 0 || authoring.rollbackCalls != 0 ||
					refresher.calls != 0 {
					t.Fatalf(
						"refresh calls = put:%d delete:%d rollback:%d refresh:%d, want zero",
						authoring.putCalls,
						authoring.deleteCalls,
						authoring.rollbackCalls,
						refresher.calls,
					)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			authoring := &soulIfMatchTestAuthoring{}
			refresher := &soulIfMatchTestRefresher{}
			fixture := newHandlerFixture(
				t,
				testutil.StubSessionManager{},
				testutil.StubObserver{},
				testutil.StubWorkspaceService{},
				nil,
				nil,
			)
			fixture.Handlers.SoulAuthoring = authoring
			fixture.Handlers.SoulRefresher = refresher
			tc.registerRoute(fixture)

			req := httptest.NewRequestWithContext(
				context.Background(),
				tc.method,
				tc.path,
				bytes.NewReader(tc.body),
			)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("If-Match", `"sha256:stale"`)
			recorder := httptest.NewRecorder()
			fixture.Engine.ServeHTTP(recorder, req)

			if got, want := recorder.Code, http.StatusBadRequest; got != want {
				t.Fatalf("%s %s status = %d, want %d body=%s", tc.method, tc.path, got, want, recorder.Body.String())
			}

			var payload contract.ErrorPayload
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("json.Unmarshal(error payload) error = %v; body=%s", err, recorder.Body.String())
			}
			if got, want := payload.Error, tc.wantError; got != want {
				t.Fatalf("payload.Error = %q, want %q", got, want)
			}

			tc.assertCalls(t, authoring, refresher)
		})
	}
}
