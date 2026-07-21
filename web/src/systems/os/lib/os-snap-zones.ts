import type { OsRect, OsSnapZone, OsSnapZoneId } from "./os-types";

/**
 * Pure zone resolver (ADR-009, Hermes/FancyZones provenance): pointer position
 * against the work-area rect snapshotted at drag start. Emits at most ONE zone
 * per resolution — corners outrank halves, so seams never publish dual
 * targets. The top edge alone resolves nothing (zoom owns "fill the desktop").
 */

/** FancyZones `DefaultSensitivityRadius` — capture distance from an edge (px). */
export const OS_SNAP_SENSITIVITY_RADIUS = 20;

/** Movement (px) below which a head press stays a click (Hermes drag threshold). */
export const OS_SNAP_DRAG_THRESHOLD = 4;

/** The v1 zone catalog: fractions of the work area (`_techspec.md` §Window Snap Layer). */
export const OS_SNAP_ZONES: Record<OsSnapZoneId, OsSnapZone> = {
  left: { fx: 0, fy: 0, fw: 0.5, fh: 1 },
  right: { fx: 0.5, fy: 0, fw: 0.5, fh: 1 },
  "top-left": { fx: 0, fy: 0, fw: 0.5, fh: 0.5 },
  "top-right": { fx: 0.5, fy: 0, fw: 0.5, fh: 0.5 },
  "bottom-left": { fx: 0, fy: 0.5, fw: 0.5, fh: 0.5 },
  "bottom-right": { fx: 0.5, fy: 0.5, fw: 0.5, fh: 0.5 },
};

export interface OsSnapPoint {
  x: number;
  y: number;
}

/**
 * Resolves the zone the pointer targets, or `null` away from every edge.
 * `area` is the work-area rect in the same coordinate space as `point`
 * (win-layer coordinates), snapshotted once at drag start.
 */
export function resolveSnapZone(
  point: OsSnapPoint,
  area: OsRect,
  radius: number = OS_SNAP_SENSITIVITY_RADIUS
): OsSnapZoneId | null {
  const nearLeft = point.x <= area.x + radius;
  const nearRight = point.x >= area.x + area.w - radius;
  const nearTop = point.y <= area.y + radius;
  const nearBottom = point.y >= area.y + area.h - radius;
  // Degenerate work areas can put one point inside both bands; the closer
  // edge wins so the resolver stays total and single-valued.
  const horizontal =
    nearLeft && nearRight
      ? point.x - area.x <= area.x + area.w - point.x
        ? "left"
        : "right"
      : nearLeft
        ? "left"
        : nearRight
          ? "right"
          : null;
  if (horizontal === null) return null;
  if (nearTop && !nearBottom) return `top-${horizontal}`;
  if (nearBottom && !nearTop) return `bottom-${horizontal}`;
  return horizontal;
}
