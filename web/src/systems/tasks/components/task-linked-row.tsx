import { Link } from "@tanstack/react-router";
import { ChevronRight } from "lucide-react";

import { cn, OwnerAvatar, Time } from "@agh/ui";

import { ownerAvatarKindFor, taskOwnerLabel } from "../lib/task-formatters";
import type { TaskChildSummary } from "../types";

export type TaskLinkedRowState = "done" | "active" | "todo";

export interface TaskLinkedRowProps {
  taskId: string;
  title: string;
  state: TaskLinkedRowState;
  owner?: TaskChildSummary["owner"] | null;
  timeIso?: string | null;
  testId?: string;
}

const DOT_CLASS: Record<TaskLinkedRowState, string> = {
  done: "bg-success",
  active: "border-[1.5px] border-accent bg-transparent",
  todo: "border-[1.5px] border-faint bg-transparent",
};

/**
 * Shared row anatomy for subtasks and dependencies (§4.3): status dot, title,
 * owner, freshness, chevron. The whole row is the link — no per-row accent
 * button (accent budget stays with the head primary).
 */
export function TaskLinkedRow({
  taskId,
  title,
  state,
  owner,
  timeIso,
  testId,
}: TaskLinkedRowProps) {
  const ownerName = owner ? taskOwnerLabel(owner) : null;
  return (
    <Link
      className={cn(
        "grid grid-cols-[14px_minmax(0,1fr)_auto_14px] items-center gap-3 px-4 py-2.5",
        "border-t border-line-soft transition-colors duration-fast first:border-t-0 hover:bg-row-hover",
        "focus-visible:outline-none focus-visible:shadow-focus-ring",
        timeIso ? "sm:grid-cols-[14px_minmax(0,1fr)_auto_auto_14px]" : null
      )}
      data-testid={testId}
      params={{ id: taskId }}
      to="/tasks/$id"
    >
      <span
        aria-hidden="true"
        className={cn("size-2 justify-self-center rounded-full", DOT_CLASS[state])}
        data-state={state}
      />
      <span className="truncate text-ws-name font-medium text-fg-strong">
        {state === "done" ? <s className="text-muted decoration-faint">{title}</s> : title}
      </span>
      {ownerName && owner ? (
        <span className="inline-flex min-w-0 items-center gap-1.5 text-form-label text-muted">
          <OwnerAvatar
            name={ownerName}
            ownerId={owner.ref ?? ownerName}
            ownerKind={ownerAvatarKindFor(owner.kind)}
            size="sm"
          />
          <span className="hidden truncate sm:inline">{ownerName}</span>
        </span>
      ) : (
        <span aria-hidden="true" />
      )}
      {timeIso ? (
        <span className="hidden text-eyebrow tabular-nums text-subtle sm:inline">
          <Time iso={timeIso} mode="relative" />
        </span>
      ) : null}
      <ChevronRight aria-hidden="true" className="size-3.5 text-faint" />
    </Link>
  );
}
