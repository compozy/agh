package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/spf13/cobra"
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
		Use:   "network",
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
		Use:   "status",
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
		Short: "List visible local and remote peers",
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
	return &cobra.Command{
		Use:   "channels",
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
		Use:   "list",
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
	cmd.Flags().StringVar(&flags.channel, "channel", "", "Target channel")
	cmd.Flags().IntVar(&flags.limit, "limit", 0, "Maximum number of threads to return")
	cmd.Flags().StringVar(&flags.after, "after", "", "Cursor after which to list threads")
	mustMarkFlagRequired(cmd, "channel")
	return cmd
}

func newNetworkThreadsShowCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	var flags networkThreadsFlags
	cmd := &cobra.Command{
		Use:   "show",
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
	cmd.Flags().StringVar(&flags.channel, "channel", "", "Target channel")
	cmd.Flags().StringVar(&flags.threadID, networkSurfaceThread, "", "Public thread id")
	mustMarkFlagRequired(cmd, "channel")
	mustMarkFlagRequired(cmd, networkSurfaceThread)
	return cmd
}

func newNetworkThreadsMessagesCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	var flags networkThreadsFlags
	cmd := &cobra.Command{
		Use:   "messages",
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
	mustMarkFlagRequired(cmd, "channel")
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
		Use:   "list",
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
	cmd.Flags().StringVar(&flags.channel, "channel", "", "Target channel")
	cmd.Flags().StringVar(&flags.peerID, "peer", "", "Peer id filter")
	cmd.Flags().IntVar(&flags.limit, "limit", 0, "Maximum number of direct rooms to return")
	cmd.Flags().StringVar(&flags.after, "after", "", "Cursor after which to list direct rooms")
	mustMarkFlagRequired(cmd, "channel")
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
	cmd.Flags().StringVar(&flags.channel, "channel", "", "Target channel")
	cmd.Flags().StringVar(&flags.peerID, "peer", "", "Remote peer id")
	mustMarkFlagRequired(cmd, "session")
	mustMarkFlagRequired(cmd, "channel")
	mustMarkFlagRequired(cmd, "peer")
	return cmd
}

func newNetworkDirectsShowCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	var flags networkDirectsFlags
	cmd := &cobra.Command{
		Use:   "show",
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
	cmd.Flags().StringVar(&flags.channel, "channel", "", "Target channel")
	cmd.Flags().StringVar(&flags.directID, networkSurfaceDirect, "", "Direct room id")
	mustMarkFlagRequired(cmd, "channel")
	mustMarkFlagRequired(cmd, networkSurfaceDirect)
	return cmd
}

func newNetworkDirectsMessagesCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	var flags networkDirectsFlags
	cmd := &cobra.Command{
		Use:   "messages",
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
	mustMarkFlagRequired(cmd, "channel")
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
	cmd.Flags().StringVar(channel, "channel", "", "Target channel")
	cmd.Flags().StringVar(containerID, containerFlagName, "", containerUsage)
	cmd.Flags().IntVar(limit, "limit", 0, "Maximum number of messages to return")
	cmd.Flags().StringVar(before, "before", "", "Cursor before which to list messages")
	cmd.Flags().StringVar(after, "after", "", "Cursor after which to list messages")
	cmd.Flags().StringVar(kind, "kind", "", "Envelope kind filter")
	cmd.Flags().StringVar(workID, "work", "", "Work id filter")
}

func newNetworkWorkCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "work",
		Short: "Inspect lifecycle-bearing network work",
	}
	cmd.AddCommand(newNetworkWorkLookupCommand(deps, workspaceRef, "lookup"))
	cmd.AddCommand(newNetworkWorkLookupCommand(deps, workspaceRef, "status"))
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
		Use:   "send",
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
	mustMarkFlagRequired(cmd, "channel")
	mustMarkFlagRequired(cmd, "kind")
	mustMarkFlagRequired(cmd, "body")
	return cmd
}

