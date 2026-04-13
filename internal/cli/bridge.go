package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/spf13/cobra"
)

const bridgeDeliveryDefaultsFlag = "delivery-defaults"

func newBridgeCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bridge",
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
	cmd.AddCommand(newBridgeTestDeliveryCommand(deps))
	return cmd
}

func newBridgeListCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List bridge instances",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			items, err := client.ListBridges(cmd.Context())
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, bridgeListBundle(items, deps.now))
		},
	}
}

func newBridgeGetCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Show one bridge instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
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
		statusRaw        string
		includePeer      bool
		includeThread    bool
		includeGroup     bool
		deliveryDefaults string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a bridge instance",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			scope, err := parseBridgeScope(scopeRaw)
			if err != nil {
				return err
			}
			if scope == bridgepkg.ScopeWorkspace && strings.TrimSpace(workspaceID) == "" {
				return errors.New("cli: --workspace-id is required when --scope=workspace")
			}
			status, err := resolveBridgeStatus(enabled, statusRaw)
			if err != nil {
				return err
			}

			payload := CreateBridgeRequest{
				Scope:         scope,
				WorkspaceID:   strings.TrimSpace(workspaceID),
				Platform:      strings.TrimSpace(platform),
				ExtensionName: strings.TrimSpace(extensionName),
				DisplayName:   strings.TrimSpace(displayName),
				Enabled:       enabled,
				Status:        status,
				RoutingPolicy: bridgepkg.RoutingPolicy{
					IncludePeer:   includePeer,
					IncludeThread: includeThread,
					IncludeGroup:  includeGroup,
				},
			}

			if raw, err := parseOptionalBridgeJSON(deliveryDefaults); err != nil {
				return err
			} else if raw != nil {
				payload.DeliveryDefaults = *raw
			}

			item, err := client.CreateBridge(cmd.Context(), payload)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, bridgeBundle(item))
		},
	}
	cmd.Flags().StringVar(&scopeRaw, "scope", string(bridgepkg.ScopeGlobal), "Bridge scope: global or workspace")
	cmd.Flags().StringVar(&workspaceID, "workspace-id", "", "Owning workspace ID for workspace-scoped bridges")
	cmd.Flags().StringVar(&platform, "platform", "", "Messaging platform name")
	cmd.Flags().StringVar(&extensionName, "extension", "", "Owning extension name")
	cmd.Flags().StringVar(&displayName, "display-name", "", "Operator-facing bridge display name")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "Whether the instance starts enabled")
	cmd.Flags().StringVar(&statusRaw, "status", "", "Lifecycle status (defaults to starting when enabled, disabled otherwise)")
	cmd.Flags().BoolVar(&includePeer, "include-peer", false, "Include peer identity in routing")
	cmd.Flags().BoolVar(&includeThread, "include-thread", false, "Include thread identity in routing")
	cmd.Flags().BoolVar(&includeGroup, "include-group", false, "Include group identity in routing")
	cmd.Flags().StringVar(&deliveryDefaults, bridgeDeliveryDefaultsFlag, "", "JSON object or null for delivery target defaults")
	mustMarkFlagRequired(cmd, "platform")
	mustMarkFlagRequired(cmd, "extension")
	mustMarkFlagRequired(cmd, "display-name")
	return cmd
}

func newBridgeUpdateCommand(deps commandDeps) *cobra.Command {
	var (
		displayName      string
		includePeer      bool
		includeThread    bool
		includeGroup     bool
		deliveryDefaults string
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update mutable bridge fields",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			displayChanged := cmd.Flags().Changed("display-name")
			routingChanged := bridgeRoutingFlagsChanged(cmd)
			deliveryChanged := cmd.Flags().Changed(bridgeDeliveryDefaultsFlag)
			if !displayChanged && !routingChanged && !deliveryChanged {
				return errors.New("cli: at least one update flag is required")
			}

			req := UpdateBridgeRequest{}
			if displayChanged {
				trimmed := strings.TrimSpace(displayName)
				if trimmed == "" {
					return errors.New("cli: --display-name cannot be empty")
				}
				req.DisplayName = &trimmed
			}

			if routingChanged {
				current, err := client.GetBridge(cmd.Context(), args[0])
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
				raw, err := parseRequiredBridgeJSON(strings.TrimSpace(deliveryDefaults))
				if err != nil {
					return err
				}
				req.DeliveryDefaults = raw
			}

			item, err := client.UpdateBridge(cmd.Context(), args[0], req)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, bridgeBundle(item))
		},
	}
	cmd.Flags().StringVar(&displayName, "display-name", "", "New operator-facing bridge display name")
	cmd.Flags().BoolVar(&includePeer, "include-peer", false, "Override whether routing includes peer identity")
	cmd.Flags().BoolVar(&includeThread, "include-thread", false, "Override whether routing includes thread identity")
	cmd.Flags().BoolVar(&includeGroup, "include-group", false, "Override whether routing includes group identity")
	cmd.Flags().StringVar(&deliveryDefaults, bridgeDeliveryDefaultsFlag, "", "JSON object or null for delivery target defaults")
	return cmd
}

func newBridgeEnableCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "enable <id>",
		Short: "Enable a bridge instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
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

func newBridgeRoutesCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "routes <id>",
		Short: "Inspect routes for one bridge instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			routes, err := client.BridgeRoutes(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, bridgeRoutesBundle(routes, deps.now))
		},
	}
}

func newBridgeTestDeliveryCommand(deps commandDeps) *cobra.Command {
	var (
		message  string
		peerID   string
		threadID string
		groupID  string
		modeRaw  string
	)

	cmd := &cobra.Command{
		Use:   "test-delivery <id>",
		Short: "Resolve a typed outbound delivery target for one bridge instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			mode := bridgepkg.DeliveryMode(strings.TrimSpace(modeRaw)).Normalize()
			if mode != "" {
				if err := mode.Validate(); err != nil {
					return err
				}
			}

			item, err := client.TestBridgeDelivery(cmd.Context(), args[0], BridgeTestDeliveryRequest{
				Message: strings.TrimSpace(message),
				Target: BridgeDeliveryTargetInput{
					PeerID:   strings.TrimSpace(peerID),
					ThreadID: strings.TrimSpace(threadID),
					GroupID:  strings.TrimSpace(groupID),
					Mode:     mode,
				},
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, bridgeTestDeliveryBundle(item))
		},
	}
	cmd.Flags().StringVar(&message, "message", "", "Optional dry-run message label")
	cmd.Flags().StringVar(&peerID, "peer-id", "", "Override target peer ID")
	cmd.Flags().StringVar(&threadID, "thread-id", "", "Override target thread ID")
	cmd.Flags().StringVar(&groupID, "group-id", "", "Override target group ID")
	cmd.Flags().StringVar(&modeRaw, "mode", "", "Delivery mode: direct-send or reply")
	return cmd
}

func bridgeListBundle(items []BridgeRecord, now func() time.Time) outputBundle {
	return listBundle(
		items,
		items,
		"Bridges",
		[]string{"ID", "Name", "Platform", "Extension", "Scope", "Workspace", "Status", "Routing", "Updated"},
		"bridges",
		[]string{"id", "display_name", "platform", "extension_name", "scope", "workspace_id", "status", "routing", "updated_at"},
		func(item BridgeRecord) []string {
			return []string{
				stringOrDash(item.ID),
				stringOrDash(item.DisplayName),
				stringOrDash(item.Platform),
				stringOrDash(item.ExtensionName),
				stringOrDash(string(item.Scope)),
				stringOrDash(item.WorkspaceID),
				stringOrDash(string(item.Status)),
				stringOrDash(bridgeRoutingPolicyLabel(item.RoutingPolicy)),
				stringOrDash(formatAge(now, item.UpdatedAt)),
			}
		},
		func(item BridgeRecord) []string {
			return []string{
				item.ID,
				item.DisplayName,
				item.Platform,
				item.ExtensionName,
				string(item.Scope),
				item.WorkspaceID,
				string(item.Status),
				bridgeRoutingPolicyLabel(item.RoutingPolicy),
				formatTime(item.UpdatedAt),
			}
		},
	)
}

