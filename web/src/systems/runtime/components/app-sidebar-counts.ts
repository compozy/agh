import { isSessionRunning, type SessionPayload } from "@/systems/session";
import type { AgentPayload } from "@/systems/agent";

export interface AgentsCount {
  live: number;
  total: number;
}

export function computeAgentsCount(
  agents: AgentPayload[] | undefined,
  sessions: SessionPayload[] | undefined
): AgentsCount {
  const total = agents?.length ?? 0;
  if (total === 0) return { live: 0, total: 0 };
  const liveNames = new Set<string>();
  for (const session of sessions ?? []) {
    if (isSessionRunning(session)) {
      liveNames.add(session.agent_name);
    }
  }
  let live = 0;
  for (const agent of agents ?? []) {
    if (liveNames.has(agent.name)) live += 1;
  }
  return { live, total };
}
