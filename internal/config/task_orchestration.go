package config

import (
	"fmt"
	"time"
)

const (
	// MaxTaskOrchestrationRuntime is the largest accepted runtime watchdog budget.
	MaxTaskOrchestrationRuntime = 24 * time.Hour

	TaskCoordinatorModeInherit = "inherit"
	TaskCoordinatorModeGuided  = "guided"
	TaskWorkerModeInherit      = "inherit"
	TaskSandboxModeInherit     = "inherit"
	TaskSandboxModeNone        = "none"
	TaskReviewPolicyNone       = "none"
	TaskReviewPolicyOnSuccess  = "on_success"
	TaskReviewPolicyOnFailure  = "on_failure"
	TaskReviewPolicyAlways     = "always"
	TaskReviewFailureBlockTask = "block_task"
	TaskReviewFailureFailTask  = "fail_task"
)

// TaskConfig controls task runtime behavior.
type TaskConfig struct {
	Orchestration TaskOrchestrationConfig `toml:"orchestration"`
	Recovery      TaskRecoveryConfig      `toml:"recovery"`
}

// TaskRecoveryConfig controls task-run recovery verbs.
type TaskRecoveryConfig struct {
	AllowAgentForce bool `toml:"allow_agent_force"`
}

// TaskOrchestrationConfig controls bounded task orchestration behavior.
type TaskOrchestrationConfig struct {
	SummaryMaxBytes           int                            `toml:"summary_max_bytes"`
	ContextBodyMaxBytes       int                            `toml:"context_body_max_bytes"`
	ContextPriorAttempts      int                            `toml:"context_prior_attempts"`
	ContextRecentEvents       int                            `toml:"context_recent_events"`
	SpawnFailureLimit         int                            `toml:"spawn_failure_limit"`
	SchedulerBadTickThreshold int                            `toml:"scheduler_bad_tick_threshold"`
	SchedulerBadTickCooldown  time.Duration                  `toml:"scheduler_bad_tick_cooldown"`
	DefaultMaxRuntime         time.Duration                  `toml:"default_max_runtime"`
	BridgeNotificationTimeout time.Duration                  `toml:"bridge_notification_timeout"`
	Profile                   TaskOrchestrationProfileConfig `toml:"profile"`
	Review                    TaskOrchestrationReviewConfig  `toml:"review"`
}

// TaskOrchestrationProfileConfig controls task execution profile defaults and gates.
type TaskOrchestrationProfileConfig struct {
	DefaultCoordinatorMode    string `toml:"default_coordinator_mode"`
	DefaultWorkerMode         string `toml:"default_worker_mode"`
	DefaultSandboxMode        string `toml:"default_sandbox_mode"`
	AllowTaskProviderOverride bool   `toml:"allow_task_provider_override"`
	AllowTaskSandboxNone      bool   `toml:"allow_task_sandbox_none"`
}

// TaskOrchestrationReviewConfig controls task review gate defaults and bounds.
type TaskOrchestrationReviewConfig struct {
	DefaultPolicy             string        `toml:"default_policy"`
	MaxRounds                 int           `toml:"max_rounds"`
	MaxReviewAttempts         int           `toml:"max_review_attempts"`
	Timeout                   time.Duration `toml:"timeout"`
	RapidTerminalWindow       time.Duration `toml:"rapid_terminal_window"`
	RapidTerminalLimit        int           `toml:"rapid_terminal_limit"`
	MissingWorkMaxItems       int           `toml:"missing_work_max_items"`
	MissingWorkItemMaxBytes   int           `toml:"missing_work_item_max_bytes"`
	ReasonMaxBytes            int           `toml:"reason_max_bytes"`
	ReviewTextMaxBytes        int           `toml:"review_text_max_bytes"`
	NextRoundGuidanceMaxBytes int           `toml:"next_round_guidance_max_bytes"`
	FailurePolicy             string        `toml:"failure_policy"`
}

