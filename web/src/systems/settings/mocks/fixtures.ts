import type {
  SettingsAutomationSection,
  SettingsSandboxEntry,
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
import {
  storyAgentNames,
  storyCompany,
  storyHeroNetworkChannel,
  storyWorkspacePaths,
} from "@/storybook/fintech-scenario";

export const settingsGeneralSectionFixture: SettingsGeneralSection = {
  section: "general",
  scope: "global",
  available_scopes: ["global"],
  actions: {
    restart: { available: true, behavior: "action_trigger", name: "restart" },
  },
  config: {
    daemon: { socket: "/tmp/agh.sock" },
    defaults: { agent: storyAgentNames.product, provider: "claude", sandbox: "local" },
    http: { host: "127.0.0.1", port: 2123 },
    limits: { max_concurrent_agents: 20 },
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
    default_channel: storyHeroNetworkChannel,
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

export const settingsMemoryConfigFixture: SettingsMemorySection["config"] = {
  controller: {
    default_op_on_fail: "noop",
    llm: {
      enabled: true,
      max_tokens_out: 256,
      model: "anthropic/claude-haiku-4",
      prompt_version: "v1",
      timeout: "250ms",
      top_k: 5,
    },
    max_latency: "300ms",
    mode: "hybrid",
    policy: {
      allow_origins: ["cli", "http", "uds", "tool", "extractor", "dreaming", "file", "provider"],
      max_content_chars: 4096,
      max_writes_per_min: 60,
    },
  },
  daily: {
    archive_path: "_system/archive",
    cold_archive_days: 30,
    dreaming_window: 7,
    hard_delete_days: 0,
    max_archive_bytes: 1073741824,
    max_bytes: 1048576,
    max_lines: 5000,
    rotate_format: "{date}.{seq}.md",
    sweep_hour: 3,
  },
  decisions: {
    keep_audit_summary: true,
    max_post_content_bytes: 65536,
    prune_after_applied_days: 90,
  },
  dream: {
    agent: storyAgentNames.compliance,
    check_interval: "30m",
    debounce: "10m",
    enabled: true,
    gates: {
      min_recall_count: 2,
      min_score: 0.75,
      min_unpromoted: 5,
    },
    min_hours: 24,
    min_sessions: 3,
    prompt_version: "v1",
    scoring: {
      recency_half_life_days: 14,
      weights: {
        frequency: 0.3,
        freshness: 0.15,
        recency: 0.2,
        relevance: 0.35,
      },
    },
  },
  enabled: true,
  extractor: {
    deadline: "60s",
    dlq_path: "~/.agh/memory/_system/extractor/failures",
    enabled: true,
    inbox_path: "~/.agh/memory/_inbox",
    mode: "post_message",
    model: "",
    queue: {
      capacity: 1,
      coalesce_max: 16,
    },
    sandbox_inbox_only: true,
    throttle_turns: 1,
  },
  file: {
    max_bytes: 25600,
    max_lines: 200,
  },
  global_dir: "~/.agh/memory",
  provider: {
    cooldown: "30s",
    failure_threshold: 5,
    name: "",
    timeout: "2s",
  },
  recall: {
    freshness: {
      banner_after_days: 1,
    },
    fusion: "weighted",
    include_already_surfaced: false,
    include_system: false,
    raw_candidates: 50,
    signals: {
      metrics_enabled: true,
      queue_capacity: 256,
      worker_retry_max: 3,
    },
    top_k: 5,
    weights: {
      bm25_trigram: 0.2,
      bm25_unicode: 0.55,
      recall_signal: 0.1,
      recency: 0.15,
    },
  },
  session: {
    cold_archive_days: 30,
    events_purge_grace: "24h",
    hard_delete_days: 0,
    ledger_format: "jsonl",
    ledger_root: "~/.agh/sessions",
    max_archive_bytes: 10737418240,
    unbound_partition: "_unbound",
  },
  workspace: {
    auto_create: true,
    toml_path: "<workspace>/.agh/workspace.toml",
  },
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
  config: settingsMemoryConfigFixture,
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
      base_url: storyCompany.registryBaseUrl,
    },
    allowed_marketplace_mcp: ["merchant-docs"],
    allowed_marketplace_hooks: [],
  },
  links: [{ label: "skills", path: "/skills" }],
};

export const settingsHooksExtensionsSectionFixture: SettingsHooksExtensionsSection = {
  section: "hooks-extensions",
  scope: "global",
  available_scopes: ["global"],
  config: {
    marketplace: { registry: "northstar", base_url: storyCompany.hooksMarketplaceBaseUrl },
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
        matcher: { agent_name: storyAgentNames.support },
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
      requires_env: ["DAYTONA_TOKEN"],
      missing_env: ["DAYTONA_TOKEN"],
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
    command_available: true,
    settings: {
      command: "npx -y @agentclientprotocol/claude-agent-acp@latest",
      display_name: "Claude Code",
      models: {
        default: "claude-sonnet-4-6",
        curated: [{ id: "claude-sonnet-4-6" }, { id: "claude-haiku-4-5" }],
      },
      harness: "acp",
      auth_mode: "native_cli",
      env_policy: "filtered",
      home_policy: "operator",
      auth_status_command: "claude auth status",
      auth_login_command: "claude login",
    },
    auth_status: {
      mode: "native_cli",
      env_policy: "filtered",
      home_policy: "operator",
      state: "native_cli",
      message: "Provider owns authentication through its native CLI login state.",
      status_command: "claude auth status",
      login_command: "claude login",
    },
    source_metadata: {
      available_targets: ["global-config"],
      effective_source: { kind: "global-config", scope: "global" },
      shadowed_sources: [{ kind: "builtin-provider", scope: "global" }],
    },
    fallback: {
      settings: {
        command: "npx -y @agentclientprotocol/claude-agent-acp@latest",
        models: {
          default: "claude-sonnet-4-6",
          curated: [{ id: "claude-sonnet-4-6" }, { id: "claude-haiku-4-5" }],
        },
        auth_mode: "native_cli",
        env_policy: "filtered",
        home_policy: "operator",
      },
      source: { kind: "builtin-provider", scope: "global" },
    },
  },
  {
    name: "codex",
    default: false,
    command_available: true,
    settings: {
      command: "npx -y @zed-industries/codex-acp@latest",
      models: {
        default: "gpt-5.4",
        curated: [
          {
            id: "gpt-5.4",
            supports_reasoning: true,
            reasoning_efforts: ["low", "medium", "high"],
            default_reasoning_effort: "medium",
          },
          { id: "gpt-5.4-mini" },
        ],
      },
      harness: "acp",
      auth_mode: "native_cli",
      env_policy: "filtered",
      home_policy: "operator",
      auth_status_command: "codex auth status",
      auth_login_command: "codex login",
    },
    auth_status: {
      mode: "native_cli",
      env_policy: "filtered",
      home_policy: "operator",
      state: "native_cli",
      message: "Provider owns authentication through its native CLI login state.",
      status_command: "codex auth status",
      login_command: "codex login",
    },
    source_metadata: {
      available_targets: ["global-config"],
      effective_source: { kind: "builtin-provider", scope: "global" },
    },
  },
  {
    name: "openrouter",
    default: false,
    command_available: true,
    settings: {
      command: "npx -y pi-acp@latest",
      display_name: "OpenRouter",
      models: {
        default: "openai/gpt-5.4",
        curated: [
          { id: "openai/gpt-5.4", supports_reasoning: true },
          { id: "anthropic/claude-sonnet-4-6" },
        ],
      },
      harness: "pi_acp",
      runtime_provider: "openrouter",
      auth_mode: "bound_secret",
      env_policy: "filtered",
      home_policy: "operator",
      credential_slots: [
        {
          name: "api_key",
          target_env: "OPENROUTER_API_KEY",
          secret_ref: "env:OPENROUTER_API_KEY",
          kind: "api_key",
          required: true,
        },
      ],
    },
    auth_status: {
      mode: "bound_secret",
      env_policy: "filtered",
      home_policy: "operator",
      state: "missing_required",
      message: "Missing required AGH-managed provider credential.",
    },
    credentials: [
      {
        name: "api_key",
        target_env: "OPENROUTER_API_KEY",
        secret_ref: "env:OPENROUTER_API_KEY",
        kind: "api_key",
        required: true,
        present: false,
        source: "env",
      },
    ],
    source_metadata: {
      available_targets: ["global-config"],
      effective_source: { kind: "builtin-provider", scope: "global" },
    },
  },
  {
    name: "blackbox",
    default: false,
    command_available: true,
    settings: {
      command: "blackbox --experimental-acp",
      display_name: "BLACKBOX AI",
      harness: "acp",
      auth_mode: "native_cli",
      env_policy: "filtered",
      home_policy: "operator",
    },
    auth_status: {
      mode: "native_cli",
      env_policy: "filtered",
      home_policy: "operator",
      state: "native_cli",
      message: "Provider owns authentication through its native CLI login state.",
    },
    source_metadata: {
      available_targets: ["global-config"],
      effective_source: { kind: "builtin-provider", scope: "global" },
    },
  },
  {
    name: "cline",
    default: false,
    command_available: true,
    settings: { command: "npx -y cline@latest --acp", display_name: "Cline", harness: "acp" },
    source_metadata: {
      available_targets: ["global-config"],
      effective_source: { kind: "builtin-provider", scope: "global" },
    },
  },
  {
    name: "goose",
    default: false,
    command_available: true,
    settings: { command: "goose acp", display_name: "Goose", harness: "acp" },
    source_metadata: {
      available_targets: ["global-config"],
      effective_source: { kind: "builtin-provider", scope: "global" },
    },
  },
  {
    name: "hermes",
    default: false,
    command_available: true,
    settings: { command: "hermes acp", display_name: "Hermes", harness: "acp" },
    source_metadata: {
      available_targets: ["global-config"],
      effective_source: { kind: "builtin-provider", scope: "global" },
    },
  },
  {
    name: "junie",
    default: false,
    command_available: true,
    settings: {
      command: "junie --acp true",
      display_name: "Junie",
      harness: "acp",
    },
    source_metadata: {
      available_targets: ["global-config"],
      effective_source: { kind: "builtin-provider", scope: "global" },
    },
  },
  {
    name: "kimi-cli",
    default: false,
    command_available: true,
    settings: {
      command: "kimi acp",
      display_name: "Kimi CLI",
      harness: "acp",
      auth_mode: "native_cli",
      env_policy: "filtered",
      home_policy: "operator",
    },
    auth_status: {
      mode: "native_cli",
      env_policy: "filtered",
      home_policy: "operator",
      state: "native_cli",
      message: "Provider owns authentication through its native CLI login state.",
    },
    source_metadata: {
      available_targets: ["global-config"],
      effective_source: { kind: "builtin-provider", scope: "global" },
    },
  },
  {
    name: "openclaw",
    default: false,
    command_available: true,
    settings: { command: "openclaw acp", display_name: "OpenClaw", harness: "acp" },
    source_metadata: {
      available_targets: ["global-config"],
      effective_source: { kind: "builtin-provider", scope: "global" },
    },
  },
  {
    name: "openhands",
    default: false,
    command_available: true,
    settings: { command: "openhands acp", display_name: "OpenHands", harness: "acp" },
    source_metadata: {
      available_targets: ["global-config"],
      effective_source: { kind: "builtin-provider", scope: "global" },
    },
  },
  {
    name: "qoder",
    default: false,
    command_available: true,
    settings: {
      command: "npx -y @qoder-ai/qodercli@latest --acp",
      display_name: "Qoder CLI",
      harness: "acp",
      auth_mode: "native_cli",
      env_policy: "filtered",
      home_policy: "operator",
    },
    auth_status: {
      mode: "native_cli",
      env_policy: "filtered",
      home_policy: "operator",
      state: "native_cli",
      message: "Provider owns authentication through its native CLI login state.",
    },
    source_metadata: {
      available_targets: ["global-config"],
      effective_source: { kind: "builtin-provider", scope: "global" },
    },
  },
  {
    name: "qwen-code",
    default: false,
    command_available: true,
    settings: {
      command: "npx -y @qwen-code/qwen-code@latest --acp --experimental-skills",
      display_name: "Qwen Code",
      models: { default: "qwen3.6-plus", curated: [{ id: "qwen3.6-plus" }] },
      harness: "acp",
    },
    source_metadata: {
      available_targets: ["global-config"],
      effective_source: { kind: "builtin-provider", scope: "global" },
    },
  },
];

export const settingsSandboxFixtures: SettingsSandboxEntry[] = [
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
    args: [storyWorkspacePaths.risk],
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
    requires_env: ["DAYTONA_TOKEN"],
    missing_env: ["DAYTONA_TOKEN"],
  },
];

export const settingsProvidersCollectionFixture = {
  providers: settingsProviderFixtures,
};

export const settingsSandboxesCollectionFixture = {
  sandboxes: settingsSandboxFixtures,
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
  operation_id: "restart_northstar_pay",
  status: "pending",
  active_session_count: 2,
  status_url: "/api/settings/actions/restart/restart_northstar_pay",
};

export const settingsRestartStatusFixture: SettingsRestartStatus = {
  operation_id: "restart_northstar_pay",
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
