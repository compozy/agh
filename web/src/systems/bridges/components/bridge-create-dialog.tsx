import { Loader2 } from "lucide-react";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldSet,
  FieldTitle,
  Input,
  NativeSelect,
  NativeSelectOption,
  Pill,
  Switch,
  Textarea,
} from "@agh/ui";

import {
  buildBridgeProviderKey,
  describeBridgeDmPolicy,
  describeBridgeProviderConfigSchema,
  describeBridgeRoutingPolicy,
  describeBridgeSecretSlot,
  findBridgeProviderByKey,
  isBridgeProviderSelectable,
} from "../lib/bridge-formatters";
import { parseBridgeProviderConfig } from "../lib/bridge-drafts";
import type { BridgeCreateDraft, BridgeProvider } from "../types";
import { BridgeProviderCard } from "./bridge-provider-card";

import { pillVariantFromTone } from "@/lib/pill-variant";
interface BridgeCreateDialogProps {
  activeWorkspaceId?: string | null;
  activeWorkspaceName?: string | null;
  draft: BridgeCreateDraft;
  isPending: boolean;
  onDraftChange: (draft: BridgeCreateDraft) => void;
  onOpenChange: (open: boolean) => void;
  onSubmit: () => void;
  open: boolean;
  providers: BridgeProvider[];
}

