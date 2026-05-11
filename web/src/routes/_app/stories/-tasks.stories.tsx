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
    "routes/app/tasks",
    "Real-shell stories for the tasks workspace route, covering list, empty, kanban, dashboard, inbox, and loading/error states."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default list mode with the split pane and preview panel rendered in the app shell.
 */
export const DefaultList: Story = {
  args: {},
  parameters: appRouteParameters("/tasks"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Empty-state branch before any task contracts exist in the selected workspace.
 */
export const Empty: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/tasks"),
    ...storybookMswParameters({
      tasks: [http.get("/api/tasks", () => HttpResponse.json({ tasks: [] }))],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Kanban mode reached from the real mode pills in the tasks header.
 */
export const Kanban: Story = {
  args: {},
  parameters: appRouteParameters("/tasks"),
  render: () => <StorybookWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("tasks-mode-kanban"));
    await expect(canvas.findByTestId("tasks-kanban-board")).resolves.toBeDefined();
  },
};

/**
 * Dashboard mode with the aggregate cards and queue health rendered from MSW data.
 */
export const Dashboard: Story = {
  args: {},
  parameters: appRouteParameters("/tasks"),
  render: () => <StorybookWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("tasks-mode-dashboard"));
    await expect(canvas.findByTestId("tasks-dashboard-view")).resolves.toBeDefined();
  },
};

/**
 * Inbox mode with grouped actionable items and lane tabs rendered through the app route.
 */
export const Inbox: Story = {
  args: {},
  parameters: appRouteParameters("/tasks"),
  render: () => <StorybookWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("tasks-mode-inbox"));
    await expect(canvas.findByTestId("tasks-inbox-view")).resolves.toBeDefined();
    await expect(canvas.findByTestId("tasks-inbox-group-needs_review")).resolves.toBeDefined();
  },
};

/**
 * List rail loading state while the tasks collection request is still pending.
 */
export const Loading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/tasks"),
    ...storybookMswParameters({
      tasks: [
        http.get("/api/tasks", async () => {
          await delay("infinite");
          return HttpResponse.json({ tasks: [] });
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Dashboard error branch shown after switching modes when the aggregate endpoint fails.
 */
export const Error: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/tasks"),
    ...storybookMswParameters({
      tasks: [
        http.get("/api/observe/tasks/dashboard", () =>
          HttpResponse.json({ error: "Dashboard unavailable" }, { status: 500 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("tasks-mode-dashboard"));
    await expect(canvas.findByTestId("tasks-dashboard-error")).resolves.toBeDefined();
  },
};
