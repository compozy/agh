import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  expectFetchRequest,
  fetchRequest,
  mockEmptyResponse,
  mockJsonResponse,
} from "@/test/fetch-test-utils";

import {
  SessionApiError,
  SessionLedgerUnavailableError,
  SessionNotFoundError,
  cancelSessionPrompt,
  clearSessionConversation,
  createSession,
  deleteSession,
  fetchSession,
  fetchSessionEvents,
  fetchSessionLedger,
  fetchSessionTranscript,
  fetchSessions,
  repairSession,
  resumeSession,
  stopSession,
} from "../session-api";

const mockSession = {
  id: "sess-001",
  name: "Test Session",
  agent_name: "claude-agent",
  workspace_id: "ws_alpha",
  workspace_path: "/tmp/test",
  state: "active",
  created_at: "2026-04-01T00:00:00Z",
  updated_at: "2026-04-01T01:00:00Z",
};

const mockRepair = {
  session_id: "sess-001",
  issues: [
    {
      code: "event_sequence_gap",
      severity: "warning",
      turn_id: "turn-1",
      detail: "gap before sequence 4",
    },
  ],
  actions: [
    {
      code: "append_terminal_error",
      turn_id: "turn-1",
      event_id: "ev-repair-1",
      persisted: false,
    },
  ],
  persisted: false,
};

beforeEach(() => {
  vi.stubGlobal("fetch", vi.fn());
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("fetchSessions", () => {
  it("returns parsed SessionPayload array", async () => {
    const sessions = [mockSession, { ...mockSession, id: "sess-002", name: "Second" }];
    mockJsonResponse({ sessions });

    const result = await fetchSessions();

    expect(result).toEqual(sessions);
    expect(result).toHaveLength(2);
    await expectFetchRequest({ path: "/api/sessions" });
  });

  it("passes abort signal to fetch", async () => {
    mockJsonResponse({ sessions: [] });

    const controller = new AbortController();
    await fetchSessions(undefined, controller.signal);

    await expectFetchRequest({
      path: "/api/sessions",
      signal: controller.signal,
    });
  });

  it("adds the workspace filter when provided", async () => {
    mockJsonResponse({ sessions: [] });

    await fetchSessions("ws_alpha");

    await expectFetchRequest({ path: "/api/sessions?workspace=ws_alpha" });
  });

  it("treats blank workspace filters as unfiltered requests", async () => {
    mockJsonResponse({ sessions: [] });

    await fetchSessions("   ");

    await expectFetchRequest({ path: "/api/sessions" });
  });

  it("throws on non-ok response", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 500 }));

    await expect(fetchSessions()).rejects.toThrow("Failed to fetch sessions: 500");
  });

  it("returns empty array when server returns empty list", async () => {
    mockJsonResponse({ sessions: [] });

    const result = await fetchSessions();

    expect(result).toEqual([]);
  });
});

describe("createSession", () => {
  it("sends correct POST body with agent_name", async () => {
    mockJsonResponse({ session: mockSession });

    const result = await createSession({ agent_name: "claude-agent" });

    expect(result).toEqual(mockSession);
    await expectFetchRequest({
      body: { agent_name: "claude-agent" },
      method: "POST",
      path: "/api/sessions",
    });
  });

  it("allows create session without agent_name", async () => {
    mockJsonResponse({ session: mockSession });

    await createSession({});

    await expectFetchRequest({
      body: {},
      method: "POST",
      path: "/api/sessions",
    });
  });

  it("sends optional name and workspace", async () => {
    mockJsonResponse({ session: mockSession });

    await createSession({
      agent_name: "claude-agent",
      name: "My Session",
      workspace: "/home",
    });

    await expectFetchRequest({
      body: {
        agent_name: "claude-agent",
        name: "My Session",
        workspace: "/home",
      },
      method: "POST",
      path: "/api/sessions",
    });
  });

  it("sends workspace_path when creating from an explicit path", async () => {
    mockJsonResponse({ session: mockSession });

    await createSession({
      agent_name: "claude-agent",
      workspace_path: "/workspace/demo",
    });

    await expectFetchRequest({
      body: {
        agent_name: "claude-agent",
        workspace_path: "/workspace/demo",
      },
      method: "POST",
      path: "/api/sessions",
    });
  });

  it("throws 'Max sessions reached' on 409", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 409 }));

    await expect(createSession({ agent_name: "claude-agent" })).rejects.toThrow(
      "Max sessions reached"
    );
  });

  it("throws generic error for other failures", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 500 }));

    await expect(createSession({ agent_name: "claude-agent" })).rejects.toThrow(
      "Failed to create session: 500"
    );
  });
});

describe("fetchSession", () => {
  it("returns single SessionPayload on success", async () => {
    mockJsonResponse({ session: mockSession });

    const result = await fetchSession("sess-001");

    expect(result).toEqual(mockSession);
    await expectFetchRequest({ path: "/api/sessions/sess-001" });
  });

  it("throws 404 error for unknown session", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 404 }));

    await expect(fetchSession("unknown")).rejects.toThrow("Session not found: unknown");
  });

  it("encodes session id in URL", async () => {
    mockJsonResponse({ session: mockSession });

    await fetchSession("id with spaces");

    await expectFetchRequest({ path: "/api/sessions/id%20with%20spaces" });
  });
});

