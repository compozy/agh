package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	channelspkg "github.com/pedronauck/agh/internal/channels"
	"github.com/spf13/cobra"
)

const channelDeliveryDefaultsFlag = "delivery-defaults"

func newChannelCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channel",
		Short: "Manage channel instances",
	}

	cmd.AddCommand(newChannelListCommand(deps))
	cmd.AddCommand(newChannelGetCommand(deps))
	cmd.AddCommand(newChannelCreateCommand(deps))
	cmd.AddCommand(newChannelUpdateCommand(deps))
	cmd.AddCommand(newChannelEnableCommand(deps))
	cmd.AddCommand(newChannelDisableCommand(deps))
	cmd.AddCommand(newChannelRestartCommand(deps))
	cmd.AddCommand(newChannelRoutesCommand(deps))
	cmd.AddCommand(newChannelTestDeliveryCommand(deps))
	return cmd
}

func newChannelListCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List channel instances",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			items, err := client.ListChannels(cmd.Context())
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, channelListBundle(items, deps.now))
		},
	}
}

func newChannelGetCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Show one channel instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			item, err := client.GetChannel(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, channelBundle(item))
		},
	}
}

func newChannelCreateCommand(deps commandDeps) *cobra.Command {
	var (
		scopeRaw         string
		workspaceID      string
		platform         string
		extensionName    string
		displayName      string
		enabled          bool
		statusRaw        string
		includePeer      bool
		includeThread    bool
		includeGroup     bool
		deliveryDefaults string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a channel instance",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			scope, err := parseChannelScope(scopeRaw)
			if err != nil {
				return err
			}
			status, err := resolveChannelStatus(enabled, statusRaw)
			if err != nil {
				return err
			}

			payload := CreateChannelRequest{
				Scope:         scope,
				WorkspaceID:   strings.TrimSpace(workspaceID),
				Platform:      strings.TrimSpace(platform),
				ExtensionName: strings.TrimSpace(extensionName),
				DisplayName:   strings.TrimSpace(displayName),
				Enabled:       enabled,
				Status:        status,
				RoutingPolicy: channelspkg.RoutingPolicy{
					IncludePeer:   includePeer,
					IncludeThread: includeThread,
					IncludeGroup:  includeGroup,
				},
			}

			if raw, err := parseOptionalChannelJSON(deliveryDefaults); err != nil {
				return err
			} else if raw != nil {
				payload.DeliveryDefaults = *raw
			}

			item, err := client.CreateChannel(cmd.Context(), payload)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, channelBundle(item))
		},
	}
	cmd.Flags().StringVar(&scopeRaw, "scope", string(channelspkg.ScopeGlobal), "Channel scope: global or workspace")
	cmd.Flags().StringVar(&workspaceID, "workspace-id", "", "Owning workspace ID for workspace-scoped channels")
	cmd.Flags().StringVar(&platform, "platform", "", "Messaging platform name")
	cmd.Flags().StringVar(&extensionName, "extension", "", "Owning extension name")
	cmd.Flags().StringVar(&displayName, "display-name", "", "Operator-facing channel display name")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "Whether the instance starts enabled")
	cmd.Flags().StringVar(&statusRaw, "status", "", "Lifecycle status (defaults to starting when enabled, disabled otherwise)")
	cmd.Flags().BoolVar(&includePeer, "include-peer", false, "Include peer identity in routing")
	cmd.Flags().BoolVar(&includeThread, "include-thread", false, "Include thread identity in routing")
	cmd.Flags().BoolVar(&includeGroup, "include-group", false, "Include group identity in routing")
	cmd.Flags().StringVar(&deliveryDefaults, channelDeliveryDefaultsFlag, "", "JSON object or null for delivery target defaults")
	_ = cmd.MarkFlagRequired("platform")
	_ = cmd.MarkFlagRequired("extension")
	_ = cmd.MarkFlagRequired("display-name")
	return cmd
}

