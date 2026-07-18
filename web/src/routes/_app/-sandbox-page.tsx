import { AlertCircle, Boxes, Pencil, Plus, Trash2 } from "lucide-react";

import {
  useSandboxPage,
  type SandboxDraft,
  type SandboxEditorState,
} from "@/hooks/routes/use-sandbox-page";
import {
  restartBannerPropsFor,
  SettingsEditorDialog,
  SettingsFieldRow,
  SettingsSourceBadge,
  type SettingsSandboxEntry,
} from "@/systems/settings";
import {
  Button,
  Empty,
  Eyebrow,
  Input,
  NativeSelect,
  NativeSelectOption,
  PageHead,
  PageShell,
  Pill,
  RestartBanner,
  Section,
  Spinner,
  StatusLine,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@agh/ui";
import { SandboxDeleteDialog, SandboxLastActionAlert } from "./-sandbox-dialogs";

export function SandboxPage() {
  const page = useSandboxPage();

  if (page.isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="sandbox-page-loading">
        <Spinner className="size-5 text-subtle" />
      </div>
    );
  }

  if (page.error || !page.envelope) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="sandbox-page-error">
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-danger" />
          <p className="text-sm text-subtle">{page.error?.message ?? "Failed to load sandboxes"}</p>
        </div>
      </div>
    );
  }

  const bannerProps = restartBannerPropsFor("sandbox", page.restart);
  const statusLine = (
    <StatusLine
      data-testid="sandbox-page-status-line"
      status="connected"
      items={[
        {
          key: "total",
          value: <span data-testid="sandbox-page-total">{page.counts.total} profiles</span>,
          tone: "neutral",
        },
        {
          key: "workspaces",
          value: (
            <span data-testid="sandbox-page-workspaces">
              {page.counts.totalWorkspaces} workspace references
            </span>
          ),
          tone: "neutral",
        },
      ]}
    />
  );

  return (
    <PageShell
      density="route"
      data-testid="sandbox-shell"
      banner={bannerProps ? <RestartBanner {...bannerProps} className="px-9" /> : undefined}
      head={
        <div className="flex flex-wrap items-start justify-between gap-3">
          <PageHead
            count={page.counts.total}
            data-testid="sandbox-page-head"
            icon={Boxes}
            title="Sandbox"
            variant="compact"
          />
          {statusLine}
        </div>
      }
    >
      {page.lastAction ? (
        <SandboxLastActionAlert action={page.lastAction} onDismiss={page.dismissLastAction} />
      ) : null}

      <Section
        data-testid="sandbox-page-header-row"
        label="Profiles"
        note={`${page.counts.total} defined · used across ${page.counts.totalWorkspaces} workspaces`}
        right={
          <Button
            type="button"
            variant="default"
            size="sm"
            onClick={page.openCreate}
            data-testid="sandbox-page-create"
          >
            <Plus className="size-3" />
            New sandbox profile
          </Button>
        }
      />

      {page.sandboxes.length === 0 ? (
        <Empty
          icon={Boxes}
          title="No sandbox profiles defined"
          description='Use "New sandbox profile" to create an overlay profile referenceable by workspaces.'
          data-testid="sandbox-page-empty"
        />
      ) : (
        <SandboxTable
          sandboxes={page.sandboxes}
          onEdit={page.openEdit}
          onDelete={page.openDelete}
        />
      )}

      <SandboxEditor
        editor={page.editor}
        isValid={page.editorIsValid}
        isSaving={page.editorIsSaving}
        error={page.editorError}
        warnings={page.editorWarnings}
        existingNames={page.sandboxes.map(entry => entry.name)}
        onChange={page.updateDraft}
        onClose={page.closeEditor}
        onSave={page.saveEditor}
      />

      <SandboxDeleteDialog
        target={page.deleteTarget.mode === "open" ? page.deleteTarget.entry : null}
        error={page.deleteError}
        isDeleting={page.deleteIsPending}
        onClose={page.closeDelete}
        onConfirm={page.confirmDelete}
      />
    </PageShell>
  );
}

