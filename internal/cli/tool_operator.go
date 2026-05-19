package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pedronauck/agh/internal/api/contract"
	toolspkg "github.com/pedronauck/agh/internal/tools"
	"github.com/spf13/cobra"
)

const (
	toolOperatorToolIDKey = "tool_id"
)

const (
	toolOperatorReasonsKey = "reasons"
)

const (
	toolOperatorBackendValue     = "Backend"
	toolOperatorExpiresValue     = "Expires"
	toolOperatorSourceValue      = "Source"
	toolOperatorStatusValue      = "Status"
	toolOperatorTitleValue       = "Title"
	toolOperatorToolIDValue      = "Tool ID"
	toolOperatorAvailableKey     = "available"
	toolOperatorBackendKey       = "backend"
	toolOperatorCLIKey           = "cli"
	toolOperatorDisabledKey      = "disabled"
	toolOperatorExpiresAtKey     = "expires_at"
	toolOperatorListKey          = "list"
	toolOperatorSearchQueryValue = "search <query>"
)

type toolScopeFlags struct {
	workspaceID string
	sessionID   string
	agentName   string
}

type toolInvokeFlags struct {
	scope                toolScopeFlags
	input                string
	inputFile            string
	toolCallID           string
	turnID               string
	correlationID        string
	approvalToken        string
	sensitiveInputFields []string
}

type toolApprovalFlags struct {
	scope       toolScopeFlags
	input       string
	inputFile   string
	inputDigest string
}

type toolCommandError struct {
	response ToolErrorResponseRecord
	err      error
}

func (e *toolCommandError) Error() string {
	if e == nil {
		return nilToolErrorString
	}
	if e.err != nil {
		return redactToolDiagnostic(e.err.Error())
	}
	apiErr := newToolAPIError(0, "", e.response)
	return apiErr.Error()
}

