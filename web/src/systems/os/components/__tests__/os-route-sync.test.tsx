import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from "@tanstack/react-router";
import { render, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { OsShellContext, type OsShellHandle } from "../../contexts/os-shell-context";
import { RoutingCoordinator, type OsRouterPort } from "../../lib/routing-coordinator";
import { createDesktopStore } from "../../stores/desktop-store";
import { createOsRouteSync } from "../os-route-sync";

function createHarness(initialPath: string) {
  const store = createDesktopStore();
  const pushes: string[] = [];
  const port: OsRouterPort = {
    navigate: location => pushes.push(location.pathname),
    replace: location => pushes.push(location.pathname),
  };
  const coordinator = new RoutingCoordinator(store, port);
  const shell: OsShellHandle = { store, coordinator, flushPersistence: () => {} };

  const rootRoute = createRootRoute();
  const tasksRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: "/tasks",
    component: createOsRouteSync("tasks"),
  });
  const taskDetailRoute = createRoute({
    getParentRoute: () => tasksRoute,
    path: "$id",
    component: createOsRouteSync("tasks"),
  });
  const router = createRouter({
    routeTree: rootRoute.addChildren([tasksRoute.addChildren([taskDetailRoute])]),
    history: createMemoryHistory({ initialEntries: [initialPath] }),
  });

  return { store, coordinator, shell, router, pushes };
}

describe("createOsRouteSync", () => {
  it("Should report the matched location to the coordinator, land it in the store, and render null (UT-057)", async () => {
    const { store, coordinator, shell, router } = createHarness("/tasks/t9");

    const { container } = render(
      <OsShellContext.Provider value={shell}>
        <RouterProvider router={router as never} />
      </OsShellContext.Provider>
    );

    // Boot order: the sync-controller reports while hydrating; the snapshot
    // applies; hydration completion applies the URL as the final intent.
    await waitFor(() => expect(router.state.status).toBe("idle"));
    store.getState().hydrate([]);
    coordinator.completeHydration();

    const win = store.getState().windows["app:tasks"];
    expect(win).toBeDefined();
    expect(win.location.pathname).toBe("/tasks/t9");
    expect(store.getState().focusedId).toBe("app:tasks");
    expect(container.textContent).toBe("");
  });

  it("Should keep search params in the reported window location", async () => {
    const { store, coordinator, shell, router } = createHarness("/tasks?mode=kanban");

    render(
      <OsShellContext.Provider value={shell}>
        <RouterProvider router={router as never} />
      </OsShellContext.Provider>
    );

    await waitFor(() => expect(router.state.status).toBe("idle"));
    store.getState().hydrate([]);
    coordinator.completeHydration();

    const win = store.getState().windows["app:tasks"];
    expect(win.location.search).toMatchObject({ mode: "kanban" });
  });
});
