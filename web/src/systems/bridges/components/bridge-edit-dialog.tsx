import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldSet,
  FieldTitle,
  Input,
  MetadataList,
  NativeSelect,
  NativeSelectOption,
  Pill,
  Section,
  Spinner,
  Textarea,
} from "@agh/ui";

import { parseBridgeProviderConfig } from "../lib/bridge-drafts";
import {
  describeBridgeDmPolicy,
  describeBridgeProviderConfigSchema,
  describeBridgeRoutingPolicy,
} from "../lib/bridge-formatters";
import type { BridgeProvider, BridgeUpdateDraft } from "../types";
import { BridgeDeliveryFields, BridgeRoutingFields } from "./bridge-delivery-fields";

interface BridgeEditDialogProps {
  allowProviderDefaultDmPolicy: boolean;
  bridgeName?: string;
  draft: BridgeUpdateDraft;
  isPending: boolean;
  onDraftChange: (draft: BridgeUpdateDraft) => void;
  onOpenChange: (open: boolean) => void;
  onSubmit: () => void;
  open: boolean;
  provider?: BridgeProvider;
}

export function BridgeEditDialog({
  allowProviderDefaultDmPolicy,
  bridgeName,
  draft,
  isPending,
  onDraftChange,
  onOpenChange,
  onSubmit,
  open,
  provider,
}: BridgeEditDialogProps) {
  const providerConfigError = parseBridgeProviderConfig(draft.providerConfigText).error;
  const canSubmit = Boolean(draft.displayName.trim() && !providerConfigError);

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="gap-0 p-0 text-fg sm:max-w-3xl" showCloseButton={false} unframed>
        <div
          className="flex max-h-[min(80vh,var(--height-modal-tall))] flex-col"
          data-testid="bridge-edit-dialog"
        >
          <DialogHeader variant="ruled">
            <DialogTitle>Edit bridge</DialogTitle>
            <DialogDescription>
              Update mutable settings for {bridgeName ?? "the selected bridge"}, then restart the
              runtime to apply provider-owned changes.
            </DialogDescription>
          </DialogHeader>

          <div className="flex-1 overflow-y-auto p-5">
            <FieldSet className="gap-6">
              <FieldGroup className="grid gap-4 lg:grid-cols-2">
                <Field>
                  <FieldContent>
                    <FieldTitle>Display name</FieldTitle>
                    <FieldDescription>
                      Operator-visible label for the bridge instance.
                    </FieldDescription>
                  </FieldContent>
                  <Input
                    aria-label="Bridge display name"
                    data-testid="bridge-edit-display-name-input"
                    onChange={event => onDraftChange({ ...draft, displayName: event.target.value })}
                    placeholder="Support bridge"
                    value={draft.displayName}
                  />
                </Field>

                <Field>
                  <FieldContent>
                    <FieldTitle>DM policy</FieldTitle>
                    <FieldDescription>
                      {describeBridgeDmPolicy(draft.dmPolicy === "" ? undefined : draft.dmPolicy)}
                    </FieldDescription>
                  </FieldContent>
                  <NativeSelect
                    aria-label="Direct message policy"
                    data-testid="bridge-edit-dm-policy-select"
                    onChange={event =>
                      onDraftChange({
                        ...draft,
                        dmPolicy: event.target.value as BridgeUpdateDraft["dmPolicy"],
                      })
                    }
                    value={draft.dmPolicy}
                  >
                    {allowProviderDefaultDmPolicy ? (
                      <NativeSelectOption value="">Use provider default</NativeSelectOption>
                    ) : null}
                    <NativeSelectOption value="open">Open</NativeSelectOption>
                    <NativeSelectOption value="allowlist">Allowlist</NativeSelectOption>
                    <NativeSelectOption value="pairing">Pairing</NativeSelectOption>
                  </NativeSelect>
                </Field>
              </FieldGroup>

              <Section label="Provider runtime">
                <p className="text-small-body text-muted">
                  Provider-owned settings remain separate from generic delivery defaults.
                </p>
                <MetadataList className="mt-3">
                  <MetadataList.Row
                    className="rounded-md border border-line bg-canvas-soft px-4 py-3"
                    label="Config schema"
                    termProps={{ className: "mb-2 text-muted" }}
                    valueProps={{ className: "text-small-body text-fg" }}
                  >
                    {describeBridgeProviderConfigSchema(provider?.config_schema)}
                  </MetadataList.Row>
                  {provider?.secret_slots?.length ? (
                    <div className="mt-3 flex items-center gap-2">
                      <Pill mono>{provider.secret_slots.length}</Pill>
                      <p className="text-xs text-muted">
                        Secret slots are managed inline from the detail panel.
                      </p>
                    </div>
                  ) : null}
                </MetadataList>
                <Field className="mt-4">
                  <FieldContent>
                    <FieldTitle>Provider config</FieldTitle>
                    <FieldDescription>
                      Enter a JSON object for provider-specific settings such as tenant identifiers,
                      webhook URLs, or provider mode flags.
                    </FieldDescription>
                  </FieldContent>
                  <Textarea
                    aria-invalid={Boolean(providerConfigError)}
                    aria-label="Provider configuration JSON"
                    className="min-h-32 font-mono text-xs"
                    data-testid="bridge-edit-provider-config-input"
                    onChange={event =>
                      onDraftChange({ ...draft, providerConfigText: event.target.value })
                    }
                    placeholder={`{\n  "mode": "bot"\n}`}
                    spellCheck={false}
                    value={draft.providerConfigText}
                  />
                  {providerConfigError ? (
                    <p
                      className="text-small-body text-danger"
                      data-testid="bridge-edit-provider-config-error"
                    >
                      {providerConfigError}
                    </p>
                  ) : null}
                </Field>
              </Section>

              <Section label="Routing policy">
                <p className="mb-3 text-small-body text-muted">
                  {describeBridgeRoutingPolicy(draft.routingPolicy)}
                </p>
                <BridgeRoutingFields
                  onChange={routingPolicy => onDraftChange({ ...draft, routingPolicy })}
                  testIdPrefix="bridge-edit"
                  value={draft.routingPolicy}
                />
              </Section>

              <Section label="Delivery defaults">
                <p className="mb-3 text-small-body text-muted">
                  These defaults are applied when resolving outbound delivery targets.
                </p>
                <BridgeDeliveryFields
                  onChange={deliveryDefaults => onDraftChange({ ...draft, deliveryDefaults })}
                  testIdPrefix="bridge-edit"
                  value={draft.deliveryDefaults}
                />
              </Section>
            </FieldSet>
          </div>

          <DialogFooter variant="ruled">
            <Button onClick={() => onOpenChange(false)} size="sm" type="button" variant="outline">
              Cancel
            </Button>
            <Button
              data-testid="submit-bridge-edit"
              disabled={!canSubmit || isPending}
              onClick={onSubmit}
              size="sm"
              type="button"
            >
              {isPending ? (
                <>
                  <Spinner className="size-3" />
                  Saving…
                </>
              ) : (
                "Save changes"
              )}
            </Button>
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  );
}
