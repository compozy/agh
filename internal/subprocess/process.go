// Package subprocess provides shared subprocess lifecycle primitives for AGH.
package subprocess

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	exec "os/exec"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/execabs"
)

const (
	defaultMaxMessageBytes     = 10 << 20
	defaultShutdownTimeout     = 10 * time.Second
	defaultPostSignalGrace     = 250 * time.Millisecond
	defaultProcessGroupWait    = time.Second
	defaultHealthFailureThresh = 2

	initializeMethod = "initialize"
	shutdownMethod   = "shutdown"
)

var (
	// ErrNotInitialized reports that an operational request was attempted before initialize completed.
	ErrNotInitialized = errors.New("subprocess: not initialized")
	// ErrShutdownInProgress reports that the process is draining and will not accept new requests.
	ErrShutdownInProgress = errors.New("subprocess: shutdown in progress")
	// ErrTransportDisabled reports that JSON-RPC transport methods were called on a raw-process launch.
	ErrTransportDisabled = errors.New("subprocess: transport disabled")
)

type processState int

const (
	processStateStarting processState = iota
	processStateReady
	processStateDraining
	processStateStopped
)

// LaunchConfig configures a managed subprocess.
type LaunchConfig struct {
	Command string
	Args    []string
	Dir     string
	Env     []string

	Logger *slog.Logger

	// DisableTransport leaves stdout unread so callers like ACP can attach their own protocol layer.
	DisableTransport bool

	// MaxMessageBytes bounds a single encoded JSON-RPC frame when transport is enabled.
	MaxMessageBytes int

	// ShutdownTimeout bounds the cooperative shutdown wait before signal escalation.
	ShutdownTimeout time.Duration
	// PostSignalGrace bounds the wait between SIGTERM and SIGKILL escalation.
	PostSignalGrace time.Duration
	// ShutdownReason is sent as the shutdown RPC reason when transport is enabled.
	ShutdownReason string

	// HealthFailureThreshold overrides the consecutive probe failures needed to mark the process unhealthy.
	HealthFailureThreshold int
}

// Process manages one subprocess and its optional JSON-RPC transport.
type Process struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr *boundedBuffer
	logger *slog.Logger

	transport *transport

	lifecycleCtx    context.Context
	cancelLifecycle context.CancelFunc

	done          chan struct{}
	waitMu        sync.RWMutex
	waitErr       error
	stopRequested bool
	stopMu        sync.RWMutex
	closeInputMu  sync.Mutex
	inputClosed   bool

	stateMu sync.RWMutex
	state   processState

	transportErrMu sync.RWMutex
	transportErr   error

	shutdownTimeout time.Duration
	postSignalGrace time.Duration
	shutdownReason  string

	healthThreshold int
	health          healthMonitor
}

type launchRuntimeConfig struct {
	logger          *slog.Logger
	maxMessageBytes int
	shutdownTimeout time.Duration
	postSignalGrace time.Duration
	healthThreshold int
}

// Launch starts a managed subprocess and optionally attaches the shared JSON-RPC transport.
func Launch(ctx context.Context, cfg LaunchConfig) (*Process, error) {
	if ctx == nil {
		return nil, errors.New("subprocess: context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if cfg.Command == "" {
		return nil, errors.New("subprocess: command is required")
	}

	runtime := resolveLaunchRuntime(cfg)
	cmd, stdin, stdout, stderr, err := startManagedCommand(cfg)
	if err != nil {
		return nil, err
	}
	lifecycleCtx, cancelLifecycle := context.WithCancel(context.Background())
	process := &Process{
		cmd:             cmd,
		stdin:           stdin,
		stdout:          stdout,
		stderr:          stderr,
		logger:          runtime.logger,
		lifecycleCtx:    lifecycleCtx,
		cancelLifecycle: cancelLifecycle,
		done:            make(chan struct{}),
		state:           processStateStarting,
		shutdownTimeout: runtime.shutdownTimeout,
		postSignalGrace: runtime.postSignalGrace,
		shutdownReason:  cfg.defaultShutdownReason(),
		healthThreshold: runtime.healthThreshold,
	}

	if !cfg.DisableTransport {
		process.transport = newTransport(process, runtime.maxMessageBytes)
		process.transport.start()
	}

	go process.waitForExit()

	return process, nil
}

func resolveLaunchRuntime(cfg LaunchConfig) launchRuntimeConfig {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	maxMessageBytes := cfg.MaxMessageBytes
	if maxMessageBytes <= 0 {
		maxMessageBytes = defaultMaxMessageBytes
	}

	shutdownTimeout := cfg.ShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = defaultShutdownTimeout
	}

	postSignalGrace := cfg.PostSignalGrace
	if postSignalGrace <= 0 {
		postSignalGrace = defaultPostSignalGrace
	}

	healthThreshold := cfg.HealthFailureThreshold
	if healthThreshold <= 0 {
		healthThreshold = defaultHealthFailureThresh
	}

	return launchRuntimeConfig{
		logger:          logger,
		maxMessageBytes: maxMessageBytes,
		shutdownTimeout: shutdownTimeout,
		postSignalGrace: postSignalGrace,
		healthThreshold: healthThreshold,
	}
}

