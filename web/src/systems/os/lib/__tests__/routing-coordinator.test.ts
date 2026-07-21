import { describe, expect, it } from "vitest";

import { createDesktopStore } from "../../stores/desktop-store";
import { encodeDesktopPayload, encodeWindowPayload } from "../os-state-payloads";
import { RoutingCoordinator, type OsRouterPort } from "../routing-coordinator";
import { OS_DESKTOP_KEY, osWindowKey, type OsStateEntry, type OsWindow } from "../os-types";

interface RecordingPort extends OsRouterPort {
  pushes: Array<{ pathname: string; search: Record<string, unknown> }>;
  replaces: Array<{ pathname: string; search: Record<string, unknown> }>;
}

function createPort(): RecordingPort {
  const port: RecordingPort = {
    pushes: [],
    replaces: [],
    navigate(location) {
      port.pushes.push(location);
    },
    replace(location) {
      port.replaces.push(location);
    },
  };
  return port;
}

function windowEntry(win: OsWindow, seq: number): OsStateEntry {
  return {
    key: osWindowKey(win.id),
    value: encodeWindowPayload(win),
    rev: 1,
    seq,
    deleted: false,
    updated_at: "2026-07-20T00:00:00Z",
  };
}

function makeWindow(overrides: Partial<OsWindow> & Pick<OsWindow, "id" | "app">): OsWindow {
  return {
    instanceKey: null,
    location: { pathname: "/tasks", search: {} },
    rect: { x: 10, y: 10, w: 400, h: 300 },
    prevRect: null,
    z: 1,
    minimized: false,
    maximized: false,
    snap: null,
    ...overrides,
  };
}

