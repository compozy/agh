import { Button } from "@agh/ui";
import type { WindowManagerConfig } from "@/systems/os";

import { useWindowManagerConfigEditor } from "../hooks/use-window-manager-config-editor";
import {
  WindowManagerBehaviorFields,
  WindowManagerBindingFields,
  WindowManagerGeometryFields,
} from "./window-manager-config-fields";

interface WindowManagerConfigEditorProps {
  config: WindowManagerConfig;
}

/** Complete global config editor; successful writes update the shared hot Query atom. */
export function WindowManagerConfigEditor({
  config: initialConfig,
}: WindowManagerConfigEditorProps) {
  const editor = useWindowManagerConfigEditor(initialConfig);

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
          type="button"
          variant="ghost"
          disabled={!editor.dirty || editor.isSaving}
          onClick={editor.reset}
        >
          Reset
        </Button>
        <Button type="button" disabled={!editor.canSave} onClick={editor.save}>
          {editor.isSaving ? "Saving…" : "Save settings"}
        </Button>
      </div>
    </div>
  );
}
