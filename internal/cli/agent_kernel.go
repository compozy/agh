package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/agentidentity"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/spf13/cobra"
)

const (
	automationSessionIDKey = "session_id"
)

const (
	bridgeKindValue     = "Kind"
	networkTimestampKey = "timestamp"
)

const (
	agentKernelModelKey = "model"
)

const (
	agentKernelAgentValue   = "Agent"
	agentKernelChannelValue = "Channel"
	agentKernelFromValue    = "From"
	agentKernelRootValue    = "Root"
	agentKernelSessionValue = "Session"
	agentKernelAgentNameKey = "agent_name"
	agentKernelChannelKey   = "channel"
	agentKernelContextKey   = "context"
	agentKernelKindKey      = "kind"
	agentKernelListKey      = "list"
	agentKernelReplyKey     = "reply"
	agentKernelRunIDKey     = "run_id"
)

func newMeCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "me",
		Short: "Inspect the current AGH-managed agent session",
		Example: `  # Show the current managed session identity
  agh me

  # Print machine-readable caller state
  agh me -o json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			credentials, err := requireAgentCommandIdentity(cmd.Context(), deps, client, agentActionCLI("me"))
			if err != nil {
				return err
			}
			record, err := client.AgentMe(cmd.Context(), credentials)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentMeBundle(record))
		},
	}
	cmd.AddCommand(newMeContextCommand(deps))
	return cmd
}

func newMeContextCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   agentKernelContextKey,
		Short: "Inspect the bounded situation context for the current agent session",
		Example: `  # Show the bounded situation context injected for this session
  agh me context

  # Read the context payload as JSON
  agh me context -o json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			credentials, err := requireAgentCommandIdentity(cmd.Context(), deps, client, agentActionCLI("me.context"))
			if err != nil {
				return err
			}
			record, err := client.AgentContext(cmd.Context(), credentials)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentContextBundle(&record))
		},
	}
}

func newChannelCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ch",
		Aliases: []string{agentKernelChannelKey},
		Short:   "Use agent-facing coordination channels",
		Example: `  # List channels visible to this session
  agh ch list

  # Wait for task-run coordination messages
  agh ch recv coord-run-123 --wait -o jsonl`,
	}
	cmd.AddCommand(newChannelListCommand(deps))
	cmd.AddCommand(newChannelRecvCommand(deps))
	cmd.AddCommand(newChannelSendCommand(deps))
	cmd.AddCommand(newChannelReplyCommand(deps))
	return cmd
}

func newChannelListCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   agentKernelListKey,
		Short: "List coordination channels visible to the current agent session",
		Example: `  # List coordination channels visible to this session
  agh ch list

  # Print channel discovery metadata as JSON
  agh ch list -o json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			credentials, err := requireAgentCommandIdentity(cmd.Context(), deps, client, agentActionCLI("ch.list"))
			if err != nil {
				return err
			}
			channels, err := client.AgentChannels(cmd.Context(), credentials)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentChannelsBundle(channels))
		},
	}
}

func newChannelRecvCommand(deps commandDeps) *cobra.Command {
	var wait bool
	var limit int

	cmd := &cobra.Command{
		Use:   "recv <channel>",
		Short: "Receive queued coordination messages for a channel",
		Args:  exactOneNonBlankArg(),
		Example: `  # Receive currently queued messages
  agh ch recv coord-run-123

  # Wait for messages and stream each record as JSONL
  agh ch recv coord-run-123 --wait --limit 10 -o jsonl`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			credentials, err := requireAgentCommandIdentity(cmd.Context(), deps, client, agentActionCLI("ch.recv"))
			if err != nil {
				return err
			}
			messages, err := client.AgentChannelRecv(cmd.Context(), strings.TrimSpace(args[0]), AgentChannelRecvQuery{
				Wait:  wait,
				Limit: limit,
			}, credentials)
			if err != nil {
				return err
			}
			return writeAgentChannelMessages(cmd, messages)
		},
	}
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait until at least one matching message is available")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum messages to return")
	return cmd
}

func newChannelSendCommand(deps commandDeps) *cobra.Command {
	var bodyRaw string
	var idempotencyKey string
	flags := coordinationMetadataFlags{kind: string(contract.CoordinationMessageStatus)}

	cmd := &cobra.Command{
		Use:   "send <channel>",
		Short: "Send one task-run coordination message",
		Args:  exactOneNonBlankArg(),
		Example: `  # Send a non-authoritative status message for a run
  agh ch send coord-run-123 \
    --task-id task-123 \
    --run-id run-123 \
    --kind status \
    --correlation-id run-123 \
    --body '{"status":"investigating"}'

  # Report a blocker in the same task-bound channel
  agh ch send coord-run-123 \
    --task-id task-123 \
    --run-id run-123 \
    --kind blocker \
    --correlation-id run-123 \
		--body '{"blocked_by":"missing credentials"}'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			channel := strings.TrimSpace(args[0])
			flags.kindExplicit = cmd.Flags().Changed(agentKernelKindKey)
			body, err := parseNetworkJSONValue("--body", bodyRaw)
			if err != nil {
				return err
			}
			metadata, err := flags.metadata(channel, contract.CoordinationMessageStatus, true)
			if err != nil {
				return err
			}
			if metadata.MessageKind == contract.CoordinationMessageReply {
				return errors.New("cli: use `agh ch reply --to-message` for reply messages")
			}
			request := AgentChannelSendRequest{
				Body:           body,
				Metadata:       metadata,
				IdempotencyKey: strings.TrimSpace(idempotencyKey),
			}
			if err := contract.ValidateNoRawClaimTokenField(request); err != nil {
				return err
			}

			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			credentials, err := requireAgentCommandIdentity(cmd.Context(), deps, client, agentActionCLI("ch.send"))
			if err != nil {
				return err
			}
			message, err := client.AgentChannelSend(cmd.Context(), channel, request, credentials)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentChannelMessageBundle(message))
		},
	}
	cmd.Flags().StringVar(&bodyRaw, "body", "", "Raw JSON body for the channel message")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Optional idempotency key/message id")
	flags.bind(cmd)
	mustMarkFlagRequired(cmd, "body")
	return cmd
}

