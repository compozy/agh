import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, HttpResponse } from "msw";
import { aghApiMock } from "@/storybook/openapi-msw";
import { expect, userEvent, within } from "storybook/test";

import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
} from "@/storybook/route-story-meta";
import { MCPServerEditor } from "@/systems/settings/components/mcp-server-editor";
import { emptyDraft } from "@/systems/settings/lib/mcp-editor-model";
import { mcpManagementCollectionFixture } from "@/systems/settings/mocks";

const meta: Meta<typeof StorybookRouteCanvas> = {
  title: "systems/settings/routes/McpServers",
  component: StorybookRouteCanvas,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "MCP management route: composed status matrix, authorize/repair flow, and the stdio/HTTP/SSE editor. Stories back the Task 08 Visual Contract capture set.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// The nine-server reference matrix, plus a selectable Vault inventory entry.
const managementMsw = storybookMswParameters({
  settings: [
    aghApiMock.get("/api/settings/mcp-servers", () =>
      HttpResponse.json(mcpManagementCollectionFixture)
    ),
    aghApiMock.get("/api/vault/secrets", () =>
      HttpResponse.json({
        secrets: [
          {
            ref: "vault:mcp/ws/ws-platform/github-local/env/github_personal_access_token",
            namespace: "mcp",
            present: true,
            created_at: "2026-07-01T00:00:00Z",
            updated_at: "2026-07-01T00:00:00Z",
          },
        ],
      })
    ),
  ],
});

function managementParams(path: string) {
  return { ...appRouteParameters(path), ...managementMsw };
}

async function openAuthorizeWaiting(canvas: ReturnType<typeof within>) {
  await userEvent.click(
    await canvas.findByTestId("settings-page-mcp-servers-row-linear-authorize")
  );
  await canvas.findByTestId("settings-page-mcp-authorize-url");
}

/** matrix-desktop / matrix-mobile */
export const Matrix: Story = {
  parameters: managementParams("/marketplace/mcps?tab=installed"),
  render: () => <StorybookWorkspaceSetup />,
};

/** selected-needs-login-desktop */
export const SelectedNeedsLogin: Story = {
  parameters: managementParams("/marketplace/mcps?tab=installed&server=linear"),
  render: () => <StorybookWorkspaceSetup />,
};

/** authorize-waiting-desktop / authorize-mobile */
export const AuthorizeWaiting: Story = {
  tags: ["play-fn"],
  parameters: managementParams("/marketplace/mcps?tab=installed"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await openAuthorizeWaiting(canvas);
    await expect(canvas.getByTestId("settings-page-mcp-authorize-waiting")).toBeInTheDocument();
  },
};

/** authorize-manual-desktop */
export const AuthorizeManual: Story = {
  tags: ["play-fn"],
  parameters: managementParams("/marketplace/mcps?tab=installed"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await openAuthorizeWaiting(canvas);
    await userEvent.click(canvas.getByTestId("settings-page-mcp-authorize-manual-trigger"));
    await canvas.findByTestId("settings-page-mcp-authorize-manual");
  },
};

/** auth-failure-desktop */
export const AuthFailure: Story = {
  tags: ["play-fn"],
  parameters: {
    ...appRouteParameters("/marketplace/mcps?tab=installed"),
    ...storybookMswParameters({
      settings: [
        aghApiMock.get("/api/settings/mcp-servers", () =>
          HttpResponse.json(mcpManagementCollectionFixture)
        ),
        aghApiMock.get("/api/vault/secrets", () => HttpResponse.json({ secrets: [] })),
        // Exchange returns without a confirmed token -> the UI stays failed.
        aghApiMock.post("/api/settings/mcp-servers/{name}/auth/exchange", () =>
          HttpResponse.json({
            server_name: "linear",
            scope: "workspace",
            status: "needs_login",
            token_present: false,
            refreshable: true,
          })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await openAuthorizeWaiting(canvas);
    await userEvent.click(canvas.getByTestId("settings-page-mcp-authorize-manual-trigger"));
    await userEvent.type(
      await canvas.findByTestId("settings-page-mcp-authorize-manual-input"),
      "rejected-code"
    );
    await userEvent.click(canvas.getByTestId("settings-page-mcp-authorize-exchange"));
    await canvas.findByTestId("settings-page-mcp-authorize-failure");
  },
};

/** auth-begin-failure-desktop */
export const AuthBeginFailure: Story = {
  tags: ["play-fn"],
  parameters: {
    ...appRouteParameters("/marketplace/mcps?tab=installed"),
    ...storybookMswParameters({
      settings: [
        aghApiMock.get("/api/settings/mcp-servers", () =>
          HttpResponse.json(mcpManagementCollectionFixture)
        ),
        aghApiMock.get("/api/vault/secrets", () => HttpResponse.json({ secrets: [] })),
        aghApiMock.post("/api/settings/mcp-servers/{name}/auth/begin", () =>
          HttpResponse.json({ error: "OAuth provider unavailable" }, { status: 503 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(
      await canvas.findByTestId("settings-page-mcp-servers-row-linear-authorize")
    );
    await canvas.findByText("Authorization could not be started");
    await canvas.findByTestId("settings-page-mcp-authorize-retry");
  },
};

/** authenticated-token-desktop */
export const Authenticated: Story = {
  tags: ["play-fn"],
  parameters: managementParams("/marketplace/mcps?tab=installed"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await openAuthorizeWaiting(canvas);
    await userEvent.click(canvas.getByTestId("settings-page-mcp-authorize-manual-trigger"));
    await userEvent.type(
      await canvas.findByTestId("settings-page-mcp-authorize-manual-input"),
      "valid-code"
    );
    await userEvent.click(canvas.getByTestId("settings-page-mcp-authorize-exchange"));
    await canvas.findByTestId("settings-page-mcp-authorize-confirmed");
  },
};

/** editor-stdio-desktop */
export const EditorStdio: Story = {
  tags: ["play-fn"],
  parameters: managementParams("/marketplace/mcps?tab=installed"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(
      await canvas.findByTestId("settings-page-mcp-servers-row-github-local-edit")
    );
    await canvas.findByTestId("settings-mcp-editor-stdio");
  },
};

/** editor-stdio-incomplete-secret-desktop */
export const EditorStdioIncompleteSecret: Story = {
  render: () => (
    <div className="min-h-screen bg-canvas">
      <MCPServerEditor
        open
        mode="edit"
        draft={{
          ...emptyDraft("stdio"),
          name: "github-local",
          command: "npx",
          args: ["-y", "@modelcontextprotocol/server-github"],
          secretEnv: [
            {
              key: "GITHUB_PERSONAL_ACCESS_TOKEN",
              binding: { mode: "typed", existing: false, typedValue: "", vaultRef: "" },
            },
          ],
        }}
        scope="workspace"
        errors={{ secretEnv: { 0: "Enter a value or select a Vault reference" } }}
        isValid={false}
        isSaving={false}
        saveError={null}
        vaultInventory={{ status: "ready", refs: [] }}
        target="config"
        availableTargets={["config"]}
        entry={null}
        onChange={() => undefined}
        onTargetChange={() => undefined}
        onClose={() => undefined}
        onSave={() => undefined}
      />
    </div>
  ),
};

/** editor-http-desktop / remote-editor-mobile */
export const EditorHttp: Story = {
  tags: ["play-fn"],
  parameters: managementParams("/marketplace/mcps?tab=installed"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("settings-page-mcp-servers-row-linear-edit"));
    await canvas.findByTestId("settings-mcp-editor-remote");
  },
};

/** editor-sse-desktop */
export const EditorSse: Story = {
  tags: ["play-fn"],
  parameters: managementParams("/marketplace/mcps?tab=installed"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("settings-page-mcp-servers-row-sentry-edit"));
    await canvas.findByTestId("settings-mcp-editor-remote");
  },
};

/**
 * loading-desktop. Driven at global scope so the route's workspace-scope loader
 * resolves and the mounted matrix shows its in-component skeleton for the pending
 * global query.
 */
export const Loading: Story = {
  parameters: {
    ...appRouteParameters("/marketplace/mcps?tab=installed"),
    ...storybookMswParameters({
      settings: [
        aghApiMock.get("/api/settings/mcp-servers", async ({ request }) => {
          if (new URL(request.url).searchParams.get("scope") === "global") await delay("infinite");
          return HttpResponse.json(mcpManagementCollectionFixture);
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/** empty-desktop */
export const Empty: Story = {
  parameters: {
    ...appRouteParameters("/marketplace/mcps?tab=installed"),
    ...storybookMswParameters({
      settings: [
        aghApiMock.get("/api/settings/mcp-servers", () =>
          HttpResponse.json({ ...mcpManagementCollectionFixture, mcp_servers: [], scope: "global" })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * error-desktop. Global scope so the workspace-scope loader resolves; the mounted
 * matrix then renders its in-component retry block for the failing global query.
 */
export const Error: Story = {
  parameters: {
    ...appRouteParameters("/marketplace/mcps?tab=installed"),
    ...storybookMswParameters({
      settings: [
        aghApiMock.get("/api/settings/mcp-servers", ({ request }) => {
          if (new URL(request.url).searchParams.get("scope") === "global") {
            return HttpResponse.json({ error: "Failed to load MCP servers" }, { status: 500 });
          }
          return HttpResponse.json(mcpManagementCollectionFixture);
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};
