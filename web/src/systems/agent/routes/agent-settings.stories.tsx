import type { Meta, StoryObj } from "@storybook/react-vite";
import { HttpResponse } from "msw";
import { fireEvent, expect, userEvent, within } from "storybook/test";

import { storyAgentNames, storyWorkspaceIds } from "@/storybook/fintech-scenario";
import { aghApiMock } from "@/storybook/openapi-msw";
import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
} from "@/storybook/route-story-meta";

const settingsRoute = `/agents/${storyAgentNames.fraud}/settings`;

function AgentWorkspaceSetup() {
  return <StorybookWorkspaceSetup workspaceId={storyWorkspaceIds.risk} />;
}

const meta: Meta<typeof StorybookRouteCanvas> = {
  title: "systems/agent/routes/AgentSettings",
  component: StorybookRouteCanvas,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Full-page agent settings editor with section nav, dirty topbar actions, and Danger zone.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Basics: Story = {
  args: {},
  parameters: appRouteParameters(settingsRoute),
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.findByTestId("agent-settings-page")).resolves.toBeDefined();
    await expect(canvas.findByTestId("agent-settings-basics")).resolves.toBeDefined();
    await expect(canvas.findByTestId("agent-settings-name")).resolves.toHaveAttribute("readonly");
  },
};

export const Danger: Story = {
  args: {},
  parameters: appRouteParameters(`${settingsRoute}?section=danger`),
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.findByTestId("agent-settings-danger")).resolves.toBeDefined();
    await expect(canvas.findByTestId("agent-settings-delete")).resolves.toBeDefined();
  },
};

export const Runtime: Story = {
  args: {},
  parameters: appRouteParameters(`${settingsRoute}?section=runtime`),
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.findByTestId("agent-settings-runtime")).resolves.toBeDefined();
  },
};

export const Instructions: Story = {
  args: {},
  parameters: appRouteParameters(`${settingsRoute}?section=instructions`),
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.findByTestId("agent-settings-instructions")).resolves.toBeDefined();
  },
};

export const Access: Story = {
  args: {},
  parameters: appRouteParameters(`${settingsRoute}?section=access`),
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.findByTestId("agent-settings-access")).resolves.toBeDefined();
  },
};

export const McpServers: Story = {
  args: {},
  parameters: appRouteParameters(`${settingsRoute}?section=mcp`),
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.findByTestId("agent-settings-mcp")).resolves.toBeDefined();
  },
};

export const DirtyInstructions: Story = {
  args: {},
  parameters: appRouteParameters(`${settingsRoute}?section=instructions`),
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const prompt = await canvas.findByTestId("agent-settings-prompt");
    await fireEvent.change(prompt, { target: { value: "Review every release gate before ship." } });
    await expect(canvas.findByTestId("agent-settings-unsaved")).resolves.toBeDefined();
    await expect(canvas.findByTestId("agent-settings-page-actions")).resolves.toBeDefined();
  },
};

export const DefinitionConflict: Story = {
  args: {},
  parameters: {
    ...appRouteParameters(`${settingsRoute}?section=instructions`),
    ...storybookMswParameters({
      agent: [
        aghApiMock.put("/api/agents/{name}", () =>
          HttpResponse.json({ error: "definition digest conflict" }, { status: 409 })
        ),
      ],
    }),
  },
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const prompt = await canvas.findByTestId("agent-settings-prompt");
    await fireEvent.change(prompt, { target: { value: "A concurrent definition update." } });
    await userEvent.click(await canvas.findByRole("button", { name: "Save changes" }));
    await expect(canvas.findByTestId("agent-settings-conflict-banner")).resolves.toBeDefined();
  },
};

export const PermissionDenied: Story = {
  args: {},
  parameters: {
    ...appRouteParameters(`${settingsRoute}?section=instructions`),
    ...storybookMswParameters({
      agent: [
        aghApiMock.put("/api/agents/{name}", () =>
          HttpResponse.json({ error: "mutation forbidden" }, { status: 403 })
        ),
      ],
    }),
  },
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const prompt = await canvas.findByTestId("agent-settings-prompt");
    await fireEvent.change(prompt, { target: { value: "An unauthorized definition update." } });
    await userEvent.click(await canvas.findByRole("button", { name: "Save changes" }));
    await expect(canvas.findByTestId("agent-settings-mutation-denied")).resolves.toBeDefined();
    await expect(
      canvas.findByTestId("page-actions-topbar-slot-blocked-caption")
    ).resolves.toHaveTextContent("Editing is not permitted for this agent.");
  },
};
