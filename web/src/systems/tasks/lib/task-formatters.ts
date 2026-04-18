import type {
  TaskApprovalState,
  TaskInboxLane,
  TaskListItem,
  TaskOwnerKind,
  TaskPriority,
  TaskRunStatus,
  TaskStatus,
} from "../types";

export type TaskSemanticTone = "amber" | "danger" | "green" | "neutral" | "violet";

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

export function taskStatusTone(status?: TaskStatus | null): TaskSemanticTone {
  switch (status) {
    case "completed":
      return "green";
    case "failed":
    case "canceled":
      return "danger";
    case "in_progress":
    case "ready":
      return "violet";
    case "blocked":
      return "amber";
    case "draft":
    case "pending":
    default:
      return "neutral";
  }
}

export function taskPriorityTone(priority?: TaskPriority | null): TaskSemanticTone {
  switch (priority) {
    case "urgent":
      return "danger";
    case "high":
      return "amber";
    case "medium":
      return "violet";
    case "low":
      return "neutral";
    default:
      return "neutral";
  }
}

export function taskRunStatusTone(status?: TaskRunStatus | null): TaskSemanticTone {
  switch (status) {
    case "completed":
      return "green";
    case "failed":
    case "canceled":
      return "danger";
    case "running":
    case "starting":
    case "claimed":
      return "violet";
    case "queued":
      return "amber";
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
