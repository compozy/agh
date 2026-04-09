package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/skills"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

type hookRuntime interface {
	Rebuild(context.Context) error
	Close()
	Version() int64
	DispatchSessionPostCreate(context.Context, hookspkg.SessionPostCreatePayload) (hookspkg.SessionPostCreatePayload, error)
	DispatchSessionPostStop(context.Context, hookspkg.SessionPostStopPayload) (hookspkg.SessionPostStopPayload, error)
	OnAgentEvent(context.Context, string, any)
}

type sessionLifecycleObserver interface {
	OnSessionCreated(context.Context, *session.Session)
	OnSessionStopped(context.Context, *session.Session)
}

type dreamCheckEnqueuer interface {
	EnqueueCheck(reason string, workspaceRef string)
}

type hooksNotifier struct {
	mu sync.RWMutex

	logger           *slog.Logger
	now              func() time.Time
	hooks            hookRuntime
	agentEventNotify session.Notifier
}

var _ session.Notifier = (*hooksNotifier)(nil)

func newHooksNotifier(logger *slog.Logger, now func() time.Time) *hooksNotifier {
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}

	return &hooksNotifier{
		logger: logger,
		now:    now,
	}
}

func (n *hooksNotifier) setRuntime(hooks hookRuntime, agentEventNotify session.Notifier) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.hooks = hooks
	n.agentEventNotify = agentEventNotify
}

func (n *hooksNotifier) OnSessionCreated(ctx context.Context, sess *session.Session) {
	n.dispatchSessionLifecycle(ctx, sess, hookspkg.HookSessionPostCreate)
}

func (n *hooksNotifier) OnSessionStopped(ctx context.Context, sess *session.Session) {
	n.dispatchSessionLifecycle(ctx, sess, hookspkg.HookSessionPostStop)
}

func (n *hooksNotifier) OnAgentEvent(ctx context.Context, sessionID string, event any) {
	hooks, agentEventNotify := n.runtime()
	if agentEventNotify != nil {
		agentEventNotify.OnAgentEvent(ctx, sessionID, event)
	}
	if hooks != nil {
		hooks.OnAgentEvent(ctx, sessionID, event)
	}
}

func (n *hooksNotifier) dispatchSessionLifecycle(ctx context.Context, sess *session.Session, event hookspkg.HookEvent) {
	hooks, _ := n.runtime()
	if hooks == nil || sess == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if err := hooks.Rebuild(ctx); err != nil {
		n.logger.WarnContext(
			ctx,
			"daemon: rebuild hooks before lifecycle dispatch failed",
			"session_id", strings.TrimSpace(sess.ID),
			"event", event.String(),
			"error", err,
		)
	}

	payload := hookSessionLifecyclePayload(sess, event, n.timestamp())

	var dispatchErr error
	switch event {
	case hookspkg.HookSessionPostCreate:
		_, dispatchErr = hooks.DispatchSessionPostCreate(ctx, hookspkg.SessionPostCreatePayload(payload))
	case hookspkg.HookSessionPostStop:
		_, dispatchErr = hooks.DispatchSessionPostStop(ctx, hookspkg.SessionPostStopPayload(payload))
	default:
		return
	}
	if dispatchErr != nil {
		n.logger.WarnContext(
			ctx,
			"daemon: dispatch lifecycle hooks failed",
			"session_id", strings.TrimSpace(sess.ID),
			"event", event.String(),
			"error", dispatchErr,
		)
	}
}

func (n *hooksNotifier) runtime() (hookRuntime, session.Notifier) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.hooks, n.agentEventNotify
}

func (n *hooksNotifier) timestamp() time.Time {
	if n == nil || n.now == nil {
		return time.Now().UTC()
	}
	return n.now().UTC()
}

func hookSessionLifecyclePayload(sess *session.Session, event hookspkg.HookEvent, timestamp time.Time) hookspkg.SessionLifecyclePayload {
	return hookspkg.SessionLifecyclePayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     event,
			Timestamp: timestamp,
		},
		SessionContext: hookSessionContext(sess),
	}
}

