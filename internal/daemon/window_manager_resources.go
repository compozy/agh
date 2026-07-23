package daemon

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/compozy/agh/internal/resources"
	"github.com/compozy/agh/internal/windowmanager"
)

type windowManagerLayoutRegistry struct {
	catalog *resourceCatalog[windowmanager.LayoutResource]
}

var _ windowmanager.LayoutResourceRegistry = (*windowManagerLayoutRegistry)(nil)

func newWindowManagerLayoutRegistry(
	catalog *resourceCatalog[windowmanager.LayoutResource],
) windowmanager.LayoutResourceRegistry {
	if catalog == nil {
		return nil
	}
	return &windowManagerLayoutRegistry{catalog: catalog}
}

func (r *windowManagerLayoutRegistry) Resolve(
	ctx context.Context,
	workspaceID windowmanager.WorkspaceID,
	resourceID string,
) (windowmanager.LayoutDocument, error) {
	visible, err := r.List(ctx, workspaceID)
	if err != nil {
		return windowmanager.LayoutDocument{}, err
	}
	wanted := strings.TrimSpace(resourceID)
	for _, resource := range visible {
		if resource.ID == wanted {
			return resource.Document, nil
		}
	}
	return windowmanager.LayoutDocument{}, fmt.Errorf(
		"window layout resource %q: %w",
		wanted,
		windowmanager.ErrLayoutResourceNotFound,
	)
}

func (r *windowManagerLayoutRegistry) List(
	ctx context.Context,
	workspaceID windowmanager.WorkspaceID,
) ([]windowmanager.LayoutResource, error) {
	if ctx == nil {
		return nil, errors.New("daemon: window layout registry context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r == nil || r.catalog == nil {
		return nil, errors.New("daemon: window layout catalog is required")
	}
	wantedWorkspace := strings.TrimSpace(string(workspaceID))
	selected := make(map[string]resources.Record[windowmanager.LayoutResource])
	for _, record := range r.catalog.Snapshot() {
		scope := record.Scope.Normalize()
		workspaceScoped := scope.Kind == resources.ResourceScopeKindWorkspace
		if workspaceScoped && scope.ID != wantedWorkspace {
			continue
		}
		if !workspaceScoped && scope.Kind != resources.ResourceScopeKindGlobal {
			return nil, fmt.Errorf("window layout resource %q has invalid scope %q", record.ID, scope.Kind)
		}
		if record.ID != record.Spec.ID {
			return nil, fmt.Errorf("window layout resource ID %q does not match spec ID %q", record.ID, record.Spec.ID)
		}
		current, exists := selected[record.Spec.ID]
		if !exists || (workspaceScoped && current.Scope.Normalize().Kind == resources.ResourceScopeKindGlobal) {
			selected[record.Spec.ID] = record
		}
	}
	visible := make([]windowmanager.LayoutResource, 0, len(selected))
	for _, record := range selected {
		resource := windowmanager.CloneLayoutResource(record.Spec)
		resource.Document.WorkspaceID = workspaceID
		visible = append(visible, resource)
	}
	sort.Slice(visible, func(left, right int) bool { return visible[left].ID < visible[right].ID })
	return visible, nil
}

type windowLayoutProjector struct {
	delegate *resourceCatalogProjector[windowmanager.LayoutResource]
}

func newWindowLayoutProjector(
	catalog *resourceCatalog[windowmanager.LayoutResource],
) resources.TypedProjector[windowmanager.LayoutResource] {
	if catalog == nil {
		return nil
	}
	return &windowLayoutProjector{delegate: &resourceCatalogProjector[windowmanager.LayoutResource]{
		kind: windowmanager.WindowLayoutResourceKind, catalog: catalog,
		cloneSpec: windowmanager.CloneLayoutResource,
	}}
}

func (p *windowLayoutProjector) Kind() resources.ResourceKind {
	return windowmanager.WindowLayoutResourceKind
}

func (p *windowLayoutProjector) DependsOn() []resources.ResourceKind {
	return nil
}

func (p *windowLayoutProjector) Build(
	ctx context.Context,
	records []resources.Record[windowmanager.LayoutResource],
) (resources.ProjectionPlan, error) {
	if p == nil || p.delegate == nil {
		return nil, errors.New("daemon: window layout projector is required")
	}
	for _, record := range records {
		if record.ID != record.Spec.ID {
			return nil, fmt.Errorf(
				"%w: window layout record ID %q does not match spec ID %q",
				resources.ErrValidation,
				record.ID,
				record.Spec.ID,
			)
		}
	}
	return p.delegate.Build(ctx, records)
}

func (p *windowLayoutProjector) Apply(ctx context.Context, plan resources.ProjectionPlan) error {
	if p == nil || p.delegate == nil {
		return errors.New("daemon: window layout projector is required")
	}
	return p.delegate.Apply(ctx, plan)
}
