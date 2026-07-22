import { createStore, type StoreApi } from "zustand";
import {
  OS_DESKTOP_KEY,
  OS_WINDOW_SOFT_CAP,
  osWindowId,
  type OsDesktopRuntimeStore,
  type OsWindow,
} from "../lib/os-types";
import { getOsApp } from "../lib/app-registry";
import { deriveSnapRect, rectToSnapZone } from "../lib/os-snap-geometry";
import { decodeDesktopEntry, decodeWindowEntry } from "../lib/os-state-payloads";
import { OS_SNAP_ZONES } from "../lib/os-snap-zones";
import { splitZoneBy } from "../lib/os-snap-window-targets";

import {
  clampRect,
  nextFocus,
  sameLocation,
  sameRect,
  sameSnapHint,
} from "./desktop-store-geometry";

/**
 * The window-manager store: single authority for window arrangement (windows,
 * focus, z, presentation, hydration). Side-effect-free — no action navigates
 * or writes to the daemon; the routing coordinator owns URL transitions and
 * the persistence binder owns desktop-state writes (Safety Invariant 13).
 */

interface DesktopInternals {
  /** Monotonic z source; compacted back to 1..N on hydration (invariant 12). */
  zCounter: number;
  /** Highest settled commit seq per desktop-state key (LWW guard, invariant 10). */
  settledSeqs: Record<string, number>;
  softCapEmitted: boolean;
}

export type DesktopStoreApi = StoreApi<OsDesktopRuntimeStore & DesktopInternals>;

