import {
  useEffect,
  useRef,
  useState,
  type FocusEvent,
  type PointerEvent,
  type RefObject,
} from "react";
import type { Rnd, RndDragCallback, RndResizeCallback } from "react-rnd";

import type { OsTrafficLightAction } from "../components/os-traffic-lights";
import { bindLayoutGestureCancellation } from "../lib/layout-gesture-cancellation";
import { resolveSnapTarget, type OccupiedSnapCandidate } from "../lib/snap-targets";
import type { OsRect, OsWindow } from "../lib/os-types";
import { snapTargetConfigFromConfig } from "../lib/window-manager-view";
import { windowManagerStore } from "../stores/window-manager-store";
import { useDesktop } from "./use-desktop";
import { useOsShell } from "./use-os-shell";
import {
  useWindowManagerActions,
  useWindowManagerGestureActive,
  useWindowManagerWorkArea,
} from "./use-window-manager-store";

type OsDragEvent = Parameters<RndDragCallback>[0];

export const OS_WINDOW_DRAG_HANDLE_CLASS = "os-window-drag-handle";
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
  keepMounted: boolean;
  rect: OsRect | null;
  rndRef: RefObject<Rnd | null>;
  setOverlayHost: (element: HTMLDivElement | null) => void;
  overlayHost: HTMLDivElement | null;
  handleTrafficLight: (action: OsTrafficLightAction) => void;
  handlePointerEnter: () => void;
  handlePointerDownCapture: (event: PointerEvent<HTMLElement>) => void;
  handleFocusCapture: (event: FocusEvent<HTMLElement>) => void;
  handleDragStart: RndDragCallback;
  handleDrag: RndDragCallback;
  handleDragStop: RndDragCallback;
  handleResizeStop: RndResizeCallback;
}

function dragPoint(event: OsDragEvent): { x: number; y: number } | null {
  const source = (event as { nativeEvent?: Event }).nativeEvent ?? (event as Event);
  if (source instanceof MouseEvent) return { x: source.clientX, y: source.clientY };
  if (typeof TouchEvent !== "undefined" && source instanceof TouchEvent) {
    const touch = source.touches[0];
    return touch ? { x: touch.clientX, y: touch.clientY } : null;
  }
  return null;
}

/**
 * Snap resolution, window rects, and rnd positions all live in layer
 * coordinates; pointer events arrive in viewport coordinates. Subtracting the
 * measured layer origin keeps every hot zone aligned with what the user sees.
 */
function layerPoint(event: OsDragEvent): { x: number; y: number } | null {
  const client = dragPoint(event);
  if (client === null) return null;
  const origin = windowManagerStore.getState().workArea?.origin;
  if (origin === undefined) return null;
  return { x: client.x - origin.x, y: client.y - origin.y };
}

function modifierActive(
  event: OsDragEvent,
  modifier: "alt" | "control" | "meta" | "shift" | "none"
): boolean {
  switch (modifier) {
    case "alt":
      return "altKey" in event && event.altKey;
    case "control":
      return "ctrlKey" in event && event.ctrlKey;
    case "meta":
      return "metaKey" in event && event.metaKey;
    case "shift":
      return "shiftKey" in event && event.shiftKey;
    case "none":
      return false;
  }
}

