import { Link } from "@tanstack/react-router";
import { AlertCircle, ArrowUpRight } from "lucide-react";
import { useMemo, type ReactNode } from "react";

import { Eyebrow, MonoId, OwnerAvatar, Pill, Time } from "@agh/ui";

import { ownerAvatarKindFor, taskOwnerLabel } from "../lib/task-formatters";
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
 * Per-agent card — `<OwnerAvatar lg>` + name + optional live `<Pill.Dot>`,
 * 3-metric strip (Live · Events · Descendants), recent event rail (≤ 5 rows
 * with mono identifiers + relative timestamps), and a single ghost link footer
 * to the agent's full task surface. Flat card (`bg-canvas-soft`, no border,
 * no side-stripe rail).
 */
export function AgentCard({ node, label, isLive, isRoot, timeline }: AgentCardProps) {
  const task = node.task;
  const run = node.active_run;
  const ownerKind = ownerAvatarKindFor(task.owner?.kind);
  const ownerId = task.owner?.ref ?? task.owner?.kind ?? task.id;
  const ownerName = label || taskOwnerLabel(task.owner);
  const childCount = node.child_count ?? 0;
  const taskIdentifier = task.identifier ?? task.id;
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
      className="flex w-full min-w-0 flex-col gap-4 rounded-lg bg-canvas-soft px-5 py-4"
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
              className="truncate text-small-body font-medium text-fg-strong"
              data-testid={`tasks-multi-agent-agent-label-${task.id}`}
            >
              {ownerName}
            </h3>
            {isLive ? (
              <span
                className="inline-flex items-center gap-1.5"
                data-testid={`tasks-multi-agent-agent-live-${task.id}`}
              >
                <Pill.Dot pulse tone="info" />
                <Eyebrow className="text-info">Live</Eyebrow>
              </span>
            ) : null}
          </div>
          <div
            className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 text-form-label text-muted"
            data-testid={`tasks-multi-agent-agent-meta-${task.id}`}
          >
            <MonoId data-testid={`tasks-multi-agent-agent-id-${task.id}`} value={taskIdentifier} />
            <span aria-hidden className="text-faint">
              ·
            </span>
            <span className="truncate">{task.title}</span>
          </div>
        </div>
      </header>

      <dl
        className="grid grid-cols-3 gap-x-4"
        data-testid={`tasks-multi-agent-agent-metrics-${task.id}`}
      >
        <AgentMetric label="Live" value={isLive ? "Yes" : "No"} />
        <AgentMetric label="Events" value={events.length} />
        <AgentMetric label="Descendants" value={childCount} />
      </dl>

      {failureMessage ? (
        <p
          className="flex items-start gap-1.5 rounded bg-danger-tint px-3 py-2 text-form-label text-danger"
          data-testid={`tasks-multi-agent-agent-error-${task.id}`}
        >
          <AlertCircle className="mt-0.5 size-3 shrink-0" strokeWidth={1.75} />
          <span className="truncate">{failureMessage}</span>
        </p>
      ) : null}

      {eventsTop.length > 0 ? (
        <ul
          className="flex flex-col gap-1.5 border-t border-line pt-3"
          data-testid={`tasks-multi-agent-agent-events-${task.id}`}
        >
          {eventsTop.map(event => (
            <li
              className="grid grid-cols-[1fr_auto] items-baseline gap-3 text-form-hint"
              data-testid={`tasks-multi-agent-agent-event-${event.event_id}`}
              key={event.event_id}
            >
              <span className="truncate font-mono tabular-nums text-muted">{event.event_type}</span>
              {event.timestamp ? (
                <Time
                  className="shrink-0 text-right text-mono-id text-subtle"
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
        className="flex items-center justify-end border-t border-line pt-3"
        data-testid={`tasks-multi-agent-agent-footer-${task.id}`}
      >
        <Link
          aria-label={isRoot ? "View full timeline" : `Open agent ${taskIdentifier}`}
          className="inline-flex items-center gap-1.5 text-form-hint text-muted transition-colors duration-base ease-out hover:text-fg-strong"
          data-testid={`tasks-multi-agent-agent-link-${task.id}`}
          params={{ id: task.id }}
          to="/tasks/$id"
        >
          <span>{isRoot ? "View full timeline" : "Open agent"}</span>
          <ArrowUpRight className="size-3" strokeWidth={1.75} />
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
      <Eyebrow className="text-subtle">{label}</Eyebrow>
      <span className="text-item-title font-medium tabular-nums text-fg-strong">{value}</span>
    </div>
  );
}