func newToolListCommand(deps commandDeps) *cobra.Command {
	var scope toolScopeFlags
	cmd := &cobra.Command{
		Use:   toolOperatorListKey,
		Short: "List operator-visible registry tools",
		Example: `  # List all operator-visible tools as JSON
  agh tool list -o json

  # Inspect the session-scoped operator view for one agent
  agh tool list --workspace ws-1 --session sess-1 --agent coder -o json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runToolCommand(cmd, deps, func(client DaemonClient) error {
				response, err := client.ListTools(cmd.Context(), scope.query())
				if err != nil {
					return err
				}
				return writeCommandOutput(cmd, toolListBundle(response))
			})
		},
	}
	scope.bind(cmd)
	return cmd
}

func newToolSearchCommand(deps commandDeps) *cobra.Command {
	var scope toolScopeFlags
	var limit int
	cmd := &cobra.Command{
		Use:   toolOperatorSearchQueryValue,
		Short: "Search operator-visible registry tools",
		Example: `  # Search tools by descriptor text
  agh tool search skill -o json

  # Limit search results for automation
  agh tool search task --limit 5 -o json`,
		Args: exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.TrimSpace(args[0])
			if query == "" {
				return writeToolCommandError(cmd, toolValidationCommandError(
					toolspkg.ToolID(""),
					"tool search query is required",
					toolspkg.NewValidationError("query", toolspkg.ReasonSchemaInvalid, "query is required"),
				))
			}
			return runToolCommand(cmd, deps, func(client DaemonClient) error {
				request := ToolSearchRequest{
					Query:       query,
					Limit:       limit,
					WorkspaceID: strings.TrimSpace(scope.workspaceID),
					SessionID:   strings.TrimSpace(scope.sessionID),
					AgentName:   strings.TrimSpace(scope.agentName),
				}
				response, err := client.SearchTools(cmd.Context(), request)
				if err != nil {
					return err
				}
				return writeCommandOutput(cmd, toolListBundle(response))
			})
		},
	}
	scope.bind(cmd)
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of tools to return")
	return cmd
}

func newToolInfoCommand(deps commandDeps) *cobra.Command {
	var scope toolScopeFlags
	cmd := &cobra.Command{
		Use:   "info <tool_id>",
		Short: "Show one registry tool descriptor and diagnostics",
		Example: `  # Show a tool descriptor and availability diagnostics
  agh tool info agh__skill_view -o json`,
		Args: exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseToolIDArg(args[0])
			if err != nil {
				return writeToolCommandError(cmd, err)
			}
			return runToolCommand(cmd, deps, func(client DaemonClient) error {
				response, err := client.GetTool(cmd.Context(), id.String(), scope.query())
				if err != nil {
					return err
				}
				return writeCommandOutput(cmd, toolInfoBundle(&response))
			})
		},
	}
	scope.bind(cmd)
	return cmd
}

func newToolApproveCommand(deps commandDeps) *cobra.Command {
	var flags toolApprovalFlags
	cmd := &cobra.Command{
		Use:   "approve <tool_id>",
		Short: "Mint a one-shot approval token for one tool invocation",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseToolIDArg(args[0])
			if err != nil {
				return writeToolCommandError(cmd, err)
			}
			input, err := resolveToolApprovalInput(cmd, flags)
			if err != nil {
				return writeToolCommandError(cmd, toolValidationCommandError(id, "tool approval input is invalid", err))
			}
			if strings.TrimSpace(flags.scope.sessionID) == "" {
				return writeToolCommandError(cmd, toolValidationCommandError(
					id,
					"tool approval scope is invalid",
					toolspkg.NewValidationError(
						"session_id",
						toolspkg.ReasonSchemaInvalid,
						"session id is required",
					),
				))
			}
			return runToolCommand(cmd, deps, func(client DaemonClient) error {
				request := ToolApprovalRequest{
					SessionID:   strings.TrimSpace(flags.scope.sessionID),
					WorkspaceID: strings.TrimSpace(flags.scope.workspaceID),
					AgentName:   strings.TrimSpace(flags.scope.agentName),
					Input:       input,
					InputDigest: strings.TrimSpace(flags.inputDigest),
				}
				approval, err := client.CreateToolApproval(cmd.Context(), id.String(), request)
				if err != nil {
					return err
				}
				return writeCommandOutput(cmd, toolApprovalBundle(approval))
			})
		},
	}
	flags.scope.bind(cmd)
	cmd.Flags().StringVar(&flags.input, "input", "", "Inline JSON input")
	cmd.Flags().StringVar(&flags.inputFile, "input-file", "", "Path to JSON input file, or '-' for stdin")
	cmd.Flags().StringVar(&flags.inputDigest, "input-digest", "", "Precomputed input digest")
	mustMarkFlagRequired(cmd, "session")
	return cmd
}

func newToolInvokeCommand(deps commandDeps) *cobra.Command {
	var flags toolInvokeFlags
	cmd := &cobra.Command{
		Use:   "invoke <tool_id>",
		Short: "Invoke one registry tool through daemon policy",
		Example: `  # Invoke a tool with inline JSON input
  agh tool invoke agh__tool_info --input '{"tool_id":"agh__skill_view"}' -o json

  # Invoke a tool with JSON read from a file
  agh tool invoke agh__tool_info --input-file ./input.json -o json

  # Invoke a tool with JSON read from stdin
  echo '{"tool_id":"agh__skill_view"}' | agh tool invoke agh__tool_info -o json`,
		Args: exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseToolIDArg(args[0])
			if err != nil {
				return writeToolCommandError(cmd, err)
			}
			input, err := resolveToolInvokeInput(cmd, flags)
			if err != nil {
				return writeToolCommandError(cmd, toolValidationCommandError(id, "tool input is invalid", err))
			}
			return runToolCommand(cmd, deps, func(client DaemonClient) error {
				request := ToolInvokeRequest{
					SessionID:            strings.TrimSpace(flags.scope.sessionID),
					WorkspaceID:          strings.TrimSpace(flags.scope.workspaceID),
					AgentName:            strings.TrimSpace(flags.scope.agentName),
					ToolCallID:           strings.TrimSpace(flags.toolCallID),
					TurnID:               strings.TrimSpace(flags.turnID),
					CorrelationID:        strings.TrimSpace(flags.correlationID),
					ApprovalToken:        strings.TrimSpace(flags.approvalToken),
					Input:                input,
					SensitiveInputFields: trimNonEmptyStrings(flags.sensitiveInputFields),
				}
				response, err := client.InvokeTool(cmd.Context(), id.String(), request)
				if err != nil {
					return err
				}
				return writeCommandOutput(cmd, toolInvokeBundle(sanitizeToolInvokeResponse(response)))
			})
		},
	}
	flags.scope.bind(cmd)
	cmd.Flags().StringVar(&flags.input, "input", "", "Inline JSON input")
	cmd.Flags().StringVar(&flags.inputFile, "input-file", "", "Path to JSON input file, or '-' for stdin")
	cmd.Flags().StringVar(&flags.toolCallID, "tool-call-id", "", "Optional caller tool-call id")
	cmd.Flags().StringVar(&flags.turnID, "turn-id", "", "Optional caller turn id")
	cmd.Flags().StringVar(&flags.correlationID, "correlation-id", "", "Optional correlation id")
	cmd.Flags().
		StringVar(&flags.approvalToken, "approval-token", "", "Single-use approval token for approval-gated tools")
	cmd.Flags().
		StringArrayVar(&flags.sensitiveInputFields, "sensitive-input-field", nil, "Input field path to redact in events")
	return cmd
}

func newToolsetsCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "toolsets",
		Short: "Inspect registry toolsets",
	}
	cmd.AddCommand(newToolsetsListCommand(deps))
	cmd.AddCommand(newToolsetsInfoCommand(deps))
	return cmd
}

func newToolsetsListCommand(deps commandDeps) *cobra.Command {
	var scope toolScopeFlags
	cmd := &cobra.Command{
		Use:   toolOperatorListKey,
		Short: "List registry toolsets",
		Example: `  # List known toolsets and expansion diagnostics
  agh toolsets list -o json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runToolCommand(cmd, deps, func(client DaemonClient) error {
				response, err := client.ListToolsets(cmd.Context(), scope.query())
				if err != nil {
					return err
				}
				return writeCommandOutput(cmd, toolsetListBundle(response))
			})
		},
	}
	scope.bind(cmd)
	return cmd
}

