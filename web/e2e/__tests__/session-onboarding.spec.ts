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

const browserLifecycleAgent = "browser-lifecycle-agent";
const browserLifecyclePrompt = "run browser lifecycle flow";

function browserLifecycleSessionPath(agentName: string, sessionId: string): string {
  return `/agents/${encodeURIComponent(agentName)}/sessions/${encodeURIComponent(sessionId)}`;
}

function sessionAPIPath(workspaceID: string, sessionID: string, suffix = ""): string {
  return `/api/workspaces/${encodeURIComponent(workspaceID)}/sessions/${encodeURIComponent(
    sessionID
  )}${suffix}`;
}

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
  await expect(ui.agentRow(browserLifecycleAgent)).toBeVisible();

  await ui.agentRow(browserLifecycleAgent).click();
  await expect.poll(() => new URL(appPage.url()).pathname).toBe(`/agents/${browserLifecycleAgent}`);
  await expect(ui.agentPageNewSession).toBeVisible();
  await ui.agentPageNewSession.click();

  await expect(appPage.getByTestId("session-create-dialog")).toBeVisible();
  await expect(appPage.getByTestId("session-create-agent-select")).toContainText(
    browserLifecycleAgent
  );
  await expect(appPage.getByTestId("session-create-dialog")).toHaveAttribute(
    "data-frame",
    "unframed"
  );
  await expect(
    appPage.getByTestId("session-create-dialog").locator('[data-slot="dialog-header"]')
  ).toHaveAttribute("data-variant", "ruled");
  await expect(
    appPage.getByTestId("session-create-dialog").locator('[data-slot="dialog-footer"]')
  ).toHaveAttribute("data-variant", "ruled");

  await appPage.getByTestId("session-create-model-select").click();
  await expect(appPage.getByTestId("model-command-input")).toBeVisible();
  await appPage.getByTestId("model-command-input").fill("browser-e2e-model");
  await appPage.getByTestId("model-command-input").press("Enter");
  await expect(appPage.getByTestId("session-create-model-select")).toContainText(
    "browser-e2e-model"
  );

  const createResponsePromise = appPage.waitForResponse(
    response => response.request().method() === "POST" && response.url().endsWith("/api/sessions")
  );
  await appPage.getByTestId("session-create-dialog-submit").click();
  const createResponse = await createResponsePromise;
  expect(createResponse.ok()).toBeTruthy();
  const createPayload = (await createResponse.json()) as {
    session?: { id?: string; workspace_id?: string };
  };
  const sessionId = createPayload.session?.id ?? "";
  const workspaceId = createPayload.session?.workspace_id ?? "";
  expect(sessionId).not.toBe("");
  expect(workspaceId).not.toBe("");

  await expect
    .poll(() => new URL(appPage.url()).pathname)
    .toBe(browserLifecycleSessionPath(browserLifecycleAgent, sessionId));
  await expect(ui.chatHeader).toBeVisible();
  await expect(ui.composerTextarea).toBeVisible();
  await expect(ui.stopButton).toBeVisible();

  await ui.composerTextarea.fill(browserLifecyclePrompt);
  await ui.composerTextarea.press("Enter");

  await expect(ui.permissionPrompt).toBeVisible();
  await expect(appPage.getByTestId("permission-reject-once")).toBeVisible();
  await expect(ui.chatView).toContainText("Streaming response started.");

  const approvalResponsePromise = appPage.waitForResponse(response => {
    return (
      response.request().method() === "POST" &&
      response.url().endsWith(sessionAPIPath(workspaceId, sessionId, "/approve"))
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
  await expect(ui.resumeButton).toBeVisible();

  await ui.resumeButton.click();
  await expect(ui.stopButton).toBeVisible();

  await ui.stopButton.click();
  await expect(ui.resumeButton).not.toBeVisible();

  await browserArtifacts.captureScreenshot("session-onboarding-hydrated", appPage);
});
