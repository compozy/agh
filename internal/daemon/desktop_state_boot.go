package daemon

import (
	"context"
	"errors"
	"fmt"

	"github.com/compozy/agh/internal/clientstate"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

func (d *Daemon) bootDefaultWorkspaceAndClientState(
	ctx context.Context,
	state *bootState,
	cleanup *bootCleanup,
) error {
	if err := d.ensureDefaultWorkspace(ctx, state); err != nil {
		return err
	}
	resolver, err := newDesktopStateWorkspaceResolver(state.workspaceResolver, state.logger)
	if err != nil {
		return err
	}
	engine, err := clientstate.Open(
		clientstate.DatabasePath(d.homePaths.HomeDir),
		resolver,
		clientstate.Limits{
			MaxValueBytes:       state.cfg.DesktopState.MaxValueKiB * 1024,
			MaxKeysPerWorkspace: state.cfg.DesktopState.MaxKeysPerWorkspace,
		},
		clientstate.WithLogger(state.logger),
	)
	if err != nil {
		return fmt.Errorf("daemon: open desktop-state store: %w", err)
	}
	cleanup.add(func(context.Context) error { return engine.Close() })
	state.desktopStateResolver = resolver
	state.desktopState = engine
	return nil
}

func installWorkspaceRemovalPreparer(state *bootState, sessions SessionManager) error {
	preparer, ok := sessions.(workspaceRemovalPreparer)
	if !ok {
		return errMissingWorkspaceRemovalPreparation
	}
	if state.desktopStateResolver == nil || state.desktopState == nil {
		return errors.New("daemon: desktop-state removal dependencies are required")
	}
	state.workspaceResolver.SetUnregisterPreparer(
		func(ctx context.Context, workspace workspacepkg.Workspace) (workspacepkg.UnregisterPreparation, error) {
			sessionPreparation, err := preparer.PrepareWorkspaceRemoval(ctx, workspace.ID)
			if err != nil {
				return nil, err
			}
			if sessionPreparation == nil {
				return nil, errors.New("daemon: session workspace removal preparation is required")
			}
			desktopPreparation, err := state.desktopStateResolver.prepareRemoval(workspace, state.desktopState)
			if err != nil {
				rollbackErr := sessionPreparation.Rollback(context.WithoutCancel(ctx))
				return nil, errors.Join(err, rollbackErr)
			}
			return workspaceRemovalPreparation{
				desktop: desktopPreparation,
				session: sessionPreparation,
			}, nil
		},
	)
	return nil
}

type workspaceRemovalPreparation struct {
	desktop workspacepkg.UnregisterPreparation
	session workspacepkg.UnregisterPreparation
}

func (p workspaceRemovalPreparation) BeforeDelete(ctx context.Context) error {
	if err := p.desktop.BeforeDelete(ctx); err != nil {
		return err
	}
	return p.session.BeforeDelete(ctx)
}

func (p workspaceRemovalPreparation) Commit(ctx context.Context) error {
	return errors.Join(p.desktop.Commit(ctx), p.session.Commit(ctx))
}

func (p workspaceRemovalPreparation) Rollback(ctx context.Context) error {
	return errors.Join(p.desktop.Rollback(ctx), p.session.Rollback(ctx))
}
