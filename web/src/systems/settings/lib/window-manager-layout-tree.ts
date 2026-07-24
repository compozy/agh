import type {
  WindowManagerLayoutDocument,
  WindowManagerLayoutNode,
} from "./window-manager-layout-types";

export function layoutNodeWindowIds(node: WindowManagerLayoutNode): string[] {
  if (node.kind === "leaf") return [node.windowId];
  if (node.kind === "stack") return [...node.windowIds];
  return node.children.flatMap(layoutNodeWindowIds);
}

export function desktopWindowIds(
  document: WindowManagerLayoutDocument,
  desktopId: string
): string[] {
  return Object.values(document.windows).flatMap(window =>
    window.desktopId === desktopId ? [window.id] : []
  );
}
