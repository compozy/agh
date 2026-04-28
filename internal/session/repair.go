package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/transcript"
)

const (
	RepairSeverityInfo    = "info"
	RepairSeverityWarning = "warning"
	RepairSeverityError   = "error"

	RepairIssueSequenceGap                = "event_sequence_gap"
	RepairIssueSequenceDuplicate          = "event_sequence_duplicate"
	RepairIssueSequenceRegression         = "event_sequence_regression"
	RepairIssueInvalidEventJSON           = "invalid_event_json"
	RepairIssueEventTypeMismatch          = "event_type_mismatch"
	RepairIssueNoRepairableTurn           = "no_repairable_turn"
	RepairIssueSessionNotStopped          = "session_not_stopped"
	RepairIssueStopReasonRequiresForce    = "stop_reason_requires_force"
	RepairIssueDanglingToolCallMissingID  = "dangling_tool_call_missing_id"
	RepairIssueTerminalEventAlreadyExists = "terminal_event_already_exists"

	RepairActionAppendInterruptedToolResult = "append_interrupted_tool_result"
	RepairActionAppendTerminalError         = "append_terminal_error"

	repairInterruptedToolMessage = "Tool call interrupted before a result was persisted."
	repairTerminalErrorMessage   = "Session interrupted before a terminal prompt event was persisted."
)

// SessionRepairOpts controls one persisted session repair pass.
type SessionRepairOpts struct {
	SessionID string
	DryRun    bool
	Force     bool
}

// SessionRepairResult describes the detected inconsistencies and append-only
// repair events planned or persisted for a session.
type SessionRepairResult struct {
	SessionID string
	Issues    []SessionRepairIssue
	Actions   []SessionRepairAction
	Persisted bool
}

// SessionRepairIssue is one non-mutating diagnostic discovered during repair.
type SessionRepairIssue struct {
	Code     string
	Severity string
	TurnID   string
	EventID  string
	Detail   string
}

// SessionRepairAction is one append-only mutation planned or persisted by repair.
type SessionRepairAction struct {
	Code       string
	TurnID     string
	EventID    string
	ToolCallID string
	ToolName   string
	Persisted  bool
}

type repairEvent struct {
	stored store.SessionEvent
	agent  acp.AgentEvent
}

type repairTurnState struct {
	turnID        string
	hasPromptData bool
	terminal      bool
	toolCalls     map[string]repairToolCall
	toolResults   map[string]struct{}
}

type repairToolCall struct {
	toolName string
}

type repairAnalysis struct {
	issues []SessionRepairIssue
	turn   repairTurnState
	block  bool
}

