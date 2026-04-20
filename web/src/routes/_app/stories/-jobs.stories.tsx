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
    "routes/app/jobs",
    "Full-page jobs route stories with the real shell, covering list/detail states, scope filtering, and editor flows."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/jobs"),
  render: () => <StorybookWorkspaceSetup />,
};

export const ScopeWorkspace: Story = {
  args: {},
  tags: ["play-fn"],
  parameters: appRouteParameters("/jobs"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("jobs-scope-workspace"));
    await expect(canvas.findByTestId("jobs-scope-workspace")).resolves.toHaveAttribute(
      "aria-selected",
      "true"
    );
  },
};

export const Empty: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/jobs"),
    ...storybookMswParameters({
      automation: [http.get("/api/automation/jobs", () => HttpResponse.json({ jobs: [] }))],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

export const JobsError: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/jobs"),
    ...storybookMswParameters({
      automation: [
        http.get("/api/automation/jobs", () =>
          HttpResponse.json({ error: "jobs unavailable" }, { status: 500 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

export const EditorCreate: Story = {
  args: {},
  tags: ["play-fn"],
  parameters: appRouteParameters("/jobs"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("create-job-btn"));
    await expect(within(document.body).findByTestId("automation-job-form")).resolves.toBeDefined();
  },
};

export const Loading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/jobs"),
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
