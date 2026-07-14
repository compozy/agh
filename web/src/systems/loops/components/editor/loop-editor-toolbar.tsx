import {
  Check,
  Code2,
  GitFork,
  LayoutGrid,
  Maximize,
  Minus,
  PencilLine,
  Save,
  Share2,
  Upload,
  X,
  ZoomIn,
  ZoomOut,
} from "lucide-react";
import { useReactFlow, useViewport } from "@xyflow/react";
import type { ReactNode } from "react";

import { Button, cn } from "@agh/ui";

import { LOOP_INVARIANTS, type LoopLintState } from "../../lib/loop-editor-lint";
import type { LoopEditorView } from "../../hooks/use-loop-editor";
import type { LoopSource } from "../../types";

interface LoopEditorToolbarProps {
  loopName: string;
  version: number | undefined;
  source: LoopSource;
  isDirty: boolean;
  positionsDirty: boolean;
  view: LoopEditorView;
  onViewChange: (view: LoopEditorView) => void;
  lint: LoopLintState;
  busy: boolean;
  publishDisabled: boolean;
  onAutoLayout: () => void;
  onValidate: () => void;
  onSaveDraft: () => void;
  onPublish: () => void;
}

/**
 * The editor action row + sub-toolbar (design §4.6): the Unsaved-changes chip, version
 * selector, Validate / Save draft / Publish (disabled while issues exist), plus the
 * zoom/auto-layout controls, the Graph|DSL toggle, the fork context, and the four
 * canonical invariant chips + a reference chip. Zoom rides `@xyflow/react` viewport state.
 */
export function LoopEditorToolbar({
  loopName,
  version,
  source,
  isDirty,
  positionsDirty,
  view,
  onViewChange,
  lint,
  busy,
  publishDisabled,
  onAutoLayout,
  onValidate,
  onSaveDraft,
  onPublish,
}: LoopEditorToolbarProps) {
  const flow = useReactFlow();
  const { zoom } = useViewport();
  const readOnlySource = source !== "workspace";
  const ContextIcon = readOnlySource ? GitFork : PencilLine;

  return (
    <div className="flex-none border-b border-line">
      <div className="flex h-12 items-center gap-2.5 bg-canvas px-4">
        <ContextIcon aria-hidden="true" className="size-4 text-accent-strong" />
        <span className="text-sm font-medium text-fg-strong">{loopName}</span>
        <span className="text-sm text-muted">· {readOnlySource ? "Fork & edit" : "Edit"}</span>
        <div className="ml-auto flex items-center gap-2.5">
          {isDirty ? (
            <span
              className="inline-flex items-center gap-1.5 rounded-full bg-warning-tint px-2.5 py-1 text-[11px] font-medium text-warning"
              data-testid="loop-editor-dirty-chip"
            >
              <span aria-hidden="true" className="size-1.5 rounded-full bg-current" />
              Unsaved changes
            </span>
          ) : positionsDirty ? (
            <span
              className="inline-flex items-center gap-1.5 rounded-full bg-badge-fill px-2.5 py-1 text-[11px] font-medium text-subtle"
              data-testid="loop-editor-layout-dirty-chip"
            >
              <span aria-hidden="true" className="size-1.5 rounded-full bg-current" />
              Layout unsaved
            </span>
          ) : null}
          <span
            className="inline-flex items-center gap-1.5 rounded-md border border-line bg-btn-fill px-2.5 py-1 font-mono text-[11px] text-muted"
            data-testid="loop-editor-version"
            title={
              isDirty
                ? `Published v${version ?? "?"} · unpublished edits (Publish bumps the version)`
                : `Published v${version ?? "?"}`
            }
          >
            v{version ?? "?"}
            <span className="text-faint">· {isDirty ? "unpublished edits" : "published"}</span>
          </span>
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={busy}
            onClick={onValidate}
            data-testid="loop-editor-validate"
          >
            <Check aria-hidden="true" className="size-3.5" />
            Validate
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={busy || !positionsDirty}
            onClick={onSaveDraft}
            title="Persist node positions to the annotations sidecar (structural edits publish via Publish)."
            data-testid="loop-editor-save"
          >
            <Save aria-hidden="true" className="size-3.5" />
            Save layout
          </Button>
          <Button
            type="button"
            variant="primary"
            size="sm"
            disabled={publishDisabled}
            onClick={onPublish}
            data-testid="loop-editor-publish"
          >
            <Upload aria-hidden="true" className="size-3.5" />
            Publish
          </Button>
        </div>
      </div>

      <div className="flex h-[42px] items-center gap-2.5 bg-canvas-soft px-3.5">
        <div className="flex items-center gap-0.5 rounded-md border border-line-soft bg-input-fill p-0.5">
          <ToolIcon label="Auto layout" onClick={onAutoLayout}>
            <LayoutGrid aria-hidden="true" className="size-3.5" />
          </ToolIcon>
          <ToolIcon label="Zoom out" onClick={() => void flow.zoomOut()}>
            <ZoomOut aria-hidden="true" className="size-3.5" />
          </ToolIcon>
          <span className="min-w-[38px] px-1 text-center font-mono text-[11px] text-subtle">
            {Math.round(zoom * 100)}%
          </span>
          <ToolIcon label="Zoom in" onClick={() => void flow.zoomIn()}>
            <ZoomIn aria-hidden="true" className="size-3.5" />
          </ToolIcon>
          <ToolIcon label="Fit view" onClick={() => void flow.fitView()}>
            <Maximize aria-hidden="true" className="size-3.5" />
          </ToolIcon>
        </div>

        <div
          className="flex items-center gap-0.5 rounded-md border border-line-soft bg-input-fill p-0.5"
          role="tablist"
        >
          <ViewTab active={view === "graph"} onClick={() => onViewChange("graph")}>
            <Share2 aria-hidden="true" className="size-3.5" />
            Graph
          </ViewTab>
          <ViewTab active={view === "dsl"} onClick={() => onViewChange("dsl")}>
            <Code2 aria-hidden="true" className="size-3.5" />
            DSL
          </ViewTab>
        </div>

        {readOnlySource ? (
          <span className="flex items-center gap-1.5 text-[11.5px] text-subtle">
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
                  "inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-medium",
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
    <button
      type="button"
      onClick={onClick}
      title={label}
      aria-label={label}
      className="grid size-6 place-items-center rounded-sm text-muted hover:bg-elevated hover:text-fg-strong"
    >
      {children}
    </button>
  );
}

function ViewTab({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={active}
      aria-controls="loop-editor-tabpanel"
      onClick={onClick}
      className={cn(
        "inline-flex h-6 items-center gap-1.5 rounded-sm px-3 text-[12px] font-medium transition-colors",
        active ? "bg-elevated text-fg-strong" : "text-muted hover:text-fg-strong"
      )}
    >
      {children}
    </button>
  );
}
