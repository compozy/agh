import type { Meta, StoryObj } from "@storybook/react-vite";
import { HttpResponse } from "msw";

import { aghApiMock } from "@/storybook/openapi-msw";
import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
} from "@/storybook/route-story-meta";
import { marketplaceKindFixture } from "@/systems/marketplace/mocks";

const extensionFeed = {
  ...marketplaceKindFixture("extension"),
  items: marketplaceKindFixture("extension").items.map(item =>
    item.entry_id === "otel-bridge" ? { ...item, update_available: true } : item
  ),
};

const meta: Meta<typeof StorybookRouteCanvas> = {
  title: "systems/extensions/routes/ExtensionDetail",
  component: StorybookRouteCanvas,
  parameters: { layout: "fullscreen" },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Healthy: Story = {
  parameters: {
    ...appRouteParameters("/extensions/otel-bridge"),
    ...storybookMswParameters({
      marketplace: [
        aghApiMock.get("/api/marketplace/{kind}", () => HttpResponse.json(extensionFeed)),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

export const Degraded: Story = {
  parameters: appRouteParameters("/extensions/slack-notify"),
  render: () => <StorybookWorkspaceSetup />,
};
