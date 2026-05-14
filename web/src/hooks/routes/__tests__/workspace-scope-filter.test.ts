import { describe, expect, it } from "vitest";

import { workspaceFilterForActiveScope } from "../workspace-scope-filter";

describe("workspaceFilterForActiveScope", () => {
  it("keeps all-scope operational queries bound to the active workspace", () => {
    expect(workspaceFilterForActiveScope("all", "ws_alpha")).toBe("ws_alpha");
  });

  it("keeps workspace-scope queries bound to the active workspace", () => {
    expect(workspaceFilterForActiveScope("workspace", "ws_alpha")).toBe("ws_alpha");
  });

  it("leaves global-scope queries unbound by workspace", () => {
    expect(workspaceFilterForActiveScope("global", "ws_alpha")).toBeUndefined();
  });

  it("omits the workspace filter when no active workspace exists", () => {
    expect(workspaceFilterForActiveScope("all", null)).toBeUndefined();
  });
});
