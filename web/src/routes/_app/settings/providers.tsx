import { AlertCircle, Check, KeyRound, Loader2, Plus, X } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import { Button, Pill } from "@agh/ui";

import {
  useSettingsProvidersPage,
  type ProviderDraft,
  type ProviderEditorState,
  type ProviderLastAction,
} from "@/hooks/routes/use-settings-providers-page";
import type { SettingsProviderEntry } from "@/systems/settings";
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

export const Route = createFileRoute("/_app/settings/providers")({
  component: ProvidersSettingsPage,
});

function ProvidersSettingsPage() {
  const page = useSettingsProvidersPage();

  if (page.isLoading) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-providers-loading"
      >
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (page.error || !page.envelope) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-providers-error"
      >
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {page.error?.message ?? "Failed to load providers"}
          </p>
        </div>
      </div>
    );
  }

  return (
    <SettingsPageShell
      slug="providers"
      title="Providers"
      statusLine={
        <SettingsStatusLine
          data-testid="settings-page-providers-status-line"
          daemonAvailable
          items={[
            <span key="total" data-testid="settings-page-providers-total">
              {page.counts.total} providers
            </span>,
            <span key="installed" data-testid="settings-page-providers-installed">
              {page.counts.installed} installed
            </span>,
            <span key="missing" data-testid="settings-page-providers-missing">
              {page.counts.binaryMissing} binary missing
            </span>,
            <span key="unconfigured" data-testid="settings-page-providers-unconfigured">
              {page.counts.unconfigured} unconfigured
            </span>,
          ]}
        />
      }
      actions={<SettingsPageActions slug="providers" restart={page.restart} />}
      banner={<SettingsRestartBanner slug="providers" restart={page.restart} />}
    >
      {page.lastAction ? (
        <ActionResultBanner action={page.lastAction} onDismiss={page.dismissLastAction} />
      ) : null}

      <SettingsCollectionHeader
        data-testid="settings-page-providers-header-row"
        eyebrow="Catalog"
        summary={
          <>{page.counts.total} providers shipped with the daemon or defined in config overlays</>
        }
        action={
          <Button
            type="button"
            variant="default"
            size="sm"
            onClick={page.openCreate}
            data-testid="settings-page-providers-create"
          >
            <Plus className="size-3.5" />
            New provider
          </Button>
        }
      />

      {page.providers.length === 0 ? (
        <div
          className="rounded-md border border-dashed border-[color:var(--color-divider)] px-4 py-8 text-center text-sm text-[color:var(--color-text-tertiary)]"
          data-testid="settings-page-providers-empty"
        >
          No providers are defined. Use &ldquo;New provider&rdquo; to add one to your config
          overlay.
        </div>
      ) : (
        <ProvidersTable
          providers={page.providers}
          onEdit={page.openEdit}
          onDelete={page.openDelete}
        />
      )}

      <ProviderEditor
        editor={page.editor}
        isValid={page.editorIsValid}
        isSaving={page.editorIsSaving}
        error={page.editorError}
        warnings={page.editorWarnings}
        existingNames={page.providers.map(provider => provider.name)}
        onChange={page.updateDraft}
        onClose={page.closeEditor}
        onSave={page.saveEditor}
      />

      <ProviderDeleteDialog
        target={page.deleteTarget.mode === "open" ? page.deleteTarget.entry : null}
        error={page.deleteError}
        isDeleting={page.deleteIsPending}
        onClose={page.closeDelete}
        onConfirm={page.confirmDelete}
      />
    </SettingsPageShell>
  );
}

