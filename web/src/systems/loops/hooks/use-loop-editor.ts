import { useEffect, useEffectEvent, useReducer, useRef, type SetStateAction } from "react";
import {
  addEdge,
  applyEdgeChanges,
  applyNodeChanges,
  type Connection,
  type EdgeChange,
  type NodeChange,
} from "@xyflow/react";
import { toast } from "sonner";

import { useLoop, useLoopAnnotations } from "./use-loops";
import { usePatchLoop, usePutLoopAnnotations, useValidateLoop } from "./use-loop-actions";
import { LoopValidationError } from "../adapters/loops-api";
import {
  definitionToGraph,
  editorEdgeId,
  graphToDefinition,
  type EditorEdge,
  type EditorNode,
  type RawLoopNode,
} from "../lib/codec";
import { buildDslView, type DslLine } from "../lib/loop-dsl";
import { isNodeIdPath, renameNodeId, setNodeField } from "../lib/loop-editor-draft";
import {
  applyLintToNodes,
  buildLintState,
  emptyLintState,
  type LoopLintState,
} from "../lib/loop-editor-lint";
import { layoutEditorGraph } from "../lib/loop-editor-layout";
import { buildNodeFields, type FieldPath, type FieldSpec } from "../lib/loop-node-schema";
import { uniqueNodeId, type PaletteItem } from "../lib/loop-palette";
import type { LoopDefinition, LoopDetail } from "../types";

const AUTO_VALIDATE_DEBOUNCE_MS = 400;

export type LoopEditorStatus = "no-workspace" | "loading" | "error" | "ready";
export type LoopEditorView = "graph" | "dsl";

export interface UseLoopEditorResult {
  status: LoopEditorStatus;
  loop: LoopDetail | undefined;
  errorMessage: string | undefined;
  version: number | undefined;
  nodes: EditorNode[];
  edges: EditorEdge[];
  selectedNode: EditorNode | null;
  selectedFields: FieldSpec[];
  /** Increments only on a selection switch (not a rename) — the inspector's remount key. */
  selectionSeq: number;
  view: LoopEditorView;
  setView: (view: LoopEditorView) => void;
  isDirty: boolean;
  positionsDirty: boolean;
  lint: LoopLintState;
  validateFailed: boolean;
  publishDisabled: boolean;
  busy: boolean;
  publishError: string | null;
  dslLines: DslLine[];
  onNodesChange: (changes: NodeChange<EditorNode>[]) => void;
  onEdgesChange: (changes: EdgeChange<EditorEdge>[]) => void;
  onConnect: (connection: Connection) => void;
  selectNode: (id: string | null) => void;
  revealNode: (id: string) => void;
  changeField: (path: FieldPath, value: unknown) => void;
  addNode: (item: PaletteItem) => void;
  autoLayout: () => void;
  validate: () => Promise<void>;
  publish: () => Promise<LoopDetail | null>;
  savePositions: () => Promise<void>;
}

function nodeKind(raw: RawLoopNode): string {
  return typeof raw.kind === "string" ? raw.kind : "";
}

function nextDropPosition(nodes: readonly EditorNode[]): { x: number; y: number } {
  if (nodes.length === 0) return { x: 40, y: 40 };
  const rightmost = nodes.reduce((max, node) => (node.position.x > max.position.x ? node : max));
  return { x: rightmost.position.x + 200, y: rightmost.position.y };
}

interface LoopEditorState {
  baseDefinition: LoopDefinition | null;
  edges: EditorEdge[];
  isDirty: boolean;
  lint: LoopLintState;
  nodes: EditorNode[];
  positionsDirty: boolean;
  publishError: string | null;
  selectedNodeId: string | null;
  selectionSeq: number;
  validateFailed: boolean;
  view: LoopEditorView;
}

interface LoopEditorStateAction {
  update: (current: LoopEditorState) => LoopEditorState;
}

function createLoopEditorState(): LoopEditorState {
  return {
    baseDefinition: null,
    edges: [],
    isDirty: false,
    lint: emptyLintState(),
    nodes: [],
    positionsDirty: false,
    publishError: null,
    selectedNodeId: null,
    selectionSeq: 0,
    validateFailed: false,
    view: "graph",
  };
}

function updateLoopEditorState(
  current: LoopEditorState,
  action: LoopEditorStateAction
): LoopEditorState {
  return action.update(current);
}

