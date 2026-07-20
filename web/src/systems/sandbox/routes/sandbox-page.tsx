import { AlertCircle, Boxes, Info, Plus, RefreshCw } from "lucide-react";

import {
  BlockLoading,
  Button,
  Empty,
  Eyebrow,
  Input,
  ListingPage,
  ListingToolbar,
  NativeSelect,
  NativeSelectOption,
  PageHead,
  RestartBanner,
  useTopbarSlot,
} from "@agh/ui";

import {
  useSandboxPage,
  type SandboxDraft,
  type SandboxEditorState,
  type SandboxRouteSearch,
} from "../hooks/use-sandbox-page";
import { SandboxListFilters, SandboxProfilesList, SandboxProfileSheet } from "../components";
import {
  restartBannerPropsFor,
  SettingsEditorDialog,
  SettingsFieldRow,
  SettingsSourceBadge,
} from "@/systems/settings";
import { SandboxDeleteDialog, SandboxLastActionAlert } from "../components/sandbox-dialogs";

export function SandboxPage({ search = {} }: { search?: SandboxRouteSearch }) {
  const page = useSandboxPage(search);

  useTopbarSlot({
    actions: (
      <div className="flex items-center gap-2" data-testid="sandbox-topbar-actions">
        <Button
          data-testid="sandbox-page-refresh"
          disabled={page.isRefetching}
          onClick={() => void page.refetch()}
          size="sm"
          type="button"
          variant="ghost"
        >
          <RefreshCw className={page.isRefetching ? "size-3 animate-spin" : "size-3"} />
          Refresh
        </Button>
        <Button data-testid="sandbox-page-create" onClick={page.openCreate} size="sm" type="button">
          <Plus className="size-3" />
          New sandbox profile
        </Button>
      </div>
    ),
  });

  if (page.isLoading) {
    return <BlockLoading className="flex-1" data-testid="sandbox-page-loading" />;
  }

  const bannerProps = restartBannerPropsFor("sandbox", page.restart);
  const banner = (
    <>
      {bannerProps ? <RestartBanner {...bannerProps} className="px-9" /> : null}
      {page.lastAction ? (
        <div className="px-9 pt-4">
          <SandboxLastActionAlert action={page.lastAction} onDismiss={page.dismissLastAction} />
        </div>
      ) : null}
    </>
  );

  return (
    <ListingPage banner={banner} data-testid="sandbox-shell">
      <PageHead
        count={page.counts.total}
        countTestId="sandbox-page-count"
        data-testid="sandbox-page-head"
        icon={Boxes}
        meta={
          <>
            <span>Execution boundary profiles that workspaces and sessions select by name.</span>
            <PageHead.MetaDot />
            <span data-testid="sandbox-page-total">
              {page.counts.total} {page.counts.total === 1 ? "profile" : "profiles"}
            </span>
            <PageHead.MetaDot />
            <span data-testid="sandbox-page-workspaces">
              {page.counts.totalWorkspaces} workspace references
            </span>
          </>
        }
        title="Sandbox"
      />

      <ListingToolbar>
        <ListingToolbar.Leading>
          <ListingToolbar.Search
            aria-label="Search profiles"
            data-testid="sandbox-page-search"
            onChange={page.setQuery}
            placeholder="Search profiles"
            value={page.query}
          />
          <ListingToolbar.Filters>
            <SandboxListFilters
              backend={page.backend}
              onBackendChange={page.setBackend}
              onPersistenceChange={page.setPersistence}
              persistence={page.persistence}
            />
          </ListingToolbar.Filters>
        </ListingToolbar.Leading>
        <ListingToolbar.Trailing>
          <ListingToolbar.ViewToggle onChange={page.setView} value={page.view} />
        </ListingToolbar.Trailing>
      </ListingToolbar>

      <p
        className="mb-3 flex items-center gap-2 text-xs text-subtle"
        data-testid="sandbox-page-sec-note"
      >
        <Info aria-hidden="true" className="size-3.5 shrink-0 text-faint" />
        <span>
          Profiles are global. When <span className="font-mono text-[11px]">defaults.sandbox</span>{" "}
          is unset, sessions fall back to a synthetic local profile.
        </span>
      </p>

      {page.queryError && page.sandboxes.length === 0 ? (
        <Empty
          action={
            <Button
              data-testid="sandbox-page-error-retry"
              disabled={page.isRefetching}
              onClick={() => void page.refetch()}
              size="sm"
              type="button"
              variant="ghost"
            >
              <RefreshCw className={page.isRefetching ? "size-3 animate-spin" : "size-3"} />
              Retry
            </Button>
          }
          data-testid="sandbox-page-error"
          description={
            page.queryError ?? "The daemon stopped responding before it returned the profile list."
          }
          icon={AlertCircle}
          title="Failed to load sandboxes"
        />
      ) : (
        <SandboxProfilesList
          data-testid="sandbox-page-list"
          error={page.queryError ? new Error(page.queryError) : null}
          hasActiveFilters={page.hasActiveFilters}
          onClearFilters={page.clearFilters}
          onCreate={page.openCreate}
          onDelete={page.openDelete}
          onEdit={page.openEdit}
          onSelect={page.openInspect}
          profiles={page.filtered}
          selectedName={page.selectedEntry?.name ?? null}
          view={page.view}
        />
      )}

      <SandboxProfileSheet
        entry={page.selectedEntry}
        onOpenChange={open => {
          if (!open) page.closeInspect();
        }}
        onRequestDelete={page.openDelete}
        onRequestEdit={page.openEdit}
        open={page.selectedEntry !== null}
      />

      <SandboxEditor
        editor={page.editor}
        error={page.editorError}
        existingNames={page.sandboxes.map(entry => entry.name)}
        isSaving={page.editorIsSaving}
        isValid={page.editorIsValid}
        onChange={page.updateDraft}
        onClose={page.closeEditor}
        onSave={page.saveEditor}
        warnings={page.editorWarnings}
      />

      <SandboxDeleteDialog
        error={page.deleteError}
        isDeleting={page.deleteIsPending}
        onClose={page.closeDelete}
        onConfirm={page.confirmDelete}
        target={page.deleteTarget.mode === "open" ? page.deleteTarget.entry : null}
      />
    </ListingPage>
  );
}

