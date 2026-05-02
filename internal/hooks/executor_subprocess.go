package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	exec "os/exec"
	"sort"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/diagnostics"
	"github.com/pedronauck/agh/internal/toolruntime"
	"github.com/pedronauck/agh/internal/vault"
	"golang.org/x/sys/execabs"
)

const (
	defaultSubprocessHookTimeout = 5 * time.Second
	subprocessCaptureLimitBytes  = 8 * 1024
	subprocessCaptureTruncate    = "...[truncated]"
	subprocessShutdownGrace      = 250 * time.Millisecond
	subprocessProcessGroupWait   = time.Second
)

var subprocessEnvAllowlist = []string{
	"COMSPEC",
	"HOME",
	"LANG",
	"LC_ALL",
	"LC_CTYPE",
	"LOGNAME",
	"PATH",
	"PATHEXT",
	"SHELL",
	"SYSTEMROOT",
	"TEMP",
	"TERM",
	"TMP",
	"TMPDIR",
	"USER",
	"USERPROFILE",
}

// SubprocessExecutorOption mutates a subprocess executor during construction.
type SubprocessExecutorOption func(*SubprocessExecutor)

// WithSubprocessDir configures the working directory for a subprocess hook.
func WithSubprocessDir(dir string) SubprocessExecutorOption {
	return func(executor *SubprocessExecutor) {
		executor.dir = strings.TrimSpace(dir)
	}
}

// WithSubprocessEnv configures the explicit environment overrides for a hook.
func WithSubprocessEnv(env map[string]string) SubprocessExecutorOption {
	return func(executor *SubprocessExecutor) {
		executor.env = cloneStringMap(env)
	}
}

// SecretRefResolver resolves env: and vault: refs for subprocess secret env bindings.
type SecretRefResolver interface {
	ResolveRef(context.Context, string) (string, error)
}

// WithSubprocessSecretEnv configures secret refs resolved immediately before a hook runs.
func WithSubprocessSecretEnv(env map[string]string, resolver SecretRefResolver) SubprocessExecutorOption {
	return func(executor *SubprocessExecutor) {
		executor.secretEnv = cloneStringMap(env)
		executor.secretResolver = resolver
	}
}

// WithSubprocessProcessRegistry injects the shared process registry for subprocess hook commands.
func WithSubprocessProcessRegistry(registry *toolruntime.Registry) SubprocessExecutorOption {
	return func(executor *SubprocessExecutor) {
		executor.registry = registry
	}
}

// SubprocessExecutor runs hooks through a local shell command boundary.
type SubprocessExecutor struct {
	command        string
	args           []string
	dir            string
	env            map[string]string
	secretEnv      map[string]string
	secretResolver SecretRefResolver
	registry       *toolruntime.Registry
}

var _ Executor = (*SubprocessExecutor)(nil)

// NewSubprocessExecutor constructs a subprocess-backed executor.
func NewSubprocessExecutor(command string, args []string, opts ...SubprocessExecutorOption) *SubprocessExecutor {
	executor := &SubprocessExecutor{
		command: strings.TrimSpace(command),
		args:    append([]string(nil), args...),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(executor)
		}
	}
	return executor
}

// Kind returns the executor type.
func (*SubprocessExecutor) Kind() HookExecutorKind {
	return HookExecutorSubprocess
}

// Execute runs the configured command with the JSON payload on stdin and
// returns captured stdout.
func (e *SubprocessExecutor) Execute(ctx context.Context, hook RegisteredHook, payload []byte) ([]byte, error) {
	if e == nil || e.command == "" {
		return nil, fmt.Errorf("hooks: hook %q: %w", hook.Name, ErrSubprocessCommandRequired)
	}

	timeout := subprocessHookTimeout(hook.Timeout)
	hookCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	commandPath, commandArgs, err := resolvedHookCommand(e.command, e.args)
	if err != nil {
		return nil, fmt.Errorf("hooks: hook %q: %w", hook.Name, err)
	}

	cmd := &exec.Cmd{
		Path: commandPath,
		Args: commandArgs,
	}
	configureSubprocessCommand(cmd)
	cmd.Dir = e.dir
	env, cleanup, err := e.subprocessProcessEnv(ctx)
	if err != nil {
		return nil, fmt.Errorf("hooks: hook %q: %w", hook.Name, err)
	}
	defer cleanup()
	cmd.Env = env
	cmd.Stdin = bytes.NewReader(payload)

	stdout := newLimitedSubprocessCapture()
	stderr := newLimitedSubprocessCapture()
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err = runSubprocessCommand(hookCtx, cmd, hook, payload, e.registry)
	output := []byte(stdout.String())
	if err != nil {
		return output, subprocessRunError(hookCtx, timeout, err, stderr)
	}

	return output, nil
}

