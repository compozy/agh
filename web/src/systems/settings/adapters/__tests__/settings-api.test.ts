import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { expectFetchRequest, mockJsonResponse } from "@/test/fetch-test-utils";
import {
  deleteSettingsSandbox,
  deleteSettingsMCPServer,
  deleteSettingsProvider,
  disableSettingsExtension,
  enableSettingsExtension,
  getSettingsGeneral,
  getSettingsObservability,
  getSettingsRestartStatus,
  getSettingsSkills,
  listSettingsSandboxes,
  listSettingsExtensions,
  listSettingsHooks,
  listSettingsMCPServers,
  listSettingsProviders,
  putSettingsSandbox,
  putSettingsMCPServer,
  putSettingsProvider,
  SettingsApiError,
  settingsObservabilityLogTailPath,
  triggerSettingsRestart,
  updateSettingsAutomation,
  updateSettingsGeneral,
  updateSettingsSkills,
} from "../settings-api";

const generalSectionFixture = {
  section: "general" as const,
  scope: "global" as const,
  available_scopes: ["global" as const],
  actions: {
    restart: {
      available: true,
      behavior: "action_trigger" as const,
      name: "restart",
    },
  },
  config: {
    daemon: { socket: "/tmp/agh.sock" },
    defaults: { agent: "claude-code" },
    http: { host: "127.0.0.1", port: 2123 },
    limits: { max_concurrent_agents: 4, max_sessions: 16 },
    permissions: { mode: "approve-reads" as const },
    session_timeout: "30m",
  },
  config_paths: {
    daemon_info: "/home/agh/.agh/daemon.json",
    global_config: "/home/agh/.agh/config.toml",
    global_mcp_sidecar: "/home/agh/.agh/mcp.json",
    home_dir: "/home/agh/.agh",
    log_file: "/home/agh/.agh/agh.log",
  },
  runtime: {
    active_agents: 2,
    active_sessions: 3,
    available: true,
    total_sessions: 100,
    uptime_seconds: 12000,
  },
};

const mutationFixture = {
  section: "general" as const,
  scope: "global" as const,
  behavior: "restart_required" as const,
  applied: true,
  restart_required: true,
  restart_scope: "daemon",
  warnings: ["restart the daemon"],
  write_target: "global-config" as const,
};

