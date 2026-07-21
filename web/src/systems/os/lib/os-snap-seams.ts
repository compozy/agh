import { deriveSnapRect } from "./os-snap-geometry";
import type { OsDesktopBounds, OsRect, OsSnapZone, OsWindow } from "./os-types";

/**
 * Pure seam derivation: two snapped windows whose fractions meet across the
 * gutter form a linked seam — dragging it resizes both (Windows JointResize
 * posture). Seams are EMERGENT from fractions; nothing persists a group, so
 * agent-written arrangements grow seams for free and un-snapping either side
 * dissolves the link.
 */

/** Fraction tolerance for "A's edge meets B's edge" adjacency. */
const SEAM_FRACTION_EPSILON = 0.002;

/** Minimum shared span (px) before a seam is worth a grab target. */
const SEAM_MIN_SPAN = 24;

export interface OsSnapSeam {
  id: string;
  orientation: "vertical" | "horizontal";
  /** Window on the left (vertical) or top (horizontal) side of the seam. */
  aId: string;
  /** Window on the right (vertical) or bottom (horizontal) side. */
  bId: string;
  /** The shared gutter strip in win-layer px (the drag target). */
  rect: OsRect;
  /** Boundary position as a work-area fraction (B's leading edge). */
  value: number;
  /** Stacks just above the pair; floating windows above still win the point. */
  z: number;
}

interface SeamWindow {
  win: OsWindow;
  zone: OsSnapZone;
  rect: OsRect;
}

function seamBetween(a: SeamWindow, b: SeamWindow): Omit<OsSnapSeam, "id" | "z"> | null {
  // Vertical seam: A's right edge meets B's left edge with vertical overlap.
  if (Math.abs(a.zone.fx + a.zone.fw - b.zone.fx) <= SEAM_FRACTION_EPSILON) {
    const top = Math.max(a.rect.y, b.rect.y);
    const bottom = Math.min(a.rect.y + a.rect.h, b.rect.y + b.rect.h);
    const left = a.rect.x + a.rect.w;
    const width = b.rect.x - left;
    if (bottom - top < SEAM_MIN_SPAN || width <= 0) return null;
    return {
      orientation: "vertical",
      aId: a.win.id,
      bId: b.win.id,
      rect: { x: left, y: top, w: width, h: bottom - top },
      value: b.zone.fx,
    };
  }
  // Horizontal seam: A's bottom edge meets B's top edge with horizontal overlap.
  if (Math.abs(a.zone.fy + a.zone.fh - b.zone.fy) <= SEAM_FRACTION_EPSILON) {
    const left = Math.max(a.rect.x, b.rect.x);
    const right = Math.min(a.rect.x + a.rect.w, b.rect.x + b.rect.w);
    const top = a.rect.y + a.rect.h;
    const height = b.rect.y - top;
    if (right - left < SEAM_MIN_SPAN || height <= 0) return null;
    return {
      orientation: "horizontal",
      aId: a.win.id,
      bId: b.win.id,
      rect: { x: left, y: top, w: right - left, h: height },
      value: b.zone.fy,
    };
  }
  return null;
}

/**
 * Derives every live seam among the given windows. Only snapped, visible
 * windows participate; `zoneOverrides` (the seam-drag preview) substitutes a
 * window's committed fractions so the handle tracks the boundary mid-drag.
 */
export function deriveSnapSeams(
  windows: OsWindow[],
  bounds: OsDesktopBounds,
  zoneOverrides: Record<string, OsSnapZone> | null = null
): OsSnapSeam[] {
  const eligible: SeamWindow[] = [];
  for (const win of windows) {
    if (win.minimized) continue;
    const zone = zoneOverrides?.[win.id] ?? win.snap;
    if (zone === null) continue;
    eligible.push({ win, zone, rect: deriveSnapRect(zone, bounds) });
  }
  const seams: OsSnapSeam[] = [];
  for (const a of eligible) {
    for (const b of eligible) {
      if (a.win.id === b.win.id) continue;
      const seam = seamBetween(a, b);
      if (seam === null) continue;
      seams.push({
        ...seam,
        id: `${seam.orientation}:${seam.aId}:${seam.bId}`,
        z: Math.max(a.win.z, b.win.z) + 1,
      });
    }
  }
  return seams;
}
