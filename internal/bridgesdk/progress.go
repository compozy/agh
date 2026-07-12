package bridgesdk

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	bridgepkg "github.com/compozy/agh/internal/bridges"
)

const (
	defaultProgressEditInterval  = 1500 * time.Millisecond
	defaultProgressMaxMessageLen = 4_000
	progressMessageHeadroom      = 64
)

// ProgressSink applies provider-specific progress messages and affordances.
type ProgressSink interface {
	Post(ctx context.Context, target bridgepkg.DeliveryTarget, text string) (remoteID string, err error)
	Edit(ctx context.Context, target bridgepkg.DeliveryTarget, remoteID string, text string) error
	Typing(ctx context.Context, target bridgepkg.DeliveryTarget, active bool) error
	React(ctx context.Context, target bridgepkg.DeliveryTarget, reaction string) error
}

// ProgressConfig binds settings to one target; empty mode/grouping default to all/accumulate.
type ProgressConfig struct {
	bridgepkg.ProgressConfig
	Target        bridgepkg.DeliveryTarget
	MaxMessageLen int              // Values through 64 use the conservative 4,000-unit default.
	Len           func(string) int // Nil uses byte length.
}

// ProgressAccumulator owns one route's mutable progress bubble state.
type ProgressAccumulator struct {
	mu sync.Mutex

	config    ProgressConfig
	configErr error
	sink      ProgressSink
	now       func() time.Time

	currentBubbleID string
	currentText     string
	lastLine        string
	repeatCount     int
	lastEditAt      time.Time
	canEdit         bool
	dirty           bool
	typingActive    bool
}

// NewProgressAccumulator constructs state; a zero edit interval defaults to 1.5 seconds.
func NewProgressAccumulator(
	cfg ProgressConfig,
	sink ProgressSink,
	now func() time.Time,
) *ProgressAccumulator {
	normalized, err := normalizeProgressAccumulatorConfig(cfg)
	if sink == nil {
		sink = noOpProgressSink{}
	}
	if now == nil {
		now = func() time.Time {
			return time.Now().UTC()
		}
	}
	return &ProgressAccumulator{
		config:    normalized,
		configErr: err,
		sink:      sink,
		now:       now,
		canEdit:   normalized.Grouping == bridgepkg.ProgressGroupingAccumulate,
	}
}

// OnProgress records one tool lifecycle line and applies due provider updates.
func (a *ProgressAccumulator) OnProgress(ctx context.Context, event bridgepkg.ToolProgress) error {
	if a == nil {
		return errors.New("bridgesdk: progress accumulator is required")
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.configErr != nil {
		return a.configErr
	}
	if a.config.ToolProgress == bridgepkg.ProgressModeOff {
		return nil
	}
	if err := validateProgressContext(ctx); err != nil {
		return err
	}
	if err := event.Validate(); err != nil {
		return fmt.Errorf("bridgesdk: validate progress event: %w", err)
	}
	affordanceErr := a.applyPhaseAffordancesLocked(ctx, event.Phase)

	line := renderProgressLine(event)
	if a.config.Grouping == bridgepkg.ProgressGroupingSeparate {
		return errors.Join(a.postSeparateLineLocked(ctx, line), affordanceErr)
	}

	if line == a.lastLine && a.repeatCount > 0 {
		return errors.Join(a.applyRepeatLocked(ctx), affordanceErr)
	}

	if err := a.appendLineLocked(ctx, line); err != nil {
		return errors.Join(err, affordanceErr)
	}
	a.lastLine = line
	a.repeatCount = 1
	return affordanceErr
}

// OnContent flushes pending progress, stops typing, and closes the current editable bubble.
func (a *ProgressAccumulator) OnContent(ctx context.Context) error {
	if a == nil {
		return errors.New("bridgesdk: progress accumulator is required")
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.configErr != nil {
		return a.configErr
	}
	if a.config.ToolProgress == bridgepkg.ProgressModeOff {
		a.resetBubbleLocked()
		return nil
	}
	if err := validateProgressContext(ctx); err != nil {
		return err
	}
	flushErr := a.flushLocked(ctx, true)
	typingErr := a.closeTypingLocked(ctx)
	if err := errors.Join(flushErr, typingErr); err != nil {
		return err
	}
	a.resetBubbleLocked()
	return nil
}

// Flush writes pending accumulated progress immediately.
func (a *ProgressAccumulator) Flush(ctx context.Context) error {
	if a == nil {
		return errors.New("bridgesdk: progress accumulator is required")
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.configErr != nil {
		return a.configErr
	}
	if a.config.ToolProgress == bridgepkg.ProgressModeOff {
		return nil
	}
	if err := validateProgressContext(ctx); err != nil {
		return err
	}
	return a.flushLocked(ctx, true)
}

func normalizeProgressAccumulatorConfig(cfg ProgressConfig) (ProgressConfig, error) {
	cfg.ToolProgress = cfg.ToolProgress.Normalize()
	if cfg.ToolProgress == "" {
		cfg.ToolProgress = bridgepkg.ProgressModeAll
	}
	cfg.Grouping = cfg.Grouping.Normalize()
	if cfg.Grouping == "" {
		cfg.Grouping = bridgepkg.ProgressGroupingAccumulate
	}
	if cfg.EditInterval == 0 {
		cfg.EditInterval = defaultProgressEditInterval
	}
	if cfg.MaxMessageLen <= progressMessageHeadroom {
		cfg.MaxMessageLen = defaultProgressMaxMessageLen
	}
	if cfg.Len == nil {
		cfg.Len = func(value string) int {
			return len(value)
		}
	}
	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("bridgesdk: invalid progress config: %w", err)
	}
	if !cfg.Target.IsZero() {
		if err := cfg.Target.Validate(); err != nil {
			return cfg, fmt.Errorf("bridgesdk: invalid progress target: %w", err)
		}
	}
	return cfg, nil
}

func validateProgressContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("bridgesdk: progress context is required")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("bridgesdk: progress context: %w", err)
	}
	return nil
}

