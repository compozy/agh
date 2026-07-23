import { AlertTriangle, WifiOff } from "lucide-react";

import { cn } from "@agh/ui";

import { useDesktopManagerSurfaces } from "../hooks/use-desktop-manager-surfaces";
import { useDesktop } from "../hooks/use-desktop";
import { DesktopLayoutThumbnail } from "./desktop-layout-thumbnail";
import {
  DesktopsOverview,
  type DesktopOverviewItem,
  type DesktopsOverviewState,
} from "./desktops-overview";

function overviewState(input: {
  hydration: "pending" | "live" | "degraded";
  activeDesktopId: string | null;
  desktops: readonly DesktopOverviewItem[];
  conflictMessage: string | null;
  diagnosticMessage: string | null;
}): DesktopsOverviewState {
  if (input.conflictMessage) return { status: "conflict", message: input.conflictMessage };
  if (input.hydration === "pending") return { status: "loading" };
  if (input.hydration === "degraded" && input.desktops.length === 0) {
    return {
      status: "error",
      message: input.diagnosticMessage ?? "The daemon window layout is unavailable.",
    };
  }
  return {
    status: "ready",
    desktops: input.desktops,
    activeDesktopId: input.activeDesktopId,
  };
}

/** Management overview and honest daemon-connection feedback. */
export function DesktopManagerSurfaces() {
  const model = useDesktopManagerSurfaces();
  const canMutate = useDesktop(
    state =>
      state.client !== null && state.hydration === "live" && state.connectionStatus === "connected"
  );
  const desktops: DesktopOverviewItem[] = model.desktops.map(desktop => ({
    id: desktop.id,
    name: desktop.name,
    purpose: desktop.purpose,
    thumbnail: (
      <DesktopLayoutThumbnail projection={desktop.projection} windows={desktop.windowRecords} />
    ),
    windows: desktop.windows,
  }));
  const overview = overviewState({
    hydration: model.hydration,
    activeDesktopId: model.activeDesktopId,
    desktops,
    conflictMessage: model.conflict
      ? `Revision ${model.conflict.expectedRevision} is stale; the daemon is at revision ${model.conflict.currentRevision}.`
      : null,
    diagnosticMessage: model.diagnostic?.message ?? null,
  });

  return (
    <>
      <DesktopsOverview
        open={model.overlay?.kind === "desktops-overview"}
        state={overview}
        initialFocusSegment={model.overviewSegmentRequest}
        busy={model.pending !== null}
        canMutate={canMutate}
        onOpenChange={open => {
          if (open) model.actions.openOverlay({ kind: "desktops-overview" });
          else model.actions.closeOverlay();
        }}
        onCreateDesktop={() => model.manager.createDesktop()}
        onSwitchDesktop={desktopId => model.manager.switchDesktop(desktopId)}
        onRenameDesktop={(desktopId, name) => model.manager.renameDesktop(desktopId, name)}
        onReorderDesktop={(desktopId, order) => model.manager.reorderDesktop(desktopId, order)}
        onDeleteDesktop={(desktopId, destinationId) =>
          model.manager.deleteDesktop(desktopId, destinationId)
        }
        onMoveWindow={(windowId, _sourceDesktopId, destinationDesktopId) =>
          model.manager.moveWindowToDesktop(windowId, destinationDesktopId)
        }
        onRetry={() => model.manager.refreshSnapshot()}
        onResolveConflict={() => {
          model.manager.clearConflict();
          model.manager.refreshSnapshot();
        }}
      />
      {model.hydration !== "pending" && model.connectionStatus !== "connected" ? (
        <div
          role="status"
          className={cn(
            "pointer-events-none absolute top-2 right-2 z-30 flex items-center gap-1.5",
            "rounded-pill border border-line-strong bg-shell-glass px-2.5 py-1",
            "text-form-hint text-muted backdrop-blur-shell"
          )}
        >
          {model.diagnostic ? (
            <AlertTriangle aria-hidden="true" className="size-3 text-warning" />
          ) : (
            <WifiOff aria-hidden="true" className="size-3 text-subtle" />
          )}
          {model.diagnostic?.message ??
            (model.connectionStatus === "reconnecting"
              ? "Layout reconnecting"
              : "Live layout disconnected")}
        </div>
      ) : null}
    </>
  );
}
