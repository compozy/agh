package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/session"
)

const (
	durableMemoryAugmenterOrder = 100
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
	descriptors []promptInputAugmenterDescriptor
	byName      map[HarnessAugmenter]promptInputAugmenterDescriptor
}

func defaultPromptInputAugmenterDescriptors(
	durableMemory session.PromptInputAugmenter,
) []promptInputAugmenterDescriptor {
	descriptors := make([]promptInputAugmenterDescriptor, 0, 1)
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
	return descriptors
}

func newPromptInputCompositeAugmenter(
	logger *slog.Logger,
	resolver promptInputAugmenterResolver,
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

	resolved, err := c.resolver.ResolvePrompt(info, sess.CurrentTurnSource(), acp.PromptMeta{})
	if err != nil {
		return "", fmt.Errorf("daemon: resolve prompt augmentation policy: %w", err)
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
		next, augmentErr := descriptor.Augmenter(ctx, sess, current)
		if augmentErr != nil {
			wrappedErr := fmt.Errorf("daemon: prompt augmenter %q: %w", descriptor.Name, augmentErr)
			if descriptor.Critical ||
				errors.Is(augmentErr, context.Canceled) ||
				errors.Is(augmentErr, context.DeadlineExceeded) {
				return "", wrappedErr
			}
			c.loggerForSession(sess).Warn(
				"daemon: noncritical prompt augmenter failed",
				"augmenter",
				descriptor.Name,
				"error",
				augmentErr,
			)
			continue
		}
		if strings.TrimSpace(next) == "" {
			continue
		}

		bounded, consumed := applyPromptInputAugmenterBudget(
			current,
			next,
			limited,
			remainingBudget,
			descriptor.BudgetBehavior,
		)
		if strings.TrimSpace(bounded) == "" {
			continue
		}
		current = bounded
		if limited {
			remainingBudget = max(remainingBudget-consumed, 0)
		}
	}

	return current, nil
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
