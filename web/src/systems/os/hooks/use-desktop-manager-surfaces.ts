import { useShallow } from "zustand/shallow";

import { getOsApp } from "../lib/app-registry";
import { useDesktop } from "./use-desktop";
import { useOsShell } from "./use-os-shell";
import {
  useDesktopOverviewSegmentRequest,
  usePendingWindowManagerCommand,
  useWindowManagerActions,
  useWindowManagerConflict,
  useWindowManagerDiagnostic,
  useWindowManagerOverlay,
} from "./use-window-manager-store";

/** Runtime inputs for the desktop overview and connection diagnostic surfaces. */
export function useDesktopManagerSurfaces() {
  const { manager } = useOsShell();
  const actions = useWindowManagerActions();
  const overlay = useWindowManagerOverlay();
  const overviewSegmentRequest = useDesktopOverviewSegmentRequest();
  const conflict = useWindowManagerConflict();
  const diagnostic = useWindowManagerDiagnostic();
  const pending = usePendingWindowManagerCommand();
  const hydration = useDesktop(state => state.hydration);
  const connectionStatus = useDesktop(state => state.connectionStatus);
  const activeDesktopId = useDesktop(state => state.activeDesktopId);
  const projection = useDesktop(
    useShallow(state => ({
      desktops: state.desktops,
      projections: state.projections,
      windows: state.windows,
    }))
  );
  const desktops = projection.desktops.map(desktop => {
    const windowRecords = Object.values(projection.windows).filter(
      window => window.desktopId === desktop.id
    );
    return {
      id: desktop.id,
      name: desktop.name,
      purpose: desktop.purpose,
      projection: projection.projections[desktop.id],
      windowRecords,
      windows: windowRecords.map(window => ({
        id: window.id,
        title: getOsApp(window.app).title,
        detail: window.instanceKey ?? undefined,
      })),
    };
  });

  return {
    actions,
    activeDesktopId,
    connectionStatus,
    conflict,
    desktops,
    diagnostic,
    hydration,
    manager,
    overlay,
    overviewSegmentRequest,
    pending,
  };
}
