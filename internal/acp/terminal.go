package acp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	exec "os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/kballard/go-shellquote"
	"golang.org/x/sys/execabs"

	"github.com/pedronauck/agh/internal/procutil"
	"github.com/pedronauck/agh/internal/toolruntime"
)

const (
	defaultTerminalOutputLimit = 64 * 1024
	networkCommandName         = "network"
)

type terminalManager struct {
	ctx      context.Context
	logger   *slog.Logger
	registry *toolruntime.Registry

	nextID atomic.Uint64

	mu        sync.RWMutex
	terminals map[string]*managedTerminal
}

type managedTerminal struct {
	id string

	cmd           *exec.Cmd
	processRecord *toolruntime.Handle
	outputLimit   int

	networkOwned   bool
	ownerSessionID string
	ownerTurnID    string

	mu         sync.RWMutex
	output     []byte
	truncated  bool
	exitStatus *acpsdk.TerminalExitStatus
	done       chan struct{}
}

type terminalOwnership struct {
	networkOwned   bool
	ownerSessionID string
	ownerTurnID    string
}

type terminalOutputWriter struct {
	terminal *managedTerminal
}

func (p *AgentProcess) handleCreateTerminal(
	ctx context.Context,
	request acpsdk.CreateTerminalRequest,
) (acpsdk.CreateTerminalResponse, error) {
	request, err := p.interceptCreateTerminalRequest(ctx, request)
	if err != nil {
		return acpsdk.CreateTerminalResponse{}, err
	}

	ownership := terminalOwnership{
		ownerSessionID: p.SessionID,
		ownerTurnID:    p.activeTurnID(),
	}
	if p.isNetworkTurn() {
		argv, err := terminalArgv(request)
		if err != nil {
			return acpsdk.CreateTerminalResponse{}, fmt.Errorf("%w: %s", ErrToolBlockedForNetworkTurn, err)
		}
		if !isAllowedNetworkTerminalArgv(argv) {
			return acpsdk.CreateTerminalResponse{}, ErrToolBlockedForNetworkTurn
		}
		ownership = terminalOwnership{
			networkOwned:   true,
			ownerSessionID: p.SessionID,
			ownerTurnID:    p.activeTurnID(),
		}
	}

	host := p.toolHostOrDefault()
	if localHost, ok := host.(*localToolHost); ok {
		return localHost.createTerminal(ctx, request, ownership)
	}
	response, err := host.CreateTerminal(ctx, request)
	if err != nil {
		return acpsdk.CreateTerminalResponse{}, err
	}
	p.recordTerminalOwnership(response.TerminalId, ownership)
	if err := p.registerExternalTerminalProcess(ctx, host, response.TerminalId, request, ownership); err != nil {
		p.deleteTerminalOwnership(response.TerminalId)
		if killErr := host.KillTerminal(response.TerminalId); killErr != nil {
			slog.Default().Warn(
				"acp: cleanup unregistered terminal",
				"terminal_id", response.TerminalId,
				"error", killErr,
			)
		}
		return acpsdk.CreateTerminalResponse{}, err
	}
	return response, nil
}

func (p *AgentProcess) recordTerminalOwnership(id string, ownership terminalOwnership) {
	if strings.TrimSpace(id) == "" || !ownership.networkOwned {
		return
	}

	p.terminalOwnershipMu.Lock()
	defer p.terminalOwnershipMu.Unlock()
	if p.terminalOwnership == nil {
		p.terminalOwnership = make(map[string]terminalOwnership)
	}
	p.terminalOwnership[id] = ownership
}

func (p *AgentProcess) deleteTerminalOwnership(id string) {
	if strings.TrimSpace(id) == "" {
		return
	}

	p.terminalOwnershipMu.Lock()
	defer p.terminalOwnershipMu.Unlock()
	if p.terminalOwnership == nil {
		return
	}
	delete(p.terminalOwnership, id)
}