func (a *ProgressAccumulator) applyPhaseAffordancesLocked(
	ctx context.Context,
	phase bridgepkg.ToolProgressPhase,
) error {
	active := phase == bridgepkg.ToolProgressPhaseStarted
	var typingErr error
	if a.config.Typing {
		if err := a.sink.Typing(ctx, a.config.Target, active); err != nil {
			typingErr = fmt.Errorf("bridgesdk: set progress typing state: %w", err)
		} else {
			a.typingActive = active
		}
	}
	if !a.config.Reactions {
		return typingErr
	}
	reaction := "👀"
	switch phase {
	case bridgepkg.ToolProgressPhaseCompleted:
		reaction = "✅"
	case bridgepkg.ToolProgressPhaseFailed:
		reaction = "❌"
	}
	var reactionErr error
	if err := a.sink.React(ctx, a.config.Target, reaction); err != nil {
		reactionErr = fmt.Errorf("bridgesdk: apply progress reaction: %w", err)
	}
	return errors.Join(typingErr, reactionErr)
}

func (a *ProgressAccumulator) closeTypingLocked(ctx context.Context) error {
	if !a.config.Typing || !a.typingActive {
		return nil
	}
	if err := a.sink.Typing(ctx, a.config.Target, false); err != nil {
		return fmt.Errorf("bridgesdk: set progress typing state: %w", err)
	}
	a.typingActive = false
	return nil
}

func (a *ProgressAccumulator) postSeparateLineLocked(ctx context.Context, line string) error {
	chunks, err := splitProgressText(line, a.messageLimitLocked(), a.config.Len, "")
	if err != nil {
		return err
	}
	for _, chunk := range chunks {
		if _, err := a.sink.Post(ctx, a.config.Target, chunk); err != nil {
			return fmt.Errorf("bridgesdk: post separate progress line: %w", err)
		}
	}
	return nil
}

func (a *ProgressAccumulator) applyRepeatLocked(ctx context.Context) error {
	oldSuffix := progressRepeatSuffix(a.repeatCount)
	if !strings.HasSuffix(a.currentText, oldSuffix) {
		return errors.New("bridgesdk: progress repeat state is inconsistent")
	}
	nextCount := a.repeatCount + 1
	nextSuffix := progressRepeatSuffix(nextCount)
	nextText := strings.TrimSuffix(a.currentText, oldSuffix) + nextSuffix
	if err := a.replaceCurrentTextLocked(ctx, nextText, nextSuffix); err != nil {
		return err
	}
	a.repeatCount = nextCount
	return nil
}

func (a *ProgressAccumulator) appendLineLocked(ctx context.Context, line string) error {
	if a.lastLine == "" {
		return a.replaceCurrentTextLocked(ctx, line, "")
	}

	candidate := a.currentText + "\n" + line
	if a.config.Len(candidate) <= a.messageLimitLocked() {
		return a.replaceCurrentTextLocked(ctx, candidate, "")
	}
	if err := a.flushLocked(ctx, true); err != nil {
		return err
	}
	a.freezeCurrentBubbleLocked()
	return a.replaceCurrentTextLocked(ctx, line, "")
}

