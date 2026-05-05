import process from "node:process";

import { sessionLifecycleSelectors } from "./fixtures/selectors";
import { expect, test } from "./fixtures/test";

const remoteHTTPAPIBlockedMessage =
  "remote HTTP API access is disabled unless the daemon is bound to a loopback host";

test.use({
  runtimeOptions: {
    host: "0.0.0.0",
    env: {
      ...process.env,
      AGH_TEST_TELEGRAM_TOKEN: "telegram-bot-token",
    },
  },
});

test("operator sees non-loopback HTTP API restrictions with explicit operator messaging", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const sessionUI = sessionLifecycleSelectors(appPage);

  await useGlobalWorkspaceIfPrompted(sessionUI);

  const settingsResponse = await appPage.request.get(runtime.url("/api/settings/general"));
  expect(settingsResponse.status()).toBe(403);
  await expect(settingsResponse.json()).resolves.toMatchObject({
    error: remoteHTTPAPIBlockedMessage,
  });

  await appPage.goto(runtime.url("/settings/general"), { waitUntil: "domcontentloaded" });

  await expect(appPage.getByTestId("settings-page-general-error")).toContainText(
    remoteHTTPAPIBlockedMessage
  );

  await appPage.goto(runtime.url("/settings/hooks-extensions"), { waitUntil: "domcontentloaded" });

  await expect(appPage.getByTestId("settings-page-hooks-extensions-error")).toContainText(
    remoteHTTPAPIBlockedMessage
  );

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
