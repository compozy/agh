import { createElement } from "react";
import { Book, Layers } from "lucide-react";
import { searchPath } from "fumadocs-core/breadcrumb";
import type { Folder, Item, Node, Root } from "fumadocs-core/page-tree";

const CORE_FOLDER_ID = "core";
const OVERVIEW_PAGE_ID = "runtime-overview";

function isFolder(node: Node): node is Folder {
  return node.type === "folder";
}

function isPage(node: Node): node is Item {
  return node.type === "page";
}

function findRuntimeRootFolder(pageTree: Root, id: string): Folder | undefined {
  return pageTree.children.find(
    (node): node is Folder => isFolder(node) && node.root === true && node.$id === id
  );
}

function buildOverviewPage(): Item {
  return {
    type: "page",
    $id: OVERVIEW_PAGE_ID,
    name: "Overview",
    url: "/runtime",
    icon: createElement(Book),
  };
}

function buildCoreConceptsPage(landingPage: Item): Item {
  return {
    ...landingPage,
    name: "Core Concepts",
    icon: createElement(Layers),
  };
}

function rebuildCoreFolder(coreFolder: Folder): Folder {
  const landingPage = coreFolder.children.find(isPage);
  if (!landingPage) return coreFolder;

  return {
    ...coreFolder,
    children: [
      buildOverviewPage(),
      buildCoreConceptsPage(landingPage),
      ...coreFolder.children.filter(child => child.$id !== landingPage.$id),
    ],
  };
}

function createRuntimeLandingFallback(pageTree: Root): Root | undefined {
  const coreFolder = findRuntimeRootFolder(pageTree, CORE_FOLDER_ID);
  if (!coreFolder) return undefined;

  const rebuiltCore = rebuildCoreFolder(coreFolder);
  return {
    ...pageTree,
    $id: `${pageTree.$id ?? "runtime"}-landing-fallback`,
    children: pageTree.children.map(node =>
      isFolder(node) && node.$id === coreFolder.$id ? rebuiltCore : node
    ),
  };
}

export function createRuntimeLayoutTree(pageTree: Root): Root {
  const coreFolder = findRuntimeRootFolder(pageTree, CORE_FOLDER_ID);
  if (!coreFolder) return pageTree;

  const layoutTree: Root = {
    ...pageTree,
    $id: `${pageTree.$id ?? "runtime"}-layout`,
    children: pageTree.children.map(node =>
      isFolder(node) && node.$id === coreFolder.$id ? rebuildCoreFolder(node) : node
    ),
  };

  const landingFallback = createRuntimeLandingFallback(pageTree);
  if (landingFallback) {
    layoutTree.fallback = landingFallback;
  }

  return layoutTree;
}

export function resolveSidebarRoot(tree: Root, url: string): Root | Folder {
  const path =
    searchPath(tree.children, url) ??
    (tree.fallback ? searchPath(tree.fallback.children, url) : null) ??
    [];

  return (
    path.findLast((item): item is Folder => item.type === "folder" && item.root === true) ?? tree
  );
}