func newToolsetsInfoCommand(deps commandDeps) *cobra.Command {
	var scope toolScopeFlags
	cmd := &cobra.Command{
		Use:   "info <toolset_id>",
		Short: "Show one registry toolset expansion",
		Example: `  # Show one toolset and expanded tool ids
  agh toolsets info agh__catalog -o json`,
		Args: exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseToolsetIDArg(args[0])
			if err != nil {
				return writeToolCommandError(cmd, err)
			}
			return runToolCommand(cmd, deps, func(client DaemonClient) error {
				response, err := client.GetToolset(cmd.Context(), id.String(), scope.query())
				if err != nil {
					return err
				}
				return writeCommandOutput(cmd, toolsetInfoBundle(response))
			})
		},
	}
	scope.bind(cmd)
	return cmd
}

func (f *toolScopeFlags) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.workspaceID, "workspace", "", "Workspace id for scoped diagnostics")
	cmd.Flags().StringVar(&f.sessionID, "session", "", "Session id for scoped diagnostics")
	cmd.Flags().StringVar(&f.agentName, "agent", "", "Agent name for scoped diagnostics")
}

func (f toolScopeFlags) query() ToolQuery {
	return ToolQuery{
		WorkspaceID: strings.TrimSpace(f.workspaceID),
		SessionID:   strings.TrimSpace(f.sessionID),
		AgentName:   strings.TrimSpace(f.agentName),
	}
}

