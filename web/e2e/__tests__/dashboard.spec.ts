import { execFile } from "node:child_process";
import { mkdtemp, readFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { promisify } from "node:util";

import { sessionLifecycleTestIds } from "../fixtures/selectors";
import type { BrowserRuntime, WorkspacePayload } from "../fixtures/runtime";
import { expect, test } from "../fixtures/test";
import { useGlobalWorkspaceIfPrompted } from "../fixtures/workspace";

const execFileAsync = promisify(execFile);
const activeSessionStates = new Set(["active", "starting", "stopping"]);
const dashboardAgentAlpha = "dashboard-agent-alpha";
const dashboardAgentBeta = "dashboard-agent-beta";
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
const sensitivePattern =
  /agh_claim_|claim_token["':\s]|telegram-bot-token|pkce|oauth|webhook_secret|provider[_-]?credential/i;

interface SessionSummary {
  id: string;
  state: string;
  workspace_id: string;
}

interface SettingsRestartAction {
  operation_id: string;
  status: string;
  status_url: string;
}

interface SettingsRestartStatus {
  status: string;
}

interface DashboardSnapshot {
  agents: { agents: unknown[] };
  cli?: {
    agents?: unknown;
    daemon?: unknown;
    sessions?: unknown;
    workspaces?: unknown;
  };
  daemonHTTP: unknown;
  daemonUDS?: unknown;
  health: {
    health: {
      status?: string;
      uptime_seconds?: number;
      version?: string;
    };
  };
  sessions: { sessions: SessionSummary[] };
  workspace: WorkspacePayload;
  workspaceDetail: { agents?: unknown[]; sessions?: SessionSummary[] };
  workspaces: { workspaces: WorkspacePayload[] };
}

test.use({
  runtimeOptions: {
    seed: {
      mockAgents: [
        {
          agentName: dashboardAgentAlpha,
          fixtureAgent: "browser-lifecycle-agent",
          fixturePath: browserLifecycleFixture,
        },
        {
          agentName: dashboardAgentBeta,
          fixtureAgent: "browser-lifecycle-agent",
          fixturePath: browserLifecycleFixture,
        },
      ],
    },
  },
});

test("operator sees truthful Dashboard health, metrics, navigation, artifacts, and parity evidence", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const workspace = await prepareDashboardRuntime(runtime);
  await useGlobalWorkspaceIfPrompted(workspaceShell(appPage));
  await appPage.goto(runtime.url("/"), { waitUntil: "domcontentloaded" });

  const snapshot = await captureDashboardSnapshot(runtime, workspace);
  const expectedActiveSessions = snapshot.sessions.sessions.filter(session =>
    activeSessionStates.has(session.state)
  ).length;
  const expectedAgents = snapshot.workspaceDetail.agents?.length ?? snapshot.agents.agents.length;

  await expect(appPage.getByTestId("home-shell")).toBeVisible();
  await expect(appPage.getByTestId("home-connection-indicator")).toHaveAttribute(
    "data-status",
    "connected"
  );
  await expect(appPage.getByTestId("home-daemon-card")).toHaveAttribute("data-status", "healthy", {
    timeout: 15_000,
  });
  await expect(appPage.getByTestId("home-daemon-status-label")).toHaveText(/Healthy|Degraded/);
  expect(snapshot.health.health.version?.trim()).not.toBe("");
  await expect(appPage.getByTestId("home-daemon-version")).toContainText(
    `v${snapshot.health.health.version}`
  );
  await expect(metricValue(appPage, "home-metric-active-sessions")).toHaveText(
    String(expectedActiveSessions)
  );
  await expect(metricValue(appPage, "home-metric-workspaces")).toHaveText(
    String(snapshot.workspaces.workspaces.length)
  );
  await expect(metricValue(appPage, "home-metric-agents")).toHaveText(String(expectedAgents));
  await expect(metricValue(appPage, "home-metric-uptime")).not.toHaveText("—");
  await expect(appPage.getByTestId("home-metric-active-sessions")).toContainText(
    `in ${workspace.name}`
  );

  await assertDashboardNavigation(appPage, runtime);
  await assertDashboardViewportMatrix(appPage, browserArtifacts, runtime);
  await assertDashboardFocus(appPage);

  await runtime.artifactCollector.captureJSON("browser_api_snapshots", snapshot);
  await browserArtifacts.captureScreenshot("dashboard-healthy", appPage);
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
    home_view_visible: true,
    home_connection_status: "connected",
    home_daemon_status: "healthy",
    home_active_sessions_value: String(expectedActiveSessions),
    home_workspaces_value: String(snapshot.workspaces.workspaces.length),
    home_agents_value: String(expectedAgents),
  });

  await assertNoSensitiveLeak(appPage, runtime, snapshot);
});

