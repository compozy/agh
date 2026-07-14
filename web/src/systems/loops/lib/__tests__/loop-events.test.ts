import { describe, expect, it } from "vitest";

import { applyLoopEventFrame, emptyLoopRunLiveState } from "../loop-events";
import type { LoopRunEventFrame, LoopRunEventKind } from "../../types";

function frame(kind: LoopRunEventKind, payload: unknown, seq = 1): LoopRunEventFrame {
  return {
    id: `ev-${seq}`,
    seq,
    kind,
    loop_run_id: "looprun_running",
    workspace_id: "ws_default",
    at: "2026-07-06T14:38:00Z",
    payload,
  };
}

describe("applyLoopEventFrame", () => {
  it("Should append a rail line for every frame, newest-first and bounded", () => {
    let state = emptyLoopRunLiveState();
    for (let i = 1; i <= 45; i++) {
      state = applyLoopEventFrame(state, frame("status_changed", { status: "running" }, i));
    }
    expect(state.events).toHaveLength(40);
    expect(state.events[0].seq).toBe(45);
  });

  it("Should fold a gate_verdict into the per-node verdict map", () => {
    const state = applyLoopEventFrame(
      emptyLoopRunLiveState(),
      frame("gate_verdict", {
        node_id: "review",
        generation: 1,
        verdict: "revise",
        confidence: 0.91,
        criteria: [{ id: "all_handled", type: "agent-judge", status: "revise" }],
        blocking_issues: [{ id: "issue_022", note: "no triage decision" }],
        route: "revise",
      })
    );
    const verdict = state.gateVerdicts.review;
    expect(verdict.verdict).toBe("revise");
    expect(verdict.confidence).toBe(0.91);
    expect(verdict.criteria).toHaveLength(1);
    expect(verdict.blockingIssues[0].id).toBe("issue_022");
    expect(verdict.route).toBe("revise");
  });

  it("Should append channel messages and flag the harvested result", () => {
    let state = applyLoopEventFrame(
      emptyLoopRunLiveState(),
      frame("channel_msg", { id: "m1", author: "implementer", text: "shipped" }, 1)
    );
    state = applyLoopEventFrame(
      state,
      frame(
        "channel_msg",
        { id: "m2", author: "coordinator", text: "approved", is_result: true },
        2
      )
    );
    expect(state.channelMessages).toHaveLength(2);
    expect(state.channelMessages[1].isResult).toBe(true);
  });

  it("Should capture a needs_approval payload and the latest token tick", () => {
    let state = applyLoopEventFrame(
      emptyLoopRunLiveState(),
      frame("needs_approval", {
        gate_id: "approve",
        title: "Approve merge to main?",
        facts: [{ label: "Branch", value: "main" }],
      })
    );
    state = applyLoopEventFrame(state, frame("token_tick", { tokens_used: 268_000 }, 2));
    expect(state.needsApproval?.gateId).toBe("approve");
    expect(state.needsApproval?.facts[0].value).toBe("main");
    expect(state.tokensUsed).toBe(268_000);
  });

  it("Should render generated re-attempt enum values as display labels", () => {
    const state = applyLoopEventFrame(
      emptyLoopRunLiveState(),
      frame("generation_started", { generation: 2, reattempt_strategy: "failed_only" })
    );
    expect(state.events[0].message).toBe("gen 2 · failed-only");
  });

  it("Should merge Goal turn start and completion frames by prompt identity", () => {
    const started = applyLoopEventFrame(
      emptyLoopRunLiveState(),
      frame("goal_turn_started", {
        seq: 7,
        generation: 2,
        node_id: "goal",
        item_index: 1,
        turn: 3,
        prompt_attempt: 1,
        prompt_id: "prompt_7",
        session_id: "session_1",
        binding_handle: "goal:abc",
        binding_epoch: 2,
        actor_kind: "agent",
        actor_id: "implementer",
      })
    );

    const completed = applyLoopEventFrame(
      started,
      frame(
        "goal_turn_completed",
        {
          seq: 7,
          generation: 2,
          node_id: "goal",
          item_index: 1,
          turn: 3,
          prompt_attempt: 1,
          prompt_id: "prompt_7",
          session_id: "session_1",
          result_status: "completed",
          stop_reason: "end_turn",
          verdict_outcome: "rejected",
          blocking_issues: [{ id: "issue_1", note: "Missing evidence" }],
          evidence_ref: "blob_1",
          tokens_used: 420,
        },
        2
      )
    );

    expect(completed.goalTurns).toHaveLength(1);
    expect(completed.goalTurns[0]).toMatchObject({
      promptId: "prompt_7",
      resultStatus: "completed",
      stopReason: "end_turn",
      verdictOutcome: "rejected",
      evidenceRef: "blob_1",
      tokensUsed: 420,
    });
    expect(completed.goalTurns[0].blockingIssues).toEqual([
      { id: "issue_1", note: "Missing evidence" },
    ]);
  });

  it("Should keep Goal status changes in the rail without inventing a turn", () => {
    const state = applyLoopEventFrame(
      emptyLoopRunLiveState(),
      frame("goal_status_changed", { from: "active", to: "complete" })
    );
    expect(state.events[0]).toMatchObject({
      kind: "goal_status_changed",
      tone: "warn",
      message: "active → complete",
    });
    expect(state.goalTurns).toEqual([]);
  });

  it("Should project a generation-zero coordinator failure from the terminal status event", () => {
    const state = applyLoopEventFrame(
      emptyLoopRunLiveState(),
      frame("status_changed", {
        status: "failed",
        generation: 0,
        failure: {
          kind: "coordinator_failure",
          code: "watch_poll_failed",
          cause: "The watch source failed before it could produce a generation.",
          recovery:
            "Verify the Loop watch provider and workspace prerequisites, then start a new run.",
        },
      })
    );

    expect(state.failure).toEqual({
      kind: "coordinator_failure",
      code: "watch_poll_failed",
      cause: "The watch source failed before it could produce a generation.",
      recovery: "Verify the Loop watch provider and workspace prerequisites, then start a new run.",
    });
    expect(state.events[0]).toMatchObject({ tone: "err", message: "failed" });
  });

  it("Should degrade a malformed frame to a rail line without throwing", () => {
    const state = applyLoopEventFrame(emptyLoopRunLiveState(), frame("gate_verdict", null));
    expect(state.events).toHaveLength(1);
    expect(state.gateVerdicts).toEqual({});
  });
});
