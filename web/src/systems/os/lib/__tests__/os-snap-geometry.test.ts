// Suite: pure snap geometry (derive + inverse)
// Invariant: derived rects inset inner edges by half the gutter so adjacent
// zones meet across exactly OS_SNAP_GUTTER while outer edges stay flush with
// the work area; `rectToSnapZone` is the inverse of `deriveSnapRect` within
// 1px rounding and always emits codec-valid fractions (≥ min, fx+fw ≤ 1).
// Boundary IN: pure math over bounds + fractions/rects.
// Boundary OUT: store semantics (desktop-store suite), codec validation
// (os-state-payloads suite).
import { describe, expect, it } from "vitest";

import { deriveSnapRect, rectToSnapZone, snapWorkArea } from "../os-snap-geometry";
import { OS_SNAP_ZONES } from "../os-snap-zones";
import {
  OS_SNAP_GUTTER,
  OS_SNAP_MIN_FRACTION,
  type OsDesktopBounds,
  type OsRect,
} from "../os-types";

const BOUNDS: OsDesktopBounds = { width: 1440, height: 900 };

function nearlyEqualRect(a: OsRect, b: OsRect, tolerance = 1): void {
  expect(Math.abs(a.x - b.x)).toBeLessThanOrEqual(tolerance);
  expect(Math.abs(a.y - b.y)).toBeLessThanOrEqual(tolerance);
  expect(Math.abs(a.w - b.w)).toBeLessThanOrEqual(tolerance);
  expect(Math.abs(a.h - b.h)).toBeLessThanOrEqual(tolerance);
}

describe("snap geometry", () => {
  it("Should separate adjacent halves by exactly the gutter with flush outer edges", () => {
    const area = snapWorkArea(BOUNDS);
    const left = deriveSnapRect(OS_SNAP_ZONES.left, BOUNDS);
    const right = deriveSnapRect(OS_SNAP_ZONES.right, BOUNDS);
    // Outer edges stay on the work-area boundary (no double inset).
    expect(left.x).toBe(area.x);
    expect(right.x + right.w).toBe(area.x + area.w);
    expect(left.y).toBe(area.y);
    expect(left.h).toBe(area.h);
    // The seam between them is the full gutter.
    expect(right.x - (left.x + left.w)).toBe(OS_SNAP_GUTTER);
  });

  it("Should separate stacked quarters by exactly the gutter on the vertical seam", () => {
    const top = deriveSnapRect(OS_SNAP_ZONES["top-left"], BOUNDS);
    const bottom = deriveSnapRect(OS_SNAP_ZONES["bottom-left"], BOUNDS);
    expect(bottom.y - (top.y + top.h)).toBe(OS_SNAP_GUTTER);
  });

  it("Should derive the full-area zone as the exact work area (zero inner edges)", () => {
    const area = snapWorkArea(BOUNDS);
    expect(deriveSnapRect({ fx: 0, fy: 0, fw: 1, fh: 1 }, BOUNDS)).toEqual(area);
  });

  it("Should invert derived rects back to fractions that re-derive identically", () => {
    const zones = [
      OS_SNAP_ZONES.left,
      OS_SNAP_ZONES["bottom-right"],
      { fx: 0.25, fy: 0.2, fw: 0.4, fh: 0.6 },
    ];
    for (const zone of zones) {
      const rect = deriveSnapRect(zone, BOUNDS);
      const inverted = rectToSnapZone(rect, BOUNDS);
      // The inverse is stable: re-deriving reproduces the same px rect.
      expect(deriveSnapRect(inverted, BOUNDS)).toEqual(rect);
    }
  });

  it("Should map a user-resized rect to fractions reproducing it within 1px", () => {
    // A snapped left half narrowed from its own edge (resize-in-place).
    const resized: OsRect = { x: 10, y: 8, w: 512, h: 814 };
    const zone = rectToSnapZone(resized, BOUNDS);
    nearlyEqualRect(deriveSnapRect(zone, BOUNDS), resized);
    // Edges within half a gutter of the boundary count as ON it.
    expect(zone.fx).toBe(0);
    expect(zone.fy).toBe(0);
    expect(zone.fy + zone.fh).toBe(1);
  });

  it("Should clamp inverse fractions to the codec floor and span ceiling", () => {
    const tiny = rectToSnapZone({ x: 500, y: 400, w: 10, h: 10 }, BOUNDS);
    expect(tiny.fw).toBeGreaterThanOrEqual(OS_SNAP_MIN_FRACTION);
    expect(tiny.fh).toBeGreaterThanOrEqual(OS_SNAP_MIN_FRACTION);
    const overflow = rectToSnapZone({ x: 1400, y: 850, w: 900, h: 600 }, BOUNDS);
    expect(overflow.fx + overflow.fw).toBeLessThanOrEqual(1);
    expect(overflow.fy + overflow.fh).toBeLessThanOrEqual(1);
  });

  it("Should clamp derived quarters to window minimums on short viewports", () => {
    const clamped = deriveSnapRect(OS_SNAP_ZONES["bottom-right"], { width: 500, height: 400 });
    expect(clamped.w).toBe(280);
    expect(clamped.h).toBe(180);
  });
});
