package config

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	// DefaultCoordinatorAgentName is the bundled coordinator identity used when config is silent.
	DefaultCoordinatorAgentName = "coordinator"
	// DefaultCoordinatorTTL is the conservative coordinator session TTL used by autonomy defaults.
	DefaultCoordinatorTTL = 2 * time.Hour
	// MinCoordinatorTTL is the shortest coordinator TTL accepted by config validation.
	MinCoordinatorTTL = time.Minute
	// MaxCoordinatorTTL is the longest coordinator TTL accepted by config validation.
	MaxCoordinatorTTL = 24 * time.Hour
	// DefaultCoordinatorMaxChildren is the safe per-coordinator child-session cap.
	DefaultCoordinatorMaxChildren = 5
	// MaxCoordinatorChildren is the hard MVP cap for coordinator child sessions.
	MaxCoordinatorChildren = 5
	// DefaultCoordinatorMaxActivePerWorkspace preserves one active coordinator per workspace.
	DefaultCoordinatorMaxActivePerWorkspace = 1

	// DefaultSchedulerFanOutAfter is the wake count before the convergence ladder fans out.
	DefaultSchedulerFanOutAfter = 2
	// DefaultSchedulerSpawnAfter is the wake count before a capability-matched worker is spawned.
	DefaultSchedulerSpawnAfter = 4
	// DefaultSchedulerEventAfter is the wake count before the canonical starved event is emitted.
	DefaultSchedulerEventAfter = 6
	// DefaultSchedulerNeedsAttentionAfter is the wake count before a run is parked needs_attention.
	DefaultSchedulerNeedsAttentionAfter = 10
	// DefaultSchedulerMinQueuedAge is the queued age before a claimable run starts escalating.
	DefaultSchedulerMinQueuedAge = 2 * time.Minute
)

type providerResolver interface {
	ResolveProvider(name string) (ProviderConfig, error)
}

var _ providerResolver = (*Config)(nil)

// AutonomyConfig controls opt-in autonomy features.
type AutonomyConfig struct {
	Coordinator CoordinatorConfig `toml:"coordinator"`
	Scheduler   SchedulerConfig   `toml:"scheduler"`
}

// SchedulerConfig bounds the mechanical scheduler's convergence escalation ladder. The counts are
// monotonic wake cycles a claimable run must remain queued before each tier fires.
type SchedulerConfig struct {
	FanOutAfter         int           `toml:"fan_out_after"`
	SpawnAfter          int           `toml:"spawn_after"`
	EventAfter          int           `toml:"event_after"`
	NeedsAttentionAfter int           `toml:"needs_attention_after"`
	MinQueuedAge        time.Duration `toml:"min_queued_age"`
}

// DefaultSchedulerConfig returns the built-in convergence ladder defaults.
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		FanOutAfter:         DefaultSchedulerFanOutAfter,
		SpawnAfter:          DefaultSchedulerSpawnAfter,
		EventAfter:          DefaultSchedulerEventAfter,
		NeedsAttentionAfter: DefaultSchedulerNeedsAttentionAfter,
		MinQueuedAge:        DefaultSchedulerMinQueuedAge,
	}
}

// CoordinatorConfig defines the resolved coordinator policy.
type CoordinatorConfig struct {
	Enabled               bool          `toml:"enabled"`
	AgentName             string        `toml:"agent_name"`
	Provider              string        `toml:"provider,omitempty"`
	Model                 string        `toml:"model,omitempty"`
	DefaultTTL            time.Duration `toml:"default_ttl"`
	MaxChildren           int           `toml:"max_children"`
	MaxActivePerWorkspace int           `toml:"max_active_per_workspace"`
}

// DefaultCoordinatorConfig returns the built-in coordinator policy defaults.
func DefaultCoordinatorConfig() CoordinatorConfig {
	return CoordinatorConfig{
		Enabled:               false,
		AgentName:             DefaultCoordinatorAgentName,
		DefaultTTL:            DefaultCoordinatorTTL,
		MaxChildren:           DefaultCoordinatorMaxChildren,
		MaxActivePerWorkspace: DefaultCoordinatorMaxActivePerWorkspace,
	}
}

// DefaultCoordinatorAgentDef returns the bundled coordinator identity used when no
// workspace or global agent definition has been resolved yet.
func DefaultCoordinatorAgentDef() AgentDef {
	return AgentDef{
		Name:   DefaultCoordinatorAgentName,
		Prompt: "AGH coordinator agent identity.",
	}
}

// Validate ensures autonomy config is internally consistent.
func (c AutonomyConfig) Validate(resolver providerResolver) error {
	if err := c.Coordinator.Validate("autonomy.coordinator", resolver); err != nil {
		return err
	}
	return c.Scheduler.Validate("autonomy.scheduler")
}