// RepairSession inspects one persisted session transcript and, when safe,
// appends terminal repair events for an interrupted final prompt turn.
func (m *Manager) RepairSession(
	ctx context.Context,
	opts SessionRepairOpts,
) (result *SessionRepairResult, err error) {
	if ctx == nil {
		return nil, errors.New("session: repair context is required")
	}

	target, err := normalizeStoredSessionID(opts.SessionID)
	if err != nil {
		return nil, err
	}

	meta, err := m.readMetaWithContext(ctx, target)
	if err != nil {
		return nil, err
	}

	recorder, cleanup, err := m.openQueryRecorder(ctx, target)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil && err == nil {
			err = cleanupErr
		}
	}()

	events, err := recorder.Query(ctx, store.EventQuery{})
	if err != nil {
		return nil, fmt.Errorf("session: query events for repair %q: %w", target, err)
	}

	result = &SessionRepairResult{SessionID: target}
	analysis := analyzeRepairEvents(events)
	result.Issues = append(result.Issues, analysis.issues...)
	if analysis.block {
		return result, nil
	}
	if strings.TrimSpace(analysis.turn.turnID) == "" {
		result.Issues = append(result.Issues, SessionRepairIssue{
			Code:     RepairIssueNoRepairableTurn,
			Severity: RepairSeverityInfo,
			Detail:   "no prompt turn exists in the session event store",
		})
		return result, nil
	}
	if analysis.turn.terminal {
		result.Issues = append(result.Issues, SessionRepairIssue{
			Code:     RepairIssueTerminalEventAlreadyExists,
			Severity: RepairSeverityInfo,
			TurnID:   analysis.turn.turnID,
			Detail:   "the final prompt turn already has a terminal event",
		})
		return result, nil
	}

	if strings.TrimSpace(meta.State) != string(StateStopped) {
		result.Issues = append(result.Issues, SessionRepairIssue{
			Code:     RepairIssueSessionNotStopped,
			Severity: RepairSeverityError,
			TurnID:   analysis.turn.turnID,
			Detail:   "repair only mutates stopped sessions",
		})
		return result, nil
	}
	stopReason := sessionMetaStopReason(meta)
	if !opts.Force && !repairDefaultStopReason(stopReason) {
		result.Issues = append(result.Issues, SessionRepairIssue{
			Code:     RepairIssueStopReasonRequiresForce,
			Severity: RepairSeverityError,
			TurnID:   analysis.turn.turnID,
			Detail:   fmt.Sprintf("stop reason %q requires force before repair can mutate", stopReason),
		})
		return result, nil
	}

	actions := planRepairActions(analysis.turn)
	result.Actions = append(result.Actions, actions...)
	if opts.DryRun || len(actions) == 0 {
		return result, nil
	}

	persisted, err := m.persistRepairActions(ctx, recorder, meta, actions)
	if err != nil {
		return result, err
	}
	result.Actions = persisted
	result.Persisted = len(persisted) > 0
	return result, nil
}

func analyzeRepairEvents(events []store.SessionEvent) repairAnalysis {
	ordered := append([]store.SessionEvent(nil), events...)
	slices.SortStableFunc(ordered, func(a store.SessionEvent, b store.SessionEvent) int {
		switch {
		case a.Sequence < b.Sequence:
			return -1
		case a.Sequence > b.Sequence:
			return 1
		default:
			return strings.Compare(a.ID, b.ID)
		}
	})

	analysis := repairAnalysis{}
	turns := make(map[string]*repairTurnState)
	var lastTurnID string
	var previousSequence int64

	for _, event := range ordered {
		expected := previousSequence + 1
		switch {
		case previousSequence > 0 && event.Sequence == previousSequence:
			analysis.issues = append(analysis.issues, SessionRepairIssue{
				Code:     RepairIssueSequenceDuplicate,
				Severity: RepairSeverityError,
				EventID:  event.ID,
				Detail:   fmt.Sprintf("duplicate sequence %d", event.Sequence),
			})
			analysis.block = true
		case previousSequence > 0 && event.Sequence < previousSequence:
			analysis.issues = append(analysis.issues, SessionRepairIssue{
				Code:     RepairIssueSequenceRegression,
				Severity: RepairSeverityError,
				EventID:  event.ID,
				Detail:   fmt.Sprintf("sequence regressed from %d to %d", previousSequence, event.Sequence),
			})
			analysis.block = true
		case event.Sequence != expected:
			analysis.issues = append(analysis.issues, SessionRepairIssue{
				Code:     RepairIssueSequenceGap,
				Severity: RepairSeverityWarning,
				EventID:  event.ID,
				Detail:   fmt.Sprintf("expected sequence %d, found %d", expected, event.Sequence),
			})
		}
		if event.Sequence > previousSequence {
			previousSequence = event.Sequence
		}

		agentEvent, err := transcript.UnmarshalAgentEvent(event.Content)
		if err != nil {
			analysis.issues = append(analysis.issues, SessionRepairIssue{
				Code:     RepairIssueInvalidEventJSON,
				Severity: RepairSeverityError,
				TurnID:   strings.TrimSpace(event.TurnID),
				EventID:  event.ID,
				Detail:   err.Error(),
			})
			analysis.block = true
			continue
		}

		eventType := strings.TrimSpace(event.Type)
		decodedType := strings.TrimSpace(agentEvent.Type)
		if decodedType != "" && eventType != "" && decodedType != eventType {
			analysis.issues = append(analysis.issues, SessionRepairIssue{
				Code:     RepairIssueEventTypeMismatch,
				Severity: RepairSeverityError,
				TurnID:   strings.TrimSpace(event.TurnID),
				EventID:  event.ID,
				Detail:   fmt.Sprintf("stored type %q does not match payload type %q", eventType, decodedType),
			})
			analysis.block = true
		}
		if eventType == "" {
			eventType = decodedType
		}

		turnID := strings.TrimSpace(firstNonEmpty(event.TurnID, agentEvent.TurnID))
		if turnID == "" || eventType == EventTypeSessionStopped {
			continue
		}
		turn := ensureRepairTurn(turns, turnID)
		lastTurnID = turnID
		applyRepairEvent(turn, repairEvent{stored: event, agent: agentEvent}, eventType, &analysis)
	}

	if lastTurnID != "" {
		analysis.turn = *turns[lastTurnID]
	}
	return analysis
}

