import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useSkill, useSkills } from "./use-skills";

vi.mock("../adapters/skill-api", () => ({
  listSkills: vi.fn(),
  getSkill: vi.fn(),
  enableSkill: vi.fn(),
  disableSkill: vi.fn(),
}));

import { getSkill, listSkills } from "../adapters/skill-api";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

const validSkill = {
  name: "test-skill",
  description: "A test skill",
  source: "bundled",
  enabled: true,
  dir: "/path/to/skill",
};

describe("useSkills", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("returns loading state then data", async () => {
    vi.mocked(listSkills).mockResolvedValue([validSkill]);

    const { result } = renderHook(() => useSkills("ws_123"), {
      wrapper: createWrapper(),
    });

    expect(result.current.isLoading).toBe(true);

    await waitFor(() => {
      expect(result.current.data).toHaveLength(1);
    });

    expect(result.current.data).toEqual([validSkill]);
    expect(listSkills).toHaveBeenCalledWith("ws_123", expect.any(AbortSignal));
  });

  it("does not fetch when workspace is empty", () => {
    renderHook(() => useSkills(""), {
      wrapper: createWrapper(),
    });

    expect(listSkills).not.toHaveBeenCalled();
  });
});

describe("useSkill", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("loads a single skill detail", async () => {
    vi.mocked(getSkill).mockResolvedValue(validSkill);

    const { result } = renderHook(() => useSkill("test-skill", "ws_123"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data?.name).toBe("test-skill");
    });

    expect(getSkill).toHaveBeenCalledWith("test-skill", "ws_123", expect.any(AbortSignal));
  });

  it("does not fetch when name is empty", () => {
    renderHook(() => useSkill("", "ws_123"), {
      wrapper: createWrapper(),
    });

    expect(getSkill).not.toHaveBeenCalled();
  });
});
