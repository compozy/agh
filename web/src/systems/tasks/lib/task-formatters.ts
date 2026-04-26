import type { StatusDotTone } from "@agh/ui";

import type {
  TaskApprovalState,
  TaskInboxLane,
  TaskListItem,
  TaskOwnerKind,
  TaskPriority,
  TaskRecord,
  TaskRun,
  TaskRunStatus,
  TaskStatus,
} from "../types";

export type TaskSemanticTone = "accent" | "amber" | "danger" | "green" | "neutral" | "violet";

export interface TaskStatusSignal {
  tone: StatusDotTone;
  pulse?: boolean;
}

/**
 * Maps a task status (production vocabulary OR the `docs/design/web-inspiration/`
 * shorthand of `done | running | pending | blocked | failed`) to the DESIGN.md §4
 * `StatusDot` tone and pulse. Used by `tasks-list-row`, detail header, kanban cards,
 * inbox rows and table cells so that the visual signal stays consistent.
 *
 * Color is signal, never decoration: terminal / normal states render neutral,
 * only attention-demanding states carry a semantic tone. The `accent` + `pulse`
 * combination is reserved for genuinely running work.
 */
export function taskStatusSignal(status?: TaskStatus | string | null): TaskStatusSignal {
  switch (status) {
    case "in_progress":
    case "running":
      return { tone: "accent", pulse: true };
    case "blocked":
      return { tone: "warning" };
    case "failed":
    case "canceled":
      return { tone: "danger" };
    case "completed":
    case "done":
    case "ready":
    case "pending":
    case "draft":
    default:
      return { tone: "neutral" };
  }
}

/** Convenience: short 7-char identifier for `MonoBadge` id chips in list rows. */
export function taskShortId(task: { id: string; identifier?: string | null }): string {
  if (task.identifier) return task.identifier;
  return task.id.length > 7 ? task.id.slice(0, 7) : task.id;
}

const TASK_STATUS_LABELS: Record<TaskStatus, string> = {
  draft: "Draft",
  pending: "Pending",
  blocked: "Blocked",
  ready: "Ready",
  in_progress: "In Progress",
  completed: "Completed",
  failed: "Failed",
  canceled: "Canceled",
};

const TASK_PRIORITY_LABELS: Record<TaskPriority, string> = {
  low: "Low",
  medium: "Medium",
  high: "High",
  urgent: "Urgent",
};

const TASK_INBOX_LANE_LABELS: Record<TaskInboxLane, string> = {
  my_work: "My Work",
  approvals: "Approvals",
  failed_runs: "Failed Runs",
  blocked: "Blocked",
  archived: "Archived",
};

const TASK_APPROVAL_STATE_LABELS: Record<TaskApprovalState, string> = {
  not_required: "Not Required",
  pending: "Pending Approval",
  approved: "Approved",
  rejected: "Rejected",
};

export function taskStatusLabel(status?: TaskStatus | null): string {
  if (!status) {
    return "Unknown";
  }

  return TASK_STATUS_LABELS[status] ?? status;
}

export function taskPriorityLabel(priority?: TaskPriority | null): string {
  if (!priority) {
    return "Unset";
  }

  return TASK_PRIORITY_LABELS[priority] ?? priority;
}

export function taskInboxLaneLabel(lane: TaskInboxLane): string {
  return TASK_INBOX_LANE_LABELS[lane] ?? lane;
}

export function taskApprovalStateLabel(state?: TaskApprovalState | null): string {
  if (!state) {
    return "Not Required";
  }

  return TASK_APPROVAL_STATE_LABELS[state] ?? state;
}

/**
 * Terminal + normal statuses (`completed`, `ready`, `draft`, `pending`, `idle`,
 * non-live `in_progress`) resolve to `neutral`. Attention-demanding statuses
 * keep their semantic tone. Priority never colorizes — hierarchy is driven by
 * weight and position, not hue (see `taskPriorityTone`).
 */
export function taskStatusTone(status?: TaskStatus | null): TaskSemanticTone {
  switch (status) {
    case "failed":
    case "canceled":
      return "danger";
    case "blocked":
      return "amber";
    case "in_progress":
      // In-flight work is only accent when its run is genuinely live; the list/
      // table surfaces cannot check the active run from the status alone, so
      // the default neutral keeps the row calm. The Agents panel and detail
      // header override this locally when `liveCount > 0`.
      return "neutral";
    case "completed":
    case "ready":
    case "draft":
    case "pending":
    default:
      return "neutral";
  }
}

