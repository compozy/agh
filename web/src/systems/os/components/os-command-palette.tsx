import {
  Plus,
  RefreshCcw,
  PanelLeft,
  PanelRight,
  SquareArrowDownLeft,
  SquareArrowDownRight,
  SquareArrowUpLeft,
  SquareArrowUpRight,
  Undo2,
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
}

/** Apps the palette can open: every registry app except multi-instance sessions. */
const PALETTE_APPS = Object.values(OS_APPS).filter(app => app.id !== "session");

const SNAP_COMMAND_ICONS: Record<OsSnapZoneId, LucideIcon> = {
  left: PanelLeft,
  right: PanelRight,
  "top-left": SquareArrowUpLeft,
  "top-right": SquareArrowUpRight,
  "bottom-left": SquareArrowDownLeft,
  "bottom-right": SquareArrowDownRight,
};

/**
 * The global ⌘K palette (ADR-005): apps, live sessions, window snap actions
 * (ADR-009 — the guaranteed keyboard path), and shell actions. A desktop-level
 * overlay (closed set) — it renders above the win-layer and portals to the
 * document body, never into a window. Behavior lives in `useOsCommandPalette`.
 */
export function OsCommandPalette({ open, onOpenChange }: OsCommandPaletteProps) {
  const model = useOsCommandPalette(open, onOpenChange);

  return (
    <CommandDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Command palette"
      description="Search apps, sessions, and actions"
      className="top-[16vh] sm:max-w-(--width-modal-sm)"
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
          {model.snapCommands.length > 0 ? (
            <CommandGroup heading="Window">
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
                    <CommandShortcut>{command.keys}</CommandShortcut>
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
