import { mkdtemp } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import process from "node:process";

import { settingsOperatorSelectors, sessionLifecycleSelectors } from "../fixtures/selectors";
import {
  browserSettingsOperatorFlowScenario,
  cleanupBrowserSettingsFixtures,
  seedBrowserSettingsFixtures,
} from "../fixtures/runtime";
import { expect, test } from "../fixtures/test";

test.use({
  runtimeOptions: {
    env: {
      ...process.env,
      AGH_TEST_TELEGRAM_TOKEN: "telegram-bot-token",
    },
  },
});

test("operator can navigate the settings shell and complete a restart-aware general save that survives refresh polling", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const sessionUI = sessionLifecycleSelectors(appPage);
  const settingsUI = settingsOperatorSelectors(appPage);

  await useGlobalWorkspaceIfPrompted(sessionUI);
  await appPage.goto(runtime.url("/settings/general"), { waitUntil: "domcontentloaded" });
  await expect(settingsUI.shell.shell).toBeVisible({ timeout: 20_000 });
  await expect(settingsUI.shell.sectionNav).toBeVisible({ timeout: 20_000 });

  await expect
    .poll(async () => normalizeTexts(await settingsUI.shell.sectionItems.allTextContents()))
    .toEqual([
      "General",
      "Providers",
      "Vault",
      "MCP Servers",
      "Memory",
      "Skills",
      "Automation",
      "Network",
      "Observability",
      "Hooks & Extensions",
    ]);

  await expect.poll(() => new URL(appPage.url()).pathname).toBe("/settings/general");
  await expect(settingsUI.shell.sectionActive("general")).toBeVisible();
  await expect(settingsUI.general.page).toBeVisible();

  await settingsUI.shell.sectionLink("network").click();
  await expect.poll(() => new URL(appPage.url()).pathname).toBe("/settings/network");
  await expect(settingsUI.shell.sectionActive("network")).toBeVisible();

  await settingsUI.shell.sectionLink("hooks-extensions").click();
  await expect.poll(() => new URL(appPage.url()).pathname).toBe("/settings/hooks-extensions");
  await expect(settingsUI.shell.sectionActive("hooks-extensions")).toBeVisible();

  await appPage.goBack({ waitUntil: "domcontentloaded" });
  await expect.poll(() => new URL(appPage.url()).pathname).toBe("/settings/network");
  await appPage.goForward({ waitUntil: "domcontentloaded" });
  await expect.poll(() => new URL(appPage.url()).pathname).toBe("/settings/hooks-extensions");

  await settingsUI.shell.sectionLink("general").click();
  await expect.poll(() => new URL(appPage.url()).pathname).toBe("/settings/general");
  await expect(settingsUI.general.page).toBeVisible();

  const nextTimeoutValue = await nextSessionTimeoutValue(settingsUI.general.sessionTimeoutInput);
  await settingsUI.general.sessionTimeoutInput.fill(nextTimeoutValue);
  await expect(settingsUI.general.saveButton).toBeEnabled();
  await settingsUI.general.saveButton.click();

  await expect(settingsUI.general.restartBanner).toBeVisible();
  await expect(settingsUI.general.restartBannerMessage).toContainText(
    "Changes saved. Restart the daemon to apply."
  );
  await expect(settingsUI.general.restartBannerTrigger).toBeVisible();
  await browserArtifacts.captureScreenshot("tc-func-001-settings-shell-navigation", appPage);

  await settingsUI.general.restartAction.click();

  let operationID = "";
  await expect
    .poll(async () => {
      operationID = (await settingsUI.general.restartBannerOp.textContent())?.trim() ?? "";
      return operationID;
    })
    .toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i);

  await expect(settingsUI.general.restartBannerMessage).toContainText("Restarting daemon");
  await browserArtifacts.captureScreenshot("tc-func-002-general-restart-polling", appPage);

  await reloadDaemonServedPage(appPage, runtime, "/settings/general");
  if (await settingsUI.general.restartBanner.isVisible().catch(() => false)) {
    await expect(settingsUI.general.restartBannerOp).toContainText(operationID);
  } else {
    const payload = await runtime.requestJSON<{ status: string }>(
      `/api/settings/actions/restart/${encodeURIComponent(operationID)}`
    );
    expect(payload.status).toBe("ready");
  }

  await expect
    .poll(
      async () => {
        const payload = await runtime.requestJSON<{ status: string }>(
          `/api/settings/actions/restart/${encodeURIComponent(operationID)}`
        );
        return payload.status;
      },
      {
        timeout: 45_000,
      }
    )
    .toBe("ready");

  if (await settingsUI.general.restartBanner.isVisible().catch(() => false)) {
    await expect(settingsUI.general.restartBannerMessage).toContainText(
      "Daemon restarted successfully"
    );
  }
  await browserArtifacts.captureScreenshot("tc-int-016-general-restart-ready", appPage);
});

