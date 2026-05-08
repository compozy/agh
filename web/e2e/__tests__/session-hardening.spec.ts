import { execFile } from "node:child_process";
import { readFile } from "node:fs/promises";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";
import { promisify } from "node:util";

import { sessionLifecycleSelectors } from "../fixtures/selectors";
import type { BrowserRuntime, WorkspacePayload } from "../fixtures/runtime";
import { expect, test } from "../fixtures/test";
import { useGlobalWorkspaceIfPrompted } from "../fixtures/workspace";

const execFileAsync = promisify(execFile);
const fixtureRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
  "..",
  "..",
  "internal",
  "testutil",
  "acpmock",
  "testdata"
);
const browserHardeningFixture = path.join(fixtureRoot, "browser_session_hardening_fixture.json");
const driverFaultFixture = path.join(fixtureRoot, "driver_fault_fixture.json");
const permissionAgent = "permission-hardening-agent";
const faultAgent = "faulty";
const sensitivePattern =
  /agh_claim_|claim_token["':\s]|mcp[_-]?auth|telegram-bot-token|pkce|oauth|webhook_secret|provider[_-]?credential/i;

interface SessionPayload {
  id: string;
  agent_name: string;
  provider: string;
  state: string;
  workspace_id: string;
}

interface SessionEnvelope {
  session: SessionPayload;
}

interface SessionEventEnvelope {
  events: unknown[];
}

interface SessionHistoryEnvelope {
  history: unknown[];
}

interface SessionRepairEnvelope {
  repair: {
    session_id: string;
    issues?: unknown[];
    actions?: unknown[];
  };
}

test.use({
  runtimeOptions: {
    seed: {
      mockAgents: [
        {
          fixturePath: browserHardeningFixture,
          fixtureAgent: permissionAgent,
        },
        {
          fixturePath: driverFaultFixture,
          fixtureAgent: faultAgent,
        },
      ],
    },
  },
});

test("operator rejects a permission request, records tool output, and keeps session artifacts private", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const workspace = await prepareSessionRuntime(runtime, appPage);
  const session = await createSession(runtime, permissionAgent, workspace.id);

  await appPage.goto(runtime.url(sessionPath(permissionAgent, session.id)), {
    waitUntil: "domcontentloaded",
  });

  const ui = sessionLifecycleSelectors(appPage);
  await expect(ui.chatHeader).toBeVisible();
  await expect(appPage.getByTestId("session-workspace-badge")).toHaveText(workspace.name);
  await expect(ui.composerTextarea).toBeEnabled();

  await ui.composerTextarea.fill("exercise permission hardening");
  await ui.composerTextarea.press("Enter");

  await expect(ui.chatView).toContainText("Permission hardening started.");
  await expect(ui.permissionPrompt).toBeVisible();
  await expect(appPage.getByTestId("permission-tool-input")).toContainText("hardening.txt");

  const approvalResponsePromise = appPage.waitForResponse(
    response =>
      response.request().method() === "POST" &&
      response.url().endsWith(`/api/sessions/${encodeURIComponent(session.id)}/approve`)
  );
  await appPage.getByTestId("permission-reject-always").click();
  expect((await approvalResponsePromise).ok()).toBe(true);

  await expect(ui.permissionPrompt).toBeHidden();
  await expect(appPage.getByTestId("composer-clear-button")).toBeEnabled();

  const snapshot = await captureSessionSnapshot(runtime, session.id);
  expect(JSON.stringify(snapshot.events)).toContain("tool-hardening-read-1");
  expect(JSON.stringify(snapshot.events)).toContain("hardening read complete");
  expect(JSON.stringify(snapshot.events)).toContain("reject-always");
  expect(JSON.stringify(snapshot.history)).toContain("exercise permission hardening");
  await runtime.artifactCollector.captureJSON("browser_api_snapshots", snapshot);
  await browserArtifacts.captureScreenshot("session-permission-rejected", appPage);
  const manifest = await browserArtifacts.persist(appPage);
  expect(manifest.artifacts).toEqual(
    expect.arrayContaining([
      expect.objectContaining({ kind: "browser_api_snapshots" }),
      expect.objectContaining({ kind: "browser_route_state" }),
      expect.objectContaining({ kind: "browser_screenshots" }),
      expect.objectContaining({ kind: "browser_trace" }),
    ])
  );

  const routeState = JSON.parse(
    await readFile(runtime.artifactCollector.artifactPath("browser_route_state"), "utf8")
  ) as Record<string, unknown>;
  expect(routeState).toMatchObject({
    chat_view_visible: true,
    composer_clear_button_enabled: true,
    delete_button_visible: true,
    message_count: expect.any(Number),
    permission_prompt_visible: false,
  });
  await assertNoSensitiveLeak(appPage, runtime, snapshot);
});