func newChannelReplyCommand(deps commandDeps) *cobra.Command {
	var bodyRaw string
	var toMessage string
	var idempotencyKey string
	flags := coordinationMetadataFlags{kind: string(contract.CoordinationMessageReply)}

	cmd := &cobra.Command{
		Use:   agentKernelReplyKey,
		Short: "Reply to a received coordination message",
		Example: `  # Reply to one received coordination message
  agh ch reply --to-message msg-123 --body '{"answer":"ready for review"}'

  # Add explicit correlation metadata when replying from outside an inherited delivery
  agh ch reply \
    --to-message msg-123 \
    --task-id task-123 \
    --run-id run-123 \
    --coordination-channel-id coord-run-123 \
    --correlation-id run-123 \
    --body '{"answer":"ready for review"}'`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			flags.kindExplicit = cmd.Flags().Changed(agentKernelKindKey)
			if flags.kindExplicit &&
				contract.CoordinationMessageKind(strings.TrimSpace(flags.kind)) != contract.CoordinationMessageReply {
				return errors.New("cli: --kind must be reply for `agh ch reply`")
			}
			body, err := parseNetworkJSONValue("--body", bodyRaw)
			if err != nil {
				return err
			}
			metadata, err := flags.metadata("", contract.CoordinationMessageReply, false)
			if err != nil {
				return err
			}
			if !zeroCLIAgentCoordinationMetadata(metadata) &&
				metadata.MessageKind != contract.CoordinationMessageReply {
				return errors.New("cli: --kind must be reply for `agh ch reply`")
			}
			request := AgentChannelReplyRequest{
				ReplyToMessageID: strings.TrimSpace(toMessage),
				Body:             body,
				Metadata:         metadata,
				IdempotencyKey:   strings.TrimSpace(idempotencyKey),
			}
			if err := contract.ValidateNoRawClaimTokenField(request); err != nil {
				return err
			}

			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			credentials, err := requireAgentCommandIdentity(cmd.Context(), deps, client, agentActionCLI("ch.reply"))
			if err != nil {
				return err
			}
			message, err := client.AgentChannelReply(cmd.Context(), request, credentials)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentChannelMessageBundle(message))
		},
	}
	cmd.Flags().StringVar(&toMessage, "to-message", "", "Message id to reply to")
	cmd.Flags().StringVar(&bodyRaw, "body", "", "Raw JSON body for the reply message")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Optional idempotency key/message id")
	flags.bind(cmd)
	mustMarkFlagRequired(cmd, "to-message")
	mustMarkFlagRequired(cmd, "body")
	return cmd
}

