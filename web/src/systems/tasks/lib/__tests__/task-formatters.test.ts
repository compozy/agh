import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { describe, expect, it } from "vitest";

import type { TaskListItem, TaskRun } from "../../types";
import {
  countTasksByStatus,
  formatAttemptLabel,
  formatDurationMs,
  formatPercent,
  formatRelativeTime,
  matchesTaskQuery,
  runCoordinationChannelLabel,
  runIsCoordinated,
  taskApprovalStateLabel,
  taskHandoffActionCopy,
  taskHandoffActionKey,
  taskHasApprovalPending,
  taskInboxLaneLabel,
  taskIsBlocked,
  taskIsDraft,
  taskLaneTone,
  taskLifecyclePhase,
  taskLifecyclePhaseDescription,
  taskLifecyclePhaseLabel,
  taskLifecyclePhaseTone,
  taskOwnerKindLabel,
  taskOwnerLabel,
  taskPriorityLabel,
  taskPriorityTone,
  taskRunStatusTone,
  taskStatusLabel,
  taskStatusTone,
} from "../task-formatters";

function makeTask(overrides: Partial<TaskListItem> = {}): TaskListItem {
  return {
    id: "task_001",
    title: "Review",
    status: "ready",
    scope: "workspace",
    origin: { kind: "web", ref: "op" },
    created_at: "2026-04-11T09:00:00Z",
    updated_at: "2026-04-11T09:00:00Z",
    created_by: { kind: "human", ref: "op" },
    ...overrides,
  } as TaskListItem;
}

describe("task status and priority labels", () => {
  it("labels every documented task status", () => {
    expect(taskStatusLabel("draft")).toBe("Draft");
    expect(taskStatusLabel("pending")).toBe("Pending");
    expect(taskStatusLabel("blocked")).toBe("Blocked");
    expect(taskStatusLabel("ready")).toBe("Ready");
    expect(taskStatusLabel("in_progress")).toBe("In Progress");
    expect(taskStatusLabel("completed")).toBe("Completed");
    expect(taskStatusLabel("failed")).toBe("Failed");
    expect(taskStatusLabel("canceled")).toBe("Canceled");
    expect(taskStatusLabel(null)).toBe("Unknown");
  });

  it("labels priorities", () => {
    expect(taskPriorityLabel("low")).toBe("Low");
    expect(taskPriorityLabel("urgent")).toBe("Urgent");
    expect(taskPriorityLabel(null)).toBe("Unset");
  });

  it("labels inbox lanes", () => {
    expect(taskInboxLaneLabel("my_work")).toBe("My Work");
    expect(taskInboxLaneLabel("approvals")).toBe("Approvals");
    expect(taskInboxLaneLabel("failed_runs")).toBe("Failed Runs");
    expect(taskInboxLaneLabel("blocked")).toBe("Blocked");
    expect(taskInboxLaneLabel("archived")).toBe("Archived");
  });

  it("labels approval states", () => {
    expect(taskApprovalStateLabel("pending")).toBe("Pending Approval");
    expect(taskApprovalStateLabel("approved")).toBe("Approved");
    expect(taskApprovalStateLabel(undefined)).toBe("Not Required");
  });
});

describe("task semantic tones", () => {
  it("Should resolve task statuses through TASK_STATUS_TONE PillTone dictionary", () => {
    expect(taskStatusTone("completed")).toBe("success");
    expect(taskStatusTone("failed")).toBe("danger");
    expect(taskStatusTone("canceled")).toBe("neutral");
    expect(taskStatusTone("in_progress")).toBe("info");
    expect(taskStatusTone("blocked")).toBe("danger");
    expect(taskStatusTone("ready")).toBe("neutral");
    expect(taskStatusTone("draft")).toBe("neutral");
    expect(taskStatusTone("pending")).toBe("neutral");
    expect(taskStatusTone(undefined)).toBe("neutral");
    expect(taskStatusTone(null)).toBe("neutral");
  });

  it("Should always resolve priority to neutral — priority never colorizes", () => {
    expect(taskPriorityTone("urgent")).toBe("neutral");
    expect(taskPriorityTone("high")).toBe("neutral");
    expect(taskPriorityTone("medium")).toBe("neutral");
    expect(taskPriorityTone("low")).toBe("neutral");
    expect(taskPriorityTone(undefined)).toBe("neutral");
  });

  it("Should resolve run statuses through RUN_STATUS_TONE PillTone dictionary", () => {
    expect(taskRunStatusTone("running")).toBe("info");
    expect(taskRunStatusTone("completed")).toBe("success");
    expect(taskRunStatusTone("failed")).toBe("danger");
    expect(taskRunStatusTone("canceled")).toBe("neutral");
    expect(taskRunStatusTone("queued")).toBe("neutral");
    expect(taskRunStatusTone("starting")).toBe("info");
    expect(taskRunStatusTone("claimed")).toBe("info");
    expect(taskRunStatusTone(null)).toBe("neutral");
  });

  it("Should resolve inbox lanes through TASK_LANE_TONE — approvals collapses to info", () => {
    expect(taskLaneTone("approvals")).toBe("info");
    expect(taskLaneTone("failed_runs")).toBe("danger");
    expect(taskLaneTone("blocked")).toBe("danger");
    expect(taskLaneTone("archived")).toBe("neutral");
    expect(taskLaneTone("my_work")).toBe("neutral");
  });
});

