package globaldb

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"time"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/gate"
	taskpkg "github.com/compozy/agh/internal/task"
)

func appendLoopGenerationStartedEventWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	run looppkg.Run,
	generation int,
	at time.Time,
) error {
	if generation <= 0 || generation <= run.Generation {
		return nil
	}
	payload := map[string]any{
		loopRunEventPayloadKeyGeneration: generation,
		"reattempt_strategy":             string(run.ReattemptStrategy),
		columnLoopName:                   run.LoopName,
	}
	return appendLoopRunEventWithExecutor(
		ctx,
		exec,
		run.ID,
		run.WorkspaceID,
		loopRunEventGenerationStarted,
		payload,
		at,
	)
}

func appendLoopGateVerdictEventWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	run looppkg.Run,
	generation int,
	terminal taskpkg.CoordinatorTerminal,
	at time.Time,
) error {
	if len(bytes.TrimSpace(terminal.Details)) == 0 {
		return nil
	}
	payload := loopGateVerdictEventPayload(generation, terminal)
	return appendLoopRunEventWithExecutor(ctx, exec, run.ID, run.WorkspaceID, loopRunEventGateVerdict, payload, at)
}

func loopGateVerdictEventPayload(
	generation int,
	terminal taskpkg.CoordinatorTerminal,
) map[string]any {
	gateID := strings.TrimSpace(terminal.GateID)
	var verdict gate.Verdict
	if err := json.Unmarshal(terminal.Details, &verdict); err != nil {
		return map[string]any{
			"node_id":                        gateID,
			loopRunEventPayloadKeyGeneration: generation,
			"verdict":                        loopRunEventVerdictRevise,
			loopRunEventPayloadKeyReason:     strings.TrimSpace(terminal.ReasonCode),
			"route":                          strings.TrimSpace(terminal.Status),
			"details":                        cloneLoopEventRawJSON(terminal.Details),
		}
	}
	verdictLabel := loopRunEventVerdictRevise
	if verdict.Outcome == gate.VerdictOutcomeApproved {
		verdictLabel = "pass"
	}
	criteria := make([]map[string]any, 0, len(verdict.Criteria))
	for _, criterion := range verdict.Criteria {
		status := loopRunEventVerdictRevise
		if criterion.Passed {
			status = "pass"
		}
		criteria = append(criteria, map[string]any{
			"id":                         strings.TrimSpace(criterion.ID),
			"type":                       string(criterion.Type),
			loopRunEventPayloadKeyStatus: status,
			"note":                       string(criterion.Outcome),
		})
	}
	issues := make([]map[string]any, 0, len(verdict.BlockingIssues))
	for _, issue := range verdict.BlockingIssues {
		issues = append(issues, map[string]any{
			"id":   strings.TrimSpace(issue.ID),
			"note": strings.TrimSpace(issue.Note),
		})
	}
	payload := map[string]any{
		"node_id":                        firstNonEmptyString(gateID, "definition_of_done"),
		loopRunEventPayloadKeyGeneration: generation,
		"verdict":                        verdictLabel,
		loopRunEventPayloadKeyReason: firstNonEmptyString(
			verdict.Route.ReasonCode,
			string(verdict.Outcome),
		),
		"route":           string(verdict.Route.Action),
		"criteria":        criteria,
		"blocking_issues": issues,
	}
	if len(verdict.Criteria) > 0 && verdict.Criteria[0].Confidence != nil {
		payload["confidence"] = *verdict.Criteria[0].Confidence
	}
	return payload
}

func appendLoopNeedsApprovalEventWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	run looppkg.Run,
	gateID looppkg.NodeID,
	generation int,
	title string,
	facts []map[string]string,
	at time.Time,
) error {
	return appendLoopRunEventWithExecutor(ctx, exec, run.ID, run.WorkspaceID, loopRunEventNeedsApproval, map[string]any{
		"gate_id":                        strings.TrimSpace(string(gateID)),
		loopRunEventPayloadKeyTitle:      firstNonEmptyString(title, "Approve to resume"),
		loopRunEventPayloadKeyGeneration: generation,
		"facts":                          facts,
	}, at)
}

func loopApprovalFact(label string, value string) map[string]string {
	return map[string]string{
		loopRunApprovalFactLabelKey: label,
		loopRunEventPayloadKeyValue: value,
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
