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
  Pill,
  NativeSelect,
  NativeSelectOption,
  Section,
  Spinner,
  Switch,
  Textarea,
} from "@agh/ui";

import {
  describeBridgeDmPolicy,
  describeBridgeProviderConfigSchema,
  describeBridgeRoutingPolicy,
} from "../lib/bridge-formatters";
import { parseBridgeProviderConfig } from "../lib/bridge-drafts";
import type { BridgeProvider, BridgeUpdateDraft } from "../types";

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

export function BridgeEditDialog(props: BridgeEditDialogProps) {
  return renderBridgeEditDialog(props);
}

function renderBridgeEditDialog({
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
      <DialogContent
        className="gap-0 p-0 text-(--fg) sm:max-w-3xl"
        showCloseButton={false}
        unframed
      >
        <div className="flex max-h-[min(80vh,900px)] flex-col" data-testid="bridge-edit-dialog">
          <DialogHeader variant="ruled">
            <DialogTitle>Edit Bridge</DialogTitle>
            <DialogDescription>
              Update mutable bridge settings for {bridgeName ?? "the selected bridge"} and restart
              the runtime after saving to apply provider-owned changes.
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
                    data-testid="bridge-edit-display-name-input"
                    onChange={event =>
                      onDraftChange({
                        ...draft,
                        displayName: event.target.value,
                      })
                    }
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
                <p className="text-small-body text-(--muted)">
                  Provider-owned runtime settings remain separate from generic delivery defaults.
                </p>

                <MetadataList className="mt-3">
                  <MetadataList.Row
                    className="rounded-md border border-(--line) bg-(--canvas-soft) px-4 py-3"
                    label="Config schema"
                    termProps={{ className: "mb-2 text-(--muted)" }}
                    valueProps={{ className: "text-small-body text-(--fg)" }}
                  >
                    {describeBridgeProviderConfigSchema(provider?.config_schema)}
                  </MetadataList.Row>
                  {provider?.secret_slots?.length ? (
                    <div className="mt-3 flex items-center gap-2">
                      <Pill mono>{provider.secret_slots.length}</Pill>
                      <p className="text-xs text-(--muted)">
                        Secret slots are managed inline from the detail panel.
                      </p>
                    </div>
                  ) : null}
                </MetadataList>

                <Field className="mt-4">
                  <FieldContent>
                    <FieldTitle>Provider config</FieldTitle>
                    <FieldDescription>
                      Enter a JSON object for provider-specific runtime settings such as tenant
                      identifiers, webhook URLs, or provider mode flags.
                    </FieldDescription>
                  </FieldContent>
                  <Textarea
                    aria-invalid={Boolean(providerConfigError)}
                    className="min-h-32 font-mono text-xs"
                    data-testid="bridge-edit-provider-config-input"
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
                    <p
                      className="text-small-body text-(--danger)"
                      data-testid="bridge-edit-provider-config-error"
                    >
                      {providerConfigError}
                    </p>
                  ) : null}
                </Field>
              </Section>

              <Section label="Routing policy">
                <p className="text-small-body text-(--muted)">
                  {describeBridgeRoutingPolicy(draft.routingPolicy)}
                </p>
                <FieldGroup className="mt-3 gap-3">
                  <Field orientation="horizontal">
                    <Switch
                      checked={draft.routingPolicy.include_peer}
                      data-testid="bridge-edit-routing-include-peer"
                      onCheckedChange={checked =>
                        onDraftChange({
                          ...draft,
                          routingPolicy: {
                            ...draft.routingPolicy,
                            include_peer: checked,
                          },
                        })
                      }
                    />
                    <FieldContent>
                      <FieldTitle>Include peer</FieldTitle>
                      <FieldDescription>
                        Differentiate direct targets by peer identifier.
                      </FieldDescription>
                    </FieldContent>
                  </Field>
                  <Field orientation="horizontal">
                    <Switch
                      checked={draft.routingPolicy.include_group}
                      data-testid="bridge-edit-routing-include-group"
                      onCheckedChange={checked =>
                        onDraftChange({
                          ...draft,
                          routingPolicy: {
                            ...draft.routingPolicy,
                            include_group: checked,
                          },
                        })
                      }
                    />
                    <FieldContent>
                      <FieldTitle>Include group</FieldTitle>
                      <FieldDescription>
                        Keep routes isolated per group or channel when the platform supports it.
                      </FieldDescription>
                    </FieldContent>
                  </Field>
                  <Field orientation="horizontal">
                    <Switch
                      checked={draft.routingPolicy.include_thread}
                      data-testid="bridge-edit-routing-include-thread"
                      onCheckedChange={checked =>
                        onDraftChange({
                          ...draft,
                          routingPolicy: {
                            ...draft.routingPolicy,
                            include_thread: checked,
                          },
                        })
                      }
                    />
                    <FieldContent>
                      <FieldTitle>Include thread</FieldTitle>
                      <FieldDescription>
                        Use thread identity as an additional routing dimension.
                      </FieldDescription>
                    </FieldContent>
                  </Field>
                </FieldGroup>
              </Section>

              <Section label="Delivery defaults">
                <p className="text-small-body text-(--muted)">
                  These defaults are applied when resolving outbound delivery targets.
                </p>
                <FieldGroup className="mt-3 grid gap-4 lg:grid-cols-2">
                  <Field>
                    <FieldContent>
                      <FieldTitle>Mode</FieldTitle>
                    </FieldContent>
                    <NativeSelect
                      data-testid="bridge-edit-delivery-mode-select"
                      onChange={event =>
                        onDraftChange({
                          ...draft,
                          deliveryDefaults: {
                            ...draft.deliveryDefaults,
                            mode:
                              event.target.value === ""
                                ? undefined
                                : (event.target.value as NonNullable<
                                    BridgeUpdateDraft["deliveryDefaults"]["mode"]
                                  >),
                          },
                        })
                      }
                      value={draft.deliveryDefaults.mode ?? ""}
                    >
                      <NativeSelectOption value="">Use runtime default</NativeSelectOption>
                      <NativeSelectOption value="reply">Reply</NativeSelectOption>
                      <NativeSelectOption value="direct-send">Direct send</NativeSelectOption>
                    </NativeSelect>
                  </Field>
                  <Field>
                    <FieldContent>
                      <FieldTitle>Peer ID</FieldTitle>
                    </FieldContent>
                    <Input
                      data-testid="bridge-edit-delivery-peer-input"
                      onChange={event =>
                        onDraftChange({
                          ...draft,
                          deliveryDefaults: {
                            ...draft.deliveryDefaults,
                            peer_id: event.target.value,
                          },
                        })
                      }
                      placeholder="peer_123"
                      value={draft.deliveryDefaults.peer_id ?? ""}
                    />
                  </Field>
                  <Field>
                    <FieldContent>
                      <FieldTitle>Thread ID</FieldTitle>
                    </FieldContent>
                    <Input
                      data-testid="bridge-edit-delivery-thread-input"
                      onChange={event =>
                        onDraftChange({
                          ...draft,
                          deliveryDefaults: {
                            ...draft.deliveryDefaults,
                            thread_id: event.target.value,
                          },
                        })
                      }
                      placeholder="thread_456"
                      value={draft.deliveryDefaults.thread_id ?? ""}
                    />
                  </Field>
                  <Field>
                    <FieldContent>
                      <FieldTitle>Group ID</FieldTitle>
                    </FieldContent>
                    <Input
                      data-testid="bridge-edit-delivery-group-input"
                      onChange={event =>
                        onDraftChange({
                          ...draft,
                          deliveryDefaults: {
                            ...draft.deliveryDefaults,
                            group_id: event.target.value,
                          },
                        })
                      }
                      placeholder="group_789"
                      value={draft.deliveryDefaults.group_id ?? ""}
                    />
                  </Field>
                </FieldGroup>
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
              size="sm"
              onClick={onSubmit}
              type="button"
            >
              {isPending ? (
                <>
                  <Spinner className="size-3.5" />
                  Saving…
                </>
              ) : (
                "Save Changes"
              )}
            </Button>
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  );
}
