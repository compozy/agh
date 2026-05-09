import { execFile } from "node:child_process";
import path from "node:path";
import process from "node:process";
import { promisify } from "node:util";

import { sessionLifecycleSelectors } from "../fixtures/selectors";
import type { BrowserRuntime, RuntimePaths } from "../fixtures/runtime";
import { expect, test } from "../fixtures/test";

const execFileAsync = promisify(execFile);

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
  assertLaunchRuntime(runtime);

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

  const general = await requestOperatorJSON<{ config: { session_timeout: string } }>(
    runtime,
    "/api/settings/general"
  );
  const nextTimeout = nextDurationSeconds(general.config.session_timeout);
  const udsMutation = await requestOperatorJSON<unknown>(runtime, "/api/settings/general", {
    method: "PATCH",
    body: JSON.stringify({
      config: {
        ...general.config,
        session_timeout: nextTimeout,
      },
    }),
  });
  const cliValue = await runCLIJSON(runtime.paths, [
    "config",
    "get",
    "session.limits.timeout",
    "-o",
    "json",
  ]);

  expect(JSON.stringify(udsMutation)).toContain("restart_required");
  expect(JSON.stringify(udsMutation)).toContain("global-config");
  expect(JSON.stringify(cliValue)).toContain(nextTimeout);
  await runtime.artifactCollector.captureJSON("browser_api_snapshots", {
    http_restriction: remoteHTTPAPIBlockedMessage,
    uds_mutation: udsMutation,
    cli_value: cliValue,
  });
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

function assertLaunchRuntime(
  runtime: BrowserRuntime
): asserts runtime is BrowserRuntime & { paths: RuntimePaths } {
  if (!runtime.paths) {
    throw new Error("settings transport checks require launch-mode runtime paths.");
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
  return JSON.parse(stdout) as unknown;
}

function cliEnv(paths: RuntimePaths): NodeJS.ProcessEnv {
  return {
    ...process.env,
    AGH_HOME: paths.homeDir,
    HOME: paths.homeDir,
    PATH: `${path.dirname(paths.cliShim)}:${process.env.PATH ?? ""}`,
  };
}

function nextDurationSeconds(value: string): string {
  const match = value.trim().match(/^(\d+)s$/);
  if (match) {
    return `${Number.parseInt(match[1], 10) + 1}s`;
  }
  return "46s";
}
