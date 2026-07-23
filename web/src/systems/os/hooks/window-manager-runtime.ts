import type { QueryClient } from "@tanstack/react-query";

import { clampFloatingRect } from "../lib/layout-projection";
import {
  createTileSnapTarget,
  type SnapCorner,
  type SnapSide,
  type SnapTarget,
} from "../lib/snap-targets";
import {
  arrangeLayoutCommand,
  moveWindowCommand,
  openWindowCommand,
  restoreWindowCommand,
} from "../lib/window-manager-command-builders";
import { effectiveWindowManagerConfig } from "../lib/window-manager-config";
import { arrangePeerWindows, directionalFocusTarget } from "../lib/window-manager-navigation";
import type {
  MoveWindowInput,
  OsArrangePreset,
  OsDesktopBounds,
  OsDesktopRuntime,
  OsDesktopRuntimeStore,
  OsOpenTarget,
  OsRect,
  OsWindowRoute,
  OsWallpaper,
} from "../lib/os-types";
import { OS_COMPACT_BREAKPOINT, osWindowId } from "../lib/os-types";
import type { FocusDirection } from "../lib/window-manager-types";
import { sameOsWindowRoute } from "../lib/window-manager-route";
import { windowManagerLayoutArea } from "../lib/window-manager-layout-area";
import {
  buildWindowManagerProjections,
  buildWindowManagerWindows,
  normalizedRectToWire,
  pixelRectToNormalized,
} from "../lib/window-manager-view";
import { windowManagerStore } from "../stores/window-manager-store";
import { randomWindowManagerId, WindowManagerRuntimeCore } from "./window-manager-runtime-core";

export class WindowManagerRuntime extends WindowManagerRuntimeCore implements OsDesktopRuntime {
  constructor(queryClient: QueryClient) {
    super(queryClient);
    this.initializeView();
  }

  createDesktop(): void {
    this.dispatch({
      commandId: "desktop.create",
      payload: { desktop_id: "", name: "", purpose: "standard" },
    });
  }

  renameDesktop(desktopId: string, name: string): void {
    this.dispatch({ commandId: "desktop.update", payload: { desktop_id: desktopId, name } });
  }

  reorderDesktop(desktopId: string, order: number): void {
    this.dispatch({ commandId: "desktop.reorder", payload: { desktop_id: desktopId, order } });
  }

  switchDesktop(desktopId: string): void {
    const current = this.view.activeDesktopId;
    const config = this.view.windowManagerConfig;
    const accepted = this.dispatch({
      commandId: "desktop.switch",
      payload: { desktop_id: desktopId },
    });
    if (accepted && current !== null && current !== desktopId && config !== null) {
      const fromOrder = this.view.desktops.find(desktop => desktop.id === current)?.order ?? 0;
      const toOrder = this.view.desktops.find(desktop => desktop.id === desktopId)?.order ?? 0;
      windowManagerStore.getState().actions.setTransitionIntent({
        fromDesktopId: current,
        toDesktopId: desktopId,
        direction: toOrder >= fromOrder ? "later" : "earlier",
        mode: this.reduceMotion ? "instant" : config.desktopTransition,
      });
    }
  }

  switchDesktopDirection(direction: "previous" | "next"): void {
    const active = this.view.activeDesktopId;
    const ordered = [...this.view.desktops].sort(
      (left, right) => left.order - right.order || left.id.localeCompare(right.id)
    );
    const index = ordered.findIndex(desktop => desktop.id === active);
    if (index < 0) return;
    const target = ordered[index + (direction === "next" ? 1 : -1)];
    if (target) this.switchDesktop(target.id);
  }

  deleteDesktop(desktopId: string, destinationId: string | null): void {
    this.dispatch({
      commandId: "desktop.delete",
      payload: {
        desktop_id: desktopId,
        ...(destinationId ? { destination_id: destinationId } : {}),
      },
    });
  }

  moveWindowToDesktop(windowId: string, destinationDesktopId: string): void {
    this.dispatch({
      commandId: "window.move",
      payload: {
        window_id: windowId,
        destination_desktop_id: destinationDesktopId,
        placement: "floating",
        move_group: false,
      },
    });
  }

  tileWindow(windowId: string, edge: SnapSide | SnapCorner): void {
    const config = this.view.windowManagerConfig;
    if (!this.view.windows[windowId] || config === null) return;
    const cycleStep = windowManagerStore.getState().actions.nextPlacementCycle(windowId, edge);
    this.applySnapTarget(
      windowId,
      createTileSnapTarget(
        windowManagerLayoutArea(this.workArea(), config.gaps),
        edge,
        config.snap.repeatRatios,
        cycleStep
      )
    );
  }

