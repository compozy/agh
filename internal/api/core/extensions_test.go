package core_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	core "github.com/compozy/agh/internal/api/core"
	"github.com/compozy/agh/internal/api/testutil"
	extensionpkg "github.com/compozy/agh/internal/extension"
	taskpkg "github.com/compozy/agh/internal/task"
	"github.com/gin-gonic/gin"
)

type extensionServiceStub struct {
	listFn   func(context.Context) ([]contract.ExtensionPayload, error)
	searchFn func(context.Context, string, string, int) ([]contract.ExtensionMarketplaceEntry, error)
}

func (s extensionServiceStub) List(ctx context.Context) ([]contract.ExtensionPayload, error) {
	if s.listFn != nil {
		return s.listFn(ctx)
	}
	return nil, nil
}

func (s extensionServiceStub) SearchMarketplace(
	ctx context.Context,
	query string,
	source string,
	limit int,
) ([]contract.ExtensionMarketplaceEntry, error) {
	if s.searchFn != nil {
		return s.searchFn(ctx, query, source, limit)
	}
	return nil, nil
}

func (extensionServiceStub) Install(
	context.Context,
	contract.InstallExtensionRequest,
	taskpkg.ActorContext,
) (contract.ExtensionPayload, error) {
	return contract.ExtensionPayload{}, nil
}

func (extensionServiceStub) Update(
	context.Context,
	string,
	contract.UpdateExtensionRequest,
	taskpkg.ActorContext,
) (contract.ManagedExtensionUpdatePayload, error) {
	return contract.ManagedExtensionUpdatePayload{}, nil
}

func (extensionServiceStub) Remove(
	context.Context,
	string,
	taskpkg.ActorContext,
) (contract.ManagedExtensionRemovePayload, error) {
	return contract.ManagedExtensionRemovePayload{}, nil
}

func (extensionServiceStub) Enable(context.Context, string, taskpkg.ActorContext) (contract.ExtensionPayload, error) {
	return contract.ExtensionPayload{}, nil
}

func (extensionServiceStub) Disable(context.Context, string, taskpkg.ActorContext) (contract.ExtensionPayload, error) {
	return contract.ExtensionPayload{}, nil
}

func (extensionServiceStub) Status(context.Context, string) (contract.ExtensionPayload, error) {
	return contract.ExtensionPayload{}, nil
}

func (extensionServiceStub) Provenance(context.Context, string) (contract.ExtensionProvenancePayload, error) {
	return contract.ExtensionProvenancePayload{}, nil
}

func TestListExtensionsRespectsMaskInternalErrors(t *testing.T) {
	t.Parallel()

	t.Run("Should mask internal extension errors when handler masking is enabled", func(t *testing.T) {
		// not parallel: gin.SetMode mutates process-global state.
		gin.SetMode(gin.TestMode)

		homePaths := testutil.NewTestHomePaths(t)
		cfg := testConfigWithDisabledNetwork(homePaths)
		handlers := core.NewBaseHandlers(&core.BaseHandlerConfig{
			TransportName:      "api-core-test",
			MaskInternalErrors: true,
			Extensions: extensionServiceStub{
				listFn: func(context.Context) ([]contract.ExtensionPayload, error) {
					return nil, errors.New("extension registry token=super-secret failed")
				},
			},
			HomePaths: homePaths,
			Config:    cfg,
			Logger:    testutil.DiscardLogger(),
			StartedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
			Now: func() time.Time {
				return time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC)
			},
			HTTPPort: cfg.HTTP.Port,
		})

		engine := gin.New()
		engine.Use(gin.Recovery())
		engine.GET("/extensions", handlers.ListExtensions)

		response := performRequest(t, engine, http.MethodGet, "/extensions", nil)
		if response.Code != http.StatusInternalServerError {
			t.Fatalf(
				"status = %d, want %d; body=%s",
				response.Code,
				http.StatusInternalServerError,
				response.Body.String(),
			)
		}
		if strings.Contains(response.Body.String(), "super-secret") {
			t.Fatalf("response body leaked internal error detail: %s", response.Body.String())
		}
	})
}

func TestSearchExtensionMarketplaceMapsUnavailableSourceToServiceUnavailable(t *testing.T) {
	t.Parallel()

	t.Run("Should map unavailable marketplace source to service unavailable", func(t *testing.T) {
		t.Parallel()

		homePaths := testutil.NewTestHomePaths(t)
		cfg := testConfigWithDisabledNetwork(homePaths)
		handlers := core.NewBaseHandlers(&core.BaseHandlerConfig{
			TransportName:      "api-core-test",
			MaskInternalErrors: true,
			Extensions: extensionServiceStub{
				searchFn: func(context.Context, string, string, int) ([]contract.ExtensionMarketplaceEntry, error) {
					return nil, fmt.Errorf(
						"%w: no marketplace registry sources are configured",
						extensionpkg.ErrMarketplaceSourceUnavailable,
					)
				},
			},
			HomePaths: homePaths,
			Config:    cfg,
			Logger:    testutil.DiscardLogger(),
			StartedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
			Now: func() time.Time {
				return time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC)
			},
			HTTPPort: cfg.HTTP.Port,
		})

		engine := gin.New()
		engine.Use(gin.Recovery())
		engine.GET("/extensions/marketplace", handlers.SearchExtensionMarketplace)

		response := performRequest(t, engine, http.MethodGet, "/extensions/marketplace?q=telemetry", nil)
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf(
				"status = %d, want %d; body=%s",
				response.Code,
				http.StatusServiceUnavailable,
				response.Body.String(),
			)
		}
		if !strings.Contains(response.Body.String(), http.StatusText(http.StatusServiceUnavailable)) {
			t.Fatalf("response body = %s, want masked service unavailable detail", response.Body.String())
		}
		if strings.Contains(response.Body.String(), http.StatusText(http.StatusInternalServerError)) {
			t.Fatalf("response body = %s, want no internal server error masking", response.Body.String())
		}
	})
}
