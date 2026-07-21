// Suite: zoom-menu markup + open path (component)
// Invariant: hover intent opens the menu with every section rendered inside
// its Base UI Menu.Group (a GroupLabel outside a Group throws — regression
// caught live 2026-07-21), zone items dispatch fractions against THIS window
// through the real store, and arrange items disable without a second visible
// window. Timing/dispatch semantics live in the use-os-zoom-menu hook suite.
// Boundary IN: rendered menu markup + hover-open path against the real store.
// Boundary OUT: hover timing math (hook suite), Radix/Base positioning px.
import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { OsShellContext, type OsShellHandle } from "../../contexts/os-shell-context";
import { OS_SNAP_ZONES } from "../../lib/os-snap-zones";
import { RoutingCoordinator, type OsRouterPort } from "../../lib/routing-coordinator";
import { createDesktopStore } from "../../stores/desktop-store";
import { OS_ZOOM_MENU_OPEN_DELAY_MS } from "../../hooks/use-os-zoom-menu";
import { OsZoomMenu } from "../os-zoom-menu";

function createHarness() {
  const store = createDesktopStore();
  const port: OsRouterPort = { navigate: () => {}, replace: () => {} };
  const coordinator = new RoutingCoordinator(store, port);
  store.getState().hydrate([]);
  coordinator.completeHydration();
  const shell: OsShellHandle = { store, coordinator, flushPersistence: () => {} };
  return { store, shell };
}

function renderMenu(shell: OsShellHandle, windowId: string) {
  return render(
    <OsShellContext.Provider value={shell}>
      <OsZoomMenu windowId={windowId}>
        <button type="button" data-action="zoom" aria-label="Zoom window" />
      </OsZoomMenu>
    </OsShellContext.Provider>
  );
}

function hoverOpen(container: HTMLElement) {
  const anchor = container.querySelector('[data-slot="os-zoom-menu-anchor"]');
  if (!(anchor instanceof HTMLElement)) throw new Error("zoom-menu anchor must render");
  fireEvent.pointerOver(anchor, { pointerType: "mouse" });
  act(() => {
    vi.advanceTimersByTime(OS_ZOOM_MENU_OPEN_DELAY_MS);
  });
}

describe("OsZoomMenu", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("Should open on hover with both sections rendered and dispatch a zone against this window", () => {
    const { store, shell } = createHarness();
    const id = store.getState().openOrFocus({ app: "tasks" });
    store.getState().openOrFocus({ app: "vault" });
    store.getState().clampToViewport({ width: 1440, height: 900 });

    const { container } = renderMenu(shell, id);
    hoverOpen(container);

    // Both group labels render (GroupLabel requires an enclosing Group —
    // the missing-context crash regression this suite exists for).
    expect(screen.getByTestId("os-zoom-menu")).toBeInTheDocument();
    expect(screen.getByText("Move & Resize")).toBeInTheDocument();
    expect(screen.getByText("Fill & Arrange")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("os-zoom-menu-left"));
    expect(store.getState().windows[id].snap).toEqual(OS_SNAP_ZONES.left);
  });

  it("Should disable arrange presets without a second visible window", () => {
    const { store, shell } = createHarness();
    const id = store.getState().openOrFocus({ app: "tasks" });
    store.getState().clampToViewport({ width: 1440, height: 900 });

    const { container } = renderMenu(shell, id);
    hoverOpen(container);

    expect(screen.getByTestId("os-zoom-menu-two-up")).toHaveAttribute("data-disabled");
    expect(screen.getByTestId("os-zoom-menu-grid")).toHaveAttribute("data-disabled");
    // Fill stays available — it only needs this window.
    expect(screen.getByTestId("os-zoom-menu-fill")).not.toHaveAttribute("data-disabled");
  });
});
