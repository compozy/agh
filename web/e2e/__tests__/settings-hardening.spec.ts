import { execFile } from "node:child_process";
import { readFile } from "node:fs/promises";
import path from "node:path";
import process from "node:process";
import { promisify } from "node:util";

import type { Page } from "@playwright/test";

import { sessionLifecycleSelectors } from "../fixtures/selectors";
import type { BrowserRuntime, RuntimePaths } from "../fixtures/runtime";
import { expect, test } from "../fixtures/test";

const execFileAsync = promisify(execFile);

const sensitivePattern =
  /agh_claim_[a-z0-9._-]+|["']claim_token["']\s*:\s*["']?[a-z0-9._-]{8,}|(?:authorization\s*:\s*bearer|bearer)\s+["']?[a-z0-9._-]{8,}|(?:api[_-]?key|bearer[_-]?token|mcp[_-]?auth|oauth[_-]?(?:access(?:[_-]?token)?|client(?:[_-]?secret)?|refresh(?:[_-]?token)?|secret|token)|pkce[_-]?(?:challenge|secret|verifier)|provider[_-]?credential|telegram-bot-token|browser-settings-secret)\s*[:=]\s*["']?[a-z0-9._:-]{8,}|\b\d{6,}:[a-z0-9_-]{20,}/i;

test.use({
  runtimeOptions: {
    env: {
      ...process.env,
      AGH_TEST_TELEGRAM_TOKEN: "telegram-bot-token",
    },
  },
});

test("operator stores and deletes a vault secret without plaintext readback", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  assertLaunchRuntime(runtime, "vault lifecycle");

  const secretRef = `vault:providers/browser-settings-secret-${Date.now()}`;
  const secretValue = "browser-settings-secret-value-11";

  await useGlobalWorkspaceIfPrompted(sessionLifecycleSelectors(appPage));
  await appPage.goto(runtime.url("/settings/vault"), { waitUntil: "domcontentloaded" });
  await expect(appPage.getByTestId("settings-shell")).toBeVisible({ timeout: 20_000 });
  await expect(appPage.getByTestId("settings-page-vault-create")).toBeVisible();

  await appPage.getByTestId("settings-page-vault-create").click();
  await expect(appPage.getByTestId("settings-vault-editor")).toBeVisible();
  await appPage.getByTestId("settings-vault-editor-ref-input").fill("providers/bad-ref");
  await appPage.getByTestId("settings-vault-editor-kind-input").fill("api_key");
  await appPage.getByTestId("settings-vault-editor-secret-value-input").fill(secretValue);
  await expect(appPage.getByTestId("settings-vault-editor-error")).toContainText(
    "Vault refs must start with vault:."
  );
  await expect(appPage.getByTestId("settings-vault-editor-save")).toBeDisabled();

  await appPage.getByTestId("settings-vault-editor-ref-input").fill(secretRef);
  await expect(appPage.getByTestId("settings-vault-editor-save")).toBeEnabled();
  await appPage.getByTestId("settings-vault-editor-save").click();

  await expect(appPage.getByTestId("settings-vault-editor")).toBeHidden();
  await expect(appPage.getByTestId("settings-page-vault-action-result")).toContainText(secretRef);
  await expect(appPage.locator("body")).not.toContainText(secretValue);

  await appPage.getByTestId("settings-page-vault-namespace").selectOption("providers");
  await appPage.getByTestId("settings-page-vault-prefix").fill(secretRef);
  await expect(appPage.getByTestId("vault-secrets-row")).toHaveCount(1);
  await expect(appPage.getByTestId("vault-secrets-row")).toContainText(secretRef);
  await expect(appPage.getByTestId("vault-secrets-row")).toContainText("api_key");
  await expect(appPage.getByTestId("vault-secrets-row")).not.toContainText(secretValue);

  const httpMetadata = await runtime.requestJSON<unknown>(
    `/api/vault/secrets/metadata?ref=${encodeURIComponent(secretRef)}`
  );
  const udsMetadata = await requestOperatorJSON<unknown>(
    runtime,
    `/api/vault/secrets/metadata?ref=${encodeURIComponent(secretRef)}`
  );
  const cliMetadata = await runCLIJSON(runtime.paths, ["vault", "get", secretRef, "-o", "json"]);
  const cliList = await runCLIJSON(runtime.paths, [
    "vault",
    "list",
    "--namespace",
    "providers",
    "--prefix",
    secretRef,
    "-o",
    "json",
  ]);

  const snapshot = {
    http_metadata: httpMetadata,
    uds_metadata: udsMetadata,
    cli_metadata: cliMetadata,
    cli_list: cliList,
    ui_row_count: await appPage.getByTestId("vault-secrets-row").count(),
  };
  expect(JSON.stringify(snapshot)).not.toContain(secretValue);
  expect(JSON.stringify(snapshot)).not.toMatch(sensitivePattern);
  await runtime.artifactCollector.captureJSON("browser_api_snapshots", snapshot);
  await browserArtifacts.captureScreenshot("settings-vault-lifecycle-desktop", appPage);
  await captureSettingsViewportMatrix(appPage, browserArtifacts, runtime, "/settings/vault");

  await appPage.getByTestId(`vault-secrets-delete-${secretRef}`).click();
  await expect(appPage.getByTestId("settings-vault-delete")).toBeVisible();
  await expect(appPage.getByTestId("settings-vault-delete-description")).toContainText(secretRef);
  await appPage.getByTestId("settings-vault-delete-confirm").click();

  await expect(appPage.getByTestId("settings-page-vault-action-result")).toContainText("Deleted");
  await expect(appPage.getByTestId("vault-secrets-row")).toHaveCount(0);

  const deletedResponse = await appPage.request.get(
    runtime.url(`/api/vault/secrets/metadata?ref=${encodeURIComponent(secretRef)}`)
  );
  expect(deletedResponse.status()).toBe(404);

  await browserArtifacts.persist(appPage);
  await assertNoSettingsSensitiveLeak(appPage, runtime, [secretValue]);
});

