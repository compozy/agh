import { useState } from "react";

export type DesktopOverlay =
  | "workspace-menu"
  | "session-menu"
  | "view-menu"
  | "help-menu"
  | "bell"
  | "palette"
  | "spaces"
  | "sessions";

/**
 * Single owner for shell popovers and desktop-level overlays. Opening one
 * surface closes the previous owner; stale close events cannot dismiss the
 * surface that replaced it.
 */
export function useDesktopOverlays() {
  const [activeOverlay, setActiveOverlay] = useState<DesktopOverlay | null>(null);

  const setOverlayOpen = (overlay: DesktopOverlay, open: boolean) => {
    setActiveOverlay(current => {
      if (open) return overlay;
      return current === overlay ? null : current;
    });
  };

  const toggleOverlay = (overlay: DesktopOverlay) => {
    setActiveOverlay(current => (current === overlay ? null : overlay));
  };

  return { activeOverlay, setOverlayOpen, toggleOverlay };
}
