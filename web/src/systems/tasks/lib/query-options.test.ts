import { describe, expect, it } from "vitest";

import {
  taskDashboardOptions,
  taskDetailOptions,
  taskInboxOptions,
  taskRunDetailOptions,
  taskRunsOptions,
  taskTimelineOptions,
  taskTreeOptions,
  tasksListOptions,
} from "./query-options";

describe("tasks list options", () => {
  it("uses the default stale and refetch cadence", () => {
    const options = tasksListOptions();

    expect(options.staleTime).toBe(15_000);
    expect(options.refetchInterval).toBe(30_000);
    expect(options.enabled).toBe(true);
  });

  it("carries filters into the query key", () => {
    const options = tasksListOptions({
      scope: "workspace",
      workspace: "ws_alpha",
      status: "ready",
      limit: 20,
    });

    expect(options.queryKey).toContain("workspace");
    expect(options.queryKey).toContain("ws_alpha");
    expect(options.queryKey).toContain("ready");
    expect(options.queryKey).toContain("20");
  });

  it("supports an explicit disabled state for the list query", () => {
    expect(tasksListOptions({}, false).enabled).toBe(false);
  });
});

describe("tasks detail and run options", () => {
  it("disables detail queries for empty ids", () => {
    expect(taskDetailOptions("").enabled).toBe(false);
    expect(taskDetailOptions("task_1", false).enabled).toBe(false);
    expect(taskDetailOptions("task_1").enabled).toBe(true);
  });

  it("uses the live cadence for runs, timeline, tree, and run detail", () => {
    expect(taskRunsOptions("task_1").refetchInterval).toBe(15_000);
    expect(taskTimelineOptions("task_1").refetchInterval).toBe(15_000);
    expect(taskTimelineOptions("task_1").staleTime).toBe(5_000);
    expect(taskTreeOptions("task_1").refetchInterval).toBe(15_000);
    expect(taskRunDetailOptions("run_1").refetchInterval).toBe(15_000);
  });

  it("disables live queries when ids are missing", () => {
    expect(taskRunsOptions("").enabled).toBe(false);
    expect(taskTimelineOptions("").enabled).toBe(false);
    expect(taskTreeOptions("").enabled).toBe(false);
    expect(taskRunDetailOptions("").enabled).toBe(false);
  });

  it("carries timeline filters into the query key", () => {
    const options = taskTimelineOptions("task_1", { after_sequence: 12, limit: 30 });

    expect(options.queryKey).toEqual(["tasks", "timeline", "task_1", "12", "30"]);
  });
});

describe("tasks dashboard and inbox options", () => {
  it("uses the default cadence for aggregate reads", () => {
    const dashboardOptions = taskDashboardOptions({ scope: "workspace" });
    const inboxOptions = taskInboxOptions({ lane: "approvals" });

    expect(dashboardOptions.staleTime).toBe(15_000);
    expect(dashboardOptions.refetchInterval).toBe(30_000);
    expect(inboxOptions.staleTime).toBe(15_000);
    expect(inboxOptions.refetchInterval).toBe(30_000);
    expect(dashboardOptions.queryKey).toContain("workspace");
    expect(inboxOptions.queryKey).toContain("approvals");
  });

  it("supports explicit disabled state for aggregate reads", () => {
    expect(taskDashboardOptions({}, false).enabled).toBe(false);
    expect(taskInboxOptions({}, false).enabled).toBe(false);
  });
});
