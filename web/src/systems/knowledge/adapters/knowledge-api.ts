import {
  memoryHeaderSchema,
  memoryConsolidateResponseSchema,
  memoryMutationResponseSchema,
  memoryReadResponseSchema,
  type MemoryHeader,
  type MemoryConsolidateResponse,
  type MemoryMutationResponse,
} from "../types";
import type { ZodType } from "zod";

export class KnowledgeApiError extends Error {
  constructor(
    message: string,
    public readonly status: number
  ) {
    super(message);
    this.name = "KnowledgeApiError";
  }
}

function parseOrThrow<T>(schema: ZodType<T>, input: unknown, message: string, status: number): T {
  const parsed = schema.safeParse(input);
  if (!parsed.success) {
    throw new KnowledgeApiError(message, status);
  }
  return parsed.data;
}

export async function listMemories(
  scope?: string,
  workspace?: string,
  signal?: AbortSignal
): Promise<MemoryHeader[]> {
  const params = new URLSearchParams();
  if (scope) params.set("scope", scope);
  if (workspace) params.set("workspace", workspace);
  const qs = params.toString();
  const url = `/api/memory${qs ? `?${qs}` : ""}`;

  const res = await fetch(url, { signal });
  if (!res.ok) {
    throw new KnowledgeApiError(`Failed to fetch memories: ${res.status}`, res.status);
  }
  const json = await res.json();
  return parseOrThrow(
    memoryHeaderSchema.array(),
    json,
    "Invalid memories list response",
    res.status
  );
}

export async function readMemory(
  scope: string,
  filename: string,
  workspace?: string,
  signal?: AbortSignal
): Promise<string> {
  const params = new URLSearchParams();
  params.set("scope", scope);
  if (workspace) params.set("workspace", workspace);
  const url = `/api/memory/${encodeURIComponent(filename)}?${params.toString()}`;

  const res = await fetch(url, { signal });
  if (!res.ok) {
    if (res.status === 404) {
      throw new KnowledgeApiError(`Memory not found: ${filename}`, 404);
    }
    throw new KnowledgeApiError(`Failed to read memory "${filename}": ${res.status}`, res.status);
  }
  const json = await res.json();
  const parsed = parseOrThrow(
    memoryReadResponseSchema,
    json,
    `Invalid memory payload for "${filename}"`,
    res.status
  );
  return parsed.content;
}

export async function writeMemory(
  filename: string,
  content: string,
  scope?: string,
  workspace?: string,
  signal?: AbortSignal
): Promise<MemoryMutationResponse> {
  const res = await fetch(`/api/memory/${encodeURIComponent(filename)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ content, scope, workspace }),
    signal,
  });
  if (!res.ok) {
    throw new KnowledgeApiError(`Failed to write memory "${filename}": ${res.status}`, res.status);
  }
  const json = await res.json();
  return parseOrThrow(
    memoryMutationResponseSchema,
    json,
    `Invalid memory mutation response for "${filename}"`,
    res.status
  );
}

export async function deleteMemory(
  scope: string,
  filename: string,
  workspace?: string,
  signal?: AbortSignal
): Promise<MemoryMutationResponse> {
  const params = new URLSearchParams();
  params.set("scope", scope);
  if (workspace) params.set("workspace", workspace);
  const url = `/api/memory/${encodeURIComponent(filename)}?${params.toString()}`;

  const res = await fetch(url, { method: "DELETE", signal });
  if (!res.ok) {
    if (res.status === 404) {
      throw new KnowledgeApiError(`Memory not found: ${filename}`, 404);
    }
    throw new KnowledgeApiError(`Failed to delete memory "${filename}": ${res.status}`, res.status);
  }
  const json = await res.json();
  return parseOrThrow(
    memoryMutationResponseSchema,
    json,
    `Invalid memory deletion response for "${filename}"`,
    res.status
  );
}

export async function consolidateMemory(
  workspace?: string,
  signal?: AbortSignal
): Promise<MemoryConsolidateResponse> {
  const res = await fetch("/api/memory/consolidate", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ workspace }),
    signal,
  });
  if (!res.ok) {
    throw new KnowledgeApiError(`Failed to consolidate memory: ${res.status}`, res.status);
  }
  const json = await res.json();
  return parseOrThrow(
    memoryConsolidateResponseSchema,
    json,
    "Invalid memory consolidate response",
    res.status
  );
}