test("dashboard degrades one failed metric without hiding daemon health", async ({
  appPage,
  runtime,
}) => {
  await prepareDashboardRuntime(runtime);
  await useGlobalWorkspaceIfPrompted(workspaceShell(appPage));
  await appPage.route("**/api/sessions**", async route => {
    await route.fulfill({
      contentType: "application/json",
      status: 503,
      body: JSON.stringify({ error: "sessions unavailable" }),
    });
  });

  await appPage.goto(runtime.url("/"), { waitUntil: "domcontentloaded" });

  await expect(appPage.getByTestId("home-daemon-card")).toHaveAttribute("data-status", "healthy", {
    timeout: 15_000,
  });
  await expect(metricValue(appPage, "home-metric-active-sessions")).toHaveText("—");
  await expect(appPage.getByTestId("home-metric-active-sessions")).toContainText("unavailable");
  await expect(appPage.getByTestId("home-error")).toBeHidden();
});

test("dashboard shows reconnecting state and recovers when health requests resume", async ({
  appPage,
  runtime,
}) => {
  await prepareDashboardRuntime(runtime);
  await useGlobalWorkspaceIfPrompted(workspaceShell(appPage));
  await appPage.route("**/api/observe/health", async route => {
    await route.abort("failed");
  });

  await appPage.goto(runtime.url("/"), { waitUntil: "domcontentloaded" });

  await expect(appPage.getByTestId("home-connection-indicator")).toHaveAttribute(
    "data-status",
    /reconnecting|disconnected/,
    { timeout: 15_000 }
  );
  await expect(appPage.getByTestId("home-daemon-disconnected")).toContainText("agh daemon", {
    timeout: 15_000,
  });

  await appPage.unroute("**/api/observe/health");
  await expect(appPage.getByTestId("home-connection-indicator")).toHaveAttribute(
    "data-status",
    "connected",
    { timeout: 20_000 }
  );
  await expect(appPage.getByTestId("home-daemon-card")).toHaveAttribute("data-status", "healthy");
});

test("dashboard refreshes after daemon restart action without stale health", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const workspace = await prepareDashboardRuntime(runtime);
  await useGlobalWorkspaceIfPrompted(workspaceShell(appPage));
  await appPage.goto(runtime.url("/"), { waitUntil: "domcontentloaded" });

  const beforeRestart = await captureDashboardSnapshot(runtime, workspace);
  await expect(appPage.getByTestId("home-daemon-card")).toHaveAttribute("data-status", "healthy", {
    timeout: 15_000,
  });
  await expect(metricValue(appPage, "home-metric-workspaces")).toHaveText(
    String(beforeRestart.workspaces.workspaces.length)
  );

  const restart = await runtime.requestJSON<SettingsRestartAction>(
    "/api/settings/actions/restart",
    {
      method: "POST",
      body: "{}",
    }
  );
  expect(restart.operation_id).toMatch(
    /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
  );

  await appPage.reload({ waitUntil: "domcontentloaded" });
  await expect(appPage.getByTestId("home-shell")).toBeVisible({ timeout: 15_000 });
  await browserArtifacts.captureScreenshot("dashboard-restart-polling", appPage);

  await expect
    .poll(async () => await pollRestartStatus(runtime, restart.status_url), {
      timeout: 45_000,
    })
    .toBe("ready");
  await appPage.reload({ waitUntil: "domcontentloaded" });

  const afterRestart = await captureDashboardSnapshot(runtime, workspace);
  await expect(appPage.getByTestId("home-connection-indicator")).toHaveAttribute(
    "data-status",
    "connected"
  );
  await expect(appPage.getByTestId("home-daemon-card")).toHaveAttribute("data-status", "healthy");
  await expect(metricValue(appPage, "home-metric-workspaces")).toHaveText(
    String(afterRestart.workspaces.workspaces.length)
  );
  expect(afterRestart.daemonHTTP).toBeDefined();
  await browserArtifacts.captureScreenshot("dashboard-restart-ready", appPage);
});

