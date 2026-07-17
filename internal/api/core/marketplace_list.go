package core

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/compozy/agh/internal/api/contract"
	bundlepkg "github.com/compozy/agh/internal/bundles"
	extensionpkg "github.com/compozy/agh/internal/extension"
	marketplacepkg "github.com/compozy/agh/internal/marketplace"
	registrypkg "github.com/compozy/agh/internal/registry"
	settingspkg "github.com/compozy/agh/internal/settings"
)

const (
	remoteSkillEntryPrefix = "skill_"
	bundleEntryPrefix      = "bundle_"
)

func (h *BaseHandlers) marketplaceKindResult(
	ctx context.Context,
	kind string,
	query string,
	limit int,
	scope marketplaceReadScope,
) (contract.MarketplaceKindResult, error) {
	switch kind {
	case contract.MarketplaceKindMCP:
		return h.curatedMarketplaceKind(ctx, marketplacepkg.KindMCP, query, limit, scope)
	case contract.MarketplaceKindExtension:
		return h.curatedMarketplaceKind(ctx, marketplacepkg.KindExtension, query, limit, scope)
	case contract.MarketplaceKindSkill:
		if strings.TrimSpace(query) == "" {
			return h.curatedMarketplaceKind(ctx, marketplacepkg.KindSkill, "", limit, scope)
		}
		return h.remoteSkillMarketplaceKind(ctx, query, limit)
	case contract.MarketplaceKindBundle:
		return h.bundleMarketplaceKind(ctx, query, limit, scope)
	default:
		return contract.MarketplaceKindResult{}, errors.Join(
			ErrMarketplaceNotFound, fmt.Errorf("unknown marketplace kind %q", kind),
		)
	}
}

func (h *BaseHandlers) curatedMarketplaceKind(
	ctx context.Context,
	kind marketplacepkg.Kind,
	query string,
	limit int,
	scope marketplaceReadScope,
) (contract.MarketplaceKindResult, error) {
	if h == nil || h.MarketplaceCatalog == nil {
		return contract.MarketplaceKindResult{}, errors.Join(
			ErrMarketplaceUnavailable, errors.New("catalog is not configured"),
		)
	}
	result, err := h.MarketplaceCatalog.Browse(ctx, kind, query, limit)
	if err != nil {
		return contract.MarketplaceKindResult{}, err
	}
	installed, err := h.marketplaceInstallIndex(ctx, string(kind), scope)
	if err != nil {
		return contract.MarketplaceKindResult{}, err
	}
	items := make([]contract.MarketplaceListingPayload, 0, len(result.Entries))
	for _, entry := range result.Entries {
		item, mapErr := h.curatedMarketplaceListing(ctx, entry, installed)
		if mapErr != nil {
			return contract.MarketplaceKindResult{}, mapErr
		}
		items = append(items, item)
	}
	return contract.MarketplaceKindResult{
		Kind:       string(kind),
		Stale:      result.State.Stale,
		ErrorClass: result.State.ErrorClass,
		Error:      result.State.LastError,
		Items:      items,
	}, nil
}

func (h *BaseHandlers) remoteSkillMarketplaceKind(
	ctx context.Context,
	query string,
	limit int,
) (contract.MarketplaceKindResult, error) {
	if h == nil {
		return contract.MarketplaceKindResult{}, errors.Join(
			ErrMarketplaceUnavailable, errors.New("skill marketplace is not configured"),
		)
	}
	listings, err := h.skillMarketplaceService().Search(ctx, query, limit)
	if err != nil {
		return contract.MarketplaceKindResult{}, err
	}
	installed, err := h.skillInstallIndex(ctx)
	if err != nil {
		return contract.MarketplaceKindResult{}, err
	}
	curatedEntryIDs, err := h.curatedSkillEntryIDs(ctx, listings)
	if err != nil {
		return contract.MarketplaceKindResult{}, err
	}
	items := make([]contract.MarketplaceListingPayload, 0, len(listings))
	for _, listing := range listings {
		item := remoteSkillMarketplaceListing(listing, installed)
		if entryID := curatedEntryIDs[strings.TrimSpace(listing.Slug)]; entryID != "" {
			item.EntryID = entryID
		}
		items = append(items, item)
	}
	return contract.MarketplaceKindResult{Kind: contract.MarketplaceKindSkill, Items: items}, nil
}

