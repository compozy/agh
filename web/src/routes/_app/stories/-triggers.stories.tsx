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
    "routes/app/triggers",
    "Full-page triggers route stories with the real shell, covering list/detail states, scope filtering, and editor flows."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/triggers"),
  render: () => <StorybookWorkspaceSetup />,
};

export const ScopeWorkspace: Story = {
  args: {},
  tags: ["play-fn"],
  parameters: appRouteParameters("/triggers"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("triggers-scope-workspace"));
    await expect(canvas.findByTestId("triggers-scope-workspace")).resolves.toHaveAttribute(
      "aria-selected",
      "true"
    );
  },
};

export const Empty: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/triggers"),
    ...storybookMswParameters({
      automation: [http.get("/api/automation/triggers", () => HttpResponse.json({ triggers: [] }))],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

export const TriggersError: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/triggers"),
    ...storybookMswParameters({
      automation: [
        http.get("/api/automation/triggers", () =>
          HttpResponse.json({ error: "triggers unavailable" }, { status: 500 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

export const EditorCreate: Story = {
  args: {},
  tags: ["play-fn"],
  parameters: appRouteParameters("/triggers"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("create-trigger-btn"));
    await expect(
      within(document.body).findByTestId("automation-trigger-form")
    ).resolves.toBeDefined();
  },
};

export const Loading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/triggers"),
    ...storybookMswParameters({
      automation: [
        http.get("/api/automation/triggers", async () => {
          await delay("infinite");
          return HttpResponse.json({ triggers: [] });
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};
