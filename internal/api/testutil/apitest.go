package testutil

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	core "github.com/pedronauck/agh/internal/api/core"
	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/transcript"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

var ErrStubWorkspaceServiceNotImplemented = errors.New("stub workspace service method not implemented")

type StubSessionManager struct {
	CreateFn     func(context.Context, session.CreateOpts) (*session.Session, error)
	ListFn       func() []*session.SessionInfo
	ListAllFn    func(context.Context) ([]*session.SessionInfo, error)
	StatusFn     func(context.Context, string) (*session.SessionInfo, error)
	EventsFn     func(context.Context, string, store.EventQuery) ([]store.SessionEvent, error)
	HistoryFn    func(context.Context, string, store.EventQuery) ([]store.TurnHistory, error)
	TranscriptFn func(context.Context, string) ([]transcript.Message, error)
	StopFn       func(context.Context, string) error
	ResumeFn     func(context.Context, string) (*session.Session, error)
	PromptFn     func(context.Context, string, string) (<-chan acp.AgentEvent, error)
	ApproveFn    func(context.Context, string, acp.ApproveRequest) error
}

func (s StubSessionManager) Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error) {
	if s.CreateFn != nil {
		return s.CreateFn(ctx, opts)
	}
	return nil, nil
}

func (s StubSessionManager) List() []*session.SessionInfo {
	if s.ListFn != nil {
		return s.ListFn()
	}
	if s.ListAllFn != nil {
		infos, err := s.ListAllFn(context.Background())
		if err != nil {
			return []*session.SessionInfo{}
		}
		return infos
	}
	return nil
}

func (s StubSessionManager) ListAll(ctx context.Context) ([]*session.SessionInfo, error) {
	if s.ListAllFn != nil {
		return s.ListAllFn(ctx)
	}
	return nil, nil
}

func (s StubSessionManager) Status(ctx context.Context, id string) (*session.SessionInfo, error) {
	if s.StatusFn != nil {
		return s.StatusFn(ctx, id)
	}
	return nil, session.ErrSessionNotFound
}

func (s StubSessionManager) Events(ctx context.Context, id string, query store.EventQuery) ([]store.SessionEvent, error) {
	if s.EventsFn != nil {
		return s.EventsFn(ctx, id, query)
	}
	return nil, nil
}

func (s StubSessionManager) History(ctx context.Context, id string, query store.EventQuery) ([]store.TurnHistory, error) {
	if s.HistoryFn != nil {
		return s.HistoryFn(ctx, id, query)
	}
	return nil, nil
}

func (s StubSessionManager) Transcript(ctx context.Context, id string) ([]transcript.Message, error) {
	if s.TranscriptFn != nil {
		return s.TranscriptFn(ctx, id)
	}
	return nil, nil
}

func (s StubSessionManager) Stop(ctx context.Context, id string) error {
	if s.StopFn != nil {
		return s.StopFn(ctx, id)
	}
	return nil
}

func (s StubSessionManager) Resume(ctx context.Context, id string) (*session.Session, error) {
	if s.ResumeFn != nil {
		return s.ResumeFn(ctx, id)
	}
	return nil, nil
}

func (s StubSessionManager) Prompt(ctx context.Context, id string, msg string) (<-chan acp.AgentEvent, error) {
	if s.PromptFn != nil {
		return s.PromptFn(ctx, id, msg)
	}
	ch := make(chan acp.AgentEvent)
	close(ch)
	return ch, nil
}

func (s StubSessionManager) ApprovePermission(ctx context.Context, id string, req acp.ApproveRequest) error {
	if s.ApproveFn != nil {
		return s.ApproveFn(ctx, id, req)
	}
	return nil
}

type StubObserver struct {
	QueryEventsFn      func(context.Context, store.EventSummaryQuery) ([]store.EventSummary, error)
	QueryHookCatalogFn func(context.Context, hookspkg.CatalogFilter) ([]hookspkg.CatalogEntry, error)
	QueryHookRunsFn    func(context.Context, store.HookRunQuery) ([]hookspkg.HookRunRecord, error)
	QueryHookEventsFn  func(context.Context) ([]hookspkg.EventDescriptor, error)
	HealthFn           func(context.Context) (observe.Health, error)
}

func (s StubObserver) QueryEvents(ctx context.Context, query store.EventSummaryQuery) ([]store.EventSummary, error) {
	if s.QueryEventsFn != nil {
		return s.QueryEventsFn(ctx, query)
	}
	return nil, nil
}

