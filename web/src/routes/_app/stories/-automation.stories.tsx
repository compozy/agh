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
    "routes/app/automation",
    "Full-page automation route stories with the real shell, covering tabs, empty states, loading and the editor flows."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default jobs view with list, detail, and run history.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/automation"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Trigger management tab after switching from jobs.
 */
export const Triggers: Story = {
  args: {},
  parameters: appRouteParameters("/automation"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("automation-kind-triggers"));
    await expect(canvas.findByTestId("automation-list-panel")).resolves.toBeDefined();
  },
};

/**
 * Empty jobs branch when no automation jobs exist for the current filter.
 */
export const Empty: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/automation"),
    ...storybookMswParameters({
      automation: [http.get("/api/automation/jobs", () => HttpResponse.json({ jobs: [] }))],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Job creation flow opened from the page header CTA.
 */
export const CreateJob: Story = {
  args: {},
  parameters: appRouteParameters("/automation"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("create-automation-btn"));
    await expect(canvas.findByTestId("automation-job-form")).resolves.toBeDefined();
  },
};

/**
 * Route-level loading state before the initial jobs query resolves.
 */
export const Loading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/automation"),
    ...storybookMswParameters({
      automation: [
        http.get("/api/automation/jobs", async () => {
          await delay("infinite");
          return HttpResponse.json({ jobs: [] });
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};
