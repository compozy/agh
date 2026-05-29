import { execFile } from "node:child_process";
import { createHash } from "node:crypto";
import { mkdir, mkdtemp, readFile, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";
import { promisify } from "node:util";

import type { Page } from "@playwright/test";

import { captureRouteState } from "../fixtures/browser-artifact-session";
import { settingsOperatorSelectors, sessionLifecycleSelectors } from "../fixtures/selectors";
import type { BrowserRuntime, RuntimePaths, WorkspacePayload } from "../fixtures/runtime";
import {
  assertNoSensitiveArtifactPayload,
  captureBrowserTransportSnapshot,
  captureViewportEvidence,
  requestBrowserRuntimeOperatorJSON,
  runBrowserRuntimeCLIJSON,
  sensitiveArtifactPattern,
} from "../fixtures/scenario-contracts";
import { expect, test } from "../fixtures/test";
import { useGlobalWorkspaceIfPrompted } from "../fixtures/workspace";

const execFileAsync = promisify(execFile);

const extensionName = "browser-tool-provider";
const toolID = "ext__browser_tool_provider__search";
const bundleName = "browser-extensibility";
const bundleProfile = "default";
const bundleChannel = "extensibility-ops";
const extensionSecretSentinel = "browser-extension-secret-value-13";
const toolPermissionFixture = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
  "..",
  "..",
  "internal",
  "testutil",
  "acpmock",
  "testdata",
  "tool_permission_fixture.json"
);
const denyPermissionFixture = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
  "..",
  "..",
  "internal",
  "testutil",
  "acpmock",
  "testdata",
  "browser_session_hardening_fixture.json"
);
const toolPermissionAgent = "golden";
const denyPermissionAgent = "permission-hardening-agent";

interface ExtensionPayload {
  name: string;
  enabled: boolean;
  state: string;
  daemon_running: boolean;
  health?: string;
  capabilities?: string[];
  bundles?: Array<{ name: string; profiles?: string[] }>;
}

interface ExtensionResponse {
  extension: ExtensionPayload;
}

interface ExtensionsResponse {
  extensions: ExtensionPayload[];
}

interface ToolPayload {
  descriptor: {
    tool_id: string;
    backend: { kind: string; extension_id?: string; handler?: string };
    source: { kind: string; owner: string; resource_id?: string };
    visibility?: string;
    read_only: boolean;
    risk: string;
  };
  availability: { available: boolean; executable: boolean };
  decision: { approval_required: boolean; callable: boolean; visible_to_operator: boolean };
}

interface ToolsResponse {
  tools: ToolPayload[];
}

interface ToolInvokeResponse {
  tool_id: string;
  status: string;
  result: {
    content?: Array<{ type: string; text?: string }>;
    structured?: unknown;
    preview?: string;
    redactions?: Array<{ path: string; reason: string }>;
    truncated: boolean;
    bytes: number;
    duration_ms: number;
  };
  events: Array<{
    tool_id: string;
    source_kind?: string;
    source_owner?: string;
    redacted_input_fields?: string[];
    input_digest?: string;
    correlation_id?: string;
  }>;
}

interface ToolApprovalPayload {
  approval_token: string;
  input_digest: string;
  tool_id: string;
}

interface ToolApprovalResponse {
  approval: ToolApprovalPayload;
}

interface BundleCatalogResponse {
  bundles: Array<{
    extension_name: string;
    bundle_name: string;
    profiles?: Array<{ name: string }>;
  }>;
}

interface BundleActivation {
  id: string;
  extension_name: string;
  bundle_name: string;
  profile_name: string;
  scope: string;
  bind_primary_channel_as_default: boolean;
  channels?: Array<{ name: string }>;
  inventory?: Array<{ resource_kind: string; resource_name: string }>;
}

interface BundleActivationResponse {
  activation: BundleActivation;
}

interface BundleActivationsResponse {
  activations: BundleActivation[];
}

interface BundleNetworkSettingsResponse {
  network: {
    configured_default_channel?: string;
    effective_default_channel?: string;
    effective_default_source?: string;
    declared_channels?: Array<{ activation_id: string; name: string; primary: boolean }>;
  };
}

interface SettingsRestartAction {
  operation_id: string;
  status_url: string;
}

interface SettingsRestartStatus {
  status: string;
}

interface SessionEnvelope {
  session: {
    id: string;
    agent_name: string;
    state: string;
    workspace_id: string;
  };
}

interface SessionHistoryEnvelope {
  messages?: unknown[];
}

interface SessionEventEnvelope {
  events?: unknown[];
}

interface ExecFailure {
  stdout: string;
  stderr: string;
  code?: unknown;
}

test.use({
  runtimeOptions: {
    env: {
      ...process.env,
      AGH_BROWSER_EXTENSION_SECRET: extensionSecretSentinel,
    },
    readyTimeoutMs: 45_000,
    seed: {
      mockAgents: [
        {
          fixturePath: toolPermissionFixture,
          fixtureAgent: toolPermissionAgent,
        },
        {
          fixturePath: denyPermissionFixture,
          fixtureAgent: denyPermissionAgent,
        },
      ],
    },
    toolsExternalDefault: "enabled",
  },
});