export function taskPriorityTone(_priority?: TaskPriority | null): TaskSemanticTone {
  // Priority is always neutral. Hierarchy is expressed via weight and position,
  // not color — stacking amber/violet/danger per row is decoration, not signal.
  return "neutral";
}

export function taskRunStatusTone(status?: TaskRunStatus | null): TaskSemanticTone {
  switch (status) {
    case "failed":
    case "canceled":
      return "danger";
    case "running":
      return "accent";
    case "queued":
      return "amber";
    case "completed":
    case "starting":
    case "claimed":
    default:
      return "neutral";
  }
}

export function taskLaneTone(lane: TaskInboxLane): TaskSemanticTone {
  switch (lane) {
    case "approvals":
      return "violet";
    case "failed_runs":
      return "danger";
    case "blocked":
      return "amber";
    case "archived":
      return "neutral";
    case "my_work":
    default:
      return "green";
  }
}

export function taskHasApprovalPending(task: Pick<TaskListItem, "approval_state">): boolean {
  return task.approval_state === "pending";
}

export function taskIsDraft(task: Pick<TaskListItem, "draft" | "status">): boolean {
  return task.draft === true || task.status === "draft";
}

export function taskIsBlocked(task: Pick<TaskListItem, "status">): boolean {
  return task.status === "blocked";
}

export function matchesTaskQuery(
  task: Pick<TaskListItem, "title" | "identifier">,
  query: string
): boolean {
  const normalized = query.trim().toLowerCase();
  if (normalized === "") {
    return true;
  }

  const title = task.title?.toLowerCase() ?? "";
  const identifier = task.identifier?.toLowerCase() ?? "";

  return title.includes(normalized) || identifier.includes(normalized);
}

const TASK_OWNER_KIND_LABELS: Record<TaskOwnerKind, string> = {
  human: "Human",
  agent_session: "Agent",
  automation: "Automation",
  extension: "Extension",
  network_peer: "Peer",
  pool: "Pool",
};

export function taskOwnerKindLabel(kind?: TaskOwnerKind | null): string {
  if (!kind) {
    return "Unassigned";
  }

  return TASK_OWNER_KIND_LABELS[kind] ?? kind;
}

export function taskOwnerLabel(
  owner?: Pick<NonNullable<TaskListItem["owner"]>, "kind" | "ref"> | null
): string {
  if (!owner) {
    return "Unassigned";
  }

  return owner.ref || taskOwnerKindLabel(owner.kind);
}

const SECOND = 1000;
const MINUTE = 60 * SECOND;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

export function formatRelativeTime(value?: string | null, now: Date = new Date()): string {
  if (!value) {
    return "—";
  }

  const ts = Date.parse(value);
  if (Number.isNaN(ts)) {
    return "—";
  }

  const delta = Math.max(0, now.getTime() - ts);
  if (delta < MINUTE) {
    return "now";
  }

  if (delta < HOUR) {
    const minutes = Math.floor(delta / MINUTE);
    return `${minutes}m`;
  }

  if (delta < DAY) {
    const hours = Math.floor(delta / HOUR);
    return `${hours}h`;
  }

  const days = Math.floor(delta / DAY);
  return `${days}d`;
}

export function formatDurationMs(ms?: number | null): string {
  if (typeof ms !== "number" || !Number.isFinite(ms) || ms < 0) {
    return "—";
  }

  if (ms < SECOND) {
    return `${Math.round(ms)}ms`;
  }

  if (ms < MINUTE) {
    const seconds = Math.round(ms / SECOND);
    return `${seconds}s`;
  }

  if (ms < HOUR) {
    const minutes = Math.floor(ms / MINUTE);
    const seconds = Math.floor((ms % MINUTE) / SECOND);
    return seconds === 0 ? `${minutes}m` : `${minutes}m ${seconds}s`;
  }

  const hours = Math.floor(ms / HOUR);
  const minutes = Math.floor((ms % HOUR) / MINUTE);
  return minutes === 0 ? `${hours}h` : `${hours}h ${minutes}m`;
}

export function formatPercent(value?: number | null): string {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return "—";
  }

  const rounded = Math.max(0, Math.min(100, Math.round(value)));
  return `${rounded}%`;
}

export function formatAttemptLabel(current?: number | null, total?: number | null): string | null {
  if (typeof current !== "number") {
    return null;
  }

  if (typeof total === "number" && total > 0) {
    return `attempt ${current} of ${total}`;
  }

  return `attempt ${current}`;
}

