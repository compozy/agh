import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

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
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ workspaces: [mockWorkspace] }),
    } as Response);

    const result = await fetchWorkspaces();

    expect(result).toEqual([mockWorkspace]);
    expect(fetch).toHaveBeenCalledWith("/api/workspaces", { signal: undefined });
  });

  it("passes an abort signal to fetch", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ workspaces: [] }),
    } as Response);

    const controller = new AbortController();
    await fetchWorkspaces(controller.signal);

    expect(fetch).toHaveBeenCalledWith("/api/workspaces", { signal: controller.signal });
  });

  it("rejects incomplete API responses", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ workspaces: [{ id: "ws_alpha" }] }),
    } as Response);

    await expect(fetchWorkspaces()).rejects.toThrow();
  });
});

describe("resolveWorkspace", () => {
  it("posts a path to the resolve endpoint", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ workspace: mockWorkspace }),
    } as Response);

    const result = await resolveWorkspace({ path: "/workspace/alpha" });

    expect(result).toEqual(mockWorkspace);
    expect(fetch).toHaveBeenCalledWith("/api/workspaces/resolve", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path: "/workspace/alpha" }),
      signal: undefined,
    });
  });
});