func startManagedCommand(cfg LaunchConfig) (*exec.Cmd, io.WriteCloser, io.ReadCloser, *boundedBuffer, error) {
	commandPath, commandArgs, err := resolvedCommand(cfg.Command, cfg.Args)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	cmd := &exec.Cmd{
		Path: commandPath,
		Args: commandArgs,
	}
	configureManagedCommand(cmd)
	cmd.Dir = cfg.Dir
	if len(cfg.Env) > 0 {
		cmd.Env = append([]string(nil), cfg.Env...)
	} else {
		cmd.Env = os.Environ()
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("subprocess: open stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("subprocess: open stdout pipe: %w", err)
	}

	stderr := &boundedBuffer{limit: 128 * 1024}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("subprocess: start %q: %w", cfg.Command, err)
	}

	return cmd, stdin, stdout, stderr, nil
}

func resolvedCommand(command string, args []string) (string, []string, error) {
	resolvedPath, err := execabs.LookPath(command)
	if err != nil {
		return "", nil, fmt.Errorf("subprocess: resolve executable %q: %w", command, err)
	}

	commandArgs := make([]string, 0, len(args)+1)
	commandArgs = append(commandArgs, resolvedPath)
	commandArgs = append(commandArgs, args...)
	return resolvedPath, commandArgs, nil
}

// PID returns the operating-system process identifier.
func (p *Process) PID() int {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return 0
	}
	return p.cmd.Process.Pid
}

// Stdin exposes the subprocess stdin writer for callers that disable the shared transport.
func (p *Process) Stdin() io.WriteCloser {
	if p == nil {
		return nil
	}
	return p.stdin
}

// Stdout exposes the subprocess stdout reader for callers that disable the shared transport.
func (p *Process) Stdout() io.ReadCloser {
	if p == nil {
		return nil
	}
	return p.stdout
}

// Done closes when the subprocess exits.
func (p *Process) Done() <-chan struct{} {
	if p == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return p.done
}

// Wait blocks until the subprocess exits and returns its final error state.
func (p *Process) Wait() error {
	if p == nil {
		return nil
	}
	<-p.Done()
	p.waitMu.RLock()
	defer p.waitMu.RUnlock()
	return p.waitErr
}

// Stderr returns the captured stderr tail for diagnostics.
func (p *Process) Stderr() string {
	if p == nil || p.stderr == nil {
		return ""
	}
	return p.stderr.String()
}

// HandleMethod registers an inbound JSON-RPC request handler.
func (p *Process) HandleMethod(method string, handler HandlerFunc) error {
	if p == nil {
		return errors.New("subprocess: process is required")
	}
	if p.transport == nil {
		return ErrTransportDisabled
	}
	return p.transport.handleMethod(method, handler)
}

// Call sends an outbound JSON-RPC request and decodes the response.
func (p *Process) Call(ctx context.Context, method string, params, result any) error {
	if ctx == nil {
		return errors.New("subprocess: context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if p == nil {
		return errors.New("subprocess: process is required")
	}
	if p.transport == nil {
		return ErrTransportDisabled
	}

	switch p.currentState() {
	case processStateStarting:
		if method != initializeMethod {
			return fmt.Errorf("subprocess: call %q: %w", method, ErrNotInitialized)
		}
	case processStateDraining:
		if method != shutdownMethod {
			return fmt.Errorf("subprocess: call %q: %w", method, ErrShutdownInProgress)
		}
	case processStateStopped:
		if waitErr := p.Wait(); waitErr != nil {
			return waitErr
		}
		return errors.New("subprocess: process already stopped")
	}

	return p.transport.call(ctx, method, params, result)
}

// Shutdown performs cooperative shutdown when transport is enabled, then escalates signals if needed.
func (p *Process) Shutdown(ctx context.Context) error {
	if ctx == nil {
		return errors.New("subprocess: context is required")
	}
	if p == nil {
		return errors.New("subprocess: process is required")
	}

	select {
	case <-p.Done():
		return p.Wait()
	default:
	}

	p.markStopRequested()
	p.setState(processStateDraining)

	var errs []error
	var stopCtxErr error
	if p.transport != nil && p.currentState() != processStateStopped {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), p.shutdownTimeout)
		defer cancel()

		var response ShutdownResponse
		err := p.Call(shutdownCtx, shutdownMethod, ShutdownRequest{
			Reason:     p.shutdownReason,
			DeadlineMS: p.shutdownTimeout.Milliseconds(),
		}, &response)
		if err != nil {
			errs = append(errs, fmt.Errorf("subprocess: cooperative shutdown: %w", err))
		}
	}

	if err := p.closeInput(); err != nil {
		errs = append(errs, fmt.Errorf("subprocess: close stdin: %w", err))
	}

	if waitErr := p.waitWithContext(ctx, p.shutdownTimeout); waitErr == nil {
		return joinShutdownResult(errs, p.Wait(), stopCtxErr)
	} else if !errors.Is(waitErr, context.DeadlineExceeded) {
		return errors.Join(append(errs, waitErr)...)
	} else if ctxErr := ctx.Err(); ctxErr != nil {
		stopCtxErr = ctxErr
	}

	if err := terminateManagedProcess(p.cmd); err != nil {
		errs = append(errs, fmt.Errorf("subprocess: terminate process tree: %w", err))
	}

	if waitErr := p.waitWithContext(ctx, p.postSignalGrace); waitErr == nil {
		return joinShutdownResult(errs, p.Wait(), stopCtxErr)
	} else if !errors.Is(waitErr, context.DeadlineExceeded) {
		return errors.Join(append(errs, waitErr)...)
	} else if ctxErr := ctx.Err(); ctxErr != nil {
		stopCtxErr = ctxErr
	}

	if err := killManagedProcess(p.cmd); err != nil {
		errs = append(errs, fmt.Errorf("subprocess: kill process tree: %w", err))
	}

	select {
	case <-p.Done():
		return joinShutdownResult(errs, p.Wait(), stopCtxErr)
	case <-ctx.Done():
		return errors.Join(append(errs, ctx.Err())...)
	}
}