type coordinationMetadataFlags struct {
	taskID                string
	runID                 string
	workflowID            string
	coordinationChannelID string
	kind                  string
	kindExplicit          bool
	correlationID         string
	extRaw                string
}

func (f *coordinationMetadataFlags) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.taskID, "task-id", "", "Task id for coordination metadata")
	cmd.Flags().StringVar(&f.runID, "run-id", "", "Run id for coordination metadata")
	cmd.Flags().StringVar(&f.workflowID, "workflow-id", "", "Workflow id for coordination metadata")
	cmd.Flags().
		StringVar(&f.coordinationChannelID, "coordination-channel-id", "", "Coordination channel id; defaults to channel")
	cmd.Flags().StringVar(&f.kind, agentKernelKindKey, f.kind, "Coordination message kind")
	cmd.Flags().StringVar(&f.correlationID, "correlation-id", "", "Correlation id; defaults to run id")
	cmd.Flags().StringVar(&f.extRaw, "metadata-ext", "", "Optional JSON object for coordination metadata ext")
}

func (f coordinationMetadataFlags) metadata(
	channel string,
	defaultKind contract.CoordinationMessageKind,
	required bool,
) (contract.CoordinationMessageMetadataPayload, error) {
	metadataExt, err := parseNetworkJSONObjectMap("--metadata-ext", f.extRaw)
	if err != nil {
		return contract.CoordinationMessageMetadataPayload{}, err
	}

	kindOverride := f.kindExplicit &&
		contract.CoordinationMessageKind(strings.TrimSpace(f.kind)) != defaultKind
	if !required &&
		strings.TrimSpace(f.taskID) == "" &&
		strings.TrimSpace(f.runID) == "" &&
		strings.TrimSpace(f.workflowID) == "" &&
		strings.TrimSpace(f.coordinationChannelID) == "" &&
		strings.TrimSpace(f.correlationID) == "" &&
		len(metadataExt) == 0 &&
		!kindOverride {
		return contract.CoordinationMessageMetadataPayload{}, nil
	}

	kind := contract.CoordinationMessageKind(firstCLIValue(f.kind, string(defaultKind)))
	metadata := contract.CoordinationMessageMetadataPayload{
		TaskID:                strings.TrimSpace(f.taskID),
		RunID:                 strings.TrimSpace(f.runID),
		WorkflowID:            strings.TrimSpace(f.workflowID),
		CoordinationChannelID: firstCLIValue(f.coordinationChannelID, channel),
		MessageKind:           kind,
		CorrelationID:         firstCLIValue(f.correlationID, f.runID),
		Ext:                   metadataExt,
	}
	if err := metadata.Validate(); err != nil {
		return contract.CoordinationMessageMetadataPayload{}, err
	}
	return metadata, nil
}

func requireAgentCommandIdentity(
	ctx context.Context,
	deps commandDeps,
	client DaemonClient,
	originRef string,
) (agentidentity.Credentials, error) {
	if _, err := resolveAgentCallerFromEnv(ctx, deps, client, "", originRef); err != nil {
		return agentidentity.Credentials{}, err
	}
	return agentCredentialsFromEnv(deps), nil
}

func agentActionCLI(action string) string {
	return "agent." + strings.TrimSpace(action)
}

func writeAgentChannelMessages(cmd *cobra.Command, messages []AgentChannelMessageRecord) error {
	mode, err := resolveOutputFormat(cmd)
	if err != nil {
		return err
	}
	if mode == OutputJSONL {
		for _, message := range messages {
			if err := writeJSONLine(cmd, message); err != nil {
				return err
			}
		}
		return nil
	}
	return writeCommandOutput(cmd, agentChannelMessagesBundle(messages))
}

