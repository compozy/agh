import { fileURLToPath } from "node:url";
import path from "node:path";

import { sessionLifecycleSelectors } from "../fixtures/selectors";
import { expect, test } from "../fixtures/test";
import { ensureGlobalWorkspace, useGlobalWorkspaceIfPrompted } from "../fixtures/workspace";

const browserLifecycleFixture = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
  "..",
  "..",
  "internal",
  "testutil",
  "acpmock",
  "testdata",
  "browser_session_lifecycle_fixture.json"
);
const mockAgentProvider = "acpmock";
const reasoningCatalogModel = "qa-browser-model-alt";
const reasoningCatalogModelLabel = "QA Browser Model Alt";

test.use({ viewport: { width: 1440, height: 900 } });

test("agent navigation renders the managed default agent after first-run setup", async ({
  appPage,
  runtime,
}) => {
  const ui = sessionLifecycleSelectors(appPage);

  await ensureGlobalWorkspace(runtime);
  await appPage.goto(runtime.url("/"), { waitUntil: "domcontentloaded" });
  await useGlobalWorkspaceIfPrompted(ui);

  await expect(appPage.getByTestId("agents-empty")).toHaveCount(0);
  await expect(ui.agentRow("general")).toBeVisible();
});

test("dashboard reports the agents endpoint failure during shell bootstrap", async ({
  page,
  runtime,
}) => {
  await page.route("**/api/agents**", async route => {
    await route.fulfill({
      status: 500,
      contentType: "application/json",
      body: JSON.stringify({ error: "agents unavailable" }),
    });
  });

  await ensureGlobalWorkspace(runtime);
  await page.goto(runtime.url("/"), { waitUntil: "domcontentloaded" });
  const ui = sessionLifecycleSelectors(page);
  await useGlobalWorkspaceIfPrompted(ui);

  await expect(page.getByRole("heading", { name: "Unable to load dashboard" })).toBeVisible({
    timeout: 20_000,
  });
  await expect(page.getByText("agents unavailable")).toBeVisible();
});