func newChannelUpdateCommand(deps commandDeps) *cobra.Command {
	var (
		displayName      string
		includePeer      bool
		includeThread    bool
		includeGroup     bool
		deliveryDefaults string
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update mutable channel fields",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			displayChanged := cmd.Flags().Changed("display-name")
			routingChanged := channelRoutingFlagsChanged(cmd)
			deliveryChanged := cmd.Flags().Changed(channelDeliveryDefaultsFlag)
			if !displayChanged && !routingChanged && !deliveryChanged {
				return errors.New("cli: at least one update flag is required")
			}

			req := UpdateChannelRequest{}
			if displayChanged {
				trimmed := strings.TrimSpace(displayName)
				if trimmed == "" {
					return errors.New("cli: --display-name cannot be empty")
				}
				req.DisplayName = &trimmed
			}

			if routingChanged {
				current, err := client.GetChannel(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				policy := current.RoutingPolicy
				if cmd.Flags().Changed("include-peer") {
					policy.IncludePeer = includePeer
				}
				if cmd.Flags().Changed("include-thread") {
					policy.IncludeThread = includeThread
				}
				if cmd.Flags().Changed("include-group") {
					policy.IncludeGroup = includeGroup
				}
				req.RoutingPolicy = &policy
			}

			if deliveryChanged {
				raw, err := parseRequiredChannelJSON(strings.TrimSpace(deliveryDefaults))
				if err != nil {
					return err
				}
				req.DeliveryDefaults = raw
			}

			item, err := client.UpdateChannel(cmd.Context(), args[0], req)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, channelBundle(item))
		},
	}
	cmd.Flags().StringVar(&displayName, "display-name", "", "New operator-facing channel display name")
	cmd.Flags().BoolVar(&includePeer, "include-peer", false, "Override whether routing includes peer identity")
	cmd.Flags().BoolVar(&includeThread, "include-thread", false, "Override whether routing includes thread identity")
	cmd.Flags().BoolVar(&includeGroup, "include-group", false, "Override whether routing includes group identity")
	cmd.Flags().StringVar(&deliveryDefaults, channelDeliveryDefaultsFlag, "", "JSON object or null for delivery target defaults")
	return cmd
}

func newChannelEnableCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "enable <id>",
		Short: "Enable a channel instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			item, err := client.EnableChannel(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, channelBundle(item))
		},
	}
}

func newChannelDisableCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "disable <id>",
		Short: "Disable a channel instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			item, err := client.DisableChannel(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, channelBundle(item))
		},
	}
}

func newChannelRestartCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "restart <id>",
		Short: "Restart a channel instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			item, err := client.RestartChannel(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, channelBundle(item))
		},
	}
}

func newChannelRoutesCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "routes <id>",
		Short: "Inspect routes for one channel instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			routes, err := client.ChannelRoutes(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, channelRoutesBundle(routes, deps.now))
		},
	}
}

func newChannelTestDeliveryCommand(deps commandDeps) *cobra.Command {
	var (
		message  string
		peerID   string
		threadID string
		groupID  string
		modeRaw  string
	)

	cmd := &cobra.Command{
		Use:   "test-delivery <id>",
		Short: "Resolve a typed outbound delivery target for one channel instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			mode := channelspkg.DeliveryMode(strings.TrimSpace(modeRaw)).Normalize()
			if mode != "" {
				if err := mode.Validate(); err != nil {
					return err
				}
			}

			item, err := client.TestChannelDelivery(cmd.Context(), args[0], ChannelTestDeliveryRequest{
				Message: strings.TrimSpace(message),
				Target: ChannelDeliveryTargetInput{
					PeerID:   strings.TrimSpace(peerID),
					ThreadID: strings.TrimSpace(threadID),
					GroupID:  strings.TrimSpace(groupID),
					Mode:     mode,
				},
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, channelTestDeliveryBundle(item))
		},
	}
	cmd.Flags().StringVar(&message, "message", "", "Optional dry-run message label")
	cmd.Flags().StringVar(&peerID, "peer-id", "", "Override target peer ID")
	cmd.Flags().StringVar(&threadID, "thread-id", "", "Override target thread ID")
	cmd.Flags().StringVar(&groupID, "group-id", "", "Override target group ID")
	cmd.Flags().StringVar(&modeRaw, "mode", "", "Delivery mode: direct-send or reply")
	return cmd
}

