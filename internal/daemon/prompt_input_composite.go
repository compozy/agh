package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/session"
)

const (
	durableMemoryAugmenterOrder = 100
	skillsAugmenterOrder        = 150
	situationAugmenterOrder     = 200
	situationAugmenterBudget    = 20_000
)

type promptInputAugmenterBudgetBehavior string

const (
	promptInputAugmenterBudgetBehaviorTrim promptInputAugmenterBudgetBehavior = "trim"
	promptInputAugmenterBudgetBehaviorOmit promptInputAugmenterBudgetBehavior = "omit"
)

type promptInputAugmenterResolver interface {
	ResolvePrompt(
		info *session.Info,
		source session.TurnSource,
		meta acp.PromptMeta,
	) (ResolvedHarnessContext, error)
}

type promptInputAugmenterDescriptor struct {
	Name           HarnessAugmenter
	Order          int
	Budget         int
	BudgetBehavior promptInputAugmenterBudgetBehavior
	Critical       bool
	Augmenter      session.PromptInputAugmenter
}

type promptInputComposite struct {
	logger      *slog.Logger
	resolver    promptInputAugmenterResolver
	recorder    *harnessLifecycleRecorder
	descriptors []promptInputAugmenterDescriptor
	byName      map[HarnessAugmenter]promptInputAugmenterDescriptor
}

func defaultPromptInputAugmenterDescriptors(
	durableMemory session.PromptInputAugmenter,
	skillsCatalog session.PromptInputAugmenter,
	situationAugmenters ...session.PromptInputAugmenter,
) []promptInputAugmenterDescriptor {
	descriptors := make([]promptInputAugmenterDescriptor, 0, 3)
	if durableMemory != nil {
		descriptors = append(descriptors, promptInputAugmenterDescriptor{
			Name:           HarnessAugmenterDurableMemory,
			Order:          durableMemoryAugmenterOrder,
			Budget:         memory.RecallAugmenterBudget,
			BudgetBehavior: promptInputAugmenterBudgetBehaviorTrim,
			Critical:       false,
			Augmenter:      durableMemory,
		})
	}
	if skillsCatalog != nil {
		descriptors = append(descriptors, promptInputAugmenterDescriptor{
			Name:           HarnessAugmenterSkills,
			Order:          skillsAugmenterOrder,
			Budget:         startupSkillsSectionBudget,
			BudgetBehavior: promptInputAugmenterBudgetBehaviorTrim,
			Critical:       false,
			Augmenter:      skillsCatalog,
		})
	}
	if len(situationAugmenters) > 0 && situationAugmenters[0] != nil {
		descriptors = append(descriptors, promptInputAugmenterDescriptor{
			Name:           HarnessAugmenterSituation,
			Order:          situationAugmenterOrder,
			Budget:         situationAugmenterBudget,
			BudgetBehavior: promptInputAugmenterBudgetBehaviorOmit,
			Critical:       false,
			Augmenter:      situationAugmenters[0],
		})
	}
	return descriptors
}

func newPromptInputCompositeAugmenter(
	logger *slog.Logger,
	resolver promptInputAugmenterResolver,
	recorder *harnessLifecycleRecorder,
	descriptors ...promptInputAugmenterDescriptor,
) (session.PromptInputAugmenter, error) {
	if resolver == nil {
		return nil, nil
	}

	normalized, err := normalizePromptInputAugmenterDescriptors(descriptors)
	if err != nil {
		return nil, err
	}

	composite := &promptInputComposite{
		logger:      logger,
		resolver:    resolver,
		recorder:    recorder,
		descriptors: normalized,
		byName:      make(map[HarnessAugmenter]promptInputAugmenterDescriptor, len(normalized)),
	}
	for _, descriptor := range normalized {
		composite.byName[descriptor.Name] = descriptor
	}
	return composite.Augment, nil
}