function SandboxTable({
  sandboxes,
  onEdit,
  onDelete,
}: {
  sandboxes: SettingsSandboxEntry[];
  onEdit: (entry: SettingsSandboxEntry) => void;
  onDelete: (entry: SettingsSandboxEntry) => void;
}) {
  return (
    <div className="overflow-hidden rounded-lg border border-line" data-testid="sandbox-page-list">
      <Table>
        <TableHeader>
          <TableRow className="bg-elevated">
            <TableHead className="eyebrow text-muted">Name</TableHead>
            <TableHead className="eyebrow text-muted">Backend</TableHead>
            <TableHead className="eyebrow text-muted">Profile</TableHead>
            <TableHead className="eyebrow text-muted">Source</TableHead>
            <TableHead className="eyebrow text-right text-muted">Usage</TableHead>
            <TableHead className="eyebrow w-[1%] text-right text-muted">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {sandboxes.map(entry => (
            <SandboxRow key={entry.name} entry={entry} onEdit={onEdit} onDelete={onDelete} />
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function SandboxRow({
  entry,
  onEdit,
  onDelete,
}: {
  entry: SettingsSandboxEntry;
  onEdit: (entry: SettingsSandboxEntry) => void;
  onDelete: (entry: SettingsSandboxEntry) => void;
}) {
  const profile = entry.profile;
  const source = entry.source_metadata.effective_source;
  const shadowed = entry.source_metadata.shadowed_sources ?? [];
  const deletable = source.kind !== "builtin-provider";

  return (
    <TableRow data-testid={`sandbox-page-card-${entry.name}`}>
      <TableCell>
        <span className="font-mono text-sm text-fg">{entry.name}</span>
      </TableCell>
      <TableCell>
        <div className="flex flex-col gap-1">
          <Pill mono tone={backendTone(profile.backend)}>
            {profile.backend}
          </Pill>
          <span className="text-xs text-subtle">{backendLabel(profile.backend)}</span>
        </div>
      </TableCell>
      <TableCell className="text-xs">
        <div
          className="flex flex-col gap-0.5"
          data-testid={`sandbox-page-card-${entry.name}-profile`}
        >
          <ProfileLine label="sync_mode" value={profile.sync_mode ?? "--"} />
          <ProfileLine label="persistence" value={profile.persistence ?? "--"} />
          <ProfileLine label="runtime_root" value={profile.runtime_root ?? "--"} />
        </div>
      </TableCell>
      <TableCell>
        <SettingsSourceBadge
          data-testid={`sandbox-page-card-${entry.name}-source`}
          source={source}
          shadowed={shadowed}
        />
      </TableCell>
      <TableCell
        className="text-right font-mono text-xs text-muted"
        data-testid={`sandbox-page-card-${entry.name}-usage`}
      >
        {entry.workspace_usage_count}{" "}
        {entry.workspace_usage_count === 1 ? "workspace" : "workspaces"}
      </TableCell>
      <TableCell>
        <div className="flex items-center justify-end gap-1">
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            onClick={() => onEdit(entry)}
            aria-label={`Edit ${entry.name}`}
            data-testid={`sandbox-page-card-${entry.name}-edit`}
          >
            <Pencil className="size-3" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            onClick={() => onDelete(entry)}
            disabled={!deletable}
            aria-label={`Delete ${entry.name}`}
            title={
              deletable
                ? undefined
                : "Builtin sandboxes cannot be deleted -- override them instead."
            }
            data-testid={`sandbox-page-card-${entry.name}-delete`}
          >
            <Trash2 className="size-3" />
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}
function ProfileLine({ label, value }: { label: string; value: string }) {
  return (
    <span className="flex items-center gap-2 whitespace-nowrap">
      <Eyebrow className="text-muted">{label}</Eyebrow>
      <span className="font-mono text-fg">{value}</span>
    </span>
  );
}

function backendLabel(backend: string): string {
  const map: Record<string, string> = {
    local: "host process · no sandbox",
    daytona: "cloud workspace · Daytona",
  };
  return map[backend] ?? `custom backend · ${backend}`;
}
function backendTone(backend: string): "success" | "info" | "neutral" {
  if (backend === "local") return "success";
  if (backend === "daytona") return "info";
  return "neutral";
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
              : "Name is immutable -- create a new sandbox to rename."
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
        {preserved.join(", ")} -- edited outside this dialog and included as-is in the PUT replace.
      </span>
    </p>
  );
}
