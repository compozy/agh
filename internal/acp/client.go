package acp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/kballard/go-shellquote"

	acpsdk "github.com/coder/acp-go-sdk"
	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/sandbox"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/toolruntime"
)

const (
	clientAgentWaitingKey = "agent_waiting"
	clientDefaultKey      = "default"
)

const (
	defaultStopTimeout          = 5 * time.Second
	defaultPromptBufSize        = 128
	defaultPromptDrain          = 50 * time.Millisecond
	defaultPermissionWait       = 5 * time.Minute
	defaultProcessRecordTimeout = time.Second
	defaultClientName           = "agh"
	defaultClientVersion        = "dev"
)

var (
	// ErrAgentDoesNotSupportSession reports that resume was requested for an ACP agent without session/load support.
	ErrAgentDoesNotSupportSession = errors.New("acp: agent does not support session/load")
	// ErrLoadSessionFailed reports that ACP session/load failed during resume.
	ErrLoadSessionFailed = errors.New("acp: load session failed")
	// errModelConfigOptionRequired reports that a requested model needs an advertised ACP model option.
	errModelConfigOptionRequired = errors.New("acp: model config option required")
	// errProcessConnectionUninitialized reports that the driver received a process without an ACP connection.
	errProcessConnectionUninitialized = errors.New("acp: process connection is not initialized")
	// errProcessLifecycleUninitialized reports that the driver received a process without a managed lifecycle.
	errProcessLifecycleUninitialized = errors.New("acp: process lifecycle is not initialized")
)

const requestErrorResourceNotFoundCode = -32002

// Option customizes the ACP driver.
type Option func(*Driver)

