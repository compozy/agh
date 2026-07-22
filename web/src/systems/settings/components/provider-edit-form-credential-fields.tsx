import { KeyRound, Plus, X } from "lucide-react";

import { Button, Input } from "@agh/ui";

import type { ProviderDraft } from "../types";
import type { ProviderDraftChange } from "./provider-edit-form";
import { ModalSettingsFieldRow } from "./settings-field-row";

type CredentialSlotDraft = ProviderDraft["credential_slots"][number];

interface ProviderCredentialFieldsProps {
  draft: ProviderDraft;
  onChange: ProviderDraftChange;
}

export function ProviderCredentialFields({ draft, onChange }: ProviderCredentialFieldsProps) {
  const disabled = draft.auth_mode !== "bound_secret";

  return (
    <>
      <ModalSettingsFieldRow
        data-testid="settings-providers-editor-api-key"
        label="Target env"
        description="Environment variable injected from the provider credential slot."
        control={
          <div className="flex items-center gap-2">
            <KeyRound aria-hidden="true" className="size-3 text-subtle" />
            <Input
              className="w-56 font-mono"
              data-testid="settings-providers-editor-api-key-input"
              value={draft.target_env}
              placeholder="ANTHROPIC_API_KEY"
              disabled={disabled}
              onChange={event =>
                onChange(current => ({ ...current, target_env: event.target.value }))
              }
            />
          </div>
        }
      />
      <ModalSettingsFieldRow
        data-testid="settings-providers-editor-secret-ref"
        label="Secret ref"
        description="Bound credential source injected into the target env var at launch."
        control={
          <div className="flex items-center gap-2">
            <KeyRound aria-hidden="true" className="size-3 text-subtle" />
            <Input
              className="w-72 font-mono"
              data-testid="settings-providers-editor-secret-ref-input"
              value={draft.secret_ref}
              placeholder="env:OPENROUTER_API_KEY"
              disabled={disabled}
              onChange={event =>
                onChange(current => ({ ...current, secret_ref: event.target.value }))
              }
            />
          </div>
        }
      />
      <ModalSettingsFieldRow
        data-testid="settings-providers-editor-secret-value"
        label="API key"
        description="Write-only value stored when the secret ref uses vault:."
        control={
          <Input
            className="w-72 font-mono"
            data-testid="settings-providers-editor-secret-value-input"
            value={draft.secret_value}
            type="password"
            placeholder="sk-..."
            disabled={disabled}
            onChange={event =>
              onChange(current => ({ ...current, secret_value: event.target.value }))
            }
          />
        }
      />
      <AdditionalCredentialSlotsEditor draft={draft} onChange={onChange} />
    </>
  );
}

function AdditionalCredentialSlotsEditor({ draft, onChange }: ProviderCredentialFieldsProps) {
  const additionalSlots = draft.credential_slots.slice(1);
  const disabled = draft.auth_mode !== "bound_secret";

  return (
    <ModalSettingsFieldRow
      data-testid="settings-providers-editor-credential-slots"
      label="More slots"
      description="Additional credential refs injected into provider subprocess env."
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