test("operator installs a local extension tool provider, invokes it over transports, activates bundle resources, and verifies fail-closed manifest security", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  assertLaunchRuntime(runtime, "extensibility tool/resource lifecycle");

  const extensionDir = await createBrowserToolProviderExtension();
  const checksumFailureDir = await createChecksumFailureExtension();
  const badManifestDir = await createInvalidExtensionManifest();
  const ui = settingsOperatorSelectors(appPage);

  await useGlobalWorkspaceIfPrompted(appPage);

  const installed = await runBrowserRuntimeCLIJSON<ExtensionPayload>(runtime, [
    "extension",
    "install",
    "--allow-unverified",
    "--yes",
    extensionDir,
  ]);
  expect(projectExtension(installed)).toMatchObject({
    name: extensionName,
    enabled: true,
    state: "active",
  });

  await expect
    .poll(async () => {
      const payload = await runtime.requestJSON<ExtensionResponse>(
        `/api/extensions/${extensionName}`
      );
      return payload.extension.state;
    })
    .toBe("active");

  await appPage.goto(runtime.url("/settings/hooks-extensions"), { waitUntil: "domcontentloaded" });
  await expect(ui.hooksExtensions.page).toBeVisible({ timeout: 20_000 });
  await expect(ui.hooksExtensions.extensionToggle(extensionName)).toBeVisible();
  await expect(ui.hooksExtensions.extensionToggle(extensionName)).toBeChecked();

  const routeState = await captureRouteState(appPage);
  const viewportEvidence = await captureViewportEvidence({
    page: appPage,
    browserArtifacts,
    moduleName: "extensibility-tools-resources",
    assertVisible: async () => {
      await expect(ui.hooksExtensions.page).toBeVisible();
      await expect(ui.hooksExtensions.extensionToggle(extensionName)).toBeVisible();
    },
  });

  const httpExtension = await runtime.requestJSON<ExtensionResponse>(
    `/api/extensions/${extensionName}`
  );
  const udsExtension = await requestBrowserRuntimeOperatorJSON<ExtensionResponse>(
    runtime,
    `/api/extensions/${extensionName}`
  );
  const cliExtension = await runBrowserRuntimeCLIJSON<ExtensionPayload>(runtime, [
    "extension",
    "status",
    extensionName,
  ]);
  await captureBrowserTransportSnapshot(runtime, "TC-EXT-001-extension", {
    http: projectExtension(httpExtension.extension),
    uds: projectExtension(udsExtension.extension),
    cli: projectExtension(cliExtension),
  });

  const httpTools = await runtime.requestJSON<ToolsResponse>("/api/tools");
  const udsTools = await requestBrowserRuntimeOperatorJSON<ToolsResponse>(runtime, "/api/tools");
  const cliTools = await runBrowserRuntimeCLIJSON<ToolsResponse>(runtime, ["tool", "list"]);
  const httpTool = expectTool(httpTools, "HTTP tool registry");
  const udsTool = expectTool(udsTools, "UDS tool registry");
  const cliTool = expectTool(cliTools, "CLI tool registry");
  expect(projectTool(httpTool)).toEqual(projectTool(udsTool));
  expect(projectTool(cliTool)).toMatchObject(projectTool(httpTool));
  await captureBrowserTransportSnapshot(runtime, "TC-EXT-001-tool-registry", {
    http: projectTool(httpTool),
    uds: projectTool(udsTool),
    cli: projectTool(cliTool),
  });

  const safeInput = { query: "release-readiness" };
  const sensitiveInput = { query: "browser-extension-api-key=secret-13" };
  const sessionID = "browser-extensibility-session";
  const httpApproval = await createToolApprovalHTTP(runtime, safeInput, sessionID);
  const udsApproval = await createToolApprovalUDS(runtime, safeInput, sessionID);
  const cliApproval = await runBrowserRuntimeCLIJSON<ToolApprovalResponse>(runtime, [
    "tool",
    "approve",
    toolID,
    "--session",
    sessionID,
    "--input",
    JSON.stringify(sensitiveInput),
  ]);
  expect(httpApproval.approval.input_digest).toBe(udsApproval.approval.input_digest);
  expect(cliApproval.approval.tool_id).toBe(toolID);

  const httpInvoke = await invokeToolHTTP(runtime, safeInput, "tc-ext-001-http");
  const udsInvoke = await invokeToolUDS(runtime, safeInput, "tc-ext-001-uds");
  const cliInvoke = await runBrowserRuntimeCLIJSON<ToolInvokeResponse>(runtime, [
    "tool",
    "invoke",
    toolID,
    "--input",
    JSON.stringify(sensitiveInput),
    "--sensitive-input-field",
    "query",
    "--correlation-id",
    "tc-ext-001-cli",
  ]);

  expectInvokeSucceeded(httpInvoke, "release-readiness");
  expectInvokeSucceeded(udsInvoke, "release-readiness");
  expect(cliInvoke.result.preview).not.toContain("browser-extension-api-key");
  expect(JSON.stringify(cliInvoke)).not.toContain(sensitiveInput.query);
  expect(JSON.stringify(cliInvoke)).not.toMatch(sensitiveArtifactPattern);
  expect(cliInvoke.result.redactions?.some(redaction => redaction.path === "query")).toBe(true);

  await captureBrowserTransportSnapshot(runtime, "TC-EXT-001-tool-invoke", {
    http: projectInvoke(httpInvoke),
    uds: projectInvoke(udsInvoke),
    cli: projectInvoke(cliInvoke),
  });

  const httpCatalog = await runtime.requestJSON<BundleCatalogResponse>("/api/bundles/catalog");
  const udsCatalog = await requestBrowserRuntimeOperatorJSON<BundleCatalogResponse>(
    runtime,
    "/api/bundles/catalog"
  );
  const cliCatalog = await runBrowserRuntimeCLIJSON<BundleCatalogResponse>(runtime, [
    "bundle",
    "catalog",
  ]);
  expectBundleCatalog(httpCatalog, "HTTP bundle catalog");
  expectBundleCatalog(udsCatalog, "UDS bundle catalog");
  expectBundleCatalog(cliCatalog, "CLI bundle catalog");

  const activationRequest = JSON.stringify({
    extension_name: extensionName,
    bundle_name: bundleName,
    profile_name: bundleProfile,
    scope: "global",
    bind_primary_channel_as_default: true,
  });
  const httpPreview = await runtime.requestJSON<BundleActivationResponse>("/api/bundles/preview", {
    method: "POST",
    body: activationRequest,
  });
  expectBundleActivation(httpPreview.activation, "HTTP bundle preview");

  const cliPreview = await runBrowserRuntimeCLIJSON<BundleActivation>(runtime, [
    "bundle",
    "preview",
    "--extension",
    extensionName,
    "--bundle",
    bundleName,
    "--profile",
    bundleProfile,
    "--bind-primary-channel-as-default",
  ]);
  expectBundleActivation(cliPreview, "CLI bundle preview");

  const httpActivation = await runtime.requestJSON<BundleActivationResponse>(
    "/api/bundles/activations",
    {
      method: "POST",
      body: activationRequest,
    }
  );
  expectBundleActivation(httpActivation.activation, "HTTP bundle activation");

  const activationID = httpActivation.activation.id;
  const udsActivation = await requestBrowserRuntimeOperatorJSON<BundleActivationResponse>(
    runtime,
    `/api/bundles/activations/${encodeURIComponent(activationID)}`
  );
  const cliActivation = await runBrowserRuntimeCLIJSON<BundleActivation>(runtime, [
    "bundle",
    "get",
    activationID,
  ]);
  expectBundleActivation(udsActivation.activation, "UDS bundle activation");
  expectBundleActivation(cliActivation, "CLI bundle activation");

  expect(httpActivation.activation.channels?.length ?? 0).toBeGreaterThan(0);
  expect(udsActivation.activation.channels?.length ?? 0).toBeGreaterThan(0);
  expect(cliActivation.channels?.length ?? 0).toBeGreaterThan(0);

  const networkHTTP = await runtime.requestJSON<BundleNetworkSettingsResponse>(
    "/api/bundles/network/settings"
  );
  const networkUDS = await requestBrowserRuntimeOperatorJSON<BundleNetworkSettingsResponse>(
    runtime,
    "/api/bundles/network/settings"
  );
  const networkCLI = await runBrowserRuntimeCLIJSON<BundleNetworkSettingsResponse["network"]>(
    runtime,
    ["bundle", "network-settings"]
  );
  expect(networkHTTP.network.effective_default_channel).toBe(bundleChannel);
  expect(networkUDS.network.effective_default_channel).toBe(bundleChannel);
  expect(networkCLI.effective_default_channel).toBe(bundleChannel);

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
  await expect
    .poll(async () => await pollRestartStatus(runtime, restart.status_url), {
      timeout: 45_000,
    })
    .toBe("ready");
  await appPage.reload({ waitUntil: "domcontentloaded" });
  await expect(ui.hooksExtensions.page).toBeVisible({ timeout: 20_000 });
  await expect(ui.hooksExtensions.extensionToggle(extensionName)).toBeChecked();
  await expect
    .poll(async () => {
      const payload = await runtime.requestJSON<ExtensionResponse>(
        `/api/extensions/${extensionName}`
      );
      return payload.extension.state;
    })
    .toBe("active");
  const postRestartTool = expectTool(
    await runtime.requestJSON<ToolsResponse>("/api/tools"),
    "HTTP tool registry after restart"
  );
  expect(projectTool(postRestartTool)).toMatchObject(projectTool(httpTool));
  const postRestartInvoke = await invokeToolHTTP(
    runtime,
    { query: "post-restart-readiness" },
    "tc-ext-001-restart"
  );
  expectInvokeSucceeded(postRestartInvoke, "post-restart-readiness");

  const crashFailure = await requestToolInvokeFailure(appPage, runtime, "crash-once");
  expect(crashFailure.status).toBeGreaterThanOrEqual(500);
  expect(crashFailure.body).toMatch(/extension|tool|process|provider|closed|failed/i);
  await expect
    .poll(
      async () => {
        try {
          const recovered = await invokeToolHTTP(
            runtime,
            { query: "recovered-after-crash" },
            "tc-ext-001-crash-recovery"
          );
          return recovered.result.preview ?? "";
        } catch {
          return "";
        }
      },
      { timeout: 30_000 }
    )
    .toContain("recovered-after-crash");

  const updatedActivation = await runBrowserRuntimeCLIJSON<BundleActivation>(runtime, [
    "bundle",
    "update",
    activationID,
    "--clear-primary-channel-default",
  ]);
  expect(updatedActivation.bind_primary_channel_as_default).toBe(false);

  const failureHTTP = await appPage.request.post(runtime.url("/api/extensions"), {
    data: { path: checksumFailureDir, checksum: "sha256:bad-checksum" },
  });
  expect(failureHTTP.status()).toBe(422);
  const checksumFailureBody = (await failureHTTP.json()) as { error?: unknown };
  expect(checksumFailureBody).toHaveProperty("error");
  expect(JSON.stringify(checksumFailureBody.error)).toMatch(/checksum|mismatch/i);

  const invalidInstall = await runCLIExpectFailure(runtime.paths, [
    "extension",
    "install",
    badManifestDir,
  ]);
  expect(`${invalidInstall.stdout}\n${invalidInstall.stderr}`).toMatch(
    /manifest|capabilities|invalid/i
  );

  await runBrowserRuntimeCLIJSON<{ deactivated: string }>(runtime, [
    "bundle",
    "deactivate",
    activationID,
  ]);
  const remainingActivations = await runtime.requestJSON<BundleActivationsResponse>(
    "/api/bundles/activations"
  );
  expect(remainingActivations.activations.some(activation => activation.id === activationID)).toBe(
    false
  );

  const extensionList = await runtime.requestJSON<ExtensionsResponse>("/api/extensions");
  const daemonLog = await readFile(runtime.paths.daemonLog, "utf8");
  const finalArtifacts = {
    route_state: routeState,
    viewport_evidence: viewportEvidence,
    extensions: extensionList.extensions.map(projectExtension),
    bundle_catalog: {
      http: projectBundleCatalog(httpCatalog),
      uds: projectBundleCatalog(udsCatalog),
      cli: projectBundleCatalog(cliCatalog),
    },
    bundle_activation: {
      http: projectBundleActivation(httpActivation.activation),
      uds: projectBundleActivation(udsActivation.activation),
      cli: projectBundleActivation(cliActivation),
      updated: projectBundleActivation(updatedActivation),
    },
    resources: {
      http_inventory: httpActivation.activation.inventory ?? [],
      uds_inventory: udsActivation.activation.inventory ?? [],
      cli_inventory: cliActivation.inventory ?? [],
    },
    network: {
      http: networkHTTP.network,
      uds: networkUDS.network,
      cli: networkCLI,
    },
    fail_closed: {
      checksum_status: failureHTTP.status(),
      checksum_error: checksumFailureBody.error,
      invalid_manifest_stderr: invalidInstall.stderr.slice(0, 500),
    },
    disruption: {
      crash_status: crashFailure.status,
      post_restart: projectInvoke(postRestartInvoke),
    },
  };
  expect(daemonLog).not.toContain(extensionSecretSentinel);
  expect(JSON.stringify(finalArtifacts)).not.toContain(extensionSecretSentinel);
  assertNoSensitiveArtifactPayload([finalArtifacts, daemonLog]);
  await runtime.artifactCollector.captureJSON("browser_api_snapshots", finalArtifacts);
  await runtime.artifactCollector.captureJSON("browser_route_state", routeState);
  await browserArtifacts.persist(appPage);
});