func (e *SubprocessExecutor) subprocessProcessEnv(ctx context.Context) ([]string, func(), error) {
	env := cloneStringMap(e.env)
	cleanups := []func(){}
	if len(e.secretEnv) == 0 {
		return subprocessProcessEnv(env), func() {}, nil
	}
	if ctx == nil {
		return nil, func() {}, errors.New("secret env context is required")
	}
	keys := make([]string, 0, len(e.secretEnv))
	for key := range e.secretEnv {
		keys = append(keys, strings.TrimSpace(key))
	}
	sort.Strings(keys)
	for _, key := range keys {
		ref := vault.NormalizeRef(e.secretEnv[key])
		value, err := e.resolveSecretRef(ctx, ref)
		if err != nil {
			runSubprocessSecretCleanups(cleanups)
			return nil, func() {}, fmt.Errorf("resolve secret_env.%s: %w", key, err)
		}
		env[key] = value
		cleanups = append(cleanups, diagnostics.RegisterDynamicSecret(value))
	}
	return subprocessProcessEnv(env), func() { runSubprocessSecretCleanups(cleanups) }, nil
}

func (e *SubprocessExecutor) resolveSecretRef(ctx context.Context, ref string) (string, error) {
	if e.secretResolver != nil {
		return e.secretResolver.ResolveRef(ctx, ref)
	}
	envName, err := vault.EnvNameFromRef(ref)
	if err != nil {
		return "", err
	}
	value, ok := os.LookupEnv(envName)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%w: env:%s", vault.ErrMissingSecret, envName)
	}
	return value, nil
}

func runSubprocessSecretCleanups(cleanups []func()) {
	for index := len(cleanups) - 1; index >= 0; index-- {
		if cleanups[index] != nil {
			cleanups[index]()
		}
	}
}

func resolvedHookCommand(command string, args []string) (string, []string, error) {
	resolvedPath, err := execabs.LookPath(command)
	if err != nil {
		return "", nil, fmt.Errorf("resolve executable %q: %w", command, err)
	}

	commandArgs := make([]string, 0, len(args)+1)
	commandArgs = append(commandArgs, resolvedPath)
	commandArgs = append(commandArgs, args...)
	return resolvedPath, commandArgs, nil
}

func subprocessHookTimeout(timeout time.Duration) time.Duration {
	if timeout > 0 {
		return timeout
	}

	return defaultSubprocessHookTimeout
}

func runSubprocessCommand(
	ctx context.Context,
	cmd *exec.Cmd,
	hook RegisteredHook,
	payload []byte,
	registry *toolruntime.Registry,
) error {
	if err := cmd.Start(); err != nil {
		return err
	}

	record, err := registerSubprocessHook(ctx, cmd, hook, payload, registry)
	if err != nil {
		return errors.Join(err, cleanupStartedSubprocessCommand(cmd))
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	select {
	case err := <-waitCh:
		return errors.Join(err, completeSubprocessHook(context.Background(), record, cmd, err))
	case <-ctx.Done():
		checkpointErr := checkpointSubprocessHook(
			context.Background(),
			record,
			toolruntime.ProcessStateInterrupting,
			ctx.Err().Error(),
		)
		terminateErr := terminateSubprocessCommand(cmd)
		timer := time.NewTimer(subprocessShutdownGrace)
		defer timer.Stop()

		select {
		case err := <-waitCh:
			groupErr := forceSubprocessCommandExit(cmd, subprocessProcessGroupWait)
			completeErr := completeSubprocessHook(context.Background(), record, cmd, errors.Join(err, groupErr))
			return errors.Join(checkpointErr, terminateErr, err, groupErr, completeErr)
		case <-timer.C:
			killErr := killSubprocessCommand(cmd)
			waitErr := <-waitCh
			groupErr := forceSubprocessCommandExit(cmd, subprocessProcessGroupWait)
			completeErr := completeSubprocessHook(context.Background(), record, cmd, errors.Join(waitErr, groupErr))
			return errors.Join(
				checkpointErr,
				terminateErr,
				killErr,
				waitErr,
				groupErr,
				completeErr,
			)
		}
	}
}

func cleanupStartedSubprocessCommand(cmd *exec.Cmd) error {
	if cmd == nil {
		return nil
	}
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	terminateErr := terminateSubprocessCommand(cmd)
	timer := time.NewTimer(subprocessShutdownGrace)
	defer timer.Stop()

	select {
	case waitErr := <-waitCh:
		return errors.Join(terminateErr, waitErr, forceSubprocessCommandExit(cmd, subprocessProcessGroupWait))
	case <-timer.C:
		killErr := killSubprocessCommand(cmd)
		waitErr := <-waitCh
		return errors.Join(
			terminateErr,
			killErr,
			waitErr,
			forceSubprocessCommandExit(cmd, subprocessProcessGroupWait),
		)
	}
}

func registerSubprocessHook(
	ctx context.Context,
	cmd *exec.Cmd,
	hook RegisteredHook,
	payload []byte,
	registry *toolruntime.Registry,
) (*toolruntime.Handle, error) {
	if registry == nil || cmd == nil || cmd.Process == nil {
		return nil, nil
	}
	owner := subprocessHookOwner(hook, payload)
	command := cmd.Path
	args := []string(nil)
	if len(cmd.Args) > 0 {
		command = cmd.Args[0]
		args = cmd.Args[1:]
	}
	return registry.Register(ctx, toolruntime.RegisterConfig{
		Source:         toolruntime.ProcessSourceHook,
		Owner:          owner,
		PID:            cmd.Process.Pid,
		ProcessGroupID: cmd.Process.Pid,
		Command:        command,
		Args:           args,
		Cwd:            cmd.Dir,
		Interrupt: func(_ context.Context, _ toolruntime.ProcessRecord) error {
			return terminateSubprocessCommand(cmd)
		},
	})
}

func subprocessHookOwner(hook RegisteredHook, payload []byte) toolruntime.ProcessOwner {
	owner := toolruntime.ProcessOwner{HookName: strings.TrimSpace(hook.Name)}
	var contextPayload struct {
		SessionID  string `json:"session_id"`
		TurnID     string `json:"turn_id"`
		ToolCallID string `json:"tool_call_id"`
		SandboxID  string `json:"sandbox_id"`
	}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &contextPayload); err != nil {
			return owner
		}
	}
	owner.SessionID = contextPayload.SessionID
	owner.TurnID = contextPayload.TurnID
	owner.ToolCallID = contextPayload.ToolCallID
	owner.SandboxID = contextPayload.SandboxID
	return owner
}

