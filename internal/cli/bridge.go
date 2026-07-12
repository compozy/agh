package cli

import (
	"errors"
	"strings"

	"github.com/compozy/agh/internal/api/contract"
	bridgepkg "github.com/compozy/agh/internal/bridges"
	"github.com/spf13/cobra"
)

const (
	taskThreadValue = "Thread"
)

const (
	taskPeerValue       = "Peer"
	bundlePlatformValue = "Platform"
)

const (
	taskGroupValue          = "Group"
	taskBridgeInstanceIDKey = "bridge_instance_id"
)

const (
	bridgeModeKey = "mode"
)

const (
	bridgeAgentValue        = "Agent"
	bridgeBridgeValue       = "Bridge"
	bridgeCreatedValue      = "Created"
	bridgeEnabledValue      = "Enabled"
	bridgeExtensionValue    = "Extension"
	bridgeMessageValue      = "Message"
	bridgeModeValue         = "Mode"
	bridgeAgentNameKey      = "agent_name"
	bridgeBindingNameKey    = "binding_name"
	bridgeBridgeKey         = "bridge"
	bridgeCreateKey         = "create"
	bridgeCreatedAtKey      = "created_at"
	bridgeDeletedKey        = "deleted"
	bridgeDisplayNameKey    = "display_name"
	bridgeEnabledKey        = "enabled"
	bridgeGetIDValue        = "get <id>"
	bridgeGroupIDKey        = "group_id"
	bridgeLastActivityAtKey = "last_activity_at"
	bridgeListKey           = "list"
	bridgeMessageKey        = "message"
	bridgePeerIDKey         = "peer_id"
	bridgePlatformKey       = "platform"
	bridgeResolvedValue     = "resolved"
	bridgeScopeKey          = "scope"
	bridgeStepValue         = "Step"
	bridgeSessionIDKey      = "session_id"
	bridgeStatusKey         = "status"
	bridgeThreadIDKey       = "thread_id"
	bridgeUnresolvedValue   = "unresolved"
	bridgeUpdateIDValue     = "update <id>"
	bridgeUpdatedAtKey      = "updated_at"
	bridgeWorkspaceIDKey    = "workspace_id"
)

const (
	bridgeDeliveryDefaultsFlag = "delivery-defaults"
	bridgeProviderConfigFlag   = "provider-config"
)

func newBridgeCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   bridgeBridgeKey,
		Short: "Manage bridge instances",
	}

	cmd.AddCommand(newBridgeListCommand(deps))
	cmd.AddCommand(newBridgeGetCommand(deps))
	cmd.AddCommand(newBridgeCreateCommand(deps))
	cmd.AddCommand(newBridgeUpdateCommand(deps))
	cmd.AddCommand(newBridgeEnableCommand(deps))
	cmd.AddCommand(newBridgeDisableCommand(deps))
	cmd.AddCommand(newBridgeRestartCommand(deps))
	cmd.AddCommand(newBridgeRoutesCommand(deps))
	cmd.AddCommand(newBridgeTargetsCommand(deps))
	cmd.AddCommand(newBridgeResolveCommand(deps))
	cmd.AddCommand(newBridgeSecretBindingsCommand(deps))
	cmd.AddCommand(newBridgeTestDeliveryCommand(deps))
	return cmd
}

func newBridgeGetCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   bridgeGetIDValue,
		Short: "Show one bridge instance",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			item, err := client.GetBridge(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, bridgeBundle(item))
		},
	}
}

