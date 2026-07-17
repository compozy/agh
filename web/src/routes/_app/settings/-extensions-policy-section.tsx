import type { Dispatch, SetStateAction } from "react";

import { SettingsFieldRow, type SettingsHooksExtensionsSection } from "@/systems/settings";
import { Button, Input, Section, Spinner, Switch } from "@agh/ui";

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
    <Section
      data-testid="settings-page-extensions-policy-section"
      label="Extensions policy"
      right={
        <SaveControls
          canMutate={canMutate}
          error={error}
          isDirty={isDirty}
          isSaving={isSaving}
          onReset={onReset}
          onSave={onSave}
          warnings={warnings}
        />
      }
    >
      <SettingsFieldRow
        data-testid="settings-page-extensions-policy-registry"
        description="Identifier of the marketplace publisher"
        hint="CONFIG.TOML"
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
        hint="OPTIONAL"
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
        hint="TRUST"
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
    </Section>
  );
}

function SaveControls({
  isDirty,
  isSaving,
  canMutate,
  error,
  warnings,
  onSave,
  onReset,
}: {
  isDirty: boolean;
  isSaving: boolean;
  canMutate: boolean;
  error: string | null;
  warnings?: string[];
  onSave: () => void;
  onReset: () => void;
}) {
  return (
    <div
      className="flex flex-wrap items-center gap-2"
      data-testid="settings-page-extensions-policy-controls"
    >
      <span
        aria-live={error ? "assertive" : "polite"}
        className={error ? "text-xs text-danger" : "text-xs text-muted"}
      >
        {error ?? (warnings?.length ? warnings.join(" · ") : isDirty ? "Unsaved changes" : "")}
      </span>
      <Button
        data-testid="settings-page-extensions-policy-reset"
        disabled={!isDirty || isSaving}
        onClick={onReset}
        size="sm"
        type="button"
        variant="ghost"
      >
        Discard
      </Button>
      <Button
        data-testid="settings-page-extensions-policy-save"
        disabled={!isDirty || isSaving || !canMutate}
        onClick={onSave}
        size="sm"
        type="button"
      >
        {isSaving ? <Spinner className="size-3" /> : null}
        {isSaving ? "Saving…" : "Save policy"}
      </Button>
    </div>
  );
}