// Driver launches ACP agent subprocesses and brokers JSON-RPC traffic.
type Driver struct {
	logger               *slog.Logger
	stopTimeout          time.Duration
	promptBufferCap      int
	promptDrainWait      time.Duration
	permissionWait       time.Duration
	processRecordTimeout time.Duration
	launcher             sandbox.Launcher
	toolHost             sandbox.ToolHost
	processRegistry      *toolruntime.Registry
	steerSource          SteerSource
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

// WithLauncher overrides the sandbox launcher used by default for new ACP sessions.
func WithLauncher(launcher sandbox.Launcher) Option {
	return func(driver *Driver) {
		driver.launcher = launcher
	}
}

// WithToolHost overrides the sandbox tool host used by default for new ACP sessions.
func WithToolHost(toolHost sandbox.ToolHost) Option {
	return func(driver *Driver) {
		driver.toolHost = toolHost
	}
}

// WithProcessRegistry injects shared tool process tracking and scoped interrupts.
func WithProcessRegistry(registry *toolruntime.Registry) Option {
	return func(driver *Driver) {
		driver.processRegistry = registry
	}
}

// WithProcessRecordTimeout bounds process registry writes for ACP subprocesses.
func WithProcessRecordTimeout(timeout time.Duration) Option {
	return func(driver *Driver) {
		driver.processRecordTimeout = timeout
	}
}

// WithSteerSource injects the staged busy-input source consumed at tool-result boundaries.
func WithSteerSource(source SteerSource) Option {
	return func(driver *Driver) {
		driver.steerSource = source
	}
}

// New constructs an ACP driver with sensible defaults.
func New(opts ...Option) *Driver {
	driver := &Driver{
		logger:               slog.Default(),
		stopTimeout:          defaultStopTimeout,
		promptBufferCap:      defaultPromptBufSize,
		promptDrainWait:      defaultPromptDrain,
		permissionWait:       defaultPermissionWait,
		processRecordTimeout: defaultProcessRecordTimeout,
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
	if driver.processRecordTimeout <= 0 {
		driver.processRecordTimeout = defaultProcessRecordTimeout
	}
	if driver.launcher == nil {
		driver.launcher = newLocalLauncher(driver.logger, driver.stopTimeout)
	}
	return driver
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

	return reqErr.Code == requestErrorResourceNotFoundCode && requestErrorIndicatesSessionLoss(reqErr)
}

// Prompt starts one prompt turn and returns the streamed event channel.
func (d *Driver) Prompt(ctx context.Context, proc *AgentProcess, req PromptRequest) (<-chan AgentEvent, error) {
	if ctx == nil {
		return nil, errors.New("acp: context is required")
	}
	if proc == nil {
		return nil, errors.New("acp: agent process is required")
	}
	if proc.conn == nil {
		return nil, errProcessConnectionUninitialized
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	promptCtx, cancelPrompt := context.WithCancel(ctx)
	active, err := proc.beginPrompt(req.TurnID, d.promptBufferCap, cancelPrompt)
	if err != nil {
		cancelPrompt()
		return nil, err
	}

	go d.runPrompt(promptCtx, proc, active, req)
	return active.events, nil
}

// Cancel cancels the local active prompt or falls back to ACP cooperative cancellation.
func (d *Driver) Cancel(ctx context.Context, proc *AgentProcess) error {
	if ctx == nil {
		return errors.New("acp: context is required")
	}
	if proc == nil {
		return errors.New("acp: agent process is required")
	}
	if proc.conn == nil {
		return errProcessConnectionUninitialized
	}
	if strings.TrimSpace(proc.SessionID) == "" {
		return errors.New("acp: session id is required")
	}
	if proc.cancelCurrentPrompt() {
		return nil
	}
	return proc.conn.SendNotification(ctx, acpsdk.AgentMethodSessionCancel, acpsdk.CancelNotification{
		SessionId: acpsdk.SessionId(proc.SessionID),
	})
}

// Interrupt signals processes matching a scoped toolruntime selector.
func (d *Driver) Interrupt(
	ctx context.Context,
	scope toolruntime.InterruptScope,
) (toolruntime.InterruptReport, error) {
	if ctx == nil {
		return toolruntime.InterruptReport{}, errors.New("acp: interrupt context is required")
	}
	if d == nil || d.processRegistry == nil {
		return toolruntime.InterruptReport{}, toolruntime.ErrProcessNotFound
	}
	return d.processRegistry.Interrupt(ctx, scope)
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
	if proc.done == nil {
		return errProcessLifecycleUninitialized
	}

	select {
	case <-proc.Done():
		return proc.Wait()
	default:
	}
	if proc.handle == nil && proc.managed == nil && proc.cmd == nil {
		return errProcessLifecycleUninitialized
	}

	proc.markStopRequested()
	if proc.processRecord != nil {
		recordCtx, cancelRecord := processRecordContext(ctx, d.processRecordTimeout)
		err := proc.processRecord.Checkpoint(recordCtx, toolruntime.ProcessCheckpoint{
			State: toolruntime.ProcessStateInterrupting,
			Error: "ACP stop requested",
		})
		cancelRecord()
		if err != nil && d.logger != nil {
			d.logger.Warn("acp: checkpoint process interrupt", "pid", proc.PID, "error", err)
		}
	}
	errs := d.cancelSessionForStop(ctx, proc)
	if proc.handle != nil {
		return stopAgentProcessAndWait(ctx, proc, errs, proc.handle.Stop)
	}
	if proc.managed != nil {
		return stopAgentProcessAndWait(ctx, proc, errs, proc.managed.Shutdown)
	}

	return d.stopExecCommand(ctx, proc, errs)
}

func (d *Driver) cancelSessionForStop(ctx context.Context, proc *AgentProcess) []error {
	if strings.TrimSpace(proc.SessionID) == "" {
		return nil
	}
	cancelCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
	defer cancel()

	if err := d.Cancel(cancelCtx, proc); err != nil && !errors.Is(err, context.Canceled) {
		return []error{fmt.Errorf("acp: cancel session prompt: %w", err)}
	}
	return nil
}

func stopAgentProcessAndWait(
	ctx context.Context,
	proc *AgentProcess,
	errs []error,
	stopFn func(context.Context) error,
) error {
	if err := stopFn(ctx); err != nil {
		errs = append(errs, err)
	}
	select {
	case <-proc.Done():
		return errors.Join(append(errs, proc.Wait())...)
	case <-ctx.Done():
		return errors.Join(append(errs, ctx.Err())...)
	}
}

func (d *Driver) stopExecCommand(ctx context.Context, proc *AgentProcess, errs []error) error {
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
	defer func() {
		if active != nil && active.cancel != nil {
			active.cancel()
		}
		proc.endPrompt(active)
	}()

	stopReporter := startPromptActivityReporter(ctx, req)
	defer stopReporter()

	stopCancellationNotifier := d.startPromptCancellationNotifier(ctx, proc, active)
	defer stopCancellationNotifier()

	promptRequest, err := buildWirePromptRequest(proc, req)
	if err != nil {
		emitPromptBuildError(proc, req, err)
		return
	}
	response, err := acpsdk.SendRequest[wirePromptResponse](
		proc.conn,
		ctx,
		acpsdk.AgentMethodSessionPrompt,
		promptRequest,
	)

	if err != nil {
		if proc.stopWasRequested() && shouldSuppressPromptErrorOnStop(err) {
			return
		}
		failure, _ := FailureFromError(err, store.FailurePrompt)
		event := AgentEvent{
			Type:      EventTypeError,
			SessionID: proc.SessionID,
			TurnID:    req.TurnID,
			Timestamp: timeNowUTC(),
			Error:     firstNonEmptyFailureText(failureSummary(failure), err.Error()),
			Failure:   failure,
			Raw:       requestErrorRaw(err),
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
		StopReason: string(response.StopReason), PromptStopReason: PromptStopReason(response.StopReason),
	}
	if !usage.IsZero() {
		doneEvent.Usage = &usage
	}
	d.waitForPromptQuiescence(active)
	proc.emitPromptEvent(doneEvent)
}

func (d *Driver) startPromptCancellationNotifier(
	ctx context.Context,
	proc *AgentProcess,
	active *activePromptState,
) func() {
	if ctx.Err() != nil {
		d.sendPromptCancellationNotification(ctx, proc, active)
		return func() {}
	}

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			d.sendPromptCancellationNotification(ctx, proc, active)
		case <-done:
		}
	}()
	return func() {
		close(done)
	}
}

func (d *Driver) sendPromptCancellationNotification(
	ctx context.Context,
	proc *AgentProcess,
	active *activePromptState,
) {
	if strings.TrimSpace(proc.SessionID) == "" {
		return
	}
	notifyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
	defer cancel()
	err := proc.conn.SendNotification(
		notifyCtx,
		acpsdk.AgentMethodSessionCancel,
		acpsdk.CancelNotification{SessionId: acpsdk.SessionId(proc.SessionID)},
	)
	if err != nil && !errors.Is(err, context.Canceled) {
		d.logger.WarnContext(
			notifyCtx,
			"acp: send session cancel notification",
			"session_id",
			proc.SessionID,
			"turn_id",
			active.turnID,
			"error",
			err,
		)
	}
}

func buildWirePromptRequest(proc *AgentProcess, req PromptRequest) (acpsdk.PromptRequest, error) {
	promptText, includedSystemPrompt, promptDelivery := proc.nextPromptText(req.Message)
	promptRequest := acpsdk.PromptRequest{
		SessionId: acpsdk.SessionId(proc.SessionID),
		Prompt:    []acpsdk.ContentBlock{textBlockWithPromptCacheControl(promptText, proc.promptCacheControl)},
	}
	meta := req.Meta.Normalize()
	if includedSystemPrompt {
		if promptDelivery == "" {
			promptDelivery = SystemPromptDeliveryFirstTurnPrefix
		}
		meta.System = &PromptSystemMeta{
			PromptDelivery: string(promptDelivery),
		}
	}
	if !meta.IsZero() {
		if err := meta.Validate(); err != nil {
			return acpsdk.PromptRequest{}, err
		}
		metaMap, err := meta.ToMap()
		if err != nil {
			return acpsdk.PromptRequest{}, err
		}
		promptRequest.Meta = metaMap
	}
	return promptRequest, nil
}

func emitPromptBuildError(proc *AgentProcess, req PromptRequest, err error) {
	proc.emitPromptEvent(AgentEvent{
		Type:      EventTypeError,
		SessionID: proc.SessionID,
		TurnID:    req.TurnID,
		Timestamp: timeNowUTC(),
		Error:     err.Error(),
	})
}

func shouldSuppressPromptErrorOnStop(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if reqErr, ok := errors.AsType[*acpsdk.RequestError](err); ok {
		text := strings.ToLower(strings.TrimSpace(requestErrorDiagnosticText(reqErr)))
		if strings.Contains(text, "context canceled") ||
			strings.Contains(text, "context deadline exceeded") ||
			strings.Contains(text, "peer disconnected before response") {
			return true
		}
	}
	failure, ok := FailureFromError(err, store.FailurePrompt)
	return ok && failure != nil && failure.Kind == store.FailureCanceled
}

func startPromptActivityReporter(ctx context.Context, req PromptRequest) func() {
	if req.ActivityReporter == nil {
		return func() {}
	}
	interval := req.ActivityHeartbeatInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}

	done := make(chan struct{})
	report := func(ts time.Time) {
		req.ActivityReporter(PromptActivityReport{
			Timestamp: ts,
			Kind:      clientAgentWaitingKey,
			Detail:    "waiting for session/prompt response",
		})
	}
	report(timeNowUTC())

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			case ts := <-ticker.C:
				report(ts.UTC())
			}
		}
	}()

	return func() {
		close(done)
	}
}