describe("stopSession", () => {
  it("calls POST stop endpoint", async () => {
    mockEmptyResponse();

    await stopSession("sess-001");

    await expectFetchRequest({
      method: "POST",
      path: "/api/sessions/sess-001/stop",
    });
  });

  it("throws 404 error for unknown session", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 404 }));

    await expect(stopSession("unknown")).rejects.toThrow("Session not found: unknown");
  });

  it("throws generic error for other failures", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 500 }));

    await expect(stopSession("sess-001")).rejects.toThrow('Failed to stop session "sess-001": 500');
  });
});

describe("deleteSession", () => {
  it("calls DELETE endpoint", async () => {
    mockEmptyResponse();

    await deleteSession("sess-001");

    await expectFetchRequest({
      method: "DELETE",
      path: "/api/sessions/sess-001",
    });
  });

  it("throws 404 error for unknown session", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 404 }));

    await expect(deleteSession("unknown")).rejects.toThrow("Session not found: unknown");
  });

  it("throws generic error for other failures", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 500 }));

    await expect(deleteSession("sess-001")).rejects.toThrow(
      'Failed to delete session "sess-001": 500'
    );
  });
});

describe("cancelSessionPrompt", () => {
  it("calls POST prompt cancel endpoint", async () => {
    mockEmptyResponse();

    await cancelSessionPrompt("sess-001");

    await expectFetchRequest({
      method: "POST",
      path: "/api/sessions/sess-001/prompt/cancel",
    });
  });

  it("throws 404 error for unknown session", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 404 }));

    await expect(cancelSessionPrompt("unknown")).rejects.toThrow("Session not found: unknown");
  });

  it("passes abort signal to fetch", async () => {
    mockEmptyResponse();

    const controller = new AbortController();
    await cancelSessionPrompt("sess-001", controller.signal);

    await expectFetchRequest({
      method: "POST",
      path: "/api/sessions/sess-001/prompt/cancel",
      signal: controller.signal,
    });
  });
});

describe("resumeSession", () => {
  it("calls POST resume endpoint", async () => {
    mockJsonResponse({ session: { ...mockSession, state: "active" } });

    const result = await resumeSession("sess-001");

    expect(result.state).toBe("active");
    await expectFetchRequest({
      method: "POST",
      path: "/api/sessions/sess-001/resume",
    });
  });

  it("throws 404 error for unknown session", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 404 }));

    await expect(resumeSession("unknown")).rejects.toThrow("Session not found: unknown");
  });
});

describe("repairSession", () => {
  it("calls POST repair endpoint with query flags and returns the repair payload", async () => {
    mockJsonResponse({ repair: mockRepair });

    const result = await repairSession("sess-001", { dry_run: true, force: true });

    expect(result).toEqual(mockRepair);
    const request = fetchRequest();
    const url = new URL(request.url);
    expect(request.method).toBe("POST");
    expect(url.pathname).toBe("/api/sessions/sess-001/repair");
    expect(url.searchParams.get("dry_run")).toBe("true");
    expect(url.searchParams.get("force")).toBe("true");
  });

  it("throws 404 error for unknown session", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 404 }));

    await expect(repairSession("unknown")).rejects.toBeInstanceOf(SessionNotFoundError);
    await expect(repairSession("unknown")).rejects.toThrow("Session not found: unknown");
  });

  it("throws typed adapter error for non-404 failures", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 500 }));

    await expect(repairSession("sess-001")).rejects.toBeInstanceOf(SessionApiError);
    await expect(repairSession("sess-001")).rejects.toMatchObject({
      message: 'Failed to repair session "sess-001": 500',
      status: 500,
      sessionId: "sess-001",
    });
  });

  it("passes abort signal to fetch", async () => {
    mockJsonResponse({ repair: mockRepair });

    const controller = new AbortController();
    await repairSession("sess-001", {}, controller.signal);

    await expectFetchRequest({
      method: "POST",
      path: "/api/sessions/sess-001/repair",
      signal: controller.signal,
    });
  });
});