/**
 * The fork-and-edit editor view-model: it loads the one canonical definition + its
 * position sidecar, holds the draft as editor-session state (no server draft store,
 * §9.13), and drives the bijective codec, the shared-linter validate loop, the publish
 * (expected_version CAS), and the positions save. The GUI never owns invariants — every
 * chip and per-node badge comes from a `validate`/publish verdict.
 */
export function useLoopEditor(workspaceId: string, name: string): UseLoopEditorResult {
  const enabled = workspaceId !== "" && name !== "";
  const loopQuery = useLoop(workspaceId, name, enabled);
  const annotationsQuery = useLoopAnnotations(workspaceId, name, enabled);
  const validateMutation = useValidateLoop();
  const patchMutation = usePatchLoop();
  const annotationsMutation = usePutLoopAnnotations();

  const [editorState, dispatchEditorState] = useReducer(
    updateLoopEditorState,
    undefined,
    createLoopEditorState
  );
  const {
    baseDefinition,
    edges,
    isDirty,
    lint,
    nodes,
    positionsDirty,
    publishError,
    selectedNodeId,
    selectionSeq,
    validateFailed,
    view,
  } = editorState;
  const setEditorField = <Field extends keyof LoopEditorState>(
    field: Field,
    value: SetStateAction<LoopEditorState[Field]>
  ) => {
    dispatchEditorState({
      update: current => {
        const currentValue = current[field];
        const nextValue =
          typeof value === "function"
            ? (value as (previous: LoopEditorState[Field]) => LoopEditorState[Field])(currentValue)
            : value;
        return { ...current, [field]: nextValue };
      },
    });
  };
  const setBaseDefinition = (value: SetStateAction<LoopDefinition | null>) =>
    setEditorField("baseDefinition", value);
  const setEdges = (value: SetStateAction<EditorEdge[]>) => setEditorField("edges", value);
  const setDirty = (value: SetStateAction<boolean>) => setEditorField("isDirty", value);
  const setLint = (value: SetStateAction<LoopLintState>) => setEditorField("lint", value);
  const setNodes = (value: SetStateAction<EditorNode[]>) => setEditorField("nodes", value);
  const setPositionsDirty = (value: SetStateAction<boolean>) =>
    setEditorField("positionsDirty", value);
  const setPublishError = (value: SetStateAction<string | null>) =>
    setEditorField("publishError", value);
  const setSelectedNodeId = (value: SetStateAction<string | null>) =>
    setEditorField("selectedNodeId", value);
  const setSelectionSeq = (value: SetStateAction<number>) => setEditorField("selectionSeq", value);
  const setValidateFailed = (value: SetStateAction<boolean>) =>
    setEditorField("validateFailed", value);
  const setView = (value: SetStateAction<LoopEditorView>) => setEditorField("view", value);

  const initedKeyRef = useRef<string | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Monotonic token so an out-of-order debounced validate never overwrites a newer verdict.
  const validateSeqRef = useRef(0);
  const annotationsErrorNotifiedRef = useRef(false);

  // Seed the editable draft once the definition + settled sidecar arrive. This syncs
  // server state into local editor state (a legit external-system → draft sync).
  useEffect(() => {
    const definition = loopQuery.data?.definition;
    if (!definition || annotationsQuery.isLoading) return;
    const key = `${workspaceId}:${name}`;
    if (initedKeyRef.current === key) return;
    initedKeyRef.current = key;
    const graph = definitionToGraph(definition);
    const laid = layoutEditorGraph(graph.nodes, graph.edges, annotationsQuery.data ?? []);
    dispatchEditorState({
      update: () => ({
        ...createLoopEditorState(),
        baseDefinition: definition,
        edges: graph.edges,
        nodes: laid,
        selectedNodeId: laid[0]?.id ?? null,
      }),
    });
  }, [loopQuery.data, annotationsQuery.data, annotationsQuery.isLoading, workspaceId, name]);

  // Positions are cosmetic (auto-layout is the fallback), but a broken sidecar should be
  // observable, not silently swallowed. Surface it once per error, non-blocking.
  useEffect(() => {
    if (annotationsQuery.isError && !annotationsErrorNotifiedRef.current) {
      annotationsErrorNotifiedRef.current = true;
      toast.error("Could not load saved node positions — using auto-layout.");
    }
    if (!annotationsQuery.isError) annotationsErrorNotifiedRef.current = false;
  }, [annotationsQuery.isError]);

  const runValidation = async (options: { notify?: boolean } = {}) => {
    const base = baseDefinition;
    if (!base) return;
    const definition = graphToDefinition(base, nodes, edges);
    const seq = ++validateSeqRef.current;
    try {
      const result = await validateMutation.mutateAsync({
        workspaceId,
        name,
        data: { definition },
      });
      // Drop a stale verdict: a later validate has already superseded this one.
      if (seq !== validateSeqRef.current) return;
      setValidateFailed(false);
      const state = buildLintState(result);
      setLint(state);
      setNodes(current => applyLintToNodes(current, state.byNode));
    } catch {
      // A transport failure never fabricates a pass/fail (the daemon linter is the only
      // invariant authority). Mark the failure AND demote the verdict to unvalidated so a
      // stale "all pass" from a prior verdict can't linger over an edited graph the daemon
      // never confirmed — the dock shows "unavailable — retry", not a claimed pass.
      if (seq === validateSeqRef.current) {
        setValidateFailed(true);
        setLint(current => (current.validated ? { ...current, validated: false } : current));
      }
      if (options.notify) toast.error("Validation could not reach the daemon. Try again.");
    }
  };

  // Live re-lint after structural edits so the chips + Publish gate stay truthful.
  // The Effect Event reads the latest draft without making the debounce depend on
  // the render-local validation function identity.
  const runAutoValidation = useEffectEvent(() => runValidation());
  const structuralKey = JSON.stringify({
    n: nodes.map(node => node.data.raw),
    e: edges.map(edge => edge.data?.raw),
  });
  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      void runAutoValidation();
    }, AUTO_VALIDATE_DEBOUNCE_MS);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [structuralKey]);

  const onNodesChange = (changes: NodeChange<EditorNode>[]) => {
    setNodes(current => applyNodeChanges(changes, current));
    for (const change of changes) {
      if (change.type === "position") setPositionsDirty(true);
      if (change.type === "remove") setDirty(true);
    }
  };

  const onEdgesChange = (changes: EdgeChange<EditorEdge>[]) => {
    setEdges(current => applyEdgeChanges(changes, current));
    if (changes.some(change => change.type === "remove")) setDirty(true);
  };

  const onConnect = (connection: Connection) => {
    const { source, target } = connection;
    if (!source || !target) return;
    setEdges(current => {
      const edge: EditorEdge = {
        id: editorEdgeId(source, target, current.length),
        source,
        target,
        data: { raw: { from: source, to: target } },
      };
      return addEdge(edge, current);
    });
    setDirty(true);
  };

  // Bumped on a genuine selection *switch* (click / reveal / add) but NOT on a rename of the
  // already-selected node, so the inspector's field container is keyed by this — a rename
  // never remounts it (which would drop focus after each keystroke, R-001 round 7).
  const selectNode = (id: string | null) => {
    setSelectedNodeId(id);
    setSelectionSeq(seq => seq + 1);
  };
  const revealNode = (id: string) => {
    setSelectedNodeId(id);
    setSelectionSeq(seq => seq + 1);
    setView("graph");
  };

  const changeField = (path: FieldPath, value: unknown) => {
    const targetId = selectedNodeId;
    if (!targetId) return;
    setPublishError(null);
    if (isNodeIdPath(path)) {
      const newId = String(value).trim();
      if (newId === "" || newId === targetId) return;
      // Reject a rename onto an id another node already uses — two nodes sharing an id
      // would duplicate React Flow keys and make selection ambiguous before the daemon
      // rejects it. The author keeps the old id until they pick a free one.
      if (nodes.some(node => node.id === newId)) return;
      const renamed = renameNodeId(nodes, edges, targetId, newId);
      setNodes(renamed.nodes);
      setEdges(renamed.edges);
      setSelectedNodeId(newId);
    } else {
      setNodes(current => setNodeField(current, targetId, path, value));
    }
    setDirty(true);
  };

  const addNode = (item: PaletteItem) => {
    dispatchEditorState({
      update: current => {
        const existing = new Set(current.nodes.map(node => node.id));
        const id = uniqueNodeId(item.idBase, existing);
        const raw = item.buildRaw(id);
        const node: EditorNode = {
          id,
          type: "loopNode",
          position: nextDropPosition(current.nodes),
          data: { raw, nodeClass: item.nodeClass, kind: nodeKind(raw), hasError: false },
        };
        return {
          ...current,
          isDirty: true,
          nodes: [...current.nodes, node],
          selectedNodeId: id,
          selectionSeq: current.selectionSeq + 1,
        };
      },
    });
  };

  const publish = async (): Promise<LoopDetail | null> => {
    const base = baseDefinition;
    if (!base) return null;

    // Publish validates atomically on the daemon. Its verdict must not be overwritten by an
    // older passive validation that was queued or already in flight for the same draft.
    validateSeqRef.current += 1;
    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
      debounceRef.current = null;
    }

    setPublishError(null);
    const definition = graphToDefinition(base, nodes, edges);
    try {
      const updated = await patchMutation.mutateAsync({
        workspaceId,
        name,
        data: { definition, expected_version: base.meta.version ?? null },
      });
      setBaseDefinition(updated.definition);
      setDirty(false);
      // A successful publish means the daemon accepted the definition — a validated-clean
      // state, not the pre-validation neutral state.
      const clean = buildLintState({ valid: true, errors: [] });
      setLint(clean);
      setNodes(current => applyLintToNodes(current, clean.byNode));
      return updated;
    } catch (error) {
      // A publish 422 carries the same per-node lint body as validate — map it back onto
      // nodes + chips immediately (task-22 MUST), not just a generic banner.
      if (error instanceof LoopValidationError) {
        const state = buildLintState(error.result);
        setLint(state);
        setNodes(current => applyLintToNodes(current, state.byNode));
        setPublishError(
          `Publish rejected — ${state.errorCount} issue${state.errorCount === 1 ? "" : "s"} to resolve.`
        );
        return null;
      }
      setPublishError(error instanceof Error ? error.message : "Failed to publish loop");
      return null;
    }
  };

  const autoLayout = () => {
    setNodes(current => layoutEditorGraph(current, edges, []));
    setPositionsDirty(true);
  };

  const savePositions = async () => {
    const annotations = nodes.map(node => ({
      node_id: node.id,
      x: Math.round(node.position.x),
      y: Math.round(node.position.y),
    }));
    try {
      await annotationsMutation.mutateAsync({ workspaceId, name, data: { annotations } });
      setPositionsDirty(false);
    } catch {
      // Mirror the load-failure toast: a save failure keeps the "Layout unsaved" chip and
      // must not be silently swallowed by the caller's `void`.
      toast.error("Could not save node positions. Try again.");
    }
  };

  const selectedNode = nodes.find(node => node.id === selectedNodeId) ?? null;
  const selectedFields = selectedNode
    ? buildNodeFields(selectedNode.data.raw, baseDefinition ?? undefined)
    : [];
  const dslBase = baseDefinition;
  // Only serialize when the DSL panel is visible — skip graph conversion + YAML
  // emission entirely while editing on the Graph canvas.
  const dslLines =
    dslBase && view === "dsl"
      ? buildDslView(
          graphToDefinition(dslBase, nodes, edges) as unknown as Record<string, unknown>,
          lint.byNode
        )
      : [];

  const status: LoopEditorStatus =
    workspaceId === ""
      ? "no-workspace"
      : loopQuery.isLoading
        ? "loading"
        : loopQuery.error || !loopQuery.data
          ? "error"
          : "ready";

  const busy =
    validateMutation.isPending || patchMutation.isPending || annotationsMutation.isPending;

  return {
    status,
    loop: loopQuery.data,
    errorMessage: loopQuery.error?.message,
    version: baseDefinition?.meta.version ?? loopQuery.data?.version,
    nodes,
    edges,
    selectedNode,
    selectedFields,
    selectionSeq,
    view,
    setView,
    isDirty,
    positionsDirty,
    lint,
    validateFailed,
    // Gated on known blocking errors only: when no verdict exists (e.g. validate is
    // unreachable) Publish stays enabled because publish runs the shared linter atomically
    // and returns a 422 the editor maps onto nodes — no invalid definition can ship.
    publishDisabled: loopQuery.data?.source !== "workspace" || lint.hasBlockingErrors || busy,
    busy,
    publishError,
    dslLines,
    onNodesChange,
    onEdgesChange,
    onConnect,
    selectNode,
    revealNode,
    changeField,
    addNode,
    autoLayout,
    validate: () => runValidation({ notify: true }),
    publish,
    savePositions,
  };
}
