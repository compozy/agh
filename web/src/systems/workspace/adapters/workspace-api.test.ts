import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { expectFetchRequest, mockJsonResponse } from "@/test/fetch-test-utils";

import {
  fetchWorkspace,
  fetchWorkspaces,
  resolveWorkspace,
  WorkspaceApiError,
} from "./workspace-api";

const mockWorkspace = {
  id: "ws_alpha",
  root_dir: "/workspace/alpha",
  add_dirs: ["/workspace/shared"],
  name: "alpha",
  created_at: "2026-04-06T10:00:00Z",
  updated_at: "2026-04-06T10:00:00Z",
};
const mockWorkspaceDetail = {
  agents: [{ name: "coder", prompt: "code", provider: "openai" }],
  sessions: [],
  skills: [],
  workspace: mockWorkspace,
};

beforeEach(() => {
  vi.stubGlobal("fetch", vi.fn());
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("fetchWorkspaces", () => {
  it("parses the workspace list response", async () => {
    mockJsonResponse({ workspaces: [mockWorkspace] });

    const result = await fetchWorkspaces();

    expect(result).toEqual([mockWorkspace]);
    await expectFetchRequest({ path: "/api/workspaces" });
  });

  it("passes an abort signal to fetch", async () => {
    mockJsonResponse({ workspaces: [] });

    const controller = new AbortController();
    await fetchWorkspaces(controller.signal);

    await expectFetchRequest({
      path: "/api/workspaces",
      signal: controller.signal,
    });
  });
});

describe("fetchWorkspace", () => {
  it("loads the resolved workspace detail payload", async () => {
    mockJsonResponse(mockWorkspaceDetail);

    const result = await fetchWorkspace("ws_alpha");

    expect(result).toEqual(mockWorkspaceDetail);
    await expectFetchRequest({
      path: "/api/workspaces/ws_alpha",
    });
  });

  it("passes an abort signal to fetch", async () => {
    mockJsonResponse(mockWorkspaceDetail);

    const controller = new AbortController();
    await fetchWorkspace("ws_alpha", controller.signal);

    await expectFetchRequest({
      path: "/api/workspaces/ws_alpha",
      signal: controller.signal,
    });
  });

  it("throws WorkspaceApiError with status when the request fails", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 503 }));

    await expect(fetchWorkspace("ws_alpha")).rejects.toThrow(WorkspaceApiError);

    try {
      await fetchWorkspace("ws_alpha");
    } catch (error) {
      expect(error).toBeInstanceOf(WorkspaceApiError);
      expect((error as WorkspaceApiError).status).toBe(503);
      expect((error as WorkspaceApiError).message).toBe("Failed to fetch workspace: 503");
    }
  });
});

describe("resolveWorkspace", () => {
  it("posts a path to the resolve endpoint", async () => {
    mockJsonResponse({ workspace: mockWorkspace });

    const result = await resolveWorkspace({ path: "/workspace/alpha" });

    expect(result).toEqual(mockWorkspace);
    await expectFetchRequest({
      body: { path: "/workspace/alpha" },
      method: "POST",
      path: "/api/workspaces/resolve",
    });
  });
});

describe("WorkspaceApiError", () => {
  it("preserves the status code for consumers", () => {
    const error = new WorkspaceApiError("boom", 422);

    expect(error.name).toBe("WorkspaceApiError");
    expect(error.status).toBe(422);
    expect(error.message).toBe("boom");
  });
});