func (h *BaseHandlers) curatedSkillEntryIDs(
	ctx context.Context,
	listings []registrypkg.Listing,
) (map[string]string, error) {
	entryIDs := make(map[string]string)
	if h.MarketplaceCatalog == nil || len(listings) == 0 {
		return entryIDs, nil
	}
	slugs := make([]string, 0, len(listings))
	for _, listing := range listings {
		if slug := strings.TrimSpace(listing.Slug); slug != "" {
			slugs = append(slugs, slug)
		}
	}
	if len(slugs) == 0 {
		return entryIDs, nil
	}
	entries, err := h.MarketplaceCatalog.ResolveSkillInstalls(ctx, slugs)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if slug := strings.TrimSpace(entry.InstallSlug); slug != "" {
			entryIDs[slug] = entry.EntryID
		}
	}
	return entryIDs, nil
}

func (h *BaseHandlers) bundleMarketplaceKind(
	ctx context.Context,
	query string,
	limit int,
	scope marketplaceReadScope,
) (contract.MarketplaceKindResult, error) {
	if h == nil || h.Bundles == nil {
		return contract.MarketplaceKindResult{}, errors.Join(
			ErrMarketplaceUnavailable, errors.New("bundle catalog is not configured"),
		)
	}
	catalog, err := h.Bundles.Catalog(ctx)
	if err != nil {
		return contract.MarketplaceKindResult{}, err
	}
	activations, err := h.Bundles.ListActivations(ctx)
	if err != nil {
		return contract.MarketplaceKindResult{}, err
	}
	activationIndex := bundleActivationIndex(activations, scope)
	needle := strings.ToLower(strings.TrimSpace(query))
	filtered := make([]bundlepkg.CatalogEntry, 0, len(catalog))
	for _, entry := range catalog {
		if needle != "" && !strings.Contains(strings.ToLower(
			entry.ExtensionName+" "+entry.Bundle.Name+" "+entry.Bundle.Description,
		), needle) {
			continue
		}
		filtered = append(filtered, entry)
	}
	total := len(filtered)
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	items := make([]contract.MarketplaceListingPayload, 0, len(filtered))
	for _, entry := range filtered {
		key := bundleIdentityKey(entry.ExtensionName, entry.Bundle.Name)
		activation, installed := activationIndex[key]
		managePath := ""
		if installed {
			managePath = "/extensions/bundles/" + url.PathEscape(activation.Activation.ID)
		}
		items = append(items, contract.MarketplaceListingPayload{
			Kind: contract.MarketplaceKindBundle, EntryID: encodeBundleEntryID(entry.ExtensionName, entry.Bundle.Name),
			Name: entry.Bundle.Name, Description: entry.Bundle.Description, Source: entry.ExtensionName,
			Installed: installed, UpdateAvailable: installed && activation.SpecDrift, ManagePath: managePath,
		})
	}
	return contract.MarketplaceKindResult{
		Kind: contract.MarketplaceKindBundle, Total: &total, Items: items,
	}, nil
}

type marketplaceInstall struct {
	name       string
	version    string
	managePath string
}

func (h *BaseHandlers) marketplaceInstallIndex(
	ctx context.Context,
	kind string,
	scope marketplaceReadScope,
) (map[string]marketplaceInstall, error) {
	switch kind {
	case contract.MarketplaceKindMCP:
		return h.mcpInstallIndex(ctx, scope)
	case contract.MarketplaceKindExtension:
		return h.extensionInstallIndex(ctx)
	case contract.MarketplaceKindSkill:
		return h.skillInstallIndex(ctx)
	default:
		return nil, errors.Join(ErrMarketplaceNotFound, fmt.Errorf("unknown marketplace kind %q", kind))
	}
}

