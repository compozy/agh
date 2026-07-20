import { resolveAppForPath } from "./app-registry";
import {
  osWindowId,
  type OsDesktopStore,
  type OsOpenTarget,
  type OsWindowLocation,
} from "./os-types";

/**
 * The routing coordinator is the ONLY URL↔WM bridge (Safety Invariant 13).
 * Every transition carries an explicit cause:
 *
 * - `route-pop` (browser back/forward, deep link, in-window `<Link>` match):
 *   reconciles URL → store and never writes history.
 * - `user-open` / `user-focus` (dock, palette, rail, window activation): the
 *   store updates first, then exactly one `navigate` (push) reflects the
 *   focused window in the URL.
 * - `hydrate`: the daemon snapshot applies before the initial URL intent; a
 *   cold deep link wins focus without losing the restored desktop.
 * - `workspace-switch`: swap + rehydrate, then one navigate to the target
 *   space's focused location (or `/`).
 */

export type OsRouteCause =
  | "user-open"
  | "user-focus"
  | "route-pop"
  | "hydrate"
  | "workspace-switch";

export interface OsRouterPort {
  /** Pushes one history entry reflecting the location. */
  navigate(location: OsWindowLocation): void;
  /** Replaces the current entry (hydration URL truth-up only). */
  replace(location: OsWindowLocation): void;
}

interface StoreLike {
  getState(): OsDesktopStore;
}

export class RoutingCoordinator {
  private readonly store: StoreLike;
  private readonly router: OsRouterPort;
  private phase: "hydrating" | "ready" = "hydrating";
  private cycle: "boot" | "workspace-switch" = "boot";
  private initialIntent: OsWindowLocation | null = null;

  constructor(store: StoreLike, router: OsRouterPort) {
    this.store = store;
    this.router = router;
  }

  /** Sync-controllers report every matched location here (cause: route-pop). */
  reportRouteMatch(location: OsWindowLocation): void {
    if (this.phase === "hydrating") {
      if (this.cycle === "boot") this.initialIntent = location;
      return;
    }
    this.reconcile(location);
  }

  /**
   * Boot: applies the daemon snapshot's focus truth, then the initial URL as
   * the final focus intent (Routing Model rule 4) — no history write; the URL
   * is trued up by `replace` when a restored focus exists at a neutral URL.
   * Workspace switch: one push to the new space's focused location (rule 8).
   */
  completeHydration(): void {
    if (this.phase !== "hydrating") return;
    this.phase = "ready";
    const intent = this.initialIntent;
    this.initialIntent = null;
    if (this.cycle === "workspace-switch") {
      this.cycle = "boot";
      this.navigateToFocusedOrDesktop();
      return;
    }
    if (intent && intent.pathname !== "/") {
      this.reconcile(intent);
      return;
    }
    const state = this.store.getState();
    const focused = state.focusedId !== null ? state.windows[state.focusedId] : null;
    if (focused) this.router.replace(focused.location);
  }

  /** Dock, palette, rail, menubar: open-or-focus then one history entry. */
  userOpen(target: OsOpenTarget): string {
    const state = this.store.getState();
    const id = state.openOrFocus(target);
    const win = this.store.getState().windows[id];
    if (win) this.router.navigate(win.location);
    return id;
  }

  /**
   * Window activation (pointerdown, Tab, Enter — Routing Model rule 5). When
   * the activation target is a link, the link's own navigation writes the one
   * history entry and reconciliation follows it (rule 3 coalescing).
   */
  userFocus(windowId: string, opts: { viaLink?: boolean } = {}): void {
    const state = this.store.getState();
    const win = state.windows[windowId];
    if (!win || (state.focusedId === windowId && !win.minimized)) return;
    state.focusWindow(windowId);
    if (opts.viaLink) return;
    const focused = this.store.getState().windows[windowId];
    if (focused) this.router.navigate(focused.location);
  }

  /** Close: successor focus follows ADR-002 (next-top window, else desktop). */
  userClose(windowId: string): void {
    const state = this.store.getState();
    if (!state.windows[windowId]) return;
    const wasFocused = state.focusedId === windowId;
    state.closeWindow(windowId);
    if (!wasFocused) return;
    this.navigateToFocusedOrDesktop();
  }

  /** Minimize: when focus moves to a successor, the URL follows it. */
  userMinimize(windowId: string): void {
    const state = this.store.getState();
    const win = state.windows[windowId];
    if (!win || win.minimized) return;
    const wasFocused = state.focusedId === windowId;
    state.minimizeWindow(windowId);
    if (!wasFocused) return;
    this.navigateToFocusedOrDesktop();
  }

  /** Workspace switch: rehydration restarts; completeHydration navigates once. */
  beginWorkspaceSwitch(): void {
    this.phase = "hydrating";
    this.cycle = "workspace-switch";
    this.initialIntent = null;
  }

  private navigateToFocusedOrDesktop(): void {
    const state = this.store.getState();
    const focused = state.focusedId !== null ? state.windows[state.focusedId] : null;
    this.router.navigate(focused ? focused.location : { pathname: "/", search: {} });
  }

  /**
   * Route-originated reconciliation (rule 2): store updates only, no history.
   * The desktop URL (`/`) opens nothing — it focuses an existing dashboard
   * window or leaves the desktop as-is (first run stays empty, US-001.EC-1).
   */
  private reconcile(location: OsWindowLocation): void {
    const resolved = resolveAppForPath(location.pathname);
    if (!resolved) return;
    const state = this.store.getState();
    const { app, instanceKey } = resolved;
    if (location.pathname === "/") {
      const existing = state.windows[osWindowId(app.id)];
      if (!existing) return;
      state.focusWindow(existing.id);
      state.setLocation(existing.id, location);
      return;
    }
    const id = state.openOrFocus({
      app: app.id,
      instanceKey: instanceKey ?? undefined,
      location,
    });
    state.setLocation(id, location);
  }
}
