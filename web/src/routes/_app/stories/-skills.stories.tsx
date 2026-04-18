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

const meta: Meta<typeof StorybookRouteCanvas> = {
  ...createRouteStoryMeta(
    "routes/app/skills",
    "Full-shell route stories for installed and marketplace skill browsing, including detail and content branches."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default installed-skills route with list and detail panels.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/skills"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Marketplace tab selected from the top-level route tabs.
 */
export const Marketplace: Story = {
  args: {},
  parameters: appRouteParameters("/skills"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("tab-marketplace"));
    await expect(canvas.findByTestId("marketplace-view")).resolves.toBeDefined();
  },
};

/**
 * Empty installed-skills branch when the skill catalog is empty.
 */
export const Empty: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/skills"),
    ...storybookMswParameters({
      skill: [http.get("/api/skills", () => HttpResponse.json({ skills: [] }))],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Deferred content fetch state after requesting full skill content.
 */
export const ViewContent: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/skills"),
    ...storybookMswParameters({
      skill: [
        http.get("/api/skills/:name/content", async () => {
          await delay("infinite");
          return HttpResponse.json({ content: "" });
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("skill-detail-view-content"));
    await expect(canvas.findByTestId("skill-detail-content-loading")).resolves.toBeDefined();
  },
};
