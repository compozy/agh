package udsapi

import (
	"context"
	"testing"

	"github.com/pedronauck/agh/internal/modelcatalog"
)

func TestUDSHandlersModelCatalogDependency(t *testing.T) {
	t.Parallel()

	t.Run("ShouldPassModelCatalogServiceToBaseHandlers", func(t *testing.T) {
		t.Parallel()

		service := udsModelCatalogServiceStub{}
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

type udsModelCatalogServiceStub struct{}

func (udsModelCatalogServiceStub) ListModels(
	context.Context,
	modelcatalog.ListOptions,
) ([]modelcatalog.Model, error) {
	return nil, nil
}

func (udsModelCatalogServiceStub) Refresh(
	context.Context,
	modelcatalog.RefreshOptions,
) ([]modelcatalog.SourceStatus, error) {
	return nil, nil
}

func (udsModelCatalogServiceStub) ListSourceStatus(
	context.Context,
	string,
) ([]modelcatalog.SourceStatus, error) {
	return nil, nil
}
