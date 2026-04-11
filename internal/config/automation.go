package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	automationpkg "github.com/pedronauck/agh/internal/automation/model"
)

// AutomationConfig holds TOML-defined automation defaults, jobs, and triggers.
type AutomationConfig struct {
	Enabled           bool                          `toml:"enabled"`
	Timezone          string                        `toml:"timezone,omitempty"`
	MaxConcurrentJobs int                           `toml:"max_concurrent_jobs"`
	DefaultFireLimit  automationpkg.FireLimitConfig `toml:"default_fire_limit"`
	Jobs              []AutomationJob               `toml:"jobs,omitempty"`
	Triggers          []AutomationTrigger           `toml:"triggers,omitempty"`
}

// AutomationJob holds a config-defined scheduled job before workspace resolution.
type AutomationJob struct {
	Scope     automationpkg.AutomationScope `toml:"scope"`
	Name      string                        `toml:"name"`
	AgentName string                        `toml:"agent"`
	Workspace string                        `toml:"workspace,omitempty"`
	Prompt    string                        `toml:"prompt"`
	Schedule  automationpkg.ScheduleSpec    `toml:"schedule"`
	Enabled   bool                          `toml:"enabled"`
	Retry     automationpkg.RetryConfig     `toml:"retry,omitempty"`
	FireLimit automationpkg.FireLimitConfig `toml:"fire_limit,omitempty"`
	Source    automationpkg.JobSource       `toml:"-"`
}

// AutomationTrigger holds a config-defined trigger before workspace resolution.
type AutomationTrigger struct {
	Scope        automationpkg.AutomationScope `toml:"scope"`
	Name         string                        `toml:"name"`
	AgentName    string                        `toml:"agent"`
	Workspace    string                        `toml:"workspace,omitempty"`
	Prompt       string                        `toml:"prompt"`
	Event        string                        `toml:"event"`
	Filter       map[string]string             `toml:"filter,omitempty"`
	Enabled      bool                          `toml:"enabled"`
	Retry        automationpkg.RetryConfig     `toml:"retry,omitempty"`
	FireLimit    automationpkg.FireLimitConfig `toml:"fire_limit,omitempty"`
	Source       automationpkg.JobSource       `toml:"-"`
	EndpointSlug string                        `toml:"endpoint_slug,omitempty"`
}

type automationOverlay struct {
	Enabled           *bool                          `toml:"enabled"`
	Timezone          *string                        `toml:"timezone"`
	MaxConcurrentJobs *int                           `toml:"max_concurrent_jobs"`
	DefaultFireLimit  *automationpkg.FireLimitConfig `toml:"default_fire_limit"`
	Jobs              []parsedAutomationJob          `toml:"jobs"`
	Triggers          []parsedAutomationTrigger      `toml:"triggers"`
}

type parsedAutomationJob struct {
	Scope     automationpkg.AutomationScope  `toml:"scope"`
	Name      string                         `toml:"name"`
	AgentName string                         `toml:"agent"`
	Workspace string                         `toml:"workspace"`
	Prompt    string                         `toml:"prompt"`
	Schedule  *automationpkg.ScheduleSpec    `toml:"schedule"`
	Enabled   *bool                          `toml:"enabled"`
	Retry     *automationpkg.RetryConfig     `toml:"retry"`
	FireLimit *automationpkg.FireLimitConfig `toml:"fire_limit"`
}

type parsedAutomationTrigger struct {
	Scope        automationpkg.AutomationScope  `toml:"scope"`
	Name         string                         `toml:"name"`
	AgentName    string                         `toml:"agent"`
	Workspace    string                         `toml:"workspace"`
	Prompt       string                         `toml:"prompt"`
	Event        string                         `toml:"event"`
	Filter       map[string]string              `toml:"filter"`
	Enabled      *bool                          `toml:"enabled"`
	Retry        *automationpkg.RetryConfig     `toml:"retry"`
	FireLimit    *automationpkg.FireLimitConfig `toml:"fire_limit"`
	EndpointSlug string                         `toml:"endpoint_slug"`
}

// Validate ensures the automation config is internally consistent.
func (c AutomationConfig) Validate() error {
	if strings.TrimSpace(c.Timezone) == "" {
		return errors.New("automation.timezone is required")
	}
	if _, err := time.LoadLocation(strings.TrimSpace(c.Timezone)); err != nil {
		return fmt.Errorf("automation.timezone is invalid: %w", err)
	}
	if c.MaxConcurrentJobs <= 0 {
		return fmt.Errorf("automation.max_concurrent_jobs must be positive: %d", c.MaxConcurrentJobs)
	}
	if err := c.DefaultFireLimit.Validate("automation.default_fire_limit"); err != nil {
		return err
	}

	for i, job := range c.Jobs {
		if err := job.Validate(fmt.Sprintf("automation.jobs[%d]", i)); err != nil {
			return err
		}
	}
	for i, trigger := range c.Triggers {
		if err := trigger.Validate(fmt.Sprintf("automation.triggers[%d]", i)); err != nil {
			return err
		}
	}

	return nil
}

