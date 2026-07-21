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
  /**
   * Lifecycle actions for the focused window — the guaranteed keyboard path
   * where browsers reserve ⌘W/⌘M (US-003.AC-4/EC-3). `zoom` is null in
   * compact presentation (the control has no meaning in a stack).
   */
  focusedWindowActions: { close(): void; minimize(): void; zoom: (() => void) | null } | null;
  openApp(app: OsAppId): void;
  jumpToSession(sessionId: string, agentName: string): void;
  dispatchSnap(command: OsSnapCommand): void;
  toggleRail(): void;
  newSession(): void;
  openSpaces(): void;
  openAppearance(): void;
  switchWorkspace(workspaceId: string): void;
}

/**
 * Palette view-model: sources (apps come from the registry constant), the
 * focused-window snap surface (ADR-009), and the run-and-close dispatch
 * pattern. Keeps `OsCommandPalette` purely presentational.
 */
export function useOsCommandPalette(
  open: boolean,
  onOpenChange: (open: boolean) => void,
  options: { onOpenSpaces?: () => void } = {}
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
  const focusedId = useDesktop(state => state.focusedId);
  const presentation = useDesktop(state => state.presentation);

  const run = (action: () => void) => {
    onOpenChange(false);
    action();
  };

  const focusedWindowActions =
    focusedId === null
      ? null
      : {
          close: () => run(() => coordinator.userClose(focusedId)),
          minimize: () => run(() => coordinator.userMinimize(focusedId)),
          zoom:
            presentation === "floating"
              ? () => run(() => store.getState().toggleZoom(focusedId))
              : null,
        };

  return {
    focusedWindowActions,
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
    openSpaces: () => run(() => options.onOpenSpaces?.()),
    openAppearance: () =>
      run(() =>
        coordinator.userOpen({
          app: "settings",
          location: { pathname: "/settings/appearance", search: {} },
        })
      ),
    switchWorkspace: workspaceId => run(() => setActiveWorkspaceId(workspaceId)),
  };
}
