import { mkdtemp } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import process from "node:process";

import { reloadDaemonServedPage } from "../fixtures/navigation";
import { settingsOperatorSelectors, sessionLifecycleSelectors } from "../fixtures/selectors";
import {
  browserSettingsOperatorFlowScenario,
  cleanupBrowserSettingsFixtures,
  seedBrowserSettingsFixtures,
} from "../fixtures/runtime";
import { expect, test } from "../fixtures/test";
import { ensureGlobalWorkspace, useGlobalWorkspaceIfPrompted } from "../fixtures/workspace";

test.use({
  runtimeOptions: {
    env: {
      ...process.env,
      AGH_TEST_TELEGRAM_TOKEN: "telegram-bot-token",
    },
    extensionsAllowUnverified: true,
  },
});

test("operator can navigate the settings shell and complete a restart-aware general save that survives refresh polling", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const sessionUI = sessionLifecycleSelectors(appPage);
  const settingsUI = settingsOperatorSelectors(appPage);

  await ensureGlobalWorkspace(runtime);
  await useGlobalWorkspaceIfPrompted(sessionUI);
  await appPage.goto(runtime.url("/settings/general"), { waitUntil: "domcontentloaded" });
  await expect(settingsUI.shell.shell).toBeVisible({ timeout: 20_000 });
  await expect(settingsUI.shell.sectionNav).toBeVisible({ timeout: 20_000 });

  await expect
    .poll(async () => normalizeTexts(await settingsUI.shell.sectionItems.allTextContents()))
    .toEqual([
      "General",
      "Providers",
      "Memory",
      "Skills",
      "Automation",
      "Network",
      "Observability",
      "Hooks",
      "Extensions",
    ]);

  await expect.poll(() => new URL(appPage.url()).pathname).toBe("/settings/general");
  await expect(settingsUI.shell.sectionLink("general")).toHaveAttribute("aria-current", "page");
  await expect(settingsUI.general.page).toBeVisible();

  await settingsUI.shell.sectionLink("network").click();
  await expect.poll(() => new URL(appPage.url()).pathname).toBe("/settings/network");
  await expect(settingsUI.shell.sectionLink("network")).toHaveAttribute("aria-current", "page");

  await settingsUI.shell.sectionLink("hooks").click();
  await expect.poll(() => new URL(appPage.url()).pathname).toBe("/settings/hooks");
  await expect(settingsUI.shell.sectionActive("hooks")).toBeVisible();

  await settingsUI.shell.sectionLink("extensions").click();
  await expect.poll(() => new URL(appPage.url()).pathname).toBe("/settings/extensions");
  await expect(settingsUI.shell.sectionActive("extensions")).toBeVisible();

  await appPage.goBack({ waitUntil: "domcontentloaded" });
  await expect.poll(() => new URL(appPage.url()).pathname).toBe("/settings/hooks");
  await appPage.goForward({ waitUntil: "domcontentloaded" });
  await expect.poll(() => new URL(appPage.url()).pathname).toBe("/settings/extensions");

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

  await reloadDaemonServedPage(appPage, runtime, "/settings/general", {
    readyTestId: "settings-page-general",
  });
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
    await ensureGlobalWorkspace(runtime);
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
    await expect.poll(() => new URL(appPage.url()).pathname).toBe("/marketplace/skills");
    await expect.poll(() => new URL(appPage.url()).search).toBe("?tab=installed");
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

  await ensureGlobalWorkspace(runtime);
  await useGlobalWorkspaceIfPrompted(sessionUI);
  await appPage.goto(runtime.url("/settings/providers"), { waitUntil: "domcontentloaded" });

  await expect(settingsUI.providers.page).toBeVisible();
  await expect(settingsUI.providers.list).toBeVisible();
  await expect(settingsUI.providers.card(builtinProviderName)).toBeVisible();

  await appPage.getByTestId(`settings-page-providers-card-${builtinProviderName}-open`).click();
  await appPage.getByTestId("provider-inspector-edit").click();
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

  await appPage.getByTestId(`settings-page-providers-card-${builtinProviderName}-open`).click();
  await appPage.getByTestId("provider-inspector-delete").click();
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

  await ensureGlobalWorkspace(runtime);
  await useGlobalWorkspaceIfPrompted(sessionUI);
  await appPage.goto(runtime.url("/marketplace/mcps?tab=installed"), {
    waitUntil: "domcontentloaded",
  });

  await expect(settingsUI.mcpServers.page).toBeVisible();
  await expect(settingsUI.mcpServers.scopeWorkspace).toBeVisible();
  await expect(settingsUI.mcpServers.scopeGlobal).toBeVisible();

  await appPage.getByTestId("workspace-switcher").click();
  await appPage.getByTestId(`workspace-command-item-${workspace.id}`).click();
  await expect(appPage.getByTestId("workspace-switcher")).toHaveAttribute("aria-expanded", "false");
  await settingsUI.mcpServers.scopeWorkspace.click();
  await expect(settingsUI.mcpServers.scopeWorkspace).toHaveAttribute("aria-pressed", "true");

  await settingsUI.mcpServers.scopeGlobal.click();
  await expect(settingsUI.mcpServers.scopeGlobal).toHaveAttribute("aria-pressed", "true");

  await createMCPServerViaUI(settingsUI, {
    name: browserSettingsOperatorFlowScenario.mcpServers.global.name,
    command: browserSettingsOperatorFlowScenario.mcpServers.global.command,
    target: browserSettingsOperatorFlowScenario.mcpServers.global.target,
  });

  await expect(settingsUI.mcpServers.actionResult).toContainText(
    `Saved "${browserSettingsOperatorFlowScenario.mcpServers.global.name}" · global-mcp-sidecar · applied now`
  );
  await expect(
    settingsUI.mcpServers.row(browserSettingsOperatorFlowScenario.mcpServers.global.name)
  ).toBeVisible();

  await settingsUI.mcpServers.scopeWorkspace.click();
  await expect(settingsUI.mcpServers.scopeWorkspace).toHaveAttribute("aria-pressed", "true");

  await createMCPServerViaUI(settingsUI, {
    name: browserSettingsOperatorFlowScenario.mcpServers.workspace.name,
    command: browserSettingsOperatorFlowScenario.mcpServers.workspace.command,
    target: browserSettingsOperatorFlowScenario.mcpServers.workspace.target,
  });

  await expect(settingsUI.mcpServers.actionResult).toContainText(
    `Saved "${browserSettingsOperatorFlowScenario.mcpServers.workspace.name}" · workspace-config · applied now`
  );
  await expect(
    settingsUI.mcpServers.row(browserSettingsOperatorFlowScenario.mcpServers.workspace.name)
  ).toBeVisible();

  await expect(
    settingsUI.mcpServers.row(browserSettingsOperatorFlowScenario.mcpServers.global.name)
  ).toBeVisible();
  await expect(
    settingsUI.mcpServers.row(browserSettingsOperatorFlowScenario.mcpServers.workspace.name)
  ).toBeVisible();

  await settingsUI.mcpServers
    .editRow(browserSettingsOperatorFlowScenario.mcpServers.workspace.name)
    .click();
  await appPage.getByRole("menuitem", { name: "Remove…" }).click();
  await appPage
    .getByLabel("Type to confirm")
    .fill(browserSettingsOperatorFlowScenario.mcpServers.workspace.name);
  await appPage
    .getByTestId(
      `marketplace-confirm-${browserSettingsOperatorFlowScenario.mcpServers.workspace.name}`
    )
    .click();

  await expect(settingsUI.mcpServers.actionResult).toContainText(
    `${browserSettingsOperatorFlowScenario.mcpServers.workspace.name} removed`
  );
  await expect(
    settingsUI.mcpServers.row(browserSettingsOperatorFlowScenario.mcpServers.workspace.name)
  ).not.toBeVisible();
  await browserArtifacts.captureScreenshot("tc-int-011-mcp-workspace-scope", appPage);
});

