package cli

import (
	"errors"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const (
	bundleModelKey = "model"
)

const (
	bundleAgentValue       = "Agent"
	bundleBundleValue      = "Bundle"
	bundleCreatedValue     = "Created"
	bundleDescriptionValue = "Description"
	bundleEventValue       = "Event"
	bundleKindValue        = "Kind"
	bundleModelValue       = "Model"
	bundleNameValue        = "Name"
	bundleProfileValue     = "Profile"
	bundleUpdatedValue     = "Updated"
	bundleBundleKey        = "bundle"
	bundleExtensionKey     = "extension"
	bundleGlobalKey        = "global"
	bundleHeartbeatKey     = "heartbeat"
	bundleKindKey          = "kind"
	bundleListKey          = "list"
	bundleProfileKey       = "profile"
)

type bundleActivationFlags struct {
	extensionName               string
	bundleName                  string
	profileName                 string
	scope                       string
	workspace                   string
	bindPrimaryChannelAsDefault bool
}

func newBundleCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   bundleBundleKey,
		Short: "Manage extension bundle presets",
	}
	cmd.AddCommand(newBundleCatalogCommand(deps))
	cmd.AddCommand(newBundlePreviewCommand(deps))
	cmd.AddCommand(newBundleActivateCommand(deps))
	cmd.AddCommand(newBundleListCommand(deps))
	cmd.AddCommand(newBundleGetCommand(deps))
	cmd.AddCommand(newBundleUpdateCommand(deps))
	cmd.AddCommand(newBundleDeactivateCommand(deps))
	cmd.AddCommand(newBundleNetworkSettingsCommand(deps))
	return cmd
}

func newBundleCatalogCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "catalog",
		Short: "List available extension bundle presets",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			items, err := client.ListBundleCatalog(cmd.Context())
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, bundleCatalogBundle(items))
		},
	}
}

func newBundlePreviewCommand(deps commandDeps) *cobra.Command {
	flags := bundleActivationFlags{scope: bundleGlobalKey}
	cmd := &cobra.Command{
		Use:   "preview",
		Short: "Preview a bundle activation without writing resources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			request := bundleActivationRequestFromFlags(flags)
			item, err := client.PreviewBundleActivation(cmd.Context(), request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, bundleActivationBundle(item))
		},
	}
	addBundleActivationFlags(cmd, &flags)
	return cmd
}

func newBundleActivateCommand(deps commandDeps) *cobra.Command {
	flags := bundleActivationFlags{scope: bundleGlobalKey}
	cmd := &cobra.Command{
		Use:   "activate",
		Short: "Activate a bundle preset",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			request := bundleActivationRequestFromFlags(flags)
			item, err := client.ActivateBundle(cmd.Context(), request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, bundleActivationBundle(item))
		},
	}
	addBundleActivationFlags(cmd, &flags)
	return cmd
}

func newBundleListCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   bundleListKey,
		Short: "List active bundle presets",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			items, err := client.ListBundleActivations(cmd.Context())
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, bundleActivationListBundle(items))
		},
	}
}

func newBundleGetCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "get <activation-id>",
		Short: "Show one bundle activation",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			item, err := client.GetBundleActivation(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, bundleActivationBundle(item))
		},
	}
}

func newBundleUpdateCommand(deps commandDeps) *cobra.Command {
	var (
		bindPrimaryChannelAsDefault bool
		clearPrimaryChannelDefault  bool
	)
	cmd := &cobra.Command{
		Use:   "update <activation-id>",
		Short: "Update bundle activation overlays",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if bindPrimaryChannelAsDefault == clearPrimaryChannelDefault {
				return errors.New(
					"cli: set either --bind-primary-channel-as-default or --clear-primary-channel-default",
				)
			}
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			item, err := client.UpdateBundleActivation(
				cmd.Context(),
				args[0],
				UpdateBundleActivationRequest{
					BindPrimaryChannelAsDefault: bindPrimaryChannelAsDefault,
				},
			)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, bundleActivationBundle(item))
		},
	}
	cmd.Flags().
		BoolVar(
			&bindPrimaryChannelAsDefault,
			"bind-primary-channel-as-default",
			false,
			"Bind the bundle primary channel as the effective default",
		)
	cmd.Flags().
		BoolVar(&clearPrimaryChannelDefault, "clear-primary-channel-default", false, "Clear the bundle default-channel bind")
	return cmd
}

func newBundleDeactivateCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "deactivate <activation-id>",
		Short: "Deactivate a bundle preset and remove owned resources",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			if err := client.DeactivateBundle(cmd.Context(), args[0]); err != nil {
				return err
			}
			return writeCommandOutput(cmd, outputBundle{
				jsonValue: map[string]string{"deactivated": strings.TrimSpace(args[0])},
				human: func() (string, error) {
					return renderHumanSection("Bundle Deactivated", []keyValue{
						{Label: "Activation", Value: strings.TrimSpace(args[0])},
					}), nil
				},
				toon: func() (string, error) {
					return renderToonObject(
						"bundle_deactivated",
						[]string{"activation_id"},
						[]string{args[0]},
					), nil
				},
			})
		},
	}
}

func newBundleNetworkSettingsCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "network-settings",
		Short: "Show bundle-derived network settings",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			settings, err := client.BundleNetworkSettings(cmd.Context())
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, bundleNetworkSettingsBundle(settings))
		},
	}
}

func addBundleActivationFlags(cmd *cobra.Command, flags *bundleActivationFlags) {
	cmd.Flags().StringVar(&flags.extensionName, bundleExtensionKey, "", "Extension name")
	cmd.Flags().StringVar(&flags.bundleName, bundleBundleKey, "", "Bundle name")
	cmd.Flags().StringVar(&flags.profileName, bundleProfileKey, "", "Bundle profile name")
	cmd.Flags().
		StringVar(&flags.scope, automationScopeKey, bundleGlobalKey, "Activation scope: global or workspace")
	cmd.Flags().
		StringVar(&flags.workspace, workspaceSkillSource, "", "Workspace id, name, or path for workspace scope")
	cmd.Flags().
		BoolVar(
			&flags.bindPrimaryChannelAsDefault,
			"bind-primary-channel-as-default",
			false,
			"Bind the bundle primary channel as the effective default",
		)
	mustMarkFlagRequired(cmd, bundleExtensionKey)
	mustMarkFlagRequired(cmd, bundleBundleKey)
	mustMarkFlagRequired(cmd, bundleProfileKey)
}

func bundleActivationRequestFromFlags(flags bundleActivationFlags) ActivateBundleRequest {
	return ActivateBundleRequest{
		ExtensionName:               strings.TrimSpace(flags.extensionName),
		BundleName:                  strings.TrimSpace(flags.bundleName),
		ProfileName:                 strings.TrimSpace(flags.profileName),
		Scope:                       strings.TrimSpace(flags.scope),
		Workspace:                   strings.TrimSpace(flags.workspace),
		BindPrimaryChannelAsDefault: flags.bindPrimaryChannelAsDefault,
	}
}

func bundleCatalogBundle(items []BundleCatalogRecord) outputBundle {
	return listBundle(
		struct {
			Bundles []BundleCatalogRecord `json:"bundles"`
		}{Bundles: items},
		items,
		"Bundle Catalog",
		[]string{
			bridgeExtensionValue,
			bundleBundleValue,
			"Profiles",
			"Agents",
			"Jobs",
			"Triggers",
			"Bridges",
		},
		"bundles",
		[]string{
			bundleExtensionKey,
			bundleBundleKey,
			"profiles",
			"agents",
			"jobs",
			"triggers",
			"bridges",
		},
		func(item BundleCatalogRecord) []string {
			agents, jobs, triggers, bridges := bundleCatalogCounts(item)
			return []string{
				stringOrDash(item.ExtensionName),
				stringOrDash(item.BundleName),
				strings.Join(bundleProfileNames(item), ","),
				strconv.Itoa(agents),
				strconv.Itoa(jobs),
				strconv.Itoa(triggers),
				strconv.Itoa(bridges),
			}
		},
		func(item BundleCatalogRecord) []string {
			agents, jobs, triggers, bridges := bundleCatalogCounts(item)
			return []string{
				item.ExtensionName,
				item.BundleName,
				strings.Join(bundleProfileNames(item), "|"),
				strconv.Itoa(agents),
				strconv.Itoa(jobs),
				strconv.Itoa(triggers),
				strconv.Itoa(bridges),
			}
		},
	)
}

