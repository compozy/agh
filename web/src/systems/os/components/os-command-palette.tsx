import {
  Columns2,
  LayoutGrid,
  Maximize2,
  Minus,
  Palette,
  PanelBottom,
  PanelLeft,
  PanelRight,
  PanelTop,
  PanelsTopLeft,
  Plus,
  RefreshCcw,
  SquareArrowDownLeft,
  SquareArrowDownRight,
  SquareArrowUpLeft,
  SquareArrowUpRight,
  Undo2,
  X,
  type LucideIcon,
} from "lucide-react";

import {
  Command,
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandShortcut,
} from "@agh/ui";

import { useOsCommandPalette } from "../hooks/use-os-command-palette";
import { OS_APPS } from "../lib/app-registry";
import type { OsSnapZoneId } from "../lib/os-types";

export interface OsCommandPaletteProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Opens the Spaces overview overlay (⇧⌘S parity — US-019.EC-3 fallback). */
  onOpenSpaces?: () => void;
}

/** Apps the palette can open: every registry app except multi-instance sessions. */
const PALETTE_APPS = Object.values(OS_APPS).filter(app => app.id !== "session");

const SNAP_COMMAND_ICONS: Record<OsSnapZoneId, LucideIcon> = {
  left: PanelLeft,
  right: PanelRight,
  top: PanelTop,
  bottom: PanelBottom,
  "top-left": SquareArrowUpLeft,
  "top-right": SquareArrowUpRight,
  "bottom-left": SquareArrowDownLeft,
  "bottom-right": SquareArrowDownRight,
};

const ARRANGE_COMMAND_ICONS: Record<"two-up" | "grid", LucideIcon> = {
  "two-up": Columns2,
  grid: LayoutGrid,
};

/**
 * The global ⌘K palette (ADR-005): apps, live sessions, window snap actions
 * (ADR-009 — the guaranteed keyboard path), and shell actions. A desktop-level
 * overlay (closed set) — it renders above the win-layer and portals to the
 * document body, never into a window. Behavior lives in `useOsCommandPalette`.
 */
