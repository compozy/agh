package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/modelcatalog"
	"github.com/gin-gonic/gin"
)

func TestBaseHandlersModelCatalogDependency(t *testing.T) {
	t.Parallel()

	t.Run("Should carry model catalog service from config", func(t *testing.T) {
		t.Parallel()

		service := coreModelCatalogServiceStub{}
		handlers := NewBaseHandlers(&BaseHandlerConfig{ModelCatalog: service})
		if handlers.ModelCatalog == nil {
			t.Fatal("NewBaseHandlers() ModelCatalog = nil, want injected service")
		}
		if handlers.ModelCatalog != service {
			t.Fatalf("NewBaseHandlers() ModelCatalog = %#v, want %#v", handlers.ModelCatalog, service)
		}
	})
}

func TestProviderModelPayloadConversion(t *testing.T) {
	t.Parallel()

	t.Run("Should preserve nullable availability and source stale fields", func(t *testing.T) {
		t.Parallel()

		effort := modelcatalog.ReasoningEffortHigh
		model := modelcatalog.Model{
			ProviderID:             "codex",
			ModelID:                "gpt-5.4",
			DisplayName:            "GPT-5.4",
			Available:              nil,
			AvailabilityState:      modelcatalog.AvailabilityStateUnknown,
			Stale:                  true,
			RefreshedAt:            time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC),
			SupportsReasoning:      new(true),
			ReasoningEfforts:       []modelcatalog.ReasoningEffort{modelcatalog.ReasoningEffortHigh},
			DefaultReasoningEffort: &effort,
			Sources: []modelcatalog.SourceRef{
				{
					SourceID:    modelcatalog.SourceIDConfig,
					SourceKind:  modelcatalog.SourceKindConfig,
					Priority:    modelcatalog.PriorityConfig,
					RefreshedAt: time.Date(2026, 5, 7, 11, 0, 0, 0, time.UTC),
					Stale:       true,
					LastError:   "cached provider config",
				},
			},
		}

		payload := ProviderModelPayloadFromModel(model)
		if payload.Available != nil {
			t.Fatalf("Available = %#v, want nil", payload.Available)
		}
		if !payload.Stale || len(payload.Sources) != 1 || !payload.Sources[0].Stale {
			t.Fatalf("Payload = %#v, want stale model and source", payload)
		}
		if payload.DefaultReasoningEffort == nil || *payload.DefaultReasoningEffort != "high" {
			t.Fatalf("DefaultReasoningEffort = %#v, want high", payload.DefaultReasoningEffort)
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("json.Marshal(payload) error = %v", err)
		}
		if !strings.Contains(string(encoded), `"available":null`) {
			t.Fatalf("payload JSON = %s, want nullable available field", encoded)
		}
	})

	t.Run("Should redact source errors in native and OpenAI projections", func(t *testing.T) {
		t.Parallel()

		model := seedModelCatalogModel("codex", "gpt-5.4")
		model.LastError = "provider failed with api_key=sk-native-secret-token"
		model.Sources[0].LastError = "source failed with OAUTH_TOKEN=oauth-secret-token"

		nativePayload := ProviderModelPayloadFromModel(model)
		assertRedactedModelCatalogPayload(t, nativePayload.LastError, "sk-native-secret-token")
		assertRedactedModelCatalogPayload(t, nativePayload.Sources[0].LastError, "oauth-secret-token")

		openAIPayload := OpenAIModelPayloadFromModel(model)
		assertRedactedModelCatalogPayload(t, openAIPayload.AGH.LastError, "sk-native-secret-token")

		statusPayloads := SourceStatusPayloadsFromStatuses([]modelcatalog.SourceStatus{
			{
				SourceID:     modelcatalog.SourceIDModelsDev,
				SourceKind:   modelcatalog.SourceKindModelsDev,
				ProviderID:   "codex",
				RefreshState: modelcatalog.RefreshStateFailed,
				LastError:    "models.dev failed with Bearer ya29.api-secret-token",
			},
		})
		if got, want := len(statusPayloads), 1; got != want {
			t.Fatalf("len(statusPayloads) = %d, want %d", got, want)
		}
		assertRedactedModelCatalogPayload(t, statusPayloads[0].LastError, "ya29.api-secret-token")
	})
}

