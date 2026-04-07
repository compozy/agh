package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/session"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

type dreamServiceFactory func(opts ...memory.Option) dreamService

type dreamService interface {
	ShouldRun() (bool, error)
	Run(ctx context.Context, spawn memory.SessionSpawner, workspace string) error
}

type runtimeDreamTrigger struct {
	enabled            bool
	service            dreamService
	spawner            memory.SessionSpawner
	lastConsolidatedAt func() (time.Time, error)
}

func (t runtimeDreamTrigger) Trigger(ctx context.Context, workspace string) (bool, string, error) {
	if !t.Enabled() || t.service == nil || t.spawner == nil {
		return false, "dream consolidation is disabled", nil
	}

	shouldRun, err := t.service.ShouldRun()
	if err != nil {
		return false, "", err
	}
	if !shouldRun {
		return false, "dream consolidation gates are not satisfied", nil
	}
	if err := t.service.Run(ctx, t.spawner, strings.TrimSpace(workspace)); err != nil {
		if errors.Is(err, memory.ErrLockUnavailable) {
			return false, "dream consolidation is already running", nil
		}
		return false, "", err
	}

	return true, "", nil
}

func (t runtimeDreamTrigger) LastConsolidatedAt() (time.Time, error) {
	if t.lastConsolidatedAt == nil {
		return time.Time{}, nil
	}
	return t.lastConsolidatedAt()
}

func (t runtimeDreamTrigger) Enabled() bool {
	return t.enabled
}

type dreamCheckRequest struct {
	reason       string
	workspaceRef string
}

func (d *Daemon) startDreamLoop(parent context.Context) {
	d.mu.Lock()
	if d.dreamService == nil || d.dreamSpawner == nil || d.dreamCheckCh != nil {
		d.mu.Unlock()
		return
	}

	dreamCtx, cancel := context.WithCancel(parent)
	dreamCheckCh := make(chan dreamCheckRequest, 1)
	d.dreamCancel = cancel
	d.dreamCheckCh = dreamCheckCh
	service := d.dreamService
	spawner := d.dreamSpawner
	logger := d.logger
	interval := d.config.Memory.Dream.CheckInterval
	d.dreamWG.Add(1)
	d.mu.Unlock()
	if logger == nil {
		logger = slog.Default()
	}

	go func() {
		defer d.dreamWG.Done()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-dreamCtx.Done():
				return
			case <-ticker.C:
				d.runDreamCheck(dreamCtx, logger, service, spawner, "ticker", "")
			case request := <-dreamCheckCh:
				d.runDreamCheck(dreamCtx, logger, service, spawner, request.reason, request.workspaceRef)
			}
		}
	}()
}

func (d *Daemon) enqueueDreamCheck(reason string, workspaceRef string) {
	d.mu.Lock()
	dreamCheckCh := d.dreamCheckCh
	d.mu.Unlock()

	if dreamCheckCh == nil {
		return
	}

	select {
	case dreamCheckCh <- dreamCheckRequest{
		reason:       strings.TrimSpace(reason),
		workspaceRef: strings.TrimSpace(workspaceRef),
	}:
	default:
		d.runtimeLogger().Debug("daemon: dream check already queued", "reason", reason, "workspace_ref", workspaceRef)
	}
}

func (d *Daemon) runDreamCheck(ctx context.Context, logger *slog.Logger, service dreamService, spawner memory.SessionSpawner, reason string, workspaceRef string) {
	if service == nil || spawner == nil {
		return
	}
	if logger == nil {
		logger = slog.Default()
	}

	logger.Debug("daemon: evaluating dream consolidation gates", "reason", reason, "workspace_ref", workspaceRef)
	shouldRun, err := service.ShouldRun()
	if err != nil {
		logger.Warn("daemon: dream gate evaluation failed", "reason", reason, "workspace_ref", workspaceRef, "error", err)
		return
	}
	if !shouldRun {
		logger.Debug("daemon: dream consolidation skipped", "reason", reason, "workspace_ref", workspaceRef)
		return
	}

	logger.Info("daemon: starting dream consolidation", "reason", reason, "workspace_ref", workspaceRef)
	if err := service.Run(ctx, spawner, workspaceRef); err != nil {
		if errors.Is(err, memory.ErrLockUnavailable) {
			logger.Debug("daemon: dream consolidation already running", "reason", reason, "workspace_ref", workspaceRef)
			return
		}
		logger.Warn("daemon: dream consolidation failed", "reason", reason, "workspace_ref", workspaceRef, "error", err)
		return
	}
	logger.Info("daemon: dream consolidation completed", "reason", reason, "workspace_ref", workspaceRef)
}

func (d *Daemon) makeDreamSpawner(sessions SessionManager, resolver workspacepkg.WorkspaceResolver, cfg aghconfig.Config, globalMemoryDir string) memory.SessionSpawner {
	if !cfg.Memory.Enabled || !cfg.Memory.Dream.Enabled || sessions == nil || resolver == nil {
		return nil
	}

	return func(ctx context.Context, goal, prompt, workspace string) error {
		workspaces, err := d.resolveDreamWorkspaces(ctx, sessions, resolver, globalMemoryDir, workspace)
		if err != nil {
			return err
		}

		for _, workspace := range workspaces {
			if err := spawnDreamSession(ctx, sessions, cfg.Memory.Dream.Agent, goal, prompt, workspace); err != nil {
				return err
			}
		}

		return nil
	}
}

