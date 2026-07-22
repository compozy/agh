import { type FilterI18nConfig } from "./hooks/use-filter-context";
import { useFilterSubmenuContent } from "./hooks/use-filter-submenu-content";
import { Input } from "@agh/ui/components/input";
import { cn } from "@agh/ui/lib/utils";
import {
  DropdownMenuCheckboxItem,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
} from "@agh/ui/components/dropdown-menu";
import { ScrollArea } from "@agh/ui/components/scroll-area";
import type React from "react";
import { type FiltersMenuAction } from "./hooks/use-filters";
import { createFilter } from "./hooks/filter-helpers";
import { Kbd } from "@agh/ui/components/kbd";

import type { Filter, FilterFieldConfig } from "./filters-types";

interface FilterSubmenuContentProps<T = unknown> {
  field: FilterFieldConfig<T>;
  currentValues: T[];
  isMultiSelect: boolean;
  onToggle: (value: T, isSelected: boolean) => void;
  i18n: FilterI18nConfig;
  isActive?: boolean;
  onActive?: () => void;
  onBack?: () => void;
  onClose?: () => void;
}

function FilterSubmenuContent<T = unknown>({
  field,
  currentValues,
  isMultiSelect,
  onToggle,
  i18n,
  isActive,
  onActive,
  onBack,
  onClose,
}: FilterSubmenuContentProps<T>) {
  const {
    activeHighlightedIndex,
    baseId,
    filteredOptions,
    focusSubmenuListbox,
    focusSubmenuSearchInput,
    handleListboxKeyDown,
    handleSearchInputChange,
    handleSearchInputKeyDown,
    highlightSubmenuOption,
    inputRef,
    searchInput,
  } = useFilterSubmenuContent({
    field,
    currentValues,
    isMultiSelect,
    isActive,
    onBack,
    onClose,
    onToggle,
  });
  const selectedValues = new Set(currentValues);

  return (
    <div className="flex flex-col" onMouseEnter={onActive}>
      {field.searchable !== false && (
        <>
          <Input
            ref={focusSubmenuSearchInput}
            role="combobox"
            aria-autocomplete="list"
            aria-expanded={true}
            aria-haspopup="listbox"
            aria-controls={`${baseId}-listbox`}
            aria-activedescendant={
              activeHighlightedIndex >= 0 ? `${baseId}-item-${activeHighlightedIndex}` : undefined
            }
            placeholder={i18n.placeholders.searchField(field.label || "")}
            className={cn(
              "h-8 rounded-none border-0 bg-transparent! px-2 shadow-none",
              "focus-visible:border-border focus-visible:ring-0 focus-visible:ring-offset-0",
              isActive && "placeholder:text-foreground"
            )}
            value={searchInput}
            onBlur={() => isActive && inputRef.current?.focus()}
            onChange={handleSearchInputChange}
            onFocus={() => onActive?.()}
            onMouseEnter={e => {
              onActive?.();
              e.stopPropagation();
            }}
            onClick={e => e.stopPropagation()}
            onKeyDown={handleSearchInputKeyDown}
          />
          <DropdownMenuSeparator />
        </>
      )}
      <div className="relative flex max-h-full">
        <div
          className="flex max-h-[min(var(--available-height),24rem)] w-full scroll-pt-2 scroll-pb-2 flex-col overscroll-contain outline-hidden"
          role="listbox"
          id={`${baseId}-listbox`}
          ref={focusSubmenuListbox}
          tabIndex={field.searchable === false ? 0 : -1}
          onKeyDown={handleListboxKeyDown}
        >
          <ScrollArea className="size-full min-h-0 **:data-[slot=scroll-area-scrollbar]:m-0 **:data-[slot=scroll-area-viewport]:h-full **:data-[slot=scroll-area-viewport]:overscroll-contain">
            {filteredOptions.length === 0 ? (
              <div className="text-muted-foreground py-2 text-center text-small-body">
                {i18n.noResultsFound}
              </div>
            ) : (
              <DropdownMenuGroup>
                {filteredOptions.map((option, index) => {
                  const isSelected = selectedValues.has(option.value);
                  const isHighlighted = activeHighlightedIndex === index;
                  const itemId = `${baseId}-item-${index}`;

                  return (
                    <DropdownMenuCheckboxItem
                      key={String(option.value)}
                      id={itemId}
                      role="option"
                      aria-selected={isHighlighted}
                      data-highlighted={isHighlighted || undefined}
                      onMouseEnter={() => highlightSubmenuOption(index)}
                      checked={isSelected}
                      className={cn(
                        "data-highlighted:bg-accent data-highlighted:text-accent-foreground",
                        option.className
                      )}
                      onSelect={e => {
                        if (isMultiSelect) e.preventDefault();
                      }}
                      onCheckedChange={() => onToggle(option.value as T, isSelected)}
                    >
                      {option.icon}
                      <span className="truncate">{option.label}</span>
                    </DropdownMenuCheckboxItem>
                  );
                })}
              </DropdownMenuGroup>
            )}
          </ScrollArea>
        </div>
      </div>
    </div>
  );
}

