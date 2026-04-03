import { agentsResponseSchema, agentResponseSchema, type AgentPayload } from "../types";

export async function fetchAgents(signal?: AbortSignal): Promise<AgentPayload[]> {
  const res = await fetch("/api/agents", { signal });
  if (!res.ok) {
    throw new Error(`Failed to fetch agents: ${res.status}`);
  }
  const json = await res.json();
  const parsed = agentsResponseSchema.parse(json);
  return parsed.agents;
}

export async function fetchAgent(name: string, signal?: AbortSignal): Promise<AgentPayload> {
  const res = await fetch(`/api/agents/${encodeURIComponent(name)}`, { signal });
  if (!res.ok) {
    if (res.status === 404) {
      throw new Error(`Agent not found: ${name}`);
    }
    throw new Error(`Failed to fetch agent "${name}": ${res.status}`);
  }
  const json = await res.json();
  const parsed = agentResponseSchema.parse(json);
  return parsed.agent;
}