function ProvidersTable({
  providers,
  onEdit,
  onDelete,
}: {
  providers: SettingsProviderEntry[];
  onEdit: (entry: SettingsProviderEntry) => void;
  onDelete: (entry: SettingsProviderEntry) => void;
}) {
  return (
    <div
      className="overflow-hidden rounded-lg border border-[color:var(--color-divider)]"
      data-testid="settings-page-providers-list"
    >
      <table className="w-full border-collapse text-sm">
        <thead className="bg-[color:var(--color-surface-elevated)] text-[0.6rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
          <tr>
            <th className="px-4 py-2.5 text-left">Provider</th>
            <th className="px-4 py-2.5 text-left">Command</th>
            <th className="px-4 py-2.5 text-left">Default model</th>
            <th className="px-4 py-2.5 text-left">API key env</th>
            <th className="px-4 py-2.5 text-left">Source</th>
            <th className="px-4 py-2.5 text-right">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-[color:var(--color-divider)]">
          {providers.map(provider => (
            <ProviderRow
              key={provider.name}
              provider={provider}
              onEdit={onEdit}
              onDelete={onDelete}
            />
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ProviderRow({
  provider,
  onEdit,
  onDelete,
}: {
  provider: SettingsProviderEntry;
  onEdit: (entry: SettingsProviderEntry) => void;
  onDelete: (entry: SettingsProviderEntry) => void;
}) {
  const source = provider.source_metadata.effective_source;
  const shadowed = provider.source_metadata.shadowed_sources ?? [];
  const deletable = source.kind !== "builtin-provider";
  const tone = providerStateTone(provider);

  return (
    <tr data-testid={`settings-page-providers-row-${provider.name}`}>
      <td className="px-4 py-3">
        <div className="flex items-center gap-2.5">
          <span
            className={`inline-flex size-2 shrink-0 rounded-full ${tone.dot}`}
            aria-hidden="true"
            data-testid={`settings-page-providers-row-${provider.name}-status`}
            data-tone={tone.label}
          />
          <span className="font-mono text-sm text-[color:var(--color-text-primary)]">
            {provider.name}
          </span>
          {provider.default ? <Pill variant="accent">DEFAULT</Pill> : null}
        </div>
      </td>
      <td
        className="px-4 py-3 font-mono text-xs text-[color:var(--color-text-secondary)]"
        data-testid={`settings-page-providers-row-${provider.name}-command`}
      >
        {provider.settings.command ?? <EmptyCell />}
      </td>
      <td
        className="px-4 py-3 font-mono text-xs text-[color:var(--color-text-secondary)]"
        data-testid={`settings-page-providers-row-${provider.name}-model`}
      >
        {provider.settings.default_model ?? <EmptyCell />}
      </td>
      <td className="px-4 py-3 font-mono text-xs">
        <div className="flex items-center gap-2">
          <span
            className="text-[color:var(--color-text-secondary)]"
            data-testid={`settings-page-providers-row-${provider.name}-api-key`}
          >
            {provider.settings.api_key_env ?? <EmptyCell />}
          </span>
          {provider.settings.api_key_env ? (
            <Pill
              variant={provider.api_key_env_present ? "success" : "accent"}
              data-testid={`settings-page-providers-row-${provider.name}-api-key-state`}
            >
              {provider.api_key_env_present ? "SET" : "MISSING"}
            </Pill>
          ) : null}
        </div>
      </td>
      <td className="px-4 py-3">
        <SettingsSourceBadge
          data-testid={`settings-page-providers-row-${provider.name}-source`}
          source={source}
          shadowed={shadowed}
        />
      </td>
      <td className="px-4 py-3">
        <div className="flex items-center justify-end gap-2">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onEdit(provider)}
            data-testid={`settings-page-providers-row-${provider.name}-edit`}
          >
            Edit
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onDelete(provider)}
            disabled={!deletable}
            data-testid={`settings-page-providers-row-${provider.name}-delete`}
            title={
              deletable
                ? undefined
                : "Builtin providers cannot be deleted — edit the overlay to override them."
            }
          >
            Delete
          </Button>
        </div>
      </td>
    </tr>
  );
}

function EmptyCell() {
  return <span className="text-[color:var(--color-text-label)]">—</span>;
}

function providerStateTone(provider: SettingsProviderEntry) {
  if (!provider.command_available) {
    return { dot: "bg-[color:var(--color-warning)]", label: "binary-missing" };
  }
  if (!provider.api_key_env_present && provider.settings.api_key_env) {
    return { dot: "bg-[color:var(--color-warning)]", label: "unconfigured" };
  }
  return { dot: "bg-[color:var(--color-text-tertiary)]", label: "installed" };
}

interface ProviderEditorProps {
  editor: ProviderEditorState;
  isValid: boolean;
  isSaving: boolean;
  error: string | null;
  warnings?: string[];
  existingNames: string[];
  onChange: (updater: (draft: ProviderDraft) => ProviderDraft) => void;
  onClose: () => void;
  onSave: () => void;
}

function ProviderEditor({
  editor,
  isValid,
  isSaving,
  error,
  warnings,
  existingNames,
  onChange,
  onClose,
  onSave,
}: ProviderEditorProps) {
  const open = editor.mode !== "closed";
  if (!open) {
    return null;
  }

  const isCreate = editor.mode === "create";
  const draft = editor.draft;
  const entry = editor.mode === "edit" ? editor.entry : null;

  const title = isCreate
    ? "New provider"
    : `Edit provider · ${editor.mode === "edit" ? editor.name : ""}`;
  const description = isCreate
    ? "Add a new provider overlay. Saved entries replace any prior overlay definition for this name."
    : "Saving replaces the entire overlay entry for this provider with the values below (full PUT).";

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
      slug="providers"
      description={description}
      metadata={
        entry ? (
          <SettingsSourceBadge
            data-testid="settings-providers-editor-source"
            source={entry.source_metadata.effective_source}
            shadowed={entry.source_metadata.shadowed_sources ?? []}
          />
        ) : null
      }
      error={error ?? (nameConflict ? `A provider named "${draft.name}" already exists.` : null)}
      warnings={warnings}
      canSave={isValid && !nameConflict}
      isSaving={isSaving}
      saveLabel={isCreate ? "Create provider" : "Replace overlay"}
      onSave={onSave}
      onOpenChange={next => {
        if (!next) onClose();
      }}
    >
      <div className="flex flex-col gap-3">
        <SettingsFieldRow
          data-testid="settings-providers-editor-name"
          label="Name"
          description={
            isCreate
              ? "Lower-case identifier used in agent frontmatter and CLI flags."
              : "Name is immutable — create a new provider to rename."
          }
          hint={isCreate ? "REQUIRED" : "LOCKED"}
          control={
            <input
              className="h-8 w-56 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 font-mono text-sm text-[color:var(--color-text-primary)] disabled:opacity-60"
              data-testid="settings-providers-editor-name-input"
              value={draft.name}
              placeholder="e.g. claude"
              disabled={!isCreate}
              onChange={event => onChange(current => ({ ...current, name: event.target.value }))}
            />
          }
        />
        <SettingsFieldRow
          data-testid="settings-providers-editor-command"
          label="Command"
          description="Executable used to launch the ACP subprocess."
          hint="OVERLAY"
          control={
            <input
              className="h-8 w-72 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 font-mono text-sm text-[color:var(--color-text-primary)]"
              data-testid="settings-providers-editor-command-input"
              value={draft.command}
              placeholder="npx @agentclientprotocol/claude-agent-acp@latest"
              onChange={event => onChange(current => ({ ...current, command: event.target.value }))}
            />
          }
        />
        <SettingsFieldRow
          data-testid="settings-providers-editor-model"
          label="Default model"
          description="Sent to the provider when an agent does not specify one."
          hint="OPTIONAL"
          control={
            <input
              className="h-8 w-56 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 font-mono text-sm text-[color:var(--color-text-primary)]"
              data-testid="settings-providers-editor-model-input"
              value={draft.default_model}
              placeholder="gpt-5-turbo"
              onChange={event =>
                onChange(current => ({ ...current, default_model: event.target.value }))
              }
            />
          }
        />
        <SettingsFieldRow
          data-testid="settings-providers-editor-api-key"
          label="API key env"
          description="Environment variable the daemon reads before spawning this provider."
          hint="OPTIONAL"
          control={
            <div className="flex items-center gap-2">
              <KeyRound className="size-3.5 text-[color:var(--color-text-tertiary)]" />
              <input
                className="h-8 w-56 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 font-mono text-sm text-[color:var(--color-text-primary)]"
                data-testid="settings-providers-editor-api-key-input"
                value={draft.api_key_env}
                placeholder="ANTHROPIC_API_KEY"
                onChange={event =>
                  onChange(current => ({ ...current, api_key_env: event.target.value }))
                }
              />
            </div>
          }
        />
      </div>
    </SettingsEditorDialog>
  );
}

function ProviderDeleteDialog({
  target,
  error,
  isDeleting,
  onClose,
  onConfirm,
}: {
  target: SettingsProviderEntry | null;
  error: string | null;
  isDeleting: boolean;
  onClose: () => void;
  onConfirm: () => void;
}) {
  const open = Boolean(target);
  const fallback = target?.fallback ?? null;

  return (
    <SettingsDeleteDialog
      open={open}
      slug="providers"
      title={target ? `Delete provider "${target.name}"?` : "Delete provider"}
      description={
        target
          ? "Removing the overlay keeps the provider config in other overlays or builtin definitions, if any."
          : null
      }
      fallbackNote={
        fallback ? (
          <div className="flex flex-col gap-1" data-testid="settings-providers-delete-builtin">
            <span className="font-medium">Builtin provider will be revealed</span>
            <span>
              After delete, the effective provider falls back to the builtin definition shipped with
              the daemon. The provider stays available with its shipped defaults.
            </span>
          </div>
        ) : null
      }
      error={error}
      isDeleting={isDeleting}
      confirmLabel="Delete overlay"
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
  action: ProviderLastAction;
  onDismiss: () => void;
}) {
  const isSaved = action.kind === "saved";
  const tone = isSaved ? "success" : "info";
  const toneClasses =
    tone === "success"
      ? "border-[color:var(--color-success)] bg-[color:var(--color-success-tint)] text-[color:var(--color-success)]"
      : "border-[color:var(--color-info)] bg-[color:var(--color-info-tint)] text-[color:var(--color-info)]";

  const restartBadge = action.result.restart_required
    ? "restart required to apply"
    : "applied immediately";

  const message = isSaved
    ? `Saved provider "${action.name}" · ${restartBadge}.`
    : action.hadFallback
      ? `Deleted overlay for "${action.name}" · builtin fallback now effective · ${restartBadge}.`
      : `Deleted overlay for "${action.name}" · ${restartBadge}.`;

  return (
    <div
      className={`flex items-center justify-between gap-3 rounded-md border px-3 py-2 text-xs ${toneClasses}`}
      data-testid="settings-page-providers-action-result"
      data-kind={action.kind}
      role="status"
    >
      <span className="flex items-center gap-2">
        <Check className="size-3.5" />
        <span>{message}</span>
      </span>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={onDismiss}
        data-testid="settings-page-providers-action-result-dismiss"
      >
        <X className="size-3.5" />
      </Button>
    </div>
  );
}
