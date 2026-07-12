import { Plug, Settings2 } from "lucide-react";

import {
  Eyebrow,
  Field,
  FieldContent,
  FieldDescription,
  FieldTitle,
  FormSection,
  Input,
  NativeSelect,
  NativeSelectOption,
  Pill,
  Textarea,
} from "@agh/ui";

import {
  describeBridgeDmPolicy,
  describeBridgeProviderConfigSchema,
  describeBridgeSecretSlot,
} from "../lib/bridge-formatters";
import type { BridgeCreateDraft, BridgeProvider } from "../types";
import { RuntimeMetadataTile } from "./bridge-runtime-metadata";

interface RuntimeStepProps {
  activeWorkspaceId?: string | null;
  activeWorkspaceName?: string | null;
  draft: BridgeCreateDraft;
  onDraftChange: (draft: BridgeCreateDraft) => void;
  provider: BridgeProvider;
  providerConfigError?: string;
}

export function RuntimeStep({
  activeWorkspaceId,
  activeWorkspaceName,
  draft,
  onDraftChange,
  provider,
  providerConfigError,
}: RuntimeStepProps) {
  const configSchema = describeBridgeProviderConfigSchema(provider.config_schema);

  return (
    <>
      <FormSection
        data-testid="bridge-wizard-section-identity"
        description="Operator-visible label and ownership scope for the bridge instance."
        icon={Settings2}
        title="Identity"
      >
        <div className="grid gap-4 md:grid-cols-2">
          <Field>
            <FieldContent>
              <FieldTitle>Display name</FieldTitle>
              <FieldDescription>Surfaces in lists, detail headers, and alerts.</FieldDescription>
            </FieldContent>
            <Input
              data-testid="bridge-display-name-input"
              onChange={event =>
                onDraftChange({
                  ...draft,
                  displayName: event.target.value,
                })
              }
              placeholder={provider.display_name ?? "Support bridge"}
              value={draft.displayName}
            />
          </Field>

          <Field>
            <FieldContent>
              <FieldTitle>Scope</FieldTitle>
              <FieldDescription>
                Workspace scope uses {activeWorkspaceName ?? "the active workspace"} as the owning
                context.
              </FieldDescription>
            </FieldContent>
            <NativeSelect
              data-testid="bridge-scope-select"
              onChange={event =>
                onDraftChange({
                  ...draft,
                  scope: event.target.value as BridgeCreateDraft["scope"],
                })
              }
              value={draft.scope}
            >
              <NativeSelectOption value="global">Global</NativeSelectOption>
              <NativeSelectOption disabled={!activeWorkspaceId} value="workspace">
                Workspace {activeWorkspaceName ? `(${activeWorkspaceName})` : ""}
              </NativeSelectOption>
            </NativeSelect>
          </Field>
        </div>
      </FormSection>

      <FormSection
        data-testid="bridge-wizard-section-runtime"
        description="Provider-owned runtime configuration, DM policy, and secret requirements stay separate from generic routing and delivery defaults."
        icon={Plug}
        rightLabel={configSchema}
        title="Provider runtime"
      >
        <div className="grid gap-3 lg:grid-cols-2" data-testid="bridge-provider-runtime-section">
          <RuntimeMetadataTile label="Config schema">
            <span data-testid="bridge-provider-config-schema">{configSchema}</span>
          </RuntimeMetadataTile>
          <RuntimeMetadataTile
            label="Secret slots"
            right={<Pill mono>{provider.secret_slots?.length ?? 0}</Pill>}
          >
            {provider.secret_slots?.length ? (
              <ul className="mt-1 flex flex-col gap-1.5" data-testid="bridge-provider-secret-slots">
                {provider.secret_slots.map(slot => (
                  <li className="rounded-xs bg-canvas-tint px-3 py-2" key={slot.name}>
                    <div className="flex flex-wrap items-center gap-1.5">
                      <Eyebrow className="text-muted">{slot.name}</Eyebrow>
                      <Pill mono tone={slot.required === false ? "neutral" : "warning"}>
                        <Eyebrow>{slot.required === false ? "OPTIONAL" : "REQUIRED"}</Eyebrow>
                      </Pill>
                    </div>
                    <p className="mt-1 text-xs leading-relaxed text-muted">
                      {describeBridgeSecretSlot(slot)}
                    </p>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-small-body leading-relaxed text-muted">
                This provider does not declare secret slot requirements in its manifest.
              </p>
            )}
          </RuntimeMetadataTile>
        </div>

        <Field>
          <FieldContent>
            <FieldTitle>DM policy</FieldTitle>
            <FieldDescription>
              {describeBridgeDmPolicy(draft.dmPolicy === "" ? undefined : draft.dmPolicy)}
            </FieldDescription>
          </FieldContent>
          <NativeSelect
            data-testid="bridge-dm-policy-select"
            onChange={event =>
              onDraftChange({
                ...draft,
                dmPolicy: event.target.value as BridgeCreateDraft["dmPolicy"],
              })
            }
            value={draft.dmPolicy}
          >
            <NativeSelectOption value="">Use provider default</NativeSelectOption>
            <NativeSelectOption value="open">Open</NativeSelectOption>
            <NativeSelectOption value="allowlist">Allowlist</NativeSelectOption>
            <NativeSelectOption value="pairing">Pairing</NativeSelectOption>
          </NativeSelect>
        </Field>

        <Field>
          <FieldContent>
            <FieldTitle>Provider config</FieldTitle>
            <FieldDescription>
              Enter a JSON object for provider-specific runtime settings such as tenant identifiers,
              webhook URLs, or provider mode flags.
            </FieldDescription>
          </FieldContent>
          <Textarea
            aria-invalid={Boolean(providerConfigError)}
            className="min-h-32 font-mono text-xs"
            data-testid="bridge-provider-config-input"
            onChange={event =>
              onDraftChange({
                ...draft,
                providerConfigText: event.target.value,
              })
            }
            placeholder={`{\n  "mode": "bot"\n}`}
            spellCheck={false}
            value={draft.providerConfigText}
          />
          {providerConfigError ? (
            <p className="text-small-body text-danger" data-testid="bridge-provider-config-error">
              {providerConfigError}
            </p>
          ) : (
            <p className="text-xs leading-relaxed text-subtle">Hint: {configSchema}</p>
          )}
        </Field>
      </FormSection>
    </>
  );
}
