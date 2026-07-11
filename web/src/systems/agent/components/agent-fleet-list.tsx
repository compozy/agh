import { AlertTriangle, Users2 } from "lucide-react";

import {
  Alert,
  AlertDescription,
  Button,
  Empty,
  Skeleton,
  SkeletonRows,
  type ListingViewMode,
} from "@agh/ui";

import type { AgentFleetRowModel } from "../lib/agent-fleet-projection";
import { AgentFleetCard } from "./agent-fleet-card";
import { AgentFleetRow } from "./agent-fleet-row";

export interface AgentFleetListProps {
  rows: readonly AgentFleetRowModel[];
  view: ListingViewMode;
  isLoading: boolean;
  sessionsPartial: boolean;
  isFirstRunEmpty: boolean;
  isFilteredEmpty: boolean;
  newSessionDisabled?: boolean;
  onClearFilters: () => void;
  onCreateAgent: () => void;
  onNewSession: (agentName: string) => void;
}

function AgentFleetList({
  rows,
  view,
  isLoading,
  sessionsPartial,
  isFirstRunEmpty,
  isFilteredEmpty,
  newSessionDisabled = false,
  onClearFilters,
  onCreateAgent,
  onNewSession,
}: AgentFleetListProps) {
  if (isLoading) {
    if (view === "cards") {
      return (
        <div
          aria-busy="true"
          aria-label="Loading agents"
          className="grid grid-cols-1 gap-2.5 min-[720px]:grid-cols-2 min-[1100px]:grid-cols-3"
          data-testid="agent-fleet-loading"
        >
          {Array.from({ length: 6 }, (_, index) => (
            <div
              className="flex flex-col gap-3 rounded-lg bg-canvas-soft p-4"
              key={`agent-fleet-card-skeleton-${index}`}
            >
              <div className="flex items-start gap-3">
                <Skeleton className="size-6 shrink-0 rounded" />
                <div className="flex min-w-0 flex-1 flex-col gap-1.5">
                  <Skeleton className="h-3 w-32" />
                  <Skeleton className="h-2.5 w-48 max-w-full" />
                </div>
              </div>
              <Skeleton className="h-px w-full" />
              <div className="flex items-center justify-between">
                <Skeleton className="h-4 w-16" />
                <Skeleton className="size-[26px] rounded-md" />
              </div>
            </div>
          ))}
        </div>
      );
    }

    return (
      <div
        aria-busy="true"
        aria-label="Loading agents"
        className="min-h-0 flex-1 overflow-hidden rounded-lg border border-line-soft bg-canvas-soft"
        data-testid="agent-fleet-loading"
      >
        <SkeletonRows count={8} className="gap-0" rowClassName="px-4 py-3">
          <div className="flex items-center gap-3.5">
            <Skeleton className="size-[34px] shrink-0 rounded-md" />
            <div className="flex min-w-0 flex-1 flex-col gap-1.5">
              <Skeleton className="h-3 w-40" />
              <Skeleton className="h-2.5 w-64 max-w-full" />
            </div>
            <Skeleton className="h-3 w-24" />
          </div>
        </SkeletonRows>
      </div>
    );
  }

  if (isFirstRunEmpty) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
        data-testid="agent-fleet-empty"
      >
        <Empty
          action={
            <Button
              data-testid="agent-fleet-empty-create"
              onClick={onCreateAgent}
              size="sm"
              type="button"
            >
              New agent
            </Button>
          }
          description="Agents define the provider, model, and instructions a session runs with."
          icon={Users2}
          title="No agents yet"
        />
      </div>
    );
  }

  if (isFilteredEmpty) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
        data-testid="agent-fleet-filtered-empty"
      >
        <Empty
          action={
            <Button
              data-testid="agent-fleet-clear-filters"
              onClick={onClearFilters}
              size="sm"
              type="button"
              variant="ghost"
            >
              Clear filters
            </Button>
          }
          description="Try a different search or clear the active filters."
          icon={Users2}
          title="No agents match"
        />
      </div>
    );
  }

  return (
    <div className="min-h-0 flex-1" data-testid="agent-fleet-list">
      {sessionsPartial ? (
        <Alert data-testid="agent-fleet-sessions-notice" role="status" variant="warning">
          <AlertTriangle aria-hidden="true" className="size-4" />
          <AlertDescription>Session status unavailable</AlertDescription>
        </Alert>
      ) : null}
      {view === "cards" ? (
        <div
          className="grid grid-cols-1 gap-2.5 min-[720px]:grid-cols-2 min-[1100px]:grid-cols-3"
          data-slot="agent-fleet-cards"
          data-testid="agent-fleet-card-grid"
        >
          {rows.map(row => (
            <AgentFleetCard
              key={row.agent.name}
              newSessionDisabled={newSessionDisabled}
              onNewSession={onNewSession}
              row={row}
            />
          ))}
        </div>
      ) : (
        <div
          className="overflow-hidden rounded-lg border border-line-soft bg-canvas-soft"
          data-slot="agent-fleet-rows"
          role="list"
        >
          {rows.map(row => (
            <AgentFleetRow
              key={row.agent.name}
              newSessionDisabled={newSessionDisabled}
              onNewSession={onNewSession}
              row={row}
            />
          ))}
        </div>
      )}
    </div>
  );
}

export { AgentFleetList };
