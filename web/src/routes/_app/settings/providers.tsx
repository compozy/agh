import { AlertCircle, Check, Database, KeyRound, Loader2, Plus, Trash2, X } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import {
  Alert,
  AlertAction,
  AlertDescription,
  Button,
  ConfirmDialog,
  Empty,
  Input,
  NativeSelect,
  NativeSelectOption,
  PageShell,
  Section,
  Textarea,
} from "@agh/ui";

import {
  useSettingsProvidersPage,
  type ProviderDraft,
  type ProviderEditorState,
  type ProviderLastAction,
} from "@/hooks/routes/use-settings-providers-page";
import type { SettingsProviderEntry } from "@/systems/settings";
import {
  ProvidersGrid,
  SettingsEditorDialog,
  SettingsFieldRow,
  SettingsPageActions,
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
        <Loader2 className="size-5 animate-spin text-(--color-text-tertiary)" />
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
          <AlertCircle className="size-6 text-(--color-danger)" />
          <p className="text-sm text-(--color-text-tertiary)">
            {page.error?.message ?? "Failed to load providers"}
          </p>
        </div>
      </div>
    );
  }

  return (
    <PageShell
      slug="providers"
      title="Providers"
      statusLine={
        <SettingsStatusLine
          data-testid="settings-page-providers-status-line"
          status="connected"
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

      <Section
        data-testid="settings-page-providers-header-row"
        label="Catalog"
        note={
          <>{page.counts.total} providers shipped with the daemon or defined in config overlays</>
        }
        right={
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
        <Empty
          icon={Database}
          title="No providers configured"
          description='Use "New provider" to add an overlay entry to your config.'
          data-testid="settings-page-providers-empty"
        />
      ) : (
        <ProvidersGrid
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
    </PageShell>
  );
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

function ProviderEditor(props: ProviderEditorProps) {
  return renderProviderEditor(props);
}

function renderProviderEditor({
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
  const secretError =
    draft.secret_value.trim() && !draft.secret_ref.trim().startsWith("vault:")
      ? "API key values can only be saved into vault: refs."
      : null;

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
      error={
        error ??
        secretError ??
        (nameConflict ? `A provider named "${draft.name}" already exists.` : null)
      }
      warnings={warnings}
      canSave={isValid && !nameConflict && !secretError}
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
              : "Name is immutable -- create a new provider to rename."
          }
          hint={isCreate ? "REQUIRED" : "LOCKED"}
          control={
            <Input
              className="w-56 font-mono disabled:opacity-60"
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
            <Input
              className="w-72 font-mono"
              data-testid="settings-providers-editor-command-input"
              value={draft.command}
              placeholder="npx @agentclientprotocol/claude-agent-acp@latest"
              onChange={event => onChange(current => ({ ...current, command: event.target.value }))}
            />
          }
        />
        <SettingsFieldRow
          data-testid="settings-providers-editor-display-name"
          label="Display name"
          description="Operator-facing label shown beside the provider id."
          hint="OPTIONAL"
          control={
            <Input
              className="w-56"
              data-testid="settings-providers-editor-display-name-input"
              value={draft.display_name}
              placeholder="OpenRouter"
              onChange={event =>
                onChange(current => ({ ...current, display_name: event.target.value }))
              }
            />
          }
        />
        <SettingsFieldRow
          data-testid="settings-providers-editor-model"
          label="Default model"
          description="Sent to the provider when an agent does not specify one."
          hint="OPTIONAL"
          control={
            <Input
              className="w-56 font-mono"
              data-testid="settings-providers-editor-model-input"
              value={draft.model_default}
              placeholder="gpt-5-turbo"
              onChange={event =>
                onChange(current => ({ ...current, model_default: event.target.value }))
              }
            />
          }
        />
        <SettingsFieldRow
          data-testid="settings-providers-editor-curated-models"
          label="Curated models"
          description="Provider-scoped model IDs stored under models.curated."
          hint="OPTIONAL"
          control={
            <Textarea
              className="min-h-24 w-72 font-mono text-xs"
              data-testid="settings-providers-editor-curated-models-input"
              value={draft.curated_models}
              placeholder={"gpt-5.4\ngpt-5.4-mini"}
              onChange={event =>
                onChange(current => ({ ...current, curated_models: event.target.value }))
              }
            />
          }
        />
        <SettingsFieldRow
          data-testid="settings-providers-editor-harness"
          label="Harness"
          description="Runtime adapter used to launch the provider."
          hint="REQUIRED"
          control={
            <Input
              className="w-40 font-mono"
              data-testid="settings-providers-editor-harness-input"
              value={draft.harness}
              placeholder="acp or pi_acp"
              onChange={event => onChange(current => ({ ...current, harness: event.target.value }))}
            />
          }
        />
        <SettingsFieldRow
          data-testid="settings-providers-editor-runtime-provider"
          label="Runtime provider"
          description="Downstream provider id used by the selected harness."
          hint="PI"
          control={
            <Input
              className="w-56 font-mono"
              data-testid="settings-providers-editor-runtime-provider-input"
              value={draft.runtime_provider}
              placeholder="openrouter"
              onChange={event =>
                onChange(current => ({ ...current, runtime_provider: event.target.value }))
              }
            />
          }
        />
        <SettingsFieldRow
          data-testid="settings-providers-editor-transport"
          label="Transport"
          description="Provider API family or Pi models override transport."
          hint="OPTIONAL"
          control={
            <Input
              className="w-56 font-mono"
              data-testid="settings-providers-editor-transport-input"
              value={draft.transport}
              placeholder="openai"
              onChange={event =>
                onChange(current => ({ ...current, transport: event.target.value }))
              }
            />
          }
        />
        <SettingsFieldRow
          data-testid="settings-providers-editor-base-url"
          label="Base URL"
          description="Custom API base URL for Pi-backed model overrides."
          hint="OPTIONAL"
          control={
            <Input
              className="w-72 font-mono"
              data-testid="settings-providers-editor-base-url-input"
              value={draft.base_url}
              placeholder="https://openrouter.ai/api/v1"
              onChange={event =>
                onChange(current => ({ ...current, base_url: event.target.value }))
              }
            />
          }
        />
        <SettingsFieldRow
          data-testid="settings-providers-editor-auth-mode"
          label="Auth mode"
          description="Owner of provider authentication at launch."
          hint="REQUIRED"
          control={
            <NativeSelect
              className="w-44 font-mono"
              data-testid="settings-providers-editor-auth-mode-input"
              value={draft.auth_mode}
              onChange={event =>
                onChange(current => ({
                  ...current,
                  auth_mode: event.target.value,
                  ...(event.target.value === "bound_secret"
                    ? {}
                    : { target_env: "", secret_ref: "", secret_value: "", credential_slots: [] }),
                }))
              }
            >
              {["native_cli", "bound_secret", "none"].map(option => (
                <NativeSelectOption key={option} value={option}>
                  {option}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          }
        />
        <SettingsFieldRow
          data-testid="settings-providers-editor-env-policy"
          label="Env policy"
          description="Daemon environment inheritance policy for provider subprocesses."
          hint="REQUIRED"
          control={
            <NativeSelect
              className="w-40 font-mono"
              data-testid="settings-providers-editor-env-policy-input"
              value={draft.env_policy}
              onChange={event =>
                onChange(current => ({ ...current, env_policy: event.target.value }))
              }
            >
              {["filtered", "isolated"].map(option => (
                <NativeSelectOption key={option} value={option}>
                  {option}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          }
        />
        <SettingsFieldRow
          data-testid="settings-providers-editor-home-policy"
          label="Home policy"
          description="Provider CLI state location policy."
          hint="REQUIRED"
          control={
            <NativeSelect
              className="w-40 font-mono"
              data-testid="settings-providers-editor-home-policy-input"
              value={draft.home_policy}
              onChange={event =>
                onChange(current => ({ ...current, home_policy: event.target.value }))
              }
            >
              {["operator", "isolated"].map(option => (
                <NativeSelectOption key={option} value={option}>
                  {option}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          }
        />
        <SettingsFieldRow
          data-testid="settings-providers-editor-auth-status-command"
          label="Status command"
          description="Provider-owned command used for auth diagnostics."
          hint="OPTIONAL"
          control={
            <Input
              className="w-72 font-mono"
              data-testid="settings-providers-editor-auth-status-command-input"
              value={draft.auth_status_command}
              placeholder="codex auth status"
              onChange={event =>
                onChange(current => ({ ...current, auth_status_command: event.target.value }))
              }
            />
          }
        />
        <SettingsFieldRow
          data-testid="settings-providers-editor-auth-login-command"
          label="Login command"
          description="Provider-owned command opened by provider auth login."
          hint="OPTIONAL"
          control={
            <Input
              className="w-72 font-mono"
              data-testid="settings-providers-editor-auth-login-command-input"
              value={draft.auth_login_command}
              placeholder="codex login"
              onChange={event =>
                onChange(current => ({ ...current, auth_login_command: event.target.value }))
              }
            />
          }
        />
        <SettingsFieldRow
          data-testid="settings-providers-editor-api-key"
          label="Target env"
          description="Environment variable injected from the provider credential slot."
          hint="OPTIONAL"
          control={
            <div className="flex items-center gap-2">
              <KeyRound className="size-3.5 text-(--color-text-tertiary)" />
              <Input
                className="w-56 font-mono"
                data-testid="settings-providers-editor-api-key-input"
                value={draft.target_env}
                placeholder="ANTHROPIC_API_KEY"
                disabled={draft.auth_mode !== "bound_secret"}
                onChange={event =>
                  onChange(current => ({ ...current, target_env: event.target.value }))
                }
              />
            </div>
          }
        />
        <SettingsFieldRow
          data-testid="settings-providers-editor-secret-ref"
          label="Secret ref"
          description="Bound credential source injected into the target env var at launch."
          hint="BOUND"
          control={
            <div className="flex items-center gap-2">
              <KeyRound className="size-3.5 text-(--color-text-tertiary)" />
              <Input
                className="w-72 font-mono"
                data-testid="settings-providers-editor-secret-ref-input"
                value={draft.secret_ref}
                placeholder="env:OPENROUTER_API_KEY"
                disabled={draft.auth_mode !== "bound_secret"}
                onChange={event =>
                  onChange(current => ({ ...current, secret_ref: event.target.value }))
                }
              />
            </div>
          }
        />
        <SettingsFieldRow
          data-testid="settings-providers-editor-secret-value"
          label="API key"
          description="Write-only value stored when the secret ref uses vault:."
          hint="WRITE-ONLY"
          control={
            <Input
              className="w-72 font-mono"
              data-testid="settings-providers-editor-secret-value-input"
              value={draft.secret_value}
              type="password"
              placeholder="sk-..."
              disabled={draft.auth_mode !== "bound_secret"}
              onChange={event =>
                onChange(current => ({ ...current, secret_value: event.target.value }))
              }
            />
          }
        />
        <AdditionalCredentialSlotsEditor draft={draft} onChange={onChange} />
      </div>
    </SettingsEditorDialog>
  );
}

type CredentialSlotDraft = ProviderDraft["credential_slots"][number];

function AdditionalCredentialSlotsEditor({
  draft,
  onChange,
}: {
  draft: ProviderDraft;
  onChange: (updater: (draft: ProviderDraft) => ProviderDraft) => void;
}) {
  const additionalSlots = draft.credential_slots.slice(1);
  const disabled = draft.auth_mode !== "bound_secret";

  return (
    <SettingsFieldRow
      data-testid="settings-providers-editor-credential-slots"
      label="More slots"
      description="Additional credential refs injected into provider subprocess env."
      hint="OPTIONAL"
      control={
        <div className="flex w-full max-w-176 flex-col gap-2">
          {additionalSlots.length === 0 ? (
            <span
              className="font-mono text-xs text-(--color-text-tertiary)"
              data-testid="settings-providers-editor-credential-slots-empty"
            >
              No additional credential slots
            </span>
          ) : (
            additionalSlots.map((slot, offset) => {
              const index = offset + 1;
              return (
                <div
                  className="grid gap-2 rounded-md border border-(--color-divider) p-2 md:grid-cols-[8rem_11rem_1fr_7rem_2rem]"
                  data-testid={`settings-providers-editor-credential-slot-${index}`}
                  key={slot.name || slot.target_env || slot.secret_ref || slot.kind}
                >
                  <Input
                    className="font-mono"
                    aria-label={`Credential slot ${index} name`}
                    value={slot.name}
                    placeholder="organization"
                    disabled={disabled}
                    onChange={event =>
                      onChange(current =>
                        updateCredentialSlot(current, index, { name: event.target.value })
                      )
                    }
                  />
                  <Input
                    className="font-mono"
                    aria-label={`Credential slot ${index} target env`}
                    value={slot.target_env}
                    placeholder="OPENROUTER_ORG_ID"
                    disabled={disabled}
                    onChange={event =>
                      onChange(current =>
                        updateCredentialSlot(current, index, { target_env: event.target.value })
                      )
                    }
                  />
                  <Input
                    className="font-mono"
                    aria-label={`Credential slot ${index} secret ref`}
                    value={slot.secret_ref}
                    placeholder="env:OPENROUTER_ORG_ID"
                    disabled={disabled}
                    onChange={event =>
                      onChange(current =>
                        updateCredentialSlot(current, index, { secret_ref: event.target.value })
                      )
                    }
                  />
                  <Input
                    className="font-mono"
                    aria-label={`Credential slot ${index} vault value`}
                    type="password"
                    value={draft.credential_secret_values[index] ?? ""}
                    placeholder="value"
                    disabled={disabled}
                    onChange={event =>
                      onChange(current =>
                        updateCredentialSecretValue(current, index, event.target.value)
                      )
                    }
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    aria-label={`Remove credential slot ${index}`}
                    disabled={disabled}
                    onClick={() => onChange(current => removeCredentialSlot(current, index))}
                  >
                    <X className="size-3.5" />
                  </Button>
                </div>
              );
            })
          )}
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="w-fit"
            disabled={disabled}
            onClick={() => onChange(addCredentialSlot)}
            data-testid="settings-providers-editor-add-credential-slot"
          >
            <Plus className="size-3.5" />
            Add slot
          </Button>
        </div>
      }
    />
  );
}

function addCredentialSlot(draft: ProviderDraft): ProviderDraft {
  const slots =
    draft.credential_slots.length > 0 ? [...draft.credential_slots] : [primarySlot(draft)];
  const values = [...draft.credential_secret_values];
  slots.push({
    name: `credential_${slots.length + 1}`,
    target_env: "",
    secret_ref: "",
    kind: "api_key",
    required: false,
  });
  values.length = slots.length;
  values[slots.length - 1] = "";
  return { ...draft, credential_slots: slots, credential_secret_values: values };
}

function primarySlot(draft: ProviderDraft): CredentialSlotDraft {
  const targetEnv = draft.target_env.trim();
  return {
    name: "api_key",
    target_env: targetEnv,
    secret_ref: draft.secret_ref.trim() || (targetEnv ? `env:${targetEnv}` : ""),
    kind: "api_key",
    required: true,
  };
}

function updateCredentialSlot(
  draft: ProviderDraft,
  index: number,
  patch: Partial<CredentialSlotDraft>
): ProviderDraft {
  const slots = [...draft.credential_slots];
  const current = slots[index];
  if (!current) {
    return draft;
  }
  slots[index] = { ...current, ...patch };
  return { ...draft, credential_slots: slots };
}

function updateCredentialSecretValue(
  draft: ProviderDraft,
  index: number,
  value: string
): ProviderDraft {
  const values = [...draft.credential_secret_values];
  values[index] = value;
  return { ...draft, credential_secret_values: values };
}

function removeCredentialSlot(draft: ProviderDraft, index: number): ProviderDraft {
  const slots = draft.credential_slots.filter((_, currentIndex) => currentIndex !== index);
  const values = draft.credential_secret_values.filter((_, currentIndex) => currentIndex !== index);
  return { ...draft, credential_slots: slots, credential_secret_values: values };
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
    <ConfirmDialog
      open={open}
      title={target ? `Delete provider "${target.name}"?` : "Delete provider"}
      description={
        target
          ? "Removing the overlay keeps the provider config in other overlays or builtin definitions, if any."
          : null
      }
      note={
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
      isPending={isDeleting}
      cancelLabel="Cancel"
      confirmLabel="Delete overlay"
      confirmIcon={Trash2}
      contentProps={{ "data-testid": "settings-providers-delete" }}
      titleProps={{ "data-testid": "settings-providers-delete-title" }}
      noteProps={{ "data-testid": "settings-providers-delete-fallback" }}
      errorProps={{ "data-testid": "settings-providers-delete-error" }}
      cancelButtonProps={{
        "data-testid": "settings-providers-delete-cancel",
        disabled: isDeleting,
      }}
      confirmButtonProps={{
        "data-testid": "settings-providers-delete-confirm",
      }}
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
  const restartBadge = action.result.restart_required
    ? "restart required to apply"
    : "applied immediately";

  const message = isSaved
    ? `Saved provider "${action.name}" · ${restartBadge}.`
    : action.hadFallback
      ? `Deleted overlay for "${action.name}" · builtin fallback now effective · ${restartBadge}.`
      : `Deleted overlay for "${action.name}" · ${restartBadge}.`;

  return (
    <Alert
      variant={isSaved ? "success" : "info"}
      role="status"
      data-testid="settings-page-providers-action-result"
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
          data-testid="settings-page-providers-action-result-dismiss"
        >
          <X className="size-3.5" />
        </Button>
      </AlertAction>
    </Alert>
  );
}
