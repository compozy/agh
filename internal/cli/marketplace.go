package cli

import (
	"strconv"
	"strings"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/spf13/cobra"
)

const (
	marketplaceDefaultLimit = 20
	marketplaceInstalledKey = "installed"
	marketplaceScopeKey     = "scope"
	outputStaleKey          = "stale"
	outputStaleValue        = "Stale"
)

func newMarketplaceCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   marketplaceSkillSource,
		Short: "Discover installable AGH capabilities",
	}
	cmd.AddCommand(newMarketplaceSearchCommand(deps))
	cmd.AddCommand(newMarketplaceInfoCommand(deps))
	cmd.AddCommand(newMarketplaceRefreshCommand(deps))
	return cmd
}

func newMarketplaceSearchCommand(deps commandDeps) *cobra.Command {
	var kind string
	var readScope marketplaceReadFlagValues
	limit := marketplaceDefaultLimit
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search or browse the marketplace",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scope, err := readScope.value()
			if err != nil {
				return err
			}
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			query := ""
			if len(args) > 0 {
				query = args[0]
			}
			if strings.TrimSpace(kind) == "" {
				response, err := client.SearchMarketplace(cmd.Context(), query, limit, scope)
				if err != nil {
					return err
				}
				return writeCommandOutput(cmd, marketplaceGroupedBundle(response))
			}
			response, err := client.BrowseMarketplace(cmd.Context(), kind, query, limit, scope)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, marketplaceListingsBundle(response, response.Items))
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "", "Limit results to mcp, extension, skill, or bundle")
	cmd.Flags().IntVar(&limit, "limit", marketplaceDefaultLimit, "Maximum results per marketplace kind")
	addMarketplaceReadFlags(cmd, &readScope)
	return cmd
}

func newMarketplaceInfoCommand(deps commandDeps) *cobra.Command {
	var readScope marketplaceReadFlagValues
	cmd := &cobra.Command{
		Use:   "info <kind> <entry_id>",
		Short: "Show one marketplace entry",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			scope, err := readScope.value()
			if err != nil {
				return err
			}
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			response, err := client.MarketplaceInfo(cmd.Context(), args[0], args[1], scope)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, marketplaceEntryBundle(response))
		},
	}
	addMarketplaceReadFlags(cmd, &readScope)
	return cmd
}

type marketplaceReadFlagValues struct {
	scope       string
	workspaceID string
}

func (v marketplaceReadFlagValues) value() (MarketplaceReadScope, error) {
	result := MarketplaceReadScope{
		Scope:       contract.SettingsWorkspaceScopeKind(strings.TrimSpace(v.scope)),
		WorkspaceID: strings.TrimSpace(v.workspaceID),
	}
	if _, err := result.queryValues(); err != nil {
		return MarketplaceReadScope{}, err
	}
	return result, nil
}

func addMarketplaceReadFlags(cmd *cobra.Command, values *marketplaceReadFlagValues) {
	cmd.Flags().StringVar(
		&values.scope,
		marketplaceScopeKey,
		string(contract.SettingsWorkspaceScopeGlobal),
		"Installed-state scope: global or workspace",
	)
	cmd.Flags().StringVar(&values.workspaceID, "workspace", "", "Workspace ID for workspace scope")
}

func newMarketplaceRefreshCommand(deps commandDeps) *cobra.Command {
	var kind string
	cmd := &cobra.Command{
		Use:   "refresh",
		Short: "Refresh curated marketplace catalogs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			response, err := client.RefreshMarketplace(cmd.Context(), kind)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, marketplaceRefreshBundle(response))
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "", "Refresh only mcp, extension, or skill")
	return cmd
}

func marketplaceGroupedBundle(response MarketplaceSearchRecord) outputBundle {
	items := make([]MarketplaceListingRecord, 0)
	for _, result := range response.Kinds {
		items = append(items, result.Items...)
	}
	bundle := marketplaceListingsBundle(response, items)
	bundle.jsonl = func(cmd *cobra.Command) error {
		return writeJSONLines(cmd, response.Kinds)
	}
	return bundle
}

func marketplaceListingsBundle(jsonValue any, items []MarketplaceListingRecord) outputBundle {
	return listBundle(
		jsonValue,
		items,
		"Marketplace Results",
		[]string{
			bundleKindValue,
			"Entry ID",
			automationNameValue,
			daemonVersionValue,
			"Installed",
			authoredContextSourceValue,
		},
		marketplaceSkillSource,
		[]string{
			networkKindKey,
			"entry_id",
			automationNameKey,
			daemonVersionKey,
			marketplaceInstalledKey,
			automationSourceKey,
		},
		func(item MarketplaceListingRecord) []string {
			return []string{
				item.Kind,
				item.EntryID,
				item.Name,
				stringOrDash(item.Version),
				strconv.FormatBool(item.Installed),
				item.Source,
			}
		},
		func(item MarketplaceListingRecord) []string {
			return []string{
				item.Kind,
				item.EntryID,
				item.Name,
				item.Version,
				strconv.FormatBool(item.Installed),
				item.Source,
			}
		},
	)
}

func marketplaceEntryBundle(response MarketplaceEntryRecord) outputBundle {
	entry := response.Entry
	return outputBundle{
		jsonValue: response,
		jsonl: func(cmd *cobra.Command) error {
			return writeJSONLine(cmd, response)
		},
		human: func() (string, error) {
			return renderHumanSection("Marketplace Entry", []keyValue{
				{Label: bundleKindValue, Value: entry.Kind},
				{Label: "Entry ID", Value: entry.EntryID},
				{Label: automationNameValue, Value: entry.Name},
				{Label: "Description", Value: entry.Description},
				{Label: "Version", Value: stringOrDash(entry.Version)},
				{Label: authoredContextSourceValue, Value: entry.Source},
				{Label: "Installed", Value: strconv.FormatBool(entry.Installed)},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject(
				"marketplace_entry",
				[]string{
					networkKindKey,
					"entry_id",
					automationNameKey,
					extensionMarketplaceDescriptionKey,
					daemonVersionKey,
					automationSourceKey,
					marketplaceInstalledKey,
				},
				[]string{
					entry.Kind,
					entry.EntryID,
					entry.Name,
					entry.Description,
					entry.Version,
					entry.Source,
					strconv.FormatBool(entry.Installed),
				},
			), nil
		},
	}
}

func marketplaceRefreshBundle(response MarketplaceRefreshRecord) outputBundle {
	return listBundle(
		response,
		response.Kinds,
		"Marketplace Refresh",
		[]string{bundleKindValue, "Outcome", "Entries", outputStaleValue, "Error Class"},
		"marketplace_refresh",
		[]string{networkKindKey, "outcome", "entry_count", outputStaleKey, "error_class"},
		func(item contract.MarketplaceRefreshKindPayload) []string {
			return []string{
				item.Kind,
				item.Outcome,
				strconv.Itoa(item.EntryCount),
				strconv.FormatBool(item.Stale),
				stringOrDash(item.ErrorClass),
			}
		},
		func(item contract.MarketplaceRefreshKindPayload) []string {
			return []string{
				item.Kind,
				item.Outcome,
				strconv.Itoa(item.EntryCount),
				strconv.FormatBool(item.Stale),
				item.ErrorClass,
			}
		},
	)
}
