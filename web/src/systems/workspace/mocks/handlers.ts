import { http, HttpResponse, type HttpHandler } from "msw";

import { primaryWorkspaceFixture, workspaceDetailFixture, workspaceFixtures } from "./fixtures";

function resolveWorkspaceFromPath(path: string) {
  const trimmedPath = path.trim();
  const existingWorkspace = workspaceFixtures.find(workspace => workspace.root_dir === trimmedPath);

  if (existingWorkspace) {
    return existingWorkspace;
  }

  const name = trimmedPath.split("/").filter(Boolean).at(-1) ?? "workspace";

  return {
    ...primaryWorkspaceFixture,
    id: `ws_${name.replace(/[^a-zA-Z0-9]+/g, "_").toLowerCase()}`,
    name,
    root_dir: trimmedPath,
    updated_at: "2026-04-17T18:10:00Z",
  };
}

export const handlers: HttpHandler[] = [
  http.get("/api/workspaces", () => HttpResponse.json({ workspaces: workspaceFixtures })),
  http.get("/api/workspaces/:id", ({ params }) => {
    const id = String(params.id);
    const workspace = workspaceFixtures.find(candidate => candidate.id === id);

    if (!workspace) {
      return HttpResponse.json({ error: `Workspace not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json({
      ...workspaceDetailFixture,
      workspace,
    });
  }),
  http.post("/api/workspaces/resolve", async ({ request }) => {
    const body = (await request.json()) as { path?: string };
    const path = body.path?.trim();

    if (!path) {
      return HttpResponse.json({ error: "Workspace path is required." }, { status: 400 });
    }

    return HttpResponse.json({
      workspace: resolveWorkspaceFromPath(path),
    });
  }),
];