func joinShutdownResult(errs []error, waitErr error, stopCtxErr error) error {
	joined := errors.Join(append(errs, waitErr)...)
	if stopCtxErr == nil {
		return joined
	}
	return errors.Join(joined, stopCtxErr)
}

func (p *Process) waitForExit() {
	waitErr := p.cmd.Wait()
	p.cancelLifecycle()

	var groupWaitErr error
	if p.stopWasRequested() {
		groupWaitErr = forceManagedProcessGroupExit(p.cmd, defaultProcessGroupWait)
	}

	if p.stopWasRequested() {
		waitErr = nil
		if groupWaitErr != nil {
			waitErr = fmt.Errorf("subprocess: wait for process tree exit: %w", groupWaitErr)
		}
	} else if waitErr != nil {
		waitErr = fmt.Errorf("subprocess: process exited: %w", attachStderr(waitErr, p.Stderr()))
	}

	if p.transport != nil {
		p.transport.shutdown(waitErr)
	}
	p.stopHealthMonitor()
	p.setState(processStateStopped)

	transportErr := p.currentTransportError()
	if p.stopWasRequested() && isBenignTransportShutdownError(transportErr) {
		transportErr = nil
	}
	if waitErr == nil && transportErr != nil {
		waitErr = transportErr
	} else if waitErr != nil && transportErr != nil {
		waitErr = errors.Join(waitErr, transportErr)
	}

	p.waitMu.Lock()
	p.waitErr = waitErr
	p.waitMu.Unlock()

	close(p.done)
}

func (p *Process) waitWithContext(ctx context.Context, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = time.Millisecond
	}
	waitCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case <-p.Done():
		return nil
	case <-waitCtx.Done():
		return waitCtx.Err()
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Process) closeInput() error {
	p.closeInputMu.Lock()
	defer p.closeInputMu.Unlock()
	if p.inputClosed || p.stdin == nil {
		return nil
	}
	p.inputClosed = true
	if err := p.stdin.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		return err
	}
	return nil
}

func (p *Process) currentState() processState {
	p.stateMu.RLock()
	defer p.stateMu.RUnlock()
	return p.state
}

func (p *Process) setState(state processState) {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()
	if p.state == processStateStopped {
		return
	}
	p.state = state
}

func (p *Process) markReady() {
	p.setState(processStateReady)
}

func (p *Process) markStopRequested() {
	p.stopMu.Lock()
	defer p.stopMu.Unlock()
	p.stopRequested = true
}

func (p *Process) stopWasRequested() bool {
	p.stopMu.RLock()
	defer p.stopMu.RUnlock()
	return p.stopRequested
}

func (p *Process) recordTransportError(err error) {
	if err == nil {
		return
	}
	p.transportErrMu.Lock()
	defer p.transportErrMu.Unlock()
	if p.transportErr == nil {
		p.transportErr = err
	}
}

func (p *Process) currentTransportError() error {
	p.transportErrMu.RLock()
	defer p.transportErrMu.RUnlock()
	return p.transportErr
}

func attachStderr(err error, stderr string) error {
	if stderr == "" {
		return err
	}
	return fmt.Errorf("%w: stderr=%s", err, stderr)
}

func isBenignTransportShutdownError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrClosed) {
		return true
	}
	return strings.Contains(err.Error(), "file already closed")
}

func (cfg LaunchConfig) defaultShutdownReason() string {
	if cfg.ShutdownReason != "" {
		return cfg.ShutdownReason
	}
	return "daemon_shutdown"
}

type boundedBuffer struct {
	mu    sync.Mutex
	buf   []byte
	limit int
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.limit <= 0 {
		return len(p), nil
	}

	if len(p) >= b.limit {
		b.buf = append(b.buf[:0], p[len(p)-b.limit:]...)
		return len(p), nil
	}

	if overflow := len(b.buf) + len(p) - b.limit; overflow > 0 {
		copy(b.buf, b.buf[overflow:])
		b.buf = b.buf[:len(b.buf)-overflow]
	}

	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *boundedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}
