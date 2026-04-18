import { http, HttpResponse, type HttpHandler } from "msw";

import {
  memoryConsolidationFixture,
  memoryHeadersFixture,
  memoryMutationFixture,
  memoryReadFixtures,
} from "./fixtures";

function filterMemories(scope?: string | null) {
  if (scope === "workspace") {
    return memoryHeadersFixture.filter(memory => memory.filename.startsWith("workspace/"));
  }

  if (scope === "global") {
    return memoryHeadersFixture.filter(memory => memory.filename.startsWith("global/"));
  }

  return memoryHeadersFixture;
}

export const handlers: HttpHandler[] = [
  http.get("/api/memory", ({ request }) => {
    const scope = new URL(request.url).searchParams.get("scope");
    return HttpResponse.json(filterMemories(scope));
  }),
  http.get("/api/memory/:filename", ({ params }) => {
    const filename = decodeURIComponent(String(params.filename));
    const memory = memoryReadFixtures[filename];

    if (!memory) {
      return HttpResponse.json({ error: `Memory not found: ${filename}` }, { status: 404 });
    }

    return HttpResponse.json(memory);
  }),
  http.put("/api/memory/:filename", () => HttpResponse.json(memoryMutationFixture)),
  http.delete("/api/memory/:filename", () => HttpResponse.json(memoryMutationFixture)),
  http.post("/api/memory/consolidate", () => HttpResponse.json(memoryConsolidationFixture)),
];