func bundleActivationListBundle(items []BundleActivationRecord) outputBundle {
	return listBundle(
		struct {
			Activations []BundleActivationRecord `json:"activations"`
		}{Activations: items},
		items,
		"Bundle Activations",
		[]string{
			"ID",
			bridgeExtensionValue,
			bundleBundleValue,
			bundleProfileValue,
			automationScopeValue,
			authoredContextWorkspaceValue,
			"Agents",
			"Inventory",
		},
		"bundle_activations",
		[]string{
			"id",
			bundleExtensionKey,
			bundleBundleKey,
			bundleProfileKey,
			automationScopeKey,
			workspaceSkillSource,
			"agents",
			"inventory",
		},
		func(item BundleActivationRecord) []string {
			return []string{
				stringOrDash(item.ID),
				stringOrDash(item.ExtensionName),
				stringOrDash(item.BundleName),
				stringOrDash(item.ProfileName),
				stringOrDash(item.Scope),
				stringOrDash(item.WorkspaceID),
				strconv.Itoa(len(item.Agents)),
				strconv.Itoa(len(item.Inventory)),
			}
		},
		func(item BundleActivationRecord) []string {
			return []string{
				item.ID,
				item.ExtensionName,
				item.BundleName,
				item.ProfileName,
				item.Scope,
				item.WorkspaceID,
				strconv.Itoa(len(item.Agents)),
				strconv.Itoa(len(item.Inventory)),
			}
		},
	)
}

func bundleActivationBundle(item BundleActivationRecord) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanBlocks(
				renderHumanSection("Bundle Activation", []keyValue{
					{Label: "ID", Value: item.ID},
					{Label: bridgeExtensionValue, Value: item.ExtensionName},
					{Label: bundleBundleValue, Value: item.BundleName},
					{Label: bundleProfileValue, Value: item.ProfileName},
					{Label: automationScopeValue, Value: item.Scope},
					{Label: authoredContextWorkspaceValue, Value: stringOrDash(item.WorkspaceID)},
					{
						Label: "Default Channel Bind",
						Value: formatBool(item.BindPrimaryChannelAsDefault),
					},
					{Label: bundleCreatedValue, Value: formatTime(item.CreatedAt)},
					{Label: bundleUpdatedValue, Value: formatTime(item.UpdatedAt)},
				}),
				bundleChannelsTable(item.Channels),
				bundleAgentsTable(item.Agents),
				bundleJobsTable(item.Jobs),
				bundleTriggersTable(item.Triggers),
				bundleBridgesTable(item.Bridges),
				bundleInventoryTable(item.Inventory),
			), nil
		},
		toon: func() (string, error) {
			return renderHumanBlocks(
				renderToonObject(
					"bundle_activation",
					[]string{
						"id",
						bundleExtensionKey,
						bundleBundleKey,
						bundleProfileKey,
						automationScopeKey,
						workspaceSkillSource,
					},
					[]string{
						item.ID,
						item.ExtensionName,
						item.BundleName,
						item.ProfileName,
						item.Scope,
						item.WorkspaceID,
					},
				),
				renderToonArray(
					"agents",
					[]string{
						automationNameKey,
						cliProviderKey,
						bundleModelKey,
						authoredContextSoulKey,
						bundleHeartbeatKey,
					},
					bundleAgentRows(item.Agents),
				),
				renderToonArray(
					"inventory",
					[]string{bundleKindKey, "id", automationNameKey},
					bundleInventoryRows(item.Inventory),
				),
			), nil
		},
	}
}

func bundleNetworkSettingsBundle(item BundleNetworkSettingsRecord) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanBlocks(
				renderHumanSection("Bundle Network Settings", []keyValue{
					{
						Label: "Configured Default",
						Value: stringOrDash(item.ConfiguredDefaultChannel),
					},
					{Label: "Effective Default", Value: stringOrDash(item.EffectiveDefaultChannel)},
					{Label: "Effective Source", Value: stringOrDash(item.EffectiveDefaultSource)},
				}),
				renderHumanTable(
					"Declared Channels",
					[]string{
						"Activation",
						bridgeExtensionValue,
						bundleBundleValue,
						bundleProfileValue,
						authoredContextWorkspaceValue,
						bundleNameValue,
						"Primary",
					},
					bundleDeclaredChannelRows(item.DeclaredChannels),
				),
			), nil
		},
		toon: func() (string, error) {
			return renderHumanBlocks(
				renderToonObject(
					"bundle_network",
					[]string{"configured_default", "effective_default", "effective_source"},
					[]string{
						item.ConfiguredDefaultChannel,
						item.EffectiveDefaultChannel,
						item.EffectiveDefaultSource,
					},
				),
				renderToonArray(
					"declared_channels",
					[]string{
						"activation",
						bundleExtensionKey,
						bundleBundleKey,
						bundleProfileKey,
						workspaceSkillSource,
						automationNameKey,
						"primary",
					},
					bundleDeclaredChannelRows(item.DeclaredChannels),
				),
			), nil
		},
	}
}