func (a *ProgressAccumulator) replaceCurrentTextLocked(
	ctx context.Context,
	text string,
	protectedTail string,
) error {
	chunks, err := splitProgressText(text, a.messageLimitLocked(), a.config.Len, protectedTail)
	if err != nil {
		return err
	}
	for index, chunk := range chunks {
		a.currentText = chunk
		a.dirty = true
		force := len(chunks) > 1
		if err := a.flushLocked(ctx, force); err != nil {
			return err
		}
		if index < len(chunks)-1 {
			a.freezeCurrentBubbleLocked()
		}
	}
	return nil
}

func (a *ProgressAccumulator) flushLocked(ctx context.Context, force bool) error {
	if !a.dirty || a.currentText == "" {
		return nil
	}
	if got, limit := a.config.Len(a.currentText), a.messageLimitLocked(); got > limit {
		return fmt.Errorf("bridgesdk: progress message length %d exceeds provider limit %d", got, limit)
	}
	if a.currentBubbleID == "" || !a.canEdit {
		remoteID, err := a.sink.Post(ctx, a.config.Target, a.currentText)
		if err != nil {
			return fmt.Errorf("bridgesdk: post progress bubble: %w", err)
		}
		a.currentBubbleID = strings.TrimSpace(remoteID)
		if a.currentBubbleID == "" {
			return errors.New("bridgesdk: post progress bubble returned an empty remote id")
		}
		a.canEdit = true
		a.lastEditAt = a.now()
		a.dirty = false
		return nil
	}

	now := a.now()
	if !force && now.Sub(a.lastEditAt) < a.config.EditInterval {
		return nil
	}
	if err := a.sink.Edit(ctx, a.config.Target, a.currentBubbleID, a.currentText); err != nil {
		return fmt.Errorf("bridgesdk: edit progress bubble: %w", err)
	}
	a.lastEditAt = now
	a.dirty = false
	return nil
}

func (a *ProgressAccumulator) messageLimitLocked() int {
	return a.config.MaxMessageLen - progressMessageHeadroom
}

func (a *ProgressAccumulator) freezeCurrentBubbleLocked() {
	a.currentBubbleID = ""
	a.currentText = ""
	a.lastEditAt = time.Time{}
	a.canEdit = a.config.Grouping == bridgepkg.ProgressGroupingAccumulate
	a.dirty = false
}

func (a *ProgressAccumulator) resetBubbleLocked() {
	a.freezeCurrentBubbleLocked()
	a.lastLine = ""
	a.repeatCount = 0
}

func renderProgressLine(event bridgepkg.ToolProgress) string {
	label := strings.TrimSpace(event.Label)
	if label == "" {
		label = strings.TrimSpace(event.ToolID)
	}
	line := label
	if preview := strings.TrimSpace(event.Preview); preview != "" {
		line += " " + preview
	}
	emoji := strings.TrimSpace(event.Emoji)
	switch event.Phase {
	case bridgepkg.ToolProgressPhaseCompleted:
		emoji = "✅"
		if duration := time.Duration(event.DurationMS) * time.Millisecond; duration >= 30*time.Second {
			line += " · " + duration.Round(100*time.Millisecond).String()
		}
	case bridgepkg.ToolProgressPhaseFailed:
		emoji = "❌"
		if detail := strings.TrimSpace(event.Error); detail != "" {
			line += " — " + detail
		}
	}
	if emoji != "" {
		line = emoji + " " + line
	}
	return strings.TrimSpace(line)
}

func progressRepeatSuffix(count int) string {
	if count <= 1 {
		return ""
	}
	return fmt.Sprintf(" (×%d)", count)
}

type noOpProgressSink struct{}

var _ ProgressSink = noOpProgressSink{}

func (noOpProgressSink) Post(
	context.Context,
	bridgepkg.DeliveryTarget,
	string,
) (string, error) {
	return "noop", nil
}

func (noOpProgressSink) Edit(
	context.Context,
	bridgepkg.DeliveryTarget,
	string,
	string,
) error {
	return nil
}

func (noOpProgressSink) Typing(context.Context, bridgepkg.DeliveryTarget, bool) error {
	return nil
}

func (noOpProgressSink) React(context.Context, bridgepkg.DeliveryTarget, string) error {
	return nil
}
