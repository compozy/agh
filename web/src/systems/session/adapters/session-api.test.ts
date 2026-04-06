import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  createSession,
  fetchSession,
  fetchSessionEvents,
  fetchSessionHistory,
  fetchSessions,
  resumeSession,
  stopSession,
} from "./session-api";

const mockSession = {
  id: "sess-001",
  name: "Test Session",
  agent_name: "claude-agent",
  workspace: "/tmp/test",
  state: "active",
  created_at: "2026-04-01T00:00:00Z",
  updated_at: "2026-04-01T01:00:00Z",
};

beforeEach(() => {
  vi.stubGlobal("fetch", vi.fn());
});

afterEach(() => {
  vi.restoreAllMocks();
});

// --- fetchSessions ---

describe("fetchSessions", () => {
  it("returns parsed SessionPayload array", async () => {
    const sessions = [mockSession, { ...mockSession, id: "sess-002", name: "Second" }];
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ sessions }),
    } as Response);

    const result = await fetchSessions();
    expect(result).toEqual(sessions);
    expect(result).toHaveLength(2);
  });

  it("passes abort signal to fetch", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ sessions: [] }),
    } as Response);

    const controller = new AbortController();
    await fetchSessions(controller.signal);
    expect(fetch).toHaveBeenCalledWith("/api/sessions", { signal: controller.signal });
  });

  it("throws on non-ok response", async () => {
    vi.mocked(fetch).mockResolvedValue({ ok: false, status: 500 } as Response);
    await expect(fetchSessions()).rejects.toThrow("Failed to fetch sessions: 500");
  });

  it("returns empty array when server returns empty list", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ sessions: [] }),
    } as Response);

    const result = await fetchSessions();
    expect(result).toEqual([]);
  });
});

// --- createSession ---

describe("createSession", () => {
  it("sends correct POST body with agent_name", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ session: mockSession }),
    } as Response);

    const result = await createSession({ agent_name: "claude-agent" });
    expect(result).toEqual(mockSession);
    expect(fetch).toHaveBeenCalledWith("/api/sessions", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ agent_name: "claude-agent" }),
      signal: undefined,
    });
  });

  it("allows create session without agent_name", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ session: mockSession }),
    } as Response);

    await createSession({});
    expect(fetch).toHaveBeenCalledWith("/api/sessions", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({}),
      signal: undefined,
    });
  });

  it("sends optional name and workspace", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ session: mockSession }),
    } as Response);

    await createSession({ agent_name: "claude-agent", name: "My Session", workspace: "/home" });
    expect(fetch).toHaveBeenCalledWith("/api/sessions", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        agent_name: "claude-agent",
        name: "My Session",
        workspace: "/home",
      }),
      signal: undefined,
    });
  });

  it("throws 'Max sessions reached' on 409", async () => {
    vi.mocked(fetch).mockResolvedValue({ ok: false, status: 409 } as Response);
    await expect(createSession({ agent_name: "claude-agent" })).rejects.toThrow(
      "Max sessions reached"
    );
  });

  it("throws generic error for other failures", async () => {
    vi.mocked(fetch).mockResolvedValue({ ok: false, status: 500 } as Response);
    await expect(createSession({ agent_name: "claude-agent" })).rejects.toThrow(
      "Failed to create session: 500"
    );
  });
});

// --- fetchSession ---

describe("fetchSession", () => {
  it("returns single SessionPayload on success", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ session: mockSession }),
    } as Response);

    const result = await fetchSession("sess-001");
    expect(result).toEqual(mockSession);
    expect(fetch).toHaveBeenCalledWith("/api/sessions/sess-001", { signal: undefined });
  });

  it("throws 404 error for unknown session", async () => {
    vi.mocked(fetch).mockResolvedValue({ ok: false, status: 404 } as Response);
    await expect(fetchSession("unknown")).rejects.toThrow("Session not found: unknown");
  });

  it("encodes session id in URL", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ session: mockSession }),
    } as Response);

    await fetchSession("id with spaces");
    expect(fetch).toHaveBeenCalledWith("/api/sessions/id%20with%20spaces", { signal: undefined });
  });
});

// --- stopSession ---

