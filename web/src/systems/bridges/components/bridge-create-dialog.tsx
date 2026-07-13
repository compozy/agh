import { ArrowLeft, ArrowRight, Check, ChevronRight, Settings2, Waypoints } from "lucide-react";
import { useId, useState } from "react";

import { Button, Dialog, DialogContent, DialogTitle, Eyebrow, FormSection, Spinner } from "@agh/ui";

import { parseBridgeProviderConfig } from "../lib/bridge-drafts";
import {
  describeBridgeRoutingPolicy,
  findBridgeProviderByKey,
  isBridgeProviderSelectable,
} from "../lib/bridge-formatters";
import type { BridgeCreateDraft, BridgeProvider } from "../types";
import { BridgeCreateProviderStep } from "./bridge-create-provider-step";
import {
  BridgeCreateRuntimeMissingProvider,
  BridgeCreateRuntimeStep,
} from "./bridge-create-runtime-step";
import { BridgeDeliveryFields, BridgeRoutingFields } from "./bridge-delivery-fields";
import {
  BridgeManifestHandoff,
  type BridgeManifestCommittedState,
} from "./bridge-manifest-handoff";

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
  manifestState?: BridgeManifestCommittedState | null;
  onDraftChange: (draft: BridgeCreateDraft) => void;
  onOpenChange: (open: boolean) => void;
  onSubmit: () => void;
  open: boolean;
  providers: BridgeProvider[];
  supportsManifest?: boolean;
}

function stepStatus(index: number, currentIndex: number): "complete" | "current" | "pending" {
  if (index < currentIndex) return "complete";
  if (index === currentIndex) return "current";
  return "pending";
}

function stepBadgeClass(status: "complete" | "current" | "pending"): string {
  const base =
    "inline-flex size-5 shrink-0 items-center justify-center rounded-full text-fg-strong transition-colors duration-base ease-out";
  if (status === "complete") return `${base} bg-success-tint text-success`;
  if (status === "current") return `${base} bg-surface-glaze text-fg-strong`;
  return `${base} bg-canvas-soft text-subtle`;
}

function BridgeWizardStepNav({ currentIndex }: { currentIndex: number }) {
  return (
    <nav
      aria-label="Bridge create steps"
      className="grid min-w-0 grid-cols-3 items-center gap-1 border-b border-line bg-canvas-tint px-3 py-2.5 text-eyebrow sm:flex sm:gap-2 sm:px-5"
      data-testid="bridge-wizard-stepper"
    >
      {WIZARD_STEPS.map((item, index) => {
        const status = stepStatus(index, currentIndex);
        return (
          <span
            aria-current={status === "current" ? "step" : undefined}
            className="flex min-w-0 items-center justify-center gap-1.5 sm:justify-start sm:gap-2"
            data-status={status}
            data-testid={item.testId}
            key={item.id}
          >
            <span
              aria-hidden="true"
              className={stepBadgeClass(status)}
              data-slot="bridge-wizard-step-badge"
            >
              {status === "complete" ? (
                <Check height={11} strokeWidth={2} width={11} />
              ) : (
                <span className="font-mono text-eyebrow">{index + 1}</span>
              )}
            </span>
            <Eyebrow
              className={`min-w-0 truncate ${status === "current" ? "text-fg-strong" : "text-muted"}`}
            >
              {item.label}
            </Eyebrow>
            {index < WIZARD_STEPS.length - 1 ? (
              <ChevronRight
                aria-hidden="true"
                className="ml-1 hidden shrink-0 text-faint sm:block"
                height={12}
                strokeWidth={1.75}
                width={12}
              />
            ) : null}
          </span>
        );
      })}
    </nav>
  );
}

function BridgeDeliveryStep({
  draft,
  onDraftChange,
}: {
  draft: BridgeCreateDraft;
  onDraftChange: (draft: BridgeCreateDraft) => void;
}) {
  return (
    <>
      <FormSection
        data-testid="bridge-wizard-section-routing"
        description={describeBridgeRoutingPolicy(draft.routingPolicy)}
        icon={Waypoints}
        title="Routing policy"
      >
        <BridgeRoutingFields
          onChange={routingPolicy => onDraftChange({ ...draft, routingPolicy })}
          testIdPrefix="bridge"
          value={draft.routingPolicy}
        />
      </FormSection>
      <FormSection
        data-testid="bridge-wizard-section-delivery"
        description="These defaults are applied when resolving outbound delivery targets."
        icon={Settings2}
        title="Delivery defaults"
      >
        <BridgeDeliveryFields
          onChange={deliveryDefaults => onDraftChange({ ...draft, deliveryDefaults })}
          testIdPrefix="bridge"
          value={draft.deliveryDefaults}
        />
      </FormSection>
    </>
  );
}