beforeEach(() => {
  vi.stubGlobal("fetch", vi.fn());
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("SettingsApiError", () => {
  it("captures status code and name", () => {
    const err = new SettingsApiError("boom", 500);
    expect(err.name).toBe("SettingsApiError");
    expect(err.status).toBe(500);
  });
});

describe("section reads and updates", () => {
  it("loads the general section envelope", async () => {
    mockJsonResponse(generalSectionFixture);

    const result = await getSettingsGeneral();

    expect(result).toEqual(generalSectionFixture);
    await expectFetchRequest({ path: "/api/settings/general" });
  });

  it("loads the observability section and exposes log tail URL", async () => {
    const observability = {
      section: "observability" as const,
      scope: "global" as const,
      available_scopes: ["global" as const],
      config: {
        enabled: true,
        max_global_bytes: 1024,
        retention_days: 7,
        transcripts: {
          enabled: true,
          max_bytes_per_session: 4096,
          segment_bytes: 2048,
        },
      },
      log_tail: {
        available: true,
        stream_url: "/api/settings/observability/log-tail",
        transport: "sse" as const,
      },
      runtime: {
        active_agents: 0,
        active_sessions: 0,
        available: true,
        global_db_size_bytes: 128,
        session_db_size_bytes: 64,
        uptime_seconds: 1000,
      },
    };

    mockJsonResponse(observability);

    const result = await getSettingsObservability();

    expect(result.log_tail.available).toBe(true);
    expect(settingsObservabilityLogTailPath()).toBe("/api/settings/observability/log-tail");
    await expectFetchRequest({ path: "/api/settings/observability" });
  });

  it("updates the general section and returns the mutation result", async () => {
    mockJsonResponse(mutationFixture);

    const body = {
      config: {
        daemon: { socket: "/tmp/next.sock" },
        defaults: { agent: "claude-code" },
        http: { host: "127.0.0.1", port: 2123 },
        limits: { max_concurrent_agents: 4, max_sessions: 16 },
        permissions: { mode: "approve-reads" as const },
        session_timeout: "45m",
      },
    };

    const result = await updateSettingsGeneral(body);

    expect(result).toEqual(mutationFixture);
    expect(result.restart_required).toBe(true);
    await expectFetchRequest({
      body,
      method: "PATCH",
      path: "/api/settings/general",
    });
  });

  it("updates the automation section", async () => {
    mockJsonResponse({ ...mutationFixture, section: "automation" as const });

    const body = {
      config: {
        default_fire_limit: { max: 12, window: "1h" },
        enabled: true,
        max_concurrent_jobs: 5,
        timezone: "UTC",
      },
    };

    const result = await updateSettingsAutomation(body);

    expect(result.section).toBe("automation");
    await expectFetchRequest({
      body,
      method: "PATCH",
      path: "/api/settings/automation",
    });
  });

  it("loads the skills section with agent-scope query params", async () => {
    const skillsSection = {
      section: "skills" as const,
      scope: "agent" as const,
      available_scopes: ["global" as const, "agent" as const],
      agent_name: "coder",
      workspace_id: "ws-polybot",
      runtime_available: true,
      discovered_count: 3,
      disabled_count: 1,
      config: {
        enabled: true,
        disabled_skills: ["review"],
        poll_interval: "5m",
        marketplace: { registry: "agh" },
      },
      links: [{ label: "skills", path: "/skills" }],
    };

    mockJsonResponse(skillsSection);

    const result = await getSettingsSkills({
      scope: "agent",
      agent_name: " coder ",
      workspace_id: " ws-polybot ",
    });

    expect(result).toEqual(skillsSection);
    await expectFetchRequest({
      path: "/api/settings/skills?scope=agent&workspace_id=ws-polybot&agent_name=coder",
    });
  });

  it("updates the skills section with agent-scope query params", async () => {
    mockJsonResponse({
      ...mutationFixture,
      section: "skills" as const,
      scope: "agent" as const,
      agent_name: "coder",
      workspace_id: "ws-polybot",
      write_target: "workspace-agent-file" as const,
    });

    const body = {
      config: {
        enabled: true,
        disabled_skills: ["review"],
        poll_interval: "5m",
        marketplace: { registry: "agh" },
      },
    };

    const result = await updateSettingsSkills(body, {
      scope: "agent",
      agent_name: " coder ",
      workspace_id: " ws-polybot ",
    });

    expect(result.scope).toBe("agent");
    expect(result.write_target).toBe("workspace-agent-file");
    await expectFetchRequest({
      body,
      method: "PATCH",
      path: "/api/settings/skills?scope=agent&workspace_id=ws-polybot&agent_name=coder",
    });
  });

  it("throws typed errors on failed section reads", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 500 }));

    await expect(getSettingsGeneral()).rejects.toBeInstanceOf(SettingsApiError);
    await expect(getSettingsGeneral()).rejects.toThrow("Failed to load general settings: 500");
  });
});

