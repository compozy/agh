import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useSessionStore } from "./use-session-store";
import {
  useClearSessionConversation,
  useCreateSession,
  useDeleteSession,
} from "./use-session-actions";
import { sessionKeys } from "../lib/query-keys";
import type { SessionPayload } from "../types";

vi.mock("../adapters/session-api", () => ({
  clearSessionConversation: vi.fn(),
  createSession: vi.fn(),
  deleteSession: vi.fn(),
  stopSession: vi.fn(),
  resumeSession: vi.fn(),
}));

import { clearSessionConversation, createSession, deleteSession } from "../adapters/session-api";

function createWrapper(queryClient: QueryClient) {
  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

const createdSession: SessionPayload = {
  id: "sess-created",
  name: "Created session",
  agent_name: "claude-agent",
  workspace_id: "ws_alpha",
  workspace_path: "/workspace/alpha",
  state: "active",
  created_at: "2026-04-20T10:00:00Z",
  updated_at: "2026-04-20T10:00:01Z",
};

const staleCreatedSession: SessionPayload = {
  ...createdSession,
  name: "Stale session",
  state: "starting",
};

const existingSession: SessionPayload = {
  ...createdSession,
  id: "sess-existing",
  name: "Existing session",
  updated_at: "2026-04-19T10:00:00Z",
};

const otherWorkspaceSession: SessionPayload = {
  ...createdSession,
  id: "sess-other",
  workspace_id: "ws_beta",
  workspace_path: "/workspace/beta",
  name: "Other workspace session",
};

describe("session actions", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useSessionStore.setState({ drafts: {} });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("useCreateSession seeds detail cache, updates matching lists without duplication, and invalidates in background", async () => {
    vi.mocked(createSession).mockResolvedValue(createdSession);

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    queryClient.setQueryData(sessionKeys.list(), [staleCreatedSession, existingSession]);
    queryClient.setQueryData(sessionKeys.list("ws_alpha"), [existingSession]);
    queryClient.setQueryData(sessionKeys.list("ws_beta"), [otherWorkspaceSession]);
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    const { result } = renderHook(() => useCreateSession(), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({
        agent_name: createdSession.agent_name,
        workspace: createdSession.workspace_id,
      });
    });

    expect(createSession).toHaveBeenCalledWith({
      agent_name: createdSession.agent_name,
      workspace: createdSession.workspace_id,
    });
    expect(queryClient.getQueryData(sessionKeys.detail(createdSession.id))).toEqual(createdSession);
    expect(queryClient.getQueryData(sessionKeys.list())).toEqual([createdSession, existingSession]);
    expect(queryClient.getQueryData(sessionKeys.list("ws_alpha"))).toEqual([
      createdSession,
      existingSession,
    ]);
    expect(queryClient.getQueryData(sessionKeys.list("ws_beta"))).toEqual([otherWorkspaceSession]);
    expect(invalidateSpy).toHaveBeenNthCalledWith(1, {
      queryKey: sessionKeys.detail(createdSession.id),
    });
    expect(invalidateSpy).toHaveBeenNthCalledWith(2, { queryKey: sessionKeys.lists() });
  });

  it("useClearSessionConversation clears transcript caches optimistically without touching drafts", async () => {
    vi.mocked(clearSessionConversation).mockResolvedValue(createdSession);

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    queryClient.setQueryData(sessionKeys.detail(createdSession.id), createdSession);
    queryClient.setQueryData(sessionKeys.transcript(createdSession.id), [
      { id: "history-1", role: "assistant", content: "existing" },
    ]);
    queryClient.setQueryData(sessionKeys.history(createdSession.id), [{ id: "turn-1" }]);
    useSessionStore.getState().setDraft(createdSession.id, { text: "keep me" });

    const { result } = renderHook(() => useClearSessionConversation(), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync(createdSession.id);
    });

    expect(clearSessionConversation).toHaveBeenCalledWith(createdSession.id);
    expect(queryClient.getQueryData(sessionKeys.detail(createdSession.id))).toEqual(createdSession);
    expect(queryClient.getQueryData(sessionKeys.transcript(createdSession.id))).toEqual([]);
    expect(queryClient.getQueryData(sessionKeys.history(createdSession.id))).toEqual([]);
    expect(useSessionStore.getState().drafts[createdSession.id]?.text).toBe("keep me");
  });

  it("useClearSessionConversation rolls back optimistic cache changes on failure", async () => {
    vi.mocked(clearSessionConversation).mockRejectedValue(new Error("clear failed"));

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    queryClient.setQueryData(sessionKeys.detail(createdSession.id), createdSession);

    const transcriptSnapshot = [{ id: "history-1", role: "assistant", content: "existing" }];
    const historySnapshot = [{ id: "turn-1" }];
    queryClient.setQueryData(sessionKeys.transcript(createdSession.id), transcriptSnapshot);
    queryClient.setQueryData(sessionKeys.history(createdSession.id), historySnapshot);
    useSessionStore.getState().setDraft(createdSession.id, { text: "keep me" });

    const { result } = renderHook(() => useClearSessionConversation(), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await expect(result.current.mutateAsync(createdSession.id)).rejects.toThrow("clear failed");
    });

    expect(queryClient.getQueryData(sessionKeys.transcript(createdSession.id))).toEqual(
      transcriptSnapshot
    );
    expect(queryClient.getQueryData(sessionKeys.history(createdSession.id))).toEqual(
      historySnapshot
    );
    expect(useSessionStore.getState().drafts[createdSession.id]?.text).toBe("keep me");
  });

  it("useDeleteSession removes cached session data and clears the draft", async () => {
    vi.mocked(deleteSession).mockResolvedValue(undefined);

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    queryClient.setQueryData(sessionKeys.detail(createdSession.id), createdSession);
    queryClient.setQueryData(sessionKeys.transcript(createdSession.id), [
      { id: "history-1", role: "assistant", content: "existing" },
    ]);
    queryClient.setQueryData(sessionKeys.history(createdSession.id), [{ id: "turn-1" }]);
    queryClient.setQueryData(sessionKeys.events(createdSession.id), [{ id: "event-1" }]);
    useSessionStore.getState().setDraft(createdSession.id, { text: "remove me" });

    const { result } = renderHook(() => useDeleteSession(), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync(createdSession.id);
    });

    expect(deleteSession).toHaveBeenCalledWith(createdSession.id);
    expect(queryClient.getQueryData(sessionKeys.detail(createdSession.id))).toBeUndefined();
    expect(queryClient.getQueryData(sessionKeys.transcript(createdSession.id))).toBeUndefined();
    expect(queryClient.getQueryData(sessionKeys.history(createdSession.id))).toBeUndefined();
    expect(queryClient.getQueryData(sessionKeys.events(createdSession.id))).toBeUndefined();
    expect(useSessionStore.getState().drafts[createdSession.id]).toBeUndefined();
  });

  it("useDeleteSession preserves cached session data and drafts on failure", async () => {
    vi.mocked(deleteSession).mockRejectedValue(new Error("delete failed"));

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    queryClient.setQueryData(sessionKeys.detail(createdSession.id), createdSession);

    const transcriptSnapshot = [{ id: "history-1", role: "assistant", content: "existing" }];
    const historySnapshot = [{ id: "turn-1" }];
    const eventsSnapshot = [{ id: "event-1" }];
    queryClient.setQueryData(sessionKeys.transcript(createdSession.id), transcriptSnapshot);
    queryClient.setQueryData(sessionKeys.history(createdSession.id), historySnapshot);
    queryClient.setQueryData(sessionKeys.events(createdSession.id), eventsSnapshot);
    useSessionStore.getState().setDraft(createdSession.id, { text: "keep me" });

    const { result } = renderHook(() => useDeleteSession(), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await expect(result.current.mutateAsync(createdSession.id)).rejects.toThrow("delete failed");
    });

    expect(queryClient.getQueryData(sessionKeys.detail(createdSession.id))).toEqual(createdSession);
    expect(queryClient.getQueryData(sessionKeys.transcript(createdSession.id))).toEqual(
      transcriptSnapshot
    );
    expect(queryClient.getQueryData(sessionKeys.history(createdSession.id))).toEqual(
      historySnapshot
    );
    expect(queryClient.getQueryData(sessionKeys.events(createdSession.id))).toEqual(eventsSnapshot);
    expect(useSessionStore.getState().drafts[createdSession.id]?.text).toBe("keep me");
  });
});
