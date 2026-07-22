"use client";

import { cva } from "class-variance-authority";
import type React from "react";
import { XIcon } from "lucide-react";
import { FilterContext, type FilterI18nConfig, useFilterContext } from "./hooks/use-filter-context";
import { Button } from "@agh/ui/components/button";
import { getFieldsMap } from "./hooks/filter-helpers";
import { cn } from "@agh/ui/lib/utils";
import { ButtonGroup, ButtonGroupText } from "@agh/ui/components/button-group";
import { useFilters } from "./hooks/use-filters";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@agh/ui/components/dropdown-menu";
import { ScrollArea } from "@agh/ui/components/scroll-area";

import { FilterOperatorDropdown } from "./filter-operators";
import { FiltersMenuFieldList, FiltersMenuSearchInput } from "./filters-menu";
import { FilterValueSelector } from "./filter-value-selector";
import type { Filter, FilterFieldsConfig } from "./filters-types";

// Container variant for filters wrapper
const filtersContainerVariants = cva("flex flex-wrap items-center", {
  variants: {
    variant: {
      solid: "gap-2",
      default: "",
    },
    size: {
      sm: "gap-1.5",
      default: "gap-2.5",
      lg: "gap-3.5",
    },
  },
  defaultVariants: {
    variant: "default",
    size: "default",
  },
});

interface FilterRemoveButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  icon?: React.ReactNode;
}

function FilterRemoveButton({ className, icon = <XIcon />, ...props }: FilterRemoveButtonProps) {
  const context = useFilterContext();

  return (
    <Button
      variant="outline"
      size={context.size === "sm" ? "icon-sm" : context.size === "lg" ? "icon-lg" : "icon"}
      className={className}
      {...props}
    >
      {icon}
    </Button>
  );
}

interface FiltersContentProps<T = unknown> {
  filters: Filter<T>[];
  fields: FilterFieldsConfig<T>;
  onChange: (filters: Filter<T>[]) => void;
  focusFilterId?: string | null;
}

const FiltersContent = <T = unknown,>({
  filters,
  fields,
  onChange,
  focusFilterId = null,
}: FiltersContentProps<T>) => {
  const context = useFilterContext();
  const fieldsMap = getFieldsMap(fields);

  const updateFilter = (filterId: string, updates: Partial<Filter<T>>) => {
    onChange(
      filters.map(filter => {
        if (filter.id === filterId) {
          const updatedFilter = { ...filter, ...updates };
          if (updates.operator === "empty" || updates.operator === "not_empty") {
            updatedFilter.values = [] as T[];
          }
          return updatedFilter;
        }
        return filter;
      })
    );
  };

  const removeFilter = (filterId: string) => {
    onChange(filters.filter(filter => filter.id !== filterId));
  };

  return (
    <div
      className={cn(
        filtersContainerVariants({
          variant: context.variant,
          size: context.size,
        }),
        context.className
      )}
    >
      {filters.map(filter => {
        const field = fieldsMap[filter.field];
        if (!field) return null;

        if (field.type === "toggle") {
          return (
            <ButtonGroup key={filter.id}>
              <ButtonGroupText className="bg-background dark:bg-input/30">
                {field.icon}
                {field.label}
              </ButtonGroupText>
              <FilterRemoveButton onClick={() => removeFilter(filter.id)} />
            </ButtonGroup>
          );
        }

        return (
          <ButtonGroup key={filter.id}>
            <ButtonGroupText className="bg-background dark:bg-input/30">
              {field.icon}
              {field.label}
            </ButtonGroupText>

            <FilterOperatorDropdown<T>
              field={field}
              operator={filter.operator}
              values={filter.values}
              onChange={operator => updateFilter(filter.id, { operator })}
            />

            <FilterValueSelector<T>
              field={field}
              values={filter.values}
              onChange={values => updateFilter(filter.id, { values })}
              operator={filter.operator}
              focusOnMount={filter.id === focusFilterId}
            />

            <FilterRemoveButton onClick={() => removeFilter(filter.id)} />
          </ButtonGroup>
        );
      })}
    </div>
  );
};