describe("routing coordinator", () => {
  it("Should never navigate on route-pop, write exactly one entry per user cause, and let a deep link win focus after hydration (UT-080, invariant 13)", () => {
    const store = createDesktopStore();
    const port = createPort();
    const coordinator = new RoutingCoordinator(store, port);

    // Boot: the initial URL intent is recorded, not applied, while hydrating.
    coordinator.reportRouteMatch({ pathname: "/tasks/t9", search: {} });
    expect(Object.keys(store.getState().windows)).toHaveLength(0);

    // Snapshot restores a settings-focused desktop, then hydration completes:
    // the deep link wins focus, the restored desktop survives, no history write.
    store.getState().hydrate([
      windowEntry(
        makeWindow({
          id: "app:settings",
          app: "settings",
          location: { pathname: "/settings/general", search: {} },
          z: 1,
        }),
        1
      ),
      {
        key: OS_DESKTOP_KEY,
        value: encodeDesktopPayload({
          focusedId: "app:settings",
          railOpen: false,
          wallpaper: "ember",
        }),
        rev: 1,
        seq: 2,
        deleted: false,
        updated_at: "2026-07-20T00:00:00Z",
      },
    ]);
    coordinator.completeHydration();

    const state = store.getState();
    expect(state.windows["app:settings"]).toBeDefined();
    expect(state.windows["app:tasks"].location.pathname).toBe("/tasks/t9");
    expect(state.focusedId).toBe("app:tasks");
    expect(port.pushes).toHaveLength(0);
    expect(port.replaces).toHaveLength(0);

    // route-pop (browser Back to the settings location): reconcile only.
    coordinator.reportRouteMatch({ pathname: "/settings/general", search: {} });
    expect(store.getState().focusedId).toBe("app:settings");
    expect(port.pushes).toHaveLength(0);

    // user-focus writes exactly one history entry.
    coordinator.userFocus("app:tasks");
    expect(store.getState().focusedId).toBe("app:tasks");
    expect(port.pushes).toHaveLength(1);
    expect(port.pushes[0].pathname).toBe("/tasks/t9");
  });

  it("Should true up a neutral boot URL to the restored focus via replace (rule 4)", () => {
    const store = createDesktopStore();
    const port = createPort();
    const coordinator = new RoutingCoordinator(store, port);

    coordinator.reportRouteMatch({ pathname: "/", search: {} });
    store.getState().hydrate([
      windowEntry(
        makeWindow({ id: "app:tasks", app: "tasks", location: { pathname: "/tasks", search: {} } }),
        1
      ),
      {
        key: OS_DESKTOP_KEY,
        value: encodeDesktopPayload({
          focusedId: "app:tasks",
          railOpen: false,
          wallpaper: "ember",
        }),
        rev: 1,
        seq: 2,
        deleted: false,
        updated_at: "2026-07-20T00:00:00Z",
      },
    ]);
    coordinator.completeHydration();

    expect(port.pushes).toHaveLength(0);
    expect(port.replaces).toHaveLength(1);
    expect(port.replaces[0].pathname).toBe("/tasks");
  });

  it("Should keep the desktop empty on a first-run boot at the root (US-001.EC-1)", () => {
    const store = createDesktopStore();
    const port = createPort();
    const coordinator = new RoutingCoordinator(store, port);

    coordinator.reportRouteMatch({ pathname: "/", search: {} });
    store.getState().hydrate([]);
    coordinator.completeHydration();

    expect(Object.keys(store.getState().windows)).toHaveLength(0);
    expect(port.pushes).toHaveLength(0);
    expect(port.replaces).toHaveLength(0);
  });

  it("Should open then navigate exactly once for user-open, and reconcile idempotently", () => {
    const store = createDesktopStore();
    const port = createPort();
    const coordinator = new RoutingCoordinator(store, port);
    store.getState().hydrate([]);
    coordinator.completeHydration();

    coordinator.userOpen({ app: "tasks" });
    expect(port.pushes).toHaveLength(1);
    expect(port.pushes[0].pathname).toBe("/tasks");
    const stateBeforeRouteMatch = store.getState();
    const zBeforeRouteMatch = stateBeforeRouteMatch.windows["app:tasks"].z;

    // The resulting route match performs no store transition and therefore
    // cannot interrupt an in-flight window gesture (rule 3).
    coordinator.reportRouteMatch({ pathname: "/tasks", search: {} });
    expect(port.pushes).toHaveLength(1);
    expect(Object.keys(store.getState().windows)).toEqual(["app:tasks"]);
    expect(store.getState()).toBe(stateBeforeRouteMatch);
    expect(store.getState().windows["app:tasks"].z).toBe(zBeforeRouteMatch);
  });

  it("Should follow close/minimize with one navigation to the successor or the desktop", () => {
    const store = createDesktopStore();
    const port = createPort();
    const coordinator = new RoutingCoordinator(store, port);
    store.getState().hydrate([]);
    coordinator.completeHydration();

    coordinator.userOpen({ app: "tasks" });
    coordinator.userOpen({ app: "settings" });
    port.pushes.length = 0;

    coordinator.userClose("app:settings");
    expect(store.getState().focusedId).toBe("app:tasks");
    expect(port.pushes).toHaveLength(1);
    expect(port.pushes[0].pathname).toBe("/tasks");

    coordinator.userMinimize("app:tasks");
    expect(store.getState().focusedId).toBeNull();
    expect(port.pushes).toHaveLength(2);
    expect(port.pushes[1].pathname).toBe("/");
  });

  it("Should skip the focus navigation when activation came through a link (rule 3 coalescing)", () => {
    const store = createDesktopStore();
    const port = createPort();
    const coordinator = new RoutingCoordinator(store, port);
    store.getState().hydrate([]);
    coordinator.completeHydration();

    coordinator.userOpen({ app: "tasks" });
    coordinator.userOpen({ app: "settings" });
    port.pushes.length = 0;

    coordinator.userFocus("app:tasks", { viaLink: true });
    expect(store.getState().focusedId).toBe("app:tasks");
    expect(port.pushes).toHaveLength(0);
  });
});
