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

  it("Should degrade a malformed frame to a rail line without throwing", () => {
    const state = applyLoopEventFrame(emptyLoopRunLiveState(), frame("gate_verdict", null));
    expect(state.events).toHaveLength(1);
    expect(state.gateVerdicts).toEqual({});
  });
});