test("operator approves and denies tool permission prompts with accessible browser controls and transport evidence", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  assertLaunchRuntime(runtime, "extensibility permission prompt lifecycle");
  const workspace = await prepareExtensibilitySessionRuntime(runtime, appPage);

  const approved = await createSession(runtime, toolPermissionAgent, workspace.id);
  await appPage.goto(runtime.url(sessionPath(toolPermissionAgent, approved.session.id)), {
    waitUntil: "domcontentloaded",
  });
  const ui = sessionLifecycleSelectors(appPage);
  await expect(ui.chatHeader).toBeVisible();
  await ui.composerTextarea.fill("exercise golden");
  await ui.composerTextarea.press("Enter");
  await expect(ui.chatView).toContainText("hello from golden");
  await expect(ui.permissionPrompt).toBeVisible();
  await expect(appPage.getByText("Permission Required")).toBeVisible();
  await expect(appPage.getByRole("button", { name: /allow always/i })).toBeVisible();
  await expect(appPage.getByRole("button", { name: /reject always/i })).toBeVisible();
  await assertPermissionKeyboardPath(appPage);

  const approveResponsePromise = appPage.waitForResponse(
    response =>
      response.request().method() === "POST" &&
      response.url().endsWith(sessionAPIPath(workspace.id, approved.session.id, "/approve"))
  );
  await appPage.getByTestId("permission-allow-always").click();
  expect((await approveResponsePromise).ok()).toBe(true);
  await expect(ui.permissionPrompt).toBeHidden();
  const approvedSnapshot = await captureSessionSnapshot(runtime, workspace.id, approved.session.id);
  expect(JSON.stringify(approvedSnapshot.events)).toContain("allow-always");

  const denied = await createSession(runtime, denyPermissionAgent, workspace.id);
  await appPage.goto(runtime.url(sessionPath(denyPermissionAgent, denied.session.id)), {
    waitUntil: "domcontentloaded",
  });
  await expect(ui.chatHeader).toBeVisible();
  await ui.composerTextarea.fill("exercise permission hardening");
  await ui.composerTextarea.press("Enter");
  await expect(ui.chatView).toContainText("Permission hardening started.");
  await expect(ui.permissionPrompt).toBeVisible();
  await expect(appPage.getByTestId("permission-tool-input")).toContainText("hardening.txt");

  const rejectResponsePromise = appPage.waitForResponse(
    response =>
      response.request().method() === "POST" &&
      response.url().endsWith(sessionAPIPath(workspace.id, denied.session.id, "/approve"))
  );
  await appPage.getByTestId("permission-reject-always").click();
  expect((await rejectResponsePromise).ok()).toBe(true);
  await expect(ui.permissionPrompt).toBeHidden();
  await expect(appPage.getByTestId("permission-rejected-notice")).toContainText(
    "Permission Rejected"
  );
  const deniedSnapshot = await captureSessionSnapshot(runtime, workspace.id, denied.session.id);
  expect(JSON.stringify(deniedSnapshot.events)).toContain("perm-hardening-reject-1");
  expect(JSON.stringify(deniedSnapshot.events)).toContain("reject-always");

  await runtime.artifactCollector.captureJSON("browser_api_snapshots", {
    approved: approvedSnapshot,
    denied: deniedSnapshot,
  });
  await browserArtifacts.captureScreenshot("extensibility-tool-permission-approve-deny", appPage);
  const manifest = await browserArtifacts.persist(appPage);
  expect(manifest.artifacts).toEqual(
    expect.arrayContaining([
      expect.objectContaining({ kind: "browser_api_snapshots" }),
      expect.objectContaining({ kind: "browser_route_state" }),
      expect.objectContaining({ kind: "browser_screenshots" }),
    ])
  );
});

