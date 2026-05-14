import type React from "react";
import { useCallback, useId, useMemo, useRef, useState } from "react";

import type { FilterFieldConfig } from "../filters";
import { scheduleFilterDomSync, scrollFilterOptionIntoView } from "./use-filter-context";

interface UseFilterSubmenuContentOptions<T = unknown> {
  field: FilterFieldConfig<T>;
  currentValues: T[];
  isMultiSelect: boolean;
  isActive?: boolean;
  onBack?: () => void;
  onClose?: () => void;
  onToggle: (value: T, isSelected: boolean) => void;
}

export function useFilterSubmenuContent<T = unknown>({
  field,
  currentValues,
  isMultiSelect,
  isActive,
  onBack,
  onClose,
  onToggle,
}: UseFilterSubmenuContentOptions<T>) {
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

  const highlightSubmenuOption = useCallback(
    (index: number) => {
      setHighlightedIndex(index);
      if (isActive) {
        scrollFilterOptionIntoView(baseId, index);
      }
    },
    [baseId, isActive]
  );

  const filteredOptions = useMemo(() => {
    return (
      field.options?.filter(option => {
        const isSelected = currentValues.includes(option.value);
        if (isSelected) return true;
        if (!searchInput) return true;
        return option.label.toLowerCase().includes(searchInput.toLowerCase());
      }) || []
    );
  }, [currentValues, field.options, searchInput]);

  const activeHighlightedIndex =
    highlightedIndex >= 0 ? highlightedIndex : isActive && filteredOptions.length > 0 ? 0 : -1;

  const handleSearchInputChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    setSearchInput(event.target.value);
    setHighlightedIndex(-1);
  }, []);

  const selectHighlightedOption = useCallback(() => {
    const option = filteredOptions[activeHighlightedIndex];
    if (!option) return;

    onToggle(option.value as T, currentValues.includes(option.value));
    if (!isMultiSelect) {
      onBack?.();
    }
  }, [activeHighlightedIndex, currentValues, filteredOptions, isMultiSelect, onBack, onToggle]);

  const handleNavigationKeyDown = useCallback(
    (event: React.KeyboardEvent<HTMLElement>) => {
      if (event.key === "ArrowDown") {
        event.preventDefault();
        if (filteredOptions.length > 0) {
          highlightSubmenuOption(
            activeHighlightedIndex < filteredOptions.length - 1 ? activeHighlightedIndex + 1 : 0
          );
        }
      } else if (event.key === "ArrowUp") {
        event.preventDefault();
        if (filteredOptions.length > 0) {
          highlightSubmenuOption(
            activeHighlightedIndex > 0 ? activeHighlightedIndex - 1 : filteredOptions.length - 1
          );
        }
      } else if (event.key === "ArrowLeft") {
        event.preventDefault();
        onBack?.();
      } else if (event.key === "Enter" && activeHighlightedIndex >= 0) {
        event.preventDefault();
        selectHighlightedOption();
      } else if (event.key === "Escape") {
        event.preventDefault();
        onClose?.();
      }

      event.stopPropagation();
    },
    [
      activeHighlightedIndex,
      filteredOptions.length,
      highlightSubmenuOption,
      onBack,
      onClose,
      selectHighlightedOption,
    ]
  );

  const handleListboxKeyDown = useCallback(
    (event: React.KeyboardEvent<HTMLDivElement>) => {
      if (field.searchable === false) {
        handleNavigationKeyDown(event);
      }
    },
    [field.searchable, handleNavigationKeyDown]
  );

  return {
    activeHighlightedIndex,
    baseId,
    filteredOptions,
    focusSubmenuListbox,
    focusSubmenuSearchInput,
    handleListboxKeyDown,
    handleSearchInputChange,
    handleSearchInputKeyDown: handleNavigationKeyDown,
    highlightSubmenuOption,
    inputRef,
    searchInput,
  };
}
