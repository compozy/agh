import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";
import { useEffect } from "react";
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
    "routes/app/settings/environments",
    "Environment profile route stories covering the table layout, empty state, editor flow, delete warnings, and request failures."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default environment catalog with workspace usage counts rendered in a @agh/ui Table.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/settings/environments"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Dirty editor state — the new-environment dialog is open with the name field
 * pre-filled so the Create button enables for the story.
 */
export const Dirty: Story = {
  args: {},
  parameters: appRouteParameters("/settings/environments"),
  render: () => (
    <>
      <StorybookWorkspaceSetup />
      <StorybookEnvironmentsDirtySetup />
    </>
  ),
};

/**
 * Empty-state branch shown before any environment profiles have been created.
 */
export const Empty: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/environments"),
    ...storybookMswParameters({
      settings: [
        http.get("/api/settings/environments", () => HttpResponse.json({ environments: [] })),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Create-environment editor opened from the collection header.
 */
export const CreateEnvironment: Story = {
  args: {},
  parameters: appRouteParameters("/settings/environments"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("settings-page-environments-create"));
    await userEvent.type(
      await canvas.findByTestId("settings-environments-editor-name-input"),
      "preview"
    );
    await userEvent.click(await canvas.findByTestId("settings-environments-editor-save"));
    await expect(
      canvas.findByTestId("settings-page-environments-action-result")
    ).resolves.toBeDefined();
  },
};

/**
 * Delete dialog with workspace usage warning for an environment still referenced by workspaces.
 */
export const DeleteProfile: Story = {
  args: {},
  parameters: appRouteParameters("/settings/environments"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(
      await canvas.findByTestId("settings-page-environments-card-local-delete")
    );
    await expect(canvas.findByTestId("settings-environments-delete-usage")).resolves.toBeDefined();
  },
};

/**
 * Loading state while the environments collection is still resolving.
 */
export const Loading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/environments"),
    ...storybookMswParameters({
      settings: [
        http.get("/api/settings/environments", async () => {
          await delay("infinite");
          return HttpResponse.json({ environments: [] });
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Error branch when the environments request fails.
 */
export const Error: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/environments"),
    ...storybookMswParameters({
      settings: [
        http.get("/api/settings/environments", () =>
          HttpResponse.json({ error: "Failed to load environments" }, { status: 500 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Reaches the dirty editor state by opening the new-environment dialog and
 * seeding the name field. RAF-polls until the route mounts.
 */
function StorybookEnvironmentsDirtySetup() {
  useEffect(() => {
    let cancelled = false;
    let stage: "open" | "fill" = "open";
    const setValue = (element: HTMLInputElement, next: string) => {
      const setter = Object.getOwnPropertyDescriptor(
        window.HTMLInputElement.prototype,
        "value"
      )?.set;
      setter?.call(element, next);
      element.dispatchEvent(new Event("input", { bubbles: true }));
      element.dispatchEvent(new Event("change", { bubbles: true }));
    };
    const advance = () => {
      if (cancelled) return;
      if (stage === "open") {
        const trigger = document.querySelector<HTMLButtonElement>(
          '[data-testid="settings-page-environments-create"]'
        );
        if (trigger) {
          trigger.click();
          stage = "fill";
        }
      } else if (stage === "fill") {
        const input = document.querySelector<HTMLInputElement>(
          '[data-testid="settings-environments-editor-name-input"]'
        );
        if (input) {
          setValue(input, "dirty-environment");
          return;
        }
      }
      requestAnimationFrame(advance);
    };
    requestAnimationFrame(advance);
    return () => {
      cancelled = true;
    };
  }, []);
  return null;
}
