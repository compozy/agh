import { describe, expect, it } from "vitest";

import { knowledgeKeys } from "./query-keys";

describe("knowledgeKeys", () => {
  it("has hierarchically structured keys", () => {
    expect(knowledgeKeys.all).toEqual(["knowledge"]);
    expect(knowledgeKeys.list("global", "/ws")).toEqual(["knowledge", "list", "global", "/ws"]);
    expect(knowledgeKeys.detail("global", "user_role.md", "/ws")).toEqual([
      "knowledge",
      "detail",
      "global",
      "user_role.md",
      "/ws",
    ]);
  });

  it("list keys extend all keys", () => {
    const list = knowledgeKeys.list("global");
    expect(list[0]).toBe(knowledgeKeys.all[0]);
  });

  it("detail keys extend all keys", () => {
    const detail = knowledgeKeys.detail("global", "test.md");
    expect(detail[0]).toBe(knowledgeKeys.all[0]);
  });

  it("uses empty string for omitted optional params", () => {
    expect(knowledgeKeys.list()).toEqual(["knowledge", "list", "", ""]);
    expect(knowledgeKeys.detail("global", "test.md")).toEqual([
      "knowledge",
      "detail",
      "global",
      "test.md",
      "",
    ]);
  });
});
