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
import {
  resolveWindowManagerActions,
  type WindowManagerActionId,
} from "../lib/window-manager-command-registry";
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
  onOpenDesktops: () => void;
  onOpenWorkspaces: () => void;
  activeOverlay: DesktopOverlay | null;
  onOverlayOpenChange: (overlay: DesktopOverlay, open: boolean) => void;
  attention: OsAttentionModel;
}

function workspaceMonogram(name: string): string {
  return name.trim().slice(0, 2).toUpperCase() || "WS";
}

const SHELL_SHORTCUT_ROWS: Array<{ id: string; keys: string; label: string }> = [
  { id: "palette.open", keys: "⌘K", label: "Command palette" },
  { id: "session.new", keys: "⌘N", label: "New session" },
  { id: "overlay.close", keys: "Esc", label: "Close overlay" },
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
  onOpenDesktops,
  onOpenWorkspaces,
  activeOverlay,
  onOverlayOpenChange,
  attention,
}: DesktopMenubarProps) {
  const { manager, coordinator } = useOsShell();
  const focusedId = useDesktop(state => state.focusedId);
  const hydration = useDesktop(state => state.hydration);
  const windowManagerConfig = useDesktop(state => state.windowManagerConfig);
  const hasFocusedWindow = focusedId !== null;
  const windowManagerActions = resolveWindowManagerActions(windowManagerConfig?.shortcuts ?? {});
  const shortcutFor = (actionId: WindowManagerActionId) =>
    windowManagerActions.find(action => action.id === actionId)?.shortcutLabel ?? null;
  const shortcutRows = [
    ...SHELL_SHORTCUT_ROWS,
    ...windowManagerActions.flatMap(action =>
      action.shortcutLabel
        ? [{ id: action.id, keys: action.shortcutLabel, label: action.label }]
        : []
    ),
  ];

  const workspaceName = activeWorkspace?.name ?? "—";
  const focusAttentionRow = (row: OsAttentionRow) => {
    onOverlayOpenChange("bell", false);
    if (row.kind === "session") {
      coordinator.userOpen({
        app: "session",
        instanceKey: row.id,
        route: {
          pathname: `/agents/${encodeURIComponent(row.agentName)}/sessions/${encodeURIComponent(row.id)}`,
          search: {},
        },
      });
      return;
    }
    coordinator.userOpen({
      app: "tasks",
      route: { pathname: `/tasks/${encodeURIComponent(row.id)}`, search: {} },
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
                  <DropdownMenuItem
                    data-testid="os-menu-desktops-overview"
                    onClick={onOpenDesktops}
                  >
                    Desktops overview
                    {shortcutFor("desktop.overview") ? (
                      <DropdownMenuShortcut>{shortcutFor("desktop.overview")}</DropdownMenuShortcut>
                    ) : null}
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    data-testid="os-menu-workspaces-overview"
                    onClick={onOpenWorkspaces}
                  >
                    Workspaces…
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    data-testid="os-menu-appearance"
                    onClick={() =>
                      coordinator.userOpen({
                        app: "settings",
                        route: { pathname: "/settings/appearance", search: {} },
                      })
                    }
                  >
                    Appearance…
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    disabled={!hasFocusedWindow}
                    onClick={() => {
                      if (focusedId !== null) void coordinator.userMinimize(focusedId);
                    }}
                  >
                    Minimize window
                    {shortcutFor("window.minimize") ? (
                      <DropdownMenuShortcut>{shortcutFor("window.minimize")}</DropdownMenuShortcut>
                    ) : null}
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    disabled={!hasFocusedWindow}
                    onClick={() => focusedId !== null && manager.getState().zoomWindow(focusedId)}
                  >
                    Zoom window
                    {shortcutFor("window.zoom") ? (
                      <DropdownMenuShortcut>{shortcutFor("window.zoom")}</DropdownMenuShortcut>
                    ) : null}
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    disabled={!hasFocusedWindow}
                    onClick={() => {
                      if (focusedId !== null) void coordinator.userClose(focusedId);
                    }}
                  >
                    Close window
                    {shortcutFor("window.close") ? (
                      <DropdownMenuShortcut>{shortcutFor("window.close")}</DropdownMenuShortcut>
                    ) : null}
                  </DropdownMenuItem>
                </>
              ) : null}
              {menu === "Help" ? (
                <div className="flex flex-col gap-1 px-2 py-1.5" data-testid="os-help-shortcuts">
                  {shortcutRows.map(row => (
                    <div
                      key={row.id}
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
          <PopoverContent
            align="end"
            className="w-80 max-w-[calc(100vw-16px)] p-2"
            data-testid="os-bell-popover"
          >
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
