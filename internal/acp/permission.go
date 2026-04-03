package acp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	acpsdk "github.com/coder/acp-go-sdk"
	aghconfig "github.com/pedronauck/agh/internal/config"
)

var (
	// ErrPermissionDenied reports that the configured static policy rejected an operation.
	ErrPermissionDenied = errors.New("acp: permission denied")
	// ErrPathOutsideWorkspace reports that a requested path escapes the session root.
	ErrPathOutsideWorkspace = errors.New("acp: path outside session workspace")
)

type permissionOperation string

const (
	permissionReadTextFile     permissionOperation = "fs/read_text_file"
	permissionWriteTextFile    permissionOperation = "fs/write_text_file"
	permissionCreateTerminal   permissionOperation = "terminal/create"
	permissionRequestToolGrant permissionOperation = "session/request_permission"
)

type permissionDecision string

const (
	decisionAllow permissionDecision = "allow"
	decisionDeny  permissionDecision = "deny"
)

type permissionPolicy struct {
	mode aghconfig.PermissionMode
	root string
}

func newPermissionPolicy(mode aghconfig.PermissionMode, root string) (permissionPolicy, error) {
	effectiveMode := mode
	if effectiveMode == "" {
		effectiveMode = aghconfig.PermissionModeApproveReads
	}
	if err := effectiveMode.Validate("permissions.mode"); err != nil {
		return permissionPolicy{}, err
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return permissionPolicy{}, fmt.Errorf("acp: resolve permission root: %w", err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return permissionPolicy{}, fmt.Errorf("acp: evaluate permission root %q: %w", absRoot, err)
	}

	return permissionPolicy{
		mode: effectiveMode,
		root: filepath.Clean(resolvedRoot),
	}, nil
}

func (p permissionPolicy) authorize(op permissionOperation) error {
	if p.isAllowed(op) {
		return nil
	}
	return fmt.Errorf("%w: %s blocked by %s", ErrPermissionDenied, op, p.mode)
}

func (p permissionPolicy) isAllowed(op permissionOperation) bool {
	switch p.mode {
	case aghconfig.PermissionModeApproveAll:
		return true
	case aghconfig.PermissionModeApproveReads:
		return op == permissionReadTextFile
	case aghconfig.PermissionModeDenyAll:
		return false
	default:
		return false
	}
}

func (p permissionPolicy) resolvePath(requestPath string) (string, error) {
	target := strings.TrimSpace(requestPath)
	if target == "" {
		return "", errors.New("acp: request path is required")
	}

	if !filepath.IsAbs(target) {
		target = filepath.Join(p.root, target)
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("acp: resolve request path %q: %w", requestPath, err)
	}

	resolvedTarget, err := resolveExistingAwarePath(absTarget)
	if err != nil {
		return "", err
	}
	if !isWithinRoot(p.root, resolvedTarget) {
		return "", fmt.Errorf("%w: %s", ErrPathOutsideWorkspace, requestPath)
	}

	return resolvedTarget, nil
}

func (p permissionPolicy) resolvePathList(locations []acpsdk.ToolCallLocation) ([]string, error) {
	if len(locations) == 0 {
		return nil, nil
	}

	resolved := make([]string, 0, len(locations))
	for _, location := range locations {
		path, err := p.resolvePath(location.Path)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, path)
	}
	return resolved, nil
}

func resolveExistingAwarePath(target string) (string, error) {
	cleanTarget := filepath.Clean(target)
	if _, err := os.Stat(cleanTarget); err == nil {
		resolved, resolveErr := filepath.EvalSymlinks(cleanTarget)
		if resolveErr != nil {
			return "", fmt.Errorf("acp: evaluate request path %q: %w", cleanTarget, resolveErr)
		}
		return filepath.Clean(resolved), nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("acp: stat request path %q: %w", cleanTarget, err)
	}

	parent := filepath.Dir(cleanTarget)
	existingParent, err := firstExistingAncestor(parent)
	if err != nil {
		return "", err
	}
	resolvedAncestor, err := filepath.EvalSymlinks(existingParent)
	if err != nil {
		return "", fmt.Errorf("acp: evaluate ancestor %q: %w", existingParent, err)
	}
	relativeParent, err := filepath.Rel(existingParent, parent)
	if err != nil {
		return "", fmt.Errorf("acp: resolve ancestor relationship for %q: %w", cleanTarget, err)
	}
	resolvedParent := filepath.Join(resolvedAncestor, relativeParent)
	return filepath.Join(resolvedParent, filepath.Base(cleanTarget)), nil
}

func firstExistingAncestor(path string) (string, error) {
	current := filepath.Clean(path)
	for {
		if _, err := os.Stat(current); err == nil {
			return current, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("acp: stat ancestor %q: %w", current, err)
		}

		next := filepath.Dir(current)
		if next == current {
			return "", fmt.Errorf("acp: no existing ancestor for %q", path)
		}
		current = next
	}
}

func isWithinRoot(root, target string) bool {
	cleanRoot := filepath.Clean(root)
	cleanTarget := filepath.Clean(target)
	if cleanRoot == cleanTarget {
		return true
	}
	return strings.HasPrefix(cleanTarget, cleanRoot+string(os.PathSeparator))
}

func selectPermissionOutcome(options []acpsdk.PermissionOption, decision permissionDecision) acpsdk.RequestPermissionOutcome {
	var preferred []acpsdk.PermissionOptionKind
	switch decision {
	case decisionAllow:
		preferred = []acpsdk.PermissionOptionKind{
			acpsdk.PermissionOptionKindAllowOnce,
			acpsdk.PermissionOptionKindAllowAlways,
		}
	default:
		preferred = []acpsdk.PermissionOptionKind{
			acpsdk.PermissionOptionKindRejectOnce,
			acpsdk.PermissionOptionKindRejectAlways,
		}
	}

	for _, kind := range preferred {
		for _, option := range options {
			if option.Kind == kind {
				return acpsdk.NewRequestPermissionOutcomeSelected(option.OptionId)
			}
		}
	}

	return acpsdk.NewRequestPermissionOutcomeCancelled()
}
