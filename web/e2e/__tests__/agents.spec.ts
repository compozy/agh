import { fileURLToPath } from "node:url";
import path from "node:path";

import { sessionLifecycleSelectors } from "../fixtures/selectors";
import { expect, test } from "../fixtures/test";

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

test.use({ viewport: { width: 1440, height: 900 } });

test("agent navigation renders the empty list state when no agents are installed", async ({
  appPage,
}) => {
  const ui = sessionLifecycleSelectors(appPage);

  await useGlobalWorkspaceIfPrompted(ui);

  await expect(appPage.getByTestId("agents-empty")).toBeVisible();
  await expect(appPage.getByTestId("agents-empty")).toContainText(
    "Run `agh install` to bootstrap AGH"
  );
});

test("agent navigation renders the error state when the agents endpoint fails", async ({
  page,
  runtime,
}) => {
  await page.route("**/api/agents", async route => {
    await route.fulfill({
      status: 500,
      contentType: "application/json",
      body: JSON.stringify({ error: "agents unavailable" }),
    });
  });

  await page.goto(runtime.url("/"), { waitUntil: "domcontentloaded" });
  const ui = sessionLifecycleSelectors(page);
  await useGlobalWorkspaceIfPrompted(ui);

  await expect(page.getByTestId("agents-error")).toBeVisible({ timeout: 20_000 });
  await expect(page.getByTestId("agents-error")).toContainText("Could not load agents");
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
    await expect(appPage.getByTestId("agent-page-header")).toContainText("agent-detail-primary");
    await expect(appPage.getByTestId("agent-info-panel")).toBeVisible();
    await expect(appPage.getByTestId("agent-info-mcp-servers")).toBeVisible();
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
});

async function useGlobalWorkspaceIfPrompted(
  sessionUI: ReturnType<typeof sessionLifecycleSelectors>
) {
  await Promise.race([
    sessionUI.workspaceOnboarding.waitFor({ state: "visible", timeout: 5_000 }).catch(() => null),
    sessionUI.appSidebar.waitFor({ state: "visible", timeout: 5_000 }).catch(() => null),
  ]);

  if (await sessionUI.workspaceOnboarding.isVisible().catch(() => false)) {
    await sessionUI.workspaceUseGlobal.click();
    await expect(sessionUI.workspaceOnboarding).toBeHidden();
  }

  await expect(sessionUI.appSidebar).toBeVisible();
}
