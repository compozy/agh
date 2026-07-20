import { ChevronRight, PanelsTopLeft } from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  Icon,
  Pill,
} from "@agh/ui";

import type { WorkspacePayload } from "@/systems/workspace";

import type { OsWallpaper, OsWindow } from "../lib/os-types";

export interface OsSpacesOverviewProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workspaces: WorkspacePayload[];
  activeWorkspaceId: string | null;
  activeWallpaper: OsWallpaper;
  activeWindows: OsWindow[];
  onSelectWorkspace: (workspaceId: string) => void;
}

function countLabel(count: number, noun: string): string {
  return `${count} ${noun}${count === 1 ? "" : "s"}`;
}

function describeActiveWindows(windows: OsWindow[]): string {
  if (windows.length === 0) return "No windows open";
  const visible = windows.filter(window => !window.minimized).length;
  const minimized = windows.length - visible;
  if (minimized === 0) return `${countLabel(visible, "window")} visible`;
  if (visible === 0) return `${countLabel(minimized, "window")} minimized`;
  return `${visible} visible · ${minimized} minimized`;
}

/**
 * Task 04's truthful Spaces seam: real workspaces are switchable now, while
 * only the active workspace exposes arrangement data currently loaded by the
 * desktop store. Task 08 replaces the inactive-state copy with persisted live
 * thumbnails once per-workspace arrangements are available client-side.
 */
export function OsSpacesOverview({
  open,
  onOpenChange,
  workspaces,
  activeWorkspaceId,
  activeWallpaper,
  activeWindows,
  onSelectWorkspace,
}: OsSpacesOverviewProps) {
  const selectWorkspace = (workspaceId: string) => {
    if (workspaceId !== activeWorkspaceId) onSelectWorkspace(workspaceId);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        unframed
        className="grid max-h-[calc(100vh-2rem)] grid-rows-[auto_minmax(0,1fr)] sm:max-w-(--width-modal-md)"
        data-testid="os-spaces-overview"
      >
        <DialogHeader variant="ruled">
          <DialogTitle>Spaces overview</DialogTitle>
          <DialogDescription>
            Switch workspaces. The active space shows its current desktop state.
          </DialogDescription>
        </DialogHeader>

        <div className="overflow-y-auto p-4">
          <div className="grid gap-2 sm:grid-cols-2">
            {workspaces.map(workspace => {
              const active = workspace.id === activeWorkspaceId;
              const detail = active
                ? `${describeActiveWindows(activeWindows)} · ${activeWallpaper} wallpaper`
                : "Select to load this space";

              return (
                <button
                  key={workspace.id}
                  type="button"
                  data-workspace-id={workspace.id}
                  data-current={active ? "true" : undefined}
                  aria-label={`${active ? "Current workspace" : "Switch to"} ${workspace.name}. ${detail}`}
                  className={[
                    "group flex min-w-0 items-center gap-3 rounded-lg border px-3 py-3 text-left",
                    "transition-colors duration-base hover:bg-row-hover",
                    "focus-visible:shadow-focus-ring focus-visible:outline-none",
                    active
                      ? "border-line-focus bg-surface-glaze shadow-highlight"
                      : "border-line bg-canvas",
                  ].join(" ")}
                  onClick={() => selectWorkspace(workspace.id)}
                >
                  <span className="grid size-9 shrink-0 place-items-center rounded-md border border-line-strong bg-elevated text-muted">
                    <Icon as={PanelsTopLeft} size="lg" aria-hidden="true" />
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="flex min-w-0 items-center gap-2">
                      <span className="truncate text-small-body font-semibold text-fg-strong">
                        {workspace.name}
                      </span>
                      {active ? (
                        <Pill tone="neutral" size="xs">
                          Current
                        </Pill>
                      ) : null}
                    </span>
                    <span className="mt-0.5 block truncate text-micro text-muted">{detail}</span>
                  </span>
                  <Icon
                    as={ChevronRight}
                    size="sm"
                    aria-hidden="true"
                    className="shrink-0 text-subtle transition-colors duration-base group-hover:text-fg"
                  />
                </button>
              );
            })}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
