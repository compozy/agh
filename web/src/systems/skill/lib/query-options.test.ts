import { describe, expect, it } from "vitest";

import { skillDetailOptions, skillsListOptions } from "./query-options";

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
