import { useState, type ReactNode } from "react";

import { OsShellContext, type OsShellHandle } from "../../contexts/os-shell-context";
import { RoutingCoordinator, type OsRouterPort } from "../../lib/routing-coordinator";
import { createDesktopStore, type DesktopStoreApi } from "../../stores/desktop-store";

/**
 * Story-only shell harness: a REAL desktop store + coordinator behind
 * `OsShellContext`, so store-connected components (seams, zoom menu) run
 * their production behavior inside the canvas. `setup` arranges the store
 * once (lazy state initializer — one store per mount).
 */
export function StoryShellProvider({
  setup,
  children,
}: {
  setup?: (store: DesktopStoreApi) => void;
  children: ReactNode;
}) {
  const [shell] = useState<OsShellHandle>(() => {
    const store = createDesktopStore();
    const port: OsRouterPort = { navigate: () => {}, replace: () => {} };
    const coordinator = new RoutingCoordinator(store, port);
    store.getState().hydrate([]);
    coordinator.completeHydration();
    setup?.(store);
    return { store, coordinator, flushPersistence: () => {} };
  });
  return <OsShellContext.Provider value={shell}>{children}</OsShellContext.Provider>;
}