test("operator applies Memory, Network, Automation, and Observability settings with config parity", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  assertLaunchRuntime(runtime, "settings parity");

  await useGlobalWorkspaceIfPrompted(sessionLifecycleSelectors(appPage));

  const memoryBefore = await runtime.requestJSON<{ config: { recall: { top_k: number } } }>(
    "/api/settings/memory"
  );
  const nextTopK = memoryBefore.config.recall.top_k + 1;
  await appPage.goto(runtime.url("/settings/memory"), { waitUntil: "domcontentloaded" });
  await expect(appPage.getByTestId("settings-page-memory-recall-top-k-input")).toBeVisible();
  await appPage.getByTestId("settings-page-memory-recall-top-k-input").fill(String(nextTopK));
  await expect(appPage.getByTestId("settings-page-memory-save")).toBeEnabled();
  await appPage.getByTestId("settings-page-memory-save").click();
  await expect(appPage.getByTestId("settings-page-memory-save-applied")).toContainText(
    /restart required/i
  );

  const nextChannel = `settings-${Date.now()}`;
  await appPage.goto(runtime.url("/settings/network"), { waitUntil: "domcontentloaded" });
  await expect(appPage.getByTestId("settings-page-network-default-channel-input")).toBeVisible();
  await appPage.getByTestId("settings-page-network-default-channel-input").fill(nextChannel);
  await expect(appPage.getByTestId("settings-page-network-save")).toBeEnabled();
  await appPage.getByTestId("settings-page-network-save").click();
  await expect(appPage.getByTestId("settings-page-network-save-applied")).toContainText(
    /restart required/i
  );
  await expect(appPage.getByTestId("settings-page-network-restart-banner")).toBeVisible();

  const automationBefore = await runtime.requestJSON<{
    config: { max_concurrent_jobs: number };
  }>("/api/settings/automation");
  const nextMaxConcurrent = automationBefore.config.max_concurrent_jobs + 1;
  await appPage.goto(runtime.url("/settings/automation"), { waitUntil: "domcontentloaded" });
  await expect(appPage.getByTestId("settings-page-automation-max-concurrent-input")).toBeVisible();
  await appPage
    .getByTestId("settings-page-automation-max-concurrent-input")
    .fill(String(nextMaxConcurrent));
  await expect(appPage.getByTestId("settings-page-automation-save")).toBeEnabled();
  await appPage.getByTestId("settings-page-automation-save").click();
  await expect(appPage.getByTestId("settings-page-automation-save-applied")).toContainText(
    /restart required/i
  );

  const observabilityBefore = await runtime.requestJSON<{
    config: { retention_days: number };
  }>("/api/settings/observability");
  const nextRetentionDays = observabilityBefore.config.retention_days + 1;
  await appPage.goto(runtime.url("/settings/observability"), { waitUntil: "domcontentloaded" });
  await expect(appPage.getByTestId("settings-page-observability-retention-days")).toBeVisible();
  await appPage.getByTestId("settings-page-observability-retention-days").fill("-1");
  await expect(appPage.getByTestId("settings-page-observability-save-invalid")).toContainText(
    "Resolve validation errors"
  );
  await expect(appPage.getByTestId("settings-page-observability-save")).toBeDisabled();
  await appPage
    .getByTestId("settings-page-observability-retention-days")
    .fill(String(nextRetentionDays));
  await expect(appPage.getByTestId("settings-page-observability-save")).toBeEnabled();
  await appPage.getByTestId("settings-page-observability-save").click();
  await expect(appPage.getByTestId("settings-page-observability-save-applied")).toContainText(
    /restart required/i
  );

  await appPage.goto(runtime.url("/settings/providers"), { waitUntil: "domcontentloaded" });
  await expect(appPage.getByTestId("settings-page-providers-card-codex")).toBeVisible();
  await expect(appPage.getByTestId("settings-page-providers-card-codex-catalog")).toBeVisible();

  const parity = {
    http: {
      memory: await runtime.requestJSON<unknown>("/api/settings/memory"),
      network: await runtime.requestJSON<unknown>("/api/settings/network"),
      automation: await runtime.requestJSON<unknown>("/api/settings/automation"),
      observability: await runtime.requestJSON<unknown>("/api/settings/observability"),
      provider_catalog: await runtime.requestJSON<unknown>("/api/providers/codex/models/status"),
    },
    uds: {
      memory: await requestOperatorJSON<unknown>(runtime, "/api/settings/memory"),
      network: await requestOperatorJSON<unknown>(runtime, "/api/settings/network"),
      automation: await requestOperatorJSON<unknown>(runtime, "/api/settings/automation"),
      observability: await requestOperatorJSON<unknown>(runtime, "/api/settings/observability"),
      provider_catalog: await requestOperatorJSON<unknown>(
        runtime,
        "/api/providers/codex/models/status"
      ),
    },
    cli: {
      memory_top_k: await runCLIJSON(runtime.paths, [
        "config",
        "get",
        "memory.recall.top_k",
        "-o",
        "json",
      ]),
      network_default_channel: await runCLIJSON(runtime.paths, [
        "config",
        "get",
        "network.default_channel",
        "-o",
        "json",
      ]),
      automation_max_concurrent_jobs: await runCLIJSON(runtime.paths, [
        "config",
        "get",
        "automation.max_concurrent_jobs",
        "-o",
        "json",
      ]),
      observability_retention_days: await runCLIJSON(runtime.paths, [
        "config",
        "get",
        "observability.retention_days",
        "-o",
        "json",
      ]),
      provider_catalog: await runCLIJSON(runtime.paths, [
        "provider",
        "models",
        "status",
        "codex",
        "-o",
        "json",
      ]),
    },
    config_file_excerpt: await readFile(runtime.paths.configFile, "utf8"),
  };

  expect(JSON.stringify(parity.http.memory)).toContain(`"top_k":${nextTopK}`);
  expect(JSON.stringify(parity.http.network)).toContain(nextChannel);
  expect(JSON.stringify(parity.http.automation)).toContain(
    `"max_concurrent_jobs":${nextMaxConcurrent}`
  );
  expect(JSON.stringify(parity.http.observability)).toContain(
    `"retention_days":${nextRetentionDays}`
  );
  expect(JSON.stringify(parity.cli.memory_top_k)).toContain(`"value":${nextTopK}`);
  expect(JSON.stringify(parity.cli.network_default_channel)).toContain(nextChannel);
  expect(JSON.stringify(parity.cli.automation_max_concurrent_jobs)).toContain(
    `"value":${nextMaxConcurrent}`
  );
  expect(JSON.stringify(parity.cli.observability_retention_days)).toContain(
    `"value":${nextRetentionDays}`
  );
  expect(parity.config_file_excerpt).toContain("[network]");
  expect(parity.config_file_excerpt).toContain(`default_channel = "${nextChannel}"`);
  expect(JSON.stringify(parity)).not.toMatch(sensitivePattern);

  await runtime.artifactCollector.captureJSON("browser_api_snapshots", parity);
  await browserArtifacts.captureScreenshot("settings-operational-sections-parity", appPage);
  await captureSettingsViewportMatrix(
    appPage,
    browserArtifacts,
    runtime,
    "/settings/observability"
  );
  await browserArtifacts.persist(appPage);
  await assertNoSettingsSensitiveLeak(appPage, runtime, []);
});

