import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Eyebrow,
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldSet,
  FieldTitle,
  Input,
  Item,
  ItemContent,
  ItemGroup,
  ItemHeader,
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

export function BridgeCreateDialog(props: BridgeCreateDialogProps) {
  return renderBridgeCreateDialog(props);
}

function renderBridgeCreateDialog({
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
    !providerConfigError
  );

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent
        className="gap-0 p-0 text-(--fg) sm:max-w-4xl"
        showCloseButton={false}
        unframed
      >
        <div className="flex max-h-[min(80vh,900px)] flex-col" data-testid="bridge-create-dialog">
          <DialogHeader variant="ruled">
            <DialogTitle>Create Bridge</DialogTitle>
            <DialogDescription>
              Select an installed provider, configure provider-owned runtime settings separately
              from delivery defaults, and scope the bridge globally or to the active workspace.
            </DialogDescription>
          </DialogHeader>

          <div className="flex-1 overflow-y-auto p-5">
            <FieldSet className="gap-6">
              <Section label="Provider">
                <p className="text-small-body text-(--muted)">
                  Only providers with healthy runtime state can be selected for bridge creation.
                </p>
                {providers.length === 0 ? (
                  <div
                    className="mt-3 rounded-md border border-dashed border-(--line) bg-(--canvas-soft) px-5 py-8 text-center text-small-body leading-6 text-(--muted)"
                    data-testid="bridge-provider-empty"
                  >
                    No bridge providers are currently available. Install or enable a bridge adapter
                    extension before creating a new bridge.
                  </div>
                ) : (
                  <div className="mt-3 grid gap-3 lg:grid-cols-2">
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
              </Section>

              {selectedProvider ? (
                <Section data-testid="bridge-provider-runtime-section" label="Provider runtime">
                  <p className="text-small-body text-(--muted)">
                    Provider-owned runtime configuration, DM policy, and secret requirements stay
                    separate from generic routing and delivery defaults.
                  </p>

                  <MetadataList className="mt-3 grid gap-3 lg:grid-cols-2">
                    <MetadataList.Row
                      className="rounded-md border border-(--line) bg-(--canvas-soft) px-4 py-3"
                      label="Config schema"
                      termProps={{ className: "mb-2 text-(--muted)" }}
                      valueProps={{
                        className: "text-small-body text-(--fg)",
                        "data-testid": "bridge-provider-config-schema",
                      }}
                    >
                      {describeBridgeProviderConfigSchema(selectedProvider.config_schema)}
                    </MetadataList.Row>

                    <div className="rounded-md border border-(--line) bg-(--canvas-soft) px-4 py-3">
                      <div className="flex items-center justify-between gap-3">
                        <Eyebrow tone="neutral">Secret slots</Eyebrow>
                        <Pill mono>{selectedProvider.secret_slots?.length ?? 0}</Pill>
                      </div>
                      {selectedProvider.secret_slots?.length ? (
                        <ItemGroup
                          className="mt-3 gap-2"
                          data-testid="bridge-provider-secret-slots"
                        >
                          {selectedProvider.secret_slots.map(slot => (
                            <Item
                              key={slot.name}
                              className="rounded-md border border-(--line) bg-(--canvas-soft) px-3 py-2"
                            >
                              <ItemContent>
                                <ItemHeader className="justify-start">
                                  <Eyebrow tone="accent">{slot.name}</Eyebrow>
                                  <Pill mono tone={slot.required === false ? "neutral" : "warning"}>
                                    {slot.required === false ? "OPTIONAL" : "REQUIRED"}
                                  </Pill>
                                </ItemHeader>
                                <p className="text-xs leading-relaxed text-(--muted)">
                                  {describeBridgeSecretSlot(slot)}
                                </p>
                              </ItemContent>
                            </Item>
                          ))}
                        </ItemGroup>
                      ) : (
                        <p className="mt-3 text-small-body leading-relaxed text-(--muted)">
                          This provider does not declare secret slot requirements in its manifest.
                        </p>
                      )}
                    </div>
                  </MetadataList>

                  <FieldGroup className="mt-4 gap-4">
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
                          className="text-small-body text-(--danger)"
                          data-testid="bridge-provider-config-error"
                        >
                          {providerConfigError}
                        </p>
                      ) : (
                        <p className="text-xs leading-relaxed text-(--subtle)">
                          Hint: {describeBridgeProviderConfigSchema(selectedProvider.config_schema)}
                        </p>
                      )}
                    </Field>
                  </FieldGroup>
                </Section>
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

              <Section label="Routing policy">
                <p className="text-small-body text-(--muted)">
                  {describeBridgeRoutingPolicy(draft.routingPolicy)}
                </p>
                <FieldGroup className="mt-3 gap-3">
                  <Field orientation="horizontal">
                    <Switch
                      checked={draft.routingPolicy.include_peer}
                      data-testid="bridge-routing-include-peer"
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
                      data-testid="bridge-routing-include-group"
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
                      data-testid="bridge-routing-include-thread"
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
              </Section>
            </FieldSet>
          </div>

          <DialogFooter variant="ruled">
            <Button onClick={() => onOpenChange(false)} size="sm" type="button" variant="outline">
              Cancel
            </Button>
            <Button
              data-testid="submit-bridge-create"
              disabled={!canSubmit || isPending}
              size="sm"
              onClick={onSubmit}
              type="button"
            >
              {isPending ? (
                <>
                  <Spinner className="size-3.5" />
                  Creating…
                </>
              ) : (
                "Create Bridge"
              )}
            </Button>
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  );
}
