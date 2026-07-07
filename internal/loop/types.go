// Package loop owns loop-domain validation and compile surfaces.
package loop

import (
	"encoding/json"

	"github.com/compozy/agh/internal/loop/dsl"
)

const (
	// LoopMaxFanoutWidth is the absolute structural fan-out width ceiling.
	LoopMaxFanoutWidth = 64
	// LoopMaxGateRevisions is the absolute structural gate revision ceiling.
	LoopMaxGateRevisions = dsl.GateMaxRevisionsCeiling
	// LoopMaxNoProgressWindow is the compile-time generation no-progress ceiling.
	LoopMaxNoProgressWindow = 30
	// LoopFailureBreakerLimit is the compile-time consecutive-failure breaker ceiling.
	LoopFailureBreakerLimit = 2
	// LoopMaxAncestryDepth is the maximum run-loop parent chain depth.
	LoopMaxAncestryDepth = 8
)

// Linter validates loop definitions without performing IO.
type Linter interface {
	Lint(def dsl.Definition) []LintError
}

// LintSeverity classifies a lint result.
type LintSeverity string

const (
	// SeverityError blocks publish/compile.
	SeverityError LintSeverity = "error"
	// SeverityWarning is diagnostic-only.
	SeverityWarning LintSeverity = "warning"
)

// LintError is the per-node shape surfaced to authoring clients.
type LintError struct {
	NodeID   dsl.NodeID   `json:"node_id,omitempty"`
	Code     string       `json:"code"`
	Message  string       `json:"message"`
	Severity LintSeverity `json:"severity"`
}

const (
	// CodeCycle reports a graph cycle.
	CodeCycle = "cycle"
	// CodeUnreachableNode reports a node unreachable from source roots.
	CodeUnreachableNode = "unreachable_node"
	// CodeNonTerminatingStructure reports a body with no terminal path.
	CodeNonTerminatingStructure = "non_terminating_structure"
	// CodeFanOutUnbounded reports a fan-out without finite materialization bounds.
	CodeFanOutUnbounded = "fan_out_unbounded"
	// CodeFanOutCeilingExceeded reports fan-out width beyond LoopMaxFanoutWidth.
	CodeFanOutCeilingExceeded = "fan_out_ceiling_exceeded"
	// CodeGateMaxRevisionsCeilingExceeded reports gate max_revisions beyond its ceiling.
	CodeGateMaxRevisionsCeilingExceeded = "gate_max_revisions_ceiling_exceeded"
	// CodeNodeIDInvalid reports a non-snake_case node id.
	CodeNodeIDInvalid = "node_id_invalid"
	// CodeVerdictPolicyRequiresJudge reports revise_until_clean without a judge/human source.
	CodeVerdictPolicyRequiresJudge = "verdict_policy_requires_judge"
	// CodeInvalidHarvest reports an unsupported or malformed harvest policy.
	CodeInvalidHarvest = "invalid_harvest"
	// CodeUnknownActionKind reports action kinds that are neither reserved nor resolvable ToolIDs.
	CodeUnknownActionKind = "unknown_action_kind"
	// CodeUnknownControlKind reports a control kind outside the closed enum.
	CodeUnknownControlKind = "unknown_control_kind"
	// CodeUnknownSourceKind reports a source kind outside the closed enum.
	CodeUnknownSourceKind = "unknown_source_kind"
	// CodeWatchKindRequired reports a watch-source missing its extension source kind.
	CodeWatchKindRequired = "watch_kind_required"
	// CodeDuplicateNodeID reports repeated node IDs.
	CodeDuplicateNodeID = "duplicate_node_id"
	// CodeUnknownTerminalState reports contract terminal states outside the closed enum.
	CodeUnknownTerminalState = "unknown_terminal_state"
)

// ToolSchemaSnapshot is the pure tool-schema view consumed by lint and compile.
type ToolSchemaSnapshot struct {
	ToolID             string          `json:"tool_id"`
	InputSchema        json.RawMessage `json:"input_schema,omitempty"`
	OutputSchema       json.RawMessage `json:"output_schema,omitempty"`
	InputSchemaDigest  string          `json:"input_schema_digest,omitempty"`
	OutputSchemaDigest string          `json:"output_schema_digest,omitempty"`
}

// ToolSchemaSource resolves open action ToolIDs without tying lint to runtime IO.
type ToolSchemaSource interface {
	Snapshot(toolID string) (ToolSchemaSnapshot, bool)
}
