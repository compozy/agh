import type { OsSnapPoint } from "./os-snap-zones";
import {
  OS_SNAP_GUTTER,
  OS_SNAP_MIN_FRACTION,
  OS_WINDOW_MIN_HEIGHT,
  OS_WINDOW_MIN_WIDTH,
  type OsRect,
  type OsSnapZone,
  type OsSplitSide,
} from "./os-types";

/**
 * Pure window-target resolver: while a drag hovers an existing window (and no
 * desktop-edge band armed — the session enforces that precedence), the
 * directional third of the target under the pointer offers a split — the
 * dragged window takes that half of the TARGET's footprint, the target keeps
 * the other, both snapped. Targets and splits that would violate window
 * minimums resolve nothing.
 */

/** Directional band: the outer third of the target arms a split side. */
const SPLIT_BAND = 1 / 3;

export interface OsWindowTargetCandidate {
  id: string;
  /** Visual rect in win-layer px (derived for snapped, committed for floating). */
  rect: OsRect;
  z: number;
}

export interface OsWindowTargetHit {
  targetId: string;
  side: OsSplitSide;
  /** Claimed half of the target's rect (px), gutter-split, for the overlay. */
  rect: OsRect;
}

/** The half of `rect` a drop on `side` claims, with the gutter split evenly. */
export function claimedHalf(rect: OsRect, side: OsSplitSide): OsRect {
  if (side === "left" || side === "right") {
    const w = Math.round((rect.w - OS_SNAP_GUTTER) / 2);
    const x = side === "left" ? rect.x : rect.x + rect.w - w;
    return { x, y: rect.y, w, h: rect.h };
  }
  const h = Math.round((rect.h - OS_SNAP_GUTTER) / 2);
  const y = side === "top" ? rect.y : rect.y + rect.h - h;
  return { x: rect.x, y, w: rect.w, h };
}

export function resolveWindowTarget(
  point: OsSnapPoint,
  candidates: OsWindowTargetCandidate[]
): OsWindowTargetHit | null {
  let top: OsWindowTargetCandidate | null = null;
  for (const candidate of candidates) {
    const { rect } = candidate;
    const inside =
      point.x >= rect.x &&
      point.x <= rect.x + rect.w &&
      point.y >= rect.y &&
      point.y <= rect.y + rect.h;
    if (!inside) continue;
    if (top === null || candidate.z > top.z) top = candidate;
  }
  if (top === null) return null;
  const rx = (point.x - top.rect.x) / Math.max(top.rect.w, 1);
  const ry = (point.y - top.rect.y) / Math.max(top.rect.h, 1);
  const side: OsSplitSide | null =
    rx <= SPLIT_BAND
      ? "left"
      : rx >= 1 - SPLIT_BAND
        ? "right"
        : ry <= SPLIT_BAND
          ? "top"
          : ry >= 1 - SPLIT_BAND
            ? "bottom"
            : null;
  if (side === null) return null;
  // Suppress splits whose halves could not hold a minimum window.
  if (side === "left" || side === "right") {
    if (top.rect.w / 2 < OS_WINDOW_MIN_WIDTH + OS_SNAP_GUTTER) return null;
  } else if (top.rect.h / 2 < OS_WINDOW_MIN_HEIGHT + OS_SNAP_GUTTER) {
    return null;
  }
  return { targetId: top.id, side, rect: claimedHalf(top.rect, side) };
}

/**
 * Splits a zone in fraction space: `dragged` takes the `side` half, `target`
 * keeps the complement. `null` when a half would break the codec floor.
 */
export function splitZoneBy(
  zone: OsSnapZone,
  side: OsSplitSide
): { dragged: OsSnapZone; target: OsSnapZone } | null {
  if (side === "left" || side === "right") {
    const fw = zone.fw / 2;
    if (fw < OS_SNAP_MIN_FRACTION) return null;
    const first: OsSnapZone = { fx: zone.fx, fy: zone.fy, fw, fh: zone.fh };
    const second: OsSnapZone = { fx: zone.fx + fw, fy: zone.fy, fw: zone.fw - fw, fh: zone.fh };
    return side === "left"
      ? { dragged: first, target: second }
      : { dragged: second, target: first };
  }
  const fh = zone.fh / 2;
  if (fh < OS_SNAP_MIN_FRACTION) return null;
  const first: OsSnapZone = { fx: zone.fx, fy: zone.fy, fw: zone.fw, fh };
  const second: OsSnapZone = { fx: zone.fx, fy: zone.fy + fh, fw: zone.fw, fh: zone.fh - fh };
  return side === "top" ? { dragged: first, target: second } : { dragged: second, target: first };
}
