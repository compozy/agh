import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";

import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
  createRouteStoryMeta,
} from "@/storybook/route-story";
import { daemonHealthFixture } from "@/systems/daemon/mocks/fixtures";

const meta: Meta<typeof StorybookRouteCanvas> = {
  ...createRouteStoryMeta(
    "routes/app/home",
    "Full app-shell route stories for the home dashboard — daemon status + key metrics — and the onboarding branch."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Healthy daemon — green StatusDot, connected pill, populated metric grid.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Daemon responded but reported a non-healthy status — warning StatusDot, still
 * connected pill, metrics still populated from the partial payload.
 */
export const Degraded: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/"),
    ...storybookMswParameters({
      daemon: [
        http.get("/api/observe/health", () =>
          HttpResponse.json({
            health: { ...daemonHealthFixture, status: "degraded" },
            memory: {
              dream_enabled: false,
              global_files: 0,
              workspace_files: 0,
              last_consolidation: null,
            },
          })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Daemon unreachable — danger StatusDot, disconnected pill, recovery hint card
 * via Empty + ConnectionIndicator composition.
 */
export const Disconnected: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/"),
    ...storybookMswParameters({
      daemon: [
        http.get("/api/observe/health", () =>
          HttpResponse.json({ error: "daemon offline" }, { status: 503 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * First paint with slow network — skeletons in the daemon and metric sections.
 */
export const Loading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/"),
    ...storybookMswParameters({
      daemon: [
        http.get("/api/observe/health", async () => {
          await delay("infinite");
          return HttpResponse.json({});
        }),
      ],
      workspace: [
        http.get("/api/workspaces", async () => {
          await delay("infinite");
          return HttpResponse.json({ workspaces: [] });
        }),
      ],
      agent: [
        http.get("/api/agents", async () => {
          await delay("infinite");
          return HttpResponse.json({ agents: [] });
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Workspaces fetch fails — Empty error card replaces the dashboard body.
 */
export const Error: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/"),
    ...storybookMswParameters({
      workspace: [
        http.get("/api/workspaces", () =>
          HttpResponse.json({ error: "workspaces unavailable" }, { status: 500 })
        ),
      ],
    }),
  },
};

/**
 * Onboarding branch shown when the daemon has no configured workspaces yet.
 * The shell handles this above the route, so the route component never renders.
 */
export const Onboarding: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/"),
    ...storybookMswParameters({
      workspace: [http.get("/api/workspaces", () => HttpResponse.json({ workspaces: [] }))],
    }),
  },
};
