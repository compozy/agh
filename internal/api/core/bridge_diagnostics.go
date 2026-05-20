package core

import (
	"context"
	"fmt"
	"strings"

	"github.com/pedronauck/agh/internal/api/contract"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
)

func (h *BaseHandlers) bridgeHealthPayloadForInstance(
	ctx context.Context,
	bridges BridgeService,
	instance bridgepkg.BridgeInstance,
	health contract.BridgeHealthPayload,
) (contract.BridgeHealthPayload, error) {
	instanceID := strings.TrimSpace(instance.ID)
	if health.BridgeInstanceID == "" {
		health.BridgeInstanceID = instanceID
	}
	if health.Status == "" {
		health.Status = instance.Status
	}
	health.Degradation = cloneBridgeDegradation(instance.Degradation)
	diagnostics, err := h.bridgeDiagnosticsForInstance(ctx, bridges, instance, health)
	if err != nil {
		return contract.BridgeHealthPayload{}, err
	}
	health.Diagnostics = diagnostics
	return health, nil
}

func (h *BaseHandlers) bridgeDiagnosticsForInstance(
	ctx context.Context,
	bridges BridgeService,
	instance bridgepkg.BridgeInstance,
	health contract.BridgeHealthPayload,
) ([]bridgepkg.BridgeDiagnostic, error) {
	provider, catalogAvailable, err := bridgeProviderForInstance(ctx, bridges, instance)
	if err != nil {
		return nil, err
	}
	bindings, err := bridgeSecretBindingsForDiagnostics(ctx, bridges, instance, provider)
	if err != nil {
		return nil, err
	}
	return bridgepkg.BuildBridgeDiagnostics(bridgepkg.BridgeDiagnosticsInput{
		Instance:                 instance,
		Provider:                 provider,
		ProviderCatalogAvailable: catalogAvailable,
		SecretBindings:           bindings,
		RouteCount:               health.RouteCount,
		DeliveryBacklog:          health.DeliveryBacklog,
		DeliveryFailuresTotal:    health.DeliveryFailuresTotal,
		AuthFailuresTotal:        health.AuthFailuresTotal,
		LastError:                health.LastError,
	}), nil
}

func bridgeProviderForInstance(
	ctx context.Context,
	bridges BridgeService,
	instance bridgepkg.BridgeInstance,
) (*bridgepkg.BridgeProvider, bool, error) {
	providers, err := bridges.ListProviders(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("load bridge providers for diagnostics: %w", err)
	}
	catalogAvailable := len(providers) > 0
	for _, provider := range providers {
		if !bridgeProviderMatchesInstance(provider, instance) {
			continue
		}
		matched := provider
		return &matched, catalogAvailable, nil
	}
	return nil, catalogAvailable, nil
}

func bridgeSecretBindingsForDiagnostics(
	ctx context.Context,
	bridges BridgeService,
	instance bridgepkg.BridgeInstance,
	provider *bridgepkg.BridgeProvider,
) ([]bridgepkg.BridgeSecretBinding, error) {
	if provider == nil || !bridgeProviderHasRequiredSecretSlots(*provider) {
		return nil, nil
	}
	bindings, err := bridges.ListSecretBindings(ctx, strings.TrimSpace(instance.ID))
	if err != nil {
		return nil, fmt.Errorf("load bridge secret bindings for diagnostics: %w", err)
	}
	return bindings, nil
}

func bridgeProviderMatchesInstance(provider bridgepkg.BridgeProvider, instance bridgepkg.BridgeInstance) bool {
	return strings.EqualFold(strings.TrimSpace(provider.Platform), strings.TrimSpace(instance.Platform)) &&
		strings.EqualFold(strings.TrimSpace(provider.ExtensionName), strings.TrimSpace(instance.ExtensionName))
}

func bridgeProviderHasRequiredSecretSlots(provider bridgepkg.BridgeProvider) bool {
	for _, slot := range provider.SecretSlots {
		normalized := slot.Normalize()
		if normalized.Required && strings.TrimSpace(normalized.Name) != "" {
			return true
		}
	}
	return false
}
