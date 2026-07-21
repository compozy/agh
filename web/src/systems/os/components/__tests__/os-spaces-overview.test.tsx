// Suite: Spaces overview
// Invariant: the overlay exposes real active-space state and switches only to real workspaces.
// Boundary IN: OsSpacesOverview interaction, labeling, and close behavior.
// Boundary OUT: per-workspace persistence/thumbnails (Task 08) and browser journey wiring.
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { WorkspacePayload } from "@/systems/workspace";

import type { OsWindow } from "../../lib/os-types";
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

function createWindow(id: string, minimized: boolean): OsWindow {
  return {
    id,
    app: id === "app:dashboard" ? "dashboard" : "tasks",
    instanceKey: null,
    location: { pathname: id === "app:dashboard" ? "/" : "/tasks", search: {} },
    rect: { x: 100, y: 60, w: 640, h: 480 },
    prevRect: null,
    z: minimized ? 1 : 2,
    minimized,
    maximized: false,
    snap: null,
  };
}

function renderOverview(onSelectWorkspace = vi.fn(), onOpenChange = vi.fn()) {
  render(
    <OsSpacesOverview
      open
      onOpenChange={onOpenChange}
      workspaces={WORKSPACES}
      activeWorkspaceId="w-agh"
      activeWallpaper="ember"
      activeWindows={[createWindow("app:dashboard", false), createWindow("app:tasks", true)]}
      onSelectWorkspace={onSelectWorkspace}
    />
  );
  return { onSelectWorkspace, onOpenChange };
}

describe("OsSpacesOverview", () => {
  it("Should expose only real active-space state and switch to another workspace", async () => {
    const user = userEvent.setup();
    const { onSelectWorkspace, onOpenChange } = renderOverview();

    expect(screen.getByRole("dialog", { name: "Spaces overview" })).toBeInTheDocument();
    expect(
      screen.getByRole("button", {
        name: "Current workspace agh. 1 visible · 1 minimized · ember wallpaper",
      })
    ).toBeInTheDocument();

    await user.click(
      screen.getByRole("button", {
        name: "Switch to runtime-labs. Select to load this space",
      })
    );

    expect(onSelectWorkspace).toHaveBeenCalledWith("w-labs");
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("Should close without reselecting when the current workspace is chosen", async () => {
    const user = userEvent.setup();
    const { onSelectWorkspace, onOpenChange } = renderOverview();

    await user.click(
      screen.getByRole("button", {
        name: "Current workspace agh. 1 visible · 1 minimized · ember wallpaper",
      })
    );

    expect(onSelectWorkspace).not.toHaveBeenCalled();
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
