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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@agh/ui";
import {
  useSettingsEnvironmentsPage,
  type EnvironmentDraft,
  type EnvironmentEditorState,
  type EnvironmentLastAction,
} from "@/hooks/routes/use-settings-environments-page";
import type { SettingsEnvironmentEntry } from "@/systems/settings";
import {
  SettingsCollectionHeader,
  SettingsDeleteDialog,
  SettingsEditorDialog,
  SettingsFieldRow,
  SettingsPageActions,
  SettingsPageShell,
  SettingsRestartBanner,
  SettingsSourceBadge,
  SettingsStatusLine,
} from "@/systems/settings/components";

export const Route = createFileRoute("/_app/settings/environments")({
  component: EnvironmentsSettingsPage,
});

function EnvironmentsSettingsPage() {
  const page = useSettingsEnvironmentsPage();

  if (page.isLoading) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-environments-loading"
      >
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (page.error || !page.envelope) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-environments-error"
      >
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {page.error?.message ?? "Failed to load environments"}
          </p>
        </div>
      </div>
    );
  }

  return (
    <SettingsPageShell
      slug="environments"
      title="Environments"
      statusLine={
        <SettingsStatusLine
          data-testid="settings-page-environments-status-line"
          daemonAvailable
          items={[
            <span key="total" data-testid="settings-page-environments-total">
              {page.counts.total} profiles
            </span>,
            <span key="workspaces" data-testid="settings-page-environments-workspaces">
              {page.counts.totalWorkspaces} workspace references
            </span>,
          ]}
        />
      }
      actions={<SettingsPageActions slug="environments" restart={page.restart} />}
      banner={<SettingsRestartBanner slug="environments" restart={page.restart} />}
    >
      {page.lastAction ? (
        <ActionResultBanner action={page.lastAction} onDismiss={page.dismissLastAction} />
      ) : null}

      <SettingsCollectionHeader
        data-testid="settings-page-environments-header-row"
        eyebrow="Profiles"
        summary={`${page.counts.total} defined · used across ${page.counts.totalWorkspaces} workspaces`}
        action={
          <Button
            type="button"
            variant="default"
            size="sm"
            onClick={page.openCreate}
            data-testid="settings-page-environments-create"
          >
            <Plus className="size-3.5" />
            New environment
          </Button>
        }
      />

      {page.environments.length === 0 ? (
        <Empty
          icon={Boxes}
          title="No environments defined"
          description='Use "New environment" to create an overlay profile referenceable by workspaces.'
          data-testid="settings-page-environments-empty"
        />
      ) : (
        <EnvironmentsTable
          environments={page.environments}
          onEdit={page.openEdit}
          onDelete={page.openDelete}
        />
      )}

      <EnvironmentEditor
        editor={page.editor}
        isValid={page.editorIsValid}
        isSaving={page.editorIsSaving}
        error={page.editorError}
        warnings={page.editorWarnings}
        existingNames={page.environments.map(entry => entry.name)}
        onChange={page.updateDraft}
        onClose={page.closeEditor}
        onSave={page.saveEditor}
      />

      <EnvironmentDeleteDialog
        target={page.deleteTarget.mode === "open" ? page.deleteTarget.entry : null}
        error={page.deleteError}
        isDeleting={page.deleteIsPending}
        onClose={page.closeDelete}
        onConfirm={page.confirmDelete}
      />
    </SettingsPageShell>
  );
}

