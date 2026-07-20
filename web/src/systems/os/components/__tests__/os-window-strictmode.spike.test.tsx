import { StrictMode } from "react";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { OsShellContext, type OsShellHandle } from "../../contexts/os-shell-context";
import { RoutingCoordinator, type OsRouterPort } from "../../lib/routing-coordinator";
import { createDesktopStore } from "../../stores/desktop-store";
import { OsWindow } from "../os-window";

/**
 * ADR-003 spike assertion: react-rnd (controlled) under React 19 StrictMode.
 * StrictMode double-invokes mount effects; older drag libraries broke on the
 * remount (findDOMNode, stale listeners). The spike pins: (1) mount/remount
 * renders without errors, (2) a drag gesture commits exactly one rect to the
 * store at gesture end, (3) the controlled position round-trips.
 * Failure here triggers the ADR-003 fallback (custom geometry / use-gesture).
 */
describe("react-rnd StrictMode spike (ADR-003)", () => {
  it("Should mount, drag, and commit under StrictMode with no console errors", async () => {
    const errors: unknown[][] = [];
    const consoleError = vi.spyOn(console, "error").mockImplementation((...args) => {
      errors.push(args);
    });

    const store = createDesktopStore();
    const port: OsRouterPort = { navigate: () => {}, replace: () => {} };
    const coordinator = new RoutingCoordinator(store, port);
    store.getState().hydrate([]);
    coordinator.completeHydration();
    coordinator.userOpen({ app: "tasks" });
    const flushPersistence = vi.fn();
    const shell: OsShellHandle = { store, coordinator, flushPersistence };

    render(
      <StrictMode>
        <OsShellContext.Provider value={shell}>
          <div style={{ position: "relative", width: 1440, height: 900 }}>
            <OsWindow windowId="app:tasks" rootCrumb="agh" />
          </div>
        </OsShellContext.Provider>
      </StrictMode>
    );
    await screen.findByTestId("os-pending-app");

    const rectBefore = store.getState().windows["app:tasks"].rect;
    const handle = document.querySelector(".os-window-drag-handle") as HTMLElement;
    expect(handle).not.toBeNull();

    // One drag gesture: down → move → up. react-draggable listens on mouse
    // events. jsdom has no layout engine, so `bounds="parent"` clamps the
    // gesture to (0,0) — the spike asserts the gesture LIFECYCLE (drag starts,
    // gesture end commits once, controlled position round-trips) rather than
    // pixel math; real geometry is exercised by the browser E2E journeys.
    fireEvent.mouseDown(handle, { clientX: 300, clientY: 100 });
    fireEvent.mouseMove(handle, { clientX: 360, clientY: 140 });
    fireEvent.mouseUp(handle, { clientX: 360, clientY: 140 });

    await waitFor(() => {
      const rect = store.getState().windows["app:tasks"].rect;
      expect({ x: rect.x, y: rect.y }).not.toEqual({ x: rectBefore.x, y: rectBefore.y });
    });
    expect(flushPersistence).toHaveBeenCalledTimes(1);

    const rndErrors = errors.filter(args =>
      args.some(arg => typeof arg === "string" && /findDOMNode|Warning|error/i.test(arg))
    );
    expect(rndErrors).toEqual([]);
    consoleError.mockRestore();
  });
});
