// @vitest-environment node

import { mkdtemp, mkdir, readFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { describe, expect, it, vi } from "vitest";

import {
  applyBrowserRuntimeSeed,
  browserAutomationOperatorFlowScenario,
  browserBridgeOperatorFlowScenario,
  browserNetworkOperatorFlowScenario,
  seedBrowserBridgeOperatorFlow,
  seedBrowserAutomationOperatorFlow,
  seedBrowserNetworkOperatorFlow,
  seedBrowserRuntimeHome,
  type BrowserRuntimeSeedClient,
} from "./runtime-seed";

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

describe("browser runtime seed helpers", () => {
  it("writes fixture-backed mock agent definitions into the isolated browser runtime home", async () => {
    const homeDir = await mkdtemp(path.join(os.tmpdir(), "agh-browser-runtime-home-"));
    await mkdir(path.join(homeDir, "agents"), { recursive: true });
    await mkdir(path.join(homeDir, "logs"), { recursive: true });

    await seedBrowserRuntimeHome(
      {
        homeDir,
        repoRoot: path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "..", ".."),
      },
      {
        mockAgents: [
          {
            fixturePath: browserLifecycleFixture,
            fixtureAgent: "browser-lifecycle-agent",
          },
        ],
      }
    );

    const agentDef = await readFile(
      path.join(homeDir, "agents", "browser-lifecycle-agent", "AGENT.md"),
      "utf8"
    );

    expect(agentDef).toContain("name: browser-lifecycle-agent");
    expect(agentDef).toContain("provider: claude");
    expect(agentDef).toContain("--fixture");
    expect(agentDef).toContain("browser_session_lifecycle_fixture.json");
    expect(agentDef).toContain("--agent browser-lifecycle-agent");
  });

  it("creates seeded workspace and session state through public runtime surfaces", async () => {
    const requestJSON = vi.fn(async () => ({
      session: {
        id: "sess_browser_01",
        agent_name: "browser-lifecycle-agent",
        workspace_id: "ws_home",
        state: "active",
        name: "browser-session",
      },
    }));
    const resolveWorkspace = vi.fn(async () => ({
      id: "ws_home",
      root_dir: "/tmp/browser-home",
      name: "Browser Home",
    }));

    const seeded = await applyBrowserRuntimeSeed(
      {
        requestJSON: requestJSON as BrowserRuntimeSeedClient["requestJSON"],
        resolveWorkspace: resolveWorkspace as BrowserRuntimeSeedClient["resolveWorkspace"],
      },
      {
        workspace: { rootDir: "/tmp/browser-home" },
        session: {
          agentName: "browser-lifecycle-agent",
        },
      }
    );

    expect(resolveWorkspace).toHaveBeenCalledWith("/tmp/browser-home");
    expect(requestJSON).toHaveBeenCalledWith(
      "/api/sessions",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          agent_name: "browser-lifecycle-agent",
          workspace: "ws_home",
        }),
      })
    );
    expect(seeded.workspace?.id).toBe("ws_home");
    expect(seeded.session?.id).toBe("sess_browser_01");
  });

  it("materializes deterministic network channel, peer, and timeline state through public runtime surfaces", async () => {
    const requestJSON = vi.fn(async (pathname: string, init?: RequestInit) => {
      if (pathname === "/api/network/channels/browser-builders") {
        return {
          channel: {
            channel: "browser-builders",
            sessions: [
              {
                id: "sess_ops",
                agent_name: "mock-ops-coordinator",
                workspace_id: "ws_browser",
                state: "active",
              },
              {
                id: "sess_patch",
                agent_name: "mock-patch-worker",
                workspace_id: "ws_browser",
                state: "active",
              },
            ],
          },
        };
      }

      if (pathname === "/api/network/peers?channel=browser-builders") {
        return {
          peers: [
            {
              channel: "browser-builders",
              peer_id: "peer_ops",
              session_id: "sess_ops",
            },
            {
              channel: "browser-builders",
              peer_id: "peer_patch",
              session_id: "sess_patch",
            },
          ],
        };
      }

      if (pathname === "/api/network/send") {
        return {
          message: {
            id: JSON.parse(init?.body as string).id,
          },
        };
      }

      if (pathname === "/api/network/channels/browser-builders/messages") {
        return {
          messages: [
            {
              message_id: browserNetworkOperatorFlowScenario.messageIds.say,
            },
          ],
        };
      }

      if (pathname === "/api/network/status") {
        return {
          network: {
            kind_metrics: [
              {
                kind: "direct",
                sent: 1,
                delivered: 1,
              },
              {
                kind: "trace",
                sent: 1,
                delivered: 1,
              },
            ],
          },
        };
      }

      throw new Error(`unexpected request ${pathname}`);
    });

    const seeded = await seedBrowserNetworkOperatorFlow(
      {
        requestJSON: requestJSON as BrowserRuntimeSeedClient["requestJSON"],
      },
      {
        channel: "browser-builders",
        initiatorAgentName: "mock-ops-coordinator",
        responderAgentName: "mock-patch-worker",
      }
    );

    expect(seeded).toEqual({
      channel: "browser-builders",
      initiator: {
        id: "sess_ops",
        agent_name: "mock-ops-coordinator",
        workspace_id: "ws_browser",
        state: "active",
        peerId: "peer_ops",
      },
      responder: {
        id: "sess_patch",
        agent_name: "mock-patch-worker",
        workspace_id: "ws_browser",
        state: "active",
        peerId: "peer_patch",
      },
      messageIds: browserNetworkOperatorFlowScenario.messageIds,
    });

    const sendBodies = requestJSON.mock.calls
      .filter(([pathname]) => pathname === "/api/network/send")
      .map(([, init]) => JSON.parse(init?.body as string));

    expect(sendBodies).toEqual([
      expect.objectContaining({
        session_id: "sess_ops",
        channel: "browser-builders",
        kind: "say",
        id: browserNetworkOperatorFlowScenario.messageIds.say,
      }),
      expect.objectContaining({
        session_id: "sess_patch",
        channel: "browser-builders",
        kind: "direct",
        to: "peer_ops",
        id: browserNetworkOperatorFlowScenario.messageIds.direct,
      }),
      expect.objectContaining({
        session_id: "sess_patch",
        channel: "browser-builders",
        kind: "trace",
        to: "peer_ops",
        id: browserNetworkOperatorFlowScenario.messageIds.trace,
      }),
    ]);

    expect(requestJSON).toHaveBeenCalledWith("/api/network/status");
  });

  it("seeds deterministic automation jobs, triggers, and visible run history through public runtime surfaces", async () => {
    const requestJSON = vi.fn(async (pathname: string, _init?: RequestInit) => {
      if (pathname === "/api/automation/jobs") {
        return {
          job: {
            id: "job_browser_deploy_review",
            name: browserAutomationOperatorFlowScenario.job.initialName,
            agent_name: "browser-automation-runner",
            prompt: browserAutomationOperatorFlowScenario.job.prompt,
            scope: "global",
            source: "dynamic",
            enabled: true,
            created_at: "2026-04-17T10:00:00Z",
            updated_at: "2026-04-17T10:00:00Z",
            schedule: {
              mode: "cron",
              expr: browserAutomationOperatorFlowScenario.job.scheduleExpr,
            },
            retry: { strategy: "none", max_retries: 0, base_delay: "" },
            fire_limit: { max: 12, window: "1h" },
            next_run: "2026-04-18T09:00:00Z",
          },
        };
      }

      if (pathname === "/api/automation/triggers") {
        return {
          trigger: {
            id: "trg_browser_deploy_review",
            name: browserAutomationOperatorFlowScenario.trigger.name,
            agent_name: "browser-automation-runner",
            prompt: browserAutomationOperatorFlowScenario.trigger.prompt,
            event: browserAutomationOperatorFlowScenario.trigger.event,
            endpoint_slug: browserAutomationOperatorFlowScenario.trigger.endpointSlug,
            webhook_id: browserAutomationOperatorFlowScenario.trigger.webhookID,
            scope: "global",
            source: "dynamic",
            enabled: true,
            filter: { "data.branch": "main" },
            created_at: "2026-04-17T10:01:00Z",
            updated_at: "2026-04-17T10:01:00Z",
            retry: { strategy: "none", max_retries: 0, base_delay: "" },
            fire_limit: { max: 12, window: "1h" },
          },
        };
      }

      if (pathname === "/api/automation/jobs/job_browser_deploy_review/trigger") {
        return {
          run: {
            id: "run_browser_deploy_001",
            job_id: "job_browser_deploy_review",
            status: "running",
            attempt: 1,
            started_at: "2026-04-17T10:02:00Z",
          },
        };
      }

      if (pathname === "/api/automation/runs/run_browser_deploy_001") {
        return {
          run: {
            id: "run_browser_deploy_001",
            job_id: "job_browser_deploy_review",
            session_id: "sess_browser_automation_01",
            status: "completed",
            attempt: 1,
            started_at: "2026-04-17T10:02:00Z",
            ended_at: "2026-04-17T10:02:05Z",
          },
        };
      }

      if (pathname === "/api/automation/jobs/job_browser_deploy_review/runs?limit=10") {
        return {
          runs: [
            {
              id: "run_browser_deploy_001",
              job_id: "job_browser_deploy_review",
              session_id: "sess_browser_automation_01",
              status: "completed",
              attempt: 1,
              started_at: "2026-04-17T10:02:00Z",
              ended_at: "2026-04-17T10:02:05Z",
            },
          ],
        };
      }

      if (pathname === "/api/sessions/sess_browser_automation_01/transcript") {
        return {
          messages: [
            {
              id: "msg_user_automation",
              role: "user",
              content: browserAutomationOperatorFlowScenario.job.prompt,
            },
            {
              id: "msg_assistant_automation",
              role: "assistant",
              content: browserAutomationOperatorFlowScenario.transcript.assistant,
            },
          ],
        };
      }

      throw new Error(`unexpected request ${pathname}`);
    });

    const seeded = await seedBrowserAutomationOperatorFlow(
      {
        requestJSON: requestJSON as BrowserRuntimeSeedClient["requestJSON"],
      },
      {
        agentName: "browser-automation-runner",
      }
    );

    expect(seeded.job.id).toBe("job_browser_deploy_review");
    expect(seeded.trigger.id).toBe("trg_browser_deploy_review");
    expect(seeded.baselineRun.id).toBe("run_browser_deploy_001");
    expect(seeded.baselineRun.session_id).toBe("sess_browser_automation_01");

    expect(requestJSON).toHaveBeenNthCalledWith(
      1,
      "/api/automation/jobs",
      expect.objectContaining({ method: "POST" })
    );
    expect(JSON.parse(requestJSON.mock.calls[0]?.[1]?.body as string)).toEqual(
      expect.objectContaining({
        agent_name: "browser-automation-runner",
        name: browserAutomationOperatorFlowScenario.job.initialName,
        prompt: browserAutomationOperatorFlowScenario.job.prompt,
      })
    );
    expect(requestJSON).toHaveBeenCalledWith(
      "/api/automation/jobs/job_browser_deploy_review/trigger",
      expect.objectContaining({ method: "POST" })
    );
    expect(requestJSON).toHaveBeenCalledWith(
      "/api/automation/jobs/job_browser_deploy_review/runs?limit=10"
    );
    expect(requestJSON).toHaveBeenCalledWith("/api/sessions/sess_browser_automation_01/transcript");
  });

  it("installs a bridge-capable extension and creates deterministic disabled bridge prerequisites", async () => {
    const prepareExtension = vi.fn(async () => ({
      checksum: "bridge-checksum",
      extensionDir: "/tmp/telegram-reference",
      markers: {
        crashOnce: "/tmp/markers/adapter-crash-once.json",
        delivery: "/tmp/markers/adapter-deliveries.jsonl",
        handshake: "/tmp/markers/adapter-handshake.json",
        ingest: "/tmp/markers/adapter-ingest.jsonl",
        ownership: "/tmp/markers/adapter-ownership.json",
        shutdown: "/tmp/markers/adapter-shutdown.log",
        starts: "/tmp/markers/adapter-starts.log",
        state: "/tmp/markers/adapter-states.jsonl",
        updates: "/tmp/markers/adapter-updates.jsonl",
      },
    }));
    const requestJSON = vi.fn(async (pathname: string, init?: RequestInit) => {
      if (pathname === "/api/extensions") {
        return {
          extension: {
            enabled: true,
            health: "healthy",
            name: "telegram-reference",
            state: "active",
          },
        };
      }

      if (pathname === "/api/extensions/telegram-reference") {
        return {
          extension: {
            enabled: true,
            health: "healthy",
            name: "telegram-reference",
            state: "active",
          },
        };
      }

      if (pathname === "/api/bridges/providers") {
        return {
          providers: [
            {
              config_schema: {
                schema: "provider-config",
                version: "2026-04-15",
              },
              description: "Telegram bridge provider",
              display_name: "Telegram",
              enabled: true,
              extension_name: "telegram-reference",
              health: "healthy",
              platform: "telegram",
              secret_slots: [
                {
                  description: "Bot API token",
                  name: "bot_token",
                  required: true,
                },
              ],
              state: "active",
            },
          ],
        };
      }

      if (pathname === "/api/workspaces") {
        return {
          workspaces: [
            {
              id: "ws_browser_01",
              name: "agh-browser-workspace",
              root_dir: "/tmp/agh-browser-workspace",
            },
          ],
        };
      }

      if (pathname === "/api/bridges") {
        return {
          bridge: {
            created_at: "2026-04-17T12:00:00Z",
            display_name: browserBridgeOperatorFlowScenario.bridge.initialName,
            enabled: false,
            extension_name: "telegram-reference",
            id: "brg_browser_01",
            platform: "telegram",
            provider_config: browserBridgeOperatorFlowScenario.bridge.initialProviderConfig,
            routing_policy: {
              include_group: false,
              include_peer: true,
              include_thread: true,
            },
            scope: "workspace",
            status: "disabled",
            updated_at: "2026-04-17T12:00:00Z",
            workspace_id: "ws_browser_01",
          },
          health: {
            auth_failures_total: 0,
            bridge_instance_id: "brg_browser_01",
            delivery_backlog: 0,
            delivery_dropped_total: 0,
            delivery_failures_total: 0,
            route_count: 0,
            status: "disabled",
          },
        };
      }

      if (pathname === "/api/bridges/brg_browser_01") {
        return {
          bridge: {
            created_at: "2026-04-17T12:00:00Z",
            display_name: browserBridgeOperatorFlowScenario.bridge.initialName,
            enabled: false,
            extension_name: "telegram-reference",
            id: "brg_browser_01",
            platform: "telegram",
            provider_config: browserBridgeOperatorFlowScenario.bridge.initialProviderConfig,
            routing_policy: {
              include_group: false,
              include_peer: true,
              include_thread: true,
            },
            scope: "workspace",
            status: "disabled",
            updated_at: "2026-04-17T12:00:00Z",
            workspace_id: "ws_browser_01",
          },
          health: {
            auth_failures_total: 0,
            bridge_instance_id: "brg_browser_01",
            delivery_backlog: 0,
            delivery_dropped_total: 0,
            delivery_failures_total: 0,
            route_count: 0,
            status: "disabled",
          },
        };
      }

      if (pathname === "/api/bridges/brg_browser_01/secret-bindings/bot_token") {
        return {
          binding: {
            binding_name: "bot_token",
            kind: "bot_token",
            vault_ref: `env:${browserBridgeOperatorFlowScenario.secretBinding.envName}`,
          },
        };
      }

      throw new Error(`unexpected request ${pathname} ${init?.method ?? "GET"}`);
    });

    const seeded = await seedBrowserBridgeOperatorFlow(
      {
        requestJSON: requestJSON as BrowserRuntimeSeedClient["requestJSON"],
      },
      { prepareExtension }
    );

    expect(prepareExtension).toHaveBeenCalledTimes(1);
    expect(requestJSON).toHaveBeenCalledWith(
      "/api/extensions",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          checksum: "bridge-checksum",
          path: "/tmp/telegram-reference",
        }),
      })
    );
    expect(requestJSON).toHaveBeenCalledWith(
      "/api/bridges/brg_browser_01/secret-bindings/bot_token",
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({
          kind: "bot_token",
          vault_ref: `env:${browserBridgeOperatorFlowScenario.secretBinding.envName}`,
        }),
      })
    );
    expect(requestJSON).toHaveBeenCalledWith(
      "/api/bridges",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          display_name: browserBridgeOperatorFlowScenario.bridge.initialName,
          enabled: false,
          extension_name: "telegram-reference",
          platform: "telegram",
          provider_config: browserBridgeOperatorFlowScenario.bridge.initialProviderConfig,
          routing_policy: {
            include_group: false,
            include_peer: true,
            include_thread: true,
          },
          scope: "workspace",
          status: "disabled",
          workspace_id: "ws_browser_01",
        }),
      })
    );
    expect(seeded).toMatchObject({
      bridge: {
        display_name: browserBridgeOperatorFlowScenario.bridge.initialName,
        id: "brg_browser_01",
        status: "disabled",
      },
      extension: {
        checksum: "bridge-checksum",
        dir: "/tmp/telegram-reference",
        name: "telegram-reference",
        platform: "telegram",
      },
      health: {
        route_count: 0,
        status: "disabled",
      },
      provider: {
        extension_name: "telegram-reference",
        platform: "telegram",
      },
    });
  });
});
