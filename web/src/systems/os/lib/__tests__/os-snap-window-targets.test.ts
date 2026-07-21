// Suite: pure window-target resolver + fraction splitter
// Invariant: the topmost window under the pointer offers a directional split
// from its outer thirds (center offers none), suppressed when a half could
// not hold a minimum window; `claimedHalf` splits the gutter evenly; and
// `splitZoneBy` divides fractions so both halves stay codec-valid or returns
// null. Desktop-edge precedence lives in the session (e2e journey).
// Boundary IN: pure math over pointer + candidate rects / zones.
// Boundary OUT: candidate snapshots (os-snap-candidates), store commits.
import { describe, expect, it } from "vitest";

import { claimedHalf, resolveWindowTarget, splitZoneBy } from "../os-snap-window-targets";
import { OS_SNAP_GUTTER } from "../os-types";

const TARGET = { id: "app:vault", rect: { x: 100, y: 100, w: 800, h: 600 }, z: 2 };

describe("resolveWindowTarget", () => {
  it("Should resolve the directional third under the pointer on the topmost window", () => {
    const above = { id: "app:tasks", rect: { x: 100, y: 100, w: 800, h: 600 }, z: 5 };
    // Left third → left side of the TOPMOST candidate.
    const hit = resolveWindowTarget({ x: 150, y: 400 }, [TARGET, above]);
    expect(hit?.targetId).toBe("app:tasks");
    expect(hit?.side).toBe("left");
    // Right, top, and bottom thirds map to their sides.
    expect(resolveWindowTarget({ x: 850, y: 400 }, [TARGET])?.side).toBe("right");
    expect(resolveWindowTarget({ x: 500, y: 150 }, [TARGET])?.side).toBe("top");
    expect(resolveWindowTarget({ x: 500, y: 650 }, [TARGET])?.side).toBe("bottom");
    // Dead center arms nothing — plain move stays possible over a window.
    expect(resolveWindowTarget({ x: 500, y: 400 }, [TARGET])).toBeNull();
    // Outside every candidate resolves nothing.
    expect(resolveWindowTarget({ x: 50, y: 50 }, [TARGET])).toBeNull();
  });

  it("Should suppress splits whose halves could not hold a minimum window", () => {
    // 500/2 = 250 < 280 + gutter → horizontal splits suppressed.
    const narrow = { id: "app:vault", rect: { x: 0, y: 0, w: 500, h: 600 }, z: 1 };
    expect(resolveWindowTarget({ x: 50, y: 300 }, [narrow])).toBeNull();
    // 300/2 = 150 < 180 + gutter → vertical splits suppressed.
    const short = { id: "app:vault", rect: { x: 0, y: 0, w: 800, h: 300 }, z: 1 };
    expect(resolveWindowTarget({ x: 400, y: 30 }, [short])).toBeNull();
  });

  it("Should claim halves that split the gutter evenly", () => {
    const left = claimedHalf(TARGET.rect, "left");
    const right = claimedHalf(TARGET.rect, "right");
    expect(left.w).toBe((TARGET.rect.w - OS_SNAP_GUTTER) / 2);
    expect(right.x + right.w).toBe(TARGET.rect.x + TARGET.rect.w);
    expect(right.x - (left.x + left.w)).toBe(OS_SNAP_GUTTER);
    const bottom = claimedHalf(TARGET.rect, "bottom");
    expect(bottom.y + bottom.h).toBe(TARGET.rect.y + TARGET.rect.h);
  });
});

describe("splitZoneBy", () => {
  it("Should divide a zone so dragged takes the side half and target the rest", () => {
    const zone = { fx: 0.5, fy: 0, fw: 0.5, fh: 1 };
    const split = splitZoneBy(zone, "bottom");
    expect(split?.dragged).toEqual({ fx: 0.5, fy: 0.5, fw: 0.5, fh: 0.5 });
    expect(split?.target).toEqual({ fx: 0.5, fy: 0, fw: 0.5, fh: 0.5 });
    const horizontal = splitZoneBy(zone, "left");
    expect(horizontal?.dragged).toEqual({ fx: 0.5, fy: 0, fw: 0.25, fh: 1 });
    expect(horizontal?.target).toEqual({ fx: 0.75, fy: 0, fw: 0.25, fh: 1 });
  });

  it("Should refuse splits that would break the codec floor", () => {
    expect(splitZoneBy({ fx: 0, fy: 0, fw: 0.15, fh: 1 }, "left")).toBeNull();
    expect(splitZoneBy({ fx: 0, fy: 0, fw: 1, fh: 0.19 }, "top")).toBeNull();
  });
});
