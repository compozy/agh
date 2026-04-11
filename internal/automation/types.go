package automation

import modelpkg "github.com/pedronauck/agh/internal/automation/model"

// DefaultTimezone is the default schedule timezone used by automation config.
const DefaultTimezone = modelpkg.DefaultTimezone

// DefaultMaxConcurrentJobs is the default global automation concurrency limit.
const DefaultMaxConcurrentJobs = modelpkg.DefaultMaxConcurrentJobs

// AutomationScope identifies the visibility boundary of an automation resource.
type AutomationScope = modelpkg.AutomationScope

const (
	// AutomationScopeGlobal targets daemon-wide automation without a workspace binding.
	AutomationScopeGlobal = modelpkg.AutomationScopeGlobal
	// AutomationScopeWorkspace targets automation bound to a specific workspace.
	AutomationScopeWorkspace = modelpkg.AutomationScopeWorkspace
)

// JobSource identifies where a job or trigger definition originated.
type JobSource = modelpkg.JobSource

const (
	// JobSourceConfig identifies a TOML-backed automation definition.
	JobSourceConfig = modelpkg.JobSourceConfig
	// JobSourceDynamic identifies a runtime-created automation definition.
	JobSourceDynamic = modelpkg.JobSourceDynamic
)

// ScheduleMode identifies how a scheduled job determines its next fire time.
type ScheduleMode = modelpkg.ScheduleMode

const (
	// ScheduleModeCron evaluates a cron expression.
	ScheduleModeCron = modelpkg.ScheduleModeCron
	// ScheduleModeEvery evaluates a Go duration interval.
	ScheduleModeEvery = modelpkg.ScheduleModeEvery
	// ScheduleModeAt evaluates a one-shot RFC3339 timestamp.
	ScheduleModeAt = modelpkg.ScheduleModeAt
)

// RetryStrategy identifies how failed runs should be retried.
type RetryStrategy = modelpkg.RetryStrategy

const (
	// RetryStrategyNone disables retries after a failed run.
	RetryStrategyNone = modelpkg.RetryStrategyNone
	// RetryStrategyBackoff retries failed runs with exponential backoff.
	RetryStrategyBackoff = modelpkg.RetryStrategyBackoff
)

// RunStatus identifies the current lifecycle state of an automation run.
type RunStatus = modelpkg.RunStatus

const (
	// RunScheduled reports a run that has been accepted but not yet started.
	RunScheduled = modelpkg.RunScheduled
	// RunRunning reports a run that is actively dispatching or executing.
	RunRunning = modelpkg.RunRunning
	// RunCompleted reports a run that finished successfully.
	RunCompleted = modelpkg.RunCompleted
	// RunFailed reports a run that finished with an error.
	RunFailed = modelpkg.RunFailed
	// RunCancelled reports a run that was cancelled before completion.
	RunCancelled = modelpkg.RunCancelled
)

// ActivationSource identifies which ingress path produced an activation envelope.
type ActivationSource = modelpkg.ActivationSource

const (
	// ActivationSourceObserver identifies observer-backed trigger ingress.
	ActivationSourceObserver = modelpkg.ActivationSourceObserver
	// ActivationSourceHook identifies hook-backed trigger ingress.
	ActivationSourceHook = modelpkg.ActivationSourceHook
	// ActivationSourceWebhook identifies external webhook ingress.
	ActivationSourceWebhook = modelpkg.ActivationSourceWebhook
	// ActivationSourceExtension identifies extension-provided ingress.
	ActivationSourceExtension = modelpkg.ActivationSourceExtension
)

// Job is the canonical scheduled automation definition used by runtime and storage layers.
type Job = modelpkg.Job

// ScheduleSpec describes how a job should be scheduled.
type ScheduleSpec = modelpkg.ScheduleSpec

// Trigger is the canonical event-driven automation definition used by runtime and storage layers.
type Trigger = modelpkg.Trigger

// RetryConfig defines retry behavior for a failed automation run.
type RetryConfig = modelpkg.RetryConfig

// FireLimitConfig caps how often a job or trigger may fire within a rolling window.
type FireLimitConfig = modelpkg.FireLimitConfig

// Run records the execution state of a single automation fire.
type Run = modelpkg.Run

// ActivationEnvelope is the normalized trigger input regardless of source.
type ActivationEnvelope = modelpkg.ActivationEnvelope
