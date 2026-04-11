package acp

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

	"github.com/kballard/go-shellquote"

	acpsdk "github.com/coder/acp-go-sdk"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/subprocess"
)

const (
	defaultStopTimeout    = 5 * time.Second
	defaultPromptBufSize  = 128
	defaultPromptDrain    = 50 * time.Millisecond
	defaultPermissionWait = 5 * time.Minute
	defaultClientName     = "agh"
	defaultClientVersion  = "dev"
)

var (
	// ErrAgentDoesNotSupportSession reports that resume was requested for an ACP agent without session/load support.
	ErrAgentDoesNotSupportSession = errors.New("acp: agent does not support session/load")
	// ErrLoadSessionFailed reports that ACP session/load failed during resume.
	ErrLoadSessionFailed = errors.New("acp: load session failed")
)

const requestErrorResourceNotFoundCode = -32002

// Option customizes the ACP driver.
type Option func(*Driver)

// Driver launches ACP agent subprocesses and brokers JSON-RPC traffic.
type Driver struct {
	logger          *slog.Logger
	stopTimeout     time.Duration
	promptBufferCap int
	promptDrainWait time.Duration
	permissionWait  time.Duration
}

// WithLogger directs driver diagnostics to the provided logger.
func WithLogger(logger *slog.Logger) Option {
	return func(driver *Driver) {
		driver.logger = logger
	}
}

// WithStopTimeout overrides how long Stop waits before escalating to SIGKILL.
func WithStopTimeout(timeout time.Duration) Option {
	return func(driver *Driver) {
		driver.stopTimeout = timeout
	}
}

// WithPromptBufferSize overrides the per-prompt event buffer size.
func WithPromptBufferSize(size int) Option {
	return func(driver *Driver) {
		driver.promptBufferCap = size
	}
}

// WithPromptDrainWait overrides how long Prompt waits for trailing asynchronous updates.
func WithPromptDrainWait(wait time.Duration) Option {
	return func(driver *Driver) {
		driver.promptDrainWait = wait
	}
}

// WithPermissionTimeout overrides how long an interactive permission request waits for approval.
func WithPermissionTimeout(timeout time.Duration) Option {
	return func(driver *Driver) {
		driver.permissionWait = timeout
	}
}

// New constructs an ACP driver with sensible defaults.
func New(opts ...Option) *Driver {
	driver := &Driver{
		logger:          slog.Default(),
		stopTimeout:     defaultStopTimeout,
		promptBufferCap: defaultPromptBufSize,
		promptDrainWait: defaultPromptDrain,
		permissionWait:  defaultPermissionWait,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(driver)
		}
	}
	if driver.logger == nil {
		driver.logger = slog.Default()
	}
	if driver.stopTimeout <= 0 {
		driver.stopTimeout = defaultStopTimeout
	}
	if driver.promptBufferCap <= 0 {
		driver.promptBufferCap = defaultPromptBufSize
	}
	if driver.promptDrainWait <= 0 {
		driver.promptDrainWait = defaultPromptDrain
	}
	if driver.permissionWait <= 0 {
		driver.permissionWait = defaultPermissionWait
	}
	return driver
}

// Start launches a subprocess, completes ACP initialization, and creates or resumes a session.
func (d *Driver) Start(ctx context.Context, opts StartOpts) (*AgentProcess, error) {
	if ctx == nil {
		return nil, errors.New("acp: context is required")
	}

	normalized, err := normalizeStartOpts(opts)
	if err != nil {
		return nil, err
	}

	process, err := d.spawnProcess(normalized)
	if err != nil {
		return nil, err
	}

	if err := d.initializeConnection(ctx, process, normalized.AgentName); err != nil {
		return nil, d.cleanupFailedStart(process, err)
	}
	if err := d.negotiateSession(ctx, process, normalized); err != nil {
		return nil, d.cleanupFailedStart(process, err)
	}
	return process, nil
}

