package daemon

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pedronauck/agh/internal/session"
	skillspkg "github.com/pedronauck/agh/internal/skills"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

type promptSkillsRegistry interface {
	ForWorkspace(ctx context.Context, resolved *workspacepkg.ResolvedWorkspace) ([]*skillspkg.Skill, error)
	ForAgent(
		ctx context.Context,
		resolved *workspacepkg.ResolvedWorkspace,
		agentName string,
	) ([]*skillspkg.Skill, error)
}

type promptSkillsWorkspaceResolver interface {
	Resolve(ctx context.Context, target string) (workspacepkg.ResolvedWorkspace, error)
}

func newSkillsCatalogAugmenter(
	registry promptSkillsRegistry,
	workspaceResolver func() promptSkillsWorkspaceResolver,
) session.PromptInputAugmenter {
	if registry == nil {
		return nil
	}

	return func(ctx context.Context, sess *session.Session, message string) (string, error) {
		if sess == nil {
			return message, nil
		}

		info := sess.Info()
		if info == nil {
			return message, nil
		}

		workspace, err := resolvePromptSkillsWorkspace(ctx, workspaceResolver, info.WorkspaceID, info.Workspace)
		if err != nil {
			return "", fmt.Errorf("daemon: resolve prompt skills workspace: %w", err)
		}

		var skills []*skillspkg.Skill
		agentName := strings.TrimSpace(info.AgentName)
		if agentName != "" {
			skills, err = registry.ForAgent(ctx, workspace, agentName)
		} else {
			skills, err = registry.ForWorkspace(ctx, workspace)
		}
		if err != nil {
			return "", fmt.Errorf("daemon: load current skills catalog for session %q: %w", info.ID, err)
		}

		catalog := skillspkg.BuildCurrentCatalog(skills)
		if strings.TrimSpace(catalog) == "" {
			return message, nil
		}
		if strings.TrimSpace(message) == "" {
			return catalog, nil
		}
		return catalog + "\n\n" + message, nil
	}
}

func resolvePromptSkillsWorkspace(
	ctx context.Context,
	resolverGetter func() promptSkillsWorkspaceResolver,
	workspaceID string,
	workspaceRoot string,
) (*workspacepkg.ResolvedWorkspace, error) {
	target := firstTrimmed(workspaceID, workspaceRoot)
	var resolver promptSkillsWorkspaceResolver
	if resolverGetter != nil {
		resolver = resolverGetter()
	}
	if resolver != nil && target != "" {
		resolved, err := resolver.Resolve(ctx, target)
		if err == nil {
			return &resolved, nil
		}
		if isContextError(err) {
			return nil, err
		}
	}

	if target == "" {
		return nil, nil
	}
	return &workspacepkg.ResolvedWorkspace{
		Workspace: workspacepkg.Workspace{
			ID:      strings.TrimSpace(workspaceID),
			RootDir: strings.TrimSpace(workspaceRoot),
		},
	}, nil
}

func firstTrimmed(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func isContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