describe("collection endpoints", () => {
  it("lists providers", async () => {
    const providers = {
      collection: "providers" as const,
      scope: "global" as const,
      available_scopes: ["global" as const],
      providers: [
        {
          name: "openai",
          default: true,
          command_available: true,
          settings: {
            command: "codex",
            auth_mode: "native_cli",
            env_policy: "filtered",
            home_policy: "operator",
            auth_status_command: "codex auth status",
            auth_login_command: "codex login",
            credential_slots: [
              {
                name: "api_key",
                target_env: "OPENAI_API_KEY",
                secret_ref: "env:OPENAI_API_KEY",
                kind: "api_key",
                required: false,
              },
            ],
          },
          source_metadata: {
            available_targets: ["global-config" as const],
            effective_source: { kind: "global-config" as const, scope: "global" as const },
          },
          auth_status: {
            mode: "native_cli",
            env_policy: "filtered",
            home_policy: "operator",
            state: "native_cli",
          },
        },
      ],
    };

    mockJsonResponse(providers);
    const result = await listSettingsProviders();

    expect(result.providers).toHaveLength(1);
    expect(result.providers[0]?.source_metadata.effective_source.kind).toBe("global-config");
    expect(result.providers[0]?.settings.auth_mode).toBe("native_cli");
    expect(result.providers[0]?.auth_status?.state).toBe("native_cli");
    await expectFetchRequest({ path: "/api/settings/providers" });
  });

  it("puts a provider overlay and reports write target", async () => {
    mockJsonResponse({ ...mutationFixture, section: "general" as const });

    const result = await putSettingsProvider("openai", {
      settings: {
        command: "codex",
        default_model: "gpt-5",
        auth_mode: "native_cli",
        env_policy: "filtered",
        home_policy: "operator",
      },
    });

    expect(result.write_target).toBe("global-config");
    await expectFetchRequest({
      body: {
        settings: {
          command: "codex",
          default_model: "gpt-5",
          auth_mode: "native_cli",
          env_policy: "filtered",
          home_policy: "operator",
        },
      },
      method: "PUT",
      path: "/api/settings/providers/openai",
    });
  });

  it("maps 404 to a not-found delete error for providers", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 404 }));

    await expect(deleteSettingsProvider("missing")).rejects.toThrow("Provider not found: missing");
  });

  it("lists sandboxes", async () => {
    const sandboxes = {
      collection: "sandboxes" as const,
      scope: "global" as const,
      available_scopes: ["global" as const],
      sandboxes: [
        {
          name: "local",
          profile: { backend: "host" },
          workspace_usage_count: 2,
          source_metadata: {
            available_targets: ["global-config" as const],
            effective_source: { kind: "global-config" as const, scope: "global" as const },
          },
        },
      ],
    };

    mockJsonResponse(sandboxes);
    const result = await listSettingsSandboxes();

    expect(result.sandboxes[0]?.workspace_usage_count).toBe(2);
    await expectFetchRequest({ path: "/api/settings/sandboxes" });
  });

  it("deletes sandboxes and propagates mutation responses", async () => {
    mockJsonResponse({ ...mutationFixture, section: "general" as const });

    const result = await deleteSettingsSandbox("local");

    expect(result.applied).toBe(true);
    await expectFetchRequest({
      method: "DELETE",
      path: "/api/settings/sandboxes/local",
    });
  });

  it("puts sandboxes", async () => {
    mockJsonResponse({ ...mutationFixture, section: "general" as const });

    const body = { profile: { backend: "daytona" } };
    await putSettingsSandbox("cloud", body);

    await expectFetchRequest({
      body,
      method: "PUT",
      path: "/api/settings/sandboxes/cloud",
    });
  });

  it("lists hooks", async () => {
    const hooks = {
      collection: "hooks" as const,
      scope: "global" as const,
      available_scopes: ["global" as const],
      hooks: [],
    };

    mockJsonResponse(hooks);
    await listSettingsHooks();
    await expectFetchRequest({ path: "/api/settings/hooks" });
  });

  it("lists MCP servers with scope filter and preserves workspace identifier", async () => {
    const mcp = {
      collection: "mcp-servers" as const,
      scope: "workspace" as const,
      available_scopes: ["global" as const, "workspace" as const],
      mcp_servers: [],
      workspace_id: "ws_alpha",
    };

    mockJsonResponse(mcp);

    await listSettingsMCPServers({ scope: "workspace", workspace_id: "  ws_alpha  " });

    await expectFetchRequest({
      path: "/api/settings/mcp-servers?scope=workspace&workspace_id=ws_alpha",
    });
  });

  it("puts MCP server with target selector", async () => {
    mockJsonResponse({ ...mutationFixture, section: "general" as const });

    await putSettingsMCPServer(
      "github",
      { server: { name: "github", command: "gh" } },
      { scope: "global", target: "sidecar" }
    );

    await expectFetchRequest({
      body: { server: { name: "github", command: "gh" } },
      method: "PUT",
      path: "/api/settings/mcp-servers/github?scope=global&target=sidecar",
    });
  });

  it("deletes MCP server with auto target", async () => {
    mockJsonResponse({ ...mutationFixture, section: "general" as const });

    await deleteSettingsMCPServer("github", { scope: "global", target: "auto" });

    await expectFetchRequest({
      method: "DELETE",
      path: "/api/settings/mcp-servers/github?scope=global&target=auto",
    });
  });
});

