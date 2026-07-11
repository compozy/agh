//go:build integration && !windows

package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	aghcontract "github.com/compozy/agh/internal/api/contract"
	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/testutil/acpmock"
	e2etest "github.com/compozy/agh/internal/testutil/e2e"
)

const goalCommandJudgeScript = `#!/bin/sh
set -eu
mode="$1"
if [ "$mode" = "rejections" ]; then
  counter_file=".goal-judge-rejections"
  attempt=0
  if [ -f "$counter_file" ]; then
    attempt="$(cat "$counter_file")"
  fi
  attempt=$((attempt + 1))
  printf '%s' "$attempt" > "$counter_file"
  if [ "$attempt" -lt 3 ]; then
    printf '{"verdict":"revise","blocking_issues":[{"id":"attempt_%s","note":"Objective needs another turn."}],"confidence":0.9,"evidence":{"attempt":%s}}\n' "$attempt" "$attempt"
    exit 0
  fi
fi
printf '{"verdict":"pass","blocking_issues":[],"confidence":1.0,"evidence":{"mode":"%s"}}\n' "$mode"
`

func TestDaemonE2EGoalCommandsShouldSurviveControlsDisconnectAndRestart(t *testing.T) {
	acpmock.RequireDriver(t)

	homePaths := e2etest.NewHomePaths(t)
	workspaceRoot := t.TempDir()
	options := goalCommandRuntimeOptions(t, homePaths, workspaceRoot)
	harness := e2etest.StartRuntimeHarness(t, options)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	rejectionSession := createFixtureBackedSession(t, ctx, harness, "goal-rejections", "goal-rejections")
	status := startGoalThroughHTTPAndDisconnect(
		t,
		ctx,
		harness,
		rejectionSession.ID,
		"Complete after two evidence-backed revisions",
	)
	if status != http.StatusAccepted {
		t.Fatalf("HTTP disconnected Goal start status = %d, want %d", status, http.StatusAccepted)
	}
	rejectionSnapshot := waitForGoalSnapshot(
		t,
		ctx,
		harness,
		rejectionSession.ID,
		func(goal *aghcontract.GoalSnapshot) bool {
			if goal != nil && goal.RunStatus == aghcontract.LoopRunStatusNeedsApproval {
				events, eventsErr := harness.SessionEvents(ctx, rejectionSession.ID)
				t.Fatalf(
					"rejection Goal unexpectedly requires approval: snapshot=%#v cause=%v events=%#v events_error=%v",
					*goal,
					goalReasonValue(goal.Cause),
					events.Events,
					eventsErr,
				)
			}
			return goal != nil && goal.RunStatus == aghcontract.LoopRunStatusDone && goal.Status == "complete"
		},
	)
	rejectionTurns := waitForGoalTurns(
		t,
		ctx,
		harness,
		rejectionSnapshot.RunID,
		func(page aghcontract.GoalTurnPage) bool {
			return len(page.Turns) == 3 && page.Turns[2].ResultStatus != nil
		},
	)
	assertGoalJudgeOutcomes(t, rejectionTurns, []string{"rejected", "rejected", "approved"})

	pauseSession := createFixtureBackedSession(t, ctx, harness, "goal-pause", "goal-pause")
	startedStatus, started := callGoalCommandUDS(
		t,
		ctx,
		harness,
		pauseSession.ID,
		"/goal Pause and resume at a durable boundary",
	)
	if startedStatus != http.StatusAccepted || started.Outcome != aghcontract.GoalOutcomeStarted ||
		started.Snapshot == nil {
		t.Fatalf("pause Goal start = status:%d result:%#v", startedStatus, started)
	}
	pauseStatus, paused := callGoalCommandUDS(t, ctx, harness, pauseSession.ID, "/goal pause")
	if pauseStatus != http.StatusOK || paused.Outcome != aghcontract.GoalOutcomePaused {
		t.Fatalf("pause Goal command = status:%d result:%#v", pauseStatus, paused)
	}
	waitForGoalSnapshot(t, ctx, harness, pauseSession.ID, func(goal *aghcontract.GoalSnapshot) bool {
		return goal != nil && goal.RunStatus == aghcontract.LoopRunStatusPaused
	})
	resumeStatus, resumed := callGoalCommandUDS(t, ctx, harness, pauseSession.ID, "/goal resume")
	if resumeStatus != http.StatusOK || resumed.Outcome != aghcontract.GoalOutcomeResumed {
		t.Fatalf("resume paused Goal command = status:%d result:%#v", resumeStatus, resumed)
	}
	pauseCompleted := waitForGoalSnapshot(t, ctx, harness, pauseSession.ID, func(goal *aghcontract.GoalSnapshot) bool {
		return goal != nil && goal.RunStatus == aghcontract.LoopRunStatusDone
	})
	pauseTurns := waitForGoalTurns(t, ctx, harness, pauseCompleted.RunID, func(page aghcontract.GoalTurnPage) bool {
		return len(page.Turns) >= 1 && page.Turns[len(page.Turns)-1].ResultStatus != nil
	})
	assertMonotonicGoalTurns(t, pauseTurns)

	approvalSession := createFixtureBackedSession(t, ctx, harness, "goal-approval", "goal-approval")
	approvalStartStatus, approvalStart := callGoalCommandUDS(
		t,
		ctx,
		harness,
		approvalSession.ID,
		"/goal Recover from a refused turn through an explicit approval grant",
	)
	if approvalStartStatus != http.StatusAccepted || approvalStart.Snapshot == nil {
		t.Fatalf("approval Goal start = status:%d result:%#v", approvalStartStatus, approvalStart)
	}
	waitForGoalSnapshot(t, ctx, harness, approvalSession.ID, func(goal *aghcontract.GoalSnapshot) bool {
		return goal != nil && goal.RunStatus == aghcontract.LoopRunStatusNeedsApproval
	})
	grantStatus, granted := callGoalCommandUDS(t, ctx, harness, approvalSession.ID, "/goal resume")
	if grantStatus != http.StatusOK || granted.Outcome != aghcontract.GoalOutcomeResumed {
		t.Fatalf("approval grant command = status:%d result:%#v", grantStatus, granted)
	}
	approvalCompleted := waitForGoalSnapshot(
		t,
		ctx,
		harness,
		approvalSession.ID,
		func(goal *aghcontract.GoalSnapshot) bool {
			return goal != nil && goal.RunStatus == aghcontract.LoopRunStatusDone
		},
	)
	approvalTurns := waitForGoalTurns(
		t,
		ctx,
		harness,
		approvalCompleted.RunID,
		func(page aghcontract.GoalTurnPage) bool {
			return len(page.Turns) >= 2 && page.Turns[len(page.Turns)-1].ResultStatus != nil
		},
	)
	assertMonotonicGoalTurns(t, approvalTurns)

	clearSession := createFixtureBackedSession(t, ctx, harness, "goal-clear", "goal-clear")
	clearStartStatus, clearStart := callGoalCommandUDS(
		t,
		ctx,
		harness,
		clearSession.ID,
		"/goal Keep one prompt active until an explicit clear",
	)
	if clearStartStatus != http.StatusAccepted || clearStart.Snapshot == nil {
		t.Fatalf("clear Goal start = status:%d result:%#v", clearStartStatus, clearStart)
	}
	clearRunID := clearStart.Snapshot.RunID
	waitForGoalTurns(t, ctx, harness, clearRunID, func(page aghcontract.GoalTurnPage) bool {
		return len(page.Turns) == 1 && page.Turns[0].ResultStatus == nil
	})
	staleStatus, stale := callGoalCommandUDS(
		t,
		ctx,
		harness,
		clearSession.ID,
		"/goal replace stale-run-id Replace the wrong Run",
	)
	if staleStatus != http.StatusConflict || stale.Outcome != aghcontract.GoalOutcomeError ||
		stale.ReasonCode == nil || *stale.ReasonCode != aghcontract.GoalReasonReplaceStale ||
		stale.Snapshot == nil || stale.Snapshot.RunID != clearRunID {
		t.Fatalf("stale replacement result = status:%d result:%#v", staleStatus, stale)
	}
	clearStatus, cleared := callGoalCommandUDS(t, ctx, harness, clearSession.ID, "/goal clear")
	if clearStatus != http.StatusOK || cleared.Outcome != aghcontract.GoalOutcomeCleared || cleared.Snapshot != nil {
		t.Fatalf("clear active Goal result = status:%d result:%#v", clearStatus, cleared)
	}
	clearedProjection, err := getGoalSnapshot(ctx, harness, clearSession.ID)
	if err != nil {
		t.Fatalf("get cleared Goal snapshot error = %v", err)
	}
	if clearedProjection.Goal != nil {
		t.Fatalf("cleared Goal projection = %#v, want nil", clearedProjection.Goal)
	}
	clearedTurns := waitForGoalTurns(t, ctx, harness, clearRunID, func(page aghcontract.GoalTurnPage) bool {
		return len(page.Turns) == 1 && page.Turns[0].ResultStatus != nil
	})
	if clearedTurns.Turns[0].ReasonCode == nil ||
		*clearedTurns.Turns[0].ReasonCode != aghcontract.GoalReasonControlRevokedInFlight {
		t.Fatalf("cleared active turn = %#v", clearedTurns.Turns[0])
	}
	time.Sleep(200 * time.Millisecond)
	afterClear, err := getGoalTurns(ctx, harness, clearRunID)
	if err != nil {
		t.Fatalf("get Goal turns after clear fence error = %v", err)
	}
	if len(afterClear.Turns) != len(clearedTurns.Turns) {
		t.Fatalf("Goal turns after clear = %d, want stable %d", len(afterClear.Turns), len(clearedTurns.Turns))
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := harness.Stop(stopCtx); err != nil {
		stopCancel()
		t.Fatalf("Stop(before Goal restart) error = %v", err)
	}
	stopCancel()

	restarted := e2etest.StartRuntimeHarness(t, options)
	restartedSnapshot := waitForGoalSnapshot(
		t,
		ctx,
		restarted,
		rejectionSession.ID,
		func(goal *aghcontract.GoalSnapshot) bool {
			return goal != nil && goal.RunID == rejectionSnapshot.RunID &&
				goal.RunStatus == aghcontract.LoopRunStatusDone
		},
	)
	if restartedSnapshot.TurnsUsed != 3 || restartedSnapshot.LastVerdict == nil ||
		restartedSnapshot.LastVerdict.Outcome != "approved" {
		t.Fatalf("restarted Goal snapshot = %#v", restartedSnapshot)
	}
	restartedTurns, err := getGoalTurns(ctx, restarted, rejectionSnapshot.RunID)
	if err != nil {
		t.Fatalf("get restarted Goal turns error = %v", err)
	}
	assertGoalJudgeOutcomes(t, restartedTurns, []string{"rejected", "rejected", "approved"})
	assertGoalTurnsCLIParity(t, ctx, restarted, restartedTurns, rejectionSnapshot.RunID)
}

func goalCommandRuntimeOptions(
	t testing.TB,
	homePaths aghconfig.HomePaths,
	workspaceRoot string,
) e2etest.RuntimeHarnessOptions {
	t.Helper()
	fixturePath := mockFixturePath(t, "goal_command_fixture.json")
	agents := []struct {
		fixture string
		name    string
	}{
		{fixture: "goal_rejections", name: "goal-rejections"},
		{fixture: "goal_pause", name: "goal-pause"},
		{fixture: "goal_approval", name: "goal-approval"},
		{fixture: "goal_clear", name: "goal-clear"},
	}
	mockAgents := make([]e2etest.MockAgentSpec, 0, len(agents))
	for _, agent := range agents {
		mockAgents = append(mockAgents, e2etest.MockAgentSpec{
			FixturePath: fixturePath, FixtureAgent: agent.fixture, AgentName: agent.name,
		})
	}
	return e2etest.RuntimeHarnessOptions{
		HomePaths:  homePaths,
		MockAgents: mockAgents,
		Workspace: e2etest.WorkspaceSeedOptions{
			Root: workspaceRoot,
			Files: map[string]string{
				filepath.Join(aghconfig.DirName, aghconfig.ConfigName): `
[memory]
enabled = false

[loops.defaults.delivery.model_defaults]
judge = "goal-e2e-judge"
`,
				"goal-judge.sh": goalCommandJudgeScript,
			},
		},
		StartTimeout: 30 * time.Second,
	}
}

func goalSessionPromptPath(harness *e2etest.RuntimeHarness, sessionID string) string {
	return "/api/workspaces/" + url.PathEscape(harness.WorkspaceID) +
		"/sessions/" + url.PathEscape(strings.TrimSpace(sessionID)) + "/prompt"
}

func startGoalThroughHTTPAndDisconnect(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	sessionID string,
	objective string,
) int {
	t.Helper()
	payload, err := json.Marshal(aghcontract.SendPromptRequest{Message: "/goal " + strings.TrimSpace(objective)})
	if err != nil {
		t.Fatalf("marshal disconnected Goal start error = %v", err)
	}
	requestCtx, cancel := context.WithCancel(ctx)
	request, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodPost,
		harness.HTTPURL(goalSessionPromptPath(harness, sessionID)),
		bytes.NewReader(payload),
	)
	if err != nil {
		cancel()
		t.Fatalf("create disconnected Goal start request error = %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := harness.HTTPClient.Do(request)
	if err != nil {
		cancel()
		t.Fatalf("send disconnected Goal start request error = %v", err)
	}
	if response.StatusCode != http.StatusAccepted {
		body, readErr := io.ReadAll(response.Body)
		closeErr := response.Body.Close()
		cancel()
		if err := errors.Join(readErr, closeErr); err != nil {
			t.Fatalf("read failed Goal start response error = %v", err)
		}
		t.Fatalf("HTTP Goal start status = %d; body=%s", response.StatusCode, body)
	}
	cancel()
	if err := response.Body.Close(); err != nil {
		t.Fatalf("close disconnected Goal start response error = %v", err)
	}
	return response.StatusCode
}

func callGoalCommandUDS(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	sessionID string,
	command string,
) (int, aghcontract.GoalCommandResult) {
	t.Helper()
	payload, err := json.Marshal(aghcontract.SendPromptRequest{Message: strings.TrimSpace(command)})
	if err != nil {
		t.Fatalf("marshal Goal command error = %v", err)
	}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		harness.UDSURL(goalSessionPromptPath(harness, sessionID)),
		bytes.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("create Goal command request error = %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := harness.UDSClient.Do(request)
	if err != nil {
		t.Fatalf("send Goal command request error = %v", err)
	}
	body, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		t.Fatalf("read Goal command response error = %v", err)
	}
	if contentType := response.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("Goal command content type = %q, want application/json", contentType)
	}
	var result aghcontract.GoalCommandResult
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decode Goal command result error = %v; body=%s", err, body)
	}
	if result.Outcome == "" {
		t.Fatalf("Goal command returned no structured result; status=%d body=%s", response.StatusCode, body)
	}
	return response.StatusCode, result
}