func bridgeBundle(item BridgeRecord) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanSection("Bridge", []keyValue{
				{Label: "ID", Value: stringOrDash(item.ID)},
				{Label: "Name", Value: stringOrDash(item.DisplayName)},
				{Label: "Platform", Value: stringOrDash(item.Platform)},
				{Label: "Extension", Value: stringOrDash(item.ExtensionName)},
				{Label: "Scope", Value: stringOrDash(string(item.Scope))},
				{Label: "Workspace", Value: stringOrDash(item.WorkspaceID)},
				{Label: "Enabled", Value: fmt.Sprintf("%t", item.Enabled)},
				{Label: "Status", Value: stringOrDash(string(item.Status))},
				{Label: "Routing", Value: stringOrDash(bridgeRoutingPolicyLabel(item.RoutingPolicy))},
				{Label: "Delivery Defaults", Value: stringOrDash(compactJSON(item.DeliveryDefaults))},
				{Label: "Created", Value: stringOrDash(formatTime(item.CreatedAt))},
				{Label: "Updated", Value: stringOrDash(formatTime(item.UpdatedAt))},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("bridge", []string{
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
				bridgeRoutingPolicyLabel(item.RoutingPolicy),
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

func bridgeRoutesBundle(routes []BridgeRouteRecord, now func() time.Time) outputBundle {
	return listBundle(
		routes,
		routes,
		"Bridge Routes",
		[]string{"Hash", "Scope", "Workspace", "Peer", "Thread", "Group", "Session", "Agent", "Last Active"},
		"bridge_routes",
		[]string{"routing_key_hash", "scope", "workspace_id", "peer_id", "thread_id", "group_id", "session_id", "agent_name", "last_activity_at"},
		func(route BridgeRouteRecord) []string {
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
		func(route BridgeRouteRecord) []string {
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

func bridgeTestDeliveryBundle(item BridgeTestDeliveryRecord) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanBlocks(
				renderHumanSection("Test Delivery", []keyValue{
					{Label: "Status", Value: stringOrDash(item.Status)},
					{Label: "Message", Value: stringOrDash(item.Message)},
				}),
				renderHumanSection("Delivery Target", []keyValue{
					{Label: "Bridge", Value: stringOrDash(item.DeliveryTarget.BridgeInstanceID)},
					{Label: "Peer", Value: stringOrDash(item.DeliveryTarget.PeerID)},
					{Label: "Thread", Value: stringOrDash(item.DeliveryTarget.ThreadID)},
					{Label: "Group", Value: stringOrDash(item.DeliveryTarget.GroupID)},
					{Label: "Mode", Value: stringOrDash(string(item.DeliveryTarget.Mode))},
				}),
			), nil
		},
		toon: func() (string, error) {
			return renderToonObject("test_delivery", []string{
				"status", "message", "bridge_instance_id", "peer_id", "thread_id", "group_id", "mode",
			}, []string{
				item.Status,
				item.Message,
				item.DeliveryTarget.BridgeInstanceID,
				item.DeliveryTarget.PeerID,
				item.DeliveryTarget.ThreadID,
				item.DeliveryTarget.GroupID,
				string(item.DeliveryTarget.Mode),
			}), nil
		},
	}
}

func parseBridgeScope(raw string) (bridgepkg.Scope, error) {
	scope := bridgepkg.Scope(strings.TrimSpace(raw)).Normalize()
	if err := scope.Validate(); err != nil {
		return "", err
	}
	return scope, nil
}

func resolveBridgeStatus(enabled bool, raw string) (bridgepkg.BridgeStatus, error) {
	if strings.TrimSpace(raw) == "" {
		if enabled {
			return bridgepkg.BridgeStatusStarting, nil
		}
		return bridgepkg.BridgeStatusDisabled, nil
	}

	status := bridgepkg.BridgeStatus(strings.TrimSpace(raw)).Normalize()
	if err := status.Validate(); err != nil {
		return "", err
	}
	return status, nil
}

func bridgeRoutingFlagsChanged(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	return cmd.Flags().Changed("include-peer") ||
		cmd.Flags().Changed("include-thread") ||
		cmd.Flags().Changed("include-group")
}

func parseOptionalBridgeJSON(raw string) (*json.RawMessage, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	return parseRequiredBridgeJSON(trimmed)
}

func parseRequiredBridgeJSON(raw string) (*json.RawMessage, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("cli: delivery defaults must be valid JSON; use null to clear")
	}
	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return nil, errors.New("cli: delivery defaults must be valid JSON")
	}
	switch decoded.(type) {
	case nil, map[string]any:
	default:
		return nil, errors.New("cli: delivery defaults must be a JSON object or null")
	}
	value := json.RawMessage(trimmed)
	return &value, nil
}

func bridgeRoutingPolicyLabel(policy bridgepkg.RoutingPolicy) string {
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