export function countTasksByStatus(tasks: TaskListItem[]): Record<TaskStatus, number> {
  const counts: Record<TaskStatus, number> = {
    draft: 0,
    pending: 0,
    blocked: 0,
    ready: 0,
    in_progress: 0,
    completed: 0,
    failed: 0,
    canceled: 0,
  };

  for (const task of tasks) {
    counts[task.status] = (counts[task.status] ?? 0) + 1;
  }

  return counts;
}

export type TaskLifecyclePhase =
  | "saved_intent"
  | "awaiting_approval"
  | "ready_to_start"
  | "queued"
  | "running"
  | "completed"
  | "failed"
  | "canceled"
  | "blocked";

type TaskLifecycleInput = Pick<TaskListItem, "status" | "approval_state" | "draft"> & {
  active_run?: TaskListItem["active_run"] | null;
};

const ACTIVE_RUN_STATUSES = new Set<TaskRunStatus>(["running", "starting", "claimed"]);

/**
 * Resolves the manual-first lifecycle phase for a task.
 *
 * The lifecycle is a UI-only narrative built from the canonical task status,
 * approval state, and the task list `active_run` summary so creation
 * (saved_intent) reads as separate from publish/start/approval handoff
 * (queued/running). Channel availability is a separate concern — see
 * `runIsCoordinated` and `runCoordinationChannelLabel`.
 */
export function taskLifecyclePhase(task: TaskLifecycleInput): TaskLifecyclePhase {
  if (task.status === "completed") {
    return "completed";
  }

  if (task.status === "failed") {
    return "failed";
  }

  if (task.status === "canceled") {
    return "canceled";
  }

  if (taskIsDraft(task)) {
    return "saved_intent";
  }

  if (task.approval_state === "pending") {
    return "awaiting_approval";
  }

  const activeRun = task.active_run;
  if (activeRun) {
    if (activeRun.status && ACTIVE_RUN_STATUSES.has(activeRun.status)) {
      return "running";
    }

    if (activeRun.status === "queued") {
      return "queued";
    }
  }

  if (task.status === "blocked") {
    return "blocked";
  }

  if (task.status === "in_progress") {
    return "running";
  }

  return "ready_to_start";
}

const TASK_LIFECYCLE_PHASE_LABELS: Record<TaskLifecyclePhase, string> = {
  saved_intent: "Saved intent",
  awaiting_approval: "Awaiting approval",
  ready_to_start: "Ready to start",
  queued: "Coordinator handoff",
  running: "Running",
  completed: "Completed",
  failed: "Failed",
  canceled: "Canceled",
  blocked: "Blocked",
};

export function taskLifecyclePhaseLabel(phase: TaskLifecyclePhase): string {
  return TASK_LIFECYCLE_PHASE_LABELS[phase];
}

const TASK_LIFECYCLE_PHASE_DESCRIPTIONS: Record<TaskLifecyclePhase, string> = {
  saved_intent:
    "Task is saved intent. Publish or start to enqueue an executable run for the coordinator.",
  awaiting_approval:
    "Approval gates execution. Approving enqueues an executable run for the coordinator.",
  ready_to_start:
    "Task is ready. Start enqueues a coordinator-handoff run; manual workers may also claim it.",
  queued: "Coordinator handoff is in flight. A worker session will claim this queued run.",
  running:
    "A worker session is executing the active run. Channel messages support coordination only.",
  completed: "The latest run completed. Task ownership and terminal status are durable.",
  failed: "The latest run failed. Retry, cancel, or follow up — channel chatter never owns status.",
  canceled: "The task or its run was canceled.",
  blocked: "Blocked by a dependency or policy. Resolve the blocker before the run can be enqueued.",
};

export function taskLifecyclePhaseDescription(phase: TaskLifecyclePhase): string {
  return TASK_LIFECYCLE_PHASE_DESCRIPTIONS[phase];
}

const TASK_LIFECYCLE_PHASE_TONES: Record<TaskLifecyclePhase, TaskSemanticTone> = {
  saved_intent: "neutral",
  awaiting_approval: "violet",
  ready_to_start: "neutral",
  queued: "amber",
  running: "accent",
  completed: "neutral",
  failed: "danger",
  canceled: "danger",
  blocked: "amber",
};

