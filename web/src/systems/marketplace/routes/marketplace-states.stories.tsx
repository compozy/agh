import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, HttpResponse } from "msw";

import { aghApiMock } from "@/storybook/openapi-msw";
import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
} from "@/storybook/route-story-meta";
import { marketplaceKindFixture } from "@/systems/marketplace/mocks";

const meta: Meta<typeof StorybookRouteCanvas> = {
  title: "systems/marketplace/routes/MarketplaceStates",
  component: StorybookRouteCanvas,
  parameters: { layout: "fullscreen" },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const LandingAllZero: Story = {
  parameters: appRouteParameters("/marketplace?q=no-such-capability"),
  render: () => <StorybookWorkspaceSetup />,
};

export const LandingError: Story = {
  parameters: {
    ...appRouteParameters("/marketplace"),
    ...storybookMswParameters({
      marketplace: [
        aghApiMock.get("/api/marketplace/search", () =>
          HttpResponse.json({ error: "Marketplace catalog unavailable" }, { status: 503 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

export const KindLoading: Story = {
  parameters: {
    ...appRouteParameters("/marketplace?kind=skills"),
    ...storybookMswParameters({
      marketplace: [
        aghApiMock.get("/api/marketplace/{kind}", async () => {
          await delay("infinite");
          return HttpResponse.json(marketplaceKindFixture("skill"));
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

export const KindError: Story = {
  parameters: {
    ...appRouteParameters("/marketplace?kind=skills"),
    ...storybookMswParameters({
      marketplace: [
        aghApiMock.get("/api/marketplace/{kind}", () =>
          HttpResponse.json({ error: "Skill catalog unavailable" }, { status: 503 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

export const DetailNotFound: Story = {
  parameters: appRouteParameters("/marketplace/skill/removed-entry"),
  render: () => <StorybookWorkspaceSetup />,
};

export const DetailPolicyBlocked: Story = {
  parameters: appRouteParameters("/marketplace/extension/policy-blocked"),
  render: () => <StorybookWorkspaceSetup />,
};

export const DetailUnverifiedExtension: Story = {
  parameters: appRouteParameters("/marketplace/extension/slack-notify"),
  render: () => <StorybookWorkspaceSetup />,
};

export const DetailBundle: Story = {
  parameters: appRouteParameters("/marketplace/bundle/dep-kit"),
  render: () => <StorybookWorkspaceSetup />,
};

export const DetailRemoteMCP: Story = {
  parameters: appRouteParameters("/marketplace/mcp/linear"),
  render: () => <StorybookWorkspaceSetup />,
};

export const DetailLoading: Story = {
  parameters: {
    ...appRouteParameters("/marketplace/skill/git-flow"),
    ...storybookMswParameters({
      marketplace: [
        aghApiMock.get("/api/marketplace/{kind}/{entry_id}", async () => {
          await delay("infinite");
          return HttpResponse.json({ error: "unreachable" }, { status: 503 });
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};
