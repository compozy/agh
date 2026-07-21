// Suite: Spaces overview
// Invariant: every workspace card shows real rollups — agent/session counts from the workspace
// detail endpoint, window counts from real arrangements — and switching targets real workspaces.
// Boundary IN: OsSpacesOverview header, cards, labeling, and close behavior.
// Boundary OUT: HTTP transport (seeded query cache) and browser journey wiring (E2E-009).
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { workspaceDetailOptions, type WorkspacePayload } from "@/systems/workspace";
import { workspaceDetailFixture } from "@/systems/workspace/mocks";

import { OsShellContext, type OsShellHandle } from "../../contexts/os-shell-context";
import { RoutingCoordinator, type OsRouterPort } from "../../lib/routing-coordinator";
import { encodeDesktopPayload, encodeWindowPayload } from "../../lib/os-state-payloads";
import type { OsStateEntry, OsWindow } from "../../lib/os-types";
import { createDesktopStore } from "../../stores/desktop-store";
import { OsSpacesOverview } from "../os-spaces-overview";

const WORKSPACES: WorkspacePayload[] = [
  {
    id: "w-agh",
    name: "agh",
    root_dir: "/work/agh",
    add_dirs: [],
    created_at: "2026-07-20T00:00:00Z",
    updated_at: "2026-07-20T00:00:00Z",
  },
  {
    id: "w-labs",
    name: "runtime-labs",
    root_dir: "/work/runtime-labs",
    add_dirs: [],
    created_at: "2026-07-20T00:00:00Z",
    updated_at: "2026-07-20T00:00:00Z",
  },
];

function makeWindow(id: string, overrides: Partial<OsWindow> = {}): OsWindow {
  return {
    id,
    app: "tasks",
    instanceKey: null,
    location: { pathname: "/tasks", search: {} },
    rect: { x: 120, y: 80, w: 640, h: 480 },
    prevRect: null,
    z: 1,
    minimized: false,
    maximized: false,
    snap: null,
    ...overrides,
  };
}

function entryFor(win: OsWindow, seq: number): OsStateEntry {
  return {
    key: `win:${win.id}`,
    value: encodeWindowPayload(win),
    rev: 1,
    seq,
    deleted: false,
    updated_at: "2026-07-20T00:00:00Z",
  };
}

function requireItem<T>(value: T | undefined, label: string): T {
  if (value === undefined) throw new Error(`workspaceDetailFixture must carry ${label}`);
  return value;
}

const agentItemFixture = requireItem(workspaceDetailFixture.agents?.[0], "an agent");
const sessionItemFixture = requireItem(workspaceDetailFixture.sessions?.[0], "a session");

function seedWorkspaceDetail(
  queryClient: QueryClient,
  workspace: WorkspacePayload,
  agents: string[],
  sessionCount: number
) {
  queryClient.setQueryData(workspaceDetailOptions(workspace.id).queryKey, {
    ...workspaceDetailFixture,
    ...workspace,
    agents: agents.map(name => ({ ...agentItemFixture, name })),
    sessions: Array.from({ length: sessionCount }, (_, index) => ({
      ...sessionItemFixture,
      session_id: `sess-${workspace.id}-${index}`,
    })),
  });
}

function renderOverview() {
  const store = createDesktopStore();
  const port: OsRouterPort = { navigate: () => {}, replace: () => {} };
  const shell: OsShellHandle = {
    store,
    coordinator: new RoutingCoordinator(store, port),
    flushPersistence: () => {},
  };
  store.getState().hydrate([]);
  store.getState().clampToViewport({ width: 1440, height: 900 });
  store.getState().openOrFocus({ app: "dashboard", location: { pathname: "/", search: {} } });
  store.getState().openOrFocus({ app: "tasks" });
  store.getState().minimizeWindow("app:tasks");

  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  queryClient.setQueryData(["os", "desktop-state", "w-labs"], [
    entryFor(
      makeWindow("app:vault", { app: "vault", location: { pathname: "/vault", search: {} } }),
      1
    ),
    entryFor(
      makeWindow("app:knowledge", {
        app: "knowledge",
        z: 2,
        location: { pathname: "/knowledge", search: {} },
      }),
      2
    ),
    {
      key: "desktop",
      value: encodeDesktopPayload({
        focusedId: "app:vault",
        railOpen: false,
        wallpaper: "carbon",
      }),
      rev: 1,
      seq: 3,
      deleted: false,
      updated_at: "2026-07-20T00:00:00Z",
    },
  ] satisfies OsStateEntry[]);
  seedWorkspaceDetail(queryClient, WORKSPACES[0]!, ["planner", "reviewer"], 1);
  seedWorkspaceDetail(queryClient, WORKSPACES[1]!, ["builder"], 2);

  const onSelectWorkspace = vi.fn();
  const onOpenChange = vi.fn();
  const onNewSpace = vi.fn();
  render(
    <QueryClientProvider client={queryClient}>
      <OsShellContext.Provider value={shell}>
        <OsSpacesOverview
          open
          onOpenChange={onOpenChange}
          workspaces={WORKSPACES}
          activeWorkspaceId="w-agh"
          onSelectWorkspace={onSelectWorkspace}
          onNewSpace={onNewSpace}
        />
      </OsShellContext.Provider>
    </QueryClientProvider>
  );
  return { onSelectWorkspace, onOpenChange, onNewSpace };
}

describe("OsSpacesOverview", () => {
  it("Should render real agent/session/window rollups per card and the header subtitle", async () => {
    renderOverview();

    expect(await screen.findByTestId("os-space-meta-w-agh")).toHaveTextContent(
      "2 agents · 1 session · 1 window"
    );
    expect(await screen.findByTestId("os-space-meta-w-labs")).toHaveTextContent(
      "1 agent · 2 sessions · 2 windows"
    );
    expect(screen.getByTestId("os-spaces-subtitle")).toHaveTextContent(
      "2 spaces · 3 agents distributed"
    );
    expect(screen.getByLabelText("Agent planner")).toBeInTheDocument();
    expect(screen.getByLabelText("Agent builder")).toBeInTheDocument();
  });

  it("Should switch to another workspace and close, or only close on the current one", async () => {
    const user = userEvent.setup();
    const { onSelectWorkspace, onOpenChange } = renderOverview();

    await user.click(
      await screen.findByRole("button", {
        name: "Switch to runtime-labs. 1 agent · 2 sessions · 2 windows",
      })
    );
    expect(onSelectWorkspace).toHaveBeenCalledWith("w-labs");
    expect(onOpenChange).toHaveBeenCalledWith(false);

    onSelectWorkspace.mockClear();
    await user.click(
      screen.getByRole("button", {
        name: "Current workspace agh. 2 agents · 1 session · 1 window",
      })
    );
    expect(onSelectWorkspace).not.toHaveBeenCalled();
  });

  it("Should close the overlay and open workspace setup from New space", async () => {
    const user = userEvent.setup();
    const { onOpenChange, onNewSpace } = renderOverview();

    await user.click(screen.getByTestId("os-spaces-new-space"));
    expect(onOpenChange).toHaveBeenCalledWith(false);
    expect(onNewSpace).toHaveBeenCalledTimes(1);
  });
});