test("operator can distinguish skills actions that apply now from policy changes that require restart", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const sessionUI = sessionLifecycleSelectors(appPage);
  const settingsUI = settingsOperatorSelectors(appPage);
  const seeded = await seedBrowserSettingsFixtures(runtime, {
    disabledSkills: [browserSettingsOperatorFlowScenario.skills.disabledSkill],
  });

  try {
    await useGlobalWorkspaceIfPrompted(sessionUI);
    await appPage.goto(runtime.url("/settings/skills"), { waitUntil: "domcontentloaded" });

    await expect(settingsUI.skills.page).toBeVisible();
    await expect(settingsUI.skills.disabledList).toBeVisible();
    await expect(
      settingsUI.skills.disabledToggle(browserSettingsOperatorFlowScenario.skills.disabledSkill)
    ).toBeVisible();

    await settingsUI.skills
      .disabledToggle(browserSettingsOperatorFlowScenario.skills.disabledSkill)
      .click();
    await expect(settingsUI.skills.disabledSave).toBeEnabled();
    await settingsUI.skills.disabledSave.click();

    await expect(settingsUI.skills.disabledApplied).toContainText("applied immediately");
    await expect(settingsUI.skills.restartBanner).not.toBeVisible();

    await settingsUI.skills.operationalLink.click();
    await expect.poll(() => new URL(appPage.url()).pathname).toBe("/skills");
    await appPage.goBack({ waitUntil: "domcontentloaded" });
    await expect.poll(() => new URL(appPage.url()).pathname).toBe("/settings/skills");

    await settingsUI.skills.policyRegistryInput.fill("clawhub");
    await settingsUI.skills.policyBaseURLInput.fill("https://skills.example/browser-updated");
    await expect(settingsUI.skills.policySave).toBeEnabled();
    await settingsUI.skills.policySave.click();

    await expect(settingsUI.skills.policyApplied).toContainText("restart required");
    await expect(settingsUI.skills.restartBanner).toBeVisible();
    await expect(settingsUI.skills.restartBanner).toContainText("Restart the daemon");
    await browserArtifacts.captureScreenshot("tc-func-005-skills-applied-now-vs-restart", appPage);
  } finally {
    await cleanupBrowserSettingsFixtures(runtime, seeded);
  }
});

