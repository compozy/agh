import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";

import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookGeneralDraftDirtySetup,
  StorybookGeneralSavingSetup,
  StorybookRestartPhaseSetup,
} from "@/storybook/settings-state-helpers";
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
    "General settings route stories rendered through the real app shell, including loading, error, dirty, saving, and all restart banner tones."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default general settings page with runtime status and editable defaults.
 * Represents the idle shell state for the route story.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/settings/general"),
  render: () => <StorybookWorkspaceSetup />,
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

/**
 * Dirty shell state -- the default-agent field has been edited so the save-bar
 * reads Unsaved changes + the Save button enables.
 */
export const Dirty: Story = {
  args: {},
  parameters: appRouteParameters("/settings/general"),
  render: () => (
    <>
      <StorybookWorkspaceSetup />
      <StorybookGeneralDraftDirtySetup />
    </>
  ),
};

/**
 * Saving shell state -- the PATCH endpoint hangs so the Save button shows the
 * spinner + Saving... label for the story.
 */
export const Saving: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/general"),
    ...storybookMswParameters({
      settings: [
        http.patch("/api/settings/general", async () => {
          await delay("infinite");
          return HttpResponse.json({});
        }),
      ],
    }),
  },
  render: () => (
    <>
      <StorybookWorkspaceSetup />
      <StorybookGeneralSavingSetup />
    </>
  ),
};

/**
 * Restart-warning banner -- mutation recorded as restart-required.
 */
export const RestartWarning: Story = {
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
 * Restart-polling banner -- operation started, status still pending, spinner
 * visible in the banner.
 */
export const RestartPolling: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/general"),
    ...storybookMswParameters({
      settings: [
        http.get("/api/settings/restart/:operationId", async () => {
          await delay("infinite");
          return HttpResponse.json({});
        }),
      ],
    }),
  },
  render: () => (
    <>
      <StorybookWorkspaceSetup />
      <StorybookRestartPhaseSetup
        section="general"
        overrides={{
          mutationRestartRequired: true,
          operationId: "op_polling",
          status: "pending",
          activeSessionCount: 2,
        }}
      />
    </>
  ),
};

/**
 * Restart-success banner -- operation completed, Dismiss button visible.
 */
export const RestartSuccess: Story = {
  args: {},
  parameters: appRouteParameters("/settings/general"),
  render: () => (
    <>
      <StorybookWorkspaceSetup />
      <StorybookRestartPhaseSetup
        section="general"
        overrides={{
          mutationRestartRequired: true,
          operationId: "op_success",
          status: "ready",
        }}
      />
    </>
  ),
};

/**
 * Restart-failure banner -- operation failed with a reason suffix + Dismiss.
 */
export const RestartFailure: Story = {
  args: {},
  parameters: appRouteParameters("/settings/general"),
  render: () => (
    <>
      <StorybookWorkspaceSetup />
      <StorybookRestartPhaseSetup
        section="general"
        overrides={{
          mutationRestartRequired: true,
          operationId: "op_failure",
          status: "failed",
          failureReason: "helper exited non-zero",
        }}
      />
    </>
  ),
};
