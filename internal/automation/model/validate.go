package model

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	cron "github.com/robfig/cron/v3"
)

const (
	defaultRetryMaxRetries = 3
	defaultRetryBaseDelay  = "2s"
	defaultFireLimitMax    = 12
	defaultFireLimitWindow = "1h"
	webhookIDPrefix        = "wbh_"
)

var standardCronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

var jobTaskNetworkChannelPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// DefaultRetryConfig returns the default retry policy for automation definitions.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{Strategy: RetryStrategyNone}
}

// DefaultBackoffRetryConfig returns the default exponential backoff retry policy.
func DefaultBackoffRetryConfig() RetryConfig {
	return RetryConfig{
		Strategy:   RetryStrategyBackoff,
		MaxRetries: defaultRetryMaxRetries,
		BaseDelay:  defaultRetryBaseDelay,
	}
}

// DefaultFireLimitConfig returns the default rolling fire-limit policy.
func DefaultFireLimitConfig() FireLimitConfig {
	return FireLimitConfig{
		Max:    defaultFireLimitMax,
		Window: defaultFireLimitWindow,
	}
}

// Validate ensures the scope is one of the supported automation scope values.
func (s AutomationScope) Validate(path string) error {
	switch s {
	case AutomationScopeGlobal, AutomationScopeWorkspace:
		return nil
	default:
		return fmt.Errorf("%s must be one of %q or %q: %q", path, AutomationScopeGlobal, AutomationScopeWorkspace, s)
	}
}

// Validate ensures the source is one of the supported automation source values.
func (s JobSource) Validate(path string) error {
	switch s {
	case JobSourceConfig, JobSourceDynamic:
		return nil
	default:
		return fmt.Errorf("%s must be one of %q or %q: %q", path, JobSourceConfig, JobSourceDynamic, s)
	}
}

// Validate ensures the run status is one of the supported lifecycle states.
func (s RunStatus) Validate(path string) error {
	switch s {
	case RunScheduled, RunRunning, RunDelegated, RunCompleted, RunFailed, RunCancelled:
		return nil
	default:
		return fmt.Errorf("%s must be one of %q, %q, %q, %q, %q, or %q: %q", path, RunScheduled, RunRunning, RunDelegated, RunCompleted, RunFailed, RunCancelled, s)
	}
}

// Validate ensures the activation source is one of the supported ingress values.
func (s ActivationSource) Validate(path string) error {
	switch s {
	case ActivationSourceObserver, ActivationSourceHook, ActivationSourceWebhook, ActivationSourceExtension:
		return nil
	default:
		return fmt.Errorf("%s must be one of %q, %q, %q, or %q: %q", path, ActivationSourceObserver, ActivationSourceHook, ActivationSourceWebhook, ActivationSourceExtension, s)
	}
}

// Validate ensures the schedule mode is one of the supported scheduling modes.
func (m ScheduleMode) Validate(path string) error {
	switch m {
	case ScheduleModeCron, ScheduleModeEvery, ScheduleModeAt:
		return nil
	default:
		return fmt.Errorf("%s must be one of %q, %q, or %q: %q", path, ScheduleModeCron, ScheduleModeEvery, ScheduleModeAt, m)
	}
}

// Validate ensures the retry strategy is supported.
func (s RetryStrategy) Validate(path string) error {
	switch s {
	case RetryStrategyNone, RetryStrategyBackoff:
		return nil
	default:
		return fmt.Errorf("%s must be one of %q or %q: %q", path, RetryStrategyNone, RetryStrategyBackoff, s)
	}
}

// ValidateScopeBinding enforces the global/workspace binding invariants shared by jobs, triggers, and envelopes.
func ValidateScopeBinding(scope AutomationScope, workspaceBinding string, path string, workspaceField string) error {
	scopePath := nestedPath(path, "scope")
	if err := scope.Validate(scopePath); err != nil {
		return err
	}

	workspacePath := nestedPath(path, workspaceField)
	switch scope {
	case AutomationScopeGlobal:
		if strings.TrimSpace(workspaceBinding) != "" {
			return fmt.Errorf("%s must be empty when %s is %q", workspacePath, scopePath, AutomationScopeGlobal)
		}
	case AutomationScopeWorkspace:
		if strings.TrimSpace(workspaceBinding) == "" {
			return fmt.Errorf("%s is required when %s is %q", workspacePath, scopePath, AutomationScopeWorkspace)
		}
	}

	return nil
}

