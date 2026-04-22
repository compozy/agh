import { execFile } from "node:child_process";
import { mkdir, mkdtemp, readFile, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";
import { promisify } from "node:util";

import { sessionLifecycleSelectors } from "./fixtures/selectors";
import { expect, test } from "./fixtures/test";

const execFileAsync = promisify(execFile);

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
const overrideProvider = "qa-browser-override";
const driftedDefaultProvider = "gemini";

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

interface WorkspaceDetailPayload {
  id: string;
  providers: Array<{ name: string }>;
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

test("operator can create a provider-override session and gets an inline resume failure when that provider disappears", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  if (!runtime.paths) {
    throw new Error("provider override browser test requires launch-mode runtime paths");
  }

  const ui = sessionLifecycleSelectors(appPage);
  const workspaceRoot = await mkdtemp(path.join(os.tmpdir(), "agh-provider-override-workspace-"));
  const overrideCommand = await readAgentCommand(runtime.paths.homeDir, browserLifecycleAgent);

  await writeWorkspaceConfig({
    rootDir: workspaceRoot,
    defaultProvider: "claude",
    overrideCommand,
    includeOverride: true,
  });

  const workspace = await runtime.resolveWorkspace(workspaceRoot);
  const workspaceDetail = await runtime.requestJSON<WorkspaceDetailPayload>(
    `/api/workspaces/${encodeURIComponent(workspace.id)}`
  );

  await appPage.goto(runtime.url("/"), { waitUntil: "domcontentloaded" });
  await expect(ui.appSidebar).toBeVisible();
  await expect(ui.newSessionButton(browserLifecycleAgent)).toBeVisible();

  await ui.newSessionButton(browserLifecycleAgent).click();

  await expect(appPage.getByTestId("session-create-dialog")).toBeVisible();
  await expect(appPage.getByTestId("session-create-agent-select")).toHaveValue(
    browserLifecycleAgent
  );
  await expect(appPage.getByTestId("session-create-provider-select")).toHaveValue("claude");

  const dialogOptions = await appPage
    .getByTestId("session-create-provider-select")
    .locator("option")
    .evaluateAll(options => options.map(option => (option as HTMLOptionElement).value));
  expect(dialogOptions).toEqual(workspaceDetail.providers.map(provider => provider.name));

  await browserArtifacts.captureScreenshot("session-provider-dialog-desktop", appPage);
  await appPage.setViewportSize({ width: 375, height: 812 });
  await expect(appPage.getByTestId("session-create-provider-select")).toBeVisible();
  await browserArtifacts.captureScreenshot("session-provider-dialog-mobile", appPage);
  await appPage.setViewportSize({ width: 1280, height: 800 });

  await appPage.getByTestId("session-create-provider-select").selectOption(overrideProvider);

  const createRequestPromise = appPage.waitForRequest(
    request => request.method() === "POST" && request.url().endsWith("/api/sessions")
  );
  const createResponsePromise = appPage.waitForResponse(
    response => response.request().method() === "POST" && response.url().endsWith("/api/sessions")
  );

  await appPage.getByTestId("session-create-dialog-submit").click();

  const createRequest = await createRequestPromise;
  const createResponse = await createResponsePromise;
  const createRequestBody = createRequest.postDataJSON() as {
    agent_name?: string;
    provider?: string;
    workspace?: string;
  };
  expect(createRequestBody).toMatchObject({
    agent_name: browserLifecycleAgent,
    provider: overrideProvider,
    workspace: workspace.id,
  });
  expect(createResponse.ok()).toBeTruthy();

  const createdSession = (await createResponse.json()) as SessionEnvelope;
  expect(createdSession.session.provider).toBe(overrideProvider);

  await expect
    .poll(() => new URL(appPage.url()).pathname)
    .toContain(`/session/${createdSession.session.id}`);
  await expect(ui.chatHeader).toBeVisible();
  await expect(appPage.getByTestId("session-provider-badge")).toHaveText(overrideProvider);
  await browserArtifacts.captureScreenshot("session-provider-created", appPage);

  await assertSessionParity(runtime, createdSession.session.id, overrideProvider);

  await ui.stopButton.click();
  await expect(ui.resumeButton).toBeVisible();

  await writeWorkspaceConfig({
    rootDir: workspaceRoot,
    defaultProvider: driftedDefaultProvider,
    overrideCommand,
    includeOverride: true,
  });

  await ui.resumeButton.click();
  await expect(ui.stopButton).toBeVisible();
  await expect(appPage.getByTestId("session-provider-badge")).toHaveText(overrideProvider);
  await assertSessionParity(runtime, createdSession.session.id, overrideProvider);

  await ui.stopButton.click();
  await expect(ui.resumeButton).toBeVisible();

  await writeWorkspaceConfig({
    rootDir: workspaceRoot,
    defaultProvider: driftedDefaultProvider,
    overrideCommand,
    includeOverride: false,
  });

  const failedResumeResponsePromise = appPage.waitForResponse(
    response =>
      response.request().method() === "POST" &&
      response.url().endsWith(`/api/sessions/${createdSession.session.id}/resume`)
  );

  await ui.resumeButton.click();

  const failedResumeResponse = await failedResumeResponsePromise;
  const failedResumeBody = await failedResumeResponse.text();
  expect(failedResumeResponse.ok()).toBeFalsy();
  expect(failedResumeBody).toContain(createdSession.session.id);
  expect(failedResumeBody).toContain(overrideProvider);

  await expect(appPage.getByTestId("session-resume-failure")).toBeVisible();
  await expect(appPage.getByTestId("session-resume-failure-provider")).toHaveText(overrideProvider);
  await expect(appPage.getByTestId("session-resume-failure-meta")).toContainText(
    createdSession.session.id
  );
  await expect(appPage.getByTestId("session-resume-failure-meta")).toContainText(
    browserLifecycleAgent
  );
  await browserArtifacts.captureScreenshot("session-provider-resume-failure", appPage);

  await appPage.reload({ waitUntil: "domcontentloaded" });
  await expect(ui.resumeButton).toBeVisible();
  await ui.resumeButton.click();
  await expect(appPage.getByTestId("session-resume-failure")).toBeVisible();
});

async function assertSessionParity(
  runtime: {
    requestJSON: <T>(pathname: string, init?: RequestInit) => Promise<T>;
    requestOperatorJSON?: <T>(pathname: string, init?: RequestInit) => Promise<T>;
    paths?: { cliShim: string; homeDir: string };
  },
  sessionID: string,
  expectedProvider: string
): Promise<void> {
  const httpRecord = await runtime.requestJSON<SessionEnvelope>(
    `/api/sessions/${encodeURIComponent(sessionID)}`
  );
  expect(httpRecord.session.provider).toBe(expectedProvider);

  if (!runtime.requestOperatorJSON) {
    throw new Error("provider override parity check requires operator UDS access");
  }
  const udsRecord = await runtime.requestOperatorJSON<SessionEnvelope>(
    `/api/sessions/${encodeURIComponent(sessionID)}`
  );
  expect(udsRecord.session.provider).toBe(expectedProvider);

  if (!runtime.paths) {
    throw new Error("provider override parity check requires runtime CLI paths");
  }
  const { stdout } = await execFileAsync(
    runtime.paths.cliShim,
    ["session", "status", sessionID, "-o", "json"],
    {
      env: cliEnv(runtime.paths),
    }
  );
  const cliRecord = JSON.parse(stdout) as SessionPayload;
  expect(cliRecord.provider).toBe(expectedProvider);
}

function cliEnv(paths: { cliShim: string; homeDir: string }): NodeJS.ProcessEnv {
  return {
    ...process.env,
    AGH_HOME: paths.homeDir,
    HOME: paths.homeDir,
    PATH: `${path.dirname(paths.cliShim)}:${process.env.PATH ?? ""}`,
  };
}

async function readAgentCommand(homeDir: string, agentName: string): Promise<string> {
  const agentDefPath = path.join(homeDir, "agents", agentName, "AGENT.md");
  const agentDef = await readFile(agentDefPath, "utf8");
  const match = agentDef.match(/^command:\s+(.+)$/m);
  if (!match) {
    throw new Error(`agent definition ${agentDefPath} is missing a command line`);
  }
  return match[1].trim();
}

async function writeWorkspaceConfig(input: {
  rootDir: string;
  defaultProvider: string;
  overrideCommand: string;
  includeOverride: boolean;
}): Promise<void> {
  const configDir = path.join(input.rootDir, ".agh");
  const configPath = path.join(configDir, "config.toml");

  await mkdir(configDir, { recursive: true });

  const lines = [
    "[defaults]",
    `agent = "${browserLifecycleAgent}"`,
    `provider = "${input.defaultProvider}"`,
    "",
  ];

  if (input.includeOverride) {
    lines.push(
      `[providers.${overrideProvider}]`,
      `command = "${escapeTomlString(input.overrideCommand)}"`,
      `default_model = "qa-browser-model"`,
      `api_key_env = "QA_BROWSER_API_KEY"`,
      ""
    );
  }

  await writeFile(configPath, `${lines.join("\n")}\n`, "utf8");
}

function escapeTomlString(value: string): string {
  return value.replaceAll("\\", "\\\\").replaceAll('"', '\\"');
}
