import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, HttpResponse } from "msw";

import { aghApiMock } from "@/storybook/openapi-msw";
import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
} from "@/storybook/route-story-meta";
import { marketplaceSearchFixture } from "@/systems/marketplace/mocks";

const meta: Meta<typeof StorybookRouteCanvas> = {
  title: "systems/marketplace/routes/Marketplace",
  component: StorybookRouteCanvas,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Route-connected Marketplace stories for grouped browse, kind navigation, partial failure, loading, and detail states.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const LandingPopulated: Story = {
  parameters: appRouteParameters("/marketplace"),
  render: () => <StorybookWorkspaceSetup />,
};

export const LandingLoading: Story = {
  parameters: {
    ...appRouteParameters("/marketplace"),
    ...storybookMswParameters({
      marketplace: [
        aghApiMock.get("/api/marketplace/search", async () => {
          await delay("infinite");
          return HttpResponse.json(marketplaceSearchFixture);
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

export const LandingPartialFailure: Story = {
  parameters: {
    ...appRouteParameters("/marketplace?q=run"),
    ...storybookMswParameters({
      marketplace: [
        aghApiMock.get("/api/marketplace/search", () =>
          HttpResponse.json({
            ...marketplaceSearchFixture,
            query: "run",
            kinds: marketplaceSearchFixture.kinds.map(result =>
              result.kind === "extension"
                ? { ...result, error: "extension source unavailable", items: [], total: null }
                : result
            ),
          })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

export const KindSkill: Story = {
  parameters: appRouteParameters("/marketplace?kind=skills"),
  render: () => <StorybookWorkspaceSetup />,
};

export const DetailSkill: Story = {
  parameters: appRouteParameters("/marketplace/skill/git-flow"),
  render: () => <StorybookWorkspaceSetup />,
};