test("operator can manage restart-aware hooks and extension policy on split settings routes", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const sessionUI = sessionLifecycleSelectors(appPage);
  const settingsUI = settingsOperatorSelectors(appPage);
  const seeded = await seedBrowserSettingsFixtures(runtime, {
    hooks: [
      {
        name: browserSettingsOperatorFlowScenario.hooks.hookName,
        declaration: {
          name: browserSettingsOperatorFlowScenario.hooks.hookName,
          event: "turn.end",
          mode: "sync",
          command: "/bin/echo",
          args: ["settings-hook"],
          matcher: {},
          required: true,
          enabled: true,
        },
      },
    ],
  });

  try {
    await ensureGlobalWorkspace(runtime);
    await useGlobalWorkspaceIfPrompted(sessionUI);
    await appPage.goto(runtime.url("/settings/hooks"), {
      waitUntil: "domcontentloaded",
    });

    await expect(settingsUI.hooks.page).toBeVisible();
    await expect(
      settingsUI.hooks.hookToggle(browserSettingsOperatorFlowScenario.hooks.hookName)
    ).toBeVisible();

    await settingsUI.hooks.hookToggle(browserSettingsOperatorFlowScenario.hooks.hookName).click();
    await expect(
      settingsUI.hooks.hookToggle(browserSettingsOperatorFlowScenario.hooks.hookName)
    ).not.toBeChecked();
    await expect(settingsUI.hooks.restartBanner).toBeVisible();

    // Prove failure-fatality (`required`) and dispatch enablement (`enabled`) are
    // independent fields: disabling the hook flips `enabled` to false while
    // `required` stays true in the real daemon response.
    const hooksResponse = await runtime.requestJSON<{
      hooks: Array<{ name: string; declaration: { enabled?: boolean; required?: boolean } }>;
    }>("/api/settings/hooks");
    const toggledHook = hooksResponse.hooks.find(
      hook => hook.name === browserSettingsOperatorFlowScenario.hooks.hookName
    );
    expect(toggledHook?.declaration.enabled).toBe(false);
    expect(toggledHook?.declaration.required).toBe(true);

    await appPage.goto(runtime.url("/settings/extensions"), {
      waitUntil: "domcontentloaded",
    });
    await expect(settingsUI.extensions.page).toBeVisible();
    await expect(settingsUI.hooks.hooksList).toHaveCount(0);

    await settingsUI.extensions.policyRegistryInput.fill("github");
    await settingsUI.extensions.policyBaseURLInput.fill(
      "https://extensions.example/browser-updated"
    );
    await expect(settingsUI.extensions.policySave).toBeEnabled();
    await settingsUI.extensions.policySave.click();

    await expect(settingsUI.extensions.restartBanner).toBeVisible();
    await browserArtifacts.captureScreenshot("tc-func-012-extensions-policy", appPage);
  } finally {
    await cleanupBrowserSettingsFixtures(runtime, seeded);
  }
});

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
