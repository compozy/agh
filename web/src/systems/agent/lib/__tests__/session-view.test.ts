import { describe, expect, it } from "vitest";

import type { SessionPayload } from "@/systems/session";
import { primarySessionFixture } from "@/systems/session/testing";
import { isMemoryExtractionSession, splitAgentSessions } from "../session-view";

function makeSession(overrides: Partial<SessionPayload>): SessionPayload {
  return {
    ...primarySessionFixture,
    ...overrides,
  };
}

describe("session view helpers", () => {
  it("classifies dream sessions as memory extraction sessions", () => {
    const memoryExtractionSession = makeSession({ id: "sess-memory", type: "dream" });
    const userSession = makeSession({ id: "sess-user", type: "user" });

    expect(isMemoryExtractionSession(memoryExtractionSession)).toBe(true);
    expect(isMemoryExtractionSession(userSession)).toBe(false);
  });

  it("keeps every non-dream session in the normal session group", () => {
    const userSession = makeSession({ id: "sess-user", type: "user" });
    const spawnedSession = makeSession({ id: "sess-spawned", type: "spawned" });
    const memoryExtractionSession = makeSession({ id: "sess-memory", type: "dream" });

    const split = splitAgentSessions([memoryExtractionSession, userSession, spawnedSession]);

    expect(split.normalSessions.map(session => session.id)).toEqual(["sess-user", "sess-spawned"]);
    expect(split.memoryExtractionSessions.map(session => session.id)).toEqual(["sess-memory"]);
  });
});
