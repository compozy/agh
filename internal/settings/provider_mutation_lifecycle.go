package settings

import (
	"context"
	"fmt"
	"maps"
	"reflect"

	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/config/lifecycle"
)

func (s *service) validateProviderWrite(
	ctx context.Context,
	name string,
	settings ProviderSettings,
) (bool, error) {
	currentConfig, _, err := s.loadConfig(ctx, ScopeGlobal, "")
	if err != nil {
		return false, fmt.Errorf("load config for provider %q mutation validation: %w", name, err)
	}
	currentProvider, currentErr := currentConfig.ResolveProvider(name)
	nextConfig := currentConfig
	nextConfig.Providers = make(map[string]aghconfig.ProviderConfig, len(currentConfig.Providers)+1)
	maps.Copy(nextConfig.Providers, currentConfig.Providers)
	nextConfig.Providers[name] = providerConfigFromSettings(settings)
	if err := nextConfig.Validate(); err != nil {
		return false, fmt.Errorf("validate provider %q mutation: %w", name, err)
	}
	nextProvider, err := nextConfig.ResolveProvider(name)
	if err != nil {
		return false, fmt.Errorf("resolve provider %q after mutation validation: %w", name, err)
	}
	if currentErr != nil {
		return false, nil
	}
	currentProvider.Models = aghconfig.ProviderModelsConfig{}
	nextProvider.Models = aghconfig.ProviderModelsConfig{}
	return reflect.DeepEqual(currentProvider, nextProvider), nil
}

func mutationResultForProvider(target WriteTargetKind, modelOnly bool) MutationResult {
	if !modelOnly {
		return mutationResultForCollection(CollectionProviders, ScopeGlobal, "", target)
	}
	return MutationResult{
		Section:     SectionName(CollectionProviders),
		Scope:       ScopeGlobal,
		WriteTarget: target,
		Behavior:    MutationBehaviorAppliedNow,
		Applied:     true,
		Lifecycle:   lifecycle.Live,
		DiffClass:   lifecycle.DiffClassLive,
	}
}