func (d *Driver) spawnProcess(normalized StartOpts) (*AgentProcess, error) {
	command, args, err := parseCommandString(normalized.Command)
	if err != nil {
		return nil, err
	}

	policy, err := newPermissionPolicy(normalized.Permissions, normalized.Cwd)
	if err != nil {
		return nil, err
	}

	managed, err := subprocess.Launch(context.Background(), subprocess.LaunchConfig{
		Command:          command,
		Args:             args,
		Dir:              normalized.Cwd,
		Env:              normalized.Env,
		Logger:           d.logger,
		DisableTransport: true,
		ShutdownTimeout:  d.stopTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("acp: start subprocess %q: %w", normalized.Command, err)
	}
	procCtx, cancelProcess := context.WithCancel(context.Background())

	process := &AgentProcess{
		PID:                managed.PID(),
		AgentName:          normalized.AgentName,
		Command:            command,
		Args:               append([]string(nil), args...),
		Cwd:                normalized.Cwd,
		StartedAt:          timeNowUTC(),
		managed:            managed,
		cancelProcess:      cancelProcess,
		permissions:        policy,
		done:               make(chan struct{}),
		pendingPermissions: make(map[string]*pendingPermission),
		permissionTimeout:  d.permissionWait,
		systemPrompt:       normalized.SystemPrompt,
	}
	process.terminals = newTerminalManager(procCtx, d.logger)
	process.conn = acpsdk.NewConnection(process.handleInbound, managed.Stdin(), managed.Stdout())
	process.conn.SetLogger(d.logger)

	go process.waitForExit()

	return process, nil
}

func (d *Driver) initializeConnection(ctx context.Context, process *AgentProcess, agentName string) error {
	initRequest := acpsdk.InitializeRequest{
		ProtocolVersion: acpsdk.ProtocolVersionNumber,
		ClientCapabilities: acpsdk.ClientCapabilities{
			Fs: acpsdk.FileSystemCapability{
				ReadTextFile:  true,
				WriteTextFile: true,
			},
			Terminal: true,
		},
		ClientInfo: &acpsdk.Implementation{
			Name:    defaultClientName,
			Version: defaultClientVersion,
		},
	}
	initializeResponse, err := acpsdk.SendRequest[acpsdk.InitializeResponse](process.conn, ctx, acpsdk.AgentMethodInitialize, initRequest)
	if err != nil {
		return fmt.Errorf("acp: initialize session for %q: %w", agentName, err)
	}

	process.Caps = ACPCaps{
		SupportsLoadSession: initializeResponse.AgentCapabilities.LoadSession,
	}
	return nil
}

func (d *Driver) negotiateSession(ctx context.Context, process *AgentProcess, normalized StartOpts) error {
	if normalized.ResumeSessionID != "" {
		return d.loadSession(ctx, process, normalized)
	}
	return d.createSession(ctx, process, normalized)
}

func (d *Driver) loadSession(ctx context.Context, process *AgentProcess, normalized StartOpts) error {
	if !process.Caps.SupportsLoadSession {
		return fmt.Errorf("%w: agent %q does not support session/load for resume %q", ErrAgentDoesNotSupportSession, normalized.AgentName, normalized.ResumeSessionID)
	}

	loadRequest := acpsdk.LoadSessionRequest{
		Cwd:        normalized.Cwd,
		McpServers: toSDKMCPServers(normalized.MCPServers),
		SessionId:  acpsdk.SessionId(normalized.ResumeSessionID),
	}
	loadWireRequest := wireLoadSessionRequest{
		Cwd:            loadRequest.Cwd,
		McpServers:     loadRequest.McpServers,
		AdditionalDirs: append([]string(nil), normalized.AdditionalDirs...),
		SessionID:      loadRequest.SessionId,
	}
	loadResponse, err := acpsdk.SendRequest[acpsdk.LoadSessionResponse](process.conn, ctx, acpsdk.AgentMethodSessionLoad, loadWireRequest)
	if err != nil {
		return fmt.Errorf("%w: load session %q for %q: %w", ErrLoadSessionFailed, normalized.ResumeSessionID, normalized.AgentName, err)
	}

	process.SessionID = normalized.ResumeSessionID
	process.Caps = captureCaps(process.Caps.SupportsLoadSession, loadResponse.Modes, loadResponse.Models)
	return nil
}

func (d *Driver) createSession(ctx context.Context, process *AgentProcess, normalized StartOpts) error {
	newRequest := acpsdk.NewSessionRequest{
		Cwd:        normalized.Cwd,
		McpServers: toSDKMCPServers(normalized.MCPServers),
	}
	newWireRequest := wireNewSessionRequest{
		Cwd:            newRequest.Cwd,
		McpServers:     newRequest.McpServers,
		AdditionalDirs: append([]string(nil), normalized.AdditionalDirs...),
	}
	newResponse, err := acpsdk.SendRequest[acpsdk.NewSessionResponse](process.conn, ctx, acpsdk.AgentMethodSessionNew, newWireRequest)
	if err != nil {
		return fmt.Errorf("acp: create session for %q: %w", normalized.AgentName, err)
	}

	process.SessionID = string(newResponse.SessionId)
	process.Caps = captureCaps(process.Caps.SupportsLoadSession, newResponse.Modes, newResponse.Models)
	return nil
}

func (d *Driver) cleanupFailedStart(process *AgentProcess, startErr error) error {
	if startErr == nil || process == nil {
		return startErr
	}
	if stopErr := d.Stop(context.Background(), process); stopErr != nil {
		return errors.Join(startErr, fmt.Errorf("acp: stop failed while cleaning up failed start: %w", stopErr))
	}
	return startErr
}

// IsLoadSessionResourceMissing reports whether a resume failed because the
// upstream ACP implementation no longer knows the referenced session id.
func IsLoadSessionResourceMissing(err error) bool {
	if !errors.Is(err, ErrLoadSessionFailed) {
		return false
	}

	var reqErr *acpsdk.RequestError
	if !errors.As(err, &reqErr) {
		return false
	}

	return reqErr.Code == requestErrorResourceNotFoundCode &&
		strings.Contains(strings.ToLower(strings.TrimSpace(reqErr.Message)), "resource not found")
}

// Prompt starts one prompt turn and returns the streamed event channel.
func (d *Driver) Prompt(ctx context.Context, proc *AgentProcess, req PromptRequest) (<-chan AgentEvent, error) {
	if ctx == nil {
		return nil, errors.New("acp: context is required")
	}
	if proc == nil {
		return nil, errors.New("acp: agent process is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	active, err := proc.beginPrompt(req.TurnID, d.promptBufferCap)
	if err != nil {
		return nil, err
	}

	go d.runPrompt(ctx, proc, active, req)
	return active.events, nil
}

// Cancel sends the ACP cooperative cancellation notification for the active session.
func (d *Driver) Cancel(ctx context.Context, proc *AgentProcess) error {
	if ctx == nil {
		return errors.New("acp: context is required")
	}
	if proc == nil {
		return errors.New("acp: agent process is required")
	}
	if strings.TrimSpace(proc.SessionID) == "" {
		return errors.New("acp: session id is required")
	}
	return proc.conn.SendNotification(ctx, acpsdk.AgentMethodSessionCancel, acpsdk.CancelNotification{
		SessionId: acpsdk.SessionId(proc.SessionID),
	})
}

// ApprovePermission resolves a pending interactive permission request for the process.
func (d *Driver) ApprovePermission(ctx context.Context, proc *AgentProcess, req ApproveRequest) error {
	if ctx == nil {
		return errors.New("acp: context is required")
	}
	if proc == nil {
		return errors.New("acp: agent process is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return proc.ResolvePermission(req)
}

// Stop terminates the subprocess and waits for it to exit.
func (d *Driver) Stop(ctx context.Context, proc *AgentProcess) error {
	if ctx == nil {
		return errors.New("acp: context is required")
	}
	if proc == nil {
		return errors.New("acp: agent process is required")
	}

	select {
	case <-proc.Done():
		return proc.Wait()
	default:
	}

	proc.markStopRequested()
	var errs []error
	if strings.TrimSpace(proc.SessionID) != "" {
		cancelCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		_ = d.Cancel(cancelCtx, proc)
		cancel()
	}
	if proc.managed != nil {
		if err := proc.managed.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
		select {
		case <-proc.Done():
			return errors.Join(append(errs, proc.Wait())...)
		case <-ctx.Done():
			return errors.Join(append(errs, ctx.Err())...)
		}
	}

	if err := terminateManagedProcess(proc.cmd); err != nil {
		errs = append(errs, fmt.Errorf("acp: terminate subprocess tree: %w", err))
	}
	if proc.cancelProcess != nil {
		proc.cancelProcess()
	}

	waitCtx, cancelWait := context.WithTimeout(context.Background(), d.stopTimeout)
	defer cancelWait()

	select {
	case <-proc.Done():
	case <-waitCtx.Done():
		if err := killManagedProcess(proc.cmd); err != nil {
			errs = append(errs, fmt.Errorf("acp: kill subprocess tree: %w", err))
		}
		select {
		case <-proc.Done():
		case <-ctx.Done():
			return errors.Join(append(errs, ctx.Err())...)
		}
	case <-ctx.Done():
		return errors.Join(append(errs, ctx.Err())...)
	}

	return errors.Join(append(errs, proc.Wait())...)
}

func (d *Driver) runPrompt(ctx context.Context, proc *AgentProcess, active *activePromptState, req PromptRequest) {
	defer proc.endPrompt(active)

	cancellationDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			if strings.TrimSpace(proc.SessionID) != "" {
				_ = proc.conn.SendNotification(context.Background(), acpsdk.AgentMethodSessionCancel, acpsdk.CancelNotification{
					SessionId: acpsdk.SessionId(proc.SessionID),
				})
			}
		case <-cancellationDone:
		}
	}()

	promptRequest := acpsdk.PromptRequest{
		SessionId: acpsdk.SessionId(proc.SessionID),
		Prompt:    []acpsdk.ContentBlock{acpsdk.TextBlock(proc.nextPromptText(req.Message))},
	}
	response, err := acpsdk.SendRequest[wirePromptResponse](proc.conn, ctx, acpsdk.AgentMethodSessionPrompt, promptRequest)
	close(cancellationDone)

	if err != nil {
		event := AgentEvent{
			Type:      EventTypeError,
			SessionID: proc.SessionID,
			TurnID:    req.TurnID,
			Timestamp: timeNowUTC(),
			Error:     err.Error(),
		}
		proc.emitPromptEvent(event)
		return
	}

	usage := proc.mergePromptUsage(tokenUsageFromPromptResponse(req.TurnID, response.Usage))
	doneEvent := AgentEvent{
		Type:       EventTypeDone,
		SessionID:  proc.SessionID,
		TurnID:     req.TurnID,
		Timestamp:  timeNowUTC(),
		StopReason: string(response.StopReason),
	}
	if !usage.IsZero() {
		doneEvent.Usage = &usage
	}
	d.waitForPromptQuiescence(active)
	proc.emitPromptEvent(doneEvent)
}

func (p *AgentProcess) waitForExit() {
	var waitErr error
	switch {
	case p.managed != nil:
		waitErr = p.managed.Wait()
	case p.cmd != nil:
		waitErr = p.cmd.Wait()
	default:
		waitErr = nil
	}
	if p.stopWasRequested() {
		waitErr = nil
	} else if waitErr != nil {
		waitErr = fmt.Errorf("acp: subprocess exited: %w", attachStderr(waitErr, p.Stderr()))
	}
	p.setWaitError(waitErr)
	if p.cancelProcess != nil {
		p.cancelProcess()
	}
	if p.terminals != nil {
		p.terminals.closeAll()
	}
	close(p.done)
}

func normalizeStartOpts(opts StartOpts) (StartOpts, error) {
	if err := opts.Validate(); err != nil {
		return StartOpts{}, err
	}

	cwd, err := normalizeWorkspaceDir(opts.Cwd, "cwd")
	if err != nil {
		return StartOpts{}, err
	}

	normalized := opts
	normalized.Cwd = cwd
	additionalDirs, err := normalizeAdditionalDirs(cwd, opts.AdditionalDirs)
	if err != nil {
		return StartOpts{}, err
	}
	normalized.AdditionalDirs = additionalDirs
	if normalized.Permissions == "" {
		normalized.Permissions = aghconfig.PermissionModeApproveReads
	}
	if normalized.AdditionalDirs != nil {
		normalized.AdditionalDirs = append([]string(nil), normalized.AdditionalDirs...)
	}
	if normalized.Env != nil {
		normalized.Env = append([]string(nil), normalized.Env...)
	}
	if normalized.MCPServers != nil {
		normalized.MCPServers = append([]aghconfig.MCPServer(nil), normalized.MCPServers...)
	}
	normalized.SystemPrompt = strings.TrimSpace(normalized.SystemPrompt)

	return normalized, nil
}

func normalizeWorkspaceDir(path string, field string) (string, error) {
	target := strings.TrimSpace(path)
	absPath, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("acp: resolve %s %q: %w", field, path, err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("acp: stat %s %q: %w", field, absPath, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("acp: %s %q is not a directory", field, absPath)
	}
	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", fmt.Errorf("acp: evaluate %s %q: %w", field, absPath, err)
	}
	canonicalPath, err := filepath.Abs(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("acp: resolve canonical %s %q: %w", field, resolvedPath, err)
	}
	return filepath.Clean(canonicalPath), nil
}

func normalizeAdditionalDirs(rootDir string, dirs []string) ([]string, error) {
	if len(dirs) == 0 {
		return nil, nil
	}

	normalized := make([]string, 0, len(dirs))
	seen := make(map[string]struct{}, len(dirs))

	for i, dir := range dirs {
		trimmed := strings.TrimSpace(dir)
		if trimmed == "" {
			continue
		}

		canonicalDir, err := normalizeWorkspaceDir(trimmed, fmt.Sprintf("additional_dirs[%d]", i))
		if err != nil {
			return nil, err
		}
		if canonicalDir == rootDir {
			continue
		}
		if _, ok := seen[canonicalDir]; ok {
			continue
		}

		seen[canonicalDir] = struct{}{}
		normalized = append(normalized, canonicalDir)
	}

	return normalized, nil
}

func parseCommandString(command string) (string, []string, error) {
	parts, err := shellquote.Split(command)
	if err != nil {
		return "", nil, fmt.Errorf("acp: parse command %q: %w", command, err)
	}
	if len(parts) == 0 {
		return "", nil, errors.New("acp: command is empty")
	}
	return parts[0], parts[1:], nil
}

func toSDKMCPServers(servers []aghconfig.MCPServer) []acpsdk.McpServer {
	if len(servers) == 0 {
		return []acpsdk.McpServer{}
	}

	converted := make([]acpsdk.McpServer, 0, len(servers))
	for _, server := range servers {
		envKeys := make([]string, 0, len(server.Env))
		for key := range server.Env {
			envKeys = append(envKeys, key)
		}
		sort.Strings(envKeys)

		env := make([]acpsdk.EnvVariable, 0, len(server.Env))
		for _, key := range envKeys {
			env = append(env, acpsdk.EnvVariable{Name: key, Value: server.Env[key]})
		}

		converted = append(converted, acpsdk.McpServer{
			Stdio: &acpsdk.McpServerStdio{
				Name:    server.Name,
				Command: server.Command,
				Args:    append([]string(nil), server.Args...),
				Env:     env,
			},
		})
	}
	return converted
}

func captureCaps(loadSession bool, modes *acpsdk.SessionModeState, models *acpsdk.SessionModelState) ACPCaps {
	caps := ACPCaps{SupportsLoadSession: loadSession}
	if modes != nil {
		caps.SupportedModes = make([]string, 0, len(modes.AvailableModes))
		for _, mode := range modes.AvailableModes {
			caps.SupportedModes = append(caps.SupportedModes, string(mode.Id))
		}
	}
	if models != nil {
		caps.SupportedModels = make([]string, 0, len(models.AvailableModels))
		for _, model := range models.AvailableModels {
			caps.SupportedModels = append(caps.SupportedModels, string(model.ModelId))
		}
	}
	return caps
}

func attachStderr(err error, stderr string) error {
	if strings.TrimSpace(stderr) == "" {
		return err
	}
	return fmt.Errorf("%w: stderr=%s", err, strings.TrimSpace(stderr))
}

func (d *Driver) waitForPromptQuiescence(active *activePromptState) {
	if active == nil || d.promptDrainWait <= 0 {
		return
	}
	timer := time.NewTimer(d.promptDrainWait)
	maxTimer := time.NewTimer(2 * d.promptDrainWait)
	defer timer.Stop()
	defer maxTimer.Stop()

	for {
		select {
		case <-active.activity:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(d.promptDrainWait)
		case <-timer.C:
			return
		case <-maxTimer.C:
			return
		}
	}
}