func runToolCommand(cmd *cobra.Command, deps commandDeps, run func(DaemonClient) error) error {
	client, err := clientFromDeps(deps)
	if err != nil {
		return err
	}
	err = run(client)
	if err != nil {
		return writeToolCommandError(cmd, err)
	}
	return nil
}

func parseToolIDArg(raw string) (toolspkg.ToolID, error) {
	id := toolspkg.ToolID(strings.TrimSpace(raw))
	if err := id.Validate(); err != nil {
		return "", toolValidationCommandError("", "tool id is invalid", err)
	}
	return id, nil
}

func parseToolsetIDArg(raw string) (toolspkg.ToolsetID, error) {
	id := toolspkg.ToolsetID(strings.TrimSpace(raw))
	if err := id.Validate(); err != nil {
		return "", toolValidationCommandError("", "toolset id is invalid", err)
	}
	return id, nil
}

func resolveToolInvokeInput(cmd *cobra.Command, flags toolInvokeFlags) (json.RawMessage, error) {
	inlineChanged := cmd.Flags().Lookup("input") != nil && cmd.Flags().Lookup("input").Changed
	inputFile := strings.TrimSpace(flags.inputFile)
	if inlineChanged && inputFile != "" {
		return nil, toolspkg.NewValidationError(
			"input",
			toolspkg.ReasonSchemaInvalid,
			"provide --input or --input-file, not both",
		)
	}
	if inlineChanged {
		return parseToolInputJSON("input", flags.input)
	}
	if inputFile != "" {
		return readToolInputFile(cmd, inputFile)
	}
	stdinContent, err := readOptionalCommandInput(cmd.InOrStdin())
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(stdinContent) != "" {
		return parseToolInputJSON("stdin", stdinContent)
	}
	return json.RawMessage(`{}`), nil
}

func resolveToolApprovalInput(cmd *cobra.Command, flags toolApprovalFlags) (json.RawMessage, error) {
	inlineChanged := cmd.Flags().Lookup("input") != nil && cmd.Flags().Lookup("input").Changed
	inputFile := strings.TrimSpace(flags.inputFile)
	if inlineChanged && inputFile != "" {
		return nil, toolspkg.NewValidationError(
			"input",
			toolspkg.ReasonSchemaInvalid,
			"provide --input or --input-file, not both",
		)
	}
	if inlineChanged {
		return parseToolInputJSON("input", flags.input)
	}
	if inputFile != "" {
		return readToolInputFile(cmd, inputFile)
	}
	return nil, nil
}

func readToolInputFile(cmd *cobra.Command, path string) (json.RawMessage, error) {
	var payload []byte
	var err error
	if path == "-" {
		payload, err = io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return nil, fmt.Errorf("cli: read tool input stdin: %w", err)
		}
	} else {
		payload, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("cli: read tool input file: %w", err)
		}
	}
	return parseToolInputJSON("input", string(payload))
}

func parseToolInputJSON(field string, raw string) (json.RawMessage, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, toolspkg.NewValidationError(field, toolspkg.ReasonSchemaInvalid, "input JSON is required")
	}
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, []byte(trimmed)); err != nil {
		return nil, toolspkg.NewValidationError(field, toolspkg.ReasonSchemaInvalid, "input must be valid JSON")
	}
	return json.RawMessage(compacted.String()), nil
}

func writeToolCommandError(cmd *cobra.Command, err error) error {
	if err == nil {
		return nil
	}
	response, ok := toolErrorResponseForError(err)
	if !ok {
		return err
	}
	mode, modeErr := resolveOutputFormat(cmd)
	if modeErr != nil {
		return modeErr
	}
	if mode == OutputJSON {
		if writeErr := writeJSON(cmd, response); writeErr != nil {
			return errors.Join(err, writeErr)
		}
	}
	return err
}