describe("stopSession", () => {
  it("calls DELETE endpoint", async () => {
    vi.mocked(fetch).mockResolvedValue({ ok: true } as Response);

    await stopSession("sess-001");
    expect(fetch).toHaveBeenCalledWith("/api/sessions/sess-001", {
      method: "DELETE",
      signal: undefined,
    });
  });

  it("throws 404 error for unknown session", async () => {
    vi.mocked(fetch).mockResolvedValue({ ok: false, status: 404 } as Response);
    await expect(stopSession("unknown")).rejects.toThrow("Session not found: unknown");
  });

  it("throws generic error for other failures", async () => {
    vi.mocked(fetch).mockResolvedValue({ ok: false, status: 500 } as Response);
    await expect(stopSession("sess-001")).rejects.toThrow('Failed to stop session "sess-001": 500');
  });
});

// --- resumeSession ---

describe("resumeSession", () => {
  it("calls POST resume endpoint", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ session: { ...mockSession, state: "active" } }),
    } as Response);

    const result = await resumeSession("sess-001");
    expect(result.state).toBe("active");
    expect(fetch).toHaveBeenCalledWith("/api/sessions/sess-001/resume", {
      method: "POST",
      signal: undefined,
    });
  });

  it("throws 404 error for unknown session", async () => {
    vi.mocked(fetch).mockResolvedValue({ ok: false, status: 404 } as Response);
    await expect(resumeSession("unknown")).rejects.toThrow("Session not found: unknown");
  });
});

// --- fetchSessionEvents ---

describe("fetchSessionEvents", () => {
  const mockEvents = [
    {
      id: "evt-1",
      session_id: "sess-001",
      sequence: 1,
      turn_id: "turn-1",
      type: "agent_message",
      agent_name: "claude-agent",
      content: { text: "Hello" },
      timestamp: "2026-04-01T00:00:00Z",
    },
  ];

  it("returns parsed events array", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ events: mockEvents }),
    } as Response);

    const result = await fetchSessionEvents("sess-001");
    expect(result).toEqual(mockEvents);
  });

  it("passes query params correctly", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ events: [] }),
    } as Response);

    await fetchSessionEvents("sess-001", {
      since: "2026-04-01T00:00:00Z",
      limit: 50,
      after_sequence: 10,
      type: "agent_message",
      agent_name: "claude-agent",
      turn_id: "turn-1",
    });

    const calledUrl = vi.mocked(fetch).mock.calls[0][0] as string;
    const url = new URL(calledUrl);
    expect(url.pathname).toBe("/api/sessions/sess-001/events");
    expect(url.searchParams.get("since")).toBe("2026-04-01T00:00:00Z");
    expect(url.searchParams.get("limit")).toBe("50");
    expect(url.searchParams.get("after_sequence")).toBe("10");
    expect(url.searchParams.get("type")).toBe("agent_message");
    expect(url.searchParams.get("agent_name")).toBe("claude-agent");
    expect(url.searchParams.get("turn_id")).toBe("turn-1");
  });

  it("omits undefined params", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ events: [] }),
    } as Response);

    await fetchSessionEvents("sess-001", { limit: 10 });

    const calledUrl = vi.mocked(fetch).mock.calls[0][0] as string;
    const url = new URL(calledUrl);
    expect(url.searchParams.get("limit")).toBe("10");
    expect(url.searchParams.has("since")).toBe(false);
    expect(url.searchParams.has("type")).toBe(false);
  });

  it("throws 404 for unknown session", async () => {
    vi.mocked(fetch).mockResolvedValue({ ok: false, status: 404 } as Response);
    await expect(fetchSessionEvents("unknown")).rejects.toThrow("Session not found: unknown");
  });
});

// --- fetchSessionHistory ---

describe("fetchSessionHistory", () => {
  const mockHistory = [
    {
      turn_id: "turn-1",
      events: [
        {
          id: "evt-1",
          session_id: "sess-001",
          sequence: 1,
          turn_id: "turn-1",
          type: "agent_message",
          agent_name: "claude-agent",
          content: { text: "Hello" },
          timestamp: "2026-04-01T00:00:00Z",
        },
      ],
    },
  ];

  it("returns parsed history array", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ history: mockHistory }),
    } as Response);

    const result = await fetchSessionHistory("sess-001");
    expect(result).toEqual(mockHistory);
    expect(fetch).toHaveBeenCalledWith("/api/sessions/sess-001/history", { signal: undefined });
  });

  it("throws 404 for unknown session", async () => {
    vi.mocked(fetch).mockResolvedValue({ ok: false, status: 404 } as Response);
    await expect(fetchSessionHistory("unknown")).rejects.toThrow("Session not found: unknown");
  });
});