func checkpointSubprocessHook(
	ctx context.Context,
	record *toolruntime.Handle,
	state toolruntime.ProcessState,
	reason string,
) error {
	if record == nil {
		return nil
	}
	return record.Checkpoint(ctx, toolruntime.ProcessCheckpoint{
		State: state,
		Error: reason,
	})
}

func completeSubprocessHook(
	ctx context.Context,
	record *toolruntime.Handle,
	cmd *exec.Cmd,
	err error,
) error {
	if record == nil {
		return nil
	}
	completion := toolruntime.ProcessCompletion{Err: err}
	if cmd != nil && cmd.ProcessState != nil {
		exitCode := cmd.ProcessState.ExitCode()
		if exitCode >= 0 {
			completion.ExitCode = &exitCode
		}
	}
	return record.Complete(ctx, completion)
}

func subprocessProcessEnv(env map[string]string) []string {
	merged := make(map[string]string, len(subprocessEnvAllowlist)+len(env))
	for _, key := range subprocessEnvAllowlist {
		if value, ok := os.LookupEnv(key); ok {
			merged[key] = value
		}
	}
	maps.Copy(merged, env)

	keys := make([]string, 0, len(merged))
	for key := range merged {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, key+"="+merged[key])
	}

	return values
}

func subprocessRunError(ctx context.Context, timeout time.Duration, err error, stderr *limitedSubprocessCapture) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("hook timed out after %s: %w", timeout, ctx.Err())
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return fmt.Errorf("hook canceled: %w", ctx.Err())
	}
	if stderr == nil || stderr.Len() == 0 {
		return fmt.Errorf("hook command failed: %w", err)
	}

	return fmt.Errorf("hook command failed: %w (%s)", err, subprocessCaptureSummary(stderr))
}

type limitedSubprocessCapture struct {
	buf       bytes.Buffer
	truncated bool
}

func newLimitedSubprocessCapture() *limitedSubprocessCapture {
	return &limitedSubprocessCapture{}
}

func (c *limitedSubprocessCapture) Write(payload []byte) (int, error) {
	if c == nil {
		return len(payload), nil
	}

	remaining := subprocessCaptureLimitBytes - c.buf.Len()
	switch {
	case remaining <= 0:
		c.truncated = true
	case len(payload) > remaining:
		_, _ = c.buf.Write(payload[:remaining])
		c.truncated = true
	default:
		_, _ = c.buf.Write(payload)
	}

	return len(payload), nil
}

func (c *limitedSubprocessCapture) String() string {
	if c == nil {
		return ""
	}

	value := c.buf.String()
	if !c.truncated {
		return value
	}

	return value + subprocessCaptureTruncate
}

func (c *limitedSubprocessCapture) Len() int {
	if c == nil {
		return 0
	}

	return c.buf.Len()
}

func (c *limitedSubprocessCapture) Truncated() bool {
	return c != nil && c.truncated
}

func subprocessCaptureSummary(capture *limitedSubprocessCapture) string {
	if capture == nil || capture.Len() == 0 {
		return "redacted output (0 bytes)"
	}
	if capture.Truncated() {
		return fmt.Sprintf("redacted output (%d+ bytes, truncated)", capture.Len())
	}

	return fmt.Sprintf("redacted output (%d bytes)", capture.Len())
}