interface SandboxEditorProps {
  editor: SandboxEditorState;
  isValid: boolean;
  isSaving: boolean;
  error: string | null;
  warnings?: string[];
  existingNames: string[];
  onChange: (updater: (draft: SandboxDraft) => SandboxDraft) => void;
  onClose: () => void;
  onSave: () => void;
}

function SandboxEditor({
  editor,
  isValid,
  isSaving,
  error,
  warnings,
  existingNames,
  onChange,
  onClose,
  onSave,
}: SandboxEditorProps) {
  const open = editor.mode !== "closed";
  if (!open) return null;

  const isCreate = editor.mode === "create";
  const draft = editor.draft;
  const entry = editor.mode === "edit" ? editor.entry : null;

  const title = isCreate
    ? "New sandbox profile"
    : `Edit sandbox · ${editor.mode === "edit" ? editor.name : ""}`;
  const description = isCreate
    ? "Create a new sandbox overlay. Saving writes a new overlay entry."
    : "Saving replaces the overlay profile with the values below (full PUT). Unset optional fields are cleared.";

  const lowerName = draft.name.trim().toLowerCase();
  const nameConflict =
    isCreate &&
    lowerName.length > 0 &&
    existingNames.some(existing => existing.toLowerCase() === lowerName);

  return (
    <SettingsEditorDialog
      open={open}
      mode={isCreate ? "create" : "edit"}
      slug="sandbox"
      title={title}
      description={description}
      metadata={
        entry ? (
          <div className="flex flex-col gap-1">
            <SettingsSourceBadge
              data-testid="sandbox-editor-source"
              source={entry.source_metadata.effective_source}
              shadowed={entry.source_metadata.shadowed_sources ?? []}
            />
            {entry.workspace_usage_count > 0 ? (
              <span className="text-xs text-subtle" data-testid="sandbox-editor-usage">
                {entry.workspace_usage_count} workspaces depend on this profile
              </span>
            ) : null}
          </div>
        ) : null
      }
      error={error ?? (nameConflict ? `A sandbox named "${draft.name}" already exists.` : null)}
      warnings={warnings}
      canSave={isValid && !nameConflict}
      isSaving={isSaving}
      saveLabel={isCreate ? "Create sandbox profile" : "Replace profile"}
      onSave={onSave}
      onOpenChange={next => {
        if (!next) onClose();
      }}
    >
      <div className="flex flex-col gap-3">
        <SettingsFieldRow
          variant="modal"
          data-testid="sandbox-editor-name"
          label="Name"
          description={
            isCreate
              ? "Lower-case identifier referenced by workspaces."
              : "Name is immutable — create a new sandbox to rename."
          }
          hint={isCreate ? "REQUIRED" : "LOCKED"}
          control={
            <Input
              className="w-56 font-mono disabled:opacity-60"
              data-testid="sandbox-editor-name-input"
              value={draft.name}
              placeholder="e.g. local"
              disabled={!isCreate}
              onChange={event => onChange(current => ({ ...current, name: event.target.value }))}
            />
          }
        />
        <SettingsFieldRow
          variant="modal"
          data-testid="sandbox-editor-backend"
          label="Backend"
          description="Which execution backend the sandbox uses."
          hint="REQUIRED"
          control={
            <NativeSelect
              className="w-56 font-mono"
              data-testid="sandbox-editor-backend-input"
              value={draft.backend}
              onChange={event => onChange(current => ({ ...current, backend: event.target.value }))}
            >
              <NativeSelectOption value="local">local</NativeSelectOption>
              <NativeSelectOption value="daytona">daytona</NativeSelectOption>
            </NativeSelect>
          }
        />
        <SettingsFieldRow
          variant="modal"
          data-testid="sandbox-editor-sync-mode"
          label="Sync mode"
          description="How files move between host and sandbox."
          hint="OPTIONAL"
          control={
            <Input
              className="w-56 font-mono"
              data-testid="sandbox-editor-sync-mode-input"
              value={draft.sync_mode}
              placeholder="none | session-bidir | turn-bidir"
              onChange={event =>
                onChange(current => ({ ...current, sync_mode: event.target.value }))
              }
            />
          }
        />
        <SettingsFieldRow
          variant="modal"
          data-testid="sandbox-editor-persistence"
          label="Persistence"
          description="Workspace lifecycle between sessions."
          hint="OPTIONAL"
          control={
            <Input
              className="w-56 font-mono"
              data-testid="sandbox-editor-persistence-input"
              value={draft.persistence}
              placeholder="transient | reuse | archive"
              onChange={event =>
                onChange(current => ({ ...current, persistence: event.target.value }))
              }
            />
          }
        />
        <SettingsFieldRow
          variant="modal"
          data-testid="sandbox-editor-runtime-root"
          label="Runtime root"
          description="Directory mounted as the working root."
          hint="OPTIONAL"
          control={
            <Input
              className="w-72 font-mono"
              data-testid="sandbox-editor-runtime-root-input"
              value={draft.runtime_root}
              placeholder="~ | /workspace | /home/user"
              onChange={event =>
                onChange(current => ({ ...current, runtime_root: event.target.value }))
              }
            />
          }
        />
        <PreservedFieldsNotice
          preserved={[
            draft.preserved.daytona ? "daytona" : null,
            draft.preserved.network ? "network" : null,
            draft.preserved.env ? "env" : null,
          ].filter((value): value is string => Boolean(value))}
        />
      </div>
    </SettingsEditorDialog>
  );
}

function PreservedFieldsNotice({ preserved }: { preserved: string[] }) {
  if (preserved.length === 0) return null;
  return (
    <p
      className="rounded-md border border-line bg-elevated px-3 py-2 text-xs text-subtle"
      data-testid="sandbox-editor-preserved"
    >
      <Eyebrow className="text-muted">preserved on save</Eyebrow>
      <span className="ml-2">
        {preserved.join(", ")} — edited outside this dialog and included as-is in the PUT replace.
      </span>
    </p>
  );
}
