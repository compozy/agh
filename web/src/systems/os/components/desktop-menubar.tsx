import { Check } from "lucide-react";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
  Kbd,
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@agh/ui";

import type { WorkspacePayload } from "@/systems/workspace";

import type { OsAttentionModel } from "../hooks/use-os-attention";
import type { DesktopOverlay } from "../hooks/use-desktop-overlays";
import type { OsAttentionRow } from "../lib/attention-model";
import { useOsShell } from "../hooks/use-os-shell";
import { useDesktop } from "../hooks/use-desktop";
import { OsHydrationStatus } from "./os-hydration-status";
import { OsMenuBar } from "./os-menubar";
import { AttentionBell } from "./attention-bell";

export interface DesktopMenubarProps {
  workspaces: WorkspacePayload[];
  activeWorkspace: WorkspacePayload | undefined;
  onSelectWorkspace: (workspaceId: string) => void;
  onAddWorkspace: () => void;
  onNewSession: () => void;
  onOpenPalette: () => void;
  onOpenSpaces: () => void;
  activeOverlay: DesktopOverlay | null;
  onOverlayOpenChange: (overlay: DesktopOverlay, open: boolean) => void;
  attention: OsAttentionModel;
}

function workspaceMonogram(name: string): string {
  return name.trim().slice(0, 2).toUpperCase() || "WS";
}

const SHORTCUT_ROWS: Array<{ keys: string; label: string }> = [
  { keys: "⌘K", label: "Command palette" },
  { keys: "⌘N", label: "New session" },
  { keys: "⇧⌘S", label: "Spaces overview" },
  { keys: "⌘W", label: "Close window" },
  { keys: "⌘M", label: "Minimize window" },
  { keys: "Esc", label: "Close overlay" },
];

function menuOverlay(menu: string): DesktopOverlay | null {
  if (menu === "Session") return "session-menu";
  if (menu === "View") return "view-menu";
  if (menu === "Help") return "help-menu";
  return null;
}

/**
 * The wired menubar: workspace switcher, Session/View/Help menus, the bell
 * aggregator seam (attention rows land with the attention-surfaces task), the
 * ⌘K chip, and the settings cog. All actions are runtime-backed — no menu
 * item renders without a working mechanism (SD-007).
 */