function assertLaunchRuntime(
  runtime: BrowserRuntime,
  label: string
): asserts runtime is BrowserRuntime & {
  paths: RuntimePaths;
} {
  if (!runtime.paths) {
    throw new Error(`${label} requires launch-mode runtime paths`);
  }
}

function canonicalizeJSON(value: unknown): string {
  if (value === null || typeof value !== "object") {
    return JSON.stringify(value);
  }
  if (Array.isArray(value)) {
    return `[${value.map(canonicalizeJSON).join(",")}]`;
  }
  const record = value as Record<string, unknown>;
  return `{${Object.keys(record)
    .sort()
    .map(key => `${JSON.stringify(key)}:${canonicalizeJSON(record[key])}`)
    .join(",")}}`;
}

function schemaDigest(schema: unknown): string {
  return createHash("sha256").update(canonicalizeJSON(schema)).digest("hex");
}

async function createBrowserToolProviderExtension(): Promise<string> {
  const rootDir = await mkdtemp(path.join(os.tmpdir(), "agh-browser-tool-provider-"));
  const bundlesDir = path.join(rootDir, "bundles");
  await mkdir(bundlesDir, { recursive: true });

  const inputSchema = {
    type: "object",
    required: ["query"],
    properties: {
      query: { type: "string" },
    },
  };

  await writeFile(
    path.join(rootDir, "extension.json"),
    JSON.stringify(
      {
        extension: {
          name: extensionName,
          version: "0.1.0",
          description: "Browser E2E extension tool provider",
          min_agh_version: "0.0.0",
        },
        capabilities: { provides: ["tool.provider"] },
        subprocess: {
          command: "node",
          args: ["extension.js"],
        },
        resources: {
          bundles: ["bundles"],
          tools: {
            search: {
              id: toolID,
              display_title: "Browser extension search",
              description: "Search extension-owned browser evidence",
              read_only: true,
              visibility: "session",
              risk: "read",
              max_result_bytes: 4096,
              backend: { kind: "extension_host", handler: "search" },
              input_schema: inputSchema,
              toolsets: ["agh__catalog"],
            },
          },
        },
      },
      null,
      2
    ),
    "utf8"
  );

  await writeFile(
    path.join(bundlesDir, "browser-extensibility.toml"),
    `
name = "${bundleName}"
description = "Browser E2E bundle proving extension resource projection"

[[profiles]]
name = "${bundleProfile}"
description = "Default extensibility profile"

[profiles.channels]
primary = "${bundleChannel}"

[[profiles.channels.items]]
name = "${bundleChannel}"
description = "Bundle-projected channel for E2E"
`.trimStart(),
    "utf8"
  );

  await writeFile(
    path.join(rootDir, "extension.js"),
    extensionRuntimeSource(schemaDigest(inputSchema)),
    "utf8"
  );

  return rootDir;
}

