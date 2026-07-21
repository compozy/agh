export { DesktopShell } from "./components/desktop-shell";
export { createOsRouteSync } from "./components/os-route-sync";
export { OsRouteNotFound } from "./components/os-route-not-found";
export { OsShellContext, type OsShellHandle } from "./contexts/os-shell-context";
export { useOsShell } from "./hooks/use-os-shell";
export { useDesktop } from "./hooks/use-desktop";
export { OS_APPS, getOsApp, resolveAppForPath, matchSessionInstance } from "./lib/app-registry";
export { RoutingCoordinator, type OsRouterPort } from "./lib/routing-coordinator";
export { OsStateClient, type OsSocket, type OsSocketFactory } from "./lib/os-state-client";
export { createDesktopPersistence } from "./lib/desktop-persistence";
export { createDesktopStore, desktopStore, type DesktopStoreApi } from "./stores/desktop-store";
export {
  OS_COMPACT_BREAKPOINT,
  OS_RECT_DEBOUNCE_MS,
  OS_WINDOW_SOFT_CAP,
  osWindowId,
  osWindowKey,
  type OsAppId,
  type OsDesktopStore,
  type OsRect,
  type OsStateEntry,
  type OsStateEvent,
  type OsWindow,
  type OsWindowLocation,
} from "./lib/os-types";
