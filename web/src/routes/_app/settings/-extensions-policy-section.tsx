import type { Dispatch, SetStateAction } from "react";

import {
  SettingsFieldRow,
  SettingsGroup,
  SettingsInlineSaveControls,
  type SettingsHooksExtensionsSection,
} from "@/systems/settings";
import { Input, Switch } from "@agh/ui";

type PolicyConfig = SettingsHooksExtensionsSection["config"];

interface PolicySectionProps {
  draft: PolicyConfig;
  setDraft: Dispatch<SetStateAction<PolicyConfig>>;
  isDirty: boolean;
  isSaving: boolean;
  error: string | null;
  warnings?: string[];
  canMutate: boolean;
  onSave: () => void;
  onReset: () => void;
}

export function PolicySection({
  draft,
  setDraft,
  isDirty,
  isSaving,
  error,
  warnings,
  canMutate,
  onSave,
  onReset,
}: PolicySectionProps) {
  return (
    <SettingsGroup
      data-testid="settings-page-extensions-policy-section"
      title="Extensions policy"
      action={
        <SettingsInlineSaveControls
          canSave={canMutate}
          controlTestIdPrefix="settings-page-extensions-policy"
          error={error}
          isDirty={isDirty}
          isSaving={isSaving}
          onReset={onReset}
          onSave={onSave}
          saveLabel="Save policy"
          testId="settings-page-extensions-policy-controls"
          warnings={warnings}
        />
      }
    >
      <SettingsFieldRow
        data-testid="settings-page-extensions-policy-registry"
        description="Identifier of the marketplace publisher"
        label="Marketplace registry"
        control={
          <Input
            className="w-56 font-mono"
            data-testid="settings-page-extensions-policy-registry-input"
            disabled={!canMutate}
            onChange={event =>
              setDraft(current => ({
                ...current,
                marketplace: { ...current.marketplace, registry: event.target.value },
              }))
            }
            value={draft.marketplace.registry ?? ""}
          />
        }
      />
      <SettingsFieldRow
        data-testid="settings-page-extensions-policy-base-url"
        description="Override the registry's default endpoint"
        label="Base URL"
        control={
          <Input
            className="w-72 font-mono"
            data-testid="settings-page-extensions-policy-base-url-input"
            disabled={!canMutate}
            onChange={event =>
              setDraft(current => ({
                ...current,
                marketplace: { ...current.marketplace, base_url: event.target.value },
              }))
            }
            placeholder="https://"
            value={draft.marketplace.base_url ?? ""}
          />
        }
      />
      <SettingsFieldRow
        data-testid="settings-page-extensions-policy-allow-unverified"
        description="Unverified extensions can be installed after an explicit warning"
        label="Allow unverified extensions"
        control={
          <Switch
            aria-label="Allow unverified extensions"
            checked={draft.marketplace.allow_unverified}
            data-testid="settings-page-extensions-policy-allow-unverified-input"
            disabled={!canMutate}
            onCheckedChange={allowUnverified =>
              setDraft(current => ({
                ...current,
                marketplace: {
                  ...current.marketplace,
                  allow_unverified: allowUnverified,
                },
              }))
            }
            size="sm"
          />
        }
      />
    </SettingsGroup>
  );
}
