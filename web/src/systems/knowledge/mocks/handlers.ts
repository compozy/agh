import { HttpResponse, type HttpHandler } from "msw";
import { aghApiMock } from "@/storybook/openapi-msw";

import {
  memoryDecisionsFixture,
  memoryDeleteFixture,
  memoryDreamTriggerFixture,
  memoryEditFixture,
  memoryHeadersFixture,
  memoryReadFixtures,
  memorySearchFixture,
  memoryWriteFixture,
} from "./fixtures";
import type { MemoryHeader } from "../types";

interface MemorySelector {
  scope?: string | null;
  workspaceId?: string | null;
  agentName?: string | null;
  agentTier?: string | null;
}

function readSelector(url: URL): MemorySelector {
  return {
    scope: url.searchParams.get("scope"),
    workspaceId: url.searchParams.get("workspace_id"),
    agentName: url.searchParams.get("agent_name"),
    agentTier: url.searchParams.get("agent_tier"),
  };
}

function matchesSelector(memory: MemoryHeader, selector: MemorySelector): boolean {
  if (selector.scope && memory.scope !== selector.scope) return false;
  if (selector.workspaceId && memory.workspace_id !== selector.workspaceId) return false;
  if (selector.agentName && memory.agent_name !== selector.agentName) return false;
  if (selector.agentTier && memory.agent_tier !== selector.agentTier) return false;
  return true;
}

function filterMemories(selector: MemorySelector): MemoryHeader[] {
  if (!selector.scope) {
    return memoryHeadersFixture;
  }
  return memoryHeadersFixture.filter(memory => matchesSelector(memory, selector));
}

export const handlers: HttpHandler[] = [
  aghApiMock.get("/api/memory", ({ request }) => {
    const selector = readSelector(new URL(request.url));
    return HttpResponse.json({ memories: filterMemories(selector) });
  }),
  aghApiMock.get("/api/memory/decisions", ({ request }) => {
    const selector = readSelector(new URL(request.url));
    if (!selector.scope) {
      return HttpResponse.json(memoryDecisionsFixture);
    }
    const decisions = memoryDecisionsFixture.decisions.filter(decision => {
      if (selector.scope && decision.scope !== selector.scope) return false;
      if (selector.workspaceId && decision.workspace_id !== selector.workspaceId) return false;
      if (selector.agentName && decision.agent_name !== selector.agentName) return false;
      if (selector.agentTier && decision.agent_tier !== selector.agentTier) return false;
      return true;
    });
    return HttpResponse.json({ decisions });
  }),
  aghApiMock.get("/api/memory/{filename}", ({ params }) => {
    const filename = decodeURIComponent(String(params.filename));
    const memory = memoryReadFixtures[filename];
    if (!memory) {
      return HttpResponse.json(
        { code: "memory.not_found", message: `Memory not found: ${filename}` },
        { status: 404 }
      );
    }
    return HttpResponse.json(memory);
  }),
  aghApiMock.post("/api/memory", () => HttpResponse.json(memoryWriteFixture)),
  aghApiMock.patch("/api/memory/{filename}", () => HttpResponse.json(memoryEditFixture)),
  aghApiMock.delete("/api/memory/{filename}", () => HttpResponse.json(memoryDeleteFixture)),
  aghApiMock.post("/api/memory/search", () => HttpResponse.json(memorySearchFixture)),
  aghApiMock.post("/api/memory/dreams/trigger", () => HttpResponse.json(memoryDreamTriggerFixture)),
];
