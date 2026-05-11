import type { StatusDotProps, StatusDotTone, StatusDotVariant } from "@agh/ui";

import type { TaskInboxItem, TaskInboxLane } from "../types";

/**
 * UI-only inbox lane vocabulary per ADR-006 §5 — five lanes in declared order
 * (`My work · Mentions · Failed runs · Updates · Approvals`). The backend
 * inbox query also exposes `blocked` and `archived` lanes; both flow into
 * `My work` when present so the lane switcher matches the proposal exactly.
 */
export type InboxUiLane = "my_work" | "mentions" | "failed_runs" | "updates" | "approvals";

export type InboxLaneFilterId = "all" | InboxUiLane;

export interface InboxLaneDefinition {
  id: InboxUiLane;
  label: string;
}

export const INBOX_UI_LANES: InboxLaneDefinition[] = [
  { id: "my_work", label: "My work" },
  { id: "mentions", label: "Mentions" },
  { id: "failed_runs", label: "Failed runs" },
  { id: "updates", label: "Updates" },
  { id: "approvals", label: "Approvals" },
];

/**
 * UI-only inbox group vocabulary per ADR-006 §3 — five groups with a dot tone
 * each (warning solid / danger solid / warning ring / accent solid / faint
 * ring). Group membership is derived from backend item shape via
 * `resolveInboxGroupId`.
 */
export type InboxGroupId = "needs_review" | "blocked" | "stuck" | "mentions" | "updates";

export interface InboxGroupDefinition {
  id: InboxGroupId;
  label: string;
  dotTone: StatusDotTone;
  dotVariant: StatusDotVariant;
}

export const INBOX_GROUPS: InboxGroupDefinition[] = [
  { id: "needs_review", label: "Needs review", dotTone: "warning", dotVariant: "solid" },
  { id: "blocked", label: "Blocked", dotTone: "danger", dotVariant: "solid" },
  { id: "stuck", label: "Stuck", dotTone: "warning", dotVariant: "ring" },
  { id: "mentions", label: "Mentions", dotTone: "accent", dotVariant: "solid" },
  { id: "updates", label: "Updates", dotTone: "faint", dotVariant: "ring" },
];

/** Convenience accessor for `<StatusDot>` props derived from a group id. */
export function inboxGroupDotProps(group: InboxGroupId): Pick<StatusDotProps, "tone" | "variant"> {
  const definition = INBOX_GROUPS.find(entry => entry.id === group);
  return {
    tone: definition?.dotTone ?? "faint",
    variant: definition?.dotVariant ?? "ring",
  };
}

/**
 * Routes an inbox item into one of the five UI groups. Mapping rules:
 *
 * - Approval pending + `approvals` lane → `needs_review`.
 * - `blocked` lane OR task status `blocked` → `blocked`.
 * - Backend `archived` items + read state → `updates` (low-priority feed).
 * - `failed_runs` lane → `needs_review` (failures require operator action).
 * - Anything else with `my_work` lane → `updates` (informational).
 *
 * `stuck` and `mentions` have no backing signal in the MVP and stay empty
 * until follow-up work lights them up (per techspec MVP boundary).
 */
export function resolveInboxGroupId(item: TaskInboxItem): InboxGroupId {
  if (item.lane === "approvals") {
    return "needs_review";
  }
  if (item.lane === "blocked" || item.task.status === "blocked") {
    return "blocked";
  }
  if (item.lane === "failed_runs") {
    return "needs_review";
  }
  return "updates";
}

/** Maps a backend inbox lane onto the UI lane vocabulary. */
export function backendLaneToUiLane(lane: TaskInboxLane): InboxUiLane {
  switch (lane) {
    case "approvals":
      return "approvals";
    case "failed_runs":
      return "failed_runs";
    default:
      // `blocked`, `archived`, `my_work` collapse into `My work` per ADR-006 §5
      // — the proposal kanban surfaces all backend lanes without a dedicated
      // archive switch.
      return "my_work";
  }
}
