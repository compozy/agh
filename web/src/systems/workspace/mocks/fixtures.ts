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
