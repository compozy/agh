import { Repeat2 } from "lucide-react";

import { Empty, Eyebrow } from "@agh/ui";

import type { LoopBindingKind } from "../../lib/loop-bindings";
import type { LoopCatalogFilter, LoopKindFilter } from "../../lib/loop-catalog";
import { countByKind, groupLoopCatalog, loopCategories } from "../../lib/loop-catalog";
import type { LoopCatalogEntry } from "../../types";
import { LoopCatalogFilters } from "./loop-catalog-filters";
import { LoopCatalogRow } from "./loop-catalog-row";

interface LoopCatalogProps {
  entries: readonly LoopCatalogEntry[];
  filter: LoopCatalogFilter;
  onFilterChange: (filter: LoopCatalogFilter) => void;
  /** Attached loop-target automation kinds per loop name, for the binding badge. */
  boundLoops: ReadonlyMap<string, readonly LoopBindingKind[]>;
  onRun: (entry: LoopCatalogEntry) => void;
}

const NO_BINDINGS: readonly LoopBindingKind[] = [];

/**
 * The Loops catalog: kind + category filters over a grouped list (Read-only,
 * Workspace), each row carrying its goal, meta, binding badge, last outcome, 30d
 * success rate, and an inline Run launch (design §4.1).
 */
export function LoopCatalog({
  entries,
  filter,
  onFilterChange,
  boundLoops,
  onRun,
}: LoopCatalogProps) {
  const categories = loopCategories(entries);
  const kindCounts: Record<LoopKindFilter, number> = {
    all: countByKind(entries, "all"),
    "read-only": countByKind(entries, "read-only"),
    workspace: countByKind(entries, "workspace"),
  };
  const groups = groupLoopCatalog(entries, filter);
  return (
    <div className="flex flex-col gap-5" data-testid="loop-catalog">
      <LoopCatalogFilters
        kind={filter.kind}
        onKindChange={kind => onFilterChange({ ...filter, kind })}
        kindCounts={kindCounts}
        categories={categories}
        category={filter.category}
        onCategoryChange={category => onFilterChange({ ...filter, category })}
      />
      {groups.length === 0 ? (
        <Empty
          className="mx-auto my-10 max-w-md"
          description="No Loops match the current kind and category filters."
          icon={Repeat2}
          title="No matching loops"
        />
      ) : (
        groups.map(group => (
          <section key={group.kind} data-testid={`loop-group-${group.kind}`}>
            <div className="flex items-center gap-2 px-0.5 pb-2">
              <Eyebrow className="text-subtle">{group.label}</Eyebrow>
              <span className="font-mono text-[11px] tabular-nums text-faint">
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
        ))
      )}
    </div>
  );
}
