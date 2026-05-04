import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";
import { useEffect } from "react";
import { expect, userEvent, within } from "storybook/test";

import { storyDefaultWorkspaceId, storyDefaultWorkspaceName } from "@/storybook/fintech-scenario";
import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
  createRouteStoryMeta,
} from "@/storybook/route-story";

const meta: Meta<typeof StorybookRouteCanvas> = {
  ...createRouteStoryMeta(
    "routes/app/settings/mcp-servers",
    "MCP server settings route stories covering global scope, workspace overrides, editor and delete flows, and request failures."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default global-scope MCP catalog rendered inside the settings shell.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/settings/mcp-servers"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Dirty editor state — the add-server dialog is open with the name field
 * pre-filled, matching the mid-edit route state.
 */
export const Dirty: Story = {
  args: {},
  parameters: appRouteParameters("/settings/mcp-servers"),
  render: () => (
    <>
      <StorybookWorkspaceSetup />
      <StorybookMCPServersDirtySetup />
    </>
  ),
};

/**
 * Empty catalog branch exercising the `@agh/ui` `Empty` primitive when the
 * global scope returns no MCP servers.
 */
export const Empty: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/mcp-servers"),
    ...storybookMswParameters({
      settings: [
        http.get("/api/settings/mcp-servers", () => HttpResponse.json({ mcp_servers: [] })),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Workspace scope after switching from the global catalog, showing the empty override branch.
 */
export const WorkspaceOverrides: Story = {
  args: {},
  parameters: appRouteParameters("/settings/mcp-servers"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(
      await canvas.findByTestId(
        `settings-page-mcp-servers-scope-workspace-${storyDefaultWorkspaceId}`
      )
    );
    await expect(
      canvas.findByTestId("settings-page-mcp-servers-scope-label")
    ).resolves.toHaveTextContent(storyDefaultWorkspaceName);
    await expect(canvas.findByTestId("settings-page-mcp-servers-empty")).resolves.toBeDefined();
  },
};

/**
 * Server editor opened from the collection header and saved through the real mutation path.
 */
export const CreateServer: Story = {
  args: {},
  parameters: appRouteParameters("/settings/mcp-servers"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("settings-page-mcp-servers-create"));
    await userEvent.type(
      await canvas.findByTestId("settings-mcp-servers-editor-name-input"),
      "slack"
    );
    await userEvent.type(
      await canvas.findByTestId("settings-mcp-servers-editor-command-input"),
      "npx -y @modelcontextprotocol/server-slack"
    );
    await userEvent.click(await canvas.findByTestId("settings-mcp-servers-editor-save"));
    await expect(
      canvas.findByTestId("settings-page-mcp-servers-action-result")
    ).resolves.toBeDefined();
  },
};

/**
 * Delete dialog showing how shadowed definitions become effective again after removal.
 */
export const DeleteServer: Story = {
  args: {},
  parameters: appRouteParameters("/settings/mcp-servers"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(
      await canvas.findByTestId("settings-page-mcp-servers-row-filesystem-delete")
    );
    await expect(
      canvas.findByTestId("settings-mcp-servers-delete-shadowed")
    ).resolves.toBeDefined();
  },
};

/**
 * Loading state while the scoped MCP catalog is still fetching.
 */
export const Loading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/mcp-servers"),
    ...storybookMswParameters({
      settings: [
        http.get("/api/settings/mcp-servers", async () => {
          await delay("infinite");
          return HttpResponse.json({ mcp_servers: [] });
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Error branch when the MCP settings request cannot be loaded.
 */
export const Error: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/mcp-servers"),
    ...storybookMswParameters({
      settings: [
        http.get("/api/settings/mcp-servers", () =>
          HttpResponse.json({ error: "Failed to load MCP servers" }, { status: 500 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Reaches the dirty editor state by opening the add-server dialog and seeding
 * the name field. RAF-polls until the route mounts.
 */
function StorybookMCPServersDirtySetup() {
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
          '[data-testid="settings-page-mcp-servers-create"]'
        );
        if (trigger) {
          trigger.click();
          stage = "fill";
        }
      } else if (stage === "fill") {
        const input = document.querySelector<HTMLInputElement>(
          '[data-testid="settings-mcp-servers-editor-name-input"]'
        );
        if (input) {
          setValue(input, "dirty-server");
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
