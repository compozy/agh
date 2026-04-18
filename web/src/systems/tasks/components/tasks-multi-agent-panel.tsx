import { Link } from "@tanstack/react-router";
import { AlertCircle, ArrowUpRight, ChevronRight, Loader2 } from "lucide-react";

import { Pill } from "@agh/ui";
import { cn } from "@/lib/utils";

import type { MultiAgentAgent, MultiAgentLiveState } from "@/hooks/routes/use-task-detail-page";

import {
  formatAttemptLabel,
  formatRelativeTime,
  taskOwnerLabel,
  taskRunStatusTone,
  taskStatusLabel,
  taskStatusTone,
} from "../lib/task-formatters";
import type { TaskTimelineItem } from "../types";
import { TasksTimelinePanel } from "./tasks-timeline-panel";

import { pillVariantFromTone } from "@/lib/pill-variant";
export interface TasksMultiAgentPanelProps {
  agents: MultiAgentAgent[];
  state: MultiAgentLiveState;
  liveCount: number;
  descendantCount: number;
  activeDescendants: number;
  timeline: TaskTimelineItem[];
  timelineLive?: boolean;
  timelineLoading?: boolean;
  timelineErrorMessage?: string | null;
  canLoadMoreTimeline?: boolean;
  onLoadMoreTimeline?: () => void;
  errorMessage?: string | null;
}

export function TasksMultiAgentPanel({
  agents,
  state,
  liveCount,
  descendantCount,
  activeDescendants,
  timeline,
  timelineLive = false,
  timelineLoading = false,
  timelineErrorMessage = null,
  canLoadMoreTimeline = false,
  onLoadMoreTimeline,
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
      <div
        className="flex min-h-[240px] flex-1 flex-col items-center justify-center gap-2 px-6 text-center"
        data-testid="tasks-multi-agent-disconnected"
      >
        <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
        <p className="text-sm text-[color:var(--color-text-secondary)]">
          {errorMessage ??
            "Live tree unavailable right now. Updates will resume once the connection is restored."}
        </p>
      </div>
    );
  }

  return (
    <section
      aria-label="Multi-agent live view"
      className="flex min-h-0 flex-1 flex-col gap-5 px-6 py-5"
      data-testid="tasks-multi-agent-panel"
    >
      <header
        className="flex flex-wrap items-center justify-between gap-3"
        data-testid="tasks-multi-agent-header"
      >
        <div className="flex flex-col gap-1">
          <p className="font-mono text-[0.66rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
            Multi-Agent Live
          </p>
          <p className="text-sm text-[color:var(--color-text-secondary)]">
            {descendantCount === 0
              ? "No child runs yet."
              : `${descendantCount} ${descendantCount === 1 ? "descendant" : "descendants"} · ${activeDescendants} active`}
          </p>
        </div>
        <AgentsLivePill count={liveCount} />
      </header>

      {state === "no-descendants" ? (
        <div
          className="flex min-h-[200px] items-center justify-center rounded-2xl border border-dashed border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-6 py-5 text-center text-sm text-[color:var(--color-text-secondary)]"
          data-testid="tasks-multi-agent-empty"
        >
          This task has no descendants. Multi-agent live surfaces will appear once child runs spawn.
        </div>
      ) : (
        <ul className="flex flex-col gap-3" data-testid="tasks-multi-agent-agents">
          {agents.map(agent => (
            <li key={agent.node.task.id}>
              <TasksMultiAgentAgentCard agent={agent} />
            </li>
          ))}
        </ul>
      )}

      {state === "no-active" ? (
        <div
          className="rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-3 text-sm text-[color:var(--color-text-secondary)]"
          data-testid="tasks-multi-agent-no-active"
        >
          No runs are currently active. Descendant status will refresh as soon as a run resumes.
        </div>
      ) : null}

      <section
        aria-label="Interleaved agent timeline"
        className="flex min-h-0 flex-col gap-3 rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)]"
        data-testid="tasks-multi-agent-timeline"
      >
        <div className="flex items-center justify-between gap-2 border-b border-[color:var(--color-divider)] px-5 py-3">
          <p className="font-mono text-[0.62rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
            Interleaved Timeline · all runs, dedup by (run_id, seq)
          </p>
          {timelineLive ? (
            <span
              className="inline-flex items-center gap-1 font-mono text-[0.6rem] uppercase tracking-[0.14em] text-[color:var(--color-accent)]"
              data-testid="tasks-multi-agent-timeline-live"
            >
              <span className="size-1.5 animate-pulse rounded-full bg-[color:var(--color-accent)]" />
              Live
            </span>
          ) : null}
        </div>
        <TasksTimelinePanel
          canLoadMore={canLoadMoreTimeline}
          errorMessage={timelineErrorMessage}
          isLive={timelineLive}
          isLoading={timelineLoading}
          items={timeline}
          onLoadMore={onLoadMoreTimeline}
        />
      </section>
    </section>
  );
}

interface TasksMultiAgentAgentCardProps {
  agent: MultiAgentAgent;
}

