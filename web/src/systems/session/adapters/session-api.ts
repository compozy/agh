import {
  sessionResponseSchema,
  sessionEventsResponseSchema,
  sessionTranscriptResponseSchema,
  sessionHistoryResponseSchema,
  sessionsResponseSchema,
  type SessionPayload,
  type SessionEventPayload,
  type TranscriptMessage,
  type TurnHistoryPayload,
} from "../types";

// --- List Sessions ---

export async function fetchSessions(signal?: AbortSignal): Promise<SessionPayload[]> {
  const res = await fetch("/api/sessions", { signal });
  if (!res.ok) {
    throw new Error(`Failed to fetch sessions: ${res.status}`);
  }
  const json = await res.json();
  const parsed = sessionsResponseSchema.parse(json);
  return parsed.sessions;
}

// --- Create Session ---

export interface CreateSessionParams {
  agent_name?: string;
  name?: string;
  workspace?: string;
}

export async function createSession(
  params: CreateSessionParams,
  signal?: AbortSignal
): Promise<SessionPayload> {
  const res = await fetch("/api/sessions", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(params),
    signal,
  });
  if (!res.ok) {
    if (res.status === 409) {
      throw new Error("Max sessions reached");
    }
    throw new Error(`Failed to create session: ${res.status}`);
  }
  const json = await res.json();
  const parsed = sessionResponseSchema.parse(json);
  return parsed.session;
}

// --- Get Session ---

export async function fetchSession(id: string, signal?: AbortSignal): Promise<SessionPayload> {
  const res = await fetch(`/api/sessions/${encodeURIComponent(id)}`, { signal });
  if (!res.ok) {
    if (res.status === 404) {
      throw new Error(`Session not found: ${id}`);
    }
    throw new Error(`Failed to fetch session "${id}": ${res.status}`);
  }
  const json = await res.json();
  const parsed = sessionResponseSchema.parse(json);
  return parsed.session;
}

// --- Stop Session ---

export async function stopSession(id: string, signal?: AbortSignal): Promise<void> {
  const res = await fetch(`/api/sessions/${encodeURIComponent(id)}`, {
    method: "DELETE",
    signal,
  });
  if (!res.ok) {
    if (res.status === 404) {
      throw new Error(`Session not found: ${id}`);
    }
    throw new Error(`Failed to stop session "${id}": ${res.status}`);
  }
}

// --- Resume Session ---

export async function resumeSession(id: string, signal?: AbortSignal): Promise<SessionPayload> {
  const res = await fetch(`/api/sessions/${encodeURIComponent(id)}/resume`, {
    method: "POST",
    signal,
  });
  if (!res.ok) {
    if (res.status === 404) {
      throw new Error(`Session not found: ${id}`);
    }
    throw new Error(`Failed to resume session "${id}": ${res.status}`);
  }
  const json = await res.json();
  const parsed = sessionResponseSchema.parse(json);
  return parsed.session;
}

// --- Session Events ---

export interface FetchSessionEventsParams {
  since?: string;
  limit?: number;
  after_sequence?: number;
  type?: string;
  agent_name?: string;
  turn_id?: string;
}

export async function fetchSessionEvents(
  id: string,
  params?: FetchSessionEventsParams,
  signal?: AbortSignal
): Promise<SessionEventPayload[]> {
  const url = new URL(`/api/sessions/${encodeURIComponent(id)}/events`, window.location.origin);
  if (params) {
    if (params.since) url.searchParams.set("since", params.since);
    if (params.limit != null) url.searchParams.set("limit", String(params.limit));
    if (params.after_sequence != null)
      url.searchParams.set("after_sequence", String(params.after_sequence));
    if (params.type) url.searchParams.set("type", params.type);
    if (params.agent_name) url.searchParams.set("agent_name", params.agent_name);
    if (params.turn_id) url.searchParams.set("turn_id", params.turn_id);
  }
  const res = await fetch(url.toString(), { signal });
  if (!res.ok) {
    if (res.status === 404) {
      throw new Error(`Session not found: ${id}`);
    }
    throw new Error(`Failed to fetch session events "${id}": ${res.status}`);
  }
  const json = await res.json();
  const parsed = sessionEventsResponseSchema.parse(json);
  return parsed.events;
}

// --- Approve Permission ---

export type PermissionDecision = "allow-once" | "allow-always" | "reject-once" | "reject-always";

export interface ApproveSessionParams {
  request_id: string;
  turn_id: string;
  decision: PermissionDecision;
}

export async function approveSession(
  id: string,
  params: ApproveSessionParams,
  signal?: AbortSignal
): Promise<void> {
  const res = await fetch(`/api/sessions/${encodeURIComponent(id)}/approve`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(params),
    signal,
  });
  if (!res.ok) {
    if (res.status === 404) {
      throw new Error(`Session not found: ${id}`);
    }
    throw new Error(`Failed to approve permission: ${res.status}`);
  }
}

// --- Session History ---

export async function fetchSessionHistory(
  id: string,
  signal?: AbortSignal
): Promise<TurnHistoryPayload[]> {
  const res = await fetch(`/api/sessions/${encodeURIComponent(id)}/history`, { signal });
  if (!res.ok) {
    if (res.status === 404) {
      throw new Error(`Session not found: ${id}`);
    }
    throw new Error(`Failed to fetch session history "${id}": ${res.status}`);
  }
  const json = await res.json();
  const parsed = sessionHistoryResponseSchema.parse(json);
  return parsed.history;
}

// --- Session Transcript ---

export async function fetchSessionTranscript(
  id: string,
  signal?: AbortSignal
): Promise<TranscriptMessage[]> {
  const res = await fetch(`/api/sessions/${encodeURIComponent(id)}/transcript`, { signal });
  if (!res.ok) {
    if (res.status === 404) {
      throw new Error(`Session not found: ${id}`);
    }
    throw new Error(`Failed to fetch session transcript "${id}": ${res.status}`);
  }
  const json = await res.json();
  const parsed = sessionTranscriptResponseSchema.parse(json);
  return parsed.messages;
}
