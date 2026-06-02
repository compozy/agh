import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { SessionPayload } from "../../types";
import { useSession, useSessionById, useSessions } from "../use-sessions";

vi.mock("../../adapters/session-api", () => ({
  fetchSession: vi.fn(),
  fetchSessionLedger: vi.fn(),
  fetchSessionRecap: vi.fn(),
  fetchSessions: vi.fn(),
  fetchSessionEvents: vi.fn(),
  fetchSessionHistory: vi.fn(),
  fetchSessionTranscript: vi.fn(),
  SessionApiError: class SessionApiError extends Error {
    constructor(
      message: string,
      public readonly status: number,
      public readonly sessionId?: string
    ) {
      super(message);
      this.name = "SessionApiError";
    }
  },
  SessionLedgerUnavailableError: class SessionLedgerUnavailableError extends Error {},
  SessionNotFoundError: class SessionNotFoundError extends Error {
    constructor(public readonly sessionId: string) {
      super(`Session not found: ${sessionId}`);
      this.name = "SessionNotFoundError";
    }
  },
}));

import { fetchSession, fetchSessions } from "../../adapters/session-api";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

function makeSession(overrides: Partial<SessionPayload> = {}): SessionPayload {
  return {
    id: "sess-001",
    agent_name: "claude-agent",
    provider: "claude",
    workspace_id: "ws_alpha",
    workspace_path: "/workspace/alpha",
    state: "active",
    badge: "idle",
    attachable: true,
    created_at: "2026-04-06T10:00:00Z",
    updated_at: "2026-04-06T10:00:00Z",
    ...overrides,
  };
}

describe("useSessions", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("loads sessions for the selected workspace filter", async () => {
    vi.mocked(fetchSessions).mockResolvedValue([
      {
        id: "sess-001",
        agent_name: "claude-agent",
        provider: "claude",
        workspace_id: "ws_alpha",
        workspace_path: "/workspace/alpha",
        state: "active",
        badge: "idle",
        attachable: true,
        created_at: "2026-04-06T10:00:00Z",
        updated_at: "2026-04-06T10:00:00Z",
      },
    ]);

    const { result } = renderHook(() => useSessions("ws_alpha"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data).toHaveLength(1);
    });

    expect(result.current.data?.[0]?.provider).toBe("claude");
    expect(fetchSessions).toHaveBeenCalledWith("ws_alpha", expect.any(AbortSignal));
  });

  it("disables the query when the workspace filter is unavailable", async () => {
    renderHook(() => useSessions(null, { enabled: false }), {
      wrapper: createWrapper(),
    });

    expect(fetchSessions).not.toHaveBeenCalled();
  });
});

describe("useSession", () => {
  it("loads a single session detail", async () => {
    vi.mocked(fetchSession).mockResolvedValue({
      id: "sess-001",
      agent_name: "claude-agent",
      provider: "claude",
      workspace_id: "ws_alpha",
      workspace_path: "/workspace/alpha",
      state: "active",
      badge: "idle",
      attachable: true,
      created_at: "2026-04-06T10:00:00Z",
      updated_at: "2026-04-06T10:00:00Z",
    });

    const { result } = renderHook(() => useSession("sess-001", "ws_alpha"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data?.id).toBe("sess-001");
    });

    expect(result.current.data?.provider).toBe("claude");
    expect(fetchSession).toHaveBeenCalledWith("ws_alpha", "sess-001", expect.any(AbortSignal));
  });
});

describe("useSessionById", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("resolves the owning workspace before loading session detail", async () => {
    vi.mocked(fetchSessions).mockResolvedValue([
      makeSession({ id: "sess-001", workspace_id: "ws_alpha" }),
    ]);
    vi.mocked(fetchSession).mockResolvedValue(
      makeSession({ id: "sess-001", workspace_id: "ws_alpha", name: "Detailed session" })
    );

    const { result } = renderHook(() => useSessionById("sess-001"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data?.name).toBe("Detailed session");
    });

    expect(fetchSessions).toHaveBeenCalledWith(undefined, expect.any(AbortSignal));
    expect(fetchSession).toHaveBeenCalledWith("ws_alpha", "sess-001", expect.any(AbortSignal));
  });

  it("reports not found without probing the active workspace when the global list has no session", async () => {
    vi.mocked(fetchSessions).mockResolvedValue([]);

    const { result } = renderHook(() => useSessionById("missing-session"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.error?.message).toBe("Session not found: missing-session");
    });

    expect(fetchSession).not.toHaveBeenCalled();
  });
});
