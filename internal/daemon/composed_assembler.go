package daemon

import (
	"context"
	"fmt"
	"strings"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/session"
)

// ComposedAssembler combines prepend and append prompt providers around the
// base agent prompt.
type ComposedAssembler struct {
	prependProviders []session.PromptProvider
	appendProviders  []session.PromptProvider
}

// ComposedAssemblerOption customizes the prompt provider chain for a
// ComposedAssembler.
type ComposedAssemblerOption func(*ComposedAssembler)

var _ session.PromptAssembler = (*ComposedAssembler)(nil)

// NewComposedAssembler constructs a ComposedAssembler from prompt provider
// groups.
func NewComposedAssembler(opts ...ComposedAssemblerOption) *ComposedAssembler {
	assembler := &ComposedAssembler{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(assembler)
	}
	return assembler
}

// WithPrependPromptProviders appends variadic prompt providers ahead of the
// base agent prompt.
func WithPrependPromptProviders(providers ...session.PromptProvider) ComposedAssemblerOption {
	copied := append([]session.PromptProvider(nil), providers...)
	return func(assembler *ComposedAssembler) {
		if assembler == nil || len(copied) == 0 {
			return
		}
		assembler.prependProviders = append(assembler.prependProviders, copied...)
	}
}

// WithAppendPromptProviders appends variadic prompt providers after the base
// agent prompt.
func WithAppendPromptProviders(providers ...session.PromptProvider) ComposedAssemblerOption {
	copied := append([]session.PromptProvider(nil), providers...)
	return func(assembler *ComposedAssembler) {
		if assembler == nil || len(copied) == 0 {
			return
		}
		assembler.appendProviders = append(assembler.appendProviders, copied...)
	}
}

// Assemble renders prepend prompt sections, the trimmed base agent prompt, and
// append prompt sections into one composed system prompt.
func (a *ComposedAssembler) Assemble(ctx context.Context, agent aghconfig.AgentDef, workspace string) (string, error) {
	basePrompt := strings.TrimSpace(agent.Prompt)
	if a == nil {
		return basePrompt, nil
	}

	sections := make([]string, 0, len(a.prependProviders)+len(a.appendProviders)+1)

	prependSections, err := gatherPromptSections(ctx, workspace, "prepend", a.prependProviders)
	if err != nil {
		return "", err
	}
	sections = append(sections, prependSections...)

	if basePrompt != "" {
		sections = append(sections, basePrompt)
	}

	appendSections, err := gatherPromptSections(ctx, workspace, "append", a.appendProviders)
	if err != nil {
		return "", err
	}
	sections = append(sections, appendSections...)

	return strings.Join(sections, "\n\n"), nil
}

func gatherPromptSections(ctx context.Context, workspace string, position string, providers []session.PromptProvider) ([]string, error) {
	sections := make([]string, 0, len(providers))
	for idx, provider := range providers {
		if provider == nil {
			continue
		}

		section, err := provider.PromptSection(ctx, workspace)
		if err != nil {
			return nil, fmt.Errorf("daemon: %s prompt provider %d: %w", position, idx, err)
		}

		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}

		sections = append(sections, section)
	}

	return sections, nil
}
