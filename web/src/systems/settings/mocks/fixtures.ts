import type {
  SettingsAutomationSection,
  SettingsEnvironmentEntry,
  SettingsExtensionEntry,
  SettingsGeneralSection,
  SettingsHookEntry,
  SettingsHooksExtensionsSection,
  SettingsMCPServerEntry,
  SettingsMemorySection,
  SettingsMutationResult,
  SettingsNetworkSection,
  SettingsObservabilitySection,
  SettingsProviderEntry,
  SettingsRestartResponse,
  SettingsRestartStatus,
  SettingsSkillsSection,
} from "@/systems/settings";

export const settingsGeneralSectionFixture: SettingsGeneralSection = {
  section: "general",
  scope: "global",
  available_scopes: ["global"],
  actions: {
    restart: { available: true, behavior: "action_trigger", name: "restart" },
  },
  config: {
    daemon: { socket: "/tmp/agh.sock" },
    defaults: { agent: "general", provider: "claude", environment: "local" },
    http: { host: "127.0.0.1", port: 2123 },
    limits: { max_sessions: 10, max_concurrent_agents: 20 },
    permissions: { mode: "approve-all" },
    session_timeout: "0s",
  },
  config_paths: {
    daemon_info: "/tmp/daemon.json",
    global_config: "~/.agh/config.toml",
    global_mcp_sidecar: "~/.agh/mcp.json",
    home_dir: "~/.agh",
    log_file: "~/.agh/agh.log",
  },
  runtime: {
    active_agents: 7,
    active_sessions: 4,
    available: true,
    http_host: "127.0.0.1",
    http_port: 2123,
    socket: "~/.agh/daemon.sock",
    total_sessions: 12,
    uptime_seconds: 3600,
  },
};

export const settingsNetworkSectionFixture: SettingsNetworkSection = {
  section: "network",
  scope: "global",
  available_scopes: ["global"],
  config: {
    enabled: true,
    port: 4222,
    default_channel: "agh",
    greet_interval: 30,
    max_payload: 131072,
    max_queue_depth: 1024,
    max_replay_age: 86400,
  },
  runtime: {
    available: true,
    enabled: true,
    status: "ready",
    listener_host: "127.0.0.1",
    listener_port: 4222,
    local_peers: 2,
    remote_peers: 1,
    channels: 4,
    queued_messages: 7,
    queued_sessions: 0,
    delivery_workers: 3,
  },
  links: [{ label: "network", path: "/network" }],
};

export const settingsAutomationSectionFixture: SettingsAutomationSection = {
  section: "automation",
  scope: "global",
  available_scopes: ["global"],
  config: {
    enabled: true,
    timezone: "UTC",
    max_concurrent_jobs: 8,
    default_fire_limit: { max: 5, window: "1m" },
  },
  runtime: {
    available: true,
    running: true,
    scheduler_running: true,
    job_enabled: 3,
    job_total: 5,
    trigger_enabled: 2,
    trigger_total: 4,
    last_synced_at: "2026-04-17T10:00:00Z",
    next_fire: "2026-04-17T12:00:00Z",
  },
  links: [
    { label: "jobs", path: "/jobs" },
    { label: "triggers", path: "/triggers" },
  ],
};

export const settingsMemorySectionFixture: SettingsMemorySection = {
  section: "memory",
  scope: "global",
  available_scopes: ["global"],
  actions: {
    consolidate: {
      available: true,
      behavior: "action_trigger",
      name: "consolidate",
    },
  },
  config: {
    dream: {
      agent: "general",
      check_interval: "30m",
      enabled: true,
      min_hours: 24,
      min_sessions: 3,
    },
    enabled: true,
    global_dir: "~/.agh/memory",
  },
  health: {
    available: true,
    dream_enabled: true,
    file_count: 42,
    last_consolidated_at: "2026-04-17T14:00:00Z",
  },
};

export const settingsObservabilitySectionFixture: SettingsObservabilitySection = {
  section: "observability",
  scope: "global",
  available_scopes: ["global"],
  config: {
    enabled: true,
    max_global_bytes: 1024 * 1024 * 1024,
    retention_days: 7,
    transcripts: {
      enabled: true,
      max_bytes_per_session: 256 * 1024 * 1024,
      segment_bytes: 1024 * 1024,
    },
  },
  log_tail: {
    available: true,
    stream_url: "/api/settings/observability/log-tail",
    transport: "sse",
  },
  runtime: {
    active_agents: 2,
    active_sessions: 4,
    available: true,
    global_db_size_bytes: 180 * 1024 * 1024,
    session_db_size_bytes: 132 * 1024 * 1024,
    uptime_seconds: 3600,
  },
};

export const settingsSkillsSectionFixture: SettingsSkillsSection = {
  section: "skills",
  scope: "global",
  available_scopes: ["global"],
  runtime_available: true,
  discovered_count: 12,
  disabled_count: 2,
  config: {
    enabled: true,
    disabled_skills: ["alpha", "beta"],
    poll_interval: "5m",
    marketplace: {
      registry: "agh",
      base_url: "https://registry.example.com",
    },
    allowed_marketplace_mcp: ["mcp-one"],
    allowed_marketplace_hooks: [],
  },
  links: [{ label: "skills", path: "/skills" }],
};