func (p *AgentProcess) registerExternalTerminalProcess(
	ctx context.Context,
	host ToolHost,
	id string,
	request acpsdk.CreateTerminalRequest,
	ownership terminalOwnership,
) error {
	if p.processRegistry == nil || strings.TrimSpace(id) == "" {
		return nil
	}
	argv, err := terminalArgv(request)
	if err != nil {
		return err
	}
	registerCtx, cancel := withoutCancelPreservingDeadline(ctx)
	defer cancel()
	cwd := p.Cwd
	if request.Cwd != nil {
		cwd = *request.Cwd
	}
	var handle *toolruntime.Handle
	handle, err = p.processRegistry.Register(registerCtx, toolruntime.RegisterConfig{
		Source: toolruntime.ProcessSourceSandboxTerminal,
		Owner: toolruntime.ProcessOwner{
			SessionID:  ownership.ownerSessionID,
			TurnID:     ownership.ownerTurnID,
			TerminalID: id,
		},
		Command: argv[0],
		Args:    argv[1:],
		Cwd:     cwd,
		Interrupt: func(callbackCtx context.Context, _ toolruntime.ProcessRecord) error {
			if killErr := host.KillTerminal(id); killErr != nil {
				return killErr
			}
			if handle != nil {
				return handle.Complete(
					context.WithoutCancel(callbackCtx),
					toolruntime.ProcessCompletion{Err: errors.New("terminal interrupted")},
				)
			}
			return nil
		},
	})
	if err != nil {
		return fmt.Errorf("acp: register terminal process %q: %w", id, err)
	}

	p.terminalProcessMu.Lock()
	defer p.terminalProcessMu.Unlock()
	if p.terminalProcesses == nil {
		p.terminalProcesses = make(map[string]*toolruntime.Handle)
	}
	p.terminalProcesses[id] = handle
	return nil
}

func (p *AgentProcess) completeExternalTerminalProcess(
	ctx context.Context,
	id string,
	completion toolruntime.ProcessCompletion,
) {
	handle := p.takeExternalTerminalProcess(id)
	if handle == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := handle.Complete(ctx, completion); err != nil {
		slog.Default().Warn("acp: complete terminal process record", "terminal_id", id, "error", err)
	}
}

func (p *AgentProcess) takeExternalTerminalProcess(id string) *toolruntime.Handle {
	if strings.TrimSpace(id) == "" {
		return nil
	}
	p.terminalProcessMu.Lock()
	defer p.terminalProcessMu.Unlock()
	if p.terminalProcesses == nil {
		return nil
	}
	handle := p.terminalProcesses[id]
	delete(p.terminalProcesses, id)
	return handle
}

func (p *AgentProcess) handleKillTerminal(
	request acpsdk.KillTerminalRequest,
) (acpsdk.KillTerminalResponse, error) {
	if err := p.ensureNetworkTurnTerminalAccess(request.TerminalId, false); err != nil {
		return acpsdk.KillTerminalResponse{}, err
	}
	if err := p.toolHostOrDefault().KillTerminal(request.TerminalId); err != nil {
		return acpsdk.KillTerminalResponse{}, err
	}
	p.deleteTerminalOwnership(request.TerminalId)
	p.completeExternalTerminalProcess(
		context.Background(),
		request.TerminalId,
		toolruntime.ProcessCompletion{Err: errors.New("terminal killed")},
	)
	return acpsdk.KillTerminalResponse{}, nil
}

func (p *AgentProcess) handleTerminalOutput(
	request acpsdk.TerminalOutputRequest,
) (acpsdk.TerminalOutputResponse, error) {
	if err := p.ensureNetworkTurnTerminalAccess(request.TerminalId, true); err != nil {
		return acpsdk.TerminalOutputResponse{}, err
	}
	host := p.toolHostOrDefault()
	if localHost, ok := host.(*localToolHost); ok {
		output, truncated, exitStatus, err := localHost.terminalOutputStatus(request.TerminalId)
		if err != nil {
			return acpsdk.TerminalOutputResponse{}, err
		}
		return acpsdk.TerminalOutputResponse{
			Output:     output,
			Truncated:  truncated,
			ExitStatus: exitStatus,
		}, nil
	}
	output, err := host.TerminalOutput(request.TerminalId)
	if err != nil {
		return acpsdk.TerminalOutputResponse{}, err
	}
	return acpsdk.TerminalOutputResponse{
		Output: output,
	}, nil
}

