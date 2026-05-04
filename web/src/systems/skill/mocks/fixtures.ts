import type { SkillActionResponse, SkillPayload } from "../types";
import {
  storySkillNames,
  storyWorkspacePaths,
  storyWorkspaceSkillDir,
} from "@/storybook/fintech-scenario";

export const skillFixtures: SkillPayload[] = [
  {
    name: storySkillNames.executiveBrief,
    description:
      "Turn cross-functional launch traffic into a concise executive brief with risks, owners, and next steps.",
    source: "workspace",
    enabled: true,
    dir: storyWorkspaceSkillDir(storySkillNames.executiveBrief),
    version: "1.2.0",
    metadata: {
      tags: ["executive", "launch", "briefing"],
      downloads: 318,
    },
    provenance: {
      installed_at: "2026-04-17T16:40:00Z",
      registry: "workspace",
      slug: "workspace",
      version: "1.2.0",
    },
  },
  {
    name: storySkillNames.launchCopy,
    description:
      "Polish launch headlines, CRM copy, pricing claims, and ad lines without violating the approved guardrails.",
    source: "workspace",
    enabled: true,
    dir: storyWorkspaceSkillDir(storySkillNames.launchCopy, storyWorkspacePaths.growth),
    version: "1.0.3",
    metadata: {
      tags: ["marketing", "copy", "claims"],
      downloads: 284,
    },
  },
  {
    name: storySkillNames.frontendQa,
    description:
      "Run launch-surface QA for hero states, pricing banners, mobile breakpoints, and fallback banners.",
    source: "workspace",
    enabled: true,
    dir: storyWorkspaceSkillDir(storySkillNames.frontendQa, storyWorkspacePaths.product),
    version: "1.1.0",
    metadata: {
      tags: ["frontend", "qa", "launch"],
      downloads: 227,
    },
  },
  {
    name: storySkillNames.financePrep,
    description:
      "Prepare launch GMV, burn, and reserve snapshots for finance reviews and launch-room decisions.",
    source: "workspace",
    enabled: true,
    dir: storyWorkspaceSkillDir(storySkillNames.financePrep, storyWorkspacePaths.finance),
    version: "0.9.4",
    metadata: {
      tags: ["finance", "gmv", "reporting"],
      downloads: 141,
    },
  },
  {
    name: storySkillNames.merchantEscalation,
    description:
      "Guide support and risk through launch-day merchant escalations with clear customer-safe next steps.",
    source: "marketplace",
    enabled: false,
    dir: "/opt/agh/skills/merchant-escalation-handoff",
    version: "0.8.2",
    metadata: {
      tags: ["support", "risk", "merchant"],
      downloads: 173,
    },
    provenance: {
      installed_at: "2026-04-17T15:00:00Z",
      registry: "community",
      slug: "community",
      version: "0.8.2",
    },
  },
];

export const primarySkillFixture: SkillPayload = skillFixtures[0];

export const skillContentFixtures: Record<string, string> = {
  [storySkillNames.executiveBrief]:
    "# Executive Brief Synth\n\nSummarize blockers, owners, fallbacks, and launch readiness in four bullet points.\n",
  [storySkillNames.launchCopy]:
    "# Launch Copy Polish\n\nRewrite launch copy so pricing language stays approved and conversion-friendly.\n",
  [storySkillNames.frontendQa]:
    "# Frontend Launch QA\n\nVerify hero states, pricing banners, fallback banners, and mobile spacing before launch.\n",
  [storySkillNames.financePrep]:
    "# Burn Report Prep\n\nCompile launch GMV, burn, reserve exposure, and refund-risk notes for finance.\n",
  [storySkillNames.merchantEscalation]:
    "# Merchant Escalation Handoff\n\nPrepare a merchant-safe escalation summary with owner, ETA, and next update.\n",
};

export const skillActionFixture: SkillActionResponse = {
  ok: true,
};