function EnvironmentsTable({
  environments,
  onEdit,
  onDelete,
}: {
  environments: SettingsEnvironmentEntry[];
  onEdit: (entry: SettingsEnvironmentEntry) => void;
  onDelete: (entry: SettingsEnvironmentEntry) => void;
}) {
  return (
    <div
      className="overflow-hidden rounded-lg border border-[color:var(--color-divider)]"
      data-testid="settings-page-environments-list"
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
          {environments.map(entry => (
            <EnvironmentRow key={entry.name} entry={entry} onEdit={onEdit} onDelete={onDelete} />
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function EnvironmentRow({
  entry,
  onEdit,
  onDelete,
}: {
  entry: SettingsEnvironmentEntry;
  onEdit: (entry: SettingsEnvironmentEntry) => void;
  onDelete: (entry: SettingsEnvironmentEntry) => void;
}) {
  const profile = entry.profile;
  const source = entry.source_metadata.effective_source;
  const shadowed = entry.source_metadata.shadowed_sources ?? [];
  const deletable = source.kind !== "builtin-provider";

  return (
    <TableRow data-testid={`settings-page-environments-card-${entry.name}`}>
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
          data-testid={`settings-page-environments-card-${entry.name}-profile`}
        >
          <ProfileLine label="sync_mode" value={profile.sync_mode ?? "—"} />
          <ProfileLine label="persistence" value={profile.persistence ?? "—"} />
          <ProfileLine label="runtime_root" value={profile.runtime_root ?? "—"} />
        </div>
      </TableCell>
      <TableCell>
        <SettingsSourceBadge
          data-testid={`settings-page-environments-card-${entry.name}-source`}
          source={source}
          shadowed={shadowed}
        />
      </TableCell>
      <TableCell
        className="text-right font-mono text-xs text-[color:var(--color-text-secondary)]"
        data-testid={`settings-page-environments-card-${entry.name}-usage`}
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
            data-testid={`settings-page-environments-card-${entry.name}-edit`}
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
              deletable
                ? undefined
                : "Builtin environments cannot be deleted — override them instead."
            }
            data-testid={`settings-page-environments-card-${entry.name}-delete`}
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
    e2b: "firecracker microVM · E2B",
  };
  return map[backend] ?? `custom backend · ${backend}`;
}

function backendTone(backend: string): "success" | "info" | "accent" | "neutral" {
  if (backend === "local") return "success";
  if (backend === "daytona") return "info";
  if (backend === "e2b") return "accent";
  return "neutral";
}

interface EnvironmentEditorProps {
  editor: EnvironmentEditorState;
  isValid: boolean;
  isSaving: boolean;
  error: string | null;
  warnings?: string[];
  existingNames: string[];
  onChange: (updater: (draft: EnvironmentDraft) => EnvironmentDraft) => void;
  onClose: () => void;
  onSave: () => void;
}

function EnvironmentEditor({
  editor,
  isValid,
  isSaving,
  error,
  warnings,
  existingNames,
  onChange,
  onClose,
  onSave,
}: EnvironmentEditorProps) {
  const open = editor.mode !== "closed";
  if (!open) return null;

  const isCreate = editor.mode === "create";
  const draft = editor.draft;
  const entry = editor.mode === "edit" ? editor.entry : null;

  const title = isCreate
    ? "New environment"
    : `Edit environment · ${editor.mode === "edit" ? editor.name : ""}`;
  const description = isCreate
    ? "Create a new environment overlay. Saving writes a new overlay entry."
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
      slug="environments"
      description={description}
      metadata={
        entry ? (
          <div className="flex flex-col gap-1">
            <SettingsSourceBadge
              data-testid="settings-environments-editor-source"
              source={entry.source_metadata.effective_source}
              shadowed={entry.source_metadata.shadowed_sources ?? []}
            />
            {entry.workspace_usage_count > 0 ? (
              <span
                className="text-xs text-[color:var(--color-text-tertiary)]"
                data-testid="settings-environments-editor-usage"
              >
                {entry.workspace_usage_count} workspaces depend on this profile
              </span>
            ) : null}
          </div>
        ) : null
      }
      error={
        error ?? (nameConflict ? `An environment named "${draft.name}" already exists.` : null)
      }
      warnings={warnings}
      canSave={isValid && !nameConflict}
      isSaving={isSaving}
      saveLabel={isCreate ? "Create environment" : "Replace profile"}
      onSave={onSave}
      onOpenChange={next => {
        if (!next) onClose();
      }}
    >
      <div className="flex flex-col gap-3">
        <SettingsFieldRow
          data-testid="settings-environments-editor-name"
          label="Name"
          description={
            isCreate
              ? "Lower-case identifier referenced by workspaces."
              : "Name is immutable — create a new environment to rename."
          }
          hint={isCreate ? "REQUIRED" : "LOCKED"}
          control={
            <Input
              className="w-56 font-mono disabled:opacity-60"
              data-testid="settings-environments-editor-name-input"
              value={draft.name}
              placeholder="e.g. local"
              disabled={!isCreate}
              onChange={event => onChange(current => ({ ...current, name: event.target.value }))}
            />
          }
        />
        <SettingsFieldRow
          data-testid="settings-environments-editor-backend"
          label="Backend"
          description="Which execution backend the environment uses."
          hint="REQUIRED"
          control={
            <NativeSelect
              className="w-56 font-mono"
              data-testid="settings-environments-editor-backend-input"
              value={draft.backend}
              onChange={event => onChange(current => ({ ...current, backend: event.target.value }))}
            >
              <NativeSelectOption value="local">local</NativeSelectOption>
              <NativeSelectOption value="daytona">daytona</NativeSelectOption>
              <NativeSelectOption value="e2b">e2b</NativeSelectOption>
            </NativeSelect>
          }
        />
        <SettingsFieldRow
          data-testid="settings-environments-editor-sync-mode"
          label="Sync mode"
          description="How files move between host and sandbox."
          hint="OPTIONAL"
          control={
            <Input
              className="w-56 font-mono"
              data-testid="settings-environments-editor-sync-mode-input"
              value={draft.sync_mode}
              placeholder="none | session-bidir | turn-bidir"
              onChange={event =>
                onChange(current => ({ ...current, sync_mode: event.target.value }))
              }
            />
          }
        />
        <SettingsFieldRow
          data-testid="settings-environments-editor-persistence"
          label="Persistence"
          description="Workspace lifecycle between sessions."
          hint="OPTIONAL"
          control={
            <Input
              className="w-56 font-mono"
              data-testid="settings-environments-editor-persistence-input"
              value={draft.persistence}
              placeholder="transient | reuse | archive"
              onChange={event =>
                onChange(current => ({ ...current, persistence: event.target.value }))
              }
            />
          }
        />
        <SettingsFieldRow
          data-testid="settings-environments-editor-runtime-root"
          label="Runtime root"
          description="Directory mounted as the working root."
          hint="OPTIONAL"
          control={
            <Input
              className="w-72 font-mono"
              data-testid="settings-environments-editor-runtime-root-input"
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
      data-testid="settings-environments-editor-preserved"
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

function EnvironmentDeleteDialog({
  target,
  error,
  isDeleting,
  onClose,
  onConfirm,
}: {
  target: SettingsEnvironmentEntry | null;
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
      slug="environments"
      title={target ? `Delete environment "${target.name}"?` : "Delete environment"}
      description={
        target
          ? "Removing the overlay stops making this profile selectable for new workspaces."
          : null
      }
      fallbackNote={
        hasUsage ? (
          <div className="flex flex-col gap-1" data-testid="settings-environments-delete-usage">
            <span className="font-medium">
              {usage} {usage === 1 ? "workspace" : "workspaces"} currently reference this profile
            </span>
            <span>
              Existing sessions continue to run against their recorded profile. New sessions will
              fail to resolve this environment until another profile with the same name is added.
            </span>
          </div>
        ) : null
      }
      error={error}
      isDeleting={isDeleting}
      confirmLabel="Delete profile"
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
  action: EnvironmentLastAction;
  onDismiss: () => void;
}) {
  const isSaved = action.kind === "saved";
  const restartBadge = action.result.restart_required
    ? "restart required to apply"
    : "applied immediately";
  const message = isSaved
    ? `Saved environment "${action.name}" · ${restartBadge}.`
    : action.usageCount > 0
      ? `Deleted "${action.name}" · ${action.usageCount} workspaces affected · ${restartBadge}.`
      : `Deleted "${action.name}" · ${restartBadge}.`;

  return (
    <Alert
      variant={isSaved ? "success" : "info"}
      role="status"
      data-testid="settings-page-environments-action-result"
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
          data-testid="settings-page-environments-action-result-dismiss"
        >
          <X className="size-3.5" />
        </Button>
      </AlertAction>
    </Alert>
  );
}
