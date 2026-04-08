package skills

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

const (
	defaultHookTimeout      = 5 * time.Second
	hookCaptureLimitBytes   = 8 * 1024
	hookCaptureTruncateNote = "...[truncated]"
	hookShutdownGracePeriod = 250 * time.Millisecond
)

var hookEnvAllowlist = []string{
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

// HookRunner dispatches subprocess hooks for skill lifecycle events.
type HookRunner struct {
	allowedMarketplaceHooks []string
	logger                  *slog.Logger
}

// HookPayload is the JSON payload written to hook stdin.
type HookPayload struct {
	SessionID string `json:"session_id"`
	AgentName string `json:"agent_name"`
	Workspace string `json:"workspace"`
	Event     string `json:"event"`
}

// HookResult captures the outcome of a single hook execution.
type HookResult struct {
	SkillName string
	Event     HookEvent
	Output    string
	Error     error
	Duration  time.Duration
}

// NewHookRunner constructs a HookRunner with the supplied config and logger.
func NewHookRunner(cfg aghconfig.SkillsConfig, logger *slog.Logger) *HookRunner {
	if logger == nil {
		logger = slog.Default()
	}

	return &HookRunner{
		allowedMarketplaceHooks: cloneStrings(cfg.AllowedMarketplaceHooks),
		logger:                  logger,
	}
}

// RunHooks executes all hooks matching the given event, in precedence order.
func (hr *HookRunner) RunHooks(ctx context.Context, event HookEvent, skills []*Skill, payload HookPayload) []HookResult {
	if len(skills) == 0 {
		return nil
	}
	if hr == nil {
		hr = NewHookRunner(aghconfig.SkillsConfig{}, nil)
	}
	if hr.logger == nil {
		hr.logger = slog.Default()
	}

	ordered := orderSkillsForHooks(skills, event)
	if len(ordered) == 0 {
		return nil
	}

	allowedMarketplace := marketplaceAllowlist(hr.allowedMarketplaceHooks)
	payload.Event = string(event)
	results := make([]HookResult, 0)
	for _, skill := range ordered {
		if !marketplaceSkillAllowed(skill, allowedMarketplace) {
			hr.logger.Warn(
				"blocked hook",
				"skill_name", skill.Meta.Name,
				"event", event,
				"source", skillSourceName(skill.Source),
			)
			continue
		}
		for _, hook := range skill.Hooks {
			if hook.Event != event {
				continue
			}

			result := hr.runHook(ctx, skill, hook, payload)
			results = append(results, result)
		}
	}

	if len(results) == 0 {
		return nil
	}

	return results
}

func (hr *HookRunner) runHook(ctx context.Context, skill *Skill, hook HookDecl, payload HookPayload) HookResult {
	result := HookResult{
		SkillName: skillName(skill),
		Event:     hook.Event,
	}

	started := time.Now()

	if strings.TrimSpace(hook.Command) == "" {
		result.Error = errors.New("hook command is required")
		result.Duration = time.Since(started)
		hr.logHookFailure(skill, hook, result, nil, nil)
		return result
	}

	stdinPayload, err := json.Marshal(payload)
	if err != nil {
		result.Error = fmt.Errorf("marshal hook payload: %w", err)
		result.Duration = time.Since(started)
		hr.logHookFailure(skill, hook, result, nil, nil)
		return result
	}

	timeout := hookTimeout(hook.Timeout)
	hookCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.Command(hook.Command, hook.Args...)
	configureHookCommand(cmd)
	cmd.Dir = hookCommandDir(skill)
	cmd.Stdin = bytes.NewReader(stdinPayload)
	cmd.Env = hookProcessEnv(hook.Env)

	stdout := newHookCapture()
	stderr := newHookCapture()
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err = runHookCommand(hookCtx, cmd)
	result.Output = stdout.String()
	if err == nil {
		result.Duration = time.Since(started)
		return result
	}

	result.Error = hookRunError(hookCtx, timeout, err, stderr)
	result.Duration = time.Since(started)
	hr.logHookFailure(skill, hook, result, stdout, stderr)
	return result
}

func runHookCommand(ctx context.Context, cmd *exec.Cmd) error {
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
		_ = terminateHookCommand(cmd)
		timer := time.NewTimer(hookShutdownGracePeriod)
		defer timer.Stop()

		select {
		case err := <-waitCh:
			return err
		case <-timer.C:
			_ = killHookCommand(cmd)
			return <-waitCh
		}
	}
}

