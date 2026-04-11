import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { expectFetchRequest, mockJsonResponse } from "@/test/fetch-test-utils";
import {
  consolidateMemory,
  deleteMemory,
  KnowledgeApiError,
  listMemories,
  readMemory,
  writeMemory,
} from "@/systems/knowledge/adapters/knowledge-api";

const validHeader = {
  filename: "user_role.md",
  mod_time: "2026-04-01T12:00:00Z",
  name: "User Role",
  type: "user",
};

beforeEach(() => {
  vi.stubGlobal("fetch", vi.fn());
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("listMemories", () => {
  it("calls GET /api/memory?scope=:scope&workspace=:ws and returns typed array", async () => {
    mockJsonResponse([validHeader]);

    const result = await listMemories("global", "/home/user/project");

    expect(result).toEqual([validHeader]);
    await expectFetchRequest({
      path: "/api/memory?scope=global&workspace=%2Fhome%2Fuser%2Fproject",
    });
  });

  it("calls GET /api/memory with no params when scope and workspace are omitted", async () => {
    mockJsonResponse([]);

    const result = await listMemories();

    expect(result).toEqual([]);
    await expectFetchRequest({ path: "/api/memory" });
  });

  it("passes abort signal to fetch", async () => {
    mockJsonResponse([]);

    const controller = new AbortController();
    await listMemories("global", undefined, controller.signal);

    await expectFetchRequest({
      path: "/api/memory?scope=global",
      signal: controller.signal,
    });
  });

  it("throws KnowledgeApiError on non-2xx response", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 500 }));

    await expect(listMemories()).rejects.toThrow(KnowledgeApiError);
    await expect(listMemories()).rejects.toThrow("Failed to fetch memories: 500");
  });
});

describe("readMemory", () => {
  it("calls GET /api/memory/:filename?scope=:scope and returns content string", async () => {
    mockJsonResponse({ content: "# Memory content" });

    const result = await readMemory("global", "user_role.md");

    expect(result).toBe("# Memory content");
    await expectFetchRequest({ path: "/api/memory/user_role.md?scope=global" });
  });

  it("includes workspace in query params", async () => {
    mockJsonResponse({ content: "data" });

    await readMemory("workspace", "project_ctx.md", "/home/user/project");

    await expectFetchRequest({
      path: "/api/memory/project_ctx.md?scope=workspace&workspace=%2Fhome%2Fuser%2Fproject",
    });
  });

  it("throws KnowledgeApiError with 404 for unknown memory", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 404 }));

    await expect(readMemory("global", "missing.md")).rejects.toThrow(
      "Memory not found: missing.md"
    );

    try {
      await readMemory("global", "missing.md");
    } catch (error) {
      expect(error).toBeInstanceOf(KnowledgeApiError);
      expect((error as KnowledgeApiError).status).toBe(404);
    }
  });

  it("throws KnowledgeApiError for other failures", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 503 }));

    await expect(readMemory("global", "test.md")).rejects.toThrow(
      'Failed to read memory "test.md": 503'
    );
  });

  it("encodes filename in URL", async () => {
    mockJsonResponse({ content: "" });

    await readMemory("global", "my file.md");

    await expectFetchRequest({ path: "/api/memory/my%20file.md?scope=global" });
  });
});

describe("writeMemory", () => {
  it("calls PUT /api/memory/:filename with body", async () => {
    mockJsonResponse({ ok: true });

    const result = await writeMemory("test.md", "content here", "global", "/ws");

    expect(result).toEqual({ ok: true });
    await expectFetchRequest({
      body: { content: "content here", scope: "global", workspace: "/ws" },
      method: "PUT",
      path: "/api/memory/test.md",
    });
  });

  it("passes abort signal to fetch", async () => {
    mockJsonResponse({ ok: true });

    const controller = new AbortController();
    await writeMemory("test.md", "content here", "global", "/ws", controller.signal);

    await expectFetchRequest({
      body: { content: "content here", scope: "global", workspace: "/ws" },
      method: "PUT",
      path: "/api/memory/test.md",
      signal: controller.signal,
    });
  });

  it("encodes filename in URL", async () => {
    mockJsonResponse({ ok: true });

    await writeMemory("my file @1.md", "content here", "global", "/ws");

    await expectFetchRequest({
      body: { content: "content here", scope: "global", workspace: "/ws" },
      method: "PUT",
      path: "/api/memory/my%20file%20%401.md",
    });
  });

  it("throws KnowledgeApiError on non-2xx response", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 400 }));

    await expect(writeMemory("test.md", "bad")).rejects.toThrow(KnowledgeApiError);
    await expect(writeMemory("test.md", "bad")).rejects.toThrow(
      'Failed to write memory "test.md": 400'
    );
  });
});

describe("deleteMemory", () => {
  it("calls DELETE /api/memory/:filename?scope=:scope", async () => {
    mockJsonResponse({ ok: true });

    const result = await deleteMemory("global", "old.md");

    expect(result).toEqual({ ok: true });
    await expectFetchRequest({
      method: "DELETE",
      path: "/api/memory/old.md?scope=global",
    });
  });

  it("includes workspace in query params", async () => {
    mockJsonResponse({ ok: true });

    await deleteMemory("workspace", "project.md", "/home/user/proj");

    await expectFetchRequest({
      method: "DELETE",
      path: "/api/memory/project.md?scope=workspace&workspace=%2Fhome%2Fuser%2Fproj",
    });
  });

  it("passes abort signal to fetch", async () => {
    mockJsonResponse({ ok: true });

    const controller = new AbortController();
    await deleteMemory("global", "old.md", undefined, controller.signal);

    await expectFetchRequest({
      method: "DELETE",
      path: "/api/memory/old.md?scope=global",
      signal: controller.signal,
    });
  });

  it("throws KnowledgeApiError with 404 for unknown memory", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 404 }));

    await expect(deleteMemory("global", "missing.md")).rejects.toThrow(
      "Memory not found: missing.md"
    );
  });

  it("throws KnowledgeApiError on non-2xx response", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 500 }));

    await expect(deleteMemory("global", "test.md")).rejects.toThrow(KnowledgeApiError);
  });
});

describe("consolidateMemory", () => {
  it("calls POST /api/memory/consolidate with workspace", async () => {
    mockJsonResponse({ triggered: true });

    const result = await consolidateMemory("/home/user/project");

    expect(result).toEqual({ triggered: true });
    await expectFetchRequest({
      body: { workspace: "/home/user/project" },
      method: "POST",
      path: "/api/memory/consolidate",
    });
  });

  it("passes abort signal to fetch", async () => {
    mockJsonResponse({ triggered: true });

    const controller = new AbortController();
    await consolidateMemory("/home/user/project", controller.signal);

    await expectFetchRequest({
      body: { workspace: "/home/user/project" },
      method: "POST",
      path: "/api/memory/consolidate",
      signal: controller.signal,
    });
  });

  it("throws KnowledgeApiError on non-2xx response", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 500 }));

    await expect(consolidateMemory()).rejects.toThrow(KnowledgeApiError);
    await expect(consolidateMemory()).rejects.toThrow("Failed to consolidate memory: 500");
  });
});

describe("KnowledgeApiError", () => {
  it("has correct name and status properties", () => {
    const error = new KnowledgeApiError("test error", 404);

    expect(error.name).toBe("KnowledgeApiError");
    expect(error.status).toBe(404);
    expect(error.message).toBe("test error");
    expect(error).toBeInstanceOf(Error);
  });
});
