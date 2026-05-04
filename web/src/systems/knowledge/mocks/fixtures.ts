import type {
  MemoryConsolidateResponse,
  MemoryHeader,
  MemoryMutationResponse,
  MemoryReadResponse,
} from "../types";
import { storyAgentNames } from "@/storybook/fintech-scenario";

export const memoryHeadersFixture: MemoryHeader[] = [
  {
    filename: "global/operator-style.md",
    mod_time: "2026-04-17T17:30:00Z",
    name: "Operator Style",
    type: "user",
    description: "Guidance for calm, evidence-first operator communication during launch week.",
  },
  {
    filename: "global/launch-week-brief.md",
    mod_time: "2026-04-17T16:50:00Z",
    name: "Launch Week Brief",
    type: "project",
    description:
      "Canonical launch narrative, KPI targets, cutover sequence, and cross-functional owners.",
  },
  {
    filename: "global/pricing-claims-guardrails.md",
    mod_time: "2026-04-17T15:15:00Z",
    name: "Pricing Claims Guardrails",
    type: "reference",
    description:
      "Approved phrasing for pricing, fees, and guarantee language across ads, site copy, and support.",
    agent_name: storyAgentNames.copywriter,
  },
  {
    filename: "workspace/executive-risk-memo.md",
    mod_time: "2026-04-17T17:05:00Z",
    name: "Executive Risk Memo",
    type: "reference",
    description:
      "CTO and CFO notes on the remaining launch blockers and acceptable fallback paths.",
    agent_name: storyAgentNames.cto,
  },
  {
    filename: "workspace/support-macro-pack.md",
    mod_time: "2026-04-17T16:25:00Z",
    name: "Support Macro Pack",
    type: "reference",
    description:
      "Launch-day support macros for merchant onboarding delays, failed payouts, and pricing questions.",
    agent_name: storyAgentNames.support,
  },
  {
    filename: "workspace/kpi-glossary.md",
    mod_time: "2026-04-17T14:45:00Z",
    name: "KPI Glossary",
    type: "reference",
    description:
      "Shared definitions for GMV, activation, reserve exposure, payback, and support SLA.",
    agent_name: storyAgentNames.cfo,
  },
];

export const memoryReadFixtures: Record<string, MemoryReadResponse> = {
  "global/operator-style.md": {
    content:
      "# Operator Style\n\nState the fact pattern first, then the decision, then the next concrete action.\n",
  },
  "global/launch-week-brief.md": {
    content:
      "# Launch Week Brief\n\nNorthstar Pay Checkout launches at 18:30 UTC across Brazil and Mexico with a target of 1,200 pilot merchants and $2.4M GMV in week one.\n",
  },
  "global/pricing-claims-guardrails.md": {
    content:
      "# Pricing Claims Guardrails\n\nAvoid saying zero fees. Approved claim: predictable blended processing with launch-week credits for pilot merchants.\n",
  },
  "workspace/executive-risk-memo.md": {
    content:
      "# Executive Risk Memo\n\nRemaining blockers: partner settlement timeout visibility, launch-hero fallback copy, and support queue load above the four-minute SLA threshold.\n",
  },
  "workspace/support-macro-pack.md": {
    content:
      "# Support Macro Pack\n\n1. Acknowledge the issue.\n2. Confirm whether funds are safe.\n3. Set the next ETA and owner.\n",
  },
  "workspace/kpi-glossary.md": {
    content:
      "# KPI Glossary\n\n- GMV: processed launch volume.\n- Activation: merchant completed the first successful checkout.\n- Reserve exposure: held funds awaiting fraud review.\n",
  },
};

export const memoryMutationFixture: MemoryMutationResponse = {
  ok: true,
};

export const memoryConsolidationFixture: MemoryConsolidateResponse = {
  triggered: true,
  reason: "launch-week-refresh",
};
