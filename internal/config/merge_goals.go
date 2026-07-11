package config

type goalsOverlay struct {
	MaxTurns          *int     `toml:"max_turns"`
	ContextNudgeRatio *float64 `toml:"context_nudge_ratio"`
}

func (o goalsOverlay) Apply(dst *GoalsConfig) {
	if o.MaxTurns != nil {
		dst.MaxTurns = *o.MaxTurns
	}
	if o.ContextNudgeRatio != nil {
		dst.ContextNudgeRatio = *o.ContextNudgeRatio
	}
}
