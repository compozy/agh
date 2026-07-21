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

/**
 * Persisted snap zone: normalized fractions (0..1) of the desktop work area
 * (ADR-009). Clients derive px locally; fractions are the cross-client truth.
 */
export interface OsSnapZone {
  fx: number;
  fy: number;
  fw: number;
  fh: number;
}

/** The six v1 pointer zones (ADR-009): halves + corner quarters. Top edge stays unbound — zoom owns full. */
export type OsSnapZoneId =
  | "left"
  | "right"
  | "top-left"
  | "top-right"
  | "bottom-left"
  | "bottom-right";

/** Ephemeral drag hint: the zone a release would snap `windowId` into. */
export interface OsSnapHint {
  windowId: string;
  zoneId: OsSnapZoneId;
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
  /** Derived-geometry authority while set; exclusive with `maximized` (invariant 19). */
  snap: OsSnapZone | null;
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
  railCollapsedAgentIds: string[];
  wallpaper: OsWallpaper;
  /** Desktop-doc motion preference (in-product toggle; system pref composes at read sites). */
  reduceMotion: boolean;
  dockMagnify: boolean;
  presentation: OsPresentation;
  hydration: OsHydration;
  openOrFocus(target: OsOpenTarget): string;
  closeWindow(id: string): void;
  focusWindow(id: string): void;
  minimizeWindow(id: string): void;
  restoreWindow(id: string): void;
  toggleZoom(id: string): void;
  /**
   * Snaps a window to a fractional zone (invariant 19): stores the pre-snap
   * rect in `prevRect` and clears `maximized`. `zone: null` restores the
   * stored `prevRect` exactly. No-op in compact presentation.
   */
  snapWindow(id: string, zone: OsSnapZone | null): void;
  toggleRail(): void;
  openRail(): void;
  closeRail(): void;
  toggleRailGroup(agentId: string): void;
  commitRect(id: string, rect: OsRect): void;
  setLocation(id: string, loc: OsWindowLocation): void;
  applyRemote(event: OsStateEvent): void;
}

/** Operational WM state used by the shell around the exact public contract. */
export interface OsDesktopRuntimeStore extends OsDesktopStore {
  /** One-shot soft-cap guidance flag; set once when the 13th window opens. */
  softCapNotice: boolean;
  /** Last measured win-layer box; snapped/maximized geometry derives from it. */
  desktopBounds: OsDesktopBounds | null;
  /** Live snap-drag hint (one per desktop); never persisted. */
  snapHint: OsSnapHint | null;
  setSnapHint(hint: OsSnapHint | null): void;
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

/** Window minimum size (px); Rnd resize and derived snap rects clamp to it. */
export const OS_WINDOW_MIN_WIDTH = 280;
export const OS_WINDOW_MIN_HEIGHT = 180;

/**
 * Work-area insets (px) inside the win-layer — the "fill the desktop" box both
 * maximized windows and snap fractions derive from (one authority, ADR-009).
 */
export const OS_WORK_AREA_INSETS = { top: 8, right: 10, bottom: 78, left: 10 } as const;

/** Codec floor for snap fractions: a zone spans ≥10% of the work area per axis. */
export const OS_SNAP_MIN_FRACTION = 0.1;

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
