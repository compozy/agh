import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, HttpResponse } from "msw";
import { aghApiMock } from "@/storybook/openapi-msw";
import { expect, userEvent, waitFor, within } from "storybook/test";

import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
} from "@/storybook/route-story-meta";

const meta: Meta<typeof StorybookRouteCanvas> = {
  title: "systems/automation/routes/Triggers",
  component: StorybookRouteCanvas,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Full-page triggers route stories with the real shell, covering list/detail states, scope filtering, and editor flows.",
      },
    },
  },
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
    await waitFor(() => expect(canvas.getByTestId("triggers-scope-workspace")).toBeVisible(), {
      timeout: 5000,
    });
    const workspaceScope = canvas.getByTestId("triggers-scope-workspace");
    await userEvent.click(workspaceScope);
    await expect(workspaceScope).toHaveAttribute("aria-selected", "true");
  },
};

export const Empty: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/triggers"),
    ...storybookMswParameters({
      automation: [
        aghApiMock.get("/api/automation/triggers", () => HttpResponse.json({ triggers: [] })),
      ],
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
        aghApiMock.get("/api/automation/triggers", () =>
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
    await waitFor(() => expect(canvas.getByTestId("create-trigger-btn")).toBeEnabled(), {
      timeout: 5000,
    });
    await userEvent.click(canvas.getByTestId("create-trigger-btn"));
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
        aghApiMock.get("/api/automation/triggers", async () => {
          await delay("infinite");
          return HttpResponse.json({ triggers: [] });
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};
