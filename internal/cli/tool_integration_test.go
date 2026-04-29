//go:build integration

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	core "github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/api/testutil"
	"github.com/pedronauck/agh/internal/api/udsapi"
	aghconfig "github.com/pedronauck/agh/internal/config"
	toolspkg "github.com/pedronauck/agh/internal/tools"
)

func TestCLIToolCommandsMatchUDSContractsIntegration(t *testing.T) {
	t.Parallel()

	homePaths := testutil.NewTestHomePaths(t)
	cfg := testutil.ConfigWithDisabledNetwork(homePaths)
	cfg.Daemon.Socket = shortSocketPath(t)
	registry := newCLIToolIntegrationRegistry()
	server, err := udsapi.New(
		udsapi.WithHomePaths(homePaths),
		udsapi.WithConfig(&cfg),
		udsapi.WithSocketPath(cfg.Daemon.Socket),
		udsapi.WithLogger(discardLogger()),
		udsapi.WithSessionManager(testutil.StubSessionManager{}),
		udsapi.WithTaskService(testutil.StubTaskManager{}),
		udsapi.WithObserver(testutil.StubObserver{}),
		udsapi.WithWorkspaceResolver(testutil.StubWorkspaceService{}),
		udsapi.WithToolRegistry(registry),
		udsapi.WithToolsetRegistry(registry),
	)
	if err != nil {
		t.Fatalf("udsapi.New() error = %v", err)
	}
	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("server.Start() error = %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	})

	direct, err := NewClient(cfg.Daemon.Socket)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	deps := toolIntegrationDeps(t, homePaths, cfg)

	t.Run("Should match tool list payload", func(t *testing.T) {
		directPayload, err := direct.ListTools(context.Background(), ToolQuery{WorkspaceID: "ws-1"})
		if err != nil {
			t.Fatalf("ListTools() error = %v", err)
		}
		stdout, _, err := executeRootCommand(
			t,
			deps,
			"tool",
			"list",
			"--workspace",
			"ws-1",
			"-o",
			"json",
		)
		if err != nil {
			t.Fatalf("tool list error = %v", err)
		}
		var cliPayload ToolsResponseRecord
		decodeJSONOutput(t, stdout, &cliPayload)
		assertContractJSONEqual(t, cliPayload, directPayload)
	})

	t.Run("Should match tool search payload", func(t *testing.T) {
		request := ToolSearchRequest{Query: "skill", Limit: 1, WorkspaceID: "ws-1"}
		directPayload, err := direct.SearchTools(context.Background(), request)
		if err != nil {
			t.Fatalf("SearchTools() error = %v", err)
		}
		stdout, _, err := executeRootCommand(
			t,
			deps,
			"tool",
			"search",
			"skill",
			"--limit",
			"1",
			"--workspace",
			"ws-1",
			"-o",
			"json",
		)
		if err != nil {
			t.Fatalf("tool search error = %v", err)
		}
		var cliPayload ToolsResponseRecord
		decodeJSONOutput(t, stdout, &cliPayload)
		assertContractJSONEqual(t, cliPayload, directPayload)
	})

	t.Run("Should match tool invoke payload", func(t *testing.T) {
		request := ToolInvokeRequest{SessionID: "sess-1", Input: json.RawMessage(`{"message":"hello"}`)}
		directPayload, err := direct.InvokeTool(context.Background(), toolspkg.ToolIDSkillView.String(), request)
		if err != nil {
			t.Fatalf("InvokeTool() error = %v", err)
		}
		stdout, _, err := executeRootCommand(
			t,
			deps,
			"tool",
			"invoke",
			toolspkg.ToolIDSkillView.String(),
			"--session",
			"sess-1",
			"--input",
			`{"message":"hello"}`,
			"-o",
			"json",
		)
		if err != nil {
			t.Fatalf("tool invoke error = %v", err)
		}
		var cliPayload ToolInvokeResponseRecord
		decodeJSONOutput(t, stdout, &cliPayload)
		assertContractJSONEqual(t, cliPayload, directPayload)
	})

	t.Run("Should match toolset info payload", func(t *testing.T) {
		directPayload, err := direct.GetToolset(context.Background(), toolspkg.ToolsetIDCatalog.String(), ToolQuery{})
		if err != nil {
			t.Fatalf("GetToolset() error = %v", err)
		}
		stdout, _, err := executeRootCommand(
			t,
			deps,
			"toolsets",
			"info",
			toolspkg.ToolsetIDCatalog.String(),
			"-o",
			"json",
		)
		if err != nil {
			t.Fatalf("toolsets info error = %v", err)
		}
		var cliPayload ToolsetResponseRecord
		decodeJSONOutput(t, stdout, &cliPayload)
		assertContractJSONEqual(t, cliPayload, directPayload)
	})
}

func assertContractJSONEqual(t *testing.T, got any, want any) {
	t.Helper()

	var gotValue any
	if err := json.Unmarshal(mustJSON(t, got), &gotValue); err != nil {
		t.Fatalf("json.Unmarshal(got) error = %v", err)
	}
	var wantValue any
	if err := json.Unmarshal(mustJSON(t, want), &wantValue); err != nil {
		t.Fatalf("json.Unmarshal(want) error = %v", err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("contract JSON = %#v, want %#v", gotValue, wantValue)
	}
}

func toolIntegrationDeps(
	t *testing.T,
	homePaths aghconfig.HomePaths,
	cfg aghconfig.Config,
) commandDeps {
	t.Helper()

	return commandDeps{
		loadConfig: func() (aghconfig.Config, error) {
			return cfg, nil
		},
		resolveHome: func() (aghconfig.HomePaths, error) {
			return homePaths, nil
		},
		ensureHome: func(aghconfig.HomePaths) error { return nil },
		newClient:  NewClient,
		getwd: func() (string, error) {
			return "/workspace/project", nil
		},
		getenv: func(string) string { return "" },
		now: func() time.Time {
			return time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
		},
	}
}

type cliToolIntegrationRegistry struct {
	mu    sync.Mutex
	views []toolspkg.ToolView
}

var _ core.ToolRegistry = (*cliToolIntegrationRegistry)(nil)
var _ core.ToolsetRegistry = (*cliToolIntegrationRegistry)(nil)

func newCLIToolIntegrationRegistry() *cliToolIntegrationRegistry {
	return &cliToolIntegrationRegistry{views: []toolspkg.ToolView{
		cliToolIntegrationView(toolspkg.ToolIDSkillView, true),
		cliToolIntegrationView("agh__operator_diag", false),
	}}
}

func (r *cliToolIntegrationRegistry) List(
	_ context.Context,
	scope toolspkg.Scope,
) ([]toolspkg.ToolView, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	views := make([]toolspkg.ToolView, 0, len(r.views))
	for i := range r.views {
		view := r.views[i]
		if scope.Operator || (view.Decision.VisibleToSession && view.Decision.Callable) {
			views = append(views, view)
		}
	}
	return views, nil
}

func (r *cliToolIntegrationRegistry) Search(
	ctx context.Context,
	scope toolspkg.Scope,
	query toolspkg.SearchQuery,
) ([]toolspkg.ToolView, error) {
	views, err := r.List(ctx, scope)
	if err != nil {
		return nil, err
	}
	needle := strings.TrimSpace(strings.ToLower(query.Query))
	if needle == "" {
		return limitCLIToolIntegrationViews(views, query.Limit), nil
	}
	filtered := make([]toolspkg.ToolView, 0, len(views))
	for i := range views {
		view := views[i]
		if strings.Contains(strings.ToLower(view.Descriptor.ID.String()+" "+view.Descriptor.Description), needle) {
			filtered = append(filtered, view)
		}
	}
	return limitCLIToolIntegrationViews(filtered, query.Limit), nil
}

func (r *cliToolIntegrationRegistry) Get(
	ctx context.Context,
	scope toolspkg.Scope,
	id toolspkg.ToolID,
) (toolspkg.ToolView, error) {
	if err := id.Validate(); err != nil {
		return toolspkg.ToolView{}, err
	}
	views, err := r.List(ctx, scope)
	if err != nil {
		return toolspkg.ToolView{}, err
	}
	for i := range views {
		if views[i].Descriptor.ID == id {
			return views[i], nil
		}
	}
	return toolspkg.ToolView{}, toolspkg.NewToolError(
		toolspkg.ErrorCodeNotFound,
		id,
		fmt.Sprintf("tool %q not found", id),
		toolspkg.ErrToolNotFound,
		toolspkg.ReasonToolUnknown,
	)
}

func (r *cliToolIntegrationRegistry) Call(
	ctx context.Context,
	scope toolspkg.Scope,
	request toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	if _, err := r.Get(ctx, scope, request.ToolID); err != nil {
		return toolspkg.ToolResult{}, err
	}
	return toolspkg.ToolResult{
		Content:    []toolspkg.ToolContent{{Type: "text", Text: "ok"}},
		Structured: json.RawMessage(`{"ok":true}`),
		Preview:    "ok",
		DurationMS: 5,
	}, nil
}

func (r *cliToolIntegrationRegistry) ListToolsets(
	context.Context,
	toolspkg.Scope,
) ([]toolspkg.ToolsetView, error) {
	return []toolspkg.ToolsetView{cliToolIntegrationToolset()}, nil
}

func (r *cliToolIntegrationRegistry) GetToolset(
	_ context.Context,
	_ toolspkg.Scope,
	id toolspkg.ToolsetID,
) (toolspkg.ToolsetView, error) {
	if id == toolspkg.ToolsetIDCatalog {
		return cliToolIntegrationToolset(), nil
	}
	return toolspkg.ToolsetView{}, toolspkg.NewToolError(
		toolspkg.ErrorCodeNotFound,
		toolspkg.ToolID(id),
		fmt.Sprintf("toolset %q not found", id),
		toolspkg.ErrToolNotFound,
		toolspkg.ReasonToolsetUnknown,
	)
}

func cliToolIntegrationView(id toolspkg.ToolID, callable bool) toolspkg.ToolView {
	visibility := toolspkg.VisibilityModel
	if !callable {
		visibility = toolspkg.VisibilityOperator
	}
	return toolspkg.ToolView{
		Descriptor: toolspkg.Descriptor{
			ID:          id,
			Backend:     toolspkg.BackendRef{Kind: toolspkg.BackendNativeGo, NativeName: id.String()},
			Description: "Skill registry integration tool",
			InputSchema: json.RawMessage(`{"type":"object"}`),
			Source:      toolspkg.SourceRef{Kind: toolspkg.SourceBuiltin, Owner: toolspkg.BuiltinSourceOwner},
			Visibility:  visibility,
			Risk:        toolspkg.RiskRead,
			ReadOnly:    true,
			Toolsets:    []toolspkg.ToolsetID{toolspkg.ToolsetIDCatalog},
		},
		Availability: toolspkg.Availability{
			Registered: true,
			Enabled:    true,
			Available:  true,
			Authorized: true,
			Executable: callable,
		},
		Decision: toolspkg.EffectiveToolDecision{
			VisibleToOperator: true,
			VisibleToSession:  callable,
			Callable:          callable,
		},
	}
}

func cliToolIntegrationToolset() toolspkg.ToolsetView {
	return toolspkg.ToolsetView{
		Toolset:       toolspkg.Toolset{ID: toolspkg.ToolsetIDCatalog, Tools: []string{"agh__skill_view"}},
		ExpandedTools: []toolspkg.ToolID{toolspkg.ToolIDSkillView},
	}
}

func limitCLIToolIntegrationViews(views []toolspkg.ToolView, limit int) []toolspkg.ToolView {
	if limit <= 0 || limit >= len(views) {
		return views
	}
	return views[:limit]
}
