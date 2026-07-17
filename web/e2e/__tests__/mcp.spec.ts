import { mkdtemp } from "node:fs/promises";
import os from "node:os";
import path from "node:path";

import { closeMCPAuthServer, startMCPAuthServer } from "../fixtures/mcp-auth-server";
import { sessionLifecycleSelectors } from "../fixtures/selectors";
import {
  type BrowserSettingsFixturesResult,
  cleanupBrowserSettingsFixtures,
  seedBrowserSettingsFixtures,
} from "../fixtures/runtime";
import { expect, test } from "../fixtures/test";
import { ensureGlobalWorkspace, useGlobalWorkspaceIfPrompted } from "../fixtures/workspace";

const SERVER_NAME = "linear";

test("operator authorizes an OAuth MCP server end to end against the fake AS and lands preselected", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const sessionUI = sessionLifecycleSelectors(appPage);
  const authServer = await startMCPAuthServer();
  const workspaceRoot = await mkdtemp(path.join(os.tmpdir(), "agh-mcp-authorize-"));
  const workspace = await runtime.resolveWorkspace(workspaceRoot);

  let seeded: BrowserSettingsFixturesResult | undefined;
  try {
    seeded = await seedBrowserSettingsFixtures(runtime, {
      mcpServers: [
        {
          name: SERVER_NAME,
          scope: "workspace",
          workspaceId: workspace.id,
          workspaceRootDir: workspaceRoot,
          target: "config",
          server: {
            name: SERVER_NAME,
            transport: "http",
            url: `${authServer.baseURL}/mcp`,
            auth: {
              type: "oauth2_pkce",
              client_id: "agh-e2e-public",
              issuer_url: authServer.baseURL,
              scopes: ["read", "write"],
            },
          },
        },
      ],
    });

    await ensureGlobalWorkspace(runtime);
    await useGlobalWorkspaceIfPrompted(sessionUI);

    await appPage.goto(runtime.url(`/mcp?scope=workspace&server=${SERVER_NAME}`), {
      waitUntil: "domcontentloaded",
    });
    await appPage.getByTestId("workspace-switcher").click();
    await appPage.getByTestId(`workspace-command-item-${workspace.id}`).click();

    const row = appPage.getByTestId(`settings-page-mcp-servers-row-${SERVER_NAME}`);
    await expect(row).toBeVisible();
    await expect(
      row.getByTestId(`settings-page-mcp-servers-row-${SERVER_NAME}-name`)
    ).toHaveAttribute("aria-current", "true");
    await expect(appPage.getByTestId("settings-page-mcp-selection")).toBeVisible();
    await expect(
      appPage.getByTestId(`settings-page-mcp-servers-row-${SERVER_NAME}-auth-pill`)
    ).toContainText("Needs login");

    await appPage.getByTestId(`settings-page-mcp-servers-row-${SERVER_NAME}-authorize`).click();
    const urlBlock = appPage.getByTestId("settings-page-mcp-authorize-url");
    await expect(urlBlock).toBeVisible();
    const liveUrl = (await urlBlock.locator("code").innerText()).trim();
    const state = new URL(liveUrl).searchParams.get("state");
    expect(state, "begin must return a live PKCE state").toBeTruthy();

    await appPage.getByTestId("settings-page-mcp-authorize-manual-trigger").click();
    const callbackUrl = runtime.url(
      `/api/mcp/oauth/callback?code=auth-code&state=${encodeURIComponent(state ?? "")}`
    );
    await appPage.getByTestId("settings-page-mcp-authorize-manual-input").fill(callbackUrl);
    await appPage.getByTestId("settings-page-mcp-authorize-exchange").click();

    await expect(appPage.getByTestId("settings-page-mcp-authorize-confirmed")).toBeVisible({
      timeout: 15_000,
    });
    await appPage.getByTestId("settings-page-mcp-authorize-done").click();

    await expect(
      appPage.getByTestId(`settings-page-mcp-servers-row-${SERVER_NAME}-auth-pill`)
    ).toContainText("Authenticated", { timeout: 15_000 });

    await browserArtifacts.captureScreenshot("mcp-authorize-confirmed", appPage);
  } finally {
    if (seeded) await cleanupBrowserSettingsFixtures(runtime, seeded);
    await closeMCPAuthServer(authServer.server);
  }
});

test("operator completes the browser authorize path and the scoped list polls to confirmation", async ({
  appPage,
  runtime,
}) => {
  const sessionUI = sessionLifecycleSelectors(appPage);
  const authServer = await startMCPAuthServer();
  const workspaceRoot = await mkdtemp(path.join(os.tmpdir(), "agh-mcp-browser-authorize-"));
  const workspace = await runtime.resolveWorkspace(workspaceRoot);

  let popup: typeof appPage | undefined;
  let seeded: BrowserSettingsFixturesResult | undefined;
  try {
    seeded = await seedBrowserSettingsFixtures(runtime, {
      mcpServers: [
        {
          name: SERVER_NAME,
          scope: "workspace",
          workspaceId: workspace.id,
          workspaceRootDir: workspaceRoot,
          target: "config",
          server: {
            name: SERVER_NAME,
            transport: "http",
            url: `${authServer.baseURL}/mcp`,
            auth: {
              type: "oauth2_pkce",
              client_id: "agh-e2e-public",
              issuer_url: authServer.baseURL,
              scopes: ["read", "write"],
            },
          },
        },
      ],
    });

    await ensureGlobalWorkspace(runtime);
    await useGlobalWorkspaceIfPrompted(sessionUI);
    await appPage.goto(runtime.url(`/mcp?scope=workspace&server=${SERVER_NAME}`), {
      waitUntil: "domcontentloaded",
    });
    await appPage.getByTestId("workspace-switcher").click();
    await appPage.getByTestId(`workspace-command-item-${workspace.id}`).click();
    await appPage.getByTestId(`settings-page-mcp-servers-row-${SERVER_NAME}-authorize`).click();

    const popupPromise = appPage.context().waitForEvent("page");
    await appPage.getByTestId("settings-page-mcp-authorize-open-url").click();
    popup = await popupPromise;

    await expect.poll(() => authServer.requests, { timeout: 10_000 }).toContain("GET /authorize");
    await expect.poll(() => authServer.requests, { timeout: 10_000 }).toContain("POST /token");
    await expect(appPage.getByTestId("settings-page-mcp-authorize-confirmed")).toBeVisible({
      timeout: 15_000,
    });
  } finally {
    if (popup && !popup.isClosed()) await popup.close();
    if (seeded) await cleanupBrowserSettingsFixtures(runtime, seeded);
    await closeMCPAuthServer(authServer.server);
  }
});
