import type { Root } from "fumadocs-core/page-tree";
import { getLayoutTabs } from "fumadocs-ui/layouts/shared";
import { describe, expect, it } from "vitest";
import { createRuntimeLayoutTree, resolveSidebarRoot } from "./runtime-navigation";

function getNodeName(node: { name: unknown }): string {
  return typeof node.name === "string" ? node.name : "";
}

const runtimePageTree: Root = {
  name: "Runtime",
  children: [
    {
      type: "folder",
      $id: "core",
      name: "Core Concepts",
      root: true,
      children: [
        {
          type: "page",
          $id: "core/index.mdx",
          name: "Index",
          url: "/runtime/core",
        },
        {
          type: "folder",
          $id: "core/overview",
          name: "Overview",
          children: [],
        },
      ],
    },
    {
      type: "folder",
      $id: "cli-reference",
      name: "CLI Reference",
      root: true,
      children: [
        {
          type: "page",
          $id: "cli-reference/index.mdx",
          name: "Index",
          url: "/runtime/cli-reference",
        },
        {
          type: "page",
          $id: "cli-reference/agh.mdx",
          name: "Agh",
          url: "/runtime/cli-reference/agh",
        },
      ],
    },
    {
      type: "folder",
      $id: "api-reference",
      name: "API Reference",
      root: true,
      children: [
        {
          type: "page",
          $id: "api-reference/index.mdx",
          name: "Index",
          url: "/runtime/api-reference",
        },
      ],
    },
  ],
};

describe("runtime navigation tree", () => {
  it("defaults the runtime landing page sidebar to core concepts", () => {
    const layoutTree = createRuntimeLayoutTree(runtimePageTree);
    const root = resolveSidebarRoot(layoutTree, "/runtime");

    expect(root.type).toBe("folder");
    if (root.type !== "folder") {
      throw new Error("expected runtime landing sidebar to resolve to a folder");
    }

    expect(getNodeName(root)).toBe("Core Concepts");
    expect(
      root.children.some(node => node.type === "folder" && getNodeName(node) === "Overview")
    ).toBe(true);
    expect(root.children.some(node => node.type === "page" && getNodeName(node) === "Index")).toBe(
      false
    );
    expect(
      root.children.some(node => getNodeName(node as { name: unknown }) === "CLI Reference")
    ).toBe(false);
    expect(
      root.children.some(node => getNodeName(node as { name: unknown }) === "API Reference")
    ).toBe(false);
  });

  it("keeps cli pages scoped to the cli reference root", () => {
    const layoutTree = createRuntimeLayoutTree(runtimePageTree);
    const root = resolveSidebarRoot(layoutTree, "/runtime/cli-reference/agh");

    expect(root.type).toBe("folder");
    if (root.type !== "folder") {
      throw new Error("expected cli sidebar to resolve to a folder");
    }

    expect(getNodeName(root)).toBe("CLI Reference");
    expect(root.children.some(node => node.type === "page" && getNodeName(node) === "Agh")).toBe(
      true
    );
  });

  it("preserves the original tabs and page tree targets", () => {
    const layoutTree = createRuntimeLayoutTree(runtimePageTree);
    const coreTab = getLayoutTabs(runtimePageTree).find(
      tab => getNodeName({ name: tab.title }) === "Core Concepts"
    );
    const coreFolder = runtimePageTree.children[0];

    expect(getLayoutTabs(runtimePageTree)).toHaveLength(3);
    expect(coreTab?.url).toBe("/runtime/core");
    expect(coreFolder.type).toBe("folder");
    if (coreFolder.type !== "folder") {
      throw new Error("expected core fixture to stay as a folder");
    }

    expect(coreFolder.children[0]).toMatchObject({
      type: "page",
      url: "/runtime/core",
    });
    expect(layoutTree.fallback).toBeDefined();
  });
});