func normalizePromptInputAugmenterDescriptors(
	descriptors []promptInputAugmenterDescriptor,
) ([]promptInputAugmenterDescriptor, error) {
	if len(descriptors) == 0 {
		return nil, nil
	}

	normalized := make([]promptInputAugmenterDescriptor, 0, len(descriptors))
	seen := make(map[HarnessAugmenter]struct{}, len(descriptors))
	for _, descriptor := range descriptors {
		name := HarnessAugmenter(strings.TrimSpace(string(descriptor.Name)))
		if name == "" {
			return nil, errorsPromptInputDescriptor("name is required")
		}
		if _, exists := seen[name]; exists {
			return nil, fmt.Errorf("daemon: duplicate prompt input augmenter descriptor %q", name)
		}
		if descriptor.Augmenter == nil {
			return nil, fmt.Errorf("daemon: prompt input augmenter %q is missing an augmenter", name)
		}

		normalized = append(normalized, promptInputAugmenterDescriptor{
			Name:           name,
			Order:          descriptor.Order,
			Budget:         max(descriptor.Budget, 0),
			BudgetBehavior: normalizePromptInputAugmenterBudgetBehavior(descriptor.BudgetBehavior),
			Critical:       descriptor.Critical,
			Augmenter:      descriptor.Augmenter,
		})
		seen[name] = struct{}{}
	}

	slices.SortStableFunc(normalized, func(left, right promptInputAugmenterDescriptor) int {
		if left.Order != right.Order {
			return left.Order - right.Order
		}
		return strings.Compare(string(left.Name), string(right.Name))
	})
	return normalized, nil
}

func errorsPromptInputDescriptor(detail string) error {
	return fmt.Errorf("daemon: prompt input augmenter descriptor %s", detail)
}

func normalizePromptInputAugmenterBudgetBehavior(
	behavior promptInputAugmenterBudgetBehavior,
) promptInputAugmenterBudgetBehavior {
	switch promptInputAugmenterBudgetBehavior(strings.TrimSpace(string(behavior))) {
	case promptInputAugmenterBudgetBehaviorOmit:
		return promptInputAugmenterBudgetBehaviorOmit
	case "", promptInputAugmenterBudgetBehaviorTrim:
		return promptInputAugmenterBudgetBehaviorTrim
	default:
		return promptInputAugmenterBudgetBehaviorTrim
	}
}

func (c *promptInputComposite) Augment(
	ctx context.Context,
	sess *session.Session,
	message string,
) (string, error) {
	if c == nil || c.resolver == nil || sess == nil {
		return message, nil
	}

	info := sess.Info()
	if info == nil {
		return message, nil
	}

	source := sess.CurrentTurnSource()
	if source == session.TurnSourceSynthetic {
		return message, nil
	}

	resolved, err := c.resolver.ResolvePrompt(info, source, sess.CurrentPromptMeta())
	if err != nil {
		return "", fmt.Errorf("daemon: resolve prompt augmentation policy: %w", err)
	}
	timestamp := time.Now().UTC()
	if c.recorder != nil {
		timestamp = c.recorder.timestamp(time.Time{})
		c.recorder.RecordPromptContextResolved(ctx, info, resolved, timestamp)
	}

	descriptors, err := c.selectedDescriptors(resolved.Policy.EnableAugmenters)
	if err != nil {
		return "", err
	}
	if len(descriptors) == 0 {
		return message, nil
	}

	limited := aggregatePromptInputBudget(descriptors) > 0
	remainingBudget := aggregatePromptInputBudget(descriptors)
	current := message

	for _, descriptor := range descriptors {
		var stepErr error
		current, remainingBudget, stepErr = c.applyAugmenterDescriptor(
			ctx,
			sess,
			info,
			resolved,
			descriptor,
			current,
			remainingBudget,
			limited,
			timestamp,
		)
		if stepErr != nil {
			return "", stepErr
		}
	}

	return current, nil
}

