import { Loader2 } from "lucide-react";

import { Pill } from "@/components/design-system";
import { Button, Input } from "@agh/ui";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldSet,
  FieldTitle,
} from "@/components/ui/field";
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";

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
      <DialogContent
        className="max-w-[calc(100%-2rem)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-0 text-[color:var(--color-text-primary)] sm:max-w-3xl"
        showCloseButton={false}
      >
        <form
          className="flex max-h-[min(80vh,900px)] flex-col"
          data-testid="bridge-edit-dialog"
          onSubmit={event => {
            event.preventDefault();
            onSubmit();
          }}
        >
          <DialogHeader className="space-y-2 px-6 pt-6">
            <DialogTitle>Edit Bridge</DialogTitle>
            <DialogDescription className="text-[color:var(--color-text-secondary)]">
              Update mutable bridge settings for {bridgeName ?? "the selected bridge"} and restart
              the runtime after saving to apply provider-owned changes.
            </DialogDescription>
          </DialogHeader>

          <div className="flex-1 overflow-y-auto px-6 py-6">
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

              <section className="space-y-4 rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] p-4">
                <div className="space-y-1">
                  <p className="font-mono text-[0.68rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
                    Provider runtime
                  </p>
                  <p className="text-sm text-[color:var(--color-text-secondary)]">
                    Provider-owned runtime settings remain separate from generic delivery defaults.
                  </p>
                </div>

                <div className="rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-3">
                  <p className="font-mono text-[0.64rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
                    Config schema
                  </p>
                  <p className="mt-2 text-sm text-[color:var(--color-text-primary)]">
                    {describeBridgeProviderConfigSchema(provider?.config_schema)}
                  </p>
                  {provider?.secret_slots?.length ? (
                    <div className="mt-3 flex items-center gap-2">
                      <Pill kind="tag" tone="neutral">
                        {provider.secret_slots.length}
                      </Pill>
                      <p className="text-xs text-[color:var(--color-text-secondary)]">
                        Secret slots are managed inline from the detail panel.
                      </p>
                    </div>
                  ) : null}
                </div>

                <Field>
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
                      className="text-sm text-[color:var(--color-danger)]"
                      data-testid="bridge-edit-provider-config-error"
                    >
                      {providerConfigError}
                    </p>
                  ) : null}
                </Field>
              </section>

              <section className="space-y-4 rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] p-4">
                <div className="space-y-1">
                  <p className="font-mono text-[0.68rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
                    Routing policy
                  </p>
                  <p className="text-sm text-[color:var(--color-text-secondary)]">
                    {describeBridgeRoutingPolicy(draft.routingPolicy)}
                  </p>
                </div>
                <FieldGroup className="gap-3">
                  <Field orientation="horizontal">
                    <Switch
                      checked={draft.routingPolicy.include_peer}
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
              </section>

              <section className="space-y-4 rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] p-4">
                <div className="space-y-1">
                  <p className="font-mono text-[0.68rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
                    Delivery defaults
                  </p>
                  <p className="text-sm text-[color:var(--color-text-secondary)]">
                    These defaults are applied when resolving outbound delivery targets.
                  </p>
                </div>
                <FieldGroup className="grid gap-4 lg:grid-cols-2">
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
              </section>
            </FieldSet>
          </div>

          <div className="flex items-center justify-end gap-2 border-t border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-6 py-4">
            <Button
              className="border-[color:var(--color-divider)] bg-transparent text-[color:var(--color-text-primary)] hover:bg-[color:var(--color-hover)]"
              onClick={() => onOpenChange(false)}
              size="lg"
              type="button"
              variant="outline"
            >
              Cancel
            </Button>
            <Button
              data-testid="submit-bridge-edit"
              disabled={!canSubmit || isPending}
              size="lg"
              type="submit"
            >
              {isPending ? (
                <>
                  <Loader2 className="size-4 animate-spin" />
                  Saving…
                </>
              ) : (
                "Save Changes"
              )}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
