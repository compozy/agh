package core

import (
	"context"
	"testing"

	"github.com/pedronauck/agh/internal/modelcatalog"
)

func TestBaseHandlersModelCatalogDependency(t *testing.T) {
	t.Parallel()

	t.Run("ShouldCarryModelCatalogServiceFromConfig", func(t *testing.T) {
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
