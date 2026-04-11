import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { expectFetchRequest, mockJsonResponse } from "@/test/fetch-test-utils";

import { fetchWorkspaces, resolveWorkspace } from "./workspace-api";

const mockWorkspace = {
  id: "ws_alpha",
  root_dir: "/workspace/alpha",
  add_dirs: ["/workspace/shared"],
  name: "alpha",
  created_at: "2026-04-06T10:00:00Z",
  updated_at: "2026-04-06T10:00:00Z",
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
