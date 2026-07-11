package session

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	aghconfig "github.com/compozy/agh/internal/config"
	hookspkg "github.com/compozy/agh/internal/hooks"
)

func (m *Manager) prepareCreateStart(ctx context.Context, opts CreateOpts) (sessionStartSpec, error) {
	opts, err := m.dispatchSessionPreCreate(ctx, opts)
	if err != nil {
		return sessionStartSpec{}, err
	}

	resolvedWorkspace, err := m.resolveCreateWorkspace(ctx, opts)
	if err != nil {
		return sessionStartSpec{}, err
	}
	sandboxDisabled, err := applyCreateSandboxOverride(&resolvedWorkspace, opts)
	if err != nil {
		return sessionStartSpec{}, err
	}
	cwd, err := ResolveSessionCWD(resolvedWorkspace.RootDir, opts.CWD)
	if err != nil {
		return sessionStartSpec{}, err
	}

	agentName, err := aghconfig.ResolveAgentName(opts.AgentName, resolvedWorkspace.Config.Defaults)
	if err != nil {
		return sessionStartSpec{}, fmt.Errorf("session: resolve agent name: %w", err)
	}
	sessionID, err := m.createSessionID(opts.DesiredSessionID)
	if err != nil {
		return sessionStartSpec{}, err
	}
	sandboxID := strings.TrimSpace(m.newSandboxID())
	if sandboxID == "" {
		return sessionStartSpec{}, errors.New("session: sandbox id generator returned empty id")
	}
	lineage, err := m.normalizeCreateLineage(ctx, sessionID, opts.Type, opts.Lineage)
	if err != nil {
		return sessionStartSpec{}, err
	}

	return sessionStartSpec{
		sessionID:               sessionID,
		sandboxID:               sandboxID,
		sessionName:             strings.TrimSpace(opts.Name),
		agentName:               strings.TrimSpace(agentName),
		provider:                strings.TrimSpace(opts.Provider),
		model:                   strings.TrimSpace(opts.Model),
		reasoningEffort:         strings.TrimSpace(opts.ReasoningEffort),
		permissions:             opts.Permissions,
		sandboxDisabled:         sandboxDisabled,
		workspace:               resolvedWorkspace,
		cwd:                     cwd,
		channel:                 strings.TrimSpace(opts.Channel),
		promptOverlay:           strings.TrimSpace(opts.PromptOverlay),
		contractOverlay:         strings.TrimSpace(opts.ContractOverlay),
		runtimeMode:             strings.TrimSpace(opts.RuntimeMode),
		sessionType:             normalizeSessionType(opts.Type),
		lineage:                 lineage,
		allowedToolsOverride:    append([]string(nil), opts.AllowedToolsOverride...),
		creationProfile:         cloneCreationProfile(opts.CreationProfile),
		creationIdentity:        cloneCreationIdentity(opts.CreationIdentity),
		creationIdentityPinned:  opts.CreationProfile != nil || opts.CreationIdentity != nil,
		creationIdentityEnabled: m.creationStore != nil || opts.CreationProfile != nil || opts.CreationIdentity != nil,
		parentSoulDigest:        strings.TrimSpace(opts.ParentSoulDigest),
		postEvent:               hookspkg.HookSessionPostCreate,
		startAction:             sessionStartActionCreate,
		cleanupSessionDir:       true,
	}, nil
}

func (m *Manager) createSessionID(desired string) (string, error) {
	sessionID := strings.TrimSpace(desired)
	if sessionID == "" {
		sessionID = strings.TrimSpace(m.newSessionID())
		if sessionID == "" {
			return "", errors.New("session: session id generator returned empty id")
		}
	}
	if len(sessionID) > 128 || sessionID == "." || sessionID == ".." || filepath.Base(sessionID) != sessionID ||
		strings.ContainsAny(sessionID, `/\\`) {
		return "", fmt.Errorf("%w: invalid preallocated session id %q", ErrValidation, sessionID)
	}
	return sessionID, nil
}

// ResolveSessionCWD normalizes a creation CWD and rejects workspace escape.
func ResolveSessionCWD(root string, requested string) (string, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "." || root == "" {
		return "", fmt.Errorf("%w: resolved workspace root is required", ErrValidation)
	}
	target := strings.TrimSpace(requested)
	if target == "" {
		target = root
	} else if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}
	target = filepath.Clean(target)
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: session cwd must remain within the workspace", ErrValidation)
	}
	info, err := os.Stat(target)
	if err != nil {
		return "", fmt.Errorf("%w: inspect session cwd %q: %w", ErrValidation, target, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%w: session cwd %q is not a directory", ErrValidation, target)
	}
	return target, nil
}
