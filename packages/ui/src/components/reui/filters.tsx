"use client";

import { cva } from "class-variance-authority";
import type React from "react";
import { useCallback, useMemo } from "react";

import { Button } from "@agh/ui/components/button";
import { ButtonGroup, ButtonGroupText } from "@agh/ui/components/button-group";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@agh/ui/components/dropdown-menu";
import { Input } from "@agh/ui/components/input";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
  InputGroupText,
} from "@agh/ui/components/input-group";
import { Kbd } from "@agh/ui/components/kbd";
import { ScrollArea } from "@agh/ui/components/scroll-area";
import { Tooltip, TooltipContent, TooltipTrigger } from "@agh/ui/components/tooltip";
import { cn } from "@agh/ui/lib/utils";
import { AlertCircleIcon, CheckIcon, XIcon } from "lucide-react";

import { createFilter, createFilterGroup, getFieldsMap } from "./hooks/filter-helpers";
import {
  DEFAULT_I18N,
  FilterContext,
  type FilterContextValue,
  type FilterI18nConfig,
  useFilterContext,
} from "./hooks/use-filter-context";
import { useFilterInput } from "./hooks/use-filter-input";
import { useFilterSubmenuContent } from "./hooks/use-filter-submenu-content";
import { useFilters, type FiltersMenuAction } from "./hooks/use-filters";
import { useSelectOptionsPopover } from "./hooks/use-select-options-popover";

export { createFilter, createFilterGroup, DEFAULT_I18N };
export type { FilterI18nConfig };

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

function FilterInput<T = unknown>({
  field,
  focusOnMount,
  onBlur,
  onKeyDown,
  className,
  ...props
}: React.InputHTMLAttributes<HTMLInputElement> & {
  className?: string;
  field?: FilterFieldConfig<T>;
  focusOnMount?: boolean;
}) {
  const {
    context,
    focusInputOnMount,
    handleKeyDown,
    isValid,
    validateFilterInputOnBlur,
    validationMessage,
  } = useFilterInput({
    field,
    focusOnMount,
    onBlur,
    onKeyDown,
    pattern: props.pattern,
  });

  return (
    <InputGroup
      className={cn(
        "w-36",
        context.size == "sm" && "h-7!",
        context.size == "default" && "h-8!",
        context.size == "lg" && "h-9!",
        className
      )}
    >
      {field?.prefix && (
        <InputGroupAddon>
          <InputGroupText>{field.prefix}</InputGroupText>
        </InputGroupAddon>
      )}
      <InputGroupInput
        ref={focusInputOnMount}
        aria-invalid={!isValid}
        aria-describedby={
          !isValid && validationMessage ? `${field?.key || "input"}-error` : undefined
        }
        onBlur={validateFilterInputOnBlur}
        onKeyDown={handleKeyDown}
        className={cn(
          context.size == "sm" && "h-7! text-form-label",
          context.size == "default" && "h-8!",
          context.size == "lg" && "h-9!"
        )}
        {...props}
      />
      {!isValid && validationMessage && (
        <InputGroupAddon align="inline-end">
          <Tooltip>
            <TooltipTrigger render={<InputGroupButton size="icon-xs" />}>
              <AlertCircleIcon className="text-destructive size-3" />
            </TooltipTrigger>
            <TooltipContent>
              <p>{validationMessage}</p>
            </TooltipContent>
          </Tooltip>
        </InputGroupAddon>
      )}

      {field?.suffix && (
        <InputGroupAddon align="inline-end">
          <InputGroupText>{field.suffix}</InputGroupText>
        </InputGroupAddon>
      )}
    </InputGroup>
  );
}

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

// Generic types for flexible filter system
export interface FilterOption<T = unknown> {
  value: T;
  label: string;
  icon?: React.ReactNode;
  metadata?: Record<string, unknown>;
  className?: string;
}

export interface FilterOperator {
  value: string;
  label: string;
  supportsMultiple?: boolean;
}

// Custom renderer props interface
export interface CustomRendererProps<T = unknown> {
  field: FilterFieldConfig<T>;
  values: T[];
  onChange: (values: T[]) => void;
  operator: string;
}

// Grouped field configuration interface
export interface FilterFieldGroup<T = unknown> {
  group?: string;
  fields: FilterFieldConfig<T>[];
}

