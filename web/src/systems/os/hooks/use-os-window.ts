import {
  useEffect,
  useRef,
  useState,
  useSyncExternalStore,
  type FocusEvent,
  type PointerEvent,
  type RefObject,
} from "react";
import type { Rnd, RndDragCallback, RndResizeCallback } from "react-rnd";

import type { OsTrafficLightAction } from "../components/os-traffic-lights";
import { deriveSnapRect, snapWorkArea } from "../lib/os-snap-geometry";
import {
  createSnapZoneSession,
  resolutionToHint,
  type SnapZoneSession,
} from "../lib/os-snap-session";
import { snapTargetCandidates } from "../lib/os-snap-candidates";
import { OS_SNAP_ZONES } from "../lib/os-snap-zones";
import type { OsRect, OsWindow } from "../lib/os-types";
import { useDesktop } from "./use-desktop";
import { useOsReducedMotion } from "./use-os-reduced-motion";
import { useOsShell } from "./use-os-shell";
import { useOsSnapDrag } from "./use-os-snap-drag";

/** Genie commit delay (os-v2.js:412): the 0.3s spring travel, slightly early. */
const OS_MINIMIZE_ANIMATION_MS = 290;
/** Prototype fold target inside the dock item (os-v2.js `scale(.06)`). */
const OS_MINIMIZE_SCALE = 0.06;

type OsDragEvent = Parameters<RndDragCallback>[0];

export const OS_WINDOW_DRAG_HANDLE_CLASS = "os-window-drag-handle";
/** Head controls that must never start a drag. */
export const OS_WINDOW_DRAG_CANCEL_SELECTOR = [
  '[data-slot="os-traffic-lights"]',
  '[data-slot="topbar-back"]',
  '[data-slot="topbar-crumb"]',
  '[data-slot="topbar-crumb-more"]',
  '[data-slot="topbar-nav"]',
  '[data-slot="topbar-trailing"]',
].join(", ");

export interface OsWindowModel {
  win: OsWindow | undefined;
  focused: boolean;
  /** Body stays mounted while minimized when a window-scoped dialog is open. */
  keepMounted: boolean;
  /**
   * Locally derived rect while the window is in a derived-geometry state:
   * `snap` fractions × work area, or the work-area fill for `maximized`.
   * `null` while floating (the committed `rect` is authoritative).
   */
  derivedRect: OsRect | null;
  /** Live drag-away preview rect while a snapped window detaches; null at rest. */
  ghostRect: OsRect | null;
  /** Genie-minimize transform while the fold animation runs; null otherwise. */
  minimizeTransform: string | null;
  rndRef: RefObject<Rnd | null>;
  setOverlayHost: (element: HTMLDivElement | null) => void;
  overlayHost: HTMLDivElement | null;
  handleTrafficLight: (action: OsTrafficLightAction) => void;
  handlePointerDownCapture: (event: PointerEvent<HTMLElement>) => void;
  handleFocusCapture: (event: FocusEvent<HTMLElement>) => void;
  handleDragStart: RndDragCallback;
  handleDrag: RndDragCallback;
  handleDragStop: RndDragCallback;
  handleResizeStop: RndResizeCallback;
}

function useOverlayPresence(overlayHost: HTMLDivElement | null): boolean {
  const subscribe = (onStoreChange: () => void) => {
    if (!overlayHost) return () => undefined;
    const observer = new MutationObserver(onStoreChange);
    observer.observe(overlayHost, { childList: true });
    return () => observer.disconnect();
  };
  const getSnapshot = () => (overlayHost?.childElementCount ?? 0) > 0;
  return useSyncExternalStore(subscribe, getSnapshot, () => false);
}

function dragPoint(event: OsDragEvent): { x: number; y: number } | null {
  // react-draggable hands over native events from its window listeners;
  // unwrap the synthetic wrapper when React delivers one instead.
  const source = (event as { nativeEvent?: Event }).nativeEvent ?? (event as Event);
  if (source instanceof MouseEvent) return { x: source.clientX, y: source.clientY };
  if (typeof TouchEvent !== "undefined" && source instanceof TouchEvent) {
    const touch = source.touches[0];
    return touch ? { x: touch.clientX, y: touch.clientY } : null;
  }
  return null;
}