// DefaultTaskConfig returns built-in task runtime defaults.
func DefaultTaskConfig() TaskConfig {
	return TaskConfig{
		Orchestration: TaskOrchestrationConfig{
			SummaryMaxBytes:           4096,
			ContextBodyMaxBytes:       8192,
			ContextPriorAttempts:      5,
			ContextRecentEvents:       50,
			SpawnFailureLimit:         5,
			SchedulerBadTickThreshold: 6,
			SchedulerBadTickCooldown:  5 * time.Minute,
			DefaultMaxRuntime:         0,
			BridgeNotificationTimeout: 10 * time.Second,
			Profile: TaskOrchestrationProfileConfig{
				DefaultCoordinatorMode:    TaskCoordinatorModeInherit,
				DefaultWorkerMode:         TaskWorkerModeInherit,
				DefaultSandboxMode:        TaskSandboxModeInherit,
				AllowTaskProviderOverride: true,
				AllowTaskSandboxNone:      true,
			},
			Review: TaskOrchestrationReviewConfig{
				DefaultPolicy:             TaskReviewPolicyNone,
				MaxRounds:                 3,
				MaxReviewAttempts:         2,
				Timeout:                   20 * time.Minute,
				RapidTerminalWindow:       2 * time.Minute,
				RapidTerminalLimit:        3,
				MissingWorkMaxItems:       20,
				MissingWorkItemMaxBytes:   512,
				ReasonMaxBytes:            2048,
				ReviewTextMaxBytes:        12000,
				NextRoundGuidanceMaxBytes: 4096,
				FailurePolicy:             TaskReviewFailureBlockTask,
			},
		},
		Recovery: TaskRecoveryConfig{
			AllowAgentForce: true,
		},
	}
}

// Validate ensures task config is safe to consume.
func (c TaskConfig) Validate() error {
	return c.Orchestration.Validate("task.orchestration")
}

// Validate ensures task orchestration config is safe to consume.
func (c TaskOrchestrationConfig) Validate(path string) error {
	if c.SummaryMaxBytes <= 0 {
		return fmt.Errorf("%s.summary_max_bytes must be positive: %d", path, c.SummaryMaxBytes)
	}
	if c.ContextBodyMaxBytes <= 0 {
		return fmt.Errorf("%s.context_body_max_bytes must be positive: %d", path, c.ContextBodyMaxBytes)
	}
	if c.ContextPriorAttempts < 0 {
		return fmt.Errorf("%s.context_prior_attempts must be >= 0: %d", path, c.ContextPriorAttempts)
	}
	if c.ContextRecentEvents < 0 {
		return fmt.Errorf("%s.context_recent_events must be >= 0: %d", path, c.ContextRecentEvents)
	}
	if c.SpawnFailureLimit <= 0 {
		return fmt.Errorf("%s.spawn_failure_limit must be positive: %d", path, c.SpawnFailureLimit)
	}
	if c.SchedulerBadTickThreshold <= 0 {
		return fmt.Errorf(
			"%s.scheduler_bad_tick_threshold must be positive: %d",
			path,
			c.SchedulerBadTickThreshold,
		)
	}
	if err := validateWholeSecondDuration(
		path+".scheduler_bad_tick_cooldown",
		c.SchedulerBadTickCooldown,
		false,
	); err != nil {
		return err
	}
	if err := validateWholeSecondDuration(path+".default_max_runtime", c.DefaultMaxRuntime, true); err != nil {
		return err
	}
	if c.DefaultMaxRuntime > MaxTaskOrchestrationRuntime {
		return fmt.Errorf(
			"%s.default_max_runtime must be <= %s: %s",
			path,
			MaxTaskOrchestrationRuntime,
			c.DefaultMaxRuntime,
		)
	}
	if err := validateWholeSecondDuration(
		path+".bridge_notification_timeout",
		c.BridgeNotificationTimeout,
		false,
	); err != nil {
		return err
	}
	if err := c.Profile.Validate(path + ".profile"); err != nil {
		return err
	}
	return c.Review.Validate(path + ".review")
}