func ensureRepairTurn(turns map[string]*repairTurnState, turnID string) *repairTurnState {
	turn := turns[turnID]
	if turn != nil {
		return turn
	}
	turn = &repairTurnState{
		turnID:      turnID,
		toolCalls:   make(map[string]repairToolCall),
		toolResults: make(map[string]struct{}),
	}
	turns[turnID] = turn
	return turn
}

func applyRepairEvent(
	turn *repairTurnState,
	event repairEvent,
	eventType string,
	analysis *repairAnalysis,
) {
	switch eventType {
	case acp.EventTypeUserMessage,
		acp.EventTypeSyntheticReentry,
		acp.EventTypeAgentMessage,
		acp.EventTypeThought,
		acp.EventTypePlan,
		acp.EventTypeRuntimeProgress,
		acp.EventTypeRuntimeWarning,
		acp.EventTypePermission,
		acp.EventTypeSystem:
		turn.hasPromptData = true
	case acp.EventTypeToolCall:
		turn.hasPromptData = true
		toolCallID := strings.TrimSpace(event.agent.ToolCallID)
		if toolCallID == "" {
			analysis.issues = append(analysis.issues, SessionRepairIssue{
				Code:     RepairIssueDanglingToolCallMissingID,
				Severity: RepairSeverityWarning,
				TurnID:   turn.turnID,
				EventID:  event.stored.ID,
				Detail:   "tool_call event cannot be individually closed because tool_call_id is empty",
			})
			return
		}
		turn.toolCalls[toolCallID] = repairToolCall{
			toolName: strings.TrimSpace(event.agent.Title),
		}
	case acp.EventTypeToolResult:
		turn.hasPromptData = true
		toolCallID := strings.TrimSpace(event.agent.ToolCallID)
		if toolCallID != "" {
			turn.toolResults[toolCallID] = struct{}{}
		}
	case acp.EventTypeDone, acp.EventTypeError:
		turn.hasPromptData = true
		turn.terminal = true
	}
}

func repairDefaultStopReason(reason store.StopReason) bool {
	switch reason {
	case store.StopAgentCrashed, store.StopError:
		return true
	default:
		return false
	}
}

func planRepairActions(turn repairTurnState) []SessionRepairAction {
	actions := make([]SessionRepairAction, 0, len(turn.toolCalls)+1)
	toolCallIDs := make([]string, 0, len(turn.toolCalls))
	for toolCallID := range turn.toolCalls {
		if _, ok := turn.toolResults[toolCallID]; ok {
			continue
		}
		toolCallIDs = append(toolCallIDs, toolCallID)
	}
	slices.Sort(toolCallIDs)

	for _, toolCallID := range toolCallIDs {
		toolCall := turn.toolCalls[toolCallID]
		actions = append(actions, SessionRepairAction{
			Code:       RepairActionAppendInterruptedToolResult,
			TurnID:     turn.turnID,
			ToolCallID: toolCallID,
			ToolName:   toolCall.toolName,
		})
	}
	if turn.hasPromptData {
		actions = append(actions, SessionRepairAction{
			Code:   RepairActionAppendTerminalError,
			TurnID: turn.turnID,
		})
	}
	return actions
}

