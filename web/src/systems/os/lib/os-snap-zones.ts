import type { OsRect, OsSnapZone, OsSnapZoneId } from "./os-types";

/**
 * Pure zone resolver (ADR-009 v2, puter-style bands): pointer position against
 * the work-area rect snapshotted at drag start. Side bands arm halves; corner
 * quarters are reachable from EITHER adjacent edge within the corner reach, so
 * corner targets are 150×32 strips instead of tiny squares. Emits at most ONE
 * zone per resolution. The center of the top edge alone resolves nothing
 * (zoom owns "fill the desktop"), and the bottom-center strip stays unbound
 * (the dock lives there).
 */

/** Pointer band depth (px) along each desktop edge that arms snap zones. */
export const OS_SNAP_EDGE_BAND = 32;

/** Corner reach (px) along an edge — how far from the corner a quarter arms. */
export const OS_SNAP_CORNER_REACH = 150;

/** Extra slack (px) beyond the band before the ACTIVE zone releases (hysteresis). */
export const OS_SNAP_EXIT_SLACK = 16;

/** Movement (px) below which a head press stays a click (Hermes drag threshold). */
export const OS_SNAP_DRAG_THRESHOLD = 4;

/**
 * The zone catalog: fractions of the work area (`_techspec.md` §Window Snap
 * Layer). `top`/`bottom` are preset-only (zoom menu + palette) — the drag
 * resolver never emits them (top edge unbound; bottom center owns the dock).
 */
export const OS_SNAP_ZONES: Record<OsSnapZoneId, OsSnapZone> = {
  left: { fx: 0, fy: 0, fw: 0.5, fh: 1 },
  right: { fx: 0.5, fy: 0, fw: 0.5, fh: 1 },
  top: { fx: 0, fy: 0, fw: 1, fh: 0.5 },
  bottom: { fx: 0, fy: 0.5, fw: 1, fh: 0.5 },
  "top-left": { fx: 0, fy: 0, fw: 0.5, fh: 0.5 },
  "top-right": { fx: 0.5, fy: 0, fw: 0.5, fh: 0.5 },
  "bottom-left": { fx: 0, fy: 0.5, fw: 0.5, fh: 0.5 },
  "bottom-right": { fx: 0.5, fy: 0.5, fw: 0.5, fh: 0.5 },
};

export interface OsSnapPoint {
  x: number;
  y: number;
}

function zoneAt(
  point: OsSnapPoint,
  area: OsRect,
  band: number,
  reach: number
): OsSnapZoneId | null {
  const nearLeft = point.x <= area.x + band;
  const nearRight = point.x >= area.x + area.w - band;
  const nearTop = point.y <= area.y + band;
  const nearBottom = point.y >= area.y + area.h - band;
  // Degenerate work areas can put one point inside both bands; the closer
  // edge wins so the resolver stays total and single-valued.
  const side =
    nearLeft && nearRight
      ? point.x - area.x <= area.x + area.w - point.x
        ? "left"
        : "right"
      : nearLeft
        ? "left"
        : nearRight
          ? "right"
          : null;
  if (side !== null) {
    if (point.y <= area.y + reach) return `top-${side}`;
    if (point.y >= area.y + area.h - reach) return `bottom-${side}`;
    return side;
  }
  const vertical =
    nearTop && nearBottom
      ? point.y - area.y <= area.y + area.h - point.y
        ? "top"
        : "bottom"
      : nearTop
        ? "top"
        : nearBottom
          ? "bottom"
          : null;
  if (vertical !== null) {
    const withinLeftReach = point.x <= area.x + reach;
    const withinRightReach = point.x >= area.x + area.w - reach;
    if (withinLeftReach && withinRightReach) {
      return point.x - area.x <= area.x + area.w - point.x
        ? `${vertical}-left`
        : `${vertical}-right`;
    }
    if (withinLeftReach) return `${vertical}-left`;
    if (withinRightReach) return `${vertical}-right`;
    // Center strips stay unbound: top belongs to zoom, bottom to the dock.
    return null;
  }
  return null;
}

/**
 * Resolves the zone the pointer targets, or `null` away from every edge.
 * `area` is the work-area rect in the same coordinate space as `point`
 * (win-layer coordinates), snapshotted once at drag start. Passing the
 * currently active zone as `current` adds exit hysteresis: the active zone
 * only releases once the pointer leaves the band + `OS_SNAP_EXIT_SLACK`,
 * killing flicker at the band boundary; switching zones stays immediate.
 */
export function resolveSnapZone(
  point: OsSnapPoint,
  area: OsRect,
  current: OsSnapZoneId | null = null
): OsSnapZoneId | null {
  const zone = zoneAt(point, area, OS_SNAP_EDGE_BAND, OS_SNAP_CORNER_REACH);
  if (zone !== null) return zone;
  if (current !== null) {
    const kept = zoneAt(
      point,
      area,
      OS_SNAP_EDGE_BAND + OS_SNAP_EXIT_SLACK,
      OS_SNAP_CORNER_REACH + OS_SNAP_EXIT_SLACK
    );
    if (kept === current) return current;
  }
  return null;
}