func toolErrorResponseForError(err error) (ToolErrorResponseRecord, bool) {
	if apiErr, ok := errors.AsType[*toolAPIError](err); ok {
		return apiErr.Response(), true
	}
	if commandErr, ok := errors.AsType[*toolCommandError](err); ok {
		return sanitizeToolErrorResponse(commandErr.response), true
	}
	if toolErr, ok := errors.AsType[*toolspkg.ToolError](err); ok {
		return sanitizeToolErrorResponse(ToolErrorResponseRecord{
			Error: contract.ToolErrorPayload{
				Code:        toolErr.Code,
				Message:     toolErr.Error(),
				ToolID:      toolErr.ToolID,
				ReasonCodes: append([]toolspkg.ReasonCode(nil), toolErr.ReasonCodes...),
				Layer:       toolOperatorCLIKey,
			},
		}), true
	}
	if validationErr, ok := errors.AsType[*toolspkg.ValidationError](err); ok {
		return sanitizeToolErrorResponse(ToolErrorResponseRecord{
			Error: contract.ToolErrorPayload{
				Code:        toolspkg.ErrorCodeInvalidInput,
				Message:     validationErr.Error(),
				ReasonCodes: []toolspkg.ReasonCode{validationErr.Reason},
				Layer:       toolOperatorCLIKey,
			},
		}), true
	}
	return ToolErrorResponseRecord{}, false
}

func toolValidationCommandError(
	id toolspkg.ToolID,
	message string,
	err error,
) *toolCommandError {
	reason := toolspkg.ReasonSchemaInvalid
	if extracted, ok := toolspkg.ReasonOf(err); ok {
		reason = extracted
	}
	payload := contract.ToolErrorPayload{
		Code:        toolspkg.ErrorCodeInvalidInput,
		Message:     message,
		ReasonCodes: []toolspkg.ReasonCode{reason},
		Layer:       toolOperatorCLIKey,
	}
	if id != "" {
		payload.ToolID = id
	}
	return &toolCommandError{
		response: ToolErrorResponseRecord{Error: payload},
		err:      fmt.Errorf("%s: %w", message, err),
	}
}

func toolListBundle(response ToolsResponseRecord) outputBundle {
	return listBundle(
		response,
		response.Tools,
		toolOperatorToolsValue,
		[]string{"TOOL ID", "BACKEND", "SOURCE", "STATUS", "CALLABLE", "REASONS"},
		"tools",
		[]string{
			toolOperatorToolIDKey,
			toolOperatorBackendKey,
			automationSourceKey,
			automationStatusKey,
			"callable",
			toolOperatorReasonsKey,
		},
		func(item ToolRecord) []string {
			return []string{
				item.Descriptor.ToolID.String(),
				string(item.Descriptor.Backend.Kind),
				toolSourceSummary(item.Descriptor.Source),
				toolAvailabilitySummary(item.Availability),
				formatBool(item.Decision.Callable),
				joinReasons(item.Availability.ReasonCodes, item.Decision.ReasonCodes),
			}
		},
		func(item ToolRecord) []string {
			return []string{
				item.Descriptor.ToolID.String(),
				string(item.Descriptor.Backend.Kind),
				toolSourceSummary(item.Descriptor.Source),
				toolAvailabilitySummary(item.Availability),
				formatBool(item.Decision.Callable),
				joinReasons(item.Availability.ReasonCodes, item.Decision.ReasonCodes),
			}
		},
	)
}

