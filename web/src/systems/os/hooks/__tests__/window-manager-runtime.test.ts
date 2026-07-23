// Suite: window-manager runtime command admission
// Invariant: presentation intent is retained only for a semantic command accepted by the shared
// command serializer.
// Owning layer: the runtime → interaction-store command boundary.
import { QueryClient } from "@tanstack/react-query";
import { afterEach, describe, expect, it, vi } from "vitest";

import { executeWindowManagerCommand } from "../../adapters/window-manager-api";
import { windowManagerKeys } from "../../lib/window-manager-query";
import type { WindowManagerConfig, WindowManagerSnapshot } from "../../lib/window-manager-types";
import {
  selectDesktopTransitionIntent,
  windowManagerStore,
} from "../../stores/window-manager-store";
import { WindowManagerRuntime } from "../window-manager-runtime";

vi.mock("../../adapters/window-manager-api", async importOriginal => {
  const actual = await importOriginal<typeof import("../../adapters/window-manager-api")>();
  return { ...actual, executeWindowManagerCommand: vi.fn() };
});

const CONFIG: WindowManagerConfig = {
  newWindowPolicy: "floating",
  smallViewportPolicy: "stack",
  focusPolicy: "click_directional",
  focusWrap: true,
  focusFollowsPointer: false,
  raiseOnFocus: true,
  dragAwayPolicy: "window",
  groupMoveModifier: "alt",
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
  desktops: [
    {
      id: "desktop:one",
      name: "One",
      order: 0,
      purpose: "standard",
      focusOwner: null,
      groups: [],
      floating: [],
    },
    {
      id: "desktop:two",
      name: "Two",
      order: 1,
      purpose: "standard",
      focusOwner: null,
      groups: [],
      floating: [],
    },
  ],
  windows: {},
  history: { undo: [], redo: [] },
  overrides: {},
  updatedAt: "2026-07-22T00:00:00Z",
};

afterEach(() => {
  windowManagerStore.getState().actions.unbindClient();
  windowManagerStore.getState().actions.setWorkArea(null);
  vi.clearAllMocks();
});