export function DesktopMenubar({
  workspaces,
  activeWorkspace,
  onSelectWorkspace,
  onAddWorkspace,
  onNewSession,
  onOpenPalette,
  onOpenSpaces,
  activeOverlay,
  onOverlayOpenChange,
  attention,
}: DesktopMenubarProps) {
  const { store, coordinator } = useOsShell();
  const focusedId = useDesktop(state => state.focusedId);
  const hydration = useDesktop(state => state.hydration);
  const hasFocusedWindow = focusedId !== null;

  const workspaceName = activeWorkspace?.name ?? "—";
  const focusAttentionRow = (row: OsAttentionRow) => {
    onOverlayOpenChange("bell", false);
    if (row.kind === "session") {
      coordinator.userOpen({
        app: "session",
        instanceKey: row.id,
        location: {
          pathname: `/agents/${encodeURIComponent(row.agentName)}/sessions/${encodeURIComponent(row.id)}`,
          search: {},
        },
      });
      return;
    }
    coordinator.userOpen({
      app: "tasks",
      location: { pathname: `/tasks/${encodeURIComponent(row.id)}`, search: {} },
    });
  };

  return (
    <OsMenuBar
      workspace={{ name: workspaceName, monogram: workspaceMonogram(workspaceName) }}
      status={<OsHydrationStatus hydration={hydration} />}
      notifications={attention.notificationCount}
      onLogoClick={() => coordinator.userOpen({ app: "dashboard" })}
      onCommandClick={onOpenPalette}
      onSettingsClick={() => coordinator.userOpen({ app: "settings" })}
      wrapWorkspaceTrigger={trigger => (
        <DropdownMenu
          open={activeOverlay === "workspace-menu"}
          onOpenChange={open => onOverlayOpenChange("workspace-menu", open)}
        >
          <DropdownMenuTrigger render={trigger} />
          <DropdownMenuContent align="start" data-testid="os-workspace-menu">
            {workspaces.map(workspace => (
              <DropdownMenuItem
                key={workspace.id}
                data-testid={`os-workspace-option-${workspace.id}`}
                onClick={() => onSelectWorkspace(workspace.id)}
              >
                <span className="grid size-4 place-items-center rounded-xs border border-line-strong bg-elevated font-mono text-micro font-semibold">
                  {workspaceMonogram(workspace.name)}
                </span>
                {workspace.name}
                {workspace.id === activeWorkspace?.id ? (
                  <Check className="ml-auto size-3 text-accent" />
                ) : null}
              </DropdownMenuItem>
            ))}
            <DropdownMenuSeparator />
            <DropdownMenuItem data-testid="os-workspace-add" onClick={onAddWorkspace}>
              Add workspace…
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )}
      wrapMenuTrigger={(menu, trigger) => {
        const overlay = menuOverlay(menu);
        if (overlay === null) return trigger;
        return (
          <DropdownMenu
            open={activeOverlay === overlay}
            onOpenChange={open => onOverlayOpenChange(overlay, open)}
          >
            <DropdownMenuTrigger render={trigger} />
            <DropdownMenuContent align="start" data-testid={`os-menu-${menu.toLowerCase()}`}>
              {menu === "Session" ? (
                <DropdownMenuItem data-testid="os-menu-new-session" onClick={onNewSession}>
                  New session
                  <DropdownMenuShortcut>⌘N</DropdownMenuShortcut>
                </DropdownMenuItem>
              ) : null}
              {menu === "View" ? (
                <>
                  <DropdownMenuItem data-testid="os-menu-spaces-overview" onClick={onOpenSpaces}>
                    Spaces overview
                    <DropdownMenuShortcut>⇧⌘S</DropdownMenuShortcut>
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    disabled={!hasFocusedWindow}
                    onClick={() => focusedId !== null && coordinator.userMinimize(focusedId)}
                  >
                    Minimize window
                    <DropdownMenuShortcut>⌘M</DropdownMenuShortcut>
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    disabled={!hasFocusedWindow}
                    onClick={() => focusedId !== null && store.getState().toggleZoom(focusedId)}
                  >
                    Zoom window
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    disabled={!hasFocusedWindow}
                    onClick={() => focusedId !== null && coordinator.userClose(focusedId)}
                  >
                    Close window
                    <DropdownMenuShortcut>⌘W</DropdownMenuShortcut>
                  </DropdownMenuItem>
                </>
              ) : null}
              {menu === "Help" ? (
                <div className="flex flex-col gap-1 px-2 py-1.5" data-testid="os-help-shortcuts">
                  {SHORTCUT_ROWS.map(row => (
                    <div
                      key={row.keys}
                      className="flex items-center justify-between gap-6 text-small-body text-muted"
                    >
                      <span>{row.label}</span>
                      <Kbd>{row.keys}</Kbd>
                    </div>
                  ))}
                </div>
              ) : null}
            </DropdownMenuContent>
          </DropdownMenu>
        );
      }}
      wrapBellTrigger={trigger => (
        <Popover
          open={activeOverlay === "bell"}
          onOpenChange={open => onOverlayOpenChange("bell", open)}
        >
          <PopoverTrigger render={trigger} />
          <PopoverContent align="end" className="w-80 p-2" data-testid="os-bell-popover">
            <AttentionBell
              rows={attention.rows}
              sessionsDisconnected={attention.sessionsDisconnected}
              tasksDisconnected={attention.tasksDisconnected}
              loading={attention.loading}
              onSelect={focusAttentionRow}
            />
          </PopoverContent>
        </Popover>
      )}
    />
  );
}
