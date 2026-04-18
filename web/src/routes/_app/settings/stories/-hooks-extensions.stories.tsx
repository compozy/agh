import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";
import { expect, userEvent, within } from "storybook/test";

import { storybookMswParameters } from "@/storybook/msw";
import { settingsHooksExtensionsSectionFixture } from "@/systems/settings/mocks";
import {
  StorybookRestartBannerSetup,
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
  createRouteStoryMeta,
} from "@/storybook/route-story";

const meta: Meta<typeof StorybookRouteCanvas> = {
  ...createRouteStoryMeta(
    "routes/app/settings/hooks-extensions",
    "Hooks and extensions route stories covering policy editing, empty collections, toggle flows, and request boundary states."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default hooks and extensions surface with transport parity, hook declarations and installed extensions.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/settings/hooks-extensions"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Empty collections branch when no hooks are configured and no extensions are currently visible.
 */
export const EmptyCollections: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/hooks-extensions"),
    ...storybookMswParameters({
      settings: [
        http.get("/api/settings/hooks-extensions", () =>
          HttpResponse.json({
            ...settingsHooksExtensionsSectionFixture,
            hooks: [],
            installed: [],
          })
        ),
        http.get("/api/extensions", () => HttpResponse.json({ extensions: [] })),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Hook toggle flow that produces the route-level action result banner.
 */
export const ToggleHook: Story = {
  args: {},
  parameters: appRouteParameters("/settings/hooks-extensions"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(
      await canvas.findByTestId("settings-page-hooks-extensions-hooks-row-pre-commit-lint-toggle")
    );
    await expect(
      canvas.findByTestId("settings-page-hooks-extensions-action-result")
    ).resolves.toBeDefined();
  },
};

/**
 * Restart banner after saving policy that changes extension capabilities or declaration loading.
 */
export const RestartBanner: Story = {
  args: {},
  parameters: appRouteParameters("/settings/hooks-extensions"),
  render: () => (
    <>
      <StorybookWorkspaceSetup />
      <StorybookRestartBannerSetup section="hooks-extensions" />
    </>
  ),
};

/**
 * Loading state while the hooks/extensions section envelope is still fetching.
 */
export const Loading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/hooks-extensions"),
    ...storybookMswParameters({
      settings: [
        http.get("/api/settings/hooks-extensions", async () => {
          await delay("infinite");
          return HttpResponse.json({});
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Error branch shown when the hooks/extensions section request fails.
 */
export const Error: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/hooks-extensions"),
    ...storybookMswParameters({
      settings: [
        http.get("/api/settings/hooks-extensions", () =>
          HttpResponse.json({ error: "Failed to load hooks and extensions" }, { status: 500 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};