func hookSessionContext(sess *session.Session) hookspkg.SessionContext {
	if sess == nil {
		return hookspkg.SessionContext{}
	}

	info := sess.Info()
	if info == nil {
		return hookspkg.SessionContext{}
	}

	return hookspkg.SessionContext{
		SessionID:    strings.TrimSpace(info.ID),
		SessionName:  strings.TrimSpace(info.Name),
		SessionType:  string(info.Type),
		AgentName:    strings.TrimSpace(info.AgentName),
		WorkspaceID:  strings.TrimSpace(info.WorkspaceID),
		Workspace:    strings.TrimSpace(info.Workspace),
		ACPSessionID: strings.TrimSpace(info.ACPSessionID),
		State:        string(info.State),
		CreatedAt:    info.CreatedAt,
		UpdatedAt:    info.UpdatedAt,
	}
}

func sessionFromHookPayload(payload hookspkg.SessionLifecyclePayload) *session.Session {
	return &session.Session{
		ID:           strings.TrimSpace(payload.SessionID),
		Name:         strings.TrimSpace(payload.SessionName),
		AgentName:    strings.TrimSpace(payload.AgentName),
		WorkspaceID:  strings.TrimSpace(payload.WorkspaceID),
		Workspace:    strings.TrimSpace(payload.Workspace),
		Type:         session.SessionType(strings.TrimSpace(payload.SessionType)),
		State:        session.SessionState(strings.TrimSpace(payload.State)),
		ACPSessionID: strings.TrimSpace(payload.ACPSessionID),
		CreatedAt:    payload.CreatedAt,
		UpdatedAt:    payload.UpdatedAt,
	}
}

func daemonNativeHooks(observer sessionLifecycleObserver, dreamRuntime dreamCheckEnqueuer) ([]hookspkg.HookDecl, map[string]hookspkg.Executor) {
	decls := make([]hookspkg.HookDecl, 0, 3)
	executors := make(map[string]hookspkg.Executor, 3)

	if observer != nil {
		const (
			createName = "daemon.observe.session_post_create"
			stopName   = "daemon.observe.session_post_stop"
		)

		decls = append(decls,
			hookspkg.HookDecl{
				Name:         createName,
				Event:        hookspkg.HookSessionPostCreate,
				Mode:         hookspkg.HookModeSync,
				Priority:     1000,
				PrioritySet:  true,
				ExecutorKind: hookspkg.HookExecutorNative,
			},
			hookspkg.HookDecl{
				Name:         stopName,
				Event:        hookspkg.HookSessionPostStop,
				Mode:         hookspkg.HookModeSync,
				Priority:     1000,
				PrioritySet:  true,
				ExecutorKind: hookspkg.HookExecutorNative,
			},
		)
		executors[createName] = hookspkg.NewTypedNativeExecutor(func(ctx context.Context, _ hookspkg.RegisteredHook, payload hookspkg.SessionLifecyclePayload) (hookspkg.SessionPostCreatePatch, error) {
			observer.OnSessionCreated(ctx, sessionFromHookPayload(payload))
			return hookspkg.SessionPostCreatePatch{}, nil
		})
		executors[stopName] = hookspkg.NewTypedNativeExecutor(func(ctx context.Context, _ hookspkg.RegisteredHook, payload hookspkg.SessionLifecyclePayload) (hookspkg.SessionPostStopPatch, error) {
			observer.OnSessionStopped(ctx, sessionFromHookPayload(payload))
			return hookspkg.SessionPostStopPatch{}, nil
		})
	}

	if dreamRuntime != nil {
		const dreamName = "daemon.dream.session_post_stop"

		decls = append(decls, hookspkg.HookDecl{
			Name:         dreamName,
			Event:        hookspkg.HookSessionPostStop,
			Mode:         hookspkg.HookModeSync,
			Priority:     900,
			PrioritySet:  true,
			ExecutorKind: hookspkg.HookExecutorNative,
		})
		executors[dreamName] = hookspkg.NewTypedNativeExecutor(func(_ context.Context, _ hookspkg.RegisteredHook, payload hookspkg.SessionLifecyclePayload) (hookspkg.SessionPostStopPatch, error) {
			if strings.TrimSpace(payload.WorkspaceID) == "" || session.SessionType(strings.TrimSpace(payload.SessionType)) == session.SessionTypeDream {
				return hookspkg.SessionPostStopPatch{}, nil
			}

			dreamRuntime.EnqueueCheck("session_stop", strings.TrimSpace(payload.WorkspaceID))
			return hookspkg.SessionPostStopPatch{}, nil
		})
	}

	return decls, executors
}

