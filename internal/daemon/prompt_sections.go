package daemon

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/skills/bundled"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const (
	bundledNetworkSkillName = "agh-network"

	startupMemorySectionOrder  = 100
	startupSkillsSectionOrder  = 100
	startupNetworkSectionOrder = 200

	startupMemorySectionBudget  = 24_000
	startupSkillsSectionBudget  = 16_000
	startupNetworkSectionBudget = 12_000
)

// PromptSectionPosition identifies whether a startup section renders before or
// after the base agent prompt.
type PromptSectionPosition string

const (
	// PromptSectionPositionPrepend renders the section before the base prompt.
	PromptSectionPositionPrepend PromptSectionPosition = "prepend"
	// PromptSectionPositionAppend renders the section after the base prompt.
	PromptSectionPositionAppend PromptSectionPosition = "append"
)

// PromptSectionBudgetBehavior controls how an oversized section is handled.
type PromptSectionBudgetBehavior string

const (
	// PromptSectionBudgetBehaviorTrim truncates the section to its declared budget.
	PromptSectionBudgetBehaviorTrim PromptSectionBudgetBehavior = "trim"
	// PromptSectionBudgetBehaviorOmit drops the section when it exceeds budget.
	PromptSectionBudgetBehaviorOmit PromptSectionBudgetBehavior = "omit"
)

// SectionPredicate decides whether a prompt section is eligible for one
// resolved startup policy.
type SectionPredicate func(ResolvedHarnessPolicy) bool

// PromptSectionDescriptor describes one startup prompt section provider plus
// its ordering, policy eligibility, and budget behavior.
type PromptSectionDescriptor struct {
	Name           string
	Position       PromptSectionPosition
	Order          int
	Budget         int
	BudgetBehavior PromptSectionBudgetBehavior
	Provider       session.PromptProvider
	Predicate      SectionPredicate
}

func defaultStartupPromptSectionDescriptors(
	memoryProvider session.PromptProvider,
	skillsProvider session.PromptProvider,
) []PromptSectionDescriptor {
	descriptors := make([]PromptSectionDescriptor, 0, 3)

	if memoryProvider != nil {
		descriptors = append(descriptors, PromptSectionDescriptor{
			Name:           string(HarnessPromptSectionMemory),
			Position:       PromptSectionPositionPrepend,
			Order:          startupMemorySectionOrder,
			Budget:         startupMemorySectionBudget,
			BudgetBehavior: PromptSectionBudgetBehaviorTrim,
			Provider:       memoryProvider,
			Predicate:      policyIncludesSection(HarnessPromptSectionMemory),
		})
	}

	if skillsProvider != nil {
		descriptors = append(descriptors, PromptSectionDescriptor{
			Name:           string(HarnessPromptSectionSkills),
			Position:       PromptSectionPositionAppend,
			Order:          startupSkillsSectionOrder,
			Budget:         startupSkillsSectionBudget,
			BudgetBehavior: PromptSectionBudgetBehaviorTrim,
			Provider:       skillsProvider,
			Predicate:      policyIncludesSection(HarnessPromptSectionSkills),
		})
	}

	descriptors = append(descriptors, PromptSectionDescriptor{
		Name:           string(HarnessPromptSectionNetwork),
		Position:       PromptSectionPositionAppend,
		Order:          startupNetworkSectionOrder,
		Budget:         startupNetworkSectionBudget,
		BudgetBehavior: PromptSectionBudgetBehaviorOmit,
		Provider:       bundledPromptSectionProvider(bundledNetworkSkillName),
		Predicate:      policyIncludesSection(HarnessPromptSectionNetwork),
	})

	return descriptors
}

func defaultStartupPromptSectionDescriptorsFromProviders(
	prependProviders []session.PromptProvider,
	appendProviders []session.PromptProvider,
) []PromptSectionDescriptor {
	var memoryProvider session.PromptProvider
	for _, provider := range prependProviders {
		if provider != nil {
			memoryProvider = provider
			break
		}
	}

	var skillsProvider session.PromptProvider
	for _, provider := range appendProviders {
		if provider != nil {
			skillsProvider = provider
			break
		}
	}

	return defaultStartupPromptSectionDescriptors(memoryProvider, skillsProvider)
}

func policyIncludesSection(section HarnessPromptSection) SectionPredicate {
	return func(policy ResolvedHarnessPolicy) bool {
		return containsHarnessSection(policy.IncludeSections, section)
	}
}

func containsHarnessSection(sections []HarnessPromptSection, target HarnessPromptSection) bool {
	return slices.Contains(sections, target)
}

type promptSectionProviderFunc func(context.Context, *workspacepkg.ResolvedWorkspace) (string, error)

func (fn promptSectionProviderFunc) PromptSection(
	ctx context.Context,
	workspace *workspacepkg.ResolvedWorkspace,
) (string, error) {
	return fn(ctx, workspace)
}

func bundledPromptSectionProvider(name string) session.PromptProvider {
	return promptSectionProviderFunc(func(context.Context, *workspacepkg.ResolvedWorkspace) (string, error) {
		content, err := bundled.LoadContent(strings.TrimSpace(name))
		if err != nil {
			return "", fmt.Errorf("daemon: load bundled startup section %q: %w", name, err)
		}
		return content, nil
	})
}