describe("task-formatters source — STATUS_TONE migration exhaustiveness", () => {
  const formatterSource = readFileSync(
    resolve(dirname(fileURLToPath(import.meta.url)), "../task-formatters.ts"),
    "utf8"
  );

  it("Should not contain legacy tone identifiers (violet / amber / stuck / green) anywhere in the formatters source", () => {
    expect(formatterSource).not.toMatch(/\bviolet\b/);
    expect(formatterSource).not.toMatch(/\bamber\b/);
    expect(formatterSource).not.toMatch(/\bstuck\b/);
    expect(formatterSource).not.toMatch(/\bgreen\b/);
  });

  function extractFunctionBody(source: string, name: string): string {
    const start = source.indexOf(`export function ${name}(`);
    if (start < 0) throw new Error(`function ${name} not found`);
    let depth = 0;
    let openSeen = false;
    for (let i = start; i < source.length; i += 1) {
      const char = source[i];
      if (char === "{") {
        depth += 1;
        openSeen = true;
      } else if (char === "}") {
        depth -= 1;
        if (openSeen && depth === 0) {
          return source.slice(start, i + 1);
        }
      }
    }
    throw new Error(`function ${name} body not balanced`);
  }

  it("Should not contain inlined `status -> tone` switch statements inside the three migrated function bodies", () => {
    expect(extractFunctionBody(formatterSource, "taskStatusTone")).not.toMatch(/\bswitch\b/);
    expect(extractFunctionBody(formatterSource, "taskRunStatusTone")).not.toMatch(/\bswitch\b/);
    expect(extractFunctionBody(formatterSource, "taskLaneTone")).not.toMatch(/\bswitch\b/);
  });

  it("Should consume the central STATUS_TONE dictionaries from `web/src/lib/status-tone.ts`", () => {
    expect(formatterSource).toMatch(/TASK_STATUS_TONE/);
    expect(formatterSource).toMatch(/RUN_STATUS_TONE/);
    expect(formatterSource).toMatch(/TASK_LANE_TONE/);
    expect(formatterSource).toMatch(/from "@\/lib\/status-tone"/);
    expect(extractFunctionBody(formatterSource, "taskStatusTone")).toMatch(/TASK_STATUS_TONE/);
    expect(extractFunctionBody(formatterSource, "taskRunStatusTone")).toMatch(/RUN_STATUS_TONE/);
    expect(extractFunctionBody(formatterSource, "taskLaneTone")).toMatch(/TASK_LANE_TONE/);
  });
});

