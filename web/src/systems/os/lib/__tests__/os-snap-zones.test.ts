// Suite: pure snap-zone resolver
// Invariant: pointer positions inside the 20px sensitivity radius of the
// left/right edges and each corner resolve exactly one zone; center and
// top-edge-only positions resolve none (ADR-009; zoom owns "fill").
// Boundary IN: pure math over the drag-start work-area snapshot.
// Boundary OUT: hint publishing/overlay (session + component), store writes.
import { describe, expect, it } from "vitest";

import { resolveSnapZone } from "../os-snap-zones";
import type { OsRect } from "../os-types";

const AREA: OsRect = { x: 10, y: 8, w: 1420, h: 814 };

describe("resolveSnapZone", () => {
  it("Should resolve halves inside the edge bands and none at the center (UT-100)", () => {
    expect(resolveSnapZone({ x: AREA.x + 20, y: 400 }, AREA)).toBe("left");
    expect(resolveSnapZone({ x: AREA.x + AREA.w - 20, y: 400 }, AREA)).toBe("right");
    // Points past the work-area edge (over the gutter) still capture.
    expect(resolveSnapZone({ x: 0, y: 400 }, AREA)).toBe("left");
    expect(resolveSnapZone({ x: AREA.x + AREA.w + 15, y: 400 }, AREA)).toBe("right");
    // One pixel beyond the radius releases the capture.
    expect(resolveSnapZone({ x: AREA.x + 21, y: 400 }, AREA)).toBeNull();
    expect(resolveSnapZone({ x: 700, y: 400 }, AREA)).toBeNull();
  });

  it("Should resolve each corner quarter where both edge bands overlap (UT-100)", () => {
    expect(resolveSnapZone({ x: AREA.x + 5, y: AREA.y + 5 }, AREA)).toBe("top-left");
    expect(resolveSnapZone({ x: AREA.x + AREA.w - 5, y: AREA.y + 5 }, AREA)).toBe("top-right");
    expect(resolveSnapZone({ x: AREA.x + 5, y: AREA.y + AREA.h - 5 }, AREA)).toBe("bottom-left");
    expect(resolveSnapZone({ x: AREA.x + AREA.w - 5, y: AREA.y + AREA.h - 5 }, AREA)).toBe(
      "bottom-right"
    );
  });

  it("Should leave the top edge unbound — zoom owns fill (UT-100)", () => {
    expect(resolveSnapZone({ x: 700, y: AREA.y + 2 }, AREA)).toBeNull();
    expect(resolveSnapZone({ x: 700, y: AREA.y + AREA.h - 2 }, AREA)).toBeNull();
  });

  it("Should publish exactly one target at seams — corner outranks half (UT-100)", () => {
    // Inside both the left band and the top band: the corner wins, never both.
    const seam = resolveSnapZone({ x: AREA.x + 20, y: AREA.y + 20 }, AREA);
    expect(seam).toBe("top-left");
    // Exactly on the corner-band boundary: one past the vertical band falls
    // back to the half — a single deterministic target either way.
    expect(resolveSnapZone({ x: AREA.x + 20, y: AREA.y + 21 }, AREA)).toBe("left");
  });
});
