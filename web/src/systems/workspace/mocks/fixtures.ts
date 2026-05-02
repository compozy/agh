import type { WorkspaceDetailPayload, WorkspacePayload } from "@/systems/workspace/types";

export const workspaceFixtures: WorkspacePayload[] = [
  {
    id: "ws_home",
    root_dir: "/workspaces/home",
    add_dirs: [],
    name: "home",
    created_at: "2026-04-15T09:00:00Z",
    updated_at: "2026-04-17T17:40:00Z",
  },
  {
    id: "ws_storybook",
    root_dir: "/workspaces/agh2",
    add_dirs: ["/workspaces/shared"],
    name: "agh2",
    created_at: "2026-04-16T09:00:00Z",
    updated_at: "2026-04-17T17:42:00Z",
  },
];

export const primaryWorkspaceFixture: WorkspacePayload = workspaceFixtures[1];

export const workspaceDetailFixture: WorkspaceDetailPayload = {
  workspace: primaryWorkspaceFixture,
  agents: [
    {
      name: "claude-agent",
      provider: "claude",
      prompt: "Review recent changes and explain the trade-offs.",
    },
    {
      name: "codex-agent",
      provider: "codex",
      prompt: "Own implementation details and verification for coding tasks.",
    },
  ],
  providers: [
    {
      name: "claude",
      display_name: "Claude Code",
      harness: "acp",
      runtime_provider: "claude",
      default_model: "claude-sonnet-4-6",
    },
    {
      name: "codex",
      display_name: "Codex",
      harness: "acp",
      runtime_provider: "codex",
      default_model: "gpt-5.4",
    },
    {
      name: "blackbox",
      display_name: "BLACKBOX AI",
      harness: "acp",
      runtime_provider: "blackbox",
    },
    { name: "cline", display_name: "Cline", harness: "acp", runtime_provider: "cline" },
    { name: "goose", display_name: "Goose", harness: "acp", runtime_provider: "goose" },
    { name: "hermes", display_name: "Hermes", harness: "acp", runtime_provider: "hermes" },
    {
      name: "junie",
      display_name: "Junie",
      harness: "acp",
      runtime_provider: "junie",
    },
    {
      name: "kimi-cli",
      display_name: "Kimi CLI",
      harness: "acp",
      runtime_provider: "kimi-cli",
    },
    {
      name: "openclaw",
      display_name: "OpenClaw",
      harness: "acp",
      runtime_provider: "openclaw",
    },
    {
      name: "openhands",
      display_name: "OpenHands",
      harness: "acp",
      runtime_provider: "openhands",
    },
    {
      name: "qoder",
      display_name: "Qoder CLI",
      harness: "acp",
      runtime_provider: "qoder",
    },
    {
      name: "qwen-code",
      display_name: "Qwen Code",
      harness: "acp",
      runtime_provider: "qwen-code",
      default_model: "qwen3.6-plus",
    },
    {
      name: "openrouter",
      display_name: "OpenRouter",
      harness: "pi_acp",
      runtime_provider: "openrouter",
      default_model: "openai/gpt-5.4",
    },
  ],
  sessions: [
    {
      id: "sess-storybook",
      name: "Storybook rollout",
      agent_name: "codex-agent",
      provider: "codex",
      workspace_id: primaryWorkspaceFixture.id,
      workspace_path: primaryWorkspaceFixture.root_dir,
      state: "active",
      created_at: "2026-04-17T16:00:00Z",
      updated_at: "2026-04-17T17:55:00Z",
    },
  ],
  skills: [
    {
      name: "storybook-stories",
      source: "workspace",
      dir: `${primaryWorkspaceFixture.root_dir}/.agents/skills/storybook-stories`,
    },
  ],
};