func (h *BaseHandlers) mcpInstallIndex(
	ctx context.Context,
	scope marketplaceReadScope,
) (map[string]marketplaceInstall, error) {
	if h.Settings == nil {
		return nil, errors.Join(ErrMarketplaceUnavailable, errors.New("settings service is not configured"))
	}
	envelope, err := h.Settings.ListCollection(ctx, settingspkg.CollectionRequest{
		Collection: settingspkg.CollectionMCPServers, Scope: scope.scope, WorkspaceID: scope.workspaceID,
	})
	if err != nil {
		return nil, err
	}
	index := make(map[string]marketplaceInstall)
	for _, item := range envelope.MCPServers {
		entryID := strings.TrimSpace(item.CatalogEntry)
		if entryID == "" {
			continue
		}
		managePath := "/mcp?" + url.Values{
			"scope":  {string(scope.scope)},
			"server": {strings.TrimSpace(item.Name)},
		}.Encode()
		index[entryID] = marketplaceInstall{
			name:       strings.TrimSpace(item.Name),
			managePath: managePath,
		}
	}
	return index, nil
}

func (h *BaseHandlers) extensionInstallIndex(ctx context.Context) (map[string]marketplaceInstall, error) {
	if h.Extensions == nil {
		return nil, errors.Join(ErrMarketplaceUnavailable, errors.New("extension service is not configured"))
	}
	items, err := h.Extensions.List(ctx)
	if err != nil {
		return nil, err
	}
	index := make(map[string]marketplaceInstall)
	for _, item := range items {
		if item.Provenance == nil || strings.TrimSpace(item.Provenance.Slug) == "" {
			continue
		}
		index[strings.TrimSpace(item.Provenance.Slug)] = marketplaceInstall{
			name:       strings.TrimSpace(item.Name),
			version:    strings.TrimSpace(item.Version),
			managePath: "/extensions/" + url.PathEscape(strings.TrimSpace(item.Name)),
		}
	}
	return index, nil
}

func (h *BaseHandlers) skillInstallIndex(ctx context.Context) (map[string]marketplaceInstall, error) {
	service := h.InstalledSkillMarketplace
	if service == nil {
		if candidate, ok := h.skillMarketplaceService().(InstalledSkillMarketplaceService); ok {
			service = candidate
		}
	}
	if service == nil {
		return nil, errors.Join(ErrMarketplaceUnavailable, errors.New("skill marketplace is not configured"))
	}
	items, err := service.ListInstalled(ctx)
	if err != nil {
		return nil, err
	}
	index := make(map[string]marketplaceInstall)
	for _, item := range items {
		slug := strings.TrimSpace(item.Provenance.Slug)
		if slug == "" {
			continue
		}
		index[slug] = marketplaceInstall{
			name:       strings.TrimSpace(item.Name),
			version:    strings.TrimSpace(item.Provenance.Version),
			managePath: "/skills/" + url.PathEscape(strings.TrimSpace(item.Name)),
		}
	}
	return index, nil
}

func (h *BaseHandlers) curatedMarketplaceListing(
	ctx context.Context,
	entry marketplacepkg.Entry,
	installed map[string]marketplaceInstall,
) (contract.MarketplaceListingPayload, error) {
	details, err := marketplacepkg.ProjectEntry(entry)
	if err != nil {
		return contract.MarketplaceListingPayload{}, err
	}
	identity := entry.EntryID
	if entry.InstallSlug != "" {
		identity = entry.InstallSlug
	}
	installation, isInstalled := installed[identity]
	updateAvailable := false
	if entry.Kind == marketplacepkg.KindExtension || entry.Kind == marketplacepkg.KindSkill {
		updateAvailable = isInstalled && registrypkg.VersionIsNewer(installation.version, entry.Version)
	}
	result := contract.MarketplaceListingPayload{
		Kind: string(entry.Kind), EntryID: entry.EntryID, Name: entry.Name, Description: entry.Description,
		Version: entry.Version, Author: details.Author, Source: details.Source,
		InstallSlug: marketplaceInstallSlug(details),
		Transport:   marketplaceTransport(details), Tier: entry.Tier,
		PublishedAt: entry.PublishedAt, UpdatedAt: entry.UpdatedAt,
		Installed: isInstalled, InstalledName: installation.name, InstalledVersion: installation.version,
		UpdateAvailable: updateAvailable,
		ManagePath:      installation.managePath,
	}
	if entry.Kind != marketplacepkg.KindExtension {
		return result, nil
	}
	trust, err := h.Extensions.MarketplaceTrust(ctx, extensionpkg.MarketplaceTrustEvidence{
		CatalogEntryID:      entry.EntryID,
		Version:             entry.Version,
		ArchiveDigestSHA256: entry.DigestSHA256,
		RegistryTier:        entry.Tier,
	})
	if err != nil {
		return contract.MarketplaceListingPayload{}, err
	}
	result.Trust = &trust
	return result, nil
}