/** Semantic window gestures; pixels remain preview-only until one command commits. */
export function useOsWindow(windowId: string): OsWindowModel {
  const { store, manager, coordinator } = useOsShell();
  const actions = useWindowManagerActions();
  const gestureActive = useWindowManagerGestureActive(windowId);
  const workArea = useWindowManagerWorkArea();
  const win = useDesktop(state => state.windows[windowId]);
  const focused = useDesktop(state => state.focusedId === windowId);
  const [overlayHost, setOverlayHost] = useState<HTMLDivElement | null>(null);
  const rndRef = useRef<Rnd | null>(null);

  useEffect(() => {
    if (!gestureActive) return;
    return bindLayoutGestureCancellation(window, actions.cancelGesture);
  }, [actions, gestureActive]);

  const candidates = (): OccupiedSnapCandidate[] => {
    const state = store.getState();
    if (!win) return [];
    return Object.values(state.windows).flatMap(window =>
      window.id === windowId ||
      window.desktopId !== win.desktopId ||
      window.minimized ||
      window.nodeId === null
        ? []
        : [
            {
              windowId: window.id,
              nodeId: window.nodeId,
              rect: window.rect,
              z: window.layer,
              parentAxis: window.parentAxis,
            },
          ]
    );
  };

  const targetAt = (
    point: { x: number; y: number },
    capturedWorkArea: OsRect,
    swapModifierActive: boolean
  ) => {
    const config = snapTargetConfigFromConfig(manager.getState().windowManagerConfig);
    if (config === null) return null;
    return resolveSnapTarget({
      point,
      workArea: capturedWorkArea,
      candidates: candidates(),
      currentTarget: windowManagerStoreGesture() ?? undefined,
      config,
      swapModifierActive,
    });
  };

  const swapModifierHeld = (event: OsDragEvent): boolean => {
    const modifier = store.getState().windowManagerConfig?.swapModifier;
    return modifier !== undefined && modifier !== "none" && modifierActive(event, modifier);
  };

  // Escape cancelled the gesture: snap back to the source rect instead of tracking the pointer.
  const snapBackCancelled = (): boolean => {
    const gesture = windowManagerStore.getState().gesture;
    if (gesture?.status !== "cancelled" || gesture.source.windowId !== windowId) return false;
    if (win) rndRef.current?.updatePosition({ x: win.rect.x, y: win.rect.y });
    return true;
  };

  const activate = (target: EventTarget | null) => {
    if (!win || (focused && !win.minimized)) return;
    const element = target instanceof Element ? target : null;
    void coordinator.userFocus(windowId, { viaLink: Boolean(element?.closest("a[href]")) });
  };

  const handleDragStart: RndDragCallback = (event, _data) => {
    const point = layerPoint(event);
    const state = store.getState();
    const config = state.windowManagerConfig;
    if (!point || !workArea || !win || !state.snapshot || config === null) return;
    const moveGroup =
      config.dragAwayPolicy === "group" || modifierActive(event, config.groupMoveModifier);
    actions.beginGesture({
      pointerId: 0,
      point,
      workArea: workArea.rect,
      layoutRevision: state.snapshot.revision,
      source: {
        windowId,
        nodeId: win.nodeId,
        groupId: win.groupId,
        moveMode: moveGroup ? "group" : "window",
      },
    });
  };

  const handleDrag: RndDragCallback = (event, _data) => {
    if (snapBackCancelled()) return;
    const point = layerPoint(event);
    const gesture = windowManagerStore.getState().gesture;
    const currentWorkArea = windowManagerStore.getState().workArea?.rect;
    if (!point || gesture?.status !== "active" || !currentWorkArea) return;
    actions.previewGesture(
      point,
      targetAt(point, gesture.workArea, swapModifierHeld(event)),
      currentWorkArea
    );
  };

  const handleDragStop: RndDragCallback = (event, data) => {
    if (snapBackCancelled()) {
      actions.clearGesture();
      return;
    }
    const point = layerPoint(event) ?? { x: data.x, y: data.y };
    const state = store.getState();
    const currentGesture = windowManagerStore.getState().gesture;
    const currentWorkArea = windowManagerStore.getState().workArea?.rect;
    if (!win || !state.snapshot || currentGesture?.status !== "active" || !currentWorkArea) return;
    const target = targetAt(point, currentGesture.workArea, swapModifierHeld(event));
    const decision = actions.finishGesture({
      finalPoint: point,
      finalTarget: target,
      currentRevision: state.snapshot.revision,
      currentWorkArea,
      clientValid: state.client !== null,
    });
    if (decision?.kind === "commit") {
      manager.applySnapTarget(
        windowId,
        decision.command.target,
        decision.command.source.moveMode === "group"
      );
      return;
    }
    if (decision?.reason === "no-target") {
      manager.getState().commitFloatingRect(
        windowId,
        { ...win.rect, x: data.x, y: data.y },
        {
          pointer: point,
          grabOffset: {
            x: Math.max(0, point.x - data.x),
            y: Math.max(0, point.y - data.y),
          },
        }
      );
    }
    actions.clearGesture();
  };

  const handleResizeStop: RndResizeCallback = (_event, _direction, element, _delta, position) => {
    if (!win || win.placement !== "floating") return;
    manager.getState().commitFloatingRect(windowId, {
      x: position.x,
      y: position.y,
      w: element.offsetWidth,
      h: element.offsetHeight,
    });
  };

  return {
    win,
    focused,
    keepMounted: Boolean(win),
    rect: win?.rect ?? null,
    rndRef,
    overlayHost,
    setOverlayHost,
    handleTrafficLight: action => {
      if (action === "close") void coordinator.userClose(windowId);
      else if (action === "minimize") void coordinator.userMinimize(windowId);
      else void coordinator.userZoom(windowId);
    },
    handlePointerEnter: () => {
      const state = store.getState();
      if (state.windowManagerConfig?.focusFollowsPointer) {
        void coordinator.userFocus(windowId);
      }
    },
    handlePointerDownCapture: event => {
      if (store.getState().windowManagerConfig?.focusPolicy === "click_directional") {
        activate(event.target);
      }
    },
    handleFocusCapture: event => activate(event.target),
    handleDragStart,
    handleDrag,
    handleDragStop,
    handleResizeStop,
  };
}

function windowManagerStoreGesture() {
  const session = windowManagerStore.getState().gesture;
  return session?.status === "active" ? session.preview : null;
}
