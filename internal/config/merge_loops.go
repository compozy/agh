package config

type loopsOverlay struct {
	Defaults loopsDefaultsOverlay `toml:"defaults"`
}

type loopsDefaultsOverlay struct {
	Delivery loopDefaultOverlay `toml:"delivery"`
	Watch    loopDefaultOverlay `toml:"watch"`
}

type loopDefaultOverlay struct {
	IterationCap *int                         `toml:"iteration_cap"`
	NoProgress   loopNoProgressDefaultOverlay `toml:"no_progress"`
	Gates        loopGatesDefaultOverlay      `toml:"gates"`
	Budget       loopBudgetDefaultOverlay     `toml:"budget"`
	FanOutWidth  *int                         `toml:"fan_out_width"`
}

type loopNoProgressDefaultOverlay struct {
	Window *int `toml:"window"`
}

type loopGatesDefaultOverlay struct {
	MaxRevisions *int `toml:"max_revisions"`
}

type loopBudgetDefaultOverlay struct {
	Tokens       *int    `toml:"tokens"`
	WallClockSec *int    `toml:"wall_clock_sec"`
	OnExceeded   *string `toml:"on_exceeded"`
}

func (o loopsOverlay) Apply(dst *LoopsConfig) {
	o.Defaults.Apply(&dst.Defaults)
}

func (o loopsDefaultsOverlay) Apply(dst *LoopsDefaultsConfig) {
	o.Delivery.Apply(&dst.Delivery)
	o.Watch.Apply(&dst.Watch)
}

func (o loopDefaultOverlay) Apply(dst *LoopDefaultConfig) {
	if o.IterationCap != nil {
		dst.IterationCap = *o.IterationCap
	}
	o.NoProgress.Apply(&dst.NoProgress)
	o.Gates.Apply(&dst.Gates)
	o.Budget.Apply(&dst.Budget)
	if o.FanOutWidth != nil {
		dst.FanOutWidth = *o.FanOutWidth
	}
}

func (o loopNoProgressDefaultOverlay) Apply(dst *LoopNoProgressDefaultConfig) {
	if o.Window != nil {
		dst.Window = *o.Window
	}
}

func (o loopGatesDefaultOverlay) Apply(dst *LoopGatesDefaultConfig) {
	if o.MaxRevisions != nil {
		dst.MaxRevisions = *o.MaxRevisions
	}
}

func (o loopBudgetDefaultOverlay) Apply(dst *LoopBudgetDefaultConfig) {
	if o.Tokens != nil {
		dst.Tokens = *o.Tokens
	}
	if o.WallClockSec != nil {
		dst.WallClockSec = *o.WallClockSec
	}
	if o.OnExceeded != nil {
		dst.OnExceeded = *o.OnExceeded
	}
}