func (s StubObserver) Health(ctx context.Context) (observe.Health, error) {
	if s.HealthFn != nil {
		return s.HealthFn(ctx)
	}
	return observe.Health{Status: "ok"}, nil
}

func (s StubObserver) QueryHookCatalog(ctx context.Context, filter hookspkg.CatalogFilter) ([]hookspkg.CatalogEntry, error) {
	if s.QueryHookCatalogFn != nil {
		return s.QueryHookCatalogFn(ctx, filter)
	}
	return nil, nil
}

func (s StubObserver) QueryHookRuns(ctx context.Context, query store.HookRunQuery) ([]hookspkg.HookRunRecord, error) {
	if s.QueryHookRunsFn != nil {
		return s.QueryHookRunsFn(ctx, query)
	}
	return nil, nil
}

func (s StubObserver) QueryHookEvents(ctx context.Context) ([]hookspkg.EventDescriptor, error) {
	if s.QueryHookEventsFn != nil {
		return s.QueryHookEventsFn(ctx)
	}
	return nil, nil
}

type StubWorkspaceService struct {
	RegisterFn          func(context.Context, workspacepkg.RegisterOptions) (workspacepkg.Workspace, error)
	UnregisterFn        func(context.Context, string) error
	UpdateFn            func(context.Context, string, workspacepkg.UpdateOptions) error
	ListFn              func(context.Context) ([]workspacepkg.Workspace, error)
	GetFn               func(context.Context, string) (workspacepkg.Workspace, error)
	ResolveFn           func(context.Context, string) (workspacepkg.ResolvedWorkspace, error)
	ResolveOrRegisterFn func(context.Context, string) (workspacepkg.ResolvedWorkspace, error)
}

func (s StubWorkspaceService) Register(ctx context.Context, opts workspacepkg.RegisterOptions) (workspacepkg.Workspace, error) {
	if s.RegisterFn != nil {
		return s.RegisterFn(ctx, opts)
	}
	return workspacepkg.Workspace{}, ErrStubWorkspaceServiceNotImplemented
}

func (s StubWorkspaceService) Unregister(ctx context.Context, id string) error {
	if s.UnregisterFn != nil {
		return s.UnregisterFn(ctx, id)
	}
	return workspacepkg.ErrWorkspaceNotFound
}

func (s StubWorkspaceService) Update(ctx context.Context, id string, opts workspacepkg.UpdateOptions) error {
	if s.UpdateFn != nil {
		return s.UpdateFn(ctx, id, opts)
	}
	return workspacepkg.ErrWorkspaceNotFound
}

func (s StubWorkspaceService) List(ctx context.Context) ([]workspacepkg.Workspace, error) {
	if s.ListFn != nil {
		return s.ListFn(ctx)
	}
	return nil, nil
}

func (s StubWorkspaceService) Get(ctx context.Context, ref string) (workspacepkg.Workspace, error) {
	if s.GetFn != nil {
		return s.GetFn(ctx, ref)
	}
	return workspacepkg.Workspace{}, workspacepkg.ErrWorkspaceNotFound
}

func (s StubWorkspaceService) Resolve(ctx context.Context, ref string) (workspacepkg.ResolvedWorkspace, error) {
	if s.ResolveFn != nil {
		return s.ResolveFn(ctx, ref)
	}
	return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceNotFound
}

func (s StubWorkspaceService) ResolveOrRegister(ctx context.Context, path string) (workspacepkg.ResolvedWorkspace, error) {
	if s.ResolveOrRegisterFn != nil {
		return s.ResolveOrRegisterFn(ctx, path)
	}
	return workspacepkg.ResolvedWorkspace{}, ErrStubWorkspaceServiceNotImplemented
}

type StubSkillsRegistry struct {
	GetFn          func(name string) (*skills.Skill, bool)
	ListFn         func() []*skills.Skill
	ForWorkspaceFn func(ctx context.Context, resolved workspacepkg.ResolvedWorkspace) ([]*skills.Skill, error)
	LoadContentFn  func(ctx context.Context, skill *skills.Skill) (string, error)
	SetEnabledFn   func(name string, resolved *workspacepkg.ResolvedWorkspace, enabled bool) error
}

func (s StubSkillsRegistry) Get(name string) (*skills.Skill, bool) {
	if s.GetFn != nil {
		return s.GetFn(name)
	}
	return nil, false
}