func (p *AgentProcess) waitForExit(ctx context.Context, processRecordTimeout time.Duration) {
	var waitErr error
	var groupWaitErr error
	switch {
	case p.handle != nil:
		waitErr = p.handle.Wait()
	case p.managed != nil:
		waitErr = p.managed.Wait()
	case p.cmd != nil:
		waitErr = p.cmd.Wait()
		if p.stopWasRequested() {
			groupWaitErr = forceManagedProcessGroupExit(p.cmd, time.Second)
		}
	default:
		waitErr = nil
	}
	if p.stopWasRequested() {
		waitErr = nil
		if groupWaitErr != nil {
			waitErr = fmt.Errorf("acp: wait for subprocess tree exit: %w", groupWaitErr)
		}
	} else if waitErr != nil {
		waitErr = WrapFailure(
			store.FailureProcess,
			"ACP subprocess exited unexpectedly",
			fmt.Errorf("acp: subprocess exited: %w", attachStderr(waitErr, p.Stderr())),
		)
	}
	p.setWaitError(waitErr)
	if p.processRecord != nil {
		recordCtx, cancelRecord := processRecordContext(ctx, processRecordTimeout)
		err := p.processRecord.Complete(recordCtx, toolruntime.ProcessCompletion{Err: waitErr})
		cancelRecord()
		if err != nil {
			slog.Default().Warn("acp: complete process record", "pid", p.PID, "error", err)
		}
	}
	if p.cancelProcess != nil {
		p.cancelProcess()
	}
	if p.terminals != nil {
		p.terminals.closeAll()
	}
	close(p.done)
}

func processRecordContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = defaultProcessRecordTimeout
	}
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(context.WithoutCancel(parent), timeout)
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
	if strings.TrimSpace(normalized.SystemPrompt) == "" {
		normalized.SystemPromptDelivery = ""
	} else if normalized.SystemPromptDelivery == "" {
		normalized.SystemPromptDelivery = SystemPromptDeliveryFirstTurnPrefix
	}
	normalized.PreferredModel = strings.TrimSpace(normalized.PreferredModel)
	normalized.ReasoningEffort = strings.TrimSpace(normalized.ReasoningEffort)

	return normalized, nil
}

func daemonMatchedEnv(base []string) []string {
	env := append([]string(nil), base...)
	if len(env) == 0 {
		env = os.Environ()
	}

	executable, err := os.Executable()
	if err != nil {
		return env
	}
	if resolved, resolveErr := filepath.EvalSymlinks(
		executable,
	); resolveErr == nil &&
		strings.TrimSpace(resolved) != "" {
		executable = resolved
	}
	executable = strings.TrimSpace(executable)
	if executable == "" {
		return env
	}

	env = setEnvValue(env, "AGH_BIN", executable)

	binDir := strings.TrimSpace(filepath.Dir(executable))
	if binDir == "" {
		return env
	}

	pathValue, _ := envValue(env, "PATH")
	env = setEnvValue(env, "PATH", prependPathEntry(pathValue, binDir))
	return env
}

func prependPathEntry(pathValue string, entry string) string {
	cleanEntry := strings.TrimSpace(entry)
	if cleanEntry == "" {
		return pathValue
	}

	separator := string(os.PathListSeparator)
	segments := strings.Split(pathValue, separator)
	filtered := make([]string, 0, len(segments))
	for _, segment := range segments {
		trimmed := strings.TrimSpace(segment)
		if trimmed == "" || trimmed == cleanEntry {
			continue
		}
		filtered = append(filtered, trimmed)
	}
	return strings.Join(append([]string{cleanEntry}, filtered...), separator)
}

func envValue(env []string, key string) (string, bool) {
	prefix := key + "="
	for _, variable := range slices.Backward(env) {
		if strings.HasPrefix(variable, prefix) {
			return variable[len(prefix):], true
		}
	}
	return "", false
}

func setEnvValue(env []string, key string, value string) []string {
	prefix := key + "="
	entry := prefix + value
	filtered := env[:0]
	for _, variable := range env {
		if strings.HasPrefix(variable, prefix) {
			continue
		}
		filtered = append(filtered, variable)
	}
	return append(filtered, entry)
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
		if server.EffectiveTransport() != aghconfig.MCPServerTransportStdio {
			continue
		}
		if strings.TrimSpace(server.Command) == "" {
			continue
		}
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

func captureCaps(
	loadSession bool,
	modes *acpsdk.SessionModeState,
	configOptions []acpsdk.SessionConfigOption,
) Caps {
	caps := Caps{SupportsLoadSession: loadSession}
	if modes != nil {
		caps.SupportedModes = make([]string, 0, len(modes.AvailableModes))
		for _, mode := range modes.AvailableModes {
			caps.SupportedModes = append(caps.SupportedModes, string(mode.Id))
		}
	}
	caps.ConfigOptions = sessionConfigOptionsFromSDK(configOptions)
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
