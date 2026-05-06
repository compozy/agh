import { describe, expect, it } from "vitest";

import { knowledgeKeys } from "../query-keys";

describe("knowledgeKeys", () => {
  it("Should expose hierarchical keys for list, detail, search and decisions", () => {
    expect(knowledgeKeys.all).toEqual(["knowledge"]);
    expect(knowledgeKeys.lists()).toEqual(["knowledge", "list"]);
    expect(knowledgeKeys.list({ scope: "global" })).toEqual([
      "knowledge",
      "list",
      "global",
      "",
      "",
      "",
    ]);
    expect(
      knowledgeKeys.list({
        scope: "agent",
        workspaceId: "ws_launch",
        agentName: "cto",
        agentTier: "workspace",
      })
    ).toEqual(["knowledge", "list", "agent", "ws_launch", "cto", "workspace"]);

    expect(knowledgeKeys.details()).toEqual(["knowledge", "detail"]);
    expect(knowledgeKeys.detail("user_role.md", { scope: "global" })).toEqual([
      "knowledge",
      "detail",
      "user_role.md",
      "global",
      "",
      "",
      "",
    ]);

    expect(knowledgeKeys.searches()).toEqual(["knowledge", "search"]);
    expect(
      knowledgeKeys.search("launch", {
        scope: "workspace",
        workspaceId: "ws_launch",
      })
    ).toEqual(["knowledge", "search", "launch", "workspace", "ws_launch", "", ""]);

    expect(knowledgeKeys.decisions()).toEqual(["knowledge", "decisions"]);
    expect(
      knowledgeKeys.decisionsFor("global/user.md", {
        scope: "global",
      })
    ).toEqual(["knowledge", "decisions", "global/user.md", "global", "", "", ""]);
  });

  it("Should pad missing selector segments with empty strings", () => {
    expect(knowledgeKeys.list()).toEqual(["knowledge", "list", "", "", "", ""]);
    expect(knowledgeKeys.detail("test.md")).toEqual([
      "knowledge",
      "detail",
      "test.md",
      "",
      "",
      "",
      "",
    ]);
  });

  it("Should keep list and detail keys rooted at the all key", () => {
    expect(knowledgeKeys.list({ scope: "global" })[0]).toBe(knowledgeKeys.all[0]);
    expect(knowledgeKeys.detail("test.md", { scope: "global" })[0]).toBe(knowledgeKeys.all[0]);
    expect(knowledgeKeys.search("x", { scope: "global" })[0]).toBe(knowledgeKeys.all[0]);
    expect(knowledgeKeys.decisionsFor("test.md", { scope: "global" })[0]).toBe(
      knowledgeKeys.all[0]
    );
  });
});