// Validate ensures the convergence ladder thresholds are positive and monotonic.
func (c SchedulerConfig) Validate(path string) error {
	if c.FanOutAfter <= 0 {
		return fmt.Errorf("%s.fan_out_after must be positive: %d", path, c.FanOutAfter)
	}
	if c.SpawnAfter < c.FanOutAfter {
		return fmt.Errorf("%s.spawn_after must be >= fan_out_after: %d < %d", path, c.SpawnAfter, c.FanOutAfter)
	}
	if c.EventAfter < c.SpawnAfter {
		return fmt.Errorf("%s.event_after must be >= spawn_after: %d < %d", path, c.EventAfter, c.SpawnAfter)
	}
	if c.NeedsAttentionAfter < c.EventAfter {
		return fmt.Errorf(
			"%s.needs_attention_after must be >= event_after: %d < %d",
			path,
			c.NeedsAttentionAfter,
			c.EventAfter,
		)
	}
	if c.MinQueuedAge <= 0 {
		return fmt.Errorf("%s.min_queued_age must be positive: %s", path, c.MinQueuedAge)
	}
	return nil
}

// Validate ensures coordinator policy is safe to consume.
func (c CoordinatorConfig) Validate(path string, resolver providerResolver) error {
	if strings.TrimSpace(c.AgentName) == "" {
		return fmt.Errorf("%s.agent_name is required", path)
	}
	if c.Provider != "" && strings.TrimSpace(c.Provider) == "" {
		return fmt.Errorf("%s.provider is required when set", path)
	}
	if c.Model != "" && strings.TrimSpace(c.Model) == "" {
		return fmt.Errorf("%s.model is required when set", path)
	}
	if c.DefaultTTL < MinCoordinatorTTL || c.DefaultTTL > MaxCoordinatorTTL {
		return fmt.Errorf(
			"%s.default_ttl must be between %s and %s: %s",
			path,
			MinCoordinatorTTL,
			MaxCoordinatorTTL,
			c.DefaultTTL,
		)
	}
	if c.MaxChildren <= 0 {
		return fmt.Errorf("%s.max_children must be positive: %d", path, c.MaxChildren)
	}
	if c.MaxChildren > MaxCoordinatorChildren {
		return fmt.Errorf("%s.max_children must be <= %d: %d", path, MaxCoordinatorChildren, c.MaxChildren)
	}
	if c.MaxActivePerWorkspace != DefaultCoordinatorMaxActivePerWorkspace {
		return fmt.Errorf(
			"%s.max_active_per_workspace must be %d to preserve coordinator uniqueness: %d",
			path,
			DefaultCoordinatorMaxActivePerWorkspace,
			c.MaxActivePerWorkspace,
		)
	}

	providerName := strings.TrimSpace(c.Provider)
	if providerName == "" {
		return nil
	}
	if resolver == nil {
		return fmt.Errorf("%s.provider resolver is required", path)
	}
	provider, err := resolver.ResolveProvider(providerName)
	if err != nil {
		return fmt.Errorf("%s.provider: %w", path, err)
	}
	if strings.TrimSpace(c.Model) == "" &&
		strings.TrimSpace(provider.Models.Default) == "" &&
		provider.RequiresRuntimeModel() {
		return fmt.Errorf("%s.model is required when provider %q has no default model", path, providerName)
	}
	return nil
}

// ResolveCoordinatorConfig resolves coordinator runtime policy using the
// precedence config overlay > fallback agent definition > provider defaults.
func (c *Config) ResolveCoordinatorConfig(fallback AgentDef) (CoordinatorConfig, error) {
	if c == nil {
		return CoordinatorConfig{}, errors.New("config is required")
	}

	resolved := c.Autonomy.Coordinator
	if strings.TrimSpace(resolved.AgentName) == "" {
		resolved.AgentName = strings.TrimSpace(fallback.Name)
	}
	if strings.TrimSpace(resolved.AgentName) == "" {
		resolved.AgentName = DefaultCoordinatorAgentName
	}
	if err := resolved.Validate("autonomy.coordinator", c); err != nil {
		return CoordinatorConfig{}, err
	}

	providerName := firstTrimmedNonEmpty(resolved.Provider, fallback.Provider, c.Defaults.Provider)
	model := firstTrimmedNonEmpty(resolved.Model, fallback.Model)
	if providerName != "" {
		provider, err := c.ResolveProvider(providerName)
		if err != nil {
			return CoordinatorConfig{}, fmt.Errorf("autonomy.coordinator.provider: %w", err)
		}
		if model == "" {
			model = strings.TrimSpace(provider.Models.Default)
		}
		if model == "" && provider.RequiresRuntimeModel() {
			return CoordinatorConfig{}, fmt.Errorf(
				"autonomy.coordinator.model is required when provider %q has no default model",
				providerName,
			)
		}
	} else if model != "" {
		return CoordinatorConfig{}, errors.New("autonomy.coordinator.provider is required when model is set")
	}

	resolved.AgentName = strings.TrimSpace(resolved.AgentName)
	resolved.Provider = providerName
	resolved.Model = model
	return resolved, nil
}

func firstTrimmedNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
