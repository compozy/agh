import { useEffect, useEffectEvent } from "react";

import { OS_SNAP_COMMANDS } from "../lib/os-snap-commands";
import { OS_SNAP_ZONES } from "../lib/os-snap-zones";
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

/** ⌃⌥ chord (Rectangle-style, ADR-009); matched by `code` so layouts can't shift it. */
function isSnapChord(event: KeyboardEvent): boolean {
  return event.ctrlKey && event.altKey && !event.metaKey && !event.shiftKey;
}

/** AltGr aliases ⌃⌥ on some layouts — never steal keystrokes from text entry. */
function isEditableTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  return (
    target.isContentEditable ||
    target instanceof HTMLInputElement ||
    target instanceof HTMLTextAreaElement ||
    target instanceof HTMLSelectElement
  );
}

/**
 * The shell's global shortcut set (ADR-005): ⌘K palette (owned here — the old
 * RuntimeSelector registry is deleted), ⌘N new session, ⇧⌘S Spaces, ⌘W
 * close, ⌘M minimize, Esc focus return, and the ⌃⌥ snap chords (ADR-009 —
 * halves/quarters/restore; no-ops in compact presentation, UT-061 gating).
 * `preventDefault` applies where the browser yields; menus and the palette
 * remain the discoverable fallback.
 */
export function useOsShortcuts(handlers: OsShortcutHandlers): void {
  const { store, coordinator } = useOsShell();
  const handleKeyDown = useEffectEvent((event: KeyboardEvent) => {
    if (event.key === "Escape") {
      handlers.onEscape();
      return;
    }
    if (isSnapChord(event)) {
      if (isEditableTarget(event.target)) return;
      const state = store.getState();
      if (state.presentation === "compact" || state.focusedId === null) return;
      const command = OS_SNAP_COMMANDS.find(row => row.code === event.code);
      if (!command) return;
      event.preventDefault();
      state.snapWindow(
        state.focusedId,
        command.zoneId === null ? null : OS_SNAP_ZONES[command.zoneId]
      );
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
