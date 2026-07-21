// Suite: Spaces overview
// Invariant: every workspace card shows its real arrangement — the active space from the live
// store, other spaces from persisted desktop state — and switching targets real workspaces only.
// Boundary IN: OsSpacesOverview cards, thumbnails, labeling, and close behavior.
// Boundary OUT: HTTP transport (seeded query cache) and browser journey wiring (E2E-009).
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { WorkspacePayload } from "@/systems/workspace";

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

  const onSelectWorkspace = vi.fn();
  const onOpenChange = vi.fn();
  render(
    <QueryClientProvider client={queryClient}>
      <OsShellContext.Provider value={shell}>
        <OsSpacesOverview
          open
          onOpenChange={onOpenChange}
          workspaces={WORKSPACES}
          activeWorkspaceId="w-agh"
          onSelectWorkspace={onSelectWorkspace}
        />
      </OsShellContext.Provider>
    </QueryClientProvider>
  );
  return { onSelectWorkspace, onOpenChange };
}

describe("OsSpacesOverview", () => {
  it("Should render live thumbnails for the active space and persisted ones for the rest", async () => {
    renderOverview();

    const activeCard = await screen.findByRole("button", {
      name: "Current workspace agh. 1 window",
    });
    // The active space mirrors the live store: minimized windows never thumbnail.
    expect(activeCard.querySelectorAll('[data-slot="os-space-mini-win"]')).toHaveLength(1);

    const labsCard = await screen.findByRole("button", {
      name: "Switch to runtime-labs. 2 windows",
    });
    expect(labsCard.querySelectorAll('[data-slot="os-space-mini-win"]')).toHaveLength(2);
    expect(labsCard.querySelector('[data-slot="os-space-thumb"]')).toHaveAttribute(
      "data-wallpaper",
      "carbon"
    );
  });

  it("Should switch to another workspace and close, or only close on the current one", async () => {
    const user = userEvent.setup();
    const { onSelectWorkspace, onOpenChange } = renderOverview();

    await user.click(
      await screen.findByRole("button", { name: "Switch to runtime-labs. 2 windows" })
    );
    expect(onSelectWorkspace).toHaveBeenCalledWith("w-labs");
    expect(onOpenChange).toHaveBeenCalledWith(false);

    onSelectWorkspace.mockClear();
    await user.click(screen.getByRole("button", { name: "Current workspace agh. 1 window" }));
    expect(onSelectWorkspace).not.toHaveBeenCalled();
  });
});