// Validate ensures the schedule spec matches the selected mode and has a valid expression payload.
func (s ScheduleSpec) Validate(path string) error {
	if err := s.Mode.Validate(nestedPath(path, "mode")); err != nil {
		return err
	}

	switch s.Mode {
	case ScheduleModeCron:
		if strings.TrimSpace(s.Expr) == "" {
			return errors.New(nestedPath(path, "expr") + " is required when schedule.mode is \"cron\"")
		}
		if strings.TrimSpace(s.Interval) != "" {
			return errors.New(nestedPath(path, "interval") + " must be empty when schedule.mode is \"cron\"")
		}
		if strings.TrimSpace(s.Time) != "" {
			return errors.New(nestedPath(path, "time") + " must be empty when schedule.mode is \"cron\"")
		}
		if _, err := standardCronParser.Parse(strings.TrimSpace(s.Expr)); err != nil {
			return fmt.Errorf("%s is invalid: %w", nestedPath(path, "expr"), err)
		}
	case ScheduleModeEvery:
		if strings.TrimSpace(s.Interval) == "" {
			return errors.New(nestedPath(path, "interval") + " is required when schedule.mode is \"every\"")
		}
		if strings.TrimSpace(s.Expr) != "" {
			return errors.New(nestedPath(path, "expr") + " must be empty when schedule.mode is \"every\"")
		}
		if strings.TrimSpace(s.Time) != "" {
			return errors.New(nestedPath(path, "time") + " must be empty when schedule.mode is \"every\"")
		}
		interval, err := time.ParseDuration(strings.TrimSpace(s.Interval))
		if err != nil {
			return fmt.Errorf("%s is invalid: %w", nestedPath(path, "interval"), err)
		}
		if interval <= 0 {
			return fmt.Errorf("%s must be positive: %s", nestedPath(path, "interval"), interval)
		}
	case ScheduleModeAt:
		if strings.TrimSpace(s.Time) == "" {
			return errors.New(nestedPath(path, "time") + " is required when schedule.mode is \"at\"")
		}
		if strings.TrimSpace(s.Expr) != "" {
			return errors.New(nestedPath(path, "expr") + " must be empty when schedule.mode is \"at\"")
		}
		if strings.TrimSpace(s.Interval) != "" {
			return errors.New(nestedPath(path, "interval") + " must be empty when schedule.mode is \"at\"")
		}
		if _, err := time.Parse(time.RFC3339, strings.TrimSpace(s.Time)); err != nil {
			return fmt.Errorf("%s is invalid: %w", nestedPath(path, "time"), err)
		}
	}

	return nil
}

// Validate ensures the retry configuration is internally consistent.
func (c RetryConfig) Validate(path string) error {
	if err := c.Strategy.Validate(nestedPath(path, "strategy")); err != nil {
		return err
	}

	switch c.Strategy {
	case RetryStrategyNone:
		if c.MaxRetries != 0 {
			return fmt.Errorf("%s must be zero when retry.strategy is %q: %d", nestedPath(path, "max_retries"), RetryStrategyNone, c.MaxRetries)
		}
		if strings.TrimSpace(c.BaseDelay) != "" {
			return fmt.Errorf("%s must be empty when retry.strategy is %q: %q", nestedPath(path, "base_delay"), RetryStrategyNone, c.BaseDelay)
		}
	case RetryStrategyBackoff:
		if c.MaxRetries <= 0 {
			return fmt.Errorf("%s must be positive when retry.strategy is %q: %d", nestedPath(path, "max_retries"), RetryStrategyBackoff, c.MaxRetries)
		}
		if strings.TrimSpace(c.BaseDelay) == "" {
			return errors.New(nestedPath(path, "base_delay") + " is required when retry.strategy is \"backoff\"")
		}
		delay, err := time.ParseDuration(strings.TrimSpace(c.BaseDelay))
		if err != nil {
			return fmt.Errorf("%s is invalid: %w", nestedPath(path, "base_delay"), err)
		}
		if delay <= 0 {
			return fmt.Errorf("%s must be positive: %s", nestedPath(path, "base_delay"), delay)
		}
	}

	return nil
}

