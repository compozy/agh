import type { SessionPayload } from "../types";

const INTERNAL_MEMORY_EXTRACTOR_SPAWN_ROLE = "memory-extractor";

export function isInternalSession(session: SessionPayload): boolean {
  return (
    session.type === "dream" ||
    normalizedSpawnRole(session) === INTERNAL_MEMORY_EXTRACTOR_SPAWN_ROLE
  );
}

export function filterVisibleSessions(sessions: SessionPayload[]): SessionPayload[] {
  return sessions.filter(session => !isInternalSession(session));
}

function normalizedSpawnRole(session: SessionPayload): string {
  return session.lineage?.spawn_role?.trim().toLowerCase() ?? "";
}
