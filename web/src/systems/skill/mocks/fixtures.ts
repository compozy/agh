import type { SkillActionResponse, SkillPayload } from "../types";

export const skillFixtures: SkillPayload[] = [
  {
    name: "storybook-stories",
    description: "Create, update, or refactor Storybook stories following repo conventions.",
    source: "workspace",
    enabled: true,
    dir: "/Users/pedro/Dev/compozy/agh2/.agents/skills/storybook-stories",
    version: "1.0.0",
    metadata: {
      tags: ["ui", "storybook", "documentation"],
      downloads: 128,
    },
    provenance: {
      installed_at: "2026-04-17T17:00:00Z",
      registry: "workspace",
      slug: "workspace",
      version: "1.0.0",
    },
  },
  {
    name: "no-workarounds",
    description: "Enforce root-cause fixes over workarounds, hacks, and symptom patches.",
    source: "bundled",
    enabled: true,
    dir: "/Users/pedro/Dev/compozy/agh2/.agents/skills/no-workarounds",
    metadata: {
      tags: ["quality", "review"],
      downloads: 512,
    },
  },
  {
    name: "code-review",
    description: "Marketplace skill for structured review and remediation workflows.",
    source: "marketplace",
    enabled: false,
    dir: "/Users/pedro/.codex/skills/code-review",
    version: "0.9.1",
    metadata: {
      tags: ["testing", "review"],
      downloads: 64,
    },
    provenance: {
      installed_at: "2026-04-17T15:00:00Z",
      registry: "community",
      slug: "community",
      version: "0.9.1",
    },
  },
];

export const primarySkillFixture: SkillPayload = skillFixtures[0];

export const skillContentFixtures: Record<string, string> = {
  "storybook-stories":
    "# Storybook Stories\n\nUse this skill when adding or updating Storybook stories.\n",
  "no-workarounds": "# No Workarounds\n\nFix the disease, not the symptom.\n",
  "code-review": "# Code Review\n\nRun a high-signal code review with actionable findings.\n",
};

export const skillActionFixture: SkillActionResponse = {
  ok: true,
};