func (p *AgentProcess) handleWaitForTerminalExit(
	ctx context.Context,
	request acpsdk.WaitForTerminalExitRequest,
) (acpsdk.WaitForTerminalExitResponse, error) {
	if err := p.ensureNetworkTurnTerminalAccess(request.TerminalId, true); err != nil {
		return acpsdk.WaitForTerminalExitResponse{}, err
	}
	host := p.toolHostOrDefault()
	if localHost, ok := host.(*localToolHost); ok {
		exitStatus, err := localHost.waitForTerminalExitStatus(ctx, request.TerminalId)
		if err != nil {
			return acpsdk.WaitForTerminalExitResponse{}, err
		}
		if exitStatus == nil {
			return acpsdk.WaitForTerminalExitResponse{}, nil
		}
		return acpsdk.WaitForTerminalExitResponse{
			ExitCode: exitStatus.ExitCode,
			Signal:   exitStatus.Signal,
		}, nil
	}
	exitCode, err := host.WaitForTerminalExit(ctx, request.TerminalId)
	if err != nil {
		return acpsdk.WaitForTerminalExitResponse{}, err
	}
	p.completeExternalTerminalProcess(
		context.Background(),
		request.TerminalId,
		toolruntime.ProcessCompletion{ExitCode: new(exitCode)},
	)
	return acpsdk.WaitForTerminalExitResponse{
		ExitCode: new(exitCode),
	}, nil
}

func (p *AgentProcess) handleReleaseTerminal(
	request acpsdk.ReleaseTerminalRequest,
) (acpsdk.ReleaseTerminalResponse, error) {
	if err := p.ensureNetworkTurnTerminalAccess(request.TerminalId, false); err != nil {
		return acpsdk.ReleaseTerminalResponse{}, err
	}
	if err := p.toolHostOrDefault().ReleaseTerminal(request.TerminalId); err != nil {
		return acpsdk.ReleaseTerminalResponse{}, err
	}
	p.deleteTerminalOwnership(request.TerminalId)
	p.completeExternalTerminalProcess(
		context.Background(),
		request.TerminalId,
		toolruntime.ProcessCompletion{Error: "terminal released"},
	)
	return acpsdk.ReleaseTerminalResponse{}, nil
}

func (p *AgentProcess) toolHostOrDefault() ToolHost {
	p.toolHostMu.Lock()
	defer p.toolHostMu.Unlock()

	if p.toolHost != nil {
		return p.toolHost
	}
	procCtx := p.processCtx
	if procCtx == nil {
		procCtx = context.Background()
	}
	host := newLocalToolHostFromPolicy(
		procCtx,
		p.Cwd,
		p.permissions,
		slog.Default(),
		WithLocalProcessRegistry(p.processRegistry),
	)
	if p.terminals != nil {
		host.terminals = p.terminals
	} else {
		p.terminals = host.terminals
	}
	p.toolHost = host
	return host
}

func newTerminalManager(
	ctx context.Context,
	logger *slog.Logger,
	registries ...*toolruntime.Registry,
) *terminalManager {
	var registry *toolruntime.Registry
	if len(registries) > 0 {
		registry = registries[0]
	}
	return &terminalManager{
		ctx:       ctx,
		logger:    logger,
		registry:  registry,
		terminals: make(map[string]*managedTerminal),
	}
}

