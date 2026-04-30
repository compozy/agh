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

func newNetworkCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Operate the daemon-owned network runtime",
	}

	cmd.AddCommand(newNetworkStatusCommand(deps))
	cmd.AddCommand(newNetworkPeersCommand(deps))
	cmd.AddCommand(newNetworkChannelsCommand(deps))
	cmd.AddCommand(newNetworkSendCommand(deps))
	cmd.AddCommand(newNetworkInboxCommand(deps))
	return cmd
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

func newNetworkPeersCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "peers [channel]",
		Short: "List visible local and remote peers",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			query := NetworkPeersQuery{}
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

func newNetworkChannelsCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "channels",
		Short: "List active runtime channels",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			channels, err := client.NetworkChannels(cmd.Context())
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, networkChannelsBundle(channels))
		},
	}
}

func newNetworkSendCommand(deps commandDeps) *cobra.Command {
	var (
		sessionID     string
		channel       string
		kind          string
		to            string
		bodyRaw       string
		interactionID string
		replyTo       string
		traceID       string
		causationID   string
		expiresAtRaw  string
		id            string
		extRaw        string
	)

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send one envelope through the daemon-owned network runtime",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			body, err := parseNetworkJSONValue("--body", bodyRaw)
			if err != nil {
				return err
			}
			ext, err := parseNetworkJSONObjectMap("--ext", extRaw)
			if err != nil {
				return err
			}
			if err := validateNetworkSendNoRawClaimToken(body, ext); err != nil {
				return err
			}
			expiresAt, err := parseNetworkExpiresAt(expiresAtRaw)
			if err != nil {
				return err
			}

			message, err := client.NetworkSend(cmd.Context(), NetworkSendRequest{
				SessionID:     strings.TrimSpace(sessionID),
				Channel:       strings.TrimSpace(channel),
				Kind:          strings.TrimSpace(kind),
				To:            strings.TrimSpace(to),
				Body:          body,
				InteractionID: strings.TrimSpace(interactionID),
				ReplyTo:       strings.TrimSpace(replyTo),
				TraceID:       strings.TrimSpace(traceID),
				CausationID:   strings.TrimSpace(causationID),
				ExpiresAt:     expiresAt,
				ID:            strings.TrimSpace(id),
				Ext:           ext,
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, networkSendBundle(message))
		},
	}

	cmd.Flags().StringVar(&sessionID, "session", "", "Local source session id")
	cmd.Flags().StringVar(&channel, "channel", "", "Target channel")
	cmd.Flags().StringVar(&kind, "kind", "", "Envelope kind")
	cmd.Flags().StringVar(&to, "to", "", "Directed target peer id")
	cmd.Flags().StringVar(&bodyRaw, "body", "", "Raw JSON object for the envelope body")
	cmd.Flags().StringVar(&interactionID, "interaction-id", "", "Optional interaction id")
	cmd.Flags().StringVar(&replyTo, "reply-to", "", "Optional reply-to message id")
	cmd.Flags().StringVar(&traceID, "trace-id", "", "Optional trace id")
	cmd.Flags().StringVar(&causationID, "causation-id", "", "Optional causation id")
	cmd.Flags().StringVar(&expiresAtRaw, "expires-at", "", "Optional expiry as unix seconds or RFC3339")
	cmd.Flags().StringVar(&id, "id", "", "Optional explicit message id")
	cmd.Flags().StringVar(&extRaw, "ext", "", "Optional JSON object of extension metadata")
	mustMarkFlagRequired(cmd, "session")
	mustMarkFlagRequired(cmd, "channel")
	mustMarkFlagRequired(cmd, "kind")
	mustMarkFlagRequired(cmd, "body")
	return cmd
}

func newNetworkInboxCommand(deps commandDeps) *cobra.Command {
	var sessionID string

	cmd := &cobra.Command{
		Use:   "inbox",
		Short: "Show queued inbound messages for one session",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			messages, err := client.NetworkInbox(cmd.Context(), strings.TrimSpace(sessionID))
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

func networkSendBundle(message NetworkSendRecord) outputBundle {
	return outputBundle{
		jsonValue: message,
		human: func() (string, error) {
			return renderHumanSection("Network Message", []keyValue{
				{Label: "ID", Value: stringOrDash(message.ID)},
				{Label: "Session", Value: stringOrDash(message.SessionID)},
				{Label: "Channel", Value: stringOrDash(message.Channel)},
				{Label: "Kind", Value: stringOrDash(message.Kind)},
				{Label: "To", Value: stringOrDash(message.To)},
				{Label: "Interaction", Value: stringOrDash(message.InteractionID)},
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
				"kind",
				"to",
				"interaction_id",
				"reply_to",
				"trace_id",
				"causation_id",
				"expires_at",
				"ext",
			}, []string{
				message.ID,
				message.SessionID,
				message.Channel,
				message.Kind,
				message.To,
				message.InteractionID,
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
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("cli: %s is required", flagName)
	}
	if !json.Valid([]byte(trimmed)) {
		return nil, fmt.Errorf("cli: %s must be valid JSON", flagName)
	}
	return json.RawMessage(trimmed), nil
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