test("operator sees restart failure and active-session warning without losing recovery controls", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  await useGlobalWorkspaceIfPrompted(sessionLifecycleSelectors(appPage));
  await appPage.goto(runtime.url("/settings/general"), { waitUntil: "domcontentloaded" });
  await expect(appPage.getByTestId("settings-page-general-session-timeout-input")).toBeVisible();

  await appPage.route("**/api/settings/actions/restart/op-settings-failed", async route => {
    const now = new Date().toISOString();
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        operation_id: "op-settings-failed",
        status: "failed",
        status_url: "/api/settings/actions/restart/op-settings-failed",
        active_session_count: 2,
        failure_reason: "browser restart fault injection",
        started_at: now,
        updated_at: now,
        completed_at: now,
      }),
    });
  });
  await appPage.route("**/api/settings/actions/restart", async route => {
    const now = new Date().toISOString();
    await route.fulfill({
      status: 202,
      contentType: "application/json",
      body: JSON.stringify({
        operation_id: "op-settings-failed",
        status: "stopping",
        status_url: "/api/settings/actions/restart/op-settings-failed",
        active_session_count: 2,
        started_at: now,
        updated_at: now,
      }),
    });
  });

  const currentTimeout = await appPage
    .getByTestId("settings-page-general-session-timeout-input")
    .inputValue();
  await appPage
    .getByTestId("settings-page-general-session-timeout-input")
    .fill(nextNumberString(currentTimeout));
  await expect(appPage.getByTestId("settings-page-general-save")).toBeEnabled();
  await appPage.getByTestId("settings-page-general-save").click();
  await expect(appPage.getByTestId("settings-page-general-restart-banner")).toBeVisible();

  await appPage.getByTestId("settings-page-general-restart-banner-trigger").click();
  await expect(appPage.getByTestId("settings-page-general-restart-banner-message")).toContainText(
    "Daemon restart failed: browser restart fault injection"
  );
  await expect(
    appPage.getByTestId("settings-page-general-restart-banner-active-sessions")
  ).toContainText("2 active sessions");
  await expect(appPage.getByTestId("settings-page-general-restart-banner-trigger")).toBeEnabled();

  const snapshot = {
    restart_failure_message: await appPage
      .getByTestId("settings-page-general-restart-banner-message")
      .textContent(),
    active_sessions: await appPage
      .getByTestId("settings-page-general-restart-banner-active-sessions")
      .textContent(),
  };
  await runtime.artifactCollector.captureJSON("browser_api_snapshots", snapshot);
  await browserArtifacts.captureScreenshot("settings-restart-failure-active-sessions", appPage);
  await browserArtifacts.persist(appPage);
  await assertNoSettingsSensitiveLeak(appPage, runtime, []);
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

