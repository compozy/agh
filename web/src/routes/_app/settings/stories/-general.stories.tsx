import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";

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
    "routes/app/settings/general",
    "General settings route stories rendered through the real app shell, including loading, error, and restart-required layout states."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default general settings page with runtime status and editable defaults.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/settings/general"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Restart-required banner state after a mutation touched daemon-wide configuration.
 */
export const RestartBanner: Story = {
  args: {},
  parameters: appRouteParameters("/settings/general"),
  render: () => (
    <>
      <StorybookWorkspaceSetup />
      <StorybookRestartBannerSetup section="general" />
    </>
  ),
};

/**
 * Initial loading state while the section envelope is still resolving.
 */
export const Loading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/general"),
    ...storybookMswParameters({
      settings: [
        http.get("/api/settings/general", async () => {
          await delay("infinite");
          return HttpResponse.json({});
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Error branch shown when the general settings request fails.
 */
export const Error: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/general"),
    ...storybookMswParameters({
      settings: [
        http.get("/api/settings/general", () =>
          HttpResponse.json({ error: "Failed to load general settings" }, { status: 500 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};
