import { useState, type ReactNode } from "react";

import { OsShellContext, type OsShellHandle } from "../../contexts/os-shell-context";
import { RoutingCoordinator, type OsRouterPort } from "../../lib/routing-coordinator";
import { createDesktopStore, type DesktopStoreApi } from "../../stores/desktop-store";

/**
 * Build a story-only shell (real desktop store + coordinator). Optional
 * `arrange` mutates the store before the first paint — call from the
 * `createShell` factory passed to `StoryShellProvider`, never during render.
 */
export function createStoryShell(arrange?: (store: DesktopStoreApi) => void): OsShellHandle {
  const store = createDesktopStore();
  const port: OsRouterPort = { navigate: () => {}, replace: () => {} };
  const coordinator = new RoutingCoordinator(store, port);
  store.getState().hydrate([]);
  coordinator.completeHydration();
  arrange?.(store);
  return { store, coordinator, flushPersistence: () => {} };
}

/**
 * Story-only shell harness: a REAL desktop store + coordinator behind
 * `OsShellContext`, so store-connected components (seams, zoom menu) run
 * their production behavior inside the canvas. `createShell` is the lazy
 * `useState` initializer — one shell per mount.
 */
export function StoryShellProvider({
  createShell = createStoryShell,
  children,
}: {
  createShell?: () => OsShellHandle;
  children: ReactNode;
}) {
  const [shell] = useState(createShell);
  return <OsShellContext.Provider value={shell}>{children}</OsShellContext.Provider>;
}
