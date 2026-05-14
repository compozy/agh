import { useCallback, useId, useMemo, useRef, useState } from "react";

import type { FilterFieldConfig, FilterOption } from "../filters";
import {
  scheduleFilterDomSync,
  scrollFilterOptionIntoView,
  useFilterContext,
} from "./use-filter-context";

interface UseSelectOptionsPopoverOptions<T = unknown> {
  field: FilterFieldConfig<T>;
  values: T[];
  onChange: (values: T[]) => void;
  onClose?: () => void;
}

export function useSelectOptionsPopover<T = unknown>({
  field,
  values,
  onChange,
  onClose,
}: UseSelectOptionsPopoverOptions<T>) {
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

  const highlightOption = useCallback(
    (index: number) => {
      setHighlightedIndex(index);
      if (open) {
        scrollFilterOptionIntoView(baseId, index);
      }
    },
    [baseId, open]
  );

  const isMultiSelect = field.type === "multiselect" || values.length > 1;
  const effectiveValues = (field.value !== undefined ? (field.value as T[]) : values) || [];
  const selectedOptions =
    field.options?.filter(option => effectiveValues.includes(option.value)) || [];
  const unselectedOptions =
    field.options?.filter(option => !effectiveValues.includes(option.value)) || [];
  const filteredSelectedOptions = selectedOptions;
  const filteredUnselectedOptions = unselectedOptions.filter(option =>
    option.label.toLowerCase().includes(searchInput.toLowerCase())
  );

  const allFilteredOptions = useMemo(
    () => [...filteredSelectedOptions, ...filteredUnselectedOptions],
    [filteredSelectedOptions, filteredUnselectedOptions]
  );

  const handleClose = useCallback(() => {
    setOpen(false);
    setSearchInput("");
    setHighlightedIndex(-1);
    onClose?.();
  }, [onClose]);

  const handleOpenChange = useCallback((nextOpen: boolean) => {
    setOpen(nextOpen);
    setHighlightedIndex(-1);
    if (!nextOpen) {
      setSearchInput("");
    }
  }, []);

  const handleSearchInputChange = useCallback((value: string) => {
    setSearchInput(value);
    setHighlightedIndex(-1);
  }, []);

  const toggleOption = useCallback(
    (option: FilterOption<T>) => {
      const isSelected = effectiveValues.includes(option.value as T);
      const next = isSelected
        ? (effectiveValues.filter(value => value !== option.value) as T[])
        : isMultiSelect
          ? ([...effectiveValues, option.value] as T[])
          : ([option.value] as T[]);

      if (
        !isSelected &&
        isMultiSelect &&
        field.maxSelections &&
        next.length > field.maxSelections
      ) {
        return;
      }

      if (field.onValueChange) {
        field.onValueChange(next);
      } else {
        onChange(next);
      }
      if (!isMultiSelect) handleClose();
    },
    [effectiveValues, field, handleClose, isMultiSelect, onChange]
  );

  return {
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
    selectedOptions,
    searchInput,
    toggleOption,
  };
}