async function createInvalidExtensionManifest(): Promise<string> {
  const rootDir = await mkdtemp(path.join(os.tmpdir(), "agh-invalid-extension-"));
  await writeFile(
    path.join(rootDir, "extension.json"),
    JSON.stringify(
      {
        extension: {
          name: "browser-invalid-extension",
          version: "0.1.0",
          description: "Invalid browser E2E extension",
          min_agh_version: "0.0.0",
        },
        capabilities: { provides: ["bad capability"] },
      },
      null,
      2
    ),
    "utf8"
  );
  return rootDir;
}

async function createChecksumFailureExtension(): Promise<string> {
  const rootDir = await mkdtemp(path.join(os.tmpdir(), "agh-checksum-failure-extension-"));
  await writeFile(
    path.join(rootDir, "extension.json"),
    JSON.stringify(
      {
        extension: {
          name: "browser-checksum-reject",
          version: "0.1.0",
          description: "Valid extension used to prove checksum rejection",
          min_agh_version: "0.0.0",
        },
        capabilities: { provides: [] },
      },
      null,
      2
    ),
    "utf8"
  );
  return rootDir;
}

function extensionRuntimeSource(inputSchemaDigest: string): string {
  return `
const readline = require("node:readline");

let initialized = false;

const rl = readline.createInterface({ input: process.stdin });

function sendResult(id, result) {
  process.stdout.write(JSON.stringify({ jsonrpc: "2.0", id, result }) + "\\n");
}

function sendError(id, code, message) {
  process.stdout.write(JSON.stringify({ jsonrpc: "2.0", id, error: { code, message } }) + "\\n");
}

rl.on("line", line => {
  if (!line.trim()) {
    return;
  }
  let frame;
  try {
    frame = JSON.parse(line);
  } catch {
    sendError(null, -32700, "Parse error");
    return;
  }
  const id = frame.id ?? null;
  if (frame.method === "initialize") {
    initialized = true;
    sendResult(id, {
      protocol_version: "1",
      extension_info: {
        name: "${extensionName}",
        version: "0.1.0",
        sdk_name: "browser-e2e-fixture",
        sdk_version: "0.1.0"
      },
      accepted_capabilities: { provides: ["tool.provider"], actions: [], security: [] },
      implemented_methods: ["health_check", "provide_tools", "tools/call", "shutdown"],
      supported_hook_events: [],
      supports: { health_check: true }
    });
    return;
  }
  if (!initialized) {
    sendError(id, -32003, "Not initialized");
    return;
  }
  if (frame.method === "health_check") {
    sendResult(id, { healthy: true, message: "", details: {} });
    return;
  }
  if (frame.method === "provide_tools") {
    sendResult(id, {
      tools: [{
        id: "${toolID}",
        handler: "search",
        input_schema_digest: "${inputSchemaDigest}",
        read_only: true,
        risk: "read",
        capabilities: []
      }]
    });
    return;
  }
  if (frame.method === "tools/call") {
    const params = frame.params || {};
    const input = params.input || {};
    const query = typeof input.query === "string" ? input.query : "";
    if (query === "crash-once") {
      process.stderr.write("browser extension crash-once disruption probe\\n");
      process.exit(42);
      return;
    }
    const containsSecret = /secret|token|api[_-]?key/i.test(query);
    const safeQuery = containsSecret ? "[redacted]" : query;
    sendResult(id, {
      result: {
        content: [{ type: "text", text: "result " + safeQuery }],
        structured: { query: safeQuery, extension: "${extensionName}" },
        preview: "result " + safeQuery,
        redactions: containsSecret ? [{ path: "query", reason: "secret_metadata" }] : [],
        truncated: false,
        bytes: Buffer.byteLength(safeQuery),
        duration_ms: 0
      }
    });
    return;
  }
  if (frame.method === "shutdown") {
    sendResult(id, {});
    process.exit(0);
  }
  sendError(id, -32601, "Method not found");
});
`;
}

