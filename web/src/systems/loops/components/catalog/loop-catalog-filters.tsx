import { PillGroup, type PillGroupItem } from "@agh/ui";

import type { LoopKindFilter } from "../../lib/loop-catalog";

interface LoopCatalogFiltersProps {
  kind: LoopKindFilter;
  onKindChange: (kind: LoopKindFilter) => void;
  kindCounts: Record<LoopKindFilter, number>;
  categories: readonly string[];
  category: string | null;
  onCategoryChange: (category: string | null) => void;
}

const KIND_LABELS: Record<LoopKindFilter, string> = {
  all: "All",
  "read-only": "Read-only",
  workspace: "Workspace",
};

const KIND_ORDER: readonly LoopKindFilter[] = ["all", "read-only", "workspace"];

/**
 * Catalog toolbar: a kind segment (All / Read-only / Workspace with counts) and a
 * data-driven category pill row. Categories come from the values actually present
 * in the catalog, so no empty taxonomy renders.
 */
export function LoopCatalogFilters({
  kind,
  onKindChange,
  kindCounts,
  categories,
  category,
  onCategoryChange,
}: LoopCatalogFiltersProps) {
  const kindItems: PillGroupItem<LoopKindFilter>[] = KIND_ORDER.map(value => ({
    value,
    label: KIND_LABELS[value],
    badge: kindCounts[value],
    testId: `loop-kind-${value}`,
  }));
  return (
    <div className="flex flex-wrap items-center gap-2.5" data-testid="loop-catalog-filters">
      <PillGroup
        aria-label="Filter by kind"
        items={kindItems}
        value={kind}
        onChange={onKindChange}
        size="sm"
      />
      {categories.length > 0 ? (
        <div className="flex flex-wrap items-center gap-0.5" aria-label="Filter by category">
          <CategoryPill
            active={category === null}
            label="All categories"
            onSelect={() => onCategoryChange(null)}
            testId="loop-category-all"
          />
          {categories.map(name => (
            <CategoryPill
              key={name}
              active={category === name}
              label={name}
              onSelect={() => onCategoryChange(name)}
              testId={`loop-category-${name}`}
            />
          ))}
        </div>
      ) : null}
    </div>
  );
}

interface CategoryPillProps {
  active: boolean;
  label: string;
  onSelect: () => void;
  testId: string;
}

function CategoryPill({ active, label, onSelect, testId }: CategoryPillProps) {
  return (
    <button
      type="button"
      aria-pressed={active}
      data-testid={testId}
      onClick={onSelect}
      className={`h-6 rounded-md px-2.5 text-xs transition-colors ${
        active
          ? "bg-row-hover text-fg-strong"
          : "text-subtle hover:bg-row-hover hover:text-fg-strong"
      }`}
    >
      {label}
    </button>
  );
}
