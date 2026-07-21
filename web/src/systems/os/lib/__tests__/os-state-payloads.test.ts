// Suite: os_shell payload codec
// Invariant: `win:*` docs round-trip losslessly and invalid `snap` salvages to
// null without dropping the window (invariant 19; US-001.EC-2 posture).
// Boundary IN: encode/decode of desktop-state entries.
// Boundary OUT: store semantics (desktop-store suite) and wire transport.
import { describe, expect, it } from "vitest";

import { decodeWindowEntry, encodeWindowPayload } from "../os-state-payloads";
import type { OsStateEntry, OsWindow } from "../os-types";

const SNAPPED_WINDOW: OsWindow = {
  id: "app:tasks",
  app: "tasks",
  instanceKey: null,
  location: { pathname: "/tasks", search: {} },
  rect: { x: 10, y: 8, w: 710, h: 814 },
  prevRect: { x: 40, y: 30, w: 500, h: 380 },
  z: 3,
  minimized: false,
  maximized: false,
  snap: { fx: 0, fy: 0, fw: 0.5, fh: 1 },
};

function entryFor(value: Record<string, unknown> | null): OsStateEntry {
  return {
    key: "win:app:tasks",
    value,
    rev: 1,
    seq: 1,
    deleted: false,
    updated_at: "2026-07-20T00:00:00Z",
  };
}

describe("os-state payload codec", () => {
  it("Should round-trip a window doc with snap fractions intact (UT-099)", () => {
    const decoded = decodeWindowEntry(entryFor(encodeWindowPayload(SNAPPED_WINDOW)));
    expect(decoded).toEqual(SNAPPED_WINDOW);
  });

  it("Should salvage invalid snap to null while keeping the window (UT-099)", () => {
    const invalidSnaps: unknown[] = [
      { fx: 1.2, fy: 0, fw: 0.5, fh: 1 }, // out-of-range origin
      { fx: 0, fy: 0, fw: -0.5, fh: 1 }, // negative size
      { fx: 0, fy: 0, fw: 0.04, fh: 1 }, // sub-minimum span (< 10% per axis)
      { fx: 0.8, fy: 0, fw: 0.5, fh: 1 }, // overflow past the work area
      "left-half", // wrong shape entirely
    ];
    for (const snap of invalidSnaps) {
      const decoded = decodeWindowEntry(entryFor({ ...encodeWindowPayload(SNAPPED_WINDOW), snap }));
      expect(decoded).not.toBeNull();
      expect(decoded?.snap).toBeNull();
      expect(decoded?.rect).toEqual(SNAPPED_WINDOW.rect);
    }
  });

  it("Should decode an absent snap field as null (UT-099)", () => {
    const payload: Record<string, unknown> = { ...encodeWindowPayload(SNAPPED_WINDOW) };
    delete payload.snap;
    const decoded = decodeWindowEntry(entryFor(payload));
    expect(decoded).not.toBeNull();
    expect(decoded?.snap).toBeNull();
  });

  it("Should keep maximized and drop snap when a doc claims both derived states (invariant 19)", () => {
    const decoded = decodeWindowEntry(
      entryFor({ ...encodeWindowPayload(SNAPPED_WINDOW), maximized: true })
    );
    expect(decoded?.maximized).toBe(true);
    expect(decoded?.snap).toBeNull();
  });
});
