import { fileURLToPath } from "node:url";
import path from "node:path";

import { sessionLifecycleSelectors } from "./fixtures/selectors";
import { expect, test } from "./fixtures/test";

const browserLifecycleFixture = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
  "..",
  "internal",
  "testutil",
  "acpmock",
  "testdata",
  "browser_session_lifecycle_fixture.json"
);

const browserLifecycleAgent = "browser-lifecycle-agent";
const browserLifecyclePrompt = "run browser lifecycle flow";

test.use({
  runtimeOptions: {
    seed: {
      mockAgents: [
        {
          fixturePath: browserLifecycleFixture,
          fixtureAgent: browserLifecycleAgent,
        },
      ],
    },
  },
});

test("operator can onboard, create a session, approve work, stop/resume, and reload with transcript continuity", async ({
  appPage,
  browserArtifacts,
}) => {
  const ui = sessionLifecycleSelectors(appPage);

  await expect(ui.workspaceOnboarding).toBeVisible();
  await ui.workspaceUseGlobal.click();

  await expect(ui.workspaceOnboarding).toBeHidden();
  await expect(ui.appSidebar).toBeVisible();
  await expect(ui.newSessionButton(browserLifecycleAgent)).toBeVisible();

  await ui.newSessionButton(browserLifecycleAgent).click();

  await expect(ui.chatHeader).toBeVisible();
  await expect(ui.composerTextarea).toBeVisible();
  await expect(ui.stopButton).toBeVisible();

  await ui.composerTextarea.fill(browserLifecyclePrompt);
  await ui.composerSendButton.click();

  await expect(ui.permissionPrompt).toBeVisible();
  await expect(ui.chatView).toContainText(browserLifecyclePrompt);
  await expect(ui.chatView).toContainText("Streaming response started.");
  await expect(ui.composerTextarea).toBeDisabled();

  await ui.permissionAllowOnce.click();

  await expect(ui.permissionPrompt).toBeHidden();
  await expect(ui.chatView).toContainText("Streaming response started.");
  await expect(ui.chatView).toContainText("Approval granted.");
  await expect(ui.chatView).toContainText("Session continued after approval.");

  const sessionPath = new URL(appPage.url()).pathname;

  await ui.stopButton.click();
  await expect(ui.resumeButton).toBeVisible();

  await ui.resumeButton.click();
  await expect(ui.stopButton).toBeVisible();

  await appPage.reload({ waitUntil: "domcontentloaded" });

  await expect.poll(() => new URL(appPage.url()).pathname).toBe(sessionPath);
  await expect(ui.chatHeader).toBeVisible();
  await expect(ui.chatView).toContainText(browserLifecyclePrompt);
  await expect(ui.chatView).toContainText("Session continued after approval.");
  await expect(ui.stopButton).toBeVisible();

  await browserArtifacts.captureScreenshot("session-onboarding-hydrated", appPage);
});
