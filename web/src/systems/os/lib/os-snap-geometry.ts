import {
  OS_WINDOW_MIN_HEIGHT,
  OS_WINDOW_MIN_WIDTH,
  OS_WORK_AREA_INSETS,
  type OsDesktopBounds,
  type OsRect,
  type OsSnapZone,
} from "./os-types";

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
 * Work area × fractions, edge-rounded so adjacent zones tile without seams,
 * clamped to window minimums (fraction overflow accepted on short viewports).
 */
export function deriveSnapRect(zone: OsSnapZone, bounds: OsDesktopBounds): OsRect {
  const area = snapWorkArea(bounds);
  const x = area.x + Math.round(area.w * zone.fx);
  const y = area.y + Math.round(area.h * zone.fy);
  const right = area.x + Math.round(area.w * Math.min(zone.fx + zone.fw, 1));
  const bottom = area.y + Math.round(area.h * Math.min(zone.fy + zone.fh, 1));
  return {
    x,
    y,
    w: Math.max(right - x, OS_WINDOW_MIN_WIDTH),
    h: Math.max(bottom - y, OS_WINDOW_MIN_HEIGHT),
  };
}
