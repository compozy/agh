import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";
import { expect, userEvent, within } from "storybook/test";

import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRestartBannerSetup,
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
  createRouteStoryMeta,
} from "@/storybook/route-story";

const meta: Meta<typeof StorybookRouteCanvas> = {
  ...createRouteStoryMeta(
    "routes/app/settings/memory",
    "Memory settings route stories covering the save surface, consolidation action, restart banner, and failure states."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default memory page with persistence controls and dream thresholds.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/settings/memory"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Consolidation action triggered from the dream section header.
 */
export const ConsolidateTriggered: Story = {
  args: {},
  parameters: appRouteParameters("/settings/memory"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("settings-page-memory-consolidate"));
    await expect(
      canvas.findByTestId("settings-page-memory-action-message")
    ).resolves.toHaveTextContent("Consolidation triggered");
  },
};

/**
 * Restart-required banner after a memory configuration mutation that affects daemon behavior.
 */
export const RestartBanner: Story = {
  args: {},
  parameters: appRouteParameters("/settings/memory"),
  render: () => (
    <>
      <StorybookWorkspaceSetup />
      <StorybookRestartBannerSetup section="memory" />
    </>
  ),
};

/**
 * Initial loading state while the memory section envelope is still resolving.
 */
export const Loading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/memory"),
    ...storybookMswParameters({
      settings: [
        http.get("/api/settings/memory", async () => {
          await delay("infinite");
          return HttpResponse.json({});
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Error branch shown when the memory settings request fails.
 */
export const Error: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/memory"),
    ...storybookMswParameters({
      settings: [
        http.get("/api/settings/memory", () =>
          HttpResponse.json({ error: "Failed to load memory settings" }, { status: 500 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};
