import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";

import { storyHeroNetworkChannel } from "@/storybook/fintech-scenario";
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
    "Real-shell stories for the channel-pivot network route, covering the threads tab, the directs tab, the activity tab, and MSW-backed empty / loading / disabled branches."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default network route on the threads tab for the hero channel.
 */
export const ThreadsTab: Story = {
  args: {},
  parameters: appRouteParameters(`/network/${storyHeroNetworkChannel}/threads`),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Direct rooms tab for the hero channel.
 */
export const DirectsTab: Story = {
  args: {},
  parameters: appRouteParameters(`/network/${storyHeroNetworkChannel}/directs`),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Activity tab for the hero channel , read-only cross-surface feed.
 */
export const ActivityTab: Story = {
  args: {},
  parameters: appRouteParameters(`/network/${storyHeroNetworkChannel}/activity`),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Empty network branch when no channels have materialized yet.
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