func (m *terminalManager) create(
	ctx context.Context,
	cwd string,
	request acpsdk.CreateTerminalRequest,
	ownership terminalOwnership,
) (acpsdk.CreateTerminalResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	argv, err := terminalArgv(request)
	if err != nil {
		return acpsdk.CreateTerminalResponse{}, err
	}

	requestEnv := request.Env
	if ownership.networkOwned {
		requestEnv = nil
	}
	cmdEnv := mergeCommandEnv(procutil.FilteredDaemonEnv(os.Environ()), requestEnv)
	cmd, err := newManagedTerminalCommand(argv, cmdEnv)
	if err != nil {
		return acpsdk.CreateTerminalResponse{}, err
	}
	configureManagedCommand(cmd)
	cmd.Dir = cwd
	cmd.Env = cmdEnv
	outputLimit := defaultTerminalOutputLimit
	if request.OutputByteLimit != nil {
		outputLimit = max(*request.OutputByteLimit, 0)
	}

	term := &managedTerminal{
		id:             fmt.Sprintf("term-%d", m.nextID.Add(1)),
		cmd:            cmd,
		outputLimit:    outputLimit,
		networkOwned:   ownership.networkOwned,
		ownerSessionID: strings.TrimSpace(ownership.ownerSessionID),
		ownerTurnID:    strings.TrimSpace(ownership.ownerTurnID),
		done:           make(chan struct{}),
	}
	writer := &terminalOutputWriter{terminal: term}
	cmd.Stdout = writer
	cmd.Stderr = writer

	if err := cmd.Start(); err != nil {
		return acpsdk.CreateTerminalResponse{}, fmt.Errorf("acp: start terminal command %q: %w", argv[0], err)
	}
	if err := registerManagedCommand(cmd); err != nil {
		cleanupErr := cleanupStartedTerminalCommand(cmd)
		return acpsdk.CreateTerminalResponse{}, errors.Join(
			fmt.Errorf("acp: register terminal process tree for %q: %w", argv[0], err),
			cleanupErr,
		)
	}
	if err := m.registerTerminalProcess(ctx, term, cwd, argv); err != nil {
		return acpsdk.CreateTerminalResponse{}, errors.Join(err, cleanupStartedTerminalCommand(cmd))
	}
	if m.ctx != nil {
		watchTerminalShutdown(m.ctx, term.done, func() {
			term.checkpointInterrupting(context.Background(), "manager shutdown")
			if err := killManagedProcess(cmd); err != nil {
				m.logTerminalKillError(term.id, "manager shutdown", err)
			}
		})
	}

	m.mu.Lock()
	m.terminals[term.id] = term
	m.mu.Unlock()

	waitCtx := context.WithoutCancel(ctx)
	go term.wait(waitCtx)

	return acpsdk.CreateTerminalResponse{TerminalId: term.id}, nil
}

func cleanupStartedTerminalCommand(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	var errs []error
	if err := killManagedProcess(cmd); err != nil {
		errs = append(errs, fmt.Errorf("acp: kill terminal after start cleanup: %w", err))
	}
	if err := cmd.Wait(); err != nil {
		errs = append(errs, fmt.Errorf("acp: wait terminal after start cleanup: %w", err))
	}
	if err := forceManagedProcessGroupExit(cmd, 250*time.Millisecond); err != nil {
		errs = append(errs, fmt.Errorf("acp: wait for terminal process tree after start cleanup: %w", err))
	}
	return errors.Join(errs...)
}

func (m *terminalManager) registerTerminalProcess(
	ctx context.Context,
	term *managedTerminal,
	cwd string,
	argv []string,
) error {
	if m.registry == nil || term == nil || term.cmd == nil || term.cmd.Process == nil {
		return nil
	}
	registerCtx, cancel := withoutCancelPreservingDeadline(ctx)
	defer cancel()
	handle, err := m.registry.Register(registerCtx, toolruntime.RegisterConfig{
		Source: toolruntime.ProcessSourceACPTerminal,
		Owner: toolruntime.ProcessOwner{
			SessionID:  term.ownerSessionID,
			TurnID:     term.ownerTurnID,
			TerminalID: term.id,
		},
		PID:            term.cmd.Process.Pid,
		ProcessGroupID: term.cmd.Process.Pid,
		Command:        argv[0],
		Args:           argv[1:],
		Cwd:            cwd,
		Interrupt: func(_ context.Context, _ toolruntime.ProcessRecord) error {
			return terminateManagedProcess(term.cmd)
		},
	})
	if err != nil {
		return fmt.Errorf("acp: register terminal process %q: %w", term.id, err)
	}
	term.processRecord = handle
	return nil
}

func newManagedTerminalCommand(argv []string, env []string) (*exec.Cmd, error) {
	if len(argv) == 0 {
		return nil, errors.New("acp: terminal command is required")
	}

	resolvedPath, err := lookTerminalExecutable(argv[0], env)
	if err != nil {
		return nil, fmt.Errorf("acp: resolve terminal executable %q: %w", argv[0], err)
	}

	commandArgs := make([]string, 0, len(argv))
	commandArgs = append(commandArgs, resolvedPath)
	commandArgs = append(commandArgs, argv[1:]...)
	return &exec.Cmd{
		Path: resolvedPath,
		Args: commandArgs,
	}, nil
}

