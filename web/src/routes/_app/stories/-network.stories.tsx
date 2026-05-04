import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";
import { expect, userEvent, within } from "storybook/test";

import { storyHeroNetworkChannel, storyPeerIds } from "@/storybook/fintech-scenario";
import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
  createRouteStoryMeta,
} from "@/storybook/route-story";
import { networkStatusFixture } from "@/systems/network/mocks";

const storybookNetworkStatus = networkStatusFixture;

const meta: Meta<typeof StorybookRouteCanvas> = {
  ...createRouteStoryMeta(
    "routes/app/network",
    "Real-shell stories for the network workspace route, covering channels, direct rooms, the create-channel dialog, and MSW-backed empty/loading branches."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default network route with the first channel selected from the MSW fixtures.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/network"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Direct-message room loaded through the real route search parameters.
 */
export const DirectRoom: Story = {
  args: {},
  parameters: appRouteParameters(`/network?peer=${storyPeerIds.remote}`),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Create-channel dialog opened from the network sidebar action.
 */
export const CreateDialog: Story = {
  args: {},
  parameters: appRouteParameters("/network"),
  render: () => <StorybookWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("network-open-create-dialog"));
    await expect(
      within(canvasElement.ownerDocument.body).findByTestId("network-create-channel-dialog")
    ).resolves.toBeDefined();
  },
};

/**
 * Empty network branch when no channels or peers have materialized yet.
 */
export const EmptyChannels: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/network"),
    ...storybookMswParameters({
      network: [
        http.get("/api/network/channels", () => HttpResponse.json({ channels: [] })),
        http.get("/api/network/peers", () => HttpResponse.json({ peers: [] })),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Disabled network branch shown when the daemon reports network support off.
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
              ...storybookNetworkStatus,
              channels: 0,
              delivery_workers: 0,
              configured_default_channel: storyHeroNetworkChannel,
              enabled: false,
              effective_default_channel: storyHeroNetworkChannel,
              local_peers: 0,
              queued_messages: 0,
              remote_peers: 0,
              status: "stopped",
            },
          })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * First paint while the network status request is still pending.
 */
export const Loading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/network"),
    ...storybookMswParameters({
      network: [
        http.get("/api/network/status", async () => {
          await delay("infinite");
          return HttpResponse.json({ network: storybookNetworkStatus });
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};
