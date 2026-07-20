import { useEffect, useState, type FocusEvent, type PointerEvent } from "react";
import type { RndDragCallback, RndResizeCallback } from "react-rnd";

import type { OsTrafficLightAction } from "../components/os-traffic-lights";
import type { OsWindow } from "../lib/os-types";
import { useDesktop } from "./use-desktop";
import { useOsShell } from "./use-os-shell";

export interface OsWindowModel {
  win: OsWindow | undefined;
  focused: boolean;
  /** Body stays mounted while minimized when a window-scoped dialog is open. */
  keepMounted: boolean;
  setOverlayHost: (element: HTMLDivElement | null) => void;
  overlayHost: HTMLDivElement | null;
  handleTrafficLight: (action: OsTrafficLightAction) => void;
  handlePointerDownCapture: (event: PointerEvent<HTMLElement>) => void;
  handleFocusCapture: (event: FocusEvent<HTMLElement>) => void;
  handleDragStop: RndDragCallback;
  handleResizeStop: RndResizeCallback;
}

/**
 * Window behavior: focus activation (pointer AND keyboard — Routing Model
 * rule 5), traffic-light actions, gesture-end rect commits with immediate
 * persistence flush (invariant 15), and the minimize-unmount posture with the
 * open-dialog exemption observed through the overlay host (invariant 18).
 */
export function useOsWindow(windowId: string): OsWindowModel {
  const { store, coordinator, flushPersistence } = useOsShell();
  const win = useDesktop(state => state.windows[windowId]);
  const focused = useDesktop(state => state.focusedId === windowId);
  const [overlayHost, setOverlayHost] = useState<HTMLDivElement | null>(null);
  const [hasOpenOverlay, setHasOpenOverlay] = useState(false);

  // The dedicated overlay host is the only portal target; watching its
  // children is what makes the minimize exemption observable (invariant 18).
  useEffect(() => {
    if (!overlayHost) return;
    const update = () => setHasOpenOverlay(overlayHost.childElementCount > 0);
    update();
    const observer = new MutationObserver(update);
    observer.observe(overlayHost, { childList: true });
    return () => observer.disconnect();
  }, [overlayHost]);

  const handleTrafficLight = (action: OsTrafficLightAction) => {
    if (action === "close") coordinator.userClose(windowId);
    else if (action === "minimize") coordinator.userMinimize(windowId);
    else store.getState().toggleZoom(windowId);
  };

  const activate = (target: EventTarget | null) => {
    if (!win || (focused && !win.minimized)) return;
    const element = target instanceof Element ? target : null;
    coordinator.userFocus(windowId, { viaLink: Boolean(element?.closest("a[href]")) });
  };

  const handleDragStop: RndDragCallback = (_event, data) => {
    const current = store.getState().windows[windowId];
    if (!current) return;
    store.getState().commitRect(windowId, { ...current.rect, x: data.x, y: data.y });
    flushPersistence();
  };

  const handleResizeStop: RndResizeCallback = (_event, _dir, element, _delta, position) => {
    store.getState().commitRect(windowId, {
      x: position.x,
      y: position.y,
      w: element.offsetWidth,
      h: element.offsetHeight,
    });
    flushPersistence();
  };

  return {
    win,
    focused,
    keepMounted: win ? !win.minimized || hasOpenOverlay : false,
    overlayHost,
    setOverlayHost,
    handleTrafficLight,
    handlePointerDownCapture: event => activate(event.target),
    handleFocusCapture: event => activate(event.target),
    handleDragStop,
    handleResizeStop,
  };
}