/**
 * Window behavior: focus activation (pointer AND keyboard — Routing Model
 * rule 5), traffic-light actions, gesture-end rect commits with immediate
 * persistence flush (invariant 15), the minimize-unmount posture with the
 * open-dialog exemption (invariant 18), and the snap layer (ADR-009): zone
 * resolution during Rnd drags, snap commit on release, Esc abort, and the
 * derived rect + drag-away session while snapped (invariant 19).
 */
export function useOsWindow(windowId: string): OsWindowModel {
  const { store, coordinator, flushPersistence } = useOsShell();
  const win = useDesktop(state => state.windows[windowId]);
  const focused = useDesktop(state => state.focusedId === windowId);
  const desktopBounds = useDesktop(state => state.desktopBounds);
  const seamZone = useDesktop(state => state.seamPreview?.[windowId] ?? null);
  const reducedMotion = useOsReducedMotion();
  const [overlayHost, setOverlayHost] = useState<HTMLDivElement | null>(null);
  const hasOpenOverlay = useOverlayPresence(overlayHost);
  const snapDrag = useOsSnapDrag(windowId);
  const rndRef = useRef<Rnd | null>(null);
  const zoneSessionRef = useRef<SnapZoneSession | null>(null);
  const [zoneDragActive, setZoneDragActive] = useState(false);
  const [minimizeTransform, setMinimizeTransform] = useState<string | null>(null);
  const minimizeTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (minimizeTimer.current !== null) {
        clearTimeout(minimizeTimer.current);
        minimizeTimer.current = null;
      }
    };
  }, []);

  // While a zone drag runs, Esc belongs to the drag alone (Hermes escape
  // discipline); the effect owns the listener lifecycle, including unmount
  // mid-gesture.
  useEffect(() => {
    if (!zoneDragActive) return;
    const onEscape = (key: KeyboardEvent) => {
      if (key.key !== "Escape") return;
      key.preventDefault();
      key.stopPropagation();
      zoneSessionRef.current?.abort();
    };
    window.addEventListener("keydown", onEscape, true);
    return () => window.removeEventListener("keydown", onEscape, true);
  }, [zoneDragActive]);

  useEffect(() => {
    return () => {
      zoneSessionRef.current?.abort();
      zoneSessionRef.current = null;
    };
  }, []);

  const derivedRect = (() => {
    if (!win) return null;
    if (win.snap !== null) {
      if (desktopBounds === null) return win.rect;
      // A live seam drag overrides the committed fractions frame-by-frame.
      return deriveSnapRect(seamZone ?? win.snap, desktopBounds);
    }
    if (win.maximized) {
      // Pre-measure (bounds null) the caller falls back to the CSS-inset fill.
      return desktopBounds === null ? null : snapWorkArea(desktopBounds);
    }
    return null;
  })();

  /**
   * Genie minimize (US-015.AC-2 motion tier): the pointer gesture folds the
   * window into its dock item before the minimize commits; reduced motion —
   * system preference OR the in-product toggle — makes it instant, and the
   * compact stack never animates (the dock is a tab bar there).
   */
  const beginMinimize = () => {
    const state = store.getState();
    const target = state.windows[windowId];
    if (!target || target.minimized || minimizeTimer.current !== null) return;
    if (reducedMotion || state.presentation === "compact") {
      coordinator.userMinimize(windowId);
      return;
    }
    const frame = document.querySelector(`[data-testid="os-window-${windowId}"]`);
    const dockItem = document.querySelector(`[data-slot="os-dock-item"][data-app="${target.app}"]`);
    if (!(frame instanceof HTMLElement) || !(dockItem instanceof HTMLElement)) {
      coordinator.userMinimize(windowId);
      return;
    }
    const frameRect = frame.getBoundingClientRect();
    const dockRect = dockItem.getBoundingClientRect();
    const dx = dockRect.left + dockRect.width / 2 - (frameRect.left + frameRect.width / 2);
    const dy = dockRect.top + dockRect.height / 2 - (frameRect.top + frameRect.height / 2);
    setMinimizeTransform(`translate(${dx}px, ${dy}px) scale(${OS_MINIMIZE_SCALE})`);
    minimizeTimer.current = setTimeout(() => {
      minimizeTimer.current = null;
      setMinimizeTransform(null);
      coordinator.userMinimize(windowId);
    }, OS_MINIMIZE_ANIMATION_MS);
  };

  const handleTrafficLight = (action: OsTrafficLightAction) => {
    if (action === "close") coordinator.userClose(windowId);
    else if (action === "minimize") beginMinimize();
    else store.getState().toggleZoom(windowId);
  };

  const activate = (target: EventTarget | null) => {
    if (!win || (focused && !win.minimized)) return;
    const element = target instanceof Element ? target : null;
    coordinator.userFocus(windowId, { viaLink: Boolean(element?.closest("a[href]")) });
  };

  const handlePointerDownCapture = (event: PointerEvent<HTMLElement>) => {
    activate(event.target);
    if (!win || win.snap === null) return;
    const element = event.target instanceof Element ? event.target : null;
    if (!element?.closest('[data-slot="os-window-head"]')) return;
    if (element.closest(OS_WINDOW_DRAG_CANCEL_SELECTOR)) return;
    snapDrag.onHeadPointerDown(event);
  };

  const handleDragStart: RndDragCallback = (_event, data) => {
    const state = store.getState();
    const bounds = state.desktopBounds;
    const layerElement = data.node.closest('[data-slot="os-win-layer"]');
    if (bounds === null || !(layerElement instanceof HTMLElement)) return;
    const session = createSnapZoneSession({
      layer: layerElement.getBoundingClientRect(),
      bounds,
      windowTargets: snapTargetCandidates(state, windowId),
      publish: resolution => store.getState().setSnapHint(resolutionToHint(windowId, resolution)),
    });
    zoneSessionRef.current = session;
    setZoneDragActive(true);
  };

  const handleDrag: RndDragCallback = (event, _data) => {
    const session = zoneSessionRef.current;
    if (!session) return;
    const point = dragPoint(event);
    if (point) session.move(point.x, point.y);
  };

  const handleDragStop: RndDragCallback = (_event, data) => {
    const session = zoneSessionRef.current;
    zoneSessionRef.current = null;
    setZoneDragActive(false);
    const current = store.getState().windows[windowId];
    if (!current) {
      session?.abort();
      return;
    }
    if (session) {
      if (session.aborted()) {
        // Esc abort (US-021.EC-1): nothing commits; the ghost springs back to
        // the store-owned rect (controlled position stays authoritative).
        rndRef.current?.updatePosition({ x: current.rect.x, y: current.rect.y });
        return;
      }
      const resolution = session.end();
      if (resolution !== null) {
        if (resolution.kind === "zone") {
          store.getState().snapWindow(windowId, OS_SNAP_ZONES[resolution.zoneId]);
        } else {
          store
            .getState()
            .splitWindows(windowId, resolution.target.targetId, resolution.target.side);
        }
        return;
      }
    }
    store.getState().commitRect(windowId, { ...current.rect, x: data.x, y: data.y });
    flushPersistence();
  };

  const handleResizeStop: RndResizeCallback = (_event, _dir, element, _delta, position) => {
    const state = store.getState();
    const current = state.windows[windowId];
    if (!current) return;
    const rect = {
      x: position.x,
      y: position.y,
      w: element.offsetWidth,
      h: element.offsetHeight,
    };
    // Snapped windows stay snapped through a resize (fraction rewrite);
    // floating and maximized windows land a floating px rect (maximized
    // detaches — resizing a zoomed window makes it a normal window).
    if (current.snap !== null) state.resizeSnapped(windowId, rect);
    else state.commitRect(windowId, rect);
    flushPersistence();
  };

  return {
    win,
    focused,
    keepMounted: win ? !win.minimized || hasOpenOverlay : false,
    derivedRect,
    ghostRect: win && win.snap !== null ? snapDrag.dragRect : null,
    minimizeTransform,
    rndRef,
    overlayHost,
    setOverlayHost,
    handleTrafficLight,
    handlePointerDownCapture,
    handleFocusCapture: event => activate(event.target),
    handleDragStart,
    handleDrag,
    handleDragStop,
    handleResizeStop,
  };
}
