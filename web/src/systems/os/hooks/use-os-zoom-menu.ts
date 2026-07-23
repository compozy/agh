import { useEffect, useRef, useState } from "react";
import type { PointerEvent as ReactPointerEvent } from "react";

import type { OsArrangePreset } from "../lib/os-types";
import { arrangePeerWindows } from "../lib/window-manager-navigation";
import {
  dispatchWindowPlacement,
  type WindowPlacementCommand,
} from "../lib/window-manager-command-registry";
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
  floating: boolean;
  placementEnabled: boolean;
  /** Arrange presets need at least one other visible window (truthful UI). */
  arrangeEnabled: boolean;
  dispatchPlacement(command: WindowPlacementCommand): void;
  dispatchMakeFloating(): void;
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
  const { manager, store } = useOsShell();
  const [open, setOpen] = useState(false);
  const openTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const closeTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const floating = useDesktop(state => state.windows[windowId]?.placement === "floating");
  const placementEnabled = useDesktop(state => state.windowManagerConfig !== null);
  const arrangeEnabled = useDesktop(
    state => arrangePeerWindows(state.windows, windowId).length > 0
  );

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
    floating,
    placementEnabled,
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
    dispatchPlacement: command =>
      dispatch(() => {
        if (store.getState().windowManagerConfig !== null) {
          dispatchWindowPlacement(manager, windowId, command);
        }
      }),
    dispatchMakeFloating: () => dispatch(() => manager.getState().toggleFloating(windowId)),
    dispatchFill: () => dispatch(() => manager.getState().zoomWindow(windowId)),
    dispatchArrange: preset => dispatch(() => manager.getState().arrangeLayout(windowId, preset)),
  };
}
