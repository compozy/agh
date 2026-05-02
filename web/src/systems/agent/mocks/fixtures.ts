import type { AgentPayload } from "../types";

export const agentFixtures: AgentPayload[] = [
  {
    name: "claude-agent",
    provider: "claude",
    prompt: "Review recent changes and explain the trade-offs.",
  },
  {
    name: "codex-agent",
    provider: "codex",
    model: "gpt-5.4",
    prompt: "Own implementation details and verification for coding tasks.",
  },
  {
    name: "qwen-agent",
    provider: "qwen-code",
    model: "qwen3.6-plus",
    prompt: "Use Qwen Code for model-managed implementation work.",
  },
  {
    name: "gemini-agent",
    provider: "gemini",
    prompt: "Summarize architectural decisions for the current workspace.",
  },
];

export const primaryAgentFixture: AgentPayload = agentFixtures[0];
