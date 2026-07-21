// Suite: zoom-menu behavior hook
// Invariant: hover intent opens after OS_ZOOM_MENU_OPEN_DELAY_MS for mouse
// pointers only and closes after the grace period unless the content is
// entered; every dispatch targets THIS window (snap fractions, fill = zoom,
// arrange presets), closing the menu; arrange gating requires a second
// visible window and restore gating follows snapped state.
// Boundary IN: hook behavior against the real store (fake timers).
// Boundary OUT: menu markup/glyphs (Storybook story + e2e journey), Radix
// positioning.
import { act, renderHook } from "@testing-library/react";
import type { PointerEvent as ReactPointerEvent, ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { OsShellContext, type OsShellHandle } from "../../contexts/os-shell-context";
import { OS_SNAP_COMMANDS } from "../../lib/os-snap-commands";
import { OS_SNAP_ZONES } from "../../lib/os-snap-zones";
import { RoutingCoordinator, type OsRouterPort } from "../../lib/routing-coordinator";
import { createDesktopStore } from "../../stores/desktop-store";
import {
  OS_ZOOM_MENU_CLOSE_GRACE_MS,
  OS_ZOOM_MENU_OPEN_DELAY_MS,
  useOsZoomMenu,
} from "../use-os-zoom-menu";

function createHarness() {
  const store = createDesktopStore();
  const port: OsRouterPort = { navigate: () => {}, replace: () => {} };
  const coordinator = new RoutingCoordinator(store, port);
  store.getState().hydrate([]);
  coordinator.completeHydration();
  const shell: OsShellHandle = { store, coordinator, flushPersistence: () => {} };
  const wrapper = ({ children }: { children: ReactNode }) => (
    <OsShellContext.Provider value={shell}>{children}</OsShellContext.Provider>
  );
  return { store, wrapper };
}

function pointer(pointerType: string): ReactPointerEvent<HTMLElement> {
  return { pointerType } as ReactPointerEvent<HTMLElement>;
}

describe("useOsZoomMenu", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("Should open on mouse hover intent only and close after the grace period", () => {
    const { store, wrapper } = createHarness();
    const id = store.getState().openOrFocus({ app: "tasks" });
    const { result } = renderHook(() => useOsZoomMenu(id), { wrapper });

    // Touch pointers never hover-open.
    act(() => result.current.onHoverEnter(pointer("touch")));
    act(() => vi.advanceTimersByTime(OS_ZOOM_MENU_OPEN_DELAY_MS + 50));
    expect(result.current.open).toBe(false);

    act(() => result.current.onHoverEnter(pointer("mouse")));
    act(() => vi.advanceTimersByTime(OS_ZOOM_MENU_OPEN_DELAY_MS - 1));
    expect(result.current.open).toBe(false);
    act(() => vi.advanceTimersByTime(1));
    expect(result.current.open).toBe(true);

    // Leaving closes only after the grace period…
    act(() => result.current.onHoverLeave());
    act(() => vi.advanceTimersByTime(OS_ZOOM_MENU_CLOSE_GRACE_MS - 1));
    expect(result.current.open).toBe(true);
    act(() => vi.advanceTimersByTime(1));
    expect(result.current.open).toBe(false);

    // …and crossing into the content cancels the pending close.
    act(() => result.current.onHoverEnter(pointer("mouse")));
    act(() => vi.advanceTimersByTime(OS_ZOOM_MENU_OPEN_DELAY_MS));
    act(() => result.current.onHoverLeave());
    act(() => result.current.onContentEnter());
    act(() => vi.advanceTimersByTime(OS_ZOOM_MENU_CLOSE_GRACE_MS + 100));
    expect(result.current.open).toBe(true);
  });

  it("Should dispatch snap, fill, and arrange against this window and close the menu", () => {
    const { store, wrapper } = createHarness();
    const a = store.getState().openOrFocus({ app: "tasks" });
    const b = store.getState().openOrFocus({ app: "vault" });
    store.getState().clampToViewport({ width: 1440, height: 900 });
    const { result } = renderHook(() => useOsZoomMenu(a), { wrapper });

    const left = OS_SNAP_COMMANDS.find(command => command.zoneId === "left");
    act(() => result.current.dispatchSnap(left as NonNullable<typeof left>));
    expect(store.getState().windows[a].snap).toEqual(OS_SNAP_ZONES.left);
    expect(result.current.open).toBe(false);
    expect(result.current.snapped).toBe(true);

    act(() => result.current.dispatchFill());
    expect(store.getState().windows[a].maximized).toBe(true);
    expect(store.getState().windows[a].snap).toBeNull();

    act(() => result.current.dispatchArrange("two-up"));
    expect(store.getState().windows[a].snap).toEqual(OS_SNAP_ZONES.left);
    expect(store.getState().windows[b].snap).toEqual(OS_SNAP_ZONES.right);
  });

  it("Should gate arrange on a second visible window", () => {
    const { store, wrapper } = createHarness();
    const a = store.getState().openOrFocus({ app: "tasks" });
    const { result, rerender } = renderHook(() => useOsZoomMenu(a), { wrapper });
    expect(result.current.arrangeEnabled).toBe(false);

    const b = store.getState().openOrFocus({ app: "vault" });
    rerender();
    expect(result.current.arrangeEnabled).toBe(true);

    act(() => store.getState().minimizeWindow(b));
    expect(result.current.arrangeEnabled).toBe(false);
  });
});
