package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Register persists a workspace registration hint and eagerly resolves it.
func (r *Resolver) Register(ctx context.Context, opts RegisterOptions) (Workspace, error) {
	if err := checkContext(ctx); err != nil {
		return Workspace{}, err
	}

	ws, err := r.createWorkspaceRegistration(ctx, opts)
	if err != nil {
		return Workspace{}, err
	}

	resolved, err := r.Resolve(ctx, ws.ID)
	if err != nil {
		deleteErr := r.store.DeleteWorkspace(context.WithoutCancel(ctx), ws.ID)
		if deleteErr != nil && !errors.Is(deleteErr, ErrWorkspaceNotFound) {
			return Workspace{}, errors.Join(
				err,
				fmt.Errorf("workspace: rollback workspace registration %q: %w", ws.ID, deleteErr),
			)
		}
		return Workspace{}, err
	}

	r.logger.Info("workspace.register",
		"workspace_id", resolved.ID,
		"root_dir", resolved.RootDir,
		"name", resolved.Name,
	)

	return resolved.Workspace, nil
}

// Unregister removes a persisted workspace registration and its cached snapshot.
func (r *Resolver) Unregister(ctx context.Context, id string) error {
	if err := checkContext(ctx); err != nil {
		return err
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return errors.New("workspace: workspace id is required")
	}

	if err := r.store.DeleteWorkspace(ctx, trimmedID); err != nil {
		return fmt.Errorf("workspace: unregister %q: %w", trimmedID, err)
	}

	r.Invalidate(trimmedID)
	return nil
}

// Update mutates an existing workspace registration.
func (r *Resolver) Update(ctx context.Context, id string, opts UpdateOptions) error {
	if err := checkContext(ctx); err != nil {
		return err
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return errors.New("workspace: workspace id is required")
	}

	ws, err := r.store.GetWorkspace(ctx, trimmedID)
	if err != nil {
		return fmt.Errorf("workspace: load workspace %q: %w", trimmedID, err)
	}

	if opts.Name != nil {
		name := strings.TrimSpace(*opts.Name)
		if name == "" {
			return errors.New("workspace: workspace name is required")
		}
		ws.Name = name
	}

	if opts.AdditionalDirs != nil {
		additionalDirs, normalizeErr := normalizeAdditionalDirs(ws.RootDir, *opts.AdditionalDirs)
		if normalizeErr != nil {
			return normalizeErr
		}
		ws.AdditionalDirs = additionalDirs
	}

	if opts.DefaultAgent != nil {
		ws.DefaultAgent = strings.TrimSpace(*opts.DefaultAgent)
	}
	if opts.EnvironmentRef != nil {
		ws.EnvironmentRef = strings.TrimSpace(*opts.EnvironmentRef)
	}

	ws.UpdatedAt = r.now()
	if err := r.store.UpdateWorkspace(ctx, ws); err != nil {
		return fmt.Errorf("workspace: update workspace %q: %w", trimmedID, err)
	}

	r.Invalidate(trimmedID)
	return nil
}

// List returns every registered workspace in stable store order.
func (r *Resolver) List(ctx context.Context) ([]Workspace, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	workspaces, err := r.store.ListWorkspaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("workspace: list workspaces: %w", err)
	}

	return cloneWorkspaces(workspaces), nil
}

// Get resolves a persisted workspace registration without computing a full snapshot.
func (r *Resolver) Get(ctx context.Context, idOrNameOrPath string) (Workspace, error) {
	if err := checkContext(ctx); err != nil {
		return Workspace{}, err
	}

	ws, err := r.lookupWorkspace(ctx, idOrNameOrPath)
	if err != nil {
		return Workspace{}, err
	}

	return cloneWorkspace(ws), nil
}

