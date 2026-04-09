import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  consolidateMemory,
  deleteMemory,
  KnowledgeApiError,
  listMemories,
  readMemory,
  writeMemory,
} from "./knowledge-api";

const validHeader = {
  filename: "user_role.md",
  mod_time: "2026-04-01T12:00:00Z",
  name: "User Role",
  type: "user",
};

describe("listMemories", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("calls GET /api/memory?scope=:scope&workspace=:ws and returns typed array", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve([validHeader]),
    } as Response);

    const result = await listMemories("global", "/home/user/project");
    expect(result).toEqual([validHeader]);
    expect(fetch).toHaveBeenCalledWith(
      "/api/memory?scope=global&workspace=%2Fhome%2Fuser%2Fproject",
      { signal: undefined }
    );
  });

  it("calls GET /api/memory with no params when scope and workspace are omitted", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve([]),
    } as Response);

    const result = await listMemories();
    expect(result).toEqual([]);
    expect(fetch).toHaveBeenCalledWith("/api/memory", { signal: undefined });
  });

  it("passes abort signal to fetch", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve([]),
    } as Response);

    const controller = new AbortController();
    await listMemories("global", undefined, controller.signal);
    expect(fetch).toHaveBeenCalledWith("/api/memory?scope=global", {
      signal: controller.signal,
    });
  });

  it("throws KnowledgeApiError on non-2xx response", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 500,
    } as Response);

    await expect(listMemories()).rejects.toThrow(KnowledgeApiError);
    await expect(listMemories()).rejects.toThrow("Failed to fetch memories: 500");
  });
});

describe("readMemory", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("calls GET /api/memory/:filename?scope=:scope and returns content string", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ content: "# Memory content" }),
    } as Response);

    const result = await readMemory("global", "user_role.md");
    expect(result).toBe("# Memory content");
    expect(fetch).toHaveBeenCalledWith("/api/memory/user_role.md?scope=global", {
      signal: undefined,
    });
  });

  it("includes workspace in query params", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ content: "data" }),
    } as Response);

    await readMemory("workspace", "project_ctx.md", "/home/user/project");
    expect(fetch).toHaveBeenCalledWith(
      "/api/memory/project_ctx.md?scope=workspace&workspace=%2Fhome%2Fuser%2Fproject",
      { signal: undefined }
    );
  });

  it("throws KnowledgeApiError with 404 for unknown memory", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 404,
    } as Response);

    await expect(readMemory("global", "missing.md")).rejects.toThrow(
      "Memory not found: missing.md"
    );
    try {
      await readMemory("global", "missing.md");
    } catch (err) {
      expect(err).toBeInstanceOf(KnowledgeApiError);
      expect((err as KnowledgeApiError).status).toBe(404);
    }
  });

  it("throws KnowledgeApiError for other failures", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 503,
    } as Response);

    await expect(readMemory("global", "test.md")).rejects.toThrow(
      'Failed to read memory "test.md": 503'
    );
  });

  it("encodes filename in URL", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ content: "" }),
    } as Response);

    await readMemory("global", "my file.md");
    expect(fetch).toHaveBeenCalledWith("/api/memory/my%20file.md?scope=global", {
      signal: undefined,
    });
  });
});

describe("writeMemory", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("calls PUT /api/memory/:filename with body", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ ok: true }),
    } as Response);

    const result = await writeMemory("test.md", "content here", "global", "/ws");
    expect(result).toEqual({ ok: true });
    expect(fetch).toHaveBeenCalledWith("/api/memory/test.md", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ content: "content here", scope: "global", workspace: "/ws" }),
    });
  });

  it("throws KnowledgeApiError on non-2xx response", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 400,
    } as Response);

    await expect(writeMemory("test.md", "bad")).rejects.toThrow(KnowledgeApiError);
    await expect(writeMemory("test.md", "bad")).rejects.toThrow(
      'Failed to write memory "test.md": 400'
    );
  });
});

describe("deleteMemory", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("calls DELETE /api/memory/:filename?scope=:scope", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ ok: true }),
    } as Response);

    const result = await deleteMemory("global", "old.md");
    expect(result).toEqual({ ok: true });
    expect(fetch).toHaveBeenCalledWith("/api/memory/old.md?scope=global", { method: "DELETE" });
  });

  it("includes workspace in query params", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ ok: true }),
    } as Response);

    await deleteMemory("workspace", "project.md", "/home/user/proj");
    expect(fetch).toHaveBeenCalledWith(
      "/api/memory/project.md?scope=workspace&workspace=%2Fhome%2Fuser%2Fproj",
      { method: "DELETE" }
    );
  });

  it("throws KnowledgeApiError with 404 for unknown memory", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 404,
    } as Response);

    await expect(deleteMemory("global", "missing.md")).rejects.toThrow(
      "Memory not found: missing.md"
    );
  });

  it("throws KnowledgeApiError on non-2xx response", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 500,
    } as Response);

    await expect(deleteMemory("global", "test.md")).rejects.toThrow(KnowledgeApiError);
  });
});

describe("consolidateMemory", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("calls POST /api/memory/consolidate with workspace", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ triggered: true }),
    } as Response);

    const result = await consolidateMemory("/home/user/project");
    expect(result).toEqual({ triggered: true });
    expect(fetch).toHaveBeenCalledWith("/api/memory/consolidate", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ workspace: "/home/user/project" }),
    });
  });

  it("throws KnowledgeApiError on non-2xx response", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 500,
    } as Response);

    await expect(consolidateMemory()).rejects.toThrow(KnowledgeApiError);
    await expect(consolidateMemory()).rejects.toThrow("Failed to consolidate memory: 500");
  });
});

describe("KnowledgeApiError", () => {
  it("has correct name and status properties", () => {
    const err = new KnowledgeApiError("test error", 404);
    expect(err.name).toBe("KnowledgeApiError");
    expect(err.status).toBe(404);
    expect(err.message).toBe("test error");
    expect(err).toBeInstanceOf(Error);
  });
});