export function BridgeCreateDialog({
  activeWorkspaceId,
  activeWorkspaceName,
  draft,
  isPending,
  onDraftChange,
  onOpenChange,
  onSubmit,
  open,
  providers,
}: BridgeCreateDialogProps) {
  const selectedProvider = findBridgeProviderByKey(providers, draft.selectedProviderKey);
  const providerConfigError = parseBridgeProviderConfig(draft.providerConfigText).error;
  const canSubmit = Boolean(
    selectedProvider &&
    isBridgeProviderSelectable(selectedProvider) &&
    draft.displayName.trim() &&
    !providerConfigError &&
    (draft.scope === "global" || activeWorkspaceId)
  );

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent
        className="max-w-[calc(100%-2rem)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-0 text-[color:var(--color-text-primary)] sm:max-w-4xl"
        showCloseButton={false}
      >
        <form
          className="flex max-h-[min(80vh,900px)] flex-col"
          data-testid="bridge-create-dialog"
          onSubmit={event => {
            event.preventDefault();
            onSubmit();
          }}
        >
          <DialogHeader className="space-y-2 px-6 pt-6">
            <DialogTitle>Create Bridge</DialogTitle>
            <DialogDescription className="text-[color:var(--color-text-secondary)]">
              Select an installed provider, configure provider-owned runtime settings separately
              from delivery defaults, and scope the bridge globally or to the active workspace.
            </DialogDescription>
          </DialogHeader>

          <div className="flex-1 overflow-y-auto px-6 py-6">
            <FieldSet className="gap-6">
              <section className="space-y-3">
                <div className="space-y-1">
                  <p className="font-mono text-[0.68rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
                    Provider
                  </p>
                  <p className="text-sm text-[color:var(--color-text-secondary)]">
                    Only providers with healthy runtime state can be selected for bridge creation.
                  </p>
                </div>
                {providers.length === 0 ? (
                  <div
                    className="rounded-xl border border-dashed border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-5 py-8 text-center text-sm leading-6 text-[color:var(--color-text-secondary)]"
                    data-testid="bridge-provider-empty"
                  >
                    No bridge providers are currently available. Install or enable a bridge adapter
                    extension before creating a new bridge.
                  </div>
                ) : (
                  <div className="grid gap-3 lg:grid-cols-2">
                    {providers.map(provider => (
                      <BridgeProviderCard
                        key={buildBridgeProviderKey(provider)}
                        onSelect={() =>
                          onDraftChange({
                            ...draft,
                            displayName:
                              !draft.displayName.trim() ||
                              draft.displayName.trim() === selectedProvider?.display_name
                                ? provider.display_name
                                : draft.displayName,
                            selectedProviderKey: buildBridgeProviderKey(provider),
                          })
                        }
                        provider={provider}
                        selected={buildBridgeProviderKey(provider) === draft.selectedProviderKey}
                      />
                    ))}
                  </div>
                )}
              </section>

              {selectedProvider ? (
                <section
                  className="space-y-4 rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] p-4"
                  data-testid="bridge-provider-runtime-section"
                >
                  <div className="space-y-1">
                    <p className="font-mono text-[0.68rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
                      Provider runtime
                    </p>
                    <p className="text-sm text-[color:var(--color-text-secondary)]">
                      Provider-owned runtime configuration, DM policy, and secret requirements stay
                      separate from generic routing and delivery defaults.
                    </p>
                  </div>

                  <div className="grid gap-4 lg:grid-cols-2">
                    <div className="rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-3">
                      <p className="font-mono text-[0.64rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
                        Config schema
                      </p>
                      <p
                        className="mt-2 text-sm text-[color:var(--color-text-primary)]"
                        data-testid="bridge-provider-config-schema"
                      >
                        {describeBridgeProviderConfigSchema(selectedProvider.config_schema)}
                      </p>
                    </div>

                    <div className="rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-3">
                      <div className="flex items-center justify-between gap-3">
                        <p className="font-mono text-[0.64rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
                          Secret slots
                        </p>
                        <Pill>{selectedProvider.secret_slots?.length ?? 0}</Pill>
                      </div>
                      {selectedProvider.secret_slots?.length ? (
                        <ul className="mt-3 space-y-2" data-testid="bridge-provider-secret-slots">
                          {selectedProvider.secret_slots.map(slot => (
                            <li
                              key={slot.name}
                              className="rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-3 py-2"
                            >
                              <div className="flex flex-wrap items-center gap-2">
                                <span className="font-mono text-[0.7rem] uppercase tracking-[0.12em] text-[color:var(--color-text-primary)]">
                                  {slot.name}
                                </span>
                                <Pill
                                  variant={pillVariantFromTone(
                                    slot.required === false ? "neutral" : "amber"
                                  )}
                                >
                                  {slot.required === false ? "optional" : "required"}
                                </Pill>
                              </div>
                              <p className="mt-2 text-xs leading-relaxed text-[color:var(--color-text-secondary)]">
                                {describeBridgeSecretSlot(slot)}
                              </p>
                            </li>
                          ))}
                        </ul>
                      ) : (
                        <p className="mt-3 text-sm leading-relaxed text-[color:var(--color-text-secondary)]">
                          This provider does not declare secret slot requirements in its manifest.
                        </p>
                      )}
                    </div>
                  </div>

                  <FieldGroup className="gap-4">
                    <Field>
                      <FieldContent>
                        <FieldTitle>DM policy</FieldTitle>
                        <FieldDescription>
                          {describeBridgeDmPolicy(
                            draft.dmPolicy === "" ? undefined : draft.dmPolicy
                          )}
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
                          Enter a JSON object for provider-specific runtime settings such as tenant
                          identifiers, webhook URLs, or provider mode flags.
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
                        <p
                          className="text-sm text-[color:var(--color-danger)]"
                          data-testid="bridge-provider-config-error"
                        >
                          {providerConfigError}
                        </p>
                      ) : (
                        <p className="text-xs leading-relaxed text-[color:var(--color-text-tertiary)]">
                          Hint: {describeBridgeProviderConfigSchema(selectedProvider.config_schema)}
                        </p>
                      )}
                    </Field>
                  </FieldGroup>
                </section>
              ) : null}

              <FieldGroup className="grid gap-4 lg:grid-cols-2">
                <Field>
                  <FieldContent>
                    <FieldTitle>Display name</FieldTitle>
                    <FieldDescription>
                      Operator-visible label for the bridge instance.
                    </FieldDescription>
                  </FieldContent>
                  <Input
                    data-testid="bridge-display-name-input"
                    onChange={event =>
                      onDraftChange({
                        ...draft,
                        displayName: event.target.value,
                      })
                    }
                    placeholder={selectedProvider?.display_name ?? "Support bridge"}
                    value={draft.displayName}
                  />
                </Field>

                <Field>
                  <FieldContent>
                    <FieldTitle>Scope</FieldTitle>
                    <FieldDescription>
                      Workspace scope uses {activeWorkspaceName ?? "the active workspace"} as the
                      owning context.
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
              </FieldGroup>

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
                      data-testid="bridge-delivery-mode-select"
                      onChange={event =>
                        onDraftChange({
                          ...draft,
                          deliveryDefaults: {
                            ...draft.deliveryDefaults,
                            mode:
                              event.target.value === ""
                                ? undefined
                                : (event.target.value as NonNullable<
                                    BridgeCreateDraft["deliveryDefaults"]["mode"]
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
                      data-testid="bridge-delivery-peer-input"
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
                      data-testid="bridge-delivery-thread-input"
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
                      data-testid="bridge-delivery-group-input"
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
              data-testid="submit-bridge-create"
              disabled={!canSubmit || isPending}
              size="lg"
              type="submit"
            >
              {isPending ? (
                <>
                  <Loader2 className="size-4 animate-spin" />
                  Creating…
                </>
              ) : (
                "Create Bridge"
              )}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