export const settingsHooksExtensionsSectionFixture: SettingsHooksExtensionsSection = {
  section: "hooks-extensions",
  scope: "global",
  available_scopes: ["global"],
  config: {
    marketplace: { registry: "github", base_url: "https://api.github.com" },
    resources: {
      allowed_kinds: ["snapshot", "artifact"],
      max_scope: "workspace",
      snapshot_rate_limit: { queue: 100, requests: 30, window: "5m" },
      operator_write_rate_limit: { queue: 20, requests: 10, window: "1m" },
    },
  },
  hooks: [
    {
      name: "pre-commit-lint",
      declaration: {
        name: "pre-commit-lint",
        event: "tool.pre_call",
        mode: "sync",
        command: "make",
        args: ["lint"],
        matcher: { tool_name: "Bash" },
        required: true,
      },
      source_metadata: {
        available_targets: ["global-config"],
        effective_source: { kind: "global-config", scope: "global" },
      },
    },
    {
      name: "slack-notify",
      declaration: {
        name: "slack-notify",
        event: "permission.denied",
        mode: "async",
        command: "node",
        args: ["./hooks/slack.js"],
        matcher: { agent_name: "coder" },
        required: false,
      },
      source_metadata: {
        available_targets: ["global-config"],
        effective_source: { kind: "global-config", scope: "global" },
      },
    },
  ],
  installed: [
    {
      name: "daytona",
      enabled: true,
      version: "1.2.3",
      state: "running",
      health: "healthy",
    },
  ],
  transport_parity: {
    known: true,
    settings_http: true,
    settings_uds: true,
    extensions_http: true,
    extensions_uds: true,
  },
};

export const settingsProviderFixtures: SettingsProviderEntry[] = [
  {
    name: "claude",
    default: true,
    api_key_env_present: true,
    command_available: true,
    settings: {
      command: "npx claude",
      default_model: "claude-opus",
      api_key_env: "ANTHROPIC_API_KEY",
    },
    source_metadata: {
      available_targets: ["global-config"],
      effective_source: { kind: "global-config", scope: "global" },
      shadowed_sources: [{ kind: "builtin-provider", scope: "global" }],
    },
    fallback: {
      settings: { command: "npx claude" },
      source: { kind: "builtin-provider", scope: "global" },
    },
  },
  {
    name: "codex",
    default: false,
    api_key_env_present: false,
    command_available: true,
    settings: {
      command: "npx codex",
      api_key_env: "OPENAI_API_KEY",
    },
    source_metadata: {
      available_targets: ["global-config"],
      effective_source: { kind: "builtin-provider", scope: "global" },
    },
  },
];

export const settingsEnvironmentFixtures: SettingsEnvironmentEntry[] = [
  {
    name: "local",
    workspace_usage_count: 3,
    profile: {
      backend: "local",
      sync_mode: "none",
      persistence: "transient",
      runtime_root: "~",
    },
    source_metadata: {
      available_targets: ["global-config"],
      effective_source: { kind: "global-config", scope: "global" },
    },
  },
  {
    name: "builtin-local",
    workspace_usage_count: 0,
    profile: {
      backend: "local",
      sync_mode: "none",
      persistence: "transient",
      runtime_root: "~",
    },
    source_metadata: {
      available_targets: ["global-config"],
      effective_source: { kind: "builtin-provider", scope: "global" },
    },
  },
];

export const settingsMCPServerFixtures: SettingsMCPServerEntry[] = [
  {
    name: "filesystem",
    transport: "stdio",
    command: "npx -y @modelcontextprotocol/server-filesystem",
    args: ["~/Dev"],
    scope: "global",
    source_metadata: {
      available_targets: ["global-mcp-sidecar", "global-config"],
      effective_source: { kind: "global-mcp-sidecar", scope: "global" },
      shadowed_sources: [{ kind: "global-config", scope: "global" }],
    },
  },
  {
    name: "github",
    transport: "stdio",
    command: "npx -y @modelcontextprotocol/server-github",
    env: { GITHUB_TOKEN: "env:GITHUB_TOKEN" },
    scope: "global",
    source_metadata: {
      available_targets: ["global-mcp-sidecar"],
      effective_source: { kind: "global-mcp-sidecar", scope: "global" },
    },
  },
];

export const settingsHookFixtures: SettingsHookEntry[] =
  settingsHooksExtensionsSectionFixture.hooks ?? [];

export const settingsExtensionFixtures: SettingsExtensionEntry[] = [
  {
    name: "daytona",
    enabled: true,
    version: "1.2.3",
    state: "ready",
    source: "marketplace",
    type: "backend",
    daemon_running: true,
    health: "healthy",
  },
];

export const settingsProvidersCollectionFixture = {
  providers: settingsProviderFixtures,
};

export const settingsEnvironmentsCollectionFixture = {
  environments: settingsEnvironmentFixtures,
};

export const settingsHooksCollectionFixture = {
  hooks: settingsHookFixtures,
};

export const settingsMCPServersCollectionFixture = {
  mcp_servers: settingsMCPServerFixtures,
};

export const settingsExtensionsCollectionFixture = {
  extensions: settingsExtensionFixtures,
};

export const settingsRestartResponseFixture: SettingsRestartResponse = {
  operation_id: "restart_storybook",
  status: "pending",
  active_session_count: 2,
  status_url: "/api/settings/actions/restart/restart_storybook",
};

export const settingsRestartStatusFixture: SettingsRestartStatus = {
  operation_id: "restart_storybook",
  status: "ready",
  active_session_count: 0,
  old_pid: 1000,
  old_socket_path: "/tmp/agh.sock",
  old_started_at: "2026-04-17T10:00:00Z",
  started_at: "2026-04-17T10:05:00Z",
  updated_at: "2026-04-17T10:05:05Z",
};

export const settingsAppliedMutationFixture: SettingsMutationResult = {
  applied: true,
  section: "general",
  scope: "global",
  behavior: "applied_now",
  restart_required: false,
  restart_scope: "none",
  warnings: [],
};

export const settingsRestartRequiredMutationFixture: SettingsMutationResult = {
  applied: true,
  section: "general",
  scope: "global",
  behavior: "restart_required",
  restart_required: true,
  restart_scope: "daemon",
  warnings: [],
};
