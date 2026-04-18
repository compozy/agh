import type {
  MemoryConsolidateResponse,
  MemoryHeader,
  MemoryMutationResponse,
  MemoryReadResponse,
} from "../types";

export const memoryHeadersFixture: MemoryHeader[] = [
  {
    filename: "global/user-role.md",
    mod_time: "2026-04-17T17:30:00Z",
    name: "User Role",
    type: "user",
    description: "Guidance that shapes the assistant's tone and ownership.",
  },
  {
    filename: "workspace/project-context.md",
    mod_time: "2026-04-17T16:10:00Z",
    name: "Project Context",
    type: "project",
    description: "Workspace-local notes about Storybook rollout decisions.",
    agent_name: "codex-agent",
  },
  {
    filename: "workspace/release-checklist.md",
    mod_time: "2026-04-17T14:45:00Z",
    name: "Release Checklist",
    type: "reference",
    description: "Operational checklist for release verification.",
  },
];

export const memoryReadFixtures: Record<string, MemoryReadResponse> = {
  "global/user-role.md": {
    content:
      "# User Role\n\nYou own the outcome end to end.\n\n- Prefer direct fixes.\n- Verify before handoff.\n",
  },
  "workspace/project-context.md": {
    content:
      "# Project Context\n\nThe Storybook rollout uses dual instances and per-system MSW fixtures.\n",
  },
  "workspace/release-checklist.md": {
    content:
      "# Release Checklist\n\n1. Run web lint.\n2. Run web typecheck.\n3. Build both Storybooks.\n",
  },
};

export const memoryMutationFixture: MemoryMutationResponse = {
  ok: true,
};

export const memoryConsolidationFixture: MemoryConsolidateResponse = {
  triggered: true,
  reason: "storybook-fixture",
};
