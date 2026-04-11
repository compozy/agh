import { describe, expect, it } from "vitest";

import { memoriesListOptions, memoryDetailOptions } from "@/systems/knowledge/lib/query-options";

describe("memoriesListOptions", () => {
  it("includes correct staleTime and refetchInterval", () => {
    const options = memoriesListOptions("global");
    expect(options.staleTime).toBe(30_000);
    expect(options.refetchInterval).toBe(60_000);
  });

  it("includes scope and workspace in query key", () => {
    const options = memoriesListOptions("global", "/ws");
    expect(options.queryKey).toEqual(["knowledge", "list", "global", "/ws"]);
  });

  it("uses empty strings for omitted params", () => {
    const options = memoriesListOptions();
    expect(options.queryKey).toEqual(["knowledge", "list", "", ""]);
  });
});

describe("memoryDetailOptions", () => {
  it("includes correct staleTime", () => {
    const options = memoryDetailOptions("global", "test.md");
    expect(options.staleTime).toBe(30_000);
  });

  it("includes scope, filename, and workspace in query key", () => {
    const options = memoryDetailOptions("global", "user_role.md", "/ws");
    expect(options.queryKey).toEqual(["knowledge", "detail", "global", "user_role.md", "/ws"]);
  });

  it("is disabled when scope is omitted", () => {
    const options = memoryDetailOptions(undefined, "test.md");
    expect(options.enabled).toBe(false);
  });

  it("is disabled when filename is empty", () => {
    const options = memoryDetailOptions("global", "");
    expect(options.enabled).toBe(false);
  });

  it("is enabled when both scope and filename are provided", () => {
    const options = memoryDetailOptions("global", "test.md");
    expect(options.enabled).toBe(true);
  });
});