test.describe("seeded agent detail", () => {
  test.use({
    runtimeOptions: {
      seed: {
        mockAgents: [
          {
            fixturePath: browserLifecycleFixture,
            fixtureAgent: "browser-lifecycle-agent",
            agentName: "agent-detail-primary",
          },
          {
            fixturePath: browserLifecycleFixture,
            fixtureAgent: "browser-lifecycle-agent",
            agentName: "agent-detail-secondary",
            category_path: ["Engineering"],
          },
        ],
      },
    },
  });

  test("operator opens an agent detail page and changes the session command picker selection", async ({
    appPage,
  }) => {
    const ui = sessionLifecycleSelectors(appPage);

    await useGlobalWorkspaceIfPrompted(ui);

    await expect(ui.agentRow("agent-detail-primary")).toBeVisible();
    await expect(ui.agentRow("agent-detail-secondary")).toBeVisible();
    await ui.agentRow("agent-detail-primary").click();
    await expect.poll(() => new URL(appPage.url()).pathname).toBe("/agents/agent-detail-primary");

    await expect(appPage.getByTestId("agent-detail-page")).toBeVisible();
    await expect(appPage.getByRole("heading", { name: "agent-detail-primary" })).toBeVisible();
    await expect(appPage.getByRole("heading", { exact: true, name: "MCP Servers" })).toBeVisible();
    await expect(
      appPage.getByRole("heading", { exact: true, name: "No MCP servers" })
    ).toBeVisible();
    await expect(appPage.getByTestId("agent-sessions-empty")).toBeVisible();
    await expect(appPage.getByTestId("agent-stats-grid")).not.toContainText(
      String.fromCharCode(0x2014)
    );

    await appPage.getByTestId("agent-page-new-session").click();
    const trigger = appPage.getByTestId("session-create-agent-select");
    await expect(trigger).toBeVisible();
    await trigger.click();

    const secondary = appPage.getByTestId("agent-command-item-agent-detail-secondary");
    await expect(secondary).toBeVisible();
    await secondary.click();
    await expect(trigger).toContainText("agent-detail-secondary");

    await trigger.click();
    await expect(secondary).toHaveAttribute("data-checked", "true");
  });

  test("operator creates an agent with a persisted model and reasoning default", async ({
    appPage,
    runtime,
  }) => {
    if (!runtime.paths) {
      throw new Error("agent create browser test requires launch-mode runtime paths");
    }

    await ensureGlobalWorkspace(runtime);
    const workspace = await runtime.resolveWorkspace(runtime.paths.homeDir);
    await appPage.goto(runtime.url("/"), { waitUntil: "domcontentloaded" });
    await useGlobalWorkspaceIfPrompted(sessionLifecycleSelectors(appPage));

    await appPage.getByTestId("sidebar-create-agent").click();
    await expect(appPage.getByTestId("agent-create-dialog")).toBeVisible();
    await appPage.getByTestId("agent-create-name").fill("reasoning-default-agent");
    await appPage.getByTestId("agent-create-next").click();
    await expect(appPage.getByTestId("agent-create-runtime")).toBeVisible();

    const runtimeTrigger = appPage.getByTestId("agent-create-runtime-select");
    await runtimeTrigger.locator('button[data-focus="model"]').first().click();
    await expect(appPage.getByTestId("runtime-selector-popup")).toBeVisible();
    await appPage
      .locator(`[data-provider="${mockAgentProvider}"][data-model="${reasoningCatalogModel}"]`)
      .click();

    const reasoningStrip = appPage.getByTestId("runtime-selector-reasoning");
    await expect(reasoningStrip).toHaveAttribute("data-reasoning-mode", "levels");
    await reasoningStrip.locator('button[data-rz="high"]').click();
    await appPage.keyboard.press("Escape");
    await expect(runtimeTrigger).toContainText(reasoningCatalogModelLabel);
    await expect(runtimeTrigger.locator('button[data-focus="reasoning"]')).toContainText("High");

    await appPage.getByTestId("agent-create-next").click();
    await appPage.getByTestId("agent-create-prompt").fill("Keep runtime selection truthful.");
    await appPage.getByTestId("agent-create-next").click();
    await expect(appPage.getByTestId("agent-create-access")).toBeVisible();

    const createRequestPromise = appPage.waitForRequest(
      request => request.method() === "POST" && request.url().endsWith("/api/agents")
    );
    const createResponsePromise = appPage.waitForResponse(
      response => response.request().method() === "POST" && response.url().endsWith("/api/agents")
    );
    await appPage.getByTestId("submit-agent-create").click();

    const createRequest = await createRequestPromise;
    const createResponse = await createResponsePromise;
    expect(createRequest.postDataJSON()).toMatchObject({
      scope: "workspace",
      workspace: workspace.id,
      agent: {
        name: "reasoning-default-agent",
        provider: mockAgentProvider,
        model: reasoningCatalogModel,
        reasoning_effort: "high",
        prompt: "Keep runtime selection truthful.",
      },
    });
    expect(createResponse.ok()).toBe(true);

    const created = (await createResponse.json()) as {
      agent: { name: string; provider: string; model?: string; reasoning_effort?: string };
    };
    expect(created.agent).toMatchObject({
      name: "reasoning-default-agent",
      provider: mockAgentProvider,
      model: reasoningCatalogModel,
      reasoning_effort: "high",
    });
    await expect
      .poll(() => new URL(appPage.url()).pathname)
      .toBe("/agents/reasoning-default-agent");

    const rehydrated = await runtime.requestJSON<{
      agent: { name: string; provider: string; model?: string; reasoning_effort?: string };
    }>(`/api/agents/reasoning-default-agent?workspace=${encodeURIComponent(workspace.id)}`);
    expect(rehydrated.agent).toMatchObject({
      name: "reasoning-default-agent",
      provider: mockAgentProvider,
      model: reasoningCatalogModel,
      reasoning_effort: "high",
    });
  });
});