func toolInfoBundle(response *ToolResponseRecord) outputBundle {
	tool := response.Tool
	return outputBundle{
		jsonValue: response,
		human: func() (string, error) {
			rows := []keyValue{
				{Label: toolOperatorToolIDValue, Value: tool.Descriptor.ToolID.String()},
				{Label: toolOperatorTitleValue, Value: stringOrDash(tool.Descriptor.DisplayTitle)},
				{Label: toolOperatorBackendValue, Value: string(tool.Descriptor.Backend.Kind)},
				{Label: toolOperatorSourceValue, Value: toolSourceSummary(tool.Descriptor.Source)},
				{Label: "Risk", Value: string(tool.Descriptor.Risk)},
				{Label: "Visibility", Value: string(tool.Descriptor.Visibility)},
				{Label: toolOperatorStatusValue, Value: toolAvailabilitySummary(tool.Availability)},
				{Label: "Callable", Value: formatBool(tool.Decision.Callable)},
				{Label: "Approval Required", Value: formatBool(tool.Decision.ApprovalRequired)},
				{
					Label: "Reasons",
					Value: stringOrDash(joinReasons(tool.Availability.ReasonCodes, tool.Decision.ReasonCodes)),
				},
				{Label: "Input Schema", Value: stringOrDash(compactJSON(tool.Descriptor.InputSchema))},
			}
			if output := compactJSON(tool.Descriptor.OutputSchema); output != "" {
				rows = append(rows, keyValue{Label: "Output Schema", Value: output})
			}
			return renderHumanSection("Tool", rows), nil
		},
		toon: func() (string, error) {
			return renderToonObject(
				"tool",
				[]string{
					toolOperatorToolIDKey,
					toolOperatorBackendKey,
					automationSourceKey,
					automationStatusKey,
					"callable",
					toolOperatorReasonsKey,
				},
				[]string{
					tool.Descriptor.ToolID.String(),
					string(tool.Descriptor.Backend.Kind),
					toolSourceSummary(tool.Descriptor.Source),
					toolAvailabilitySummary(tool.Availability),
					formatBool(tool.Decision.Callable),
					joinReasons(tool.Availability.ReasonCodes, tool.Decision.ReasonCodes),
				},
			), nil
		},
	}
}