func daemonExecutorResolver(nativeExecutors map[string]hookspkg.Executor) hookspkg.ExecutorResolver {
	return func(decl hookspkg.HookDecl) (hookspkg.Executor, error) {
		if decl.ExecutorKind == hookspkg.HookExecutorNative {
			executor := nativeExecutors[strings.TrimSpace(decl.Name)]
			if executor == nil {
				return nil, fmt.Errorf("daemon: missing native hook executor for %q", decl.Name)
			}
			return executor, nil
		}
		return defaultDaemonExecutorResolver(decl)
	}
}

func defaultDaemonExecutorResolver(decl hookspkg.HookDecl) (hookspkg.Executor, error) {
	switch decl.ExecutorKind {
	case hookspkg.HookExecutorSubprocess:
		return hookspkg.NewSubprocessExecutor(
			decl.Command,
			decl.Args,
			hookspkg.WithSubprocessEnv(decl.Env),
		), nil
	case hookspkg.HookExecutorWASM:
		return &hookspkg.WasmExecutor{}, nil
	case hookspkg.HookExecutorNative:
		return nil, fmt.Errorf("daemon: native executor for hook %q requires an explicit binding", decl.Name)
	default:
		return nil, fmt.Errorf("daemon: unsupported executor kind %q for hook %q", decl.ExecutorKind, decl.Name)
	}
}

func configDeclarationProvider(registry Registry, workspaceResolver workspacepkg.WorkspaceResolver, logger *slog.Logger) hookspkg.DeclarationProvider {
	if logger == nil {
		logger = slog.Default()
	}
	return func(ctx context.Context) ([]hookspkg.HookDecl, error) {
		decls, err := workspaceHookDeclarations(ctx, registry, workspaceResolver, logger)
		if err != nil {
			return nil, err
		}
		return filterHookDeclsBySource(decls, hookspkg.HookSourceConfig), nil
	}
}

func agentDeclarationProvider(registry Registry, workspaceResolver workspacepkg.WorkspaceResolver, logger *slog.Logger) hookspkg.DeclarationProvider {
	if logger == nil {
		logger = slog.Default()
	}
	return func(ctx context.Context) ([]hookspkg.HookDecl, error) {
		decls, err := workspaceHookDeclarations(ctx, registry, workspaceResolver, logger)
		if err != nil {
			return nil, err
		}
		return filterHookDeclsBySource(decls, hookspkg.HookSourceAgentDefinition), nil
	}
}

func skillDeclarationProvider(skillsRegistry *skills.Registry, registry Registry, workspaceResolver workspacepkg.WorkspaceResolver, allowedMarketplaceHooks []string, logger *slog.Logger) hookspkg.DeclarationProvider {
	if logger == nil {
		logger = slog.Default()
	}
	allowed := marketplaceHookAllowlist(allowedMarketplaceHooks)

	return func(ctx context.Context) ([]hookspkg.HookDecl, error) {
		if skillsRegistry == nil || registry == nil || workspaceResolver == nil {
			return nil, nil
		}

		workspaces, err := registeredWorkspaces(ctx, registry, workspaceResolver, logger)
		if err != nil {
			return nil, err
		}

		decls := make([]hookspkg.HookDecl, 0, len(workspaces))
		for _, resolved := range workspaces {
			activeSkills, err := skillsRegistry.ForWorkspace(ctx, resolved)
			if err != nil {
				return nil, fmt.Errorf("daemon: resolve active skills for workspace %q: %w", resolved.ID, err)
			}

			for _, skill := range activeSkills {
				if !marketplaceHookAllowed(skill, allowed) {
					logger.Warn(
						"daemon: blocked hook",
						"skill_name", skill.Meta.Name,
						"workspace_id", resolved.ID,
						"source", skills.SkillSourceName(skill.Source),
					)
					continue
				}
				decls = append(decls, scopeWorkspaceHookDecls(skill.Hooks, resolved)...)
			}
		}

		return decls, nil
	}
}

func workspaceHookDeclarations(ctx context.Context, registry Registry, workspaceResolver workspacepkg.WorkspaceResolver, logger *slog.Logger) ([]hookspkg.HookDecl, error) {
	workspaces, err := registeredWorkspaces(ctx, registry, workspaceResolver, logger)
	if err != nil {
		return nil, err
	}

	decls := make([]hookspkg.HookDecl, 0, len(workspaces))
	for _, resolved := range workspaces {
		workspaceDecls, err := aghconfig.HookDeclarations(resolved.Config, resolved.Agents)
		if err != nil {
			return nil, fmt.Errorf("daemon: load hook declarations for workspace %q: %w", resolved.ID, err)
		}
		decls = append(decls, scopeWorkspaceHookDecls(workspaceDecls, resolved)...)
	}

	return decls, nil
}

