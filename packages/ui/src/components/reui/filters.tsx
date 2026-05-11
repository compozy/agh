"use client";

import { useRender } from "@base-ui/react/use-render";
import { cva } from "class-variance-authority";
import type React from "react";
import {
  createContext,
  use,
  useCallback,
  useEffect,
  useId,
  useMemo,
  useReducer,
  useRef,
  useState,
} from "react";

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

// i18n Configuration Interface
export interface FilterI18nConfig {
  // UI Labels
  addFilter: string;
  searchFields: string;
  noFieldsFound: string;
  noResultsFound: string;
  select: string;
  true: string;
  false: string;
  min: string;
  max: string;
  to: string;
  typeAndPressEnter: string;
  selected: string;
  selectedCount: string;
  percent: string;
  defaultCurrency: string;
  defaultColor: string;
  addFilterTitle: string;

  // Operators
  operators: {
    is: string;
    isNot: string;
    isAnyOf: string;
    isNotAnyOf: string;
    includesAll: string;
    excludesAll: string;
    before: string;
    after: string;
    between: string;
    notBetween: string;
    contains: string;
    notContains: string;
    startsWith: string;
    endsWith: string;
    isExactly: string;
    equals: string;
    notEquals: string;
    greaterThan: string;
    lessThan: string;
    overlaps: string;
    includes: string;
    excludes: string;
    includesAllOf: string;
    includesAnyOf: string;
    empty: string;
    notEmpty: string;
  };

  // Placeholders
  placeholders: {
    enterField: (fieldType: string) => string;
    selectField: string;
    searchField: (fieldName: string) => string;
    enterKey: string;
    enterValue: string;
  };

  // Helper functions
  helpers: {
    formatOperator: (operator: string) => string;
  };

  // Validation
  validation: {
    invalidEmail: string;
    invalidUrl: string;
    invalidTel: string;
    invalid: string;
  };
}

// Default English i18n configuration
export const DEFAULT_I18N: FilterI18nConfig = {
  // UI Labels
  addFilter: "Filter",
  searchFields: "Filter...",
  noFieldsFound: "No filters found.",
  noResultsFound: "No results found.",
  select: "Select...",
  true: "True",
  false: "False",
  min: "Min",
  max: "Max",
  to: "to",
  typeAndPressEnter: "Type and press Enter to add tag",
  selected: "selected",
  selectedCount: "selected",
  percent: "%",
  defaultCurrency: "$",
  defaultColor: "#000000",
  addFilterTitle: "Add filter",

  // Operators
  operators: {
    is: "is",
    isNot: "is not",
    isAnyOf: "is any of",
    isNotAnyOf: "is not any of",
    includesAll: "includes all",
    excludesAll: "excludes all",
    before: "before",
    after: "after",
    between: "between",
    notBetween: "not between",
    contains: "contains",
    notContains: "does not contain",
    startsWith: "starts with",
    endsWith: "ends with",
    isExactly: "is exactly",
    equals: "equals",
    notEquals: "not equals",
    greaterThan: "greater than",
    lessThan: "less than",
    overlaps: "overlaps",
    includes: "includes",
    excludes: "excludes",
    includesAllOf: "includes all of",
    includesAnyOf: "includes any of",
    empty: "is empty",
    notEmpty: "is not empty",
  },

  // Placeholders
  placeholders: {
    enterField: (fieldType: string) => `Enter ${fieldType}...`,
    selectField: "Select...",
    searchField: (fieldName: string) => `Search ${fieldName.toLowerCase()}...`,
    enterKey: "Enter key...",
    enterValue: "Enter value...",
  },

  // Helper functions
  helpers: {
    formatOperator: (operator: string) => operator.replace(/_/g, " "),
  },

  // Validation
  validation: {
    invalidEmail: "Invalid email format",
    invalidUrl: "Invalid URL format",
    invalidTel: "Invalid phone format",
    invalid: "Invalid input format",
  },
};

// Context for all Filter component props
interface FilterContextValue {
  variant: "solid" | "default";
  size: "sm" | "default" | "lg";
  radius: "default" | "full";
  i18n: FilterI18nConfig;
  className?: string;
  showSearchInput?: boolean;
  trigger?: React.ReactNode;
  allowMultiple?: boolean;
}

const FilterContext = createContext<FilterContextValue>({
  variant: "default",
  size: "default",
  radius: "default",
  i18n: DEFAULT_I18N,
  className: undefined,
  showSearchInput: true,
  trigger: undefined,
  allowMultiple: true,
});

const useFilterContext = () => use(FilterContext);

