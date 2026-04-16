import { searchPath } from "fumadocs-core/breadcrumb";
import type { Folder, Node, Root } from "fumadocs-core/page-tree";

function isFolder(node: Node): node is Folder {
  return node.type === "folder";
}

function findRuntimeRootFolder(pageTree: Root, id: string): Folder | undefined {
  return pageTree.children.find(
    (node): node is Folder => isFolder(node) && node.root === true && node.$id === id
  );
}

function createRuntimeLandingFallback(pageTree: Root): Root | undefined {
  const coreFolder = findRuntimeRootFolder(pageTree, "core");
  const landingPage = coreFolder?.children.find(
    (child): child is Extract<Node, { type: "page" }> => child.type === "page"
  );

  if (!coreFolder || !landingPage) return undefined;

  return {
    ...pageTree,
    $id: `${pageTree.$id ?? "runtime"}-landing-fallback`,
    children: pageTree.children.map(node =>
      isFolder(node) && node.$id === coreFolder.$id
        ? {
            ...node,
            index: {
              ...landingPage,
              url: "/runtime",
            },
            children: node.children.filter(child => child.$id !== landingPage.$id),
          }
        : node
    ),
  };
}

export function createRuntimeLayoutTree(pageTree: Root): Root {
  const landingFallback = createRuntimeLandingFallback(pageTree);
  if (!landingFallback) return pageTree;

  return {
    ...pageTree,
    $id: `${pageTree.$id ?? "runtime"}-layout`,
    fallback: landingFallback,
  };
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