test("workspace-scoped Dashboard metrics change when the active workspace changes", async ({
  appPage,
  runtime,
}) => {
  if (!runtime.paths?.homeDir) {
    throw new Error("Dashboard workspace switching requires launch-mode runtime paths.");
  }

  const alpha = await prepareDashboardRuntime(runtime);
  const betaRoot = await mkdtemp(path.join(os.tmpdir(), "agh-dashboard-workspace-beta-"));
  const beta = await runtime.resolveWorkspace(betaRoot);

  await useGlobalWorkspaceIfPrompted(workspaceShell(appPage));
  await appPage.goto(runtime.url("/"), { waitUntil: "domcontentloaded" });
  await appPage.getByTestId(`workspace-avatar-${beta.id}`).click();

  const betaSnapshot = await captureDashboardSnapshot(runtime, beta);
  const betaActiveSessions = betaSnapshot.sessions.sessions.filter(session =>
    activeSessionStates.has(session.state)
  ).length;
  expect(betaActiveSessions).toBe(0);
  await expect(appPage.getByTestId(`workspace-avatar-${beta.id}`)).toHaveAttribute(
    "aria-pressed",
    "true"
  );
  await expect(metricValue(appPage, "home-metric-active-sessions")).toHaveText(
    String(betaActiveSessions)
  );
  await expect(appPage.getByTestId("home-metric-active-sessions")).toContainText(`in ${beta.name}`);

  await appPage.getByTestId(`workspace-avatar-${alpha.id}`).click();
  const alphaSnapshot = await captureDashboardSnapshot(runtime, alpha);
  const alphaActiveSessions = alphaSnapshot.sessions.sessions.filter(session =>
    activeSessionStates.has(session.state)
  ).length;
  expect(alphaActiveSessions).toBeGreaterThan(0);
  await expect(appPage.getByTestId(`workspace-avatar-${alpha.id}`)).toHaveAttribute(
    "aria-pressed",
    "true"
  );
  await expect(metricValue(appPage, "home-metric-active-sessions")).toHaveText(
    String(alphaActiveSessions)
  );
  await expect(appPage.getByTestId("home-metric-active-sessions")).toContainText(
    `in ${alpha.name}`
  );
});

async function prepareDashboardRuntime(runtime: BrowserRuntime): Promise<WorkspacePayload> {
  if (!runtime.paths?.homeDir) {
    throw new Error("Dashboard E2E requires launch-mode runtime paths.");
  }

  const workspace = await runtime.resolveWorkspace(runtime.paths.homeDir);
  const sessions = await runtime.requestJSON<{ sessions: SessionSummary[] }>(
    `/api/sessions?workspace=${encodeURIComponent(workspace.id)}`
  );
  const hasDashboardSession = sessions.sessions.some(
    session => session.workspace_id === workspace.id && activeSessionStates.has(session.state)
  );
  if (!hasDashboardSession) {
    await runtime.requestJSON<{ session: SessionSummary }>("/api/sessions", {
      method: "POST",
      body: JSON.stringify({
        agent_name: dashboardAgentAlpha,
        workspace: workspace.id,
      }),
    });
  }

  await expect
    .poll(async () => {
      const payload = await runtime.requestJSON<{ sessions: SessionSummary[] }>(
        `/api/sessions?workspace=${encodeURIComponent(workspace.id)}`
      );
      return payload.sessions.filter(session => activeSessionStates.has(session.state)).length;
    })
    .toBeGreaterThan(0);

  return workspace;
}

async function pollRestartStatus(runtime: BrowserRuntime, statusURL: string): Promise<string> {
  try {
    const payload = await runtime.requestJSON<SettingsRestartStatus>(statusURL);
    return payload.status;
  } catch {
    return "restarting";
  }
}

async function captureDashboardSnapshot(
  runtime: BrowserRuntime,
  workspace: WorkspacePayload
): Promise<DashboardSnapshot> {
  const [health, daemonHTTP, daemonUDS, workspaces, workspaceDetail, sessions, agents, cli] =
    await Promise.all([
      runtime.requestJSON<DashboardSnapshot["health"]>("/api/observe/health"),
      runtime.requestJSON<unknown>("/api/daemon/status"),
      runtime.requestOperatorJSON?.<unknown>("/api/daemon/status"),
      runtime.requestJSON<DashboardSnapshot["workspaces"]>("/api/workspaces"),
      runtime.requestJSON<DashboardSnapshot["workspaceDetail"]>(
        `/api/workspaces/${encodeURIComponent(workspace.id)}`
      ),
      runtime.requestJSON<DashboardSnapshot["sessions"]>(
        `/api/sessions?workspace=${encodeURIComponent(workspace.id)}`
      ),
      runtime.requestJSON<DashboardSnapshot["agents"]>(
        `/api/agents?workspace=${encodeURIComponent(workspace.id)}`
      ),
      captureCLISnapshot(runtime, workspace),
    ]);

  return {
    agents,
    cli,
    daemonHTTP,
    daemonUDS,
    health,
    sessions,
    workspace,
    workspaceDetail,
    workspaces,
  };
}