  applySnapTarget(windowId: string, target: SnapTarget, moveGroup = false): void {
    const window = this.view.windows[windowId];
    if (!window) return;
    windowManagerStore
      .getState()
      .actions.trackPlacementTarget(windowId, target.kind === "tile" ? target.edge : null);
    if (target.kind === "zoom") {
      this.zoomWindow(windowId);
      return;
    }
    if (target.kind === "tile") {
      const config = this.view.windowManagerConfig;
      if (config === null) return;
      const frame = pixelRectToNormalized(
        target.rect,
        windowManagerLayoutArea(this.workArea(), config.gaps)
      );
      this.dispatch({
        commandId: "layout.arrange",
        payload: {
          desktop_id: window.desktopId,
          window_ids: [windowId],
          arrangement: "horizontal",
          frame: normalizedRectToWire(frame),
          group_id: randomWindowManagerId("group"),
        },
      });
      return;
    }
    const placement =
      target.kind === "stack" ? "center" : target.kind === "insert" ? target.relation : target.side;
    this.moveWindow(windowId, {
      destinationDesktopId: window.desktopId,
      targetWindowId: target.targetWindowId,
      placement,
      moveGroup,
    });
  }

  focusDirection(direction: FocusDirection): void {
    const config = this.view.windowManagerConfig;
    const activeDesktopId = this.view.activeDesktopId;
    if (config === null || activeDesktopId === null) return;
    const target = directionalFocusTarget({
      windows: Object.values(this.view.windows).filter(
        window => window.desktopId === activeDesktopId
      ),
      focusedId: this.view.focusedId,
      direction,
      wrap: config.focusWrap,
    });
    if (target === null) return;
    this.dispatch({
      commandId: "window.focus",
      payload: { window_id: target, direction: "" },
    });
  }

  undoLayout(): void {
    this.dispatch({ commandId: "layout.undo", payload: {} });
  }

  redoLayout(): void {
    this.dispatch({ commandId: "layout.redo", payload: {} });
  }

  balanceFocusedLayout(): void {
    const focused = this.view.focusedId ? this.view.windows[this.view.focusedId] : undefined;
    if (!focused) return;
    this.balanceLayout(focused.groupId ?? undefined, focused.nodeId ?? undefined);
  }

  protected buildView(): OsDesktopRuntimeStore {
    const snapshot = this.snapshot();
    const area = this.workArea();
    const globalConfig = this.config();
    const config =
      snapshot && globalConfig
        ? effectiveWindowManagerConfig(globalConfig, snapshot.overrides)
        : null;
    const projections = buildWindowManagerProjections(snapshot, this.client, area, config);
    const windows = buildWindowManagerWindows({
      snapshot: config ? snapshot : null,
      client: this.client,
      workArea: area,
      projections,
      raiseOnFocus: config?.raiseOnFocus ?? false,
    });
    const connectionStatus = windowManagerStore.getState().connectionStatus;
    const loadError = this.currentLoadError();
    const hydration =
      snapshot !== null && config !== null
        ? loadError
          ? "degraded"
          : "live"
        : loadError
          ? "degraded"
          : "pending";
    const viewportRejected =
      area.w < OS_COMPACT_BREAKPOINT && config?.smallViewportPolicy === "reject";

    return {
      snapshot,
      windowManagerConfig: config,
      client: this.client,
      desktops: snapshot?.desktops ?? [],
      projections,
      windows,
      activeDesktopId: this.client?.activeDesktopId ?? snapshot?.desktops[0]?.id ?? null,
      focusedId: this.client?.focusedWindowId ?? null,
      railCollapsedAgentIds: this.railCollapsedAgentIds,
      wallpaper: this.wallpaper,
      reduceMotion: this.reduceMotion,
      dockMagnify: this.dockMagnify,
      presentation:
        area.w < OS_COMPACT_BREAKPOINT && config?.smallViewportPolicy === "stack"
          ? "compact"
          : "floating",
      viewportState: viewportRejected ? "rejected" : "ready",
      hydration,
      connectionStatus,
      desktopBounds: { width: area.w, height: area.h },
      openOrFocus: this.openOrFocus,
      closeWindow: this.closeWindow,
      focusWindow: this.focusWindow,
      minimizeWindow: this.minimizeWindow,
      restoreWindow: this.restoreWindow,
      zoomWindow: this.zoomWindow,
      toggleFloating: this.toggleFloating,
      moveWindow: this.moveWindow,
      arrangeLayout: this.arrangeLayout,
      commitFloatingRect: this.commitFloatingRect,
      resizeLayout: this.resizeLayout,
      balanceLayout: this.balanceLayout,
      navigateWindow: this.navigateWindow,
      toggleRailGroup: this.toggleRailGroup,
      setWallpaper: this.setWallpaper,
      setDockMagnify: this.setDockMagnify,
      setReduceMotion: this.setReduceMotion,
      setDesktopBounds: this.setDesktopBounds,
    };
  }