func registeredWorkspaces(ctx context.Context, registry Registry, workspaceResolver workspacepkg.WorkspaceResolver, logger *slog.Logger) ([]workspacepkg.ResolvedWorkspace, error) {
	if registry == nil || workspaceResolver == nil {
		return nil, nil
	}

	workspaces, err := registry.ListWorkspaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("daemon: list workspaces for hooks rebuild: %w", err)
	}
	slices.SortFunc(workspaces, func(left, right workspacepkg.Workspace) int {
		return strings.Compare(strings.TrimSpace(left.ID), strings.TrimSpace(right.ID))
	})

	resolvedWorkspaces := make([]workspacepkg.ResolvedWorkspace, 0, len(workspaces))
	for _, workspace := range workspaces {
		resolved, err := workspaceResolver.Resolve(ctx, workspace.ID)
		switch {
		case err == nil:
			resolvedWorkspaces = append(resolvedWorkspaces, resolved)
		case errors.Is(err, workspacepkg.ErrWorkspaceNotFound), errors.Is(err, workspacepkg.ErrWorkspaceRootMissing):
			if logger != nil {
				logger.Warn(
					"daemon: skipped workspace while rebuilding hooks",
					"workspace_id", workspace.ID,
					"workspace_root", workspace.RootDir,
					"error", err,
				)
			}
		default:
			return nil, fmt.Errorf("daemon: resolve workspace %q for hooks rebuild: %w", workspace.ID, err)
		}
	}

	return resolvedWorkspaces, nil
}

func filterHookDeclsBySource(decls []hookspkg.HookDecl, source hookspkg.HookSource) []hookspkg.HookDecl {
	filtered := make([]hookspkg.HookDecl, 0, len(decls))
	for _, decl := range decls {
		if decl.Source != source {
			continue
		}
		filtered = append(filtered, cloneDaemonHookDecl(decl))
	}
	return filtered
}

func scopeWorkspaceHookDecls(decls []hookspkg.HookDecl, resolved workspacepkg.ResolvedWorkspace) []hookspkg.HookDecl {
	scoped := make([]hookspkg.HookDecl, 0, len(decls))
	for _, decl := range decls {
		cloned := cloneDaemonHookDecl(decl)
		cloned.Matcher.WorkspaceID = strings.TrimSpace(resolved.ID)
		cloned.Matcher.WorkspaceRoot = strings.TrimSpace(resolved.RootDir)
		scoped = append(scoped, cloned)
	}
	return scoped
}

func cloneDaemonHookDecl(src hookspkg.HookDecl) hookspkg.HookDecl {
	cloned := src
	cloned.Args = append([]string(nil), src.Args...)
	cloned.Env = cloneStringMap(src.Env)
	cloned.Metadata = cloneStringMap(src.Metadata)
	if src.Matcher.ToolReadOnly != nil {
		value := *src.Matcher.ToolReadOnly
		cloned.Matcher.ToolReadOnly = &value
	}
	return cloned
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(src))
	for key, value := range src {
		cloned[key] = value
	}
	return cloned
}

func marketplaceHookAllowlist(values []string) map[string]struct{} {
	allowed := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		allowed[trimmed] = struct{}{}
	}

	return allowed
}

func marketplaceHookAllowed(skill *skills.Skill, allowedMarketplaceHooks map[string]struct{}) bool {
	if skill == nil {
		return false
	}

	switch skill.Source {
	case skills.SourceBundled, skills.SourceUser, skills.SourceAdditional, skills.SourceWorkspace:
		return true
	case skills.SourceMarketplace:
		for _, key := range marketplaceHookConsentKeys(skill) {
			if _, ok := allowedMarketplaceHooks[key]; ok {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func marketplaceHookConsentKeys(skill *skills.Skill) []string {
	if skill == nil || skill.Provenance == nil {
		return nil
	}

	keys := make([]string, 0, 3)
	if slug := strings.TrimSpace(skill.Provenance.Slug); slug != "" {
		keys = append(keys, slug)
		if registry := strings.TrimSpace(skill.Provenance.Registry); registry != "" {
			keys = append(keys, registry+":"+slug)
		}
	}
	if hash := strings.TrimSpace(skill.Provenance.Hash); hash != "" {
		keys = append(keys, hash)
	}

	return keys
}
