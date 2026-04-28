import { Link } from "@tanstack/react-router";
import { AlertCircle, ArrowUpRight, ChevronRight, Loader2, Users } from "lucide-react";
import { useMemo } from "react";

import { Empty, Pill } from "@agh/ui";
import { cn } from "@/lib/utils";

import type { MultiAgentAgent, MultiAgentLiveState } from "@/hooks/routes/use-task-detail-page";

import {
  formatRelativeTime,
  taskOwnerLabel,
  taskStatusLabel,
  taskStatusTone,
} from "../lib/task-formatters";
import type { TaskTimelineItem } from "../types";

import { pillVariantFromTone } from "@/lib/pill-variant";

/**
 * Window (in ms) during which an agent is considered "freshly active" — only
 * then does its StatusDot pulse. Keeps decorative motion off idle rows.
 */
const LIVE_FRESHNESS_MS = 2_000;

/** Per-card event strip depth — collapsed view links to the Events tab for the rest. */
const AGENT_TIMELINE_DEPTH = 5;

export interface TasksMultiAgentPanelProps {
  agents: MultiAgentAgent[];
  state: MultiAgentLiveState;
  liveCount: number;
  descendantCount: number;
  activeDescendants: number;
  timeline: TaskTimelineItem[];
  errorMessage?: string | null;
}

/**
 * Agents tab — one card per agent in the tree, keyed by the live/idle state of
 * its active run. Each card shows a compact 5-event strip sourced from the
 * shared timeline. The interleaved cross-agent view used to live here; that
 * responsibility has moved to the Events tab (`TasksTimelinePanel`).
 */