func (c *promptInputComposite) applyAugmenterDescriptor(
	ctx context.Context,
	sess *session.Session,
	info *session.Info,
	resolved ResolvedHarnessContext,
	descriptor promptInputAugmenterDescriptor,
	current string,
	remainingBudget int,
	limited bool,
	timestamp time.Time,
) (string, int, error) {
	next, augmentErr := descriptor.Augmenter(ctx, sess, current)
	if augmentErr != nil {
		return c.handleAugmenterFailure(
			ctx,
			sess,
			info,
			resolved,
			descriptor,
			current,
			remainingBudget,
			timestamp,
			augmentErr,
		)
	}

	nextCurrent, nextBudget := c.applyAugmentedMessage(
		ctx,
		info,
		resolved,
		descriptor,
		current,
		next,
		remainingBudget,
		limited,
		timestamp,
	)
	return nextCurrent, nextBudget, nil
}

func (c *promptInputComposite) handleAugmenterFailure(
	ctx context.Context,
	sess *session.Session,
	info *session.Info,
	resolved ResolvedHarnessContext,
	descriptor promptInputAugmenterDescriptor,
	current string,
	remainingBudget int,
	timestamp time.Time,
	augmentErr error,
) (string, int, error) {
	wrappedErr := fmt.Errorf("daemon: prompt augmenter %q: %w", descriptor.Name, augmentErr)
	if c.recorder != nil {
		c.recorder.RecordAugmenterFailed(ctx, info, resolved, descriptor, augmentErr, timestamp)
	}
	if descriptor.Critical ||
		errors.Is(augmentErr, context.Canceled) ||
		errors.Is(augmentErr, context.DeadlineExceeded) {
		return "", remainingBudget, wrappedErr
	}
	c.loggerForSession(sess).Warn(
		"daemon: noncritical prompt augmenter failed",
		"augmenter",
		descriptor.Name,
		"error",
		augmentErr,
	)
	return current, remainingBudget, nil
}

func (c *promptInputComposite) applyAugmentedMessage(
	ctx context.Context,
	info *session.Info,
	resolved ResolvedHarnessContext,
	descriptor promptInputAugmenterDescriptor,
	current string,
	next string,
	remainingBudget int,
	limited bool,
	timestamp time.Time,
) (string, int) {
	if strings.TrimSpace(next) == "" {
		c.recordAugmenterApplied(
			ctx,
			info,
			resolved,
			descriptor,
			"blank",
			0,
			remainingBudget,
			timestamp,
		)
		return current, remainingBudget
	}

	descriptorBudget := remainingBudget
	if limited && descriptor.Budget > 0 {
		descriptorBudget = min(remainingBudget, descriptor.Budget)
	}
	bounded, consumed := applyPromptInputAugmenterBudget(
		current,
		next,
		limited,
		descriptorBudget,
		descriptor.BudgetBehavior,
	)
	if strings.TrimSpace(bounded) == "" {
		nextBudget := max(remainingBudget-consumed, 0)
		c.recordAugmenterApplied(
			ctx,
			info,
			resolved,
			descriptor,
			"omitted",
			consumed,
			nextBudget,
			timestamp,
		)
		return current, remainingBudget
	}

	outcome := "applied"
	if bounded == current {
		outcome = "unchanged"
	} else if bounded != next {
		outcome = "trimmed"
	}

	nextBudget := remainingBudget
	if limited {
		nextBudget = max(remainingBudget-consumed, 0)
	}
	c.recordAugmenterApplied(
		ctx,
		info,
		resolved,
		descriptor,
		outcome,
		consumed,
		nextBudget,
		timestamp,
	)
	return bounded, nextBudget
}

