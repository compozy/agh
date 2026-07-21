import { useEffect, useRef } from "react";
import type { KeyboardEvent as ReactKeyboardEvent, PointerEvent as ReactPointerEvent } from "react";

import { snapWorkArea } from "../lib/os-snap-geometry";
import type { OsSnapSeam } from "../lib/os-snap-seams";
import {
  OS_SNAP_GUTTER,
  OS_SNAP_MIN_FRACTION,
  OS_WINDOW_MIN_HEIGHT,
  OS_WINDOW_MIN_WIDTH,
  type OsSnapZone,
} from "../lib/os-types";
import { useOsShell } from "./use-os-shell";

/** Keyboard nudge per arrow press, as a work-area fraction (Shift ×5). */
export const OS_SEAM_KEY_STEP = 0.02;

/**
 * Linked seam gesture (Windows JointResize posture): dragging the gutter
 * between two snapped windows moves their shared boundary — per-frame updates
 * land in the runtime `seamPreview` (zero persistence churn) and release
 * commits BOTH windows' fractions through `snapWindow`. Pointer discipline
 * ports the use-gesture DragEngine patterns (`.resources/use-gesture`,
 * engines/DragEngine.ts): single-pointer identity guard, `pointercancel` and
 * `lostpointercapture` end the gesture instead of stalling it, and capture
 * release is `hasPointerCapture`-guarded.
 */

interface SeamAxis {
  /** Boundary floor/ceiling as work-area fractions (min window + gutter kept). */
  min: number;
  max: number;
  /** Work-area px along the drag axis. */
  size: number;
  /** Work-area origin along the drag axis (win-layer coordinates). */
  origin: number;
}

interface SeamSession {
  seam: OsSnapSeam;
  target: HTMLElement;
  pointerId: number;
  startClient: number;
  axis: SeamAxis;
  aZone: OsSnapZone;
  bZone: OsSnapZone;
  preview: { a: OsSnapZone; b: OsSnapZone } | null;
  raf: number;
  pendingClient: number | null;
  cleanup(): void;
}

function splitZones(
  seam: OsSnapSeam,
  aZone: OsSnapZone,
  bZone: OsSnapZone,
  value: number
): { a: OsSnapZone; b: OsSnapZone } {
  if (seam.orientation === "vertical") {
    return {
      a: { ...aZone, fw: value - aZone.fx },
      b: { ...bZone, fx: value, fw: bZone.fx + bZone.fw - value },
    };
  }
  return {
    a: { ...aZone, fh: value - aZone.fy },
    b: { ...bZone, fy: value, fh: bZone.fy + bZone.fh - value },
  };
}

export interface OsSeamDragModel {
  onSeamPointerDown(seam: OsSnapSeam, event: ReactPointerEvent<HTMLElement>): void;
  onSeamKeyDown(seam: OsSnapSeam, event: ReactKeyboardEvent<HTMLElement>): void;
}

