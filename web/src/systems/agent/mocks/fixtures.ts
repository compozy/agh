import type { AgentPayload } from "../types";
import { storyAgentNames } from "@/storybook/fintech-scenario";

export const agentFixtures: AgentPayload[] = [
  {
    name: storyAgentNames.cto,
    provider: "claude",
    prompt:
      "Own launch command, arbitrate technical risk, and produce concise operator-ready briefings for the 18:30 UTC cutover.",
  },
  {
    name: storyAgentNames.cfo,
    provider: "claude",
    prompt:
      "Track burn, launch revenue pacing, reserve exposure, and operator-visible finance decisions across launch week.",
  },
  {
    name: storyAgentNames.product,
    provider: "gemini",
    prompt:
      "Maintain the launch checklist, align product decisions across teams, and keep the operator on the highest-leverage next step.",
  },
  {
    name: storyAgentNames.frontend,
    provider: "codex",
    model: "gpt-5.4",
    prompt:
      "Validate launch UI states, patch landing-page regressions, and summarize any customer-facing risk before ship.",
  },
  {
    name: storyAgentNames.marketing,
    provider: "gemini",
    prompt:
      "Coordinate launch timing, CRM sends, ad spend windows, and campaign sequencing across the go-to-market team.",
  },
  {
    name: storyAgentNames.copywriter,
    provider: "claude",
    prompt:
      "Polish launch headlines, claims, emails, and support macros so every operator-facing draft is publishable.",
  },
  {
    name: storyAgentNames.fraud,
    provider: "claude",
    prompt:
      "Investigate suspicious payout holds, reserve anomalies, and launch-day risk spikes before operators approve merchant actions.",
  },
  {
    name: storyAgentNames.support,
    provider: "claude",
    prompt:
      "Handle merchant escalations, cluster repeat issues, and prepare the next support reply with customer-safe language.",
  },
  {
    name: storyAgentNames.compliance,
    provider: "qwen-code",
    model: "qwen3.6-plus",
    prompt:
      "Check KYB evidence, sanctions flags, and claims compliance before a launch-room decision is finalized.",
  },
  {
    name: storyAgentNames.release,
    provider: "codex",
    model: "gpt-5.4",
    prompt:
      "Own release verification, canary promotion, rollback guardrails, and cross-system launch readiness updates.",
  },
  {
    name: storyAgentNames.platform,
    provider: "codex",
    model: "gpt-5.4",
    prompt:
      "Investigate webhook failures, partner API drift, and rollout bottlenecks across the checkout platform.",
  },
];

export const primaryAgentFixture: AgentPayload = agentFixtures[0];
