import {
  ArrowLeft,
  ArrowRight,
  Check,
  ChevronRight,
  Plug,
  Settings2,
  Waypoints,
} from "lucide-react";
import { useId, useMemo, useState } from "react";

import {
  bridgeKindIconRegistry,
  Button,
  CatalogCard,
  Dialog,
  DialogContent,
  DialogTitle,
  Eyebrow,
  Field,
  FieldContent,
  FieldDescription,
  FieldTitle,
  FormSection,
  Input,
  KindChip,
  KindIcon,
  NativeSelect,
  NativeSelectOption,
  Pill,
  Spinner,
  Switch,
  Textarea,
} from "@agh/ui";

import { providerHealthTone, providerStateTone } from "@/systems/model-catalog";
import { parseBridgeProviderConfig } from "../lib/bridge-drafts";
import {
  buildBridgeProviderKey,
  describeBridgeDmPolicy,
  describeBridgeProviderConfigSchema,
  describeBridgeRoutingPolicy,
  describeBridgeSecretSlot,
  findBridgeProviderByKey,
  isBridgeProviderSelectable,
} from "../lib/bridge-formatters";
import type { BridgeCreateDraft, BridgeProvider } from "../types";

type WizardStep = "provider" | "runtime" | "delivery";

interface WizardStepDescriptor {
  id: WizardStep;
  label: string;
  testId: string;
}

const WIZARD_STEPS: readonly WizardStepDescriptor[] = [
  { id: "provider", label: "Provider", testId: "bridge-wizard-step-provider" },
  { id: "runtime", label: "Runtime", testId: "bridge-wizard-step-runtime" },
  { id: "delivery", label: "Delivery", testId: "bridge-wizard-step-delivery" },
] as const;

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
  return <BridgeCreateDialogContent {...props} />;
}