describe("clearSessionConversation", () => {
  it("calls POST clear endpoint and returns the refreshed session payload", async () => {
    mockJsonResponse({ session: mockSession });

    const result = await clearSessionConversation("sess-001");

    expect(result).toEqual(mockSession);
    await expectFetchRequest({
      method: "POST",
      path: "/api/sessions/sess-001/clear",
    });
  });

  it("throws 409 when a prompt is still running", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 409 }));

    await expect(clearSessionConversation("sess-001")).rejects.toThrow(
      'Cannot clear session "sess-001" while a prompt is still running'
    );
  });

  it("throws on invalid response payload", async () => {
    mockJsonResponse({ nope: true });

    await expect(clearSessionConversation("sess-001")).rejects.toThrow(
      'Failed to clear session "sess-001": invalid response payload'
    );
  });
});

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
    mockJsonResponse({ events: mockEvents });

    const result = await fetchSessionEvents("sess-001");

    expect(result).toEqual(mockEvents);
    await expectFetchRequest({ path: "/api/sessions/sess-001/events" });
  });

  it("passes query params correctly", async () => {
    mockJsonResponse({ events: [] });

    await fetchSessionEvents("sess-001", {
      since: "2026-04-01T00:00:00Z",
      limit: 50,
      after_sequence: 10,
      type: "agent_message",
      agent_name: "claude-agent",
      turn_id: "turn-1",
    });

    const request = fetchRequest();
    const url = new URL(request.url);
    expect(url.pathname).toBe("/api/sessions/sess-001/events");
    expect(url.searchParams.get("since")).toBe("2026-04-01T00:00:00Z");
    expect(url.searchParams.get("limit")).toBe("50");
    expect(url.searchParams.get("after_sequence")).toBe("10");
    expect(url.searchParams.get("type")).toBe("agent_message");
    expect(url.searchParams.get("agent_name")).toBe("claude-agent");
    expect(url.searchParams.get("turn_id")).toBe("turn-1");
  });

  it("omits undefined params", async () => {
    mockJsonResponse({ events: [] });

    await fetchSessionEvents("sess-001", { limit: 10 });

    const request = fetchRequest();
    const url = new URL(request.url);
    expect(url.searchParams.get("limit")).toBe("10");
    expect(url.searchParams.has("since")).toBe(false);
    expect(url.searchParams.has("type")).toBe(false);
  });

  it("throws 404 for unknown session", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 404 }));

    await expect(fetchSessionEvents("unknown")).rejects.toThrow("Session not found: unknown");
  });
});

describe("fetchSessionLedger", () => {
  const mockLedger = {
    meta: {
      version: 1,
      session_id: "sess-001",
      workspace_id: "ws_alpha",
      root_session_id: "sess-root",
      parent_session_id: "sess-parent",
      spawn_depth: 1,
      path: "/sessions/ws_alpha/sess-001/ledger.jsonl",
      checksum: "sha256:abc",
      created_at: "2026-04-20T10:00:00Z",
      stopped_at: "2026-04-20T11:00:00Z",
    },
    events: [
      { sequence: 1, event_type: "session.started", emitted_at: "2026-04-20T10:00:00Z" },
      { sequence: 2, event_type: "memory.recall", emitted_at: "2026-04-20T10:01:00Z" },
    ],
  };

  it("returns the materialized ledger response on success", async () => {
    mockJsonResponse(mockLedger);

    const result = await fetchSessionLedger("sess-001");

    expect(result).toEqual(mockLedger);
    await expectFetchRequest({ path: "/api/memory/sessions/sess-001/ledger" });
  });

  it("throws SessionLedgerUnavailableError when the ledger has not materialized (404)", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 404 }));

    await expect(fetchSessionLedger("sess-001")).rejects.toBeInstanceOf(
      SessionLedgerUnavailableError
    );
    await expect(fetchSessionLedger("sess-001")).rejects.toMatchObject({
      message: "Session ledger not materialized: sess-001",
      status: 404,
      sessionId: "sess-001",
    });
  });

  it("throws a typed adapter error for non-404 failures", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 500 }));

    await expect(fetchSessionLedger("sess-001")).rejects.toBeInstanceOf(SessionApiError);
    await expect(fetchSessionLedger("sess-001")).rejects.toMatchObject({
      message: 'Failed to fetch session ledger "sess-001": 500',
      status: 500,
      sessionId: "sess-001",
    });
  });

  it("passes the abort signal through to fetch", async () => {
    mockJsonResponse(mockLedger);

    const controller = new AbortController();
    await fetchSessionLedger("sess-001", controller.signal);

    await expectFetchRequest({
      path: "/api/memory/sessions/sess-001/ledger",
      signal: controller.signal,
    });
  });
});

describe("fetchSessionTranscript", () => {
  const mockTranscript = {
    messages: [
      {
        id: "evt-1",
        role: "assistant",
        parts: [{ type: "text", text: "Hello", state: "done" }],
      },
      {
        id: "tool-1",
        role: "assistant",
        parts: [
          {
            type: "tool-Read",
            toolCallId: "tool-1",
            state: "output-available",
            input: { file_path: "/tmp/file.ts" },
            output: {
              type: "tool_result",
              title: "Read",
              raw: { stdout: "done", file_path: "/tmp/file.ts" },
            },
          },
        ],
      },
    ],
  };

  it("returns parsed transcript messages", async () => {
    mockJsonResponse(mockTranscript);

    const result = await fetchSessionTranscript("sess-001");

    expect(result).toEqual(mockTranscript.messages);
    await expectFetchRequest({ path: "/api/sessions/sess-001/transcript" });
  });

  it("returns an empty transcript without treating it as an invalid AI SDK message list", async () => {
    mockJsonResponse({ messages: [] });

    const result = await fetchSessionTranscript("sess-001");

    expect(result).toEqual([]);
    await expectFetchRequest({ path: "/api/sessions/sess-001/transcript" });
  });

  it("throws 404 for unknown session", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 404 }));

    await expect(fetchSessionTranscript("unknown")).rejects.toThrow("Session not found: unknown");
  });
});
