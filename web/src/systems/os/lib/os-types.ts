import type { components } from "@/generated/agh-openapi";

/** The 14 desktop apps. Window ids derive from these (`app:<id>` | `session:<sessionId>`). */
export type OsAppId =
  | "dashboard"
  | "session"
  | "tasks"
  | "agents"
  | "network"
  | "loops"
  | "jobs"
  | "triggers"
  | "marketplace"
  | "bridges"
  | "knowledge"
  | "sandbox"
  | "vault"
  | "settings";

export interface OsRect {
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface OsWindowLocation {
  pathname: string;
  search: Record<string, unknown>;
}

export interface OsWindow {
  id: string;
  app: OsAppId;
  instanceKey: string | null;
  location: OsWindowLocation;
  rect: OsRect;
  prevRect: OsRect | null;
  z: number;
  minimized: boolean;
  maximized: boolean;
}

export type OsWallpaper = "ember" | "mesh" | "carbon";
export type OsPresentation = "floating" | "compact";
export type OsHydration = "pending" | "live" | "degraded";

/** Canonical wire DTO for one desktop-state entry (generated from OpenAPI). */
export type OsStateEntry = components["schemas"]["DesktopStateEntry"];

/** One live desktop-state change delivered over the stream. */
export interface OsStateEvent {
  entry: OsStateEntry;
  origin: string;
}

export interface OsOpenTarget {
  app: OsAppId;
  instanceKey?: string;
  location?: OsWindowLocation;
}

/** Viewport box the win-layer occupies; rect clamping is computed against it. */
export interface OsDesktopBounds {
  width: number;
  height: number;
}

export interface OsDesktopStore {
  windows: Record<string, OsWindow>;
  focusedId: string | null;
  railOpen: boolean;
  wallpaper: OsWallpaper;
  presentation: OsPresentation;
  hydration: OsHydration;
  openOrFocus(target: OsOpenTarget): string;
  closeWindow(id: string): void;
  focusWindow(id: string): void;
  minimizeWindow(id: string): void;
  restoreWindow(id: string): void;
  toggleZoom(id: string): void;
  toggleRail(): void;
  commitRect(id: string, rect: OsRect): void;
  setLocation(id: string, loc: OsWindowLocation): void;
  applyRemote(event: OsStateEvent): void;
}

/** Operational WM state used by the shell around the exact public contract. */
export interface OsDesktopRuntimeStore extends OsDesktopStore {
  /** One-shot soft-cap guidance flag; set once when the 13th window opens. */
  softCapNotice: boolean;
  /** Snapshot adoption: replaces the mirror, drops invalid entries, compacts z. */
  hydrate(entries: OsStateEntry[]): void;
  setPresentation(presentation: OsPresentation): void;
  setHydration(hydration: OsHydration): void;
  /** Records the settled commit seq for a key so stale remote events are ignored. */
  markSettled(key: string, seq: number): void;
  /** Pulls out-of-bounds windows back into the desktop after a viewport resize. */
  clampToViewport(bounds: OsDesktopBounds): void;
  clearSoftCapNotice(): void;
  /** Clears the desktop for a workspace switch; hydration restarts as pending. */
  resetForWorkspace(): void;
}

/** Presentation breakpoint (px); below it the desktop stacks (compact). */
export const OS_COMPACT_BREAKPOINT = 960;
/** Trailing debounce for rect persistence writes. */
export const OS_RECT_DEBOUNCE_MS = 250;
/** Soft open-window guidance threshold (toast, no hard cap). */
export const OS_WINDOW_SOFT_CAP = 12;

/** Desktop gutters (px) the clamp respects: menubar above, dock strip below. */
export const OS_DESKTOP_GUTTERS = { top: 4, right: 8, bottom: 120, left: 8 } as const;

export function osWindowId(app: OsAppId, instanceKey?: string | null): string {
  return app === "session" && instanceKey ? `session:${instanceKey}` : `app:${app}`;
}

/** Desktop-state key for one window (`win:<windowId>`). */
export function osWindowKey(windowId: string): string {
  return `win:${windowId}`;
}

/** Desktop-state key for the desktop-wide prefs doc. */
export const OS_DESKTOP_KEY = "desktop";

export function windowIdFromKey(key: string): string | null {
  return key.startsWith("win:") ? key.slice(4) : null;
}
