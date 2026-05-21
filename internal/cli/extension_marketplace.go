package cli

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	extensionpkg "github.com/pedronauck/agh/internal/extension"
)

const (
	extensionMarketplaceSlugValue = "Slug"
	extensionMarketplaceSlugKey   = "slug"
)

const (
	skillOutputRegistryKey = "registry"
)

const (
	extensionMarketplaceDescriptionValue  = "Description"
	extensionMarketplacePathValue         = "Path"
	extensionMarketplaceStatusValue       = "Status"
	extensionMarketplaceCurrentVersionKey = "current_version"
	extensionMarketplaceDescriptionKey    = "description"
	extensionMarketplacePathKey           = "path"
)

const (
	defaultExtensionRegistrySearchLimit = 20
)

type extensionRemoveItem = ManagedExtensionRemoveRecord

type extensionUpdateItem = ExtensionUpdateRecord

func searchExtensions(
	ctx context.Context,
	deps commandDeps,
	query string,
	sourceFilter string,
	limit int,
) ([]ExtensionMarketplaceRecord, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("cli: search limit must be positive: %d", limit)
	}
	client, err := requireExtensionDaemonClient(deps)
	if err != nil {
		return nil, err
	}
	return client.SearchExtensionMarketplace(ctx, query, sourceFilter, limit)
}

func installMarketplaceExtension(
	ctx context.Context,
	deps commandDeps,
	slug string,
	sourceFilter string,
	version string,
	asset string,
	allowUnverified bool,
) (ExtensionRecord, error) {
	client, err := requireExtensionDaemonClient(deps)
	if err != nil {
		return ExtensionRecord{}, err
	}
	item, err := client.InstallExtension(ctx, InstallExtensionRequest{
		Slug:            strings.TrimSpace(slug),
		Source:          strings.TrimSpace(sourceFilter),
		Version:         strings.TrimSpace(version),
		Asset:           strings.TrimSpace(asset),
		AllowUnverified: allowUnverified,
	})
	if err != nil {
		return ExtensionRecord{}, err
	}
	return item, nil
}

func removeInstalledExtension(
	ctx context.Context,
	deps commandDeps,
	name string,
) (extensionRemoveItem, error) {
	client, err := requireExtensionDaemonClient(deps)
	if err != nil {
		return extensionRemoveItem{}, err
	}
	return client.RemoveExtension(ctx, name)
}

func updateMarketplaceExtensions(
	ctx context.Context,
	deps commandDeps,
	args []string,
	updateAll bool,
	checkOnly bool,
	version string,
	allowUnverified bool,
) ([]extensionUpdateItem, error) {
	client, err := requireExtensionDaemonClient(deps)
	if err != nil {
		return nil, err
	}
	targets, err := selectDaemonMarketplaceExtensionsForUpdate(ctx, client, args, updateAll)
	if err != nil {
		return nil, err
	}
	items := make([]extensionUpdateItem, 0, len(targets))
	for _, target := range targets {
		updated, err := client.UpdateExtension(ctx, target.Name, UpdateExtensionRequest{
			Version:         strings.TrimSpace(version),
			CheckOnly:       checkOnly,
			AllowUnverified: allowUnverified && !checkOnly,
		})
		if err != nil {
			return nil, err
		}
		items = append(items, daemonExtensionUpdateItem(target, updated, checkOnly))
	}
	return items, nil
}

func selectDaemonMarketplaceExtensionsForUpdate(
	ctx context.Context,
	client DaemonClient,
	args []string,
	updateAll bool,
) ([]ExtensionRecord, error) {
	if updateAll {
		items, err := client.ListExtensions(ctx)
		if err != nil {
			return nil, err
		}
		selected := make([]ExtensionRecord, 0, len(items))
		for _, item := range items {
			if extensionRecordMarketplaceSlug(item) != "" {
				selected = append(selected, item)
			}
		}
		return selected, nil
	}
	name := ""
	if len(args) > 0 {
		name = strings.TrimSpace(args[0])
	}
	if name == "" {
		return nil, errors.New("cli: extension name is required unless --all is set")
	}
	item, err := client.ExtensionStatus(ctx, name)
	if err != nil {
		return nil, err
	}
	if extensionRecordMarketplaceSlug(item) == "" {
		return nil, fmt.Errorf("cli: extension %q is not a marketplace-installed extension", item.Name)
	}
	return []ExtensionRecord{item}, nil
}

