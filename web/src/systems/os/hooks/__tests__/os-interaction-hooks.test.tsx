// Suite: OS interaction hooks
// Invariant: shell shortcuts and zoom hover/dispatch honor editable targets and live client state.
// Owning layer: the current useOsShortcuts/useOsZoomMenu interaction boundary.
import { act, fireEvent, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { OsShellContext, type OsShellHandle } from "../../contexts/os-shell-context";
import { RoutingCoordinator } from "../../lib/routing-coordinator";
import type {
  OsDesktopRuntimeStore,
  OsWindow,
  WindowManagerController,
  WindowManagerCommandOutcome,
} from "../../lib/os-types";
import type { WindowManagerConfig, WindowManagerSnapshot } from "../../lib/window-manager-types";
import {
  OS_ZOOM_MENU_CLOSE_GRACE_MS,
  OS_ZOOM_MENU_OPEN_DELAY_MS,
  useOsZoomMenu,
} from "../use-os-zoom-menu";
import { useOsShortcuts, type OsShortcutHandlers } from "../use-os-shortcuts";

const CONFIG: WindowManagerConfig = {
  newWindowPolicy: "floating",
  smallViewportPolicy: "stack",
  focusPolicy: "click_directional",
  focusWrap: true,
  focusFollowsPointer: false,
  raiseOnFocus: true,
  dragAwayPolicy: "window",
  groupMoveModifier: "alt",
  swapModifier: "shift",
  historyLimit: 50,
  desktopTransition: "slide",
  gaps: { inner: 8, top: 0, right: 0, bottom: 0, left: 0 },
  snap: {
    edgeBand: 32,
    cornerReach: 150,
    exitSlack: 16,
    repeatRatios: [0.5, 0.666667, 0.333333],
  },
  bindings: { topCenter: "zoom", bottomCenter: "reserved" },
  shortcuts: {},
};

const SNAPSHOT: WindowManagerSnapshot = {
  version: 1,
  workspaceId: "workspace:test",
  revision: 7,
  desktops: [],
  windows: {},
  history: { undo: [], redo: [] },
  overrides: {},
  updatedAt: "2026-07-22T00:00:00Z",
};

function windowFixture(id: string, layer: number): OsWindow {
  return {
    id,
    app: "tasks",
    instanceKey: null,
    route: { pathname: "/tasks", search: {} },
    desktopId: "desktop:main",
    placement: "floating",
    rect: { x: 40 * layer, y: 40, w: 600, h: 420 },
    layer,
    minimized: false,
    groupId: null,
    nodeId: null,
    stackId: null,
    stackActive: true,
    parentAxis: null,
  };
}

function acceptedOutcome(): WindowManagerCommandOutcome {
  return { accepted: true, completion: Promise.resolve(true) };
}

function createShell({ live = true, withPeer = true } = {}) {
  const primary = windowFixture("window:primary", 2);
  const peer = windowFixture("window:peer", 1);
  const zoomWindow = vi.fn(() => acceptedOutcome());
  const toggleFloating = vi.fn();
  const arrangeLayout = vi.fn();
  const state: OsDesktopRuntimeStore = {
    snapshot: SNAPSHOT,
    windowManagerConfig: CONFIG,
    client: {
      workspaceId: "workspace:test",
      clientId: "client:web",
      activeDesktopId: "desktop:main",
      focusedWindowId: primary.id,
      focusOrder: [primary.id],
      connectedAt: "2026-07-22T00:00:00Z",
      presentationRevision: 1,
    },
    desktops: [],
    projections: {},
    windows: withPeer ? { [primary.id]: primary, [peer.id]: peer } : { [primary.id]: primary },
    activeDesktopId: "desktop:main",
    focusedId: primary.id,
    railCollapsedAgentIds: [],
    wallpaper: "ember",
    reduceMotion: false,
    dockMagnify: true,
    presentation: "floating",
    viewportState: "ready",
    hydration: "live",
    connectionStatus: live ? "connected" : "disconnected",
    desktopBounds: { width: 1280, height: 800, origin: { x: 0, y: 0 } },
    openOrFocus: vi.fn(target => ({
      windowId: `app:${target.app}`,
      ...acceptedOutcome(),
    })),
    closeWindow: vi.fn(async () => true),
    focusWindow: vi.fn(() => acceptedOutcome()),
    minimizeWindow: vi.fn(async () => true),
    restoreWindow: vi.fn(),
    zoomWindow,
    toggleFloating,
    moveWindow: vi.fn(),
    arrangeLayout,
    commitFloatingRect: vi.fn(),
    resizeLayout: vi.fn(),
    balanceLayout: vi.fn(),
    navigateWindow: vi.fn(() => acceptedOutcome()),
    toggleRailGroup: vi.fn(),
    setWallpaper: vi.fn(),
    setDockMagnify: vi.fn(),
    setReduceMotion: vi.fn(),
    setDesktopBounds: vi.fn(),
  };
  const controller: WindowManagerController = {
    getState: () => state,
    getInitialState: () => state,
    subscribe: () => () => {},
    bind: vi.fn(),
    unbind: vi.fn(),
    setClient: vi.fn(),
    setConnectionStatus: vi.fn(),
    setLoadError: vi.fn(),
    createDesktop: vi.fn(),
    renameDesktop: vi.fn(),
    reorderDesktop: vi.fn(),
    switchDesktop: vi.fn(),
    switchDesktopDirection: vi.fn(),
    deleteDesktop: vi.fn(),
    moveWindowToDesktop: vi.fn(),
    tileWindow: vi.fn(),
    applySnapTarget: vi.fn(),
    focusDirection: vi.fn(),
    undoLayout: vi.fn(),
    redoLayout: vi.fn(),
    balanceFocusedLayout: vi.fn(),
    clearConflict: vi.fn(),
    refreshSnapshot: vi.fn(),
  };
  const coordinator = new RoutingCoordinator(controller, {
    navigate: vi.fn(),
    replace: vi.fn(),
  });
  const shell: OsShellHandle = { store: controller, manager: controller, coordinator };
  const wrapper = ({ children }: { children: ReactNode }) => (
    <OsShellContext.Provider value={shell}>{children}</OsShellContext.Provider>
  );
  return { controller, state, wrapper };
}

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe("useOsShortcuts", () => {
  it("Should route shell shortcuts independently of window-manager availability", () => {
    const { wrapper } = createShell({ live: false });
    const handlers: OsShortcutHandlers = {
      onPalette: vi.fn(),
      onNewSession: vi.fn(),
      onDesktops: vi.fn(),
      onEscape: vi.fn(),
    };
    renderHook(() => useOsShortcuts(handlers), { wrapper });

    fireEvent.keyDown(document, { key: "k", code: "KeyK", metaKey: true });
    fireEvent.keyDown(document, { key: "n", code: "KeyN", metaKey: true });
    fireEvent.keyDown(document, { key: "Escape", code: "Escape" });

    expect(handlers.onPalette).toHaveBeenCalledOnce();
    expect(handlers.onNewSession).toHaveBeenCalledOnce();
    expect(handlers.onEscape).toHaveBeenCalledOnce();
  });

  it("Should dispatch window-manager chords only for a live client outside editable targets", () => {
    const liveShell = createShell();
    const unavailableShell = createShell({ live: false });
    const handlers: OsShortcutHandlers = {
      onPalette: vi.fn(),
      onNewSession: vi.fn(),
      onDesktops: vi.fn(),
      onEscape: vi.fn(),
    };
    const unavailable = renderHook(() => useOsShortcuts(handlers), {
      wrapper: unavailableShell.wrapper,
    });

    fireEvent.keyDown(document, { key: "z", code: "KeyZ", metaKey: true });
    expect(unavailableShell.controller.undoLayout).not.toHaveBeenCalled();
    unavailable.unmount();

    renderHook(() => useOsShortcuts(handlers), { wrapper: liveShell.wrapper });
    const input = document.createElement("input");
    document.body.append(input);
    fireEvent.keyDown(input, { key: "z", code: "KeyZ", metaKey: true });
    expect(liveShell.controller.undoLayout).not.toHaveBeenCalled();

    fireEvent.keyDown(document, { key: "z", code: "KeyZ", metaKey: true });
    expect(liveShell.controller.undoLayout).toHaveBeenCalledOnce();
    input.remove();
  });
});

describe("useOsZoomMenu", () => {
  it("Should apply mouse-only hover intent and preserve the close grace period", () => {
    vi.useFakeTimers();
    const { wrapper } = createShell();
    const { result } = renderHook(() => useOsZoomMenu("window:primary"), { wrapper });
    const mouseEvent = { pointerType: "mouse" } as Parameters<
      typeof result.current.onHoverEnter
    >[0];
    const touchEvent = { pointerType: "touch" } as Parameters<
      typeof result.current.onHoverEnter
    >[0];

    act(() => result.current.onHoverEnter(touchEvent));
    act(() => vi.advanceTimersByTime(OS_ZOOM_MENU_OPEN_DELAY_MS));
    expect(result.current.open).toBe(false);

    act(() => result.current.onHoverEnter(mouseEvent));
    act(() => vi.advanceTimersByTime(OS_ZOOM_MENU_OPEN_DELAY_MS - 1));
    expect(result.current.open).toBe(false);
    act(() => vi.advanceTimersByTime(1));
    expect(result.current.open).toBe(true);

    act(() => result.current.onHoverLeave());
    act(() => vi.advanceTimersByTime(OS_ZOOM_MENU_CLOSE_GRACE_MS - 1));
    expect(result.current.open).toBe(true);
    act(() => result.current.onContentEnter());
    act(() => vi.advanceTimersByTime(1));
    expect(result.current.open).toBe(true);
  });

  it("Should gate placement dispatch on live availability and close after an accepted action", () => {
    const liveShell = createShell();
    const unavailableShell = createShell({ live: false });
    const live = renderHook(() => useOsZoomMenu("window:primary"), {
      wrapper: liveShell.wrapper,
    });
    const unavailable = renderHook(() => useOsZoomMenu("window:primary"), {
      wrapper: unavailableShell.wrapper,
    });

    act(() => live.result.current.onOpenChange(true));
    act(() =>
      live.result.current.dispatchPlacement({
        id: "window.tile.left",
        placement: "left",
        label: "Tile left half",
      })
    );
    expect(liveShell.controller.tileWindow).toHaveBeenCalledWith("window:primary", "left");
    expect(live.result.current.open).toBe(false);

    act(() => unavailable.result.current.dispatchFill());
    expect(unavailableShell.state.zoomWindow).not.toHaveBeenCalled();
  });
});