describe("task predicates and counts", () => {
  it("detects draft, blocked, and approval-pending tasks", () => {
    expect(taskIsDraft(makeTask({ draft: true }))).toBe(true);
    expect(taskIsDraft(makeTask({ status: "draft" }))).toBe(true);
    expect(taskIsDraft(makeTask())).toBe(false);
    expect(taskIsBlocked(makeTask({ status: "blocked" }))).toBe(true);
    expect(taskIsBlocked(makeTask())).toBe(false);
    expect(taskHasApprovalPending(makeTask({ approval_state: "pending" }))).toBe(true);
    expect(taskHasApprovalPending(makeTask({ approval_state: "approved" }))).toBe(false);
  });

  it("matches queries by title and identifier", () => {
    const task = makeTask({ title: "Review PR", identifier: "TASK-42" });

    expect(matchesTaskQuery(task, "")).toBe(true);
    expect(matchesTaskQuery(task, "review")).toBe(true);
    expect(matchesTaskQuery(task, "TASK-42")).toBe(true);
    expect(matchesTaskQuery(task, "missing")).toBe(false);
  });

  it("formats owner labels with kind fallbacks", () => {
    expect(taskOwnerKindLabel("agent_session")).toBe("Agent");
    expect(taskOwnerKindLabel("network_peer")).toBe("Peer");
    expect(taskOwnerKindLabel(null)).toBe("Unassigned");
    expect(taskOwnerLabel(null)).toBe("Unassigned");
    expect(taskOwnerLabel({ kind: "agent_session", ref: "Coder" })).toBe("Coder");
    expect(taskOwnerLabel({ kind: "agent_session", ref: "" })).toBe("Agent");
  });

  it("formats relative time and attempt labels", () => {
    const now = new Date("2026-04-11T10:00:00Z");
    expect(formatRelativeTime("2026-04-11T09:59:30Z", now)).toBe("now");
    expect(formatRelativeTime("2026-04-11T09:30:00Z", now)).toBe("30m");
    expect(formatRelativeTime("2026-04-11T08:00:00Z", now)).toBe("2h");
    expect(formatRelativeTime("2026-04-09T10:00:00Z", now)).toBe("2d");
    expect(formatRelativeTime(null)).toBe("—");

    expect(formatAttemptLabel(2, 3)).toBe("attempt 2 of 3");
    expect(formatAttemptLabel(1)).toBe("attempt 1");
    expect(formatAttemptLabel(null)).toBeNull();
  });

  it("formats durations and percentages for dashboard metrics", () => {
    expect(formatDurationMs(0)).toBe("0ms");
    expect(formatDurationMs(450)).toBe("450ms");
    expect(formatDurationMs(12_000)).toBe("12s");
    expect(formatDurationMs(167_000)).toBe("2m 47s");
    expect(formatDurationMs(3_600_000)).toBe("1h");
    expect(formatDurationMs(3_900_000)).toBe("1h 5m");
    expect(formatDurationMs(null)).toBe("—");
    expect(formatDurationMs(-10)).toBe("—");

    expect(formatPercent(43)).toBe("43%");
    expect(formatPercent(100)).toBe("100%");
    expect(formatPercent(120)).toBe("100%");
    expect(formatPercent(-5)).toBe("0%");
    expect(formatPercent(null)).toBe("—");
  });

  it("counts tasks by status", () => {
    const counts = countTasksByStatus([
      makeTask({ status: "ready" }),
      makeTask({ status: "ready" }),
      makeTask({ status: "failed" }),
    ]);

    expect(counts.ready).toBe(2);
    expect(counts.failed).toBe(1);
    expect(counts.draft).toBe(0);
  });
});

describe("task lifecycle phases — manual-first signaling", () => {
  it("treats draft tasks without runs as saved intent, not executable", () => {
    const phase = taskLifecyclePhase(makeTask({ status: "draft", draft: true, active_run: null }));
    expect(phase).toBe("saved_intent");
    expect(taskLifecyclePhaseLabel(phase)).toBe("Saved intent");
    expect(taskLifecyclePhaseDescription(phase)).toMatch(/saved intent/i);
    expect(taskLifecyclePhaseDescription(phase)).toMatch(/coordinator/i);
  });

  it("treats ready tasks without runs as ready_to_start, not running", () => {
    const phase = taskLifecyclePhase(makeTask({ status: "ready", active_run: null }));
    expect(phase).toBe("ready_to_start");
    expect(taskLifecyclePhaseDescription(phase)).toMatch(/start enqueues/i);
  });

  it("uses the active run to tell queued from running", () => {
    const queued = taskLifecyclePhase(
      makeTask({
        status: "in_progress",
        active_run: makeRun("queued"),
      } as Partial<TaskListItem>)
    );
    const running = taskLifecyclePhase(
      makeTask({
        status: "in_progress",
        active_run: makeRun("running"),
      } as Partial<TaskListItem>)
    );

    expect(queued).toBe("queued");
    expect(running).toBe("running");
    expect(taskLifecyclePhaseLabel(queued)).toBe("Coordinator handoff");
    expect(taskLifecyclePhaseLabel(running)).toBe("Running");
  });

  it("treats agent-created approval-pending tasks as awaiting approval", () => {
    const phase = taskLifecyclePhase(
      makeTask({
        status: "blocked",
        approval_policy: "manual",
        approval_state: "pending",
        active_run: null,
      })
    );

    expect(phase).toBe("awaiting_approval");
    expect(taskLifecyclePhaseDescription(phase)).toMatch(/approving enqueues/i);
  });

  it("falls back to terminal phases without inferring activity from status", () => {
    expect(taskLifecyclePhase(makeTask({ status: "completed", active_run: null }))).toBe(
      "completed"
    );
    expect(taskLifecyclePhase(makeTask({ status: "failed", active_run: null }))).toBe("failed");
    expect(taskLifecyclePhase(makeTask({ status: "canceled", active_run: null }))).toBe("canceled");
    expect(taskLifecyclePhase(makeTask({ status: "blocked", active_run: null }))).toBe("blocked");
  });

  it("Should never mark saved intent or ready as activity in lifecycle tones", () => {
    expect(taskLifecyclePhaseTone("saved_intent")).toBe("neutral");
    expect(taskLifecyclePhaseTone("ready_to_start")).toBe("neutral");
    expect(taskLifecyclePhaseTone("queued")).toBe("neutral");
    expect(taskLifecyclePhaseTone("running")).toBe("accent");
    expect(taskLifecyclePhaseTone("awaiting_approval")).toBe("info");
    expect(taskLifecyclePhaseTone("blocked")).toBe("danger");
    expect(taskLifecyclePhaseTone("failed")).toBe("danger");
    expect(taskLifecyclePhaseTone("canceled")).toBe("danger");
    expect(taskLifecyclePhaseTone("completed")).toBe("neutral");
  });
});