function TasksMultiAgentAgentCard({ agent }: TasksMultiAgentAgentCardProps) {
  const node = agent.node;
  const task = node.task;
  const run = node.active_run;
  const depthIndent = Math.min(node.depth ?? 0, 3);
  const attempts = run ? formatAttemptLabel(run.attempt, run.max_attempts) : null;
  const ownerLabel = taskOwnerLabel(task.owner);
  const lineage = agent.isRoot ? "primary task" : `child of ${node.parent_task_id ?? "—"}`;

  return (
    <article
      className={cn(
        "relative flex flex-col gap-3 rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-4 py-3 transition-colors",
        agent.isLive && "border-[color:var(--color-accent)]"
      )}
      data-depth={depthIndent}
      data-is-root={agent.isRoot ? "true" : "false"}
      data-testid={`tasks-multi-agent-agent-${task.id}`}
      style={depthIndent > 0 ? { marginLeft: `${depthIndent * 16}px` } : undefined}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 flex-1 items-start gap-3">
          <AgentAvatar label={agent.label} isLive={agent.isLive} />
          <div className="min-w-0 flex-1">
            <div
              className="flex flex-wrap items-center gap-2 text-xs text-[color:var(--color-text-secondary)]"
              data-testid={`tasks-multi-agent-agent-meta-${task.id}`}
            >
              <span className="text-sm font-semibold text-[color:var(--color-text-primary)]">
                {agent.label}
              </span>
              {run?.id ? (
                <span className="font-mono text-[color:var(--color-text-primary)]">{run.id}</span>
              ) : null}
              <span>· {lineage}</span>
              {attempts ? <span>· {attempts}</span> : null}
            </div>
            <p
              className="mt-1 truncate text-sm text-[color:var(--color-text-primary)]"
              data-testid={`tasks-multi-agent-agent-title-${task.id}`}
            >
              {task.title}
            </p>
            <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-[color:var(--color-text-secondary)]">
              <Pill variant={pillVariantFromTone(taskStatusTone(task.status))}>
                {taskStatusLabel(task.status)}
              </Pill>
              {run ? (
                <Pill variant={pillVariantFromTone(taskRunStatusTone(run.status))}>
                  {run.status}
                </Pill>
              ) : null}
              <span>· Owner {ownerLabel}</span>
              {node.last_activity_at ? (
                <span>· Updated {formatRelativeTime(node.last_activity_at)}</span>
              ) : null}
              {node.child_count ? (
                <span>
                  · {node.child_count} {node.child_count === 1 ? "child" : "children"}
                </span>
              ) : null}
            </div>
            {run?.error ? (
              <p
                className="mt-2 flex items-start gap-1 text-xs text-[color:var(--color-danger)]"
                data-testid={`tasks-multi-agent-agent-error-${task.id}`}
              >
                <AlertCircle className="mt-0.5 size-3 shrink-0" />
                <span className="truncate">{run.error}</span>
              </p>
            ) : null}
          </div>
        </div>
        <div className="flex shrink-0 flex-col items-end gap-2">
          {agent.isRoot ? (
            <span
              className="inline-flex items-center gap-1 rounded-md bg-[color:var(--color-accent-tint)] px-2 py-0.5 font-mono text-[0.6rem] uppercase tracking-[0.14em] text-[color:var(--color-accent)]"
              data-testid={`tasks-multi-agent-agent-primary-${task.id}`}
            >
              Primary · Pinned
            </span>
          ) : null}
          {agent.isLive ? (
            <span
              className="inline-flex items-center gap-1 font-mono text-[0.6rem] uppercase tracking-[0.14em] text-[color:var(--color-accent)]"
              data-testid={`tasks-multi-agent-agent-live-${task.id}`}
            >
              <span className="size-1.5 animate-pulse rounded-full bg-[color:var(--color-accent)]" />
              Live
            </span>
          ) : null}
        </div>
      </div>

      <div
        className="flex flex-wrap items-center justify-end gap-3 border-t border-[color:var(--color-divider)] pt-3"
        data-testid={`tasks-multi-agent-agent-actions-${task.id}`}
      >
        {run?.session_id ? (
          <Link
            className="inline-flex items-center gap-1 font-mono text-[0.64rem] uppercase tracking-[0.14em] text-[color:var(--color-accent)] hover:underline"
            data-testid={`tasks-multi-agent-agent-session-${task.id}`}
            params={{ id: run.session_id }}
            to="/session/$id"
          >
            Open session <ArrowUpRight className="size-3" />
          </Link>
        ) : null}
        {run?.id ? (
          <Link
            className="inline-flex items-center gap-1 font-mono text-[0.64rem] uppercase tracking-[0.14em] text-[color:var(--color-accent)] hover:underline"
            data-testid={`tasks-multi-agent-agent-run-${task.id}`}
            params={{ id: task.id, runId: run.id }}
            to="/tasks/$id/runs/$runId"
          >
            Open run <ArrowUpRight className="size-3" />
          </Link>
        ) : null}
        {!agent.isRoot ? (
          <Link
            className="inline-flex items-center gap-1 font-mono text-[0.64rem] uppercase tracking-[0.14em] text-[color:var(--color-accent)] hover:underline"
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

interface AgentsLivePillProps {
  count: number;
}

function AgentsLivePill({ count }: AgentsLivePillProps) {
  const isLive = count > 0;
  return (
    <span
      className={cn(
        "inline-flex items-center gap-2 rounded-md border px-3 py-1 font-mono text-[0.62rem] uppercase tracking-[0.14em]",
        isLive
          ? "border-[color:var(--color-accent)] bg-[color:var(--color-accent-tint)] text-[color:var(--color-accent)]"
          : "border-[color:var(--color-divider)] bg-[color:var(--color-surface)] text-[color:var(--color-text-secondary)]"
      )}
      data-testid="tasks-multi-agent-live-count"
    >
      {isLive ? (
        <span className="size-1.5 animate-pulse rounded-full bg-[color:var(--color-accent)]" />
      ) : (
        <span className="size-1.5 rounded-full bg-[color:var(--color-text-tertiary)]" />
      )}
      {count} {count === 1 ? "agent" : "agents"} live
    </span>
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
          : "border-[color:var(--color-divider)] bg-[color:var(--color-surface)] text-[color:var(--color-text-secondary)]"
      )}
    >
      {initial}
    </span>
  );
}
