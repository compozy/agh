import { cn, Time, type PillTone } from "@agh/ui";

import { humanizeTaskEvent } from "../lib/task-activity-copy";
import { resolveEventTone } from "../lib/timeline-visuals";
import type { TaskTimelineItem } from "../types";

const TONE_DOT: Record<PillTone, string> = {
  neutral: "bg-faint",
  accent: "bg-accent",
  success: "bg-success",
  warning: "bg-warning",
  danger: "bg-danger",
  info: "bg-info",
};

/**
 * One humanized activity row (§4.6): plain-language title first, optional
 * detail, freshness right-aligned, and the raw `event_type · seq` kept as
 * quiet mono microtext for operators.
 */
export function TaskActivityItem({ item, isLive }: { item: TaskTimelineItem; isLive: boolean }) {
  const view = humanizeTaskEvent(item);
  const tone = resolveEventTone(item.event_type, isLive);

  return (
    <div
      className="grid grid-cols-[14px_minmax(0,1fr)_auto] gap-3 border-t border-line-soft px-4 py-2.5 first:border-t-0"
      data-category={view.category}
      data-testid={`tasks-activity-item-${item.event_id}`}
    >
      <span
        aria-hidden="true"
        className={cn("mt-[5px] size-[7px] justify-self-center rounded-full", TONE_DOT[tone])}
      />
      <div className="min-w-0">
        <div className="text-ws-name font-medium text-fg-strong">{view.title}</div>
        {view.detail ? (
          <p className="mt-0.5 max-w-[62ch] text-small-body leading-relaxed text-muted">
            {view.detail}
          </p>
        ) : null}
      </div>
      <div className="flex shrink-0 flex-col items-end gap-0.5">
        {item.timestamp ? (
          <Time
            className="text-eyebrow tabular-nums text-subtle"
            iso={item.timestamp}
            mode="relative"
          />
        ) : null}
        <span className="font-mono text-micro text-faint">
          {item.event_type} · {item.sequence}
        </span>
      </div>
    </div>
  );
}
