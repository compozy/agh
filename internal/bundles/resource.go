package bundles

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pedronauck/agh/internal/resources"

	extensionpkg "github.com/pedronauck/agh/internal/extension"
)

const (
	// BundleResourceKind is the canonical desired-state kind for extension bundles.
	BundleResourceKind resources.ResourceKind = "bundle"
	// BundleActivationResourceKind is the canonical desired-state kind for active bundle profiles.
	BundleActivationResourceKind resources.ResourceKind = "bundle.activation"

	BundleActivationOwnerKind resources.ResourceOwnerKind = "bundle.activation"

	bundleResourceMaxBytes           = 512 << 10
	bundleActivationResourceMaxBytes = 64 << 10
)

// BundleResourceSpec is the canonical desired-state payload for bundle catalog records.
type BundleResourceSpec struct {
	ExtensionName              string                  `json:"extension_name"`
	Bundle                     extensionpkg.BundleSpec `json:"bundle"`
	OwnerBridgePlatform        string                  `json:"owner_bridge_platform,omitempty"`
	OwnerProvidesBridgeAdapter bool                    `json:"owner_provides_bridge_adapter,omitempty"`
}

// ActivationResourceSpec is the canonical desired-state payload for bundle activation records.
type ActivationResourceSpec struct {
	ExtensionName               string `json:"extension_name"`
	BundleName                  string `json:"bundle_name"`
	ProfileName                 string `json:"profile_name"`
	SpecContentHash             string `json:"spec_content_hash,omitempty"`
	BindPrimaryChannelAsDefault bool   `json:"bind_primary_channel_default"`
}

// NewBundleResourceCodec builds the typed codec for bundle records.
func NewBundleResourceCodec() (resources.KindCodec[BundleResourceSpec], error) {
	return resources.NewJSONCodec(BundleResourceKind, bundleResourceMaxBytes, validateBundleResourceSpec)
}

// NewActivationResourceCodec builds the typed codec for bundle.activation records.
func NewActivationResourceCodec() (resources.KindCodec[ActivationResourceSpec], error) {
	return resources.NewJSONCodec(
		BundleActivationResourceKind,
		bundleActivationResourceMaxBytes,
		validateActivationResourceSpec,
	)
}

// BundleResourceID returns the stable canonical resource ID for one extension bundle.
func BundleResourceID(extensionName string, bundleName string) string {
	return stableID("bun", extensionName, bundleName)
}

// ActivationResourceID returns the stable canonical resource ID for one bundle profile activation.
func ActivationResourceID(
	extensionName string,
	bundleName string,
	profileName string,
	scope Scope,
	workspaceID string,
) string {
	return stableID(
		"act",
		extensionName,
		bundleName,
		profileName,
		string(scope.Normalize()),
		strings.TrimSpace(workspaceID),
	)
}

func validateBundleResourceSpec(
	_ context.Context,
	scope resources.ResourceScope,
	spec BundleResourceSpec,
) (BundleResourceSpec, error) {
	normalizedScope := scope.Normalize()
	if err := normalizedScope.Validate("scope"); err != nil {
		return BundleResourceSpec{}, err
	}
	next := normalizeBundleResourceSpec(spec)
	if strings.TrimSpace(next.ExtensionName) == "" {
		return BundleResourceSpec{}, errors.New("bundles: resource extension_name is required")
	}
	manifest := &extensionpkg.Manifest{
		Name: strings.TrimSpace(next.ExtensionName),
		Bridge: extensionpkg.BridgeConfig{
			Platform: strings.TrimSpace(next.OwnerBridgePlatform),
		},
	}
	if next.OwnerProvidesBridgeAdapter || strings.TrimSpace(next.OwnerBridgePlatform) != "" {
		manifest.Capabilities.Provides = []string{"bridge.adapter"}
		next.OwnerProvidesBridgeAdapter = true
	}
	if err := next.Bundle.Validate(manifest); err != nil {
		return BundleResourceSpec{}, fmt.Errorf("bundles: validate bundle resource: %w", err)
	}
	return next, nil
}

