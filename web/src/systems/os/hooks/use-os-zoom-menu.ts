import { useEffect, useRef, useState } from "react";
import type { PointerEvent as ReactPointerEvent } from "react";

import type { OsSnapCommand } from "../lib/os-snap-commands";
import { OS_SNAP_ZONES } from "../lib/os-snap-zones";
import type { OsArrangePreset } from "../lib/os-types";
import { useDesktop } from "./use-desktop";
import { useOsShell } from "./use-os-shell";

/** Hover intent before the zoom menu opens (macOS green-button posture). */
export const OS_ZOOM_MENU_OPEN_DELAY_MS = 250;
/** Grace period crossing from the button into the menu before it closes. */
export const OS_ZOOM_MENU_CLOSE_GRACE_MS = 300;

export interface OsZoomMenuModel {
  open: boolean;
  onOpenChange(open: boolean): void;
  onHoverEnter(event: ReactPointerEvent<HTMLElement>): void;
  onHoverLeave(): void;
  onContentEnter(): void;
  /** Restore renders only while the window is snapped (palette parity). */
  snapped: boolean;
  /** Arrange presets need at least one other visible window (truthful UI). */
  arrangeEnabled: boolean;
  dispatchSnap(command: OsSnapCommand): void;
  dispatchFill(): void;
  dispatchArrange(preset: OsArrangePreset): void;
}

/**
 * Zoom-menu behavior: hover intent (250ms open / 300ms close grace, mouse
 * pointers only — touch never hover-opens), dispatches against THIS window
 * (not the focused one), and close-on-dispatch. Click on the zoom button
 * stays `toggleZoom`; the menu is the pointer-affordance mirror of the
 * palette/chords surface, which remains the guaranteed keyboard path.
 */
export function useOsZoomMenu(windowId: string): OsZoomMenuModel {
  const { store } = useOsShell();
  const [open, setOpen] = useState(false);
  const openTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const closeTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const snapped = useDesktop(state => (state.windows[windowId]?.snap ?? null) !== null);
  const arrangeEnabled = useDesktop(state => {
    for (const win of Object.values(state.windows)) {
      if (win.id !== windowId && !win.minimized) return true;
    }
    return false;
  });

  const clearTimers = () => {
    if (openTimer.current !== null) {
      clearTimeout(openTimer.current);
      openTimer.current = null;
    }
    if (closeTimer.current !== null) {
      clearTimeout(closeTimer.current);
      closeTimer.current = null;
    }
  };

  useEffect(() => clearTimers, []);

  const dispatch = (action: () => void) => {
    clearTimers();
    setOpen(false);
    action();
  };

  return {
    open,
    snapped,
    arrangeEnabled,
    onOpenChange: next => {
      clearTimers();
      setOpen(next);
    },
    onHoverEnter: event => {
      if (event.pointerType !== "mouse") return;
      if (closeTimer.current !== null) {
        clearTimeout(closeTimer.current);
        closeTimer.current = null;
      }
      if (open || openTimer.current !== null) return;
      openTimer.current = setTimeout(() => {
        openTimer.current = null;
        setOpen(true);
      }, OS_ZOOM_MENU_OPEN_DELAY_MS);
    },
    onHoverLeave: () => {
      if (openTimer.current !== null) {
        clearTimeout(openTimer.current);
        openTimer.current = null;
      }
      if (!open || closeTimer.current !== null) return;
      closeTimer.current = setTimeout(() => {
        closeTimer.current = null;
        setOpen(false);
      }, OS_ZOOM_MENU_CLOSE_GRACE_MS);
    },
    onContentEnter: () => {
      if (closeTimer.current !== null) {
        clearTimeout(closeTimer.current);
        closeTimer.current = null;
      }
    },
    dispatchSnap: command =>
      dispatch(() =>
        store
          .getState()
          .snapWindow(windowId, command.zoneId === null ? null : OS_SNAP_ZONES[command.zoneId])
      ),
    dispatchFill: () =>
      dispatch(() => {
        const state = store.getState();
        if (!state.windows[windowId]?.maximized) state.toggleZoom(windowId);
      }),
    dispatchArrange: preset => dispatch(() => store.getState().arrangeWindows(windowId, preset)),
  };
}