func lookTerminalExecutable(command string, env []string) (string, error) {
	if hasPathSeparator(command) {
		return execabs.LookPath(command)
	}
	pathValue, ok := envValueFromList(env, "PATH")
	if !ok {
		return execabs.LookPath(command)
	}
	for _, dir := range filepath.SplitList(pathValue) {
		if strings.TrimSpace(dir) == "" || !filepath.IsAbs(dir) {
			continue
		}
		for _, candidate := range terminalExecutableCandidates(filepath.Join(dir, command), env) {
			if isExecutableFile(candidate) {
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("%w: %s", exec.ErrNotFound, command)
}

func hasPathSeparator(command string) bool {
	if strings.ContainsRune(command, os.PathSeparator) {
		return true
	}
	return os.PathSeparator != '/' && strings.Contains(command, "/")
}

func terminalExecutableCandidates(path string, env []string) []string {
	if runtime.GOOS != "windows" || filepath.Ext(path) != "" {
		return []string{path}
	}
	pathExt, ok := envValueFromList(env, "PATHEXT")
	if !ok || strings.TrimSpace(pathExt) == "" {
		pathExt = ".COM;.EXE;.BAT;.CMD"
	}
	extensions := filepath.SplitList(pathExt)
	candidates := make([]string, 0, len(extensions)+1)
	for _, extension := range extensions {
		trimmed := strings.TrimSpace(extension)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, ".") {
			trimmed = "." + trimmed
		}
		candidates = append(candidates, path+trimmed)
	}
	candidates = append(candidates, path)
	return candidates
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode().Perm()&0o111 != 0
}

func envValueFromList(env []string, key string) (string, bool) {
	for index := len(env) - 1; index >= 0; index-- {
		name, value, ok := strings.Cut(env[index], "=")
		if !ok {
			continue
		}
		if runtime.GOOS == "windows" {
			if strings.EqualFold(name, key) {
				return value, true
			}
			continue
		}
		if name == key {
			return value, true
		}
	}
	return "", false
}

func (m *terminalManager) kill(id string) error {
	term, err := m.lookup(id)
	if err != nil {
		return err
	}
	term.checkpointInterrupting(context.Background(), "terminal killed")
	if err := killManagedProcess(term.cmd); err != nil {
		return fmt.Errorf("acp: kill terminal %q: %w", id, err)
	}
	return nil
}

func (m *terminalManager) output(id string) (string, bool, *acpsdk.TerminalExitStatus, error) {
	term, err := m.lookup(id)
	if err != nil {
		return "", false, nil, err
	}
	output, truncated, exitStatus := term.snapshot()
	return output, truncated, exitStatus, nil
}

func (m *terminalManager) wait(ctx context.Context, id string) (*acpsdk.TerminalExitStatus, error) {
	term, err := m.lookup(id)
	if err != nil {
		return nil, err
	}
	select {
	case <-term.done:
		_, _, exitStatus := term.snapshot()
		return exitStatus, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *terminalManager) release(id string) error {
	term, err := m.lookup(id)
	if err != nil {
		return err
	}
	term.checkpointInterrupting(context.Background(), "terminal released")
	if killErr := killManagedProcess(term.cmd); killErr != nil {
		m.logTerminalKillError(id, "release", killErr)
	}
	m.mu.Lock()
	delete(m.terminals, id)
	m.mu.Unlock()
	return nil
}

func (m *terminalManager) closeAll() {
	m.mu.RLock()
	terminals := make([]*managedTerminal, 0, len(m.terminals))
	for _, terminal := range m.terminals {
		terminals = append(terminals, terminal)
	}
	m.mu.RUnlock()

	for _, terminal := range terminals {
		terminal.checkpointInterrupting(context.Background(), "terminal manager closing")
		if err := killManagedProcess(terminal.cmd); err != nil {
			m.logTerminalKillError(terminal.id, "close_all", err)
		}
	}
}

func (m *terminalManager) logTerminalKillError(id string, reason string, err error) {
	if err == nil || m.logger == nil {
		return
	}
	m.logger.Warn("acp: kill terminal", "terminal_id", id, "reason", reason, "error", err)
}

func (m *terminalManager) lookup(id string) (*managedTerminal, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	term, ok := m.terminals[id]
	if !ok {
		return nil, fmt.Errorf("acp: terminal %q not found", id)
	}
	return term, nil
}

func (w *terminalOutputWriter) Write(p []byte) (int, error) {
	w.terminal.appendOutput(p)
	return len(p), nil
}

func (t *managedTerminal) appendOutput(p []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	var truncated bool
	t.output, truncated = appendTerminalOutputWindow(t.output, p, t.outputLimit)
	if truncated {
		t.truncated = true
	}
}

func withoutCancelPreservingDeadline(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		return context.Background(), func() {}
	}

	detached := context.WithoutCancel(ctx)
	deadline, ok := ctx.Deadline()
	if !ok {
		return detached, func() {}
	}
	return context.WithDeadline(detached, deadline)
}

func (t *managedTerminal) wait(ctx context.Context) {
	err := t.cmd.Wait()
	groupWaitErr := forceManagedProcessGroupExit(t.cmd, 250*time.Millisecond)
	exitStatus := &acpsdk.TerminalExitStatus{}
	if t.cmd.ProcessState != nil {
		exitCode := t.cmd.ProcessState.ExitCode()
		if exitCode >= 0 {
			exitStatus.ExitCode = new(exitCode)
		}
	}
	if err != nil && exitStatus.ExitCode == nil {
		signalText := err.Error()
		exitStatus.Signal = &signalText
	}
	if groupWaitErr != nil && exitStatus.Signal == nil {
		signalText := groupWaitErr.Error()
		exitStatus.Signal = &signalText
	}

	t.mu.Lock()
	t.exitStatus = exitStatus
	t.mu.Unlock()
	t.completeProcess(ctx, exitStatus, err, groupWaitErr)
	close(t.done)
}

func (t *managedTerminal) checkpointInterrupting(ctx context.Context, reason string) {
	if t == nil || t.processRecord == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := t.processRecord.Checkpoint(ctx, toolruntime.ProcessCheckpoint{
		State: toolruntime.ProcessStateInterrupting,
		Error: reason,
	}); err != nil {
		slog.Default().Warn("acp: checkpoint terminal process record", "terminal_id", t.id, "error", err)
	}
}

func (t *managedTerminal) completeProcess(
	ctx context.Context,
	exitStatus *acpsdk.TerminalExitStatus,
	waitErr error,
	groupWaitErr error,
) {
	if t == nil || t.processRecord == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	completion := toolruntime.ProcessCompletion{}
	if exitStatus != nil && exitStatus.ExitCode != nil {
		completion.ExitCode = exitStatus.ExitCode
	}
	if waitErr != nil {
		completion.Err = waitErr
	} else if groupWaitErr != nil {
		completion.Err = groupWaitErr
	}
	if err := t.processRecord.Complete(ctx, completion); err != nil {
		slog.Default().Warn("acp: complete terminal process record", "terminal_id", t.id, "error", err)
	}
}

func (t *managedTerminal) snapshot() (string, bool, *acpsdk.TerminalExitStatus) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	output := string(append([]byte(nil), t.output...))
	var exitStatus *acpsdk.TerminalExitStatus
	if t.exitStatus != nil {
		copyStatus := *t.exitStatus
		exitStatus = &copyStatus
	}
	return output, t.truncated, exitStatus
}

func terminalArgv(request acpsdk.CreateTerminalRequest) ([]string, error) {
	command := strings.TrimSpace(request.Command)
	if command == "" {
		return nil, errors.New("acp: terminal command is required")
	}

	argv, err := shellquote.Split(command)
	if err != nil {
		return nil, fmt.Errorf("acp: parse terminal command %q: %w", request.Command, err)
	}
	if len(argv) == 0 {
		return nil, errors.New("acp: terminal command is required")
	}
	return append(argv, request.Args...), nil
}

func isAllowedNetworkTerminalArgv(argv []string) bool {
	if len(argv) < 3 || argv[0] != "agh" || argv[1] != networkCommandName {
		return false
	}

	switch argv[2] {
	case "send", "peers", "channels", "status", "inbox", "threads", "directs", "work":
		return true
	default:
		return false
	}
}

func (p *AgentProcess) ensureNetworkTurnTerminalAccess(id string, requireSameTurn bool) error {
	if !p.isNetworkTurn() {
		return nil
	}

	ownership, err := p.lookupTerminalOwnership(id)
	if err != nil {
		return err
	}
	if !ownership.networkOwned {
		return ErrToolBlockedForNetworkTurn
	}
	if requireSameTurn && strings.TrimSpace(ownership.ownerTurnID) != p.activeTurnID() {
		return ErrToolBlockedForNetworkTurn
	}
	return nil
}

func (p *AgentProcess) lookupTerminalOwnership(id string) (terminalOwnership, error) {
	host := p.toolHostOrDefault()
	if localHost, ok := host.(*localToolHost); ok {
		return localHost.terminalOwnership(id)
	}

	p.terminalOwnershipMu.RLock()
	ownership, ok := p.terminalOwnership[id]
	p.terminalOwnershipMu.RUnlock()
	if !ok {
		return terminalOwnership{}, ErrToolBlockedForNetworkTurn
	}
	return ownership, nil
}

func mergeCommandEnv(base []string, variables []acpsdk.EnvVariable) []string {
	merged := make(map[string]string, len(base)+len(variables))
	order := make([]string, 0, len(base)+len(variables))

	for _, entry := range base {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		name := parts[0]
		if _, exists := merged[name]; !exists {
			order = append(order, name)
		}
		merged[name] = parts[1]
	}

	for _, variable := range variables {
		if _, exists := merged[variable.Name]; !exists {
			order = append(order, variable.Name)
		}
		merged[variable.Name] = variable.Value
	}

	result := make([]string, 0, len(order))
	for _, name := range order {
		result = append(result, name+"="+merged[name])
	}
	return result
}

func cloneNonEmptyStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return append([]string(nil), values...)
}

