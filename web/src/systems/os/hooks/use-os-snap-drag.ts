import { useEffect, useRef, useState, type PointerEvent as ReactPointerEvent } from "react";

import { getOsApp } from "../lib/app-registry";
import { snapTargetCandidates } from "../lib/os-snap-candidates";
import { deriveSnapRect } from "../lib/os-snap-geometry";
import {
  createSnapZoneSession,
  resolutionToHint,
  type SnapZoneSession,
} from "../lib/os-snap-session";
import { OS_SNAP_DRAG_THRESHOLD, OS_SNAP_ZONES } from "../lib/os-snap-zones";
import type { OsRect } from "../lib/os-types";
import { useOsShell } from "./use-os-shell";

export interface OsSnapDragModel {
  /** Rect the detached window follows while the pointer holds it; null at rest. */
  dragRect: OsRect | null;
  /** Head pointerdown entry for the snapped (non-Rnd) window path. */
  onHeadPointerDown: (event: ReactPointerEvent<HTMLElement>) => void;
}

interface DetachedSession {
  target: HTMLElement;
  pointerId: number;
  startX: number;
  startY: number;
  zone: SnapZoneSession;
  engaged: boolean;
  lastRect: OsRect | null;
  cleanup: () => void;
}

const HEAD_GRAB_MAX_DY = 40;

/**
 * Drag-away session for a snapped window (US-021.AC-3): past the 4px
 * threshold the window detaches at its pre-snap dimensions under the pointer
 * (Windows behavior), zones stay live for a direct re-snap, release commits —
 * `commitRect` clears `snap` in the same mutation (invariant 19) — and Esc
 * aborts with geometry unchanged. Snapped windows bypass react-rnd, so this
 * hook owns the pointer machinery the Rnd path gets for free.
 */
export function useOsSnapDrag(windowId: string): OsSnapDragModel {
  const { store, flushPersistence } = useOsShell();
  const [dragRect, setDragRect] = useState<OsRect | null>(null);
  const sessionRef = useRef<DetachedSession | null>(null);

  const teardown = () => {
    const session = sessionRef.current;
    if (!session) return;
    sessionRef.current = null;
    session.cleanup();
    setDragRect(null);
  };

  useEffect(() => {
    return () => {
      const session = sessionRef.current;
      if (!session) return;
      sessionRef.current = null;
      session.zone.abort();
      session.cleanup();
    };
  }, []);

  const onHeadPointerDown = (event: ReactPointerEvent<HTMLElement>) => {
    if (event.button !== 0 || sessionRef.current !== null) return;
    const state = store.getState();
    const win = state.windows[windowId];
    if (!win || win.snap === null || state.presentation !== "floating") return;
    const bounds = state.desktopBounds;
    if (bounds === null) return;
    const layerElement = event.currentTarget.closest('[data-slot="os-win-layer"]');
    if (!(layerElement instanceof HTMLElement)) return;

    const layer = layerElement.getBoundingClientRect();
    const derived = deriveSnapRect(win.snap, bounds);
    const fallback = getOsApp(win.app).defaultRect;
    const restoreW = win.prevRect?.w ?? fallback.w;
    const restoreH = win.prevRect?.h ?? fallback.h;
    const grabFx = Math.min(
      Math.max((event.clientX - layer.left - derived.x) / Math.max(derived.w, 1), 0),
      1
    );
    const grabDy = Math.min(Math.max(event.clientY - layer.top - derived.y, 0), HEAD_GRAB_MAX_DY);

    const zone = createSnapZoneSession({
      layer,
      bounds,
      windowTargets: snapTargetCandidates(state, windowId),
      publish: resolution => store.getState().setSnapHint(resolutionToHint(windowId, resolution)),
      onFrame: point => {
        const session = sessionRef.current;
        if (!session || !session.engaged) return;
        const rect = {
          x: Math.round(point.x - grabFx * restoreW),
          y: Math.round(point.y - grabDy),
          w: restoreW,
          h: restoreH,
        };
        session.lastRect = rect;
        setDragRect(rect);
      },
    });

    const target = event.currentTarget;
    const pointerId = event.pointerId;

    const onPointerMove = (move: PointerEvent) => {
      const session = sessionRef.current;
      if (!session || move.pointerId !== session.pointerId || session.zone.aborted()) return;
      if (!session.engaged) {
        const distance = Math.hypot(move.clientX - session.startX, move.clientY - session.startY);
        if (distance < OS_SNAP_DRAG_THRESHOLD) return;
        session.engaged = true;
        try {
          session.target.setPointerCapture(session.pointerId);
        } catch {
          // Synthetic pointers (tests, automation) have no active pointer.
        }
      }
      session.zone.move(move.clientX, move.clientY);
    };

    const finish = (commit: boolean) => {
      const session = sessionRef.current;
      if (!session) return;
      const resolution = session.zone.end();
      const { engaged, lastRect } = session;
      const aborted = session.zone.aborted();
      teardown();
      if (!commit || aborted || !engaged) return;
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
      if (lastRect === null) return;
      store.getState().commitRect(windowId, lastRect);
      flushPersistence();
    };

    const onPointerUp = (up: PointerEvent) => {
      if (up.pointerId !== sessionRef.current?.pointerId) return;
      finish(true);
    };

    const onPointerCancel = (cancel: PointerEvent) => {
      const session = sessionRef.current;
      if (!session || cancel.pointerId !== session.pointerId) return;
      session.zone.abort();
      finish(false);
    };

    // Capture loss (tab switch, OS interruption) ends the gesture instead of
    // stalling it (use-gesture DragEngine posture, issue #494 class).
    const onLostCapture = () => {
      const session = sessionRef.current;
      if (!session || !session.engaged) return;
      session.zone.abort();
      finish(false);
    };

    const onKeyDown = (key: KeyboardEvent) => {
      const session = sessionRef.current;
      if (!session || key.key !== "Escape") return;
      key.preventDefault();
      key.stopPropagation();
      session.zone.abort();
      session.lastRect = null;
      setDragRect(null);
    };

    const cleanup = () => {
      window.removeEventListener("pointermove", onPointerMove, true);
      window.removeEventListener("pointerup", onPointerUp, true);
      window.removeEventListener("pointercancel", onPointerCancel, true);
      window.removeEventListener("keydown", onKeyDown, true);
      target.removeEventListener("lostpointercapture", onLostCapture);
      try {
        target.releasePointerCapture(pointerId);
      } catch {
        // Capture may never have engaged; mirrored guard.
      }
    };

    sessionRef.current = {
      target,
      pointerId,
      startX: event.clientX,
      startY: event.clientY,
      zone,
      engaged: false,
      lastRect: null,
      cleanup,
    };
    window.addEventListener("pointermove", onPointerMove, true);
    window.addEventListener("pointerup", onPointerUp, true);
    window.addEventListener("pointercancel", onPointerCancel, true);
    window.addEventListener("keydown", onKeyDown, true);
    target.addEventListener("lostpointercapture", onLostCapture);
  };

  return { dragRect, onHeadPointerDown };
}
