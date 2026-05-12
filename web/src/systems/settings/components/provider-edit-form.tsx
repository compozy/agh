import { KeyRound, Plus, X } from "lucide-react";

import { Button, Input, NativeSelect, NativeSelectOption, Textarea } from "@agh/ui";

import { SettingsFieldRow } from "./settings-field-row";
import type { ProviderDraft } from "../types";

type CredentialSlotDraft = ProviderDraft["credential_slots"][number];

interface ProviderEditFormProps {
  mode: "create" | "edit";
  draft: ProviderDraft;
  onChange: (updater: (draft: ProviderDraft) => ProviderDraft) => void;
}

export function ProviderEditForm({ mode, draft, onChange }: ProviderEditFormProps) {
  const isCreate = mode === "create";

  return (
    <div className="flex flex-col gap-3">
      <SettingsFieldRow
        variant="modal"
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
        variant="modal"
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
        variant="modal"
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
        variant="modal"
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
        variant="modal"
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
        variant="modal"
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
        variant="modal"
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
        variant="modal"
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
            onChange={event => onChange(current => ({ ...current, transport: event.target.value }))}
          />
        }
      />
      <SettingsFieldRow
        variant="modal"
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
            onChange={event => onChange(current => ({ ...current, base_url: event.target.value }))}
          />
        }
      />
      <SettingsFieldRow
        variant="modal"
        data-testid="settings-providers-editor-auth-mode"
        label="Auth mode"
        description="Owner of provider authentication at launch."
        hint="REQUIRED"
        control={
          <NativeSelect
            className="w-44 font-mono"
            data-testid="settings-providers-editor-auth-mode-input"
            value={draft.auth_mode}
            onChange={event => {
              const authMode = event.target.value;
              onChange(current => ({
                ...current,
                auth_mode: authMode,
                ...(authMode === "bound_secret"
                  ? {}
                  : {
                      target_env: "",
                      secret_ref: "",
                      secret_value: "",
                      credential_slots: [],
                      credential_secret_values: [],
                    }),
              }));
            }}
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
        variant="modal"
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
        variant="modal"
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
        variant="modal"
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
        variant="modal"
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
        variant="modal"
        data-testid="settings-providers-editor-api-key"
        label="Target env"
        description="Environment variable injected from the provider credential slot."
        hint="OPTIONAL"
        control={
          <div className="flex items-center gap-2">
            <KeyRound aria-hidden="true" className="size-3 text-subtle" />
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
        variant="modal"
        data-testid="settings-providers-editor-secret-ref"
        label="Secret ref"
        description="Bound credential source injected into the target env var at launch."
        hint="BOUND"
        control={
          <div className="flex items-center gap-2">
            <KeyRound aria-hidden="true" className="size-3 text-subtle" />
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
        variant="modal"
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
  );
}

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
      variant="modal"
      data-testid="settings-providers-editor-credential-slots"
      label="More slots"
      description="Additional credential refs injected into provider subprocess env."
      hint="OPTIONAL"
      control={
        <div className="flex w-full max-w-176 flex-col gap-2">
          {additionalSlots.length === 0 ? (
            <span
              className="font-mono text-xs text-subtle"
              data-testid="settings-providers-editor-credential-slots-empty"
            >
              No additional credential slots
            </span>
          ) : (
            additionalSlots.map((slot, offset) => {
              const index = offset + 1;
              return (
                <div
                  className="grid gap-2 rounded-md border border-line p-2 md:grid-cols-[8rem_11rem_1fr_7rem_2rem]"
                  data-testid={`settings-providers-editor-credential-slot-${index}`}
                  key={`credential-slot-${index}`}
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
                    <X aria-hidden="true" className="size-3" />
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
            <Plus aria-hidden="true" className="size-3" />
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