function assertLaunchRuntime(
  runtime: BrowserRuntime,
  context: string
): asserts runtime is BrowserRuntime & { paths: RuntimePaths } {
  if (!runtime.paths) {
    throw new Error(`${context} checks require launch-mode runtime paths.`);
  }
}

async function requestOperatorJSON<T>(
  runtime: BrowserRuntime & { paths: RuntimePaths },
  pathname: string,
  init?: RequestInit
): Promise<T> {
  if (!runtime.requestOperatorJSON) {
    throw new Error(`operator request ${pathname} requires UDS support`);
  }
  return await runtime.requestOperatorJSON<T>(pathname, init);
}

async function runCLIJSON(paths: RuntimePaths, args: string[]): Promise<unknown> {
  const { stdout } = await execFileAsync(paths.cliShim, args, { env: cliEnv(paths) });
  return JSON.parse(extractJSON(stdout)) as unknown;
}

function extractJSON(stdout: string): string {
  const trimmed = stdout.trim();
  const objectIndex = trimmed.indexOf("{");
  const arrayIndex = trimmed.indexOf("[");
  const start = [objectIndex, arrayIndex].filter(index => index >= 0).sort((a, b) => a - b)[0];
  if (start === undefined) {
    throw new Error(`CLI output did not contain JSON: ${stdout}`);
  }
  return trimmed.slice(start);
}

