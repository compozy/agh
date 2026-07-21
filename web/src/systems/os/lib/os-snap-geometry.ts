import {
  OS_SNAP_GUTTER,
  OS_SNAP_MIN_FRACTION,
  OS_WINDOW_MIN_HEIGHT,
  OS_WINDOW_MIN_WIDTH,
  OS_WORK_AREA_INSETS,
  type OsDesktopBounds,
  type OsRect,
  type OsSnapZone,
} from "./os-types";

/** Fraction tolerance treating an edge as ON the work-area boundary. */
const EDGE_EPSILON = 0.001;

/**
 * Derived snap geometry (ADR-009): pure math from the win-layer box to the
 * rendered rect. Each client derives px locally from the persisted fractions;
 * viewport resize re-derives and never commits (invariant 19).
 */

/** The "fill the desktop" box: win-layer bounds minus menubar/dock insets. */
export function snapWorkArea(bounds: OsDesktopBounds): OsRect {
  const { top, right, bottom, left } = OS_WORK_AREA_INSETS;
  return {
    x: left,
    y: top,
    w: Math.max(1, bounds.width - left - right),
    h: Math.max(1, bounds.height - top - bottom),
  };
}

/**
 * Work area × fractions, clamped to window minimums (fraction overflow
 * accepted on short viewports). Inner edges — the ones not on the work-area
 * boundary — inset by `OS_SNAP_GUTTER / 2`, so adjacent zones meet across a
 * full gutter while outer edges keep the work-area insets.
 */
export function deriveSnapRect(zone: OsSnapZone, bounds: OsDesktopBounds): OsRect {
  const area = snapWorkArea(bounds);
  const half = OS_SNAP_GUTTER / 2;
  const fr = Math.min(zone.fx + zone.fw, 1);
  const fb = Math.min(zone.fy + zone.fh, 1);
  const x = area.x + Math.round(area.w * zone.fx) + (zone.fx <= EDGE_EPSILON ? 0 : half);
  const y = area.y + Math.round(area.h * zone.fy) + (zone.fy <= EDGE_EPSILON ? 0 : half);
  const right = area.x + Math.round(area.w * fr) - (fr >= 1 - EDGE_EPSILON ? 0 : half);
  const bottom = area.y + Math.round(area.h * fb) - (fb >= 1 - EDGE_EPSILON ? 0 : half);
  return {
    x,
    y,
    w: Math.max(right - x, OS_WINDOW_MIN_WIDTH),
    h: Math.max(bottom - y, OS_WINDOW_MIN_HEIGHT),
  };
}

function clampFraction(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max);
}

/**
 * Inverse of `deriveSnapRect`: a client-resized px rect back to work-area
 * fractions, un-applying the half-gutter inner insets (edges landing within
 * half a gutter of the boundary count as ON it, so an edge-flush resize maps
 * to exactly 0/1). Clamped so the result always satisfies the codec contract
 * — each axis spans ≥ `OS_SNAP_MIN_FRACTION` and `fx+fw`/`fy+fh` stay ≤ 1.
 * Re-deriving the returned zone reproduces the rect within 1px rounding.
 */
export function rectToSnapZone(rect: OsRect, bounds: OsDesktopBounds): OsSnapZone {
  const area = snapWorkArea(bounds);
  const half = OS_SNAP_GUTTER / 2;
  const left = rect.x - area.x <= half ? 0 : rect.x - half - area.x;
  const top = rect.y - area.y <= half ? 0 : rect.y - half - area.y;
  const rectRight = rect.x + rect.w;
  const rectBottom = rect.y + rect.h;
  const right = area.x + area.w - rectRight <= half ? area.w : rectRight + half - area.x;
  const bottom = area.y + area.h - rectBottom <= half ? area.h : rectBottom + half - area.y;
  const fx = clampFraction(left / area.w, 0, 1 - OS_SNAP_MIN_FRACTION);
  const fy = clampFraction(top / area.h, 0, 1 - OS_SNAP_MIN_FRACTION);
  const fw = clampFraction((right - left) / area.w, OS_SNAP_MIN_FRACTION, 1 - fx);
  const fh = clampFraction((bottom - top) / area.h, OS_SNAP_MIN_FRACTION, 1 - fy);
  return { fx, fy, fw, fh };
}
