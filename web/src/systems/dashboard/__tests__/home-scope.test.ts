import { describe, expect, it } from "vitest";

import { homeScopeForActiveWorkspace } from "../lib/home-scope";
import { normalizeHomeUsageWindow } from "../types";

const HOME_DIR = "/Users/tester";

function workspace(id: string, rootDir: string) {
  return { id, root_dir: rootDir, name: id };
}

describe("homeScopeForActiveWorkspace", () => {
  it("Should map the home workspace to the global scope with an empty param", () => {
    const scope = homeScopeForActiveWorkspace(workspace("ws-home", HOME_DIR), HOME_DIR);
    expect(scope.workspaceParam).toBe("");
    expect(scope.taskScope).toEqual({ scope: "global" });
  });

  it("Should scope a project workspace by its id", () => {
    const scope = homeScopeForActiveWorkspace(
      workspace("ws-proj", "/Users/tester/dev/proj"),
      HOME_DIR
    );
    expect(scope.workspaceParam).toBe("ws-proj");
    expect(scope.taskScope).toEqual({ scope: "workspace", workspace: "ws-proj" });
  });

  it("Should fall back to the global scope while resolution is incomplete", () => {
    expect(homeScopeForActiveWorkspace(undefined, HOME_DIR).workspaceParam).toBe("");
    expect(homeScopeForActiveWorkspace(workspace("ws", "/x"), undefined).workspaceParam).toBe("");
  });
});

describe("normalizeHomeUsageWindow", () => {
  it("Should keep supported windows and fall back to 30 otherwise", () => {
    expect(normalizeHomeUsageWindow(7)).toBe(7);
    expect(normalizeHomeUsageWindow(90)).toBe(90);
    expect(normalizeHomeUsageWindow(13)).toBe(30);
    expect(normalizeHomeUsageWindow(Number.NaN)).toBe(30);
  });
});
