import { render, screen } from "@testing-library/react";
import { syncDataLoaderFeature } from "@headless-tree/core";
import { useTree } from "@headless-tree/react";
import { describe, expect, it, vi } from "vitest";

import { Tree, TreeItem, TreeItemLabel } from "./tree";

interface TestTreeItem {
  kind: "root" | "folder" | "leaf";
  label: string;
}

const testData: Record<string, TestTreeItem> = {
  root: { kind: "root", label: "" },
  folder: { kind: "folder", label: "Marketing" },
  leaf: { kind: "leaf", label: "Sales agent" },
};

const testChildren: Record<string, string[]> = {
  root: ["folder"],
  folder: ["leaf"],
  leaf: [],
};

function TreeHarness() {
  const tree = useTree<TestTreeItem>({
    rootItemId: "root",
    getItemName: item => item.getItemData().label,
    isItemFolder: item => item.getItemData().kind === "folder",
    initialState: { expandedItems: ["folder"] },
    dataLoader: {
      getItem: id => testData[id] ?? { kind: "leaf", label: "" },
      getChildren: id => testChildren[id] ?? [],
    },
    features: [syncDataLoaderFeature],
  });

  return (
    <Tree tree={tree} aria-label="Tree test">
      {tree.getItems().map(item => {
        const data = item.getItemData();
        if (data.kind === "root") return null;
        return (
          <TreeItem key={item.getId()} item={item} data-testid={`tree-item-${item.getId()}`}>
            <TreeItemLabel item={item}>{data.label}</TreeItemLabel>
          </TreeItem>
        );
      })}
    </Tree>
  );
}

describe("Tree", () => {
  it("Should emit aria-expanded for folders only", () => {
    render(<TreeHarness />);

    expect(screen.getByTestId("tree-item-folder")).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByTestId("tree-item-leaf")).not.toHaveAttribute("aria-expanded");
  });

  it("Should ignore missing optional tree features without warning", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => undefined);
    try {
      render(<TreeHarness />);
      expect(warn).not.toHaveBeenCalled();
    } finally {
      warn.mockRestore();
    }
  });
});
