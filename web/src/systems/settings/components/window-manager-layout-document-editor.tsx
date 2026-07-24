import { Button, Eyebrow, Input } from "@agh/ui";

import type { WindowManagerLayoutEditorModel } from "../hooks/use-window-manager-layout-editor";
import type { WindowManagerLayoutProfilesModel } from "../hooks/use-window-manager-layout-profiles";
import { desktopWindowIds, layoutNodeWindowIds } from "../lib/window-manager-layout-tree";
import { windowManagerLayoutDocumentToWire } from "../lib/window-manager-layout-schema";
import type { WindowManagerLayoutDocument } from "../lib/window-manager-layout-types";
import { SettingsGroup } from "./settings-group";
import { WindowManagerLayoutNodeEditor } from "./window-manager-layout-node-editor";
import { WindowManagerLayoutProfiles } from "./window-manager-layout-profiles";

interface WindowManagerLayoutDocumentEditorProps {
  editor: WindowManagerLayoutEditorModel;
  profilesEditor: WindowManagerLayoutProfilesModel;
}

function downloadDocument(document: WindowManagerLayoutDocument): void {
  const blob = new Blob([JSON.stringify(windowManagerLayoutDocumentToWire(document), null, 2)], {
    type: "application/json",
  });
  const url = URL.createObjectURL(blob);
  const anchor = window.document.createElement("a");
  anchor.href = url;
  anchor.download = "window-layout.json";
  anchor.click();
  URL.revokeObjectURL(url);
}

/** Raw document editor with mandatory daemon validation + semantic preview fence. */
export function WindowManagerLayoutDocumentEditor({
  editor,
  profilesEditor,
}: WindowManagerLayoutDocumentEditorProps) {
  return (
    <div className="flex flex-col gap-6">
      <WindowManagerLayoutProfiles editor={profilesEditor} />

      <SettingsGroup
        bare
        title="Current document"
        description="Edit normalized group frames, split axes and weights, or convert a branch to a stack."
        action={
          <div className="flex flex-wrap gap-2">
            <input
              ref={editor.importInput}
              accept="application/json,.json"
              className="hidden"
              type="file"
              onChange={event => {
                const file = event.target.files?.[0];
                if (file) void editor.importDocument(file);
                event.target.value = "";
              }}
            />
            <Button
              size="sm"
              className="min-h-11"
              type="button"
              variant="outline"
              onClick={() => editor.importInput.current?.click()}
            >
              Import JSON
            </Button>
            <Button
              size="sm"
              className="min-h-11"
              type="button"
              variant="outline"
              onClick={() => downloadDocument(editor.draft)}
            >
              Export JSON
            </Button>
          </div>
        }
      >
        {editor.importError ? (
          <p className="text-form-label text-danger">{editor.importError}</p>
        ) : null}
        <div className="flex flex-col gap-4">
          {editor.draft.desktops.map((desktop, desktopIndex) => {
            const availableWindowIds = desktopWindowIds(editor.draft, desktop.id);
            return (
              <section
                className="flex flex-col gap-3 rounded-lg border border-line bg-canvas-soft p-4"
                key={desktop.id}
              >
                <header className="flex flex-wrap items-center gap-3">
                  <Eyebrow className="text-subtle">Desktop {desktop.order + 1}</Eyebrow>
                  <Input
                    className="h-11 min-w-48 flex-1"
                    value={desktop.name}
                    onChange={event => {
                      const desktops = structuredClone(editor.draft.desktops);
                      const target = desktops[desktopIndex];
                      if (!target) return;
                      target.name = event.target.value;
                      editor.updateDraft({ ...editor.draft, desktops });
                    }}
                  />
                  <span className="font-mono text-micro text-faint">{desktop.id}</span>
                </header>

                {desktop.groups.map((group, groupIndex) => (
                  <div className="flex flex-col gap-3" key={group.id}>
                    <div className="flex flex-wrap items-end gap-2">
                      <span className="mr-auto font-mono text-mono-id text-muted">{group.id}</span>
                      {(["x", "y", "w", "h"] as const).map(key => (
                        <label
                          className="flex w-20 flex-col gap-1 text-form-label text-muted"
                          key={key}
                        >
                          {key}
                          <Input
                            className="h-11"
                            max={1}
                            min={0}
                            step={0.05}
                            type="number"
                            value={group.frame[key]}
                            onChange={event =>
                              editor.setGroupFrame(
                                desktopIndex,
                                groupIndex,
                                key,
                                Number(event.target.value)
                              )
                            }
                          />
                        </label>
                      ))}
                    </div>
                    <WindowManagerLayoutNodeEditor
                      availableWindowIds={availableWindowIds}
                      node={group.root}
                      onChange={root => editor.setGroupRoot(desktopIndex, groupIndex, root)}
                    />
                    <p className="font-mono text-micro text-faint">
                      {layoutNodeWindowIds(group.root).length} structural windows
                    </p>
                  </div>
                ))}

                {desktop.floating.length > 0 ? (
                  <p className="font-mono text-micro text-subtle">
                    Floating: {desktop.floating.join(", ")}
                  </p>
                ) : null}
              </section>
            );
          })}
        </div>
      </SettingsGroup>

      {editor.validation !== null || editor.reviewed !== null ? (
        <section
          className="border border-line bg-canvas px-4 py-3"
          data-testid="window-manager-layout-review"
        >
          <p className="text-small-body font-semibold text-fg">
            {editor.validation?.valid ? "Daemon validation passed" : "Daemon validation failed"}
          </p>
          {editor.reviewed ? (
            <p className="mt-1 text-form-label text-muted">
              Preview {editor.reviewed.preview.changed ? "changes" : "keeps"} the layout ·{" "}
              {editor.reviewed.preview.changes.desktopIds.length} desktops ·{" "}
              {editor.reviewed.preview.changes.windowIds.length} windows
            </p>
          ) : null}
          {editor.validation?.diagnostics.map(diagnostic => (
            <p
              className="mt-1 font-mono text-micro text-danger"
              key={`${diagnostic.code}:${diagnostic.path}`}
            >
              {diagnostic.path ? `${diagnostic.path}: ` : ""}
              {diagnostic.message}
            </p>
          ))}
        </section>
      ) : null}

      <div className="sticky bottom-3 flex flex-wrap items-center justify-end gap-2 border border-line bg-elevated px-3 py-2 shadow-overlay">
        {editor.mutationError ? (
          <p className="mr-auto text-form-label text-danger">{editor.mutationError.message}</p>
        ) : (
          <p className="mr-auto text-form-label text-subtle">
            Revision {editor.revision} · apply unlocks only after validation and preview.
          </p>
        )}
        <Button
          className="min-h-11"
          type="button"
          variant="ghost"
          disabled={!editor.dirty || editor.review.isPending || editor.apply.isPending}
          onClick={editor.reset}
        >
          Reset
        </Button>
        <Button
          className="min-h-11"
          type="button"
          variant="outline"
          disabled={editor.review.isPending || editor.apply.isPending}
          onClick={() => editor.review.mutate()}
        >
          {editor.review.isPending ? "Reviewing…" : "Validate and preview"}
        </Button>
        <Button
          className="min-h-11"
          type="button"
          disabled={!editor.reviewCurrent || editor.apply.isPending || editor.review.isPending}
          onClick={() => editor.apply.mutate()}
        >
          {editor.apply.isPending ? "Applying…" : "Apply reviewed layout"}
        </Button>
      </div>
    </div>
  );
}
