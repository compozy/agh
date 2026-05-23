package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/spf13/cobra"
)

const (
	toolOperatorPreviewValue = "Preview"
	networkLocalKey          = "local"
)

const (
	networkOpenedByValue = "Opened By"
)

const (
	networkOpenWorkValue = "Open Work"
	networkSentKey       = "sent"
)

const (
	networkChannelValue          = "Channel"
	networkCreatedByValue        = "Created By"
	networkDirectIDValue         = "Direct ID"
	networkEnabledValue          = "Enabled"
	networkFromValue             = "From"
	networkKindValue             = "Kind"
	networkLastActivityValue     = "Last Activity"
	networkMessageValue          = "Message"
	networkMessagesValue         = "Messages"
	networkOpenedAtValue         = "Opened At"
	networkPresenceValue         = "Presence"
	networkStateValue            = "State"
	networkStatusValue           = "Status"
	networkSurfaceValue          = "Surface"
	networkThreadIDValue         = "Thread ID"
	networkTimestampValue        = "Timestamp"
	networkTitleValue            = "Title"
	networkWorkspaceValue        = "Workspace"
	networkChannelKey            = "channel"
	networkChannelsKey           = "channels"
	networkDirectIDKey           = "direct_id"
	networkEnabledKey            = "enabled"
	networkKindKey               = "kind"
	networkLastActivityAtKey     = "last_activity_at"
	networkLastMessagePreviewKey = "last_message_preview"
	networkListKey               = "list"
	networkMessageCountKey       = "message_count"
	networkMessageIDKey          = "message_id"
	networkNetworkKey            = "network"
	networkOpenWorkCountKey      = "open_work_count"
	networkOpenedAtKey           = "opened_at"
	networkOpenedByPeerIDKey     = "opened_by_peer_id"
	networkSendKey               = "send"
	networkShowKey               = "show"
	networkStatusKey             = "status"
	networkSurfaceKey            = "surface"
	networkThreadIDKey           = "thread_id"
	networkTitleKey              = "title"
	networkWorkIDKey             = "work_id"
)

const (
	networkSurfaceThread  = "thread"
	networkSurfaceDirect  = "direct"
	networkKindSay        = "say"
	networkKindCapability = "capability"
	networkKindReceipt    = "receipt"
	networkKindTrace      = "trace"
	networkKindGreet      = "greet"
	networkKindWhois      = "whois"
)

func newNetworkCommand(deps commandDeps) *cobra.Command {
	var workspaceRef string
	cmd := &cobra.Command{
		Use:   networkNetworkKey,
		Short: "Operate the daemon-owned network runtime",
	}

	cmd.AddCommand(newNetworkStatusCommand(deps))
	cmd.PersistentFlags().
		StringVar(&workspaceRef, "workspace", "", "Workspace root, name, or ID for scoped network data")
	cmd.AddCommand(newNetworkPeersCommand(deps, &workspaceRef))
	cmd.AddCommand(newNetworkChannelsCommand(deps, &workspaceRef))
	cmd.AddCommand(newNetworkThreadsCommand(deps, &workspaceRef))
	cmd.AddCommand(newNetworkDirectsCommand(deps, &workspaceRef))
	cmd.AddCommand(newNetworkWorkCommand(deps, &workspaceRef))
	cmd.AddCommand(newNetworkSendCommand(deps, &workspaceRef))
	cmd.AddCommand(newNetworkInboxCommand(deps, &workspaceRef))
	return cmd
}

func resolveNetworkWorkspaceRef(
	cmd *cobra.Command,
	deps commandDeps,
	client DaemonClient,
	workspaceRef *string,
) (string, error) {
	raw := ""
	if workspaceRef != nil {
		raw = strings.TrimSpace(*workspaceRef)
	}
	return resolveCLIWorkspaceRouteRef(cmd.Context(), deps, client, raw)
}

func newNetworkStatusCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   networkStatusKey,
		Short: "Show network runtime status and queue metrics",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			status, err := client.NetworkStatus(cmd.Context())
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, networkStatusBundle(status))
		},
	}
}

func newNetworkPeersCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	return &cobra.Command{
		Use:   "peers [channel]",
		Short: "List visible local and remote peers with derived presence",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := resolveNetworkWorkspaceRef(cmd, deps, client, workspaceRef)
			if err != nil {
				return err
			}

			query := NetworkPeersQuery{WorkspaceRef: workspace}
			if len(args) == 1 {
				query.Channel = strings.TrimSpace(args[0])
			}

			peers, err := client.NetworkPeers(cmd.Context(), query)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, networkPeersBundle(peers))
		},
	}
}

func newNetworkChannelsCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   networkChannelsKey,
		Short: "List active runtime channels",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := resolveNetworkWorkspaceRef(cmd, deps, client, workspaceRef)
			if err != nil {
				return err
			}

			channels, err := client.NetworkChannels(cmd.Context(), workspace)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, networkChannelsBundle(channels))
		},
	}
	cmd.AddCommand(newNetworkChannelsCreateCommand(deps, workspaceRef))
	return cmd
}

type networkCreateChannelFlags struct {
	channel    string
	purpose    string
	agentNames []string
}

func newNetworkChannelsCreateCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	var flags networkCreateChannelFlags
	cmd := &cobra.Command{
		Use:   "create [channel]",
		Short: "Create a runtime channel and start one session per selected agent",
		Args:  cobra.MaximumNArgs(1),
		Example: `  # Create a launch channel with two local agents
  agh network --workspace ~/dev/ad8 channels create ad8_launch \
    --purpose "Coordinate launch work" \
    --agent site_copywriter \
    --agent growth_marketer \
    -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := resolveNetworkWorkspaceRef(cmd, deps, client, workspaceRef)
			if err != nil {
				return err
			}
			channel, err := resolveNetworkCreateChannelName(args, flags.channel)
			if err != nil {
				return err
			}
			agentNames := trimSpawnAtoms(flags.agentNames)
			if len(agentNames) == 0 {
				return errors.New("cli: at least one --agent is required")
			}
			purpose := strings.TrimSpace(flags.purpose)
			if purpose == "" {
				return errors.New("cli: --purpose cannot be empty")
			}
			created, err := client.CreateNetworkChannel(cmd.Context(), workspace, CreateNetworkChannelRequest{
				Channel:    channel,
				Purpose:    purpose,
				AgentNames: agentNames,
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, networkChannelBundle(created))
		},
	}
	cmd.Flags().StringVar(&flags.channel, networkChannelKey, "", "Channel name when not passed as an argument")
	cmd.Flags().StringVar(&flags.purpose, "purpose", "", "Human-readable channel purpose")
	cmd.Flags().StringArrayVar(
		&flags.agentNames,
		"agent",
		nil,
		"Agent definition name to launch in the channel (repeatable)",
	)
	mustMarkFlagRequired(cmd, "purpose")
	mustMarkFlagRequired(cmd, "agent")
	return cmd
}

func resolveNetworkCreateChannelName(args []string, channelFlag string) (string, error) {
	fromArg := ""
	if len(args) == 1 {
		fromArg = strings.TrimSpace(args[0])
	}
	fromFlag := strings.TrimSpace(channelFlag)
	switch {
	case fromArg == "" && fromFlag == "":
		return "", errors.New("cli: channel is required")
	case fromArg != "" && fromFlag != "" && fromArg != fromFlag:
		return "", errors.New("cli: channel argument and --channel must match")
	case fromArg != "":
		return fromArg, nil
	default:
		return fromFlag, nil
	}
}

type networkThreadsFlags struct {
	channel  string
	threadID string
	limit    int
	before   string
	after    string
	kind     string
	workID   string
}

func newNetworkThreadsCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "threads",
		Short: "Inspect public network threads",
	}
	cmd.AddCommand(newNetworkThreadsListCommand(deps, workspaceRef))
	cmd.AddCommand(newNetworkThreadsShowCommand(deps, workspaceRef))
	cmd.AddCommand(newNetworkThreadsMessagesCommand(deps, workspaceRef))
	return cmd
}

func newNetworkThreadsListCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	var flags networkThreadsFlags
	cmd := &cobra.Command{
		Use:   networkListKey,
		Short: "List public threads in a channel",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := resolveNetworkWorkspaceRef(cmd, deps, client, workspaceRef)
			if err != nil {
				return err
			}
			threads, err := client.NetworkThreads(cmd.Context(), NetworkThreadsQuery{
				WorkspaceRef: workspace,
				Channel:      strings.TrimSpace(flags.channel),
				Limit:        flags.limit,
				After:        strings.TrimSpace(flags.after),
			})
			if err != nil {
				return err
			}
			return writeCommandOutputWithJSONL(cmd, networkThreadsBundle(threads), threads)
		},
	}
	cmd.Flags().StringVar(&flags.channel, networkChannelKey, "", "Target channel")
	cmd.Flags().IntVar(&flags.limit, "limit", 0, "Maximum number of threads to return")
	cmd.Flags().StringVar(&flags.after, "after", "", "Cursor after which to list threads")
	mustMarkFlagRequired(cmd, networkChannelKey)
	return cmd
}

func newNetworkThreadsShowCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	var flags networkThreadsFlags
	cmd := &cobra.Command{
		Use:   networkShowKey,
		Short: "Show one public thread",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := resolveNetworkWorkspaceRef(cmd, deps, client, workspaceRef)
			if err != nil {
				return err
			}
			thread, err := client.NetworkThread(
				cmd.Context(),
				workspace,
				strings.TrimSpace(flags.channel),
				strings.TrimSpace(flags.threadID),
			)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, networkThreadBundle(thread))
		},
	}
	cmd.Flags().StringVar(&flags.channel, networkChannelKey, "", "Target channel")
	cmd.Flags().StringVar(&flags.threadID, networkSurfaceThread, "", "Public thread id")
	mustMarkFlagRequired(cmd, networkChannelKey)
	mustMarkFlagRequired(cmd, networkSurfaceThread)
	return cmd
}

func newNetworkThreadsMessagesCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	var flags networkThreadsFlags
	cmd := &cobra.Command{
		Use:   sessionMessagesKey,
		Short: "List messages in one public thread",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := resolveNetworkWorkspaceRef(cmd, deps, client, workspaceRef)
			if err != nil {
				return err
			}
			messages, err := client.NetworkThreadMessages(cmd.Context(), NetworkConversationMessagesQuery{
				WorkspaceRef: workspace,
				Channel:      strings.TrimSpace(flags.channel),
				ThreadID:     strings.TrimSpace(flags.threadID),
				Limit:        flags.limit,
				Before:       strings.TrimSpace(flags.before),
				After:        strings.TrimSpace(flags.after),
				Kind:         strings.TrimSpace(flags.kind),
				WorkID:       strings.TrimSpace(flags.workID),
			})
			if err != nil {
				return err
			}
			return writeCommandOutputWithJSONL(cmd, networkThreadMessagesBundle(messages), messages)
		},
	}
	registerNetworkMessageReadFlags(
		cmd,
		&flags.channel,
		networkSurfaceThread,
		"Public thread id",
		&flags.threadID,
		&flags.limit,
		&flags.before,
		&flags.after,
		&flags.kind,
		&flags.workID,
	)
	cmd.Flags().Lookup(networkSurfaceThread).Usage = "Public thread id"
	mustMarkFlagRequired(cmd, networkChannelKey)
	mustMarkFlagRequired(cmd, networkSurfaceThread)
	return cmd
}

type networkDirectsFlags struct {
	channel  string
	directID string
	peerID   string
	session  string
	limit    int
	before   string
	after    string
	kind     string
	workID   string
}

func newNetworkDirectsCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "directs",
		Short: "Inspect restricted direct rooms",
	}
	cmd.AddCommand(newNetworkDirectsListCommand(deps, workspaceRef))
	cmd.AddCommand(newNetworkDirectsResolveCommand(deps, workspaceRef))
	cmd.AddCommand(newNetworkDirectsShowCommand(deps, workspaceRef))
	cmd.AddCommand(newNetworkDirectsMessagesCommand(deps, workspaceRef))
	return cmd
}

func newNetworkDirectsListCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	var flags networkDirectsFlags
	cmd := &cobra.Command{
		Use:   networkListKey,
		Short: "List direct rooms in a channel",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := resolveNetworkWorkspaceRef(cmd, deps, client, workspaceRef)
			if err != nil {
				return err
			}
			directs, err := client.NetworkDirects(cmd.Context(), NetworkDirectsQuery{
				WorkspaceRef: workspace,
				Channel:      strings.TrimSpace(flags.channel),
				PeerID:       strings.TrimSpace(flags.peerID),
				Limit:        flags.limit,
				After:        strings.TrimSpace(flags.after),
			})
			if err != nil {
				return err
			}
			return writeCommandOutputWithJSONL(cmd, networkDirectsBundle(directs), directs)
		},
	}
	cmd.Flags().StringVar(&flags.channel, networkChannelKey, "", "Target channel")
	cmd.Flags().StringVar(&flags.peerID, "peer", "", "Peer id filter")
	cmd.Flags().IntVar(&flags.limit, "limit", 0, "Maximum number of direct rooms to return")
	cmd.Flags().StringVar(&flags.after, "after", "", "Cursor after which to list direct rooms")
	mustMarkFlagRequired(cmd, networkChannelKey)
	return cmd
}

func newNetworkDirectsResolveCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	var flags networkDirectsFlags
	cmd := &cobra.Command{
		Use:   "resolve",
		Short: "Create or return the deterministic direct room for two peers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := resolveNetworkWorkspaceRef(cmd, deps, client, workspaceRef)
			if err != nil {
				return err
			}
			direct, err := client.NetworkDirectResolve(
				cmd.Context(),
				workspace,
				strings.TrimSpace(flags.channel),
				NetworkDirectResolveRequest{
					SessionID: strings.TrimSpace(flags.session),
					PeerID:    strings.TrimSpace(flags.peerID),
				},
			)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, networkDirectBundle(direct))
		},
	}
	cmd.Flags().StringVar(&flags.session, "session", "", "Local source session id")
	cmd.Flags().StringVar(&flags.channel, networkChannelKey, "", "Target channel")
	cmd.Flags().StringVar(&flags.peerID, "peer", "", "Remote peer id")
	mustMarkFlagRequired(cmd, "session")
	mustMarkFlagRequired(cmd, networkChannelKey)
	mustMarkFlagRequired(cmd, "peer")
	return cmd
}

func newNetworkDirectsShowCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	var flags networkDirectsFlags
	cmd := &cobra.Command{
		Use:   networkShowKey,
		Short: "Show one direct room",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := resolveNetworkWorkspaceRef(cmd, deps, client, workspaceRef)
			if err != nil {
				return err
			}
			direct, err := client.NetworkDirect(
				cmd.Context(),
				workspace,
				strings.TrimSpace(flags.channel),
				strings.TrimSpace(flags.directID),
			)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, networkDirectBundle(direct))
		},
	}
	cmd.Flags().StringVar(&flags.channel, networkChannelKey, "", "Target channel")
	cmd.Flags().StringVar(&flags.directID, networkSurfaceDirect, "", "Direct room id")
	mustMarkFlagRequired(cmd, networkChannelKey)
	mustMarkFlagRequired(cmd, networkSurfaceDirect)
	return cmd
}

func newNetworkDirectsMessagesCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	var flags networkDirectsFlags
	cmd := &cobra.Command{
		Use:   sessionMessagesKey,
		Short: "List messages in one direct room",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := resolveNetworkWorkspaceRef(cmd, deps, client, workspaceRef)
			if err != nil {
				return err
			}
			messages, err := client.NetworkDirectMessages(cmd.Context(), NetworkConversationMessagesQuery{
				WorkspaceRef: workspace,
				Channel:      strings.TrimSpace(flags.channel),
				DirectID:     strings.TrimSpace(flags.directID),
				Limit:        flags.limit,
				Before:       strings.TrimSpace(flags.before),
				After:        strings.TrimSpace(flags.after),
				Kind:         strings.TrimSpace(flags.kind),
				WorkID:       strings.TrimSpace(flags.workID),
			})
			if err != nil {
				return err
			}
			return writeCommandOutputWithJSONL(cmd, networkDirectMessagesBundle(messages), messages)
		},
	}
	registerNetworkMessageReadFlags(
		cmd,
		&flags.channel,
		networkSurfaceDirect,
		"Direct room id",
		&flags.directID,
		&flags.limit,
		&flags.before,
		&flags.after,
		&flags.kind,
		&flags.workID,
	)
	mustMarkFlagRequired(cmd, networkChannelKey)
	mustMarkFlagRequired(cmd, networkSurfaceDirect)
	return cmd
}

func registerNetworkMessageReadFlags(
	cmd *cobra.Command,
	channel *string,
	containerFlagName string,
	containerUsage string,
	containerID *string,
	limit *int,
	before *string,
	after *string,
	kind *string,
	workID *string,
) {
	cmd.Flags().StringVar(channel, networkChannelKey, "", "Target channel")
	cmd.Flags().StringVar(containerID, containerFlagName, "", containerUsage)
	cmd.Flags().IntVar(limit, "limit", 0, "Maximum number of messages to return")
	cmd.Flags().StringVar(before, "before", "", "Cursor before which to list messages")
	cmd.Flags().StringVar(after, "after", "", "Cursor after which to list messages")
	cmd.Flags().StringVar(kind, networkKindKey, "", "Envelope kind filter")
	cmd.Flags().StringVar(workID, "work", "", "Work id filter")
}

func newNetworkWorkCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "work",
		Short: "Inspect lifecycle-bearing network work",
	}
	cmd.AddCommand(newNetworkWorkLookupCommand(deps, workspaceRef, "lookup"))
	cmd.AddCommand(newNetworkWorkLookupCommand(deps, workspaceRef, networkStatusKey))
	return cmd
}

func newNetworkWorkLookupCommand(deps commandDeps, workspaceRef *string, use string) *cobra.Command {
	var workID string
	cmd := &cobra.Command{
		Use:   use,
		Short: "Show one network work item",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := resolveNetworkWorkspaceRef(cmd, deps, client, workspaceRef)
			if err != nil {
				return err
			}
			work, err := client.NetworkWork(cmd.Context(), workspace, strings.TrimSpace(workID))
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, networkWorkBundle(work))
		},
	}
	cmd.Flags().StringVar(&workID, "work", "", "Network work id")
	mustMarkFlagRequired(cmd, "work")
	return cmd
}

type networkSendFlags struct {
	sessionID    string
	channel      string
	surface      string
	threadID     string
	directID     string
	kind         string
	to           string
	bodyRaw      string
	workID       string
	replyTo      string
	traceID      string
	causationID  string
	expiresAtRaw string
	id           string
	extRaw       string
}

func newNetworkSendCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	var flags networkSendFlags
	cmd := &cobra.Command{
		Use:   networkSendKey,
		Short: "Send one envelope through the daemon-owned network runtime",
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, err := parseNetworkJSONValue("--body", flags.bodyRaw)
			if err != nil {
				return err
			}
			ext, err := parseNetworkJSONObjectMap("--ext", flags.extRaw)
			if err != nil {
				return err
			}
			if err := validateNetworkSendNoRawClaimToken(body, ext); err != nil {
				return err
			}
			expiresAt, err := parseNetworkExpiresAt(flags.expiresAtRaw)
			if err != nil {
				return err
			}
			if err := validateNetworkSendFlags(flags); err != nil {
				return err
			}
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := resolveNetworkWorkspaceRef(cmd, deps, client, workspaceRef)
			if err != nil {
				return err
			}

			message, err := client.NetworkSend(cmd.Context(), NetworkSendRequest{
				WorkspaceID: workspace,
				SessionID:   strings.TrimSpace(flags.sessionID),
				Channel:     strings.TrimSpace(flags.channel),
				Surface:     strings.TrimSpace(flags.surface),
				ThreadID:    strings.TrimSpace(flags.threadID),
				DirectID:    strings.TrimSpace(flags.directID),
				Kind:        strings.TrimSpace(flags.kind),
				To:          strings.TrimSpace(flags.to),
				Body:        body,
				WorkID:      strings.TrimSpace(flags.workID),
				ReplyTo:     strings.TrimSpace(flags.replyTo),
				TraceID:     strings.TrimSpace(flags.traceID),
				CausationID: strings.TrimSpace(flags.causationID),
				ExpiresAt:   expiresAt,
				ID:          strings.TrimSpace(flags.id),
				Ext:         ext,
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, networkSendBundle(message))
		},
	}

	registerNetworkSendFlags(cmd, &flags)
	mustMarkFlagRequired(cmd, "session")
	mustMarkFlagRequired(cmd, networkChannelKey)
	mustMarkFlagRequired(cmd, networkKindKey)
	mustMarkFlagRequired(cmd, "body")
	return cmd
}

func registerNetworkSendFlags(cmd *cobra.Command, flags *networkSendFlags) {
	cmd.Flags().StringVar(&flags.sessionID, "session", "", "Local source session id")
	cmd.Flags().StringVar(&flags.channel, networkChannelKey, "", "Target channel")
	cmd.Flags().StringVar(&flags.surface, networkSurfaceKey, "", "Conversation surface: thread or direct")
	cmd.Flags().StringVar(&flags.threadID, networkSurfaceThread, "", "Thread id for thread-surface messages")
	cmd.Flags().StringVar(&flags.directID, networkSurfaceDirect, "", "Direct room id for direct-surface messages")
	cmd.Flags().StringVar(&flags.kind, networkKindKey, "", "Envelope kind")
	cmd.Flags().StringVar(&flags.to, "to", "", "Directed target peer id")
	cmd.Flags().StringVar(&flags.bodyRaw, "body", "", "Raw JSON object for the envelope body")
	cmd.Flags().StringVar(&flags.workID, "work", "", "Optional work id")
	cmd.Flags().StringVar(&flags.replyTo, "reply-to", "", "Optional reply-to message id")
	cmd.Flags().StringVar(&flags.traceID, "trace-id", "", "Optional trace id")
	cmd.Flags().StringVar(&flags.causationID, "causation-id", "", "Optional causation id")
	cmd.Flags().StringVar(&flags.expiresAtRaw, "expires-at", "", "Optional expiry as unix seconds or RFC3339")
	cmd.Flags().StringVar(&flags.id, "id", "", "Optional explicit message id")
	cmd.Flags().StringVar(&flags.extRaw, "ext", "", "Optional JSON object of extension metadata")
}

func newNetworkInboxCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	var sessionID string

	cmd := &cobra.Command{
		Use:   "inbox",
		Short: "Show queued inbound messages for one session",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := resolveNetworkWorkspaceRef(cmd, deps, client, workspaceRef)
			if err != nil {
				return err
			}

			messages, err := client.NetworkInbox(cmd.Context(), workspace, strings.TrimSpace(sessionID))
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, networkInboxBundle(messages))
		},
	}

	cmd.Flags().StringVar(&sessionID, "session", "", "Target session id")
	mustMarkFlagRequired(cmd, "session")
	return cmd
}

func networkStatusBundle(status NetworkStatusRecord) outputBundle {
	rows := []keyValue{
		{Label: networkEnabledValue, Value: strconv.FormatBool(status.Enabled)},
		{Label: networkStatusValue, Value: stringOrDash(status.Status)},
		{Label: "Listener", Value: stringOrDash(networkListener(&status))},
		{Label: "Local Peers", Value: strconv.Itoa(status.LocalPeers)},
		{Label: "Remote Peers", Value: strconv.Itoa(status.RemotePeers)},
		{Label: "Channels", Value: strconv.Itoa(status.Channels)},
		{Label: "Queued Messages", Value: strconv.Itoa(status.QueuedMessages)},
		{Label: "Queued Sessions", Value: strconv.Itoa(status.QueuedSessions)},
		{Label: "Delivery Workers", Value: strconv.Itoa(status.DeliveryWorkers)},
		{Label: "Messages Sent", Value: strconv.FormatInt(status.MessagesSent, 10)},
		{Label: "Messages Received", Value: strconv.FormatInt(status.MessagesReceived, 10)},
		{Label: "Messages Rejected", Value: strconv.FormatInt(status.MessagesRejected, 10)},
		{Label: "Messages Delivered", Value: strconv.FormatInt(status.MessagesDelivered, 10)},
		{Label: "Workflow Tagged", Value: strconv.FormatInt(status.WorkflowTaggedEvents, 10)},
		{Label: "Handoff Tagged", Value: strconv.FormatInt(status.HandoffTaggedEvents, 10)},
		{Label: "Last Disconnect", Value: stringOrDash(status.LastDisconnect)},
	}
	fields := []string{
		networkEnabledKey, networkStatusKey, "listener", "local_peers", "remote_peers", networkChannelsKey,
		"queued_messages", "queued_sessions", "delivery_workers", "messages_sent",
		"messages_received", "messages_rejected", "messages_delivered",
		"workflow_tagged_events", "handoff_tagged_events", "last_disconnect",
	}
	values := []string{
		strconv.FormatBool(status.Enabled),
		status.Status,
		networkListener(&status),
		strconv.Itoa(status.LocalPeers),
		strconv.Itoa(status.RemotePeers),
		strconv.Itoa(status.Channels),
		strconv.Itoa(status.QueuedMessages),
		strconv.Itoa(status.QueuedSessions),
		strconv.Itoa(status.DeliveryWorkers),
		strconv.FormatInt(status.MessagesSent, 10),
		strconv.FormatInt(status.MessagesReceived, 10),
		strconv.FormatInt(status.MessagesRejected, 10),
		strconv.FormatInt(status.MessagesDelivered, 10),
		strconv.FormatInt(status.WorkflowTaggedEvents, 10),
		strconv.FormatInt(status.HandoffTaggedEvents, 10),
		status.LastDisconnect,
	}

	return outputBundle{
		jsonValue: status,
		human: func() (string, error) {
			return renderHumanBlocks(
				renderHumanSection("Network", rows),
				renderHumanTable(
					"Kind Metrics",
					[]string{networkKindValue, "Sent", "Received", "Rejected", "Delivered"},
					networkKindMetricRows(status.KindMetrics),
				),
			), nil
		},
		toon: func() (string, error) {
			return renderHumanBlocks(
				renderToonObject(networkNetworkKey, fields, values),
				renderToonArray(
					"network_kind_metrics",
					[]string{networkKindKey, networkSentKey, "received", "rejected", "delivered"},
					networkKindMetricRows(status.KindMetrics),
				),
			), nil
		},
	}
}

func networkPeersBundle(peers []NetworkPeerRecord) outputBundle {
	return listBundle(
		peers,
		peers,
		"Network Peers",
		[]string{
			taskPeerValue,
			"Display",
			agentKernelSessionValue,
			networkChannelValue,
			networkPresenceValue,
			"Local",
			"Joined",
			"Last Seen",
			"Expires",
		},
		"network_peers",
		[]string{
			"peer_id",
			"display_name",
			"session_id",
			networkChannelKey,
			"presence_state",
			networkLocalKey,
			"joined_at",
			"last_seen",
			mcpAuthExpiresAtKey,
		},
		func(peer NetworkPeerRecord) []string {
			return []string{
				stringOrDash(peer.PeerID),
				stringOrDash(optionalString(peer.PeerCard.DisplayName)),
				stringOrDash(optionalString(peer.SessionID)),
				stringOrDash(peer.Channel),
				stringOrDash(networkPeerPresenceLabel(peer)),
				strconv.FormatBool(peer.Local),
				stringOrDash(formatTimePtr(peer.JoinedAt)),
				stringOrDash(formatTimePtr(peer.LastSeen)),
				stringOrDash(formatTimePtr(peer.ExpiresAt)),
			}
		},
		func(peer NetworkPeerRecord) []string {
			return []string{
				peer.PeerID,
				optionalString(peer.PeerCard.DisplayName),
				optionalString(peer.SessionID),
				peer.Channel,
				peer.PresenceState,
				strconv.FormatBool(peer.Local),
				formatTimePtr(peer.JoinedAt),
				formatTimePtr(peer.LastSeen),
				formatTimePtr(peer.ExpiresAt),
			}
		},
	)
}

func networkPeerPresenceLabel(peer NetworkPeerRecord) string {
	state := strings.TrimSpace(peer.PresenceState)
	if state == "" {
		return ""
	}
	if peer.LastSeenAgeSeconds == nil {
		return state
	}
	return fmt.Sprintf("%s %ds ago", state, *peer.LastSeenAgeSeconds)
}

func networkChannelsBundle(channels []NetworkChannelRecord) outputBundle {
	return listBundle(
		channels,
		channels,
		"Network Channels",
		[]string{networkChannelValue, "Peers"},
		"network_channels",
		[]string{networkChannelKey, "peer_count"},
		func(channel NetworkChannelRecord) []string {
			return []string{
				stringOrDash(channel.Channel),
				strconv.Itoa(channel.PeerCount),
			}
		},
		func(channel NetworkChannelRecord) []string {
			return []string{
				channel.Channel,
				strconv.Itoa(channel.PeerCount),
			}
		},
	)
}

func networkChannelBundle(channel NetworkChannelDetailRecord) outputBundle {
	fields := []string{
		networkChannelKey,
		automationWorkspaceIDKey,
		"purpose",
		"created_by",
		"peer_count",
		"session_count",
		networkMessageCountKey,
	}
	values := []string{
		channel.Channel,
		channel.WorkspaceID,
		channel.Purpose,
		channel.CreatedBy,
		strconv.Itoa(channel.PeerCount),
		strconv.Itoa(channel.SessionCount),
		strconv.Itoa(channel.MessageCount),
	}
	return outputBundle{
		jsonValue: channel,
		human: func() (string, error) {
			return renderHumanBlocks(
				renderHumanSection("Network Channel", []keyValue{
					{Label: networkChannelValue, Value: stringOrDash(channel.Channel)},
					{Label: networkWorkspaceValue, Value: stringOrDash(channel.WorkspaceID)},
					{Label: "Purpose", Value: stringOrDash(channel.Purpose)},
					{Label: networkCreatedByValue, Value: stringOrDash(channel.CreatedBy)},
					{Label: "Peers", Value: strconv.Itoa(channel.PeerCount)},
					{Label: "Sessions", Value: strconv.Itoa(channel.SessionCount)},
					{Label: networkMessagesValue, Value: strconv.Itoa(channel.MessageCount)},
				}),
			), nil
		},
		toon: func() (string, error) {
			return renderToonObject("network_channel", fields, values), nil
		},
	}
}

func networkThreadsBundle(threads []NetworkThreadRecord) outputBundle {
	return listBundle(
		contract.NetworkThreadsResponse{Threads: threads},
		threads,
		"Network Threads",
		[]string{
			taskThreadValue,
			"Root",
			networkOpenedByValue,
			networkMessagesValue,
			"Participants",
			networkOpenWorkValue,
			networkLastActivityValue,
			toolOperatorPreviewValue,
		},
		"network_threads",
		[]string{
			networkChannelKey,
			networkThreadIDKey,
			"root_message_id",
			networkOpenedByPeerIDKey,
			networkMessageCountKey,
			"participant_count",
			networkOpenWorkCountKey,
			networkLastActivityAtKey,
			networkLastMessagePreviewKey,
		},
		networkThreadHumanRow,
		networkThreadToonRow,
	)
}

func networkThreadBundle(thread NetworkThreadRecord) outputBundle {
	return outputBundle{
		jsonValue: contract.NetworkThreadResponse{Thread: thread},
		human: func() (string, error) {
			return renderHumanSection("Network Thread", networkThreadKeyValues(thread)), nil
		},
		toon: func() (string, error) {
			return renderToonObject(
				"network_thread",
				[]string{
					networkChannelKey,
					networkThreadIDKey,
					"root_message_id",
					networkTitleKey,
					networkOpenedByPeerIDKey,
					"opened_session_id",
					networkOpenedAtKey,
					networkLastActivityAtKey,
					networkMessageCountKey,
					"participant_count",
					networkOpenWorkCountKey,
					networkLastMessagePreviewKey,
				},
				[]string{
					thread.Channel,
					thread.ThreadID,
					thread.RootMessageID,
					thread.Title,
					thread.OpenedByPeerID,
					thread.OpenedSessionID,
					formatTimePtr(thread.OpenedAt),
					formatTimePtr(thread.LastActivityAt),
					strconv.Itoa(thread.MessageCount),
					strconv.Itoa(thread.ParticipantCount),
					strconv.Itoa(thread.OpenWorkCount),
					thread.LastMessagePreview,
				},
			), nil
		},
	}
}

func networkThreadMessagesBundle(messages []NetworkConversationMessageRecord) outputBundle {
	return networkMessagesBundle(contract.NetworkThreadMessagesResponse{Messages: messages}, messages)
}

func networkDirectsBundle(directs []NetworkDirectRoomRecord) outputBundle {
	return listBundle(
		contract.NetworkDirectRoomsResponse{Directs: directs},
		directs,
		"Network Direct Rooms",
		[]string{
			"Direct",
			"Peer A",
			"Peer B",
			networkMessagesValue,
			networkOpenWorkValue,
			networkLastActivityValue,
			toolOperatorPreviewValue,
		},
		"network_directs",
		[]string{
			networkChannelKey,
			networkDirectIDKey,
			"peer_a",
			"peer_b",
			networkMessageCountKey,
			networkOpenWorkCountKey,
			networkLastActivityAtKey,
			networkLastMessagePreviewKey,
		},
		networkDirectHumanRow,
		networkDirectToonRow,
	)
}

func networkDirectBundle(direct NetworkDirectRoomRecord) outputBundle {
	return outputBundle{
		jsonValue: contract.NetworkDirectRoomResponse{Direct: direct},
		human: func() (string, error) {
			return renderHumanSection("Network Direct Room", networkDirectKeyValues(direct)), nil
		},
		toon: func() (string, error) {
			return renderToonObject(
				"network_direct",
				[]string{
					networkChannelKey,
					networkDirectIDKey,
					"peer_a",
					"peer_b",
					networkOpenedAtKey,
					networkLastActivityAtKey,
					networkMessageCountKey,
					networkOpenWorkCountKey,
					networkLastMessagePreviewKey,
				},
				[]string{
					direct.Channel,
					direct.DirectID,
					direct.PeerA,
					direct.PeerB,
					formatTimePtr(direct.OpenedAt),
					formatTimePtr(direct.LastActivityAt),
					strconv.Itoa(direct.MessageCount),
					strconv.Itoa(direct.OpenWorkCount),
					direct.LastMessagePreview,
				},
			), nil
		},
	}
}

func networkDirectMessagesBundle(messages []NetworkConversationMessageRecord) outputBundle {
	return networkMessagesBundle(contract.NetworkDirectRoomMessagesResponse{Messages: messages}, messages)
}

func networkMessagesBundle(jsonValue any, messages []NetworkConversationMessageRecord) outputBundle {
	return listBundle(
		jsonValue,
		messages,
		"Network Messages",
		[]string{
			networkMessageValue,
			networkSurfaceValue,
			taskThreadValue,
			"Direct",
			networkKindValue,
			"Direction",
			networkFromValue,
			"To",
			"Work",
			networkTimestampValue,
			toolOperatorPreviewValue,
		},
		"network_messages",
		[]string{
			networkMessageIDKey,
			networkChannelKey,
			networkSurfaceKey,
			networkThreadIDKey,
			networkDirectIDKey,
			networkKindKey,
			"direction",
			"peer_from",
			"peer_to",
			networkWorkIDKey,
			networkTimestampKey,
			"preview_text",
		},
		networkMessageHumanRow,
		networkMessageToonRow,
	)
}

func networkWorkBundle(work NetworkWorkRecord) outputBundle {
	return outputBundle{
		jsonValue: contract.NetworkWorkResponse{Work: work},
		human: func() (string, error) {
			return renderHumanSection("Network Work", []keyValue{
				{Label: "Work ID", Value: stringOrDash(work.WorkID)},
				{Label: networkChannelValue, Value: stringOrDash(work.Channel)},
				{Label: networkSurfaceValue, Value: stringOrDash(work.Surface)},
				{Label: networkThreadIDValue, Value: stringOrDash(work.ThreadID)},
				{Label: networkDirectIDValue, Value: stringOrDash(work.DirectID)},
				{Label: networkOpenedByValue, Value: stringOrDash(work.OpenedByPeerID)},
				{Label: "Opened Session", Value: stringOrDash(work.OpenedSessionID)},
				{Label: "Target Peer", Value: stringOrDash(work.TargetPeerID)},
				{Label: networkStateValue, Value: stringOrDash(work.State)},
				{Label: networkOpenedAtValue, Value: stringOrDash(formatTimePtr(work.OpenedAt))},
				{Label: networkLastActivityValue, Value: stringOrDash(formatTimePtr(work.LastActivityAt))},
				{Label: "Terminal At", Value: stringOrDash(formatTimePtr(work.TerminalAt))},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject(
				"network_work",
				[]string{
					networkWorkIDKey,
					networkChannelKey,
					networkSurfaceKey,
					networkThreadIDKey,
					networkDirectIDKey,
					networkOpenedByPeerIDKey,
					"opened_session_id",
					"target_peer_id",
					networkStateKey,
					networkOpenedAtKey,
					networkLastActivityAtKey,
					"terminal_at",
				},
				[]string{
					work.WorkID,
					work.Channel,
					work.Surface,
					work.ThreadID,
					work.DirectID,
					work.OpenedByPeerID,
					work.OpenedSessionID,
					work.TargetPeerID,
					work.State,
					formatTimePtr(work.OpenedAt),
					formatTimePtr(work.LastActivityAt),
					formatTimePtr(work.TerminalAt),
				},
			), nil
		},
	}
}

func networkSendBundle(message NetworkSendRecord) outputBundle {
	return outputBundle{
		jsonValue: message,
		human: func() (string, error) {
			return renderHumanSection("Network Message", []keyValue{
				{Label: "ID", Value: stringOrDash(message.ID)},
				{Label: agentKernelSessionValue, Value: stringOrDash(message.SessionID)},
				{Label: networkChannelValue, Value: stringOrDash(message.Channel)},
				{Label: networkSurfaceValue, Value: stringOrDash(message.Surface)},
				{Label: networkThreadIDValue, Value: stringOrDash(message.ThreadID)},
				{Label: networkDirectIDValue, Value: stringOrDash(message.DirectID)},
				{Label: networkKindValue, Value: stringOrDash(message.Kind)},
				{Label: "To", Value: stringOrDash(message.To)},
				{Label: "Work ID", Value: stringOrDash(message.WorkID)},
				{Label: "Reply To", Value: stringOrDash(message.ReplyTo)},
				{Label: "Trace ID", Value: stringOrDash(message.TraceID)},
				{Label: "Causation ID", Value: stringOrDash(message.CausationID)},
				{Label: "Expires At", Value: stringOrDash(formatUnixSeconds(message.ExpiresAt))},
				{Label: "Ext", Value: stringOrDash(networkCompactExt(message.Ext))},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("network_message", []string{
				"id",
				"session_id",
				networkChannelKey,
				networkSurfaceKey,
				networkThreadIDKey,
				networkDirectIDKey,
				networkKindKey,
				"to",
				networkWorkIDKey,
				"reply_to",
				"trace_id",
				"causation_id",
				mcpAuthExpiresAtKey,
				"ext",
			}, []string{
				message.ID,
				message.SessionID,
				message.Channel,
				message.Surface,
				message.ThreadID,
				message.DirectID,
				message.Kind,
				message.To,
				message.WorkID,
				message.ReplyTo,
				message.TraceID,
				message.CausationID,
				formatUnixSeconds(message.ExpiresAt),
				networkCompactExt(message.Ext),
			}), nil
		},
	}
}

func networkInboxBundle(messages []NetworkEnvelopeRecord) outputBundle {
	return listBundle(
		messages,
		messages,
		"Network Inbox",
		[]string{
			"ID",
			networkKindValue,
			networkChannelValue,
			networkFromValue,
			"To",
			"Reply To",
			"Trace",
			"Workflow",
			"Handoff",
		},
		"network_inbox",
		[]string{
			"id",
			networkKindKey,
			networkChannelKey,
			"from",
			"to",
			"reply_to",
			"trace_id",
			"causation_id",
			"workflow_id",
			"handoff_version",
			mcpAuthExpiresAtKey,
		},
		func(message NetworkEnvelopeRecord) []string {
			return []string{
				stringOrDash(message.ID),
				stringOrDash(message.Kind),
				stringOrDash(message.Channel),
				stringOrDash(message.From),
				stringOrDash(optionalString(message.To)),
				stringOrDash(optionalString(message.ReplyTo)),
				stringOrDash(optionalString(message.TraceID)),
				stringOrDash(networkExtString(message.Ext, "agh.workflow_id")),
				stringOrDash(networkExtString(message.Ext, "agh.handoff_version")),
			}
		},
		func(message NetworkEnvelopeRecord) []string {
			return []string{
				message.ID,
				message.Kind,
				message.Channel,
				message.From,
				optionalString(message.To),
				optionalString(message.ReplyTo),
				optionalString(message.TraceID),
				optionalString(message.CausationID),
				networkExtString(message.Ext, "agh.workflow_id"),
				networkExtString(message.Ext, "agh.handoff_version"),
				formatUnixSeconds(message.ExpiresAt),
			}
		},
	)
}

func writeCommandOutputWithJSONL[T any](cmd *cobra.Command, bundle outputBundle, items []T) error {
	mode, err := resolveOutputFormat(cmd)
	if err != nil {
		return err
	}
	if mode == OutputJSONL {
		return writeJSONLines(cmd, items)
	}
	return writeCommandOutput(cmd, bundle)
}

func networkThreadHumanRow(thread NetworkThreadRecord) []string {
	return []string{
		stringOrDash(thread.ThreadID),
		stringOrDash(thread.RootMessageID),
		stringOrDash(thread.OpenedByPeerID),
		strconv.Itoa(thread.MessageCount),
		strconv.Itoa(thread.ParticipantCount),
		strconv.Itoa(thread.OpenWorkCount),
		stringOrDash(formatTimePtr(thread.LastActivityAt)),
		stringOrDash(thread.LastMessagePreview),
	}
}

func networkThreadToonRow(thread NetworkThreadRecord) []string {
	return []string{
		thread.Channel,
		thread.ThreadID,
		thread.RootMessageID,
		thread.OpenedByPeerID,
		strconv.Itoa(thread.MessageCount),
		strconv.Itoa(thread.ParticipantCount),
		strconv.Itoa(thread.OpenWorkCount),
		formatTimePtr(thread.LastActivityAt),
		thread.LastMessagePreview,
	}
}

func networkThreadKeyValues(thread NetworkThreadRecord) []keyValue {
	return []keyValue{
		{Label: networkChannelValue, Value: stringOrDash(thread.Channel)},
		{Label: networkThreadIDValue, Value: stringOrDash(thread.ThreadID)},
		{Label: "Root Message", Value: stringOrDash(thread.RootMessageID)},
		{Label: networkTitleValue, Value: stringOrDash(thread.Title)},
		{Label: networkOpenedByValue, Value: stringOrDash(thread.OpenedByPeerID)},
		{Label: "Opened Session", Value: stringOrDash(thread.OpenedSessionID)},
		{Label: networkOpenedAtValue, Value: stringOrDash(formatTimePtr(thread.OpenedAt))},
		{Label: networkLastActivityValue, Value: stringOrDash(formatTimePtr(thread.LastActivityAt))},
		{Label: networkMessagesValue, Value: strconv.Itoa(thread.MessageCount)},
		{Label: "Participants", Value: strconv.Itoa(thread.ParticipantCount)},
		{Label: networkOpenWorkValue, Value: strconv.Itoa(thread.OpenWorkCount)},
		{Label: toolOperatorPreviewValue, Value: stringOrDash(thread.LastMessagePreview)},
	}
}

func networkDirectHumanRow(direct NetworkDirectRoomRecord) []string {
	return []string{
		stringOrDash(direct.DirectID),
		stringOrDash(direct.PeerA),
		stringOrDash(direct.PeerB),
		strconv.Itoa(direct.MessageCount),
		strconv.Itoa(direct.OpenWorkCount),
		stringOrDash(formatTimePtr(direct.LastActivityAt)),
		stringOrDash(direct.LastMessagePreview),
	}
}

func networkDirectToonRow(direct NetworkDirectRoomRecord) []string {
	return []string{
		direct.Channel,
		direct.DirectID,
		direct.PeerA,
		direct.PeerB,
		strconv.Itoa(direct.MessageCount),
		strconv.Itoa(direct.OpenWorkCount),
		formatTimePtr(direct.LastActivityAt),
		direct.LastMessagePreview,
	}
}

func networkDirectKeyValues(direct NetworkDirectRoomRecord) []keyValue {
	return []keyValue{
		{Label: networkChannelValue, Value: stringOrDash(direct.Channel)},
		{Label: networkDirectIDValue, Value: stringOrDash(direct.DirectID)},
		{Label: "Peer A", Value: stringOrDash(direct.PeerA)},
		{Label: "Peer B", Value: stringOrDash(direct.PeerB)},
		{Label: networkOpenedAtValue, Value: stringOrDash(formatTimePtr(direct.OpenedAt))},
		{Label: networkLastActivityValue, Value: stringOrDash(formatTimePtr(direct.LastActivityAt))},
		{Label: networkMessagesValue, Value: strconv.Itoa(direct.MessageCount)},
		{Label: networkOpenWorkValue, Value: strconv.Itoa(direct.OpenWorkCount)},
		{Label: toolOperatorPreviewValue, Value: stringOrDash(direct.LastMessagePreview)},
	}
}

func networkMessageHumanRow(message NetworkConversationMessageRecord) []string {
	return []string{
		stringOrDash(message.MessageID),
		stringOrDash(message.Surface),
		stringOrDash(message.ThreadID),
		stringOrDash(message.DirectID),
		stringOrDash(message.Kind),
		stringOrDash(message.Direction),
		stringOrDash(message.PeerFrom),
		stringOrDash(message.PeerTo),
		stringOrDash(message.WorkID),
		stringOrDash(formatTime(message.Timestamp)),
		stringOrDash(networkMessagePreview(message)),
	}
}

func networkMessageToonRow(message NetworkConversationMessageRecord) []string {
	return []string{
		message.MessageID,
		message.Channel,
		message.Surface,
		message.ThreadID,
		message.DirectID,
		message.Kind,
		message.Direction,
		message.PeerFrom,
		message.PeerTo,
		message.WorkID,
		formatTime(message.Timestamp),
		networkMessagePreview(message),
	}
}

func networkMessagePreview(message NetworkConversationMessageRecord) string {
	if trimmed := strings.TrimSpace(message.PreviewText); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(message.Text)
}

func networkKindMetricRows(metrics []NetworkKindMetricRecord) [][]string {
	rows := make([][]string, 0, len(metrics))
	for _, metric := range metrics {
		rows = append(rows, []string{
			metric.Kind,
			strconv.FormatInt(metric.Sent, 10),
			strconv.FormatInt(metric.Received, 10),
			strconv.FormatInt(metric.Rejected, 10),
			strconv.FormatInt(metric.Delivered, 10),
		})
	}
	return rows
}

func parseNetworkJSONValue(flagName string, raw string) (json.RawMessage, error) {
	payload, err := parseRequiredJSONRawMessage(raw)
	if errors.Is(err, errEmptyJSONFlag) {
		return nil, fmt.Errorf("cli: %s is required", flagName)
	}
	if err != nil {
		return nil, fmt.Errorf("cli: %s must be valid JSON: %w", flagName, err)
	}
	return payload, nil
}

func parseNetworkJSONObjectMap(flagName string, raw string) (map[string]json.RawMessage, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	if !strings.HasPrefix(trimmed, "{") {
		return nil, fmt.Errorf("cli: %s must be a JSON object", flagName)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return nil, fmt.Errorf("cli: decode %s: %w", flagName, err)
	}
	return payload, nil
}

func validateNetworkSendNoRawClaimToken(body json.RawMessage, ext map[string]json.RawMessage) error {
	payload := struct {
		Body json.RawMessage            `json:"body"`
		Ext  map[string]json.RawMessage `json:"ext,omitempty"`
	}{
		Body: body,
		Ext:  ext,
	}
	if err := contract.ValidateNoRawClaimTokenField(payload); err != nil {
		return fmt.Errorf(
			"cli: network_raw_token_rejected: --body/--ext must not contain raw claim_token fields: %w",
			err,
		)
	}
	return nil
}

func validateNetworkSendFlags(flags networkSendFlags) error {
	kind := strings.TrimSpace(flags.kind)
	if kind == networkSurfaceDirect {
		return errors.New("cli: --kind direct is not supported; use --surface direct with --kind say")
	}

	surface := strings.TrimSpace(flags.surface)
	threadID := strings.TrimSpace(flags.threadID)
	directID := strings.TrimSpace(flags.directID)
	workID := strings.TrimSpace(flags.workID)
	if err := validateNetworkSendConversationFlags(kind, surface, threadID, directID); err != nil {
		return err
	}
	return validateNetworkSendKindScope(kind, surface, threadID, directID, workID)
}

func validateNetworkSendConversationFlags(kind string, surface string, threadID string, directID string) error {
	if threadID != "" && directID != "" {
		return errors.New("cli: --thread and --direct cannot be used together")
	}
	if surface == "" {
		if threadID != "" || directID != "" {
			return errors.New("cli: --surface is required when --thread or --direct is set")
		}
		if networkKindRequiresConversation(kind) {
			return fmt.Errorf("cli: --surface is required for --kind %s", kind)
		}
	} else {
		switch surface {
		case networkSurfaceThread:
			if threadID == "" {
				return errors.New("cli: --thread is required when --surface thread is set")
			}
			if directID != "" {
				return errors.New("cli: --direct cannot be used when --surface thread is set")
			}
		case networkSurfaceDirect:
			if directID == "" {
				return errors.New("cli: --direct is required when --surface direct is set")
			}
			if threadID != "" {
				return errors.New("cli: --thread cannot be used when --surface direct is set")
			}
		default:
			return errors.New("cli: --surface must be thread or direct")
		}
	}
	return nil
}

func validateNetworkSendKindScope(kind string, surface string, threadID string, directID string, workID string) error {
	if networkKindForbidsConversation(kind) && (surface != "" || threadID != "" || directID != "" || workID != "") {
		return fmt.Errorf("cli: --kind %s cannot include --surface, --thread, --direct, or --work", kind)
	}
	if networkKindRequiresWork(kind) && workID == "" {
		return fmt.Errorf("cli: --kind %s requires --work", kind)
	}
	return nil
}

func networkKindForbidsConversation(kind string) bool {
	switch kind {
	case networkKindGreet, networkKindWhois:
		return true
	default:
		return false
	}
}

func networkKindRequiresWork(kind string) bool {
	switch kind {
	case networkKindCapability, networkKindReceipt, networkKindTrace:
		return true
	default:
		return false
	}
}

func networkKindRequiresConversation(kind string) bool {
	switch kind {
	case networkKindSay, networkKindCapability, networkKindReceipt, networkKindTrace:
		return true
	default:
		return false
	}
}

func parseNetworkExpiresAt(raw string) (*int64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	if unixSeconds, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return &unixSeconds, nil
	}

	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil, errors.New("cli: --expires-at must be unix seconds or RFC3339")
	}
	unixSeconds := parsed.UTC().Unix()
	return &unixSeconds, nil
}

func networkCompactExt(ext map[string]json.RawMessage) string {
	if len(ext) == 0 {
		return ""
	}
	payload, err := json.Marshal(ext)
	if err != nil {
		return ""
	}
	return compactJSON(payload)
}

func networkExtString(ext map[string]json.RawMessage, key string) string {
	if len(ext) == 0 {
		return ""
	}
	raw, ok := ext[strings.TrimSpace(key)]
	if !ok || len(raw) == 0 {
		return ""
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return strings.TrimSpace(text)
	}
	return strings.TrimSpace(compactJSON(raw))
}

func formatUnixSeconds(value *int64) string {
	if value == nil {
		return ""
	}
	return time.Unix(*value, 0).UTC().Format(time.RFC3339)
}

func formatTimePtr(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatTime(*value)
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
