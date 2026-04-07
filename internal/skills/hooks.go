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
)

const defaultHookTimeout = 5 * time.Second

// HookRunner dispatches subprocess hooks for skill lifecycle events.
type HookRunner struct {
	logger *slog.Logger
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

// NewHookRunner constructs a HookRunner with the supplied logger.
func NewHookRunner(logger *slog.Logger) *HookRunner {
	if logger == nil {
		logger = slog.Default()
	}

	return &HookRunner{logger: logger}
}

// RunHooks executes all hooks matching the given event, in precedence order.
func (hr *HookRunner) RunHooks(ctx context.Context, event HookEvent, skills []*Skill, payload HookPayload) []HookResult {
	if len(skills) == 0 {
		return nil
	}
	if hr == nil {
		hr = NewHookRunner(nil)
	}
	if hr.logger == nil {
		hr.logger = slog.Default()
	}

	ordered := orderSkillsForHooks(skills, event)
	if len(ordered) == 0 {
		return nil
	}

	payload.Event = string(event)
	results := make([]HookResult, 0)
	for _, skill := range ordered {
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
		hr.logHookFailure(skill, hook, result)
		return result
	}

	stdinPayload, err := json.Marshal(payload)
	if err != nil {
		result.Error = fmt.Errorf("marshal hook payload: %w", err)
		result.Duration = time.Since(started)
		hr.logHookFailure(skill, hook, result)
		return result
	}

	timeout := hookTimeout(hook.Timeout)
	hookCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(hookCtx, hook.Command, hook.Args...)
	cmd.Stdin = bytes.NewReader(stdinPayload)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if len(hook.Env) > 0 {
		cmd.Env = append(os.Environ(), hookEnv(hook.Env)...)
	}

	err = cmd.Run()
	result.Output = stdout.String()
	if err == nil {
		result.Duration = time.Since(started)
		return result
	}

	result.Error = hookRunError(hookCtx, timeout, err, stderr.String())
	result.Duration = time.Since(started)
	hr.logHookFailure(skill, hook, result)
	return result
}

func (hr *HookRunner) logHookFailure(skill *Skill, hook HookDecl, result HookResult) {
	hr.logger.Warn(
		"hook execution failed",
		"skill_name", result.SkillName,
		"event", hook.Event,
		"source", skillSourceName(skill.Source),
		"command", hook.Command,
		"duration", result.Duration,
		"output", result.Output,
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

func hookTimeout(timeout time.Duration) time.Duration {
	if timeout > 0 {
		return timeout
	}

	return defaultHookTimeout
}

func hookEnv(env map[string]string) []string {
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, key+"="+env[key])
	}

	return values
}

func hookRunError(ctx context.Context, timeout time.Duration, err error, stderr string) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("hook timed out after %s: %w", timeout, ctx.Err())
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return fmt.Errorf("hook canceled: %w", ctx.Err())
	}

	trimmed := strings.TrimSpace(stderr)
	if trimmed == "" {
		return fmt.Errorf("hook command failed: %w", err)
	}

	return fmt.Errorf("hook command failed: %w (stderr: %s)", err, trimmed)
}
