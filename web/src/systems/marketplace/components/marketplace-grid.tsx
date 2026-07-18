import { Skeleton, type ListingViewMode } from "@agh/ui";

import type { MarketplaceListing } from "../types";
import { MarketplaceCard } from "./marketplace-card";
import { MarketplaceRow } from "./marketplace-row";

interface MarketplaceGridProps {
  entries: readonly MarketplaceListing[];
  isEntryPending?: (entry: MarketplaceListing) => boolean;
  onAction: (entry: MarketplaceListing) => void;
  view?: ListingViewMode;
}

function MarketplaceGrid({
  entries,
  isEntryPending,
  onAction,
  view = "rows",
}: MarketplaceGridProps) {
  if (view === "cards") {
    return (
      <div
        className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3"
        data-testid="marketplace-grid"
        data-view="cards"
      >
        {entries.map(entry => (
          <MarketplaceCard
            entry={entry}
            key={`${entry.kind}:${entry.entry_id}`}
            onAction={onAction}
            pending={isEntryPending?.(entry) ?? false}
          />
        ))}
      </div>
    );
  }

  return (
    <div
      className="overflow-hidden rounded-lg border border-line bg-canvas-soft"
      data-testid="marketplace-grid"
      data-view="rows"
    >
      {entries.map(entry => (
        <MarketplaceRow
          entry={entry}
          key={`${entry.kind}:${entry.entry_id}`}
          onAction={onAction}
          pending={isEntryPending?.(entry) ?? false}
        />
      ))}
    </div>
  );
}

function MarketplaceGridSkeleton({
  count = 3,
  view = "rows",
}: {
  count?: number;
  view?: ListingViewMode;
}) {
  if (view === "cards") {
    return (
      <div
        aria-label="Loading marketplace entries"
        className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3"
        role="status"
      >
        {Array.from({ length: count }, (_, index) => (
          <div className="flex min-h-38 flex-col rounded-lg bg-canvas-soft p-4" key={index}>
            <Skeleton className="h-2.5 w-3/5" />
            <Skeleton className="mt-1.5 h-2.5 w-5/6" />
            <Skeleton className="mt-auto h-2.5 w-2/5" />
          </div>
        ))}
      </div>
    );
  }

  return (
    <div
      aria-label="Loading marketplace entries"
      className="overflow-hidden rounded-lg border border-line bg-canvas-soft"
      role="status"
    >
      {Array.from({ length: count }, (_, index) => (
        <div
          className="flex items-center gap-3.5 border-b border-line-soft px-4 py-3 last:border-b-0"
          key={index}
        >
          <Skeleton className="size-[34px] shrink-0 rounded-md" />
          <div className="flex min-w-0 flex-1 flex-col gap-1.5">
            <Skeleton className="h-3 w-40" />
            <Skeleton className="h-2.5 w-64 max-w-full" />
          </div>
          <Skeleton className="h-3 w-20" />
        </div>
      ))}
    </div>
  );
}

export { MarketplaceGrid, MarketplaceGridSkeleton };
export type { MarketplaceGridProps };
