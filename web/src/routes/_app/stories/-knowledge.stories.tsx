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
    "routes/app/knowledge",
    "Route stories for knowledge memory browsing with the real shell, covering tab filters, list empties and detail loading states."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default knowledge view with list and detail panel populated from MSW fixtures.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/knowledge"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Workspace filter tab selected from the route header.
 */
export const WorkspaceTab: Story = {
  args: {},
  parameters: appRouteParameters("/knowledge"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("tab-workspace"));
    await expect(canvas.findByTestId("knowledge-list-panel")).resolves.toBeDefined();
  },
};

/**
 * Empty collection branch when no memories are available.
 */
export const Empty: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/knowledge"),
    ...storybookMswParameters({
      knowledge: [http.get("/api/memory", () => HttpResponse.json([]))],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Detail panel loading state while the selected memory content is still fetching.
 */
export const ContentLoading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/knowledge"),
    ...storybookMswParameters({
      knowledge: [
        http.get("/api/memory/:filename", async () => {
          await delay("infinite");
          return HttpResponse.json({ content: "" });
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};
