import { AlertCircle, Boxes, Check, Loader2, Pencil, Plus, Trash2, X } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import {
  Alert,
  AlertAction,
  AlertDescription,
  Button,
  Empty,
  Input,
  MonoBadge,
  NativeSelect,
  NativeSelectOption,
  PageHeader,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@agh/ui";
import {
  useSandboxPage,
  type SandboxDraft,
  type SandboxEditorState,
  type SandboxLastAction,
} from "@/hooks/routes/use-sandbox-page";
import type { SettingsSandboxEntry } from "@/systems/settings";
import {
  SettingsCollectionHeader,
  SettingsDeleteDialog,
  SettingsEditorDialog,
  SettingsFieldRow,
  SettingsPageActions,
  SettingsRestartBanner,
  SettingsSourceBadge,
  SettingsStatusLine,
} from "@/systems/settings/components";

export const Route = createFileRoute("/_app/sandbox")({
  component: SandboxPage,
});

function SandboxPage() {
  const page = useSandboxPage();

  if (page.isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="sandbox-page-loading">
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (page.error || !page.envelope) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="sandbox-page-error">
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {page.error?.message ?? "Failed to load sandboxes"}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="sandbox-shell">
      <PageHeader
        count={page.counts.total}
        controls={<SettingsPageActions slug="sandbox" restart={page.restart} />}
        icon={() => <Boxes className="size-3.5" data-testid="sandbox-shell-icon" />}
        meta={
          <SettingsStatusLine
            data-testid="sandbox-page-status-line"
            daemonAvailable
            items={[
              <span key="total" data-testid="sandbox-page-total">
                {page.counts.total} profiles
              </span>,
              <span key="workspaces" data-testid="sandbox-page-workspaces">
                {page.counts.totalWorkspaces} workspace references
              </span>,
            ]}
          />
        }
        title={<span data-testid="sandbox-shell-title">Sandbox</span>}
      />
      <SettingsRestartBanner
        slug="sandbox"
        restart={page.restart}
        className="px-6 md:px-8 xl:px-10"
      />
      <div className="flex min-h-0 flex-1 flex-col gap-5 overflow-y-auto px-4 py-5 sm:px-6 md:px-8 md:py-6 xl:px-10">
        {page.lastAction ? (
          <ActionResultBanner action={page.lastAction} onDismiss={page.dismissLastAction} />
        ) : null}

        <SettingsCollectionHeader
          data-testid="sandbox-page-header-row"
          eyebrow="Profiles"
          summary={`${page.counts.total} defined · used across ${page.counts.totalWorkspaces} workspaces`}
          action={
            <Button
              type="button"
              variant="default"
              size="sm"
              onClick={page.openCreate}
              data-testid="sandbox-page-create"
            >
              <Plus className="size-3.5" />
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
      </div>
    </div>
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
    <div
      className="overflow-hidden rounded-lg border border-[color:var(--color-divider)]"
      data-testid="sandbox-page-list"
    >
      <Table>
        <TableHeader>
          <TableRow className="bg-[color:var(--color-surface-elevated)]">
            <TableHead className="text-[0.6rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
              Name
            </TableHead>
            <TableHead className="text-[0.6rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
              Backend
            </TableHead>
            <TableHead className="text-[0.6rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
              Profile
            </TableHead>
            <TableHead className="text-[0.6rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
              Source
            </TableHead>
            <TableHead className="text-right text-[0.6rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
              Usage
            </TableHead>
            <TableHead className="w-[1%] text-right text-[0.6rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
              Actions
            </TableHead>
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
        <span className="font-mono text-sm text-[color:var(--color-text-primary)]">
          {entry.name}
        </span>
      </TableCell>
      <TableCell>
        <div className="flex flex-col gap-1">
          <MonoBadge tone={backendTone(profile.backend)}>{profile.backend}</MonoBadge>
          <span className="text-xs text-[color:var(--color-text-tertiary)]">
            {backendLabel(profile.backend)}
          </span>
        </div>
      </TableCell>
      <TableCell className="text-xs">
        <div
          className="flex flex-col gap-0.5"
          data-testid={`sandbox-page-card-${entry.name}-profile`}
        >
          <ProfileLine label="sync_mode" value={profile.sync_mode ?? "—"} />
          <ProfileLine label="persistence" value={profile.persistence ?? "—"} />
          <ProfileLine label="runtime_root" value={profile.runtime_root ?? "—"} />
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
        className="text-right font-mono text-xs text-[color:var(--color-text-secondary)]"
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
            <Pencil className="size-3.5" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            onClick={() => onDelete(entry)}
            disabled={!deletable}
            aria-label={`Delete ${entry.name}`}
            title={
              deletable ? undefined : "Builtin sandboxes cannot be deleted — override them instead."
            }
            data-testid={`sandbox-page-card-${entry.name}-delete`}
          >
            <Trash2 className="size-3.5" />
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}

function ProfileLine({ label, value }: { label: string; value: string }) {
  return (
    <span className="flex items-center gap-2 whitespace-nowrap">
      <span className="font-mono text-[0.58rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
        {label}
      </span>
      <span className="font-mono text-[color:var(--color-text-primary)]">{value}</span>
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
      title={title}
      slug="sandbox"
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
              <span
                className="text-xs text-[color:var(--color-text-tertiary)]"
                data-testid="sandbox-editor-usage"
              >
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
      className="rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-3 py-2 text-xs text-[color:var(--color-text-tertiary)]"
      data-testid="sandbox-editor-preserved"
    >
      <span className="font-mono text-[0.62rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
        preserved on save
      </span>
      <span className="ml-2">
        {preserved.join(", ")} — edited outside this dialog and included as-is in the PUT replace.
      </span>
    </p>
  );
}

function SandboxDeleteDialog({
  target,
  error,
  isDeleting,
  onClose,
  onConfirm,
}: {
  target: SettingsSandboxEntry | null;
  error: string | null;
  isDeleting: boolean;
  onClose: () => void;
  onConfirm: () => void;
}) {
  const open = Boolean(target);
  const usage = target?.workspace_usage_count ?? 0;
  const hasUsage = usage > 0;

  return (
    <SettingsDeleteDialog
      open={open}
      slug="sandboxes"
      title={target ? `Delete sandbox profile "${target.name}"?` : "Delete sandbox profile"}
      description={
        target
          ? "Removing the overlay stops making this profile selectable for new workspaces."
          : null
      }
      fallbackNote={
        hasUsage ? (
          <div className="flex flex-col gap-1" data-testid="sandbox-delete-usage">
            <span className="font-medium">
              {usage} {usage === 1 ? "workspace" : "workspaces"} currently reference this profile
            </span>
            <span>
              Existing sessions continue to run against their recorded profile. New sessions will
              fail to resolve this sandbox until another profile with the same name is added.
            </span>
          </div>
        ) : null
      }
      error={error}
      isDeleting={isDeleting}
      confirmLabel="Delete sandbox profile"
      onConfirm={onConfirm}
      onOpenChange={next => {
        if (!next) onClose();
      }}
    />
  );
}

function ActionResultBanner({
  action,
  onDismiss,
}: {
  action: SandboxLastAction;
  onDismiss: () => void;
}) {
  const isSaved = action.kind === "saved";
  const restartBadge = action.result.restart_required
    ? "restart required to apply"
    : "applied immediately";
  const message = isSaved
    ? `Saved sandbox "${action.name}" · ${restartBadge}.`
    : action.usageCount > 0
      ? `Deleted "${action.name}" · ${action.usageCount} workspaces affected · ${restartBadge}.`
      : `Deleted "${action.name}" · ${restartBadge}.`;

  return (
    <Alert
      variant={isSaved ? "success" : "info"}
      role="status"
      data-testid="sandbox-page-action-result"
      data-kind={action.kind}
    >
      <Check className="size-3.5" />
      <AlertDescription className="text-xs">{message}</AlertDescription>
      <AlertAction>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={onDismiss}
          data-testid="sandbox-page-action-result-dismiss"
        >
          <X className="size-3.5" />
        </Button>
      </AlertAction>
    </Alert>
  );
}
