"use client";

import { cn } from "../../lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from "../dropdown-menu";
import { ScrollArea } from "../scroll-area";
import { filtersContainerVariants } from "./filter-layout";
import type { FiltersProps } from "./filter-types";
import { FiltersContent } from "./filters-content";
import { FiltersMenuFieldList } from "./filters-menu";
import { FiltersMenuSearchInput } from "./filters-menu-search";
import { FilterContext, type FilterContextValue } from "./hooks/use-filter-context";
import { useFilters } from "./hooks/use-filters";

function Filters<T = unknown>({
  filters,
  fields,
  onChange,
  className,
  variant = "default",
  size = "default",
  radius = "default",
  i18n,
  showSearchInput = true,
  trigger,
  allowMultiple = true,
  menuPopupClassName,
  enableShortcut = false,
  shortcutKey = "f",
  shortcutLabel = "F",
}: FiltersProps<T>) {
  const state = useFilters({
    allowMultiple,
    enableShortcut,
    fields,
    filters,
    i18n,
    onChange,
    shortcutKey,
    trigger,
  });
  const contextValue: FilterContextValue = {
    variant,
    size,
    radius,
    i18n: state.mergedI18n,
    className,
    trigger,
    allowMultiple,
  };

  return (
    <FilterContext.Provider value={contextValue}>
      <div className={cn(filtersContainerVariants({ variant, size }), className)}>
        {state.selectableFields.length > 0 ? (
          <DropdownMenu open={state.addFilterOpen} onOpenChange={state.handleAddFilterOpenChange}>
            <DropdownMenuTrigger render={state.triggerButton} />
            <DropdownMenuContent
              className={cn("w-filters-menu-stack", menuPopupClassName)}
              align="start"
            >
              {showSearchInput ? (
                <FiltersMenuSearchInput
                  activeMenu={state.activeMenu}
                  addFilter={state.addFilter}
                  addFilterOpen={state.addFilterOpen}
                  enableShortcut={enableShortcut}
                  filteredFields={state.filteredFields}
                  focusRootInput={state.focusRootInput}
                  highlightRootOption={state.highlightRootOption}
                  i18n={state.mergedI18n}
                  menuSearchInput={state.menuSearchInput}
                  openSubMenu={state.openSubMenu}
                  rootHighlightedIndex={state.rootHighlightedIndex}
                  rootId={state.rootId}
                  rootInputRef={state.rootInputRef}
                  setMenuState={state.setMenuState}
                  shortcutLabel={shortcutLabel}
                />
              ) : null}

              <div className="relative flex max-h-full">
                <div
                  className="flex max-h-[min(var(--available-height),24rem)] w-full scroll-pt-2 scroll-pb-2 flex-col overscroll-contain"
                  role="listbox"
                  id={`${state.rootId}-listbox`}
                  onMouseEnter={state.activateRootMenu}
                >
                  <ScrollArea className="**:data-[slot=scroll-area-scrollbar]:m-0">
                    <FiltersMenuFieldList
                      activeMenu={state.activeMenu}
                      addFilter={state.addFilter}
                      filters={filters}
                      filteredFields={state.filteredFields}
                      highlightRootOption={state.highlightRootOption}
                      i18n={state.mergedI18n}
                      markLastAddedFilter={state.markLastAddedFilter}
                      onChange={onChange}
                      openSubMenu={state.openSubMenu}
                      rootHighlightedIndex={state.rootHighlightedIndex}
                      rootId={state.rootId}
                      sessionFilterIds={state.sessionFilterIds}
                      setMenuState={state.setMenuState}
                    />
                  </ScrollArea>
                </div>
              </div>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : null}

        <FiltersContent
          filters={filters}
          fields={fields}
          onChange={onChange}
          focusFilterId={state.lastAddedFilterId}
        />
      </div>
    </FilterContext.Provider>
  );
}

export { Filters };
export type {
  CustomRendererProps,
  Filter,
  FilterFieldConfig,
  FilterFieldGroup,
  FilterFieldsConfig,
  FilterGroup,
  FilterOperator,
  FilterOption,
} from "./filter-types";
