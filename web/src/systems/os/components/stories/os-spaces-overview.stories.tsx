import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import type { WorkspacePayload } from "@/systems/workspace";

import type { OsWindow } from "../../lib/os-types";
import { OsSpacesOverview } from "../os-spaces-overview";
import { DesktopShell } from "./_desktop";

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
    id: "w-runtime",
    name: "runtime-labs",
    root_dir: "/work/runtime-labs",
    add_dirs: [],
    created_at: "2026-07-20T00:00:00Z",
    updated_at: "2026-07-20T00:00:00Z",
  },
];

const ACTIVE_WINDOWS: OsWindow[] = [
  {
    id: "app:dashboard",
    app: "dashboard",
    instanceKey: null,
    location: { pathname: "/", search: {} },
    rect: { x: 120, y: 72, w: 720, h: 520 },
    prevRect: null,
    z: 2,
    minimized: false,
    maximized: false,
  },
  {
    id: "app:tasks",
    app: "tasks",
    instanceKey: null,
    location: { pathname: "/tasks", search: {} },
    rect: { x: 240, y: 112, w: 680, h: 480 },
    prevRect: null,
    z: 1,
    minimized: true,
    maximized: false,
  },
];

const meta: Meta<typeof OsSpacesOverview> = {
  title: "systems/os/components/OsSpacesOverview",
  component: OsSpacesOverview,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "A truthful workspace switcher: the active space reflects the desktop state loaded in memory, while inactive workspaces remain explicit load targets until persisted arrangements ship.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Open over the live desktop shell, with one visible window and one minimized
 * window in the current space.
 */
export const Open: Story = {
  args: {
    open: true,
    onOpenChange: fn(),
    workspaces: WORKSPACES,
    activeWorkspaceId: "w-agh",
    activeWallpaper: "ember",
    activeWindows: ACTIVE_WINDOWS,
    onSelectWorkspace: fn(),
  },
  render: args => (
    <DesktopShell wallpaper="ember" deskHint>
      <OsSpacesOverview {...args} />
    </DesktopShell>
  ),
};