// Union type for both flat and grouped field configurations
export type FilterFieldsConfig<T = unknown> = FilterFieldConfig<T>[] | FilterFieldGroup<T>[];

export interface FilterFieldConfig<T = unknown> {
  key?: string;
  label?: string;
  icon?: React.ReactNode;
  type?: "select" | "multiselect" | "text" | "custom" | "separator" | "toggle";
  // Group-level configuration
  group?: string;
  fields?: FilterFieldConfig<T>[];
  // Field-specific options
  options?: FilterOption<T>[];
  operators?: FilterOperator[];
  customRenderer?: (props: CustomRendererProps<T>) => React.ReactNode;
  customValueRenderer?: (values: T[], options: FilterOption<T>[]) => React.ReactNode;
  placeholder?: string;
  searchable?: boolean;
  maxSelections?: number;
  min?: number;
  max?: number;
  step?: number;
  prefix?: string | React.ReactNode;
  suffix?: string | React.ReactNode;
  pattern?: string;
  validation?: (value: unknown) => boolean | { valid: boolean; message?: string };
  allowCustomValues?: boolean;
  className?: string;
  menuPopupClassName?: string;
  // Grouping options (legacy support)
  groupLabel?: string;
  // Boolean field options
  onLabel?: string;
  offLabel?: string;
  // Input event handlers
  onInputChange?: (e: React.ChangeEvent<HTMLInputElement>) => void;
  // Default operator to use when creating a filter for this field
  defaultOperator?: string;
  // Controlled values support for this field
  value?: T[];
  onValueChange?: (values: T[]) => void;
}

// Helper function to create operators from i18n config
const createOperatorsFromI18n = (i18n: FilterI18nConfig): Record<string, FilterOperator[]> => ({
  select: [
    { value: "is", label: i18n.operators.is },
    { value: "is_not", label: i18n.operators.isNot },
    { value: "empty", label: i18n.operators.empty },
    { value: "not_empty", label: i18n.operators.notEmpty },
  ],
  multiselect: [
    { value: "is_any_of", label: i18n.operators.isAnyOf },
    { value: "is_not_any_of", label: i18n.operators.isNotAnyOf },
    { value: "includes_all", label: i18n.operators.includesAll },
    { value: "excludes_all", label: i18n.operators.excludesAll },
    { value: "empty", label: i18n.operators.empty },
    { value: "not_empty", label: i18n.operators.notEmpty },
  ],
  text: [
    { value: "contains", label: i18n.operators.contains },
    { value: "not_contains", label: i18n.operators.notContains },
    { value: "starts_with", label: i18n.operators.startsWith },
    { value: "ends_with", label: i18n.operators.endsWith },
    { value: "is", label: i18n.operators.isExactly },
    { value: "empty", label: i18n.operators.empty },
    { value: "not_empty", label: i18n.operators.notEmpty },
  ],
  custom: [
    { value: "is", label: i18n.operators.is },
    { value: "after", label: i18n.operators.after },
    { value: "is", label: i18n.operators.is },
    { value: "between", label: i18n.operators.between },
    { value: "empty", label: i18n.operators.empty },
    { value: "not_empty", label: i18n.operators.notEmpty },
  ],
});

// Default operators for different field types (using default i18n)
export const DEFAULT_OPERATORS: Record<string, FilterOperator[]> =
  createOperatorsFromI18n(DEFAULT_I18N);

// Helper function to get operators for a field
const getOperatorsForField = <T = unknown,>(
  field: FilterFieldConfig<T>,
  values: T[],
  i18n: FilterI18nConfig
): FilterOperator[] => {
  if (field.operators) return field.operators;

  const operators = createOperatorsFromI18n(i18n);

  // Determine field type for operator selection
  let fieldType = field.type || "select";

  // If it's a select field but has multiple values, treat as multiselect
  if (fieldType === "select" && values.length > 1) {
    fieldType = "multiselect";
  }

  // If it's a multiselect field or has multiselect operators, use multiselect operators
  if (fieldType === "multiselect" || field.type === "multiselect") {
    return operators.multiselect;
  }

  return operators[fieldType] || operators.select;
};

interface FilterOperatorDropdownProps<T = unknown> {
  field: FilterFieldConfig<T>;
  operator: string;
  values: T[];
  onChange: (operator: string) => void;
}

