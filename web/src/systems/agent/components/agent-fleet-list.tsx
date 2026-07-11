import { Users2 } from "lucide-react";

import { Button, Empty, Skeleton, SkeletonRows } from "@agh/ui";

import type { AgentFleetRowModel } from "../lib/agent-fleet-projection";
import { AgentFleetRow } from "./agent-fleet-row";

export interface AgentFleetListProps {
  rows: readonly AgentFleetRowModel[];
  isLoading: boolean;
  sessionsPartial: boolean;
  isFirstRunEmpty: boolean;
  isFilteredEmpty: boolean;
  onClearFilters: () => void;
  onCreateAgent: () => void;
}

function AgentFleetList({
  rows,
  isLoading,
  sessionsPartial,
  isFirstRunEmpty,
  isFilteredEmpty,
  onClearFilters,
  onCreateAgent,
}: AgentFleetListProps) {
  if (isLoading) {
    return (
      <div data-testid="agent-fleet-loading" className="min-h-0 flex-1">
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
          title="No agents match"
        />
      </div>
    );
  }

  return (
    <div className="min-h-0 flex-1" data-testid="agent-fleet-list">
      {sessionsPartial ? (
        <p
          className="border-b border-line-soft px-4 py-2 text-small-body text-muted"
          data-testid="agent-fleet-sessions-notice"
          role="status"
        >
          Session status unavailable
        </p>
      ) : null}
      <div className="list" data-slot="agent-fleet-rows">
        {rows.map(row => (
          <AgentFleetRow key={row.agent.name} row={row} />
        ))}
      </div>
    </div>
  );
}

export { AgentFleetList };