func (r *Resolver) createWorkspaceRegistration(ctx context.Context, opts RegisterOptions) (Workspace, error) {
	rootDir, err := canonicalRoot(opts.RootDir)
	if err != nil {
		return Workspace{}, err
	}

	additionalDirs, err := normalizeAdditionalDirs(rootDir, opts.AdditionalDirs)
	if err != nil {
		return Workspace{}, err
	}

	name := strings.TrimSpace(opts.Name)
	if name == "" {
		name, err = r.nextWorkspaceName(ctx, rootDir)
		if err != nil {
			return Workspace{}, err
		}
	}

	now := r.now()
	ws := Workspace{
		ID:             r.idGenerator("ws"),
		RootDir:        rootDir,
		AdditionalDirs: additionalDirs,
		Name:           name,
		DefaultAgent:   strings.TrimSpace(opts.DefaultAgent),
		EnvironmentRef: strings.TrimSpace(opts.EnvironmentRef),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	for {
		if err := checkContext(ctx); err != nil {
			return Workspace{}, err
		}

		if insertErr := r.store.InsertWorkspace(ctx, ws); insertErr != nil {
			switch {
			case errors.Is(insertErr, ErrWorkspaceNameTaken) && strings.TrimSpace(opts.Name) == "":
				name, err = r.nextWorkspaceName(ctx, rootDir)
				if err != nil {
					return Workspace{}, err
				}
				ws.Name = name
				ws.ID = r.idGenerator("ws")
				ws.CreatedAt = now
				ws.UpdatedAt = now
				continue
			default:
				return Workspace{}, fmt.Errorf("workspace: register workspace %q: %w", rootDir, insertErr)
			}
		}

		return ws, nil
	}
}

func (r *Resolver) nextWorkspaceName(ctx context.Context, rootDir string) (string, error) {
	workspaces, err := r.store.ListWorkspaces(ctx)
	if err != nil {
		return "", fmt.Errorf("workspace: list workspaces for name dedup: %w", err)
	}

	taken := make(map[string]struct{}, len(workspaces))
	for _, ws := range workspaces {
		if name := strings.TrimSpace(ws.Name); name != "" {
			taken[name] = struct{}{}
		}
	}

	return UniqueWorkspaceName(rootDir, taken), nil
}

func (r *Resolver) lookupWorkspace(ctx context.Context, idOrNameOrPath string) (Workspace, error) {
	target := strings.TrimSpace(idOrNameOrPath)
	if target == "" {
		return Workspace{}, errors.New("workspace: workspace identifier is required")
	}

	switch {
	case strings.HasPrefix(target, "ws_"), strings.HasPrefix(target, "ws-"):
		ws, err := r.store.GetWorkspace(ctx, target)
		switch {
		case err == nil:
			return ws, nil
		case !errors.Is(err, ErrWorkspaceNotFound):
			return Workspace{}, fmt.Errorf("workspace: lookup workspace %q: %w", target, err)
		}
		ws, err = r.store.GetWorkspaceByName(ctx, target)
		if err != nil {
			return Workspace{}, fmt.Errorf("workspace: lookup workspace %q by name fallback: %w", target, err)
		}
		return ws, nil
	case filepath.IsAbs(target):
		canonicalPath, err := canonicalRoot(target)
		if err != nil {
			return Workspace{}, err
		}
		ws, err := r.store.GetWorkspaceByPath(ctx, canonicalPath)
		if err != nil {
			return Workspace{}, fmt.Errorf("workspace: lookup workspace by path %q: %w", canonicalPath, err)
		}
		return ws, nil
	default:
		ws, err := r.store.GetWorkspaceByName(ctx, target)
		if err != nil {
			return Workspace{}, fmt.Errorf("workspace: lookup workspace by name %q: %w", target, err)
		}
		return ws, nil
	}
}

func (r *Resolver) refreshRootDir(ctx context.Context, ws Workspace) (Workspace, error) {
	rootDir := strings.TrimSpace(ws.RootDir)
	if rootDir == "" {
		return Workspace{}, errors.New("workspace: workspace root directory is required")
	}

	info, err := os.Stat(rootDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Workspace{}, ErrWorkspaceRootMissing
		}
		return Workspace{}, fmt.Errorf("workspace: stat workspace root %q: %w", rootDir, err)
	}
	if !info.IsDir() {
		return Workspace{}, fmt.Errorf("workspace: workspace root %q is not a directory", rootDir)
	}

	canonicalDir, err := filepath.EvalSymlinks(rootDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Workspace{}, ErrWorkspaceRootMissing
		}
		return Workspace{}, fmt.Errorf("workspace: evaluate workspace root %q: %w", rootDir, err)
	}
	canonicalDir, err = filepath.Abs(canonicalDir)
	if err != nil {
		return Workspace{}, fmt.Errorf("workspace: resolve workspace root %q: %w", canonicalDir, err)
	}

	if canonicalDir == rootDir {
		return ws, nil
	}

	updated := cloneWorkspace(ws)
	updated.RootDir = canonicalDir
	updated.UpdatedAt = r.now()
	if err := r.store.UpdateWorkspace(ctx, updated); err != nil {
		return Workspace{}, fmt.Errorf("workspace: update canonical workspace root %q: %w", canonicalDir, err)
	}

	r.Invalidate(updated.ID)
	return updated, nil
}
