package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/compozy/agh/internal/loop/dsl"
	"github.com/compozy/agh/internal/task"
)

func (r *CoordinatorRunner) terminalForFailedGeneration(
	ctx context.Context,
	run Run,
	generation int,
	noProgressWindow int,
	outputs []GenerationOutput,
	failed GenerationOutput,
) (*task.CoordinatorTerminal, error) {
	stalled, err := r.stalledBlockingIssueTerminal(ctx, run.ID, generation, noProgressWindow, outputs)
	if err != nil {
		return nil, err
	}
	if stalled != nil {
		return stalled, nil
	}
	terminal := failedOutputTerminal(run, failed)
	return &terminal, nil
}

func failedOutputTerminal(run Run, output GenerationOutput) task.CoordinatorTerminal {
	status := StatusFailed
	cause := TransitionCauseContract
	reasonCode := "node_failed"
	if run.ConsecutiveFailures >= LoopFailureBreakerLimit {
		status = StatusStalled
		cause = TransitionCauseNoProgress
		reasonCode = "circuit_breaker"
	}
	if explicitDependencyBlocker(output.OutputRef) {
		status = StatusBlocked
		cause = TransitionCauseContract
		reasonCode = output.OutputRef
	}
	return task.CoordinatorTerminal{
		Status:     string(status),
		Cause:      string(cause),
		ReasonCode: reasonCode,
	}
}

func (r *CoordinatorRunner) stalledBlockingIssueTerminal(
	ctx context.Context,
	runID RunID,
	generation int,
	window int,
	outputs []GenerationOutput,
) (*task.CoordinatorTerminal, error) {
	current := blockingIssueSignature(outputs)
	if len(current) == 0 {
		return nil, nil
	}
	if window <= 0 {
		return nil, nil
	}
	if window == 1 {
		return stalledBlockingIssuesTerminal(), nil
	}
	for offset := 1; offset < window; offset++ {
		previousGeneration := generation - offset
		if previousGeneration <= 0 {
			return nil, nil
		}
		previous, err := r.outputs.ListGenerationOutputs(ctx, runID, previousGeneration)
		if err != nil {
			return nil, err
		}
		if !sameStringSet(current, blockingIssueSignature(previous)) {
			return nil, nil
		}
	}
	return stalledBlockingIssuesTerminal(), nil
}

func stalledBlockingIssuesTerminal() *task.CoordinatorTerminal {
	return &task.CoordinatorTerminal{
		Status:     string(StatusStalled),
		Cause:      string(TransitionCauseNoProgress),
		ReasonCode: blockingIssuesRepeatedCode,
	}
}

func blockingIssueSignature(outputs []GenerationOutput) []string {
	seen := make(map[string]struct{})
	for _, output := range outputs {
		if output.Status != generationOutputFailed {
			continue
		}
		for _, id := range blockingIssueIDs(output.OutputRef) {
			seen[id] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return nil
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func blockingIssueIDs(value string) []string {
	var payload struct {
		BlockingIssues []struct {
			ID string `json:"id"`
		} `json:"blocking_issues"`
	}
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		return nil
	}
	ids := make([]string, 0, len(payload.BlockingIssues))
	seen := make(map[string]struct{}, len(payload.BlockingIssues))
	for _, issue := range payload.BlockingIssues {
		id := strings.TrimSpace(issue.ID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func sameStringSet(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}

func explicitDependencyBlocker(value string) bool {
	const (
		dependencyMissing   = "dependency_missing"
		credentialMissing   = "credential_missing" // #nosec G101 -- public reason code, not a credential.
		resourceUnreachable = "resource_unreachable"
	)

	switch strings.TrimSpace(value) {
	case dependencyMissing, credentialMissing, resourceUnreachable:
		return true
	default:
		return false
	}
}

func failureReasonCode(value string) string {
	var payload struct {
		ReasonCode string `json:"reason_code"`
		Code       string `json:"code"`
	}
	if err := json.Unmarshal([]byte(value), &payload); err == nil {
		if strings.TrimSpace(payload.ReasonCode) != "" {
			return strings.TrimSpace(payload.ReasonCode)
		}
		if strings.TrimSpace(payload.Code) != "" {
			return strings.TrimSpace(payload.Code)
		}
	}
	return ""
}

func graphDependencies(graph dsl.Graph) map[dsl.NodeID][]dsl.NodeID {
	dependencies := make(map[dsl.NodeID][]dsl.NodeID, len(graph.Nodes))
	for _, node := range graph.Nodes {
		dependencies[node.ID] = nil
	}
	for _, edge := range graph.Edges {
		dependencies[edge.To] = append(dependencies[edge.To], edge.From)
	}
	return dependencies
}

func graphDependents(graph dsl.Graph) map[dsl.NodeID][]dsl.NodeID {
	dependents := make(map[dsl.NodeID][]dsl.NodeID, len(graph.Nodes))
	for _, node := range graph.Nodes {
		dependents[node.ID] = nil
	}
	for _, edge := range graph.Edges {
		dependents[edge.From] = append(dependents[edge.From], edge.To)
	}
	return dependents
}

func coordinatorRunID(loopRunID RunID, generation int) string {
	return fmt.Sprintf("run.loop.%s.g%d.coordinator", loopRunID, generation)
}

func coordinatorIdempotencyKey(loopRunID RunID, generation int) string {
	return fmt.Sprintf("loop.coordinator.%s.%d", loopRunID, generation)
}

func coordinatorNodeTaskID(
	loopRunID RunID,
	generation int,
	nodeID dsl.NodeID,
	itemIndex int,
) string {
	return fmt.Sprintf("loop.%s.g%d.node.%s.%d", loopRunID, generation, nodeID, itemIndex)
}

func coordinatorNodeRunID(
	loopRunID RunID,
	generation int,
	nodeID dsl.NodeID,
	itemIndex int,
) string {
	return fmt.Sprintf("run.loop.%s.g%d.node.%s.%d", loopRunID, generation, nodeID, itemIndex)
}

func coordinatorNodeIdempotencyKey(
	loopRunID RunID,
	generation int,
	nodeID dsl.NodeID,
	itemIndex int,
) string {
	return fmt.Sprintf("loop.node.%s.%d.%s.%d", loopRunID, generation, nodeID, itemIndex)
}
