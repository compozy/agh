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
    "routes/app/tasks/detail",
    "Task detail route stories rendered inside the persistent tasks shell, covering overview, tabbed panels, loading, and not-found branches."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default overview tab for the primary Storybook task detail route.
 */
export const Overview: Story = {
  args: {},
  parameters: appRouteParameters("/tasks/task_001"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Runs tab listing prior attempts and deep-links to run detail pages.
 */
export const RunsTab: Story = {
  args: {},
  parameters: appRouteParameters("/tasks/task_001"),
  render: () => <StorybookWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("tasks-detail-tab-runs"));
    await expect(canvas.findByTestId("tasks-detail-runs-panel")).resolves.toBeDefined();
  },
};

/**
 * Timeline tab with live events and sequence metadata.
 */
export const TimelineTab: Story = {
  args: {},
  parameters: appRouteParameters("/tasks/task_001"),
  render: () => <StorybookWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("tasks-detail-tab-timeline"));
    await expect(canvas.findByTestId("tasks-timeline-panel")).resolves.toBeDefined();
  },
};

/**
 * Multi-agent tab with descendants and an interleaved timeline.
 */
export const AgentsTab: Story = {
  args: {},
  parameters: appRouteParameters("/tasks/task_001"),
  render: () => <StorybookWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("tasks-detail-tab-agents"));
    await expect(canvas.findByTestId("tasks-multi-agent-panel")).resolves.toBeDefined();
  },
};

/**
 * Children tab showing the linked child task table.
 */
export const ChildrenTab: Story = {
  args: {},
  parameters: appRouteParameters("/tasks/task_001"),
  render: () => <StorybookWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("tasks-detail-tab-children"));
    await expect(canvas.findByTestId("tasks-detail-children-panel")).resolves.toBeDefined();
  },
};

/**
 * Dependencies tab showing the current dependency references.
 */
export const DependenciesTab: Story = {
  args: {},
  parameters: appRouteParameters("/tasks/task_001"),
  render: () => <StorybookWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("tasks-detail-tab-dependencies"));
    await expect(canvas.findByTestId("tasks-detail-dependencies-panel")).resolves.toBeDefined();
  },
};

/**
 * Orchestration tab with execution profile, reviews, bridge notifications, and
 * stream resume cards.
 */
export const OrchestrationTab: Story = {
  args: {},
  parameters: appRouteParameters("/tasks/task_001"),
  render: () => <StorybookWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("tasks-detail-tab-orchestration"));
    await expect(canvas.findByTestId("tasks-detail-orchestration-panel")).resolves.toBeDefined();
  },
};

/**
 * Loading branch while the detail payload is still being fetched.
 */
export const Loading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/tasks/task_001"),
    ...storybookMswParameters({
      tasks: [
        http.get("/api/tasks/:id", async () => {
          await delay("infinite");
          return HttpResponse.json({ task: null });
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Not-found branch for a missing task id while the shell remains mounted.
 */
export const NotFound: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/tasks/task_missing"),
    ...storybookMswParameters({
      tasks: [
        http.get("/api/tasks/:id", ({ params }) =>
          HttpResponse.json({ error: `Task not found: ${String(params.id)}` }, { status: 404 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};
