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

test("operator can onboard, create a session, submit work, approve a permission request, reload transcript continuity, and resume controls", async ({
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

  await expect(appPage.getByTestId("session-create-dialog")).toBeVisible();
  await expect(appPage.getByTestId("session-create-agent-select")).toHaveValue(
    browserLifecycleAgent
  );

  const createResponsePromise = appPage.waitForResponse(
    response => response.request().method() === "POST" && response.url().endsWith("/api/sessions")
  );
  await appPage.getByTestId("session-create-dialog-submit").click();
  const createResponse = await createResponsePromise;
  expect(createResponse.ok()).toBeTruthy();
  const createPayload = (await createResponse.json()) as { session?: { id?: string } };
  const sessionId = createPayload.session?.id ?? "";
  expect(sessionId).not.toBe("");

  await expect.poll(() => new URL(appPage.url()).pathname).toBe(`/session/${sessionId}`);
  await expect(ui.chatHeader).toBeVisible();
  await expect(ui.composerTextarea).toBeVisible();
  await expect(ui.stopButton).toBeVisible();

  await ui.composerTextarea.fill(browserLifecyclePrompt);
  await ui.composerTextarea.press("Enter");

  await expect(ui.permissionPrompt).toBeVisible();
  await expect(ui.chatView).toContainText(browserLifecyclePrompt);
  await expect(ui.chatView).toContainText("Streaming response started.");

  const approvalResponsePromise = appPage.waitForResponse(response => {
    return (
      response.request().method() === "POST" &&
      response.url().endsWith(`/api/sessions/${encodeURIComponent(sessionId)}/approve`)
    );
  });
  await ui.permissionAllowOnce.click();
  const approvalResponse = await approvalResponsePromise;
  expect(approvalResponse.ok()).toBeTruthy();

  await expect(ui.chatView).toContainText("Streaming response started.");

  const sessionPath = new URL(appPage.url()).pathname;

  await appPage.reload({ waitUntil: "domcontentloaded" });

  await expect.poll(() => new URL(appPage.url()).pathname).toBe(sessionPath);
  await expect(ui.chatHeader).toBeVisible();
  await expect(ui.chatView).toContainText(browserLifecyclePrompt);
  await expect(ui.chatView).toContainText("Streaming response started.");
  await expect(ui.stopButton).toBeVisible();

  await ui.stopButton.click();
  await expect(ui.resumeButton).toBeVisible();

  await ui.resumeButton.click();
  await expect(ui.stopButton).toBeVisible();

  await browserArtifacts.captureScreenshot("session-onboarding-hydrated", appPage);
});
