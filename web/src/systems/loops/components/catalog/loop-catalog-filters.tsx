import { ListFilter } from "lucide-react";
import { useMemo } from "react";

import { Button } from "@agh/ui";
import { Filters, type Filter } from "@agh/ui/components/reui/filters";

import type { LoopCatalogEntry } from "../../types";
import type { LoopKindFilter, LoopStatusFilter } from "../../lib/loop-catalog";
import { loopCategories, loopStatuses } from "../../lib/loop-catalog";
import {
  applyLoopFilterChips,
  buildLoopFilterFields,
  loopFiltersToChips,
} from "../../lib/loop-list-filters";

export interface LoopCatalogFiltersProps {
  /** Catalog entries used to derive present category/status options. */
  entries: readonly LoopCatalogEntry[];
  kindFilter: LoopKindFilter;
  categoryFilter: string | null;
  statusFilter: LoopStatusFilter | null;
  onKindFilterChange: (next: LoopKindFilter) => void;
  onCategoryFilterChange: (next: string | null) => void;
  onStatusFilterChange: (next: LoopStatusFilter | null) => void;
}

/**
 * Loops filter chip bar for composition inside ListingToolbar.Filters.
 * Kind / category / status options are data-driven from the catalog (truthful UI).
 */
function LoopCatalogFilters({
  entries,
  kindFilter,
  categoryFilter,
  statusFilter,
  onKindFilterChange,
  onCategoryFilterChange,
  onStatusFilterChange,
}: LoopCatalogFiltersProps) {
  const categoryOptions = useMemo(() => loopCategories(entries), [entries]);
  const statusOptions = useMemo(() => loopStatuses(entries), [entries]);
  const fields = useMemo(
    () => buildLoopFilterFields(categoryOptions, statusOptions),
    [categoryOptions, statusOptions]
  );
  const chips = useMemo(
    () =>
      loopFiltersToChips({
        kind: kindFilter,
        category: categoryFilter,
        status: statusFilter,
      }),
    [categoryFilter, kindFilter, statusFilter]
  );

  const handleFiltersChange = (next: Filter<string>[]) => {
    applyLoopFilterChips(next, {
      onKindChange: onKindFilterChange,
      onCategoryChange: onCategoryFilterChange,
      onStatusChange: onStatusFilterChange,
    });
  };

  return (
    <Filters<string>
      allowMultiple={false}
      data-testid="loop-catalog-filters"
      fields={fields}
      filters={chips}
      onChange={handleFiltersChange}
      size="sm"
      trigger={
        <Button
          aria-label="Add filter"
          data-testid="loop-catalog-filters-add"
          size="sm"
          type="button"
          variant="ghost"
        >
          <ListFilter aria-hidden="true" className="size-3" />
          Filter
        </Button>
      }
    />
  );
}

export { LoopCatalogFilters };
