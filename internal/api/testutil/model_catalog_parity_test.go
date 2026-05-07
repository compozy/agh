package testutil_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/httpapi"
	"github.com/pedronauck/agh/internal/api/testutil"
	"github.com/pedronauck/agh/internal/api/udsapi"
	"github.com/pedronauck/agh/internal/cli"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/modelcatalog"
)

func TestModelCatalogTransportParity(t *testing.T) {
	t.Parallel()

	t.Run("Should return canonical native list JSON bytes for the same catalog state", func(t *testing.T) {
		t.Parallel()

		service := &parityModelCatalogService{
			models: []modelcatalog.Model{parityCatalogModel("codex", "gpt-5.4")},
		}
		httpEngine := newParityHTTPRouter(t, service)
		udsEngine := newParityUDSRouter(t, service)

		httpResp := performParityRequest(t, httpEngine, http.MethodGet, "/api/providers/codex/models")
		udsResp := performParityRequest(t, udsEngine, http.MethodGet, "/api/providers/codex/models")
		if httpResp.Code != http.StatusOK || udsResp.Code != http.StatusOK {
			t.Fatalf(
				"statuses = http:%d uds:%d, want 200; http=%s uds=%s",
				httpResp.Code,
				udsResp.Code,
				httpResp.Body.String(),
				udsResp.Body.String(),
			)
		}
		if got, want := httpResp.Body.String(), udsResp.Body.String(); got != want {
			t.Fatalf("HTTP body = %s, want UDS body %s", got, want)
		}
		var cliRecord cli.ProviderModelListRecord
		if err := json.Unmarshal(httpResp.Body.Bytes(), &cliRecord); err != nil {
			t.Fatalf("json.Unmarshal(HTTP body as CLI record) error = %v", err)
		}
		cliJSON, err := json.Marshal(cliRecord)
		if err != nil {
			t.Fatalf("json.Marshal(CLI record) error = %v", err)
		}
		if got, want := string(cliJSON), httpResp.Body.String(); got != want {
			t.Fatalf("CLI JSON = %s, want canonical native body %s", got, want)
		}

		openAIResp := performParityRequest(
			t,
			httpEngine,
			http.MethodGet,
			"/api/openai/v1/models?provider_id=codex",
		)
		if openAIResp.Code != http.StatusOK {
			t.Fatalf("OpenAI status = %d, want 200; body=%s", openAIResp.Code, openAIResp.Body.String())
		}
		var openAIPayload contract.OpenAIModelListResponse
		if err := json.Unmarshal(openAIResp.Body.Bytes(), &openAIPayload); err != nil {
			t.Fatalf("json.Unmarshal(OpenAI body) error = %v", err)
		}
		if len(openAIPayload.Data) != 1 {
			t.Fatalf("OpenAI data = %#v, want one model", openAIPayload.Data)
		}
		openAIModel := openAIPayload.Data[0]
		nativeModel := cliRecord.Models[0]
		if openAIModel.ID != nativeModel.ModelID ||
			openAIModel.OwnedBy != nativeModel.ProviderID ||
			openAIModel.AGH.ProviderID != nativeModel.ProviderID ||
			openAIModel.AGH.ModelID != nativeModel.ModelID {
			t.Fatalf("OpenAI model = %#v, want native catalog identity %#v", openAIModel, nativeModel)
		}
	})
}

func newParityHTTPRouter(t *testing.T, service *parityModelCatalogService) http.Handler {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	homePaths := newShortParityHomePaths(t)
	cfg := testutil.ConfigWithDisabledNetwork(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = 2123
	if _, err := httpapi.New(
		httpapi.WithEngine(engine),
		httpapi.WithHomePaths(homePaths),
		httpapi.WithConfig(&cfg),
		httpapi.WithHost(cfg.HTTP.Host),
		httpapi.WithPort(cfg.HTTP.Port),
		httpapi.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		httpapi.WithStartedAt(time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)),
		httpapi.WithNow(func() time.Time { return time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC) }),
		httpapi.WithSessionManager(testutil.StubSessionManager{}),
		httpapi.WithTaskService(testutil.StubTaskManager{}),
		httpapi.WithObserver(testutil.StubObserver{}),
		httpapi.WithWorkspaceResolver(testutil.StubWorkspaceService{}),
		httpapi.WithModelCatalogService(service),
	); err != nil {
		t.Fatalf("httpapi.New() error = %v", err)
	}
	return engine
}

func newParityUDSRouter(t *testing.T, service *parityModelCatalogService) http.Handler {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	homePaths := newShortParityHomePaths(t)
	cfg := testutil.ConfigWithDisabledNetwork(homePaths)
	if _, err := udsapi.New(
		udsapi.WithEngine(engine),
		udsapi.WithHomePaths(homePaths),
		udsapi.WithConfig(&cfg),
		udsapi.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		udsapi.WithStartedAt(time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)),
		udsapi.WithNow(func() time.Time { return time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC) }),
		udsapi.WithSessionManager(testutil.StubSessionManager{}),
		udsapi.WithTaskService(testutil.StubTaskManager{}),
		udsapi.WithObserver(testutil.StubObserver{}),
		udsapi.WithWorkspaceResolver(testutil.StubWorkspaceService{}),
		udsapi.WithModelCatalogService(service),
	); err != nil {
		t.Fatalf("udsapi.New() error = %v", err)
	}
	return engine
}

func performParityRequest(t *testing.T, handler http.Handler, method string, path string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(context.Background(), method, path, http.NoBody)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func newShortParityHomePaths(t *testing.T) aghconfig.HomePaths {
	t.Helper()

	root, err := os.MkdirTemp(".", ".agh-model-parity-*")
	if err != nil {
		t.Fatalf("MkdirTemp() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(root); err != nil {
			t.Errorf("RemoveAll(%q) error = %v", root, err)
		}
	})
	homePaths, err := aghconfig.ResolveHomePathsFrom(root)
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	return homePaths
}

type parityModelCatalogService struct {
	models []modelcatalog.Model
}

func (s *parityModelCatalogService) ListModels(
	context.Context,
	modelcatalog.ListOptions,
) ([]modelcatalog.Model, error) {
	return append([]modelcatalog.Model(nil), s.models...), nil
}

func (*parityModelCatalogService) Refresh(
	context.Context,
	modelcatalog.RefreshOptions,
) ([]modelcatalog.SourceStatus, error) {
	return nil, nil
}

func (*parityModelCatalogService) ListSourceStatus(
	context.Context,
	string,
) ([]modelcatalog.SourceStatus, error) {
	return nil, nil
}

func parityCatalogModel(providerID string, modelID string) modelcatalog.Model {
	available := true
	return modelcatalog.Model{
		ProviderID:        providerID,
		ModelID:           modelID,
		Available:         &available,
		AvailabilityState: modelcatalog.AvailabilityStateAvailableLive,
		Sources: []modelcatalog.SourceRef{
			{SourceID: modelcatalog.SourceIDConfig, SourceKind: modelcatalog.SourceKindConfig},
		},
	}
}
