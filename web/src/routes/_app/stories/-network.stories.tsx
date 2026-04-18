import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";
import { expect, userEvent, within } from "storybook/test";

import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
  createRouteStoryMeta,
} from "@/storybook/route-story";
import { networkStatusFixture } from "@/systems/network/mocks";

const meta: Meta<typeof StorybookRouteCanvas> = {
  ...createRouteStoryMeta(
    "routes/app/network",
    "Route-level stories for the network workspace page, including shell layout, tabs, and primary empty or disabled states."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default channels view with metrics, list and detail panel inside the real app shell.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/network"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Peers tab after switching from the default channels view.
 */
export const PeersTab: Story = {
  args: {},
  parameters: appRouteParameters("/network"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("network-tab-peers"));
    await expect(canvas.findByTestId("network-peers-list-panel")).resolves.toBeDefined();
  },
};

/**
 * Disabled runtime branch when the embedded network is turned off in config.
 */
export const Disabled: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/network"),
    ...storybookMswParameters({
      network: [
        http.get("/api/network/status", () =>
          HttpResponse.json({
            network: {
              ...networkStatusFixture,
              enabled: false,
              status: "disabled",
              channels: 0,
              local_peers: 0,
              remote_peers: 0,
              delivery_workers: 0,
              queued_messages: 0,
            },
          })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Empty channels branch when the workspace has network enabled but no channels yet.
 */
export const EmptyChannels: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/network"),
    ...storybookMswParameters({
      network: [http.get("/api/network/channels", () => HttpResponse.json({ channels: [] }))],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Channel creation dialog opened from the primary page CTA.
 */
export const CreateChannel: Story = {
  args: {},
  parameters: appRouteParameters("/network"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("open-network-create-dialog"));
    await expect(canvas.findByTestId("network-create-channel-dialog")).resolves.toBeDefined();
  },
};

/**
 * Loading state for the channels collection while the page shell remains mounted.
 */
export const Loading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/network"),
    ...storybookMswParameters({
      network: [
        http.get("/api/network/channels", async () => {
          await delay("infinite");
          return HttpResponse.json({ channels: [] });
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};
