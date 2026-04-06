package httpapi

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
)

type stubSessionManager struct {
	createFn  func(context.Context, session.CreateOpts) (*session.Session, error)
	listFn    func() []*session.SessionInfo
	listAllFn func(context.Context) ([]*session.SessionInfo, error)
	statusFn  func(context.Context, string) (*session.SessionInfo, error)
	eventsFn  func(context.Context, string, store.EventQuery) ([]store.SessionEvent, error)
	historyFn func(context.Context, string, store.EventQuery) ([]store.TurnHistory, error)
	stopFn    func(context.Context, string) error
	resumeFn  func(context.Context, string) (*session.Session, error)
	promptFn  func(context.Context, string, string) (<-chan acp.AgentEvent, error)
	approveFn func(context.Context, string, acp.ApproveRequest) error
}

func (s stubSessionManager) Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error) {
	if s.createFn != nil {
		return s.createFn(ctx, opts)
	}
	return nil, nil
}

func (s stubSessionManager) List() []*session.SessionInfo {
	if s.listFn != nil {
		return s.listFn()
	}
	if s.listAllFn != nil {
		infos, _ := s.listAllFn(context.Background())
		return infos
	}
	return nil
}

func (s stubSessionManager) ListAll(ctx context.Context) ([]*session.SessionInfo, error) {
	if s.listAllFn != nil {
		return s.listAllFn(ctx)
	}
	return nil, nil
}

func (s stubSessionManager) Status(ctx context.Context, id string) (*session.SessionInfo, error) {
	if s.statusFn != nil {
		return s.statusFn(ctx, id)
	}
	return nil, session.ErrSessionNotFound
}

func (s stubSessionManager) Events(ctx context.Context, id string, query store.EventQuery) ([]store.SessionEvent, error) {
	if s.eventsFn != nil {
		return s.eventsFn(ctx, id, query)
	}
	return nil, nil
}

func (s stubSessionManager) History(ctx context.Context, id string, query store.EventQuery) ([]store.TurnHistory, error) {
	if s.historyFn != nil {
		return s.historyFn(ctx, id, query)
	}
	return nil, nil
}

func (s stubSessionManager) Stop(ctx context.Context, id string) error {
	if s.stopFn != nil {
		return s.stopFn(ctx, id)
	}
	return nil
}

func (s stubSessionManager) Resume(ctx context.Context, id string) (*session.Session, error) {
	if s.resumeFn != nil {
		return s.resumeFn(ctx, id)
	}
	return nil, nil
}

func (s stubSessionManager) Prompt(ctx context.Context, id string, msg string) (<-chan acp.AgentEvent, error) {
	if s.promptFn != nil {
		return s.promptFn(ctx, id, msg)
	}
	ch := make(chan acp.AgentEvent)
	close(ch)
	return ch, nil
}

func (s stubSessionManager) ApprovePermission(ctx context.Context, id string, req acp.ApproveRequest) error {
	if s.approveFn != nil {
		return s.approveFn(ctx, id, req)
	}
	return nil
}

type stubObserver struct {
	queryEventsFn func(context.Context, store.EventSummaryQuery) ([]store.EventSummary, error)
	healthFn      func(context.Context) (observe.Health, error)
}

func (s stubObserver) QueryEvents(ctx context.Context, query store.EventSummaryQuery) ([]store.EventSummary, error) {
	if s.queryEventsFn != nil {
		return s.queryEventsFn(ctx, query)
	}
	return nil, nil
}

func (s stubObserver) Health(ctx context.Context) (observe.Health, error) {
	if s.healthFn != nil {
		return s.healthFn(ctx)
	}
	return observe.Health{Status: "ok"}, nil
}

type sseRecord struct {
	ID    string
	Event string
	Data  []byte
}

func newTestHandlers(t *testing.T, manager SessionManager, observer Observer, homePaths aghconfig.HomePaths) *Handlers {
	t.Helper()

	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = 2123

	return newHandlers(handlerConfig{
		sessions:     manager,
		observer:     observer,
		staticFS:     mustStaticFS(t),
		homePaths:    homePaths,
		config:       cfg,
		logger:       discardLogger(),
		startedAt:    time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		now:          func() time.Time { return time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC) },
		pollInterval: 5 * time.Millisecond,
		agentLoader:  aghconfig.LoadAgentDef,
		httpPort:     cfg.HTTP.Port,
	})
}

func newTestRouter(t *testing.T, handlers *Handlers) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(requestLoggingMiddleware(discardLogger()))
	engine.Use(corsMiddleware("127.0.0.1"))
	engine.Use(errorMiddleware())
	RegisterRoutes(engine, handlers)
	return engine
}

func mustStaticFS(t *testing.T) fs.FS {
	t.Helper()

	staticFS, err := newStaticFS()
	if err != nil {
		t.Fatalf("newStaticFS() error = %v", err)
	}

	return staticFS
}

func newTestHomePaths(t *testing.T) aghconfig.HomePaths {
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

func writeAgentDef(t *testing.T, homePaths aghconfig.HomePaths, name string) {
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

func newSessionInfo(id string) *session.SessionInfo {
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	return &session.SessionInfo{
		ID:        id,
		Name:      "demo",
		AgentName: "coder",
		Workspace: "/workspace",
		State:     session.StateActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func newSession(id string) *session.Session {
	info := newSessionInfo(id)
	return &session.Session{
		ID:        info.ID,
		Name:      info.Name,
		AgentName: info.AgentName,
		Workspace: info.Workspace,
		State:     info.State,
		CreatedAt: info.CreatedAt,
		UpdatedAt: info.UpdatedAt,
	}
}

func performRequest(t *testing.T, engine http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	return performRequestWithHeaders(t, engine, method, path, body, nil)
}

func performRequestWithHeaders(t *testing.T, engine http.Handler, method, path string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
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

func decodeJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder, dest any) {
	t.Helper()

	if err := json.Unmarshal(recorder.Body.Bytes(), dest); err != nil {
		t.Fatalf("json.Unmarshal(response) error = %v; body=%s", err, recorder.Body.String())
	}
}

func parseSSE(t *testing.T, body string) []sseRecord {
	t.Helper()

	scanner := bufio.NewScanner(strings.NewReader(body))
	records := make([]sseRecord, 0)
	current := sseRecord{}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			records = append(records, current)
			current = sseRecord{}
			continue
		}
		switch {
		case strings.HasPrefix(line, "id: "):
			current.ID = strings.TrimPrefix(line, "id: ")
		case strings.HasPrefix(line, "event: "):
			current.Event = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			current.Data = append(current.Data, []byte(strings.TrimPrefix(line, "data: "))...)
		}
	}
	if current.Event != "" || current.ID != "" || len(current.Data) > 0 {
		records = append(records, current)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner.Err() = %v", err)
	}
	return records
}

func freeTCPPort(t *testing.T) int {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen(:0) error = %v", err)
	}
	defer func() {
		_ = ln.Close()
	}()

	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener addr type = %T, want *net.TCPAddr", ln.Addr())
	}
	return tcpAddr.Port
}

func mustURL(host string, port int, path string) string {
	return fmt.Sprintf("http://%s:%d%s", host, port, path)
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