export function BridgeCreateDialog({
  activeWorkspaceId,
  activeWorkspaceName,
  draft,
  isPending,
  manifestState,
  onDraftChange,
  onOpenChange,
  onSubmit,
  open,
  providers,
  supportsManifest = false,
}: BridgeCreateDialogProps) {
  const titleId = useId();
  const [step, setStep] = useState<WizardStep>("provider");
  const committedManifestState = supportsManifest ? manifestState : null;
  const selectedProvider = findBridgeProviderByKey(providers, draft.selectedProviderKey);
  const providerConfigError = parseBridgeProviderConfig(draft.providerConfigText).error;
  const stepValidity: Record<WizardStep, boolean> = {
    provider: Boolean(selectedProvider && isBridgeProviderSelectable(selectedProvider)),
    runtime: Boolean(draft.displayName.trim() && !providerConfigError),
    delivery: true,
  };
  const currentIndex = WIZARD_STEPS.findIndex(item => item.id === step);
  const previousStep = currentIndex > 0 ? WIZARD_STEPS[currentIndex - 1] : undefined;
  const nextStep =
    currentIndex < WIZARD_STEPS.length - 1 ? WIZARD_STEPS[currentIndex + 1] : undefined;
  const canSubmit = Object.values(stepValidity).every(Boolean);

  const handleOpenChange = (next: boolean) => {
    if (!next && isPending) return;
    if (!next) setStep("provider");
    onOpenChange(next);
  };

  const selectProvider = (key: string) => {
    const provider = findBridgeProviderByKey(providers, key);
    onDraftChange({
      ...draft,
      displayName:
        !draft.displayName.trim() || draft.displayName.trim() === selectedProvider?.display_name
          ? (provider?.display_name ?? draft.displayName)
          : draft.displayName,
      selectedProviderKey: key,
    });
  };

  return (
    <Dialog onOpenChange={handleOpenChange} open={open}>
      <DialogContent
        aria-labelledby={titleId}
        className={
          committedManifestState
            ? "w-(--width-modal-lg) min-w-0 max-w-[calc(100vw-2rem)] grid-cols-[minmax(0,1fr)] grid-rows-[auto_1fr] max-h-[min(var(--height-modal-tall),calc(100vh-2rem))] sm:max-w-(--width-modal-lg)"
            : "w-(--width-modal-lg) min-w-0 max-w-[calc(100vw-2rem)] grid-cols-[minmax(0,1fr)] grid-rows-[auto_auto_1fr_auto] max-h-[min(var(--height-modal-tall),calc(100vh-2rem))] sm:max-w-(--width-modal-lg)"
        }
        data-testid="bridge-create-dialog"
        showCloseButton={false}
        unframed
      >
        <header
          className="flex min-w-0 items-center justify-between gap-3 border-b border-line px-3 py-3 sm:px-5 sm:py-3.5"
          data-slot="bridge-wizard-head"
        >
          <DialogTitle
            className="text-modal-title font-medium tracking-modal-title text-fg-strong"
            data-testid="bridge-wizard-title"
            id={titleId}
          >
            {committedManifestState ? "Set up Slack app" : "Create bridge"}
          </DialogTitle>
          <span className="min-w-0 truncate font-mono text-form-label text-muted">
            {committedManifestState?.bridgeId ?? selectedProvider?.display_name}
          </span>
        </header>

        {committedManifestState ? (
          <div className="min-h-0 min-w-0 overflow-y-auto p-3 sm:p-5">
            <BridgeManifestHandoff state={committedManifestState} />
          </div>
        ) : (
          <>
            <BridgeWizardStepNav currentIndex={currentIndex} />
            <div
              className="flex min-h-0 min-w-0 flex-col gap-4 overflow-y-auto p-3 sm:p-5"
              data-testid="bridge-wizard-body"
            >
              {step === "provider" ? (
                <BridgeCreateProviderStep
                  onSelect={selectProvider}
                  providers={providers}
                  selectedProviderKey={draft.selectedProviderKey}
                  supportsManifest={supportsManifest}
                />
              ) : null}
              {step === "runtime" && selectedProvider ? (
                <BridgeCreateRuntimeStep
                  activeWorkspaceId={activeWorkspaceId}
                  activeWorkspaceName={activeWorkspaceName}
                  draft={draft}
                  onDraftChange={onDraftChange}
                  provider={selectedProvider}
                  providerConfigError={providerConfigError}
                />
              ) : null}
              {step === "runtime" && !selectedProvider ? (
                <BridgeCreateRuntimeMissingProvider />
              ) : null}
              {step === "delivery" ? (
                <BridgeDeliveryStep draft={draft} onDraftChange={onDraftChange} />
              ) : null}
            </div>
            <footer
              className="flex min-w-0 flex-wrap items-center gap-2 border-t border-line bg-canvas-soft px-3 py-3 sm:gap-3 sm:px-5 sm:py-3.5"
              data-slot="bridge-wizard-footer"
            >
              <span
                className="shrink-0 font-mono text-form-label text-muted"
                data-testid="bridge-wizard-progress"
              >
                Step {currentIndex + 1} of {WIZARD_STEPS.length}
              </span>
              <div className="ml-auto flex max-w-full flex-wrap items-center justify-end gap-2">
                <Button
                  data-testid="bridge-wizard-cancel"
                  disabled={isPending}
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
                    <ArrowLeft aria-hidden="true" className="size-3" />
                    Back
                  </Button>
                ) : null}
                {nextStep ? (
                  <Button
                    data-testid="bridge-wizard-next"
                    disabled={!stepValidity[step]}
                    onClick={() => setStep(nextStep.id)}
                    size="sm"
                    type="button"
                  >
                    Continue
                    <ArrowRight aria-hidden="true" className="size-3" />
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
                    ) : supportsManifest ? (
                      "Create and continue"
                    ) : (
                      "Create bridge"
                    )}
                  </Button>
                )}
              </div>
            </footer>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