test("operator cancels a running prompt, clears the transcript, and deletes the session across surfaces", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const workspace = await prepareSessionRuntime(runtime, appPage);
  const session = await createSession(runtime, faultAgent, workspace.id);

  await appPage.goto(runtime.url(sessionPath(faultAgent, session.id)), {
    waitUntil: "domcontentloaded",
  });

  const ui = sessionLifecycleSelectors(appPage);
  await expect(ui.chatHeader).toBeVisible();
  await ui.composerTextarea.fill("block until canceled");
  await ui.composerTextarea.press("Enter");
  await expect(ui.chatView).toContainText("block until canceled");

  const cancelResponsePromise = appPage.waitForResponse(
    response =>
      response.request().method() === "POST" &&
      response.url().endsWith(`/api/sessions/${encodeURIComponent(session.id)}/prompt/cancel`)
  );
  await expect(ui.stopButton).toBeVisible();
  await ui.stopButton.click();
  expect((await cancelResponsePromise).ok()).toBe(true);

  await expect(appPage.getByTestId("composer-clear-button")).toBeEnabled({ timeout: 60_000 });
  const beforeClear = await captureSessionSnapshot(runtime, session.id);
  expect(JSON.stringify(beforeClear.history)).toContain("block until canceled");

  await appPage.getByTestId("composer-clear-button").click();
  await expect(appPage.getByTestId("composer-clear-dialog")).toBeVisible();
  const clearResponsePromise = appPage.waitForResponse(
    response =>
      response.request().method() === "POST" &&
      response.url().endsWith(`/api/sessions/${encodeURIComponent(session.id)}/clear`)
  );
  await appPage.getByTestId("composer-clear-confirm").click();
  expect((await clearResponsePromise).ok()).toBe(true);
  await expect(ui.chatView).not.toContainText("block until canceled");

  const afterClear = await captureSessionSnapshot(runtime, session.id);
  expect(JSON.stringify(afterClear.history)).not.toContain("block until canceled");

  const deletableSession = await createSession(runtime, faultAgent, workspace.id);
  await appPage.goto(runtime.url(sessionPath(faultAgent, deletableSession.id)), {
    waitUntil: "domcontentloaded",
  });
  await expect(ui.chatHeader).toBeVisible();
  await appPage.getByTestId("delete-button").click();
  await expect(appPage.getByTestId("delete-dialog")).toBeVisible();
  const deleteResponsePromise = appPage.waitForResponse(
    response =>
      response.request().method() === "DELETE" &&
      response.url().endsWith(`/api/sessions/${encodeURIComponent(deletableSession.id)}`)
  );
  await appPage.getByTestId("delete-dialog-confirm").click();
  expect((await deleteResponsePromise).ok()).toBe(true);
  await expect.poll(() => new URL(appPage.url()).pathname).toBe(`/agents/${faultAgent}`);

  await expect(
    runtime.requestJSON<SessionEnvelope>(`/api/sessions/${encodeURIComponent(deletableSession.id)}`)
  ).rejects.toThrow("404");
  await expect(
    runtime.requestOperatorJSON?.<SessionEnvelope>(
      `/api/sessions/${encodeURIComponent(deletableSession.id)}`
    )
  ).rejects.toThrow("404");
  const cliSessions = await listSessionsViaCLI(runtime);
  expect(cliSessions.some(record => record.id === deletableSession.id)).toBe(false);

  await runtime.artifactCollector.captureJSON("browser_api_snapshots", {
    after_clear: afterClear,
    before_clear: beforeClear,
    cli_sessions_after_delete: cliSessions,
    deleted_session_id: deletableSession.id,
  });
  await browserArtifacts.captureScreenshot("session-cancel-clear-delete", appPage);
});

