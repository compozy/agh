import { setupServer } from "msw/node";
import { afterAll, afterEach, beforeAll, describe, expect, it } from "vitest";

import { handlers, resetAgentMockState } from "../handlers";
import { FIXTURE_AGENT_DEFINITION_DIGEST, primaryAgentFixture } from "../fixtures";

const server = setupServer(...handlers);
const API = "http://localhost";

beforeAll(() => {
  server.listen({ onUnhandledRequest: "error" });
});

afterEach(() => {
  resetAgentMockState();
  server.resetHandlers();
});

afterAll(() => {
  server.close();
});

describe("agent MSW handlers", () => {
  it("Should round-trip update, delete, and duplicate", async () => {
    const name = primaryAgentFixture.name;

    const conflict = await fetch(`${API}/api/agents/${name}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        expected_digest: "stale",
        agent: { ...primaryAgentFixture, prompt: "Nope" },
      }),
    });
    expect(conflict.status).toBe(409);

    const updated = await fetch(`${API}/api/agents/${name}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        expected_digest: FIXTURE_AGENT_DEFINITION_DIGEST,
        agent: { ...primaryAgentFixture, prompt: "Updated via MSW" },
      }),
    });
    expect(updated.status).toBe(200);
    const updatedBody = (await updated.json()) as {
      agent: { prompt: string; definition_digest: string };
    };
    expect(updatedBody.agent.prompt).toBe("Updated via MSW");
    expect(updatedBody.agent.definition_digest).not.toBe(FIXTURE_AGENT_DEFINITION_DIGEST);

    const duplicated = await fetch(`${API}/api/agents/${name}/duplicate`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        name: `${name}-copy`,
        scope: "global",
        overrides: { prompt: "Copy prompt" },
      }),
    });
    expect(duplicated.status).toBe(201);
    const dupBody = (await duplicated.json()) as { agent: { name: string; prompt: string } };
    expect(dupBody.agent.name).toBe(`${name}-copy`);
    expect(dupBody.agent.prompt).toBe("Copy prompt");

    const deleted = await fetch(`${API}/api/agents/${name}-copy`, { method: "DELETE" });
    expect(deleted.status).toBe(200);
    const deletedBody = (await deleted.json()) as { name: string };
    expect(deletedBody.name).toBe(`${name}-copy`);
  });

  it("Should round-trip soul and heartbeat authored-file routes", async () => {
    const name = primaryAgentFixture.name;

    const soulGet = await fetch(`${API}/api/agents/${name}/soul`);
    expect(soulGet.status).toBe(200);

    const soulPut = await fetch(`${API}/api/agents/${name}/soul`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        body: "Be helpful.",
        expected_digest: FIXTURE_AGENT_DEFINITION_DIGEST,
      }),
    });
    expect(soulPut.status).toBe(200);
    const soulBody = (await soulPut.json()) as { soul: { digest: string; body: string } };
    expect(soulBody.soul.body).toBe("Be helpful.");

    const soulConflict = await fetch(`${API}/api/agents/${name}/soul`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ body: "stale", expected_digest: "nope" }),
    });
    expect(soulConflict.status).toBe(409);

    const hbPut = await fetch(`${API}/api/agents/${name}/heartbeat`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        body: "---\nenabled: true\n---\n",
        expected_digest: FIXTURE_AGENT_DEFINITION_DIGEST,
      }),
    });
    expect(hbPut.status).toBe(200);

    const status = await fetch(`${API}/api/agents/${name}/heartbeat/status`);
    expect(status.status).toBe(200);
    const statusBody = (await status.json()) as { active: boolean };
    expect(statusBody.active).toBe(true);

    const wake = await fetch(`${API}/api/agents/${name}/heartbeat/wake`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ session_id: "sess-1", source: "manual" }),
    });
    expect(wake.status).toBe(200);
  });
});
