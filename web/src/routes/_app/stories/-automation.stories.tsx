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
export const JobsDefault: Story = {
  args: {},
  parameters: appRouteParameters("/automation"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Trigger management tab after switching from jobs.
 */
export const TriggersDefault: Story = {
  args: {},
  tags: ["play-fn"],
  parameters: appRouteParameters("/automation"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("automation-kind-triggers"));
    await expect(canvas.findByTestId("automation-item-trg_push_review")).resolves.toBeDefined();
  },
};

/**
 * Workspace scope filter selected.
 */
export const ScopeWorkspace: Story = {
  args: {},
  tags: ["play-fn"],
  parameters: appRouteParameters("/automation"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("automation-scope-workspace"));
    const scopeButton = await canvas.findByTestId("automation-scope-workspace");
    await expect(scopeButton).toHaveAttribute("aria-selected", "true");
  },
};

/**
 * Empty jobs branch when no automation jobs exist for the current filter.
 */
export const JobsEmpty: Story = {
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
 * Empty triggers branch when no triggers exist for the current filter.
 */
export const TriggersEmpty: Story = {
  args: {},
  tags: ["play-fn"],
  parameters: {
    ...appRouteParameters("/automation"),
    ...storybookMswParameters({
      automation: [http.get("/api/automation/triggers", () => HttpResponse.json({ triggers: [] }))],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("automation-kind-triggers"));
    await expect(canvas.findByText("No triggers configured")).resolves.toBeDefined();
  },
};

/**
 * Route-level error state when the initial jobs query fails.
 */
export const AutomationError: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/automation"),
    ...storybookMswParameters({
      automation: [
        http.get("/api/automation/jobs", () =>
          HttpResponse.json({ error: "automation unavailable" }, { status: 500 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Job creation flow opened from the page header CTA.
 */
export const EditorCreate: Story = {
  args: {},
  tags: ["play-fn"],
  parameters: appRouteParameters("/automation"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("create-automation-btn"));
    await expect(within(document.body).findByTestId("automation-job-form")).resolves.toBeDefined();
  },
};

/**
 * Editor dialog opened on the existing job via the edit affordance.
 */
export const EditorEdit: Story = {
  args: {},
  tags: ["play-fn"],
  parameters: appRouteParameters("/automation"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("edit-automation-btn"));
    await expect(within(document.body).findByTestId("automation-job-form")).resolves.toBeDefined();
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
