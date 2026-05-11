import { AlertCircle } from "lucide-react";
import * as React from "react";

import { Button, OwnerAvatar, type OwnerAvatarProps } from "@agh/ui";

import { cn } from "@/lib/utils";

import { formatRelativeTime, taskOwnerLabel, taskShortId } from "../lib/task-formatters";
import type { TaskListItem, TaskOwnerKind } from "../types";

export interface TaskKanbanCardProps {
  task: TaskListItem;
  selected?: boolean;
  onSelect?: (taskId: string) => void;
  onRetry?: (taskId: string) => void;
}

/**
 * Maps the backend owner kind onto the `<OwnerAvatar>` palette tier
 * §3.5 /. Agent sessions, automation runs, extensions, network peers,
 * and worker pools all read as `agent` for color selection; humans get the
 * `human` slot ladder; unassigned tasks fall back to the system palette.
 */
function ownerAvatarKindFor(kind?: TaskOwnerKind | null): OwnerAvatarProps["ownerKind"] {
  switch (kind) {
    case "human":
      return "human";
    case "agent_session":
    case "automation":
    case "extension":
    case "network_peer":
    case "pool":
      return "agent";
    default:
      return "system";
  }
}

export function TaskKanbanCard({ task, selected = false, onSelect, onRetry }: TaskKanbanCardProps) {
  const activeRun = task.active_run ?? null;
  const isFailed = task.status === "failed";
  const failedError = isFailed && activeRun?.error ? activeRun.error : null;
  const canRetry = isFailed && Boolean(onRetry);

  const identifier = taskShortId(task);
  const ownerLabel = taskOwnerLabel(task.owner);
  const ownerId = task.owner?.ref ?? task.owner?.kind ?? "unassigned";
  const ownerKind = ownerAvatarKindFor(task.owner?.kind);
  const lastActivity = task.last_activity_at ?? task.updated_at;
  const timestamp = formatRelativeTime(lastActivity);

  const clickable = onSelect !== undefined;
  const handleClick = clickable ? () => onSelect?.(task.id) : undefined;
  const handleKeyDown = clickable
    ? (event: React.KeyboardEvent<HTMLDivElement>) => {
        if (event.target !== event.currentTarget) return;
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onSelect?.(task.id);
        }
      }
    : undefined;

  return (
    <div
      role={clickable ? "button" : undefined}
      tabIndex={clickable ? 0 : undefined}
      aria-pressed={clickable ? selected : undefined}
      data-selected={selected ? "true" : undefined}
      data-status={task.status}
      data-testid={`tasks-kanban-card-${task.id}`}
      onClick={handleClick}
      onKeyDown={handleKeyDown}
      className={cn(
        "shadow-[inset_0_0_0_1px_var(--line-soft)]",
        "relative flex w-full min-w-0 flex-col gap-[7px] overflow-hidden rounded-md bg-(--canvas-tint) p-[11px] text-left transition-colors duration-(--dur) ease-(--ease)",
        "hover:bg-(--elevated) hover:shadow-[inset_0_0_0_1px_var(--line)]",
        clickable && "cursor-pointer",
        clickable &&
          "focus-visible:shadow-[inset_0_0_0_1px_var(--line-strong)] focus-visible:outline-none focus-visible:ring-0",
        selected && "bg-(--elevated) shadow-[inset_0_0_0_1px_var(--line)]"
      )}
    >
      <h3 className="line-clamp-2 text-[12.5px] leading-[1.4] font-medium tracking-section-head text-(--fg-strong)">
        {task.title}
      </h3>

      <div className="flex min-w-0 items-center gap-[7px]">
        <span className="font-mono text-[10px] text-(--faint)" data-slot="k-card-id">
          {identifier}
        </span>
      </div>

      {failedError ? (
        <div
          className="flex items-start gap-[5px] rounded-xs bg-(--danger-tint) px-[7px] py-[5px] font-mono text-[10.5px] text-(--danger)"
          data-testid={`tasks-kanban-card-error-${task.id}`}
        >
          <AlertCircle aria-hidden="true" className="mt-px size-3 shrink-0" />
          <span className="min-w-0 wrap-break-word">{failedError}</span>
        </div>
      ) : null}

      <div className="flex min-w-0 items-center justify-between gap-2">
        <div
          className="flex min-w-0 items-center gap-[6px]"
          data-testid={`tasks-kanban-card-owner-${task.id}`}
        >
          <OwnerAvatar
            data-testid={`tasks-kanban-card-avatar-${task.id}`}
            name={ownerLabel}
            ownerId={ownerId}
            ownerKind={ownerKind}
            size="sm"
          />
          <span className="min-w-0 truncate text-[11px] text-(--muted)">{ownerLabel}</span>
        </div>
        <span
          className="shrink-0 font-mono text-[10px] tabular-nums text-(--faint)"
          data-testid={`tasks-kanban-card-time-${task.id}`}
        >
          {timestamp}
        </span>
      </div>

      {canRetry ? (
        <Button
          aria-label={`Retry ${task.title}`}
          className="self-start"
          data-testid={`tasks-kanban-card-retry-${task.id}`}
          onClick={event => {
            event.stopPropagation();
            onRetry?.(task.id);
          }}
          size="xs"
          type="button"
          variant="outline"
        >
          Retry
        </Button>
      ) : null}
    </div>
  );
}