func (c *promptInputComposite) recordAugmenterApplied(
	ctx context.Context,
	info *session.Info,
	resolved ResolvedHarnessContext,
	descriptor promptInputAugmenterDescriptor,
	outcome string,
	consumed int,
	remaining int,
	timestamp time.Time,
) {
	if c.recorder == nil {
		return
	}
	c.recorder.RecordAugmenterApplied(ctx, info, resolved, harnessAugmenterObservation{
		Name:           descriptor.Name,
		Outcome:        outcome,
		Critical:       descriptor.Critical,
		Budget:         descriptor.Budget,
		BudgetBehavior: descriptor.BudgetBehavior,
		Consumed:       consumed,
		Remaining:      remaining,
	}, timestamp)
}

func (c *promptInputComposite) selectedDescriptors(
	enabled []HarnessAugmenter,
) ([]promptInputAugmenterDescriptor, error) {
	if len(enabled) == 0 {
		return nil, nil
	}

	enabledSet := make(map[HarnessAugmenter]struct{}, len(enabled))
	for _, name := range enabled {
		enabledSet[name] = struct{}{}
		if _, ok := c.byName[name]; !ok {
			return nil, fmt.Errorf("daemon: enabled prompt augmenter %q is not registered", name)
		}
	}

	selected := make([]promptInputAugmenterDescriptor, 0, len(enabled))
	for _, descriptor := range c.descriptors {
		if _, ok := enabledSet[descriptor.Name]; ok {
			selected = append(selected, descriptor)
		}
	}
	return selected, nil
}

func (c *promptInputComposite) loggerForSession(sess *session.Session) *slog.Logger {
	logger := c.logger
	if logger == nil {
		logger = slog.Default()
	}
	if sess == nil {
		return logger
	}

	info := sess.Info()
	if info == nil {
		return logger
	}
	return logger.With("session_id", info.ID, "agent_name", info.AgentName)
}

func aggregatePromptInputBudget(descriptors []promptInputAugmenterDescriptor) int {
	total := 0
	for _, descriptor := range descriptors {
		if descriptor.Budget <= 0 {
			continue
		}
		total += descriptor.Budget
	}
	return total
}

func applyPromptInputAugmenterBudget(
	current string,
	next string,
	limited bool,
	remainingBudget int,
	behavior promptInputAugmenterBudgetBehavior,
) (string, int) {
	if !limited {
		return next, promptInputContributionRunes(current, next)
	}
	if remainingBudget <= 0 {
		return current, 0
	}

	before, after, wrapped := splitPromptInputAugmentation(current, next)
	if wrapped {
		contribution := utf8.RuneCountInString(before) + utf8.RuneCountInString(after)
		if contribution <= remainingBudget {
			return next, contribution
		}
		if normalizePromptInputAugmenterBudgetBehavior(behavior) == promptInputAugmenterBudgetBehaviorOmit {
			return current, 0
		}

		beforeBudget := min(utf8.RuneCountInString(before), remainingBudget)
		trimmedBefore := trimStringToRunes(before, beforeBudget)
		trimmedAfter := trimStringToRunes(after, remainingBudget-beforeBudget)
		return strings.TrimSpace(trimmedBefore + current + trimmedAfter), remainingBudget
	}

	contribution := promptInputContributionRunes(current, next)
	if contribution <= remainingBudget {
		return next, contribution
	}
	if normalizePromptInputAugmenterBudgetBehavior(behavior) == promptInputAugmenterBudgetBehaviorOmit {
		return current, 0
	}
	return current, 0
}

func splitPromptInputAugmentation(current string, next string) (string, string, bool) {
	if current == "" {
		return "", "", false
	}

	before, after, ok := strings.Cut(next, current)
	if !ok {
		return "", "", false
	}
	return before, after, true
}

func promptInputContributionRunes(current string, next string) int {
	before, after, wrapped := splitPromptInputAugmentation(current, next)
	if wrapped {
		return utf8.RuneCountInString(before) + utf8.RuneCountInString(after)
	}
	return max(utf8.RuneCountInString(next)-utf8.RuneCountInString(current), 0)
}