func (d *Daemon) resolveDreamWorkspaces(ctx context.Context, sessions SessionManager, resolver workspacepkg.WorkspaceResolver, globalMemoryDir string, explicitWorkspace string) ([]string, error) {
	if resolver == nil {
		return nil, errors.New("daemon: workspace resolver is required for dream consolidation")
	}

	if workspaceRef := strings.TrimSpace(explicitWorkspace); workspaceRef != "" {
		resolvedRef, err := resolveDreamWorkspaceRef(ctx, resolver, workspaceRef)
		if err != nil {
			return nil, err
		}
		return []string{resolvedRef}, nil
	}

	lockPath := memory.ConsolidationLockPath(globalMemoryDir)
	lastConsolidatedAt, err := memory.NewConsolidationLock(lockPath).LastConsolidatedAt()
	if err != nil {
		return nil, fmt.Errorf("daemon: read dream consolidation lock: %w", err)
	}

	infos, err := sessions.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("daemon: list sessions for dream consolidation: %w", err)
	}

	type workspaceCandidate struct {
		id        string
		updatedAt time.Time
	}

	latestByWorkspace := make(map[string]time.Time, len(infos))
	for _, info := range infos {
		if info == nil || info.Type == session.SessionTypeDream {
			continue
		}

		workspaceID := strings.TrimSpace(info.WorkspaceID)
		if workspaceID == "" {
			continue
		}

		updatedAt := info.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = info.CreatedAt
		}
		if !lastConsolidatedAt.IsZero() && updatedAt.Before(lastConsolidatedAt) {
			continue
		}

		if latest, ok := latestByWorkspace[workspaceID]; !ok || updatedAt.After(latest) {
			latestByWorkspace[workspaceID] = updatedAt
		}
	}

	if len(latestByWorkspace) == 0 {
		return nil, errors.New("daemon: no recent workspaces available for dream consolidation")
	}

	candidates := make([]workspaceCandidate, 0, len(latestByWorkspace))
	for workspaceID, updatedAt := range latestByWorkspace {
		candidates = append(candidates, workspaceCandidate{id: workspaceID, updatedAt: updatedAt})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].updatedAt.Equal(candidates[j].updatedAt) {
			return candidates[i].id < candidates[j].id
		}
		return candidates[i].updatedAt.After(candidates[j].updatedAt)
	})

	workspaces := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		workspaces = append(workspaces, candidate.id)
	}
	return workspaces, nil
}

func resolveDreamWorkspaceRef(ctx context.Context, resolver workspacepkg.WorkspaceResolver, workspaceRef string) (string, error) {
	trimmedRef := strings.TrimSpace(workspaceRef)
	if trimmedRef == "" {
		return "", errors.New("daemon: dream workspace is required")
	}

	var (
		resolved workspacepkg.ResolvedWorkspace
		err      error
	)
	if isPathLikeWorkspaceRef(trimmedRef) {
		normalizedPath, normalizeErr := aghconfig.ResolvePath(trimmedRef)
		if normalizeErr != nil {
			return "", fmt.Errorf("daemon: resolve dream workspace %q: %w", workspaceRef, normalizeErr)
		}
		resolved, err = resolver.ResolveOrRegister(ctx, normalizedPath)
		if err != nil {
			return "", fmt.Errorf("daemon: resolve dream workspace %q: %w", workspaceRef, err)
		}
	} else {
		resolved, err = resolver.Resolve(ctx, trimmedRef)
		if err != nil {
			return "", fmt.Errorf("daemon: resolve dream workspace %q: %w", workspaceRef, err)
		}
	}

	if strings.TrimSpace(resolved.ID) == "" {
		return "", errors.New("daemon: dream workspace id is required")
	}
	return resolved.ID, nil
}

func isPathLikeWorkspaceRef(ref string) bool {
	trimmedRef := strings.TrimSpace(ref)
	return filepath.IsAbs(trimmedRef) ||
		strings.HasPrefix(trimmedRef, ".") ||
		strings.HasPrefix(trimmedRef, "~") ||
		strings.Contains(trimmedRef, string(os.PathSeparator))
}

func spawnDreamSession(ctx context.Context, sessions SessionManager, agentName string, goal string, prompt string, workspace string) (err error) {
	dreamSession, err := sessions.Create(ctx, session.CreateOpts{
		AgentName: agentName,
		Name:      strings.TrimSpace(goal),
		Workspace: strings.TrimSpace(workspace),
		Type:      session.SessionTypeDream,
	})
	if err != nil {
		return fmt.Errorf("daemon: create dream session: %w", err)
	}
	defer func() {
		stopErr := sessions.Stop(ctx, dreamSession.ID)
		if stopErr != nil {
			err = errors.Join(err, fmt.Errorf("daemon: stop dream session %q: %w", dreamSession.ID, stopErr))
		}
	}()

	events, err := sessions.Prompt(ctx, dreamSession.ID, prompt)
	if err != nil {
		return fmt.Errorf("daemon: prompt dream session %q: %w", dreamSession.ID, err)
	}

	for range events {
	}
	return nil
}