func validateActivationResourceSpec(
	_ context.Context,
	scope resources.ResourceScope,
	spec ActivationResourceSpec,
) (ActivationResourceSpec, error) {
	normalizedScope := scope.Normalize()
	if err := normalizedScope.Validate("scope"); err != nil {
		return ActivationResourceSpec{}, err
	}
	next := normalizeActivationResourceSpec(spec)
	if strings.TrimSpace(next.ExtensionName) == "" {
		return ActivationResourceSpec{}, errors.New("bundles: activation extension_name is required")
	}
	if strings.TrimSpace(next.BundleName) == "" {
		return ActivationResourceSpec{}, errors.New("bundles: activation bundle_name is required")
	}
	if strings.TrimSpace(next.ProfileName) == "" {
		return ActivationResourceSpec{}, errors.New("bundles: activation profile_name is required")
	}
	return next, nil
}

func normalizeBundleResourceSpec(spec BundleResourceSpec) BundleResourceSpec {
	next := spec
	next.ExtensionName = strings.TrimSpace(next.ExtensionName)
	next.OwnerBridgePlatform = strings.TrimSpace(next.OwnerBridgePlatform)
	next.Bundle = cloneBundleSpec(next.Bundle)
	next.Bundle.Name = strings.TrimSpace(next.Bundle.Name)
	next.Bundle.Description = strings.TrimSpace(next.Bundle.Description)
	return next
}

func normalizeActivationResourceSpec(spec ActivationResourceSpec) ActivationResourceSpec {
	return ActivationResourceSpec{
		ExtensionName:               strings.TrimSpace(spec.ExtensionName),
		BundleName:                  strings.TrimSpace(spec.BundleName),
		ProfileName:                 strings.TrimSpace(spec.ProfileName),
		SpecContentHash:             strings.TrimSpace(spec.SpecContentHash),
		BindPrimaryChannelAsDefault: spec.BindPrimaryChannelAsDefault,
	}
}

func activationResourceSpecFromActivation(activation Activation) ActivationResourceSpec {
	return ActivationResourceSpec{
		ExtensionName:               strings.TrimSpace(activation.ExtensionName),
		BundleName:                  strings.TrimSpace(activation.BundleName),
		ProfileName:                 strings.TrimSpace(activation.ProfileName),
		SpecContentHash:             strings.TrimSpace(activation.SpecContentHash),
		BindPrimaryChannelAsDefault: activation.BindPrimaryChannelAsDefault,
	}
}

func activationFromResourceRecord(record resources.Record[ActivationResourceSpec]) Activation {
	scope := ScopeGlobal
	workspaceID := ""
	if record.Scope.Kind == resources.ResourceScopeKindWorkspace {
		scope = ScopeWorkspace
		workspaceID = strings.TrimSpace(record.Scope.ID)
	}
	return Activation{
		ID:                          strings.TrimSpace(record.ID),
		ExtensionName:               strings.TrimSpace(record.Spec.ExtensionName),
		BundleName:                  strings.TrimSpace(record.Spec.BundleName),
		ProfileName:                 strings.TrimSpace(record.Spec.ProfileName),
		Scope:                       scope,
		WorkspaceID:                 workspaceID,
		SpecContentHash:             strings.TrimSpace(record.Spec.SpecContentHash),
		BindPrimaryChannelAsDefault: record.Spec.BindPrimaryChannelAsDefault,
		CreatedAt:                   record.CreatedAt,
		UpdatedAt:                   record.UpdatedAt,
	}
}

func resourceScopeForActivation(activation Activation) resources.ResourceScope {
	if activation.Scope.Normalize() == ScopeWorkspace {
		return resources.ResourceScope{
			Kind: resources.ResourceScopeKindWorkspace,
			ID:   strings.TrimSpace(activation.WorkspaceID),
		}
	}
	return resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal}
}

func ownerForActivation(id string) resources.ResourceOwner {
	return resources.ResourceOwner{
		Kind: BundleActivationOwnerKind,
		ID:   strings.TrimSpace(id),
	}
}

func activationResourceActor(base resources.MutationActor, activationID string) resources.MutationActor {
	trimmedID := strings.TrimSpace(activationID)
	actor := base
	if actor.Kind == "" {
		actor.Kind = resources.MutationActorKindDaemon
	}
	if actor.MaxScope.Kind == "" {
		actor.MaxScope = resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal}
	}
	actor.ID = "bundle.activation." + trimmedID
	actor.Owner = ownerForActivation(trimmedID)
	actor.Source = resources.ResourceSource{
		Kind: resources.ResourceSourceKind("daemon"),
		ID:   actor.ID,
	}
	return actor
}
