import { Skeleton, type ListingViewMode } from "@agh/ui";

export function InventorySkeleton({ view = "rows" }: { view?: ListingViewMode }) {
  if (view === "cards") {
    return (
      <div
        aria-busy="true"
        aria-label="Loading inventory"
        className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3"
        data-testid="extensions-loading"
      >
        {Array.from({ length: 6 }, (_, index) => (
          <div className="flex flex-col gap-3 rounded-lg bg-canvas-soft p-4" key={index}>
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
              <Skeleton className="size-button-icon-default rounded-md" />
            </div>
          </div>
        ))}
      </div>
    );
  }

  return (
    <div
      className="overflow-hidden rounded-lg border border-line bg-canvas-soft"
      data-testid="extensions-loading"
    >
      {[0, 1, 2].map(index => (
        <div
          className="grid grid-cols-[34px_minmax(0,1fr)_auto] gap-3.5 border-b border-line-soft px-4 py-3 last:border-0"
          key={index}
        >
          <Skeleton className="size-[34px]" />
          <div className="space-y-2">
            <Skeleton className="h-3 w-40" />
            <Skeleton className="h-3 w-72" />
          </div>
          <Skeleton className="h-5 w-20" />
        </div>
      ))}
    </div>
  );
}
