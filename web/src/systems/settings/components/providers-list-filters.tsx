import { ListFilter, Search } from "lucide-react";
import { useMemo } from "react";

import { Button, InputGroup, InputGroupAddon, InputGroupInput } from "@agh/ui";
import { Filters, type Filter } from "@agh/ui/components/reui/filters";

import {
  applyProviderFilterChips,
  buildProviderFilterFields,
  providerFiltersToChips,
  type ProviderFilterHandlers,
  type ProviderFilterState,
} from "../lib/providers-list-filters";

export interface ProvidersListFiltersProps extends ProviderFilterState, ProviderFilterHandlers {
  onNameQueryChange: (next: string) => void;
  visibleCount: number;
  totalCount: number;
}

export function ProvidersListFilters({
  statusFilter,
  sourceFilter,
  harnessFilter,
  authModeFilter,
  defaultFilter,
  nameQuery,
  visibleCount,
  totalCount,
  onStatusChange,
  onSourceChange,
  onHarnessChange,
  onAuthModeChange,
  onDefaultChange,
  onNameQueryChange,
}: ProvidersListFiltersProps) {
  const fields = useMemo(() => buildProviderFilterFields(), []);
  const chips = useMemo(
    () =>
      providerFiltersToChips({
        statusFilter,
        sourceFilter,
        harnessFilter,
        authModeFilter,
        defaultFilter,
      }),
    [statusFilter, sourceFilter, harnessFilter, authModeFilter, defaultFilter]
  );

  const handleFiltersChange = (next: Filter<string>[]) => {
    applyProviderFilterChips(next, {
      onStatusChange,
      onSourceChange,
      onHarnessChange,
      onAuthModeChange,
      onDefaultChange,
    });
  };

  const filtered = chips.length > 0 || nameQuery.trim().length > 0;
  const matchLabel = filtered
    ? `${visibleCount} of ${totalCount} providers`
    : `${totalCount} providers`;

  return (
    <div
      className="flex flex-wrap items-center gap-2 border-b border-line-soft pb-3"
      data-testid="providers-list-filters"
    >
      <InputGroup className="w-56">
        <InputGroupAddon>
          <Search aria-hidden="true" className="size-3 text-subtle" />
        </InputGroupAddon>
        <InputGroupInput
          aria-label="Search providers"
          placeholder="Search by name…"
          value={nameQuery}
          onChange={event => onNameQueryChange(event.target.value)}
          data-testid="providers-list-filters-search"
        />
      </InputGroup>

      <Filters<string>
        allowMultiple={false}
        fields={fields}
        filters={chips}
        onChange={handleFiltersChange}
        size="sm"
        trigger={
          <Button
            aria-label="Add filter"
            data-testid="providers-list-filters-add"
            size="sm"
            type="button"
            variant="ghost"
          >
            <ListFilter aria-hidden="true" className="size-3" />
            Filter
          </Button>
        }
      />

      <span
        className="ml-auto text-xs text-subtle tabular-nums"
        data-testid="providers-list-filters-count"
        aria-live="polite"
      >
        {matchLabel}
      </span>
    </div>
  );
}