test("operator can replace a builtin provider with a config overlay and delete it back to builtin fallback", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const sessionUI = sessionLifecycleSelectors(appPage);
  const settingsUI = settingsOperatorSelectors(appPage);
  const builtinProviderName = await pickBuiltinProviderName(runtime);

  await useGlobalWorkspaceIfPrompted(sessionUI);
  await appPage.goto(runtime.url("/settings/providers"), { waitUntil: "domcontentloaded" });

  await expect(settingsUI.providers.page).toBeVisible();
  await expect(settingsUI.providers.list).toBeVisible();
  await expect(settingsUI.providers.card(builtinProviderName)).toBeVisible();

  await settingsUI.providers.editCard(builtinProviderName).click();
  await expect(settingsUI.providers.editor).toBeVisible();
  await settingsUI.providers.editorCommandInput.fill(
    browserSettingsOperatorFlowScenario.providers.overlayCommand
  );
  await settingsUI.providers.editorModelInput.fill(
    browserSettingsOperatorFlowScenario.providers.overlayModel
  );
  await settingsUI.providers.editorSave.click();

  await expect(settingsUI.providers.editor).toBeHidden();
  await expect(settingsUI.providers.actionResult).toContainText(
    `Saved provider "${builtinProviderName}"`
  );
  await expect(settingsUI.providers.actionResult).toContainText("restart required");
  await expect(settingsUI.providers.cardCommand(builtinProviderName)).toContainText(
    browserSettingsOperatorFlowScenario.providers.overlayCommand
  );
  await expect(settingsUI.providers.cardSource(builtinProviderName)).toContainText(/config/i);

  await settingsUI.providers.deleteCard(builtinProviderName).click();
  await expect(settingsUI.providers.deleteDialog).toBeVisible();
  await settingsUI.providers.deleteConfirm.click();

  await expect(settingsUI.providers.actionResult).toContainText(
    `Deleted overlay for "${builtinProviderName}"`
  );
  await expect(settingsUI.providers.actionResult).toContainText("builtin fallback now effective");
  await expect(settingsUI.providers.card(builtinProviderName)).toBeVisible();
  await expect(settingsUI.providers.cardSource(builtinProviderName)).toContainText(/builtin/i);
  await browserArtifacts.captureScreenshot(
    "tc-func-008-providers-crud-and-builtin-fallback",
    appPage
  );
});

test("operator can manage MCP servers across global and workspace scopes with visible target semantics", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const sessionUI = sessionLifecycleSelectors(appPage);
  const settingsUI = settingsOperatorSelectors(appPage);
  const workspaceRoot = await mkdtemp(path.join(os.tmpdir(), "agh-settings-mcp-workspace-"));
  const workspace = await runtime.resolveWorkspace(workspaceRoot);

  await useGlobalWorkspaceIfPrompted(sessionUI);
  await appPage.goto(runtime.url("/settings/mcp-servers"), { waitUntil: "domcontentloaded" });

  await expect(settingsUI.mcpServers.page).toBeVisible();
  await expect(settingsUI.mcpServers.scopeGlobal).toBeVisible();
  await expect(settingsUI.mcpServers.scopeWorkspace(workspace.id)).toBeVisible();

  await createMCPServerViaUI(settingsUI, {
    name: browserSettingsOperatorFlowScenario.mcpServers.global.name,
    command: browserSettingsOperatorFlowScenario.mcpServers.global.command,
    target: browserSettingsOperatorFlowScenario.mcpServers.global.target,
  });

  await expect(settingsUI.mcpServers.actionResult).toContainText(
    `Saved "${browserSettingsOperatorFlowScenario.mcpServers.global.name}"`
  );
  await expect(settingsUI.mcpServers.actionResult).toContainText("persisted to GLOBAL MCP");
  await expect(
    settingsUI.mcpServers.row(browserSettingsOperatorFlowScenario.mcpServers.global.name)
  ).toBeVisible();

  await settingsUI.mcpServers.scopeWorkspace(workspace.id).click();
  await expect(settingsUI.mcpServers.scopeLabel).toContainText(workspace.name);

  await createMCPServerViaUI(settingsUI, {
    name: browserSettingsOperatorFlowScenario.mcpServers.workspace.name,
    command: browserSettingsOperatorFlowScenario.mcpServers.workspace.command,
    target: browserSettingsOperatorFlowScenario.mcpServers.workspace.target,
  });

  await expect(settingsUI.mcpServers.actionResult).toContainText(
    `Saved "${browserSettingsOperatorFlowScenario.mcpServers.workspace.name}"`
  );
  await expect(settingsUI.mcpServers.actionResult).toContainText("persisted to WS CFG");
  await expect(
    settingsUI.mcpServers.row(browserSettingsOperatorFlowScenario.mcpServers.workspace.name)
  ).toBeVisible();

  await settingsUI.mcpServers.scopeGlobal.click();
  await expect(
    settingsUI.mcpServers.row(browserSettingsOperatorFlowScenario.mcpServers.global.name)
  ).toBeVisible();
  await expect(
    settingsUI.mcpServers.row(browserSettingsOperatorFlowScenario.mcpServers.workspace.name)
  ).not.toBeVisible();

  await settingsUI.mcpServers.scopeWorkspace(workspace.id).click();
  await expect(
    settingsUI.mcpServers.row(browserSettingsOperatorFlowScenario.mcpServers.workspace.name)
  ).toBeVisible();

  await settingsUI.mcpServers
    .deleteRow(browserSettingsOperatorFlowScenario.mcpServers.workspace.name)
    .click();
  await expect(settingsUI.mcpServers.deleteDialog).toBeVisible();
  await settingsUI.mcpServers.deleteConfirm.click();

  await expect(settingsUI.mcpServers.actionResult).toContainText(
    `Deleted "${browserSettingsOperatorFlowScenario.mcpServers.workspace.name}"`
  );
  await expect(
    settingsUI.mcpServers.row(browserSettingsOperatorFlowScenario.mcpServers.workspace.name)
  ).not.toBeVisible();
  await browserArtifacts.captureScreenshot("tc-int-011-mcp-workspace-scope", appPage);
});