func daemonExtensionUpdateItem(
	before ExtensionRecord,
	after ExtensionUpdateRecord,
	checkOnly bool,
) extensionUpdateItem {
	status := after.Status
	if status == "" && checkOnly {
		status = skillUpdateStatusCurrent
	}
	if status == "" {
		status = skillUpdateStatusUpdated
	}
	return extensionUpdateItem{
		Name:           firstNonEmpty(after.Name, before.Name),
		Slug:           firstNonEmpty(after.Slug, extensionRecordMarketplaceSlug(before)),
		Registry:       firstNonEmpty(after.Registry, extensionRecordMarketplaceRegistry(before)),
		CurrentVersion: firstNonEmpty(after.CurrentVersion, before.Version),
		LatestVersion:  firstNonEmpty(after.LatestVersion, before.Version),
		Path:           after.Path,
		Status:         status,
	}
}

func extensionRecordMarketplaceSlug(item ExtensionRecord) string {
	if item.Provenance == nil || item.Provenance.InstalledFrom != extensionpkg.ExtensionInstalledFromMarketplace {
		return ""
	}
	return strings.TrimSpace(item.Provenance.Slug)
}

func extensionRecordMarketplaceRegistry(item ExtensionRecord) string {
	if item.Provenance == nil {
		return ""
	}
	return strings.TrimSpace(item.Provenance.RegistryTier)
}

func extensionSearchBundle(items []ExtensionMarketplaceRecord) outputBundle {
	return listBundle(
		items,
		items,
		"Extension Registry Results",
		[]string{
			extensionMarketplaceSlugValue,
			automationNameValue,
			extensionMarketplaceDescriptionValue,
			"Author",
			daemonVersionValue,
			"Downloads",
			"Source",
		},
		"extensions",
		[]string{
			extensionMarketplaceSlugKey,
			automationNameKey,
			extensionMarketplaceDescriptionKey,
			"author",
			daemonVersionKey,
			"downloads",
			automationSourceKey,
		},
		func(item ExtensionMarketplaceRecord) []string {
			return []string{
				stringOrDash(item.Slug),
				stringOrDash(item.Name),
				stringOrDash(item.Description),
				stringOrDash(item.Author),
				stringOrDash(item.Version),
				strconv.Itoa(item.Downloads),
				stringOrDash(item.Source),
			}
		},
		func(item ExtensionMarketplaceRecord) []string {
			return []string{
				item.Slug,
				item.Name,
				item.Description,
				item.Author,
				item.Version,
				strconv.Itoa(item.Downloads),
				item.Source,
			}
		},
	)
}

func extensionRemoveBundle(item extensionRemoveItem) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanSection("Extension Remove", []keyValue{
				{Label: automationNameValue, Value: stringOrDash(item.Name)},
				{Label: extensionMarketplacePathValue, Value: stringOrDash(item.Path)},
				{Label: extensionMarketplaceStatusValue, Value: stringOrDash(item.Status)},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject(
				"extension_remove",
				[]string{automationNameKey, extensionMarketplacePathKey, automationStatusKey},
				[]string{
					item.Name,
					item.Path,
					item.Status,
				},
			), nil
		},
	}
}

func extensionUpdateBundle(items []extensionUpdateItem) outputBundle {
	return listBundle(
		items,
		items,
		"Extension Updates",
		[]string{
			automationNameValue,
			extensionMarketplaceSlugValue,
			"Registry",
			"Current",
			"Latest",
			extensionMarketplacePathValue,
			extensionMarketplaceStatusValue,
		},
		"extension_updates",
		[]string{
			automationNameKey,
			extensionMarketplaceSlugKey,
			skillOutputRegistryKey,
			extensionMarketplaceCurrentVersionKey,
			"latest_version",
			extensionMarketplacePathKey,
			automationStatusKey,
		},
		func(item extensionUpdateItem) []string {
			return []string{
				stringOrDash(item.Name),
				stringOrDash(item.Slug),
				stringOrDash(item.Registry),
				stringOrDash(item.CurrentVersion),
				stringOrDash(item.LatestVersion),
				stringOrDash(item.Path),
				stringOrDash(item.Status),
			}
		},
		func(item extensionUpdateItem) []string {
			return []string{
				item.Name,
				item.Slug,
				item.Registry,
				item.CurrentVersion,
				item.LatestVersion,
				item.Path,
				item.Status,
			}
		},
	)
}