async function captureCLISnapshot(runtime: BrowserRuntime, workspace: WorkspacePayload) {
  if (!runtime.paths) {
    return undefined;
  }
  return {
    daemon: await runCLIJSON(runtime, ["daemon", "status", "-o", "json"]),
    workspaces: await runCLIJSON(runtime, ["workspace", "list", "-o", "json"]),
    sessions: await runCLIJSON(runtime, [
      "session",
      "list",
      "--workspace",
      workspace.id,
      "-o",
      "json",
    ]),
    agents: await runCLIJSON(runtime, ["agent", "list", "--workspace", workspace.id, "-o", "json"]),
  };
}

async function runCLIJSON(runtime: BrowserRuntime, args: string[]) {
  if (!runtime.paths) {
    throw new Error(`CLI snapshot ${args.join(" ")} requires launch-mode runtime paths.`);
  }
  const { stdout } = await execFileAsync(runtime.paths.cliShim, args, {
    env: {
      ...process.env,
      AGH_HOME: runtime.paths.homeDir,
      HOME: runtime.paths.homeDir,
    },
    maxBuffer: 10 * 1024 * 1024,
  });
  return JSON.parse(stdout) as unknown;
}

function workspaceShell(page: import("@playwright/test").Page) {
  return {
    appSidebar: page.getByTestId(sessionLifecycleTestIds.appSidebar),
    workspaceOnboarding: page.getByTestId(sessionLifecycleTestIds.workspaceOnboarding),
    workspaceUseGlobal: page.getByTestId(sessionLifecycleTestIds.workspaceUseGlobal),
  };
}

function metricValue(page: import("@playwright/test").Page, testId: string) {
  return page.getByTestId(testId).locator('[data-slot="metric-value"]');
}

async function assertDashboardNavigation(
  page: import("@playwright/test").Page,
  runtime: BrowserRuntime
): Promise<void> {
  await page.getByTestId(`agent-row-${dashboardAgentAlpha}`).click();
  await expect.poll(() => new URL(page.url()).pathname).toBe(`/agents/${dashboardAgentAlpha}`);

  await page.getByTestId("nav-network").click();
  await expect.poll(() => new URL(page.url()).pathname).toBe("/network/default/threads");

  await page.getByTestId("nav-tasks").click();
  await expect.poll(() => new URL(page.url()).pathname).toBe("/tasks");

  await page.getByTestId("nav-settings").click();
  await expect.poll(() => new URL(page.url()).pathname).toBe("/settings/general");

  await page.goto(runtime.url("/"), { waitUntil: "domcontentloaded" });
  await expect(page.getByTestId("home-shell")).toBeVisible();
}

async function assertDashboardViewportMatrix(
  page: import("@playwright/test").Page,
  browserArtifacts: {
    captureScreenshot(name?: string, page?: import("@playwright/test").Page): Promise<unknown>;
  },
  runtime: BrowserRuntime
): Promise<void> {
  const viewports = [
    { width: 375, height: 812, name: "mobile" },
    { width: 768, height: 1024, name: "tablet" },
    { width: 1280, height: 900, name: "desktop" },
  ];

  for (const viewport of viewports) {
    await page.setViewportSize({ width: viewport.width, height: viewport.height });
    await page.goto(runtime.url("/"), { waitUntil: "domcontentloaded" });
    await expect(page.getByTestId("home-daemon-card")).toBeVisible();
    await expect(page.getByTestId("home-metric-active-sessions")).toBeVisible();
    await expect(page.getByTestId("home-metric-workspaces")).toBeVisible();
    await expect(page.getByTestId("home-metric-agents")).toBeVisible();
    await expect(page.getByTestId("home-metric-uptime")).toBeVisible();
    await browserArtifacts.captureScreenshot(`dashboard-${viewport.name}`, page);
  }
}

async function assertDashboardFocus(page: import("@playwright/test").Page): Promise<void> {
  const dashboardNav = page.getByTestId("nav-dashboard");
  await dashboardNav.focus();
  await expect(dashboardNav).toBeFocused();
  await expect(dashboardNav).toHaveAccessibleName("Dashboard");
}

async function assertNoSensitiveLeak(
  page: import("@playwright/test").Page,
  runtime: BrowserRuntime,
  snapshot: DashboardSnapshot
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