func marketplaceInstallSlug(details marketplacepkg.EntryDetails) string {
	switch {
	case details.Extension != nil:
		return details.Extension.InstallSlug
	case details.Skill != nil:
		return details.Skill.InstallSlug
	default:
		return ""
	}
}

func marketplaceTransport(details marketplacepkg.EntryDetails) string {
	if details.MCP == nil {
		return ""
	}
	return details.MCP.Transport
}

func remoteSkillMarketplaceListing(
	listing registrypkg.Listing,
	installed map[string]marketplaceInstall,
) contract.MarketplaceListingPayload {
	slug := strings.TrimSpace(listing.Slug)
	installation, isInstalled := installed[slug]
	downloads := listing.Downloads
	return contract.MarketplaceListingPayload{
		Kind: contract.MarketplaceKindSkill, EntryID: encodeRemoteSkillEntryID(slug),
		Name: listing.Name, Description: listing.Description, Version: listing.Version,
		Author: listing.Author, Downloads: &downloads, InstallSlug: slug, Source: listing.Source,
		Installed: isInstalled, InstalledName: installation.name, InstalledVersion: installation.version,
		UpdateAvailable: isInstalled && registrypkg.VersionIsNewer(installation.version, listing.Version),
		ManagePath:      installation.managePath,
	}
}

func bundleActivationIndex(
	activations []bundlepkg.ActivationPreview,
	scope marketplaceReadScope,
) map[string]bundlepkg.ActivationPreview {
	sort.SliceStable(activations, func(i, j int) bool {
		return activations[i].Activation.ID < activations[j].Activation.ID
	})
	index := make(map[string]bundlepkg.ActivationPreview)
	for _, item := range activations {
		activation := item.Activation
		visible := activation.Scope == bundlepkg.ScopeGlobal ||
			scope.scope == settingspkg.ScopeWorkspace && activation.Scope == bundlepkg.ScopeWorkspace &&
				strings.TrimSpace(activation.WorkspaceID) == scope.workspaceID
		if !visible {
			continue
		}
		key := bundleIdentityKey(activation.ExtensionName, activation.BundleName)
		if _, exists := index[key]; !exists {
			index[key] = item
		}
	}
	return index
}

func encodeRemoteSkillEntryID(slug string) string {
	return remoteSkillEntryPrefix + base64.RawURLEncoding.EncodeToString([]byte(strings.TrimSpace(slug)))
}

func decodeRemoteSkillEntryID(entryID string) (string, bool) {
	if !strings.HasPrefix(entryID, remoteSkillEntryPrefix) {
		return "", false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(entryID, remoteSkillEntryPrefix))
	if err != nil || strings.TrimSpace(string(decoded)) == "" {
		return "", false
	}
	return strings.TrimSpace(string(decoded)), true
}

func encodeBundleEntryID(extensionName string, bundleName string) string {
	return bundleEntryPrefix + base64.RawURLEncoding.EncodeToString(
		[]byte(bundleIdentityKey(extensionName, bundleName)),
	)
}

func decodeBundleEntryID(entryID string) (string, string, bool) {
	if !strings.HasPrefix(entryID, bundleEntryPrefix) {
		return "", "", false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(entryID, bundleEntryPrefix))
	if err != nil {
		return "", "", false
	}
	parts := strings.Split(string(decoded), "\x00")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func bundleIdentityKey(extensionName string, bundleName string) string {
	return strings.TrimSpace(extensionName) + "\x00" + strings.TrimSpace(bundleName)
}
