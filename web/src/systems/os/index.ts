export { DesktopShell } from "./components/desktop-shell";
export { createOsRouteSync } from "./components/os-route-sync";
export { OsRouteNotFound } from "./components/os-route-not-found";
export { OsShellContext, type OsShellHandle } from "./contexts/os-shell-context";
export { useOsShell } from "./hooks/use-os-shell";
export { useDesktop } from "./hooks/use-desktop";
export { OS_APPS, getOsApp, resolveAppForPath, matchSessionInstance } from "./lib/app-registry";
export { RoutingCoordinator, type OsRouterPort } from "./lib/routing-coordinator";
export { WindowManagerRuntime } from "./hooks/window-manager-runtime";
export { fetchWindowManagerSnapshot } from "./adapters/window-manager-api";
export {
  windowManagerConfigOptions,
  windowManagerKeys,
  windowManagerSnapshotOptions,
} from "./lib/window-manager-query";
export {
  type WindowManagerSocket,
  type WindowManagerSocketFactory,
} from "./hooks/use-window-manager-stream";
export {
  OS_COMPACT_BREAKPOINT,
  OS_WINDOW_SOFT_CAP,
  osWindowId,
  type OsAppId,
  type OsDesktopRuntime,
  type OsDesktopRuntimeStore,
  type OsRect,
  type OsWindow,
  type OsWindowRoute,
  type WindowManagerController,
} from "./lib/os-types";
export type {
  WindowManagerClientView,
  WindowManagerConfig,
  WindowManagerSnapshot,
} from "./lib/window-manager-types";