  private openOrFocus = (target: OsOpenTarget): string => {
    const id = osWindowId(target.app, target.instanceKey);
    const existing = this.view.windows[id];
    if (existing) {
      if (existing.minimized) {
        const authoritative = this.view.snapshot?.windows[id];
        if (authoritative) {
          this.dispatch(restoreWindowCommand(authoritative, target.route));
        }
      } else if (target.route && !sameOsWindowRoute(existing.route, target.route)) {
        this.navigateWindow(id, target.route);
      } else {
        this.focusWindow(id);
      }
      return id;
    }

    const desktopId = this.view.activeDesktopId;
    if (desktopId === null) return id;
    this.dispatch(openWindowCommand(target, id, desktopId));
    this.publish();
    return id;
  };

  private closeWindow = (id: string): Promise<boolean> => {
    return this.dispatchConfirmed({
      commandId: "window.close",
      payload: { window_id: id, minimize: false },
    });
  };

  private focusWindow = (id: string): void => {
    this.dispatch({
      commandId: "window.focus",
      payload: { window_id: id, direction: "" },
    });
  };

  private minimizeWindow = (id: string): Promise<boolean> => {
    return this.dispatchConfirmed({
      commandId: "window.close",
      payload: { window_id: id, minimize: true },
    });
  };

  private restoreWindow = (id: string): void => {
    const window = this.view.snapshot?.windows[id];
    if (!window) return;
    this.dispatch(restoreWindowCommand(window));
  };

  private zoomWindow = (id: string): void => {
    this.dispatch({ commandId: "window.zoom", payload: { window_id: id } });
  };

  private toggleFloating = (id: string): void => {
    this.dispatch({ commandId: "window.toggle_floating", payload: { window_id: id } });
  };

  private moveWindow = (id: string, input: MoveWindowInput): void => {
    this.dispatch(moveWindowCommand(id, input, this.workArea()));
  };

  private arrangeLayout = (anchorId: string, preset: OsArrangePreset): void => {
    const anchor = this.view.windows[anchorId];
    if (!anchor) return;
    const peers = arrangePeerWindows(this.view.windows, anchorId);
    const command = arrangeLayoutCommand(anchor, peers, preset, randomWindowManagerId("group"));
    if (command !== null) this.dispatch(command);
  };

  private commitFloatingRect = (id: string, rect: OsRect): void => {
    const window = this.view.windows[id];
    if (!window) return;
    const area = this.workArea();
    const clamped = clampFloatingRect({ proposedRect: rect, workArea: area });
    this.moveWindow(id, {
      destinationDesktopId: window.desktopId,
      placement: "floating",
      floatingRect: clamped,
    });
  };

  private resizeLayout = (splitId: string, boundaryIndex: number, delta: number): void => {
    this.dispatch({
      commandId: "layout.resize",
      payload: { split_id: splitId, boundary_index: boundaryIndex, delta },
      rebase: { splitId, boundaryIndex },
    });
  };

  private balanceLayout = (groupId?: string, splitId?: string): void => {
    this.dispatch({
      commandId: "layout.balance",
      payload: {
        ...(groupId ? { group_id: groupId } : {}),
        ...(splitId ? { split_id: splitId } : {}),
      },
    });
  };

  private navigateWindow = (id: string, route: OsWindowRoute): void => {
    this.dispatch({
      commandId: "window.navigate",
      payload: { window_id: id, route },
    });
  };

  private toggleRailGroup = (agentId: string): void => {
    const normalized = agentId.trim();
    if (!normalized) return;
    this.railCollapsedAgentIds = this.railCollapsedAgentIds.includes(normalized)
      ? this.railCollapsedAgentIds.filter(id => id !== normalized)
      : [...this.railCollapsedAgentIds, normalized];
    this.publish();
  };

  private setWallpaper = (wallpaper: OsWallpaper): void => {
    this.wallpaper = wallpaper;
    this.publish();
  };

  private setDockMagnify = (on: boolean): void => {
    this.dockMagnify = on;
    this.publish();
  };

  private setReduceMotion = (on: boolean): void => {
    this.reduceMotion = on;
    this.publish();
  };

  private setDesktopBounds = (bounds: OsDesktopBounds): void => {
    if (bounds.width <= 0 || bounds.height <= 0) return;
    windowManagerStore.getState().actions.setWorkArea({
      rect: { x: 0, y: 0, w: bounds.width, h: bounds.height },
    });
  };
}
