import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
  PillGroup,
  SearchInput,
  type PillGroupItem,
} from "@agh/ui";

import type { ProviderStateLabel } from "../lib/provider-state";

export type ProvidersViewMode = "rows" | "cards";

const STATUS_LABEL: Record<ProviderStateLabel | "all", string> = {
  all: "All",
  installed: "Ready",
  unconfigured: "Needs setup",
  "binary-missing": "Not installed",
};

const VIEW_ITEMS: ReadonlyArray<PillGroupItem<ProvidersViewMode>> = [
  { value: "rows", label: "Rows", testId: "settings-providers-view-rows" },
  { value: "cards", label: "Cards", testId: "settings-providers-view-cards" },
];

export interface ProvidersToolbarProps {
  nameQuery: string;
  onNameQueryChange: (next: string) => void;
  statusFilter: ProviderStateLabel | null;
  onStatusChange: (next: ProviderStateLabel | null) => void;
  view: ProvidersViewMode;
  onViewChange: (next: ProvidersViewMode) => void;
}

/**
 * Provider listing toolbar (settings-providers prototype): search + one
 * status chip + Rows|Cards toggle. The six-select filter bar is gone.
 */
export function ProvidersToolbar({
  nameQuery,
  onNameQueryChange,
  statusFilter,
  onStatusChange,
  view,
  onViewChange,
}: ProvidersToolbarProps) {
  const statusValue = statusFilter ?? "all";

  return (
    <div className="flex flex-wrap items-center justify-between gap-2">
      <div className="flex min-w-0 flex-wrap items-center gap-2">
        <SearchInput
          aria-label="Search providers"
          className="w-56"
          data-testid="settings-providers-search"
          onChange={onNameQueryChange}
          placeholder="Search providers"
          value={nameQuery}
        />
        <DropdownMenu>
          <DropdownMenuTrigger
            aria-label="Filter by status"
            className="inline-flex h-7 items-center gap-1 rounded-md border border-line bg-btn-default-fill px-2.5 text-form-label text-fg transition-colors duration-base hover:bg-btn-default-hover focus-visible:outline-none focus-visible:shadow-focus-ring"
            data-testid="settings-providers-status-filter"
          >
            <span className="text-subtle">Status</span>
            <span aria-hidden="true" className="text-faint">
              :
            </span>
            <span className="font-medium">{STATUS_LABEL[statusValue]}</span>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start">
            <DropdownMenuRadioGroup
              onValueChange={value =>
                onStatusChange(value === "all" ? null : (value as ProviderStateLabel))
              }
              value={statusValue}
            >
              {(Object.keys(STATUS_LABEL) as Array<ProviderStateLabel | "all">).map(value => (
                <DropdownMenuRadioItem
                  data-testid={`settings-providers-status-${value}`}
                  key={value}
                  value={value}
                >
                  {STATUS_LABEL[value]}
                </DropdownMenuRadioItem>
              ))}
            </DropdownMenuRadioGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <PillGroup<ProvidersViewMode>
        aria-label="View mode"
        data-testid="settings-providers-view-toggle"
        items={VIEW_ITEMS}
        onChange={onViewChange}
        size="sm"
        value={view}
      />
    </div>
  );
}