test("operator can distinguish restart-aware hook edits from immediate extension operations on hooks and extensions", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const sessionUI = sessionLifecycleSelectors(appPage);
  const settingsUI = settingsOperatorSelectors(appPage);
  const seeded = await seedBrowserSettingsFixtures(runtime, {
    hooks: [
      {
        name: browserSettingsOperatorFlowScenario.hooksExtensions.hookName,
        declaration: {
          name: browserSettingsOperatorFlowScenario.hooksExtensions.hookName,
          event: "turn.end",
          mode: "sync",
          command: "/bin/echo",
          args: ["settings-hook"],
          matcher: {},
          required: true,
        },
      },
    ],
    installBridgeExtension: true,
  });

  try {
    await useGlobalWorkspaceIfPrompted(sessionUI);
    await appPage.goto(runtime.url("/settings/hooks-extensions"), {
      waitUntil: "domcontentloaded",
    });

    await expect(settingsUI.hooksExtensions.page).toBeVisible();
    await expect(
      settingsUI.hooksExtensions.hookToggle(
        browserSettingsOperatorFlowScenario.hooksExtensions.hookName
      )
    ).toBeVisible();
    await expect(
      settingsUI.hooksExtensions.extensionToggle(
        browserSettingsOperatorFlowScenario.hooksExtensions.extensionName
      )
    ).toBeVisible();

    await settingsUI.hooksExtensions
      .hookToggle(browserSettingsOperatorFlowScenario.hooksExtensions.hookName)
      .click();
    await expect(settingsUI.hooksExtensions.actionResult).toContainText(
      `Hook "${browserSettingsOperatorFlowScenario.hooksExtensions.hookName}" disabled`
    );
    await expect(settingsUI.hooksExtensions.actionResult).toContainText(
      "restart required to reload"
    );
    await expect(settingsUI.hooksExtensions.restartBanner).toBeVisible();

    await settingsUI.hooksExtensions
      .extensionToggle(browserSettingsOperatorFlowScenario.hooksExtensions.extensionName)
      .click();
    await expect(settingsUI.hooksExtensions.actionResult).toContainText(
      `Extension "${browserSettingsOperatorFlowScenario.hooksExtensions.extensionName}" disabled`
    );
    await expect(settingsUI.hooksExtensions.actionResult).toContainText("applied immediately");
    await expect(settingsUI.hooksExtensions.restartBanner).toBeVisible();

    await settingsUI.hooksExtensions.policyRegistryInput.fill("github");
    await settingsUI.hooksExtensions.policyBaseURLInput.fill(
      "https://extensions.example/browser-updated"
    );
    await expect(settingsUI.hooksExtensions.policySave).toBeEnabled();
    await settingsUI.hooksExtensions.policySave.click();

    await expect(settingsUI.hooksExtensions.actionResult).toContainText("Policy saved");
    await expect(settingsUI.hooksExtensions.actionResult).toContainText("restart required");
    await expect(settingsUI.hooksExtensions.restartBanner).toBeVisible();
    await browserArtifacts.captureScreenshot("tc-func-012-hooks-extensions-hybrid", appPage);
  } finally {
    await cleanupBrowserSettingsFixtures(runtime, seeded);
  }
});

