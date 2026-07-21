import { useSessionCreate, useSessions } from "@/systems/session";
import { useActiveWorkspace, type WorkspacePayload } from "@/systems/workspace";

import { OS_SNAP_COMMANDS, type OsSnapCommand } from "../lib/os-snap-commands";
import { OS_SNAP_ZONES } from "../lib/os-snap-zones";
import type { OsAppId } from "../lib/os-types";
import { useDesktop } from "./use-desktop";
import { useOsShell } from "./use-os-shell";

type PaletteSession = NonNullable<ReturnType<typeof useSessions>["data"]>[number];

export interface OsCommandPaletteModel {
  paletteSessions: PaletteSession[];
  workspaces: WorkspacePayload[];
  activeWorkspaceId: string | null;
  /** Zone commands for the focused floating window; restore only while snapped (UT-101). */
  snapCommands: readonly OsSnapCommand[];
  openApp(app: OsAppId): void;
  jumpToSession(sessionId: string, agentName: string): void;
  dispatchSnap(command: OsSnapCommand): void;
  toggleRail(): void;
  newSession(): void;
  switchWorkspace(workspaceId: string): void;
}

/**
 * Palette view-model: sources (apps come from the registry constant), the
 * focused-window snap surface (ADR-009), and the run-and-close dispatch
 * pattern. Keeps `OsCommandPalette` purely presentational.
 */
export function useOsCommandPalette(
  open: boolean,
  onOpenChange: (open: boolean) => void
): OsCommandPaletteModel {
  const { coordinator, store } = useOsShell();
  const sessionCreate = useSessionCreate();
  const { workspaces, activeWorkspaceId, setActiveWorkspaceId } = useActiveWorkspace();
  const sessions = useSessions(activeWorkspaceId, {
    enabled: open && activeWorkspaceId !== null,
  });
  // Snap actions exist only in floating presentation (UT-061/UT-101 gating)
  // and act on the focused window; restore renders only while it is snapped.
  const focusedWindow = useDesktop(state =>
    state.presentation === "floating" && state.focusedId !== null
      ? state.windows[state.focusedId]
      : undefined
  );

  const run = (action: () => void) => {
    onOpenChange(false);
    action();
  };

  return {
    paletteSessions: (sessions.data ?? []).filter(
      session => typeof session.agent_name === "string" && session.agent_name.length > 0
    ),
    workspaces,
    activeWorkspaceId,
    snapCommands: focusedWindow
      ? OS_SNAP_COMMANDS.filter(command => command.zoneId !== null || focusedWindow.snap !== null)
      : [],
    openApp: app => run(() => coordinator.userOpen({ app })),
    jumpToSession: (sessionId, agentName) =>
      run(() =>
        coordinator.userOpen({
          app: "session",
          instanceKey: sessionId,
          location: {
            pathname: `/agents/${encodeURIComponent(agentName)}/sessions/${encodeURIComponent(sessionId)}`,
            search: {},
          },
        })
      ),
    dispatchSnap: command =>
      run(() => {
        const state = store.getState();
        if (state.focusedId === null) return;
        state.snapWindow(
          state.focusedId,
          command.zoneId === null ? null : OS_SNAP_ZONES[command.zoneId]
        );
      }),
    toggleRail: () => run(() => store.getState().toggleRail()),
    newSession: () => run(() => sessionCreate.openForAgent("")),
    switchWorkspace: workspaceId => run(() => setActiveWorkspaceId(workspaceId)),
  };
}
