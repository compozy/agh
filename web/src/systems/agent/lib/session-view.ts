import type { SessionPayload } from "@/systems/session";

export interface AgentSessionSplit {
  normalSessions: SessionPayload[];
  memoryExtractionSessions: SessionPayload[];
}

export function isMemoryExtractionSession(session: SessionPayload): boolean {
  return session.type === "dream";
}

export function splitAgentSessions(sessions: SessionPayload[]): AgentSessionSplit {
  const normalSessions: SessionPayload[] = [];
  const memoryExtractionSessions: SessionPayload[] = [];

  for (const session of sessions) {
    if (isMemoryExtractionSession(session)) {
      memoryExtractionSessions.push(session);
      continue;
    }

    normalSessions.push(session);
  }

  return { normalSessions, memoryExtractionSessions };
}
