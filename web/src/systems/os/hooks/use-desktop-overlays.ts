import { useState } from "react";

import { useWindowManagerActions, useWindowManagerOverlay } from "./use-window-manager-store";

export type DesktopOverlay =
  | "workspace-menu"
  | "session-menu"
  | "view-menu"
  | "help-menu"
  | "bell"
  | "palette"
  | "workspaces"
  | "desktops"
  | "sessions";

type LocalDesktopOverlay = Exclude<DesktopOverlay, "desktops">;

/** One owner for shell overlays; Desktops Overview lives in the WM interaction store. */
export function useDesktopOverlays() {
  const [localOverlay, setLocalOverlay] = useState<LocalDesktopOverlay | null>(null);
  const windowManagerOverlay = useWindowManagerOverlay();
  const actions = useWindowManagerActions();
  const activeOverlay: DesktopOverlay | null =
    windowManagerOverlay?.kind === "desktops-overview" ? "desktops" : localOverlay;

  const setOverlayOpen = (overlay: DesktopOverlay, open: boolean) => {
    if (overlay === "desktops") {
      setLocalOverlay(null);
      if (open) actions.openOverlay({ kind: "desktops-overview" });
      else actions.closeOverlay();
      return;
    }
    if (open) actions.closeOverlay();
    setLocalOverlay(current => {
      if (open) return overlay;
      return current === overlay ? null : current;
    });
  };

  const toggleOverlay = (overlay: DesktopOverlay) => {
    setOverlayOpen(overlay, activeOverlay !== overlay);
  };

  return { activeOverlay, setOverlayOpen, toggleOverlay };
}