func channelListBundle(items []ChannelRecord, now func() time.Time) outputBundle {
	return listBundle(
		items,
		items,
		"Channels",
		[]string{"ID", "Name", "Platform", "Extension", "Scope", "Workspace", "Status", "Routing", "Updated"},
		"channels",
		[]string{"id", "display_name", "platform", "extension_name", "scope", "workspace_id", "status", "routing", "updated_at"},
		func(item ChannelRecord) []string {
			return []string{
				stringOrDash(item.ID),
				stringOrDash(item.DisplayName),
				stringOrDash(item.Platform),
				stringOrDash(item.ExtensionName),
				stringOrDash(string(item.Scope)),
				stringOrDash(item.WorkspaceID),
				stringOrDash(string(item.Status)),
				stringOrDash(channelRoutingPolicyLabel(item.RoutingPolicy)),
				stringOrDash(formatAge(now, item.UpdatedAt)),
			}
		},
		func(item ChannelRecord) []string {
			return []string{
				item.ID,
				item.DisplayName,
				item.Platform,
				item.ExtensionName,
				string(item.Scope),
				item.WorkspaceID,
				string(item.Status),
				channelRoutingPolicyLabel(item.RoutingPolicy),
				formatTime(item.UpdatedAt),
			}
		},
	)
}

func channelBundle(item ChannelRecord) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanSection("Channel", []keyValue{
				{Label: "ID", Value: stringOrDash(item.ID)},
				{Label: "Name", Value: stringOrDash(item.DisplayName)},
				{Label: "Platform", Value: stringOrDash(item.Platform)},
				{Label: "Extension", Value: stringOrDash(item.ExtensionName)},
				{Label: "Scope", Value: stringOrDash(string(item.Scope))},
				{Label: "Workspace", Value: stringOrDash(item.WorkspaceID)},
				{Label: "Enabled", Value: fmt.Sprintf("%t", item.Enabled)},
				{Label: "Status", Value: stringOrDash(string(item.Status))},
				{Label: "Routing", Value: stringOrDash(channelRoutingPolicyLabel(item.RoutingPolicy))},
				{Label: "Delivery Defaults", Value: stringOrDash(compactJSON(item.DeliveryDefaults))},
				{Label: "Created", Value: stringOrDash(formatTime(item.CreatedAt))},
				{Label: "Updated", Value: stringOrDash(formatTime(item.UpdatedAt))},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("channel", []string{
				"id", "display_name", "platform", "extension_name", "scope", "workspace_id", "enabled", "status", "routing", "include_peer", "include_thread", "include_group", "delivery_defaults", "created_at", "updated_at",
			}, []string{
				item.ID,
				item.DisplayName,
				item.Platform,
				item.ExtensionName,
				string(item.Scope),
				item.WorkspaceID,
				fmt.Sprintf("%t", item.Enabled),
				string(item.Status),
				channelRoutingPolicyLabel(item.RoutingPolicy),
				fmt.Sprintf("%t", item.RoutingPolicy.IncludePeer),
				fmt.Sprintf("%t", item.RoutingPolicy.IncludeThread),
				fmt.Sprintf("%t", item.RoutingPolicy.IncludeGroup),
				compactJSON(item.DeliveryDefaults),
				formatTime(item.CreatedAt),
				formatTime(item.UpdatedAt),
			}), nil
		},
	}
}