interface FiltersMenuFieldListProps<T = unknown> {
  activeMenu: string;
  addFilter: (fieldKey: string) => void;
  filters: Filter<T>[];
  filteredFields: FilterFieldConfig<T>[];
  highlightRootOption: (index: number) => void;
  i18n: FilterI18nConfig;
  markLastAddedFilter: (filterId: string) => void;
  onChange: (filters: Filter<T>[]) => void;
  openSubMenu: string | null;
  rootHighlightedIndex: number;
  rootId: string;
  sessionFilterIds: Record<string, string>;
  setMenuState: React.Dispatch<FiltersMenuAction>;
}

export function FiltersMenuFieldList<T = unknown>({
  activeMenu,
  addFilter,
  filters,
  filteredFields,
  highlightRootOption,
  i18n,
  markLastAddedFilter,
  onChange,
  openSubMenu,
  rootHighlightedIndex,
  rootId,
  sessionFilterIds,
  setMenuState,
}: FiltersMenuFieldListProps<T>) {
  if (filteredFields.length === 0) {
    return (
      <div className="text-muted-foreground py-2 text-center text-small-body">
        {i18n.noFieldsFound}
      </div>
    );
  }

  return filteredFields.map((field, index) => {
    const isHighlighted = rootHighlightedIndex === index;
    const itemId = `${rootId}-item-${index}`;
    const hasSubMenu =
      (field.type === "select" || field.type === "multiselect") && field.options?.length;

    if (hasSubMenu) {
      const isMultiSelect = field.type === "multiselect";
      const fieldKey = field.key as string;
      const sessionFilterId = sessionFilterIds[fieldKey];
      const sessionFilter = sessionFilterId
        ? filters.find(item => item.id === sessionFilterId)
        : null;
      const currentValues = sessionFilter?.values || [];

      return (
        <DropdownMenuSub
          key={fieldKey}
          open={openSubMenu === fieldKey}
          onOpenChange={open => {
            if (open) {
              setMenuState({ openSubMenu: fieldKey });
            } else if (openSubMenu === fieldKey) {
              setMenuState({ openSubMenu: null, activeMenu: "root" });
            }
          }}
        >
          <DropdownMenuSubTrigger
            id={itemId}
            role="option"
            aria-selected={isHighlighted}
            data-highlighted={isHighlighted || undefined}
            onMouseEnter={() => {
              highlightRootOption(index);
              setMenuState({ activeMenu: "root" });
            }}
            className="data-popup-open:bg-accent data-popup-open:text-accent-foreground data-highlighted:bg-accent data-highlighted:text-accent-foreground"
          >
            {field.icon}
            <span>{field.label}</span>
          </DropdownMenuSubTrigger>
          <DropdownMenuSubContent className="w-filters-menu-default" side="right">
            <FilterSubmenuContent
              field={field}
              currentValues={currentValues}
              isMultiSelect={isMultiSelect}
              i18n={i18n}
              isActive={activeMenu === fieldKey}
              onActive={() => {
                if (field.searchable !== false) {
                  setMenuState({ activeMenu: fieldKey });
                }
              }}
              onBack={() => setMenuState({ openSubMenu: null, activeMenu: "root" })}
              onClose={() => setMenuState({ addFilterOpen: false })}
              onToggle={(value, isSelected) => {
                if (isMultiSelect) {
                  const nextValues = isSelected
                    ? (currentValues.filter(item => item !== value) as T[])
                    : ([...currentValues, value] as T[]);

                  if (sessionFilter) {
                    if (nextValues.length === 0) {
                      onChange(filters.filter(item => item.id !== sessionFilter.id));
                      setMenuState(state => ({
                        sessionFilterIds: {
                          ...state.sessionFilterIds,
                          [fieldKey]: "",
                        },
                      }));
                    } else {
                      onChange(
                        filters.map(item =>
                          item.id === sessionFilter.id ? { ...item, values: nextValues } : item
                        )
                      );
                    }
                  } else {
                    const newFilter = createFilter<T>(
                      fieldKey,
                      field.defaultOperator || "is_any_of",
                      nextValues
                    );
                    onChange([...filters, newFilter]);
                    setMenuState(state => ({
                      sessionFilterIds: {
                        ...state.sessionFilterIds,
                        [fieldKey]: newFilter.id,
                      },
                    }));
                  }
                  return;
                }

                const newFilter = createFilter<T>(fieldKey, field.defaultOperator || "is", [
                  value,
                ] as T[]);
                markLastAddedFilter(newFilter.id);
                onChange([...filters, newFilter]);
                setMenuState({ addFilterOpen: false });
              }}
            />
          </DropdownMenuSubContent>
        </DropdownMenuSub>
      );
    }

    return (
      <DropdownMenuItem
        key={field.key}
        id={itemId}
        role="option"
        aria-selected={isHighlighted}
        data-highlighted={isHighlighted || undefined}
        onMouseEnter={() => highlightRootOption(index)}
        onClick={() => field.key && addFilter(field.key)}
        className="data-highlighted:bg-accent data-highlighted:text-accent-foreground"
      >
        {field.icon}
        <span>{field.label}</span>
      </DropdownMenuItem>
    );
  });
}

