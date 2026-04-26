import { describe, expect, it } from "vitest";

import { runIsCoordinated, taskHandoffActionKey, taskLifecyclePhase } from "../lib/task-formatters";
import {
  awaitingApprovalTaskFixture,
  coordinatorEnabledWorkspaceFixture,
  queuedCoordinatedTaskFixture,
  savedIntentTaskFixture,
  TASK_FIXTURES,
} from "./fixtures";

describe("tasks fixtures cover the manual-first lifecycle states", () => {
  it("savedIntentTaskFixture is a draft with no run and resolves to publish", () => {
    expect(savedIntentTaskFixture.status).toBe("draft");
    expect(savedIntentTaskFixture.draft).toBe(true);
    expect(savedIntentTaskFixture.active_run).toBeNull();
    expect(taskLifecyclePhase(savedIntentTaskFixture)).toBe("saved_intent");
    expect(taskHandoffActionKey(savedIntentTaskFixture)).toBe("publish");
  });

  it("awaitingApprovalTaskFixture is agent-created, gated, and resolves to approve", () => {
    expect(awaitingApprovalTaskFixture.approval_policy).toBe("manual");
    expect(awaitingApprovalTaskFixture.approval_state).toBe("pending");
    expect(awaitingApprovalTaskFixture.active_run).toBeNull();
    expect(awaitingApprovalTaskFixture.created_by?.kind).toBe("agent_session");
    expect(taskLifecyclePhase(awaitingApprovalTaskFixture)).toBe("awaiting_approval");
    expect(taskHandoffActionKey(awaitingApprovalTaskFixture)).toBe("approve");
  });

  it("queuedCoordinatedTaskFixture has a queued run bound to a coordination channel", () => {
    expect(queuedCoordinatedTaskFixture.active_run?.status).toBe("queued");
    expect(runIsCoordinated(queuedCoordinatedTaskFixture.active_run)).toBe(true);
    expect(queuedCoordinatedTaskFixture.active_run?.coordination_channel?.id).toBe(
      "coord-task-queued"
    );
    expect(taskLifecyclePhase(queuedCoordinatedTaskFixture)).toBe("queued");
  });

  it("TASK_FIXTURES still cover user-created, running, failed, and approval states", () => {
    const statuses = TASK_FIXTURES.map(task => task.status);
    expect(statuses).toEqual(
      expect.arrayContaining(["in_progress", "pending", "failed", "completed", "blocked", "ready"])
    );
    expect(TASK_FIXTURES.some(task => task.approval_state === "pending")).toBe(true);
    expect(TASK_FIXTURES.some(task => task.active_run?.status === "running")).toBe(true);
    expect(TASK_FIXTURES.some(task => task.active_run?.status === "failed")).toBe(true);
  });

  it("coordinatorEnabledWorkspaceFixture marks the workspace as coordinator-enabled", () => {
    expect(coordinatorEnabledWorkspaceFixture.coordinatorEnabled).toBe(true);
    expect(coordinatorEnabledWorkspaceFixture.coordinatorAgentName).toBe("coordinator");
    expect(coordinatorEnabledWorkspaceFixture.defaultChannelDisplayName).toMatch(/coordination/i);
  });
});
