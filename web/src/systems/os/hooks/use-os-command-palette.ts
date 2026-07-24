import { useSessionCreate, useSessions } from "@/systems/session";
import { useActiveWorkspace, type WorkspacePayload } from "@/systems/workspace";

import {
  WINDOW_ARRANGE_COMMANDS,
  WINDOW_PLACEMENT_COMMANDS,
  type WindowArrangeCommand,
  type WindowManagerActionId,
  type WindowPlacementCommand,
} from "../lib/window-manager-command-registry";
import { dispatchWindowPlacement } from "../lib/window-manager-action-dispatch";
import { resolveWindowManagerActions } from "../lib/window-manager-shortcuts";
import { windowManagerCommandsAvailable } from "../lib/window-manager-command-availability";
import type { OsAppId } from "../lib/os-types";
import { useDesktop } from "./use-desktop";
import { useOsShell } from "./use-os-shell";

type PaletteSession = NonNullable<ReturnType<typeof useSessions>["data"]>[number];

export interface OsCommandPaletteModel {
  paletteSessions: PaletteSession[];
  workspaces: WorkspacePayload[];
  activeWorkspaceId: string | null;
  placementCommands: readonly WindowPlacementCommand[];
  /** Arrange presets; empty without a second visible window (truthful UI). */
  arrangeCommands: readonly WindowArrangeCommand[];
  shortcutLabels: Readonly<Partial<Record<WindowManagerActionId, string>>>;
  /**
   * Lifecycle actions for the focused window — the guaranteed keyboard path
   * where browsers reserve ⌘W/⌘M (US-003.AC-4/EC-3). `zoom` is null in
   * compact presentation (the control has no meaning in a stack).
   */
  focusedWindowActions: {
    close(): void;
    minimize(): void;
    zoom: (() => void) | null;
    makeFloating: (() => void) | null;
  } | null;
  openApp(app: OsAppId): void;
  jumpToSession(sessionId: string, agentName: string): void;
  dispatchPlacement(command: WindowPlacementCommand): void;
  dispatchArrange(command: WindowArrangeCommand): void;
  toggleSessions(): void;
  newSession(): void;
  openDesktops(): void;
  openAppearance(): void;
  switchWorkspace(workspaceId: string): void;
  commandsAvailable: boolean;
}

/**
 * Palette view-model: sources (apps come from the registry constant), the
 * focused-window snap surface (ADR-009), and the run-and-close dispatch
 * pattern. Keeps `OsCommandPalette` purely presentational.
 */
export function useOsCommandPalette(
  open: boolean,
  onOpenChange: (open: boolean) => void,
  options: { onOpenDesktops?: () => void; onToggleSessions?: () => void } = {}
): OsCommandPaletteModel {
  const { coordinator, manager, store } = useOsShell();
  const sessionCreate = useSessionCreate();
  const { workspaces, activeWorkspaceId, setActiveWorkspaceId } = useActiveWorkspace();
  const sessions = useSessions(activeWorkspaceId, {
    enabled: open && activeWorkspaceId !== null,
  });
  const focusedWindow = useDesktop(state =>
    state.presentation === "floating" && state.focusedId !== null
      ? state.windows[state.focusedId]
      : undefined
  );
  const focusedId = useDesktop(state => state.focusedId);
  const presentation = useDesktop(state => state.presentation);
  const windowManagerConfig = useDesktop(state => state.windowManagerConfig);
  const commandsAvailable = useDesktop(windowManagerCommandsAvailable);
  const hasArrangePeer = useDesktop(state => {
    if (state.presentation !== "floating" || state.focusedId === null) return false;
    for (const win of Object.values(state.windows)) {
      if (win.id !== state.focusedId && win.desktopId === state.activeDesktopId && !win.minimized) {
        return true;
      }
    }
    return false;
  });

  const run = (action: () => void) => {
    onOpenChange(false);
    action();
  };
  const shortcutLabels = Object.fromEntries(
    resolveWindowManagerActions(windowManagerConfig?.shortcuts ?? {}).flatMap(action =>
      action.shortcutLabel ? [[action.id, action.shortcutLabel] as const] : []
    )
  ) as Partial<Record<WindowManagerActionId, string>>;

  const focusedWindowActions =
    !commandsAvailable || focusedId === null
      ? null
      : {
          close: () => run(() => void coordinator.userClose(focusedId)),
          minimize: () => run(() => void coordinator.userMinimize(focusedId)),
          zoom:
            presentation === "floating"
              ? () => run(() => manager.getState().zoomWindow(focusedId))
              : null,
          makeFloating:
            focusedWindow && focusedWindow.placement !== "floating"
              ? () => run(() => manager.getState().toggleFloating(focusedId))
              : null,
        };

  return {
    focusedWindowActions,
    commandsAvailable,
    paletteSessions: (sessions.data ?? []).filter(
      session => typeof session.agent_name === "string" && session.agent_name.length > 0
    ),
    workspaces,
    activeWorkspaceId,
    shortcutLabels,
    placementCommands:
      commandsAvailable && focusedWindow && windowManagerConfig ? WINDOW_PLACEMENT_COMMANDS : [],
    arrangeCommands: commandsAvailable && hasArrangePeer ? WINDOW_ARRANGE_COMMANDS : [],
    openApp: app =>
      run(() => {
        if (commandsAvailable) void coordinator.userOpen({ app });
      }),
    jumpToSession: (sessionId, agentName) =>
      run(() => {
        if (!commandsAvailable) return;
        void coordinator.userOpen({
          app: "session",
          instanceKey: sessionId,
          route: {
            pathname: `/agents/${encodeURIComponent(agentName)}/sessions/${encodeURIComponent(sessionId)}`,
            search: {},
          },
        });
      }),
    dispatchPlacement: command =>
      run(() => {
        const state = store.getState();
        if (
          !windowManagerCommandsAvailable(state) ||
          state.focusedId === null ||
          state.windowManagerConfig === null
        )
          return;
        dispatchWindowPlacement(manager, state.focusedId, command);
      }),
    dispatchArrange: command =>
      run(() => {
        const state = store.getState();
        if (!windowManagerCommandsAvailable(state) || state.focusedId === null) return;
        state.arrangeLayout(state.focusedId, command.preset);
      }),
    toggleSessions: () => run(() => options.onToggleSessions?.()),
    newSession: () => run(() => sessionCreate.openForAgent("")),
    openDesktops: () => run(() => options.onOpenDesktops?.()),
    openAppearance: () =>
      run(() => {
        if (!commandsAvailable) return;
        void coordinator.userOpen({
          app: "settings",
          route: { pathname: "/settings/appearance", search: {} },
        });
      }),
    switchWorkspace: workspaceId => run(() => setActiveWorkspaceId(workspaceId)),
  };
}