interface FiltersMenuSearchInputProps<T = unknown> {
  activeMenu: string;
  addFilter: (fieldKey: string) => void;
  addFilterOpen: boolean;
  enableShortcut: boolean;
  filteredFields: FilterFieldConfig<T>[];
  focusRootInput: (node: HTMLInputElement | null) => void;
  highlightRootOption: (index: number) => void;
  i18n: FilterI18nConfig;
  menuSearchInput: string;
  openSubMenu: string | null;
  rootHighlightedIndex: number;
  rootId: string;
  rootInputRef: React.RefObject<HTMLInputElement | null>;
  setMenuState: React.Dispatch<FiltersMenuAction>;
  shortcutLabel?: string;
}

export function FiltersMenuSearchInput<T = unknown>({
  activeMenu,
  addFilter,
  addFilterOpen,
  enableShortcut,
  filteredFields,
  focusRootInput,
  highlightRootOption,
  i18n,
  menuSearchInput,
  openSubMenu,
  rootHighlightedIndex,
  rootId,
  rootInputRef,
  setMenuState,
  shortcutLabel,
}: FiltersMenuSearchInputProps<T>) {
  return (
    <>
      <div className="relative">
        <Input
          ref={focusRootInput}
          role="combobox"
          aria-expanded={addFilterOpen}
          aria-controls={`${rootId}-listbox`}
          aria-activedescendant={
            rootHighlightedIndex >= 0 ? `${rootId}-item-${rootHighlightedIndex}` : undefined
          }
          placeholder={i18n.searchFields}
          className={cn(
            "h-8 rounded-none border-0 bg-transparent! px-2 shadow-none",
            "focus-visible:border-border focus-visible:ring-0 focus-visible:ring-offset-0",
            activeMenu === "root" && "placeholder:text-foreground"
          )}
          value={menuSearchInput}
          onFocus={() => setMenuState({ activeMenu: "root" })}
          onMouseEnter={() => setMenuState({ activeMenu: "root" })}
          onBlur={() => activeMenu === "root" && rootInputRef.current?.focus()}
          onChange={event =>
            setMenuState({
              menuSearchInput: event.target.value,
              highlightedIndex: -1,
            })
          }
          onClick={event => event.stopPropagation()}
          onKeyDown={event => {
            if (event.key === "ArrowDown") {
              event.preventDefault();
              if (filteredFields.length > 0) {
                highlightRootOption(
                  rootHighlightedIndex < filteredFields.length - 1 ? rootHighlightedIndex + 1 : 0
                );
              }
            } else if (event.key === "ArrowUp") {
              event.preventDefault();
              if (filteredFields.length > 0) {
                highlightRootOption(
                  rootHighlightedIndex > 0 ? rootHighlightedIndex - 1 : filteredFields.length - 1
                );
              }
            } else if (
              (event.key === "ArrowRight" || event.key === "ArrowLeft") &&
              rootHighlightedIndex >= 0
            ) {
              const field = filteredFields[rootHighlightedIndex];
              const hasSubMenu =
                field &&
                (field.type === "select" || field.type === "multiselect") &&
                field.options?.length;

              if (event.key === "ArrowRight" && hasSubMenu) {
                event.preventDefault();
                setMenuState({
                  openSubMenu: field.key || null,
                  activeMenu: field.key || "root",
                });
              } else if (event.key === "ArrowLeft" && openSubMenu) {
                event.preventDefault();
                setMenuState({ openSubMenu: null, activeMenu: "root" });
              }
            } else if (event.key === "Enter" && rootHighlightedIndex >= 0) {
              event.preventDefault();
              const field = filteredFields[rootHighlightedIndex];
              if (field.key) {
                const hasSubMenu =
                  (field.type === "select" || field.type === "multiselect") &&
                  field.options?.length;
                if (!hasSubMenu) {
                  addFilter(field.key);
                } else if (openSubMenu === field.key) {
                  setMenuState({ openSubMenu: null, activeMenu: "root" });
                } else {
                  setMenuState({ openSubMenu: field.key, activeMenu: field.key });
                }
              }
            } else if (event.key === "Escape") {
              setMenuState({ addFilterOpen: false });
            }
            event.stopPropagation();
          }}
        />
        {enableShortcut && shortcutLabel && (
          <Kbd className="bg-background absolute top-1/2 right-2 -translate-y-1/2 border">
            {shortcutLabel}
          </Kbd>
        )}
      </div>
      <DropdownMenuSeparator />
    </>
  );
}