function cliEnv(paths: RuntimePaths): NodeJS.ProcessEnv {
  return {
    ...process.env,
    AGH_HOME: paths.homeDir,
    HOME: paths.homeDir,
    PATH: `${path.dirname(paths.cliShim)}:${process.env.PATH ?? ""}`,
  };
}

async function captureSettingsViewportMatrix(
  appPage: Page,
  browserArtifacts: { captureScreenshot: (name: string, page?: Page) => Promise<unknown> },
  runtime: BrowserRuntime,
  pathname: string
): Promise<void> {
  for (const width of [375, 768, 1280]) {
    await appPage.setViewportSize({ width, height: 820 });
    await appPage.goto(runtime.url(pathname), { waitUntil: "domcontentloaded" });
    await expect(appPage.getByTestId("settings-shell")).toBeVisible();
    await expect(appPage.getByTestId("settings-section-nav")).toBeVisible();
    await browserArtifacts.captureScreenshot(`settings-viewport-${width}`, appPage);
  }
}

async function assertNoSettingsSensitiveLeak(
  appPage: Page,
  runtime: BrowserRuntime,
  explicitSecrets: string[]
): Promise<void> {
  await expect(appPage.locator("body")).not.toContainText(sensitivePattern);
  for (const secret of explicitSecrets) {
    await expect(appPage.locator("body")).not.toContainText(secret);
  }

  const payloads = [
    await readFileIfExists(runtime.artifactCollector.artifactPath("browser_console")),
    await readFileIfExists(runtime.artifactCollector.artifactPath("browser_network")),
    await readFileIfExists(runtime.artifactCollector.artifactPath("browser_route_state")),
    await readFileIfExists(runtime.artifactCollector.artifactPath("browser_api_snapshots")),
  ];
  for (const payload of payloads) {
    expect(payload).not.toMatch(sensitivePattern);
    for (const secret of explicitSecrets) {
      expect(payload).not.toContain(secret);
    }
  }
  if (runtime.paths?.daemonLog) {
    const daemonLog = await readFileIfExists(runtime.paths.daemonLog);
    expect(daemonLog).not.toMatch(sensitivePattern);
    for (const secret of explicitSecrets) {
      expect(daemonLog).not.toContain(secret);
    }
  }
}

async function readFileIfExists(filePath: string): Promise<string> {
  try {
    return await readFile(filePath, "utf8");
  } catch (error) {
    const nodeError = error as NodeJS.ErrnoException;
    if (nodeError.code === "ENOENT") {
      return "";
    }
    throw error;
  }
}

function nextNumberString(value: string): string {
  const parsed = Number.parseInt(value.trim(), 10);
  return String(Number.isFinite(parsed) ? parsed + 1 : 46);
}