// Validate ensures the rolling fire-limit configuration is internally consistent.
func (c FireLimitConfig) Validate(path string) error {
	if c.Max <= 0 {
		return fmt.Errorf("%s must be positive: %d", nestedPath(path, "max"), c.Max)
	}
	if strings.TrimSpace(c.Window) == "" {
		return errors.New(nestedPath(path, "window") + " is required")
	}

	window, err := time.ParseDuration(strings.TrimSpace(c.Window))
	if err != nil {
		return fmt.Errorf("%s is invalid: %w", nestedPath(path, "window"), err)
	}
	if window <= 0 {
		return fmt.Errorf("%s must be positive: %s", nestedPath(path, "window"), window)
	}

	return nil
}

// Validate ensures the scheduled job definition is internally consistent.
func (j Job) Validate(path string) error {
	if strings.TrimSpace(j.Name) == "" {
		return errors.New(nestedPath(path, "name") + " is required")
	}
	if j.Task == nil && strings.TrimSpace(j.AgentName) == "" {
		return errors.New(nestedPath(path, "agent_name") + " is required")
	}
	if j.Task == nil && strings.TrimSpace(j.Prompt) == "" {
		return errors.New(nestedPath(path, "prompt") + " is required")
	}
	if err := ValidateScopeBinding(j.Scope, j.WorkspaceID, path, "workspace_id"); err != nil {
		return err
	}
	if err := j.Source.Validate(nestedPath(path, "source")); err != nil {
		return err
	}
	if j.Schedule == nil {
		return errors.New(nestedPath(path, "schedule") + " is required")
	}
	if err := j.Schedule.Validate(nestedPath(path, "schedule")); err != nil {
		return err
	}
	if err := j.Retry.Validate(nestedPath(path, "retry")); err != nil {
		return err
	}
	if err := j.FireLimit.Validate(nestedPath(path, "fire_limit")); err != nil {
		return err
	}
	if j.Task != nil {
		if err := j.Task.Validate(nestedPath(path, "task")); err != nil {
			return err
		}
		if j.Retry.Strategy != RetryStrategyNone {
			return fmt.Errorf("%s.strategy must be %q when %s is configured", nestedPath(path, "retry"), RetryStrategyNone, nestedPath(path, "task"))
		}
	}

	return nil
}

// Validate ensures the trigger definition is internally consistent.
func (t Trigger) Validate(path string) error {
	if strings.TrimSpace(t.Name) == "" {
		return errors.New(nestedPath(path, "name") + " is required")
	}
	if strings.TrimSpace(t.AgentName) == "" {
		return errors.New(nestedPath(path, "agent_name") + " is required")
	}
	if strings.TrimSpace(t.Prompt) == "" {
		return errors.New(nestedPath(path, "prompt") + " is required")
	}
	if strings.TrimSpace(t.Event) == "" {
		return errors.New(nestedPath(path, "event") + " is required")
	}
	if err := ValidateScopeBinding(t.Scope, t.WorkspaceID, path, "workspace_id"); err != nil {
		return err
	}
	if err := t.Source.Validate(nestedPath(path, "source")); err != nil {
		return err
	}
	if err := t.Retry.Validate(nestedPath(path, "retry")); err != nil {
		return err
	}
	if err := t.FireLimit.Validate(nestedPath(path, "fire_limit")); err != nil {
		return err
	}
	if err := ValidateTriggerFilter(t.Filter, nestedPath(path, "filter")); err != nil {
		return err
	}
	if err := ValidateTriggerPromptTemplate(t.Prompt); err != nil {
		return fmt.Errorf("%s is invalid: %w", nestedPath(path, "prompt"), err)
	}
	if strings.TrimSpace(t.Event) == "webhook" {
		if strings.TrimSpace(t.EndpointSlug) == "" && strings.TrimSpace(t.WebhookID) == "" {
			return errors.New(nestedPath(path, "endpoint_slug") + " or " + nestedPath(path, "webhook_id") + " is required when event is \"webhook\"")
		}
		if webhookID := strings.TrimSpace(t.WebhookID); webhookID != "" && !strings.HasPrefix(webhookID, webhookIDPrefix) {
			return fmt.Errorf("%s must start with %q: %q", nestedPath(path, "webhook_id"), webhookIDPrefix, webhookID)
		}
		return nil
	}
	if strings.TrimSpace(t.EndpointSlug) != "" {
		return fmt.Errorf("%s must be empty when event is %q", nestedPath(path, "endpoint_slug"), strings.TrimSpace(t.Event))
	}
	if strings.TrimSpace(t.WebhookID) != "" {
		return fmt.Errorf("%s must be empty when event is %q", nestedPath(path, "webhook_id"), strings.TrimSpace(t.Event))
	}

	return nil
}

