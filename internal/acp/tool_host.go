package acp

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	acpsdk "github.com/coder/acp-go-sdk"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/environment"
)

// ToolHost abstracts ACP file, permission, and terminal operations for a runtime.
type ToolHost = environment.ToolHost

type permissionOperation = environment.PermissionOperation

const (
	permissionReadTextFile     = environment.PermissionOperationReadTextFile
	permissionWriteTextFile    = environment.PermissionOperationWriteTextFile
	permissionCreateTerminal   = environment.PermissionOperationCreateTerminal
	permissionRequestToolGrant = environment.PermissionOperationRequestToolGrant
)

type permissionDecision = environment.PermissionDecision

const (
	decisionPending      = environment.PermissionDecisionPending
	decisionAllowOnce    = environment.PermissionDecisionAllowOnce
	decisionAllowAlways  = environment.PermissionDecisionAllowAlways
	decisionRejectOnce   = environment.PermissionDecisionRejectOnce
	decisionRejectAlways = environment.PermissionDecisionRejectAlways
)

var _ environment.ToolHost = (*localToolHost)(nil)

type localToolHost struct {
	cwd         string
	permissions permissionPolicy
	terminals   *terminalManager
}

// NewLocalToolHost returns the local daemon-host file, permission, and terminal host.
func NewLocalToolHost(
	ctx context.Context,
	root string,
	mode aghconfig.PermissionMode,
	logger *slog.Logger,
) (environment.ToolHost, error) {
	return newLocalToolHost(ctx, root, mode, logger)
}

func newLocalToolHost(
	ctx context.Context,
	root string,
	mode aghconfig.PermissionMode,
	logger *slog.Logger,
) (*localToolHost, error) {
	policy, err := newPermissionPolicy(mode, root)
	if err != nil {
		return nil, err
	}
	return newLocalToolHostFromPolicy(ctx, root, policy, logger), nil
}

func newLocalToolHostFromPolicy(
	ctx context.Context,
	root string,
	policy permissionPolicy,
	logger *slog.Logger,
) *localToolHost {
	if ctx == nil {
		ctx = context.Background()
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &localToolHost{
		cwd:         root,
		permissions: policy,
		terminals:   newTerminalManager(ctx, logger),
	}
}

func (h *localToolHost) ReadTextFile(_ context.Context, path string) (string, error) {
	if err := h.Authorize(permissionReadTextFile); err != nil {
		return "", err
	}
	resolvedPath, err := h.ResolvePath(path)
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("acp: read %q: %w", resolvedPath, err)
	}
	return string(content), nil
}

func (h *localToolHost) WriteTextFile(_ context.Context, path string, content string) error {
	if err := h.Authorize(permissionWriteTextFile); err != nil {
		return err
	}
	resolvedPath, err := h.ResolvePath(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o755); err != nil {
		return fmt.Errorf("acp: create parent directories for %q: %w", resolvedPath, err)
	}
	if err := os.WriteFile(resolvedPath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("acp: write %q: %w", resolvedPath, err)
	}
	return nil
}

func (h *localToolHost) ResolvePath(path string) (string, error) {
	return h.permissions.resolvePath(path)
}

func (h *localToolHost) Authorize(op environment.PermissionOperation) error {
	return h.permissions.authorize(op)
}

func (h *localToolHost) PermissionDecision(
	req acpsdk.RequestPermissionRequest,
) (environment.PermissionDecision, bool) {
	return h.permissions.permissionDecision(req)
}

func (h *localToolHost) CreateTerminal(
	ctx context.Context,
	req acpsdk.CreateTerminalRequest,
) (acpsdk.CreateTerminalResponse, error) {
	return h.createTerminal(ctx, req, terminalOwnership{})
}

func (h *localToolHost) createTerminal(
	_ context.Context,
	req acpsdk.CreateTerminalRequest,
	ownership terminalOwnership,
) (acpsdk.CreateTerminalResponse, error) {
	if err := h.Authorize(permissionCreateTerminal); err != nil {
		return acpsdk.CreateTerminalResponse{}, err
	}
	cwd := h.cwd
	if req.Cwd != nil {
		cwd = *req.Cwd
	}
	resolvedCwd, err := h.ResolvePath(cwd)
	if err != nil {
		return acpsdk.CreateTerminalResponse{}, err
	}
	return h.terminals.create(resolvedCwd, req, ownership)
}

func (h *localToolHost) KillTerminal(id string) error {
	return h.terminals.kill(id)
}

func (h *localToolHost) TerminalOutput(id string) (string, error) {
	output, _, _, err := h.terminalOutputStatus(id)
	return output, err
}

func (h *localToolHost) terminalOutputStatus(
	id string,
) (string, bool, *acpsdk.TerminalExitStatus, error) {
	return h.terminals.output(id)
}

func (h *localToolHost) WaitForTerminalExit(ctx context.Context, id string) (int, error) {
	exitStatus, err := h.waitForTerminalExitStatus(ctx, id)
	if err != nil {
		return 0, err
	}
	if exitStatus == nil || exitStatus.ExitCode == nil {
		return 0, nil
	}
	return *exitStatus.ExitCode, nil
}

func (h *localToolHost) waitForTerminalExitStatus(
	ctx context.Context,
	id string,
) (*acpsdk.TerminalExitStatus, error) {
	return h.terminals.wait(ctx, id)
}

func (h *localToolHost) ReleaseTerminal(id string) error {
	return h.terminals.release(id)
}

func (h *localToolHost) terminalOwnership(id string) (terminalOwnership, error) {
	term, err := h.terminals.lookup(id)
	if err != nil {
		return terminalOwnership{}, err
	}
	return terminalOwnership{
		networkOwned: term.networkOwned,
		ownerTurnID:  term.ownerTurnID,
	}, nil
}

func (h *localToolHost) Close() {
	if h == nil || h.terminals == nil {
		return
	}
	h.terminals.closeAll()
}
