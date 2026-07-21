import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { OsShellContext, type OsShellHandle } from "../../contexts/os-shell-context";
import { RoutingCoordinator, type OsRouterPort } from "../../lib/routing-coordinator";
import { createDesktopStore } from "../../stores/desktop-store";
import { OsCommandPalette } from "../os-command-palette";

const openForAgent = vi.fn();
const setActiveWorkspaceId = vi.fn();

vi.mock("@/systems/session", async importOriginal => ({
  ...(await importOriginal<Record<string, unknown>>()),
  useSessionCreate: () => ({
    openForAgent,
    isCreating: false,
    pendingAgentName: null,
    hasActiveWorkspace: true,
  }),
  useSessions: () => ({
    data: [
      { id: "s1", name: "Checkout flow polish", agent_name: "webgen", workspace_id: "w1" },
      { id: "s2", name: "Release dry-run", agent_name: "infra", workspace_id: "w1" },
    ],
    total: 2,
  }),
}));

vi.mock("@/systems/workspace", async importOriginal => ({
  ...(await importOriginal<Record<string, unknown>>()),
  useActiveWorkspace: () => ({
    workspaces: [
      { id: "w1", name: "agh" },
      { id: "w2", name: "labs" },
    ],
    hasWorkspaces: true,
    activeWorkspace: { id: "w1", name: "agh" },
    activeWorkspaceId: "w1",
    setActiveWorkspaceId,
    isLoading: false,
    isError: false,
  }),
}));

function createHarness() {
  const store = createDesktopStore();
  const pushes: string[] = [];
  const port: OsRouterPort = {
    navigate: location => pushes.push(location.pathname),
    replace: location => pushes.push(location.pathname),
  };
  const coordinator = new RoutingCoordinator(store, port);
  store.getState().hydrate([]);
  coordinator.completeHydration();
  const shell: OsShellHandle = { store, coordinator, flushPersistence: () => {} };
  return { store, shell, pushes };
}

