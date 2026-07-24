import { Button, Eyebrow, Input, NativeSelect, NativeSelectOption, RadioCard } from "@agh/ui";

import type { WindowManagerLayoutProfilesModel } from "../hooks/use-window-manager-layout-profiles";
import type {
  WindowManagerLayoutAspect,
  WindowManagerLayoutOverflow,
  WindowManagerLayoutScopeKind,
} from "../lib/window-manager-layout-types";
import { SettingsGroup } from "./settings-group";

export interface WindowManagerLayoutProfilesProps {
  editor: WindowManagerLayoutProfilesModel;
}

/** Generic `window_layout` resource discovery with server-confirmed CRUD. */
export function WindowManagerLayoutProfiles({ editor }: WindowManagerLayoutProfilesProps) {
  return (
    <SettingsGroup
      bare
      title="Named profiles"
      description="Profiles are `window_layout` resources and keep their global or workspace scope."
      action={
        <Button
          className="min-h-11"
          size="sm"
          type="button"
          variant="outline"
          onClick={editor.startNew}
        >
          New profile
        </Button>
      }
    >
      <div className="grid gap-4 min-[760px]:grid-cols-[minmax(15rem,0.8fr)_minmax(20rem,1.2fr)]">
        <div
          aria-label="Layout profiles"
          className="flex flex-col gap-2 rounded-lg border border-line bg-canvas-soft p-2"
          role="radiogroup"
        >
          {editor.profiles.length === 0 ? (
            <p className="px-4 py-5 text-form-label text-subtle">
              No layout profiles are visible to this operator.
            </p>
          ) : (
            editor.profiles.map(record => {
              const recordKey = `${record.scope.kind}:${record.scope.id}:${record.id}`;
              return (
                <RadioCard
                  key={recordKey}
                  className="min-h-11"
                  selected={recordKey === editor.selectedKey}
                  title={record.spec.displayName}
                  description={<span className="font-mono text-micro">{record.id}</span>}
                  badge={<Eyebrow className="text-faint">{record.scope.kind}</Eyebrow>}
                  onSelect={() => editor.selectProfile(record)}
                />
              );
            })
          )}
        </div>

        <div className="flex flex-col gap-3 rounded-lg border border-line bg-canvas p-4">
          <div className="grid gap-3 sm:grid-cols-2">
            <label className="flex flex-col gap-1 text-form-label text-muted">
              Profile name
              <Input
                className="h-11"
                value={editor.displayName}
                onChange={event => editor.setDisplayName(event.target.value)}
              />
            </label>
            <label className="flex flex-col gap-1 text-form-label text-muted">
              Resource ID
              <Input
                className="h-11 font-mono"
                value={editor.id}
                onChange={event => editor.setId(event.target.value)}
              />
            </label>
            <label className="flex flex-col gap-1 text-form-label text-muted">
              Scope
              <NativeSelect
                className="w-full [&>select]:h-11"
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
                className="w-full [&>select]:h-11"
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
                className="w-full [&>select]:h-11"
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
              className="min-h-11"
              type="button"
              variant="ghost"
              disabled={editor.selected === null || editor.remove.isPending}
              onClick={() => editor.remove.mutate()}
            >
              Delete
            </Button>
            <Button
              className="min-h-11"
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