describe("WindowManagerRuntime", () => {
  it("Should project the Query-owned config without a React effect mirror", () => {
    const queryClient = new QueryClient();
    queryClient.setQueryData(windowManagerKeys.snapshot("workspace:test"), SNAPSHOT);
    const runtime = new WindowManagerRuntime(queryClient);
    runtime.bind({ workspaceId: "workspace:test", clientId: "client:web" });

    expect(runtime.getState().hydration).toBe("pending");

    queryClient.setQueryData(windowManagerKeys.config(), CONFIG);

    expect(runtime.getState().hydration).toBe("live");
    expect(runtime.getState().windowManagerConfig?.historyLimit).toBe(50);
    runtime.destroy();
  });

  it("Should keep gesture samples off the runtime projection channel", () => {
    const runtime = new WindowManagerRuntime(new QueryClient());
    const onProjection = vi.fn();
    const unsubscribe = runtime.subscribe(onProjection);
    const actions = windowManagerStore.getState().actions;

    actions.beginGesture({
      pointerId: 4,
      point: { x: 640, y: 90 },
      workArea: { x: 0, y: 0, w: 1280, h: 800 },
      layoutRevision: 7,
      source: {
        windowId: "window:tasks",
        nodeId: "node:tasks",
        groupId: "group:primary",
        moveMode: "window",
      },
    });
    actions.previewGesture({ x: 320, y: 100 }, null, { x: 0, y: 0, w: 1280, h: 800 });
    actions.previewGesture({ x: 12, y: 120 }, null, { x: 0, y: 0, w: 1280, h: 800 });

    expect(onProjection).not.toHaveBeenCalled();

    actions.setConnectionStatus("connecting");

    expect(onProjection).toHaveBeenCalledOnce();
    unsubscribe();
    runtime.destroy();
  });

  it("Should not retain a desktop transition when another command owns the serializer", () => {
    const queryClient = new QueryClient();
    queryClient.setQueryData(windowManagerKeys.snapshot("workspace:test"), SNAPSHOT);
    queryClient.setQueryData(windowManagerKeys.config(), CONFIG);
    const runtime = new WindowManagerRuntime(queryClient);
    runtime.bind({ workspaceId: "workspace:test", clientId: "client:web" });
    runtime.setClient({
      workspaceId: "workspace:test",
      clientId: "client:web",
      presentationRevision: 1,
      activeDesktopId: "desktop:one",
      focusedWindowId: null,
      focusOrder: [],
      connectedAt: "2026-07-22T00:00:00Z",
    });
    expect(
      windowManagerStore.getState().actions.beginCommand({
        id: "command:pending",
        kind: "window.move",
        expectedRevision: 7,
      })
    ).toBe(true);

    runtime.switchDesktop("desktop:two");

    expect(selectDesktopTransitionIntent(windowManagerStore.getState())).toBeNull();
    expect(executeWindowManagerCommand).not.toHaveBeenCalled();
    runtime.destroy();
  });

  it("Should accept a reset presentation revision after explicit client invalidation", () => {
    const queryClient = new QueryClient();
    queryClient.setQueryData(windowManagerKeys.snapshot("workspace:test"), SNAPSHOT);
    queryClient.setQueryData(windowManagerKeys.config(), CONFIG);
    const runtime = new WindowManagerRuntime(queryClient);
    runtime.bind({ workspaceId: "workspace:test", clientId: "client:web" });
    const view = {
      workspaceId: "workspace:test",
      clientId: "client:web",
      activeDesktopId: "desktop:one",
      focusedWindowId: null,
      focusOrder: [],
      connectedAt: "2026-07-22T00:00:00Z",
    };
    runtime.setClient({ ...view, presentationRevision: 7 });

    runtime.setClient(null);
    runtime.setClient({ ...view, presentationRevision: 1 });
    runtime.setClient({ ...view, presentationRevision: 2, activeDesktopId: "desktop:two" });

    expect(runtime.getState().client).toMatchObject({
      presentationRevision: 2,
      activeDesktopId: "desktop:two",
    });
    runtime.destroy();
  });

  it("Should focus an existing same-route window instead of navigating without a route change", () => {
    const queryClient = new QueryClient();
    const snapshot: WindowManagerSnapshot = {
      ...SNAPSHOT,
      desktops: SNAPSHOT.desktops.map(desktop =>
        desktop.id === "desktop:one" ? { ...desktop, floating: ["app:tasks"] } : desktop
      ),
      windows: {
        "app:tasks": {
          id: "app:tasks",
          app: "tasks",
          instanceKey: null,
          route: {
            pathname: "/tasks",
            search: { filters: { owner: "me", state: "open" }, panel: "activity" },
          },
          placement: "floating",
          desktopId: "desktop:one",
          floatingRect: { x: 0.1, y: 0.1, w: 0.5, h: 0.5 },
          minimized: false,
          returnAnchor: null,
        },
      },
    };
    queryClient.setQueryData(windowManagerKeys.snapshot("workspace:test"), snapshot);
    queryClient.setQueryData(windowManagerKeys.config(), CONFIG);
    vi.mocked(executeWindowManagerCommand).mockReturnValue(
      new Promise<Awaited<ReturnType<typeof executeWindowManagerCommand>>>(() => {})
    );
    const runtime = new WindowManagerRuntime(queryClient);
    runtime.bind({ workspaceId: "workspace:test", clientId: "client:web" });
    runtime.setClient({
      workspaceId: "workspace:test",
      clientId: "client:web",
      presentationRevision: 1,
      activeDesktopId: "desktop:one",
      focusedWindowId: null,
      focusOrder: [],
      connectedAt: "2026-07-22T00:00:00Z",
    });

    runtime.getState().openOrFocus({
      app: "tasks",
      route: {
        pathname: "/tasks",
        search: { panel: "activity", filters: { state: "open", owner: "me" } },
      },
    });

    expect(executeWindowManagerCommand).toHaveBeenCalledOnce();
    expect(executeWindowManagerCommand).toHaveBeenCalledWith("workspace:test", "client:web", 7, {
      commandId: "window.focus",
      payload: { window_id: "app:tasks", direction: "" },
    });
    runtime.destroy();
  });

  it("Should normalize a tile command against the same gap-inset area used by its preview", () => {
    const queryClient = new QueryClient();
    const snapshot: WindowManagerSnapshot = {
      ...SNAPSHOT,
      desktops: SNAPSHOT.desktops.map(desktop =>
        desktop.id === "desktop:one" ? { ...desktop, floating: ["app:tasks"] } : desktop
      ),
      windows: {
        "app:tasks": {
          id: "app:tasks",
          app: "tasks",
          instanceKey: null,
          route: { pathname: "/tasks", search: {} },
          placement: "floating",
          desktopId: "desktop:one",
          floatingRect: { x: 0.1, y: 0.1, w: 0.5, h: 0.5 },
          minimized: false,
          returnAnchor: null,
        },
      },
    };
    const config: WindowManagerConfig = {
      ...CONFIG,
      gaps: { inner: 8, top: 7, right: 10, bottom: 13, left: 10 },
    };
    queryClient.setQueryData(windowManagerKeys.snapshot("workspace:test"), snapshot);
    queryClient.setQueryData(windowManagerKeys.config(), config);
    windowManagerStore.getState().actions.setWorkArea({ rect: { x: 10, y: 20, w: 1200, h: 800 } });
    vi.mocked(executeWindowManagerCommand).mockReturnValue(
      new Promise<Awaited<ReturnType<typeof executeWindowManagerCommand>>>(() => {})
    );
    const runtime = new WindowManagerRuntime(queryClient);
    runtime.bind({ workspaceId: "workspace:test", clientId: "client:web" });
    runtime.setClient({
      workspaceId: "workspace:test",
      clientId: "client:web",
      presentationRevision: 1,
      activeDesktopId: "desktop:one",
      focusedWindowId: "app:tasks",
      focusOrder: ["app:tasks"],
      connectedAt: "2026-07-22T00:00:00Z",
    });

    runtime.tileWindow("app:tasks", "left");

    expect(executeWindowManagerCommand).toHaveBeenCalledWith(
      "workspace:test",
      "client:web",
      7,
      expect.objectContaining({
        commandId: "layout.arrange",
        payload: expect.objectContaining({
          frame: { x: 0, y: 0, width: 0.5, height: 1 },
        }),
      })
    );
    runtime.destroy();
  });
});
