import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";

import { storybookMswParameters } from "@/storybook/msw";
import { StorybookFieldDirtySetup } from "@/storybook/settings-state-helpers";
import {
  StorybookRestartBannerSetup,
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
  createRouteStoryMeta,
} from "@/storybook/route-story";

const meta: Meta<typeof StorybookRouteCanvas> = {
  ...createRouteStoryMeta(
    "routes/app/settings/automation",
    "Automation settings route stories covering the default configuration surface plus loading, error, and restart-required states."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default automation settings page with runtime summary and scheduler controls.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/settings/automation"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Dirty shell state -- the timezone field has been edited so the save-bar
 * reads Unsaved changes + the Save button enables.
 */
export const Dirty: Story = {
  args: {},
  parameters: appRouteParameters("/settings/automation"),
  render: () => (
    <>
      <StorybookWorkspaceSetup />
      <StorybookFieldDirtySetup
        testId="settings-page-automation-timezone-input"
        value="America/Sao_Paulo"
      />
    </>
  ),
};

/**
 * Restart-required banner after changing automation config that affects the scheduler runtime.
 */
export const RestartBanner: Story = {
  args: {},
  parameters: appRouteParameters("/settings/automation"),
  render: () => (
    <>
      <StorybookWorkspaceSetup />
      <StorybookRestartBannerSetup section="automation" />
    </>
  ),
};

/**
 * Loading state while the automation section envelope is still fetching.
 */
export const Loading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/automation"),
    ...storybookMswParameters({
      settings: [
        http.get("/api/settings/automation", async () => {
          await delay("infinite");
          return HttpResponse.json({});
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Error branch shown when the automation settings request fails.
 */
export const Error: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/automation"),
    ...storybookMswParameters({
      settings: [
        http.get("/api/settings/automation", () =>
          HttpResponse.json({ error: "Failed to load automation settings" }, { status: 500 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};