func TestProviderModelCatalogHandlers(t *testing.T) {
	t.Parallel()

	t.Run("Should pass list filters and return native model payload", func(t *testing.T) {
		t.Parallel()

		service := &modelCatalogServiceSpy{
			listModelsFn: func(_ context.Context, opts modelcatalog.ListOptions) ([]modelcatalog.Model, error) {
				if got, want := opts.ProviderID, "codex"; got != want {
					t.Fatalf("ProviderID = %q, want %q", got, want)
				}
				if got, want := opts.SourceID, modelcatalog.SourceIDConfig; got != want {
					t.Fatalf("SourceID = %q, want %q", got, want)
				}
				if !opts.Refresh || !opts.IncludeStale {
					t.Fatalf("ListOptions = %#v, want refresh and include_stale", opts)
				}
				return []modelcatalog.Model{seedModelCatalogModel("codex", "gpt-5.4")}, nil
			},
		}
		engine := newModelCatalogCoreEngine(t, service)

		recorder := performModelCatalogRequest(
			t,
			engine,
			http.MethodGet,
			"/model-catalog/providers/codex/models?source_id=config&refresh=true&include_stale=true",
			nil,
		)
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
		}
		var payload contract.ProviderModelListResponse
		decodeModelCatalogResponse(t, recorder, &payload)
		if len(payload.Models) != 1 || payload.Models[0].ProviderID != "codex" {
			t.Fatalf("payload = %#v, want codex model", payload)
		}
	})

	t.Run("Should return deterministic validation error for invalid provider id", func(t *testing.T) {
		t.Parallel()

		engine := newModelCatalogCoreEngine(t, &modelCatalogServiceSpy{})

		recorder := performModelCatalogRequest(
			t,
			engine,
			http.MethodGet,
			"/model-catalog/providers/bad%20id/models",
			nil,
		)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400; body=%s", recorder.Code, recorder.Body.String())
		}
		var payload contract.ErrorPayload
		decodeModelCatalogResponse(t, recorder, &payload)
		if !strings.Contains(payload.Error, "provider_id") {
			t.Fatalf("Error = %q, want provider_id validation message", payload.Error)
		}
	})

	t.Run("Should return source statuses when refresh fails", func(t *testing.T) {
		t.Parallel()

		secret := "sk-refresh-secret-token"
		service := &modelCatalogServiceSpy{
			refreshFn: func(_ context.Context, _ modelcatalog.RefreshOptions) ([]modelcatalog.SourceStatus, error) {
				return []modelcatalog.SourceStatus{
					{
						SourceID:     modelcatalog.SourceIDConfig,
						SourceKind:   modelcatalog.SourceKindConfig,
						ProviderID:   "codex",
						RefreshState: modelcatalog.RefreshStateFailed,
						LastError:    "config source failed with api_key=" + secret,
						Stale:        true,
					},
				}, fmt.Errorf("%w: api_key=%s", modelcatalog.ErrAllSourcesFailed, secret)
			},
		}
		engine := newModelCatalogCoreEngine(t, service)

		recorder := performModelCatalogRequest(
			t,
			engine,
			http.MethodPost,
			"/model-catalog/providers/codex/models/refresh",
			nil,
		)
		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503; body=%s", recorder.Code, recorder.Body.String())
		}
		var payload contract.ProviderModelRefreshResponse
		decodeModelCatalogResponse(t, recorder, &payload)
		if len(payload.Sources) != 1 || payload.Sources[0].RefreshState != string(modelcatalog.RefreshStateFailed) {
			t.Fatalf("payload = %#v, want failed source status", payload)
		}
		if payload.Error == "" {
			t.Fatalf("payload.Error = empty, want refresh error")
		}
		assertRedactedModelCatalogPayload(t, payload.Error, secret)
		assertRedactedModelCatalogPayload(t, payload.Sources[0].LastError, secret)
	})
}

func TestOpenAIModelCatalogHandler(t *testing.T) {
	t.Parallel()

	t.Run("Should use AGH metadata and provider filter", func(t *testing.T) {
		t.Parallel()

		service := &modelCatalogServiceSpy{
			listModelsFn: func(_ context.Context, opts modelcatalog.ListOptions) ([]modelcatalog.Model, error) {
				if got, want := opts.ProviderID, "codex"; got != want {
					t.Fatalf("ProviderID = %q, want %q", got, want)
				}
				return []modelcatalog.Model{seedModelCatalogModel("codex", "gpt-5.4")}, nil
			},
		}
		engine := newModelCatalogCoreEngine(t, service)

		recorder := performModelCatalogRequest(t, engine, http.MethodGet, "/openai/v1/models?provider_id=codex", nil)
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
		}
		var payload contract.OpenAIModelListResponse
		decodeModelCatalogResponse(t, recorder, &payload)
		if payload.Object != "list" || len(payload.Data) != 1 {
			t.Fatalf("payload = %#v, want one OpenAI model list item", payload)
		}
		model := payload.Data[0]
		if model.Object != "model" || model.OwnedBy != "codex" || model.AGH.ProviderID != "codex" {
			t.Fatalf("model = %#v, want OpenAI shape with agh metadata", model)
		}
		if len(model.AGH.Sources) != 1 || model.AGH.Sources[0] != modelcatalog.SourceIDConfig {
			t.Fatalf("AGH.Sources = %#v, want config source", model.AGH.Sources)
		}
	})

	t.Run("Should return OpenAI shaped validation errors", func(t *testing.T) {
		t.Parallel()

		engine := newModelCatalogCoreEngine(t, &modelCatalogServiceSpy{})

		recorder := performModelCatalogRequest(t, engine, http.MethodGet, "/openai/v1/models?provider_id=bad%20id", nil)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400; body=%s", recorder.Code, recorder.Body.String())
		}
		var payload contract.OpenAIErrorResponse
		decodeModelCatalogResponse(t, recorder, &payload)
		if payload.Error.Code != "invalid_request" || payload.Error.Type != "invalid_request_error" {
			t.Fatalf("error = %#v, want OpenAI invalid_request error", payload.Error)
		}
	})
}