export function useOsSeamDrag(): OsSeamDragModel {
  const { store, flushPersistence } = useOsShell();
  const sessionRef = useRef<SeamSession | null>(null);

  useEffect(() => {
    return () => {
      const session = sessionRef.current;
      if (!session) return;
      sessionRef.current = null;
      session.cleanup();
      store.getState().setSeamPreview(null);
    };
  }, [store]);

  const seamAxis = (seam: OsSnapSeam, aZone: OsSnapZone, bZone: OsSnapZone): SeamAxis | null => {
    const bounds = store.getState().desktopBounds;
    if (bounds === null) return null;
    const area = snapWorkArea(bounds);
    const vertical = seam.orientation === "vertical";
    const size = vertical ? area.w : area.h;
    const minPx = (vertical ? OS_WINDOW_MIN_WIDTH : OS_WINDOW_MIN_HEIGHT) + OS_SNAP_GUTTER;
    const floor = Math.max(OS_SNAP_MIN_FRACTION, minPx / size);
    const min = (vertical ? aZone.fx : aZone.fy) + floor;
    const max = (vertical ? bZone.fx + bZone.fw : bZone.fy + bZone.fh) - floor;
    if (min >= max) return null;
    return { min, max, size, origin: vertical ? area.x : area.y };
  };

  const readZones = (seam: OsSnapSeam): { a: OsSnapZone; b: OsSnapZone } | null => {
    const windows = store.getState().windows;
    const a = windows[seam.aId]?.snap;
    const b = windows[seam.bId]?.snap;
    return a && b ? { a, b } : null;
  };

  const commit = (seam: OsSnapSeam, zones: { a: OsSnapZone; b: OsSnapZone }) => {
    const state = store.getState();
    state.snapWindow(seam.aId, zones.a);
    state.snapWindow(seam.bId, zones.b);
    flushPersistence();
  };

  const onSeamPointerDown = (seam: OsSnapSeam, event: ReactPointerEvent<HTMLElement>) => {
    // Single-pointer discipline: a second pointer never restarts the gesture.
    if (event.button !== 0 || sessionRef.current !== null) return;
    const zones = readZones(seam);
    if (zones === null) return;
    const axis = seamAxis(seam, zones.a, zones.b);
    if (axis === null) return;

    const target = event.currentTarget;
    const pointerId = event.pointerId;
    const vertical = seam.orientation === "vertical";
    event.preventDefault();
    try {
      target.setPointerCapture(pointerId);
    } catch {
      // Synthetic pointers (tests, automation) have no active pointer.
    }

    const flush = () => {
      const session = sessionRef.current;
      if (!session) return;
      session.raf = 0;
      if (session.pendingClient === null) return;
      const delta = session.pendingClient - session.startClient;
      session.pendingClient = null;
      const startValue = Math.min(Math.max(seam.value, session.axis.min), session.axis.max);
      const value = Math.min(
        Math.max(startValue + delta / session.axis.size, session.axis.min),
        session.axis.max
      );
      session.preview = splitZones(session.seam, session.aZone, session.bZone, value);
      store
        .getState()
        .setSeamPreview({ [seam.aId]: session.preview.a, [seam.bId]: session.preview.b });
    };

    const finish = (commitGesture: boolean) => {
      const session = sessionRef.current;
      if (!session) return;
      sessionRef.current = null;
      session.cleanup();
      const preview = session.preview;
      store.getState().setSeamPreview(null);
      if (commitGesture && preview !== null) commit(session.seam, preview);
    };

    const onPointerMove = (move: PointerEvent) => {
      const session = sessionRef.current;
      if (!session || move.pointerId !== session.pointerId) return;
      session.pendingClient = vertical ? move.clientX : move.clientY;
      if (session.raf === 0) {
        session.raf = requestAnimationFrame(flush);
      }
    };

    const onPointerUp = (up: PointerEvent) => {
      if (up.pointerId !== sessionRef.current?.pointerId) return;
      flush();
      finish(true);
    };

    // A canceled or capture-lost pointer ends the gesture with geometry
    // unchanged — never a stalled seam (use-gesture issue #494 class).
    const onPointerCancel = (cancel: PointerEvent) => {
      if (cancel.pointerId !== sessionRef.current?.pointerId) return;
      finish(false);
    };

    const onLostCapture = () => finish(false);

    const onKeyDown = (key: KeyboardEvent) => {
      if (key.key !== "Escape") return;
      key.preventDefault();
      key.stopPropagation();
      finish(false);
    };

    const cleanup = () => {
      window.removeEventListener("pointermove", onPointerMove, true);
      window.removeEventListener("pointerup", onPointerUp, true);
      window.removeEventListener("pointercancel", onPointerCancel, true);
      window.removeEventListener("keydown", onKeyDown, true);
      target.removeEventListener("lostpointercapture", onLostCapture);
      const session = sessionRef.current;
      if (session?.raf) cancelAnimationFrame(session.raf);
      try {
        if (target.hasPointerCapture(pointerId)) target.releasePointerCapture(pointerId);
      } catch {
        // Capture may never have engaged; mirrored guard.
      }
    };

    sessionRef.current = {
      seam,
      target,
      pointerId,
      startClient: vertical ? event.clientX : event.clientY,
      axis,
      aZone: zones.a,
      bZone: zones.b,
      preview: null,
      raf: 0,
      pendingClient: null,
      cleanup,
    };
    window.addEventListener("pointermove", onPointerMove, true);
    window.addEventListener("pointerup", onPointerUp, true);
    window.addEventListener("pointercancel", onPointerCancel, true);
    window.addEventListener("keydown", onKeyDown, true);
    target.addEventListener("lostpointercapture", onLostCapture);
  };

  const onSeamKeyDown = (seam: OsSnapSeam, event: ReactKeyboardEvent<HTMLElement>) => {
    const vertical = seam.orientation === "vertical";
    const decrease = vertical ? "ArrowLeft" : "ArrowUp";
    const increase = vertical ? "ArrowRight" : "ArrowDown";
    if (event.key !== decrease && event.key !== increase) return;
    const zones = readZones(seam);
    if (zones === null) return;
    const axis = seamAxis(seam, zones.a, zones.b);
    if (axis === null) return;
    event.preventDefault();
    const factor = event.shiftKey ? 5 : 1;
    const step = OS_SEAM_KEY_STEP * factor * (event.key === increase ? 1 : -1);
    const value = Math.min(Math.max(seam.value + step, axis.min), axis.max);
    commit(seam, splitZones(seam, zones.a, zones.b, value));
  };

  return { onSeamPointerDown, onSeamKeyDown };
}
