import { PillGroup, Skeleton, type PillGroupItem } from "@agh/ui";

import type { SessionPayload } from "@/systems/session";

import { filterAgentSessionsByStatus, type AgentSessionFilter } from "../lib/agent-detail-search";
import { AgentSessionsList } from "./agent-sessions-list";
import { AgentStatsGrid } from "./agent-stats-grid";

const FILTER_ITEMS: PillGroupItem<AgentSessionFilter>[] = [
  { value: "all", label: "All", testId: "agent-sessions-filter-all" },
  { value: "active", label: "Active", testId: "agent-sessions-filter-active" },
  { value: "failed", label: "Failed", testId: "agent-sessions-filter-failed" },
  { value: "done", label: "Done", testId: "agent-sessions-filter-done" },
];

export interface AgentSessionsTabProps {
  agentName: string;
  sessions: SessionPayload[];
  total: number;
  active: number;
  resumable: number;
  lastActivityAt: string | null;
  status: "loading" | "error" | "ready";
  paginationStatus?: "available" | "loading";
  onLoadMore: () => void;
  filter: AgentSessionFilter;
  onFilterChange: (filter: AgentSessionFilter) => void;
  onNewSession: () => void;
  onClearFilter: () => void;
}

export function AgentSessionsTab({
  agentName,
  sessions,
  total,
  active,
  resumable,
  lastActivityAt,
  status,
  paginationStatus,
  onLoadMore,
  filter,
  onFilterChange,
  onNewSession,
  onClearFilter,
}: AgentSessionsTabProps) {
  const filtered = filterAgentSessionsByStatus(sessions, filter);
  const emptyTitle = filter === "all" ? "No sessions for this agent" : `No ${filter} sessions`;
  const emptyDescription = filter === "all" ? "Start a session to see it here." : undefined;

  return (
    <div className="flex flex-col gap-4" data-testid="agent-sessions-tab">
      {status === "loading" ? (
        <div
          className="grid grid-cols-2 gap-3 md:grid-cols-4"
          data-testid="agent-sessions-metrics-loading"
        >
          {Array.from({ length: 4 }).map((_, index) => (
            <Skeleton key={index} className="h-20 rounded-md" />
          ))}
        </div>
      ) : (
        <AgentStatsGrid
          total={total}
          active={active}
          resumable={resumable}
          lastActivityAt={lastActivityAt}
          unavailable={status === "error"}
        />
      )}
      <PillGroup
        aria-label="Session filter"
        data-testid="agent-sessions-filter"
        items={FILTER_ITEMS}
        onChange={onFilterChange}
        size="md"
        value={filter}
      />
      <AgentSessionsList
        agentName={agentName}
        sessions={filtered}
        status={status}
        paginationStatus={paginationStatus}
        onLoadMore={onLoadMore}
        emptyTitle={emptyTitle}
        emptyDescription={
          emptyDescription ?? (
            <button
              type="button"
              className="text-accent hover:underline"
              onClick={onClearFilter}
              data-testid="agent-sessions-show-all"
            >
              Show all
            </button>
          )
        }
      />
      {filter === "all" && status === "ready" && sessions.length === 0 ? (
        <div className="flex justify-center">
          <button
            type="button"
            className="text-small-body text-accent hover:underline"
            onClick={onNewSession}
            data-testid="agent-sessions-empty-new"
          >
            New session
          </button>
        </div>
      ) : null}
    </div>
  );
}
