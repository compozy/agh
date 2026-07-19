import { Check, Save, Upload } from "lucide-react";

import { Button } from "@agh/ui";

interface LoopEditorTopbarActionsProps {
  version: number | undefined;
  isDirty: boolean;
  positionsDirty: boolean;
  busy: boolean;
  publishDisabled: boolean;
  onValidate: () => void;
  onSaveDraft: () => void;
  onPublish: () => void;
}

/**
 * Trailing shell-topbar cluster for the loop editor: dirty chips, published
 * version badge, Validate / Save layout / Publish.
 */
export function LoopEditorTopbarActions({
  version,
  isDirty,
  positionsDirty,
  busy,
  publishDisabled,
  onValidate,
  onSaveDraft,
  onPublish,
}: LoopEditorTopbarActionsProps) {
  return (
    <div className="flex items-center gap-2.5" data-testid="loop-editor-topbar-actions">
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
  );
}
