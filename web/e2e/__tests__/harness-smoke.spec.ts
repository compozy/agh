import { readdir, readFile, stat } from "node:fs/promises";

import type { BrowserConsoleEntry, BrowserNetworkEntry } from "../fixtures/artifacts";
import {
  assertNoSensitiveArtifactPayload,
  assertSameRuntimeFields,
  captureBrowserTransportSnapshot,
  captureViewportEvidence,
  e2eScenarioContracts,
  requestBrowserRuntimeOperatorJSON,
  runBrowserRuntimeCLIJSON,
} from "../fixtures/scenario-contracts";
import { expect, test } from "../fixtures/test";

test("boots against the daemon-served onboarding shell and captures a trace plus screenshot bundle", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  await expect(appPage.getByTestId("workspace-onboarding")).toBeVisible();

  const manifest = await browserArtifacts.persist(appPage);
  expect(manifest.artifacts).toEqual(
    expect.arrayContaining([
      expect.objectContaining({ kind: "browser_trace", path: "browser_trace.zip" }),
      expect.objectContaining({ kind: "browser_screenshots", path: "browser_screenshots" }),
    ])
  );

  const traceStat = await stat(runtime.artifactCollector.artifactPath("browser_trace"));
  expect(traceStat.isFile()).toBe(true);

  const screenshots = await readdir(runtime.artifactCollector.artifactPath("browser_screenshots"));
  expect(screenshots.length).toBeGreaterThan(0);
});

test("records harness scenario contract, viewport evidence, and HTTP UDS CLI parity", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  await expect(appPage.getByTestId("workspace-onboarding")).toBeVisible();

  const harnessContract = e2eScenarioContracts.find(contract => contract.id === "TC-HARNESS-001");
  expect(harnessContract).toMatchObject({
    artifacts: expect.arrayContaining(["browser_transport_snapshots"]),
    executionAuditIDs: expect.arrayContaining(["C1", "C6", "C12", "C13", "C16", "C17", "C18"]),
    module: "runtime-harness-transport",
    providerBoundary: "bounded_fake",
  });

  const [httpStatus, udsStatus, cliStatus] = await Promise.all([
    runtime.requestJSON<Record<string, unknown>>("/api/daemon/status"),
    requestBrowserRuntimeOperatorJSON<Record<string, unknown>>(runtime, "/api/daemon/status"),
    runBrowserRuntimeCLIJSON<Record<string, unknown>>(runtime, ["daemon", "status"]),
  ]);
  const httpProjection = daemonStatusProjection(httpStatus);
  const udsProjection = daemonStatusProjection(udsStatus);
  const cliProjection = daemonStatusProjection(cliStatus);
  assertSameRuntimeFields("daemon status HTTP/UDS", httpProjection, udsProjection, ["status"]);
  assertSameRuntimeFields("daemon status HTTP/CLI", httpProjection, cliProjection, ["status"]);

  const transportSnapshot = await captureBrowserTransportSnapshot(runtime, "TC-HARNESS-001", {
    cli: cliStatus,
    http: httpStatus,
    uds: udsStatus,
  });
  assertNoSensitiveArtifactPayload([transportSnapshot]);

  const viewportEvidence = await captureViewportEvidence({
    assertVisible: async () => {
      await expect(appPage.getByTestId("workspace-onboarding")).toBeVisible();
    },
    browserArtifacts,
    moduleName: "harness-smoke",
    page: appPage,
  });
  expect(viewportEvidence.map(viewport => viewport.width)).toEqual([375, 768, 1280]);

  await runtime.artifactCollector.captureJSON("browser_api_snapshots", {
    harness_contract: harnessContract,
    transport_snapshot: transportSnapshot,
    viewport_evidence: viewportEvidence,
  });
  const manifest = await browserArtifacts.persist(appPage);
  expect(manifest.artifacts).toEqual(
    expect.arrayContaining([
      expect.objectContaining({ kind: "browser_api_snapshots" }),
      expect.objectContaining({ kind: "browser_route_state" }),
      expect.objectContaining({ kind: "browser_transport_snapshots" }),
    ])
  );
});

function daemonStatusProjection(payload: Record<string, unknown>): Record<string, unknown> {
  const nested = payload.daemon;
  if (nested !== null && typeof nested === "object" && !Array.isArray(nested)) {
    const daemon = nested as Record<string, unknown>;
    if (typeof daemon.status === "string") {
      return { status: daemon.status };
    }
  }
  if (typeof payload.status === "string") {
    return { status: payload.status };
  }
  throw new Error(
    `daemon status payload did not expose a status field: ${JSON.stringify(payload)}`
  );
}

test("captures console and network diagnostics after a forced failure path", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  await expect(appPage.getByTestId("workspace-onboarding")).toBeVisible();

  const failure = await appPage.evaluate(async () => {
    console.error("agh-playwright-forced-console-error");
    const response = await fetch("/api/not-found");
    return { ok: response.ok, status: response.status };
  });

  expect(failure).toEqual({ ok: false, status: 404 });

  await browserArtifacts.persist(appPage);

  const consoleEntries = JSON.parse(
    await readFile(runtime.artifactCollector.artifactPath("browser_console"), "utf8")
  ) as BrowserConsoleEntry[];
  expect(
    consoleEntries.some(entry => entry.text.includes("agh-playwright-forced-console-error"))
  ).toBe(true);

  const networkEntries = JSON.parse(
    await readFile(runtime.artifactCollector.artifactPath("browser_network"), "utf8")
  ) as BrowserNetworkEntry[];
  expect(
    networkEntries.some(
      entry =>
        entry.event === "response" && entry.status === 404 && entry.url.endsWith("/api/not-found")
    )
  ).toBe(true);
});
