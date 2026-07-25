import { Fingerprint } from "lucide-react";

import {
  Field,
  FieldContent,
  FieldDescription,
  FieldLabel,
  FieldTitle,
  FormSection,
  Input,
  RequiredMark,
  Switch,
} from "@agh/ui";

import type { BridgeCreateDraft, BridgeProvider } from "../types";
import { BridgeCreateProviderStep } from "./bridge-create-provider-step";

export interface BridgeCreateSimpleSectionProps {
  draft: BridgeCreateDraft;
  onDraftChange: (draft: BridgeCreateDraft) => void;
  onSelectProvider: (key: string) => void;
  providers: BridgeProvider[];
  supportsManifest: boolean;
}

/**
 * Simple tier: where the bridge connects and how it identifies itself.
 *
 * Provider and account are fixed at creation — `UpdateBridgeRequest` omits
 * `platform` and `extension_name` — so the choice belongs here, never behind
 * Advanced.
 */
export function BridgeCreateSimpleSection({
  draft,
  onDraftChange,
  onSelectProvider,
  providers,
  supportsManifest,
}: BridgeCreateSimpleSectionProps) {
  return (
    <>
      <BridgeCreateProviderStep
        onSelect={onSelectProvider}
        providers={providers}
        selectedProviderKey={draft.selectedProviderKey}
        supportsManifest={supportsManifest}
      />

      <FormSection
        data-testid="bridge-create-section-identity"
        description="How this bridge appears in lists and receipts."
        icon={Fingerprint}
        size="compact"
        title="Identity"
      >
        <Field>
          <FieldLabel htmlFor="bridge-display-name-input">
            Display name
            <RequiredMark />
          </FieldLabel>
          <Input
            data-testid="bridge-display-name-input"
            id="bridge-display-name-input"
            onChange={event => onDraftChange({ ...draft, displayName: event.target.value })}
            placeholder="Support operations"
            value={draft.displayName}
          />
        </Field>

        <Field orientation="horizontal">
          <FieldContent>
            <FieldTitle>Enable after creation</FieldTitle>
            <FieldDescription>
              Start receiving platform events as soon as the daemon accepts the bridge.
            </FieldDescription>
          </FieldContent>
          <Switch
            aria-label="Enable after creation"
            checked={draft.enabled}
            data-testid="bridge-create-enabled"
            onCheckedChange={enabled => onDraftChange({ ...draft, enabled })}
          />
        </Field>
      </FormSection>
    </>
  );
}