// Validate ensures the config-defined job is internally consistent before runtime resolution.
func (j AutomationJob) Validate(path string) error {
	if strings.TrimSpace(j.Name) == "" {
		return errors.New(path + ".name is required")
	}
	if strings.TrimSpace(j.AgentName) == "" {
		return errors.New(path + ".agent is required")
	}
	if strings.TrimSpace(j.Prompt) == "" {
		return errors.New(path + ".prompt is required")
	}
	if err := automationpkg.ValidateScopeBinding(j.Scope, j.Workspace, path, "workspace"); err != nil {
		return err
	}
	if err := j.Source.Validate(path + ".source"); err != nil {
		return err
	}
	if err := j.Schedule.Validate(path + ".schedule"); err != nil {
		return err
	}
	if err := j.Retry.Validate(path + ".retry"); err != nil {
		return err
	}
	if err := j.FireLimit.Validate(path + ".fire_limit"); err != nil {
		return err
	}

	return nil
}

// Validate ensures the config-defined trigger is internally consistent before runtime resolution.
func (t AutomationTrigger) Validate(path string) error {
	if strings.TrimSpace(t.Name) == "" {
		return errors.New(path + ".name is required")
	}
	if strings.TrimSpace(t.AgentName) == "" {
		return errors.New(path + ".agent is required")
	}
	if strings.TrimSpace(t.Prompt) == "" {
		return errors.New(path + ".prompt is required")
	}
	if strings.TrimSpace(t.Event) == "" {
		return errors.New(path + ".event is required")
	}
	if err := automationpkg.ValidateScopeBinding(t.Scope, t.Workspace, path, "workspace"); err != nil {
		return err
	}
	if err := t.Source.Validate(path + ".source"); err != nil {
		return err
	}
	if err := t.Retry.Validate(path + ".retry"); err != nil {
		return err
	}
	if err := t.FireLimit.Validate(path + ".fire_limit"); err != nil {
		return err
	}
	if err := automationpkg.ValidateTriggerFilter(t.Filter, path+".filter"); err != nil {
		return err
	}
	if err := automationpkg.ValidateTriggerPromptTemplate(t.Prompt); err != nil {
		return fmt.Errorf("%s.prompt is invalid: %w", path, err)
	}
	if strings.TrimSpace(t.Event) == "webhook" {
		if strings.TrimSpace(t.EndpointSlug) == "" {
			return errors.New(path + ".endpoint_slug is required when event is \"webhook\"")
		}
		return nil
	}
	if strings.TrimSpace(t.EndpointSlug) != "" {
		return fmt.Errorf("%s.endpoint_slug must be empty when event is %q", path, strings.TrimSpace(t.Event))
	}

	return nil
}

func (o automationOverlay) Apply(dst *AutomationConfig) error {
	if dst == nil {
		return errors.New("automation config is required")
	}

	if o.Enabled != nil {
		dst.Enabled = *o.Enabled
	}
	if o.Timezone != nil {
		dst.Timezone = *o.Timezone
	}
	if o.MaxConcurrentJobs != nil {
		dst.MaxConcurrentJobs = *o.MaxConcurrentJobs
	}
	if o.DefaultFireLimit != nil {
		dst.DefaultFireLimit = *o.DefaultFireLimit
	}

	for i, raw := range o.Jobs {
		job, err := raw.toAutomationJob(dst.DefaultFireLimit)
		if err != nil {
			return fmt.Errorf("automation.jobs[%d]: %w", i, err)
		}
		dst.Jobs = append(dst.Jobs, job)
	}
	for i, raw := range o.Triggers {
		trigger, err := raw.toAutomationTrigger(dst.DefaultFireLimit)
		if err != nil {
			return fmt.Errorf("automation.triggers[%d]: %w", i, err)
		}
		dst.Triggers = append(dst.Triggers, trigger)
	}

	return nil
}

func (j parsedAutomationJob) toAutomationJob(defaultFireLimit automationpkg.FireLimitConfig) (AutomationJob, error) {
	job := AutomationJob{
		Scope:     j.Scope,
		Name:      j.Name,
		AgentName: j.AgentName,
		Workspace: j.Workspace,
		Prompt:    j.Prompt,
		Enabled:   true,
		Retry:     automationpkg.DefaultRetryConfig(),
		FireLimit: defaultFireLimit,
		Source:    automationpkg.JobSourceConfig,
	}
	if j.Schedule != nil {
		job.Schedule = *j.Schedule
	}
	if j.Enabled != nil {
		job.Enabled = *j.Enabled
	}
	if j.Retry != nil {
		job.Retry = *j.Retry
	}
	if j.FireLimit != nil {
		job.FireLimit = *j.FireLimit
	}

	return job, nil
}

func (t parsedAutomationTrigger) toAutomationTrigger(defaultFireLimit automationpkg.FireLimitConfig) (AutomationTrigger, error) {
	trigger := AutomationTrigger{
		Scope:        t.Scope,
		Name:         t.Name,
		AgentName:    t.AgentName,
		Workspace:    t.Workspace,
		Prompt:       t.Prompt,
		Event:        t.Event,
		Filter:       mergeStringMaps(nil, t.Filter),
		Enabled:      true,
		Retry:        automationpkg.DefaultRetryConfig(),
		FireLimit:    defaultFireLimit,
		Source:       automationpkg.JobSourceConfig,
		EndpointSlug: t.EndpointSlug,
	}
	if t.Enabled != nil {
		trigger.Enabled = *t.Enabled
	}
	if t.Retry != nil {
		trigger.Retry = *t.Retry
	}
	if t.FireLimit != nil {
		trigger.FireLimit = *t.FireLimit
	}

	return trigger, nil
}
