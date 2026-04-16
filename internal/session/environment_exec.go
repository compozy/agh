package session

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
)

const defaultEnvironmentExecTimeout = 30 * time.Second

// EnvironmentExecRequest describes one command execution inside a session environment.
type EnvironmentExecRequest struct {
	SessionID string
	Command   string
	Timeout   time.Duration
}

// EnvironmentExecResult reports the terminal execution result.
type EnvironmentExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// ExecEnvironment runs a command through the active session's environment tool host.
func (m *Manager) ExecEnvironment(ctx context.Context, req EnvironmentExecRequest) (EnvironmentExecResult, error) {
	if ctx == nil {
		return EnvironmentExecResult{}, errors.New("session: environment exec context is required")
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return EnvironmentExecResult{}, errors.New("session: environment exec session id is required")
	}
	command := strings.TrimSpace(req.Command)
	if command == "" {
		return EnvironmentExecResult{}, errors.New("session: environment exec command is required")
	}

	sess, ok := m.Get(sessionID)
	if !ok {
		return EnvironmentExecResult{}, fmt.Errorf("%w: %s", ErrSessionNotFound, sessionID)
	}
	info := sess.Info()
	if info == nil || info.State != StateActive {
		return EnvironmentExecResult{}, fmt.Errorf("%w: %s", ErrSessionNotActive, sessionID)
	}
	if info.Environment == nil {
		return EnvironmentExecResult{}, errors.New("session: environment is not configured")
	}
	process := sess.processHandle()
	if process == nil {
		return EnvironmentExecResult{}, errors.New("session: agent process is not available")
	}
	toolHost := process.ToolHost()
	if toolHost == nil {
		return EnvironmentExecResult{}, errors.New("session: environment tool host is not available")
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = defaultEnvironmentExecTimeout
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cwd := strings.TrimSpace(info.Environment.RuntimeRootDir)
	createReq := acpsdk.CreateTerminalRequest{Command: command}
	if cwd != "" {
		createReq.Cwd = &cwd
	}
	terminal, err := toolHost.CreateTerminal(execCtx, createReq)
	if err != nil {
		return EnvironmentExecResult{}, fmt.Errorf("session: environment exec create terminal: %w", err)
	}

	exitCode, waitErr := toolHost.WaitForTerminalExit(execCtx, terminal.TerminalId)
	output, outputErr := toolHost.TerminalOutput(terminal.TerminalId)
	releaseErr := toolHost.ReleaseTerminal(terminal.TerminalId)

	result := EnvironmentExecResult{ExitCode: exitCode, Stdout: output}
	if waitErr != nil {
		result.Stderr = waitErr.Error()
	}
	if outputErr != nil {
		return result, fmt.Errorf("session: environment exec read output: %w", outputErr)
	}
	if releaseErr != nil {
		return result, fmt.Errorf("session: environment exec release terminal: %w", releaseErr)
	}
	if waitErr != nil && execCtx.Err() != nil {
		return result, fmt.Errorf("session: environment exec wait: %w", waitErr)
	}
	return result, nil
}
