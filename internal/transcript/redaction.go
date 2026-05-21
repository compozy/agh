package transcript

import (
	"encoding/json"
	"strings"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/diagnostics"
	"github.com/pedronauck/agh/internal/store"
)

// RedactAgentEvent removes displayable secret material before an ACP event is
// stored, replayed, or streamed to a caller.
func RedactAgentEvent(event acp.AgentEvent) acp.AgentEvent {
	redacted := event
	redacted.Text = redactDisplayString(event.Text)
	redacted.Title = redactDisplayString(event.Title)
	redacted.ToolCallID = redactDisplayString(event.ToolCallID)
	redacted.StopReason = redactDisplayString(event.StopReason)
	redacted.Action = redactDisplayString(event.Action)
	redacted.Resource = redactDisplayString(event.Resource)
	redacted.Decision = redactDisplayString(event.Decision)
	redacted.Error = redactDisplayString(event.Error)
	redacted.Failure = redactSessionFailure(event.Failure)
	redacted.Synthetic = redactPromptSyntheticMeta(event.Synthetic)
	redacted.Runtime = redactRuntimeActivity(event.Runtime)
	redacted.Raw = redactRawMessage(event.Raw)
	return redacted
}

func redactCanonicalPayload(payload *canonicalEventPayload) {
	if payload == nil {
		return
	}
	payload.Text = redactDisplayString(payload.Text)
	payload.Title = redactDisplayString(payload.Title)
	payload.ToolName = redactDisplayString(payload.ToolName)
	payload.ToolCallID = redactDisplayString(payload.ToolCallID)
	payload.ToolInput = redactRawMessage(payload.ToolInput)
	payload.ToolResult = redactTranscriptToolResult(payload.ToolResult)
	payload.StopReason = redactDisplayString(payload.StopReason)
	payload.Action = redactDisplayString(payload.Action)
	payload.Resource = redactDisplayString(payload.Resource)
	payload.Decision = redactDisplayString(payload.Decision)
	payload.Error = redactDisplayString(payload.Error)
	payload.Failure = redactSessionFailure(payload.Failure)
	payload.Synthetic = redactPromptSyntheticMeta(payload.Synthetic)
	payload.Runtime = redactRuntimeActivity(payload.Runtime)
	payload.Raw = redactRawMessage(payload.Raw)
}

func redactTranscriptEvent(parsed event) event {
	parsed.Text = redactDisplayString(parsed.Text)
	parsed.StopReason = redactDisplayString(parsed.StopReason)
	parsed.Error = redactDisplayString(parsed.Error)
	parsed.Failure = redactSessionFailure(parsed.Failure)
	parsed.Runtime = redactRuntimeActivity(parsed.Runtime)
	parsed.Marker = redactMarker(parsed.Marker)
	parsed.ToolCallID = redactDisplayString(parsed.ToolCallID)
	parsed.ToolName = redactDisplayString(parsed.ToolName)
	parsed.ToolInput = redactRawMessage(parsed.ToolInput)
	parsed.ToolResult = redactTranscriptToolResult(parsed.ToolResult)
	return parsed
}

func redactMarker(marker *Marker) *Marker {
	if marker == nil {
		return nil
	}
	redacted := marker.Normalize()
	return &redacted
}

func redactTranscriptToolResult(result *ToolResult) *ToolResult {
	if result == nil {
		return nil
	}
	redacted := cloneToolResult(result)
	redacted.Stdout = redactDisplayString(redacted.Stdout)
	redacted.Stderr = redactDisplayString(redacted.Stderr)
	redacted.FilePath = redactDisplayString(redacted.FilePath)
	redacted.Content = redactDisplayString(redacted.Content)
	redacted.StructuredPatch = redactRawMessage(redacted.StructuredPatch)
	redacted.Error = redactDisplayString(redacted.Error)
	redacted.RawOutput = redactRawMessage(redacted.RawOutput)
	return redacted
}

