package httpapi

import (
	"context"
	"testing"

	"github.com/pedronauck/agh/internal/modelcatalog"
)

func TestHTTPHandlersModelCatalogDependency(t *testing.T) {
	t.Parallel()

	t.Run("ShouldPassModelCatalogServiceToBaseHandlers", func(t *testing.T) {
		t.Parallel()

		service := httpModelCatalogServiceStub{}
		handlers := newHandlers(&handlerConfig{modelCatalog: service})
		if handlers.BaseHandlers == nil {
			t.Fatal("newHandlers() BaseHandlers = nil")
		}
		if handlers.ModelCatalog == nil {
			t.Fatal("newHandlers() ModelCatalog = nil, want injected service")
		}
		if handlers.ModelCatalog != service {
			t.Fatalf("newHandlers() ModelCatalog = %#v, want %#v", handlers.ModelCatalog, service)
		}
	})
}

type httpModelCatalogServiceStub struct{}

func (httpModelCatalogServiceStub) ListModels(
	context.Context,
	modelcatalog.ListOptions,
) ([]modelcatalog.Model, error) {
	return nil, nil
}

func (httpModelCatalogServiceStub) Refresh(
	context.Context,
	modelcatalog.RefreshOptions,
) ([]modelcatalog.SourceStatus, error) {
	return nil, nil
}

func (httpModelCatalogServiceStub) ListSourceStatus(
	context.Context,
	string,
) ([]modelcatalog.SourceStatus, error) {
	return nil, nil
}
