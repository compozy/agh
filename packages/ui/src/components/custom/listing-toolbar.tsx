"use client";

import { LayoutGrid, List } from "lucide-react";
import * as React from "react";

import { cn } from "../../lib/utils";
import { PillGroup } from "./pill-group";
import { SearchInput, type SearchInputProps } from "./search-input";

export type ListingViewMode = "rows" | "cards";

type ListingToolbarRootProps = React.ComponentProps<"div">;
type ListingToolbarLeadingProps = React.ComponentProps<"div">;
type ListingToolbarTrailingProps = React.ComponentProps<"div">;
type ListingToolbarFiltersProps = React.ComponentProps<"div">;

type ListingToolbarSearchProps = SearchInputProps;

interface ListingToolbarViewToggleProps extends Omit<React.ComponentProps<"div">, "onChange"> {
  value: ListingViewMode;
  onChange: (next: ListingViewMode) => void;
}

const VIEW_ITEMS = [
  {
    value: "rows" as const,
    label: (
      <span className="inline-flex items-center gap-1.5">
        <List aria-hidden="true" className="size-3" />
        Rows
      </span>
    ),
    testId: "listing-view-rows",
  },
  {
    value: "cards" as const,
    label: (
      <span className="inline-flex items-center gap-1.5">
        <LayoutGrid aria-hidden="true" className="size-3" />
        Cards
      </span>
    ),
    testId: "listing-view-cards",
  },
];

/**
 * Canonical inventory listing toolbar shell.
 * Compose Search → Filters in Leading, ViewToggle in Trailing.
 * URL persistence stays in the route — this composite is presentational only.
 *
 * @example
 * ```tsx
 * <ListingToolbar>
 *   <ListingToolbar.Leading>
 *     <ListingToolbar.Search value={q} onChange={setQ} placeholder="Search skills" />
 *     <ListingToolbar.Filters>
 *       <Filters ... />
 *     </ListingToolbar.Filters>
 *   </ListingToolbar.Leading>
 *   <ListingToolbar.Trailing>
 *     <ListingToolbar.ViewToggle value={view} onChange={setView} />
 *   </ListingToolbar.Trailing>
 * </ListingToolbar>
 * ```
 */
function ListingToolbar({ className, ...props }: ListingToolbarRootProps) {
  return (
    <div
      data-slot="listing-toolbar"
      data-testid="listing-toolbar"
      className={cn("flex flex-wrap items-center gap-2.5", className)}
      {...props}
    />
  );
}

function ListingToolbarLeading({ className, ...props }: ListingToolbarLeadingProps) {
  return (
    <div
      data-slot="listing-toolbar-leading"
      className={cn("flex min-w-0 flex-1 flex-wrap items-center gap-2.5", className)}
      {...props}
    />
  );
}

function ListingToolbarTrailing({ className, ...props }: ListingToolbarTrailingProps) {
  return (
    <div
      data-slot="listing-toolbar-trailing"
      className={cn("ml-auto flex shrink-0 items-center", className)}
      {...props}
    />
  );
}

function ListingToolbarSearch({ kbd = "/", ...props }: ListingToolbarSearchProps) {
  return <SearchInput kbd={kbd} data-testid="listing-search-input" {...props} />;
}

function ListingToolbarFilters({ className, ...props }: ListingToolbarFiltersProps) {
  return (
    <div
      data-slot="listing-toolbar-filters"
      className={cn("flex min-w-0 flex-wrap items-center", className)}
      {...props}
    />
  );
}

function ListingToolbarViewToggle({
  value,
  onChange,
  className,
  ...props
}: ListingToolbarViewToggleProps) {
  return (
    <div data-slot="listing-toolbar-view" className={cn(className)} {...props}>
      <PillGroup<ListingViewMode>
        aria-label="View mode"
        data-testid="listing-view-toggle"
        items={VIEW_ITEMS}
        onChange={onChange}
        size="md"
        value={value}
      />
    </div>
  );
}

const ListingToolbarCompound = Object.assign(ListingToolbar, {
  Leading: ListingToolbarLeading,
  Trailing: ListingToolbarTrailing,
  Search: ListingToolbarSearch,
  Filters: ListingToolbarFilters,
  ViewToggle: ListingToolbarViewToggle,
});

export { ListingToolbarCompound as ListingToolbar };
export type {
  ListingToolbarFiltersProps,
  ListingToolbarLeadingProps,
  ListingToolbarRootProps as ListingToolbarProps,
  ListingToolbarSearchProps,
  ListingToolbarTrailingProps,
  ListingToolbarViewToggleProps,
};
