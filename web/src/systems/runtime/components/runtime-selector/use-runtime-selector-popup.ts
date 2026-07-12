import { useId, type ComponentProps, type KeyboardEvent, type RefObject } from "react";

import type { Popover } from "@agh/ui";

import { noteCommandKFocus } from "./command-k-registry";
import { useCommandKOwnership } from "./use-command-k-ownership";
import type { TriggerFocus } from "./trigger";
import type { RuntimeProviderOption, RuntimeSelectorValue } from "./types";
import type { RuntimeSelectorController } from "./use-runtime-selector";

type PopoverOpenChange = NonNullable<ComponentProps<typeof Popover>["onOpenChange"]>;

export interface UseRuntimeSelectorPopupArgs {
  controller: RuntimeSelectorController;
  providers: RuntimeProviderOption[];
  value: RuntimeSelectorValue;
  disabled: boolean;
  triggerRef: RefObject<HTMLDivElement | null>;
  searchRef: RefObject<HTMLInputElement | null>;
  popupRef: RefObject<HTMLDivElement | null>;
}

/**
 * DOM/ARIA wiring for the runtime selector popup: stable ids for the combobox
 * `aria-controls`/`aria-activedescendant` relationship, the trigger/search/popup
 * refs, provider name/icon resolvers, and the popover open/close + deep-link focus
 * handlers.
 *
 * `⌘K` is a single global shortcut owned by `command-k-registry`: there is exactly
 * ONE `document` listener for all mounted selectors, and it resolves one
 * deterministic owner (open selector → last-focused → sole eligible) instead of
 * letting every instance install a competing per-instance handler. The trigger's
 * `onFocus` marks this selector as the last-focused owner candidate.
 */
export function useRuntimeSelectorPopup({
  controller,
  providers,
  value,
  disabled,
  triggerRef,
  searchRef,
  popupRef,
}: UseRuntimeSelectorPopupArgs) {
  const { open, openWith, close } = controller;

  const baseId = useId();
  const popupId = `${baseId}-popup`;
  const listId = `${baseId}-list`;
  const optionId = (rowIndex: number) => `${baseId}-option-${rowIndex}`;
  const activeDescendant =
    controller.highlightIndex >= 0 ? optionId(controller.highlightIndex) : undefined;

  // Register with the single global ⌘K owner registry (one document listener for
  // all selectors; deterministic owner resolution — never a per-instance handler).
  const openModel = () => openWith("model");
  useCommandKOwnership({
    id: baseId,
    disabled,
    open,
    triggerRef,
    onOpen: openModel,
    onClose: close,
  });

  // Focus entering this trigger makes it the last-focused ⌘K owner candidate.
  const handleTriggerFocus = () => noteCommandKFocus(baseId);

  const providerNames = new Map(providers.map(provider => [provider.id, provider.name]));
  const providerName = (id: string) => providerNames.get(id) ?? id;

  const providerKinds = new Map(
    providers.map(provider => [provider.id, provider.runtime_provider ?? provider.id])
  );
  const providerKind = (id: string) => providerKinds.get(id) ?? id;

  // Resolve the popup element a deep-link intent should land on: the current
  // provider's rail radio, the active reasoning button, else the search field.
  const focusRegion = (intent: TriggerFocus): HTMLElement | null => {
    const root = popupRef.current;
    if (root && intent === "provider") {
      const railItem = root.querySelector<HTMLElement>(
        `[data-rail="${CSS.escape(value.provider)}"]`
      );
      if (railItem) return railItem;
    }
    if (root && intent === "reasoning") {
      const active = root.querySelector<HTMLElement>('[data-on="true"]');
      if (active) return active;
      const first = root.querySelector<HTMLElement>("[data-rz]");
      if (first) return first;
    }
    return searchRef.current;
  };

  const handleOpenChange: PopoverOpenChange = (next, details) => {
    if (next) {
      if (!disabled) openWith("model");
      return;
    }
    // The anchor group is not a base-ui trigger, so an outside-press landing on
    // it would close-then-reopen; ignore it and let the segment handlers drive.
    if (details.reason === "outside-press") {
      const target = details.event?.target as Node | null;
      if (target && triggerRef.current?.contains(target)) return;
    }
    close();
  };

  const handleSegment = (focus: TriggerFocus) => {
    if (disabled) return;
    controller.setFocusIntent(focus);
    if (!open) {
      openWith(focus);
      return;
    }
    // Already open: clicking a different segment must actively re-route focus to
    // that region (updating the intent ref alone would not move focus, since
    // base-ui only applies `initialFocus` on the open transition).
    focusRegion(focus)?.focus();
  };

  const resolveInitialFocus = (): HTMLElement | null => focusRegion(controller.getFocusIntent());

  const anchor = () => triggerRef.current;

  // Restore focus to the exact segment that opened the popup (the tracked intent),
  // not always the first segment. Falls back to the provider segment then the group.
  const finalFocus = () => {
    const trigger = triggerRef.current;
    if (!trigger) return null;
    const intent = controller.getFocusIntent();
    return (
      trigger.querySelector<HTMLElement>(`[data-focus="${intent}"]`) ??
      trigger.querySelector<HTMLElement>('[data-focus="provider"]') ??
      trigger
    );
  };

  const handleSearchKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      controller.moveHighlight(1);
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      controller.moveHighlight(-1);
    } else if (event.key === "Home") {
      event.preventDefault();
      controller.moveHighlightEdge("first");
    } else if (event.key === "End") {
      event.preventDefault();
      controller.moveHighlightEdge("last");
    } else if (event.key === "Enter") {
      if (controller.commitHighlight()) event.preventDefault();
    } else if (event.altKey && event.code === "KeyF") {
      // Alt+F favorites the highlighted option (announced via aria-keyshortcuts).
      // `event.code` is layout-independent so it survives Alt producing "ƒ" on
      // macOS; Cmd/Ctrl-D is intentionally avoided (browser bookmark conflict).
      if (controller.toggleHighlightedFavorite()) event.preventDefault();
    }
    // ⌘K while open is handled by the global command-k registry (rule 1: the
    // open selector owns the shortcut and toggles itself closed).
  };

  return {
    popupId,
    listId,
    optionId,
    activeDescendant,
    providerName,
    providerKind,
    anchor,
    finalFocus,
    handleOpenChange,
    handleSegment,
    handleTriggerFocus,
    resolveInitialFocus,
    handleSearchKeyDown,
  };
}