async function requestToolInvokeFailure(
  page: Page,
  runtime: BrowserRuntime,
  query: string
): Promise<{ status: number; body: string }> {
  const response = await page.request.post(runtime.url(`/api/tools/${toolID}/invoke`), {
    data: {
      input: { query },
      sensitive_input_fields: ["query"],
      correlation_id: "tc-ext-001-crash",
    },
    timeout: 15_000,
  });
  const body = await response.text();
  if (response.ok()) {
    throw new Error(`tool crash probe unexpectedly succeeded: ${body}`);
  }
  return { status: response.status(), body };
}

async function invokeToolHTTP(
  runtime: BrowserRuntime,
  input: Record<string, string>,
  correlationID: string
): Promise<ToolInvokeResponse> {
  return await runtime.requestJSON<ToolInvokeResponse>(`/api/tools/${toolID}/invoke`, {
    method: "POST",
    body: JSON.stringify({
      input,
      sensitive_input_fields: ["query"],
      correlation_id: correlationID,
    }),
  });
}

async function invokeToolUDS(
  runtime: BrowserRuntime,
  input: Record<string, string>,
  correlationID: string
): Promise<ToolInvokeResponse> {
  return await requestBrowserRuntimeOperatorJSON<ToolInvokeResponse>(
    runtime,
    `/api/tools/${toolID}/invoke`,
    {
      method: "POST",
      body: JSON.stringify({
        input,
        sensitive_input_fields: ["query"],
        correlation_id: correlationID,
      }),
    }
  );
}