func newBridgeCreateCommand(deps commandDeps) *cobra.Command {
	var (
		scopeRaw         string
		workspaceID      string
		platform         string
		extensionName    string
		displayName      string
		enabled          bool
		includePeer      bool
		includeThread    bool
		includeGroup     bool
		notificationMute bool
		providerConfig   string
		deliveryDefaults string
	)

	cmd := &cobra.Command{
		Use:   bridgeCreateKey,
		Short: "Create a bridge instance",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			payload, err := buildBridgeCreatePayload(
				scopeRaw,
				workspaceID,
				platform,
				extensionName,
				displayName,
				enabled,
				includePeer,
				includeThread,
				includeGroup,
				notificationMute,
				providerConfig,
				deliveryDefaults,
			)
			if err != nil {
				return err
			}

			item, err := client.CreateBridge(cmd.Context(), payload)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, bridgeBundle(item))
		},
	}
	cmd.Flags().
		StringVar(&scopeRaw, bridgeScopeKey, string(bridgepkg.ScopeGlobal), "Bridge scope: global or workspace")
	cmd.Flags().
		StringVar(&workspaceID, "workspace-id", "", "Owning workspace ID for workspace-scoped bridges")
	cmd.Flags().StringVar(&platform, "platform", "", "Messaging platform name")
	cmd.Flags().StringVar(&extensionName, "extension", "", "Owning extension name")
	cmd.Flags().StringVar(&displayName, "display-name", "", "Operator-facing bridge display name")
	cmd.Flags().BoolVar(&enabled, bridgeEnabledKey, true, "Whether the instance starts enabled")
	cmd.Flags().BoolVar(&includePeer, "include-peer", false, "Include peer identity in routing")
	cmd.Flags().
		BoolVar(&includeThread, "include-thread", false, "Include thread identity in routing")
	cmd.Flags().BoolVar(&includeGroup, "include-group", false, "Include group identity in routing")
	cmd.Flags().BoolVar(
		&notificationMute,
		"notification-suppress",
		false,
		"Suppress notification deliveries to this bridge",
	)
	cmd.Flags().
		StringVar(&providerConfig, bridgeProviderConfigFlag, "", "JSON object or null for provider runtime config")
	cmd.Flags().
		StringVar(&deliveryDefaults, bridgeDeliveryDefaultsFlag, "", "JSON object or null for delivery target defaults")
	mustMarkFlagRequired(cmd, "platform")
	mustMarkFlagRequired(cmd, "extension")
	mustMarkFlagRequired(cmd, "display-name")
	return cmd
}

func buildBridgeCreatePayload(
	scopeRaw string,
	workspaceID string,
	platform string,
	extensionName string,
	displayName string,
	enabled bool,
	includePeer bool,
	includeThread bool,
	includeGroup bool,
	notificationSuppress bool,
	providerConfig string,
	deliveryDefaults string,
) (CreateBridgeRequest, error) {
	scope, err := parseBridgeScope(scopeRaw)
	if err != nil {
		return CreateBridgeRequest{}, err
	}
	if scope == bridgepkg.ScopeWorkspace && strings.TrimSpace(workspaceID) == "" {
		return CreateBridgeRequest{}, errors.New(
			"cli: --workspace-id is required when --scope=workspace",
		)
	}

	payload := CreateBridgeRequest{
		Scope:         scope,
		WorkspaceID:   strings.TrimSpace(workspaceID),
		Platform:      strings.TrimSpace(platform),
		ExtensionName: strings.TrimSpace(extensionName),
		DisplayName:   strings.TrimSpace(displayName),
		Enabled:       enabled,
		RoutingPolicy: bridgepkg.RoutingPolicy{
			IncludePeer:   includePeer,
			IncludeThread: includeThread,
			IncludeGroup:  includeGroup,
		},
		NotificationSuppress: notificationSuppress,
	}

	providerRaw, err := parseOptionalBridgeJSONWithLabel(providerConfig, "provider config")
	if err != nil {
		return CreateBridgeRequest{}, err
	}
	if providerRaw != nil {
		payload.ProviderConfig = contract.BridgeProviderConfigPayload(*providerRaw)
	}
	raw, err := parseOptionalBridgeJSON(deliveryDefaults)
	if err != nil {
		return CreateBridgeRequest{}, err
	}
	if raw != nil {
		payload.DeliveryDefaults = contract.BridgeDeliveryDefaultsPayload(*raw)
	}
	if err := validateBridgeCreatePayload(payload); err != nil {
		return CreateBridgeRequest{}, err
	}
	return payload, nil
}

func newBridgeUpdateCommand(deps commandDeps) *cobra.Command {
	flags := bridgeUpdateFlags{}

	cmd := &cobra.Command{
		Use:   bridgeUpdateIDValue,
		Short: "Update mutable bridge fields",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBridgeUpdateCommand(cmd, deps, args[0], flags)
		},
	}
	cmd.Flags().
		StringVar(&flags.displayName, "display-name", "", "New operator-facing bridge display name")
	cmd.Flags().
		BoolVar(&flags.includePeer, "include-peer", false, "Override whether routing includes peer identity")
	cmd.Flags().
		BoolVar(&flags.includeThread, "include-thread", false, "Override whether routing includes thread identity")
	cmd.Flags().
		BoolVar(&flags.includeGroup, "include-group", false, "Override whether routing includes group identity")
	cmd.Flags().BoolVar(
		&flags.notificationSuppress,
		"notification-suppress",
		false,
		"Override whether notification deliveries are suppressed",
	)
	cmd.Flags().
		StringVar(&flags.providerConfig, bridgeProviderConfigFlag, "", "JSON object or null for provider runtime config")
	cmd.Flags().
		StringVar(&flags.deliveryDefaults, bridgeDeliveryDefaultsFlag, "", "JSON object or null for delivery target defaults")
	return cmd
}