function BridgeCreateDialogContent({
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
  const titleId = useId();
  const [step, setStep] = useState<WizardStep>("provider");

  const selectedProvider = findBridgeProviderByKey(providers, draft.selectedProviderKey);
  const providerConfigError = parseBridgeProviderConfig(draft.providerConfigText).error;
  const hasProviderChoice = Boolean(
    selectedProvider && isBridgeProviderSelectable(selectedProvider)
  );
  const hasIdentity = Boolean(draft.displayName.trim());

  const stepValidity: Record<WizardStep, boolean> = {
    provider: hasProviderChoice,
    runtime: hasIdentity && !providerConfigError,
    delivery: true,
  };

  const canSubmit = stepValidity.provider && stepValidity.runtime && stepValidity.delivery;

  const currentIndex = WIZARD_STEPS.findIndex(item => item.id === step);
  const isFirstStep = currentIndex === 0;
  const isLastStep = currentIndex === WIZARD_STEPS.length - 1;
  const nextStep = !isLastStep ? WIZARD_STEPS[currentIndex + 1] : undefined;
  const previousStep = !isFirstStep ? WIZARD_STEPS[currentIndex - 1] : undefined;
  const canAdvance = stepValidity[step];

  const handleOpenChange = (next: boolean) => {
    if (!next) {
      setStep("provider");
    }
    onOpenChange(next);
  };

  return (
    <Dialog onOpenChange={handleOpenChange} open={open}>
      <DialogContent
        aria-labelledby={titleId}
        className="w-(--width-modal-lg) max-w-[calc(100vw-2rem)] sm:max-w-(--width-modal-lg) grid-rows-[auto_auto_1fr_auto] max-h-[min(var(--height-modal-tall),calc(100vh-2rem))]"
        data-testid="bridge-create-dialog"
        showCloseButton={false}
        unframed
      >
        <header
          data-slot="bridge-wizard-head"
          className="flex items-center justify-between gap-3 border-b border-line px-5 py-3.5"
        >
          <DialogTitle
            id={titleId}
            data-testid="bridge-wizard-title"
            className="text-modal-title font-medium tracking-modal-title text-fg-strong"
          >
            Create bridge
          </DialogTitle>
          {selectedProvider ? (
            <span
              className="font-mono text-form-label text-muted"
              data-testid="bridge-wizard-active-provider"
            >
              {selectedProvider.display_name}
            </span>
          ) : null}
        </header>

        <nav
          aria-label="Bridge create steps"
          className="flex items-center gap-2 border-b border-line bg-canvas-tint px-5 py-2.5 text-eyebrow"
          data-testid="bridge-wizard-stepper"
        >
          {WIZARD_STEPS.map((item, index) => {
            const status = stepStatus(index, currentIndex);
            return (
              <span
                key={item.id}
                className="flex items-center gap-2"
                data-testid={item.testId}
                data-status={status}
              >
                <span
                  aria-hidden="true"
                  className={cnStepBadge(status)}
                  data-slot="bridge-wizard-step-badge"
                >
                  {status === "complete" ? (
                    <Check width={11} height={11} strokeWidth={2} />
                  ) : (
                    <span className="font-mono text-eyebrow">{index + 1}</span>
                  )}
                </span>
                <Eyebrow className={status === "current" ? "text-fg-strong" : "text-muted"}>
                  {item.label}
                </Eyebrow>
                {index < WIZARD_STEPS.length - 1 ? (
                  <ChevronRight
                    aria-hidden="true"
                    width={12}
                    height={12}
                    strokeWidth={1.75}
                    className="ml-1 text-faint"
                  />
                ) : null}
              </span>
            );
          })}
        </nav>

        <div
          className="flex min-h-0 flex-col gap-4 overflow-y-auto p-5"
          data-testid="bridge-wizard-body"
        >
          {step === "provider" ? (
            <ProviderStep
              onSelect={key => {
                const provider = findBridgeProviderByKey(providers, key);
                onDraftChange({
                  ...draft,
                  displayName:
                    !draft.displayName.trim() ||
                    draft.displayName.trim() === selectedProvider?.display_name
                      ? (provider?.display_name ?? draft.displayName)
                      : draft.displayName,
                  selectedProviderKey: key,
                });
              }}
              providers={providers}
              selectedProviderKey={draft.selectedProviderKey}
            />
          ) : null}

          {step === "runtime" && selectedProvider ? (
            <RuntimeStep
              activeWorkspaceId={activeWorkspaceId}
              activeWorkspaceName={activeWorkspaceName}
              draft={draft}
              onDraftChange={onDraftChange}
              provider={selectedProvider}
              providerConfigError={providerConfigError}
            />
          ) : null}

          {step === "runtime" && !selectedProvider ? <RuntimeMissingProviderState /> : null}

          {step === "delivery" ? (
            <DeliveryStep draft={draft} onDraftChange={onDraftChange} />
          ) : null}
        </div>

        <footer
          className="flex flex-wrap items-center gap-3 border-t border-line bg-canvas-soft px-5 py-3.5"
          data-slot="bridge-wizard-footer"
        >
          <span
            className="font-mono text-form-label text-muted"
            data-testid="bridge-wizard-progress"
          >
            Step {currentIndex + 1} of {WIZARD_STEPS.length}
          </span>
          <div className="ml-auto flex flex-wrap items-center gap-2">
            <Button
              data-testid="bridge-wizard-cancel"
              onClick={() => handleOpenChange(false)}
              size="sm"
              type="button"
              variant="outline"
            >
              Cancel
            </Button>
            {previousStep ? (
              <Button
                data-testid="bridge-wizard-back"
                onClick={() => setStep(previousStep.id)}
                size="sm"
                type="button"
                variant="outline"
              >
                <ArrowLeft className="size-3" />
                Back
              </Button>
            ) : null}
            {nextStep ? (
              <Button
                data-testid="bridge-wizard-next"
                disabled={!canAdvance}
                onClick={() => setStep(nextStep.id)}
                size="sm"
                type="button"
              >
                Continue
                <ArrowRight className="size-3" />
              </Button>
            ) : (
              <Button
                data-testid="submit-bridge-create"
                disabled={!canSubmit || isPending}
                onClick={onSubmit}
                size="sm"
                type="button"
              >
                {isPending ? (
                  <>
                    <Spinner className="size-3" />
                    Creating…
                  </>
                ) : (
                  "Create Bridge"
                )}
              </Button>
            )}
          </div>
        </footer>
      </DialogContent>
    </Dialog>
  );
}

function stepStatus(index: number, currentIndex: number): "complete" | "current" | "pending" {
  if (index < currentIndex) return "complete";
  if (index === currentIndex) return "current";
  return "pending";
}

function cnStepBadge(status: "complete" | "current" | "pending"): string {
  const base =
    "inline-flex size-5 shrink-0 items-center justify-center rounded-full text-fg-strong transition-colors duration-base ease-out";
  if (status === "complete") return `${base} bg-success-tint text-success`;
  if (status === "current") return `${base} bg-surface-glaze text-fg-strong`;
  return `${base} bg-canvas-soft text-subtle`;
}

interface ProviderStepProps {
  providers: BridgeProvider[];
  selectedProviderKey: string;
  onSelect: (providerKey: string) => void;
}

function ProviderStep({ providers, selectedProviderKey, onSelect }: ProviderStepProps) {
  return (
    <FormSection
      data-testid="bridge-wizard-section-provider"
      description="Only providers with healthy runtime state can be selected for bridge creation."
      icon={Plug}
      title="Provider"
    >
      {providers.length === 0 ? (
        <div
          className="rounded bg-canvas-tint px-5 py-8 text-center text-small-body leading-6 text-muted"
          data-testid="bridge-provider-empty"
        >
          No bridge providers are currently available. Install or enable a bridge adapter extension
          before creating a new bridge.
        </div>
      ) : (
        <div className="grid gap-3 lg:grid-cols-2" data-testid="bridge-wizard-provider-grid">
          {providers.map(provider => {
            const providerKey = buildBridgeProviderKey(provider);
            return (
              <BridgeProviderCatalogCard
                key={providerKey}
                onSelect={() => onSelect(providerKey)}
                provider={provider}
                selected={providerKey === selectedProviderKey}
              />
            );
          })}
        </div>
      )}
    </FormSection>
  );
}

interface BridgeProviderCatalogCardProps {
  provider: BridgeProvider;
  selected: boolean;
  onSelect: () => void;
}

function BridgeProviderCatalogCard({
  provider,
  selected,
  onSelect,
}: BridgeProviderCatalogCardProps) {
  const providerKey = buildBridgeProviderKey(provider);
  const selectable = isBridgeProviderSelectable(provider);

  return (
    <CatalogCard
      actionable={selectable}
      aria-disabled={selectable ? undefined : true}
      aria-pressed={selected}
      data-testid={`bridge-provider-card-${providerKey}`}
      onClick={selectable ? onSelect : undefined}
      role="button"
      selected={selected}
      tabIndex={selectable ? 0 : -1}
      onKeyDown={
        selectable
          ? event => {
              if (event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                onSelect();
              }
            }
          : undefined
      }
    >
      <div className="flex items-start gap-3">
        <CatalogCard.Logo size="lg">
          <KindIcon
            kind={provider.platform}
            registry={bridgeKindIconRegistry}
            size="md"
            tone="default"
          />
        </CatalogCard.Logo>
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <div className="flex flex-wrap items-start justify-between gap-2">
            <CatalogCard.Title className="min-w-0">{provider.display_name}</CatalogCard.Title>
            <Pill mono tone={providerHealthTone(provider.health)}>
              {provider.health}
            </Pill>
          </div>
          <div className="flex flex-wrap items-center gap-1.5">
            <KindChip kind={provider.platform} />
            <Eyebrow className="text-muted">{provider.extension_name}</Eyebrow>
          </div>
        </div>
      </div>
      <CatalogCard.Description>
        {provider.description ?? "Bridge adapter installed and ready for instance configuration."}
      </CatalogCard.Description>
      <CatalogCard.Actions className="border-t-0 pt-0">
        <Pill mono tone={providerStateTone(provider.state)}>
          {provider.state}
        </Pill>
        {selectable ? null : (
          <Pill mono tone="danger">
            UNAVAILABLE
          </Pill>
        )}
      </CatalogCard.Actions>
    </CatalogCard>
  );
}

interface RuntimeStepProps {
  activeWorkspaceId?: string | null;
  activeWorkspaceName?: string | null;
  draft: BridgeCreateDraft;
  onDraftChange: (draft: BridgeCreateDraft) => void;
  provider: BridgeProvider;
  providerConfigError?: string;
}

function RuntimeStep({
  activeWorkspaceId,
  activeWorkspaceName,
  draft,
  onDraftChange,
  provider,
  providerConfigError,
}: RuntimeStepProps) {
  const configSchema = useMemo(
    () => describeBridgeProviderConfigSchema(provider.config_schema),
    [provider.config_schema]
  );

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
                        {slot.required === false ? "OPTIONAL" : "REQUIRED"}
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

function RuntimeMissingProviderState() {
  return (
    <FormSection
      data-testid="bridge-wizard-section-runtime-missing"
      icon={Plug}
      title="Provider runtime"
    >
      <p className="text-small-body text-muted">
        Select a provider before configuring runtime details.
      </p>
    </FormSection>
  );
}

interface RuntimeMetadataTileProps {
  label: string;
  right?: React.ReactNode;
  children: React.ReactNode;
}

function RuntimeMetadataTile({ label, right, children }: RuntimeMetadataTileProps) {
  return (
    <div className="flex flex-col gap-1 rounded bg-canvas-tint px-3 py-2.5">
      <div className="flex items-center justify-between gap-2">
        <Eyebrow className="text-muted">{label}</Eyebrow>
        {right ?? null}
      </div>
      <div className="text-small-body text-fg">{children}</div>
    </div>
  );
}

interface DeliveryStepProps {
  draft: BridgeCreateDraft;
  onDraftChange: (draft: BridgeCreateDraft) => void;
}

function DeliveryStep({ draft, onDraftChange }: DeliveryStepProps) {
  return (
    <>
      <FormSection
        data-testid="bridge-wizard-section-routing"
        description={describeBridgeRoutingPolicy(draft.routingPolicy)}
        icon={Waypoints}
        title="Routing policy"
      >
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
            <FieldDescription>Differentiate direct targets by peer identifier.</FieldDescription>
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
      </FormSection>

      <FormSection
        data-testid="bridge-wizard-section-delivery"
        description="These defaults are applied when resolving outbound delivery targets."
        icon={Settings2}
        title="Delivery defaults"
      >
        <div className="grid gap-4 md:grid-cols-2">
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
        </div>
      </FormSection>
    </>
  );
}
