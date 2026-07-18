import type { ListingViewMode } from "@agh/ui";

import type { AutomationJob } from "../types";
import {
  AutomationCatalogShell,
  type AutomationCatalogPagination,
} from "./automation-catalog-shell";
import { AutomationJobCard } from "./automation-job-card";
import { AutomationJobRow } from "./automation-job-row";

export interface AutomationJobsCatalogProps {
  jobs: AutomationJob[];
  view: ListingViewMode;
  isLoading: boolean;
  errorMessage: string | null;
  hasActiveFilters: boolean;
  pagination: AutomationCatalogPagination;
  runPendingId: string | null;
  onClearFilters: () => void;
  onCreate: () => void;
  onRun: (id: string) => void;
}

/** Jobs catalog body: rows or cards with shared empty/loading/error/load-more. */
function AutomationJobsCatalog({
  jobs,
  view,
  isLoading,
  errorMessage,
  hasActiveFilters,
  pagination,
  runPendingId,
  onClearFilters,
  onCreate,
  onRun,
}: AutomationJobsCatalogProps) {
  return (
    <AutomationCatalogShell
      errorMessage={errorMessage}
      hasActiveFilters={hasActiveFilters}
      isLoading={isLoading}
      itemCount={jobs.length}
      kind="jobs"
      onClearFilters={onClearFilters}
      onCreate={onCreate}
      pagination={pagination}
      view={view}
    >
      {jobs.map(job =>
        view === "cards" ? (
          <AutomationJobCard
            isRunPending={runPendingId === job.id}
            job={job}
            key={job.id}
            onRun={onRun}
          />
        ) : (
          <AutomationJobRow
            isRunPending={runPendingId === job.id}
            job={job}
            key={job.id}
            onRun={onRun}
          />
        )
      )}
    </AutomationCatalogShell>
  );
}

export { AutomationJobsCatalog };