func agentMeBundle(record AgentMeRecord) outputBundle {
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			return renderHumanBlocks(
				renderHumanSection(agentKernelAgentValue, []keyValue{
					{Label: agentKernelSessionValue, Value: record.Self.SessionID},
					{Label: agentKernelAgentValue, Value: record.Self.AgentName},
					{Label: agentKernelProviderValue, Value: record.Self.Provider},
					{Label: agentKernelModelValue, Value: stringOrDash(record.Self.Model)},
				}),
				renderHumanSection("Workspace", []keyValue{
					{Label: "ID", Value: stringOrDash(record.Workspace.ID)},
					{Label: agentKernelRootValue, Value: stringOrDash(record.Workspace.RootDir)},
				}),
			), nil
		},
		toon: func() (string, error) {
			return renderToonObject("agent_me", []string{
				automationSessionIDKey,
				agentKernelAgentNameKey,
				cliProviderKey,
				agentKernelModelKey,
				"workspace_id",
				"workspace_root",
			}, []string{
				record.Self.SessionID,
				record.Self.AgentName,
				record.Self.Provider,
				record.Self.Model,
				record.Workspace.ID,
				record.Workspace.RootDir,
			}), nil
		},
	}
}

func agentContextBundle(record *AgentContextRecord) outputBundle {
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			return renderJSONPreview(record)
		},
		toon: func() (string, error) {
			return renderJSONPreview(record)
		},
	}
}

func agentChannelsBundle(channels []AgentChannelRecord) outputBundle {
	return listBundle(
		channels,
		channels,
		"Agent Channels",
		[]string{"ID", agentKernelChannelValue, "Purpose", taskTaskValue, taskRunValue},
		"agent_channels",
		[]string{"id", agentKernelChannelKey, "purpose", "task_id", agentKernelRunIDKey},
		func(channel AgentChannelRecord) []string {
			return []string{
				channel.ID,
				channel.Channel,
				stringOrDash(channel.Purpose),
				stringOrDash(channel.TaskID),
				stringOrDash(channel.RunID),
			}
		},
		func(channel AgentChannelRecord) []string {
			return []string{channel.ID, channel.Channel, channel.Purpose, channel.TaskID, channel.RunID}
		},
	)
}

func agentChannelMessagesBundle(messages []AgentChannelMessageRecord) outputBundle {
	return listBundle(
		messages,
		messages,
		"Agent Channel Messages",
		[]string{"ID", agentKernelChannelValue, bridgeKindValue, agentKernelFromValue, "Time"},
		"agent_channel_messages",
		[]string{"message_id", "channel_id", agentKernelKindKey, "from_session_id", networkTimestampKey},
		func(message AgentChannelMessageRecord) []string {
			return []string{
				message.MessageID,
				message.ChannelID,
				string(message.Metadata.MessageKind),
				message.FromSessionID,
				formatAgentTime(message.Timestamp),
			}
		},
		func(message AgentChannelMessageRecord) []string {
			return []string{
				message.MessageID,
				message.ChannelID,
				string(message.Metadata.MessageKind),
				message.FromSessionID,
				formatAgentTime(message.Timestamp),
			}
		},
	)
}

func agentChannelMessageBundle(message AgentChannelMessageRecord) outputBundle {
	return outputBundle{
		jsonValue: message,
		human: func() (string, error) {
			return renderHumanSection("Agent Channel Message", []keyValue{
				{Label: "ID", Value: message.MessageID},
				{Label: agentKernelChannelValue, Value: message.ChannelID},
				{Label: bridgeKindValue, Value: string(message.Metadata.MessageKind)},
				{Label: taskTaskValue, Value: stringOrDash(message.Metadata.TaskID)},
				{Label: taskRunValue, Value: stringOrDash(message.Metadata.RunID)},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("agent_channel_message", []string{
				"message_id", "channel_id", agentKernelKindKey, "task_id", agentKernelRunIDKey,
			}, []string{
				message.MessageID,
				message.ChannelID,
				string(message.Metadata.MessageKind),
				message.Metadata.TaskID,
				message.Metadata.RunID,
			}), nil
		},
	}
}

func renderJSONPreview(value any) (string, error) {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", fmt.Errorf("cli: render JSON preview: %w", err)
	}
	return string(content), nil
}

func formatAgentTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return strconv.FormatInt(value.UTC().Unix(), 10)
}

func firstCLIValue(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func zeroCLIAgentCoordinationMetadata(metadata contract.CoordinationMessageMetadataPayload) bool {
	return strings.TrimSpace(metadata.TaskID) == "" &&
		strings.TrimSpace(metadata.RunID) == "" &&
		strings.TrimSpace(metadata.WorkflowID) == "" &&
		strings.TrimSpace(metadata.CoordinationChannelID) == "" &&
		strings.TrimSpace(string(metadata.MessageKind)) == "" &&
		strings.TrimSpace(metadata.CorrelationID) == "" &&
		len(metadata.Ext) == 0
}
