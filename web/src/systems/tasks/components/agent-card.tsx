import { Link } from "@tanstack/react-router";
import { AlertCircle, ArrowUpRight } from "lucide-react";
import { useMemo, type ReactNode } from "react";

import { Eyebrow, OwnerAvatar, Pill, Time } from "@agh/ui";

import { ownerAvatarKindFor, taskOwnerLabel, taskShortId } from "../lib/task-formatters";
import type { TaskTimelineItem, TaskTreeNode } from "../types";

const AGENT_TIMELINE_DEPTH = 5;

export interface AgentCardProps {
  node: TaskTreeNode;
  /** Display label (typically owner ref or task identifier). */
  label: string;
  /** True when the agent's active run is currently executing. */
  isLive: boolean;
  /** True when this is the root node of the agents tree. */
  isRoot: boolean;
  /** Per-agent slice of the task timeline used to render the events strip. */
  timeline: TaskTimelineItem[];
}

/**
 * Per-agent card — `<OwnerAvatar lg>` + name 13/510 + optional
 * live pulse pill, 3-metric strip (Live · Events · Descendants), events strip
 * (≤ 5 rows in a 14/1fr/56 grid), and a full-width ghost footer linking to the
 * full timeline. Local to the tasks system (not promoted to
 * `@agh/ui`).
 */
export function AgentCard({ node, label, isLive, isRoot, timeline }: AgentCardProps) {
  const task = node.task;
  const run = node.active_run;
  const ownerKind = ownerAvatarKindFor(task.owner?.kind);
  const ownerId = task.owner?.ref ?? task.owner?.kind ?? task.id;
  const ownerName = label || taskOwnerLabel(task.owner);
  const childCount = node.child_count ?? 0;
  const events = useMemo(
    () => timeline.filter(item => item.task?.id === task.id),
    [timeline, task.id]
  );
  const eventsTop = events.slice(0, AGENT_TIMELINE_DEPTH);
  const failureMessage = run?.error ?? null;

  return (
    <article
      data-testid={`tasks-multi-agent-agent-${task.id}`}
      data-is-live={isLive ? "true" : "false"}
      data-is-root={isRoot ? "true" : "false"}
      className="flex w-full min-w-0 flex-col gap-3 rounded-lg bg-(--canvas-soft) px-[18px] py-4"
    >
      <header className="flex items-start gap-3">
        <OwnerAvatar
          name={ownerName}
          ownerId={ownerId}
          ownerKind={ownerKind}
          size="lg"
          data-testid={`tasks-multi-agent-agent-avatar-${task.id}`}
        />
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <div className="flex flex-wrap items-center gap-2">
            <h3
              className="truncate text-[13px] font-[510] tracking-eyebrow text-(--fg-strong)"
              data-testid={`tasks-multi-agent-agent-label-${task.id}`}
            >
              {ownerName}
            </h3>
            {isLive ? (
              <Pill data-testid={`tasks-multi-agent-agent-live-${task.id}`} tone="accent">
                <span
                  aria-hidden="true"
                  className="inline-block size-1.5 animate-pulse rounded-full bg-(--accent)"
                />
                Live
              </Pill>
            ) : null}
          </div>
          <div
            className="flex flex-wrap items-center gap-x-1.5 gap-y-1 text-eyebrow text-(--subtle)"
            data-testid={`tasks-multi-agent-agent-meta-${task.id}`}
          >
            <span data-testid={`tasks-multi-agent-agent-id-${task.id}`}>
              {taskShortId({ id: task.id, identifier: task.identifier })}
            </span>
            <span aria-hidden>·</span>
            <span className="truncate">{task.title}</span>
          </div>
        </div>
      </header>

      <dl
        className="grid grid-cols-3 gap-3"
        data-testid={`tasks-multi-agent-agent-metrics-${task.id}`}
      >
        <AgentMetric label="Live" value={isLive ? "Yes" : "No"} />
        <AgentMetric label="Events" value={events.length} />
        <AgentMetric label="Descendants" value={childCount} />
      </dl>

      {failureMessage ? (
        <p
          className="flex items-start gap-1 rounded-(--radius) bg-(--danger-tint) px-2.5 py-1.5 text-eyebrow text-(--danger)"
          data-testid={`tasks-multi-agent-agent-error-${task.id}`}
        >
          <AlertCircle className="mt-0.5 size-3 shrink-0" strokeWidth={1.75} />
          <span className="truncate">{failureMessage}</span>
        </p>
      ) : null}

      {eventsTop.length > 0 ? (
        <ul
          className="flex flex-col gap-1"
          data-testid={`tasks-multi-agent-agent-events-${task.id}`}
        >
          {eventsTop.map(event => (
            <li
              className="grid grid-cols-[14px_1fr_56px] items-baseline gap-2 text-eyebrow"
              data-testid={`tasks-multi-agent-agent-event-${event.event_id}`}
              key={event.event_id}
            >
              <span aria-hidden className="text-(--faint) tabular-nums">
                ·
              </span>
              <span className="truncate font-mono text-(--muted)">{event.event_type}</span>
              {event.timestamp ? (
                <Time
                  className="text-right text-(--subtle)"
                  iso={event.timestamp}
                  mode="relative"
                />
              ) : (
                <span aria-hidden />
              )}
            </li>
          ))}
        </ul>
      ) : null}

      <footer
        className="flex items-center justify-center border-t border-(--line) pt-3"
        data-testid={`tasks-multi-agent-agent-footer-${task.id}`}
      >
        <Link
          aria-label={isRoot ? "View full timeline" : `Open agent ${task.identifier ?? task.id}`}
          className="inline-flex w-full items-center justify-center gap-1.5 text-eyebrow text-(--muted) transition-colors duration-(--dur) ease-(--ease) hover:text-(--fg-strong)"
          data-testid={`tasks-multi-agent-agent-link-${task.id}`}
          params={{ id: task.id }}
          to="/tasks/$id"
        >
          <ArrowUpRight className="size-3" strokeWidth={1.75} />
          <span>View full timeline</span>
        </Link>
      </footer>
    </article>
  );
}

interface AgentMetricProps {
  label: string;
  value: ReactNode;
}

function AgentMetric({ label, value }: AgentMetricProps) {
  return (
    <div className="flex min-w-0 flex-col gap-1">
      <Eyebrow className="text-(--muted)">{label}</Eyebrow>
      <span className="text-[16px] font-[510] tabular-nums text-(--fg-strong)">{value}</span>
    </div>
  );
}
