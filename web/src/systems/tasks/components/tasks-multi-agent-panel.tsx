import { AlertCircle, Users } from "lucide-react";

import { BlockLoading, Empty, Eyebrow } from "@agh/ui";

import type { MultiAgentAgent, MultiAgentLiveState } from "@/hooks/routes/use-task-detail-page";

import type { TaskTimelineItem } from "../types";
import { AgentCard } from "./agent-card";

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
 * Agents tab — renders one `<AgentCard>` per agent in the tree, keyed by the
 * live/idle state of its active run. The interleaved cross-agent view used to
 * live here; that responsibility moved to the Events tab
 * (`TasksTimelinePanel`).
 *
 * Header carries an eyebrow label + running/idle count micro-line; the body is
 * a single-column stack of flat agent cards. No side-stripe accent rail, no
 * stacked accent CTAs — those defects are gone from this surface.
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
      <BlockLoading
        label="Loading agent tree"
        size="md"
        surface="bare"
        data-testid="tasks-multi-agent-loading"
      />
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
      className="flex min-h-0 w-full flex-1 flex-col gap-5 px-6 py-5"
      data-testid="tasks-multi-agent-panel"
    >
      <header className="flex flex-col gap-1.5" data-testid="tasks-multi-agent-header">
        <Eyebrow className="text-muted">Agents</Eyebrow>
        <p
          className="text-small-body text-fg-strong tabular-nums"
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
        <ul
          className="flex min-w-0 flex-col gap-3"
          data-testid="tasks-multi-agent-agents"
          role="list"
        >
          {agents.map(agent => (
            <li className="min-w-0" key={agent.node.task.id}>
              <AgentCard
                isLive={agent.isLive}
                isRoot={agent.isRoot}
                label={agent.label}
                node={agent.node}
                timeline={timeline}
              />
            </li>
          ))}
        </ul>
      )}

      {state === "no-active" ? (
        <p
          className="rounded-lg bg-canvas-soft px-4 py-3 text-form-input text-muted"
          data-testid="tasks-multi-agent-no-active"
        >
          No runs are currently active. Descendant status will refresh as soon as a run resumes.
        </p>
      ) : null}
    </section>
  );
}
