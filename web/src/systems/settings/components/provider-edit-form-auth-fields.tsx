import { Input, NativeSelect, NativeSelectOption } from "@agh/ui";

import type { ProviderDraft } from "../types";
import type { ProviderDraftChange } from "./provider-edit-form";
import { ProviderCredentialFields } from "./provider-edit-form-credential-fields";
import { SettingsFieldRow } from "./settings-field-row";

interface ProviderAuthFieldsProps {
  draft: ProviderDraft;
  onChange: ProviderDraftChange;
}

export function ProviderAuthFields({ draft, onChange }: ProviderAuthFieldsProps) {
  return (
    <>
      <SettingsFieldRow
        variant="modal"
        data-testid="settings-providers-editor-auth-mode"
        label="Auth mode"
        description="Owner of provider authentication at launch."
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
      <ProviderCredentialFields draft={draft} onChange={onChange} />
    </>
  );
}