func registerNetworkSendFlags(cmd *cobra.Command, flags *networkSendFlags) {
	cmd.Flags().StringVar(&flags.sessionID, "session", "", "Local source session id")
	cmd.Flags().StringVar(&flags.channel, "channel", "", "Target channel")
	cmd.Flags().StringVar(&flags.surface, "surface", "", "Conversation surface: thread or direct")
	cmd.Flags().StringVar(&flags.threadID, networkSurfaceThread, "", "Thread id for thread-surface messages")
	cmd.Flags().StringVar(&flags.directID, networkSurfaceDirect, "", "Direct room id for direct-surface messages")
	cmd.Flags().StringVar(&flags.kind, "kind", "", "Envelope kind")
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
		{Label: "Enabled", Value: strconv.FormatBool(status.Enabled)},
		{Label: "Status", Value: stringOrDash(status.Status)},
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
		"enabled", "status", "listener", "local_peers", "remote_peers", "channels",
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
					[]string{"Kind", "Sent", "Received", "Rejected", "Delivered"},
					networkKindMetricRows(status.KindMetrics),
				),
			), nil
		},
		toon: func() (string, error) {
			return renderHumanBlocks(
				renderToonObject("network", fields, values),
				renderToonArray(
					"network_kind_metrics",
					[]string{"kind", "sent", "received", "rejected", "delivered"},
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
		[]string{"Peer", "Display", "Session", "Channel", "Local", "Joined", "Last Seen", "Expires"},
		"network_peers",
		[]string{"peer_id", "display_name", "session_id", "channel", "local", "joined_at", "last_seen", "expires_at"},
		func(peer NetworkPeerRecord) []string {
			return []string{
				stringOrDash(peer.PeerID),
				stringOrDash(optionalString(peer.PeerCard.DisplayName)),
				stringOrDash(optionalString(peer.SessionID)),
				stringOrDash(peer.Channel),
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
				strconv.FormatBool(peer.Local),
				formatTimePtr(peer.JoinedAt),
				formatTimePtr(peer.LastSeen),
				formatTimePtr(peer.ExpiresAt),
			}
		},
	)
}

func networkChannelsBundle(channels []NetworkChannelRecord) outputBundle {
	return listBundle(
		channels,
		channels,
		"Network Channels",
		[]string{"Channel", "Peers"},
		"network_channels",
		[]string{"channel", "peer_count"},
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

func networkThreadsBundle(threads []NetworkThreadRecord) outputBundle {
	return listBundle(
		contract.NetworkThreadsResponse{Threads: threads},
		threads,
		"Network Threads",
		[]string{"Thread", "Root", "Opened By", "Messages", "Participants", "Open Work", "Last Activity", "Preview"},
		"network_threads",
		[]string{
			"channel",
			"thread_id",
			"root_message_id",
			"opened_by_peer_id",
			"message_count",
			"participant_count",
			"open_work_count",
			"last_activity_at",
			"last_message_preview",
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
					"channel",
					"thread_id",
					"root_message_id",
					"title",
					"opened_by_peer_id",
					"opened_session_id",
					"opened_at",
					"last_activity_at",
					"message_count",
					"participant_count",
					"open_work_count",
					"last_message_preview",
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
		[]string{"Direct", "Peer A", "Peer B", "Messages", "Open Work", "Last Activity", "Preview"},
		"network_directs",
		[]string{
			"channel",
			"direct_id",
			"peer_a",
			"peer_b",
			"message_count",
			"open_work_count",
			"last_activity_at",
			"last_message_preview",
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
					"channel",
					"direct_id",
					"peer_a",
					"peer_b",
					"opened_at",
					"last_activity_at",
					"message_count",
					"open_work_count",
					"last_message_preview",
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
			"Message",
			"Surface",
			"Thread",
			"Direct",
			"Kind",
			"Direction",
			"From",
			"To",
			"Work",
			"Timestamp",
			"Preview",
		},
		"network_messages",
		[]string{
			"message_id",
			"channel",
			"surface",
			"thread_id",
			"direct_id",
			"kind",
			"direction",
			"peer_from",
			"peer_to",
			"work_id",
			"timestamp",
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
				{Label: "Channel", Value: stringOrDash(work.Channel)},
				{Label: "Surface", Value: stringOrDash(work.Surface)},
				{Label: "Thread ID", Value: stringOrDash(work.ThreadID)},
				{Label: "Direct ID", Value: stringOrDash(work.DirectID)},
				{Label: "Opened By", Value: stringOrDash(work.OpenedByPeerID)},
				{Label: "Opened Session", Value: stringOrDash(work.OpenedSessionID)},
				{Label: "Target Peer", Value: stringOrDash(work.TargetPeerID)},
				{Label: "State", Value: stringOrDash(work.State)},
				{Label: "Opened At", Value: stringOrDash(formatTimePtr(work.OpenedAt))},
				{Label: "Last Activity", Value: stringOrDash(formatTimePtr(work.LastActivityAt))},
				{Label: "Terminal At", Value: stringOrDash(formatTimePtr(work.TerminalAt))},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject(
				"network_work",
				[]string{
					"work_id",
					"channel",
					"surface",
					"thread_id",
					"direct_id",
					"opened_by_peer_id",
					"opened_session_id",
					"target_peer_id",
					"state",
					"opened_at",
					"last_activity_at",
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
				{Label: "Session", Value: stringOrDash(message.SessionID)},
				{Label: "Channel", Value: stringOrDash(message.Channel)},
				{Label: "Surface", Value: stringOrDash(message.Surface)},
				{Label: "Thread ID", Value: stringOrDash(message.ThreadID)},
				{Label: "Direct ID", Value: stringOrDash(message.DirectID)},
				{Label: "Kind", Value: stringOrDash(message.Kind)},
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
				"channel",
				"surface",
				"thread_id",
				"direct_id",
				"kind",
				"to",
				"work_id",
				"reply_to",
				"trace_id",
				"causation_id",
				"expires_at",
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
		[]string{"ID", "Kind", "Channel", "From", "To", "Reply To", "Trace", "Workflow", "Handoff"},
		"network_inbox",
		[]string{
			"id",
			"kind",
			"channel",
			"from",
			"to",
			"reply_to",
			"trace_id",
			"causation_id",
			"workflow_id",
			"handoff_version",
			"expires_at",
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
		{Label: "Channel", Value: stringOrDash(thread.Channel)},
		{Label: "Thread ID", Value: stringOrDash(thread.ThreadID)},
		{Label: "Root Message", Value: stringOrDash(thread.RootMessageID)},
		{Label: "Title", Value: stringOrDash(thread.Title)},
		{Label: "Opened By", Value: stringOrDash(thread.OpenedByPeerID)},
		{Label: "Opened Session", Value: stringOrDash(thread.OpenedSessionID)},
		{Label: "Opened At", Value: stringOrDash(formatTimePtr(thread.OpenedAt))},
		{Label: "Last Activity", Value: stringOrDash(formatTimePtr(thread.LastActivityAt))},
		{Label: "Messages", Value: strconv.Itoa(thread.MessageCount)},
		{Label: "Participants", Value: strconv.Itoa(thread.ParticipantCount)},
		{Label: "Open Work", Value: strconv.Itoa(thread.OpenWorkCount)},
		{Label: "Preview", Value: stringOrDash(thread.LastMessagePreview)},
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
		{Label: "Channel", Value: stringOrDash(direct.Channel)},
		{Label: "Direct ID", Value: stringOrDash(direct.DirectID)},
		{Label: "Peer A", Value: stringOrDash(direct.PeerA)},
		{Label: "Peer B", Value: stringOrDash(direct.PeerB)},
		{Label: "Opened At", Value: stringOrDash(formatTimePtr(direct.OpenedAt))},
		{Label: "Last Activity", Value: stringOrDash(formatTimePtr(direct.LastActivityAt))},
		{Label: "Messages", Value: strconv.Itoa(direct.MessageCount)},
		{Label: "Open Work", Value: strconv.Itoa(direct.OpenWorkCount)},
		{Label: "Preview", Value: stringOrDash(direct.LastMessagePreview)},
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