function scheduleFilterDomSync(callback: () => void) {
  if (typeof window === "undefined") return;
  window.requestAnimationFrame(callback);
}

function scrollFilterOptionIntoView(baseId: string, index: number) {
  if (index < 0) return;
  scheduleFilterDomSync(() => {
    document.getElementById(`${baseId}-item-${index}`)?.scrollIntoView({ block: "nearest" });
  });
}

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
  const context = useFilterContext();
  const [isValid, setIsValid] = useState(true);
  const [validationMessage, setValidationMessage] = useState("");

  const focusInputOnMount = useCallback(
    (node: HTMLInputElement | null) => {
      if (node && focusOnMount) {
        scheduleFilterDomSync(() => node.focus());
      }
    },
    [focusOnMount]
  );

  // Validation function to check if input matches pattern
  const validateInput = (value: string, pattern?: string): boolean => {
    if (!pattern || !value) return true;
    const regex = new RegExp(pattern);
    return regex.test(value);
  };

  // Get validation message for field type
  const getValidationMessage = (): string => {
    return context.i18n.validation.invalid;
  };

  // Handle blur event - validate when user leaves input
  const validateFilterInputOnBlur = (e: React.FocusEvent<HTMLInputElement>) => {
    const value = e.target.value;
    const pattern = field?.pattern || props.pattern;

    // Only validate if there's a value and (pattern or validation function)
    if (value && (pattern || field?.validation)) {
      let valid = true;
      let customMessage = "";

      // If there's a custom validation function, use it
      if (field?.validation) {
        const result = field.validation(value);
        // Handle both boolean and object return types
        if (typeof result === "boolean") {
          valid = result;
        } else {
          valid = result.valid;
          customMessage = result.message || "";
        }
      } else if (pattern) {
        // Use pattern validation
        valid = validateInput(value, pattern);
      }

      setIsValid(valid);
      setValidationMessage(valid ? "" : customMessage || getValidationMessage());
    } else {
      // Reset validation state for empty values or no validation
      setIsValid(true);
      setValidationMessage("");
    }

    // Call the original onBlur if provided
    onBlur?.(e);
  };

  // Handle keydown event - hide validation error when user starts typing
  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    // Hide validation error when user starts typing (any key except special keys)
    if (
      !isValid &&
      !["Tab", "Escape", "Enter", "ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight"].includes(e.key)
    ) {
      setIsValid(true);
      setValidationMessage("");
    }

    // Call the original onKeyDown if provided
    onKeyDown?.(e);
  };

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
          context.size == "sm" && "h-7! text-xs",
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
              <p className="text-sm">{validationMessage}</p>
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

// Helper functions to handle both flat and grouped field configurations
const isFieldGroup = <T = unknown,>(
  item: FilterFieldConfig<T> | FilterFieldGroup<T>
): item is FilterFieldGroup<T> => {
  return "fields" in item && Array.isArray(item.fields);
};

// Helper function to check if a FilterFieldConfig is a group-level configuration
const isGroupLevelField = <T = unknown,>(field: FilterFieldConfig<T>): boolean => {
  return Boolean(field.group && field.fields);
};

const flattenFields = <T = unknown,>(fields: FilterFieldsConfig<T>): FilterFieldConfig<T>[] => {
  return fields.reduce<FilterFieldConfig<T>[]>((acc, item) => {
    if (isFieldGroup(item)) {
      return [...acc, ...item.fields];
    }
    // Handle group-level fields (new structure)
    if (isGroupLevelField(item)) {
      return [...acc, ...item.fields!];
    }
    return [...acc, item];
  }, []);
};

