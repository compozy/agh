package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/compozy/agh/internal/clientstate"
	workspacepkg "github.com/compozy/agh/internal/workspace"
	"github.com/google/uuid"
)

type desktopStateBootState struct {
	desktopStateResolver *desktopStateWorkspaceResolver
	desktopState         *clientstate.Engine
}

type desktopStateRuntime struct {
	desktopState *clientstate.Engine
}

type desktopStateWorkspaceRecord struct {
	generation clientstate.WorkspaceGeneration
	deleting   bool
}

type desktopStateWorkspaceResolver struct {
	mu       sync.Mutex
	resolver *workspacepkg.Resolver
	logger   *slog.Logger
	records  map[clientstate.WorkspaceID]desktopStateWorkspaceRecord
}

var _ clientstate.WorkspaceResolver = (*desktopStateWorkspaceResolver)(nil)

func newDesktopStateWorkspaceResolver(
	resolver *workspacepkg.Resolver,
	logger *slog.Logger,
) (*desktopStateWorkspaceResolver, error) {
	if resolver == nil {
		return nil, errors.New("daemon: desktop-state workspace resolver is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &desktopStateWorkspaceResolver{
		resolver: resolver,
		logger:   logger.With("component", "clientstate"),
		records:  make(map[clientstate.WorkspaceID]desktopStateWorkspaceRecord),
	}, nil
}

func (r *desktopStateWorkspaceResolver) ResolveWorkspace(
	ctx context.Context,
	workspaceID clientstate.WorkspaceID,
) (clientstate.WorkspaceGeneration, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	workspace, err := r.resolver.Get(ctx, string(workspaceID))
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		return "", fmt.Errorf(
			"daemon: resolve desktop-state workspace %q: %w",
			workspaceID,
			errors.Join(err, clientstate.ErrWorkspaceNotFound),
		)
	}
	if workspace.ID != string(workspaceID) {
		return "", fmt.Errorf(
			"daemon: desktop-state workspace %q resolved as %q: %w",
			workspaceID,
			workspace.ID,
			clientstate.ErrWorkspaceNotFound,
		)
	}
	record := r.records[workspaceID]
	if record.deleting {
		return "", clientstate.ErrWorkspaceNotFound
	}
	if record.generation == "" {
		record.generation = clientstate.WorkspaceGeneration(uuid.NewString())
		r.records[workspaceID] = record
	}
	return record.generation, nil
}

func (r *desktopStateWorkspaceResolver) ResolveWorkspaceForPurge(
	ctx context.Context,
	workspaceID clientstate.WorkspaceID,
) (clientstate.WorkspaceGeneration, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.records[workspaceID]
	if !ok || !record.deleting || record.generation == "" {
		return "", clientstate.ErrWorkspaceNotFound
	}
	return record.generation, nil
}

func (r *desktopStateWorkspaceResolver) prepareRemoval(
	workspace workspacepkg.Workspace,
	service *clientstate.Engine,
) (workspacepkg.UnregisterPreparation, error) {
	if service == nil {
		return nil, errors.New("daemon: desktop-state service is required")
	}
	workspaceID := clientstate.WorkspaceID(workspace.ID)
	if workspaceID == "" {
		return nil, clientstate.ErrWorkspaceNotFound
	}
	r.mu.Lock()
	record := r.records[workspaceID]
	if record.deleting {
		r.mu.Unlock()
		return nil, clientstate.ErrWorkspaceNotFound
	}
	if record.generation == "" {
		record.generation = clientstate.WorkspaceGeneration(uuid.NewString())
	}
	record.deleting = true
	r.records[workspaceID] = record
	r.mu.Unlock()
	r.logger.Info(
		"clientstate.workspace.deletion_gate",
		"workspace", workspaceID,
		"state", "deleting",
	)
	return &desktopStateRemovalPreparation{
		resolver: r, service: service, workspace: workspaceID, generation: record.generation,
	}, nil
}

type desktopStateRemovalPreparation struct {
	resolver   *desktopStateWorkspaceResolver
	service    *clientstate.Engine
	workspace  clientstate.WorkspaceID
	generation clientstate.WorkspaceGeneration
	purge      clientstate.WorkspacePurgePreparation
}

func (p *desktopStateRemovalPreparation) BeforeDelete(ctx context.Context) error {
	purge, err := p.service.PrepareWorkspacePurge(ctx, p.workspace)
	if err != nil {
		return fmt.Errorf("daemon: stage desktop-state purge for workspace %q: %w", p.workspace, err)
	}
	p.purge = purge
	return nil
}

func (p *desktopStateRemovalPreparation) Commit(ctx context.Context) error {
	if p.purge == nil {
		return errors.New("daemon: desktop-state purge was not staged")
	}
	if err := p.purge.Commit(ctx); err != nil {
		return fmt.Errorf("daemon: commit desktop-state purge for workspace %q: %w", p.workspace, err)
	}
	p.resolver.mu.Lock()
	record := p.resolver.records[p.workspace]
	if record.generation == p.generation && record.deleting {
		delete(p.resolver.records, p.workspace)
	}
	p.resolver.mu.Unlock()
	p.resolver.logger.Info(
		"clientstate.workspace.deletion_gate",
		"workspace", p.workspace,
		"state", "committed",
	)
	return nil
}

func (p *desktopStateRemovalPreparation) Rollback(ctx context.Context) error {
	if p.purge != nil {
		if err := p.purge.Rollback(ctx); err != nil {
			return fmt.Errorf("daemon: roll back desktop-state purge for workspace %q: %w", p.workspace, err)
		}
	} else if err := ctx.Err(); err != nil {
		return err
	}
	p.resolver.mu.Lock()
	record := p.resolver.records[p.workspace]
	if record.generation == p.generation {
		record.deleting = false
		p.resolver.records[p.workspace] = record
	}
	p.resolver.mu.Unlock()
	p.resolver.logger.Info(
		"clientstate.workspace.deletion_gate",
		"workspace", p.workspace,
		"state", "rolled_back",
	)
	return nil
}
