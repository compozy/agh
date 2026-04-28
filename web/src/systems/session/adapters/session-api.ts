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
    throw new Error(defaultApiErrorMessage("Failed to fetch sessions", response, error));
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
      throw new Error("Max sessions reached");
    }
    throw new Error(defaultApiErrorMessage("Failed to create session", response, error));
  }
  return requireResponseData(data, response, "Failed to create session").session;
}

export async function fetchSession(id: string, signal?: AbortSignal): Promise<SessionPayload> {
  const { data, error, response } = await apiClient.GET("/api/sessions/{id}", {
    params: { path: { id } },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new Error(`Session not found: ${id}`);
    }
    throw new Error(defaultApiErrorMessage(`Failed to fetch session "${id}"`, response, error));
  }
  return requireResponseData(data, response, `Failed to fetch session "${id}"`).session;
}

export async function deleteSession(id: string, signal?: AbortSignal): Promise<void> {
  const { error, response } = await apiClient.DELETE("/api/sessions/{id}", {
    params: { path: { id } },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new Error(`Session not found: ${id}`);
    }
    throw new Error(defaultApiErrorMessage(`Failed to delete session "${id}"`, response, error));
  }
}

export async function stopSession(id: string, signal?: AbortSignal): Promise<void> {
  const { error, response } = await apiClient.POST("/api/sessions/{id}/stop", {
    params: { path: { id } },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new Error(`Session not found: ${id}`);
    }
    throw new Error(defaultApiErrorMessage(`Failed to stop session "${id}"`, response, error));
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
      throw new Error(`Session not found: ${id}`);
    }
    throw new Error(`Failed to cancel prompt for session "${id}": ${response.status}`);
  }
}

export async function resumeSession(id: string, signal?: AbortSignal): Promise<SessionPayload> {
  const { data, error, response } = await apiClient.POST("/api/sessions/{id}/resume", {
    params: { path: { id } },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new Error(`Session not found: ${id}`);
    }
    throw new Error(defaultApiErrorMessage(`Failed to resume session "${id}"`, response, error));
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
    if (response.status === 404) {
      throw new Error(`Session not found: ${id}`);
    }
    throw new Error(defaultApiErrorMessage(`Failed to repair session "${id}"`, response, error));
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
      throw new Error(`Session not found: ${id}`);
    }
    if (response.status === 409) {
      throw new Error(`Cannot clear session "${id}" while a prompt is still running`);
    }
    throw new Error(`Failed to clear session "${id}": ${response.status}`);
  }

  const body: unknown = await response.json();
  if (!isSessionEnvelope(body)) {
    throw new Error(`Failed to clear session "${id}": invalid response payload`);
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
    if (response.status === 404) {
      throw new Error(`Session not found: ${id}`);
    }
    throw new Error(
      defaultApiErrorMessage(`Failed to fetch session events "${id}"`, response, error)
    );
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
    if (response.status === 404) {
      throw new Error(`Session not found: ${id}`);
    }
    throw new Error(defaultApiErrorMessage("Failed to approve permission", response, error));
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
    if (response.status === 404) {
      throw new Error(`Session not found: ${id}`);
    }
    throw new Error(
      defaultApiErrorMessage(`Failed to fetch session history "${id}"`, response, error)
    );
  }
  return requireResponseData(data, response, `Failed to fetch session history "${id}"`).history;
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
    if (response.status === 404) {
      throw new Error(`Session not found: ${id}`);
    }
    throw new Error(
      defaultApiErrorMessage(`Failed to fetch session transcript "${id}"`, response, error)
    );
  }

  const payload = requireResponseData(data, response, `Failed to fetch session transcript "${id}"`);

  return normalizeTranscriptMessages(payload.messages);
}
