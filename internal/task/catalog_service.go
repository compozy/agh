package task

import (
	"context"
	"fmt"
)

// ListTaskCatalog returns one batched public catalog page after enforcing read authority.
func (m *Service) ListTaskCatalog(
	ctx context.Context,
	query CatalogQuery,
	actor ActorContext,
) (CatalogPage, error) {
	if err := requireReadAuthority(actor); err != nil {
		return CatalogPage{}, err
	}
	reader, ok := m.store.(CatalogReader)
	if !ok {
		return CatalogPage{}, fmt.Errorf("task: catalog reader is not configured")
	}
	return reader.ListTaskCatalog(ctx, query)
}

var _ CatalogManager = (*Service)(nil)