export function createDesktopStore(): DesktopStoreApi {
  return createStore<OsDesktopRuntimeStore & DesktopInternals>()((set, get) => ({
    windows: {},
    focusedId: null,
    railOpen: false,
    railCollapsedAgentIds: [],
    wallpaper: "ember",
    reduceMotion: false,
    dockMagnify: true,
    presentation: "floating",
    hydration: "pending",
    softCapNotice: false,
    desktopBounds: null,
    snapHint: null,
    seamPreview: null,
    zCounter: 0,
    settledSeqs: {},
    softCapEmitted: false,

    openOrFocus: target => {
      const id = osWindowId(target.app, target.instanceKey);
      const state = get();
      const existing = state.windows[id];
      if (existing) {
        const z = state.zCounter + 1;
        const location =
          target.location && !sameLocation(target.location, existing.location)
            ? target.location
            : existing.location;
        set({
          windows: {
            ...state.windows,
            [id]: { ...existing, z, minimized: false, location },
          },
          focusedId: id,
          zCounter: z,
        });
        return id;
      }
      const app = getOsApp(target.app);
      const z = state.zCounter + 1;
      const defaultRect = { ...app.defaultRect };
      const win: OsWindow = {
        id,
        app: target.app,
        instanceKey: target.app === "session" ? (target.instanceKey ?? null) : null,
        location: target.location ?? { pathname: app.paths[0], search: {} },
        rect: state.desktopBounds ? clampRect(defaultRect, state.desktopBounds) : defaultRect,
        prevRect: null,
        z,
        minimized: false,
        maximized: false,
        snap: null,
      };
      const windows = { ...state.windows, [id]: win };
      const openCount = Object.keys(windows).length;
      const crossedCap = openCount > OS_WINDOW_SOFT_CAP && !state.softCapEmitted;
      set({
        windows,
        focusedId: id,
        zCounter: z,
        softCapNotice: crossedCap ? true : state.softCapNotice,
        softCapEmitted: state.softCapEmitted || crossedCap,
      });
      return id;
    },

    closeWindow: id => {
      const state = get();
      if (!state.windows[id]) return;
      const windows = { ...state.windows };
      delete windows[id];
      const focusedId = state.focusedId === id ? nextFocus(state.windows, id) : state.focusedId;
      set({ windows, focusedId });
    },

    focusWindow: id => {
      const state = get();
      const win = state.windows[id];
      if (!win || (state.focusedId === id && !win.minimized)) return;
      const z = state.zCounter + 1;
      set({
        windows: { ...state.windows, [id]: { ...win, z, minimized: false } },
        focusedId: id,
        zCounter: z,
      });
    },

    minimizeWindow: id => {
      const state = get();
      const win = state.windows[id];
      if (!win || win.minimized) return;
      const focusedId = state.focusedId === id ? nextFocus(state.windows, id) : state.focusedId;
      set({
        windows: { ...state.windows, [id]: { ...win, minimized: true } },
        focusedId,
      });
    },

    restoreWindow: id => {
      get().focusWindow(id);
    },

    toggleZoom: id => {
      const state = get();
      const win = state.windows[id];
      if (!win || state.presentation === "compact") return;
      // Exclusivity (invariant 19): zooming a snapped window clears `snap` and
      // keeps its original pre-derived `prevRect` so restore stays exact.
      const next: OsWindow = win.maximized
        ? { ...win, maximized: false, rect: win.prevRect ?? win.rect, prevRect: null }
        : {
            ...win,
            maximized: true,
            snap: null,
            prevRect: win.snap !== null ? win.prevRect : { ...win.rect },
          };
      set({ windows: { ...state.windows, [id]: next } });
    },

    snapWindow: (id, zone) => {
      const state = get();
      const win = state.windows[id];
      if (!win || state.presentation === "compact") return;
      if (zone === null) {
        if (win.snap === null) return;
        set({
          windows: {
            ...state.windows,
            [id]: {
              ...win,
              snap: null,
              rect: win.prevRect ? { ...win.prevRect } : win.rect,
              prevRect: null,
            },
          },
        });
        return;
      }
      // Re-snapping (or snapping a maximized window) keeps the original
      // pre-derived rect so restore returns it exactly (invariant 19).
      const prevRect = win.snap !== null || win.maximized ? win.prevRect : { ...win.rect };
      // `rect` records this client's derivation at commit time (ADR-009) for
      // thumbnails and non-rendering readers; later resizes never rewrite it.
      const rect = state.desktopBounds ? deriveSnapRect(zone, state.desktopBounds) : win.rect;
      set({
        windows: {
          ...state.windows,
          [id]: { ...win, snap: { ...zone }, maximized: false, prevRect, rect },
        },
      });
    },

    toggleRail: () => {
      set(state => ({ railOpen: !state.railOpen }));
    },

    openRail: () => {
      if (!get().railOpen) set({ railOpen: true });
    },

    closeRail: () => {
      if (get().railOpen) set({ railOpen: false });
    },

    setWallpaper: wallpaper => {
      if (get().wallpaper !== wallpaper) set({ wallpaper });
    },

    setDockMagnify: on => {
      if (get().dockMagnify !== on) set({ dockMagnify: on });
    },

    setReduceMotion: on => {
      if (get().reduceMotion !== on) set({ reduceMotion: on });
    },

    toggleRailGroup: agentId => {
      const normalized = agentId.trim();
      if (normalized === "") return;
      set(state => ({
        railCollapsedAgentIds: state.railCollapsedAgentIds.includes(normalized)
          ? state.railCollapsedAgentIds.filter(id => id !== normalized)
          : [...state.railCollapsedAgentIds, normalized],
      }));
    },

    commitRect: (id, rect) => {
      const state = get();
      const win = state.windows[id];
      if (!win || state.presentation === "compact") return;
      // A px commit lands floating geometry (invariant 19 refinement): a
      // drag-away detaches `snap`, a resize on a maximized window detaches
      // `maximized` — either way the derived branch ends in this mutation.
      if (win.snap !== null || win.maximized) {
        set({
          windows: {
            ...state.windows,
            [id]: { ...win, rect: { ...rect }, snap: null, maximized: false, prevRect: null },
          },
        });
        return;
      }
      if (sameRect(win.rect, rect)) return;
      set({ windows: { ...state.windows, [id]: { ...win, rect: { ...rect } } } });
    },

    resizeSnapped: (id, rect) => {
      const state = get();
      const win = state.windows[id];
      if (!win || state.presentation === "compact") return;
      if (win.snap === null || state.desktopBounds === null) return;
      // Resize-in-place keeps the window snapped: the resulting px rect maps
      // back to fractions so reflow stays proportional and agents keep seeing
      // a snapped window. `prevRect` survives for drag-away restore.
      const zone = rectToSnapZone(rect, state.desktopBounds);
      const derived = deriveSnapRect(zone, state.desktopBounds);
      set({
        windows: { ...state.windows, [id]: { ...win, snap: zone, rect: derived } },
      });
    },

    setLocation: (id, loc) => {
      const state = get();
      const win = state.windows[id];
      if (!win || sameLocation(win.location, loc)) return;
      set({ windows: { ...state.windows, [id]: { ...win, location: loc } } });
    },

    applyRemote: event => {
      const { entry } = event;
      const state = get();
      const settled = state.settledSeqs[entry.key] ?? 0;
      if (entry.seq <= settled) return;
      const settledSeqs = { ...state.settledSeqs, [entry.key]: entry.seq };

      if (entry.key === OS_DESKTOP_KEY) {
        if (entry.deleted) {
          set({ settledSeqs });
          return;
        }
        const desktop = decodeDesktopEntry(entry);
        if (!desktop) {
          set({ settledSeqs });
          return;
        }
        const focusedId =
          desktop.focusedId !== null && state.windows[desktop.focusedId]
            ? desktop.focusedId
            : desktop.focusedId === null
              ? null
              : state.focusedId;
        set({
          settledSeqs,
          focusedId,
          railOpen: desktop.railOpen,
          railCollapsedAgentIds: desktop.railCollapsedAgentIds,
          wallpaper: desktop.wallpaper,
          reduceMotion: desktop.reduceMotion,
          dockMagnify: desktop.dockMagnify,
        });
        return;
      }

      const windowId = entry.key.startsWith("win:") ? entry.key.slice(4) : null;
      if (windowId === null) {
        set({ settledSeqs });
        return;
      }
      if (entry.deleted) {
        if (!state.windows[windowId]) {
          set({ settledSeqs });
          return;
        }
        const windows = { ...state.windows };
        delete windows[windowId];
        const focusedId =
          state.focusedId === windowId ? nextFocus(state.windows, windowId) : state.focusedId;
        set({ settledSeqs, windows, focusedId });
        return;
      }
      const win = decodeWindowEntry(entry);
      if (!win) {
        set({ settledSeqs });
        return;
      }
      const zCounter = Math.max(state.zCounter, win.z);
      set({
        settledSeqs,
        zCounter,
        windows: { ...state.windows, [windowId]: win },
      });
    },

    hydrate: entries => {
      const state = get();
      const settledSeqs: Record<string, number> = {};
      const decoded: OsWindow[] = [];
      let desktop: ReturnType<typeof decodeDesktopEntry> = null;
      for (const entry of entries) {
        settledSeqs[entry.key] = entry.seq;
        if (entry.key === OS_DESKTOP_KEY) {
          desktop = decodeDesktopEntry(entry);
          continue;
        }
        const win = decodeWindowEntry(entry);
        if (win) decoded.push(win);
      }
      decoded.sort((a, b) => a.z - b.z);
      const windows: Record<string, OsWindow> = {};
      decoded.forEach((win, index) => {
        windows[win.id] = { ...win, z: index + 1 };
      });
      const restoredFocus = desktop?.focusedId ?? null;
      const focusedId =
        restoredFocus !== null && windows[restoredFocus] && !windows[restoredFocus].minimized
          ? restoredFocus
          : desktop
            ? null
            : nextFocus(windows, "");
      set({
        windows,
        focusedId,
        railOpen: desktop?.railOpen ?? state.railOpen,
        railCollapsedAgentIds: desktop?.railCollapsedAgentIds ?? state.railCollapsedAgentIds,
        wallpaper: desktop?.wallpaper ?? state.wallpaper,
        reduceMotion: desktop?.reduceMotion ?? state.reduceMotion,
        dockMagnify: desktop?.dockMagnify ?? state.dockMagnify,
        zCounter: decoded.length,
        settledSeqs,
        hydration: "live",
      });
    },

    setPresentation: presentation => {
      if (get().presentation === presentation) return;
      set({ presentation });
    },

    setHydration: hydration => {
      if (get().hydration === hydration) return;
      set({ hydration });
    },

    markSettled: (key, seq) => {
      const state = get();
      if ((state.settledSeqs[key] ?? 0) >= seq) return;
      set({ settledSeqs: { ...state.settledSeqs, [key]: seq } });
    },

    clampToViewport: bounds => {
      const state = get();
      if (state.presentation === "compact" || bounds.width <= 0 || bounds.height <= 0) return;
      const sameBounds =
        state.desktopBounds?.width === bounds.width &&
        state.desktopBounds?.height === bounds.height;
      let changed = false;
      const windows: Record<string, OsWindow> = {};
      for (const [id, win] of Object.entries(state.windows)) {
        // Derived-geometry windows re-derive from the viewport instead of
        // committing on resize (invariant 19) — their stored rect stays put.
        if (win.snap !== null || win.maximized) {
          windows[id] = win;
          continue;
        }
        const clamped = clampRect(win.rect, bounds);
        if (sameRect(clamped, win.rect)) {
          windows[id] = win;
        } else {
          windows[id] = { ...win, rect: clamped };
          changed = true;
        }
      }
      if (!changed && sameBounds) return;
      set({
        ...(changed ? { windows } : {}),
        desktopBounds: sameBounds ? state.desktopBounds : { ...bounds },
      });
    },

    clearSoftCapNotice: () => {
      if (!get().softCapNotice) return;
      set({ softCapNotice: false });
    },

    setSnapHint: hint => {
      const current = get().snapHint;
      if (current === hint) return;
      if (current !== null && hint !== null && sameSnapHint(current, hint)) return;
      set({ snapHint: hint });
    },

    setSeamPreview: preview => {
      if (get().seamPreview === null && preview === null) return;
      set({ seamPreview: preview });
    },

    arrangeWindows: (anchorId, preset) => {
      const state = get();
      const anchor = state.windows[anchorId];
      if (!anchor || state.presentation === "compact") return;
      const others = Object.values(state.windows)
        .filter(win => win.id !== anchorId && !win.minimized)
        .sort((a, b) => b.z - a.z);
      if (others.length === 0) return;
      if (preset === "two-up" || others.length === 1) {
        state.snapWindow(anchorId, OS_SNAP_ZONES.left);
        state.snapWindow(others[0].id, OS_SNAP_ZONES.right);
        return;
      }
      if (others.length === 2) {
        state.snapWindow(anchorId, OS_SNAP_ZONES.left);
        state.snapWindow(others[0].id, OS_SNAP_ZONES["top-right"]);
        state.snapWindow(others[1].id, OS_SNAP_ZONES["bottom-right"]);
        return;
      }
      state.snapWindow(anchorId, OS_SNAP_ZONES["top-left"]);
      state.snapWindow(others[0].id, OS_SNAP_ZONES["top-right"]);
      state.snapWindow(others[1].id, OS_SNAP_ZONES["bottom-left"]);
      state.snapWindow(others[2].id, OS_SNAP_ZONES["bottom-right"]);
    },

    splitWindows: (dragId, targetId, side) => {
      const state = get();
      const drag = state.windows[dragId];
      const target = state.windows[targetId];
      if (!drag || !target || state.presentation === "compact") return;
      if (target.minimized || target.maximized || state.desktopBounds === null) return;
      // The target's visual footprint becomes fractions (identity for snapped
      // targets, work-area inverse for floating ones), then splits evenly.
      const base = target.snap ?? rectToSnapZone(target.rect, state.desktopBounds);
      const zones = splitZoneBy(base, side);
      if (zones === null) return;
      state.snapWindow(targetId, zones.target);
      state.snapWindow(dragId, zones.dragged);
    },

    resetForWorkspace: () => {
      set({
        windows: {},
        focusedId: null,
        railOpen: false,
        railCollapsedAgentIds: [],
        wallpaper: "ember",
        reduceMotion: false,
        dockMagnify: true,
        hydration: "pending",
        softCapNotice: false,
        snapHint: null,
        seamPreview: null,
        zCounter: 0,
        settledSeqs: {},
      });
    },
  }));
}

export const desktopStore: DesktopStoreApi = createDesktopStore();
