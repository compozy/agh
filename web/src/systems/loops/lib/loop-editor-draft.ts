import type { EditorEdge, EditorNode, RawLoopEdge, RawLoopNode } from "./codec";
import type { FieldPath } from "./loop-node-schema";

/**
 * Immutable draft mutations over the editor graph. Every inspector edit flows through
 * `setNodeField`, which writes a value at a raw-JSON path on exactly one node, leaving
 * every other field (and node) untouched so the bijective codec still round-trips the
 * rest. Node-id renames cascade through the edges and the React Flow node id. Drafts
 * are editor-session state only — there is no server-side draft store in v1 (§9.13).
 */

/** Reads the value at a raw-JSON path (dotted/indexed), or `undefined` when absent. */
export function getAtPath(source: unknown, path: FieldPath): unknown {
  let current: unknown = source;
  for (const segment of path) {
    if (current === null || typeof current !== "object") return undefined;
    current = (current as Record<string | number, unknown>)[segment];
  }
  return current;
}

function cloneContainer(value: unknown, key: string | number): Record<string | number, unknown> {
  if (Array.isArray(value)) return [...value] as unknown as Record<string | number, unknown>;
  if (value && typeof value === "object") return { ...(value as Record<string | number, unknown>) };
  // Create the missing container: an array when the next key is a numeric index.
  return (typeof key === "number" ? [] : {}) as Record<string | number, unknown>;
}

/**
 * Returns a structural copy of `source` with `value` set at `path`. A `value` of
 * `undefined` deletes the leaf key rather than writing `undefined` (so clearing an
 * optional override removes it from the definition instead of pinning a null field).
 */
export function setAtPath<T extends object>(source: T, path: FieldPath, value: unknown): T {
  if (path.length === 0) return source;
  const [head, ...rest] = path;
  const existing = (source as Record<string | number, unknown>)[head];
  // Clearing a value under a missing parent is a no-op — don't synthesize an empty
  // container (e.g. clearing `retry.max` on a node with no `retry` must not add `retry: {}`).
  if (value === undefined && rest.length > 0 && (existing === undefined || existing === null)) {
    return source;
  }
  const clone = cloneContainer(source, head);
  if (rest.length === 0) {
    if (value === undefined) {
      if (Array.isArray(clone)) clone.splice(head as number, 1);
      else delete clone[head];
    } else {
      clone[head] = value;
    }
    return clone as unknown as T;
  }
  clone[head] = setAtPath(cloneContainer(clone[head], rest[0]) as object, rest, value);
  return clone as unknown as T;
}

/** Applies one field edit to a node's raw JSON, returning a new node array. */
export function setNodeField(
  nodes: readonly EditorNode[],
  nodeId: string,
  path: FieldPath,
  value: unknown
): EditorNode[] {
  return nodes.map(node => {
    if (node.id !== nodeId) return node;
    const raw = setAtPath(node.data.raw, path, value) as RawLoopNode;
    return { ...node, data: { ...node.data, raw } };
  });
}

/** Whether a field path targets the node id (rename must cascade, not a plain set). */
export function isNodeIdPath(path: FieldPath): boolean {
  return path.length === 1 && path[0] === "id";
}

function renameEdgeRaw(raw: RawLoopEdge, oldId: string, newId: string): RawLoopEdge {
  const next: RawLoopEdge = { ...raw };
  if (next.from === oldId) next.from = newId;
  if (next.to === oldId) next.to = newId;
  return next;
}

/**
 * Renames a node id everywhere it is referenced: the raw node, the React Flow node id,
 * and every edge's `source`/`target` + raw `from`/`to`. Reference strings inside other
 * nodes' templates (`nodes.<id>...`) are intentionally NOT rewritten — the linter
 * surfaces the now-`unknown_reference` so the author fixes it deliberately.
 */
export function renameNodeId(
  nodes: readonly EditorNode[],
  edges: readonly EditorEdge[],
  oldId: string,
  newId: string
): { nodes: EditorNode[]; edges: EditorEdge[] } {
  const nextNodes = nodes.map(node => {
    if (node.id !== oldId) return node;
    const raw = { ...node.data.raw, id: newId };
    return { ...node, id: newId, data: { ...node.data, raw } };
  });
  const nextEdges = edges.map(edge => {
    const source = edge.source === oldId ? newId : edge.source;
    const target = edge.target === oldId ? newId : edge.target;
    if (source === edge.source && target === edge.target) return edge;
    return {
      ...edge,
      source,
      target,
      data: edge.data
        ? { ...edge.data, raw: renameEdgeRaw(edge.data.raw, oldId, newId) }
        : edge.data,
    };
  });
  return { nodes: nextNodes, edges: nextEdges };
}
