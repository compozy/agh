import { setupServer } from "msw/node";
import { afterAll, afterEach, beforeAll, describe, expect, it } from "vitest";

import { handlers } from "../handlers";
import { sessionFixtures } from "../fixtures";

const server = setupServer(...handlers);
const API = "http://localhost";

beforeAll(() => {
  server.listen({ onUnhandledRequest: "error" });
});

afterEach(() => {
  server.resetHandlers();
});

afterAll(() => {
  server.close();
});

describe("session MSW handlers", () => {
  it("Should filter listSessions by workspace and agent", async () => {
    const sample = sessionFixtures[0]!;
    const workspace = sample.workspace_id!;
    const agent = sample.agent_name!;

    const filtered = await fetch(
      `${API}/api/sessions?workspace=${encodeURIComponent(workspace)}&agent=${encodeURIComponent(agent)}`
    );
    expect(filtered.status).toBe(200);
    const body = (await filtered.json()) as {
      sessions: Array<{ agent_name: string; workspace_id: string }>;
    };
    expect(body.sessions.length).toBeGreaterThan(0);
    expect(body.sessions.every(session => session.workspace_id === workspace)).toBe(true);
    expect(body.sessions.every(session => session.agent_name === agent)).toBe(true);

    const workspaceOnly = await fetch(
      `${API}/api/sessions?workspace=${encodeURIComponent(workspace)}`
    );
    const workspaceBody = (await workspaceOnly.json()) as {
      sessions: Array<{ workspace_id: string }>;
    };
    expect(workspaceBody.sessions.every(session => session.workspace_id === workspace)).toBe(true);
    expect(workspaceBody.sessions.length).toBeGreaterThanOrEqual(body.sessions.length);
  });
});
