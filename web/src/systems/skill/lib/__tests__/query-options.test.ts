import { describe, expect, it } from "vitest";

import {
  skillDetailOptions,
  skillMarketplaceInfoOptions,
  skillMarketplaceSearchOptions,
  skillsListOptions,
} from "../query-options";

describe("skillsListOptions", () => {
  it("includes correct staleTime and refetchInterval", () => {
    const options = skillsListOptions("ws_123");
    expect(options.staleTime).toBe(30_000);
    expect(options.refetchInterval).toBe(60_000);
  });

  it("includes workspace in query key", () => {
    const options = skillsListOptions("ws_123");
    expect(options.queryKey).toEqual(["skills", "list", "ws_123"]);
  });

  it("is disabled when workspace is empty", () => {
    const options = skillsListOptions("");
    expect(options.enabled).toBe(false);
  });

  it("is enabled when workspace is provided", () => {
    const options = skillsListOptions("ws_123");
    expect(options.enabled).toBe(true);
  });
});

describe("skillDetailOptions", () => {
  it("includes correct staleTime", () => {
    const options = skillDetailOptions("my-skill", "ws_123");
    expect(options.staleTime).toBe(30_000);
  });

  it("includes name and workspace in query key", () => {
    const options = skillDetailOptions("my-skill", "ws_123");
    expect(options.queryKey).toEqual(["skills", "detail", "my-skill", "ws_123"]);
  });

  it("is disabled when name is empty", () => {
    const options = skillDetailOptions("", "ws_123");
    expect(options.enabled).toBe(false);
  });

  it("is disabled when workspace is empty", () => {
    const options = skillDetailOptions("my-skill", "");
    expect(options.enabled).toBe(false);
  });

  it("is enabled when both name and workspace are provided", () => {
    const options = skillDetailOptions("my-skill", "ws_123");
    expect(options.enabled).toBe(true);
  });
});

describe("skillMarketplaceSearchOptions", () => {
  it("encodes trimmed query and optional limit in the key", () => {
    const options = skillMarketplaceSearchOptions("  alpha  ", 25);
    expect(options.queryKey).toEqual(["skills", "marketplace", "search", "alpha", 25]);
  });

  it("is disabled when the query is empty or whitespace", () => {
    expect(skillMarketplaceSearchOptions("").enabled).toBe(false);
    expect(skillMarketplaceSearchOptions("   ").enabled).toBe(false);
  });

  it("is enabled when the trimmed query is non-empty", () => {
    expect(skillMarketplaceSearchOptions("alpha").enabled).toBe(true);
  });
});

describe("skillMarketplaceInfoOptions", () => {
  it("includes the slug in the query key", () => {
    const options = skillMarketplaceInfoOptions("@compozy/alpha");
    expect(options.queryKey).toEqual(["skills", "marketplace", "info", "@compozy/alpha"]);
  });

  it("is disabled when the slug is empty", () => {
    expect(skillMarketplaceInfoOptions("").enabled).toBe(false);
  });

  it("respects the explicit enabled flag", () => {
    expect(skillMarketplaceInfoOptions("@compozy/alpha", false).enabled).toBe(false);
    expect(skillMarketplaceInfoOptions("@compozy/alpha", true).enabled).toBe(true);
  });
});