test("operator repairs an interrupted session through HTTP, UDS, and CLI without losing transcript evidence", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const workspace = await prepareSessionRuntime(runtime, appPage);
  const session = await createSession(runtime, faultAgent, workspace.id);

  await appPage.goto(runtime.url(sessionPath(faultAgent, session.id)), {
    waitUntil: "domcontentloaded",
  });

  const ui = sessionLifecycleSelectors(appPage);
  await expect(ui.chatHeader).toBeVisible();
  await ui.composerTextarea.fill("trigger crash mid-stream");
  await ui.composerTextarea.press("Enter");
  await expect(ui.chatView).toContainText("partial before crash", { timeout: 15_000 });
  await expect(ui.resumeButton).toBeVisible({ timeout: 20_000 });

  const beforeRepair = await captureSessionSnapshot(runtime, session.id);
  expect(JSON.stringify(beforeRepair.history)).toContain("trigger crash mid-stream");
  expect(JSON.stringify(beforeRepair.history)).toContain("partial before crash");

  const httpRepair = await runtime.requestJSON<SessionRepairEnvelope>(
    `/api/sessions/${encodeURIComponent(session.id)}/repair?dry_run=true&force=true`,
    { method: "POST" }
  );
  expect(httpRepair.repair.session_id).toBe(session.id);

  if (!runtime.requestOperatorJSON) {
    throw new Error("session repair E2E requires launch-mode UDS access.");
  }
  const udsRepair = await runtime.requestOperatorJSON<SessionRepairEnvelope>(
    `/api/sessions/${encodeURIComponent(session.id)}/repair?dry_run=true&force=true`,
    { method: "POST" }
  );
  expect(udsRepair.repair.session_id).toBe(session.id);

  const cliRepair = await repairSessionViaCLI(runtime, session.id);
  expect(JSON.stringify(cliRepair)).toContain(session.id);

  const afterRepair = await captureSessionSnapshot(runtime, session.id);
  expect(JSON.stringify(afterRepair.history)).toContain("trigger crash mid-stream");
  expect(JSON.stringify(afterRepair.history)).toContain("partial before crash");
  expect(afterRepair.session.session.state).toBe("stopped");

  await appPage.reload({ waitUntil: "domcontentloaded" });
  await expect(ui.chatView).toContainText("partial before crash");
  await expect(ui.resumeButton).toBeVisible();

  await runtime.artifactCollector.captureJSON("browser_api_snapshots", {
    after_repair: afterRepair,
    before_repair: beforeRepair,
    cli_repair: cliRepair,
    http_repair: httpRepair,
    uds_repair: udsRepair,
  });
  await browserArtifacts.captureScreenshot("session-repair-parity", appPage);
  await browserArtifacts.persist(appPage);
  await assertNoSensitiveLeak(appPage, runtime, { afterRepair, beforeRepair, cliRepair });
});

async function prepareSessionRuntime(
  runtime: BrowserRuntime,
  page: import("@playwright/test").Page
): Promise<WorkspacePayload> {
  if (!runtime.paths?.homeDir) {
    throw new Error("session hardening E2E requires launch-mode runtime paths.");
  }
  const workspace = await runtime.resolveWorkspace(runtime.paths.homeDir);
  const ui = sessionLifecycleSelectors(page);
  await page.goto(runtime.url("/"), { waitUntil: "domcontentloaded" });
  await useGlobalWorkspaceIfPrompted(ui);
  return workspace;
}

