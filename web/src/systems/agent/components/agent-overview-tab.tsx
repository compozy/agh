import { Link } from "@tanstack/react-router";

import { MetadataList, Section, Skeleton } from "@agh/ui";

import type { SessionPayload } from "@/systems/session";

import {
  formatAbsentListLabels,
  formatAbsentOverride,
  formatPromptWordCount,
  formatSkillsPolicyLine,
} from "../lib/agent-absent-value";
import { deriveAgentFleetSignals } from "../lib/fleet-signals";
import type { AgentPayload } from "../types";
import { AgentStatsGrid } from "./agent-stats-grid";

export interface AgentOverviewTabProps {
  agent: AgentPayload;
  sessions: SessionPayload[];
  sessionsTotal: number;
  activeSessionsTotal: number;
  resumableSessionsTotal: number;
  lastSessionActivityAt: string | null;
  sessionsLoading: boolean;
  sessionsError: boolean;
  onViewAllSessions: () => void;
}

export function AgentOverviewTab({
  agent,
  sessions,
  sessionsTotal,
  activeSessionsTotal,
  resumableSessionsTotal,
  lastSessionActivityAt,
  sessionsLoading,
  sessionsError,
  onViewAllSessions,
}: AgentOverviewTabProps) {
  const liveSessions = sessions.filter(session => session.state === "active").slice(0, 3);
  const mcpNames = (agent.mcp_servers ?? []).map(server => server.name);
  const toolsCount = agent.tools?.length ?? 0;
  const denyCount = agent.deny_tools?.length ?? 0;
  const toolsetsCount = agent.toolsets?.length ?? 0;

  return (
    <div className="flex flex-col gap-6" data-testid="agent-overview-tab">
      {sessionsError ? (
        <p
          className="text-small-body text-muted"
          role="status"
          data-testid="agent-overview-sessions-notice"
        >
          Session status unavailable
        </p>
      ) : null}
      {sessionsLoading ? (
        <div
          className="grid grid-cols-2 gap-3 md:grid-cols-4"
          data-testid="agent-overview-metrics-skeleton"
        >
          {Array.from({ length: 4 }).map((_, index) => (
            <Skeleton key={index} className="h-20 rounded-md" />
          ))}
        </div>
      ) : (
        <AgentStatsGrid
          total={sessionsTotal}
          active={activeSessionsTotal}
          resumable={resumableSessionsTotal}
          lastActivityAt={lastSessionActivityAt}
          unavailable={sessionsError}
        />
      )}

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(16rem,20rem)]">
        <div className="flex flex-col gap-6">
          <Section label="Runtime" data-testid="agent-overview-runtime">
            <MetadataList>
              <MetadataList.Row>
                <MetadataList.Term>Provider</MetadataList.Term>
                <MetadataList.Value>{agent.provider}</MetadataList.Value>
              </MetadataList.Row>
              <MetadataList.Row>
                <MetadataList.Term>Model</MetadataList.Term>
                <MetadataList.Value className={agent.model ? undefined : "text-muted"}>
                  {formatAbsentOverride(agent.model)}
                </MetadataList.Value>
              </MetadataList.Row>
              <MetadataList.Row>
                <MetadataList.Term>Command</MetadataList.Term>
                <MetadataList.Value className={agent.command ? "font-mono" : "text-muted"}>
                  {formatAbsentOverride(agent.command)}
                </MetadataList.Value>
              </MetadataList.Row>
              <MetadataList.Row>
                <MetadataList.Term>Permissions</MetadataList.Term>
                <MetadataList.Value className={agent.permissions ? undefined : "text-muted"}>
                  {agent.permissions?.trim() || "Default"}
                </MetadataList.Value>
              </MetadataList.Row>
            </MetadataList>
          </Section>

          <Section
            label="Live sessions"
            data-testid="agent-overview-live-sessions"
            right={
              <button
                type="button"
                className="text-small-body text-muted hover:text-fg"
                onClick={onViewAllSessions}
                data-testid="agent-overview-view-all-sessions"
              >
                View all ›
              </button>
            }
          >
            {sessionsLoading ? (
              <div className="flex flex-col gap-2">
                <Skeleton className="h-10 rounded-md" />
                <Skeleton className="h-10 rounded-md" />
              </div>
            ) : sessionsError ? (
              <p className="text-small-body text-muted">Session status unavailable</p>
            ) : liveSessions.length === 0 ? (
              <p className="text-small-body text-muted" data-testid="agent-overview-no-live">
                No active sessions
              </p>
            ) : (
              <ul className="flex flex-col gap-2">
                {liveSessions.map(session => {
                  const signals = deriveAgentFleetSignals([session]);
                  const iter =
                    session.activity?.iteration_current != null &&
                    session.activity?.iteration_max != null
                      ? `${session.activity.iteration_current}/${session.activity.iteration_max} iter`
                      : null;
                  const elapsed =
                    typeof session.activity?.elapsed_seconds === "number"
                      ? formatElapsed(session.activity.elapsed_seconds)
                      : null;
                  return (
                    <li key={session.id}>
                      <Link
                        to="/agents/$name/sessions/$id"
                        params={{ name: agent.name, id: session.id }}
                        className="flex items-center justify-between gap-3 rounded-md border border-line px-3 py-2 hover:bg-hover"
                        data-testid={`agent-overview-live-${session.id}`}
                      >
                        <span className="truncate text-body text-fg-strong">
                          {session.name || session.id}
                        </span>
                        <span className="shrink-0 font-mono text-badge tracking-mono text-muted">
                          {[iter, elapsed].filter(Boolean).join(" · ") ||
                            `${signals.active > 0 ? "active" : "idle"}`}
                        </span>
                      </Link>
                    </li>
                  );
                })}
              </ul>
            )}
          </Section>
        </div>

        <Section label="At a glance" data-testid="agent-overview-glance">
          <MetadataList>
            <MetadataList.Row>
              <MetadataList.Term>MCP servers</MetadataList.Term>
              <MetadataList.Value className={mcpNames.length === 0 ? "text-muted" : undefined}>
                {formatAbsentListLabels(mcpNames)}
              </MetadataList.Value>
            </MetadataList.Row>
            <MetadataList.Row>
              <MetadataList.Term>Tools</MetadataList.Term>
              <MetadataList.Value>
                {toolsCount} allow · {denyCount} deny
              </MetadataList.Value>
            </MetadataList.Row>
            <MetadataList.Row>
              <MetadataList.Term>Toolsets</MetadataList.Term>
              <MetadataList.Value className={toolsetsCount === 0 ? "text-muted" : undefined}>
                {toolsetsCount === 0 ? "None" : String(toolsetsCount)}
              </MetadataList.Value>
            </MetadataList.Row>
            <MetadataList.Row>
              <MetadataList.Term>Prompt</MetadataList.Term>
              <MetadataList.Value>{formatPromptWordCount(agent.prompt)}</MetadataList.Value>
            </MetadataList.Row>
            <MetadataList.Row>
              <MetadataList.Term>Skills</MetadataList.Term>
              <MetadataList.Value data-testid="agent-overview-skills">
                {formatSkillsPolicyLine(agent.skills)}
              </MetadataList.Value>
            </MetadataList.Row>
          </MetadataList>
        </Section>
      </div>
    </div>
  );
}

function formatElapsed(totalSeconds: number): string {
  if (!Number.isFinite(totalSeconds) || totalSeconds <= 0) return "";
  if (totalSeconds < 1) return "0s";
  const total = Math.floor(totalSeconds);
  if (total < 60) return `${total}s`;
  const minutes = Math.floor(total / 60);
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  const remainderMinutes = minutes % 60;
  return remainderMinutes === 0 ? `${hours}h` : `${hours}h ${remainderMinutes}m`;
}
