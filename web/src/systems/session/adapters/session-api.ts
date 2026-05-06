import {
  apiClient,
  apiRequestFailed,
  defaultApiErrorMessage,
  requireResponseData,
} from "@/lib/api-client";

import type {
  ApproveSessionParams,
  CreateSessionParams,
  FetchSessionEventsParams,
  SessionLedgerResponse,
  SessionMessage,
  SessionEventPayload,
  SessionPayload,
  SessionRepairPayload,
  SessionRepairQuery,
  TurnHistoryPayload,
} from "../types";
import { normalizeTranscriptMessages } from "../lib/message-schemas";

export type {
  ApproveSessionParams,
  CreateSessionParams,
  FetchSessionEventsParams,
  PermissionDecision,
  SessionRepairQuery,
} from "../types";

export class SessionApiError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly sessionId?: string
  ) {
    super(message);
    this.name = "SessionApiError";
  }
}

export class SessionNotFoundError extends SessionApiError {
  constructor(id: string) {
    super(`Session not found: ${id}`, 404, id);
    this.name = "SessionNotFoundError";
  }
}

function throwSessionRequestError(
  response: Response,
  error: unknown,
  fallback: string,
  sessionId?: string
): never {
  if (response.status === 404 && sessionId) {
    throw new SessionNotFoundError(sessionId);
  }
  throw new SessionApiError(
    defaultApiErrorMessage(fallback, response, error),
    response.status,
    sessionId
  );
}

