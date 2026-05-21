package extensionpkg

import (
	"errors"
	"fmt"
	"strings"
	"time"

	contract "github.com/pedronauck/agh/internal/diagnosticcontract"
	"github.com/pedronauck/agh/internal/diagnostics"
)

const (
	ExtensionInstalledFromMarketplace = "marketplace_registry"
	ExtensionInstalledFromLocalPath   = "local_path"
	ExtensionInstalledFromGitURL      = "git_url"

	ExtensionRegistryTierOfficial   = "official"
	ExtensionRegistryTierCommunity  = "community"
	ExtensionRegistryTierUnverified = "unverified"

	ExtensionTrustDecisionVerified          = "verified"
	ExtensionTrustDecisionAllowedUnverified = "allowed_unverified"
	ExtensionTrustDecisionBlocked           = "blocked"

	extensionTrustInstalledByOperator = "operator"
	extensionTrustEvidenceSourceKey   = "source"
	extensionTrustGitHubSource        = "github"
)

var ErrExtensionChecksumUnverified = errors.New("extension: checksum is unverified")

// ExtensionProvenance records one installed extension's source and trust state.
type ExtensionProvenance struct {
	Slug             string                    `json:"slug,omitempty"`
	InstalledFrom    string                    `json:"installed_from"`
	SourceURL        string                    `json:"source_url,omitempty"`
	ChecksumSHA256   string                    `json:"checksum_sha256"`
	ChecksumVerified bool                      `json:"checksum_verified"`
	RegistryTier     string                    `json:"registry_tier"`
	Permissions      []string                  `json:"permissions,omitempty"`
	InstalledAt      time.Time                 `json:"installed_at"`
	InstalledBy      string                    `json:"installed_by"`
	AllowUnverified  bool                      `json:"allow_unverified"`
	Warnings         []contract.DiagnosticItem `json:"warnings,omitempty"`
}

// ExtensionTrustError carries the canonical diagnostic for a denied extension
// trust decision.
type ExtensionTrustError struct {
	Slug   string
	Source string
	Item   contract.DiagnosticItem
}

func (e *ExtensionTrustError) Error() string {
	slug := strings.TrimSpace(e.Slug)
	if slug == "" {
		slug = managerExtensionKey
	}
	return fmt.Sprintf("%s: %s", ErrExtensionChecksumUnverified, slug)
}

func (e *ExtensionTrustError) Unwrap() error {
	return ErrExtensionChecksumUnverified
}

func (e *ExtensionTrustError) DiagnosticItem() contract.DiagnosticItem {
	if e == nil {
		return diagnostics.EmptyItem()
	}
	return e.Item
}

func newExtensionChecksumUnverifiedError(slug string, source string) *ExtensionTrustError {
	item := extensionChecksumUnverifiedDiagnostic(slug, source, false)
	return &ExtensionTrustError{Slug: strings.TrimSpace(slug), Source: strings.TrimSpace(source), Item: item}
}

// NewExtensionChecksumUnverifiedError returns the canonical trust-gate error.
func NewExtensionChecksumUnverifiedError(slug string, source string) *ExtensionTrustError {
	return newExtensionChecksumUnverifiedError(slug, source)
}

// LocalPathProvenance records an explicit trust decision for a local install.
func LocalPathProvenance(
	manifest *Manifest,
	sourcePath string,
	checksum string,
	installedAt time.Time,
	allowUnverified bool,
) ExtensionProvenance {
	provenance := ExtensionProvenance{
		InstalledFrom:    ExtensionInstalledFromLocalPath,
		SourceURL:        strings.TrimSpace(sourcePath),
		ChecksumSHA256:   strings.TrimSpace(checksum),
		ChecksumVerified: false,
		RegistryTier:     ExtensionRegistryTierUnverified,
		InstalledAt:      installedAt.UTC(),
		InstalledBy:      extensionTrustInstalledByOperator,
		AllowUnverified:  allowUnverified,
	}
	if manifest != nil {
		provenance.Permissions = extensionPermissions(manifest.Capabilities, manifest.Actions)
		provenance.Warnings = []contract.DiagnosticItem{
			extensionChecksumUnverifiedDiagnostic(manifest.Name, sourcePath, allowUnverified),
		}
	}
	return provenance
}

