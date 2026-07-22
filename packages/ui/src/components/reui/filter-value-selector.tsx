import type React from "react";
import { useFilterInput } from "./hooks/use-filter-input";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
  InputGroupText,
} from "@agh/ui/components/input-group";
import { cn } from "@agh/ui/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@agh/ui/components/tooltip";
import { AlertCircleIcon } from "lucide-react";
import { type FilterContextValue } from "./hooks/use-filter-context";
import { Input } from "@agh/ui/components/input";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@agh/ui/components/dropdown-menu";
import { ScrollArea } from "@agh/ui/components/scroll-area";
import { useSelectOptionsPopover } from "./hooks/use-select-options-popover";
import { Button } from "@agh/ui/components/button";
import { ButtonGroupText } from "@agh/ui/components/button-group";

import type { FilterFieldConfig, FilterOption } from "./filters-types";

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

export function FilterValueSelector<T = unknown>({
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
