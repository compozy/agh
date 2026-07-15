// Suite: Automation request normalization
// Invariant: Loop-target requests carry the one workspace authorized by their Automation scope.
// Boundary IN: controlled Automation drafts and request projection.
// Boundary OUT: transport and daemon validation, owned by API suites.
import { describe, expect, it } from "vitest";

import { createAutomationJobDraft, setJobTargetMode } from "../automation-drafts";
import { buildAutomationJobRequest } from "../automation-requests";

function loopJobDraft() {
  return {
    ...setJobTargetMode(createAutomationJobDraft("ws_alpha"), "loop"),
    name: "daily-loop",
    loop_target: {
      workspace_id: "ws_stale",
      loop_name: "software-delivery",
      inputs: {},
      input_mapping: {},
    },
  };
}

describe("automation request normalization", () => {
  it("Should bind workspace-scoped Job Loop targets to the outer workspace", () => {
    const draft = loopJobDraft();

    const request = buildAutomationJobRequest(draft);

    expect(request.loop_target?.workspace_id).toBe("ws_alpha");
    expect(draft.loop_target.workspace_id).toBe("ws_stale");
  });

  it("Should preserve an explicit global Job Loop target workspace", () => {
    const draft = {
      ...loopJobDraft(),
      scope: "global" as const,
      workspace_id: undefined,
    };

    expect(buildAutomationJobRequest(draft).loop_target?.workspace_id).toBe("ws_stale");
  });
});