func extensionChecksumUnverifiedDiagnostic(
	slug string,
	source string,
	allowed bool,
) contract.DiagnosticItem {
	title := "Extension checksum is unverified"
	message := "The marketplace did not provide a verifiable checksum for this extension."
	if allowed {
		message = "The extension was installed after an explicit allow_unverified trust decision."
	}
	return diagnostics.NewItem(
		"extension.checksum_unverified",
		contract.CodeExtensionChecksumUnverified,
		contract.CategoryExtension,
		title,
		message,
		contract.SeverityWarn,
		contract.FreshnessLive,
		diagnostics.WithEvidence(map[string]any{
			"slug":                          strings.TrimSpace(slug),
			extensionTrustEvidenceSourceKey: strings.TrimSpace(source),
			"allow_unverified":              allowed,
		}),
	)
}

func normalizeExtensionProvenance(value ExtensionProvenance, fallback ExtensionProvenance) ExtensionProvenance {
	if strings.TrimSpace(value.InstalledFrom) == "" {
		value.InstalledFrom = fallback.InstalledFrom
	}
	if strings.TrimSpace(value.SourceURL) == "" {
		value.SourceURL = fallback.SourceURL
	}
	if strings.TrimSpace(value.ChecksumSHA256) == "" {
		value.ChecksumSHA256 = fallback.ChecksumSHA256
	}
	if strings.TrimSpace(value.RegistryTier) == "" {
		value.RegistryTier = fallback.RegistryTier
	}
	if strings.TrimSpace(value.InstalledBy) == "" {
		value.InstalledBy = fallback.InstalledBy
	}
	if value.InstalledAt.IsZero() {
		value.InstalledAt = fallback.InstalledAt
	}
	if len(value.Permissions) == 0 {
		value.Permissions = append([]string(nil), fallback.Permissions...)
	}
	if len(value.Warnings) > 0 {
		value.Warnings = append([]contract.DiagnosticItem(nil), value.Warnings...)
	}
	value.Slug = strings.TrimSpace(value.Slug)
	value.InstalledFrom = strings.TrimSpace(value.InstalledFrom)
	value.SourceURL = strings.TrimSpace(value.SourceURL)
	value.ChecksumSHA256 = strings.TrimSpace(value.ChecksumSHA256)
	value.RegistryTier = strings.TrimSpace(value.RegistryTier)
	value.InstalledBy = strings.TrimSpace(value.InstalledBy)
	return value
}

func extensionPermissions(capabilities CapabilitiesConfig, actions ActionsConfig) []string {
	items := make([]string, 0, len(capabilities.Provides)+len(actions.Requires))
	for _, value := range capabilities.Provides {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}
	for _, value := range actions.Requires {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

func installedFromForSource(source ExtensionSource) string {
	switch source {
	case SourceMarketplace:
		return ExtensionInstalledFromMarketplace
	case SourceUser, SourceWorkspace, SourceBundled:
		return ExtensionInstalledFromLocalPath
	default:
		return ExtensionInstalledFromLocalPath
	}
}

func registryTierForSource(source ExtensionSource, registryName string) string {
	if source != SourceMarketplace {
		return ExtensionRegistryTierUnverified
	}
	switch strings.ToLower(strings.TrimSpace(registryName)) {
	case extensionTrustGitHubSource:
		return ExtensionRegistryTierCommunity
	case "":
		return ExtensionRegistryTierUnverified
	default:
		return ExtensionRegistryTierCommunity
	}
}

func extensionTrustDecision(provenance ExtensionProvenance) string {
	if provenance.ChecksumVerified {
		return ExtensionTrustDecisionVerified
	}
	if provenance.AllowUnverified {
		return ExtensionTrustDecisionAllowedUnverified
	}
	return ExtensionTrustDecisionBlocked
}
