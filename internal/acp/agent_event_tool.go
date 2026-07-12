package acp

import "encoding/json"

type agentToolPayload struct {
	name        string
	input       json.RawMessage
	failed      bool
	errorDetail string
}

func newAgentToolPayload(
	name string,
	input json.RawMessage,
	failed bool,
	errorDetail string,
) *agentToolPayload {
	if name == "" && len(input) == 0 && !failed && errorDetail == "" {
		return nil
	}
	return &agentToolPayload{
		name:        name,
		input:       CloneRawMessage(input),
		failed:      failed || errorDetail != "",
		errorDetail: errorDetail,
	}
}

// WithTool returns an event carrying an isolated optional tool payload.
func (e AgentEvent) WithTool(name string, input json.RawMessage, failed bool) AgentEvent {
	return e.WithToolDetail(name, input, failed, "")
}

// WithToolDetail returns an event carrying isolated tool metadata and an optional result error.
func (e AgentEvent) WithToolDetail(
	name string,
	input json.RawMessage,
	failed bool,
	errorDetail string,
) AgentEvent {
	e.tool = newAgentToolPayload(name, input, failed, errorDetail)
	return e
}

// ToolName returns the optional tool name.
func (e AgentEvent) ToolName() string {
	if e.tool == nil {
		return ""
	}
	return e.tool.name
}

// ToolInput returns an isolated copy of the optional tool input.
func (e AgentEvent) ToolInput() json.RawMessage {
	if e.tool == nil {
		return nil
	}
	return CloneRawMessage(e.tool.input)
}

// ToolError reports whether the optional tool result failed.
func (e AgentEvent) ToolError() bool {
	return e.tool != nil && e.tool.failed
}

// ToolErrorDetail returns the optional tool-result failure detail.
func (e AgentEvent) ToolErrorDetail() string {
	if e.tool == nil {
		return ""
	}
	return e.tool.errorDetail
}

// HasToolPayload reports whether typed tool metadata is present on the event.
func (e AgentEvent) HasToolPayload() bool {
	return e.tool != nil
}
