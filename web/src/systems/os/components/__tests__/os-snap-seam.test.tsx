// Suite: linked seam handle (component + keyboard gesture)
// Invariant: a seam handle renders between adjacent snapped windows with
// separator semantics, and arrow keys on the focused handle move the shared
// boundary by the step (Shift ×5), committing BOTH windows' fractions within
// the min-window limits. The keyboard path is the guaranteed a11y path; the
// pointer journey lives in e2e (real capture semantics).
// Boundary IN: seam layer + keyboard gesture against the real store.
// Boundary OUT: pure derivation (os-snap-seams suite), pointer capture (e2e).
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { OsShellContext, type OsShellHandle } from "../../contexts/os-shell-context";
import { OS_SNAP_ZONES } from "../../lib/os-snap-zones";
import { RoutingCoordinator, type OsRouterPort } from "../../lib/routing-coordinator";
import { createDesktopStore } from "../../stores/desktop-store";
import { OsSnapSeamLayer } from "../os-snap-seam";

function createHarness() {
  const store = createDesktopStore();
  const port: OsRouterPort = { navigate: () => {}, replace: () => {} };
  const coordinator = new RoutingCoordinator(store, port);
  store.getState().hydrate([]);
  coordinator.completeHydration();
  const shell: OsShellHandle = { store, coordinator, flushPersistence: () => {} };
  return { store, shell };
}

function snapPair(store: ReturnType<typeof createHarness>["store"]) {
  const left = store.getState().openOrFocus({ app: "tasks" });
  const right = store.getState().openOrFocus({ app: "vault" });
  store.getState().clampToViewport({ width: 1440, height: 900 });
  store.getState().snapWindow(left, OS_SNAP_ZONES.left);
  store.getState().snapWindow(right, OS_SNAP_ZONES.right);
  return { left, right };
}

describe("OsSnapSeamLayer", () => {
  it("Should render a separator handle in the gutter between snapped halves", () => {
    const { store, shell } = createHarness();
    snapPair(store);
    render(
      <OsShellContext.Provider value={shell}>
        <OsSnapSeamLayer />
      </OsShellContext.Provider>
    );
    const handle = screen.getByRole("separator");
    expect(handle).toHaveAttribute("aria-orientation", "vertical");
    expect(handle).toHaveAttribute("aria-valuenow", "50");
    expect(handle).toHaveAttribute("tabindex", "0");
  });

  it("Should move the shared boundary with arrows and commit both windows' fractions", () => {
    const { store, shell } = createHarness();
    const { left, right } = snapPair(store);
    render(
      <OsShellContext.Provider value={shell}>
        <OsSnapSeamLayer />
      </OsShellContext.Provider>
    );

    fireEvent.keyDown(screen.getByRole("separator"), { key: "ArrowRight" });
    let leftWin = store.getState().windows[left];
    let rightWin = store.getState().windows[right];
    expect(leftWin.snap?.fw).toBeCloseTo(0.52, 5);
    expect(rightWin.snap?.fx).toBeCloseTo(0.52, 5);
    expect(rightWin.snap?.fw).toBeCloseTo(0.48, 5);
    // Both stay snapped — the seam never detaches its neighbors.
    expect(leftWin.snap).not.toBeNull();
    expect(rightWin.snap).not.toBeNull();

    // Shift multiplies the nudge (use-gesture keyboard factor pattern).
    fireEvent.keyDown(screen.getByRole("separator"), { key: "ArrowRight", shiftKey: true });
    rightWin = store.getState().windows[right];
    expect(rightWin.snap?.fx).toBeCloseTo(0.62, 5);

    // The boundary clamps so each side keeps the minimum window (+ gutter).
    for (let i = 0; i < 40; i += 1) {
      fireEvent.keyDown(screen.getByRole("separator"), { key: "ArrowLeft", shiftKey: true });
    }
    leftWin = store.getState().windows[left];
    rightWin = store.getState().windows[right];
    const minFraction = (280 + 8) / 1420;
    expect((leftWin.snap?.fw ?? 0) + 1e-9).toBeGreaterThanOrEqual(minFraction);
    expect(rightWin.snap?.fx).toBeCloseTo(leftWin.snap?.fw ?? 0, 5);
  });
});
