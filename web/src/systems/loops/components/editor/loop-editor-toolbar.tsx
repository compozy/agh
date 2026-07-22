import {
  Check,
  Code2,
  GitFork,
  LayoutGrid,
  Maximize,
  Minus,
  Save,
  Share2,
  X,
  ZoomIn,
  ZoomOut,
} from "lucide-react";
import { useReactFlow, useViewport } from "@xyflow/react";
import type { ReactNode } from "react";

import { Button, cn, TabsList, TabsTrigger } from "@agh/ui";

import { LOOP_INVARIANTS, type LoopLintState } from "../../lib/loop-editor-lint";
import type { LoopSource } from "../../types";

interface LoopEditorToolbarProps {
  source: LoopSource;
  lint: LoopLintState;
  busy: boolean;
  positionsDirty: boolean;
  onAutoLayout: () => void;
  onSaveLayout: () => void;
}

/**
 * Canvas sub-toolbar (design §4.6): zoom/auto-layout, Graph|DSL toggle, fork
 * context, and the canonical invariant chips. Editor actions live in the shell
 * topbar via `useTopbarSlot`.
 */
export function LoopEditorToolbar({
  source,
  lint,
  busy,
  positionsDirty,
  onAutoLayout,
  onSaveLayout,
}: LoopEditorToolbarProps) {
  const flow = useReactFlow();
  const { zoom } = useViewport();
  const readOnlySource = source !== "workspace";

  return (
    <div className="flex h-10.5 flex-none items-center gap-2.5 border-b border-line bg-canvas-soft px-3.5">
      <div className="flex items-center gap-0.5 rounded-md border border-line-soft bg-input-fill p-0.5">
        <ToolIcon label="Auto layout" onClick={onAutoLayout}>
          <LayoutGrid aria-hidden="true" className="size-3.5" />
        </ToolIcon>
        <ToolIcon label="Zoom out" onClick={() => void flow.zoomOut()}>
          <ZoomOut aria-hidden="true" className="size-3.5" />
        </ToolIcon>
        <span className="min-w-[38px] px-1 text-center font-mono text-mono-id text-subtle">
          {Math.round(zoom * 100)}%
        </span>
        <ToolIcon label="Zoom in" onClick={() => void flow.zoomIn()}>
          <ZoomIn aria-hidden="true" className="size-3.5" />
        </ToolIcon>
        <ToolIcon label="Fit view" onClick={() => void flow.fitView()}>
          <Maximize aria-hidden="true" className="size-3.5" />
        </ToolIcon>
      </div>

      <Button
        data-testid="loop-editor-save"
        disabled={busy || !positionsDirty}
        onClick={onSaveLayout}
        size="sm"
        title="Persist node positions to the annotations sidecar. Structural edits publish through Publish."
        type="button"
        variant="ghost"
      >
        <Save aria-hidden="true" className="size-3.5" />
        Save layout
      </Button>

      <TabsList className="h-7 gap-0.5 rounded-md border border-line-soft bg-input-fill p-0.5">
        <TabsTrigger
          className="h-6 rounded-sm px-3 text-form-label after:hidden data-active:bg-elevated"
          value="graph"
        >
          <Share2 aria-hidden="true" className="size-3.5" />
          Graph
        </TabsTrigger>
        <TabsTrigger
          className="h-6 rounded-sm px-3 text-form-label after:hidden data-active:bg-elevated"
          value="dsl"
        >
          <Code2 aria-hidden="true" className="size-3.5" />
          DSL
        </TabsTrigger>
      </TabsList>

      {readOnlySource ? (
        <span className="flex items-center gap-1.5 text-form-hint text-subtle">
          <GitFork aria-hidden="true" className="size-3 text-faint" />
          Read-only {source} source · fork before publishing
        </span>
      ) : null}

      <div className="ml-auto flex items-center gap-1.5" data-testid="loop-invariant-chips">
        {LOOP_INVARIANTS.map(invariant => {
          const failed = lint.validated && lint.invariants[invariant.key] === "fail";
          // Before the first daemon verdict, chips are neutral — never a claimed pass.
          const status = !lint.validated ? "pending" : failed ? "fail" : "pass";
          return (
            <span
              key={invariant.key}
              className={cn(
                "inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-form-hint font-medium",
                status === "fail" && "bg-danger-tint text-danger",
                status === "pass" && "bg-success-tint text-success",
                status === "pending" && "bg-badge-fill text-subtle"
              )}
              data-testid={`loop-invariant-${invariant.key}`}
              data-status={status}
            >
              {status === "fail" ? (
                <X aria-hidden="true" className="size-3" />
              ) : status === "pass" ? (
                <Check aria-hidden="true" className="size-3" />
              ) : (
                <Minus aria-hidden="true" className="size-3" />
              )}
              {invariant.label}
            </span>
          );
        })}
      </div>
    </div>
  );
}

function ToolIcon({
  label,
  onClick,
  children,
}: {
  label: string;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <Button
      className="size-6 text-muted hover:bg-elevated hover:text-fg-strong"
      onClick={onClick}
      size="icon-xs"
      title={label}
      aria-label={label}
      type="button"
      variant="ghost"
    >
      {children}
    </Button>
  );
}
