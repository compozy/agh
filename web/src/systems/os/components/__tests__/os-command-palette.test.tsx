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
});
