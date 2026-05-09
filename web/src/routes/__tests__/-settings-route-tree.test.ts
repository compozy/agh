import { describe, expect, it } from "vitest";

import { routeTree } from "@/routeTree.gen";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type RouteLike = { options?: { id?: string }; children?: RouteLike[] | Record<string, RouteLike> };

function getChildren(node: RouteLike): RouteLike[] {
  if (!node.children) return [];
  return Array.isArray(node.children) ? node.children : Object.values(node.children);
}

function findChildById(node: RouteLike, id: string): RouteLike | undefined {
  return getChildren(node).find(child => child.options?.id === id);
}

describe("route tree , settings subtree", () => {
  it("mounts a /_app/settings shell containing the default index child", () => {
    const appRoute = findChildById(routeTree as unknown as RouteLike, "/_app");
    expect(appRoute, "expected /_app to be registered under root").toBeDefined();

    const settingsRoute = findChildById(appRoute as RouteLike, "/settings");
    expect(settingsRoute, "expected /settings shell under /_app").toBeDefined();

    const indexRoute = findChildById(settingsRoute as RouteLike, "/");
    expect(indexRoute, "expected default / index child under /_app/settings").toBeDefined();
  });

  it("keeps existing /_app siblings alongside the new settings subtree", () => {
    const appRoute = findChildById(routeTree as unknown as RouteLike, "/_app");
    const siblings = getChildren(appRoute as RouteLike).map(node => node.options?.id);

    expect(siblings).toEqual(
      expect.arrayContaining(["/jobs", "/triggers", "/network", "/skills", "/"])
    );
    expect(siblings).toContain("/settings");
  });
});
