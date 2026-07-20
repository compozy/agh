import { useEffect, useEffectEvent } from "react";

/**
 * Component-scoped `⌘J`/`Ctrl+J` binding for the RuntimeSelector (ADR-005 —
 * the global `⌘K` registry is deleted; `⌘K` belongs to the shell palette).
 *
 * The shortcut fires only while focus lives inside the selector's scope: the
 * nearest form/dialog that hosts the trigger (the composer surface), falling
 * back to the trigger itself. Hidden or disabled selectors never respond.
 */
export interface UseRuntimeJShortcutArgs {
  disabled: boolean;
  open: boolean;
  triggerRef: { current: HTMLElement | null };
  /** The portaled popup — part of the scope while open (it owns focus then). */
  popupRef?: { current: HTMLElement | null };
  onOpen: () => void;
  onClose: () => void;
}

export function triggerVisible(element: HTMLElement | null): boolean {
  if (!element || !element.isConnected) return false;
  // aria-hidden ancestors are a11y-hidden even when layout-visible.
  // checkVisibility() does not cover this, so gate it before the platform check.
  if (element.closest('[aria-hidden="true"]')) return false;
  // Real browsers: the platform check honors display/visibility/content-visibility
  // and the `hidden` attribute. jsdom has no layout engine, so fall back to a
  // connected-and-not-in-a-hidden-subtree heuristic.
  if (typeof element.checkVisibility === "function") return element.checkVisibility();
  return !element.closest("[hidden]");
}

function isRuntimeShortcut(event: KeyboardEvent): boolean {
  return (
    (event.metaKey || event.ctrlKey) &&
    !event.altKey &&
    !event.shiftKey &&
    event.key.toLowerCase() === "j"
  );
}

function shortcutScope(trigger: HTMLElement): HTMLElement {
  return trigger.closest<HTMLElement>("form, [role='dialog'], [data-runtime-scope]") ?? trigger;
}

export function useRuntimeJShortcut(args: UseRuntimeJShortcutArgs): void {
  const handle = useEffectEvent((event: KeyboardEvent) => {
    if (!isRuntimeShortcut(event)) return;
    if (args.disabled) return;
    const trigger = args.triggerRef.current;
    if (!triggerVisible(trigger)) return;
    const scope = trigger ? shortcutScope(trigger) : null;
    const popup = args.popupRef?.current ?? null;
    const target = event.target;
    const inScope =
      target instanceof Node &&
      ((scope?.contains(target) ?? false) || (popup?.contains(target) ?? false));
    if (!inScope) return;
    event.preventDefault();
    if (args.open) args.onClose();
    else args.onOpen();
  });

  useEffect(() => {
    const listener = (event: KeyboardEvent) => handle(event);
    document.addEventListener("keydown", listener);
    return () => document.removeEventListener("keydown", listener);
  }, []);
}