func redactSessionFailure(failure *store.SessionFailure) *store.SessionFailure {
	if failure == nil {
		return nil
	}
	redacted := failure.Normalize()
	redacted.Summary = redactDisplayString(redacted.Summary)
	redacted.CrashBundlePath = redactDisplayString(redacted.CrashBundlePath)
	return &redacted
}

func redactPromptSyntheticMeta(meta *acp.PromptSyntheticMeta) *acp.PromptSyntheticMeta {
	if meta == nil {
		return nil
	}
	redacted := meta.Normalize()
	redacted.TaskID = redactDisplayString(redacted.TaskID)
	redacted.TaskRunID = redactDisplayString(redacted.TaskRunID)
	redacted.WorkflowID = redactDisplayString(redacted.WorkflowID)
	redacted.CoordinatorSessionID = redactDisplayString(redacted.CoordinatorSessionID)
	redacted.Reason = redactDisplayString(redacted.Reason)
	redacted.Summary = redactDisplayString(redacted.Summary)
	redacted.WakeEventID = redactDisplayString(redacted.WakeEventID)
	return &redacted
}

func redactRuntimeActivity(activity *acp.RuntimeActivity) *acp.RuntimeActivity {
	redacted := cloneRuntimeActivity(activity)
	if redacted == nil {
		return nil
	}
	redacted.TurnID = redactDisplayString(redacted.TurnID)
	redacted.TurnSource = redactDisplayString(redacted.TurnSource)
	redacted.LastActivityKind = redactDisplayString(redacted.LastActivityKind)
	redacted.LastActivityDetail = redactDisplayString(redacted.LastActivityDetail)
	redacted.CurrentTool = redactDisplayString(redacted.CurrentTool)
	redacted.ToolCallID = redactDisplayString(redacted.ToolCallID)
	return redacted
}

func redactRawMessage(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err == nil {
		changed, redactedValue := redactJSONDisplayValue(value)
		if !changed {
			return acp.CloneRawMessage(raw)
		}
		data, err := json.Marshal(redactedValue)
		if err == nil {
			return json.RawMessage(data)
		}
	}
	redacted := diagnostics.Redact(string(raw))
	if json.Valid([]byte(redacted)) {
		return acp.CloneRawMessage(json.RawMessage(redacted))
	}
	return rawMessageFromValue(redacted)
}

func redactJSONDisplayValue(value any) (bool, any) {
	switch typed := value.(type) {
	case map[string]any:
		changed := false
		for key, child := range typed {
			if sensitiveDisplayJSONField(key) {
				typed[key] = "[REDACTED]"
				changed = true
				continue
			}
			childChanged, redactedChild := redactJSONDisplayValue(child)
			if childChanged {
				typed[key] = redactedChild
				changed = true
			}
		}
		return changed, typed
	case []any:
		changed := false
		for i, child := range typed {
			childChanged, redactedChild := redactJSONDisplayValue(child)
			if childChanged {
				typed[i] = redactedChild
				changed = true
			}
		}
		return changed, typed
	case string:
		redacted := redactDisplayString(typed)
		return redacted != typed, redacted
	default:
		return false, value
	}
}

func sensitiveDisplayJSONField(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(key), "-", "_"))
	if normalized == "token_present" {
		return false
	}
	for _, field := range []string{
		"api_key",
		"access_token",
		"refresh_token",
		"mcp_auth_token",
		"claim_token",
		"lease_token",
		"bot_token",
		"oauth_code",
		"authorization_code",
		"client_secret",
		"webhook_secret",
		"code_verifier",
		"pkce_verifier",
		"secret_binding",
		"authorization",
		"password",
		"token",
		"secret",
	} {
		if strings.Contains(normalized, field) {
			return true
		}
	}
	return false
}

func redactDisplayString(value string) string {
	return diagnostics.Redact(value)
}
