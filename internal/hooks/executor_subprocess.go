package hooks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	exec "os/exec"
	"sort"
	"strings"
	"time"

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

// SubprocessExecutor runs hooks through a local shell command boundary.
type SubprocessExecutor struct {
	command string
	args    []string
	dir     string
	env     map[string]string
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
	cmd.Env = subprocessProcessEnv(e.env)
	cmd.Stdin = bytes.NewReader(payload)

	stdout := newLimitedSubprocessCapture()
	stderr := newLimitedSubprocessCapture()
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err = runSubprocessCommand(hookCtx, cmd)
	output := []byte(stdout.String())
	if err != nil {
		return output, subprocessRunError(hookCtx, timeout, err, stderr)
	}

	return output, nil
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

func runSubprocessCommand(ctx context.Context, cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	select {
	case err := <-waitCh:
		return err
	case <-ctx.Done():
		terminateErr := terminateSubprocessCommand(cmd)
		timer := time.NewTimer(subprocessShutdownGrace)
		defer timer.Stop()

		select {
		case err := <-waitCh:
			return errors.Join(terminateErr, err, forceSubprocessCommandExit(cmd, subprocessProcessGroupWait))
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
