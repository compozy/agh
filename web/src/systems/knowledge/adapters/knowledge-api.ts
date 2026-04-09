import {
  memoryHeaderSchema,
  memoryReadResponseSchema,
  memoryMutationResponseSchema,
  memoryConsolidateResponseSchema,
  type MemoryHeader,
  type MemoryMutationResponse,
  type MemoryConsolidateResponse,
} from "../types";

export class KnowledgeApiError extends Error {
  constructor(
    message: string,
    public readonly status: number
  ) {
    super(message);
    this.name = "KnowledgeApiError";
  }
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
  return memoryHeaderSchema.array().parse(json);
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
  const parsed = memoryReadResponseSchema.parse(json);
  return parsed.content;
}

export async function writeMemory(
  filename: string,
  content: string,
  scope?: string,
  workspace?: string
): Promise<MemoryMutationResponse> {
  const res = await fetch(`/api/memory/${encodeURIComponent(filename)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ content, scope, workspace }),
  });
  if (!res.ok) {
    throw new KnowledgeApiError(`Failed to write memory "${filename}": ${res.status}`, res.status);
  }
  const json = await res.json();
  return memoryMutationResponseSchema.parse(json);
}

export async function deleteMemory(
  scope: string,
  filename: string,
  workspace?: string
): Promise<MemoryMutationResponse> {
  const params = new URLSearchParams();
  params.set("scope", scope);
  if (workspace) params.set("workspace", workspace);
  const url = `/api/memory/${encodeURIComponent(filename)}?${params.toString()}`;

  const res = await fetch(url, { method: "DELETE" });
  if (!res.ok) {
    if (res.status === 404) {
      throw new KnowledgeApiError(`Memory not found: ${filename}`, 404);
    }
    throw new KnowledgeApiError(`Failed to delete memory "${filename}": ${res.status}`, res.status);
  }
  const json = await res.json();
  return memoryMutationResponseSchema.parse(json);
}

export async function consolidateMemory(workspace?: string): Promise<MemoryConsolidateResponse> {
  const res = await fetch("/api/memory/consolidate", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ workspace }),
  });
  if (!res.ok) {
    throw new KnowledgeApiError(`Failed to consolidate memory: ${res.status}`, res.status);
  }
  const json = await res.json();
  return memoryConsolidateResponseSchema.parse(json);
}