describe("task handoff actions — boundary semantics", () => {
  it("draft tasks resolve to publish", () => {
    const action = taskHandoffActionKey(makeTask({ status: "draft", draft: true }));
    expect(action).toBe("publish");
    expect(taskHandoffActionCopy(action).label).toBe("Publish");
    expect(taskHandoffActionCopy(action).tooltip).toMatch(/coordinator handoff/i);
  });

  it("approval-pending tasks resolve to approve, never start", () => {
    const action = taskHandoffActionKey(
      makeTask({ approval_policy: "manual", approval_state: "pending", status: "blocked" })
    );
    expect(action).toBe("approve");
    expect(taskHandoffActionCopy(action).tooltip).toMatch(/coordinator handoff/i);
  });

  it("ready tasks resolve to start with coordinator handoff tooltip", () => {
    const action = taskHandoffActionKey(makeTask({ status: "ready", active_run: null }));
    expect(action).toBe("start");
    expect(taskHandoffActionCopy(action).label).toBe("Start run");
    expect(taskHandoffActionCopy(action).tooltip).toMatch(/coordinator handoff/i);
  });

  it("failed tasks expose retry as the executable action", () => {
    expect(taskHandoffActionKey(makeTask({ status: "failed" }))).toBe("retry");
  });

  it("never returns publish/start for terminal completed tasks", () => {
    expect(taskHandoffActionKey(makeTask({ status: "completed" }))).toBe("edit");
    expect(taskHandoffActionKey(makeTask({ status: "canceled" }))).toBe("edit");
  });
});

describe("coordination channel signal", () => {
  it("recognises runs with coordination_channel_id as coordinated", () => {
    const run = {
      coordination_channel_id: "coord-task-001",
    } as TaskRun;

    expect(runIsCoordinated(run)).toBe(true);
    expect(runCoordinationChannelLabel(run)).toBe("coord-task-001");
  });

  it("prefers the embedded display name when available", () => {
    const run = {
      coordination_channel_id: "coord-task-001",
      coordination_channel: {
        id: "coord-task-001",
        display_name: "TASK-1 coordination",
      },
    } as unknown as TaskRun;

    expect(runIsCoordinated(run)).toBe(true);
    expect(runCoordinationChannelLabel(run)).toBe("TASK-1 coordination");
  });

  it("ignores runs without channel binding", () => {
    expect(runIsCoordinated(null)).toBe(false);
    expect(runIsCoordinated({} as TaskRun)).toBe(false);
    expect(runCoordinationChannelLabel(null)).toBe("");
  });
});

function makeRun(status: TaskRun["status"]): TaskListItem["active_run"] {
  return {
    id: "run_test",
    task_id: "task_test",
    attempt: 1,
    status,
    queued_at: "2026-04-17T09:58:00Z",
  } as TaskListItem["active_run"];
}
