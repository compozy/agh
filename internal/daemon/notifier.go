package daemon

import (
	"context"
	"log/slog"
	"strings"

	"github.com/pedronauck/agh/internal/acp"
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
	registry          session.SkillRegistry
	runner            *skills.HookRunner
	workspaceResolver workspacepkg.WorkspaceResolver
	logger            *slog.Logger
}

var _ sessionHookPhase = (*skillsHookDispatcher)(nil)

func newSkillsHookDispatcher(registry session.SkillRegistry, runner *skills.HookRunner, workspaceResolver workspacepkg.WorkspaceResolver, logger *slog.Logger) *skillsHookDispatcher {
	if logger == nil {
		logger = slog.Default()
	}

	return &skillsHookDispatcher{
		registry:          registry,
		runner:            runner,
		workspaceResolver: workspaceResolver,
		logger:            logger,
	}
}

func (d *skillsHookDispatcher) OnSessionCreated(ctx context.Context, sess *session.Session) {
	d.dispatch(ctx, skills.HookSessionCreated, sess)
}

func (d *skillsHookDispatcher) OnSessionStopped(ctx context.Context, sess *session.Session) {
	d.dispatch(ctx, skills.HookSessionStopped, sess)
}

func (d *skillsHookDispatcher) dispatch(ctx context.Context, event skills.HookEvent, sess *session.Session) {
	if d == nil || sess == nil || d.registry == nil || d.runner == nil || d.workspaceResolver == nil {
		return
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

	d.runner.RunHooks(ctx, event, activeSkills, skills.HookPayload{
		SessionID: sess.ID,
		AgentName: sess.AgentName,
		Workspace: resolved.RootDir,
	})
}
