package config

import (
	"fmt"
	"math"
)

const (
	defaultGoalMaxTurns          = 20
	defaultGoalContextNudgeRatio = 0.8
	goalMaxTurnsPath             = "goals.max_turns"
	goalContextNudgeRatioPath    = "goals.context_nudge_ratio"
)

// GoalsConfig controls defaults applied to newly started Goal runs.
type GoalsConfig struct {
	MaxTurns          int     `toml:"max_turns"`
	ContextNudgeRatio float64 `toml:"context_nudge_ratio"`
}

// DefaultGoalsConfig returns the built-in Goal defaults.
func DefaultGoalsConfig() GoalsConfig {
	return GoalsConfig{
		MaxTurns:          defaultGoalMaxTurns,
		ContextNudgeRatio: defaultGoalContextNudgeRatio,
	}
}

// Validate enforces Goal default bounds without changing explicit values.
func (c GoalsConfig) Validate() error {
	if c.MaxTurns < 1 {
		return ValidationError{
			Path:    goalMaxTurnsPath,
			Message: fmt.Sprintf("must be >= 1: %d", c.MaxTurns),
		}
	}
	if math.IsNaN(c.ContextNudgeRatio) || math.IsInf(c.ContextNudgeRatio, 0) ||
		c.ContextNudgeRatio < 0 || c.ContextNudgeRatio > 1 {
		return ValidationError{
			Path:    goalContextNudgeRatioPath,
			Message: fmt.Sprintf("must be between 0 and 1: %v", c.ContextNudgeRatio),
		}
	}
	return nil
}