func (m *Manager) persistRepairActions(
	ctx context.Context,
	recorder EventRecorder,
	meta store.SessionMeta,
	actions []SessionRepairAction,
) ([]SessionRepairAction, error) {
	persisted := make([]SessionRepairAction, 0, len(actions))
	for _, action := range actions {
		event, err := m.repairActionEvent(meta, action)
		if err != nil {
			return persisted, err
		}
		content, err := marshalAgentEvent(event)
		if err != nil {
			return persisted, err
		}
		eventID := store.NewID("ev")
		if err := recorder.Record(ctx, store.SessionEvent{
			ID:        eventID,
			TurnID:    event.TurnID,
			Type:      event.Type,
			AgentName: strings.TrimSpace(meta.AgentName),
			Content:   content,
			Timestamp: event.Timestamp,
		}); err != nil {
			return persisted, fmt.Errorf("session: persist repair event for %q: %w", strings.TrimSpace(meta.ID), err)
		}

		action.EventID = eventID
		action.Persisted = true
		persisted = append(persisted, action)
		m.notifyRepairEvent(ctx, strings.TrimSpace(meta.ID), event)
	}
	return persisted, nil
}

func (m *Manager) repairActionEvent(meta store.SessionMeta, action SessionRepairAction) (acp.AgentEvent, error) {
	now := m.now().UTC()
	event := acp.AgentEvent{
		SessionID: repairACPSessionID(meta),
		TurnID:    strings.TrimSpace(action.TurnID),
		Timestamp: now,
	}

	switch action.Code {
	case RepairActionAppendInterruptedToolResult:
		raw, err := interruptedToolResultRaw(action.ToolCallID, action.ToolName)
		if err != nil {
			return acp.AgentEvent{}, err
		}
		event.Type = acp.EventTypeToolResult
		event.ToolCallID = strings.TrimSpace(action.ToolCallID)
		event.Title = firstNonEmpty(action.ToolName, "interrupted tool result")
		event.Error = repairInterruptedToolMessage
		event.Raw = raw
	case RepairActionAppendTerminalError:
		event.Type = acp.EventTypeError
		event.Error = repairTerminalErrorMessage
		event.StopReason = string(sessionMetaStopReason(meta))
		event.Failure = store.CloneSessionFailure(meta.Failure)
	default:
		return acp.AgentEvent{}, fmt.Errorf("session: unknown repair action %q", action.Code)
	}
	return event, nil
}

func interruptedToolResultRaw(toolCallID string, toolName string) (json.RawMessage, error) {
	metadata := map[string]any{
		"agh": map[string]any{
			"repair":   true,
			"toolName": strings.TrimSpace(toolName),
		},
	}
	payload := map[string]any{
		"sessionUpdate": "tool_call_update",
		"status":        "failed",
		"toolCallId":    strings.TrimSpace(toolCallID),
		"rawOutput": map[string]string{
			"stderr": repairInterruptedToolMessage,
			"error":  repairInterruptedToolMessage,
		},
		"content": []map[string]any{
			{
				"type": "content",
				"content": map[string]string{
					"type": "text",
					"text": repairInterruptedToolMessage,
				},
			},
		},
		"_meta": metadata,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("session: marshal interrupted tool result: %w", err)
	}
	return data, nil
}

func (m *Manager) notifyRepairEvent(ctx context.Context, sessionID string, event acp.AgentEvent) {
	if m == nil || m.notifier == nil {
		return
	}
	m.notifier.OnAgentEvent(ctx, sessionID, event)
}

func repairACPSessionID(meta store.SessionMeta) string {
	if meta.ACPSessionID == nil {
		return ""
	}
	return strings.TrimSpace(*meta.ACPSessionID)
}