func (s StubSkillsRegistry) List() []*skills.Skill {
	if s.ListFn != nil {
		return s.ListFn()
	}
	return nil
}

func (s StubSkillsRegistry) ForWorkspace(ctx context.Context, resolved workspacepkg.ResolvedWorkspace) ([]*skills.Skill, error) {
	if s.ForWorkspaceFn != nil {
		return s.ForWorkspaceFn(ctx, resolved)
	}
	return nil, nil
}

func (s StubSkillsRegistry) LoadContent(ctx context.Context, skill *skills.Skill) (string, error) {
	if s.LoadContentFn != nil {
		return s.LoadContentFn(ctx, skill)
	}
	return "", nil
}

func (s StubSkillsRegistry) SetEnabled(name string, resolved *workspacepkg.ResolvedWorkspace, enabled bool) error {
	if s.SetEnabledFn != nil {
		return s.SetEnabledFn(name, resolved, enabled)
	}
	return nil
}

type SSERecord struct {
	ID    string
	Event string
	Data  []byte
}

func NewTestHomePaths(t *testing.T) aghconfig.HomePaths {
	t.Helper()

	homePaths, err := aghconfig.ResolveHomePathsFrom(t.TempDir())
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if err := aghconfig.EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}
	return homePaths
}

func WriteAgentDef(t *testing.T, homePaths aghconfig.HomePaths, name string) {
	t.Helper()

	path := filepath.Join(homePaths.AgentsDir, name, "AGENT.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(agent dir) error = %v", err)
	}
	if err := os.WriteFile(path, []byte(`---
name: `+name+`
provider: fake
permissions: approve-reads
---

You are `+name+`.
`), 0o644); err != nil {
		t.Fatalf("os.WriteFile(AGENT.md) error = %v", err)
	}
}

func NewSessionInfo(id string) *session.SessionInfo {
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	return &session.SessionInfo{
		ID:          id,
		Name:        "demo",
		AgentName:   "coder",
		WorkspaceID: "ws-workspace",
		Workspace:   "/workspace",
		State:       session.StateActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func NewSession(id string) *session.Session {
	info := NewSessionInfo(id)
	return &session.Session{
		ID:          info.ID,
		Name:        info.Name,
		AgentName:   info.AgentName,
		WorkspaceID: info.WorkspaceID,
		Workspace:   info.Workspace,
		State:       info.State,
		CreatedAt:   info.CreatedAt,
		UpdatedAt:   info.UpdatedAt,
	}
}

func PerformRequest(t *testing.T, engine http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	return PerformRequestWithHeaders(t, engine, method, path, body, nil)
}

func PerformRequestWithHeaders(t *testing.T, engine http.Handler, method, path string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	return recorder
}

func DecodeJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder, dest any) {
	t.Helper()

	if err := json.Unmarshal(recorder.Body.Bytes(), dest); err != nil {
		t.Fatalf("json.Unmarshal(response) error = %v; body=%s", err, recorder.Body.String())
	}
}

func DecodeSSEData(t *testing.T, record SSERecord, dest any) {
	t.Helper()

	if err := json.Unmarshal(record.Data, dest); err != nil {
		t.Fatalf("json.Unmarshal(sse data) error = %v; data=%s", err, string(record.Data))
	}
}

func MustJSONBody(t *testing.T, value any) []byte {
	t.Helper()

	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return body
}

func ParseSSE(t *testing.T, body string) []SSERecord {
	t.Helper()

	scanner := bufio.NewScanner(strings.NewReader(body))
	records := make([]SSERecord, 0)
	current := SSERecord{}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			records = append(records, current)
			current = SSERecord{}
			continue
		}

		switch {
		case strings.HasPrefix(line, "id: "):
			current.ID = strings.TrimPrefix(line, "id: ")
		case strings.HasPrefix(line, "event: "):
			current.Event = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			if len(current.Data) > 0 {
				current.Data = append(current.Data, '\n')
			}
			current.Data = append(current.Data, []byte(strings.TrimPrefix(line, "data: "))...)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner.Err() = %v", err)
	}
	if current.Event != "" || current.ID != "" || len(current.Data) > 0 {
		records = append(records, current)
	}

	return records
}

func DiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

var _ core.SessionManager = StubSessionManager{}
var _ core.Observer = StubObserver{}
var _ core.WorkspaceService = StubWorkspaceService{}
