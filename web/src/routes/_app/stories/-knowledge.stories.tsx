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
    "Route stories for knowledge memory browsing with the real shell, covering tab filters, empty states, detail loading, and the delete confirmation dialog."
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
  tags: ["play-fn"],
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

/**
 * Detail panel error state when the memory content fetch fails.
 */
export const ContentError: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/knowledge"),
    ...storybookMswParameters({
      knowledge: [
        http.get("/api/memory/:filename", () =>
          HttpResponse.json({ error: "boom" }, { status: 500 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Delete confirmation dialog opened from the detail panel delete button.
 */
export const DeleteDialog: Story = {
  args: {},
  parameters: appRouteParameters("/knowledge"),
  render: () => <StorybookWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const deleteBtn = await canvas.findByTestId("delete-memory-btn");
    await userEvent.click(deleteBtn);
    await expect(
      within(canvasElement.ownerDocument.body).findByTestId("knowledge-delete-dialog")
    ).resolves.toBeDefined();
  },
};
