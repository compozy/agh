import { execFile } from "node:child_process";
import { mkdir, mkdtemp, readFile, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";
import { promisify } from "node:util";

import { sessionLifecycleSelectors } from "../fixtures/selectors";
import { expect, test } from "../fixtures/test";

const execFileAsync = promisify(execFile);

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
const overrideProvider = "qa-browser-override";
const driftedDefaultProvider = "gemini";

function browserLifecycleSessionPath(sessionId: string): string {
  return `/agents/${browserLifecycleAgent}/sessions/${sessionId}`;
}

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

test("operator can create a provider/model override session and attach without losing provider truth", async ({
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
  await expect(ui.agentRow(browserLifecycleAgent)).toBeVisible();

  await ui.agentRow(browserLifecycleAgent).click();
  await expect.poll(() => new URL(appPage.url()).pathname).toBe(`/agents/${browserLifecycleAgent}`);
  await expect(ui.agentPageNewSession).toBeVisible();
  await ui.agentPageNewSession.click();

  await expect(appPage.getByTestId("session-create-dialog")).toBeVisible();
  await expect(appPage.getByTestId("session-create-agent-select")).toContainText(
    browserLifecycleAgent
  );
  const providerSelect = appPage.getByTestId("session-create-provider-select");
  await expect(providerSelect).toContainText("Claude Code");
  await expect(appPage.getByTestId("session-create-provider-runtime")).toContainText("claude");

  await providerSelect.click();
  const dialogOptions = await appPage
    .locator('[data-testid^="provider-command-item-"]')
    .evaluateAll(items =>
      items
        .map(item => item.getAttribute("data-testid")?.replace("provider-command-item-", ""))
        .filter((value): value is string => Boolean(value))
        .sort()
    );
  expect(dialogOptions).toEqual(workspaceDetail.providers.map(provider => provider.name).sort());

  await browserArtifacts.captureScreenshot("session-provider-dialog-desktop", appPage);
  await appPage.setViewportSize({ width: 375, height: 812 });
  await expect(providerSelect).toBeVisible();
  await browserArtifacts.captureScreenshot("session-provider-dialog-mobile", appPage);
  await appPage.setViewportSize({ width: 1280, height: 800 });

  await appPage.getByTestId(`provider-command-item-${overrideProvider}`).click();
  const catalogRefreshResponse = appPage.waitForResponse(
    response =>
      response.request().method() === "POST" &&
      response.url().endsWith(`/api/model-catalog/providers/${overrideProvider}/models/refresh`)
  );
  const refreshCatalog = appPage.getByTestId("session-create-catalog-refresh");
  await expect(refreshCatalog).toBeEnabled();
  await refreshCatalog.click();
  expect((await catalogRefreshResponse).ok()).toBe(true);
  await expect(appPage.getByTestId("session-create-catalog-empty")).toBeVisible();

  await appPage.getByTestId("session-create-model-select").click();
  await appPage.getByTestId("model-command-input").fill("qa-browser-model");
  await expect(appPage.getByTestId("model-command-item-custom")).toBeVisible();
  await appPage.getByTestId("model-command-item-custom").click();
  await expect(appPage.getByTestId("session-create-model-select")).toContainText(
    "qa-browser-model"
  );
  await expect(appPage.getByTestId("session-create-reasoning-select")).toBeDisabled();

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
    model?: string;
    provider?: string;
    reasoning_effort?: string;
    workspace?: string;
  };
  expect(createRequestBody).toMatchObject({
    agent_name: browserLifecycleAgent,
    model: "qa-browser-model",
    provider: overrideProvider,
    workspace: workspace.id,
  });
  expect(createRequestBody).not.toHaveProperty("reasoning_effort");
  expect(createResponse.ok()).toBeTruthy();

  const createdSession = (await createResponse.json()) as SessionEnvelope;
  expect(createdSession.session.provider).toBe(overrideProvider);

  await expect
    .poll(() => new URL(appPage.url()).pathname)
    .toBe(browserLifecycleSessionPath(createdSession.session.id));
  await expect(ui.chatHeader).toBeVisible();
  await expect(appPage.getByTestId("session-provider-badge")).toHaveText(overrideProvider);
  await browserArtifacts.captureScreenshot("session-provider-created", appPage);

  await assertSessionParity(
    runtime,
    createdSession.session.workspace_id,
    createdSession.session.id,
    overrideProvider
  );

  await writeWorkspaceConfig({
    rootDir: workspaceRoot,
    defaultProvider: driftedDefaultProvider,
    overrideCommand,
    includeOverride: true,
  });

  const attachResponsePromise = appPage.waitForResponse(
    response =>
      response.request().method() === "POST" &&
      response
        .url()
        .endsWith(
          sessionAPIPath(createdSession.session.workspace_id, createdSession.session.id, "/attach")
        )
  );

  await expect(ui.resumeButton).toBeVisible();
  await ui.resumeButton.click();
  expect((await attachResponsePromise).ok()).toBe(true);
  await expect(ui.stopButton).toBeVisible();
  await expect(appPage.getByTestId("session-provider-badge")).toHaveText(overrideProvider);
  await assertSessionParity(
    runtime,
    createdSession.session.workspace_id,
    createdSession.session.id,
    overrideProvider
  );

  await ui.stopButton.click();
  await expect(ui.resumeButton).not.toBeVisible();
});

async function assertSessionParity(
  runtime: {
    requestJSON: <T>(pathname: string, init?: RequestInit) => Promise<T>;
    requestOperatorJSON?: <T>(pathname: string, init?: RequestInit) => Promise<T>;
    paths?: { cliShim: string; homeDir: string };
  },
  workspaceID: string,
  sessionID: string,
  expectedProvider: string
): Promise<void> {
  const path = sessionAPIPath(workspaceID, sessionID);
  const httpRecord = await runtime.requestJSON<SessionEnvelope>(path);
  expect(httpRecord.session.provider).toBe(expectedProvider);

  if (!runtime.requestOperatorJSON) {
    throw new Error("provider override parity check requires operator UDS access");
  }
  const udsRecord = await runtime.requestOperatorJSON<SessionEnvelope>(path);
  expect(udsRecord.session.provider).toBe(expectedProvider);

  if (!runtime.paths) {
    throw new Error("provider override parity check requires runtime CLI paths");
  }
  const { stdout } = await execFileAsync(
    runtime.paths.cliShim,
    ["session", "list", "--all", "-o", "json"],
    {
      env: cliEnv(runtime.paths),
    }
  );
  const cliRecords = JSON.parse(stdout) as SessionPayload[];
  const cliRecord = cliRecords.find(session => session.id === sessionID);
  expect(cliRecord?.provider).toBe(expectedProvider);
}

function sessionAPIPath(workspaceID: string, sessionID: string, suffix = ""): string {
  return `/api/workspaces/${encodeURIComponent(workspaceID)}/sessions/${encodeURIComponent(
    sessionID
  )}${suffix}`;
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
      `[providers.${overrideProvider}.models]`,
      `default = "qa-browser-model"`,
      `[[providers.${overrideProvider}.models.curated]]`,
      `id = "qa-browser-model"`,
      `display_name = "QA Browser Model"`,
      `supports_reasoning = true`,
      `reasoning_efforts = ["low", "medium", "high"]`,
      `default_reasoning_effort = "medium"`,
      `[[providers.${overrideProvider}.credential_slots]]`,
      `name = "api_key"`,
      `target_env = "QA_BROWSER_API_KEY"`,
      `secret_ref = "env:QA_BROWSER_API_KEY"`,
      `kind = "api_key"`,
      `required = false`,
      ""
    );
  }

  await writeFile(configPath, `${lines.join("\n")}\n`, "utf8");
}

function escapeTomlString(value: string): string {
  return value.replaceAll("\\", "\\\\").replaceAll('"', '\\"');
}
