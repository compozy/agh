// Suite: pure snap-zone resolver
// Invariant: side bands (32px) arm halves; corner quarters arm from EITHER
// adjacent edge within the 150px corner reach; the active zone releases only
// past band + exit slack (hysteresis); top/bottom center strips resolve none
// (zoom owns "fill", the dock owns the bottom). Exactly one zone per point.
// Boundary IN: pure math over the drag-start work-area snapshot.
// Boundary OUT: hint publishing/overlay (session + component), store writes.
import { describe, expect, it } from "vitest";

import {
  OS_SNAP_CORNER_REACH,
  OS_SNAP_EDGE_BAND,
  OS_SNAP_EXIT_SLACK,
  resolveSnapZone,
} from "../os-snap-zones";
import type { OsRect } from "../os-types";

const AREA: OsRect = { x: 10, y: 8, w: 1420, h: 814 };
const BAND = OS_SNAP_EDGE_BAND;
const REACH = OS_SNAP_CORNER_REACH;

describe("resolveSnapZone", () => {
  it("Should resolve halves inside the edge bands and none at the center (UT-100)", () => {
    expect(resolveSnapZone({ x: AREA.x + BAND, y: 400 }, AREA)).toBe("left");
    expect(resolveSnapZone({ x: AREA.x + AREA.w - BAND, y: 400 }, AREA)).toBe("right");
    // Points past the work-area edge (over the gutter) still capture.
    expect(resolveSnapZone({ x: 0, y: 400 }, AREA)).toBe("left");
    expect(resolveSnapZone({ x: AREA.x + AREA.w + 15, y: 400 }, AREA)).toBe("right");
    // One pixel beyond the band releases the capture (no active zone yet).
    expect(resolveSnapZone({ x: AREA.x + BAND + 1, y: 400 }, AREA)).toBeNull();
    expect(resolveSnapZone({ x: 700, y: 400 }, AREA)).toBeNull();
  });

  it("Should arm corner quarters along the full corner reach of either edge (UT-100)", () => {
    // Side band + within reach of the top → corner, not half.
    expect(resolveSnapZone({ x: AREA.x + 5, y: AREA.y + REACH }, AREA)).toBe("top-left");
    expect(resolveSnapZone({ x: AREA.x + 5, y: AREA.y + REACH + 1 }, AREA)).toBe("left");
    expect(resolveSnapZone({ x: AREA.x + AREA.w - 5, y: AREA.y + AREA.h - REACH }, AREA)).toBe(
      "bottom-right"
    );
    // Top/bottom bands arm the same corners within reach of a side edge.
    expect(resolveSnapZone({ x: AREA.x + REACH, y: AREA.y + 5 }, AREA)).toBe("top-left");
    expect(resolveSnapZone({ x: AREA.x + AREA.w - REACH, y: AREA.y + 5 }, AREA)).toBe("top-right");
    expect(resolveSnapZone({ x: AREA.x + REACH, y: AREA.y + AREA.h - 5 }, AREA)).toBe(
      "bottom-left"
    );
  });

  it("Should leave the top and bottom center strips unbound (UT-100)", () => {
    // Zoom owns "fill the desktop"; the dock lives on the bottom strip.
    expect(resolveSnapZone({ x: 700, y: AREA.y + 2 }, AREA)).toBeNull();
    expect(resolveSnapZone({ x: 700, y: AREA.y + AREA.h - 2 }, AREA)).toBeNull();
  });

  it("Should publish exactly one target at seams — corner outranks half (UT-100)", () => {
    const seam = resolveSnapZone({ x: AREA.x + BAND, y: AREA.y + BAND }, AREA);
    expect(seam).toBe("top-left");
  });

  it("Should hold the active zone through the exit slack and release beyond it (UT-100)", () => {
    const justOutside = { x: AREA.x + BAND + OS_SNAP_EXIT_SLACK, y: 400 };
    const pastSlack = { x: AREA.x + BAND + OS_SNAP_EXIT_SLACK + 1, y: 400 };
    // Without an active zone the same point resolves nothing…
    expect(resolveSnapZone(justOutside, AREA)).toBeNull();
    // …but an active "left" survives inside band + slack (no flicker),
    expect(resolveSnapZone(justOutside, AREA, "left")).toBe("left");
    // and releases once the pointer truly leaves.
    expect(resolveSnapZone(pastSlack, AREA, "left")).toBeNull();
    // Switching to a different zone never waits on hysteresis.
    expect(resolveSnapZone({ x: AREA.x + 5, y: 400 }, AREA, "right")).toBe("left");
  });
});
