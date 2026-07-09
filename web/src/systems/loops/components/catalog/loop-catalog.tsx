import { Repeat2 } from "lucide-react";
import { useMemo } from "react";

import { Button, Empty, Eyebrow, type ListingViewMode } from "@agh/ui";

import type { LoopBindingKind } from "../../lib/loop-bindings";
import type { LoopCatalogFilter } from "../../lib/loop-catalog";
import { groupLoopCatalog, hasActiveLoopFilters } from "../../lib/loop-catalog";
import { filterLoopCatalog } from "../../lib/loop-list-filters";
import type { LoopCatalogEntry } from "../../types";
import { LoopCatalogCard } from "./loop-catalog-card";
import { LoopCatalogRow } from "./loop-catalog-row";

interface LoopCatalogProps {
  entries: readonly LoopCatalogEntry[];
  searchQuery: string;
  view: ListingViewMode;
  filter: LoopCatalogFilter;
  onClearFilters: () => void;
  /** Attached loop-target automation kinds per loop name, for the binding badge. */
  boundLoops: ReadonlyMap<string, readonly LoopBindingKind[]>;
  onRun: (entry: LoopCatalogEntry) => void;
}

const NO_BINDINGS: readonly LoopBindingKind[] = [];
const GROUP_PASS_THROUGH: LoopCatalogFilter = { kind: "all", category: null, status: null };

export function LoopCatalog({
  entries,
  searchQuery,
  view,
  filter,
  onClearFilters,
  boundLoops,
  onRun,
}: LoopCatalogProps) {
  const filtered = useMemo(
    () => filterLoopCatalog(entries, searchQuery, filter),
    [entries, filter, searchQuery]
  );
  const groups = useMemo(() => groupLoopCatalog(filtered, GROUP_PASS_THROUGH), [filtered]);
  const hasActiveFilters = hasActiveLoopFilters(searchQuery, filter);
  const isEmpty = filtered.length === 0;

  if (isEmpty) {
    return (
      <div
        className="flex min-h-60 items-center justify-center p-4"
        data-testid="loop-catalog-empty"
      >
        <Empty
          action={
            hasActiveFilters ? (
              <Button
                data-testid="loop-catalog-clear-filters"
                onClick={onClearFilters}
                size="sm"
                type="button"
                variant="ghost"
              >
                Clear filters
              </Button>
            ) : undefined
          }
          className="max-w-sm"
          description={
            hasActiveFilters
              ? "Try clearing search or filters."
              : "No Loop definitions are available in this workspace yet."
          }
          icon={Repeat2}
          title={hasActiveFilters ? "No matching loops" : "No loops yet"}
        />
      </div>
    );
  }

  if (view === "cards") {
    return (
      <div
        className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3"
        data-testid="loop-catalog-card-grid"
      >
        {filtered.map(entry => (
          <LoopCatalogCard key={entry.name} entry={entry} onRun={onRun} />
        ))}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-5" data-testid="loop-catalog">
      {groups.map(group => (
        <section key={group.kind} data-testid={`loop-group-${group.kind}`}>
          <div className="flex items-center gap-2 px-1 pb-2">
            <Eyebrow className="text-muted">{group.label}</Eyebrow>
            <span className="font-mono text-mono-id tabular-nums text-faint">
              {group.entries.length}
            </span>
          </div>
          <div className="overflow-hidden rounded-lg border border-line bg-canvas-soft">
            {group.entries.map(entry => (
              <LoopCatalogRow
                key={entry.name}
                entry={entry}
                bindingKinds={boundLoops.get(entry.name) ?? NO_BINDINGS}
                onRun={onRun}
              />
            ))}
          </div>
        </section>
      ))}
    </div>
  );
}