export function TasksMultiAgentPanel({
  agents,
  state,
  liveCount,
  descendantCount,
  activeDescendants,
  timeline,
  errorMessage = null,
}: TasksMultiAgentPanelProps) {
  if (state === "loading") {
    return (
      <div
        className="flex min-h-[240px] flex-1 items-center justify-center"
        data-testid="tasks-multi-agent-loading"
      >
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (state === "disconnected") {
    return (
      <Empty
        icon={AlertCircle}
        title="Live tree unavailable"
        description={
          errorMessage ??
          "Live tree unavailable right now. Updates will resume once the connection is restored."
        }
        data-testid="tasks-multi-agent-disconnected"
      />
    );
  }

  const idleCount = Math.max(0, agents.length - liveCount);
  const totalAgents = agents.length;
  const subtitle =
    totalAgents === 0
      ? descendantCount === 0
        ? "No child runs yet."
        : `${descendantCount} ${descendantCount === 1 ? "descendant" : "descendants"} · ${activeDescendants} active`
      : `${liveCount} running · ${idleCount} idle`;

  return (
    <section
      aria-label="Agents"
      className="flex min-h-0 w-full flex-1 flex-col gap-6 px-6 py-5"
      data-testid="tasks-multi-agent-panel"
    >
      <header className="flex flex-col gap-1" data-testid="tasks-multi-agent-header">
        <h2 className="text-base font-semibold text-[color:var(--color-text-primary)]">Agents</h2>
        <p
          className="text-[13px] text-[color:var(--color-text-secondary)]"
          data-testid="tasks-multi-agent-summary"
        >
          {subtitle}
        </p>
      </header>

      {state === "no-descendants" ? (
        <Empty
          icon={Users}
          title="No descendants yet"
          description="Multi-agent live surfaces will appear once child runs spawn."
          data-testid="tasks-multi-agent-empty"
        />
      ) : (
        <ul className="flex flex-col gap-3" data-testid="tasks-multi-agent-agents">
          {agents.map(agent => (
            <li key={agent.node.task.id}>
              <TasksMultiAgentAgentCard agent={agent} timeline={timeline} />
            </li>
          ))}
        </ul>
      )}

      {state === "no-active" ? (
        <p
          className="rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-3 text-sm text-[color:var(--color-text-secondary)]"
          data-testid="tasks-multi-agent-no-active"
        >
          No runs are currently active. Descendant status will refresh as soon as a run resumes.
        </p>
      ) : null}
    </section>
  );
}

interface TasksMultiAgentAgentCardProps {
  agent: MultiAgentAgent;
  timeline: TaskTimelineItem[];
}

function TasksMultiAgentAgentCard({ agent, timeline }: TasksMultiAgentAgentCardProps) {
  const node = agent.node;
  const task = node.task;
  const run = node.active_run;
  const depthIndent = Math.min(node.depth ?? 0, 3);
  const ownerLabel = taskOwnerLabel(task.owner);

  const agentEvents = useMemo(
    () => timeline.filter(item => item.task?.id === task.id),
    [timeline, task.id]
  );
  const agentEventsTop = agentEvents.slice(0, AGENT_TIMELINE_DEPTH);
  const overflow = Math.max(0, agentEvents.length - agentEventsTop.length);
  const latestEventAt = agentEvents[0]?.timestamp ?? node.last_activity_at ?? null;
  const isFresh = isLatestEventFresh(latestEventAt);
  const pulse = agent.isLive && isFresh && run?.status === "running";

  const statusTone = taskStatusTone(task.status);

  return (
    <article
      className={cn(
        "relative flex flex-col gap-3 rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] py-3 pr-4 transition-colors",
        agent.isLive
          ? "border-l-2 border-l-[color:var(--color-accent)] pl-4"
          : "border-l-2 border-l-transparent pl-4"
      )}
      data-depth={depthIndent}
      data-is-live={agent.isLive ? "true" : "false"}
      data-is-root={agent.isRoot ? "true" : "false"}
      data-testid={`tasks-multi-agent-agent-${task.id}`}
      style={depthIndent > 0 ? { marginLeft: `${depthIndent * 16}px` } : undefined}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 flex-1 items-start gap-3">
          <AgentAvatar label={agent.label} isLive={agent.isLive} />
          <div className="min-w-0 flex-1">
            <div
              className="flex flex-wrap items-center gap-2"
              data-testid={`tasks-multi-agent-agent-meta-${task.id}`}
            >
              <Pill.Dot tone={agent.isLive ? "accent" : "neutral"} pulse={pulse} />
              <span
                className={cn(
                  "truncate text-[13px]",
                  agent.isLive
                    ? "font-medium text-[color:var(--color-text-primary)]"
                    : "text-[color:var(--color-text-secondary)]"
                )}
                data-testid={`tasks-multi-agent-agent-label-${task.id}`}
              >
                {agent.label}
              </span>
              <Pill mono data-testid={`tasks-multi-agent-agent-id-${task.id}`}>
                {task.identifier ?? task.id}
              </Pill>
              <Pill
                data-testid={`tasks-multi-agent-agent-status-${task.id}`}
                tone={pillVariantFromTone(statusTone)}
              >
                {taskStatusLabel(task.status)}
              </Pill>
            </div>
            <p
              className="mt-1 truncate text-[13px] text-[color:var(--color-text-primary)]"
              data-testid={`tasks-multi-agent-agent-title-${task.id}`}
            >
              {task.title}
            </p>
            <div className="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-[11px] text-[color:var(--color-text-tertiary)]">
              <span>Owner {ownerLabel}</span>
              {node.last_activity_at ? (
                <>
                  <span aria-hidden>·</span>
                  <span>Updated {formatRelativeTime(node.last_activity_at)}</span>
                </>
              ) : null}
              {node.child_count ? (
                <>
                  <span aria-hidden>·</span>
                  <span>
                    {node.child_count} {node.child_count === 1 ? "child" : "children"}
                  </span>
                </>
              ) : null}
            </div>
            {run?.error ? (
              <p
                className="mt-2 flex items-start gap-1 text-[11px] text-[color:var(--color-danger)]"
                data-testid={`tasks-multi-agent-agent-error-${task.id}`}
              >
                <AlertCircle className="mt-0.5 size-3 shrink-0" />
                <span className="truncate">{run.error}</span>
              </p>
            ) : null}
          </div>
        </div>
      </div>

      {agentEventsTop.length > 0 ? (
        <ul
          className="flex flex-col gap-1 border-t border-[color:var(--color-divider)] pt-3 font-mono text-[11px] text-[color:var(--color-text-secondary)]"
          data-testid={`tasks-multi-agent-agent-events-${task.id}`}
        >
          {agentEventsTop.map(event => (
            <li
              className="flex items-baseline gap-2"
              data-testid={`tasks-multi-agent-agent-event-${event.event_id}`}
              key={event.event_id}
            >
              <span className="shrink-0 text-[color:var(--color-text-tertiary)]">
                {formatEventTime(event.timestamp)}
              </span>
              <span className="truncate text-[color:var(--color-text-secondary)]">
                {event.event_type}
              </span>
            </li>
          ))}
          {overflow > 0 ? (
            <li className="pt-0.5">
              <Link
                className="inline-flex items-center gap-1 text-[11px] uppercase tracking-[0.12em] text-[color:var(--color-accent)] hover:underline"
                data-testid={`tasks-multi-agent-agent-events-more-${task.id}`}
                params={{ id: task.id }}
                to="/tasks/$id"
              >
                +{overflow} more
              </Link>
            </li>
          ) : null}
        </ul>
      ) : null}

      <div
        className="flex flex-wrap items-center justify-end gap-3 border-t border-[color:var(--color-divider)] pt-3"
        data-testid={`tasks-multi-agent-agent-actions-${task.id}`}
      >
        {run?.session_id ? (
          <Link
            className="inline-flex items-center gap-1 font-mono text-[11px] uppercase tracking-[0.12em] text-[color:var(--color-text-secondary)] hover:text-[color:var(--color-accent)]"
            data-testid={`tasks-multi-agent-agent-session-${task.id}`}
            params={{ id: run.session_id }}
            to="/session/$id"
          >
            Open session <ArrowUpRight className="size-3" />
          </Link>
        ) : null}
        {run?.id ? (
          <Link
            className="inline-flex items-center gap-1 font-mono text-[11px] uppercase tracking-[0.12em] text-[color:var(--color-accent)] hover:underline"
            data-testid={`tasks-multi-agent-agent-run-${task.id}`}
            params={{ id: task.id, runId: run.id }}
            to="/tasks/$id/runs/$runId"
          >
            Open run <ArrowUpRight className="size-3" />
          </Link>
        ) : null}
        {!agent.isRoot ? (
          <Link
            className="inline-flex items-center gap-1 font-mono text-[11px] uppercase tracking-[0.12em] text-[color:var(--color-text-secondary)] hover:text-[color:var(--color-accent)]"
            data-testid={`tasks-multi-agent-agent-task-${task.id}`}
            params={{ id: task.id }}
            to="/tasks/$id"
          >
            Open task <ChevronRight className="size-3" />
          </Link>
        ) : null}
      </div>
    </article>
  );
}

interface AgentAvatarProps {
  label: string;
  isLive: boolean;
}

function AgentAvatar({ label, isLive }: AgentAvatarProps) {
  const initial = label.charAt(0).toUpperCase() || "?";
  return (
    <span
      aria-hidden="true"
      className={cn(
        "flex size-9 shrink-0 items-center justify-center rounded-lg border text-xs font-semibold",
        isLive
          ? "border-[color:var(--color-accent)] bg-[color:var(--color-accent-tint)] text-[color:var(--color-accent)]"
          : "border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] text-[color:var(--color-text-secondary)]"
      )}
    >
      {initial}
    </span>
  );
}

function isLatestEventFresh(timestamp?: string | null, now: Date = new Date()): boolean {
  if (!timestamp) {
    return false;
  }
  const ts = Date.parse(timestamp);
  if (Number.isNaN(ts)) {
    return false;
  }
  return now.getTime() - ts <= LIVE_FRESHNESS_MS;
}

function formatEventTime(value?: string | null): string {
  if (!value) return "";
  const ts = Date.parse(value);
  if (Number.isNaN(ts)) return "";
  const date = new Date(ts);
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  const seconds = String(date.getSeconds()).padStart(2, "0");
  return `${hours}:${minutes}:${seconds}`;
}