func cloneNonEmptyEnvSlice(values []acpsdk.EnvVariable) []acpsdk.EnvVariable {
	if len(values) == 0 {
		return nil
	}
	return append([]acpsdk.EnvVariable(nil), values...)
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func appendTerminalOutputWindow(dst []byte, src []byte, limit int) ([]byte, bool) {
	if limit <= 0 {
		return nil, len(dst) > 0 || len(src) > 0
	}
	if len(src) == 0 {
		if len(dst) <= limit {
			return dst, false
		}
		return trimUTF8LeadingBytes(dst[len(dst)-limit:]), true
	}
	if len(dst)+len(src) <= limit {
		return append(dst, src...), false
	}

	var out []byte
	if cap(dst) == limit {
		out = dst[:0]
	} else {
		out = make([]byte, 0, limit)
	}

	if len(src) >= limit {
		out = append(out, src[len(src)-limit:]...)
		return trimUTF8LeadingBytes(out), true
	}

	keepFromDst := limit - len(src)
	if len(dst) > keepFromDst {
		dst = dst[len(dst)-keepFromDst:]
	}
	out = append(out, dst...)
	out = append(out, src...)
	return trimUTF8LeadingBytes(out), true
}

func trimUTF8LeadingBytes(data []byte) []byte {
	trim := 0
	for trim < len(data) && !isValidUTF8LeadingByte(data[trim]) {
		trim++
	}
	if trim == 0 {
		return data
	}
	copy(data, data[trim:])
	return data[:len(data)-trim]
}

func isValidUTF8LeadingByte(b byte) bool {
	return b < utf8.RuneSelf || (b >= 0xC2 && b <= 0xF4)
}

func watchTerminalShutdown(ctx context.Context, terminalDone <-chan struct{}, onShutdown func()) <-chan struct{} {
	watcherDone := make(chan struct{})
	if ctx == nil {
		close(watcherDone)
		return watcherDone
	}

	go func() {
		defer close(watcherDone)
		select {
		case <-ctx.Done():
			if onShutdown != nil {
				onShutdown()
			}
		case <-terminalDone:
		}
	}()

	return watcherDone
}
