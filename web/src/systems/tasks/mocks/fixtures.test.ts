import { describe, expect, it } from "vitest";

import { storyAgentNames, storyCoordinatorAgentName } from "@/storybook/fintech-scenario";
import { runIsCoordinated, taskHandoffActionKey, taskLifecyclePhase } from "../lib/task-formatters";
import {
  agentContextFixture,
  awaitingApprovalTaskFixture,
  buildBridgeNotificationCursorFixture,
  buildTaskBridgeNotificationSubscriptionFixture,
  buildTaskContextBundleFixture,
  buildTaskExecutionProfileFixture,
  buildTaskRunReviewFixture,
  buildTaskRunReviewVerdictResultFixture,
  coordinatorEnabledWorkspaceFixture,
  queuedCoordinatedTaskFixture,
  savedIntentTaskFixture,
  taskBridgeNotificationSubscriptionFixture,
  taskBridgeNotificationSubscriptionsFixture,
  taskContextBundleFixture,
  taskExecutionProfileFixture,
  taskRunReviewFixture,
  taskRunReviewListFixture,
  taskRunReviewVerdictResultFixture,
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
    const owners = TASK_FIXTURES.flatMap(task =>
      task.owner?.kind === "agent_session" ? [task.owner.ref] : []
    );
    expect(statuses).toEqual(
      expect.arrayContaining(["in_progress", "pending", "failed", "completed", "blocked", "ready"])
    );
    expect(TASK_FIXTURES.length).toBeGreaterThanOrEqual(15);
    expect(TASK_FIXTURES.some(task => task.approval_state === "pending")).toBe(true);
    expect(TASK_FIXTURES.some(task => task.active_run?.status === "running")).toBe(true);
    expect(TASK_FIXTURES.some(task => task.active_run?.status === "failed")).toBe(true);
    expect(owners).toEqual(
      expect.arrayContaining([
        storyAgentNames.product,
        storyAgentNames.frontend,
        storyAgentNames.cfo,
        storyAgentNames.marketing,
        storyAgentNames.copywriter,
      ])
    );
  });

  it("coordinatorEnabledWorkspaceFixture marks the workspace as coordinator-enabled", () => {
    expect(coordinatorEnabledWorkspaceFixture.coordinatorEnabled).toBe(true);
    expect(coordinatorEnabledWorkspaceFixture.coordinatorAgentName).toBe(storyCoordinatorAgentName);
    expect(coordinatorEnabledWorkspaceFixture.defaultChannelDisplayName).toMatch(/coordination/i);
  });
});

describe("orchestration fixtures satisfy generated contract shape", () => {
  it("Should expose all execution profile selector branches", () => {
    expect(taskExecutionProfileFixture.task_id).toBeTypeOf("string");
    expect(taskExecutionProfileFixture.coordinator.mode).toMatch(/inherit|guided/);
    expect(taskExecutionProfileFixture.worker.mode).toMatch(/inherit|select/);
    expect(taskExecutionProfileFixture.sandbox.mode).toMatch(/inherit|none|ref/);
    expect(taskExecutionProfileFixture.created_at).toBeTypeOf("string");
    expect(taskExecutionProfileFixture.updated_at).toBeTypeOf("string");

    const overlay = buildTaskExecutionProfileFixture({ task_id: "task_42" });
    expect(overlay.task_id).toBe("task_42");
  });

  it("Should expose review fixtures with status, policy, and cursor diagnostics", () => {
    expect(taskRunReviewFixture.review_id).toBeTypeOf("string");
    expect(taskRunReviewFixture.status).toBe("in_review");
    expect(taskRunReviewListFixture).toHaveLength(2);
    expect(taskRunReviewListFixture[1]?.outcome).toBe("rejected");
    const built = buildTaskRunReviewFixture({ status: "recorded", outcome: "approved" });
    expect(built.status).toBe("recorded");
    expect(built.outcome).toBe("approved");
  });

  it("Should expose verdict fixture with continuation run lineage", () => {
    expect(taskRunReviewVerdictResultFixture.review.outcome).toBe("rejected");
    expect(taskRunReviewVerdictResultFixture.continuation_run?.attempt).toBeGreaterThan(0);
    expect(taskRunReviewVerdictResultFixture.continuation_run?.task_id).toBe(
      taskRunReviewVerdictResultFixture.review.task_id
    );
    const reset = buildTaskRunReviewVerdictResultFixture({ circuit_opened: true });
    expect(reset.circuit_opened).toBe(true);
  });

  it("Should expose bridge subscription fixtures with cursor diagnostics", () => {
    expect(taskBridgeNotificationSubscriptionFixture.cursor.consumer_id).toContain(
      "bridge_task_subscription:"
    );
    expect(taskBridgeNotificationSubscriptionFixture.cursor.stream_name).toBe("task_events");
    expect(taskBridgeNotificationSubscriptionFixture.cursor.last_sequence).toBeGreaterThanOrEqual(
      0
    );
    expect(taskBridgeNotificationSubscriptionsFixture).toHaveLength(2);

    const fresh = buildBridgeNotificationCursorFixture({ last_sequence: 0 });
    expect(fresh.last_sequence).toBe(0);

    const customSub = buildTaskBridgeNotificationSubscriptionFixture({
      subscription_id: "bsub_custom",
    });
    expect(customSub.cursor.consumer_id).toBe("bridge_task_subscription:bsub_custom");
  });

  it("Should expose task context bundle with latest_event_seq and execution profile", () => {
    expect(taskContextBundleFixture.latest_event_seq).toBeGreaterThanOrEqual(0);
    expect(taskContextBundleFixture.task.id).toBeTypeOf("string");
    expect(taskContextBundleFixture.execution_profile?.coordinator.mode).toMatch(/inherit|guided/);
    const variant = buildTaskContextBundleFixture({ latest_event_seq: 99 });
    expect(variant.latest_event_seq).toBe(99);
  });

  it("Should expose agent context fixture pointing at the task bundle", () => {
    expect(agentContextFixture.task.available).toBe(true);
    expect(agentContextFixture.task.bundle?.task.id).toBe("task_001");
    expect(agentContextFixture.task.bundle?.latest_event_seq).toBeGreaterThanOrEqual(0);
  });
});
