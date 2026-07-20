import { Plus, RefreshCcw } from "lucide-react";

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

import { useSessionCreate, useSessions } from "@/systems/session";
import { useActiveWorkspace } from "@/systems/workspace";

import { useOsShell } from "../hooks/use-os-shell";
import { OS_APPS } from "../lib/app-registry";
import type { OsAppId } from "../lib/os-types";

export interface OsCommandPaletteProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/** Apps the palette can open: every registry app except multi-instance sessions. */
const PALETTE_APPS = Object.values(OS_APPS).filter(app => app.id !== "session");

/**
 * The global ⌘K palette (ADR-005): apps, live sessions, and actions. A
 * desktop-level overlay (closed set) — it renders above the win-layer and
 * portals to the document body, never into a window.
 */
export function OsCommandPalette({ open, onOpenChange }: OsCommandPaletteProps) {
  const { coordinator } = useOsShell();
  const sessionCreate = useSessionCreate();
  const { workspaces, activeWorkspaceId, setActiveWorkspaceId } = useActiveWorkspace();
  const sessions = useSessions(activeWorkspaceId, {
    enabled: open && activeWorkspaceId !== null,
  });

  const run = (action: () => void) => {
    onOpenChange(false);
    action();
  };

  const openApp = (app: OsAppId) => run(() => coordinator.userOpen({ app }));

  const jumpToSession = (sessionId: string, agentName: string) =>
    run(() =>
      coordinator.userOpen({
        app: "session",
        instanceKey: sessionId,
        location: {
          pathname: `/agents/${encodeURIComponent(agentName)}/sessions/${encodeURIComponent(sessionId)}`,
          search: {},
        },
      })
    );

  const paletteSessions = (sessions.data ?? []).filter(
    session => typeof session.agent_name === "string" && session.agent_name.length > 0
  );

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
                onSelect={() => openApp(app.id)}
              >
                <app.icon className="size-3.5 text-muted" />
                Open {app.title}
              </CommandItem>
            ))}
          </CommandGroup>
          {paletteSessions.length > 0 ? (
            <CommandGroup heading="Sessions">
              {paletteSessions.map(session => (
                <CommandItem
                  key={session.id}
                  value={`session ${session.name ?? session.id} ${session.agent_name}`}
                  data-testid={`os-palette-session-${session.id}`}
                  onSelect={() => jumpToSession(session.id, session.agent_name ?? "")}
                >
                  <OS_APPS.session.icon className="size-3.5 text-muted" />
                  <span className="min-w-0 truncate">{session.name?.trim() || session.id}</span>
                  <span className="ml-auto text-micro text-subtle">{session.agent_name}</span>
                </CommandItem>
              ))}
            </CommandGroup>
          ) : null}
          <CommandGroup heading="Actions">
            <CommandItem
              value="new session"
              data-testid="os-palette-new-session"
              onSelect={() => run(() => sessionCreate.openForAgent(""))}
            >
              <Plus className="size-3.5 text-muted" />
              New session
              <CommandShortcut>⌘N</CommandShortcut>
            </CommandItem>
            {workspaces.flatMap(workspace =>
              workspace.id === activeWorkspaceId
                ? []
                : [
                    <CommandItem
                      key={workspace.id}
                      value={`switch to ${workspace.name}`}
                      data-testid={`os-palette-workspace-${workspace.id}`}
                      onSelect={() => run(() => setActiveWorkspaceId(workspace.id))}
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