async function useGlobalWorkspaceIfPrompted(
  sessionUI: ReturnType<typeof sessionLifecycleSelectors>
) {
  await Promise.race([
    sessionUI.workspaceOnboarding.waitFor({ state: "visible", timeout: 20_000 }).catch(() => null),
    sessionUI.appSidebar.waitFor({ state: "visible", timeout: 20_000 }).catch(() => null),
  ]);

  if (await sessionUI.workspaceOnboarding.isVisible().catch(() => false)) {
    await sessionUI.workspaceUseGlobal.click();
    await expect(sessionUI.workspaceOnboarding).toBeHidden();
  }

  await expect(sessionUI.appSidebar).toBeVisible({ timeout: 20_000 });
}

function normalizeTexts(values: string[]): string[] {
  return values.map(value => value.trim()).filter(value => value !== "");
}

async function nextSessionTimeoutValue(
  input: ReturnType<typeof settingsOperatorSelectors>["general"]["sessionTimeoutInput"]
): Promise<string> {
  const currentValue = Number.parseInt((await input.inputValue()) || "0", 10);
  const primary = browserSettingsOperatorFlowScenario.general.primarySessionTimeoutSeconds;
  const fallback = browserSettingsOperatorFlowScenario.general.fallbackSessionTimeoutSeconds;
  return String(currentValue === primary ? fallback : primary);
}

async function reloadDaemonServedPage(
  page: import("@playwright/test").Page,
  runtime: { url(pathname?: string): string },
  pathname: string
) {
  const targetURL = runtime.url(pathname);

  await expect
    .poll(
      async () => {
        try {
          await page.goto(targetURL, {
            waitUntil: "domcontentloaded",
            timeout: 2_000,
          });
          return new URL(page.url()).pathname;
        } catch {
          return "";
        }
      },
      {
        timeout: 15_000,
        intervals: [250, 500, 1_000],
      }
    )
    .toBe(pathname);
}

async function pickBuiltinProviderName(runtime: {
  requestJSON<T>(pathname: string, init?: RequestInit): Promise<T>;
}) {
  const payload = await runtime.requestJSON<{
    providers: Array<{
      name: string;
      source_metadata: { effective_source: { kind: string } };
    }>;
  }>("/api/settings/providers");
  const builtinProvider =
    payload.providers.find(
      provider =>
        provider.name === "codex" &&
        provider.source_metadata.effective_source.kind === "builtin-provider"
    ) ??
    payload.providers.find(
      provider => provider.source_metadata.effective_source.kind === "builtin-provider"
    );

  if (!builtinProvider) {
    throw new Error("Expected at least one builtin provider in the settings providers list.");
  }

  return builtinProvider.name;
}

async function createMCPServerViaUI(
  settingsUI: ReturnType<typeof settingsOperatorSelectors>,
  input: {
    command: string;
    name: string;
    target: "auto" | "config" | "sidecar";
  }
) {
  await settingsUI.mcpServers.create.click();
  await expect(settingsUI.mcpServers.editor).toBeVisible();
  await settingsUI.mcpServers.editorNameInput.fill(input.name);
  await settingsUI.mcpServers.editorCommandInput.fill(input.command);
  await settingsUI.mcpServers.editorTargetInput.selectOption(input.target);
  await settingsUI.mcpServers.editorSave.click();
  await expect(settingsUI.mcpServers.editor).toBeHidden();
}