const getFieldsMap = <T = unknown,>(
  fields: FilterFieldsConfig<T>
): Record<string, FilterFieldConfig<T>> => {
  const flatFields = flattenFields(fields);
  return flatFields.reduce(
    (acc, field) => {
      // Only add fields that have a key (skip group-level configurations)
      if (field.key) {
        acc[field.key] = field;
      }
      return acc;
    },
    {} as Record<string, FilterFieldConfig<T>>
  );
};

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
              "border-input h-8 rounded-none border-0 bg-transparent! px-2 text-sm shadow-none",
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
              <div className="text-muted-foreground py-2 text-center text-sm">
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
  const [open, setOpen] = useState(false);
  const [searchInput, setSearchInput] = useState("");
  const [highlightedIndex, setHighlightedIndex] = useState(-1);
  const inputRef = useRef<HTMLInputElement>(null);
  const context = useFilterContext();
  const baseId = useId();

  const focusSearchInput = useCallback(
    (node: HTMLInputElement | null) => {
      inputRef.current = node;
      if (node && open) {
        scheduleFilterDomSync(() => node.focus());
      }
    },
    [open]
  );

  const highlightOption = (index: number) => {
    setHighlightedIndex(index);
    if (open) {
      scrollFilterOptionIntoView(baseId, index);
    }
  };

  const isMultiSelect = field.type === "multiselect" || values.length > 1;
  const effectiveValues = (field.value !== undefined ? (field.value as T[]) : values) || [];

  const selectedOptions = field.options?.filter(opt => effectiveValues.includes(opt.value)) || [];
  const unselectedOptions =
    field.options?.filter(opt => !effectiveValues.includes(opt.value)) || [];

  // Filter options based on search input
  const filteredSelectedOptions = selectedOptions; // Keep all selected visible
  const filteredUnselectedOptions = unselectedOptions.filter(opt =>
    opt.label.toLowerCase().includes(searchInput.toLowerCase())
  );

  const allFilteredOptions = useMemo(
    () => [...filteredSelectedOptions, ...filteredUnselectedOptions],
    [filteredSelectedOptions, filteredUnselectedOptions]
  );

  const handleClose = () => {
    setOpen(false);
    setSearchInput("");
    setHighlightedIndex(-1);
    onClose?.();
  };

  const toggleOption = (option: FilterOption<T>) => {
    const isSelected = effectiveValues.includes(option.value as T);
    const next = isSelected
      ? (effectiveValues.filter(value => value !== option.value) as T[])
      : isMultiSelect
        ? ([...effectiveValues, option.value] as T[])
        : ([option.value] as T[]);

    if (!isSelected && isMultiSelect && field.maxSelections && next.length > field.maxSelections) {
      return;
    }

    if (field.onValueChange) {
      field.onValueChange(next);
    } else {
      onChange(next);
    }
    if (!isMultiSelect) handleClose();
  };

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
      onSearchInputChange={value => {
        setSearchInput(value);
        setHighlightedIndex(-1);
      }}
      onHighlightOption={highlightOption}
      onRequestClose={handleClose}
      onToggleOption={toggleOption}
    />
  );

  if (inline) {
    return <div className="w-full">{menuContent}</div>;
  }

  return (
    <DropdownMenu
      open={open}
      onOpenChange={open => {
        setOpen(open);
        if (!open) {
          setSearchInput("");
          setHighlightedIndex(-1);
        } else {
          setHighlightedIndex(-1);
        }
      }}
    >
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
      <DropdownMenuContent align="start" className={cn("w-[200px] px-0", field.className)}>
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

interface FiltersMenuState {
  addFilterOpen: boolean;
  menuSearchInput: string;
  activeMenu: string;
  openSubMenu: string | null;
  highlightedIndex: number;
  lastAddedFilterId: string | null;
  sessionFilterIds: Record<string, string>;
}

type FiltersMenuAction =
  | Partial<FiltersMenuState>
  | ((state: FiltersMenuState) => Partial<FiltersMenuState>);

const FILTERS_MENU_INITIAL_STATE: FiltersMenuState = {
  addFilterOpen: false,
  menuSearchInput: "",
  activeMenu: "root",
  openSubMenu: null,
  highlightedIndex: -1,
  lastAddedFilterId: null,
  sessionFilterIds: {},
};

function filtersMenuReducer(state: FiltersMenuState, action: FiltersMenuAction): FiltersMenuState {
  const patch = typeof action === "function" ? action(state) : action;
  return { ...state, ...patch };
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
  const [searchInput, setSearchInput] = useState("");
  const [highlightedIndex, setHighlightedIndex] = useState(-1);
  const inputRef = useRef<HTMLInputElement>(null);
  const baseId = useId();

  const focusSubmenuSearchInput = useCallback(
    (node: HTMLInputElement | null) => {
      inputRef.current = node;
      if (node && isActive && field.searchable !== false) {
        scheduleFilterDomSync(() => node.focus());
      }
    },
    [field.searchable, isActive]
  );

  const focusSubmenuListbox = useCallback(
    (node: HTMLDivElement | null) => {
      if (node && isActive && field.searchable === false) {
        scheduleFilterDomSync(() => node.focus());
      }
    },
    [field.searchable, isActive]
  );

  const highlightSubmenuOption = (index: number) => {
    setHighlightedIndex(index);
    if (isActive) {
      scrollFilterOptionIntoView(baseId, index);
    }
  };

  const filteredOptions = useMemo(() => {
    return (
      field.options?.filter(option => {
        const isSelected = currentValues.includes(option.value);
        if (isSelected) return true;
        if (!searchInput) return true;
        return option.label.toLowerCase().includes(searchInput.toLowerCase());
      }) || []
    );
  }, [field.options, searchInput, currentValues]);

  const activeHighlightedIndex =
    highlightedIndex >= 0 ? highlightedIndex : isActive && filteredOptions.length > 0 ? 0 : -1;

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
              "h-8 rounded-none border-0 bg-transparent! px-2 text-sm shadow-none",
              "focus-visible:border-border focus-visible:ring-0 focus-visible:ring-offset-0",
              isActive && "placeholder:text-foreground"
            )}
            value={searchInput}
            onBlur={() => isActive && inputRef.current?.focus()}
            onChange={e => {
              setSearchInput(e.target.value);
              setHighlightedIndex(-1);
            }}
            onFocus={() => onActive?.()}
            onMouseEnter={e => {
              onActive?.();
              e.stopPropagation();
            }}
            onClick={e => e.stopPropagation()}
            onKeyDown={e => {
              if (e.key === "ArrowDown") {
                e.preventDefault();
                if (filteredOptions.length > 0) {
                  highlightSubmenuOption(
                    activeHighlightedIndex < filteredOptions.length - 1
                      ? activeHighlightedIndex + 1
                      : 0
                  );
                }
              } else if (e.key === "ArrowUp") {
                e.preventDefault();
                if (filteredOptions.length > 0) {
                  highlightSubmenuOption(
                    activeHighlightedIndex > 0
                      ? activeHighlightedIndex - 1
                      : filteredOptions.length - 1
                  );
                }
              } else if (e.key === "ArrowLeft") {
                e.preventDefault();
                onBack?.();
              } else if (e.key === "Enter" && activeHighlightedIndex >= 0) {
                e.preventDefault();
                const option = filteredOptions[activeHighlightedIndex];
                if (option) {
                  onToggle(option.value as T, currentValues.includes(option.value));
                  if (!isMultiSelect) {
                    onBack?.();
                  }
                }
              } else if (e.key === "Escape") {
                e.preventDefault();
                onClose?.();
              }
              e.stopPropagation();
            }}
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
          onKeyDown={e => {
            if (field.searchable === false) {
              if (e.key === "ArrowDown") {
                e.preventDefault();
                if (filteredOptions.length > 0) {
                  highlightSubmenuOption(
                    activeHighlightedIndex < filteredOptions.length - 1
                      ? activeHighlightedIndex + 1
                      : 0
                  );
                }
              } else if (e.key === "ArrowUp") {
                e.preventDefault();
                if (filteredOptions.length > 0) {
                  highlightSubmenuOption(
                    activeHighlightedIndex > 0
                      ? activeHighlightedIndex - 1
                      : filteredOptions.length - 1
                  );
                }
              } else if (e.key === "ArrowLeft") {
                e.preventDefault();
                onBack?.();
              } else if (e.key === "Enter" && activeHighlightedIndex >= 0) {
                e.preventDefault();
                const option = filteredOptions[activeHighlightedIndex];
                if (option) {
                  onToggle(option.value as T, currentValues.includes(option.value));
                  if (!isMultiSelect) {
                    onBack?.();
                  }
                }
              } else if (e.key === "Escape") {
                e.preventDefault();
                onClose?.();
              }
              e.stopPropagation();
            }
          }}
        >
          <ScrollArea className="size-full min-h-0 **:data-[slot=scroll-area-scrollbar]:m-0 **:data-[slot=scroll-area-viewport]:h-full **:data-[slot=scroll-area-viewport]:overscroll-contain">
            {filteredOptions.length === 0 ? (
              <div className="text-muted-foreground py-2 text-center text-sm">
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
      <div className="text-muted-foreground py-2 text-center text-sm">{i18n.noFieldsFound}</div>
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
          <DropdownMenuSubContent className="w-[200px]" side="right">
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
            "h-8 rounded-none border-0 bg-transparent! px-2 text-sm shadow-none",
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
  const [menuState, setMenuState] = useReducer(filtersMenuReducer, FILTERS_MENU_INITIAL_STATE);
  const {
    addFilterOpen,
    menuSearchInput,
    activeMenu,
    openSubMenu,
    highlightedIndex,
    lastAddedFilterId,
    sessionFilterIds,
  } = menuState;
  const rootInputRef = useRef<HTMLInputElement>(null);
  const lastAddedFilterTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const rootId = useId();

  useEffect(() => {
    if (!enableShortcut) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (
        e.key.toLowerCase() === shortcutKey.toLowerCase() &&
        !addFilterOpen &&
        !(
          document.activeElement instanceof HTMLInputElement ||
          document.activeElement instanceof HTMLTextAreaElement
        )
      ) {
        e.preventDefault();
        setMenuState({ addFilterOpen: true });
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [enableShortcut, shortcutKey, addFilterOpen]);

  useEffect(() => {
    return () => {
      if (lastAddedFilterTimerRef.current) {
        clearTimeout(lastAddedFilterTimerRef.current);
      }
    };
  }, []);

  const focusRootInput = useCallback(
    (node: HTMLInputElement | null) => {
      rootInputRef.current = node;
      if (node && addFilterOpen && activeMenu === "root") {
        scheduleFilterDomSync(() => node.focus());
      }
    },
    [activeMenu, addFilterOpen]
  );

  const markLastAddedFilter = useCallback((filterId: string) => {
    if (lastAddedFilterTimerRef.current) {
      clearTimeout(lastAddedFilterTimerRef.current);
    }
    setMenuState({ lastAddedFilterId: filterId });
    lastAddedFilterTimerRef.current = setTimeout(() => {
      setMenuState({ lastAddedFilterId: null });
      lastAddedFilterTimerRef.current = null;
    }, 1000);
  }, []);

  const mergedI18n: FilterI18nConfig = {
    ...DEFAULT_I18N,
    ...i18n,
    operators: { ...DEFAULT_I18N.operators, ...i18n?.operators },
    placeholders: { ...DEFAULT_I18N.placeholders, ...i18n?.placeholders },
    validation: { ...DEFAULT_I18N.validation, ...i18n?.validation },
  };

  const fieldsMap = useMemo(() => getFieldsMap(fields), [fields]);

  const addFilter = useCallback(
    (fieldKey: string) => {
      const field = fieldsMap[fieldKey];
      if (field && field.key) {
        const defaultOperator =
          field.defaultOperator || (field.type === "multiselect" ? "is_any_of" : "is");
        const defaultValues: unknown[] =
          field.type === "text" ? [""] : field.type === "toggle" ? [true] : [];
        const newFilter = createFilter<T>(fieldKey, defaultOperator, defaultValues as T[]);
        markLastAddedFilter(newFilter.id);
        onChange([...filters, newFilter]);
        setMenuState({
          addFilterOpen: false,
          menuSearchInput: "",
          highlightedIndex: -1,
        });
      }
    },
    [fieldsMap, filters, markLastAddedFilter, onChange]
  );

  const selectableFields = useMemo(() => {
    const flatFields = flattenFields(fields);
    return flatFields.filter(field => {
      if (!field.key || field.type === "separator") return false;
      if (allowMultiple) return true;
      return !filters.some(filter => filter.field === field.key);
    });
  }, [fields, filters, allowMultiple]);

  const filteredFields = useMemo(() => {
    return selectableFields.filter(
      f => !menuSearchInput || f.label?.toLowerCase().includes(menuSearchInput.toLowerCase())
    );
  }, [selectableFields, menuSearchInput]);

  const rootHighlightedIndex =
    highlightedIndex >= 0 ? highlightedIndex : addFilterOpen && filteredFields.length > 0 ? 0 : -1;

  const highlightRootOption = (index: number) => {
    setMenuState({ highlightedIndex: index });
    if (addFilterOpen) {
      scrollFilterOptionIntoView(rootId, index);
    }
  };

  const triggerButton = useRender({
    render: trigger as React.ReactElement,
    defaultTagName: "button",
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
          <DropdownMenu
            open={addFilterOpen}
            onOpenChange={open => {
              setMenuState({ addFilterOpen: open });
              if (!open) {
                setMenuState({
                  menuSearchInput: "",
                  sessionFilterIds: {},
                  openSubMenu: null,
                  highlightedIndex: -1,
                });
              } else {
                setMenuState({ activeMenu: "root", highlightedIndex: -1 });
              }
            }}
          >
            <DropdownMenuTrigger render={triggerButton} />
            <DropdownMenuContent className={cn("w-[220px]", menuPopupClassName)} align="start">
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
                  onMouseEnter={() => setMenuState({ activeMenu: "root" })}
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

export const createFilter = <T = unknown,>(
  field: string,
  operator?: string,
  values: T[] = []
): Filter<T> => ({
  id: `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`,
  field,
  operator: operator || "is",
  values,
});

export const createFilterGroup = <T = unknown,>(
  id: string,
  label: string,
  fields: FilterFieldConfig<T>[],
  initialFilters: Filter<T>[] = []
): FilterGroup<T> => ({
  id,
  label,
  filters: initialFilters,
  fields,
});
