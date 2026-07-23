import { useShallow } from "zustand/shallow";

import { useDesktop } from "../hooks/use-desktop";
import { useOsShell } from "../hooks/use-os-shell";
import { useWindowManagerActions } from "../hooks/use-window-manager-store";
import { DesktopPager } from "./desktop-pager";

/** Daemon-backed pager controller mounted by the bottom-chrome owner. */
export function DesktopPagerSurface() {
  const { manager } = useOsShell();
  const actions = useWindowManagerActions();
  const { activeDesktopId, desktops, compact, canSwitchDesktop } = useDesktop(
    useShallow(state => ({
      activeDesktopId: state.activeDesktopId,
      desktops: state.desktops,
      compact: state.presentation === "compact",
      canSwitchDesktop:
        state.client !== null &&
        state.hydration === "live" &&
        state.connectionStatus === "connected",
    }))
  );

  if (!activeDesktopId) return null;

  return (
    <DesktopPager
      desktops={desktops.map(desktop => ({ id: desktop.id, name: desktop.name }))}
      activeDesktopId={activeDesktopId}
      compact={compact}
      canSwitchDesktop={canSwitchDesktop}
      onSelectDesktop={desktopId => manager.switchDesktop(desktopId)}
      onOpenOverview={request => actions.requestOverviewSegment(request)}
    />
  );
}