interface FiltersProps<T = unknown> {
  filters: Filter<T>[];
  fields: FilterFieldsConfig<T>;
  onChange: (filters: Filter<T>[]) => void;
  className?: string;
  variant?: "solid" | "default";
  size?: "sm" | "default" | "lg";
  radius?: "default" | "full";
  i18n?: Partial<FilterI18nConfig>;
  showSearchInput?: boolean;
  trigger?: React.ReactNode;
  allowMultiple?: boolean;
  menuPopupClassName?: string;
  collapseAddButton?: boolean;
  enableShortcut?: boolean;
  shortcutKey?: string;
  shortcutLabel?: string;
}

export function Filters<T = unknown>({
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
  const {
    activateRootMenu,
    activeMenu,
    addFilter,
    addFilterOpen,
    filteredFields,
    focusRootInput,
    handleAddFilterOpenChange,
    highlightRootOption,
    lastAddedFilterId,
    markLastAddedFilter,
    mergedI18n,
    menuSearchInput,
    openSubMenu,
    rootHighlightedIndex,
    rootId,
    rootInputRef,
    selectableFields,
    sessionFilterIds,
    setMenuState,
    triggerButton,
  } = useFilters({
    allowMultiple,
    enableShortcut,
    fields,
    filters,
    i18n,
    onChange,
    shortcutKey,
    trigger,
  });
  const contextValue = {
    variant,
    size,
    radius,
    i18n: mergedI18n,
    className,
    trigger,
    allowMultiple,
  };

  return (
    <FilterContext.Provider value={contextValue}>
      <div className={cn(filtersContainerVariants({ variant, size }), className)}>
        {selectableFields.length > 0 && (
          <DropdownMenu open={addFilterOpen} onOpenChange={handleAddFilterOpenChange}>
            <DropdownMenuTrigger render={triggerButton} />
            <DropdownMenuContent
              className={cn("w-filters-menu-stack", menuPopupClassName)}
              align="start"
            >
              {showSearchInput && (
                <FiltersMenuSearchInput
                  activeMenu={activeMenu}
                  addFilter={addFilter}
                  addFilterOpen={addFilterOpen}
                  enableShortcut={enableShortcut}
                  filteredFields={filteredFields}
                  focusRootInput={focusRootInput}
                  highlightRootOption={highlightRootOption}
                  i18n={mergedI18n}
                  menuSearchInput={menuSearchInput}
                  openSubMenu={openSubMenu}
                  rootHighlightedIndex={rootHighlightedIndex}
                  rootId={rootId}
                  rootInputRef={rootInputRef}
                  setMenuState={setMenuState}
                  shortcutLabel={shortcutLabel}
                />
              )}

              <div className="relative flex max-h-full">
                <div
                  className="flex max-h-[min(var(--available-height),24rem)] w-full scroll-pt-2 scroll-pb-2 flex-col overscroll-contain"
                  role="listbox"
                  id={`${rootId}-listbox`}
                  onMouseEnter={activateRootMenu}
                >
                  <ScrollArea className="**:data-[slot=scroll-area-scrollbar]:m-0">
                    <FiltersMenuFieldList
                      activeMenu={activeMenu}
                      addFilter={addFilter}
                      filters={filters}
                      filteredFields={filteredFields}
                      highlightRootOption={highlightRootOption}
                      i18n={mergedI18n}
                      markLastAddedFilter={markLastAddedFilter}
                      onChange={onChange}
                      openSubMenu={openSubMenu}
                      rootHighlightedIndex={rootHighlightedIndex}
                      rootId={rootId}
                      sessionFilterIds={sessionFilterIds}
                      setMenuState={setMenuState}
                    />
                  </ScrollArea>
                </div>
              </div>
            </DropdownMenuContent>
          </DropdownMenu>
        )}

        <FiltersContent
          filters={filters}
          fields={fields}
          onChange={onChange}
          focusFilterId={lastAddedFilterId}
        />
      </div>
    </FilterContext.Provider>
  );
}

export type {
  CustomRendererProps,
  Filter,
  FilterFieldConfig,
  FilterFieldGroup,
  FilterFieldsConfig,
  FilterGroup,
  FilterOperator,
  FilterOption,
} from "./filters-types";