describe("OsCommandPalette", () => {
  beforeAll(() => {
    // cmdk scrolls the highlighted row into view; jsdom has no layout engine.
    Element.prototype.scrollIntoView = () => {};
  });

  it("Should list apps, sessions with agent meta, and actions; filter on typing; open on Enter (UT-059)", async () => {
    const user = userEvent.setup();
    const { store, shell, pushes } = createHarness();

    render(
      <OsShellContext.Provider value={shell}>
        <OsCommandPalette open onOpenChange={() => {}} />
      </OsShellContext.Provider>
    );

    // Sources: apps, sessions (label + agent meta), actions.
    expect(await screen.findByTestId("os-palette-app-dashboard")).toBeInTheDocument();
    expect(screen.getByTestId("os-palette-session-s1")).toHaveTextContent("Checkout flow polish");
    expect(screen.getByTestId("os-palette-session-s1")).toHaveTextContent("webgen");
    expect(screen.getByTestId("os-palette-new-session")).toBeInTheDocument();
    expect(screen.getByTestId("os-palette-workspace-w2")).toBeInTheDocument();

    // Typing filters the list live.
    await user.type(screen.getByPlaceholderText("Search apps, sessions, actions…"), "checkout");
    await waitFor(() => {
      expect(screen.queryByTestId("os-palette-app-dashboard")).toBeNull();
    });
    expect(screen.getByTestId("os-palette-session-s1")).toBeInTheDocument();

    // Enter runs the highlighted selection: the session window opens focused.
    await user.keyboard("{Enter}");
    expect(store.getState().windows["session:s1"]).toBeDefined();
    expect(store.getState().windows["session:s1"].location.pathname).toBe(
      "/agents/webgen/sessions/s1"
    );
    expect(pushes).toEqual(["/agents/webgen/sessions/s1"]);
  });

  it("Should open an app window from the Apps group", async () => {
    const user = userEvent.setup();
    const { store, shell, pushes } = createHarness();

    render(
      <OsShellContext.Provider value={shell}>
        <OsCommandPalette open onOpenChange={() => {}} />
      </OsShellContext.Provider>
    );

    await user.type(screen.getByPlaceholderText("Search apps, sessions, actions…"), "open tasks");
    await user.keyboard("{Enter}");

    expect(store.getState().windows["app:tasks"]).toBeDefined();
    expect(pushes).toEqual(["/tasks"]);
  });

  it("Should route the sessions action through the persisted rail toggle (UT-084)", async () => {
    const user = userEvent.setup();
    const { store, shell } = createHarness();

    render(
      <OsShellContext.Provider value={shell}>
        <OsCommandPalette open onOpenChange={() => {}} />
      </OsShellContext.Provider>
    );

    await user.type(
      screen.getByPlaceholderText("Search apps, sessions, actions…"),
      "toggle sessions"
    );
    await user.keyboard("{Enter}");
    expect(store.getState().railOpen).toBe(true);
  });

  it("Should list snap commands for the focused window and dispatch snapWindow (UT-101)", async () => {
    const user = userEvent.setup();
    const { store, shell } = createHarness();
    store.getState().openOrFocus({ app: "tasks" });
    store.getState().clampToViewport({ width: 1440, height: 900 });

    render(
      <OsShellContext.Provider value={shell}>
        <OsCommandPalette open onOpenChange={() => {}} />
      </OsShellContext.Provider>
    );

    // Every zone is listed; restore is absent while the window floats.
    expect(await screen.findByTestId("os-palette-snap-left")).toBeInTheDocument();
    expect(screen.getByTestId("os-palette-snap-top-right")).toBeInTheDocument();
    expect(screen.queryByTestId("os-palette-snap-restore")).toBeNull();

    await user.type(screen.getByPlaceholderText("Search apps, sessions, actions…"), "snap left");
    await user.keyboard("{Enter}");
    expect(store.getState().windows["app:tasks"].snap).toEqual({ fx: 0, fy: 0, fw: 0.5, fh: 1 });
  });

  it("Should offer restore for a snapped window and hide snap commands in compact (UT-101)", async () => {
    const user = userEvent.setup();
    const { store, shell } = createHarness();
    store.getState().openOrFocus({ app: "tasks" });
    store.getState().clampToViewport({ width: 1440, height: 900 });
    store.getState().commitRect("app:tasks", { x: 60, y: 50, w: 520, h: 400 });
    store.getState().snapWindow("app:tasks", { fx: 0, fy: 0, fw: 0.5, fh: 1 });

    const { rerender } = render(
      <OsShellContext.Provider value={shell}>
        <OsCommandPalette open onOpenChange={() => {}} />
      </OsShellContext.Provider>
    );

    await user.type(screen.getByPlaceholderText("Search apps, sessions, actions…"), "restore");
    await user.keyboard("{Enter}");
    expect(store.getState().windows["app:tasks"].snap).toBeNull();
    expect(store.getState().windows["app:tasks"].rect).toEqual({ x: 60, y: 50, w: 520, h: 400 });

    // Compact presentation: the commands are absent entirely (UT-061 gating).
    store.getState().setPresentation("compact");
    rerender(
      <OsShellContext.Provider value={shell}>
        <OsCommandPalette open onOpenChange={() => {}} />
      </OsShellContext.Provider>
    );
    await waitFor(() => {
      expect(screen.queryByTestId("os-palette-snap-left")).toBeNull();
    });
  });

  it("Should keep close/minimize palette-reachable everywhere and drop zoom in compact (US-003.AC-4, US-019.EC-3)", async () => {
    const user = userEvent.setup();
    const { store, shell } = createHarness();
    store.getState().openOrFocus({ app: "tasks" });

    const { rerender } = render(
      <OsShellContext.Provider value={shell}>
        <OsCommandPalette open onOpenChange={() => {}} />
      </OsShellContext.Provider>
    );

    // Floating: the full lifecycle set is listed for the focused window.
    expect(await screen.findByTestId("os-palette-close-window")).toBeInTheDocument();
    expect(screen.getByTestId("os-palette-minimize-window")).toBeInTheDocument();
    expect(screen.getByTestId("os-palette-zoom-window")).toBeInTheDocument();
    expect(screen.getByTestId("os-palette-spaces-overview")).toBeInTheDocument();
    expect(screen.getByTestId("os-palette-appearance")).toBeInTheDocument();

    // Compact: zoom disappears (meaningless in a stack); close/minimize stay.
    store.getState().setPresentation("compact");
    rerender(
      <OsShellContext.Provider value={shell}>
        <OsCommandPalette open onOpenChange={() => {}} />
      </OsShellContext.Provider>
    );
    await waitFor(() => {
      expect(screen.queryByTestId("os-palette-zoom-window")).toBeNull();
    });
    expect(screen.getByTestId("os-palette-close-window")).toBeInTheDocument();

    await user.type(
      screen.getByPlaceholderText("Search apps, sessions, actions…"),
      "minimize window"
    );
    await user.keyboard("{Enter}");
    expect(store.getState().windows["app:tasks"].minimized).toBe(true);
  });
});