async function createSession(
  runtime: BrowserRuntime,
  agentName: string,
  workspaceID: string
): Promise<SessionPayload> {
  const payload = await runtime.requestJSON<SessionEnvelope>("/api/sessions", {
    method: "POST",
    body: JSON.stringify({
      agent_name: agentName,
      workspace: workspaceID,
    }),
  });
  expect(payload.session.id).not.toBe("");
  expect(payload.session.agent_name).toBe(agentName);
  expect(payload.session.workspace_id).toBe(workspaceID);
  return payload.session;
}

async function captureSessionSnapshot(
  runtime: BrowserRuntime,
  sessionID: string
): Promise<{
  events: SessionEventEnvelope;
  history: SessionHistoryEnvelope;
  session: SessionEnvelope;
  transcript: unknown;
  udsSession?: SessionEnvelope;
}> {
  const sessionPathname = `/api/sessions/${encodeURIComponent(sessionID)}`;
  const snapshot = {
    events: await runtime.requestJSON<SessionEventEnvelope>(`${sessionPathname}/events`),
    history: await runtime.requestJSON<SessionHistoryEnvelope>(`${sessionPathname}/history`),
    session: await runtime.requestJSON<SessionEnvelope>(sessionPathname),
    transcript: await runtime.requestJSON<unknown>(`${sessionPathname}/transcript`),
    udsSession: runtime.requestOperatorJSON
      ? await runtime.requestOperatorJSON<SessionEnvelope>(sessionPathname)
      : undefined,
  };
  if (snapshot.udsSession) {
    expect(snapshot.udsSession.session.id).toBe(snapshot.session.session.id);
    expect(snapshot.udsSession.session.state).toBe(snapshot.session.session.state);
  }
  return snapshot;
}

async function listSessionsViaCLI(runtime: BrowserRuntime): Promise<SessionPayload[]> {
  if (!runtime.paths) {
    throw new Error("session hardening CLI checks require launch-mode runtime paths.");
  }
  const { stdout } = await execFileAsync(
    runtime.paths.cliShim,
    ["session", "list", "--all", "-o", "json"],
    { env: cliEnv(runtime.paths) }
  );
  return JSON.parse(stdout) as SessionPayload[];
}

async function repairSessionViaCLI(runtime: BrowserRuntime, sessionID: string): Promise<unknown> {
  if (!runtime.paths) {
    throw new Error("session hardening CLI checks require launch-mode runtime paths.");
  }
  const { stdout } = await execFileAsync(
    runtime.paths.cliShim,
    ["session", "repair", sessionID, "--dry-run", "--force", "-o", "json"],
    { env: cliEnv(runtime.paths) }
  );
  return JSON.parse(stdout) as unknown;
}

function cliEnv(paths: { cliShim: string; homeDir: string }): NodeJS.ProcessEnv {
  return {
    ...process.env,
    AGH_HOME: paths.homeDir,
    HOME: paths.homeDir,
    PATH: `${path.dirname(paths.cliShim)}:${process.env.PATH ?? ""}`,
  };
}

function sessionPath(agentName: string, sessionID: string): string {
  return `/agents/${agentName}/sessions/${sessionID}`;
}

async function assertNoSensitiveLeak(
  page: import("@playwright/test").Page,
  runtime: BrowserRuntime,
  snapshot: unknown
): Promise<void> {
  await expect(page.locator("body")).not.toContainText(sensitivePattern);
  const artifactPayloads = [
    JSON.stringify(snapshot),
    await readFile(runtime.artifactCollector.artifactPath("browser_console"), "utf8"),
    await readFile(runtime.artifactCollector.artifactPath("browser_network"), "utf8"),
    await readFile(runtime.artifactCollector.artifactPath("browser_route_state"), "utf8"),
    await readFile(runtime.artifactCollector.artifactPath("browser_api_snapshots"), "utf8"),
  ];
  for (const payload of artifactPayloads) {
    expect(payload).not.toMatch(sensitivePattern);
  }
  if (runtime.paths?.daemonLog) {
    expect(await readFile(runtime.paths.daemonLog, "utf8")).not.toMatch(sensitivePattern);
  }
}
