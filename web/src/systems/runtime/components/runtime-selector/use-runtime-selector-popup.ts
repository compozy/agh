import {
  useCallback,
  useId,
  useMemo,
  useRef,
  type ComponentProps,
  type KeyboardEvent,
} from "react";

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
}: UseRuntimeSelectorPopupArgs) {
  const { open, openWith, close } = controller;
  const triggerRef = useRef<HTMLDivElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);
  const popupRef = useRef<HTMLDivElement>(null);

  const baseId = useId();
  const popupId = `${baseId}-popup`;
  const listId = `${baseId}-list`;
  const optionId = useCallback((rowIndex: number) => `${baseId}-option-${rowIndex}`, [baseId]);
  const activeDescendant =
    controller.highlightIndex >= 0 ? optionId(controller.highlightIndex) : undefined;

  // Register with the single global ⌘K owner registry (one document listener for
  // all selectors; deterministic owner resolution — never a per-instance handler).
  const openModel = useCallback(() => openWith("model"), [openWith]);
  useCommandKOwnership({
    id: baseId,
    disabled,
    open,
    triggerRef,
    onOpen: openModel,
    onClose: close,
  });

  // Focus entering this trigger makes it the last-focused ⌘K owner candidate.
  const handleTriggerFocus = useCallback(() => noteCommandKFocus(baseId), [baseId]);

  const providerName = useMemo(() => {
    const byId = new Map(providers.map(provider => [provider.id, provider.name]));
    return (id: string) => byId.get(id) ?? id;
  }, [providers]);

  const providerKind = useMemo(() => {
    const byId = new Map(
      providers.map(provider => [provider.id, provider.runtime_provider ?? provider.id])
    );
    return (id: string) => byId.get(id) ?? id;
  }, [providers]);

  // Resolve the popup element a deep-link intent should land on: the current
  // provider's rail radio, the active reasoning button, else the search field.
  const focusRegion = useCallback(
    (intent: TriggerFocus): HTMLElement | null => {
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
    },
    [value.provider]
  );

  const handleOpenChange = useCallback<PopoverOpenChange>(
    (next, details) => {
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
    },
    [close, disabled, openWith]
  );

  const handleSegment = useCallback(
    (focus: TriggerFocus) => {
      if (disabled) return;
      controller.focusIntent.current = focus;
      if (!open) {
        openWith(focus);
        return;
      }
      // Already open: clicking a different segment must actively re-route focus to
      // that region (updating the intent ref alone would not move focus, since
      // base-ui only applies `initialFocus` on the open transition).
      focusRegion(focus)?.focus();
    },
    [controller.focusIntent, disabled, focusRegion, open, openWith]
  );

  const resolveInitialFocus = useCallback(
    (): HTMLElement | null => focusRegion(controller.focusIntent.current),
    [controller.focusIntent, focusRegion]
  );

  const anchor = useCallback(() => triggerRef.current, []);

  // Restore focus to the exact segment that opened the popup (the tracked intent),
  // not always the first segment. Falls back to the provider segment then the group.
  const finalFocus = useCallback(() => {
    const trigger = triggerRef.current;
    if (!trigger) return null;
    const intent = controller.focusIntent.current;
    return (
      trigger.querySelector<HTMLElement>(`[data-focus="${intent}"]`) ??
      trigger.querySelector<HTMLElement>('[data-focus="provider"]') ??
      trigger
    );
  }, [controller.focusIntent]);

  const handleSearchKeyDown = useCallback(
    (event: KeyboardEvent<HTMLInputElement>) => {
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
    },
    [controller]
  );

  return {
    triggerRef,
    searchRef,
    popupRef,
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
