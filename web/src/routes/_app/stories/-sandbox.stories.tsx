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
    "routes/app/settings/sandboxes",
    "Sandbox profile route stories covering the table layout, empty state, editor flow, delete warnings, and request failures."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default sandbox catalog with workspace usage counts rendered in a @agh/ui Table.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/settings/sandboxes"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Dirty editor state , the new-sandbox dialog is open with the name field
 * pre-filled so the Create button enables for the story.
 */
export const Dirty: Story = {
  args: {},
  parameters: appRouteParameters("/settings/sandboxes"),
  render: () => (
    <>
      <StorybookWorkspaceSetup />
      <StorybookSandboxesDirtySetup />
    </>
  ),
};

/**
 * Empty-state branch shown before any sandbox profiles have been created.
 */
export const Empty: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/sandboxes"),
    ...storybookMswParameters({
      settings: [http.get("/api/settings/sandboxes", () => HttpResponse.json({ sandboxes: [] }))],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Create-sandbox editor opened from the collection header.
 */
export const CreateSandbox: Story = {
  args: {},
  parameters: appRouteParameters("/settings/sandboxes"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("settings-page-sandboxes-create"));
    await userEvent.type(
      await canvas.findByTestId("settings-sandboxes-editor-name-input"),
      "preview"
    );
    await userEvent.click(await canvas.findByTestId("settings-sandboxes-editor-save"));
    await expect(
      canvas.findByTestId("settings-page-sandboxes-action-result")
    ).resolves.toBeDefined();
  },
};

/**
 * Delete dialog with workspace usage warning for a sandbox still referenced by workspaces.
 */
export const DeleteProfile: Story = {
  args: {},
  parameters: appRouteParameters("/settings/sandboxes"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("settings-page-sandboxes-card-local-delete"));
    await expect(canvas.findByTestId("settings-sandboxes-delete-usage")).resolves.toBeDefined();
  },
};

/**
 * Loading state while the sandboxes collection is still resolving.
 */
export const Loading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/sandboxes"),
    ...storybookMswParameters({
      settings: [
        http.get("/api/settings/sandboxes", async () => {
          await delay("infinite");
          return HttpResponse.json({ sandboxes: [] });
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Error branch when the sandboxes request fails.
 */
export const Error: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/sandboxes"),
    ...storybookMswParameters({
      settings: [
        http.get("/api/settings/sandboxes", () =>
          HttpResponse.json({ error: "Failed to load sandboxes" }, { status: 500 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Reaches the dirty editor state by opening the new-sandbox dialog and
 * seeding the name field. RAF-polls until the route mounts.
 */
function StorybookSandboxesDirtySetup() {
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
          '[data-testid="settings-page-sandboxes-create"]'
        );
        if (trigger) {
          trigger.click();
          stage = "fill";
        }
      } else if (stage === "fill") {
        const input = document.querySelector<HTMLInputElement>(
          '[data-testid="settings-sandboxes-editor-name-input"]'
        );
        if (input) {
          setValue(input, "dirty-sandbox");
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
