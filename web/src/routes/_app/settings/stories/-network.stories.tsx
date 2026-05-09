import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";

import { storybookMswParameters } from "@/storybook/msw";
import { StorybookFieldDirtySetup } from "@/storybook/settings-state-helpers";
import {
  StorybookRestartBannerSetup,
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
  createRouteStoryMeta,
} from "@/storybook/route-story";

const meta: Meta<typeof StorybookRouteCanvas> = {
  ...createRouteStoryMeta(
    "routes/app/settings/network",
    "Network settings route stories covering the runtime summary, restart banner, and request boundary states."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default network settings page with listener, delivery and channel defaults.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/settings/network"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Dirty shell state -- the listener port has been edited so the save-bar reads
 * Unsaved changes + the Save button enables.
 */
export const Dirty: Story = {
  args: {},
  parameters: appRouteParameters("/settings/network"),
  render: () => (
    <>
      <StorybookWorkspaceSetup />
      <StorybookFieldDirtySetup testId="settings-page-network-port-input" value="4200" />
    </>
  ),
};

/**
 * Restart-required banner after changing network settings that reconfigure the daemon listener.
 */
export const RestartBanner: Story = {
  args: {},
  parameters: appRouteParameters("/settings/network"),
  render: () => (
    <>
      <StorybookWorkspaceSetup />
      <StorybookRestartBannerSetup section="network" />
    </>
  ),
};

/**
 * Loading state while the network section is still resolving.
 */
export const Loading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/network"),
    ...storybookMswParameters({
      settings: [
        http.get("/api/settings/network", async () => {
          await delay("infinite");
          return HttpResponse.json({});
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Error branch shown when the network settings request fails.
 */
export const Error: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/network"),
    ...storybookMswParameters({
      settings: [
        http.get("/api/settings/network", () =>
          HttpResponse.json({ error: "Failed to load network settings" }, { status: 500 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};
