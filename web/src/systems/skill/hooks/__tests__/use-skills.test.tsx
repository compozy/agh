import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  useSkill,
  useSkillContent,
  useSkillMarketplaceInfo,
  useSkillMarketplaceSearch,
  useSkills,
} from "../use-skills";

vi.mock("../../adapters/skill-api", () => ({
  listSkills: vi.fn(),
  getSkill: vi.fn(),
  getSkillContent: vi.fn(),
  enableSkill: vi.fn(),
  disableSkill: vi.fn(),
  searchSkillMarketplace: vi.fn(),
  getSkillMarketplaceInfo: vi.fn(),
}));

import {
  getSkill,
  getSkillContent,
  getSkillMarketplaceInfo,
  listSkills,
  searchSkillMarketplace,
} from "../../adapters/skill-api";

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

describe("useSkillContent", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("loads skill content when enabled", async () => {
    vi.mocked(getSkillContent).mockResolvedValue("full skill content");

    const { result } = renderHook(() => useSkillContent("test-skill", "ws_123", true), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data).toBe("full skill content");
    });

    expect(getSkillContent).toHaveBeenCalledWith("test-skill", "ws_123", expect.any(AbortSignal));
  });

  it("does not fetch when disabled", () => {
    renderHook(() => useSkillContent("test-skill", "ws_123", false), {
      wrapper: createWrapper(),
    });

    expect(getSkillContent).not.toHaveBeenCalled();
  });
});

describe("useSkillMarketplaceSearch", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("fetches when the query is non-empty", async () => {
    vi.mocked(searchSkillMarketplace).mockResolvedValue([
      {
        name: "alpha",
        slug: "@compozy/alpha",
        author: "compozy",
        description: "demo",
        downloads: 1,
        source: "clawhub",
        version: "0.1.0",
      },
    ]);

    const { result } = renderHook(() => useSkillMarketplaceSearch("alpha"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data).toHaveLength(1);
    });

    expect(searchSkillMarketplace).toHaveBeenCalledWith(
      { query: "alpha", limit: undefined },
      expect.any(AbortSignal)
    );
  });

  it("does not fetch when the query is blank", () => {
    renderHook(() => useSkillMarketplaceSearch(""), {
      wrapper: createWrapper(),
    });

    expect(searchSkillMarketplace).not.toHaveBeenCalled();
  });
});

describe("useSkillMarketplaceInfo", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("fetches the marketplace info for a slug when enabled", async () => {
    vi.mocked(getSkillMarketplaceInfo).mockResolvedValue({
      name: "alpha",
      slug: "@compozy/alpha",
      author: "compozy",
      description: "demo",
      downloads: 1,
      source: "clawhub",
      version: "0.1.0",
    });

    const { result } = renderHook(() => useSkillMarketplaceInfo("@compozy/alpha"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data?.name).toBe("alpha");
    });

    expect(getSkillMarketplaceInfo).toHaveBeenCalledWith("@compozy/alpha", expect.any(AbortSignal));
  });

  it("trims padded marketplace slugs before fetching", async () => {
    vi.mocked(getSkillMarketplaceInfo).mockResolvedValue({
      name: "alpha",
      slug: "@compozy/alpha",
      author: "compozy",
      description: "demo",
      downloads: 1,
      source: "clawhub",
      version: "0.1.0",
    });

    renderHook(() => useSkillMarketplaceInfo("  @compozy/alpha  "), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(getSkillMarketplaceInfo).toHaveBeenCalledWith(
        "@compozy/alpha",
        expect.any(AbortSignal)
      );
    });
  });

  it("does not fetch when explicitly disabled", () => {
    renderHook(() => useSkillMarketplaceInfo("@compozy/alpha", false), {
      wrapper: createWrapper(),
    });

    expect(getSkillMarketplaceInfo).not.toHaveBeenCalled();
  });

  it("does not fetch when the slug is whitespace only", () => {
    renderHook(() => useSkillMarketplaceInfo("   "), {
      wrapper: createWrapper(),
    });

    expect(getSkillMarketplaceInfo).not.toHaveBeenCalled();
  });
});
