import * as React from "react";

import { cn } from "@/lib/utils";

import type { InboxGroupId } from "../lib/inbox-grouping";

export interface TasksInboxRowProps extends Omit<React.ComponentProps<"div">, "onSelect"> {
  taskId: string;
  /**
   * Inbox group this row belongs to. Drives the rail tone —
   * `needs_review` paints warning, `blocked` paints danger, `stuck` paints
   * warning ring, `mentions` paints accent, `updates` paints faint.
   */
  group: InboxGroupId;
  unread?: boolean;
  onSelect?: () => void;
  /** Top row content -- title + identifier + status/lane badges. */
  top: React.ReactNode;
  /** Optional detail rows under the top (blocking reason, error, owner · time). */
  detail?: React.ReactNode;
  /** Right-side meta column (auto-width) — owner avatar, timestamp, etc. */
  meta?: React.ReactNode;
  /** Right-side actions (auto-width column). */
  actions?: React.ReactNode;
}

/**
 * Inbox row primitive — 3-column grid `[ rail | body | meta ]`.
 * The rail is painted by the row's group tone (not `border-l-2`); body holds
 * the top + detail content; the meta column carries optional right-aligned
 * actions or meta.
 *
 * Unread state is expressed through the body's title weight (handled by the
 * consumer); the rail tone is owned by the group, not the unread flag.
 */
const RAIL_CLASS: Record<InboxGroupId, string> = {
  needs_review: "bg-warning",
  blocked: "bg-danger",
  stuck: "bg-transparent shadow-[inset_0_0_0_1px_var(--warning)]",
  mentions: "bg-accent",
  updates: "bg-transparent shadow-[inset_0_0_0_1px_var(--faint)]",
};

function TasksInboxRow({
  taskId,
  group,
  unread = false,
  onSelect,
  top,
  detail,
  meta,
  actions,
  className,
  ...props
}: TasksInboxRowProps) {
  const clickable = onSelect !== undefined;
  const handleKeyDown = clickable
    ? (event: React.KeyboardEvent<HTMLDivElement>) => {
        if (event.target !== event.currentTarget) return;
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onSelect?.();
        }
      }
    : undefined;

  const trailing = meta !== undefined || actions !== undefined;

  return (
    <div
      aria-pressed={clickable ? unread : undefined}
      data-slot="tasks-inbox-row"
      data-group={group}
      data-testid={`tasks-inbox-item-${taskId}`}
      data-unread={unread ? "true" : "false"}
      onClick={clickable ? () => onSelect?.() : undefined}
      onKeyDown={handleKeyDown}
      role={clickable ? "button" : undefined}
      tabIndex={clickable ? 0 : undefined}
      className={cn(
        "grid min-h-[44px] items-stretch gap-[12px] border-b border-line-soft py-3 pr-[14px] text-left transition-colors duration-base ease-out",
        trailing ? "grid-cols-[3px_minmax(0,1fr)_auto]" : "grid-cols-[3px_minmax(0,1fr)]",
        clickable &&
          "cursor-pointer hover:bg-hover focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-line-strong focus-visible:ring-inset",
        className
      )}
      {...props}
    >
      <span
        aria-hidden="true"
        data-slot="tasks-inbox-row-rail"
        className={cn("self-stretch rounded-r-[1px]", RAIL_CLASS[group])}
      />

      <div className="flex min-w-0 flex-col gap-[5px] pl-[8px]" data-slot="tasks-inbox-row-main">
        <div
          className="flex min-w-0 flex-wrap items-center gap-[9px]"
          data-slot="tasks-inbox-row-top"
        >
          {top}
        </div>
        {detail !== undefined ? (
          <div
            className="flex min-w-0 flex-col gap-1 text-[11.5px] tracking-eyebrow text-muted"
            data-slot="tasks-inbox-row-detail"
          >
            {detail}
          </div>
        ) : null}
      </div>

      {trailing ? (
        <div
          className="flex shrink-0 items-center gap-2"
          data-slot="tasks-inbox-row-meta"
          data-testid={`tasks-inbox-item-actions-${taskId}`}
          onClick={stopPropagation}
          onKeyDown={stopPropagation}
          role="presentation"
        >
          {meta}
          {actions}
        </div>
      ) : null}
    </div>
  );
}

function stopPropagation(event: React.SyntheticEvent) {
  event.stopPropagation();
}

export { TasksInboxRow };