async function createToolApprovalHTTP(
  runtime: BrowserRuntime,
  input: Record<string, string>,
  sessionID: string
): Promise<ToolApprovalResponse> {
  return await runtime.requestJSON<ToolApprovalResponse>(`/api/tools/${toolID}/approvals`, {
    method: "POST",
    body: JSON.stringify({
      session_id: sessionID,
      input,
    }),
  });
}

async function createToolApprovalUDS(
  runtime: BrowserRuntime,
  input: Record<string, string>,
  sessionID: string
): Promise<ToolApprovalResponse> {
  return await requestBrowserRuntimeOperatorJSON<ToolApprovalResponse>(
    runtime,
    `/api/tools/${toolID}/approvals`,
    {
      method: "POST",
      body: JSON.stringify({
        session_id: sessionID,
        input,
      }),
    }
  );
}

async function pollRestartStatus(runtime: BrowserRuntime, statusURL: string): Promise<string> {
  try {
    return (await runtime.requestJSON<SettingsRestartStatus>(statusURL)).status;
  } catch {
    return "restarting";
  }
}

async function prepareExtensibilitySessionRuntime(
  runtime: BrowserRuntime,
  page: Page
): Promise<WorkspacePayload> {
  const workspace = await runtime.resolveWorkspace(runtime.paths?.homeDir ?? process.cwd());
  await page.goto(runtime.url("/"), { waitUntil: "domcontentloaded" });
  await useGlobalWorkspaceIfPrompted(page);
  return workspace;
}

function sessionPath(agentName: string, sessionID: string): string {
  return `/agents/${agentName}/sessions/${sessionID}`;
}

async function createSession(
  runtime: BrowserRuntime,
  agentName: string,
  workspaceID: string
): Promise<SessionEnvelope> {
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
  return payload;
}

async function captureSessionSnapshot(
  runtime: BrowserRuntime,
  workspaceID: string,
  sessionID: string
): Promise<{
  events: SessionEventEnvelope;
  history: SessionHistoryEnvelope;
  session: SessionEnvelope;
  udsSession?: SessionEnvelope;
}> {
  const basePath = sessionAPIPath(workspaceID, sessionID);
  const snapshot = {
    events: await runtime.requestJSON<SessionEventEnvelope>(`${basePath}/events`),
    history: await runtime.requestJSON<SessionHistoryEnvelope>(`${basePath}/history`),
    session: await runtime.requestJSON<SessionEnvelope>(basePath),
    udsSession: runtime.requestOperatorJSON
      ? await runtime.requestOperatorJSON<SessionEnvelope>(basePath)
      : undefined,
  };
  if (snapshot.udsSession) {
    expect(snapshot.udsSession.session.id).toBe(snapshot.session.session.id);
    expect(snapshot.udsSession.session.state).toBe(snapshot.session.session.state);
  }
  return snapshot;
}

function sessionAPIPath(workspaceID: string, sessionID: string, suffix = ""): string {
  return `/api/workspaces/${encodeURIComponent(workspaceID)}/sessions/${encodeURIComponent(
    sessionID
  )}${suffix}`;
}

async function assertPermissionKeyboardPath(page: Page): Promise<void> {
  const focusOrder = [
    "permission-allow-once",
    "permission-allow-always",
    "permission-reject-once",
    "permission-reject-always",
  ];
  await page.getByTestId(focusOrder[0]).focus();
  for (const testID of focusOrder) {
    await expect(page.locator(`[data-testid="${testID}"]`)).toBeFocused();
    if (testID !== focusOrder.at(-1)) {
      await page.keyboard.press("Tab");
    }
  }
}

function expectTool(response: ToolsResponse, label: string): ToolPayload {
  const tool = response.tools.find(item => item.descriptor.tool_id === toolID);
  if (!tool) {
    throw new Error(`${label} did not expose ${toolID}`);
  }
  expect(tool.availability.available).toBe(true);
  expect(tool.availability.executable).toBe(true);
  expect(tool.decision.visible_to_operator).toBe(true);
  if (!tool.decision.callable) {
    throw new Error(`${label} exposed a non-callable tool: ${JSON.stringify(projectTool(tool))}`);
  }
  expect(tool.descriptor.backend).toMatchObject({
    kind: "extension_host",
    extension_id: extensionName,
    handler: "search",
  });
  expect(tool.descriptor.source).toMatchObject({
    kind: "extension",
    owner: extensionName,
  });
  return tool;
}

