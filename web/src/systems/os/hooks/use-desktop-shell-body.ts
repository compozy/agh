import { useRef } from "react";
import { useShallow } from "zustand/shallow";

import { useDesktop } from "./use-desktop";
import { useDesktopManagerSurfaces } from "./use-desktop-manager-surfaces";
import { useDesktopOverlays } from "./use-desktop-overlays";
import { useDesktopShellState } from "./use-desktop-shell-state";
import type { DesktopShellModel } from "./use-desktop-shell-model";
import {
  useDesktopTransitionIntent,
  useWindowManagerActions,
  useWindowManagerGesturePreview,
} from "./use-window-manager-store";
import { useOsAttention } from "./use-os-attention";
import { useOsReducedMotion } from "./use-os-reduced-motion";
import { useOsShell } from "./use-os-shell";
import { useOsShortcuts } from "./use-os-shortcuts";
import { useOsWinLayer } from "./use-os-win-layer";
import { useWorkspaceDetails } from "./use-workspace-details";

/** Composes the live runtime models consumed by the presentational desktop shell body. */
export function useDesktopShellBody(model: DesktopShellModel) {
  const desktopRef = useRef<HTMLDivElement>(null);
  const desktop = useDesktopShellState();
  const overlays = useDesktopOverlays();
  const attention = useOsAttention(model.activeWorkspace, model.sessionCatalogStreamStatus);
  const managerSurfaces = useDesktopManagerSurfaces();
  const winLayer = useOsWinLayer();
  const reducedMotion = useOsReducedMotion();
  const transition = useDesktopTransitionIntent();
  const gesturePreview = useWindowManagerGesturePreview();
  const windowManagerActions = useWindowManagerActions();
  const { manager } = useOsShell();
  const pager = useDesktop(
    useShallow(state => ({
      activeDesktopId: state.activeDesktopId,
      desktops: state.desktops,
      compact: state.presentation === "compact",
      presentation: state.presentation,
      canSwitchDesktop:
        state.client !== null &&
        state.hydration === "live" &&
        state.connectionStatus === "connected",
    }))
  );
  const workspaceDetails = useWorkspaceDetails(
    model.workspaces.map(workspace => workspace.id),
    { enabled: overlays.activeOverlay === "workspaces" }
  );

  useOsShortcuts({
    onPalette: () => overlays.toggleOverlay("palette"),
    onNewSession: () => model.sessionCreate.openForAgent(""),
    onDesktops: () => overlays.toggleOverlay("desktops"),
    onEscape: () => {
      if (overlays.activeOverlay !== null) return;
      if (document.querySelector('[data-slot="dialog-content"]')) return;
      desktopRef.current?.focus();
    },
  });

  return {
    attention,
    desktop,
    desktopRef,
    gesturePreview,
    manager,
    managerSurfaces,
    overlays,
    pager,
    reducedMotion,
    transition,
    windowManagerActions,
    winLayer,
    workspaceDetails,
  };
}