// Validate ensures the run record is internally consistent.
func (r Run) Validate(path string) error {
	if err := r.Status.Validate(nestedPath(path, "status")); err != nil {
		return err
	}
	if r.Attempt < 0 {
		return fmt.Errorf("%s must be zero or positive: %d", nestedPath(path, "attempt"), r.Attempt)
	}
	if r.StartedAt != nil && r.EndedAt != nil && r.EndedAt.Before(*r.StartedAt) {
		return fmt.Errorf("%s must not be before %s", nestedPath(path, "ended_at"), nestedPath(path, "started_at"))
	}
	if r.Status == RunDelegated {
		if strings.TrimSpace(r.TaskID) == "" {
			return fmt.Errorf("%s is required when %s is %q", nestedPath(path, "task_id"), nestedPath(path, "status"), RunDelegated)
		}
		if strings.TrimSpace(r.TaskRunID) == "" {
			return fmt.Errorf("%s is required when %s is %q", nestedPath(path, "task_run_id"), nestedPath(path, "status"), RunDelegated)
		}
	}
	return nil
}

// Validate ensures the direct task materialization configuration is internally consistent.
func (c JobTaskConfig) Validate(path string) error {
	if channel := strings.TrimSpace(c.NetworkChannel); channel != "" {
		if !jobTaskNetworkChannelPattern.MatchString(channel) {
			return fmt.Errorf("%s is invalid: channel=%q", nestedPath(path, "network_channel"), channel)
		}
	}
	if c.Owner != nil {
		if err := c.Owner.Validate(nestedPath(path, "owner")); err != nil {
			return err
		}
	}
	return nil
}

// Validate ensures the normalized activation envelope satisfies the shared scope and source invariants.
func (e ActivationEnvelope) Validate(path string) error {
	if strings.TrimSpace(e.Kind) == "" {
		return errors.New(nestedPath(path, "kind") + " is required")
	}
	if err := ValidateScopeBinding(e.Scope, e.WorkspaceID, path, "workspace_id"); err != nil {
		return err
	}
	if err := e.Source.Validate(nestedPath(path, "source")); err != nil {
		return err
	}
	return nil
}

// ValidateTriggerFilter ensures trigger filters only reference supported activation-envelope field paths.
func ValidateTriggerFilter(filter map[string]string, path string) error {
	for rawKey, rawValue := range filter {
		key := strings.TrimSpace(rawKey)
		value := strings.TrimSpace(rawValue)
		if key == "" {
			return errors.New(path + " contains an empty field path")
		}
		if value == "" {
			return fmt.Errorf("%s[%q] must not be empty", path, rawKey)
		}
		if err := validateTriggerFilterPath(key); err != nil {
			return fmt.Errorf("%s[%q]: %w", path, rawKey, err)
		}
	}
	return nil
}

func validateTriggerFilterPath(path string) error {
	switch path {
	case "kind", "scope", "workspace_id", "source":
		return nil
	}
	if strings.HasPrefix(path, "data.") && strings.TrimSpace(strings.TrimPrefix(path, "data.")) != "" {
		return nil
	}
	return fmt.Errorf("unsupported filter path %q", path)
}

func nestedPath(path string, field string) string {
	trimmedPath := strings.TrimSpace(path)
	trimmedField := strings.TrimSpace(field)
	switch {
	case trimmedPath == "":
		return trimmedField
	case trimmedField == "":
		return trimmedPath
	default:
		return trimmedPath + "." + trimmedField
	}
}
