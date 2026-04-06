import { workspaceResponseSchema, workspacesResponseSchema, type WorkspacePayload } from "../types";

export async function fetchWorkspaces(signal?: AbortSignal): Promise<WorkspacePayload[]> {
  const res = await fetch("/api/workspaces", { signal });
  if (!res.ok) {
    throw new Error(`Failed to fetch workspaces: ${res.status}`);
  }

  const json = await res.json();
  const parsed = workspacesResponseSchema.parse(json);
  return parsed.workspaces;
}

export interface ResolveWorkspaceParams {
  path: string;
}

export async function resolveWorkspace(
  params: ResolveWorkspaceParams,
  signal?: AbortSignal
): Promise<WorkspacePayload> {
  const res = await fetch("/api/workspaces/resolve", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(params),
    signal,
  });
  if (!res.ok) {
    throw new Error(`Failed to resolve workspace: ${res.status}`);
  }

  const json = await res.json();
  const parsed = workspaceResponseSchema.parse(json);
  return parsed.workspace;
}
