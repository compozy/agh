import { describe, expect, it } from "vitest";

import { tasksKeys } from "./query-keys";

describe("tasksKeys", () => {
  it("namespaces all keys under tasks", () => {
    expect(tasksKeys.all).toEqual(["tasks"]);
    expect(tasksKeys.lists()).toEqual(["tasks", "list"]);
    expect(tasksKeys.details()).toEqual(["tasks", "detail"]);
    expect(tasksKeys.runsRoot()).toEqual(["tasks", "runs"]);
    expect(tasksKeys.timelineRoot()).toEqual(["tasks", "timeline"]);
    expect(tasksKeys.treeRoot()).toEqual(["tasks", "tree"]);
    expect(tasksKeys.runDetails()).toEqual(["tasks", "run-detail"]);
    expect(tasksKeys.triageRoot()).toEqual(["tasks", "triage"]);
  });

  it("produces stable list keys from filter input", () => {
    expect(
      tasksKeys.list({
        scope: "workspace",
        workspace: "ws_alpha",
        status: "ready",
        priority: "high",
        include_drafts: true,
        approval_state: "pending",
        owner_kind: "human",
        owner_ref: "op",
        parent_task_id: "task_parent",
        network_channel: "net",
        query: "review",
        limit: 50,
      })
    ).toEqual([
      "tasks",
      "list",
      "workspace",
      "ws_alpha",
      "ready",
      "high",
      "1",
      "pending",
      "human",
      "op",
      "task_parent",
      "net",
      "review",
      "50",
    ]);

    expect(tasksKeys.list()).toEqual([
      "tasks",
      "list",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
    ]);
  });

  it("distinguishes detail, run, timeline, tree, and run-detail keys by id", () => {
    expect(tasksKeys.detail("task_1")).toEqual(["tasks", "detail", "task_1"]);
    expect(tasksKeys.runs("task_1", { status: "running", limit: 5 })).toEqual([
      "tasks",
      "runs",
      "task_1",
      "running",
      "",
      "5",
    ]);
    expect(tasksKeys.timeline("task_1", { after_sequence: 12, limit: 50 })).toEqual([
      "tasks",
      "timeline",
      "task_1",
      "12",
      "50",
    ]);
    expect(tasksKeys.tree("task_1")).toEqual(["tasks", "tree", "task_1"]);
    expect(tasksKeys.runDetail("run_1")).toEqual(["tasks", "run-detail", "run_1"]);
  });

  it("serializes dashboard and inbox filters stably", () => {
    expect(tasksKeys.dashboard({ scope: "workspace", workspace: "ws_alpha" })).toEqual([
      "tasks",
      "dashboard",
      "workspace",
      "ws_alpha",
      "",
      "",
      "",
      "",
    ]);

    expect(
      tasksKeys.inbox({
        scope: "workspace",
        workspace: "ws_alpha",
        lane: "approvals",
        unread: true,
        limit: 20,
      })
    ).toEqual(["tasks", "inbox", "workspace", "ws_alpha", "", "", "approvals", "1", "", "20"]);
  });
});