export async function fetchSessions(
  workspace?: string,
  signal?: AbortSignal
): Promise<SessionPayload[]> {
  const normalizedWorkspace = workspace?.trim();
  const { data, error, response } = await apiClient.GET("/api/sessions", {
    params:
      normalizedWorkspace == null || normalizedWorkspace === ""
        ? undefined
        : { query: { workspace: normalizedWorkspace } },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    throwSessionRequestError(response, error, "Failed to fetch sessions");
  }
  return requireResponseData(data, response, "Failed to fetch sessions").sessions;
}

export async function createSession(
  params: CreateSessionParams,
  signal?: AbortSignal
): Promise<SessionPayload> {
  const { data, error, response } = await apiClient.POST("/api/sessions", {
    body: params,
    signal,
  });
  if (apiRequestFailed(response, error)) {
    if (response.status === 409) {
      throw new SessionApiError("Max sessions reached", 409);
    }
    throwSessionRequestError(response, error, "Failed to create session");
  }
  return requireResponseData(data, response, "Failed to create session").session;
}

export async function fetchSession(id: string, signal?: AbortSignal): Promise<SessionPayload> {
  const { data, error, response } = await apiClient.GET("/api/sessions/{id}", {
    params: { path: { id } },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    throwSessionRequestError(response, error, `Failed to fetch session "${id}"`, id);
  }
  return requireResponseData(data, response, `Failed to fetch session "${id}"`).session;
}

export async function deleteSession(id: string, signal?: AbortSignal): Promise<void> {
  const { error, response } = await apiClient.DELETE("/api/sessions/{id}", {
    params: { path: { id } },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    throwSessionRequestError(response, error, `Failed to delete session "${id}"`, id);
  }
}

export async function stopSession(id: string, signal?: AbortSignal): Promise<void> {
  const { error, response } = await apiClient.POST("/api/sessions/{id}/stop", {
    params: { path: { id } },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    throwSessionRequestError(response, error, `Failed to stop session "${id}"`, id);
  }
}

export async function cancelSessionPrompt(id: string, signal?: AbortSignal): Promise<void> {
  const request = new Request(
    new URL(
      `/api/sessions/${encodeURIComponent(id)}/prompt/cancel`,
      typeof window === "undefined" ? "http://localhost" : window.location.origin
    ),
    {
      method: "POST",
      signal,
    }
  );
  const response = await globalThis.fetch(request);
  if (!response.ok) {
    if (response.status === 404) {
      throw new SessionNotFoundError(id);
    }
    throw new SessionApiError(
      `Failed to cancel prompt for session "${id}": ${response.status}`,
      response.status,
      id
    );
  }
}

export async function resumeSession(id: string, signal?: AbortSignal): Promise<SessionPayload> {
  const { data, error, response } = await apiClient.POST("/api/sessions/{id}/resume", {
    params: { path: { id } },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    throwSessionRequestError(response, error, `Failed to resume session "${id}"`, id);
  }
  return requireResponseData(data, response, `Failed to resume session "${id}"`).session;
}

export async function repairSession(
  id: string,
  query: SessionRepairQuery = {},
  signal?: AbortSignal
): Promise<SessionRepairPayload> {
  const { data, error, response } = await apiClient.POST("/api/sessions/{id}/repair", {
    params: {
      path: { id },
      query,
    },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    throwSessionRequestError(response, error, `Failed to repair session "${id}"`, id);
  }
  return requireResponseData(data, response, `Failed to repair session "${id}"`).repair;
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isSessionEnvelope(value: unknown): value is { session: SessionPayload } {
  if (!isPlainObject(value) || !("session" in value)) {
    return false;
  }

  const session = value.session;
  return isPlainObject(session) && typeof session.id === "string";
}

export async function clearSessionConversation(
  id: string,
  signal?: AbortSignal
): Promise<SessionPayload> {
  const request = new Request(
    new URL(
      `/api/sessions/${encodeURIComponent(id)}/clear`,
      typeof window === "undefined" ? "http://localhost" : window.location.origin
    ),
    {
      method: "POST",
      signal,
    }
  );

  const response = await globalThis.fetch(request);
  if (!response.ok) {
    if (response.status === 404) {
      throw new SessionNotFoundError(id);
    }
    if (response.status === 409) {
      throw new SessionApiError(
        `Cannot clear session "${id}" while a prompt is still running`,
        409,
        id
      );
    }
    throw new SessionApiError(
      `Failed to clear session "${id}": ${response.status}`,
      response.status,
      id
    );
  }

  const body: unknown = await response.json();
  if (!isSessionEnvelope(body)) {
    throw new SessionApiError(
      `Failed to clear session "${id}": invalid response payload`,
      response.status,
      id
    );
  }

  return body.session;
}

export async function fetchSessionEvents(
  id: string,
  params?: FetchSessionEventsParams,
  signal?: AbortSignal
): Promise<SessionEventPayload[]> {
  const { data, error, response } = await apiClient.GET("/api/sessions/{id}/events", {
    params: {
      path: { id },
      query: params,
    },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    throwSessionRequestError(response, error, `Failed to fetch session events "${id}"`, id);
  }
  return requireResponseData(data, response, `Failed to fetch session events "${id}"`).events;
}

export async function approveSession(
  id: string,
  params: ApproveSessionParams,
  signal?: AbortSignal
): Promise<void> {
  const { error, response } = await apiClient.POST("/api/sessions/{id}/approve", {
    params: { path: { id } },
    body: params,
    signal,
  });
  if (apiRequestFailed(response, error)) {
    throwSessionRequestError(response, error, "Failed to approve permission", id);
  }
}

export async function fetchSessionHistory(
  id: string,
  signal?: AbortSignal
): Promise<TurnHistoryPayload[]> {
  const { data, error, response } = await apiClient.GET("/api/sessions/{id}/history", {
    params: { path: { id } },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    throwSessionRequestError(response, error, `Failed to fetch session history "${id}"`, id);
  }
  return requireResponseData(data, response, `Failed to fetch session history "${id}"`).history;
}

export class SessionLedgerUnavailableError extends SessionApiError {
  constructor(id: string) {
    super(`Session ledger not materialized: ${id}`, 404, id);
    this.name = "SessionLedgerUnavailableError";
  }
}

export async function fetchSessionLedger(
  id: string,
  signal?: AbortSignal
): Promise<SessionLedgerResponse> {
  const { data, error, response } = await apiClient.GET(
    "/api/memory/sessions/{session_id}/ledger",
    {
      params: { path: { session_id: id } },
      signal,
    }
  );
  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new SessionLedgerUnavailableError(id);
    }
    throwSessionRequestError(response, error, `Failed to fetch session ledger "${id}"`, id);
  }
  return requireResponseData(data, response, `Failed to fetch session ledger "${id}"`);
}

export async function fetchSessionTranscript(
  id: string,
  signal?: AbortSignal
): Promise<SessionMessage[]> {
  const { data, error, response } = await apiClient.GET("/api/sessions/{id}/transcript", {
    params: { path: { id } },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    throwSessionRequestError(response, error, `Failed to fetch session transcript "${id}"`, id);
  }

  const payload = requireResponseData(data, response, `Failed to fetch session transcript "${id}"`);

  return normalizeTranscriptMessages(payload.messages);
}
