import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, HttpResponse } from "msw";
import { aghApiMock } from "@/storybook/openapi-msw";
import { useEffect } from "react";
import { expect, userEvent, within } from "storybook/test";

import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
} from "@/storybook/route-story-meta";
import { settingsMCPServersCollectionFixture } from "@/systems/settings/mocks";

const meta: Meta<typeof StorybookRouteCanvas> = {
  title: "systems/settings/routes/McpServers",
  component: StorybookRouteCanvas,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "MCP servers route stories covering workspace/global scope pills, editor and delete flows, and request failures.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default workspace-scope MCP catalog on the top-level System route.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/mcp"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Dirty editor state -- the add-server dialog is open with the name field
 * pre-filled, matching the mid-edit route state.
 */
export const Dirty: Story = {
  args: {},
  parameters: appRouteParameters("/mcp"),
  render: () => (
    <>
      <StorybookWorkspaceSetup />
      <StorybookMCPServersDirtySetup />
    </>
  ),
};

/**
 * Empty catalog branch exercising the `@agh/ui` `Empty` primitive when the
 * active scope returns no MCP servers.
 */
export const Empty: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/mcp"),
    ...storybookMswParameters({
      settings: [
        aghApiMock.get("/api/settings/mcp-servers", () =>
          HttpResponse.json({ ...settingsMCPServersCollectionFixture, mcp_servers: [] })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Global scope after switching from the workspace catalog.
 */
export const GlobalScope: Story = {
  args: {},
  parameters: appRouteParameters("/mcp"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("mcp-scope-global"));
    await expect(canvas.findByTestId("mcp-page-scope-label")).resolves.toHaveTextContent("global");
  },
};

/**
 * Server editor opened from the collection header and saved through the real mutation path.
 */
export const CreateServer: Story = {
  args: {},
  parameters: appRouteParameters("/mcp"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("mcp-page-create"));
    await userEvent.type(
      await canvas.findByTestId("settings-mcp-servers-editor-name-input"),
      "slack"
    );
    await userEvent.type(
      await canvas.findByTestId("settings-mcp-servers-editor-command-input"),
      "npx -y @modelcontextprotocol/server-slack"
    );
    await userEvent.click(await canvas.findByTestId("settings-mcp-servers-editor-save"));
    await expect(canvas.findByTestId("mcp-page-action-result")).resolves.toBeDefined();
  },
};

/**
 * Delete dialog showing how shadowed definitions become effective again after removal.
 */
export const DeleteServer: Story = {
  args: {},
  parameters: appRouteParameters("/mcp"),
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
    ...appRouteParameters("/mcp"),
    ...storybookMswParameters({
      settings: [
        aghApiMock.get("/api/settings/mcp-servers", async () => {
          await delay("infinite");
          return HttpResponse.json({ ...settingsMCPServersCollectionFixture, mcp_servers: [] });
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
    ...appRouteParameters("/mcp"),
    ...storybookMswParameters({
      settings: [
        aghApiMock.get("/api/settings/mcp-servers", () =>
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
          '[data-testid="mcp-page-create"]'
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