type coreModelCatalogServiceStub struct{}

func (coreModelCatalogServiceStub) ListModels(
	context.Context,
	modelcatalog.ListOptions,
) ([]modelcatalog.Model, error) {
	return nil, nil
}

func (coreModelCatalogServiceStub) Refresh(
	context.Context,
	modelcatalog.RefreshOptions,
) ([]modelcatalog.SourceStatus, error) {
	return nil, nil
}

func (coreModelCatalogServiceStub) ListSourceStatus(
	context.Context,
	string,
) ([]modelcatalog.SourceStatus, error) {
	return nil, nil
}

type modelCatalogServiceSpy struct {
	listModelsFn       func(context.Context, modelcatalog.ListOptions) ([]modelcatalog.Model, error)
	refreshFn          func(context.Context, modelcatalog.RefreshOptions) ([]modelcatalog.SourceStatus, error)
	listSourceStatusFn func(context.Context, string) ([]modelcatalog.SourceStatus, error)
}

func (s *modelCatalogServiceSpy) ListModels(
	ctx context.Context,
	opts modelcatalog.ListOptions,
) ([]modelcatalog.Model, error) {
	if s.listModelsFn != nil {
		return s.listModelsFn(ctx, opts)
	}
	return nil, errors.New("unexpected ListModels call")
}

func (s *modelCatalogServiceSpy) Refresh(
	ctx context.Context,
	opts modelcatalog.RefreshOptions,
) ([]modelcatalog.SourceStatus, error) {
	if s.refreshFn != nil {
		return s.refreshFn(ctx, opts)
	}
	return nil, errors.New("unexpected Refresh call")
}

func (s *modelCatalogServiceSpy) ListSourceStatus(
	ctx context.Context,
	providerID string,
) ([]modelcatalog.SourceStatus, error) {
	if s.listSourceStatusFn != nil {
		return s.listSourceStatusFn(ctx, providerID)
	}
	return nil, errors.New("unexpected ListSourceStatus call")
}

func newModelCatalogCoreEngine(t *testing.T, service ModelCatalogService) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	handlers := NewBaseHandlers(&BaseHandlerConfig{
		ModelCatalog: service,
		Now: func() time.Time {
			return time.Date(2026, 5, 7, 12, 30, 0, 0, time.UTC)
		},
	})
	engine := gin.New()
	engine.GET("/model-catalog/*catalog_path", handlers.ModelCatalogRoute)
	engine.POST("/model-catalog/*catalog_path", handlers.ModelCatalogRoute)
	engine.GET("/openai/v1/models", handlers.OpenAIModels)
	return engine
}

func performModelCatalogRequest(
	t *testing.T,
	engine http.Handler,
	method string,
	path string,
	body []byte,
) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader(string(body)))
	engine.ServeHTTP(recorder, req)
	return recorder
}

func decodeModelCatalogResponse(t *testing.T, recorder *httptest.ResponseRecorder, dest any) {
	t.Helper()

	if err := json.Unmarshal(recorder.Body.Bytes(), dest); err != nil {
		t.Fatalf("json.Unmarshal(response) error = %v; body=%s", err, recorder.Body.String())
	}
}

func seedModelCatalogModel(providerID string, modelID string) modelcatalog.Model {
	available := true
	return modelcatalog.Model{
		ProviderID:        providerID,
		ModelID:           modelID,
		DisplayName:       "GPT-5.4",
		Available:         &available,
		AvailabilityState: modelcatalog.AvailabilityStateAvailableLive,
		Sources: []modelcatalog.SourceRef{
			{
				SourceID:   modelcatalog.SourceIDConfig,
				SourceKind: modelcatalog.SourceKindConfig,
				Priority:   modelcatalog.PriorityConfig,
			},
		},
	}
}

func assertRedactedModelCatalogPayload(t *testing.T, value string, secret string) {
	t.Helper()

	if strings.Contains(value, secret) {
		t.Fatalf("payload value = %q, want secret redacted", value)
	}
	if !strings.Contains(value, "[REDACTED]") {
		t.Fatalf("payload value = %q, want redaction marker", value)
	}
}
