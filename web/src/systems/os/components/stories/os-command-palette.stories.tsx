import type { Meta, StoryObj } from "@storybook/react-vite";
import { HttpResponse } from "msw";
import { fn } from "storybook/test";

import { SessionCreateProvider } from "@/systems/session";
import { sessionFixtures } from "@/systems/session/mocks";
import { useActiveWorkspaceStore } from "@/systems/workspace";
import { workspaceFixtures } from "@/systems/workspace/mocks";
import { storyDefaultWorkspaceId } from "@/storybook/fintech-scenario";
import { storybookMswParameters } from "@/storybook/msw";
import { aghApiMock } from "@/storybook/openapi-msw";

import { OsShellContext, type OsShellHandle } from "../../contexts/os-shell-context";
import { RoutingCoordinator, type OsRouterPort } from "../../lib/routing-coordinator";
import { createDesktopStore } from "../../stores/desktop-store";
import { OsCommandPalette } from "../os-command-palette";
import { DesktopShell } from "./_desktop";

const PALETTE_SESSIONS = sessionFixtures.filter(
  session => session.workspace_id === storyDefaultWorkspaceId
);

function createStoryShell(): OsShellHandle {
  const store = createDesktopStore();
  const router: OsRouterPort = { navigate: () => {}, replace: () => {} };
  store.getState().hydrate([]);
  return {
    store,
    coordinator: new RoutingCoordinator(store, router),
    flushPersistence: () => {},
  };
}

const STORY_SHELL = createStoryShell();

const meta: Meta<typeof OsCommandPalette> = {
  title: "systems/os/components/OsCommandPalette",
  component: OsCommandPalette,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "The real desktop command palette rendered over the OS shell with deterministic workspace and session responses. This is the canonical VC-03 implementation surface.",
      },
    },
  },
  loaders: [
    async () => {
      await useActiveWorkspaceStore.persist.rehydrate();
      useActiveWorkspaceStore.getState().setSelectedWorkspaceId(storyDefaultWorkspaceId);
      return {};
    },
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Open — production palette, real app registry, live query hooks, and a
 * deterministic active workspace/session catalog over the resting desktop.
 */
export const Open: Story = {
  args: {
    open: true,
    onOpenChange: fn(),
  },
  parameters: storybookMswParameters({
    workspace: [
      aghApiMock.get("/api/workspaces", () => HttpResponse.json({ workspaces: workspaceFixtures })),
    ],
    session: [
      aghApiMock.get("/api/sessions", () =>
        HttpResponse.json({
          sessions: PALETTE_SESSIONS,
          page: { has_more: false, limit: 50, total: PALETTE_SESSIONS.length },
        })
      ),
    ],
  }),
  render: args => (
    <OsShellContext.Provider value={STORY_SHELL}>
      <SessionCreateProvider
        value={{
          openForAgent: fn(),
          isCreating: false,
          pendingAgentName: null,
          hasActiveWorkspace: true,
        }}
      >
        <DesktopShell wallpaper="ember" deskHint>
          <OsCommandPalette {...args} />
        </DesktopShell>
      </SessionCreateProvider>
    </OsShellContext.Provider>
  ),
};
