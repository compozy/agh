import { Button, Eyebrow, Input, NativeSelect, NativeSelectOption, cn } from "@agh/ui";

import { layoutNodeWindowIds } from "../lib/window-manager-layout-tree";
import type {
  WindowManagerLayoutAxis,
  WindowManagerLayoutNode,
} from "../lib/window-manager-layout-types";

let generatedNodeSequence = 0;

function nextNodeId(kind: WindowManagerLayoutNode["kind"]): string {
  generatedNodeSequence += 1;
  return `${kind}:settings-${generatedNodeSequence}`;
}

function toStack(node: WindowManagerLayoutNode): WindowManagerLayoutNode {
  const windowIds = layoutNodeWindowIds(node);
  return {
    id: nextNodeId("stack"),
    kind: "stack",
    windowIds,
    activeId: windowIds[0] ?? "",
  };
}

function toSplit(
  node: WindowManagerLayoutNode,
  axis: WindowManagerLayoutAxis
): WindowManagerLayoutNode {
  const windowIds = layoutNodeWindowIds(node);
  return {
    id: nextNodeId("split"),
    kind: "split",
    axis,
    children: windowIds.map(windowId => ({
      id: nextNodeId("leaf"),
      kind: "leaf" as const,
      windowId,
    })),
    weights: windowIds.map(() => 1),
  };
}

export interface WindowManagerLayoutNodeEditorProps {
  node: WindowManagerLayoutNode;
  availableWindowIds: readonly string[];
  depth?: number;
  onChange: (node: WindowManagerLayoutNode) => void;
}

/** Recursive structural editor for leaf, stack, and weighted split nodes. */
export function WindowManagerLayoutNodeEditor({
  node,
  availableWindowIds,
  depth = 0,
  onChange,
}: WindowManagerLayoutNodeEditorProps) {
  const descendants = layoutNodeWindowIds(node);
  const canStructure = descendants.length >= 2;

  return (
    <div
      className={cn(
        "flex flex-col gap-2 border border-line bg-canvas p-3",
        depth > 0 && "ml-layout-node-indent"
      )}
      data-node-kind={node.kind}
    >
      <div className="flex flex-wrap items-center gap-2">
        <Eyebrow className="text-subtle">{node.kind}</Eyebrow>
        <span className="min-w-0 flex-1 truncate font-mono text-mono-id text-muted">{node.id}</span>
        {node.kind !== "split" && canStructure ? (
          <>
            <Button
              size="xs"
              className="min-h-11"
              type="button"
              variant="ghost"
              onClick={() => onChange(toSplit(node, "horizontal"))}
            >
              Split rows
            </Button>
            <Button
              size="xs"
              className="min-h-11"
              type="button"
              variant="ghost"
              onClick={() => onChange(toSplit(node, "vertical"))}
            >
              Split columns
            </Button>
          </>
        ) : null}
        {node.kind !== "stack" && canStructure ? (
          <Button
            className="min-h-11"
            size="xs"
            type="button"
            variant="ghost"
            onClick={() => onChange(toStack(node))}
          >
            Stack
          </Button>
        ) : null}
      </div>

      {node.kind === "leaf" ? (
        <label className="flex items-center justify-between gap-3 text-form-label text-muted">
          Window
          <NativeSelect
            className="w-56 [&>select]:h-11"
            size="sm"
            value={node.windowId}
            onChange={event => onChange({ ...node, windowId: event.target.value })}
          >
            {availableWindowIds.map(windowId => (
              <NativeSelectOption key={windowId} value={windowId}>
                {windowId}
              </NativeSelectOption>
            ))}
          </NativeSelect>
        </label>
      ) : null}

      {node.kind === "stack" ? (
        <div className="flex flex-col gap-2">
          <label className="flex items-center justify-between gap-3 text-form-label text-muted">
            Active window
            <NativeSelect
              className="w-56 [&>select]:h-11"
              size="sm"
              value={node.activeId}
              onChange={event => onChange({ ...node, activeId: event.target.value })}
            >
              {node.windowIds.map(windowId => (
                <NativeSelectOption key={windowId} value={windowId}>
                  {windowId}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          </label>
          <div className="flex flex-wrap gap-1.5">
            {node.windowIds.map(windowId => (
              <span
                key={windowId}
                className="border border-line-soft bg-canvas-soft px-2 py-1 font-mono text-micro text-muted"
              >
                {windowId}
              </span>
            ))}
          </div>
        </div>
      ) : null}

      {node.kind === "split" ? (
        <div className="flex flex-col gap-2">
          <label className="flex items-center justify-between gap-3 text-form-label text-muted">
            Axis
            <NativeSelect
              className="w-40 [&>select]:h-11"
              size="sm"
              value={node.axis}
              onChange={event =>
                onChange({
                  ...node,
                  axis: event.target.value as WindowManagerLayoutAxis,
                })
              }
            >
              <NativeSelectOption value="horizontal">Horizontal</NativeSelectOption>
              <NativeSelectOption value="vertical">Vertical</NativeSelectOption>
            </NativeSelect>
          </label>
          {node.children.map((child, index) => (
            <div className="grid grid-cols-[minmax(0,1fr)_7rem] items-start gap-2" key={child.id}>
              <WindowManagerLayoutNodeEditor
                availableWindowIds={availableWindowIds}
                depth={depth + 1}
                node={child}
                onChange={next => {
                  const children = [...node.children];
                  children[index] = next;
                  onChange({ ...node, children });
                }}
              />
              <label className="flex flex-col gap-1 text-form-label text-muted">
                Weight
                <Input
                  className="h-11"
                  min={0.01}
                  size={6}
                  step={0.05}
                  type="number"
                  value={node.weights[index] ?? 1}
                  onChange={event => {
                    const weights = [...node.weights];
                    weights[index] = Number(event.target.value);
                    onChange({ ...node, weights });
                  }}
                />
              </label>
            </div>
          ))}
        </div>
      ) : null}
    </div>
  );
}
