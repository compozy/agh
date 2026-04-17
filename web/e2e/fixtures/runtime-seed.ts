import { execFile } from "node:child_process";
import { cp, mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";
import { promisify } from "node:util";

import type {
  AutomationJob,
  AutomationRun,
  AutomationTrigger,
  CreateAutomationJobRequest,
  CreateAutomationTriggerRequest,
} from "@/systems/automation";
import type {
  BridgeDetailResponse,
  BridgeHealth,
  BridgeProvider,
  BridgeRoute,
  BridgeSummary,
} from "@/systems/bridges";

const execFileAsync = promisify(execFile);

export interface WorkspacePayload {
  id: string;
  root_dir: string;
  name: string;
}

export interface SeededSessionPayload {
  id: string;
  agent_name: string;
  workspace_id: string;
  state: string;
  name?: string | null;
}

export interface BrowserMockAgentSeed {
  fixturePath: string;
  fixtureAgent?: string;
  agentName?: string;
}

export interface BrowserWorkspaceSeed {
  rootDir: string;
}

export interface BrowserSessionSeed {
  agentName: string;
  workspaceRootDir?: string;
}

export interface BrowserRuntimeSeed {
  mockAgents?: BrowserMockAgentSeed[];
  workspace?: BrowserWorkspaceSeed;
  session?: BrowserSessionSeed;
}

export interface BrowserAutomationOperatorFlowSeed {
  agentName: string;
  timeoutMs?: number;
}

export interface BridgeAdapterMarkerPaths {
  crashOnce: string;
  delivery: string;
  handshake: string;
  ingest: string;
  ownership: string;
  shutdown: string;
  starts: string;
  state: string;
  updates: string;
}

interface PreparedBrowserBridgeExtension {
  checksum: string;
  extensionDir: string;
  markers: BridgeAdapterMarkerPaths;
}

export interface BrowserBridgeOperatorFlowSeed {
  displayName?: string;
  prepareExtension?: () => Promise<{
    checksum: string;
    extensionDir: string;
    markers: BridgeAdapterMarkerPaths;
  }>;
  timeoutMs?: number;
}

export interface BrowserBridgeOperatorFlowResult {
  bridge: BridgeSummary;
  extension: {
    checksum: string;
    dir: string;
    markers: BridgeAdapterMarkerPaths;
    name: string;
    platform: string;
  };
  health: BridgeHealth;
  provider: BridgeProvider;
}

type BrowserBridgeOperatorSeedRuntime = Pick<
  BrowserRuntimeSeedClient,
  "requestJSON" | "requestOperatorJSON"
> &
  Partial<Pick<BrowserRuntimeSeedClient, "resolveWorkspace">> & {
    paths?: {
      homeDir: string;
    };
    seeded?: BrowserRuntimeSeedResult;
  };

export interface BrowserBridgeIngressSeed {
  assistantText?: string;
  messageId?: number;
  text?: string;
  timeoutMs?: number;
  updateId?: number;
}

export interface BrowserBridgeIngressResult {
  routes: BridgeRoute[];
  sessionId: string;
  transcript: string;
}

export interface BrowserAutomationOperatorFlowResult {
  job: AutomationJob;
  trigger: AutomationTrigger;
  baselineRun: AutomationRun;
}

export interface BrowserNetworkOperatorFlowSeed {
  channel: string;
  initiatorAgentName: string;
  responderAgentName: string;
  timeoutMs?: number;
}

export interface BrowserNetworkOperatorFlowParticipant extends SeededSessionPayload {
  peerId: string;
}

export interface BrowserNetworkOperatorFlowResult {
  channel: string;
  initiator: BrowserNetworkOperatorFlowParticipant;
  responder: BrowserNetworkOperatorFlowParticipant;
  messageIds: typeof browserNetworkOperatorFlowScenario.messageIds;
}

export interface BrowserRuntimeSeedResult {
  workspace?: WorkspacePayload;
  session?: SeededSessionPayload;
}

export interface BrowserRuntimeSeedClient {
  requestJSON<T>(pathname: string, init?: RequestInit): Promise<T>;
  requestOperatorJSON?<T>(pathname: string, init?: RequestInit): Promise<T>;
  resolveWorkspace(rootDir: string): Promise<WorkspacePayload>;
}

interface BrowserRuntimeSeedPaths {
  homeDir: string;
  repoRoot: string;
}

interface MockFixtureAgent {
  name: string;
  provider: string;
  model?: string;
  permissions?: string;
  prompt?: string;
}

interface MockFixture {
  agents?: MockFixtureAgent[];
}

interface NetworkChannelSeedPayload {
  channel: string;
  sessions?: SeededSessionPayload[];
}

interface NetworkPeerSeedPayload {
  channel: string;
  peer_id: string;
  session_id?: string;
}

interface NetworkMessageSeedPayload {
  message_id: string;
}

interface NetworkStatusSeedPayload {
  network?: {
    kind_metrics?: Array<{
      kind?: string;
      sent?: number;
      delivered?: number;
    }>;
  };
}

const NETWORK_OPERATOR_FLOW_TIMEOUT_MS = 15_000;
const AUTOMATION_OPERATOR_FLOW_TIMEOUT_MS = 15_000;
const BRIDGE_OPERATOR_FLOW_TIMEOUT_MS = 20_000;
const BROWSER_SEED_POLL_MS = 150;
const BRIDGE_EXTENSION_NAME = "telegram-reference";
const BRIDGE_PLATFORM = "telegram";

export const browserNetworkOperatorFlowScenario = {
  messageIds: {
    say: "browser_msg_say_01",
    direct: "browser_msg_direct_01",
    trace: "browser_msg_trace_01",
  },
  texts: {
    say: "Who can take the failing migration tests in internal/store/sessiondb?",
    direct: "I can take the failing migration tests and send back a patch summary.",
    trace: "Patch prepared and local tests now pass.",
  },
  interactionId: "browser_int_patch_42",
  traceId: "browser_trace_ops_patch_42",
} as const;

export const browserAutomationOperatorFlowScenario = {
  job: {
    initialName: "deploy-review",
    editedName: "deploy-review-updated",
    prompt: "Review payload deploy for main",
    scheduleExpr: "0 9 * * *",
    updatedScheduleExpr: "15 10 * * *",
  },
  trigger: {
    endpointSlug: "browser-deploy-review",
    event: "webhook",
    name: "deploy-review-webhook",
    prompt: `Review payload {{ index .Data "payload" }} for {{ index .Data "branch" }}`,
    webhookID: "wbh_browser_deploy_review",
    webhookSecret: "shared-secret",
  },
  transcript: {
    assistant: "Automation review completed for deploy on main.",
  },
} as const;

export const browserBridgeOperatorFlowScenario = {
  bridge: {
    initialName: "Telegram Browser Bridge",
    initialProviderConfig: {
      mode: "bot",
      webhook_url: "https://example.test/browser-bridge",
    },
    editedName: "Telegram Bridge Ops",
    editedProviderConfig: {
      mode: "bot",
      webhook_url: "https://example.test/browser-bridge-updated",
    },
  },
  ingress: {
    assistant: "Bridge summary: initial route handled.",
    messageId: 321,
    text: "Need a runtime bridge summary",
    updateId: 94001,
  },
  secretBinding: {
    envName: "AGH_TEST_TELEGRAM_TOKEN",
    name: "bot_token",
  },
  testDelivery: {
    message: "Deliver a short operator ping.",
    mode: "direct-send",
    peerId: "telegram-peer-321",
    threadId: "654",
  },
} as const;

export async function seedBrowserRuntimeHome(
  paths: BrowserRuntimeSeedPaths,
  seed: BrowserRuntimeSeed | undefined
): Promise<void> {
  const mockAgents = seed?.mockAgents ?? [];
  if (mockAgents.length === 0) {
    return;
  }

  const driverPath = path.join(paths.repoRoot, "internal/testutil/acpmock/driver/dist/index.js");
  const nodeCommand = process.env.AGH_TEST_NODE_BIN?.trim() || "node";
  const agentsDir = path.join(paths.homeDir, "agents");
  const diagnosticsDir = path.join(paths.homeDir, "logs", "acpmock");

  await mkdir(agentsDir, { recursive: true });
  await mkdir(diagnosticsDir, { recursive: true });

  for (const spec of mockAgents) {
    const registration = await loadMockAgentRegistration(
      driverPath,
      diagnosticsDir,
      nodeCommand,
      spec
    );
    const agentDir = path.join(agentsDir, registration.agentName);
    const agentDefPath = path.join(agentDir, "AGENT.md");
    await mkdir(agentDir, { recursive: true });
    await writeFile(
      agentDefPath,
      renderMockAgentDef(registration.agentName, registration.agent, registration.command),
      "utf8"
    );
  }
}

export async function applyBrowserRuntimeSeed(
  runtime: BrowserRuntimeSeedClient,
  seed: BrowserRuntimeSeed | undefined
): Promise<BrowserRuntimeSeedResult> {
  if (seed === undefined) {
    return {};
  }

  let workspace = await resolveSeedWorkspace(runtime, seed);
  if (seed.session === undefined) {
    return { workspace };
  }

  const sessionWorkspaceRoot = seed.session.workspaceRootDir ?? seed.workspace?.rootDir;
  if (sessionWorkspaceRoot === undefined || sessionWorkspaceRoot.trim() === "") {
    throw new Error(
      "session runtime seed requires a workspace root via seed.workspace or session.workspaceRootDir"
    );
  }

  if (!workspace || workspace.root_dir !== sessionWorkspaceRoot) {
    workspace = await runtime.resolveWorkspace(sessionWorkspaceRoot);
  }

  const payload = await runtime.requestJSON<{ session: SeededSessionPayload }>("/api/sessions", {
    method: "POST",
    body: JSON.stringify({
      agent_name: seed.session.agentName,
      workspace: workspace.id,
    }),
  });

  return {
    workspace,
    session: payload.session,
  };
}

export async function seedBrowserNetworkOperatorFlow(
  runtime: Pick<BrowserRuntimeSeedClient, "requestJSON">,
  seed: BrowserNetworkOperatorFlowSeed
): Promise<BrowserNetworkOperatorFlowResult> {
  const channel = seed.channel.trim();
  const initiatorAgentName = seed.initiatorAgentName.trim();
  const responderAgentName = seed.responderAgentName.trim();

  if (channel === "") {
    throw new Error("network operator flow seed requires a non-empty channel");
  }
  if (initiatorAgentName === "" || responderAgentName === "") {
    throw new Error("network operator flow seed requires both initiator and responder agents");
  }

  const timeoutMs = seed.timeoutMs ?? NETWORK_OPERATOR_FLOW_TIMEOUT_MS;

  const channelState = await waitForSeedCondition(
    async () => {
      const payload = await runtime.requestJSON<{ channel: NetworkChannelSeedPayload }>(
        `/api/network/channels/${encodeURIComponent(channel)}`
      );
      const sessions = payload.channel.sessions ?? [];
      const initiatorSession = sessions.find(session => session.agent_name === initiatorAgentName);
      const responderSession = sessions.find(session => session.agent_name === responderAgentName);

      if (!initiatorSession || !responderSession) {
        return null;
      }

      return {
        initiatorSession,
        responderSession,
      };
    },
    `network channel ${channel} to include both operator-flow agents`,
    timeoutMs
  );

  const peerState = await waitForSeedCondition(
    async () => {
      const payload = await runtime.requestJSON<{ peers: NetworkPeerSeedPayload[] }>(
        `/api/network/peers?channel=${encodeURIComponent(channel)}`
      );
      const initiatorPeer = payload.peers.find(
        peer => peer.session_id === channelState.initiatorSession.id
      );
      const responderPeer = payload.peers.find(
        peer => peer.session_id === channelState.responderSession.id
      );

      if (!initiatorPeer || !responderPeer) {
        return null;
      }

      return {
        initiatorPeer,
        responderPeer,
      };
    },
    `network peers for ${channel}`,
    timeoutMs
  );

  await sendNetworkSeedMessage(runtime, {
    session_id: channelState.initiatorSession.id,
    channel,
    kind: "say",
    id: browserNetworkOperatorFlowScenario.messageIds.say,
    trace_id: browserNetworkOperatorFlowScenario.traceId,
    body: {
      text: browserNetworkOperatorFlowScenario.texts.say,
      intent: "request-help",
      artifacts: [],
    },
  });

  await sendNetworkSeedMessage(runtime, {
    session_id: channelState.responderSession.id,
    channel,
    kind: "direct",
    to: peerState.initiatorPeer.peer_id,
    interaction_id: browserNetworkOperatorFlowScenario.interactionId,
    reply_to: browserNetworkOperatorFlowScenario.messageIds.say,
    trace_id: browserNetworkOperatorFlowScenario.traceId,
    causation_id: browserNetworkOperatorFlowScenario.messageIds.say,
    id: browserNetworkOperatorFlowScenario.messageIds.direct,
    body: {
      text: browserNetworkOperatorFlowScenario.texts.direct,
      intent: "handoff",
      artifacts: [],
    },
  });

  await sendNetworkSeedMessage(runtime, {
    session_id: channelState.responderSession.id,
    channel,
    kind: "trace",
    to: peerState.initiatorPeer.peer_id,
    interaction_id: browserNetworkOperatorFlowScenario.interactionId,
    reply_to: browserNetworkOperatorFlowScenario.messageIds.direct,
    trace_id: browserNetworkOperatorFlowScenario.traceId,
    causation_id: browserNetworkOperatorFlowScenario.messageIds.direct,
    id: browserNetworkOperatorFlowScenario.messageIds.trace,
    body: {
      state: "completed",
      message: browserNetworkOperatorFlowScenario.texts.trace,
      result: {
        summary: "Fixed migration assertion mismatch in sessiondb tests.",
      },
      artifact_refs: [],
    },
  });

  await waitForSeedCondition(
    async () => {
      const payload = await runtime.requestJSON<{ messages: NetworkMessageSeedPayload[] }>(
        `/api/network/channels/${encodeURIComponent(channel)}/messages`
      );
      const messageIds = new Set(payload.messages.map(message => message.message_id));

      return messageIds.has(browserNetworkOperatorFlowScenario.messageIds.say)
        ? payload.messages
        : null;
    },
    `network timeline for ${channel}`,
    timeoutMs
  );

  await waitForSeedCondition(
    async () => {
      const payload = await runtime.requestJSON<NetworkStatusSeedPayload>("/api/network/status");
      const kindMetrics = new Map(
        (payload.network?.kind_metrics ?? []).map(metric => [metric.kind ?? "", metric])
      );
      const direct = kindMetrics.get("direct");
      const trace = kindMetrics.get("trace");

      if ((direct?.sent ?? 0) < 1 || (direct?.delivered ?? 0) < 1) {
        return null;
      }
      if ((trace?.sent ?? 0) < 1 || (trace?.delivered ?? 0) < 1) {
        return null;
      }

      return payload.network?.kind_metrics ?? [];
    },
    `network operator metrics for ${channel}`,
    timeoutMs
  );

  return {
    channel,
    initiator: {
      ...channelState.initiatorSession,
      peerId: peerState.initiatorPeer.peer_id,
    },
    responder: {
      ...channelState.responderSession,
      peerId: peerState.responderPeer.peer_id,
    },
    messageIds: browserNetworkOperatorFlowScenario.messageIds,
  };
}

export async function seedBrowserAutomationOperatorFlow(
  runtime: Pick<BrowserRuntimeSeedClient, "requestJSON">,
  seed: BrowserAutomationOperatorFlowSeed
): Promise<BrowserAutomationOperatorFlowResult> {
  const agentName = seed.agentName.trim();
  if (agentName === "") {
    throw new Error("automation operator flow seed requires a non-empty agent name");
  }

  const timeoutMs = seed.timeoutMs ?? AUTOMATION_OPERATOR_FLOW_TIMEOUT_MS;
  const job = await createAutomationOperatorJob(runtime, agentName);
  const trigger = await createAutomationOperatorTrigger(runtime, agentName);

  const initialRun = (
    await runtime.requestJSON<{ run: AutomationRun }>(
      `/api/automation/jobs/${encodeURIComponent(job.id)}/trigger`,
      {
        method: "POST",
      }
    )
  ).run;

  const baselineRun = await waitForSeedCondition(
    async () => {
      const payload = await runtime.requestJSON<{ run: AutomationRun }>(
        `/api/automation/runs/${encodeURIComponent(initialRun.id)}`
      );
      const run = payload.run;
      return run.status === "completed" && Boolean(run.session_id) ? run : null;
    },
    `completed automation run ${initialRun.id}`,
    timeoutMs
  );

  await waitForSeedCondition(
    async () => {
      const payload = await runtime.requestJSON<{ runs: AutomationRun[] }>(
        `/api/automation/jobs/${encodeURIComponent(job.id)}/runs?limit=10`
      );
      return payload.runs.some(
        run => run.id === baselineRun.id && run.session_id === baselineRun.session_id
      )
        ? payload.runs
        : null;
    },
    `visible run history for automation job ${job.id}`,
    timeoutMs
  );

  await waitForSeedCondition(
    async () => {
      const payload = await runtime.requestJSON<{ messages: unknown[] }>(
        `/api/sessions/${encodeURIComponent(baselineRun.session_id ?? "")}/transcript`
      );
      const transcript = JSON.stringify(payload.messages);

      return transcript.includes(browserAutomationOperatorFlowScenario.job.prompt) &&
        transcript.includes(browserAutomationOperatorFlowScenario.transcript.assistant)
        ? transcript
        : null;
    },
    `automation transcript for session ${baselineRun.session_id}`,
    timeoutMs
  );

  return {
    job,
    trigger,
    baselineRun,
  };
}

export async function seedBrowserBridgeOperatorFlow(
  runtime: BrowserBridgeOperatorSeedRuntime,
  seed: BrowserBridgeOperatorFlowSeed = {}
): Promise<BrowserBridgeOperatorFlowResult> {
  const requestOperatorJSON = async <T>(pathname: string, init?: RequestInit): Promise<T> => {
    if (runtime.requestOperatorJSON) {
      return await runtime.requestOperatorJSON<T>(pathname, init);
    }
    return await runtime.requestJSON<T>(pathname, init);
  };
  const prepareExtension = seed.prepareExtension ?? prepareBrowserBridgeExtension;
  const timeoutMs = seed.timeoutMs ?? BRIDGE_OPERATOR_FLOW_TIMEOUT_MS;
  const displayName =
    seed.displayName?.trim() || browserBridgeOperatorFlowScenario.bridge.initialName;
  const prepared = await prepareExtension();

  await requestOperatorJSON<{ extension: { name: string } }>("/api/extensions", {
    method: "POST",
    body: JSON.stringify({
      checksum: prepared.checksum,
      path: prepared.extensionDir,
    }),
  });

  await waitForSeedCondition(
    async () => {
      const payload = await requestOperatorJSON<{
        extension: { enabled: boolean; health?: string; name: string; state: string };
      }>(`/api/extensions/${encodeURIComponent(BRIDGE_EXTENSION_NAME)}`);
      const extension = payload.extension;
      return extension.name === BRIDGE_EXTENSION_NAME && extension.enabled ? extension : null;
    },
    `${BRIDGE_EXTENSION_NAME} extension install`,
    timeoutMs
  );

  const provider = await waitForSeedCondition(
    async () => {
      const payload = await runtime.requestJSON<{ providers: BridgeProvider[] }>(
        "/api/bridges/providers"
      );
      return (
        payload.providers.find(
          candidate =>
            candidate.extension_name === BRIDGE_EXTENSION_NAME &&
            candidate.platform === BRIDGE_PLATFORM
        ) ?? null
      );
    },
    `${BRIDGE_EXTENSION_NAME} bridge provider`,
    timeoutMs
  );

  const workspace = await resolveBrowserBridgeWorkspace(runtime, timeoutMs);

  const createResponse = await runtime.requestJSON<BridgeDetailResponse>("/api/bridges", {
    method: "POST",
    body: JSON.stringify({
      display_name: displayName,
      enabled: false,
      extension_name: BRIDGE_EXTENSION_NAME,
      platform: BRIDGE_PLATFORM,
      provider_config: browserBridgeOperatorFlowScenario.bridge.initialProviderConfig,
      routing_policy: {
        include_group: false,
        include_peer: true,
        include_thread: true,
      },
      scope: "workspace",
      status: "disabled",
      workspace_id: workspace.id,
    }),
  });

  await runtime.requestJSON<{ binding: { binding_name: string; vault_ref: string } }>(
    `/api/bridges/${encodeURIComponent(createResponse.bridge.id)}/secret-bindings/${encodeURIComponent(
      browserBridgeOperatorFlowScenario.secretBinding.name
    )}`,
    {
      method: "PUT",
      body: JSON.stringify({
        kind: browserBridgeOperatorFlowScenario.secretBinding.name,
        vault_ref: `env:${browserBridgeOperatorFlowScenario.secretBinding.envName}`,
      }),
    }
  );

  const bridgeDetail = await waitForSeedCondition(
    async () => {
      const payload = await runtime.requestJSON<BridgeDetailResponse>(
        `/api/bridges/${encodeURIComponent(createResponse.bridge.id)}`
      );
      return payload.health?.status === "disabled" ? payload : null;
    },
    `bridge detail for ${createResponse.bridge.id}`,
    timeoutMs
  );

  return {
    bridge: bridgeDetail.bridge,
    extension: {
      checksum: prepared.checksum,
      dir: prepared.extensionDir,
      markers: prepared.markers,
      name: BRIDGE_EXTENSION_NAME,
      platform: BRIDGE_PLATFORM,
    },
    health: bridgeDetail.health,
    provider,
  };
}

async function resolveBrowserBridgeWorkspace(
  runtime: BrowserBridgeOperatorSeedRuntime,
  timeoutMs: number
): Promise<WorkspacePayload> {
  const seededWorkspace = runtime.seeded?.workspace;
  if (seededWorkspace) {
    return seededWorkspace;
  }

  const resolveWorkspace = runtime.resolveWorkspace?.bind(runtime);
  const homeDir = runtime.paths?.homeDir;
  if (resolveWorkspace && homeDir) {
    return await waitForSeedCondition(
      async () => await resolveWorkspace(homeDir),
      "browser bridge workspace",
      timeoutMs
    );
  }

  return await waitForSeedCondition(
    async () => {
      const payload = await runtime.requestJSON<{ workspaces: WorkspacePayload[] }>(
        "/api/workspaces"
      );
      return payload.workspaces[0] ?? null;
    },
    "browser bridge workspace",
    timeoutMs
  );
}

export async function triggerBrowserBridgeIngress(
  runtime: Pick<BrowserRuntimeSeedClient, "requestJSON">,
  seeded: Pick<BrowserBridgeOperatorFlowResult, "bridge" | "extension">,
  seed: BrowserBridgeIngressSeed = {}
): Promise<BrowserBridgeIngressResult> {
  const timeoutMs = seed.timeoutMs ?? BRIDGE_OPERATOR_FLOW_TIMEOUT_MS;
  const update = createBrowserBridgeIngressUpdate(seeded.bridge.id, {
    messageId: seed.messageId,
    text: seed.text,
    updateId: seed.updateId,
  });
  await appendJSONLine(seeded.extension.markers.updates, update);

  await waitForSeedCondition(
    async () => {
      const payload = await runtime.requestJSON<BridgeDetailResponse>(
        `/api/bridges/${encodeURIComponent(seeded.bridge.id)}`
      );
      return (payload.health?.route_count ?? 0) >= 1 && Boolean(payload.health?.last_success_at)
        ? payload
        : null;
    },
    `bridge ingress health for ${seeded.bridge.id}`,
    timeoutMs
  );

  const routes = await waitForSeedCondition(
    async () => {
      const payload = await runtime.requestJSON<{ routes: BridgeRoute[] }>(
        `/api/bridges/${encodeURIComponent(seeded.bridge.id)}/routes`
      );
      return payload.routes.length > 0 ? payload.routes : null;
    },
    `bridge routes for ${seeded.bridge.id}`,
    timeoutMs
  );

  const sessionId = routes[0]?.session_id?.trim();
  if (!sessionId) {
    throw new Error(`bridge ingress for ${seeded.bridge.id} did not produce a session route`);
  }

  const transcriptPayload = await waitForSeedCondition(
    async () => {
      const payload = await runtime.requestJSON<{ messages: unknown[] }>(
        `/api/sessions/${encodeURIComponent(sessionId)}/transcript`
      );
      const transcript = JSON.stringify(payload.messages);
      const assistantText =
        seed.assistantText?.trim() || browserBridgeOperatorFlowScenario.ingress.assistant;

      return transcript.includes(assistantText) ? payload : null;
    },
    `bridge transcript for session ${sessionId}`,
    timeoutMs
  );

  return {
    routes,
    sessionId,
    transcript: JSON.stringify(transcriptPayload.messages),
  };
}

async function resolveSeedWorkspace(
  runtime: BrowserRuntimeSeedClient,
  seed: BrowserRuntimeSeed
): Promise<WorkspacePayload | undefined> {
  const rootDir = seed.workspace?.rootDir?.trim();
  if (!rootDir) {
    return undefined;
  }

  return runtime.resolveWorkspace(rootDir);
}

async function loadMockAgentRegistration(
  driverPath: string,
  diagnosticsDir: string,
  nodeCommand: string,
  spec: BrowserMockAgentSeed
): Promise<{
  agentName: string;
  command: string;
  agent: MockFixtureAgent;
}> {
  const fixturePath = path.resolve(spec.fixturePath);
  const rawFixture = await readFile(fixturePath, "utf8");
  const fixture = JSON.parse(rawFixture) as MockFixture;
  const fixtureAgentName = spec.fixtureAgent?.trim() || spec.agentName?.trim() || "";
  if (fixtureAgentName === "") {
    throw new Error(`mock agent seed for ${fixturePath} requires fixtureAgent or agentName`);
  }

  const agent = fixture.agents?.find(candidate => candidate.name === fixtureAgentName);
  if (!agent) {
    throw new Error(`mock agent fixture ${fixturePath} does not contain agent ${fixtureAgentName}`);
  }

  const agentName = spec.agentName?.trim() || fixtureAgentName;
  const diagnosticsPath = path.join(diagnosticsDir, `${agentName}.jsonl`);
  const command = shellQuote([
    nodeCommand,
    driverPath,
    "--fixture",
    fixturePath,
    "--agent",
    fixtureAgentName,
    "--diagnostics",
    diagnosticsPath,
  ]);

  return { agentName, command, agent };
}

function renderMockAgentDef(name: string, agent: MockFixtureAgent, command: string): string {
  const prompt = agent.prompt?.trim() || `You are ${name}.`;
  const lines = ["---", `name: ${name}`, `provider: ${agent.provider}`, `command: ${command}`];

  if (agent.model?.trim()) {
    lines.push(`model: ${agent.model.trim()}`);
  }
  if (agent.permissions?.trim()) {
    lines.push(`permissions: ${agent.permissions.trim()}`);
  }

  lines.push("---", "", prompt, "");
  return lines.join("\n");
}

async function sendNetworkSeedMessage(
  runtime: Pick<BrowserRuntimeSeedClient, "requestJSON">,
  body: Record<string, unknown>
): Promise<void> {
  await runtime.requestJSON("/api/network/send", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

async function createAutomationOperatorJob(
  runtime: Pick<BrowserRuntimeSeedClient, "requestJSON">,
  agentName: string
): Promise<AutomationJob> {
  const request: CreateAutomationJobRequest = {
    agent_name: agentName,
    enabled: true,
    fire_limit: { max: 12, window: "1h" },
    name: browserAutomationOperatorFlowScenario.job.initialName,
    prompt: browserAutomationOperatorFlowScenario.job.prompt,
    retry: { strategy: "none", max_retries: 0, base_delay: "" },
    schedule: {
      mode: "cron",
      expr: browserAutomationOperatorFlowScenario.job.scheduleExpr,
    },
    scope: "global",
  };

  return (
    await runtime.requestJSON<{ job: AutomationJob }>("/api/automation/jobs", {
      method: "POST",
      body: JSON.stringify(request),
    })
  ).job;
}

async function createAutomationOperatorTrigger(
  runtime: Pick<BrowserRuntimeSeedClient, "requestJSON">,
  agentName: string
): Promise<AutomationTrigger> {
  const request: CreateAutomationTriggerRequest = {
    agent_name: agentName,
    enabled: true,
    endpoint_slug: browserAutomationOperatorFlowScenario.trigger.endpointSlug,
    event: browserAutomationOperatorFlowScenario.trigger.event,
    filter: {
      "data.branch": "main",
    },
    fire_limit: { max: 12, window: "1h" },
    name: browserAutomationOperatorFlowScenario.trigger.name,
    prompt: browserAutomationOperatorFlowScenario.trigger.prompt,
    retry: { strategy: "none", max_retries: 0, base_delay: "" },
    scope: "global",
    webhook_id: browserAutomationOperatorFlowScenario.trigger.webhookID,
    webhook_secret: browserAutomationOperatorFlowScenario.trigger.webhookSecret,
  };

  return (
    await runtime.requestJSON<{ trigger: AutomationTrigger }>("/api/automation/triggers", {
      method: "POST",
      body: JSON.stringify(request),
    })
  ).trigger;
}

async function prepareBrowserBridgeExtension(): Promise<PreparedBrowserBridgeExtension> {
  const repoRoot = resolveBrowserRepoRoot();
  const sourceDir = path.join(repoRoot, "sdk", "examples", BRIDGE_EXTENSION_NAME);
  const tempRoot = await mkdtemp(path.join(os.tmpdir(), "agh-browser-bridge-extension-"));
  const extensionDir = path.join(tempRoot, BRIDGE_EXTENSION_NAME);
  const markers = createBridgeAdapterMarkerPaths(path.join(extensionDir, "markers"));

  await cp(sourceDir, extensionDir, { recursive: true });
  await mkdir(path.join(extensionDir, "bin"), { recursive: true });
  await mkdir(path.dirname(markers.handshake), { recursive: true });

  const manifestPath = path.join(extensionDir, "extension.toml");
  const rawManifest = await readFile(manifestPath, "utf8");
  await writeFile(manifestPath, patchBridgeExtensionManifest(rawManifest, markers), "utf8");

  await execFileAsync(
    "go",
    [
      "build",
      "-o",
      path.join(extensionDir, "bin", BRIDGE_EXTENSION_NAME),
      "./sdk/examples/telegram-reference",
    ],
    {
      cwd: repoRoot,
      env: process.env,
    }
  );

  return {
    checksum: await computeDirectoryChecksum(extensionDir),
    extensionDir,
    markers,
  };
}

function resolveBrowserRepoRoot(): string {
  return path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "..", "..");
}

function createBridgeAdapterMarkerPaths(rootDir: string): BridgeAdapterMarkerPaths {
  return {
    crashOnce: path.join(rootDir, "adapter-crash-once.json"),
    delivery: path.join(rootDir, "adapter-deliveries.jsonl"),
    handshake: path.join(rootDir, "adapter-handshake.json"),
    ingest: path.join(rootDir, "adapter-ingest.jsonl"),
    ownership: path.join(rootDir, "adapter-ownership.json"),
    shutdown: path.join(rootDir, "adapter-shutdown.log"),
    starts: path.join(rootDir, "adapter-starts.log"),
    state: path.join(rootDir, "adapter-states.jsonl"),
    updates: path.join(rootDir, "adapter-updates.jsonl"),
  };
}

function patchBridgeExtensionManifest(manifest: string, markers: BridgeAdapterMarkerPaths): string {
  let next = manifest;
  const values = {
    AGH_BRIDGE_ADAPTER_CRASH_ONCE_PATH: markers.crashOnce,
    AGH_BRIDGE_ADAPTER_DELIVERY_PATH: markers.delivery,
    AGH_BRIDGE_ADAPTER_HANDSHAKE_PATH: markers.handshake,
    AGH_BRIDGE_ADAPTER_INGEST_PATH: markers.ingest,
    AGH_BRIDGE_ADAPTER_OWNERSHIP_PATH: markers.ownership,
    AGH_BRIDGE_ADAPTER_SHUTDOWN_PATH: markers.shutdown,
    AGH_BRIDGE_ADAPTER_STARTS_PATH: markers.starts,
    AGH_BRIDGE_ADAPTER_STATE_PATH: markers.state,
    AGH_BRIDGE_ADAPTER_UPDATES_PATH: markers.updates,
  } as const;

  for (const [envName, value] of Object.entries(values)) {
    const placeholder = `"{{env:${envName}}}"`;
    next = next.replace(new RegExp(escapeRegExp(placeholder), "g"), JSON.stringify(value));
  }

  return next;
}

async function computeDirectoryChecksum(rootDir: string): Promise<string> {
  const repoRoot = resolveBrowserRepoRoot();
  await mkdir(path.join(repoRoot, ".tmp"), { recursive: true });
  const helperRoot = await mkdtemp(path.join(repoRoot, ".tmp", "agh-browser-checksum-"));
  const helperPath = path.join(helperRoot, "main.go");

  await writeFile(
    helperPath,
    [
      "package main",
      "",
      "import (",
      '\t"fmt"',
      '\t"os"',
      '\t"strings"',
      "",
      '\textensionpkg "github.com/pedronauck/agh/internal/extension"',
      ")",
      "",
      "func main() {",
      "\tif len(os.Args) != 2 {",
      '\t\tfmt.Fprintln(os.Stderr, "extension directory is required")',
      "\t\tos.Exit(1)",
      "\t}",
      "",
      "\tchecksum, err := extensionpkg.ComputeDirectoryChecksum(strings.TrimSpace(os.Args[1]))",
      "\tif err != nil {",
      "\t\tfmt.Fprintln(os.Stderr, err)",
      "\t\tos.Exit(1)",
      "\t}",
      "",
      "\tfmt.Print(checksum)",
      "}",
      "",
    ].join("\n"),
    "utf8"
  );

  try {
    const { stdout } = await execFileAsync("go", ["run", helperPath, rootDir], {
      cwd: repoRoot,
      env: process.env,
    });
    const checksum = stdout.trim();
    if (checksum === "") {
      throw new Error(`go checksum helper returned an empty checksum for ${rootDir}`);
    }
    return checksum;
  } finally {
    await rm(helperRoot, { force: true, recursive: true });
  }
}

function createBrowserBridgeIngressUpdate(
  bridgeInstanceID: string,
  input: Pick<BrowserBridgeIngressSeed, "messageId" | "text" | "updateId">
) {
  return {
    bridge_instance_id: bridgeInstanceID,
    message: {
      chat: {
        id: 777,
        title: "ops",
        type: "supergroup",
      },
      date: Math.floor(Date.now() / 1000),
      from: {
        first_name: "Alice",
        id: 888,
        last_name: "Example",
        username: "alice",
      },
      message_id: input.messageId ?? browserBridgeOperatorFlowScenario.ingress.messageId,
      message_thread_id: Number(browserBridgeOperatorFlowScenario.testDelivery.threadId),
      text: input.text?.trim() || browserBridgeOperatorFlowScenario.ingress.text,
    },
    update_id: input.updateId ?? browserBridgeOperatorFlowScenario.ingress.updateId,
  };
}

async function appendJSONLine(targetPath: string, value: unknown): Promise<void> {
  await mkdir(path.dirname(targetPath), { recursive: true });
  const line = `${JSON.stringify(value)}\n`;
  await writeFile(targetPath, line, {
    encoding: "utf8",
    flag: "a",
  });
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

async function waitForSeedCondition<T>(
  read: () => Promise<T | null>,
  label: string,
  timeoutMs: number
): Promise<T> {
  const deadline = Date.now() + timeoutMs;
  let lastError: unknown;

  while (Date.now() < deadline) {
    try {
      const value = await read();
      if (value !== null) {
        return value;
      }
    } catch (error) {
      lastError = error;
    }

    await delay(BROWSER_SEED_POLL_MS);
  }

  const detail = lastError instanceof Error ? `; last error: ${lastError.message}` : "";
  throw new Error(`timed out waiting for ${label}${detail}`);
}

async function delay(ms: number): Promise<void> {
  await new Promise(resolve => {
    setTimeout(resolve, ms);
  });
}

function shellQuote(argv: string[]): string {
  return argv
    .map(argument => {
      if (/^[A-Za-z0-9_./:@%+=,-]+$/.test(argument)) {
        return argument;
      }
      return `'${argument.replace(/'/g, `'"'"'`)}'`;
    })
    .join(" ");
}
