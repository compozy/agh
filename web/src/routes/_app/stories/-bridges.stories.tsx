import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";
import { expect, userEvent, within } from "storybook/test";

import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
  createRouteStoryMeta,
} from "@/storybook/route-story-meta";

const meta: Meta<typeof StorybookRouteCanvas> = {
  ...createRouteStoryMeta(
    "routes/app/bridges",
    "Real-shell stories for bridges, including list-detail composition, empty states and dialog-driven bridge operations."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default bridges route with the selected bridge detail visible.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/bridges"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Empty-state branch used before the workspace has any configured bridges.
 */
export const Empty: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/bridges"),
    ...storybookMswParameters({
      bridges: [
        http.get("/api/bridges", () => HttpResponse.json({ bridges: [], bridge_health: {} })),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Bridge creation dialog opened from the primary CTA.
 */
export const CreateDialog: Story = {
  args: {},
  parameters: appRouteParameters("/bridges"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("create-bridge-btn"));
    await expect(canvas.findByTestId("bridge-create-dialog")).resolves.toBeDefined();
  },
};

/**
 * Test delivery dialog opened from the selected bridge detail panel.
 */
export const TestDelivery: Story = {
  args: {},
  parameters: appRouteParameters("/bridges"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const button = await canvas.findByTestId("open-test-delivery-btn");
    await userEvent.click(button);
    await expect(canvas.findByTestId("bridge-test-delivery-dialog")).resolves.toBeDefined();
  },
};

/**
 * Initial loading state for the bridges list query.
 */
export const Loading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/bridges"),
    ...storybookMswParameters({
      bridges: [
        http.get("/api/bridges", async () => {
          await delay("infinite");
          return HttpResponse.json({ bridges: [], bridge_health: {} });
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};