// Validate ensures task execution profile defaults are recognized.
func (c TaskOrchestrationProfileConfig) Validate(path string) error {
	switch c.DefaultCoordinatorMode {
	case TaskCoordinatorModeInherit, TaskCoordinatorModeGuided:
	default:
		return fmt.Errorf(
			"%s.default_coordinator_mode must be %q or %q: %q",
			path,
			TaskCoordinatorModeInherit,
			TaskCoordinatorModeGuided,
			c.DefaultCoordinatorMode,
		)
	}
	if c.DefaultWorkerMode != TaskWorkerModeInherit {
		return fmt.Errorf("%s.default_worker_mode must be %q: %q", path, TaskWorkerModeInherit, c.DefaultWorkerMode)
	}
	switch c.DefaultSandboxMode {
	case TaskSandboxModeInherit, TaskSandboxModeNone:
	default:
		return fmt.Errorf(
			"%s.default_sandbox_mode must be %q or %q: %q",
			path,
			TaskSandboxModeInherit,
			TaskSandboxModeNone,
			c.DefaultSandboxMode,
		)
	}
	if c.DefaultSandboxMode == TaskSandboxModeNone && !c.AllowTaskSandboxNone {
		return fmt.Errorf("%s.default_sandbox_mode %q requires allow_task_sandbox_none", path, TaskSandboxModeNone)
	}
	return nil
}

// Validate ensures review gate defaults are bounded.
func (c TaskOrchestrationReviewConfig) Validate(path string) error {
	switch c.DefaultPolicy {
	case TaskReviewPolicyNone, TaskReviewPolicyOnSuccess, TaskReviewPolicyOnFailure, TaskReviewPolicyAlways:
	default:
		return fmt.Errorf("%s.default_policy is invalid: %q", path, c.DefaultPolicy)
	}
	if c.MaxRounds <= 0 {
		return fmt.Errorf("%s.max_rounds must be positive: %d", path, c.MaxRounds)
	}
	if c.MaxReviewAttempts <= 0 {
		return fmt.Errorf("%s.max_review_attempts must be positive: %d", path, c.MaxReviewAttempts)
	}
	if err := validateWholeSecondDuration(path+".timeout", c.Timeout, false); err != nil {
		return err
	}
	if err := validateWholeSecondDuration(path+".rapid_terminal_window", c.RapidTerminalWindow, false); err != nil {
		return err
	}
	if c.RapidTerminalLimit <= 0 {
		return fmt.Errorf("%s.rapid_terminal_limit must be positive: %d", path, c.RapidTerminalLimit)
	}
	if c.MissingWorkMaxItems <= 0 {
		return fmt.Errorf("%s.missing_work_max_items must be positive: %d", path, c.MissingWorkMaxItems)
	}
	if c.MissingWorkItemMaxBytes <= 0 {
		return fmt.Errorf("%s.missing_work_item_max_bytes must be positive: %d", path, c.MissingWorkItemMaxBytes)
	}
	if c.ReasonMaxBytes <= 0 {
		return fmt.Errorf("%s.reason_max_bytes must be positive: %d", path, c.ReasonMaxBytes)
	}
	if c.ReviewTextMaxBytes <= 0 {
		return fmt.Errorf("%s.review_text_max_bytes must be positive: %d", path, c.ReviewTextMaxBytes)
	}
	if c.NextRoundGuidanceMaxBytes <= 0 {
		return fmt.Errorf("%s.next_round_guidance_max_bytes must be positive: %d", path, c.NextRoundGuidanceMaxBytes)
	}
	switch c.FailurePolicy {
	case TaskReviewFailureBlockTask, TaskReviewFailureFailTask:
	default:
		return fmt.Errorf("%s.failure_policy is invalid: %q", path, c.FailurePolicy)
	}
	return nil
}

func validateWholeSecondDuration(path string, value time.Duration, allowZero bool) error {
	if value < 0 {
		return fmt.Errorf("%s must be >= 0: %s", path, value)
	}
	if value == 0 {
		if allowZero {
			return nil
		}
		return fmt.Errorf("%s must be positive: %s", path, value)
	}
	if value%time.Second != 0 {
		return fmt.Errorf("%s must use whole-second precision: %s", path, value)
	}
	return nil
}