func channelRoutesBundle(routes []ChannelRouteRecord, now func() time.Time) outputBundle {
	return listBundle(
		routes,
		routes,
		"Channel Routes",
		[]string{"Hash", "Scope", "Workspace", "Peer", "Thread", "Group", "Session", "Agent", "Last Active"},
		"channel_routes",
		[]string{"routing_key_hash", "scope", "workspace_id", "peer_id", "thread_id", "group_id", "session_id", "agent_name", "last_activity_at"},
		func(route ChannelRouteRecord) []string {
			return []string{
				stringOrDash(route.RoutingKeyHash),
				stringOrDash(string(route.Scope)),
				stringOrDash(route.WorkspaceID),
				stringOrDash(route.PeerID),
				stringOrDash(route.ThreadID),
				stringOrDash(route.GroupID),
				stringOrDash(route.SessionID),
				stringOrDash(route.AgentName),
				stringOrDash(formatAge(now, route.LastActivityAt)),
			}
		},
		func(route ChannelRouteRecord) []string {
			return []string{
				route.RoutingKeyHash,
				string(route.Scope),
				route.WorkspaceID,
				route.PeerID,
				route.ThreadID,
				route.GroupID,
				route.SessionID,
				route.AgentName,
				formatTime(route.LastActivityAt),
			}
		},
	)
}

func channelTestDeliveryBundle(item ChannelTestDeliveryRecord) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanBlocks(
				renderHumanSection("Test Delivery", []keyValue{
					{Label: "Status", Value: stringOrDash(item.Status)},
					{Label: "Message", Value: stringOrDash(item.Message)},
				}),
				renderHumanSection("Delivery Target", []keyValue{
					{Label: "Channel", Value: stringOrDash(item.DeliveryTarget.ChannelInstanceID)},
					{Label: "Peer", Value: stringOrDash(item.DeliveryTarget.PeerID)},
					{Label: "Thread", Value: stringOrDash(item.DeliveryTarget.ThreadID)},
					{Label: "Group", Value: stringOrDash(item.DeliveryTarget.GroupID)},
					{Label: "Mode", Value: stringOrDash(string(item.DeliveryTarget.Mode))},
				}),
			), nil
		},
		toon: func() (string, error) {
			return renderToonObject("test_delivery", []string{
				"status", "message", "channel_instance_id", "peer_id", "thread_id", "group_id", "mode",
			}, []string{
				item.Status,
				item.Message,
				item.DeliveryTarget.ChannelInstanceID,
				item.DeliveryTarget.PeerID,
				item.DeliveryTarget.ThreadID,
				item.DeliveryTarget.GroupID,
				string(item.DeliveryTarget.Mode),
			}), nil
		},
	}
}

func parseChannelScope(raw string) (channelspkg.Scope, error) {
	scope := channelspkg.Scope(strings.TrimSpace(raw)).Normalize()
	if err := scope.Validate(); err != nil {
		return "", err
	}
	return scope, nil
}

func resolveChannelStatus(enabled bool, raw string) (channelspkg.ChannelStatus, error) {
	if strings.TrimSpace(raw) == "" {
		if enabled {
			return channelspkg.ChannelStatusStarting, nil
		}
		return channelspkg.ChannelStatusDisabled, nil
	}

	status := channelspkg.ChannelStatus(strings.TrimSpace(raw)).Normalize()
	if err := status.Validate(); err != nil {
		return "", err
	}
	return status, nil
}

func channelRoutingFlagsChanged(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	return cmd.Flags().Changed("include-peer") ||
		cmd.Flags().Changed("include-thread") ||
		cmd.Flags().Changed("include-group")
}

func parseOptionalChannelJSON(raw string) (*json.RawMessage, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	return parseRequiredChannelJSON(trimmed)
}

func parseRequiredChannelJSON(raw string) (*json.RawMessage, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("cli: delivery defaults must be valid JSON; use null to clear")
	}
	if !json.Valid([]byte(trimmed)) {
		return nil, errors.New("cli: delivery defaults must be valid JSON")
	}
	value := json.RawMessage(trimmed)
	return &value, nil
}

func channelRoutingPolicyLabel(policy channelspkg.RoutingPolicy) string {
	dimensions := make([]string, 0, 3)
	if policy.IncludePeer {
		dimensions = append(dimensions, "peer")
	}
	if policy.IncludeThread {
		dimensions = append(dimensions, "thread")
	}
	if policy.IncludeGroup {
		dimensions = append(dimensions, "group")
	}
	return strings.Join(dimensions, ", ")
}