function FilterOperatorDropdown<T = unknown>({
  field,
  operator,
  values,
  onChange,
}: FilterOperatorDropdownProps<T>) {
  const context = useFilterContext();
  const operators = getOperatorsForField(field, values, context.i18n);

  // Find the operator label, with fallback to formatted operator name
  const operatorLabel =
    operators.find(op => op.value === operator)?.label ||
    context.i18n.helpers.formatOperator(operator);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            variant="outline"
            size={context.size}
            className="text-muted-foreground hover:text-foreground"
          >
            {operatorLabel}
          </Button>
        }
      />
      <DropdownMenuContent align="start" className="w-fit min-w-fit">
        {operators.map(op => (
          <DropdownMenuItem
            key={op.value}
            onClick={() => onChange(op.value)}
            className={cn(
              "data-highlighted:bg-accent data-highlighted:text-accent-foreground flex items-center justify-between"
            )}
          >
            <span>{op.label}</span>
            <CheckIcon
              className={cn(
                "text-primary ms-auto",
                op.value === operator ? "opacity-100" : "opacity-0"
              )}
            />
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

interface FilterValueSelectorProps<T = unknown> {
  field: FilterFieldConfig<T>;
  values: T[];
  onChange: (values: T[]) => void;
  operator: string;
  focusOnMount?: boolean;
}

interface SelectOptionsPopoverProps<T = unknown> {
  field: FilterFieldConfig<T>;
  values: T[];
  onChange: (values: T[]) => void;
  onClose?: () => void;
  inline?: boolean;
}

interface SelectOptionsMenuContentProps<T = unknown> {
  field: FilterFieldConfig<T>;
  context: FilterContextValue;
  baseId: string;
  open: boolean;
  searchInput: string;
  searchInputRef: React.RefObject<HTMLInputElement | null>;
  focusSearchInput: (node: HTMLInputElement | null) => void;
  highlightedIndex: number;
  selectedOptions: FilterOption<T>[];
  filteredSelectedOptions: FilterOption<T>[];
  filteredUnselectedOptions: FilterOption<T>[];
  allFilteredOptions: FilterOption<T>[];
  onSearchInputChange: (value: string) => void;
  onHighlightOption: (index: number) => void;
  onRequestClose: () => void;
  onToggleOption: (option: FilterOption<T>) => void;
}

function SelectOptionsMenuContent<T = unknown>({
  field,
  context,
  baseId,
  open,
  searchInput,
  searchInputRef,
  focusSearchInput,
  highlightedIndex,
  selectedOptions,
  filteredSelectedOptions,
  filteredUnselectedOptions,
  allFilteredOptions,
  onSearchInputChange,
  onHighlightOption,
  onRequestClose,
  onToggleOption,
}: SelectOptionsMenuContentProps<T>) {
  const moveHighlight = (nextIndex: number) => {
    if (allFilteredOptions.length > 0) {
      onHighlightOption(nextIndex);
    }
  };

  return (
    <>
      {field.searchable !== false && (
        <>
          <Input
            ref={focusSearchInput}
            role="combobox"
            aria-autocomplete="list"
            aria-expanded={true}
            aria-haspopup="listbox"
            aria-controls={`${baseId}-listbox`}
            aria-activedescendant={
              highlightedIndex >= 0 ? `${baseId}-item-${highlightedIndex}` : undefined
            }
            placeholder={context.i18n.placeholders.searchField(field.label || "")}
            className={cn(
              "border-input h-8 rounded-none border-0 bg-transparent! px-2 shadow-none",
              "focus-visible:border-border focus-visible:ring-0 focus-visible:ring-offset-0",
              open && "placeholder:text-foreground"
            )}
            value={searchInput}
            onChange={event => onSearchInputChange(event.target.value)}
            onBlur={() => open && searchInputRef.current?.focus()}
            onClick={event => event.stopPropagation()}
            onKeyDown={event => {
              if (event.key === "ArrowDown") {
                event.preventDefault();
                moveHighlight(
                  highlightedIndex < allFilteredOptions.length - 1 ? highlightedIndex + 1 : 0
                );
              } else if (event.key === "ArrowUp") {
                event.preventDefault();
                moveHighlight(
                  highlightedIndex > 0 ? highlightedIndex - 1 : allFilteredOptions.length - 1
                );
              } else if (event.key === "ArrowLeft") {
                event.preventDefault();
                onRequestClose();
              } else if (event.key === "Enter" && highlightedIndex >= 0) {
                event.preventDefault();
                const option = allFilteredOptions[highlightedIndex];
                if (option) {
                  onToggleOption(option);
                }
              }
              event.stopPropagation();
            }}
          />
          <DropdownMenuSeparator />
        </>
      )}
      <div className="relative flex max-h-full">
        <div
          className="flex max-h-[min(var(--available-height),24rem)] w-full scroll-pt-2 scroll-pb-2 flex-col overscroll-contain"
          role="listbox"
          id={`${baseId}-listbox`}
        >
          <ScrollArea className="size-full min-h-0 **:data-[slot=scroll-area-scrollbar]:m-0 **:data-[slot=scroll-area-viewport]:h-full **:data-[slot=scroll-area-viewport]:overscroll-contain">
            {allFilteredOptions.length === 0 && (
              <div className="text-muted-foreground py-2 text-center text-small-body">
                {context.i18n.noResultsFound}
              </div>
            )}

            {filteredSelectedOptions.length > 0 && (
              <DropdownMenuGroup className="px-1">
                {filteredSelectedOptions.map((option, index) => {
                  const isHighlighted = highlightedIndex === index;
                  const itemId = `${baseId}-item-${index}`;

                  return (
                    <DropdownMenuCheckboxItem
                      key={String(option.value)}
                      id={itemId}
                      role="option"
                      aria-selected={isHighlighted}
                      data-highlighted={isHighlighted || undefined}
                      onMouseEnter={() => onHighlightOption(index)}
                      checked={true}
                      className={cn(
                        "data-highlighted:bg-accent data-highlighted:text-accent-foreground",
                        option.className
                      )}
                      onSelect={event => {
                        if (field.type === "multiselect" || selectedOptions.length > 1) {
                          event.preventDefault();
                        }
                      }}
                      onCheckedChange={() => onToggleOption(option)}
                    >
                      {option.icon}
                      <span className="truncate">{option.label}</span>
                    </DropdownMenuCheckboxItem>
                  );
                })}
              </DropdownMenuGroup>
            )}

            {filteredSelectedOptions.length > 0 && filteredUnselectedOptions.length > 0 && (
              <DropdownMenuSeparator className="mx-0" />
            )}

            {filteredUnselectedOptions.length > 0 && (
              <DropdownMenuGroup className="px-1">
                {filteredUnselectedOptions.map((option, index) => {
                  const overallIndex = index + filteredSelectedOptions.length;
                  const isHighlighted = highlightedIndex === overallIndex;
                  const itemId = `${baseId}-item-${overallIndex}`;

                  return (
                    <DropdownMenuCheckboxItem
                      key={String(option.value)}
                      id={itemId}
                      role="option"
                      aria-selected={isHighlighted}
                      data-highlighted={isHighlighted || undefined}
                      onMouseEnter={() => onHighlightOption(overallIndex)}
                      checked={false}
                      className={cn(
                        "data-highlighted:bg-accent data-highlighted:text-accent-foreground",
                        option.className
                      )}
                      onSelect={event => {
                        if (field.type === "multiselect" || selectedOptions.length > 1) {
                          event.preventDefault();
                        }
                      }}
                      onCheckedChange={() => onToggleOption(option)}
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
    </>
  );
}

function SelectOptionsPopover<T = unknown>({
  field,
  values,
  onChange,
  onClose,
  inline = false,
}: SelectOptionsPopoverProps<T>) {
  const {
    allFilteredOptions,
    baseId,
    context,
    filteredSelectedOptions,
    filteredUnselectedOptions,
    focusSearchInput,
    handleClose,
    handleOpenChange,
    handleSearchInputChange,
    highlightOption,
    highlightedIndex,
    inputRef,
    open,
    searchInput,
    selectedOptions,
    toggleOption,
  } = useSelectOptionsPopover({ field, values, onChange, onClose });

  const menuContent = (
    <SelectOptionsMenuContent
      field={field}
      context={context}
      baseId={baseId}
      open={open}
      searchInput={searchInput}
      searchInputRef={inputRef}
      focusSearchInput={focusSearchInput}
      highlightedIndex={highlightedIndex}
      selectedOptions={selectedOptions}
      filteredSelectedOptions={filteredSelectedOptions}
      filteredUnselectedOptions={filteredUnselectedOptions}
      allFilteredOptions={allFilteredOptions}
      onSearchInputChange={handleSearchInputChange}
      onHighlightOption={highlightOption}
      onRequestClose={handleClose}
      onToggleOption={toggleOption}
    />
  );

  if (inline) {
    return <div className="w-full">{menuContent}</div>;
  }

  return (
    <DropdownMenu open={open} onOpenChange={handleOpenChange}>
      <DropdownMenuTrigger
        render={
          <Button variant="outline" size={context.size}>
            <div className="flex items-center gap-1.5">
              {field.customValueRenderer ? (
                field.customValueRenderer(values, field.options || [])
              ) : (
                <>
                  {selectedOptions.length > 0 && (
                    <div className="flex items-center gap-1.5">
                      {selectedOptions.slice(0, 3).map(option => (
                        <div key={String(option.value)}>{option.icon}</div>
                      ))}
                    </div>
                  )}
                  {selectedOptions.length === 1
                    ? selectedOptions[0].label
                    : selectedOptions.length > 1
                      ? `${selectedOptions.length} ${context.i18n.selectedCount}`
                      : context.i18n.select}
                </>
              )}
            </div>
          </Button>
        }
      />
      <DropdownMenuContent
        align="start"
        className={cn("w-filters-menu-default px-0", field.className)}
      >
        {menuContent}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function FilterValueSelector<T = unknown>({
  field,
  values,
  onChange,
  operator,
  focusOnMount,
}: FilterValueSelectorProps<T>) {
  if (operator === "empty" || operator === "not_empty") {
    return null;
  }

  if (field.type === "toggle") {
    return null;
  }

  if (field.customRenderer) {
    return (
      <ButtonGroupText className="hover:bg-accent aria-expanded:bg-accent bg-background dark:bg-input/30 text-start whitespace-nowrap outline-hidden">
        {field.customRenderer({ field, values, onChange, operator })}
      </ButtonGroupText>
    );
  }

  if (field.type === "text") {
    return (
      <FilterInput
        type="text"
        value={(values[0] as string) || ""}
        onChange={e => onChange([e.target.value] as T[])}
        placeholder={field.placeholder}
        pattern={field.pattern}
        field={field}
        className={cn("w-36", field.className)}
        focusOnMount={focusOnMount}
      />
    );
  }

  if (field.type === "select" || field.type === "multiselect") {
    return <SelectOptionsPopover field={field} values={values} onChange={onChange} />;
  }

  return <SelectOptionsPopover field={field} values={values} onChange={onChange} />;
}
export interface Filter<T = unknown> {
  id: string;
  field: string;
  operator: string;
  values: T[];
}

export interface FilterGroup<T = unknown> {
  id: string;
  label?: string;
  filters: Filter<T>[];
  fields: FilterFieldConfig<T>[];
}

interface FiltersContentProps<T = unknown> {
  filters: Filter<T>[];
  fields: FilterFieldsConfig<T>;
  onChange: (filters: Filter<T>[]) => void;
  focusFilterId?: string | null;
}

export const FiltersContent = <T = unknown,>({
  filters,
  fields,
  onChange,
  focusFilterId = null,
}: FiltersContentProps<T>) => {
  const context = useFilterContext();
  const fieldsMap = useMemo(() => getFieldsMap(fields), [fields]);

  const updateFilter = useCallback(
    (filterId: string, updates: Partial<Filter<T>>) => {
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
    },
    [filters, onChange]
  );

  const removeFilter = useCallback(
    (filterId: string) => {
      onChange(filters.filter(filter => filter.id !== filterId));
    },
    [filters, onChange]
  );

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
                  const isSelected = currentValues.includes(option.value);
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

function FiltersMenuFieldList<T = unknown>({
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

function FiltersMenuSearchInput<T = unknown>({
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

  return (
    <FilterContext.Provider
      value={{
        variant,
        size,
        radius,
        i18n: mergedI18n,
        className,
        trigger,
        allowMultiple,
      }}
    >
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