describe("restart action", () => {
  it("triggers restart and returns operation info", async () => {
    mockJsonResponse(
      {
        operation_id: "op_001",
        status: "pending" as const,
        status_url: "/api/settings/actions/restart/op_001",
        active_session_count: 2,
      },
      { status: 202 }
    );

    const result = await triggerSettingsRestart();

    expect(result.operation_id).toBe("op_001");
    expect(result.status).toBe("pending");
    await expectFetchRequest({
      method: "POST",
      path: "/api/settings/actions/restart",
    });
  });

  it("gets restart status by operation id", async () => {
    mockJsonResponse({
      operation_id: "op_001",
      status: "ready" as const,
      old_pid: 1000,
      old_socket_path: "/tmp/agh.sock",
      old_started_at: "2026-04-17T10:00:00Z",
      active_session_count: 0,
      started_at: "2026-04-17T10:05:00Z",
      updated_at: "2026-04-17T10:05:05Z",
    });

    const result = await getSettingsRestartStatus("op_001");
    expect(result.status).toBe("ready");
    await expectFetchRequest({
      path: "/api/settings/actions/restart/op_001",
    });
  });

  it("maps 404 to a typed not-found error", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 404 }));
    await expect(getSettingsRestartStatus("missing")).rejects.toThrow(
      "Restart operation not found: missing"
    );
  });
});

describe("extension operational actions", () => {
  const extensionFixture = {
    name: "daytona",
    enabled: true,
    version: "1.2.3",
    state: "running",
    source: "marketplace",
    type: "backend",
    daemon_running: true,
    health: "healthy",
    requires_env: ["DAYTONA_TOKEN"],
    missing_env: ["DAYTONA_TOKEN"],
  };

  it("lists installed extensions through the HTTP endpoint", async () => {
    mockJsonResponse({ extensions: [extensionFixture] });

    const result = await listSettingsExtensions();

    expect(result).toEqual([extensionFixture]);
    expect(result[0]?.missing_env).toEqual(["DAYTONA_TOKEN"]);
    await expectFetchRequest({ path: "/api/extensions" });
  });

  it("enables an extension and returns the updated record", async () => {
    mockJsonResponse({ extension: extensionFixture });

    const result = await enableSettingsExtension("daytona");

    expect(result.enabled).toBe(true);
    await expectFetchRequest({
      method: "POST",
      path: "/api/extensions/daytona/enable",
    });
  });

  it("disables an extension and returns the updated record", async () => {
    mockJsonResponse({ extension: { ...extensionFixture, enabled: false } });

    const result = await disableSettingsExtension("daytona");

    expect(result.enabled).toBe(false);
    await expectFetchRequest({
      method: "POST",
      path: "/api/extensions/daytona/disable",
    });
  });

  it("maps 404 responses to a typed not-found error for enable and disable", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 404 }));
    await expect(enableSettingsExtension("missing")).rejects.toThrow(
      "Extension not found: missing"
    );

    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 404 }));
    await expect(disableSettingsExtension("missing")).rejects.toThrow(
      "Extension not found: missing"
    );
  });
});
