import type { Meta, StoryObj } from "@storybook/react-vite";
import { HttpResponse } from "msw";
import { fn } from "storybook/test";

import type { WorkspacePayload } from "@/systems/workspace";
import { storybookMswParameters } from "@/storybook/msw";
import { aghApiMock } from "@/storybook/openapi-msw";

import { OsShellContext, type OsShellHandle } from "../../contexts/os-shell-context";
import { RoutingCoordinator, type OsRouterPort } from "../../lib/routing-coordinator";
import { encodeDesktopPayload, encodeWindowPayload } from "../../lib/os-state-payloads";
import type { OsWindow } from "../../lib/os-types";
import { createDesktopStore } from "../../stores/desktop-store";
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

function labsWindow(id: string, app: OsWindow["app"], rect: OsWindow["rect"], z: number): OsWindow {
  return {
    id,
    app,
    instanceKey: null,
    location: { pathname: `/${app}`, search: {} },
    rect,
    prevRect: null,
    z,
    minimized: false,
    maximized: false,
    snap: null,
  };
}

const LABS_ENTRIES = [
  {
    key: "win:app:vault",
    value: encodeWindowPayload(
      labsWindow("app:vault", "vault", { x: 90, y: 60, w: 560, h: 430 }, 1)
    ),
    rev: 1,
    seq: 1,
    deleted: false,
    updated_at: "2026-07-20T00:00:00Z",
  },
  {
    key: "win:app:knowledge",
    value: encodeWindowPayload(
      labsWindow("app:knowledge", "knowledge", { x: 540, y: 180, w: 700, h: 500 }, 2)
    ),
    rev: 1,
    seq: 2,
    deleted: false,
    updated_at: "2026-07-20T00:00:00Z",
  },
  {
    key: "desktop",
    value: encodeDesktopPayload({
      focusedId: "app:knowledge",
      railOpen: false,
      wallpaper: "carbon",
    }),
    rev: 1,
    seq: 3,
    deleted: false,
    updated_at: "2026-07-20T00:00:00Z",
  },
];

function createStoryShell(): OsShellHandle {
  const store = createDesktopStore();
  const router: OsRouterPort = { navigate: () => {}, replace: () => {} };
  store.getState().hydrate([]);
  store.getState().clampToViewport({ width: 1440, height: 820 });
  store.getState().openOrFocus({ app: "dashboard", location: { pathname: "/", search: {} } });
  store.getState().commitRect("app:dashboard", { x: 120, y: 72, w: 720, h: 520 });
  store.getState().openOrFocus({ app: "tasks" });
  store.getState().commitRect("app:tasks", { x: 620, y: 160, w: 680, h: 480 });
  return {
    store,
    coordinator: new RoutingCoordinator(store, router),
    flushPersistence: () => {},
  };
}

const STORY_SHELL = createStoryShell();

const meta: Meta<typeof OsSpacesOverview> = {
  title: "systems/os/components/OsSpacesOverview",
  component: OsSpacesOverview,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "The ⇧⌘S Spaces overview: one card per real workspace with mini-window thumbnails — the active space from the live desktop store, other spaces from persisted desktop state. This is the canonical VC-01 implementation surface.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Open over the live desktop shell: the active space thumbnails two arranged
 * windows; runtime-labs shows its persisted carbon-wallpaper arrangement.
 */
export const Open: Story = {
  args: {
    open: true,
    onOpenChange: fn(),
    workspaces: WORKSPACES,
    activeWorkspaceId: "w-agh",
    onSelectWorkspace: fn(),
  },
  parameters: storybookMswParameters({
    workspace: [
      aghApiMock.get("/api/workspaces/{workspace_id}/desktop-state", () =>
        HttpResponse.json({ as_of_seq: 3, entries: LABS_ENTRIES })
      ),
    ],
  }),
  render: args => (
    <OsShellContext.Provider value={STORY_SHELL}>
      <DesktopShell wallpaper="ember" deskHint>
        <OsSpacesOverview {...args} />
      </DesktopShell>
    </OsShellContext.Provider>
  ),
};