export function taskLifecyclePhaseTone(phase: TaskLifecyclePhase): TaskSemanticTone {
  return TASK_LIFECYCLE_PHASE_TONES[phase];
}

export type TaskHandoffActionKey =
  | "publish"
  | "approve"
  | "reject"
  | "start"
  | "cancel"
  | "retry"
  | "edit";

/**
 * Picks the operator-facing primary handoff action for a task. UI surfaces use
 * this so that creation (intent) is never represented by an action — only the
 * boundary actions that enqueue an executable run.
 */
export function taskHandoffActionKey(task: TaskLifecycleInput): TaskHandoffActionKey {
  if (taskIsDraft(task)) {
    return "publish";
  }

  if (task.approval_state === "pending") {
    return "approve";
  }

  if (task.status === "failed") {
    return "retry";
  }

  if (task.status === "blocked") {
    return "edit";
  }

  if (task.status === "completed" || task.status === "canceled") {
    return "edit";
  }

  return "start";
}

export interface TaskHandoffActionLabel {
  label: string;
  tooltip: string;
}

const TASK_HANDOFF_ACTION_COPY: Record<TaskHandoffActionKey, TaskHandoffActionLabel> = {
  publish: {
    label: "Publish",
    tooltip:
      "Publish marks the saved intent as ready and enqueues an executable run for coordinator handoff.",
  },
  approve: {
    label: "Approve",
    tooltip:
      "Approve enqueues an executable run for coordinator handoff. Rejecting blocks execution instead.",
  },
  reject: {
    label: "Reject",
    tooltip: "Reject the task. No run is enqueued and the task moves to blocked.",
  },
  start: {
    label: "Start run",
    tooltip:
      "Start enqueues an executable run for coordinator handoff. Manual workers may also claim it.",
  },
  cancel: {
    label: "Cancel",
    tooltip: "Cancel the task. Active runs are released; coordinator stops orchestrating it.",
  },
  retry: {
    label: "Retry",
    tooltip: "Re-enqueue this task as a coordinator-handoff run.",
  },
  edit: {
    label: "Edit",
    tooltip: "Open the editor. Editing keeps the task in saved intent until you publish or start.",
  },
};

export function taskHandoffActionCopy(action: TaskHandoffActionKey): TaskHandoffActionLabel {
  return TASK_HANDOFF_ACTION_COPY[action];
}

type CoordinationCarrier = {
  coordination_channel_id?: string | null;
  coordination_channel?: {
    id?: string | null;
    display_name?: string | null;
    purpose?: string | null;
  } | null;
};

/**
 * Returns true when the run carries a coordination channel binding. Channel
 * presence supports operator/agent conversation; it never replaces task-run
 * ownership or terminal status.
 */
export function runIsCoordinated<T extends CoordinationCarrier | null | undefined>(
  run: T
): boolean {
  if (!run) {
    return false;
  }

  if (typeof run.coordination_channel_id === "string" && run.coordination_channel_id !== "") {
    return true;
  }

  return Boolean(run.coordination_channel?.id);
}

/**
 * Resolves a short human label for a coordination channel binding. Returns the
 * embedded display name when present, then falls back to the channel id, and
 * finally to a generic "Coordination channel" so the chip remains readable
 * even when the embedded payload is missing.
 */
export function runCoordinationChannelLabel<T extends CoordinationCarrier | null | undefined>(
  run: T
): string {
  if (!run) {
    return "";
  }

  const display = run.coordination_channel?.display_name?.trim();
  if (display) {
    return display;
  }

  const embeddedId = run.coordination_channel?.id?.trim();
  if (embeddedId) {
    return embeddedId;
  }

  const id = run.coordination_channel_id?.trim();
  if (id) {
    return id;
  }

  return "Coordination channel";
}

/**
 * Compatibility helpers for callers that have only the `TaskRecord` (without an
 * `active_run`). These let the detail header keep a single source of truth for
 * lifecycle copy without forcing every caller to construct a full `TaskListItem`.
 */
export function taskLifecyclePhaseFromRecord(
  task: Pick<TaskRecord, "status" | "approval_state" | "draft">,
  activeRun?: TaskListItem["active_run"] | null | undefined
): TaskLifecyclePhase {
  return taskLifecyclePhase({
    status: task.status,
    approval_state: task.approval_state,
    draft: task.draft,
    active_run: activeRun ?? null,
  });
}

export type TaskRunLike = Pick<
  TaskRun,
  "coordination_channel_id" | "coordination_channel" | "status"
>;
