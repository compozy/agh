package daemon

import (
	"context"
	"log/slog"
	"strings"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/skills"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

type sessionLifecycleCallback func(context.Context, *session.Session)

type sessionHookPhase interface {
	OnSessionCreated(ctx context.Context, session *session.Session)
	OnSessionStopped(ctx context.Context, session *session.Session)
}

type notifierFanout struct {
	notifiers          []session.Notifier
	postSessionCreated []sessionLifecycleCallback
	postSessionStopped []sessionLifecycleCallback
	hookPhase          sessionHookPhase
}

var _ session.Notifier = (*notifierFanout)(nil)

func (f *notifierFanout) OnSessionCreated(ctx context.Context, sess *session.Session) {
	for _, notifier := range f.notifiers {
		if notifier == nil {
			continue
		}
		notifier.OnSessionCreated(ctx, sess)
	}
	for _, callback := range f.postSessionCreated {
		if callback == nil {
			continue
		}
		callback(ctx, sess)
	}
	if f.hookPhase != nil {
		f.hookPhase.OnSessionCreated(ctx, sess)
	}
}

func (f *notifierFanout) OnSessionStopped(ctx context.Context, sess *session.Session) {
	for _, notifier := range f.notifiers {
		if notifier == nil {
			continue
		}
		notifier.OnSessionStopped(ctx, sess)
	}
	for _, callback := range f.postSessionStopped {
		if callback == nil {
			continue
		}
		callback(ctx, sess)
	}
	if f.hookPhase != nil {
		f.hookPhase.OnSessionStopped(ctx, sess)
	}
}

func (f *notifierFanout) OnAgentEvent(ctx context.Context, sessionID string, event acp.AgentEvent) {
	for _, notifier := range f.notifiers {
		if notifier == nil {
			continue
		}
		notifier.OnAgentEvent(ctx, sessionID, event)
	}
}

type skillsHookDispatcher struct {
	registry                session.SkillRegistry
	workspaceResolver       workspacepkg.WorkspaceResolver
	logger                  *slog.Logger
	allowedMarketplaceHooks map[string]struct{}
}

var _ sessionHookPhase = (*skillsHookDispatcher)(nil)

func newSkillsHookDispatcher(registry session.SkillRegistry, cfg aghconfig.SkillsConfig, workspaceResolver workspacepkg.WorkspaceResolver, logger *slog.Logger) *skillsHookDispatcher {
	if logger == nil {
		logger = slog.Default()
	}

	return &skillsHookDispatcher{
		registry:                registry,
		workspaceResolver:       workspaceResolver,
		logger:                  logger,
		allowedMarketplaceHooks: marketplaceHookAllowlist(cfg.AllowedMarketplaceHooks),
	}
}

func (d *skillsHookDispatcher) OnSessionCreated(ctx context.Context, sess *session.Session) {
	d.dispatch(ctx, hookspkg.HookSessionPostCreate, sess)
}

func (d *skillsHookDispatcher) OnSessionStopped(ctx context.Context, sess *session.Session) {
	d.dispatch(ctx, hookspkg.HookSessionPostStop, sess)
}

func (d *skillsHookDispatcher) dispatch(ctx context.Context, event hookspkg.HookEvent, sess *session.Session) {
	if d == nil || sess == nil || d.registry == nil || d.workspaceResolver == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	workspaceRef := strings.TrimSpace(sess.WorkspaceID)
	if workspaceRef == "" {
		workspaceRef = strings.TrimSpace(sess.Workspace)
	}
	if workspaceRef == "" {
		return
	}

	resolved, err := d.workspaceResolver.Resolve(ctx, workspaceRef)
	if err != nil {
		d.logger.Warn(
			"daemon: resolve workspace for hook dispatch failed",
			"session_id", sess.ID,
			"workspace_ref", workspaceRef,
			"event", event,
			"error", err,
		)
		return
	}

	activeSkills, err := d.registry.ForWorkspace(ctx, resolved)
	if err != nil {
		d.logger.Warn(
			"daemon: resolve active skills for hook dispatch failed",
			"session_id", sess.ID,
			"workspace_id", resolved.ID,
			"event", event,
			"error", err,
		)
		return
	}

	decls := d.skillHookDecls(activeSkills)
	if len(decls) == 0 {
		return
	}

	dispatcher := hookspkg.NewHooks(
		hookspkg.WithLogger(d.logger),
		hookspkg.WithSkillDeclarations(decls),
	)
	defer dispatcher.Close()

	if err := dispatcher.Rebuild(ctx); err != nil {
		d.logger.Warn(
			"daemon: rebuild transient hook dispatcher failed",
			"session_id", sess.ID,
			"workspace_id", resolved.ID,
			"event", event,
			"error", err,
		)
		return
	}

	payload := hookspkg.SessionLifecyclePayload{
		PayloadBase:    hookspkg.PayloadBase{Event: event},
		SessionContext: hookSessionContext(sess, resolved),
	}

	var dispatchErr error
	switch event {
	case hookspkg.HookSessionPostCreate:
		_, dispatchErr = dispatcher.DispatchSessionPostCreate(ctx, hookspkg.SessionPostCreatePayload(payload))
	case hookspkg.HookSessionPostStop:
		_, dispatchErr = dispatcher.DispatchSessionPostStop(ctx, hookspkg.SessionPostStopPayload(payload))
	default:
		return
	}
	if dispatchErr != nil {
		d.logger.Warn(
			"daemon: dispatch session hook failed",
			"session_id", sess.ID,
			"workspace_id", resolved.ID,
			"event", event,
			"error", dispatchErr,
		)
	}
}

func (d *skillsHookDispatcher) skillHookDecls(activeSkills []*skills.Skill) []hookspkg.HookDecl {
	decls := make([]hookspkg.HookDecl, 0, len(activeSkills))
	for _, skill := range activeSkills {
		if !marketplaceHookAllowed(skill, d.allowedMarketplaceHooks) {
			d.logger.Warn(
				"daemon: blocked hook",
				"skill_name", skill.Meta.Name,
				"source", skills.SkillSourceName(skill.Source),
			)
			continue
		}
		decls = append(decls, skill.Hooks...)
	}

	return decls
}

func hookSessionContext(sess *session.Session, resolved workspacepkg.ResolvedWorkspace) hookspkg.SessionContext {
	if sess == nil {
		return hookspkg.SessionContext{}
	}

	info := sess.Info()
	if info == nil {
		return hookspkg.SessionContext{}
	}

	workspaceID := strings.TrimSpace(info.WorkspaceID)
	if workspaceID == "" {
		workspaceID = strings.TrimSpace(resolved.ID)
	}

	workspaceRoot := strings.TrimSpace(resolved.RootDir)
	if workspaceRoot == "" {
		workspaceRoot = strings.TrimSpace(info.Workspace)
	}

	return hookspkg.SessionContext{
		SessionID:    info.ID,
		SessionName:  info.Name,
		SessionType:  string(info.Type),
		AgentName:    info.AgentName,
		WorkspaceID:  workspaceID,
		Workspace:    workspaceRoot,
		ACPSessionID: info.ACPSessionID,
		State:        string(info.State),
	}
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
