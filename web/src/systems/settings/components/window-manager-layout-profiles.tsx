import { Button, Eyebrow, Input, NativeSelect, NativeSelectOption } from "@agh/ui";

import { useWindowManagerLayoutProfiles } from "../hooks/use-window-manager-layout-profiles";
import type {
  WindowManagerLayoutAspect,
  WindowManagerLayoutDocument,
  WindowManagerLayoutOverflow,
  WindowManagerLayoutResourceRecord,
  WindowManagerLayoutScopeKind,
} from "../lib/window-manager-layout-types";
import { SettingsGroup } from "./settings-group";

export interface WindowManagerLayoutProfilesProps {
  workspaceId: string;
  document: WindowManagerLayoutDocument;
  profiles: readonly WindowManagerLayoutResourceRecord[];
  onLoad: (document: WindowManagerLayoutDocument) => void;
}

/** Generic `window_layout` resource discovery and optimistic CRUD. */
export function WindowManagerLayoutProfiles(props: WindowManagerLayoutProfilesProps) {
  const editor = useWindowManagerLayoutProfiles(props);

  return (
    <SettingsGroup
      bare
      title="Named profiles"
      description="Profiles are `window_layout` resources and keep their global or workspace scope."
      action={
        <Button size="sm" type="button" variant="outline" onClick={editor.startNew}>
          New profile
        </Button>
      }
    >
      <div className="grid gap-4 min-[760px]:grid-cols-[minmax(15rem,0.8fr)_minmax(20rem,1.2fr)]">
        <div className="overflow-hidden rounded-lg border border-line bg-canvas-soft">
          {editor.profiles.length === 0 ? (
            <p className="px-4 py-5 text-form-label text-subtle">
              No layout profiles are visible to this operator.
            </p>
          ) : (
            editor.profiles.map(record => {
              const recordKey = `${record.scope.kind}:${record.scope.id}:${record.id}`;
              return (
                <button
                  key={recordKey}
                  className="flex w-full items-center gap-3 border-b border-line-soft px-3 py-2.5 text-left outline-none last:border-b-0 hover:bg-row-hover focus-visible:shadow-focus-inset"
                  data-selected={recordKey === editor.selectedKey}
                  type="button"
                  onClick={() => editor.selectProfile(record)}
                >
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-small-body font-medium text-fg">
                      {record.spec.displayName}
                    </span>
                    <span className="block truncate font-mono text-micro text-subtle">
                      {record.id}
                    </span>
                  </span>
                  <Eyebrow className="text-faint">{record.scope.kind}</Eyebrow>
                </button>
              );
            })
          )}
        </div>

        <div className="flex flex-col gap-3 rounded-lg border border-line bg-canvas p-4">
          <div className="grid gap-3 sm:grid-cols-2">
            <label className="flex flex-col gap-1 text-form-label text-muted">
              Profile name
              <Input
                value={editor.displayName}
                onChange={event => editor.setDisplayName(event.target.value)}
              />
            </label>
            <label className="flex flex-col gap-1 text-form-label text-muted">
              Resource ID
              <Input
                className="font-mono"
                value={editor.id}
                onChange={event => editor.setId(event.target.value)}
              />
            </label>
            <label className="flex flex-col gap-1 text-form-label text-muted">
              Scope
              <NativeSelect
                className="w-full"
                value={editor.scope}
                onChange={event =>
                  editor.setScope(event.target.value as WindowManagerLayoutScopeKind)
                }
              >
                <NativeSelectOption value="workspace">Workspace</NativeSelectOption>
                <NativeSelectOption value="global">Global</NativeSelectOption>
              </NativeSelect>
            </label>
            <label className="flex flex-col gap-1 text-form-label text-muted">
              Aspect
              <NativeSelect
                className="w-full"
                value={editor.aspect}
                onChange={event =>
                  editor.setAspect(event.target.value as WindowManagerLayoutAspect)
                }
              >
                <NativeSelectOption value="any">Any</NativeSelectOption>
                <NativeSelectOption value="landscape">Landscape</NativeSelectOption>
                <NativeSelectOption value="portrait">Portrait</NativeSelectOption>
              </NativeSelect>
            </label>
            <label className="flex flex-col gap-1 text-form-label text-muted sm:col-span-2">
              Overflow
              <NativeSelect
                className="w-full"
                value={editor.overflow}
                onChange={event =>
                  editor.setOverflow(event.target.value as WindowManagerLayoutOverflow)
                }
              >
                <NativeSelectOption value="stack">Adapt to stack</NativeSelectOption>
                <NativeSelectOption value="reject">Reject</NativeSelectOption>
              </NativeSelect>
            </label>
          </div>
          {editor.error ? (
            <p className="text-form-label text-danger">{editor.error.message}</p>
          ) : null}
          <div className="flex justify-end gap-2">
            <Button
              type="button"
              variant="ghost"
              disabled={editor.selected === null || editor.remove.isPending}
              onClick={() => editor.remove.mutate()}
            >
              Delete
            </Button>
            <Button
              type="button"
              disabled={
                editor.id.trim() === "" ||
                editor.displayName.trim() === "" ||
                editor.save.isPending ||
                editor.remove.isPending
              }
              onClick={() => editor.save.mutate()}
            >
              {editor.save.isPending ? "Saving…" : "Save profile"}
            </Button>
          </div>
        </div>
      </div>
    </SettingsGroup>
  );
}