func (hr *HookRunner) logHookFailure(skill *Skill, hook HookDecl, result HookResult, stdout hookCapture, stderr hookCapture) {
	hr.logger.Warn(
		"hook execution failed",
		"skill_name", result.SkillName,
		"event", hook.Event,
		"source", skillSourceName(skill.Source),
		"command", hook.Command,
		"duration", result.Duration,
		"stdout", hookCaptureSummary(stdout),
		"stderr", hookCaptureSummary(stderr),
		"error", result.Error,
	)
}

func orderSkillsForHooks(skills []*Skill, event HookEvent) []*Skill {
	ordered := make([]*Skill, 0, len(skills))
	for _, skill := range skills {
		if !skillHasHookEvent(skill, event) {
			continue
		}
		ordered = append(ordered, skill)
	}

	sort.SliceStable(ordered, func(i, j int) bool {
		left := ordered[i]
		right := ordered[j]
		if left.Source != right.Source {
			return left.Source < right.Source
		}

		leftKey := strings.ToLower(skillName(left))
		rightKey := strings.ToLower(skillName(right))
		if leftKey != rightKey {
			return leftKey < rightKey
		}

		leftName := skillName(left)
		rightName := skillName(right)
		if leftName != rightName {
			return leftName < rightName
		}

		return left.FilePath < right.FilePath
	})

	return ordered
}

func skillHasHookEvent(skill *Skill, event HookEvent) bool {
	if skill == nil {
		return false
	}

	for _, hook := range skill.Hooks {
		if hook.Event == event {
			return true
		}
	}

	return false
}

func skillName(skill *Skill) string {
	if skill == nil {
		return ""
	}

	return strings.TrimSpace(skill.Meta.Name)
}

func hookCommandDir(skill *Skill) string {
	if skill == nil {
		return ""
	}

	return strings.TrimSpace(skill.Dir)
}

func hookTimeout(timeout time.Duration) time.Duration {
	if timeout > 0 {
		return timeout
	}

	return defaultHookTimeout
}

func hookProcessEnv(env map[string]string) []string {
	merged := make(map[string]string, len(hookEnvAllowlist)+len(env))
	for _, key := range hookEnvAllowlist {
		if value, ok := os.LookupEnv(key); ok {
			merged[key] = value
		}
	}
	for key, value := range env {
		merged[key] = value
	}

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

func hookRunError(ctx context.Context, timeout time.Duration, err error, stderr hookCapture) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("hook timed out after %s: %w", timeout, ctx.Err())
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return fmt.Errorf("hook canceled: %w", ctx.Err())
	}

	if stderr.Len() == 0 {
		return fmt.Errorf("hook command failed: %w", err)
	}

	return fmt.Errorf("hook command failed: %w (%s)", err, hookCaptureSummary(stderr))
}

type hookCapture interface {
	Write([]byte) (int, error)
	String() string
	Len() int
	Truncated() bool
}

type limitedHookCapture struct {
	buf       bytes.Buffer
	truncated bool
}

func newHookCapture() *limitedHookCapture {
	return &limitedHookCapture{}
}

func (c *limitedHookCapture) Write(p []byte) (int, error) {
	if c == nil {
		return len(p), nil
	}

	remaining := hookCaptureLimitBytes - c.buf.Len()
	switch {
	case remaining <= 0:
		c.truncated = true
	case len(p) > remaining:
		_, _ = c.buf.Write(p[:remaining])
		c.truncated = true
	default:
		_, _ = c.buf.Write(p)
	}

	return len(p), nil
}

func (c *limitedHookCapture) String() string {
	if c == nil {
		return ""
	}

	value := c.buf.String()
	if !c.truncated {
		return value
	}

	return value + hookCaptureTruncateNote
}

func (c *limitedHookCapture) Len() int {
	if c == nil {
		return 0
	}

	return c.buf.Len()
}

func (c *limitedHookCapture) Truncated() bool {
	return c != nil && c.truncated
}

func hookCaptureSummary(capture hookCapture) string {
	if capture == nil || capture.Len() == 0 {
		return "redacted output (0 bytes)"
	}
	if capture.Truncated() {
		return fmt.Sprintf("redacted output (%d+ bytes, truncated)", capture.Len())
	}

	return fmt.Sprintf("redacted output (%d bytes)", capture.Len())
}