func toolApprovalBundle(response ToolApprovalRecord) outputBundle {
	return outputBundle{
		jsonValue: struct {
			Approval ToolApprovalRecord `json:"approval"`
		}{Approval: response},
		human: func() (string, error) {
			return renderHumanSection("Tool Approval", []keyValue{
				{Label: toolOperatorToolIDValue, Value: response.ToolID.String()},
				{Label: "Approval Token", Value: stringOrDash(response.ApprovalToken)},
				{Label: "Input Digest", Value: stringOrDash(response.InputDigest)},
				{Label: toolOperatorExpiresValue, Value: stringOrDash(formatTime(response.ExpiresAt))},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject(
				"tool_approval",
				[]string{toolOperatorToolIDKey, "approval_token", "input_digest", toolOperatorExpiresAtKey},
				[]string{
					response.ToolID.String(),
					response.ApprovalToken,
					response.InputDigest,
					formatTime(response.ExpiresAt),
				},
			), nil
		},
	}
}

func toolInvokeBundle(response ToolInvokeResponseRecord) outputBundle {
	return outputBundle{
		jsonValue: response,
		human: func() (string, error) {
			rows := []keyValue{
				{Label: toolOperatorToolIDValue, Value: response.ToolID.String()},
				{Label: toolOperatorStatusValue, Value: stringOrDash(response.Status)},
				{Label: "Truncated", Value: formatBool(response.Truncated)},
				{Label: "Duration", Value: fmt.Sprintf("%dms", response.DurationMS)},
				{Label: "Bytes", Value: fmt.Sprintf("%d", response.Result.Bytes)},
			}
			if preview := strings.TrimSpace(response.Result.Preview); preview != "" {
				rows = append(rows, keyValue{Label: toolOperatorPreviewValue, Value: preview})
			}
			if len(response.Result.Redactions) > 0 {
				rows = append(
					rows,
					keyValue{Label: "Redactions", Value: fmt.Sprintf("%d", len(response.Result.Redactions))},
				)
			}
			return renderHumanSection("Tool Invocation", rows), nil
		},
		toon: func() (string, error) {
			return renderToonObject(
				"tool_invocation",
				[]string{toolOperatorToolIDKey, automationStatusKey, "truncated", "duration_ms", "bytes"},
				[]string{
					response.ToolID.String(),
					response.Status,
					formatBool(response.Truncated),
					fmt.Sprintf("%d", response.DurationMS),
					fmt.Sprintf("%d", response.Result.Bytes),
				},
			), nil
		},
	}
}

func toolsetListBundle(response ToolsetsResponseRecord) outputBundle {
	return listBundle(
		response,
		response.Toolsets,
		"Toolsets",
		[]string{"TOOLSET ID", "STATUS", "EXPANDED TOOLS", "REASONS"},
		"toolsets",
		[]string{"id", automationStatusKey, "expanded_tools", toolOperatorReasonsKey},
		func(item ToolsetRecord) []string {
			return []string{
				item.ID.String(),
				stringOrDash(item.Status),
				strings.Join(toolIDsToStrings(item.ExpandedTools), ","),
				joinReasons(item.ReasonCodes),
			}
		},
		func(item ToolsetRecord) []string {
			return []string{
				item.ID.String(),
				stringOrDash(item.Status),
				strings.Join(toolIDsToStrings(item.ExpandedTools), ","),
				joinReasons(item.ReasonCodes),
			}
		},
	)
}

func toolsetInfoBundle(response ToolsetResponseRecord) outputBundle {
	toolset := response.Toolset
	return outputBundle{
		jsonValue: response,
		human: func() (string, error) {
			rows := []keyValue{
				{Label: "Toolset ID", Value: toolset.ID.String()},
				{Label: toolOperatorStatusValue, Value: stringOrDash(toolset.Status)},
				{Label: toolOperatorToolsValue, Value: stringOrDash(strings.Join(toolset.Tools, ","))},
				{
					Label: "Nested Toolsets",
					Value: stringOrDash(strings.Join(toolsetIDsToStrings(toolset.Toolsets), ",")),
				},
				{
					Label: "Expanded Tools",
					Value: stringOrDash(strings.Join(toolIDsToStrings(toolset.ExpandedTools), ",")),
				},
				{Label: "Reasons", Value: stringOrDash(joinReasons(toolset.ReasonCodes))},
			}
			return renderHumanSection("Toolset", rows), nil
		},
		toon: func() (string, error) {
			return renderToonObject(
				"toolset",
				[]string{"id", automationStatusKey, "expanded_tools", toolOperatorReasonsKey},
				[]string{
					toolset.ID.String(),
					toolset.Status,
					strings.Join(toolIDsToStrings(toolset.ExpandedTools), ","),
					joinReasons(toolset.ReasonCodes),
				},
			), nil
		},
	}
}

func toolSourceSummary(source contract.ToolSourceRefPayload) string {
	owner := strings.TrimSpace(source.Owner)
	if owner == "" {
		return string(source.Kind)
	}
	return string(source.Kind) + ":" + owner
}

func toolAvailabilitySummary(availability contract.ToolAvailabilityPayload) string {
	switch {
	case availability.Conflicted:
		return "conflicted"
	case !availability.Registered:
		return "unregistered"
	case !availability.Enabled:
		return toolOperatorDisabledKey
	case !availability.Available:
		return "unavailable"
	case !availability.Authorized:
		return "auth-required"
	case !availability.Executable:
		return "not-executable"
	default:
		return toolOperatorAvailableKey
	}
}

func joinReasons(groups ...[]toolspkg.ReasonCode) string {
	seen := make(map[toolspkg.ReasonCode]struct{})
	values := make([]string, 0)
	for _, group := range groups {
		for _, reason := range group {
			if reason == "" {
				continue
			}
			if _, ok := seen[reason]; ok {
				continue
			}
			seen[reason] = struct{}{}
			values = append(values, string(reason))
		}
	}
	return strings.Join(values, ",")
}

func toolIDsToStrings(ids []toolspkg.ToolID) []string {
	values := make([]string, 0, len(ids))
	for _, id := range ids {
		values = append(values, id.String())
	}
	return values
}

func toolsetIDsToStrings(ids []toolspkg.ToolsetID) []string {
	values := make([]string, 0, len(ids))
	for _, id := range ids {
		values = append(values, id.String())
	}
	return values
}

func formatBool(value bool) string {
	if value {
		return toolBoolTrue
	}
	return toolBoolFalse
}

const (
	toolBoolFalse = "false"
)