type bridgeUpdateFlags struct {
	displayName          string
	includePeer          bool
	includeThread        bool
	includeGroup         bool
	notificationSuppress bool
	providerConfig       string
	deliveryDefaults     string
}

func runBridgeUpdateCommand(
	cmd *cobra.Command,
	deps commandDeps,
	id string,
	flags bridgeUpdateFlags,
) error {
	client, err := clientFromDeps(deps)
	if err != nil {
		return err
	}
	req, err := buildBridgeUpdateRequest(cmd, client, id, flags)
	if err != nil {
		return err
	}
	item, err := client.UpdateBridge(cmd.Context(), id, req)
	if err != nil {
		return err
	}
	return writeCommandOutput(cmd, bridgeBundle(item))
}

func buildBridgeUpdateRequest(
	cmd *cobra.Command,
	client DaemonClient,
	id string,
	flags bridgeUpdateFlags,
) (UpdateBridgeRequest, error) {
	displayChanged := cmd.Flags().Changed("display-name")
	routingChanged := bridgeRoutingFlagsChanged(cmd)
	deliveryChanged := cmd.Flags().Changed(bridgeDeliveryDefaultsFlag)
	providerChanged := cmd.Flags().Changed(bridgeProviderConfigFlag)
	notificationChanged := cmd.Flags().Changed("notification-suppress")
	if !displayChanged && !routingChanged && !deliveryChanged && !providerChanged && !notificationChanged {
		return UpdateBridgeRequest{}, errors.New("cli: at least one update flag is required")
	}

	req := UpdateBridgeRequest{}
	if displayChanged {
		trimmed := strings.TrimSpace(flags.displayName)
		if trimmed == "" {
			return UpdateBridgeRequest{}, errors.New("cli: --display-name cannot be empty")
		}
		req.DisplayName = &trimmed
	}
	if routingChanged {
		policy, err := bridgeRoutingPolicyForUpdate(cmd, client, id, flags)
		if err != nil {
			return UpdateBridgeRequest{}, err
		}
		req.RoutingPolicy = &policy
	}
	if deliveryChanged {
		value, err := bridgeDeliveryDefaultsForUpdate(flags.deliveryDefaults)
		if err != nil {
			return UpdateBridgeRequest{}, err
		}
		req.DeliveryDefaults = &value
	}
	if providerChanged {
		value, err := bridgeProviderConfigForUpdate(flags.providerConfig)
		if err != nil {
			return UpdateBridgeRequest{}, err
		}
		req.ProviderConfig = &value
	}
	if notificationChanged {
		req.NotificationSuppress = &flags.notificationSuppress
	}
	return req, nil
}

func bridgeRoutingPolicyForUpdate(
	cmd *cobra.Command,
	client DaemonClient,
	id string,
	flags bridgeUpdateFlags,
) (bridgepkg.RoutingPolicy, error) {
	current, err := client.GetBridge(cmd.Context(), id)
	if err != nil {
		return bridgepkg.RoutingPolicy{}, err
	}
	policy := current.RoutingPolicy
	if cmd.Flags().Changed("include-peer") {
		policy.IncludePeer = flags.includePeer
	}
	if cmd.Flags().Changed("include-thread") {
		policy.IncludeThread = flags.includeThread
	}
	if cmd.Flags().Changed("include-group") {
		policy.IncludeGroup = flags.includeGroup
	}
	return policy, nil
}

func bridgeDeliveryDefaultsForUpdate(
	rawValue string,
) (contract.BridgeDeliveryDefaultsPayload, error) {
	raw, err := parseRequiredBridgeJSON(strings.TrimSpace(rawValue))
	if err != nil {
		return nil, err
	}
	return contract.BridgeDeliveryDefaultsPayload(*raw), nil
}

func bridgeProviderConfigForUpdate(
	rawValue string,
) (contract.BridgeProviderConfigPayload, error) {
	raw, err := parseRequiredBridgeJSONWithLabel(strings.TrimSpace(rawValue), "provider config")
	if err != nil {
		return nil, err
	}
	return contract.BridgeProviderConfigPayload(*raw), nil
}

func newBridgeEnableCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "enable <id>",
		Short: "Enable a bridge instance",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			item, err := client.EnableBridge(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, bridgeBundle(item))
		},
	}
}

func newBridgeDisableCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "disable <id>",
		Short: "Disable a bridge instance",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			item, err := client.DisableBridge(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, bridgeBundle(item))
		},
	}
}

func newBridgeRestartCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "restart <id>",
		Short: "Restart a bridge instance",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			item, err := client.RestartBridge(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, bridgeBundle(item))
		},
	}
}
