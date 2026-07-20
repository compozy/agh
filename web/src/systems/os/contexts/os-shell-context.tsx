import { createContext } from "react";

import type { RoutingCoordinator } from "../lib/routing-coordinator";
import type { DesktopStoreApi } from "../stores/desktop-store";

export interface OsShellHandle {
  store: DesktopStoreApi;
  coordinator: RoutingCoordinator;
  /** Sends any pending debounced desktop-state writes now (gesture end). */
  flushPersistence: () => void;
}

export const OsShellContext = createContext<OsShellHandle | null>(null);