function expectInvokeSucceeded(response: ToolInvokeResponse, query: string): void {
  expect(response.tool_id).toBe(toolID);
  expect(response.status).toBe("completed");
  expect(response.result.preview).toContain(query);
  expect(response.result.truncated).toBe(false);
}

function expectBundleCatalog(response: BundleCatalogResponse, label: string): void {
  const entry = response.bundles.find(
    item => item.extension_name === extensionName && item.bundle_name === bundleName
  );
  if (!entry) {
    throw new Error(`${label} did not include ${extensionName}/${bundleName}`);
  }
  expect(entry.profiles?.some(profile => profile.name === bundleProfile)).toBe(true);
}

function expectBundleActivation(activation: BundleActivation, label: string): void {
  expect(activation.extension_name, label).toBe(extensionName);
  expect(activation.bundle_name, label).toBe(bundleName);
  expect(activation.profile_name, label).toBe(bundleProfile);
  expect(activation.scope, label).toBe("global");
  expect(
    activation.channels?.some(channel => channel.name === bundleChannel),
    label
  ).toBe(true);
}

function projectExtension(extension: ExtensionPayload): Record<string, unknown> {
  return {
    name: extension.name,
    enabled: extension.enabled,
    state: extension.state,
    daemon_running: extension.daemon_running,
    health: extension.health,
    capabilities: extension.capabilities ?? [],
    bundles: extension.bundles ?? [],
  };
}

function projectTool(tool: ToolPayload): Record<string, unknown> {
  return {
    id: tool.descriptor.tool_id,
    backend: tool.descriptor.backend,
    source: tool.descriptor.source,
    visibility: tool.descriptor.visibility,
    read_only: tool.descriptor.read_only,
    risk: tool.descriptor.risk,
    available: tool.availability.available,
    executable: tool.availability.executable,
    decision: tool.decision,
  };
}

function projectInvoke(response: ToolInvokeResponse): Record<string, unknown> {
  return {
    tool_id: response.tool_id,
    status: response.status,
    preview: response.result.preview,
    truncated: response.result.truncated,
    redactions: response.result.redactions ?? [],
    event_count: response.events.length,
    events: response.events.map(event => ({
      tool_id: event.tool_id,
      source_kind: event.source_kind,
      source_owner: event.source_owner,
      redacted_input_fields: event.redacted_input_fields ?? [],
      has_input_digest: Boolean(event.input_digest),
      correlation_id: event.correlation_id,
    })),
  };
}

function projectBundleCatalog(response: BundleCatalogResponse): Record<string, unknown> {
  return {
    bundles: response.bundles
      .filter(item => item.extension_name === extensionName)
      .map(item => ({
        extension_name: item.extension_name,
        bundle_name: item.bundle_name,
        profiles: item.profiles?.map(profile => profile.name) ?? [],
      })),
  };
}

function projectBundleActivation(activation: BundleActivation): Record<string, unknown> {
  return {
    id: activation.id,
    extension_name: activation.extension_name,
    bundle_name: activation.bundle_name,
    profile_name: activation.profile_name,
    scope: activation.scope,
    bind_primary_channel_as_default: activation.bind_primary_channel_as_default,
    channels: activation.channels?.map(channel => channel.name) ?? [],
    inventory: activation.inventory ?? [],
  };
}

async function runCLIExpectFailure(paths: RuntimePaths, args: string[]): Promise<ExecFailure> {
  const finalArgs =
    args.includes("-o") || args.includes("--output") ? args : [...args, "-o", "json"];
  try {
    const result = await execFileAsync(paths.cliShim, finalArgs, {
      env: browserRuntimeCLIEnv(paths),
      maxBuffer: 10 * 1024 * 1024,
      timeout: 30_000,
    });
    throw new Error(`CLI command unexpectedly succeeded: ${result.stdout}`);
  } catch (error) {
    const failure = error as Error & {
      stdout?: string | Buffer;
      stderr?: string | Buffer;
      code?: unknown;
    };
    if (failure.message.startsWith("CLI command unexpectedly succeeded")) {
      throw failure;
    }
    return {
      stdout: String(failure.stdout ?? ""),
      stderr: String(failure.stderr ?? ""),
      code: failure.code,
    };
  }
}

function browserRuntimeCLIEnv(paths: RuntimePaths): NodeJS.ProcessEnv {
  return {
    ...process.env,
    AGH_E2E_CLI_BIN: paths.cliShim,
    AGH_HOME: paths.homeDir,
    HOME: paths.homeDir,
    PATH: [path.dirname(paths.cliShim), process.env.PATH ?? ""]
      .filter(Boolean)
      .join(path.delimiter),
  };
}