func bundleProfileNames(item BundleCatalogRecord) []string {
	names := make([]string, 0, len(item.Profiles))
	for _, profile := range item.Profiles {
		names = append(names, strings.TrimSpace(profile.Name))
	}
	return names
}

func bundleCatalogCounts(item BundleCatalogRecord) (int, int, int, int) {
	var agents int
	var jobs int
	var triggers int
	var bridges int
	for _, profile := range item.Profiles {
		agents += profile.AgentCount
		jobs += profile.JobCount
		triggers += profile.TriggerCount
		bridges += profile.BridgeCount
	}
	return agents, jobs, triggers, bridges
}

func bundleChannelsTable(items []BundleChannelRecord) string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			stringOrDash(item.Name),
			formatBool(item.Primary),
			stringOrDash(item.Description),
		})
	}
	return renderHumanTable(
		"Channels",
		[]string{bundleNameValue, "Primary", bundleDescriptionValue},
		rows,
	)
}

func bundleAgentsTable(items []BundleAgentRecord) string {
	return renderHumanTable(
		"Agents",
		[]string{bundleNameValue, agentKernelProviderValue, bundleModelValue, "Soul", "Heartbeat"},
		bundleAgentRows(items),
	)
}

func bundleJobsTable(items []BundleJobRecord) string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			stringOrDash(item.Name),
			stringOrDash(item.AgentName),
			formatBool(item.Enabled),
		})
	}
	return renderHumanTable(
		"Jobs",
		[]string{bundleNameValue, bundleAgentValue, extensionEnabledValue},
		rows,
	)
}

func bundleTriggersTable(items []BundleTriggerRecord) string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			stringOrDash(item.Name),
			stringOrDash(item.AgentName),
			stringOrDash(item.Event),
			formatBool(item.Enabled),
		})
	}
	return renderHumanTable(
		"Triggers",
		[]string{bundleNameValue, bundleAgentValue, bundleEventValue, extensionEnabledValue},
		rows,
	)
}

func bundleBridgesTable(items []BundleBridgeRecord) string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			stringOrDash(item.Name),
			stringOrDash(item.ExtensionName),
			stringOrDash(item.Platform),
			stringOrDash(item.DisplayName),
			strconv.Itoa(len(item.SecretSlots)),
		})
	}
	return renderHumanTable(
		"Bridges",
		[]string{
			bundleNameValue,
			bridgeExtensionValue,
			bundlePlatformValue,
			"Display",
			"Secret Slots",
		},
		rows,
	)
}

func bundleInventoryTable(items []BundleInventoryRecord) string {
	return renderHumanTable(
		"Inventory",
		[]string{bundleKindValue, "ID", bundleNameValue},
		bundleInventoryRows(items),
	)
}

func bundleAgentRows(items []BundleAgentRecord) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			stringOrDash(item.Name),
			stringOrDash(item.Provider),
			stringOrDash(item.Model),
			formatBool(item.HasSoul),
			formatBool(item.HasHeartbeat),
		})
	}
	return rows
}

func bundleInventoryRows(items []BundleInventoryRecord) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			stringOrDash(item.ResourceKind),
			stringOrDash(item.ResourceID),
			stringOrDash(item.ResourceName),
		})
	}
	return rows
}

func bundleDeclaredChannelRows(items []DeclaredNetworkChannelRecord) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			stringOrDash(item.ActivationID),
			stringOrDash(item.ExtensionName),
			stringOrDash(item.BundleName),
			stringOrDash(item.ProfileName),
			stringOrDash(item.WorkspaceID),
			stringOrDash(item.Name),
			formatBool(item.Primary),
		})
	}
	return rows
}
