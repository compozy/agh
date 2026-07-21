// Suite: pure seam derivation
// Invariant: a seam exists exactly where two snapped, visible windows meet in
// fraction space with enough shared span — the handle rect is the gutter
// strip over the overlap, `value` is the boundary fraction, and preview
// overrides move the seam without touching committed state. No seam for
// floating, minimized, or merely-diagonal neighbors.
// Boundary IN: pure math over windows + bounds (+ preview overrides).
// Boundary OUT: gesture mechanics (use-os-seam-drag), store commits.
import { describe, expect, it } from "vitest";

import { deriveSnapRect } from "../os-snap-geometry";
import { deriveSnapSeams } from "../os-snap-seams";
import { OS_SNAP_ZONES } from "../os-snap-zones";
import { OS_SNAP_GUTTER, type OsDesktopBounds, type OsSnapZone, type OsWindow } from "../os-types";

const BOUNDS: OsDesktopBounds = { width: 1440, height: 900 };

function makeWindow(
  id: string,
  snap: OsSnapZone | null,
  overrides: Partial<OsWindow> = {}
): OsWindow {
  return {
    id,
    app: "tasks",
    instanceKey: null,
    location: { pathname: "/", search: {} },
    rect: { x: 10, y: 10, w: 400, h: 300 },
    prevRect: null,
    z: 1,
    minimized: false,
    maximized: false,
    snap,
    ...overrides,
  };
}

describe("deriveSnapSeams", () => {
  it("Should derive one vertical seam spanning the gutter between adjacent halves", () => {
    const left = makeWindow("app:tasks", OS_SNAP_ZONES.left, { z: 3 });
    const right = makeWindow("app:vault", OS_SNAP_ZONES.right, { z: 5 });
    const seams = deriveSnapSeams([left, right], BOUNDS);
    expect(seams).toHaveLength(1);
    const seam = seams[0];
    expect(seam.orientation).toBe("vertical");
    expect(seam.aId).toBe("app:tasks");
    expect(seam.bId).toBe("app:vault");
    expect(seam.value).toBe(0.5);
    expect(seam.z).toBe(6);
    const leftRect = deriveSnapRect(OS_SNAP_ZONES.left, BOUNDS);
    const rightRect = deriveSnapRect(OS_SNAP_ZONES.right, BOUNDS);
    expect(seam.rect.x).toBe(leftRect.x + leftRect.w);
    expect(seam.rect.w).toBe(OS_SNAP_GUTTER);
    expect(seam.rect.w).toBe(rightRect.x - (leftRect.x + leftRect.w));
    expect(seam.rect.y).toBe(leftRect.y);
    expect(seam.rect.h).toBe(leftRect.h);
  });

  it("Should derive pairwise seams for a half beside stacked quarters", () => {
    const seams = deriveSnapSeams(
      [
        makeWindow("app:tasks", OS_SNAP_ZONES.left),
        makeWindow("app:vault", OS_SNAP_ZONES["top-right"]),
        makeWindow("app:sandbox", OS_SNAP_ZONES["bottom-right"]),
      ],
      BOUNDS
    );
    const vertical = seams.filter(seam => seam.orientation === "vertical");
    const horizontal = seams.filter(seam => seam.orientation === "horizontal");
    // The half touches each quarter across the center gutter; the quarters
    // stack across the horizontal one.
    expect(vertical).toHaveLength(2);
    expect(horizontal).toHaveLength(1);
    expect(horizontal[0].aId).toBe("app:vault");
    expect(horizontal[0].bId).toBe("app:sandbox");
  });

  it("Should ignore floating, minimized, and diagonal neighbors", () => {
    expect(
      deriveSnapSeams(
        [makeWindow("app:tasks", OS_SNAP_ZONES.left), makeWindow("app:vault", null)],
        BOUNDS
      )
    ).toHaveLength(0);
    expect(
      deriveSnapSeams(
        [
          makeWindow("app:tasks", OS_SNAP_ZONES.left),
          makeWindow("app:vault", OS_SNAP_ZONES.right, { minimized: true }),
        ],
        BOUNDS
      )
    ).toHaveLength(0);
    // Diagonal quarters share a corner, not a span — no seam.
    expect(
      deriveSnapSeams(
        [
          makeWindow("app:tasks", OS_SNAP_ZONES["top-left"]),
          makeWindow("app:vault", OS_SNAP_ZONES["bottom-right"]),
        ],
        BOUNDS
      )
    ).toHaveLength(0);
  });

  it("Should track the live boundary through preview overrides", () => {
    const left = makeWindow("app:tasks", OS_SNAP_ZONES.left);
    const right = makeWindow("app:vault", OS_SNAP_ZONES.right);
    const preview = {
      "app:tasks": { fx: 0, fy: 0, fw: 0.6, fh: 1 },
      "app:vault": { fx: 0.6, fy: 0, fw: 0.4, fh: 1 },
    };
    const seams = deriveSnapSeams([left, right], BOUNDS, preview);
    expect(seams).toHaveLength(1);
    expect(seams[0].value).toBe(0.6);
    const previewLeft = deriveSnapRect(preview["app:tasks"], BOUNDS);
    expect(seams[0].rect.x).toBe(previewLeft.x + previewLeft.w);
  });
});
