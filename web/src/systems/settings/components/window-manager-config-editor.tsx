import { Button } from "@agh/ui";

import type { WindowManagerConfigEditorModel } from "../hooks/use-window-manager-config-editor";
import {
  WindowManagerBehaviorFields,
  WindowManagerBindingFields,
  WindowManagerGeometryFields,
} from "./window-manager-config-fields";

interface WindowManagerConfigEditorProps {
  editor: WindowManagerConfigEditorModel;
}

/** Presentational global config editor driven by the Settings route model. */
export function WindowManagerConfigEditor({ editor }: WindowManagerConfigEditorProps) {
  return (
    <div className="flex flex-col gap-6">
      <WindowManagerBehaviorFields draft={editor.draft} setDraft={editor.setDraft} />
      <WindowManagerGeometryFields
        draft={editor.draft}
        ratioText={editor.ratioText}
        ratiosValid={editor.ratiosValid}
        setDraft={editor.setDraft}
        setRatioText={editor.setRatioText}
      />
      <WindowManagerBindingFields
        draft={editor.draft}
        shortcutsText={editor.shortcutsText}
        shortcutsValid={editor.shortcutsValid}
        setDraft={editor.setDraft}
        setShortcutsText={editor.setShortcutsText}
      />

      <div className="sticky bottom-3 flex items-center justify-end gap-2 border border-line bg-elevated px-3 py-2 shadow-overlay">
        {editor.error ? (
          <p className="mr-auto text-form-label text-danger">{editor.error.message}</p>
        ) : null}
        <Button
          className="min-h-11"
          type="button"
          variant="ghost"
          disabled={!editor.dirty || editor.isSaving}
          onClick={editor.reset}
        >
          Reset
        </Button>
        <Button className="min-h-11" type="button" disabled={!editor.canSave} onClick={editor.save}>
          {editor.isSaving ? "Saving…" : "Save settings"}
        </Button>
      </div>
    </div>
  );
}