func getGoalSnapshot(
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	sessionID string,
) (aghcontract.SessionGoalResponse, error) {
	var response aghcontract.SessionGoalResponse
	path := "/api/workspaces/" + url.PathEscape(harness.WorkspaceID) +
		"/sessions/" + url.PathEscape(strings.TrimSpace(sessionID)) + "/goal"
	err := harness.UDSJSON(ctx, http.MethodGet, path, nil, &response)
	return response, err
}

func getGoalTurns(
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	runID string,
) (aghcontract.GoalTurnPage, error) {
	var response aghcontract.GoalTurnPage
	path := "/api/workspaces/" + url.PathEscape(harness.WorkspaceID) +
		"/loop-runs/" + url.PathEscape(strings.TrimSpace(runID)) + "/turns"
	err := harness.UDSJSON(ctx, http.MethodGet, path, nil, &response)
	return response, err
}

func waitForGoalSnapshot(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	sessionID string,
	predicate func(*aghcontract.GoalSnapshot) bool,
) aghcontract.GoalSnapshot {
	t.Helper()
	deadline := time.NewTimer(30 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	var last aghcontract.SessionGoalResponse
	var lastErr error
	for {
		last, lastErr = getGoalSnapshot(ctx, harness, sessionID)
		if lastErr == nil && predicate(last.Goal) {
			return *last.Goal
		}
		select {
		case <-ctx.Done():
			t.Fatalf(
				"wait for Goal snapshot context error = %v; last=%#v; last_error=%v",
				ctx.Err(),
				goalSnapshotValue(last),
				lastErr,
			)
		case <-deadline.C:
			t.Fatalf(
				"timed out waiting for Goal snapshot; last=%#v; last_error=%v",
				goalSnapshotValue(last),
				lastErr,
			)
		case <-ticker.C:
		}
	}
}

func goalSnapshotValue(response aghcontract.SessionGoalResponse) any {
	if response.Goal == nil {
		return nil
	}
	return *response.Goal
}

func goalReasonValue(reason *aghcontract.GoalReasonCode) any {
	if reason == nil {
		return nil
	}
	return *reason
}

func waitForGoalTurns(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	runID string,
	predicate func(aghcontract.GoalTurnPage) bool,
) aghcontract.GoalTurnPage {
	t.Helper()
	deadline := time.NewTimer(30 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	var last aghcontract.GoalTurnPage
	var lastErr error
	for {
		last, lastErr = getGoalTurns(ctx, harness, runID)
		if lastErr == nil && predicate(last) {
			return last
		}
		select {
		case <-ctx.Done():
			t.Fatalf("wait for Goal turns context error = %v; last=%#v; last_error=%v", ctx.Err(), last, lastErr)
		case <-deadline.C:
			t.Fatalf("timed out waiting for Goal turns; last=%#v; last_error=%v", last, lastErr)
		case <-ticker.C:
		}
	}
}

func assertGoalJudgeOutcomes(t testing.TB, page aghcontract.GoalTurnPage, want []string) {
	t.Helper()
	if len(page.Turns) != len(want) {
		t.Fatalf("Goal turn count = %d, want %d: %#v", len(page.Turns), len(want), page)
	}
	for index, outcome := range want {
		if page.Turns[index].VerdictOutcome == nil || string(*page.Turns[index].VerdictOutcome) != outcome {
			t.Fatalf("Goal turn %d verdict = %#v, want %q", index+1, page.Turns[index].VerdictOutcome, outcome)
		}
	}
	assertMonotonicGoalTurns(t, page)
}

func assertMonotonicGoalTurns(t testing.TB, page aghcontract.GoalTurnPage) {
	t.Helper()
	type streamKey struct {
		generation int64
		nodeID     string
		itemIndex  int
	}
	var previousSeq int64
	lastTurnByStream := make(map[streamKey]int)
	for index, turn := range page.Turns {
		key := streamKey{generation: turn.Generation, nodeID: turn.NodeID, itemIndex: turn.ItemIndex}
		expectedTurn := lastTurnByStream[key] + 1
		if turn.Seq <= previousSeq || turn.Turn != expectedTurn {
			t.Fatalf("Goal turn ordering changed at %d: %#v", index, page.Turns)
		}
		previousSeq = turn.Seq
		lastTurnByStream[key] = turn.Turn
	}
}

func assertGoalTurnsCLIParity(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	want aghcontract.GoalTurnPage,
	runID string,
) {
	t.Helper()
	var jsonPage aghcontract.GoalTurnPage
	if err := harness.CLI.RunJSON(
		ctx,
		&jsonPage,
		"loop",
		"turns",
		"--workspace",
		harness.WorkspaceID,
		"--run",
		runID,
		"-o",
		"json",
	); err != nil {
		t.Fatalf("CLI Goal turns JSON error = %v", err)
	}
	if len(jsonPage.Turns) != len(want.Turns) || jsonPage.Turns[0].Seq != want.Turns[0].Seq {
		t.Fatalf("CLI Goal turns JSON = %#v, want parity with %#v", jsonPage, want)
	}
	stdout, stderr, err := harness.CLI.Run(
		ctx,
		"loop",
		"turns",
		"--workspace",
		harness.WorkspaceID,
		"--run",
		runID,
		"-o",
		"jsonl",
	)
	if err != nil {
		t.Fatalf("CLI Goal turns JSONL error = %v; stderr=%s", err, strings.TrimSpace(stderr))
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != len(want.Turns) {
		t.Fatalf("CLI Goal turns JSONL lines = %d, want %d; stdout=%s", len(lines), len(want.Turns), stdout)
	}
	for index, line := range lines {
		var turn aghcontract.GoalTurn
		if err := json.Unmarshal([]byte(line), &turn); err != nil {
			t.Fatalf("decode CLI Goal turn JSONL line %d error = %v; line=%s", index, err, line)
		}
		if turn.Seq != want.Turns[index].Seq || turn.PromptID != want.Turns[index].PromptID {
			t.Fatalf("CLI Goal turn JSONL line %d = %#v, want %#v", index, turn, want.Turns[index])
		}
	}
}
