import process from "node:process";

import { settingsOperatorSelectors, sessionLifecycleSelectors } from "./fixtures/selectors";
import { browserSettingsOperatorFlowScenario } from "./fixtures/runtime";
import { expect, test } from "./fixtures/test";

test.use({
  runtimeOptions: {
    host: "0.0.0.0",
    env: {
      ...process.env,
      AGH_TEST_TELEGRAM_TOKEN: "telegram-bot-token",
    },
  },
});

test("operator sees non-loopback HTTP mutation restrictions with explicit operator messaging", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const sessionUI = sessionLifecycleSelectors(appPage);
  const settingsUI = settingsOperatorSelectors(appPage);

  await useGlobalWorkspaceIfPrompted(sessionUI);
  await appPage.goto(runtime.url("/settings/general"), { waitUntil: "domcontentloaded" });

  await expect(settingsUI.general.page).toBeVisible();

  const nextTimeoutValue = await nextSessionTimeoutValue(settingsUI.general.sessionTimeoutInput);

  await settingsUI.general.sessionTimeoutInput.fill(nextTimeoutValue);
  await expect(settingsUI.general.saveButton).toBeEnabled();
  await settingsUI.general.saveButton.click();

  await expect(appPage.getByTestId("settings-page-general-save-error")).toContainText(
    "loopback host"
  );
  await expect(settingsUI.general.restartBanner).not.toBeVisible();

  await appPage.goto(runtime.url("/settings/hooks-extensions"), { waitUntil: "domcontentloaded" });

  await expect(settingsUI.hooksExtensions.page).toBeVisible();
  await expect(settingsUI.hooksExtensions.transportParity).toContainText("return 403 on HTTP");

  await settingsUI.hooksExtensions.policyRegistryInput.fill("github");
  await settingsUI.hooksExtensions.policyBaseURLInput.fill(
    "https://extensions.example/non-loopback"
  );
  await expect(settingsUI.hooksExtensions.policySave).toBeEnabled();
  await settingsUI.hooksExtensions.policySave.click();

  await expect(settingsUI.hooksExtensions.policyControls).toContainText("loopback host");

  await browserArtifacts.captureScreenshot("tc-int-013-non-loopback-http-restrictions", appPage);
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

async function nextSessionTimeoutValue(
  input: ReturnType<typeof settingsOperatorSelectors>["general"]["sessionTimeoutInput"]
): Promise<string> {
  const currentValue = Number.parseInt((await input.inputValue()) || "0", 10);
  const primary = browserSettingsOperatorFlowScenario.general.primarySessionTimeoutSeconds;
  const fallback = browserSettingsOperatorFlowScenario.general.fallbackSessionTimeoutSeconds;
  return String(currentValue === primary ? fallback : primary);
}