export function OsCommandPalette({ open, onOpenChange, onOpenSpaces }: OsCommandPaletteProps) {
  const model = useOsCommandPalette(open, onOpenChange, { onOpenSpaces });

  return (
    <CommandDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Command palette"
      description="Search apps, sessions, and actions"
      // Compact clamps to the phone canvas (os-v2.css `.palette-overlay{padding-top:9vh}`).
      className="top-[9vh] min-[960px]:top-[16vh] sm:max-w-(--width-modal-sm)"
    >
      <Command data-testid="os-command-palette" shouldFilter>
        <CommandInput autoFocus placeholder="Search apps, sessions, actions…" />
        <CommandList className="max-h-[46vh]">
          <CommandEmpty>No matches — try an app, a session title, or an action.</CommandEmpty>
          <CommandGroup heading="Apps">
            {PALETTE_APPS.map(app => (
              <CommandItem
                key={app.id}
                value={`open ${app.title}`}
                data-testid={`os-palette-app-${app.id}`}
                onSelect={() => model.openApp(app.id)}
              >
                <app.icon className="size-3.5 text-muted" />
                Open {app.title}
              </CommandItem>
            ))}
          </CommandGroup>
          {model.paletteSessions.length > 0 ? (
            <CommandGroup heading="Sessions">
              {model.paletteSessions.map(session => (
                <CommandItem
                  key={session.id}
                  value={`session ${session.name ?? session.id} ${session.agent_name}`}
                  data-testid={`os-palette-session-${session.id}`}
                  onSelect={() => model.jumpToSession(session.id, session.agent_name ?? "")}
                >
                  <OS_APPS.session.icon className="size-3.5 text-muted" />
                  <span className="min-w-0 truncate">{session.name?.trim() || session.id}</span>
                  <span className="ml-auto text-micro text-subtle">{session.agent_name}</span>
                </CommandItem>
              ))}
            </CommandGroup>
          ) : null}
          {model.focusedWindowActions !== null || model.snapCommands.length > 0 ? (
            <CommandGroup heading="Window">
              {model.focusedWindowActions !== null ? (
                <>
                  <CommandItem
                    value="close window"
                    data-testid="os-palette-close-window"
                    onSelect={model.focusedWindowActions.close}
                  >
                    <X className="size-3.5 text-muted" />
                    Close window
                    <CommandShortcut>⌘W</CommandShortcut>
                  </CommandItem>
                  <CommandItem
                    value="minimize window"
                    data-testid="os-palette-minimize-window"
                    onSelect={model.focusedWindowActions.minimize}
                  >
                    <Minus className="size-3.5 text-muted" />
                    Minimize window
                    <CommandShortcut>⌘M</CommandShortcut>
                  </CommandItem>
                  {model.focusedWindowActions.zoom !== null ? (
                    <CommandItem
                      value="zoom window"
                      data-testid="os-palette-zoom-window"
                      onSelect={model.focusedWindowActions.zoom}
                    >
                      <Maximize2 className="size-3.5 text-muted" />
                      Zoom window
                    </CommandItem>
                  ) : null}
                </>
              ) : null}
              {model.snapCommands.map(command => {
                const Icon = command.zoneId === null ? Undo2 : SNAP_COMMAND_ICONS[command.zoneId];
                const testId =
                  command.zoneId === null
                    ? "os-palette-snap-restore"
                    : `os-palette-snap-${command.zoneId}`;
                return (
                  <CommandItem
                    key={command.label}
                    value={command.label}
                    data-testid={testId}
                    onSelect={() => model.dispatchSnap(command)}
                  >
                    <Icon className="size-3.5 text-muted" />
                    {command.label}
                    {command.keys ? <CommandShortcut>{command.keys}</CommandShortcut> : null}
                  </CommandItem>
                );
              })}
              {model.arrangeCommands.map(command => {
                const Icon = ARRANGE_COMMAND_ICONS[command.preset];
                return (
                  <CommandItem
                    key={command.preset}
                    value={command.label}
                    data-testid={`os-palette-arrange-${command.preset}`}
                    onSelect={() => model.dispatchArrange(command)}
                  >
                    <Icon className="size-3.5 text-muted" />
                    {command.label}
                  </CommandItem>
                );
              })}
            </CommandGroup>
          ) : null}
          <CommandGroup heading="Actions">
            <CommandItem
              value="toggle sessions rail"
              data-testid="os-palette-toggle-sessions"
              onSelect={model.toggleRail}
            >
              <OS_APPS.session.icon className="size-3.5 text-muted" />
              Toggle sessions
            </CommandItem>
            <CommandItem
              value="new session"
              data-testid="os-palette-new-session"
              onSelect={model.newSession}
            >
              <Plus className="size-3.5 text-muted" />
              New session
              <CommandShortcut>⌘N</CommandShortcut>
            </CommandItem>
            <CommandItem
              value="spaces overview"
              data-testid="os-palette-spaces-overview"
              onSelect={model.openSpaces}
            >
              <PanelsTopLeft className="size-3.5 text-muted" />
              Spaces overview
              <CommandShortcut>⇧⌘S</CommandShortcut>
            </CommandItem>
            <CommandItem
              value="appearance wallpaper motion"
              data-testid="os-palette-appearance"
              onSelect={model.openAppearance}
            >
              <Palette className="size-3.5 text-muted" />
              Appearance
            </CommandItem>
            {model.workspaces.flatMap(workspace =>
              workspace.id === model.activeWorkspaceId
                ? []
                : [
                    <CommandItem
                      key={workspace.id}
                      value={`switch to ${workspace.name}`}
                      data-testid={`os-palette-workspace-${workspace.id}`}
                      onSelect={() => model.switchWorkspace(workspace.id)}
                    >
                      <RefreshCcw className="size-3.5 text-muted" />
                      Switch to {workspace.name}
                    </CommandItem>,
                  ]
            )}
          </CommandGroup>
        </CommandList>
      </Command>
    </CommandDialog>
  );
}
