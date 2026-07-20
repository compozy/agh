import { useEffect, useEffectEvent } from "react";

import { useOsShell } from "./use-os-shell";

export interface OsShortcutHandlers {
  onPalette: () => void;
  onNewSession: () => void;
  onSpaces: () => void;
  onEscape: () => void;
}

function isPlainMod(event: KeyboardEvent): boolean {
  return (event.metaKey || event.ctrlKey) && !event.altKey && !event.shiftKey;
}

function isShiftMod(event: KeyboardEvent): boolean {
  return (event.metaKey || event.ctrlKey) && event.shiftKey && !event.altKey;
}

/**
 * The shell's global shortcut set (ADR-005): ⌘K palette (owned here — the old
 * RuntimeSelector registry is deleted), ⌘N new session, ⇧⌘S Spaces, ⌘W
 * close, ⌘M minimize, and Esc focus return. `preventDefault` applies where
 * the browser yields; menus remain the discoverable fallback.
 */
export function useOsShortcuts(handlers: OsShortcutHandlers): void {
  const { store, coordinator } = useOsShell();
  const handleKeyDown = useEffectEvent((event: KeyboardEvent) => {
    if (event.key === "Escape") {
      handlers.onEscape();
      return;
    }
    const key = event.key.toLowerCase();
    if (key === "s" && isShiftMod(event)) {
      event.preventDefault();
      handlers.onSpaces();
      return;
    }
    if (!isPlainMod(event)) return;
    if (key === "k") {
      event.preventDefault();
      handlers.onPalette();
      return;
    }
    if (key === "n") {
      event.preventDefault();
      handlers.onNewSession();
      return;
    }
    if (key !== "w" && key !== "m") return;
    const { focusedId } = store.getState();
    if (focusedId === null) return;
    event.preventDefault();
    if (key === "w") coordinator.userClose(focusedId);
    else coordinator.userMinimize(focusedId);
  });

  useEffect(() => {
    const listener = (event: KeyboardEvent) => handleKeyDown(event);
    document.addEventListener("keydown", listener);
    return () => document.removeEventListener("keydown", listener);
  }, []);
}
